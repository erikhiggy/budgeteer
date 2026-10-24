package store

import (
	"context"
	"testing"
)

// reportFixture spends across three months in Food, Coffee, an income
// category and uncategorized, plus a transfer that must never count.
func reportFixture(t *testing.T, s *Store) (food, coffee Category) {
	t.Helper()
	ctx := context.Background()
	checking, savings := twoAccounts(t, s)
	food, err := s.CreateCategory(ctx, CategoryInput{Name: "food"})
	if err != nil {
		t.Fatal(err)
	}
	coffee, err = s.CreateCategory(ctx, CategoryInput{Name: "Coffee"})
	if err != nil {
		t.Fatal(err)
	}
	pay, err := s.CreateCategory(ctx, CategoryInput{Name: "Paycheck", IsIncome: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateCategory(ctx, CategoryInput{Name: "Unused"}); err != nil {
		t.Fatal(err)
	}

	spend(t, s, checking.ID, &food.ID, "2026-08-10", -3000)
	spend(t, s, checking.ID, &food.ID, "2026-10-01", -5000)
	spend(t, s, savings.ID, &food.ID, "2026-10-31", -2500)
	spend(t, s, checking.ID, &food.ID, "2026-10-15", 1000) // a refund
	spend(t, s, checking.ID, &coffee.ID, "2026-10-02", -700)
	spend(t, s, checking.ID, nil, "2026-10-20", -700)       // uncategorized
	spend(t, s, checking.ID, &pay.ID, "2026-10-25", 300000) // income, not spending
	spend(t, s, checking.ID, &coffee.ID, "2026-11-03", 400) // refund-only month
	createTestTransfer(t, s, checking.ID, savings.ID, "2026-10-05", -40000)
	return food, coffee
}

func TestSpendingByCategory(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	food, coffee := reportFixture(t, s)

	got, err := s.SpendingByCategory(ctx, "2026-10")
	if err != nil {
		t.Fatal(err)
	}
	// Biggest first; Coffee and Uncategorized tie at 700, so by name. Income,
	// transfers and Unused (no spending) are left out.
	want := []CategorySpending{
		{&food.ID, "food", 6500},
		{&coffee.ID, "Coffee", 700},
		{nil, UncategorizedName, 700},
	}
	if got.Month != "2026-10" || len(got.Categories) != len(want) || got.TotalCents != 7900 {
		t.Fatalf("report = %+v", got)
	}
	for i, w := range want {
		g := got.Categories[i]
		if g.Name != w.Name || g.SpentCents != w.SpentCents || (g.CategoryID == nil) != (w.CategoryID == nil) ||
			(g.CategoryID != nil && *g.CategoryID != *w.CategoryID) {
			t.Errorf("categories[%d] = %+v, want %+v", i, g, w)
		}
	}

	// Negative spending is kept.
	got, err = s.SpendingByCategory(ctx, "2026-11")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Categories) != 1 || got.Categories[0].SpentCents != -400 || got.TotalCents != -400 {
		t.Errorf("Nov = %+v", got)
	}

	// A month with nothing is an empty list, not null.
	got, err = s.SpendingByCategory(ctx, "2027-01")
	if err != nil {
		t.Fatal(err)
	}
	if got.Categories == nil || len(got.Categories) != 0 || got.TotalCents != 0 {
		t.Errorf("Jan = %+v", got)
	}
}

func TestMonthlySpending(t *testing.T) {
	s, _ := newTestStore(t)
	reportFixture(t, s)

	got, err := s.MonthlySpending(context.Background(), "2026-07", "2027-01")
	if err != nil {
		t.Fatal(err)
	}
	want := []MonthSpending{
		{"2026-07", 0},
		{"2026-08", 3000},
		{"2026-09", 0},
		{"2026-10", 7900},
		{"2026-11", -400},
		{"2026-12", 0},
		{"2027-01", 0},
	}
	if got.From != "2026-07" || got.To != "2027-01" || len(got.Months) != len(want) {
		t.Fatalf("report = %+v", got)
	}
	for i, w := range want {
		if got.Months[i] != w {
			t.Errorf("months[%d] = %+v, want %+v", i, got.Months[i], w)
		}
	}
}

// The M5 done-when: report totals equal the /api/goals spent values.
func TestReportsMatchGoals(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	reportFixture(t, s)

	trend, err := s.MonthlySpending(ctx, "2026-07", "2027-01")
	if err != nil {
		t.Fatal(err)
	}
	for _, ms := range trend.Months {
		goals, err := s.ListGoals(ctx, ms.Month)
		if err != nil {
			t.Fatal(err)
		}
		byCat, err := s.SpendingByCategory(ctx, ms.Month)
		if err != nil {
			t.Fatal(err)
		}

		spent := map[int64]int64{} // 0 = uncategorized
		for _, c := range byCat.Categories {
			var id int64
			if c.CategoryID != nil {
				id = *c.CategoryID
			}
			spent[id] = c.SpentCents
		}
		total := goals.UncategorizedCents
		if spent[0] != goals.UncategorizedCents {
			t.Errorf("%s uncategorized: report %d, goals %d", ms.Month, spent[0], goals.UncategorizedCents)
		}
		for _, g := range goals.Goals {
			total += g.SpentCents
			if spent[g.CategoryID] != g.SpentCents {
				t.Errorf("%s %s: report %d, goals %d", ms.Month, g.Name, spent[g.CategoryID], g.SpentCents)
			}
		}
		if byCat.TotalCents != total || ms.SpentCents != total {
			t.Errorf("%s total: by category %d, trend %d, goals %d", ms.Month, byCat.TotalCents, ms.SpentCents, total)
		}
	}
}
