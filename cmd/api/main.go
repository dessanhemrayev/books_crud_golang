package main

import (
	"books_crud_golang/internal/database"
	"books_crud_golang/internal/handlers"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	log.Printf("Database URL: %s", databaseURL)
	if databaseURL == "" {
		fmt.Println("DATABASE_URL environment variable is not set")
		databaseURL = "postgres://books_user:password@localhost:5433/books_db?sslmode=disable"
	}
	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		fmt.Println("SERVER_PORT environment variable is not set")
		serverPort = "8080"
	}
	log.Printf("Starting server on port %s...", serverPort)
	db, err := database.Connect(databaseURL)
	if err != nil {
		fmt.Println("Failed to connect to database:", err)
		return
	}
	log.Println("Successfully connected to database")
	
	defer db.Close()
	log.Printf("Server is running on port %s", serverPort)

	taskStore := database.NewBookStore(db)
	handler := handlers.NewHandlers(taskStore)
	mux := http.NewServeMux()

	mux.Handle("/books", methodHandler(handler.GetAllBooks, "GET"))
	mux.Handle("/books/create", methodHandler(handler.CreateBook, "POST"))

	mux.Handle("/books/", bookIDHandler(handler))

	loggedMux := logginMiddleware(mux)

	serverAddr := ":" + serverPort
	log.Printf("Server is running on %s", serverAddr)
	err = http.ListenAndServe(serverAddr, loggedMux)
	if err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func methodHandler(handlerFunc http.HandlerFunc, allowedMethod string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != allowedMethod {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		handlerFunc(w, r)
	}
}

func bookIDHandler(handler *handlers.Handlers) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handler.GetBook(w, r)
		case http.MethodPost:
			handler.CreateBook(w, r)
		case http.MethodPut:
			handler.UpdateBook(w, r)
		case http.MethodDelete:
			handler.DeleteBook(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}
func logginMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s",r.Method, r.URL, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}