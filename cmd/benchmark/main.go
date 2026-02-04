package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sort"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const (
	defaultIterations = 10
	warmupIterations  = 3
)

type BenchmarkResult struct {
	Name        string
	Query       string
	Iterations  int
	TotalTime   time.Duration
	MinTime     time.Duration
	MaxTime     time.Duration
	AvgTime     time.Duration
	MedianTime  time.Duration
	P95Time     time.Duration
	RowsScanned int64
}

func main() {
	host := flag.String("host", getEnv("DB_HOST", "localhost"), "Database host")
	port := flag.String("port", getEnv("DB_PORT", "3306"), "Database port")
	user := flag.String("user", getEnv("DB_USER", "app"), "Database user")
	password := flag.String("password", getEnv("DB_PASSWORD", "app"), "Database password")
	dbname := flag.String("db", getEnv("DB_NAME", "bookdb"), "Database name")
	iterations := flag.Int("iterations", defaultIterations, "Number of iterations per query")
	showExplain := flag.Bool("explain", false, "Show EXPLAIN output for each query")

	flag.Parse()

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=Local",
		*user, *password, *host, *port, *dbname)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Connected to database")
	log.Printf("Running benchmarks with %d iterations (+ %d warmup)...\n", *iterations, warmupIterations)

	// Get table stats
	printTableStats(db)

	benchmarks := []struct {
		name  string
		query string
	}{
		{
			name:  "Simple SELECT with LIMIT",
			query: "SELECT id, title, author_id, created_at FROM books ORDER BY id LIMIT 1000",
		},
		{
			name:  "Primary key lookup",
			query: "SELECT id, title, author_id, created_at FROM books WHERE id = ?",
		},
		{
			name:  "Date range query (1 month)",
			query: "SELECT id, title, author_id, created_at FROM books WHERE created_at BETWEEN '2022-06-01' AND '2022-06-30'",
		},
		{
			name:  "Date range query (1 year)",
			query: "SELECT id, title, author_id, created_at FROM books WHERE created_at BETWEEN '2022-01-01' AND '2022-12-31'",
		},
		{
			name:  "Author lookup with books count",
			query: "SELECT a.id, a.name, COUNT(b.id) as book_count FROM authors a LEFT JOIN books b ON a.id = b.author_id WHERE a.id = ? GROUP BY a.id, a.name",
		},
		{
			name:  "JOIN books with authors",
			query: "SELECT b.id, b.title, a.name as author_name FROM books b INNER JOIN authors a ON b.author_id = a.id ORDER BY b.id LIMIT 1000",
		},
		{
			name:  "Books with specific tag",
			query: "SELECT b.id, b.title FROM books b INNER JOIN book_tags bt ON b.id = bt.book_id WHERE bt.tag_id = ?",
		},
		{
			name:  "Count books by author (TOP 10)",
			query: "SELECT author_id, COUNT(*) as cnt FROM books GROUP BY author_id ORDER BY cnt DESC LIMIT 10",
		},
		{
			name:  "Count books by year",
			query: "SELECT YEAR(created_at) as year, COUNT(*) as cnt FROM books GROUP BY YEAR(created_at) ORDER BY year",
		},
		{
			name:  "Full table count",
			query: "SELECT COUNT(*) FROM books",
		},
		{
			name:  "Count with date filter",
			query: "SELECT COUNT(*) FROM books WHERE created_at >= '2023-01-01'",
		},
		{
			name:  "Complex JOIN (books -> tags)",
			query: "SELECT b.id, b.title, GROUP_CONCAT(t.name) as tags FROM books b INNER JOIN book_tags bt ON b.id = bt.book_id INNER JOIN tags t ON bt.tag_id = t.id WHERE b.id BETWEEN ? AND ? GROUP BY b.id, b.title",
		},
		{
			name:  "Subquery: Books by prolific authors",
			query: "SELECT id, title FROM books WHERE author_id IN (SELECT author_id FROM books GROUP BY author_id HAVING COUNT(*) > 100) LIMIT 1000",
		},
	}

	fmt.Println("\n" + strings.Repeat("=", 100))
	fmt.Println("BENCHMARK RESULTS")
	fmt.Println(strings.Repeat("=", 100))

	for _, bm := range benchmarks {
		result := runBenchmark(db, bm.name, bm.query, *iterations)
		printResult(result)

		if *showExplain {
			showExplainOutput(db, bm.query)
		}
	}

	// Check partition usage
	fmt.Println("\n" + strings.Repeat("=", 100))
	fmt.Println("PARTITION INFORMATION")
	fmt.Println(strings.Repeat("=", 100))
	showPartitionInfo(db)
}

