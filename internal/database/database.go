package database

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

func Connect(databaseURL string) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", databaseURL)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	return db, nil
}

func RunMigrations(db *sqlx.DB) error {
	timeout := 5 * time.Minute
	if value := os.Getenv("MIGRATION_TIMEOUT"); value != "" {
		var err error
		timeout, err = time.ParseDuration(value)
		if err != nil || timeout <= 0 {
			return fmt.Errorf("MIGRATION_TIMEOUT must be a positive duration, got %q", value)
		}
	}

	// goose ищет файлы только в корне FS, поэтому отбрасываем префикс migrations/
	migrationsFS, err := fs.Sub(embedMigrations, "migrations")
	if err != nil {
		return fmt.Errorf("open embedded migrations: %w", err)
	}

	provider, err := goose.NewProvider(goose.DialectPostgres, db.DB, migrationsFS)
	if err != nil {
		return fmt.Errorf("create goose provider: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if _, err := provider.Up(ctx); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("apply migrations (timeout %s): %w", timeout, ctx.Err())
		}
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}
