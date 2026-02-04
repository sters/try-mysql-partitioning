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

func AuthorsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		listAuthors(w, r)
	case http.MethodPost:
		createAuthor(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func AuthorHandler(w http.ResponseWriter, r *http.Request) {
	id, err := extractID(r.URL.Path, "/authors/")
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	// Check if this is a tags sub-resource
	if strings.Contains(r.URL.Path, "/tags") {
		handleAuthorTags(w, r, id)
		return
	}

	switch r.Method {
	case http.MethodGet:
		getAuthor(w, r, id)
	case http.MethodPut:
		updateAuthor(w, r, id)
	case http.MethodDelete:
		deleteAuthor(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func listAuthors(w http.ResponseWriter, r *http.Request) {
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

	rows, err := db.DB.Query("SELECT id, name, created_at FROM authors ORDER BY id LIMIT ? OFFSET ?", limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var authors []models.Author
	for rows.Next() {
		var a models.Author
		if err := rows.Scan(&a.ID, &a.Name, &a.CreatedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		authors = append(authors, a)
	}

	respondJSON(w, authors)
}

func getAuthor(w http.ResponseWriter, r *http.Request, id int64) {
	var a models.Author
	err := db.DB.QueryRow("SELECT id, name, created_at FROM authors WHERE id = ?", id).
		Scan(&a.ID, &a.Name, &a.CreatedAt)
	if err == sql.ErrNoRows {
		http.Error(w, "Author not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, a)
}

func createAuthor(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	result, err := db.DB.Exec("INSERT INTO authors (name) VALUES (?)", input.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	id, _ := result.LastInsertId()
	var a models.Author
	db.DB.QueryRow("SELECT id, name, created_at FROM authors WHERE id = ?", id).
		Scan(&a.ID, &a.Name, &a.CreatedAt)

	w.WriteHeader(http.StatusCreated)
	respondJSON(w, a)
}

func updateAuthor(w http.ResponseWriter, r *http.Request, id int64) {
	var input struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	result, err := db.DB.Exec("UPDATE authors SET name = ? WHERE id = ?", input.Name, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Author not found", http.StatusNotFound)
		return
	}

	getAuthor(w, r, id)
}

func deleteAuthor(w http.ResponseWriter, r *http.Request, id int64) {
	result, err := db.DB.Exec("DELETE FROM authors WHERE id = ?", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Author not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleAuthorTags(w http.ResponseWriter, r *http.Request, authorID int64) {
	switch r.Method {
	case http.MethodGet:
		listAuthorTags(w, authorID)
	case http.MethodPost:
		addAuthorTag(w, r, authorID)
	case http.MethodDelete:
		tagID, err := extractTagID(r.URL.Path)
		if err != nil {
			http.Error(w, "Invalid tag ID", http.StatusBadRequest)
			return
		}
		removeAuthorTag(w, authorID, tagID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func listAuthorTags(w http.ResponseWriter, authorID int64) {
	rows, err := db.DB.Query(`
		SELECT t.id, t.name FROM tags t
		INNER JOIN author_tags at ON t.id = at.tag_id
		WHERE at.author_id = ?
	`, authorID)
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

func addAuthorTag(w http.ResponseWriter, r *http.Request, authorID int64) {
	var input struct {
		TagID int64 `json:"tag_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	_, err := db.DB.Exec("INSERT INTO author_tags (author_id, tag_id) VALUES (?, ?)", authorID, input.TagID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func removeAuthorTag(w http.ResponseWriter, authorID, tagID int64) {
	result, err := db.DB.Exec("DELETE FROM author_tags WHERE author_id = ? AND tag_id = ?", authorID, tagID)
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
