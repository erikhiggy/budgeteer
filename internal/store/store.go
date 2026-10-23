// Package store holds the app's SQL queries. Each entity gets its own file;
// handlers in package api call these methods and never touch SQL directly.
package store

import (
	"context"
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

// exists reports whether table has a row with id. table is spliced into the
// SQL, so it must be a constant, never user input.
func (s *Store) exists(ctx context.Context, table string, id int64) (bool, error) {
	var ok bool
	err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM `+table+` WHERE id = ?)`, id).Scan(&ok)
	return ok, err
}

// usage returns the name of the row with id in table and how many
// transactions refer to it through column, or sql.ErrNoRows if there's no
// such row. table and column are spliced into the SQL, so they must be
// constants, never user input.
func (s *Store) usage(ctx context.Context, table, column string, id int64) (name string, n int, err error) {
	err = s.db.QueryRowContext(ctx, `
		SELECT name, (SELECT COUNT(*) FROM transactions WHERE `+column+` = ?)
		FROM `+table+` WHERE id = ?`, id, id).Scan(&name, &n)
	return name, n, err
}

// wantOneRow turns an UPDATE or DELETE that matched no row into a NotFound
// error. what names the entity in the message, e.g. "account".
func wantOneRow(res sql.Result, what string, id int64) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return errorf(NotFound, "no %s with id %d", what, id)
	}
	return nil
}

// plural formats n with word, adding an s unless n is 1.
func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return fmt.Sprintf("%d %ss", n, word)
}
