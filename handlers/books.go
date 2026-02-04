package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/sters/try-mysql-partitioning/db"
	"github.com/sters/try-mysql-partitioning/models"
)

func BooksHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		listBooks(w, r)
	case http.MethodPost:
		createBook(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func BookHandler(w http.ResponseWriter, r *http.Request) {
	id, err := extractID(r.URL.Path, "/books/")
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	// Check if this is a tags sub-resource
	if strings.Contains(r.URL.Path, "/tags") {
		handleBookTags(w, r, id)
		return
	}

	switch r.Method {
	case http.MethodGet:
		getBook(w, r, id)
	case http.MethodPut:
		updateBook(w, r, id)
	case http.MethodDelete:
		deleteBook(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func listBooks(w http.ResponseWriter, r *http.Request) {
	limit := 100
	offset := 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	rows, err := db.DB.Query("SELECT id, title, author_id, created_at FROM books ORDER BY id LIMIT ? OFFSET ?", limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var books []models.Book
	for rows.Next() {
		var b models.Book
		if err := rows.Scan(&b.ID, &b.Title, &b.AuthorID, &b.CreatedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		books = append(books, b)
	}

	respondJSON(w, books)
}

func getBook(w http.ResponseWriter, r *http.Request, id int64) {
	var b models.Book
	err := db.DB.QueryRow("SELECT id, title, author_id, created_at FROM books WHERE id = ?", id).
		Scan(&b.ID, &b.Title, &b.AuthorID, &b.CreatedAt)
	if err == sql.ErrNoRows {
		http.Error(w, "Book not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, b)
}

func createBook(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title    string `json:"title"`
		AuthorID int64  `json:"author_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	result, err := db.DB.Exec("INSERT INTO books (title, author_id) VALUES (?, ?)", input.Title, input.AuthorID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	id, _ := result.LastInsertId()
	var b models.Book
	db.DB.QueryRow("SELECT id, title, author_id, created_at FROM books WHERE id = ?", id).
		Scan(&b.ID, &b.Title, &b.AuthorID, &b.CreatedAt)

	w.WriteHeader(http.StatusCreated)
	respondJSON(w, b)
}

func updateBook(w http.ResponseWriter, r *http.Request, id int64) {
	var input struct {
		Title    string `json:"title"`
		AuthorID int64  `json:"author_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	result, err := db.DB.Exec("UPDATE books SET title = ?, author_id = ? WHERE id = ?", input.Title, input.AuthorID, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Book not found", http.StatusNotFound)
		return
	}

	getBook(w, r, id)
}

func deleteBook(w http.ResponseWriter, r *http.Request, id int64) {
	result, err := db.DB.Exec("DELETE FROM books WHERE id = ?", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Book not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleBookTags(w http.ResponseWriter, r *http.Request, bookID int64) {
	switch r.Method {
	case http.MethodGet:
		listBookTags(w, bookID)
	case http.MethodPost:
		addBookTag(w, r, bookID)
	case http.MethodDelete:
		tagID, err := extractTagID(r.URL.Path)
		if err != nil {
			http.Error(w, "Invalid tag ID", http.StatusBadRequest)
			return
		}
		removeBookTag(w, bookID, tagID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func listBookTags(w http.ResponseWriter, bookID int64) {
	rows, err := db.DB.Query(`
		SELECT t.id, t.name FROM tags t
		INNER JOIN book_tags bt ON t.id = bt.tag_id
		WHERE bt.book_id = ?
	`, bookID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		var t models.Tag
		if err := rows.Scan(&t.ID, &t.Name); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tags = append(tags, t)
	}

	respondJSON(w, tags)
}

func addBookTag(w http.ResponseWriter, r *http.Request, bookID int64) {
	var input struct {
		TagID int64 `json:"tag_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	_, err := db.DB.Exec("INSERT INTO book_tags (book_id, tag_id) VALUES (?, ?)", bookID, input.TagID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func removeBookTag(w http.ResponseWriter, bookID, tagID int64) {
	result, err := db.DB.Exec("DELETE FROM book_tags WHERE book_id = ? AND tag_id = ?", bookID, tagID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Tag association not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
