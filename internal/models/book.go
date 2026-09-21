package models


import "time"


type Book struct {
	ID int `json:"id" db:"id"`
	Title string `json:"title" db:"title"`
	Author string `json:"author" db:"author"`
	PublishedDate time.Time `json:"published_date" db:"published_date"`
	IsAvailable bool `json:"is_available" db:"is_available"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}


type CreateBookInput struct {
	Title string `json:"title"`
	Author string `json:"author"`
	PublishedDate time.Time `json:"published_date"`
	IsAvailable bool `json:"is_available"`
}

type UpdateBookInput struct {
	Title *string `json:"title"`
	Author *string `json:"author"`
	PublishedDate *time.Time `json:"published_date"`
	IsAvailable *bool `json:"is_available"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type FavoriteBook struct {
	ID int `json:"id" db:"id"`
	BookID int `json:"book_id" db:"book_id"`
	UserID int `json:"user_id" db:"user_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}