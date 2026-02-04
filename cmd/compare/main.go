package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
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

type QueryPair struct {
	Name           string
	Description    string
	NoPartition    string
	WithPartition  string
	PartitionType  string // hash, range_year, range_id, list, key
}

type BenchResult struct {
	Name       string
	Durations  []time.Duration
	RowCount   int64
	Min        time.Duration
	Max        time.Duration
	Avg        time.Duration
	Median     time.Duration
	P95        time.Duration
}

func main() {
	host := flag.String("host", getEnv("DB_HOST", "localhost"), "Database host")
	port := flag.String("port", getEnv("DB_PORT", "3306"), "Database port")
	user := flag.String("user", getEnv("DB_USER", "app"), "Database user")
	password := flag.String("password", getEnv("DB_PASSWORD", "app"), "Database password")
	dbname := flag.String("db", getEnv("DB_NAME", "bookdb"), "Database name")
	iterations := flag.Int("iterations", defaultIterations, "Number of iterations per query")
	partitionType := flag.String("type", "hash", "Partition type to compare: hash, range_year, range_id, list, key, all")
	setupPartitions := flag.Bool("setup", false, "Create partition tables before comparison")

	flag.Parse()

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=Local&multiStatements=true",
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

	// Setup partitions if requested
	if *setupPartitions {
		setupPartitionTables(db, *partitionType)
	}

	// Print table stats
	printTableStats(db)

	// Get partition types to test
	types := []string{*partitionType}
	if *partitionType == "all" {
		types = []string{"hash", "range_year", "range_id", "list", "key"}
	}

	for _, ptype := range types {
		runComparison(db, ptype, *iterations)
	}
}

func setupPartitionTables(db *sql.DB, ptype string) {
	log.Printf("Setting up partition tables for type: %s", ptype)

	sqlFiles := map[string]string{
		"hash":       "scripts/partition/hash.sql",
		"range_year": "scripts/partition/range_by_year.sql",
		"range_id":   "scripts/partition/range_by_id.sql",
		"list":       "scripts/partition/list.sql",
		"key":        "scripts/partition/key.sql",
	}

	if ptype == "all" {
		for _, file := range sqlFiles {
			executeSQLFile(db, file)
		}
	} else if file, ok := sqlFiles[ptype]; ok {
		executeSQLFile(db, file)
	}
}

func executeSQLFile(db *sql.DB, filepath string) {
	content, err := os.ReadFile(filepath)
	if err != nil {
		log.Printf("Warning: Could not read %s: %v", filepath, err)
		return
	}

	statements := strings.Split(string(content), ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			// Ignore some expected errors
			if !strings.Contains(err.Error(), "doesn't exist") {
				log.Printf("Warning executing SQL: %v", err)
			}
		}
	}
	log.Printf("Executed: %s", filepath)
}

func printTableStats(db *sql.DB) {
	fmt.Println("\n" + strings.Repeat("=", 120))
	fmt.Println("TABLE STATISTICS")
	fmt.Println(strings.Repeat("=", 120))

	tables := []string{
		"books", "books_hash", "books_range_year", "books_range_id", "books_list", "books_key",
		"book_tags", "book_tags_hash", "book_tags_range_year", "book_tags_range_id", "book_tags_key",
	}

	for _, table := range tables {
		var count int64
		err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count)
		if err != nil {
			continue // Table doesn't exist
		}
		fmt.Printf("%-25s: %12d rows\n", table, count)
	}
}

func runComparison(db *sql.DB, ptype string, iterations int) {
	queries := getQueryPairs(ptype)

	fmt.Printf("\n%s\n", strings.Repeat("=", 120))
	fmt.Printf("PARTITION COMPARISON: %s\n", strings.ToUpper(ptype))
	fmt.Printf("%s\n", strings.Repeat("=", 120))

	for _, qp := range queries {
		if qp.PartitionType != ptype && qp.PartitionType != "all" {
			continue
		}

		// Check if partition table exists
		if !tableExists(db, extractTableName(qp.WithPartition)) {
			fmt.Printf("\n%s: SKIPPED (partition table not found)\n", qp.Name)
			continue
		}

		fmt.Printf("\n%s\n", qp.Name)
		fmt.Printf("  %s\n", qp.Description)
		fmt.Println(strings.Repeat("-", 100))

		// Benchmark without partition
		resultNoPart := benchmark(db, "No Partition", qp.NoPartition, iterations)

		// Benchmark with partition
		resultWithPart := benchmark(db, "With Partition", qp.WithPartition, iterations)

		// Print comparison
		printComparison(resultNoPart, resultWithPart)
	}
}

