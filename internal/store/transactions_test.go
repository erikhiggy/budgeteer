package store

import (
	"context"
	"testing"
)

func TestListTransactionsNewestFirst(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	a, err := s.CreateAccount(ctx, AccountInput{Name: "Checking", Type: "checking"})
	if err != nil {
		t.Fatal(err)
	}
	other, err := s.CreateAccount(ctx, AccountInput{Name: "Savings", Type: "savings"})
	if err != nil {
		t.Fatal(err)
	}
	cat, err := s.CreateCategory(ctx, CategoryInput{Name: "Coffee"})
	if err != nil {
		t.Fatal(err)
	}

	oldest, _ := s.CreateTransaction(ctx, TransactionInput{AccountID: a.ID, CategoryID: &cat.ID, Date: "2026-09-30", Payee: "Payee", Note: "Note", AmountCents: -450})
	sameDayFirst, _ := s.CreateTransaction(ctx, TransactionInput{AccountID: a.ID, CategoryID: nil, Date: "2026-10-05", Payee: "Payee", Note: "Note", AmountCents: -1000})
	sameDaySecond, _ := s.CreateTransaction(ctx, TransactionInput{AccountID: a.ID, CategoryID: &cat.ID, Date: "2026-10-05", Payee: "Payee", Note: "Note", AmountCents: -2000})
	s.CreateTransaction(ctx, TransactionInput{AccountID: other.ID, CategoryID: nil, Date: "2026-10-06", Payee: "Payee", Note: "Note", AmountCents: 5000})

	got, err := s.ListTransactions(ctx, TransactionFilter{AccountID: a.ID})
	if err != nil {
		t.Fatal(err)
	}
	// date DESC, then id DESC within a day; the other account's row is excluded.
	if len(got) != 3 || got[0].ID != sameDaySecond.ID || got[1].ID != sameDayFirst.ID || got[2].ID != oldest.ID {
		t.Fatalf("transactions = %+v", got)
	}

	want := Transaction{ID: sameDaySecond.ID, AccountID: a.ID, CategoryID: got[0].CategoryID, Date: "2026-10-05", Payee: "Payee", AmountCents: -2000, Note: "Note"}
	if got[0] != want || got[0].CategoryID == nil || *got[0].CategoryID != cat.ID {
		t.Errorf("transaction = %+v, want %+v with category %d", got[0], want, cat.ID)
	}
	if got[1].CategoryID != nil {
		t.Errorf("uncategorized transaction has category_id %d", *got[1].CategoryID)
	}
}

func TestListTransactionsDateRangeIsInclusive(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	a, err := s.CreateAccount(ctx, AccountInput{Name: "Checking", Type: "checking"})
	if err != nil {
		t.Fatal(err)
	}
	for _, date := range []string{"2026-09-30", "2026-10-01", "2026-10-15", "2026-10-31", "2026-11-01"} {
		s.CreateTransaction(ctx, TransactionInput{AccountID: a.ID, CategoryID: nil, Date: date, AmountCents: -100})
	}

	cases := []struct {
		name     string
		from, to string
		want     []string
	}{
		{"both ends", "2026-10-01", "2026-10-31", []string{"2026-10-31", "2026-10-15", "2026-10-01"}},
		{"from only", "2026-10-31", "", []string{"2026-11-01", "2026-10-31"}},
		{"to only", "", "2026-09-30", []string{"2026-09-30"}},
		{"no matches", "2027-01-01", "", []string{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := s.ListTransactions(ctx, TransactionFilter{AccountID: a.ID, From: c.from, To: c.to})
			if err != nil {
				t.Fatal(err)
			}
			if got == nil {
				t.Fatal("got nil slice, want empty slice")
			}
			dates := []string{}
			for _, tx := range got {
				dates = append(dates, tx.Date)
			}
			if len(dates) != len(c.want) {
				t.Fatalf("dates = %v, want %v", dates, c.want)
			}
			for i := range dates {
				if dates[i] != c.want[i] {
					t.Fatalf("dates = %v, want %v", dates, c.want)
				}
			}
		})
	}
}

