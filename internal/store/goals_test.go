package store

import (
	"context"
	"testing"
)

func cents(n int64) *int64 { return &n }

func month(m string) *string { return &m }

// goalFor returns category id's goal in month.
func goalFor(t *testing.T, s *Store, id int64, m string) Goal {
	t.Helper()
	all, err := s.ListGoals(context.Background(), m)
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range all.Goals {
		if g.CategoryID == id {
			return g
		}
	}
	t.Fatalf("no goal for category %d in %+v", id, all)
	return Goal{}
}

func setGoal(t *testing.T, s *Store, id int64, m string, c *int64) {
	t.Helper()
	if _, err := s.SetGoal(context.Background(), id, m, c); err != nil {
		t.Fatal(err)
	}
}

func spend(t *testing.T, s *Store, account int64, category *int64, date string, c int64) {
	t.Helper()
	if _, err := s.CreateTransaction(context.Background(), TransactionInput{AccountID: account, CategoryID: category, Date: date, AmountCents: c}); err != nil {
		t.Fatal(err)
	}
}

func eq(p *int64, want int64) bool { return p != nil && *p == want }

func TestCapsByEffectiveMonth(t *testing.T) {
	s, _ := newTestStore(t)
	cat, err := s.CreateCategory(context.Background(), CategoryInput{Name: "Groceries"})
	if err != nil {
		t.Fatal(err)
	}
	setGoal(t, s, cat.ID, "2026-10", cents(50000))
	setGoal(t, s, cat.ID, "2026-12", cents(60000))

	// The done-when: a cap set in Oct and changed in Dec, so Nov shows Oct's.
	cases := []struct {
		month    string
		cap      *int64
		capSince string
	}{
		{"2026-09", nil, ""},
		{"2026-10", cents(50000), "2026-10"},
		{"2026-11", cents(50000), "2026-10"},
		{"2026-12", cents(60000), "2026-12"},
		{"2027-03", cents(60000), "2026-12"},
	}
	for _, c := range cases {
		g := goalFor(t, s, cat.ID, c.month)
		if c.cap == nil {
			if g.CapCents != nil || g.CapSince != nil || g.AvailableCents != nil {
				t.Errorf("%s: goal = %+v, want no cap", c.month, g)
			}
			continue
		}
		if !eq(g.CapCents, *c.cap) || g.CapSince == nil || *g.CapSince != c.capSince || !eq(g.AvailableCents, *c.cap) {
			t.Errorf("%s: goal = %+v, want cap %d since %s", c.month, g, *c.cap, c.capSince)
		}
	}

	// Removing the cap from Feb on leaves earlier months alone.
	setGoal(t, s, cat.ID, "2027-02", nil)
	if g := goalFor(t, s, cat.ID, "2027-02"); g.CapCents != nil {
		t.Errorf("Feb goal = %+v, want no cap", g)
	}
	if g := goalFor(t, s, cat.ID, "2027-01"); !eq(g.CapCents, 60000) {
		t.Errorf("Jan goal = %+v, want 60000", g)
	}

	// Setting the same month again replaces it.
	setGoal(t, s, cat.ID, "2026-10", cents(45000))
	if g := goalFor(t, s, cat.ID, "2026-11"); !eq(g.CapCents, 45000) {
		t.Errorf("Nov goal = %+v, want 45000", g)
	}
}

