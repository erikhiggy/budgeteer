package store

import (
	"context"
	"testing"
)

// createTestTransfer moves cents into account a from account b (negative
// cents is money out of a into b) and returns the legs' ids.
func createTestTransfer(t *testing.T, s *Store, a, b int64, date string, cents int64) (legA, legB int64) {
	t.Helper()
	tr, err := s.CreateTransfer(context.Background(), TransferInput{
		AccountID: a, OtherAccountID: b, Date: date, Payee: "Savings plan", AmountCents: cents,
	})
	if err != nil {
		t.Fatal(err)
	}
	return tr.Legs[0].ID, tr.Legs[1].ID
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
	s, _ := newTestStore(t)
	ctx := context.Background()
	checking, savings := twoAccounts(t, s)
	out, in := createTestTransfer(t, s, checking.ID, savings.ID, "2026-10-01", -20000)
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
	s, _ := newTestStore(t)
	ctx := context.Background()
	checking, savings := twoAccounts(t, s)
	_, in := createTestTransfer(t, s, checking.ID, savings.ID, "2026-10-01", -20000)

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
	s, _ := newTestStore(t)
	ctx := context.Background()
	checking, savings := twoAccounts(t, s)
	out, _ := createTestTransfer(t, s, checking.ID, savings.ID, "2026-10-01", -20000)
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
	out, in := createTestTransfer(t, s, checking.ID, savings.ID, "2026-10-01", -20000)

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
	s, _ := newTestStore(t)
	checking, savings := twoAccounts(t, s)
	createTestTransfer(t, s, checking.ID, savings.ID, "2026-10-01", -20000)

	got, err := s.ListPayees(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("payees = %+v, want none", got)
	}
}

func TestCreateTransfer(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	checking, savings := twoAccounts(t, s)

	// From Savings' side, a positive amount is money into Savings from Checking.
	tr, err := s.CreateTransfer(ctx, TransferInput{
		AccountID: savings.ID, OtherAccountID: checking.ID, Date: "2026-10-23", Payee: "Rainy day", AmountCents: 20000, Note: "monthly",
	})
	if err != nil {
		t.Fatal(err)
	}
	in, out := tr.Legs[0], tr.Legs[1]
	if tr.ID == 0 || in.AccountID != savings.ID || in.AmountCents != 20000 || out.AccountID != checking.ID || out.AmountCents != -20000 {
		t.Fatalf("transfer = %+v", tr)
	}
	for _, leg := range tr.Legs {
		if leg.CategoryID != nil || leg.TransferID == nil || *leg.TransferID != tr.ID ||
			leg.Date != "2026-10-23" || leg.Payee != "Rainy day" || leg.Note != "monthly" {
			t.Errorf("leg = %+v", leg)
		}
	}
	if *in.TransferAccountID != checking.ID || *out.TransferAccountID != savings.ID {
		t.Errorf("other accounts = %d, %d", *in.TransferAccountID, *out.TransferAccountID)
	}

	// What it returns matches what's stored.
	for _, leg := range tr.Legs {
		got, err := s.getTransaction(ctx, leg.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.ID != leg.ID || got.AmountCents != leg.AmountCents || *got.TransferAccountID != *leg.TransferAccountID {
			t.Errorf("stored = %+v, returned %+v", got, leg)
		}
	}

	// A transfer moves both balances and leaves the total alone.
	accounts, err := s.ListAccounts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if accounts[0].BalanceCents != -20000 || accounts[1].BalanceCents != 20000 {
		t.Errorf("balances = %+v", accounts)
	}
}

func TestCreateTransferRules(t *testing.T) {
	s, conn := newTestStore(t)
	ctx := context.Background()
	checking, _ := twoAccounts(t, s)

	cases := []struct {
		name           string
		account, other int64
		msg            string
	}{
		{"same account", checking.ID, checking.ID, "other_account_id must be a different account"},
		{"unknown account", 99, checking.ID, "unknown account_id 99"},
		{"unknown other account", checking.ID, 99, "unknown other_account_id 99"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := s.CreateTransfer(ctx, TransferInput{AccountID: c.account, OtherAccountID: c.other, Date: "2026-10-23", AmountCents: -100})
			wantKind(t, err, Invalid)
			if err.Error() != c.msg {
				t.Errorf("message = %q, want %q", err, c.msg)
			}
		})
	}

	// Nothing half-made was left behind.
	var transfers, legs int
	if err := conn.QueryRow(`SELECT (SELECT COUNT(*) FROM transfers), (SELECT COUNT(*) FROM transactions)`).Scan(&transfers, &legs); err != nil {
		t.Fatal(err)
	}
	if transfers != 0 || legs != 0 {
		t.Errorf("%d transfers and %d transactions left, want none", transfers, legs)
	}
}
