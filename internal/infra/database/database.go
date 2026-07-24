package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

const filename = "wildman.db"

//go:embed migrations/*.sql
var migrationFiles embed.FS

type migration struct {
	version int
	file    string
}

var migrations = []migration{
	{version: 1, file: "migrations/001_initial.sql"},
	{version: 2, file: "migrations/002_central_catalog.sql"},
	{version: 3, file: "migrations/003_observation_track_position.sql"},
	{version: 4, file: "migrations/004_resolution_candidate_source.sql"},
	{version: 5, file: "migrations/005_candidate_tag_patch.sql"},
	{version: 6, file: "migrations/006_resolution_reviews.sql"},
	{version: 7, file: "migrations/007_audit_events.sql"},
	{version: 8, file: "migrations/008_candidate_sources.sql"},
	{version: 9, file: "migrations/009_accounts_devices_billing.sql"},
	{version: 10, file: "migrations/010_provider_metrics.sql"},
}

func Open(ctx context.Context, dataDir string) (*DB, error) {
	if dataDir == "" {
		return nil, fmt.Errorf("data directory is required")
	}

	absoluteDir, err := filepath.Abs(dataDir)
	if err != nil {
		return nil, fmt.Errorf("resolve data directory: %w", err)
	}
	if err := os.MkdirAll(absoluteDir, 0o700); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}

	databasePath := filepath.Join(absoluteDir, filename)
	dsn := "file:" + filepath.ToSlash(databasePath) +
		"?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)"

	rawDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db := &DB{DB: rawDB, dialect: DialectSQLite}
	db.SetMaxOpenConns(1)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	if err := applyMigrations(ctx, db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func OpenPostgres(ctx context.Context, databaseURL string) (*DB, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("PostgreSQL database URL is required")
	}
	rawDB, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open PostgreSQL: %w", err)
	}
	db := &DB{DB: rawDB, dialect: DialectPostgres}
	db.SetMaxOpenConns(20)
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping PostgreSQL: %w", err)
	}
	if err := applyMigrations(ctx, db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func OpenConfigured(ctx context.Context, dataDir, databaseURL string) (*DB, error) {
	if databaseURL != "" {
		return OpenPostgres(ctx, databaseURL)
	}
	return Open(ctx, dataDir)
}

func applyMigrations(ctx context.Context, db *DB) error {
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TEXT NOT NULL
		)
	`); err != nil {
		return fmt.Errorf("create migration table: %w", err)
	}

	for _, item := range migrations {
		var applied bool
		if err := db.QueryRowContext(ctx,
			"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = ?)",
			item.version,
		).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %d: %w", item.version, err)
		}
		if applied {
			continue
		}

		contents, err := fs.ReadFile(migrationFiles, item.file)
		if err != nil {
			return fmt.Errorf("read migration %d: %w", item.version, err)
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %d: %w", item.version, err)
		}
		for _, statement := range strings.Split(string(contents), ";") {
			statement = strings.TrimSpace(statement)
			if statement == "" {
				continue
			}
			if _, err := tx.ExecContext(ctx, statement); err != nil {
				tx.Rollback()
				return fmt.Errorf("apply migration %d: %w", item.version, err)
			}
		}
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)",
			item.version, time.Now().UTC().Format(time.RFC3339Nano),
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %d: %w", item.version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d: %w", item.version, err)
		}
	}

	return nil
}
