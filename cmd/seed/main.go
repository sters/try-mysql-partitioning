package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const (
	defaultAuthors   = 10000
	defaultBooks     = 1000000
	defaultTags      = 1000
	defaultBookTags  = 5000000
	defaultAuthorTags = 50000

	bulkSize    = 5000  // Records per INSERT statement
	workerCount = 8     // Parallel workers
)

var (
	totalInserted int64
	totalTarget   int64
	startTime     time.Time
)

func main() {
	host := flag.String("host", getEnv("DB_HOST", "localhost"), "Database host")
	port := flag.String("port", getEnv("DB_PORT", "3306"), "Database port")
	user := flag.String("user", getEnv("DB_USER", "app"), "Database user")
	password := flag.String("password", getEnv("DB_PASSWORD", "app"), "Database password")
	dbname := flag.String("db", getEnv("DB_NAME", "bookdb"), "Database name")

	numAuthors := flag.Int("authors", defaultAuthors, "Number of authors to insert")
	numBooks := flag.Int("books", defaultBooks, "Number of books to insert")
	numTags := flag.Int("tags", defaultTags, "Number of tags to insert")
	numBookTags := flag.Int("book-tags", defaultBookTags, "Number of book-tag associations")
	numAuthorTags := flag.Int("author-tags", defaultAuthorTags, "Number of author-tag associations")
	truncate := flag.Bool("truncate", false, "Truncate tables before seeding")

	flag.Parse()

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=Local&maxAllowedPacket=256000000",
		*user, *password, *host, *port, *dbname)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(workerCount * 2)
	db.SetMaxIdleConns(workerCount)

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Connected to database")

	if *truncate {
		log.Println("Truncating tables...")
		truncateTables(db)
	}

	totalTarget = int64(*numAuthors + *numBooks + *numTags + *numBookTags + *numAuthorTags)
	startTime = time.Now()

	// Progress reporter
	done := make(chan struct{})
	go progressReporter(done)

	// Seed in order
	seedAuthors(db, *numAuthors)
	seedTags(db, *numTags)
	seedBooks(db, *numBooks, *numAuthors)
	seedBookTags(db, *numBookTags, *numBooks, *numTags)
	seedAuthorTags(db, *numAuthorTags, *numAuthors, *numTags)

	close(done)
	time.Sleep(100 * time.Millisecond)

	elapsed := time.Since(startTime)
	log.Printf("Completed! Total time: %v, Records: %d, Rate: %.0f/sec",
		elapsed.Round(time.Second), totalInserted, float64(totalInserted)/elapsed.Seconds())
}

func truncateTables(db *sql.DB) {
	tables := []string{"book_tags", "author_tags", "books", "tags", "authors"}
	for _, t := range tables {
		if _, err := db.Exec("TRUNCATE TABLE " + t); err != nil {
			log.Printf("Warning: failed to truncate %s: %v", t, err)
		}
	}
}

func progressReporter(done chan struct{}) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			current := atomic.LoadInt64(&totalInserted)
			elapsed := time.Since(startTime)
			rate := float64(current) / elapsed.Seconds()
			remaining := float64(totalTarget-current) / rate
			pct := float64(current) / float64(totalTarget) * 100

			log.Printf("Progress: %d/%d (%.1f%%) | Rate: %.0f/sec | ETA: %v",
				current, totalTarget, pct, rate, time.Duration(remaining)*time.Second)
		}
	}
}

func seedAuthors(db *sql.DB, count int) {
	log.Printf("Seeding %d authors...", count)

	baseTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.Local)

	for i := 0; i < count; i += bulkSize {
		end := i + bulkSize
		if end > count {
			end = count
		}

		values := make([]string, 0, end-i)
		args := make([]interface{}, 0, (end-i)*2)

		for j := i; j < end; j++ {
			values = append(values, "(?, ?)")
			createdAt := baseTime.Add(time.Duration(rand.Intn(5*365*24)) * time.Hour)
			args = append(args, fmt.Sprintf("Author %d", j+1), createdAt)
		}

		query := "INSERT INTO authors (name, created_at) VALUES " + strings.Join(values, ",")
		if _, err := db.Exec(query, args...); err != nil {
			log.Printf("Error inserting authors: %v", err)
		}

		atomic.AddInt64(&totalInserted, int64(end-i))
	}
}

func seedTags(db *sql.DB, count int) {
	log.Printf("Seeding %d tags...", count)

	tagCategories := []string{"Fiction", "Non-Fiction", "Science", "History", "Art", "Technology",
		"Philosophy", "Biography", "Travel", "Cooking", "Health", "Business", "Education", "Sports"}

	for i := 0; i < count; i += bulkSize {
		end := i + bulkSize
		if end > count {
			end = count
		}

		values := make([]string, 0, end-i)
		args := make([]interface{}, 0, end-i)

		for j := i; j < end; j++ {
			values = append(values, "(?)")
			category := tagCategories[rand.Intn(len(tagCategories))]
			args = append(args, fmt.Sprintf("%s-%d", category, j+1))
		}

		query := "INSERT INTO tags (name) VALUES " + strings.Join(values, ",")
		if _, err := db.Exec(query, args...); err != nil {
			log.Printf("Error inserting tags: %v", err)
		}

		atomic.AddInt64(&totalInserted, int64(end-i))
	}
}

