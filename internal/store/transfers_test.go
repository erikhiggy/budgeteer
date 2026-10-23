package store

import (
	"context"
	"database/sql"
	"testing"
)

// insertTransfer adds a transfer's two legs with raw SQL until CreateTransfer
// exists: cents in account a, and -cents in account b. It returns the legs'
// ids.
func insertTransfer(t *testing.T, conn *sql.DB, a, b int64, date string, cents int64) (legA, legB int64) {
	t.Helper()
	res, err := conn.Exec(`INSERT INTO transfers DEFAULT VALUES`)
	if err != nil {
		t.Fatal(err)
	}
	transferID, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	ids := [2]int64{}
	for i, leg := range []struct {
		account int64
		cents   int64
	}{{a, cents}, {b, -cents}} {
		res, err := conn.Exec(
			`INSERT INTO transactions (account_id, date, payee, amount_cents, transfer_id) VALUES (?, ?, 'Savings plan', ?, ?)`,
			leg.account, date, leg.cents, transferID)
		if err != nil {
			t.Fatal(err)
		}
		if ids[i], err = res.LastInsertId(); err != nil {
			t.Fatal(err)
		}
	}
	return ids[0], ids[1]
}

// twoAccounts creates Checking and Savings.
func twoAccounts(t *testing.T, s *Store) (checking, savings Account) {
	t.Helper()
	ctx := context.Background()
	checking, err := s.CreateAccount(ctx, AccountInput{Name: "Checking", Type: "checking"})
	if err != nil {
		t.Fatal(err)
	}
	savings, err = s.CreateAccount(ctx, AccountInput{Name: "Savings", Type: "savings"})
	if err != nil {
		t.Fatal(err)
	}
	return checking, savings
}

func TestListTransactionsShowsTransferLegs(t *testing.T) {
	s, conn := newTestStore(t)
	ctx := context.Background()
	checking, savings := twoAccounts(t, s)
	out, in := insertTransfer(t, conn, checking.ID, savings.ID, "2026-10-01", -20000)
	ordinary, err := s.CreateTransaction(ctx, TransactionInput{AccountID: checking.ID, Date: "2026-10-02", AmountCents: -500})
	if err != nil {
		t.Fatal(err)
	}

	got, err := s.ListTransactions(ctx, TransactionFilter{AccountID: checking.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != ordinary.ID || got[1].ID != out {
		t.Fatalf("checking = %+v", got)
	}
	if got[0].TransferID != nil || got[0].TransferAccountID != nil {
		t.Errorf("ordinary transaction has transfer fields: %+v", got[0])
	}
	leg := got[1]
	if leg.TransferID == nil || leg.TransferAccountID == nil || *leg.TransferAccountID != savings.ID || leg.CategoryID != nil {
		t.Errorf("checking leg = %+v, want a transfer to account %d", leg, savings.ID)
	}

	got, err = s.ListTransactions(ctx, TransactionFilter{AccountID: savings.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != in || got[0].AmountCents != 20000 || *got[0].TransferAccountID != checking.ID {
		t.Errorf("savings = %+v", got)
	}
}

func TestUpdateTransferLegUpdatesBothLegs(t *testing.T) {
	s, conn := newTestStore(t)
	ctx := context.Background()
	checking, savings := twoAccounts(t, s)
	_, in := insertTransfer(t, conn, checking.ID, savings.ID, "2026-10-01", -20000)

	// Edited from the savings side: now 150.00 into savings.
	got, err := s.UpdateTransaction(ctx, in, TransactionInput{
		AccountID: savings.ID, Date: "2026-10-03", Payee: "Rainy day", AmountCents: 15000, Note: "monthly",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != in || got.AmountCents != 15000 || got.Date != "2026-10-03" || got.Payee != "Rainy day" ||
		got.TransferID == nil || *got.TransferAccountID != checking.ID {
		t.Errorf("updated = %+v", got)
	}

	other, err := s.ListTransactions(ctx, TransactionFilter{AccountID: checking.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(other) != 1 || other[0].AmountCents != -15000 || other[0].Date != "2026-10-03" ||
		other[0].Payee != "Rainy day" || other[0].Note != "monthly" {
		t.Errorf("other leg = %+v", other)
	}
	accounts, err := s.ListAccounts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if accounts[0].BalanceCents != -15000 || accounts[1].BalanceCents != 15000 {
		t.Errorf("balances = %+v", accounts)
	}
}

func TestUpdateTransferLegRules(t *testing.T) {
	s, conn := newTestStore(t)
	ctx := context.Background()
	checking, savings := twoAccounts(t, s)
	out, _ := insertTransfer(t, conn, checking.ID, savings.ID, "2026-10-01", -20000)
	cat, err := s.CreateCategory(ctx, CategoryInput{Name: "Coffee"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.UpdateTransaction(ctx, out, TransactionInput{AccountID: savings.ID, Date: "2026-10-01", AmountCents: -1})
	wantKind(t, err, Invalid)
	if err.Error() != "a transfer's accounts can't be changed" {
		t.Errorf("message = %q", err)
	}

	_, err = s.UpdateTransaction(ctx, out, TransactionInput{AccountID: checking.ID, CategoryID: &cat.ID, Date: "2026-10-01", AmountCents: -1})
	wantKind(t, err, Invalid)
	if err.Error() != "a transfer has no category" {
		t.Errorf("message = %q", err)
	}
}

func TestDeleteTransferLegDeletesBothLegs(t *testing.T) {
	s, conn := newTestStore(t)
	ctx := context.Background()
	checking, savings := twoAccounts(t, s)
	out, in := insertTransfer(t, conn, checking.ID, savings.ID, "2026-10-01", -20000)

	if err := s.DeleteTransaction(ctx, out); err != nil {
		t.Fatal(err)
	}
	var legs, transfers int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM transactions`).Scan(&legs); err != nil {
		t.Fatal(err)
	}
	if err := conn.QueryRow(`SELECT COUNT(*) FROM transfers`).Scan(&transfers); err != nil {
		t.Fatal(err)
	}
	if legs != 0 || transfers != 0 {
		t.Errorf("%d legs and %d transfers left, want none", legs, transfers)
	}
	wantKind(t, s.DeleteTransaction(ctx, in), NotFound)
}

func TestListPayeesLeavesOutTransfers(t *testing.T) {
	s, conn := newTestStore(t)
	checking, savings := twoAccounts(t, s)
	insertTransfer(t, conn, checking.ID, savings.ID, "2026-10-01", -20000)

	got, err := s.ListPayees(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("payees = %+v, want none", got)
	}
}
