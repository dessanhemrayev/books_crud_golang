package database_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"books_crud_golang/internal/database"
	"books_crud_golang/internal/handlers"
	"books_crud_golang/internal/models"

	"github.com/jmoiron/sqlx"
)

const (
	getAllBooksQuery = "SELECT id, title, author, published_date, is_available, created_at FROM books ORDER BY created_at DESC;"
	getBookQuery     = "SELECT id, title, author, published_date, is_available, created_at FROM books WHERE id = $1"
	createBookQuery  = "INSERT INTO books (title, author, published_date, is_available, created_at) VALUES ($1, $2, $3, $4, $5) RETURNING id, title, author, published_date, is_available, created_at"
	deleteBookQuery  = "DELETE FROM books WHERE id = $1"
)

func TestBookStoreGetAllBooksIncludesPublishedDate(t *testing.T) {
	publishedFirst := time.Date(2024, time.March, 12, 0, 0, 0, 0, time.UTC)
	publishedSecond := time.Date(1999, time.December, 31, 0, 0, 0, 0, time.UTC)
	createdFirst := time.Date(2026, time.September, 21, 12, 0, 0, 0, time.UTC)
	createdSecond := createdFirst.Add(-time.Hour)

	store := newBookStore(t, dbStep{
		query:   getAllBooksQuery,
		columns: bookColumns(),
		rows: [][]driver.Value{
			{int64(2), "Second", "Author B", publishedFirst, true, createdFirst},
			{int64(1), "First", "Author A", publishedSecond, false, createdSecond},
		},
	})

	books, err := store.GetAllBooks()
	if err != nil {
		t.Fatalf("GetAllBooks() error = %v", err)
	}
	if len(books) != 2 {
		t.Fatalf("GetAllBooks() returned %d books, want 2", len(books))
	}
	if !books[0].PublishedDate.Equal(publishedFirst) || !books[1].PublishedDate.Equal(publishedSecond) {
		t.Fatalf("GetAllBooks() published dates = [%v, %v], want [%v, %v]", books[0].PublishedDate, books[1].PublishedDate, publishedFirst, publishedSecond)
	}
}

func TestBookStoreGetBookByID(t *testing.T) {
	published := time.Date(1965, time.August, 1, 0, 0, 0, 0, time.UTC)
	created := time.Date(2026, time.September, 21, 12, 0, 0, 0, time.UTC)
	databaseFailure := errors.New("connection lost")

	t.Run("returns book including published date", func(t *testing.T) {
		store := newBookStore(t, dbStep{
			query:   getBookQuery,
			args:    []driver.Value{int64(7)},
			columns: bookColumns(),
			rows:    [][]driver.Value{{int64(7), "Dune", "Frank Herbert", published, true, created}},
		})

		book, err := store.GetBookByID(7)
		if err != nil {
			t.Fatalf("GetBookByID() error = %v", err)
		}
		if book.ID != 7 || !book.PublishedDate.Equal(published) {
			t.Fatalf("GetBookByID() = %+v, want ID 7 and published date %v", book, published)
		}
	})

	t.Run("maps an empty result to ErrBookNotFound", func(t *testing.T) {
		store := newBookStore(t, dbStep{
			query:   getBookQuery,
			args:    []driver.Value{int64(404)},
			columns: bookColumns(),
		})

		book, err := store.GetBookByID(404)
		if book != nil {
			t.Fatalf("GetBookByID() book = %+v, want nil", book)
		}
		if !errors.Is(err, database.ErrBookNotFound) {
			t.Fatalf("GetBookByID() error = %v, want ErrBookNotFound", err)
		}
	})

	t.Run("preserves unexpected database errors", func(t *testing.T) {
		store := newBookStore(t, dbStep{
			query: getBookQuery,
			args:  []driver.Value{int64(8)},
			err:   databaseFailure,
		})

		_, err := store.GetBookByID(8)
		if !errors.Is(err, databaseFailure) {
			t.Fatalf("GetBookByID() error = %v, want %v", err, databaseFailure)
		}
		if errors.Is(err, database.ErrBookNotFound) {
			t.Fatalf("GetBookByID() error = %v, must not be ErrBookNotFound", err)
		}
	})
}

