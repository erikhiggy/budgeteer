package store

import (
	"context"
	"testing"
)

func TestCreateAndListCategories(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	for _, in := range []CategoryInput{
		{Name: "Eating Out", IsIncome: false},
		{Name: "Coffee", IsIncome: false},
		{Name: "Rent", IsIncome: false},
	} {
		if _, err := s.CreateCategory(ctx, in); err != nil {
			t.Fatal(err)
		}
	}

	got, err := s.ListCategories(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// Expense first, then income, each sorted case-insensitively by name
	if len(got) != 5 || got[0].Name != "Coffee" || got[1].Name != "Eating Out" || got[2].Name != "Rent" || got[3].Name != "Income" || got[4].Name != "Opening balance" {
		t.Fatalf("categories = %+v", got)
	}
}

func TestListCategoriesSeededWithTwoBaseCategories(t *testing.T) {
	s, _ := newTestStore(t)
	got, err := s.ListCategories(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Name != "Income" || got[1].Name != "Opening balance" {
		t.Fatalf("categories = %+v", got)
	}
}

func TestCreateCategoryDuplicateNameIsConflict(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	if _, err := s.CreateCategory(ctx, CategoryInput{Name: "Coffee", IsIncome: false}); err != nil {
		t.Fatal(err)
	}
	_, err := s.CreateCategory(ctx, CategoryInput{Name: "COFFEE", IsIncome: false})
	wantKind(t, err, Conflict)
}

// categoryNamed returns the id of the category called name.
func categoryNamed(t *testing.T, s *Store, name string) int64 {
	t.Helper()
	all, err := s.ListCategories(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range all {
		if c.Name == name {
			return c.ID
		}
	}
	t.Fatalf("no category %q in %+v", name, all)
	return 0
}

func TestUpdateCategory(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	c, err := s.CreateCategory(ctx, CategoryInput{Name: "coffee"})
	if err != nil {
		t.Fatal(err)
	}

	name := "Coffee"
	got, err := s.UpdateCategory(ctx, c.ID, CategoryUpdate{Name: &name})
	if err != nil {
		t.Fatal(err)
	}
	if got != (Category{ID: c.ID, Name: "Coffee"}) {
		t.Errorf("updated = %+v", got)
	}

	income := true
	got, err = s.UpdateCategory(ctx, c.ID, CategoryUpdate{IsIncome: &income})
	if err != nil {
		t.Fatal(err)
	}
	if got != (Category{ID: c.ID, Name: "Coffee", IsIncome: true}) {
		t.Errorf("updated = %+v", got)
	}

	taken := "income"
	_, err = s.UpdateCategory(ctx, c.ID, CategoryUpdate{Name: &taken})
	wantKind(t, err, Conflict)

	_, err = s.UpdateCategory(ctx, 99, CategoryUpdate{Name: &name})
	wantKind(t, err, NotFound)
}

func TestOpeningBalanceCategoryIsLocked(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	id := categoryNamed(t, s, OpeningBalanceCategory)

	renamed := "Starting balance"
	_, err := s.UpdateCategory(ctx, id, CategoryUpdate{Name: &renamed})
	wantKind(t, err, Conflict)
	wantKind(t, s.DeleteCategory(ctx, id), Conflict)

	// Sending its current name, or changing only is_income, is allowed.
	same := OpeningBalanceCategory
	income := true
	if _, err := s.UpdateCategory(ctx, id, CategoryUpdate{Name: &same, IsIncome: &income}); err != nil {
		t.Errorf("update with unchanged name: %v", err)
	}

	// Income is an ordinary category.
	if err := s.DeleteCategory(ctx, categoryNamed(t, s, "Income")); err != nil {
		t.Errorf("delete Income: %v", err)
	}
}

func TestDeleteCategory(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	a, err := s.CreateAccount(ctx, AccountInput{Name: "Checking", Type: "checking"})
	if err != nil {
		t.Fatal(err)
	}
	used, err := s.CreateCategory(ctx, CategoryInput{Name: "Coffee"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateTransaction(ctx, TransactionInput{AccountID: a.ID, CategoryID: &used.ID, Date: "2026-10-01", AmountCents: -450}); err != nil {
		t.Fatal(err)
	}
	unused, err := s.CreateCategory(ctx, CategoryInput{Name: "Gifts"})
	if err != nil {
		t.Fatal(err)
	}

	err = s.DeleteCategory(ctx, used.ID)
	wantKind(t, err, Conflict)
	if want := `"Coffee" has 1 transaction; delete or recategorize them first`; err.Error() != want {
		t.Errorf("message = %q, want %q", err, want)
	}

	if err := s.DeleteCategory(ctx, unused.ID); err != nil {
		t.Fatal(err)
	}
	wantKind(t, s.DeleteCategory(ctx, unused.ID), NotFound)
}
