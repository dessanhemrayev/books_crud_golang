# Books CRUD API (Go)

A simple REST API for managing books, built with Go's standard library (`net/http`) and PostgreSQL. It provides full CRUD functionality for books, containerized PostgreSQL setup via Docker Compose, and automatic database initialization with seed data.

## Features

- Create, read, update, and delete books
- JSON request/response handling with proper HTTP status codes
- Method enforcement and request logging middleware
- PostgreSQL access via [sqlx](https://github.com/jmoiron/sqlx) with a connection pool
- One-command database setup with Docker Compose (`sql/init.sql` creates tables and seeds sample data automatically)
- Partial updates — only the fields provided in the request body are changed

## Tech Stack

| Component  | Technology                                  |
|------------|---------------------------------------------|
| Language   | Go 1.26                                     |
| HTTP       | Standard library (`net/http`)               |
| Database   | PostgreSQL                                  |
| DB access  | [sqlx](https://github.com/jmoiron/sqlx) + [lib/pq](https://github.com/lib/pq) |
| Containers | Docker Compose (PostgreSQL)                 |

## Project Structure

```
books_crud_golang/
├── cmd/
│   └── api/
│       └── main.go            # Entry point: routing, method checks, logging middleware
├── internal/
│   ├── database/
│   │   ├── database.go        # PostgreSQL connection (sqlx) and pool settings
│   │   ├── books.go           # Book data access layer (CRUD queries)
│   │   └── users.go           # User data access layer (in progress)
│   ├── handlers/
│   │   └── handlers.go        # HTTP handlers (JSON encoding/decoding, validation)
│   └── models/
│       └── book.go            # Data models (Book, create/update inputs, FavoriteBook)
├── sql/
│   └── init.sql               # Schema creation + seed data (runs on first DB start)
├── docker-compose.yml           # PostgreSQL + API services (ports 5433 / 8080)
├── Dockerfile                   # Multi-stage build of the API image
├── .dockerignore                # Keeps .env and non-build files out of the image
├── .env.example                 # Environment variable template
└── go.mod
```

## Getting Started

### Prerequisites

- [Go 1.26+](https://go.dev/dl/)
- [Docker](https://www.docker.com/) with Docker Compose (or a local PostgreSQL instance)

### 1. Clone the repository

```bash
git clone https://github.com/dessanhemrayev/books_crud_golang.git
cd books_crud_golang
```

### 2. Configure environment variables

Copy the example file and adjust the values as needed:

```bash
cp .env.example .env
```

`.env` (used by both Docker Compose and the application):

```env
POSTGRES_USER=books_user
POSTGRES_PASSWORD=password
POSTGRES_DB=books_db
DATABASE_URL=postgres://books_user:password@localhost:5433/books_db?sslmode=disable
```

### 3. Start the stack

```bash
docker compose up -d --build
```

This builds the API image from the Dockerfile and starts two containers:

- **postgres** — mapped to **localhost:5433** (data is persisted in the `books_pg_data` volume). On first start, `sql/init.sql` is executed automatically — it creates the `users`, `books`, and `favorite_books` tables and inserts sample data (5 classic books and 3 users).
- **api** — the Books CRUD API, mapped to **localhost:8080**. It waits for the Postgres healthcheck before starting, so no connection race on boot.

### 4. Run the API locally (alternative to the api container)

If you prefer to run the app on the host during development, start only the database:

```bash
docker compose up -d postgres
go run ./cmd/api
```

The `DATABASE_URL` from `.env` points to `localhost:5433`, so the local server connects to the containerized Postgres.

The server starts on port `8080` by default. You should see:

```
Successfully connected to database
Server is running on :8080
```

### 5. Verify it works

```bash
curl http://localhost:8080/books
```

## Configuration


Additional variables used by Docker Compose: `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`.

## API Reference

Base URL: `http://localhost:8080`

All responses are JSON. Errors are returned as:

```json
{ "error": "message" }
```

### Endpoints

| Method   | Endpoint         | Description                        | Success status |
|----------|------------------|------------------------------------|----------------|
| `GET`    | `/books`         | List all books (newest first)      | `200`          |
| `POST`   | `/books/create`  | Create a new book                  | `201`          |
| `GET`    | `/books/{id}`    | Get a book by ID                   | `200`          |
| `PUT`    | `/books/{id}`    | Partially update a book by ID      | `200`          |
| `DELETE` | `/books/{id}`    | Delete a book by ID                | `200`          |

Note: in the current implementation, `POST /books/{id}` is also routed to the create handler.

Unsupported methods return `405 Method Not Allowed`; unknown book IDs return `404 Not Found`.

### Book object

```json
{
  "id": 1,
  "title": "The Great Gatsby",
  "author": "F. Scott Fitzgerald",
  "published_date": "1925-04-10T00:00:00Z",
  "is_available": true,
  "created_at": "2026-09-21T10:00:00Z",
  "updated_at": "2026-09-21T10:00:00Z"
}
```

### Examples

**List all books**

```bash
curl http://localhost:8080/books
```

**Create a book** (`title`, `author`, and `published_date` are required; `published_date` uses RFC 3339 format)

```bash
curl -X POST http://localhost:8080/books/create \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Clean Code",
    "author": "Robert C. Martin",
    "published_date": "2008-08-01T00:00:00Z",
    "is_available": true
  }'
```

**Get a book by ID**

```bash
curl http://localhost:8080/books/1
```

**Partially update a book** (only the provided fields are changed)

```bash
curl -X PUT http://localhost:8080/books/1 \
  -H "Content-Type: application/json" \
  -d '{ "is_available": false }'
```

**Delete a book**

```bash
curl -X DELETE http://localhost:8080/books/1
```

```json
{ "message": "Book deleted successfully" }
```

## Database Schema

Defined in [`sql/init.sql`](sql/init.sql):

| Table             | Description                                                            |
|-------------------|------------------------------------------------------------------------|
| `books`           | `title`, `author`, `published_date`, `is_available`, timestamps        |
| `users`           | `username` (unique), `password`, `created_at`                          |
| `favorite_books`  | Join table linking users to favorite books (cascade on delete)         |

Seed data: 5 books (The Great Gatsby, To Kill a Mockingbird, 1984, Pride and Prejudice, The Catcher in the Rye) and 3 users (`user1`–`user3`).

## Work in Progress

- [ ] Favorites feature — `AddFavoriteBook` store method and `FavoriteBook` model exist, but no route/handler wiring yet
- [ ] User store — `UserStore` is a stub
- [x] Published date is included in list/get queries
- [x] Dockerfile for containerizing the API itself (multi-stage build, runs in `docker compose up`)
- [ ] Automated tests, graceful shutdown, and structured logging

