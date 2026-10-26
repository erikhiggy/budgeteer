package store

import (
	"context"
	"testing"
)

// envelopeFor returns category id's envelope in envs.
func envelopeFor(t *testing.T, envs Envelopes, id int64) Envelope {
	t.Helper()
	for _, e := range envs.Envelopes {
		if e.CategoryID == id {
			return e
		}
	}
	t.Fatalf("no envelope for category %d in %+v", id, envs)
	return Envelope{}
}

func listEnvelopes(t *testing.T, s *Store, m string) Envelopes {
	t.Helper()
	envs, err := s.ListEnvelopes(context.Background(), m)
	if err != nil {
		t.Fatal(err)
	}
	return envs
}

func assign(t *testing.T, s *Store, id int64, m string, c int64) {
	t.Helper()
	if _, err := s.SetAssignment(context.Background(), id, m, c); err != nil {
		t.Fatal(err)
	}
}

func TestEnvelopes(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	checking, err := s.CreateAccount(ctx, AccountInput{Name: "Checking", Type: "checking", OpeningBalanceCents: 100000, OpeningBalanceDate: "2026-09-15"})
	if err != nil {
		t.Fatal(err)
	}
	// Investments are off budget, so their money can't be assigned.
	if _, err := s.CreateAccount(ctx, AccountInput{Name: "Brokerage", Type: "investment", OpeningBalanceCents: 500000, OpeningBalanceDate: "2026-09-15"}); err != nil {
		t.Fatal(err)
	}
	groceries, _ := s.CreateCategory(ctx, CategoryInput{Name: "Groceries"})
	rent, _ := s.CreateCategory(ctx, CategoryInput{Name: "Rent"})
	salary, _ := s.CreateCategory(ctx, CategoryInput{Name: "Salary", IsIncome: true})

	// Before envelopes start, nothing is counted and nothing can be assigned.
	if envs := listEnvelopes(t, s, "2026-10"); envs.StartMonth != nil || envs.ReadyToAssignCents != 0 || len(envs.Envelopes) != 2 {
		t.Fatalf("not started = %+v", envs)
	}
	_, err = s.SetAssignment(ctx, groceries.ID, "2026-10", 100)
	wantKind(t, err, Conflict)

	spend(t, s, checking.ID, &groceries.ID, "2026-09-20", -5000) // before the start: just lowers the balance
	if err := s.SetEnvelopesStart(ctx, month("2026-10")); err != nil {
		t.Fatal(err)
	}
	spend(t, s, checking.ID, &salary.ID, "2026-10-01", 200000)
	spend(t, s, checking.ID, &groceries.ID, "2026-10-05", -45000)
	assign(t, s, groceries.ID, "2026-10", 40000)
	assign(t, s, rent.ID, "2026-10", 120000)
	assign(t, s, groceries.ID, "2026-11", 30000)

	// Funds: 950 at the start plus 2,000 income. Nov's 300 is already spoken for.
	oct := listEnvelopes(t, s, "2026-10")
	if oct.StartMonth == nil || *oct.StartMonth != "2026-10" || oct.ReadyToAssignCents != 105000 || oct.IncomeCents != 200000 || oct.AssignedCents != 160000 {
		t.Errorf("Oct totals = %+v", oct)
	}
	if g := envelopeFor(t, oct, groceries.ID); g.CarryoverCents != 0 || g.AssignedCents != 40000 || g.SpentCents != 45000 || g.BalanceCents != -5000 {
		t.Errorf("Oct groceries = %+v", g)
	}

	// Overspending carries into the next month; so does what's left.
	nov := listEnvelopes(t, s, "2026-11")
	if nov.ReadyToAssignCents != 105000 || nov.IncomeCents != 0 || nov.AssignedCents != 30000 {
		t.Errorf("Nov totals = %+v", nov)
	}
	if g := envelopeFor(t, nov, groceries.ID); g.CarryoverCents != -5000 || g.AssignedCents != 30000 || g.BalanceCents != 25000 {
		t.Errorf("Nov groceries = %+v", g)
	}
	if r := envelopeFor(t, nov, rent.ID); r.CarryoverCents != 120000 || r.BalanceCents != 120000 {
		t.Errorf("Nov rent = %+v", r)
	}

	// Months before the start show only zeros, and can't be assigned.
	if sep := listEnvelopes(t, s, "2026-09"); sep.ReadyToAssignCents != 0 || envelopeFor(t, sep, groceries.ID).SpentCents != 0 {
		t.Errorf("Sep = %+v", sep)
	}
	_, err = s.SetAssignment(ctx, groceries.ID, "2026-09", 100)
	wantKind(t, err, Invalid)
	_, err = s.SetAssignment(ctx, salary.ID, "2026-10", 100)
	wantKind(t, err, Invalid)
	_, err = s.SetAssignment(ctx, 999, "2026-10", 100)
	wantKind(t, err, NotFound)

	// 0 removes an assignment, and a negative one takes money back out.
	envs, err := s.SetAssignment(ctx, rent.ID, "2026-10", 0)
	if err != nil {
		t.Fatal(err)
	}
	if envs.ReadyToAssignCents != 225000 || envelopeFor(t, envs, rent.ID).AssignedCents != 0 {
		t.Errorf("after removing rent = %+v", envs)
	}
	assign(t, s, rent.ID, "2026-11", -1000)
	if r := envelopeFor(t, listEnvelopes(t, s, "2026-11"), rent.ID); r.BalanceCents != -1000 {
		t.Errorf("negative assignment: rent = %+v", r)
	}

	// Moving the start later ignores earlier assignments without deleting them.
	if err := s.SetEnvelopesStart(ctx, month("2026-11")); err != nil {
		t.Fatal(err)
	}
	if g := envelopeFor(t, listEnvelopes(t, s, "2026-11"), groceries.ID); g.CarryoverCents != 0 || g.BalanceCents != 30000 {
		t.Errorf("start Nov: groceries = %+v", g)
	}
	if err := s.SetEnvelopesStart(ctx, month("2026-10")); err != nil {
		t.Fatal(err)
	}
	if g := envelopeFor(t, listEnvelopes(t, s, "2026-11"), groceries.ID); g.CarryoverCents != -5000 {
		t.Errorf("start back to Oct: groceries = %+v", g)
	}

	// Stopping deletes every assignment.
	if err := s.SetEnvelopesStart(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if err := s.SetEnvelopesStart(ctx, month("2026-10")); err != nil {
		t.Fatal(err)
	}
	if envs := listEnvelopes(t, s, "2026-11"); envs.AssignedCents != 0 || envelopeFor(t, envs, rent.ID).CarryoverCents != 0 {
		t.Errorf("after stop and restart = %+v", envs)
	}
}

func TestFillFromCaps(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	groceries, _ := s.CreateCategory(ctx, CategoryInput{Name: "Groceries"})
	fun, _ := s.CreateCategory(ctx, CategoryInput{Name: "Fun"})
	gifts, _ := s.CreateCategory(ctx, CategoryInput{Name: "Gifts"})
	setGoal(t, s, groceries.ID, "2026-10", cents(50000))
	setGoal(t, s, fun.ID, "2026-11", cents(10000))

	_, err := s.FillFromCaps(ctx, "2026-11")
	wantKind(t, err, Conflict)

	if err := s.SetEnvelopesStart(ctx, month("2026-10")); err != nil {
		t.Fatal(err)
	}
	assign(t, s, groceries.ID, "2026-11", 30000)
	envs, err := s.FillFromCaps(ctx, "2026-11")
	if err != nil {
		t.Fatal(err)
	}
	// Groceries keeps what was assigned; Gifts has no cap.
	for id, want := range map[int64]int64{groceries.ID: 30000, fun.ID: 10000, gifts.ID: 0} {
		if e := envelopeFor(t, envs, id); e.AssignedCents != want {
			t.Errorf("%s assigned = %d, want %d", e.Name, e.AssignedCents, want)
		}
	}
	if e := envelopeFor(t, envs, fun.ID); !eq(e.CapCents, 10000) {
		t.Errorf("fun cap = %v", e.CapCents)
	}

	_, err = s.FillFromCaps(ctx, "2026-09")
	wantKind(t, err, Invalid)
}

func TestMovingToIncomeDeletesAssignments(t *testing.T) {
	s, conn := newTestStore(t)
	ctx := context.Background()
	c, _ := s.CreateCategory(ctx, CategoryInput{Name: "Side gig"})
	if err := s.SetEnvelopesStart(ctx, month("2026-10")); err != nil {
		t.Fatal(err)
	}
	assign(t, s, c.ID, "2026-10", 5000)

	income := true
	if _, err := s.UpdateCategory(ctx, c.ID, CategoryUpdate{IsIncome: &income}); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM assignments`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("%d assignments left, want 0", n)
	}
}