func TestListTransactionsUnknownAccountIsNotFound(t *testing.T) {
	s, _ := newTestStore(t)
	_, err := s.ListTransactions(context.Background(), TransactionFilter{AccountID: 99})
	wantKind(t, err, NotFound)
}

func TestCreateTransactionRoundTrip(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	a, err := s.CreateAccount(ctx, AccountInput{Name: "Checking", Type: "checking"})
	if err != nil {
		t.Fatal(err)
	}

	created, err := s.CreateTransaction(ctx, TransactionInput{AccountID: a.ID, Date: "2026-10-05", Payee: "Cafe", AmountCents: -450, Note: "latte"})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == 0 || created.CategoryID != nil {
		t.Errorf("created = %+v", created)
	}

	got, err := s.ListTransactions(ctx, TransactionFilter{AccountID: a.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != created {
		t.Errorf("list = %+v, want [%+v]", got, created)
	}
}

func TestCreateTransactionUnknownAccountIsInvalid(t *testing.T) {
	s, _ := newTestStore(t)
	_, err := s.CreateTransaction(context.Background(), TransactionInput{AccountID: 99, Date: "2026-10-05", AmountCents: -100})
	wantKind(t, err, Invalid)
	if err.Error() != "unknown account_id 99" {
		t.Errorf("message = %q", err)
	}
}

func TestCreateTransactionUnknownCategoryIsInvalid(t *testing.T) {
	s, _ := newTestStore(t)

	// first, add a real account
	a, err := s.CreateAccount(context.Background(), AccountInput{Name: "real", Type: "checking"})
	if err != nil {
		t.Fatal(err)
	}

	cat := Category{ID: 99}
	_, err = s.CreateTransaction(context.Background(), TransactionInput{AccountID: a.ID, CategoryID: &cat.ID, Date: "2026-10-05", AmountCents: -100})
	wantKind(t, err, Invalid)
	if err.Error() != "unknown category_id 99" {
		t.Errorf("message = %q", err)
	}
}

func TestUpdateTransaction(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	a, err := s.CreateAccount(ctx, AccountInput{Name: "Checking", Type: "checking"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.CreateAccount(ctx, AccountInput{Name: "Savings", Type: "savings"})
	if err != nil {
		t.Fatal(err)
	}
	cat, err := s.CreateCategory(ctx, CategoryInput{Name: "Coffee"})
	if err != nil {
		t.Fatal(err)
	}
	created, err := s.CreateTransaction(ctx, TransactionInput{AccountID: a.ID, CategoryID: &cat.ID, Date: "2026-10-01", Payee: "Cafe", AmountCents: -450})
	if err != nil {
		t.Fatal(err)
	}

	// Every field is replaced, including moving accounts and uncategorizing.
	in := TransactionInput{AccountID: b.ID, CategoryID: nil, Date: "2026-10-02", Payee: "Bakery", AmountCents: -600, Note: "bread"}
	got, err := s.UpdateTransaction(ctx, created.ID, in)
	if err != nil {
		t.Fatal(err)
	}
	if want := in.transaction(created.ID, false); got != want {
		t.Errorf("updated = %+v, want %+v", got, want)
	}
	list, err := s.ListTransactions(ctx, TransactionFilter{AccountID: b.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0] != got {
		t.Errorf("list = %+v, want [%+v]", list, got)
	}

	_, err = s.UpdateTransaction(ctx, 99, in)
	wantKind(t, err, NotFound)

	in.AccountID = 99
	_, err = s.UpdateTransaction(ctx, created.ID, in)
	wantKind(t, err, Invalid)

	in.AccountID = a.ID
	bad := int64(42)
	in.CategoryID = &bad
	_, err = s.UpdateTransaction(ctx, created.ID, in)
	wantKind(t, err, Invalid)
	if err.Error() != "unknown category_id 42" {
		t.Errorf("message = %q", err)
	}
}

func TestDeleteTransaction(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	a, err := s.CreateAccount(ctx, AccountInput{Name: "Checking", Type: "checking"})
	if err != nil {
		t.Fatal(err)
	}
	created, err := s.CreateTransaction(ctx, TransactionInput{AccountID: a.ID, Date: "2026-10-01", AmountCents: -450})
	if err != nil {
		t.Fatal(err)
	}

	if err := s.DeleteTransaction(ctx, created.ID); err != nil {
		t.Fatal(err)
	}
	list, err := s.ListTransactions(ctx, TransactionFilter{AccountID: a.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Errorf("list = %+v, want empty", list)
	}
	wantKind(t, s.DeleteTransaction(ctx, created.ID), NotFound)
}

func TestNewTransactionsAreUncleared(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	a, err := s.CreateAccount(ctx, AccountInput{Name: "Checking", Type: "checking"})
	if err != nil {
		t.Fatal(err)
	}

	created, err := s.CreateTransaction(ctx, TransactionInput{AccountID: a.ID, Date: "2026-11-01", AmountCents: -450})
	if err != nil {
		t.Fatal(err)
	}
	if created.Cleared {
		t.Error("created transaction is cleared, want uncleared")
	}
	got, err := s.ListTransactions(ctx, TransactionFilter{AccountID: a.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Cleared {
		t.Errorf("listed = %+v, want one uncleared transaction", got)
	}
}

func TestSetCleared(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	a, err := s.CreateAccount(ctx, AccountInput{Name: "Checking", Type: "checking"})
	if err != nil {
		t.Fatal(err)
	}
	other, err := s.CreateTransaction(ctx, TransactionInput{AccountID: a.ID, Date: "2026-11-01", AmountCents: -100})
	if err != nil {
		t.Fatal(err)
	}
	tx, err := s.CreateTransaction(ctx, TransactionInput{AccountID: a.ID, Date: "2026-11-02", Payee: "Cafe", Note: "latte", AmountCents: -450})
	if err != nil {
		t.Fatal(err)
	}

	got, err := s.SetCleared(ctx, tx.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	want := tx
	want.Cleared = true
	if got != want {
		t.Errorf("SetCleared = %+v, want %+v", got, want)
	}

	// Setting the same value again changes nothing.
	if again, err := s.SetCleared(ctx, tx.ID, true); err != nil || again != want {
		t.Errorf("repeat SetCleared = %+v, %v; want %+v", again, err, want)
	}

	// Only that row changed.
	listed, err := s.ListTransactions(ctx, TransactionFilter{AccountID: a.ID})
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range listed {
		if wantCleared := l.ID == tx.ID; l.Cleared != wantCleared {
			t.Errorf("transaction %d cleared = %v, want %v", l.ID, l.Cleared, wantCleared)
		}
	}
	if other.Cleared {
		t.Fatal("test setup: other started cleared")
	}

	// And it can be cleared back.
	back, err := s.SetCleared(ctx, tx.ID, false)
	if err != nil || back.Cleared {
		t.Errorf("unclearing = %+v, %v; want uncleared", back, err)
	}
}

func TestSetClearedUnknownIDIsNotFound(t *testing.T) {
	s, _ := newTestStore(t)
	_, err := s.SetCleared(context.Background(), 999, true)
	wantKind(t, err, NotFound)
}

func TestUpdateTransactionKeepsCleared(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	a, err := s.CreateAccount(ctx, AccountInput{Name: "Checking", Type: "checking"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.CreateAccount(ctx, AccountInput{Name: "Savings", Type: "savings"})
	if err != nil {
		t.Fatal(err)
	}
	tx, err := s.CreateTransaction(ctx, TransactionInput{AccountID: a.ID, Date: "2026-11-01", AmountCents: -450})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetCleared(ctx, tx.ID, true); err != nil {
		t.Fatal(err)
	}

	// Editing every field, even moving accounts, leaves the flag alone.
	got, err := s.UpdateTransaction(ctx, tx.ID, TransactionInput{AccountID: b.ID, Date: "2026-11-03", Payee: "Bakery", AmountCents: -600, Note: "bread"})
	if err != nil {
		t.Fatal(err)
	}
	if !got.Cleared {
		t.Errorf("updated = %+v, want it to stay cleared", got)
	}
	listed, err := s.ListTransactions(ctx, TransactionFilter{AccountID: b.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || !listed[0].Cleared || listed[0].AmountCents != -600 {
		t.Errorf("listed = %+v, want the edited transaction, still cleared", listed)
	}
}
