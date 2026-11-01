package store

import (
	"context"
	"testing"
)

func newGroup(t *testing.T, s *Store, name string) CategoryGroup {
	t.Helper()
	g, err := s.CreateCategoryGroup(context.Background(), CategoryGroupInput{Name: name})
	if err != nil {
		t.Fatal(err)
	}
	return g
}

// groupNames returns the names of gs, in order.
func groupNames(gs []CategoryGroup) []string {
	names := []string{}
	for _, g := range gs {
		names = append(names, g.Name)
	}
	return names
}

func wantNames(t *testing.T, gs []CategoryGroup, want ...string) {
	t.Helper()
	got := groupNames(gs)
	if len(got) != len(want) {
		t.Fatalf("groups = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("groups = %v, want %v", got, want)
		}
	}
}

func TestCreateCategoryGroup(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	housing := newGroup(t, s, "Housing")
	wealth := newGroup(t, s, "Wealth building")
	if housing.ID == 0 || housing.Name != "Housing" || housing.SortOrder != 0 {
		t.Errorf("first = %+v, want sort_order 0", housing)
	}
	// Each new group goes last.
	if wealth.SortOrder != 1 {
		t.Errorf("second = %+v, want sort_order 1", wealth)
	}

	_, err := s.CreateCategoryGroup(ctx, CategoryGroupInput{Name: "HOUSING"})
	wantKind(t, err, Conflict)
}

func TestListCategoryGroups(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	got, err := s.ListCategoryGroups(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || len(got) != 0 {
		t.Fatalf("fresh database: groups = %#v, want an empty, non-nil slice", got)
	}

	newGroup(t, s, "Wealth building")
	housing := newGroup(t, s, "Housing")
	fun := newGroup(t, s, "Fun")
	// Equal sort_orders fall back to the name.
	shared := int64(5)
	for _, id := range []int64{housing.ID, fun.ID} {
		if _, err := s.UpdateCategoryGroup(ctx, id, CategoryGroupUpdate{SortOrder: &shared}); err != nil {
			t.Fatal(err)
		}
	}
	got, err = s.ListCategoryGroups(ctx)
	if err != nil {
		t.Fatal(err)
	}
	wantNames(t, got, "Wealth building", "Fun", "Housing")
}

func TestUpdateCategoryGroup(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	housing := newGroup(t, s, "housing")
	newGroup(t, s, "Fun")

	name := "Housing"
	got, err := s.UpdateCategoryGroup(ctx, housing.ID, CategoryGroupUpdate{Name: &name})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != housing.ID || got.Name != "Housing" || got.SortOrder != housing.SortOrder {
		t.Errorf("renamed = %+v", got)
	}

	// Only the fields sent change.
	order := int64(7)
	got, err = s.UpdateCategoryGroup(ctx, housing.ID, CategoryGroupUpdate{SortOrder: &order})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Housing" || got.SortOrder != 7 {
		t.Errorf("reordered = %+v", got)
	}

	// Changing only the case of its own name is fine; taking another group's is not.
	same := "HOUSING"
	if _, err := s.UpdateCategoryGroup(ctx, housing.ID, CategoryGroupUpdate{Name: &same}); err != nil {
		t.Errorf("recasing own name: %v", err)
	}
	taken := "fun"
	_, err = s.UpdateCategoryGroup(ctx, housing.ID, CategoryGroupUpdate{Name: &taken})
	wantKind(t, err, Conflict)

	_, err = s.UpdateCategoryGroup(ctx, 999, CategoryGroupUpdate{Name: &name})
	wantKind(t, err, NotFound)
}

func TestReorderCategoryGroups(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	a := newGroup(t, s, "A")
	b := newGroup(t, s, "B")
	c := newGroup(t, s, "C")

	got, err := s.ReorderCategoryGroups(ctx, []int64{c.ID, a.ID, b.ID})
	if err != nil {
		t.Fatal(err)
	}
	wantNames(t, got, "C", "A", "B")
	for i, g := range got {
		if g.SortOrder != int64(i) {
			t.Errorf("%s sort_order = %d, want %d", g.Name, g.SortOrder, i)
		}
	}
	listed, err := s.ListCategoryGroups(ctx)
	if err != nil {
		t.Fatal(err)
	}
	wantNames(t, listed, "C", "A", "B")

	// A bad list is Invalid, and changes nothing.
	for name, ids := range map[string][]int64{
		"missing a group":   {a.ID, b.ID},
		"repeats a group":   {a.ID, a.ID, b.ID, c.ID},
		"unknown id":        {a.ID, b.ID, 999},
		"empty":             {},
		"extra unknown id":  {a.ID, b.ID, c.ID, 999},
		"swap with unknown": {c.ID, a.ID, 999},
	} {
		_, err := s.ReorderCategoryGroups(ctx, ids)
		if err == nil {
			t.Errorf("%s: want an error", name)
			continue
		}
		wantKind(t, err, Invalid)
	}
	listed, err = s.ListCategoryGroups(ctx)
	if err != nil {
		t.Fatal(err)
	}
	wantNames(t, listed, "C", "A", "B")
}

