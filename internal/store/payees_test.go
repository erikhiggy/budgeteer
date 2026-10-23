package store

import (
	"context"
	"testing"
)

func TestListPayeesLatestPerPayee(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	a, err := s.CreateAccount(ctx, AccountInput{Name: "Checking", Type: "checking"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.CreateAccount(ctx, AccountInput{Name: "Visa", Type: "credit"})
	if err != nil {
		t.Fatal(err)
	}
	coffee, err := s.CreateCategory(ctx, CategoryInput{Name: "Coffee"})
	if err != nil {
		t.Fatal(err)
	}
	food, err := s.CreateCategory(ctx, CategoryInput{Name: "Food"})
	if err != nil {
		t.Fatal(err)
	}

	for _, in := range []TransactionInput{
		{AccountID: a.ID, CategoryID: &coffee.ID, Date: "2026-10-01", Payee: "cafe"},
		{AccountID: b.ID, CategoryID: &food.ID, Date: "2026-10-05", Payee: "Cafe"}, // other account
		// Same day as the one above, but a later id: this is the latest.
		{AccountID: a.ID, CategoryID: &coffee.ID, Date: "2026-10-05", Payee: "CAFE"},
		{AccountID: a.ID, CategoryID: &food.ID, Date: "2026-10-04", Payee: "Rent"},
		{AccountID: a.ID, CategoryID: nil, Date: "2026-10-06", Payee: "Rent"},
		{AccountID: a.ID, CategoryID: &food.ID, Date: "2026-10-07", Payee: ""},
		{AccountID: a.ID, CategoryID: &food.ID, Date: "2026-09-01", Payee: "bakery"},
	} {
		in.AmountCents = -100
		if _, err := s.CreateTransaction(ctx, in); err != nil {
			t.Fatal(err)
		}
	}

	got, err := s.ListPayees(ctx)
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		payee      string
		categoryID *int64
	}{{"bakery", &food.ID}, {"CAFE", &coffee.ID}, {"Rent", nil}}
	if len(got) != len(want) {
		t.Fatalf("payees = %+v", got)
	}
	for i, w := range want {
		if got[i].Payee != w.payee || (got[i].CategoryID == nil) != (w.categoryID == nil) ||
			(w.categoryID != nil && *got[i].CategoryID != *w.categoryID) {
			t.Errorf("payees[%d] = %+v, want %s with category %v", i, got[i], w.payee, w.categoryID)
		}
	}
}

func TestListPayeesEmptyIsNotNil(t *testing.T) {
	s, _ := newTestStore(t)
	got, err := s.ListPayees(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || len(got) != 0 {
		t.Errorf("payees = %#v, want empty slice", got)
	}
}
