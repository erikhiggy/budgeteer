package store

import (
	"context"
	"fmt"
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

// CreateAccount expects in to be validated already (see package api).
func (s *Store) CreateAccount(ctx context.Context, in AccountInput) (Account, error) {
	res, err := s.db.ExecContext(ctx,
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
	return Account{ID: id, Name: in.Name, Type: in.Type}, nil
}
