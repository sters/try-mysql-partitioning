package models

import "time"

type Author struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Book struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	AuthorID  int64     `json:"author_id"`
	CreatedAt time.Time `json:"created_at"`
}

type Tag struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type BookTag struct {
	BookID    int64     `json:"book_id"`
	TagID     int64     `json:"tag_id"`
	CreatedAt time.Time `json:"created_at"`
}

type AuthorTag struct {
	AuthorID  int64     `json:"author_id"`
	TagID     int64     `json:"tag_id"`
	CreatedAt time.Time `json:"created_at"`
}
