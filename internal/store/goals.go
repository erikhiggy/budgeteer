package store

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Goal is one expense category's cap and spending for a month. See "Goals"
// in docs/api.md for how carryover and available are worked out.
type Goal struct {
	CategoryID     int64   `json:"category_id"`
	Name           string  `json:"name"`
	CapCents       *int64  `json:"cap_cents"`
	CapSince       *string `json:"cap_since"`
	RolloverSince  *string `json:"rollover_since"`
	CarryoverCents int64   `json:"carryover_cents"`
	AvailableCents *int64  `json:"available_cents"`
	SpentCents     int64   `json:"spent_cents"`
}

type Goals struct {
	Month              string `json:"month"`
	Goals              []Goal `json:"goals"`
	UncategorizedCents int64  `json:"uncategorized_cents"`
}

// capRow is one goals row: the cap from Month on, or nil to remove it.
type capRow struct {
	Month string
	Cents *int64
}

// ListGoals expects month ("YYYY-MM") to be validated already (see package
// api). It returns every expense category, sorted by name.
func (s *Store) ListGoals(ctx context.Context, month string) (Goals, error) {
	cats, err := s.expenseCategories(ctx)
	if err != nil {
		return Goals{}, err
	}
	caps, err := s.capHistory(ctx, month)
	if err != nil {
		return Goals{}, err
	}

	// Spending is needed from the earliest month any rollover starts.
	from := month
	for _, c := range cats {
		if c.RolloverSince != nil && *c.RolloverSince < from {
			from = *c.RolloverSince
		}
	}
	spent, err := s.spendingByMonth(ctx, from, month)
	if err != nil {
		return Goals{}, err
	}

	goals := Goals{Month: month, Goals: []Goal{}}
	for _, c := range cats {
		g := Goal{CategoryID: c.ID, Name: c.Name, RolloverSince: c.RolloverSince, SpentCents: spent[spendKey{c.ID, month}]}
		if row := capAt(caps[c.ID], month); row != nil && row.Cents != nil {
			g.CapCents, g.CapSince = row.Cents, &row.Month
		}
		capOr0 := func(m string) int64 {
			if row := capAt(caps[c.ID], m); row != nil && row.Cents != nil {
				return *row.Cents
			}
			return 0
		}
		if c.RolloverSince != nil && *c.RolloverSince <= month {
			for m := *c.RolloverSince; m < month; m = nextMonth(m) {
				g.CarryoverCents += capOr0(m) - spent[spendKey{c.ID, m}]
			}
			available := capOr0(month) + g.CarryoverCents
			g.AvailableCents = &available
		} else if g.CapCents != nil {
			available := *g.CapCents
			g.AvailableCents = &available
		}
		goals.Goals = append(goals.Goals, g)
	}

	goals.UncategorizedCents = spent[spendKey{0, month}]
	return goals, nil
}

// SetGoal sets categoryID's cap from month on, or removes it when cents is
// nil. It expects month and cents to be validated already (see package api),
// and returns the category's Goal for month.
func (s *Store) SetGoal(ctx context.Context, categoryID int64, month string, cents *int64) (Goal, error) {
	c, err := s.getCategory(ctx, categoryID)
	if err != nil {
		return Goal{}, err
	}
	if c.IsIncome {
		return Goal{}, errorf(Invalid, "%q is an income category, so it can't have a cap", c.Name)
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO goals (category_id, effective_month, amount_cents) VALUES (?, ?, ?)
		ON CONFLICT (category_id, effective_month) DO UPDATE SET amount_cents = excluded.amount_cents`,
		categoryID, month, cents)
	// Backstop for a category deleted since the check above.
	if isForeignKey(err) {
		return Goal{}, errorf(NotFound, "no category with id %d", categoryID)
	}
	if err != nil {
		return Goal{}, fmt.Errorf("set goal: %w", err)
	}

	all, err := s.ListGoals(ctx, month)
	if err != nil {
		return Goal{}, err
	}
	for _, g := range all.Goals {
		if g.CategoryID == categoryID {
			return g, nil
		}
	}
	return Goal{}, errors.New("set goal: category missing from goals")
}

func (s *Store) expenseCategories(ctx context.Context) ([]Category, error) {
	all, err := s.ListCategories(ctx)
	if err != nil {
		return nil, err
	}
	var expense []Category
	for _, c := range all {
		if !c.IsIncome {
			expense = append(expense, c)
		}
	}
	return expense, nil
}

// capHistory returns each category's goals rows up to and including month,
// oldest first.
func (s *Store) capHistory(ctx context.Context, month string) (map[int64][]capRow, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT category_id, effective_month, amount_cents FROM goals
		WHERE effective_month <= ?
		ORDER BY category_id, effective_month`, month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	caps := map[int64][]capRow{}
	for rows.Next() {
		var id int64
		var r capRow
		if err := rows.Scan(&id, &r.Month, &r.Cents); err != nil {
			return nil, err
		}
		caps[id] = append(caps[id], r)
	}
	return caps, rows.Err()
}

// capAt returns the row in effect for month: the latest with Month <= month,
// or nil if there's none. rows must be sorted oldest first.
func capAt(rows []capRow, month string) *capRow {
	var found *capRow
	for i := range rows {
		if rows[i].Month > month {
			break
		}
		found = &rows[i]
	}
	return found
}

// spendKey is a category and a "YYYY-MM" month. Category 0 is uncategorized.
type spendKey struct {
	CategoryID int64
	Month      string
}

// spendingByMonth returns spending for each category and month from "from"
// to "to" inclusive: -SUM(amount_cents), so money out is positive and refunds
// reduce it. Uncategorized spending is under category 0; transfer legs are
// never counted. Income categories are included, so callers pick the
// categories they want. M5's reports must agree with this.
func (s *Store) spendingByMonth(ctx context.Context, from, to string) (map[spendKey]int64, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT COALESCE(category_id, 0), substr(date, 1, 7), -SUM(amount_cents)
		FROM transactions
		WHERE date >= ? AND date < ? AND transfer_id IS NULL
		GROUP BY 1, 2`, from+"-01", nextMonth(to)+"-01")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	spent := map[spendKey]int64{}
	for rows.Next() {
		var k spendKey
		var cents int64
		if err := rows.Scan(&k.CategoryID, &k.Month, &cents); err != nil {
			return nil, err
		}
		spent[k] = cents
	}
	return spent, rows.Err()
}

// nextMonth returns the month after m ("2026-12" → "2027-01"). m must be a
// valid "YYYY-MM".
func nextMonth(m string) string {
	t, err := time.Parse("2006-01", m)
	if err != nil {
		panic(fmt.Sprintf("nextMonth(%q): %v", m, err))
	}
	return t.AddDate(0, 1, 0).Format("2006-01")
}
