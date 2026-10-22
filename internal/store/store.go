// Package store holds the app's SQL queries. Each entity gets its own file;
// handlers in package api call these methods and never touch SQL directly.
package store

import (
	"database/sql"
	"errors"
	"fmt"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

type Store struct {
	db *sql.DB
}

func New(db *sql.DB) *Store {
	return &Store{db: db}
}

// Kind classifies an *Error so the API layer can pick a status code.
type Kind int

const (
	NotFound Kind = iota + 1 // the thing asked for doesn't exist (404)
	Conflict                 // clashes with existing data, e.g. a duplicate name (409)
	Invalid                  // refers to something that doesn't exist, e.g. an unknown category_id (400)
)

// Error is a failure whose message is safe to show the user. Anything else a
// store method returns is unexpected and becomes a 500.
type Error struct {
	Kind Kind
	Msg  string
}

func (e *Error) Error() string { return e.Msg }

func errorf(kind Kind, format string, args ...any) error {
	return &Error{Kind: kind, Msg: fmt.Sprintf(format, args...)}
}

// isConstraint reports whether err is a SQLite constraint violation with the
// given extended result code.
func isConstraint(err error, code int) bool {
	var se *sqlite.Error
	return errors.As(err, &se) && se.Code() == code
}

func isUnique(err error) bool     { return isConstraint(err, sqlite3.SQLITE_CONSTRAINT_UNIQUE) }
func isForeignKey(err error) bool { return isConstraint(err, sqlite3.SQLITE_CONSTRAINT_FOREIGNKEY) }
