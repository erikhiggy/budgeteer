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
// returns an Invalid error, naming the field, if the account or category
// doesn't exist.
func (s *Store) CreateTransaction(ctx context.Context, in TransactionInput) (Transaction, error) {
	if err := s.checkRefs(ctx, in); err != nil {
		return Transaction{}, err
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
	return in.transaction(id), nil
}

// UpdateTransaction replaces every field of the transaction with id. It
// expects in to be validated already (see package api), and returns NotFound
// for an unknown id or Invalid for an unknown account or category.
func (s *Store) UpdateTransaction(ctx context.Context, id int64, in TransactionInput) (Transaction, error) {
	ok, err := s.exists(ctx, "transactions", id)
	if err != nil {
		return Transaction{}, err
	}
	if !ok {
		return Transaction{}, errorf(NotFound, "no transaction with id %d", id)
	}
	if err := s.checkRefs(ctx, in); err != nil {
		return Transaction{}, err
	}

	res, err := s.db.ExecContext(ctx, `
		UPDATE transactions
		SET account_id = ?, category_id = ?, date = ?, payee = ?, amount_cents = ?, note = ?
		WHERE id = ?`,
		in.AccountID, in.CategoryID, in.Date, in.Payee, in.AmountCents, in.Note, id,
	)
	// Backstop for a row deleted between the checks above and the update.
	if isForeignKey(err) {
		return Transaction{}, errorf(Invalid, "unknown account_id or category_id")
	}
	if err != nil {
		return Transaction{}, fmt.Errorf("update transaction: %w", err)
	}
	if err := wantOneRow(res, "transaction", id); err != nil {
		return Transaction{}, err
	}
	return in.transaction(id), nil
}

func (s *Store) DeleteTransaction(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM transactions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete transaction: %w", err)
	}
	return wantOneRow(res, "transaction", id)
}

// checkRefs returns an Invalid error, naming the field, if in refers to an
// account or category that doesn't exist. A nil CategoryID is uncategorized
// and always fine.
func (s *Store) checkRefs(ctx context.Context, in TransactionInput) error {
	ok, err := s.exists(ctx, "accounts", in.AccountID)
	if err != nil {
		return err
	}
	if !ok {
		return errorf(Invalid, "unknown account_id %d", in.AccountID)
	}

	if in.CategoryID != nil {
		ok, err := s.exists(ctx, "categories", *in.CategoryID)
		if err != nil {
			return err
		}
		if !ok {
			return errorf(Invalid, "unknown category_id %d", *in.CategoryID)
		}
	}
	return nil
}

func (in TransactionInput) transaction(id int64) Transaction {
	return Transaction{
		ID:          id,
		AccountID:   in.AccountID,
		CategoryID:  in.CategoryID,
		Date:        in.Date,
		Payee:       in.Payee,
		AmountCents: in.AmountCents,
		Note:        in.Note,
	}
}
