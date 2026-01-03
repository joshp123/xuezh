package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"

	"github.com/joshp123/xuezh/internal/xuezh/clock"
	"github.com/joshp123/xuezh/internal/xuezh/migrations"
	"github.com/joshp123/xuezh/internal/xuezh/paths"
)

func InitDB() (string, error) {
	dbPath, err := paths.DBPath()
	if err != nil {
		return "", err
	}
	resolved, err := paths.ResolveInWorkspace(dbPath)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
		return "", err
	}
	conn, err := sql.Open("sqlite3", resolved)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	if _, err := conn.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return "", err
	}
	if err := ensureMigrationsTable(conn); err != nil {
		return "", err
	}
	applied, err := appliedVersions(conn)
	if err != nil {
		return "", err
	}
	migs, err := migrations.Load()
	if err != nil {
		return "", err
	}
	for _, mig := range migs {
		if applied[mig.Version] {
			continue
		}
		if err := applyMigration(conn, mig.Version, mig.SQL); err != nil {
			return "", err
		}
	}
	return resolved, nil
}

func ensureMigrationsTable(conn *sql.DB) error {
	_, err := conn.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
		  version TEXT PRIMARY KEY,
		  applied_at TEXT NOT NULL
		)
	`)
	return err
}

func appliedVersions(conn *sql.DB) (map[string]bool, error) {
	rows, err := conn.Query("SELECT version FROM schema_migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	applied := map[string]bool{}
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}
	return applied, rows.Err()
}

func applyMigration(conn *sql.DB, version, sqlText string) error {
	if _, err := conn.Exec(sqlText); err != nil {
		return err
	}
	now, err := clock.NowUTC()
	if err != nil {
		return err
	}
	_, err = conn.Exec(
		"INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)",
		version,
		clock.FormatISO(now),
	)
	return err
}

func ListTables(dbPath string) ([]string, error) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	rows, err := conn.Query("SELECT name FROM sqlite_master WHERE type='table'")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tables, nil
}

func EnsureDBExists() error {
	_, err := InitDB()
	if err != nil {
		return fmt.Errorf("db init failed: %w", err)
	}
	return nil
}