func TestGoalsSpending(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	checking, savings := twoAccounts(t, s)
	food, err := s.CreateCategory(ctx, CategoryInput{Name: "food"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateCategory(ctx, CategoryInput{Name: "Coffee"}); err != nil {
		t.Fatal(err)
	}

	spend(t, s, checking.ID, &food.ID, "2026-10-01", -5000)
	spend(t, s, savings.ID, &food.ID, "2026-10-31", -2500)  // any account counts
	spend(t, s, checking.ID, &food.ID, "2026-10-15", 1000)  // a refund
	spend(t, s, checking.ID, &food.ID, "2026-11-01", -9999) // next month
	spend(t, s, checking.ID, nil, "2026-10-20", -700)       // uncategorized
	createTestTransfer(t, s, checking.ID, savings.ID, "2026-10-05", -40000)

	all, err := s.ListGoals(ctx, "2026-10")
	if err != nil {
		t.Fatal(err)
	}
	// Expense categories only, sorted by name case-insensitively.
	if all.Month != "2026-10" || len(all.Goals) != 2 || all.Goals[0].Name != "Coffee" || all.Goals[1].Name != "food" {
		t.Fatalf("goals = %+v", all)
	}
	if all.Goals[0].SpentCents != 0 || all.Goals[1].SpentCents != 6500 {
		t.Errorf("spent = %d, %d; want 0, 6500", all.Goals[0].SpentCents, all.Goals[1].SpentCents)
	}
	// The transfer isn't uncategorized spending.
	if all.UncategorizedCents != 700 {
		t.Errorf("uncategorized = %d, want 700", all.UncategorizedCents)
	}
}

func TestRollover(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	checking, _ := twoAccounts(t, s)
	ins, err := s.CreateCategory(ctx, CategoryInput{Name: "Insurance"})
	if err != nil {
		t.Fatal(err)
	}
	setGoal(t, s, ins.ID, "2026-10", cents(10000))
	if _, err := s.UpdateCategory(ctx, ins.ID, CategoryUpdate{SetRollover: true, RolloverSince: month("2026-10")}); err != nil {
		t.Fatal(err)
	}

	// The done-when: $100 a month from Oct with no spending is $300 by Dec.
	for _, c := range []struct {
		month                string
		carryover, available int64
	}{
		{"2026-09", 0, 0}, // before rollover and the cap: checked separately below
		{"2026-10", 0, 10000},
		{"2026-11", 10000, 20000},
		{"2026-12", 20000, 30000},
	} {
		g := goalFor(t, s, ins.ID, c.month)
		if c.month == "2026-09" {
			if g.CarryoverCents != 0 || g.AvailableCents != nil {
				t.Errorf("Sep goal = %+v, want no carryover and no available", g)
			}
			continue
		}
		if g.CarryoverCents != c.carryover || !eq(g.AvailableCents, c.available) {
			t.Errorf("%s: goal = %+v, want carryover %d, available %d", c.month, g, c.carryover, c.available)
		}
	}

	// Overspending in Nov comes out of what rolls over: 100 + (100 - 250).
	spend(t, s, checking.ID, &ins.ID, "2026-11-10", -25000)
	if g := goalFor(t, s, ins.ID, "2026-12"); g.CarryoverCents != -5000 || !eq(g.AvailableCents, 5000) {
		t.Errorf("Dec goal = %+v, want carryover -5000, available 5000", g)
	}
	if g := goalFor(t, s, ins.ID, "2026-11"); g.SpentCents != 25000 || !eq(g.AvailableCents, 20000) {
		t.Errorf("Nov goal = %+v, want spent 25000 of 20000", g)
	}

	// A month with no cap adds nothing but still subtracts its spending.
	setGoal(t, s, ins.ID, "2027-01", nil)
	if g := goalFor(t, s, ins.ID, "2027-02"); g.CapCents != nil || g.CarryoverCents != 5000 || !eq(g.AvailableCents, 5000) {
		t.Errorf("Feb goal = %+v, want no cap and 5000 carried over", g)
	}

	// Turning it on again from Dec starts fresh.
	if _, err := s.UpdateCategory(ctx, ins.ID, CategoryUpdate{SetRollover: true, RolloverSince: month("2026-12")}); err != nil {
		t.Fatal(err)
	}
	if g := goalFor(t, s, ins.ID, "2026-12"); g.CarryoverCents != 0 || !eq(g.AvailableCents, 10000) {
		t.Errorf("Dec goal after restart = %+v, want a fresh 10000", g)
	}

	// Turning it off goes back to the plain cap.
	if _, err := s.UpdateCategory(ctx, ins.ID, CategoryUpdate{SetRollover: true}); err != nil {
		t.Fatal(err)
	}
	if g := goalFor(t, s, ins.ID, "2026-12"); g.RolloverSince != nil || g.CarryoverCents != 0 || !eq(g.AvailableCents, 10000) {
		t.Errorf("Dec goal with rollover off = %+v", g)
	}
}

func TestSetGoalErrors(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	_, err := s.SetGoal(ctx, categoryNamed(t, s, "Income"), "2026-10", cents(100))
	wantKind(t, err, Invalid)
	_, err = s.SetGoal(ctx, 99, "2026-10", cents(100))
	wantKind(t, err, NotFound)
}

func TestMovingToIncomeDropsGoalsAndRollover(t *testing.T) {
	s, conn := newTestStore(t)
	ctx := context.Background()
	c, err := s.CreateCategory(ctx, CategoryInput{Name: "Side gig"})
	if err != nil {
		t.Fatal(err)
	}
	setGoal(t, s, c.ID, "2026-10", cents(5000))
	if _, err := s.UpdateCategory(ctx, c.ID, CategoryUpdate{SetRollover: true, RolloverSince: month("2026-10")}); err != nil {
		t.Fatal(err)
	}

	income := true
	got, err := s.UpdateCategory(ctx, c.ID, CategoryUpdate{IsIncome: &income})
	if err != nil {
		t.Fatal(err)
	}
	if !got.IsIncome || got.RolloverSince != nil {
		t.Errorf("updated = %+v, want income with rollover off", got)
	}
	var n int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM goals`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("%d goals left, want 0", n)
	}

	// Moving it back starts with no cap.
	expense := false
	if _, err := s.UpdateCategory(ctx, c.ID, CategoryUpdate{IsIncome: &expense}); err != nil {
		t.Fatal(err)
	}
	if g := goalFor(t, s, c.ID, "2026-10"); g.CapCents != nil {
		t.Errorf("goal after moving back = %+v, want no cap", g)
	}

	// An income category can't roll over, even when made income in the same update.
	_, err = s.UpdateCategory(ctx, c.ID, CategoryUpdate{IsIncome: &income, SetRollover: true, RolloverSince: month("2026-10")})
	wantKind(t, err, Invalid)
	_, err = s.UpdateCategory(ctx, categoryNamed(t, s, "Income"), CategoryUpdate{SetRollover: true, RolloverSince: month("2026-10")})
	wantKind(t, err, Invalid)
}
