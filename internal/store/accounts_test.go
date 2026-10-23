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
