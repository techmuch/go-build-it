// Package store opens the SQLite database and applies migrations (FR-04).
package store

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite" // pure-Go SQLite driver
)

//go:embed migrations/*.sql
var migrations embed.FS

// Open opens (or creates) the database at path and migrates it to the latest schema.
// Use ":memory:" in tests.
func Open(path string) (*sql.DB, error) {
	// Pragmas go in the DSN so every pooled connection gets them; a one-off Exec only
	// configures one connection, and the others would fail with SQLITE_BUSY at once.
	const pragmas = "_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"
	dsn := "file:" + path + "?" + pragmas
	if path == ":memory:" {
		dsn = "file::memory:?cache=shared&" + pragmas
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if path == ":memory:" {
		db.SetMaxOpenConns(1)
	}
	var timeout int
	if err := db.QueryRow("PRAGMA busy_timeout").Scan(&timeout); err != nil || timeout < 10000 {
		return nil, fmt.Errorf("sqlite pragmas not applied (busy_timeout=%d): %v", timeout, err)
	}
	if err := migrate(db); err != nil {
		return nil, err
	}
	return db, nil
}

func migrate(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (name TEXT PRIMARY KEY, applied_at TEXT NOT NULL)`); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}
	names, err := fs.Glob(migrations, "migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(names)
	for _, n := range names {
		var x string
		err := db.QueryRow(`SELECT name FROM schema_migrations WHERE name=?`, n).Scan(&x)
		if err == nil {
			continue
		}
		if err != sql.ErrNoRows {
			return fmt.Errorf("check migration %s: %w", n, err)
		}
		b, _ := migrations.ReadFile(n)
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		for _, stmt := range splitSQL(string(b)) {
			if _, err := tx.Exec(stmt); err != nil {
				tx.Rollback()
				return fmt.Errorf("migration %s: %w", n, err)
			}
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations(name, applied_at) VALUES(?,?)`, n, Now()); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func splitSQL(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ";") {
		var lines []string
		for _, l := range strings.Split(part, "\n") {
			if t := strings.TrimSpace(l); t != "" && !strings.HasPrefix(t, "--") {
				lines = append(lines, l)
			}
		}
		if len(lines) > 0 {
			out = append(out, strings.Join(lines, "\n"))
		}
	}
	return out
}

// Now returns the current time in the format stored in the database.
func Now() string { return time.Now().UTC().Format(time.RFC3339) }

// NextID returns the next sequential ID for a counter, formatted with prefix (e.g. "T-142").
func NextID(db *sql.DB, prefix string) (string, error) {
	var v int64
	err := db.QueryRow(`INSERT INTO counters(name, value) VALUES(?, 1)
		ON CONFLICT(name) DO UPDATE SET value = value + 1 RETURNING value`, prefix).Scan(&v)
	if err != nil {
		return "", fmt.Errorf("next id %s: %w", prefix, err)
	}
	return fmt.Sprintf("%s-%d", prefix, v), nil
}