func getQueryPairs(ptype string) []QueryPair {
	pairs := []QueryPair{
		// HASH partition queries
		{
			Name:          "Primary Key Lookup (single row)",
			Description:   "SELECT by id - should benefit from HASH partition pruning",
			NoPartition:   "SELECT id, title, author_id, created_at FROM books WHERE id = 500",
			WithPartition: "SELECT id, title, author_id, created_at FROM books_hash WHERE id = 500",
			PartitionType: "hash",
		},
		{
			Name:          "Full Table Scan",
			Description:   "SELECT all - partition overhead comparison",
			NoPartition:   "SELECT COUNT(*) FROM books",
			WithPartition: "SELECT COUNT(*) FROM books_hash",
			PartitionType: "hash",
		},
		{
			Name:          "Range Scan by ID",
			Description:   "SELECT id range - HASH may scan all partitions",
			NoPartition:   "SELECT id, title FROM books WHERE id BETWEEN 100 AND 500",
			WithPartition: "SELECT id, title FROM books_hash WHERE id BETWEEN 100 AND 500",
			PartitionType: "hash",
		},
		{
			Name:          "JOIN with book_tags",
			Description:   "JOIN operation - partition alignment matters",
			NoPartition:   "SELECT b.id, b.title, COUNT(bt.tag_id) FROM books b LEFT JOIN book_tags bt ON b.id = bt.book_id WHERE b.id BETWEEN 1 AND 100 GROUP BY b.id, b.title",
			WithPartition: "SELECT b.id, b.title, COUNT(bt.tag_id) FROM books_hash b LEFT JOIN book_tags_hash bt ON b.id = bt.book_id WHERE b.id BETWEEN 1 AND 100 GROUP BY b.id, b.title",
			PartitionType: "hash",
		},

		// RANGE by year queries
		{
			Name:          "Date Range Query (1 year)",
			Description:   "SELECT by date range - RANGE partition pruning",
			NoPartition:   "SELECT id, title, created_at FROM books WHERE created_at BETWEEN '2022-01-01' AND '2022-12-31'",
			WithPartition: "SELECT id, title, created_at FROM books_range_year WHERE created_at BETWEEN '2022-01-01' AND '2022-12-31'",
			PartitionType: "range_year",
		},
		{
			Name:          "Date Range Query (1 month)",
			Description:   "SELECT by specific month - single partition access",
			NoPartition:   "SELECT id, title, created_at FROM books WHERE created_at BETWEEN '2022-06-01' AND '2022-06-30'",
			WithPartition: "SELECT id, title, created_at FROM books_range_year WHERE created_at BETWEEN '2022-06-01' AND '2022-06-30'",
			PartitionType: "range_year",
		},
		{
			Name:          "Count by Year",
			Description:   "GROUP BY year - partition-wise aggregation",
			NoPartition:   "SELECT YEAR(created_at) as y, COUNT(*) FROM books GROUP BY YEAR(created_at)",
			WithPartition: "SELECT YEAR(created_at) as y, COUNT(*) FROM books_range_year GROUP BY YEAR(created_at)",
			PartitionType: "range_year",
		},
		{
			Name:          "Cross-Year Query",
			Description:   "SELECT across multiple years - multiple partitions",
			NoPartition:   "SELECT id, title FROM books WHERE created_at >= '2021-06-01' AND created_at < '2023-06-01' LIMIT 1000",
			WithPartition: "SELECT id, title FROM books_range_year WHERE created_at >= '2021-06-01' AND created_at < '2023-06-01' LIMIT 1000",
			PartitionType: "range_year",
		},

		// RANGE by ID queries
		{
			Name:          "ID Range (within partition)",
			Description:   "SELECT id range within single partition boundary",
			NoPartition:   "SELECT id, title FROM books WHERE id BETWEEN 50000 AND 99999",
			WithPartition: "SELECT id, title FROM books_range_id WHERE id BETWEEN 50000 AND 99999",
			PartitionType: "range_id",
		},
		{
			Name:          "ID Range (cross partition)",
			Description:   "SELECT id range across partition boundaries",
			NoPartition:   "SELECT id, title FROM books WHERE id BETWEEN 95000 AND 105000",
			WithPartition: "SELECT id, title FROM books_range_id WHERE id BETWEEN 95000 AND 105000",
			PartitionType: "range_id",
		},

		// LIST partition queries
		{
			Name:          "Status Filter (single value)",
			Description:   "SELECT by status - single partition access",
			NoPartition:   "SELECT COUNT(*) FROM books",
			WithPartition: "SELECT COUNT(*) FROM books_list WHERE status = 1",
			PartitionType: "list",
		},
		{
			Name:          "Status Filter (multiple values)",
			Description:   "SELECT by multiple statuses",
			NoPartition:   "SELECT COUNT(*) FROM books",
			WithPartition: "SELECT COUNT(*) FROM books_list WHERE status IN (0, 1)",
			PartitionType: "list",
		},

		// KEY partition queries
		{
			Name:          "Composite Key Lookup",
			Description:   "SELECT by composite key - KEY partition optimization",
			NoPartition:   "SELECT * FROM book_tags WHERE book_id = 100 AND tag_id = 5",
			WithPartition: "SELECT * FROM book_tags_key WHERE book_id = 100 AND tag_id = 5",
			PartitionType: "key",
		},
		{
			Name:          "Partial Key Lookup",
			Description:   "SELECT by partial key - may scan all partitions",
			NoPartition:   "SELECT * FROM book_tags WHERE book_id = 100",
			WithPartition: "SELECT * FROM book_tags_key WHERE book_id = 100",
			PartitionType: "key",
		},
	}

	return pairs
}