func TestBookStoreCreateBookPersistsAndReturnsPublishedDate(t *testing.T) {
	published := time.Date(2020, time.January, 15, 0, 0, 0, 0, time.UTC)
	created := time.Date(2026, time.September, 21, 12, 0, 0, 0, time.UTC)
	store := newBookStore(t, dbStep{
		query: createBookQuery,
		args: []driver.Value{
			"The Test Book",
			"Test Author",
			published,
			true,
			anyTime{},
		},
		columns: bookColumns(),
		rows:    [][]driver.Value{{int64(9), "The Test Book", "Test Author", published, true, created}},
	})

	book, err := store.CreateBook(&models.CreateBookInput{
		Title:         "The Test Book",
		Author:        "Test Author",
		PublishedDate: published,
		IsAvailable:   true,
	})
	if err != nil {
		t.Fatalf("CreateBook() error = %v", err)
	}
	if book.ID != 9 || book.Title != "The Test Book" || book.Author != "Test Author" || !book.PublishedDate.Equal(published) || !book.IsAvailable || !book.CreatedAt.Equal(created) {
		t.Fatalf("CreateBook() = %+v, want all returned fields populated", book)
	}
}

func TestBookStoreDeleteBook(t *testing.T) {
	databaseFailure := errors.New("delete failed")

	tests := []struct {
		name    string
		step    dbStep
		wantErr error
	}{
		{
			name: "deletes an existing book",
			step: dbStep{query: deleteBookQuery, args: []driver.Value{int64(5)}, result: driver.RowsAffected(1)},
		},
		{
			name:    "returns ErrBookNotFound when no row is deleted",
			step:    dbStep{query: deleteBookQuery, args: []driver.Value{int64(5)}, result: driver.RowsAffected(0)},
			wantErr: database.ErrBookNotFound,
		},
		{
			name:    "preserves a database error",
			step:    dbStep{query: deleteBookQuery, args: []driver.Value{int64(5)}, err: databaseFailure},
			wantErr: databaseFailure,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newBookStore(t, tt.step)
			err := store.DeleteBook(5)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("DeleteBook() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandlersDifferentiateMissingBooksFromDatabaseFailures(t *testing.T) {
	databaseFailure := errors.New("database unavailable")

	tests := []struct {
		name       string
		method     string
		body       string
		step       dbStep
		invoke     func(*handlers.Handlers, http.ResponseWriter, *http.Request)
		wantStatus int
		wantBody   string
	}{
		{
			name:       "get missing book",
			method:     http.MethodGet,
			step:       emptyBookStep(12),
			invoke:     (*handlers.Handlers).GetBook,
			wantStatus: http.StatusNotFound,
			wantBody:   `{"error":"Book not found"}`,
		},
		{
			name:       "get database failure",
			method:     http.MethodGet,
			step:       failingGetStep(12, databaseFailure),
			invoke:     (*handlers.Handlers).GetBook,
			wantStatus: http.StatusInternalServerError,
			wantBody:   `{"error":"Failed to retrieve book"}`,
		},
		{
			name:       "update missing book",
			method:     http.MethodPut,
			body:       `{"title":"Revised"}`,
			step:       emptyBookStep(12),
			invoke:     (*handlers.Handlers).UpdateBook,
			wantStatus: http.StatusNotFound,
			wantBody:   `{"error":"Book not found"}`,
		},
		{
			name:       "update database failure",
			method:     http.MethodPut,
			body:       `{"title":"Revised"}`,
			step:       failingGetStep(12, databaseFailure),
			invoke:     (*handlers.Handlers).UpdateBook,
			wantStatus: http.StatusInternalServerError,
			wantBody:   `{"error":"Failed to update book"}`,
		},
		{
			name:       "delete missing book",
			method:     http.MethodDelete,
			step:       dbStep{query: deleteBookQuery, args: []driver.Value{int64(12)}, result: driver.RowsAffected(0)},
			invoke:     (*handlers.Handlers).DeleteBook,
			wantStatus: http.StatusNotFound,
			wantBody:   `{"error":"Book not found"}`,
		},
		{
			name:       "delete database failure",
			method:     http.MethodDelete,
			step:       dbStep{query: deleteBookQuery, args: []driver.Value{int64(12)}, err: databaseFailure},
			invoke:     (*handlers.Handlers).DeleteBook,
			wantStatus: http.StatusInternalServerError,
			wantBody:   `{"error":"Failed to delete book"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newBookStore(t, tt.step)
			handler := handlers.NewHandlers(store)
			request := httptest.NewRequest(tt.method, "/books/12", strings.NewReader(tt.body))
			response := httptest.NewRecorder()

			tt.invoke(handler, response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, tt.wantStatus, response.Body.String())
			}
			if got := strings.TrimSpace(response.Body.String()); got != tt.wantBody {
				t.Fatalf("body = %q, want %q", got, tt.wantBody)
			}
			if got := response.Header().Get("Content-Type"); got != "application/json" {
				t.Fatalf("Content-Type = %q, want application/json", got)
			}
		})
	}
}

func emptyBookStep(id int64) dbStep {
	return dbStep{query: getBookQuery, args: []driver.Value{id}, columns: bookColumns()}
}

func failingGetStep(id int64, err error) dbStep {
	return dbStep{query: getBookQuery, args: []driver.Value{id}, err: err}
}

func bookColumns() []string {
	return []string{"id", "title", "author", "published_date", "is_available", "created_at"}
}

type anyTime struct{}

type dbStep struct {
	query   string
	args    []driver.Value
	columns []string
	rows    [][]driver.Value
	result  driver.Result
	err     error
}

type dbScript struct {
	t     *testing.T
	mu    sync.Mutex
	steps []dbStep
	next  int
}

func newBookStore(t *testing.T, steps ...dbStep) *database.BookStore {
	t.Helper()
	script := &dbScript{t: t, steps: steps}
	driverName := fmt.Sprintf("books-test-%d", atomic.AddUint64(&driverSequence, 1))
	sql.Register(driverName, &scriptedDriver{script: script})
	db, err := sql.Open(driverName, "")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("database close error = %v", err)
		}
		script.assertComplete()
	})
	return database.NewBookStore(sqlx.NewDb(db, driverName))
}

var driverSequence uint64

type scriptedDriver struct {
	script *dbScript
}

func (d *scriptedDriver) Open(string) (driver.Conn, error) {
	return &scriptedConn{script: d.script}, nil
}

type scriptedConn struct {
	script *dbScript
}

func (c *scriptedConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepared statements are not supported by the test driver")
}

func (c *scriptedConn) Close() error { return nil }
func (c *scriptedConn) Begin() (driver.Tx, error) {
	return nil, errors.New("transactions are not supported by the test driver")
}

func (c *scriptedConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	step := c.script.take(query, args)
	if step.err != nil {
		return nil, step.err
	}
	return &scriptedRows{columns: step.columns, rows: step.rows}, nil
}

func (c *scriptedConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	step := c.script.take(query, args)
	if step.err != nil {
		return nil, step.err
	}
	if step.result == nil {
		c.script.t.Fatalf("query %q did not define an exec result", query)
	}
	return step.result, nil
}

func (s *dbScript) take(query string, args []driver.NamedValue) dbStep {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.next >= len(s.steps) {
		s.t.Fatalf("unexpected database call: %q", query)
	}
	step := s.steps[s.next]
	s.next++
	if query != step.query {
		s.t.Fatalf("database query = %q, want %q", query, step.query)
	}
	if len(args) != len(step.args) {
		s.t.Fatalf("database argument count = %d, want %d", len(args), len(step.args))
	}
	for i, arg := range args {
		if _, ok := step.args[i].(anyTime); ok {
			if _, ok := arg.Value.(time.Time); !ok {
				s.t.Fatalf("database argument %d = %T, want time.Time", i+1, arg.Value)
			}
			continue
		}
		if !reflect.DeepEqual(arg.Value, step.args[i]) {
			s.t.Fatalf("database argument %d = %#v, want %#v", i+1, arg.Value, step.args[i])
		}
	}
	return step
}

func (s *dbScript) assertComplete() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.next != len(s.steps) {
		s.t.Errorf("used %d database steps, want %d", s.next, len(s.steps))
	}
}

type scriptedRows struct {
	columns []string
	rows    [][]driver.Value
	next    int
}

func (r *scriptedRows) Columns() []string { return r.columns }
func (r *scriptedRows) Close() error      { return nil }

func (r *scriptedRows) Next(dest []driver.Value) error {
	if r.next >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.next])
	r.next++
	return nil
}