func seedBooks(db *sql.DB, count, numAuthors int) {
	log.Printf("Seeding %d books with %d workers...", count, workerCount)

	booksPerWorker := count / workerCount
	var wg sync.WaitGroup

	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		start := w * booksPerWorker
		end := start + booksPerWorker
		if w == workerCount-1 {
			end = count
		}

		go func(workerID, start, end int) {
			defer wg.Done()
			insertBooks(db, workerID, start, end, numAuthors)
		}(w, start, end)
	}

	wg.Wait()
}

func insertBooks(db *sql.DB, workerID, start, end, numAuthors int) {
	baseTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.Local)
	titles := []string{"The Art of", "Introduction to", "Advanced", "Complete Guide to",
		"Mastering", "Understanding", "Practical", "Essential", "Modern", "Classic"}
	subjects := []string{"Programming", "Design", "Science", "History", "Mathematics",
		"Physics", "Chemistry", "Biology", "Economics", "Philosophy"}

	for i := start; i < end; i += bulkSize {
		batchEnd := i + bulkSize
		if batchEnd > end {
			batchEnd = end
		}

		values := make([]string, 0, batchEnd-i)
		args := make([]interface{}, 0, (batchEnd-i)*3)

		for j := i; j < batchEnd; j++ {
			values = append(values, "(?, ?, ?)")
			title := fmt.Sprintf("%s %s Vol.%d",
				titles[rand.Intn(len(titles))],
				subjects[rand.Intn(len(subjects))],
				j+1)
			authorID := rand.Intn(numAuthors) + 1
			createdAt := baseTime.Add(time.Duration(rand.Intn(5*365*24)) * time.Hour)
			args = append(args, title, authorID, createdAt)
		}

		query := "INSERT INTO books (title, author_id, created_at) VALUES " + strings.Join(values, ",")
		if _, err := db.Exec(query, args...); err != nil {
			log.Printf("Worker %d: Error inserting books: %v", workerID, err)
		}

		atomic.AddInt64(&totalInserted, int64(batchEnd-i))
	}
}

func seedBookTags(db *sql.DB, count, numBooks, numTags int) {
	log.Printf("Seeding %d book_tags with %d workers...", count, workerCount)

	tagsPerWorker := count / workerCount
	var wg sync.WaitGroup

	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		start := w * tagsPerWorker
		end := start + tagsPerWorker
		if w == workerCount-1 {
			end = count
		}

		go func(workerID, start, end int) {
			defer wg.Done()
			insertBookTags(db, workerID, start, end, numBooks, numTags)
		}(w, start, end)
	}

	wg.Wait()
}

func insertBookTags(db *sql.DB, workerID, start, end, numBooks, numTags int) {
	baseTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.Local)

	// Use a set to track inserted pairs within batch
	inserted := make(map[string]bool)

	for i := start; i < end; i += bulkSize {
		batchEnd := i + bulkSize
		if batchEnd > end {
			batchEnd = end
		}

		values := make([]string, 0, batchEnd-i)
		args := make([]interface{}, 0, (batchEnd-i)*3)
		count := 0

		for j := i; j < batchEnd; j++ {
			bookID := rand.Intn(numBooks) + 1
			tagID := rand.Intn(numTags) + 1
			key := fmt.Sprintf("%d-%d", bookID, tagID)

			if inserted[key] {
				continue
			}
			inserted[key] = true

			values = append(values, "(?, ?, ?)")
			createdAt := baseTime.Add(time.Duration(rand.Intn(5*365*24)) * time.Hour)
			args = append(args, bookID, tagID, createdAt)
			count++
		}

		if len(values) > 0 {
			query := "INSERT IGNORE INTO book_tags (book_id, tag_id, created_at) VALUES " + strings.Join(values, ",")
			if _, err := db.Exec(query, args...); err != nil {
				log.Printf("Worker %d: Error inserting book_tags: %v", workerID, err)
			}
		}

		atomic.AddInt64(&totalInserted, int64(batchEnd-i))

		// Clear map periodically to avoid memory issues
		if len(inserted) > 100000 {
			inserted = make(map[string]bool)
		}
	}
}

func seedAuthorTags(db *sql.DB, count, numAuthors, numTags int) {
	log.Printf("Seeding %d author_tags...", count)

	baseTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.Local)
	inserted := make(map[string]bool)

	for i := 0; i < count; i += bulkSize {
		end := i + bulkSize
		if end > count {
			end = count
		}

		values := make([]string, 0, end-i)
		args := make([]interface{}, 0, (end-i)*3)

		for j := i; j < end; j++ {
			authorID := rand.Intn(numAuthors) + 1
			tagID := rand.Intn(numTags) + 1
			key := fmt.Sprintf("%d-%d", authorID, tagID)

			if inserted[key] {
				continue
			}
			inserted[key] = true

			values = append(values, "(?, ?, ?)")
			createdAt := baseTime.Add(time.Duration(rand.Intn(5*365*24)) * time.Hour)
			args = append(args, authorID, tagID, createdAt)
		}

		if len(values) > 0 {
			query := "INSERT IGNORE INTO author_tags (author_id, tag_id, created_at) VALUES " + strings.Join(values, ",")
			if _, err := db.Exec(query, args...); err != nil {
				log.Printf("Error inserting author_tags: %v", err)
			}
		}

		atomic.AddInt64(&totalInserted, int64(end-i))
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
