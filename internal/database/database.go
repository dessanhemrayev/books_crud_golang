package database

import (
	"context"
	"embed"
	"fmt"
	"io/fs"

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
	// goose ищет файлы только в корне FS, поэтому отбрасываем префикс migrations/
	migrationsFS, err := fs.Sub(embedMigrations, "migrations")
	if err != nil {
		return fmt.Errorf("open embedded migrations: %w", err)
	}

	provider, err := goose.NewProvider(goose.DialectPostgres, db.DB, migrationsFS)
	if err != nil {
		return fmt.Errorf("create goose provider: %w", err)
	}

	if _, err := provider.Up(context.Background()); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}