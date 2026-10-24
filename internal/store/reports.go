package store

import (
	"cmp"
	"context"
	"slices"
	"strings"
)

// Both reports are built on spendingByMonth, the same numbers ListGoals
// uses, so they agree with the Goals view by construction.

type CategorySpending struct {
	// CategoryID is nil for uncategorized spending.
	CategoryID *int64 `json:"category_id"`
	Name       string `json:"name"`
	SpentCents int64  `json:"spent_cents"`
}

type SpendingByCategory struct {
	Month      string             `json:"month"`
	Categories []CategorySpending `json:"categories"`
	TotalCents int64              `json:"total_cents"`
}

type MonthSpending struct {
	Month      string `json:"month"`
	SpentCents int64  `json:"spent_cents"`
}

type MonthlySpending struct {
	From   string          `json:"from"`
	To     string          `json:"to"`
	Months []MonthSpending `json:"months"`
}

// UncategorizedName labels uncategorized spending in reports.
const UncategorizedName = "Uncategorized"

// SpendingByCategory expects month to be validated already (see package
// api). It leaves out categories with no spending, and sorts the rest
// biggest first.
func (s *Store) SpendingByCategory(ctx context.Context, month string) (SpendingByCategory, error) {
	cats, err := s.expenseCategories(ctx)
	if err != nil {
		return SpendingByCategory{}, err
	}
	spent, err := s.spendingByMonth(ctx, month, month)
	if err != nil {
		return SpendingByCategory{}, err
	}

	report := SpendingByCategory{Month: month, Categories: []CategorySpending{}}
	add := func(id *int64, name string, cents int64) {
		if cents != 0 {
			report.Categories = append(report.Categories, CategorySpending{id, name, cents})
			report.TotalCents += cents
		}
	}
	for _, c := range cats {
		add(&c.ID, c.Name, spent[spendKey{c.ID, month}])
	}
	add(nil, UncategorizedName, spent[spendKey{0, month}])

	slices.SortStableFunc(report.Categories, func(a, b CategorySpending) int {
		return cmp.Or(
			cmp.Compare(b.SpentCents, a.SpentCents),
			cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)),
		)
	})
	return report, nil
}

// MonthlySpending expects from <= to, both validated already (see package
// api). Every month in the range is included, with 0 when nothing was spent.
func (s *Store) MonthlySpending(ctx context.Context, from, to string) (MonthlySpending, error) {
	cats, err := s.expenseCategories(ctx)
	if err != nil {
		return MonthlySpending{}, err
	}
	spent, err := s.spendingByMonth(ctx, from, to)
	if err != nil {
		return MonthlySpending{}, err
	}

	// Category 0 is uncategorized, which counts as spending too.
	ids := []int64{0}
	for _, c := range cats {
		ids = append(ids, c.ID)
	}
	report := MonthlySpending{From: from, To: to, Months: []MonthSpending{}}
	for m := from; m <= to; m = nextMonth(m) {
		ms := MonthSpending{Month: m}
		for _, id := range ids {
			ms.SpentCents += spent[spendKey{id, m}]
		}
		report.Months = append(report.Months, ms)
	}
	return report, nil
}
