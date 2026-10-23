package store

import (
	"context"
	"fmt"
)

type Transfer struct {
	ID   int64          `json:"id"`
	Legs [2]Transaction `json:"legs"`
}

type TransferInput struct {
	AccountID      int64
	OtherAccountID int64
	Date           string
	Payee          string
	AmountCents    int64
	Note           string
}

// validateAccounts checks whether two accounts are different and also whether they exist
func (s *Store) validateAccounts(ctx context.Context, in TransferInput) error {
	// check both accounts exist and are different
	if in.AccountID == in.OtherAccountID {
		return errorf(Invalid, "other_account_id must be a different account")
	}

	ok, err := s.exists(ctx, "accounts", in.AccountID)
	if err != nil {
		return err
	}
	if !ok {
		return errorf(Invalid, "unknown account_id %d", in.AccountID)
	}

	ok, err = s.exists(ctx, "accounts", in.OtherAccountID)
	if err != nil {
		return err
	}
	if !ok {
		return errorf(Invalid, "unknown other_account_id %d", in.OtherAccountID)
	}
	return nil
}

// CreateTransfer expects in to be validated already (see package api). It
// creates the transfers row and both legs in one SQL transaction: AmountCents
// in AccountID and its negation in OtherAccountID. Legs[0] is in AccountID.
func (s *Store) CreateTransfer(ctx context.Context, in TransferInput) (Transfer, error) {
	err := s.validateAccounts(ctx, in)
	if err != nil {
		return Transfer{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Transfer{}, err
	}
	defer tx.Rollback() // no-op after Commit

	res, err := tx.ExecContext(ctx, `INSERT INTO transfers DEFAULT VALUES`)
	if err != nil {
		return Transfer{}, fmt.Errorf("insert transfer: %w", err)
	}
	transferID, err := res.LastInsertId()
	if err != nil {
		return Transfer{}, err
	}

	// Each leg's other account is the opposite side of the transfer.
	sides := [2]struct{ account, other, cents int64 }{
		{in.AccountID, in.OtherAccountID, in.AmountCents},
		{in.OtherAccountID, in.AccountID, -in.AmountCents},
	}
	var t Transfer
	t.ID = transferID
	for i, side := range sides {
		// category_id is left NULL: a transfer never has a category.
		res, err := tx.ExecContext(ctx,
			`INSERT INTO transactions (account_id, date, payee, amount_cents, note, transfer_id)
			VALUES (?, ?, ?, ?, ?, ?)`,
			side.account, in.Date, in.Payee, side.cents, in.Note, transferID)
		// Backstop for an account deleted since validateAccounts.
		if isForeignKey(err) {
			return Transfer{}, errorf(Invalid, "unknown account_id or other_account_id")
		}
		if err != nil {
			return Transfer{}, fmt.Errorf("insert transfer leg: %w", err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return Transfer{}, err
		}
		t.Legs[i] = Transaction{
			ID:                id,
			AccountID:         side.account,
			Date:              in.Date,
			Payee:             in.Payee,
			AmountCents:       side.cents,
			Note:              in.Note,
			TransferID:        &transferID,
			TransferAccountID: &side.other,
		}
	}

	if err := tx.Commit(); err != nil {
		return Transfer{}, err
	}
	return t, nil
}
