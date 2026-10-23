package store

import (
	"context"
	"database/sql"
	"errors"
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
	// Set on both legs of a transfer; nil on ordinary transactions.
	TransferID        *int64 `json:"transfer_id"`
	TransferAccountID *int64 `json:"transfer_account_id"` // the other leg's account
}

// transactionColumns selects a Transaction from transactions t, joined to the
// other leg o of its transfer (if any). Scan it with scanTransaction.
const transactionColumns = `
	t.id, t.account_id, t.category_id, t.date, t.payee, t.amount_cents, t.note,
	t.transfer_id, o.account_id
	FROM transactions t
	LEFT JOIN transactions o ON o.transfer_id = t.transfer_id AND o.id <> t.id`

func scanTransaction(row interface{ Scan(...any) error }) (Transaction, error) {
	var t Transaction
	err := row.Scan(&t.ID, &t.AccountID, &t.CategoryID, &t.Date, &t.Payee, &t.AmountCents, &t.Note,
		&t.TransferID, &t.TransferAccountID)
	return t, err
}

// getTransaction returns the transaction with id, or a NotFound error.
func (s *Store) getTransaction(ctx context.Context, id int64) (Transaction, error) {
	t, err := scanTransaction(s.db.QueryRowContext(ctx, `SELECT `+transactionColumns+` WHERE t.id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Transaction{}, errorf(NotFound, "no transaction with id %d", id)
	}
	return t, err
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

	where := []string{"t.account_id = ?"}
	args := []any{f.AccountID}
	if f.From != "" {
		where = append(where, "t.date >= ?")
		args = append(args, f.From)
	}
	if f.To != "" {
		where = append(where, "t.date <= ?")
		args = append(args, f.To)
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT `+transactionColumns+`
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY t.date DESC, t.id DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	transactions := []Transaction{} // non-nil so it encodes as [] rather than null
	for rows.Next() {
		t, err := scanTransaction(rows)
		if err != nil {
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
// for an unknown id or Invalid for an unknown account or category. On a
// transfer leg it updates the whole transfer (see updateTransfer).
func (s *Store) UpdateTransaction(ctx context.Context, id int64, in TransactionInput) (Transaction, error) {
	current, err := s.getTransaction(ctx, id)
	if err != nil {
		return Transaction{}, err
	}
	if current.TransferID != nil {
		return s.updateTransfer(ctx, current, in)
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

// updateTransfer applies in to leg and the other leg of its transfer: date,
// payee and note go to both, and the other leg gets the negated amount. The
// accounts can't change, and legs never have a category.
func (s *Store) updateTransfer(ctx context.Context, leg Transaction, in TransactionInput) (Transaction, error) {
	if in.AccountID != leg.AccountID {
		return Transaction{}, errorf(Invalid, "a transfer's accounts can't be changed")
	}
	if in.CategoryID != nil {
		return Transaction{}, errorf(Invalid, "a transfer has no category")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Transaction{}, err
	}
	defer tx.Rollback() // no-op after Commit

	if _, err := tx.ExecContext(ctx, `
		UPDATE transactions
		SET date = ?, payee = ?, note = ?,
			amount_cents = CASE WHEN id = ? THEN ? ELSE -? END
		WHERE transfer_id = ?`,
		in.Date, in.Payee, in.Note, leg.ID, in.AmountCents, in.AmountCents, *leg.TransferID); err != nil {
		return Transaction{}, fmt.Errorf("update transfer: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Transaction{}, err
	}

	leg.Date, leg.Payee, leg.Note, leg.AmountCents = in.Date, in.Payee, in.Note, in.AmountCents
	return leg, nil
}

// DeleteTransaction deletes the transaction with id. On a transfer leg it
// deletes the whole transfer: both legs go with the transfers row.
func (s *Store) DeleteTransaction(ctx context.Context, id int64) error {
	current, err := s.getTransaction(ctx, id)
	if err != nil {
		return err
	}
	if current.TransferID != nil {
		res, err := s.db.ExecContext(ctx, `DELETE FROM transfers WHERE id = ?`, *current.TransferID)
		if err != nil {
			return fmt.Errorf("delete transfer: %w", err)
		}
		return wantOneRow(res, "transaction", id)
	}

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
