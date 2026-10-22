package store

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"budgeteer/internal/db"
)

// newTestStore returns a Store over a fresh, fully migrated database file.
func newTestStore(t *testing.T) (*Store, *sql.DB) {
	t.Helper()
	conn, err := db.Open(context.Background(), filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return New(conn), conn
}

// wantKind fails the test unless err is a *Error of the given kind.
func wantKind(t *testing.T, err error, kind Kind) {
	t.Helper()
	var se *Error
	if !errors.As(err, &se) {
		t.Fatalf("err = %v (%T), want *store.Error", err, err)
	}
	if se.Kind != kind {
		t.Fatalf("err kind = %d (%q), want %d", se.Kind, se.Msg, kind)
	}
}
