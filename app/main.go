package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type Document struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Attribute struct {
	ID         int       `json:"id"`
	DocumentID int       `json:"document_id"`
	AttrKey    string    `json:"attr_key"`
	AttrValue  string    `json:"attr_value"`
	CreatedAt  time.Time `json:"created_at"`
}

var db *sql.DB

func main() {
	var err error
	
	// Get database connection parameters from environment variables
	dbHost := getEnv("DB_HOST", "mysql")
	dbPort := getEnv("DB_PORT", "3306")
	dbUser := getEnv("DB_USER", "root")
	dbPassword := getEnv("DB_PASSWORD", "rootpassword")
	dbName := getEnv("DB_NAME", "docmanager")
	
	// Wait for MySQL to be ready
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", dbUser, dbPassword, dbHost, dbPort, dbName)
	for i := 0; i < 30; i++ {
		db, err = sql.Open("mysql", dsn)
		if err == nil {
			err = db.Ping()
			if err == nil {
				break
			}
		}
		log.Printf("Waiting for database... (%d/30)", i+1)
		time.Sleep(2 * time.Second)
	}
	
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()
	
	log.Println("Connected to database successfully")
	
	// Setup routes
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/documents", documentsHandler)
	http.HandleFunc("/documents/", documentHandler)
	http.HandleFunc("/attributes", attributesHandler)
	http.HandleFunc("/attributes/", attributeHandler)
	
	port := getEnv("PORT", "8080")
	log.Printf("Server starting on port %s...", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `
		<html>
		<head><title>Document Management System</title></head>
		<body>
			<h1>Document Management System with MySQL Partitioning</h1>
			<h2>API Endpoints:</h2>
			<ul>
				<li>GET /documents - List all documents</li>
				<li>GET /documents/{id} - Get a document by ID</li>
				<li>POST /documents - Create a new document</li>
				<li>PUT /documents/{id} - Update a document</li>
				<li>DELETE /documents/{id} - Delete a document</li>
			</ul>
			<ul>
				<li>GET /attributes - List all attributes</li>
				<li>GET /attributes/{id} - Get an attribute by ID</li>
				<li>POST /attributes - Create a new attribute</li>
				<li>PUT /attributes/{id} - Update an attribute</li>
				<li>DELETE /attributes/{id} - Delete an attribute</li>
			</ul>
		</body>
		</html>
	`)
}

func documentsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		listDocuments(w, r)
	case "POST":
		createDocument(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func documentHandler(w http.ResponseWriter, r *http.Request) {
	id := extractID(r.URL.Path, "/documents/")
	if id == 0 {
		http.Error(w, "Invalid document ID", http.StatusBadRequest)
		return
	}
	
	switch r.Method {
	case "GET":
		getDocument(w, r, id)
	case "PUT":
		updateDocument(w, r, id)
	case "DELETE":
		deleteDocument(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func listDocuments(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, title, content, created_at, updated_at FROM documents ORDER BY created_at DESC")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	var documents []Document
	for rows.Next() {
		var doc Document
		if err := rows.Scan(&doc.ID, &doc.Title, &doc.Content, &doc.CreatedAt, &doc.UpdatedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		documents = append(documents, doc)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(documents)
}

func getDocument(w http.ResponseWriter, r *http.Request, id int) {
	var doc Document
	err := db.QueryRow("SELECT id, title, content, created_at, updated_at FROM documents WHERE id = ?", id).
		Scan(&doc.ID, &doc.Title, &doc.Content, &doc.CreatedAt, &doc.UpdatedAt)
	
	if err == sql.ErrNoRows {
		http.Error(w, "Document not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}

func createDocument(w http.ResponseWriter, r *http.Request) {
	var doc Document
	if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	// If no created_at provided, use current time
	if doc.CreatedAt.IsZero() {
		doc.CreatedAt = time.Now()
	}
	
	result, err := db.Exec("INSERT INTO documents (title, content, created_at) VALUES (?, ?, ?)",
		doc.Title, doc.Content, doc.CreatedAt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	id, _ := result.LastInsertId()
	doc.ID = int(id)
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(doc)
}

func updateDocument(w http.ResponseWriter, r *http.Request, id int) {
	var doc Document
	if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	result, err := db.Exec("UPDATE documents SET title = ?, content = ? WHERE id = ?",
		doc.Title, doc.Content, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Document not found", http.StatusNotFound)
		return
	}
	
	doc.ID = id
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}

func deleteDocument(w http.ResponseWriter, r *http.Request, id int) {
	result, err := db.Exec("DELETE FROM documents WHERE id = ?", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Document not found", http.StatusNotFound)
		return
	}
	
	w.WriteHeader(http.StatusNoContent)
}

func attributesHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		listAttributes(w, r)
	case "POST":
		createAttribute(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func attributeHandler(w http.ResponseWriter, r *http.Request) {
	id := extractID(r.URL.Path, "/attributes/")
	if id == 0 {
		http.Error(w, "Invalid attribute ID", http.StatusBadRequest)
		return
	}
	
	switch r.Method {
	case "GET":
		getAttribute(w, r, id)
	case "PUT":
		updateAttribute(w, r, id)
	case "DELETE":
		deleteAttribute(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func listAttributes(w http.ResponseWriter, r *http.Request) {
	// Check if filtering by document_id
	documentID := r.URL.Query().Get("document_id")
	
	var rows *sql.Rows
	var err error
	
	if documentID != "" {
		rows, err = db.Query("SELECT id, document_id, attr_key, attr_value, created_at FROM attributes WHERE document_id = ? ORDER BY id", documentID)
	} else {
		rows, err = db.Query("SELECT id, document_id, attr_key, attr_value, created_at FROM attributes ORDER BY id")
	}
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	var attributes []Attribute
	for rows.Next() {
		var attr Attribute
		if err := rows.Scan(&attr.ID, &attr.DocumentID, &attr.AttrKey, &attr.AttrValue, &attr.CreatedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		attributes = append(attributes, attr)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(attributes)
}

func getAttribute(w http.ResponseWriter, r *http.Request, id int) {
	var attr Attribute
	err := db.QueryRow("SELECT id, document_id, attr_key, attr_value, created_at FROM attributes WHERE id = ?", id).
		Scan(&attr.ID, &attr.DocumentID, &attr.AttrKey, &attr.AttrValue, &attr.CreatedAt)
	
	if err == sql.ErrNoRows {
		http.Error(w, "Attribute not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(attr)
}

func createAttribute(w http.ResponseWriter, r *http.Request) {
	var attr Attribute
	if err := json.NewDecoder(r.Body).Decode(&attr); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	result, err := db.Exec("INSERT INTO attributes (document_id, attr_key, attr_value) VALUES (?, ?, ?)",
		attr.DocumentID, attr.AttrKey, attr.AttrValue)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	id, _ := result.LastInsertId()
	attr.ID = int(id)
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(attr)
}

func updateAttribute(w http.ResponseWriter, r *http.Request, id int) {
	var attr Attribute
	if err := json.NewDecoder(r.Body).Decode(&attr); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	result, err := db.Exec("UPDATE attributes SET document_id = ?, attr_key = ?, attr_value = ? WHERE id = ?",
		attr.DocumentID, attr.AttrKey, attr.AttrValue, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Attribute not found", http.StatusNotFound)
		return
	}
	
	attr.ID = id
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(attr)
}

func deleteAttribute(w http.ResponseWriter, r *http.Request, id int) {
	result, err := db.Exec("DELETE FROM attributes WHERE id = ?", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Attribute not found", http.StatusNotFound)
		return
	}
	
	w.WriteHeader(http.StatusNoContent)
}

func extractID(path, prefix string) int {
	idStr := path[len(prefix):]
	id, _ := strconv.Atoi(idStr)
	return id
}
