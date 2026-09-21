package handlers

import (
	"books_crud_golang/internal/database"
	"books_crud_golang/internal/models"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
)



type Handlers struct {
	store *database.BookStore
}

func NewHandlers(store *database.BookStore) *Handlers {
	return &Handlers{store: store}
}

func respondWithJson(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func respondWithError(w http.ResponseWriter, status int, message string) {
	respondWithJson(w, status, map[string]string{"error": message})
}

func (h *Handlers) GetAllBooks(w http.ResponseWriter, r *http.Request) {
	books, err := h.store.GetAllBooks()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve books")
		return
	}
	respondWithJson(w, http.StatusOK, books)
}

func (h *Handlers) GetBook(w http.ResponseWriter, r *http.Request) {
	// Implementation for getting a book by ID
	bookID, err := getBookIDFromPath(r.URL.Path)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid book ID")
		return
	}
	book, err := h.store.GetBookByID(bookID)

	if errors.Is(err, database.ErrBookNotFound) {
		respondWithError(w, http.StatusNotFound, "Book not found")
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve book")
		return
	}
	respondWithJson(w, http.StatusOK, book)
}

func (h *Handlers) CreateBook(w http.ResponseWriter, r *http.Request) {
	var input models.CreateBookInput
	
	if  err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if strings.TrimSpace(input.Title) == "" || strings.TrimSpace(input.Author) == "" || input.PublishedDate.IsZero() {
		respondWithError(w, http.StatusBadRequest, "Title and Author are required")
		return
	}

	book, err := h.store.CreateBook(&input)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create book")
		return
	}
	respondWithJson(w, http.StatusCreated, book)
}

func (h *Handlers) UpdateBook(w http.ResponseWriter, r *http.Request) {
	bookID, err := getBookIDFromPath(r.URL.Path)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid book ID")
		return
	}

	var input models.UpdateBookInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	book, err := h.store.UpdateBook(bookID, &input)
	if errors.Is(err, database.ErrBookNotFound) {
		respondWithError(w, http.StatusNotFound, "Book not found")
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update book")
		return
	}
	respondWithJson(w, http.StatusOK, book)
}

func (h *Handlers) DeleteBook(w http.ResponseWriter, r *http.Request) {
	bookID, err := getBookIDFromPath(r.URL.Path)

	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid book ID")
		return
	}

	err = h.store.DeleteBook(bookID)
	if err != nil {
		if errors.Is(err, database.ErrBookNotFound) {
			respondWithError(w, http.StatusNotFound, "Book not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Failed to delete book")
		return
	}
	respondWithJson(w, http.StatusOK, map[string]string{"message": "Book deleted successfully"})
}


func getBookIDFromPath(path string) (int, error) {
	pathParts := strings.Split(strings.TrimPrefix(path, "/books/"), "/")
	bookID := pathParts[0]
	return strconv.Atoi(bookID)
}