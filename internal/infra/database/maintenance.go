package database

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type Verification struct {
	Integrity        string `json:"integrity"`
	MigrationVersion int    `json:"migrationVersion"`
}

func DatabasePath(dataDir string) (string, error) {
	absoluteDir, err := filepath.Abs(dataDir)
	if err != nil {
		return "", err
	}
	return filepath.Join(absoluteDir, filename), nil
}

func Backup(ctx context.Context, database *DB, outputPath string) error {
	if database.Dialect() != DialectSQLite {
		return fmt.Errorf("online backup is only supported for SQLite")
	}
	absoluteOutput, err := filepath.Abs(outputPath)
	if err != nil {
		return err
	}
	if _, err := os.Stat(absoluteOutput); err == nil {
		return fmt.Errorf("backup output already exists")
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(absoluteOutput), 0o700); err != nil {
		return err
	}
	if _, err := database.ExecContext(ctx, "VACUUM INTO ?", absoluteOutput); err != nil {
		return fmt.Errorf("create SQLite backup: %w", err)
	}
	return nil
}

func Verify(ctx context.Context, path string) (Verification, error) {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return Verification{}, err
	}
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(absolutePath)+"?mode=ro&_pragma=foreign_keys(1)")
	if err != nil {
		return Verification{}, err
	}
	defer db.Close()
	var verification Verification
	if err := db.QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&verification.Integrity); err != nil {
		return Verification{}, fmt.Errorf("run integrity check: %w", err)
	}
	if verification.Integrity != "ok" {
		return Verification{}, fmt.Errorf("SQLite integrity check failed")
	}
	if err := db.QueryRowContext(ctx, "SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&verification.MigrationVersion); err != nil {
		return Verification{}, fmt.Errorf("read migration version: %w", err)
	}
	if verification.MigrationVersion != migrations[len(migrations)-1].version {
		return Verification{}, fmt.Errorf("backup migration version %d is incompatible with expected %d", verification.MigrationVersion, migrations[len(migrations)-1].version)
	}
	return verification, nil
}

func Restore(ctx context.Context, dataDir, backupPath string) (string, error) {
	if _, err := Verify(ctx, backupPath); err != nil {
		return "", err
	}
	target, err := DatabasePath(dataDir)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return "", err
	}
	temporary, err := os.CreateTemp(filepath.Dir(target), "wildman-restore-*.db")
	if err != nil {
		return "", err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	source, err := os.Open(backupPath)
	if err != nil {
		temporary.Close()
		return "", err
	}
	_, copyErr := io.Copy(temporary, source)
	closeSourceErr := source.Close()
	syncErr := temporary.Sync()
	closeTempErr := temporary.Close()
	if copyErr != nil {
		return "", copyErr
	}
	if closeSourceErr != nil {
		return "", closeSourceErr
	}
	if syncErr != nil {
		return "", syncErr
	}
	if closeTempErr != nil {
		return "", closeTempErr
	}
	recoveryPath := ""
	if _, err := os.Stat(target); err == nil {
		recoveryPath = target + ".before-restore-" + time.Now().UTC().Format("20060102T150405Z")
		if err := os.Rename(target, recoveryPath); err != nil {
			return "", fmt.Errorf("preserve current database: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}
	if err := os.Rename(temporaryPath, target); err != nil {
		if recoveryPath != "" {
			_ = os.Rename(recoveryPath, target)
		}
		return "", fmt.Errorf("activate restored database: %w", err)
	}
	return recoveryPath, nil
}
