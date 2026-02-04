package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/sters/try-mysql-partitioning/db"
	"github.com/sters/try-mysql-partitioning/handlers"
)

func main() {
	if err := db.Init(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	mux := http.NewServeMux()

	// Authors routes
	mux.HandleFunc("/authors", handlers.AuthorsHandler)
	mux.HandleFunc("/authors/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/authors/" {
			handlers.AuthorsHandler(w, r)
			return
		}
		handlers.AuthorHandler(w, r)
	})

	// Books routes
	mux.HandleFunc("/books", handlers.BooksHandler)
	mux.HandleFunc("/books/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/books/" {
			handlers.BooksHandler(w, r)
			return
		}
		handlers.BookHandler(w, r)
	})

	// Tags routes
	mux.HandleFunc("/tags", handlers.TagsHandler)
	mux.HandleFunc("/tags/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/tags/" {
			handlers.TagsHandler(w, r)
			return
		}
		handlers.TagHandler(w, r)
	})

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := db.DB.Ping(); err != nil {
			http.Error(w, "Database unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte("OK"))
	})

	// Root
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message":"MySQL Partitioning Experiment API","endpoints":["/authors","/books","/tags","/health"]}`))
	})

	// Simple logging middleware
	handler := loggingMiddleware(mux)

	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/health") {
			log.Printf("%s %s", r.Method, r.URL.Path)
		}
		next.ServeHTTP(w, r)
	})
}
