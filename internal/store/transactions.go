package store

import (
	"context"
	"fmt"
	"strings"
)

type Transaction struct {
	ID          int64  `json:"id"`
	AccountID   int64  `json:"account_id"`
	CategoryID  *int64 `json:"category_id"` // nil = uncategorized, encodes as null
	Date        string `json:"date"`
	Payee       string `json:"payee"`
	AmountCents int64  `json:"amount_cents"`
	Note        string `json:"note"`
}

type TransactionInput struct {
	AccountID   int64
	CategoryID  *int64
	Date        string
	Payee       string
	AmountCents int64
	Note        string
}

// TransactionFilter selects one account's transactions. From and To are
// inclusive YYYY-MM-DD dates; an empty string leaves that end open.
type TransactionFilter struct {
	AccountID int64
	From      string
	To        string
}

// ListTransactions expects f to be validated already (see package api). It
// returns a NotFound error if the account doesn't exist.
func (s *Store) ListTransactions(ctx context.Context, f TransactionFilter) ([]Transaction, error) {
	ok, err := s.exists(ctx, "accounts", f.AccountID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errorf(NotFound, "no account with id %d", f.AccountID)
	}

	where := []string{"account_id = ?"}
	args := []any{f.AccountID}
	if f.From != "" {
		where = append(where, "date >= ?")
		args = append(args, f.From)
	}
	if f.To != "" {
		where = append(where, "date <= ?")
		args = append(args, f.To)
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, account_id, category_id, date, payee, amount_cents, note
		FROM transactions
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY date DESC, id DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	transactions := []Transaction{} // non-nil so it encodes as [] rather than null
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.AccountID, &t.CategoryID, &t.Date, &t.Payee, &t.AmountCents, &t.Note); err != nil {
			return nil, err
		}
		transactions = append(transactions, t)
	}
	return transactions, rows.Err()
}

// CreateTransaction expects in to be validated already (see package api). It
// returns an Invalid error, naming the field, if the account doesn't exist.
func (s *Store) CreateTransaction(ctx context.Context, in TransactionInput) (Transaction, error) {
	ok, err := s.exists(ctx, "accounts", in.AccountID)
	if err != nil {
		return Transaction{}, err
	}
	if !ok {
		return Transaction{}, errorf(Invalid, "unknown account_id %d", in.AccountID)
	}

	// We don't care about nil category ids
	if in.CategoryID != nil {
		ok, err := s.exists(ctx, "categories", *in.CategoryID)
		if err != nil {
			return Transaction{}, err
		}
		if !ok {
			return Transaction{}, errorf(Invalid, "unknown category_id %d", *in.CategoryID)
		}
	}

	res, err := s.db.ExecContext(ctx,
		`INSERT INTO transactions
		(account_id, category_id, date, payee, amount_cents, note)
		VALUES (?, ?, ?, ?, ?, ?)`,
		in.AccountID, in.CategoryID, in.Date, in.Payee, in.AmountCents, in.Note,
	)
	// Backstop for a row deleted between the check above and the insert.
	if isForeignKey(err) {
		return Transaction{}, errorf(Invalid, "unknown account_id or category_id")
	}
	if err != nil {
		return Transaction{}, fmt.Errorf("insert transaction: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Transaction{}, err
	}
	return Transaction{
		ID:          id,
		AccountID:   in.AccountID,
		CategoryID:  in.CategoryID,
		Date:        in.Date,
		Payee:       in.Payee,
		AmountCents: in.AmountCents,
		Note:        in.Note,
	}, nil
}

// exists reports whether table has a row with id. table is spliced into the
// SQL, so it must be a constant, never user input.
func (s *Store) exists(ctx context.Context, table string, id int64) (bool, error) {
	var ok bool
	err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM `+table+` WHERE id = ?)`, id).Scan(&ok)
	return ok, err
}
