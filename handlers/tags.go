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

func TagsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		listTags(w, r)
	case http.MethodPost:
		createTag(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func TagHandler(w http.ResponseWriter, r *http.Request) {
	id, err := extractID(r.URL.Path, "/tags/")
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		getTag(w, r, id)
	case http.MethodDelete:
		deleteTag(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func listTags(w http.ResponseWriter, r *http.Request) {
	rows, err := db.DB.Query("SELECT id, name FROM tags ORDER BY id")
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

func getTag(w http.ResponseWriter, r *http.Request, id int64) {
	var t models.Tag
	err := db.DB.QueryRow("SELECT id, name FROM tags WHERE id = ?", id).
		Scan(&t.ID, &t.Name)
	if err == sql.ErrNoRows {
		http.Error(w, "Tag not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, t)
}

func createTag(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	result, err := db.DB.Exec("INSERT INTO tags (name) VALUES (?)", input.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	id, _ := result.LastInsertId()
	var t models.Tag
	db.DB.QueryRow("SELECT id, name FROM tags WHERE id = ?", id).
		Scan(&t.ID, &t.Name)

	w.WriteHeader(http.StatusCreated)
	respondJSON(w, t)
}

func deleteTag(w http.ResponseWriter, r *http.Request, id int64) {
	result, err := db.DB.Exec("DELETE FROM tags WHERE id = ?", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Tag not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper functions

func extractID(path, prefix string) (int64, error) {
	trimmed := strings.TrimPrefix(path, prefix)
	// Handle sub-resources like /authors/1/tags
	if idx := strings.Index(trimmed, "/"); idx != -1 {
		trimmed = trimmed[:idx]
	}
	return strconv.ParseInt(trimmed, 10, 64)
}

func extractTagID(path string) (int64, error) {
	parts := strings.Split(path, "/tags/")
	if len(parts) < 2 {
		return 0, strconv.ErrSyntax
	}
	return strconv.ParseInt(parts[1], 10, 64)
}

func respondJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