func printTableStats(db *sql.DB) {
	fmt.Println("\n" + strings.Repeat("=", 100))
	fmt.Println("TABLE STATISTICS")
	fmt.Println(strings.Repeat("=", 100))

	tables := []string{"authors", "books", "tags", "book_tags", "author_tags"}
	for _, table := range tables {
		var count int64
		err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count)
		if err != nil {
			log.Printf("Error counting %s: %v", table, err)
			continue
		}
		fmt.Printf("%-15s: %12d rows\n", table, count)
	}
}

func runBenchmark(db *sql.DB, name, query string, iterations int) BenchmarkResult {
	result := BenchmarkResult{
		Name:       name,
		Query:      query,
		Iterations: iterations,
		MinTime:    time.Hour,
	}

	durations := make([]time.Duration, 0, iterations)

	// Determine query parameters
	needsParam := strings.Contains(query, "?")
	var params []interface{}

	if needsParam {
		// Get max IDs for random selection
		var maxBookID, maxAuthorID, maxTagID int64
		db.QueryRow("SELECT COALESCE(MAX(id), 1) FROM books").Scan(&maxBookID)
		db.QueryRow("SELECT COALESCE(MAX(id), 1) FROM authors").Scan(&maxAuthorID)
		db.QueryRow("SELECT COALESCE(MAX(id), 1) FROM tags").Scan(&maxTagID)

		if strings.Contains(query, "books WHERE id") || strings.Contains(query, "b.id BETWEEN") {
			if strings.Count(query, "?") == 2 {
				start := rand.Int63n(maxBookID-100) + 1
				params = []interface{}{start, start + 100}
			} else {
				params = []interface{}{rand.Int63n(maxBookID) + 1}
			}
		} else if strings.Contains(query, "authors") || strings.Contains(query, "a.id") {
			params = []interface{}{rand.Int63n(maxAuthorID) + 1}
		} else if strings.Contains(query, "tag_id") {
			params = []interface{}{rand.Int63n(maxTagID) + 1}
		}
	}

	// Warmup
	for i := 0; i < warmupIterations; i++ {
		if needsParam {
			db.Query(query, params...)
		} else {
			db.Query(query)
		}
	}

	// Actual benchmark
	for i := 0; i < iterations; i++ {
		// Randomize params each iteration
		if needsParam && len(params) > 0 {
			var maxID int64
			if strings.Contains(query, "books WHERE id") || strings.Contains(query, "b.id BETWEEN") {
				db.QueryRow("SELECT COALESCE(MAX(id), 1) FROM books").Scan(&maxID)
				if len(params) == 2 {
					start := rand.Int63n(maxID-100) + 1
					params = []interface{}{start, start + 100}
				} else {
					params = []interface{}{rand.Int63n(maxID) + 1}
				}
			} else if strings.Contains(query, "authors") || strings.Contains(query, "a.id") {
				db.QueryRow("SELECT COALESCE(MAX(id), 1) FROM authors").Scan(&maxID)
				params = []interface{}{rand.Int63n(maxID) + 1}
			} else if strings.Contains(query, "tag_id") {
				db.QueryRow("SELECT COALESCE(MAX(id), 1) FROM tags").Scan(&maxID)
				params = []interface{}{rand.Int63n(maxID) + 1}
			}
		}

		start := time.Now()
		var rows *sql.Rows
		var err error

		if needsParam {
			rows, err = db.Query(query, params...)
		} else {
			rows, err = db.Query(query)
		}

		if err != nil {
			log.Printf("Error running query: %v", err)
			continue
		}

		// Consume all rows
		rowCount := int64(0)
		for rows.Next() {
			rowCount++
		}
		rows.Close()

		duration := time.Since(start)
		durations = append(durations, duration)
		result.TotalTime += duration
		result.RowsScanned += rowCount

		if duration < result.MinTime {
			result.MinTime = duration
		}
		if duration > result.MaxTime {
			result.MaxTime = duration
		}
	}

	if len(durations) > 0 {
		result.AvgTime = result.TotalTime / time.Duration(len(durations))

		// Calculate median and P95
		sort.Slice(durations, func(i, j int) bool {
			return durations[i] < durations[j]
		})
		result.MedianTime = durations[len(durations)/2]
		p95Index := int(float64(len(durations)) * 0.95)
		if p95Index >= len(durations) {
			p95Index = len(durations) - 1
		}
		result.P95Time = durations[p95Index]
	}

	return result
}