func tableExists(db *sql.DB, tableName string) bool {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?", tableName).Scan(&count)
	return err == nil && count > 0
}

func extractTableName(query string) string {
	query = strings.ToLower(query)
	// Find FROM clause
	fromIdx := strings.Index(query, " from ")
	if fromIdx == -1 {
		return ""
	}
	rest := strings.TrimSpace(query[fromIdx+6:])
	// Get table name (first word)
	parts := strings.Fields(rest)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func benchmark(db *sql.DB, name, query string, iterations int) BenchResult {
	result := BenchResult{
		Name:      name,
		Durations: make([]time.Duration, 0, iterations),
		Min:       time.Hour,
	}

	// Warmup
	for i := 0; i < warmupIterations; i++ {
		rows, err := db.Query(query)
		if err == nil {
			for rows.Next() {
			}
			rows.Close()
		}
	}

	// Benchmark
	for i := 0; i < iterations; i++ {
		start := time.Now()
		rows, err := db.Query(query)
		if err != nil {
			log.Printf("Query error: %v", err)
			continue
		}

		count := int64(0)
		for rows.Next() {
			count++
		}
		rows.Close()

		duration := time.Since(start)
		result.Durations = append(result.Durations, duration)
		result.RowCount = count

		if duration < result.Min {
			result.Min = duration
		}
		if duration > result.Max {
			result.Max = duration
		}
	}

	// Calculate stats
	if len(result.Durations) > 0 {
		var total time.Duration
		for _, d := range result.Durations {
			total += d
		}
		result.Avg = total / time.Duration(len(result.Durations))

		sorted := make([]time.Duration, len(result.Durations))
		copy(sorted, result.Durations)
		sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

		result.Median = sorted[len(sorted)/2]
		p95Idx := int(float64(len(sorted)) * 0.95)
		if p95Idx >= len(sorted) {
			p95Idx = len(sorted) - 1
		}
		result.P95 = sorted[p95Idx]
	}

	return result
}

func printComparison(noPart, withPart BenchResult) {
	fmt.Printf("  %-20s | %12s | %12s | %12s | %12s | %12s | %8s\n",
		"", "Min", "Max", "Avg", "Median", "P95", "Rows")
	fmt.Printf("  %s\n", strings.Repeat("-", 98))

	fmt.Printf("  %-20s | %12v | %12v | %12v | %12v | %12v | %8d\n",
		noPart.Name,
		noPart.Min.Round(time.Microsecond),
		noPart.Max.Round(time.Microsecond),
		noPart.Avg.Round(time.Microsecond),
		noPart.Median.Round(time.Microsecond),
		noPart.P95.Round(time.Microsecond),
		noPart.RowCount)

	fmt.Printf("  %-20s | %12v | %12v | %12v | %12v | %12v | %8d\n",
		withPart.Name,
		withPart.Min.Round(time.Microsecond),
		withPart.Max.Round(time.Microsecond),
		withPart.Avg.Round(time.Microsecond),
		withPart.Median.Round(time.Microsecond),
		withPart.P95.Round(time.Microsecond),
		withPart.RowCount)

	// Calculate improvement
	if noPart.Avg > 0 && withPart.Avg > 0 {
		improvement := float64(noPart.Avg-withPart.Avg) / float64(noPart.Avg) * 100
		var indicator string
		if improvement > 0 {
			indicator = fmt.Sprintf("%.1f%% faster", improvement)
		} else if improvement < 0 {
			indicator = fmt.Sprintf("%.1f%% slower", -improvement)
		} else {
			indicator = "same"
		}
		fmt.Printf("  %-20s | %s\n", "Comparison", indicator)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
