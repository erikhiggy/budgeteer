package store

import (
	"context"
	"testing"
)

func TestCreateAndListAccounts(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	for _, in := range []AccountInput{
		{Name: "savings", Type: "savings"},
		{Name: "Checking", Type: "checking"},
	} {
		if _, err := s.CreateAccount(ctx, in); err != nil {
			t.Fatal(err)
		}
	}

	got, err := s.ListAccounts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// Sorted case-insensitively by name.
	if len(got) != 2 || got[0].Name != "Checking" || got[1].Name != "savings" {
		t.Fatalf("accounts = %+v", got)
	}
}

func TestListAccountsEmptyIsNotNil(t *testing.T) {
	s, _ := newTestStore(t)
	got, err := s.ListAccounts(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Error("got nil slice, want empty slice")
	}
}

func TestCreateAccountDuplicateNameIsConflict(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	if _, err := s.CreateAccount(ctx, AccountInput{Name: "Checking", Type: "checking"}); err != nil {
		t.Fatal(err)
	}
	_, err := s.CreateAccount(ctx, AccountInput{Name: "CHECKING", Type: "savings"})
	wantKind(t, err, Conflict)
}

func TestAccountBalanceSumsTransactions(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	a, err := s.CreateAccount(ctx, AccountInput{Name: "Checking", Type: "checking"})
	if err != nil {
		t.Fatal(err)
	}

	for _, cents := range []int64{100000, -2550, -999} {
		if _, err := s.CreateTransaction(ctx,
			TransactionInput{AccountID: a.ID, Date: "2025-10-01", AmountCents: cents}); err != nil {
			t.Fatal(err)
		}
	}

	got, err := s.ListAccounts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].BalanceCents != 96451 {
		t.Errorf("balance = %d, want 96451", got[0].BalanceCents)
	}
}

func TestCreateAccountWithOpeningBalance(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	a, err := s.CreateAccount(ctx, AccountInput{Name: "Visa", Type: "credit", OpeningBalanceCents: -12345, OpeningBalanceDate: "2026-10-23"})
	if err != nil {
		t.Fatal(err)
	}
	if a.BalanceCents != -12345 {
		t.Errorf("created balance = %d, want -12345", a.BalanceCents)
	}

	txs, err := s.ListTransactions(ctx, TransactionFilter{AccountID: a.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(txs) != 1 || txs[0].Date != "2026-10-23" || txs[0].Payee != "Opening balance" || txs[0].AmountCents != -12345 || txs[0].CategoryID == nil {
		t.Fatalf("transactions = %+v", txs)
	}
	c, err := s.getCategory(ctx, *txs[0].CategoryID)
	if err != nil || c.Name != OpeningBalanceCategory {
		t.Errorf("category = %+v, %v; want %q", c, err, OpeningBalanceCategory)
	}
}

func TestCreateAccountZeroOpeningBalanceAddsNoTransaction(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	a, err := s.CreateAccount(ctx, AccountInput{Name: "Cash", Type: "cash"})
	if err != nil {
		t.Fatal(err)
	}
	txs, err := s.ListTransactions(ctx, TransactionFilter{AccountID: a.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(txs) != 0 {
		t.Errorf("transactions = %+v, want none", txs)
	}
}

func TestCreateAccountDuplicateLeavesNoOpeningBalance(t *testing.T) {
	s, conn := newTestStore(t)
	ctx := context.Background()
	if _, err := s.CreateAccount(ctx, AccountInput{Name: "Checking", Type: "checking"}); err != nil {
		t.Fatal(err)
	}
	_, err := s.CreateAccount(ctx, AccountInput{Name: "checking", Type: "checking", OpeningBalanceCents: 500, OpeningBalanceDate: "2026-10-23"})
	wantKind(t, err, Conflict)

	var n int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM transactions`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("%d transactions left behind, want 0", n)
	}
}

func TestUpdateAccount(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	a, err := s.CreateAccount(ctx, AccountInput{Name: "checking", Type: "checking", OpeningBalanceCents: 1000, OpeningBalanceDate: "2026-10-01"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateAccount(ctx, AccountInput{Name: "Savings", Type: "savings"}); err != nil {
		t.Fatal(err)
	}

	// Changing only the case of its own name isn't a conflict.
	name := "Checking"
	got, err := s.UpdateAccount(ctx, a.ID, AccountUpdate{Name: &name})
	if err != nil {
		t.Fatal(err)
	}
	if got != (Account{ID: a.ID, Name: "Checking", Type: "checking", BalanceCents: 1000}) {
		t.Errorf("updated = %+v", got)
	}

	// Only the fields sent change.
	typ := "cash"
	got, err = s.UpdateAccount(ctx, a.ID, AccountUpdate{Type: &typ})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Checking" || got.Type != "cash" {
		t.Errorf("updated = %+v", got)
	}

	taken := "SAVINGS"
	_, err = s.UpdateAccount(ctx, a.ID, AccountUpdate{Name: &taken})
	wantKind(t, err, Conflict)

	_, err = s.UpdateAccount(ctx, 99, AccountUpdate{Name: &name})
	wantKind(t, err, NotFound)
}

func TestDeleteAccount(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	used, err := s.CreateAccount(ctx, AccountInput{Name: "Checking", Type: "checking"})
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if _, err := s.CreateTransaction(ctx, TransactionInput{AccountID: used.ID, Date: "2026-10-01", AmountCents: -100}); err != nil {
			t.Fatal(err)
		}
	}
	empty, err := s.CreateAccount(ctx, AccountInput{Name: "Spare", Type: "cash"})
	if err != nil {
		t.Fatal(err)
	}

	err = s.DeleteAccount(ctx, used.ID)
	wantKind(t, err, Conflict)
	if want := `"Checking" has 2 transactions; delete or move them first`; err.Error() != want {
		t.Errorf("message = %q, want %q", err, want)
	}

	if err := s.DeleteAccount(ctx, empty.ID); err != nil {
		t.Fatal(err)
	}
	got, err := s.ListAccounts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != used.ID {
		t.Errorf("accounts = %+v", got)
	}

	wantKind(t, s.DeleteAccount(ctx, empty.ID), NotFound)
}