func printResult(r BenchmarkResult) {
	fmt.Printf("\n%-40s\n", r.Name)
	fmt.Printf("  Query: %s\n", truncateString(r.Query, 80))
	fmt.Printf("  Iterations: %d | Total rows: %d\n", r.Iterations, r.RowsScanned)
	fmt.Printf("  Min: %v | Max: %v | Avg: %v | Median: %v | P95: %v\n",
		r.MinTime.Round(time.Microsecond),
		r.MaxTime.Round(time.Microsecond),
		r.AvgTime.Round(time.Microsecond),
		r.MedianTime.Round(time.Microsecond),
		r.P95Time.Round(time.Microsecond))
}

func showExplainOutput(db *sql.DB, query string) {
	// Replace parameters with sample values for EXPLAIN
	explainQuery := query
	for strings.Contains(explainQuery, "?") {
		explainQuery = strings.Replace(explainQuery, "?", "1", 1)
	}

	rows, err := db.Query("EXPLAIN " + explainQuery)
	if err != nil {
		log.Printf("Error running EXPLAIN: %v", err)
		return
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	fmt.Printf("  EXPLAIN:\n")
	fmt.Printf("    %s\n", strings.Join(cols, " | "))

	values := make([]interface{}, len(cols))
	valuePtrs := make([]interface{}, len(cols))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	for rows.Next() {
		rows.Scan(valuePtrs...)
		strs := make([]string, len(cols))
		for i, v := range values {
			if v == nil {
				strs[i] = "NULL"
			} else {
				strs[i] = fmt.Sprintf("%v", v)
			}
		}
		fmt.Printf("    %s\n", strings.Join(strs, " | "))
	}
}

func showPartitionInfo(db *sql.DB) {
	tables := []string{"books", "book_tags", "authors", "author_tags", "tags"}

	for _, table := range tables {
		rows, err := db.Query(`
			SELECT PARTITION_NAME, PARTITION_METHOD, PARTITION_EXPRESSION, TABLE_ROWS
			FROM INFORMATION_SCHEMA.PARTITIONS
			WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?
			ORDER BY PARTITION_ORDINAL_POSITION
		`, table)
		if err != nil {
			continue
		}

		fmt.Printf("\n%s:\n", table)

		hasPartitions := false
		for rows.Next() {
			var partName, partMethod, partExpr sql.NullString
			var tableRows sql.NullInt64
			rows.Scan(&partName, &partMethod, &partExpr, &tableRows)

			if partName.Valid && partName.String != "" {
				hasPartitions = true
				fmt.Printf("  Partition: %-15s | Method: %-10s | Expr: %-20s | Rows: %d\n",
					partName.String,
					partMethod.String,
					partExpr.String,
					tableRows.Int64)
			}
		}
		rows.Close()

		if !hasPartitions {
			fmt.Printf("  No partitions (standard table)\n")
		}
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
