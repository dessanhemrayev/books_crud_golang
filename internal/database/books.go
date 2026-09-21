package database

import (
	"books_crud_golang/internal/models"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
)

type BookStore struct {
	db *sqlx.DB
}

func NewBookStore(db *sqlx.DB) *BookStore {
	return &BookStore{db: db}
}

func (store *BookStore) GetAllBooks() ([]models.Book, error) {
	var books []models.Book
	query := "SELECT id, title, author, is_available, created_at FROM books ORDER BY created_at DESC;"

	err := store.db.Select(&books, query)
	log.Printf("Retrieved %d books from the database", len(books))
	if err != nil {
		return nil, err
	}

	return books, nil
}

func (store *BookStore) GetBookByID(id int) (*models.Book, error) {
	var book models.Book
	query := "SELECT id, title, author, is_available, created_at FROM books WHERE id = $1"
	err := store.db.Get(&book, query, id)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("Book with id %d not found", id)
	}
	if err != nil {
		return nil, err
	}
	return &book, nil
}

func (store *BookStore) CreateBook(input *models.CreateBookInput) (*models.Book, error) {
	query := `INSERT INTO books (title, author, is_available, created_at) VALUES ($1, $2, $3, $4) RETURNING id, created_at`

	now := time.Now()

	var book models.Book
	err := store.db.QueryRowx(query, input.Title, input.Author, input.IsAvailable, now).StructScan(&book)
	if err != nil {
		return nil, err
	}
	return &book, nil
}

func (store *BookStore) UpdateBook(id int, input *models.UpdateBookInput) (*models.Book, error) {
	book, err := store.GetBookByID(id)
	if err != nil {
		return nil, err
	}
	if input.Title != nil {
		book.Title = *input.Title
	}
	if input.Author != nil {
		book.Author = *input.Author
	}
	if input.PublishedDate != nil {
		book.PublishedDate = *input.PublishedDate
	}
	if input.IsAvailable != nil {
		book.IsAvailable = *input.IsAvailable
	}
	book.UpdatedAt = time.Now()

	query := `UPDATE books SET title = $1, author = $2, published_date = $3, is_available = $4, updated_at = $5 WHERE id = $6 returning id, title, author, published_date, is_available, updated_at`
	var updatedBook models.Book
	err = store.db.QueryRowx(query, book.Title, book.Author, book.PublishedDate, book.IsAvailable, book.UpdatedAt, id).StructScan(&updatedBook)
	if err != nil {
		return nil, err
	}
	return &updatedBook, nil

}

func (store *BookStore) DeleteBook(id int) error {
	query := "DELETE FROM books WHERE id = $1"
	result, err := store.db.Exec(query, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("Book with id %d not found", id)
	}
	return nil
}

func (store *BookStore) AddFavoriteBook(bookID int, userID int) (*models.FavoriteBook, error) {
	book, err := store.GetBookByID(bookID)
	if err != nil {
		return nil, err
	}

	query := `INSERT INTO favorite_books (book_id, user_id, created_at) VALUES ($1, $2, $3) RETURNING id, book_id, user_id, created_at`
	now := time.Now()
	var favoriteBook models.FavoriteBook
	err = store.db.QueryRowx(query, book.ID, userID, now).StructScan(&favoriteBook)
	if err != nil {
		return nil, err
	}
	return &favoriteBook, nil
}