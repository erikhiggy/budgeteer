package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// AccountTypes are the values the accounts.type CHECK constraint allows.
var AccountTypes = []string{"checking", "savings", "credit", "cash"}

type Account struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	BalanceCents int64  `json:"balance_cents"`
}

type AccountInput struct {
	Name string
	Type string
	// OpeningBalanceCents, if non-zero, becomes a transaction in the
	// "Opening balance" category dated OpeningBalanceDate.
	OpeningBalanceCents int64
	OpeningBalanceDate  string
}

// AccountUpdate changes the non-nil fields.
type AccountUpdate struct {
	Name *string
	Type *string
}

func (s *Store) ListAccounts(ctx context.Context) ([]Account, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT a.id, a.name, a.type, COALESCE(SUM(t.amount_cents), 0)
		FROM accounts a
		LEFT JOIN transactions t ON t.account_id = a.id
		GROUP BY a.id
		ORDER BY a.name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	accounts := []Account{} // non-nil so it encodes as [] rather than null
	for rows.Next() {
		var a Account
		if err := rows.Scan(&a.ID, &a.Name, &a.Type, &a.BalanceCents); err != nil {
			return nil, err
		}
		accounts = append(accounts, a)
	}
	return accounts, rows.Err()
}

// getAccount returns the account with id, including its balance, or a
// NotFound error.
func (s *Store) getAccount(ctx context.Context, id int64) (Account, error) {
	var a Account
	err := s.db.QueryRowContext(ctx, `
		SELECT a.id, a.name, a.type, COALESCE(SUM(t.amount_cents), 0)
		FROM accounts a
		LEFT JOIN transactions t ON t.account_id = a.id
		WHERE a.id = ?
		GROUP BY a.id`, id).Scan(&a.ID, &a.Name, &a.Type, &a.BalanceCents)
	if errors.Is(err, sql.ErrNoRows) {
		return Account{}, errorf(NotFound, "no account with id %d", id)
	}
	return a, err
}

// CreateAccount expects in to be validated already (see package api). The
// account and its opening balance transaction are created together or not
// at all.
func (s *Store) CreateAccount(ctx context.Context, in AccountInput) (Account, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Account{}, err
	}
	defer tx.Rollback() // no-op after Commit

	res, err := tx.ExecContext(ctx,
		`INSERT INTO accounts (name, type) VALUES (?, ?)`, in.Name, in.Type)
	if isUnique(err) {
		return Account{}, errorf(Conflict, "an account named %q already exists", in.Name)
	}
	if err != nil {
		return Account{}, fmt.Errorf("insert account: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Account{}, err
	}

	if in.OpeningBalanceCents != 0 {
		var categoryID int64
		if err := tx.QueryRowContext(ctx,
			`SELECT id FROM categories WHERE name = ?`, OpeningBalanceCategory).Scan(&categoryID); err != nil {
			return Account{}, fmt.Errorf("find %q category: %w", OpeningBalanceCategory, err)
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO transactions (account_id, category_id, date, payee, amount_cents) VALUES (?, ?, ?, ?, ?)`,
			id, categoryID, in.OpeningBalanceDate, OpeningBalanceCategory, in.OpeningBalanceCents); err != nil {
			return Account{}, fmt.Errorf("insert opening balance: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return Account{}, err
	}
	return Account{ID: id, Name: in.Name, Type: in.Type, BalanceCents: in.OpeningBalanceCents}, nil
}

// UpdateAccount expects in to be validated already (see package api), with at
// least one field set.
func (s *Store) UpdateAccount(ctx context.Context, id int64, in AccountUpdate) (Account, error) {
	var set []string
	var args []any
	if in.Name != nil {
		set = append(set, "name = ?")
		args = append(args, *in.Name)
	}
	if in.Type != nil {
		set = append(set, "type = ?")
		args = append(args, *in.Type)
	}

	res, err := s.db.ExecContext(ctx,
		`UPDATE accounts SET `+strings.Join(set, ", ")+` WHERE id = ?`, append(args, id)...)
	if isUnique(err) {
		return Account{}, errorf(Conflict, "an account named %q already exists", *in.Name)
	}
	if err != nil {
		return Account{}, fmt.Errorf("update account: %w", err)
	}
	if err := wantOneRow(res, "account", id); err != nil {
		return Account{}, err
	}
	return s.getAccount(ctx, id)
}

// DeleteAccount returns a Conflict error if the account has transactions.
func (s *Store) DeleteAccount(ctx context.Context, id int64) error {
	name, n, err := s.usage(ctx, "accounts", "account_id", id)
	if errors.Is(err, sql.ErrNoRows) {
		return errorf(NotFound, "no account with id %d", id)
	}
	if err != nil {
		return err
	}
	if n > 0 {
		return errorf(Conflict, "%q has %s; delete or move them first", name, plural(n, "transaction"))
	}

	res, err := s.db.ExecContext(ctx, `DELETE FROM accounts WHERE id = ?`, id)
	// Backstop for a transaction added since the count above.
	if isForeignKey(err) {
		return errorf(Conflict, "%q has transactions; delete or move them first", name)
	}
	if err != nil {
		return fmt.Errorf("delete account: %w", err)
	}
	return wantOneRow(res, "account", id)
}
