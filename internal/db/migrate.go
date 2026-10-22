package db

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
)

// Migrate applies every NNNN_name.sql file in fsys whose version is greater
// than the database's PRAGMA user_version. Each file runs in its own
// transaction together with the version bump. Versions must run 1, 2, 3, …
// with no gaps or duplicates.
func Migrate(ctx context.Context, db *sql.DB, fsys fs.FS) error {
	names, err := fs.Glob(fsys, "*.sql")
	if err != nil {
		return err
	}
	sort.Strings(names)

	var current int
	if err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&current); err != nil {
		return fmt.Errorf("read user_version: %w", err)
	}

	for i, name := range names {
		version, err := parseVersion(name)
		if err != nil {
			return err
		}
		if version != i+1 {
			return fmt.Errorf("migration %s: expected version %d", name, i+1)
		}
		if version <= current {
			continue
		}

		body, err := fs.ReadFile(fsys, name)
		if err != nil {
			return err
		}
		if err := apply(ctx, db, version, string(body)); err != nil {
			return fmt.Errorf("migration %s: %w", name, err)
		}
		current = version
	}

	if current > len(names) {
		return fmt.Errorf("database is at schema version %d but this binary only knows %d; it was opened by a newer build", current, len(names))
	}
	return nil
}

func apply(ctx context.Context, db *sql.DB, version int, body string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, body); err != nil {
		return err
	}
	// PRAGMA does not accept bound parameters; version is an int we parsed.
	if _, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", version)); err != nil {
		return err
	}
	return tx.Commit()
}

func parseVersion(name string) (int, error) {
	prefix, _, ok := strings.Cut(name, "_")
	if !ok {
		return 0, fmt.Errorf("migration %s: name must look like 0001_description.sql", name)
	}
	v, err := strconv.Atoi(prefix)
	if err != nil || v < 1 {
		return 0, fmt.Errorf("migration %s: name must look like 0001_description.sql", name)
	}
	return v, nil
}
