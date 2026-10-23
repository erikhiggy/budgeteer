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
	// Sorted case-insensitively by name.
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