func TestDeleteCategoryGroupUngroupsItsCategories(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	housing := newGroup(t, s, "Housing")
	fun := newGroup(t, s, "Fun")
	rent, err := s.CreateCategory(ctx, CategoryInput{Name: "Rent", GroupID: &housing.ID})
	if err != nil {
		t.Fatal(err)
	}
	movies, err := s.CreateCategory(ctx, CategoryInput{Name: "Movies", GroupID: &fun.ID})
	if err != nil {
		t.Fatal(err)
	}

	if err := s.DeleteCategoryGroup(ctx, housing.ID); err != nil {
		t.Fatal(err)
	}
	groups, err := s.ListCategoryGroups(ctx)
	if err != nil {
		t.Fatal(err)
	}
	wantNames(t, groups, "Fun")

	// The category survives, ungrouped; the other group's category is untouched.
	got, err := s.getCategory(ctx, rent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.GroupID != nil {
		t.Errorf("rent group_id = %d, want nil", *got.GroupID)
	}
	got, err = s.getCategory(ctx, movies.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !eq(got.GroupID, fun.ID) {
		t.Errorf("movies group_id = %v, want %d", got.GroupID, fun.ID)
	}

	wantKind(t, s.DeleteCategoryGroup(ctx, housing.ID), NotFound)
}

func TestCategoryGroupMembership(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	housing := newGroup(t, s, "Housing")
	fun := newGroup(t, s, "Fun")

	// Create ungrouped, or straight into a group.
	plain, err := s.CreateCategory(ctx, CategoryInput{Name: "Plain"})
	if err != nil {
		t.Fatal(err)
	}
	if plain.GroupID != nil {
		t.Errorf("plain = %+v, want ungrouped", plain)
	}
	rent, err := s.CreateCategory(ctx, CategoryInput{Name: "Rent", GroupID: &housing.ID})
	if err != nil {
		t.Fatal(err)
	}
	if !eq(rent.GroupID, housing.ID) {
		t.Errorf("rent = %+v, want group %d", rent, housing.ID)
	}

	// ListCategories reports it too.
	all, err := s.ListCategories(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range all {
		if c.Name == "Rent" && !eq(c.GroupID, housing.ID) {
			t.Errorf("listed rent = %+v, want group %d", c, housing.ID)
		}
		if c.Name == "Plain" && c.GroupID != nil {
			t.Errorf("listed plain = %+v, want ungrouped", c)
		}
	}

	// Move it, then leave the group alone while changing something else.
	got, err := s.UpdateCategory(ctx, rent.ID, CategoryUpdate{SetGroup: true, GroupID: &fun.ID})
	if err != nil {
		t.Fatal(err)
	}
	if !eq(got.GroupID, fun.ID) {
		t.Errorf("moved = %+v, want group %d", got, fun.ID)
	}
	name := "Rent (home)"
	got, err = s.UpdateCategory(ctx, rent.ID, CategoryUpdate{Name: &name})
	if err != nil {
		t.Fatal(err)
	}
	if !eq(got.GroupID, fun.ID) {
		t.Errorf("renamed = %+v, group should be unchanged", got)
	}

	// SetGroup with a nil group ungroups.
	got, err = s.UpdateCategory(ctx, rent.ID, CategoryUpdate{SetGroup: true})
	if err != nil {
		t.Fatal(err)
	}
	if got.GroupID != nil {
		t.Errorf("ungrouped = %+v", got)
	}

	// An unknown group is NotFound.
	missing := int64(999)
	_, err = s.CreateCategory(ctx, CategoryInput{Name: "Lost", GroupID: &missing})
	wantKind(t, err, NotFound)
	_, err = s.UpdateCategory(ctx, rent.ID, CategoryUpdate{SetGroup: true, GroupID: &missing})
	wantKind(t, err, NotFound)
}

func TestIncomeCategoriesCantBeGrouped(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	housing := newGroup(t, s, "Housing")

	_, err := s.CreateCategory(ctx, CategoryInput{Name: "Bonus", IsIncome: true, GroupID: &housing.ID})
	wantKind(t, err, Invalid)

	salary, err := s.CreateCategory(ctx, CategoryInput{Name: "Salary", IsIncome: true})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.UpdateCategory(ctx, salary.ID, CategoryUpdate{SetGroup: true, GroupID: &housing.ID})
	wantKind(t, err, Invalid)

	// Making a grouped category income in the same request as grouping it is Invalid too.
	rent, err := s.CreateCategory(ctx, CategoryInput{Name: "Rent"})
	if err != nil {
		t.Fatal(err)
	}
	income := true
	_, err = s.UpdateCategory(ctx, rent.ID, CategoryUpdate{IsIncome: &income, SetGroup: true, GroupID: &housing.ID})
	wantKind(t, err, Invalid)

	// Moving a grouped category to income ungroups it.
	grouped, err := s.CreateCategory(ctx, CategoryInput{Name: "Side gig", GroupID: &housing.ID})
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.UpdateCategory(ctx, grouped.ID, CategoryUpdate{IsIncome: &income})
	if err != nil {
		t.Fatal(err)
	}
	if !got.IsIncome || got.GroupID != nil {
		t.Errorf("moved to income = %+v, want ungrouped income", got)
	}
}

func TestEnvelopeGroups(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	checking, err := s.CreateAccount(ctx, AccountInput{Name: "Checking", Type: "checking", OpeningBalanceCents: 100000, OpeningBalanceDate: "2026-09-15"})
	if err != nil {
		t.Fatal(err)
	}
	housing := newGroup(t, s, "Housing")
	food := newGroup(t, s, "Food")
	newGroup(t, s, "Empty")
	mk := func(name string, g *CategoryGroup) Category {
		t.Helper()
		in := CategoryInput{Name: name}
		if g != nil {
			in.GroupID = &g.ID
		}
		c, err := s.CreateCategory(ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	rent := mk("Rent", &housing)
	utilities := mk("Utilities", &housing)
	groceries := mk("Groceries", &food)
	coffee := mk("Coffee", nil)

	// Before envelopes start: every group is listed, with zeros.
	before := listEnvelopes(t, s, "2026-10")
	if len(before.Groups) != 3 || before.Groups[0].Name != "Housing" || before.Groups[0].AssignedCents != 0 {
		t.Fatalf("not started: groups = %+v", before.Groups)
	}

	if err := s.SetEnvelopesStart(ctx, month("2026-10")); err != nil {
		t.Fatal(err)
	}
	assign(t, s, rent.ID, "2026-10", 80000)
	assign(t, s, utilities.ID, "2026-10", 10000)
	assign(t, s, groceries.ID, "2026-10", 40000)
	assign(t, s, coffee.ID, "2026-10", 3000)
	spend(t, s, checking.ID, &rent.ID, "2026-10-01", -75000)
	spend(t, s, checking.ID, &groceries.ID, "2026-10-05", -45000)

	oct := listEnvelopes(t, s, "2026-10")
	// In sort order, with the sums over each group's envelopes.
	want := []EnvelopeGroup{
		{ID: housing.ID, Name: "Housing", AssignedCents: 90000, SpentCents: 75000, BalanceCents: 15000},
		{ID: food.ID, Name: "Food", AssignedCents: 40000, SpentCents: 45000, BalanceCents: -5000},
		{ID: oct.Groups[2].ID, Name: "Empty"},
	}
	if len(oct.Groups) != len(want) {
		t.Fatalf("groups = %+v, want %+v", oct.Groups, want)
	}
	for i := range want {
		if oct.Groups[i] != want[i] {
			t.Errorf("group %d = %+v, want %+v", i, oct.Groups[i], want[i])
		}
	}

	// Each envelope knows its group; ungrouped ones have none and are in no group's totals.
	if e := envelopeFor(t, oct, rent.ID); !eq(e.GroupID, housing.ID) {
		t.Errorf("rent envelope = %+v, want group %d", e, housing.ID)
	}
	if e := envelopeFor(t, oct, coffee.ID); e.GroupID != nil {
		t.Errorf("coffee envelope = %+v, want no group", e)
	}

	// Balances include carryover; assigned is the month's own.
	nov := listEnvelopes(t, s, "2026-11")
	if h := nov.Groups[0]; h.AssignedCents != 0 || h.SpentCents != 0 || h.BalanceCents != 15000 {
		t.Errorf("Nov housing = %+v", h)
	}
	if f := nov.Groups[1]; f.BalanceCents != -5000 {
		t.Errorf("Nov food = %+v", f)
	}
}
