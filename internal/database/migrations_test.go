package database

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
)

func TestRunMigrationsTimeout(t *testing.T) {
	for _, value := range []string{"invalid", "0s", "-1s"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("MIGRATION_TIMEOUT", value)
			if err := RunMigrations(nil); err == nil || !strings.Contains(err.Error(), "MIGRATION_TIMEOUT") {
				t.Fatalf("expected configuration error before accessing the database, got %v", err)
			}
		})
	}

	for _, tc := range []struct {
		name, value string
		duration    time.Duration
		wait        bool
	}{
		{"default", "", 5 * time.Minute, false},
		{"configured", "10m", 10 * time.Minute, false},
		{"expires", "20ms", 20 * time.Millisecond, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("MIGRATION_TIMEOUT", tc.value)
			failure := errors.New("database unavailable")
			db := sqlx.NewDb(sql.OpenDB(migrationConnector{connect: func(ctx context.Context) (driver.Conn, error) {
				deadline, ok := ctx.Deadline()
				if !ok || time.Until(deadline) <= 0 || time.Until(deadline) > tc.duration {
					t.Fatalf("expected migration deadline within %s, got %v", tc.duration, deadline)
				}
				if tc.wait {
					<-ctx.Done()
				}
				// Drivers may return their own cancellation error, rather than ctx.Err().
				return nil, failure
			}}), "postgres")
			t.Cleanup(func() { db.Close() })
			want := failure
			if tc.wait {
				want = context.DeadlineExceeded
			}
			if err := RunMigrations(db); !errors.Is(err, want) {
				t.Fatalf("expected %v, got %v", want, err)
			}
		})
	}
}

type migrationConnector struct {
	connect func(context.Context) (driver.Conn, error)
}

func (c migrationConnector) Connect(ctx context.Context) (driver.Conn, error) {
	return c.connect(ctx)
}

func (c migrationConnector) Driver() driver.Driver { return c }
func (c migrationConnector) Open(string) (driver.Conn, error) {
	return nil, errors.New("use Connect")
}

// Run with TEST_DATABASE_URL pointing to a disposable PostgreSQL database.
func TestMigrationRollbacksPostgres(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is required for PostgreSQL migration checks")
	}
	admin, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	migrations, err := fs.Sub(embedMigrations, "migrations")
	if err != nil {
		t.Fatal(err)
	}

	for _, version := range []int64{1, 2} {
		t.Run(fmt.Sprintf("version_%d", version), func(t *testing.T) {
			schema := fmt.Sprintf("migration_test_%d", time.Now().UnixNano())
			if _, err := admin.Exec("CREATE SCHEMA " + schema); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if _, err := admin.Exec("DROP SCHEMA " + schema + " CASCADE"); err != nil {
					t.Error(err)
				}
			})
			u, err := url.Parse(dsn)
			if err != nil {
				t.Fatal(err)
			}
			query := u.Query()
			query.Set("search_path", schema)
			u.RawQuery = query.Encode()
			db, err := sqlx.Connect("postgres", u.String())
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			provider, err := goose.NewProvider(goose.DialectPostgres, db.DB, migrations)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if _, err := provider.UpTo(ctx, version); err != nil {
				t.Fatal(err)
			}
			if version == 2 {
				if _, err := db.ExecContext(ctx, `
					UPDATE users SET password = 'changed' WHERE username = 'user1';
					INSERT INTO favorite_books (book_id, user_id)
					SELECT b.id, u.id FROM books b, users u
					WHERE b.title = '1984' AND u.username = 'user1'`); err != nil {
					t.Fatal(err)
				}
			}
			// Include every row and migration history in the before/after comparison.
			snapshot := func() string {
				var data string
				if err := db.GetContext(ctx, &data, `SELECT json_build_array(
					(SELECT json_agg(u ORDER BY id) FROM users u),
					(SELECT json_agg(b ORDER BY id) FROM books b),
					(SELECT json_agg(f ORDER BY id) FROM favorite_books f),
					(SELECT json_agg(v ORDER BY id) FROM goose_db_version v))::text`); err != nil {
					t.Fatal(err)
				}
				return data
			}
			before := snapshot()
			if _, err := provider.Down(ctx); err == nil || !strings.Contains(err.Error(), "is irreversible") {
				t.Fatalf("expected explicit irreversible rollback error, got %v", err)
			}
			if after := snapshot(); after != before {
				t.Fatalf("rollback changed data or migration history: before %s; after %s", before, after)
			}
			if version == 1 {
				// Hold a lock that blocks seed insertion to exercise pq cancellation.
				tx, err := db.BeginTxx(ctx, nil)
				if err != nil {
					t.Fatal(err)
				}
				defer tx.Rollback()
				if _, err := tx.ExecContext(ctx, "LOCK TABLE users IN ACCESS EXCLUSIVE MODE"); err != nil {
					t.Fatal(err)
				}
				t.Setenv("MIGRATION_TIMEOUT", "100ms")
				if err := RunMigrations(db); !errors.Is(err, context.DeadlineExceeded) {
					t.Fatalf("expected deadline error during blocked seed migration, got %v", err)
				}
				if err := tx.Rollback(); err != nil {
					t.Fatal(err)
				}
				if after := snapshot(); after != before {
					t.Fatalf("timed out migration changed data or history: before %s; after %s", before, after)
				}
			}
			if version == 2 {
				t.Setenv("MIGRATION_TIMEOUT", "")
				if err := RunMigrations(db); err != nil {
					t.Fatalf("startup with applied migrations: %v", err)
				}
			}
		})
	}
}
