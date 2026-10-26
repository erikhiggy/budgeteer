package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// envelopesStartKey is the settings key for the month envelopes count from.
const envelopesStartKey = "envelopes_start"

// Envelope is one expense category's envelope for a month. See
// docs/envelopes.md for how the numbers are worked out.
type Envelope struct {
	CategoryID     int64  `json:"category_id"`
	Name           string `json:"name"`
	CapCents       *int64 `json:"cap_cents"`
	CarryoverCents int64  `json:"carryover_cents"`
	AssignedCents  int64  `json:"assigned_cents"`
	SpentCents     int64  `json:"spent_cents"`
	BalanceCents   int64  `json:"balance_cents"`
}

type Envelopes struct {
	Month              string     `json:"month"`
	StartMonth         *string    `json:"start_month"`
	ReadyToAssignCents int64      `json:"ready_to_assign_cents"`
	IncomeCents        int64      `json:"income_cents"`
	AssignedCents      int64      `json:"assigned_cents"`
	Envelopes          []Envelope `json:"envelopes"`
	UncategorizedCents int64      `json:"uncategorized_cents"`
}

// EnvelopesStart returns the month envelopes count from, or nil if envelope
// budgeting hasn't been started.
func (s *Store) EnvelopesStart(ctx context.Context) (*string, error) {
	var start string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, envelopesStartKey).Scan(&start)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &start, nil
}

// SetEnvelopesStart starts envelopes from month, or moves the start there.
// A nil month stops envelope budgeting and deletes every assignment. It
// expects month to be validated already (see package api).
func (s *Store) SetEnvelopesStart(ctx context.Context, month *string) error {
	if month != nil {
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO settings (key, value) VALUES (?, ?)
			ON CONFLICT (key) DO UPDATE SET value = excluded.value`, envelopesStartKey, *month)
		if err != nil {
			return fmt.Errorf("set envelopes start: %w", err)
		}
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() // no-op after Commit
	if _, err := tx.ExecContext(ctx, `DELETE FROM assignments`); err != nil {
		return fmt.Errorf("delete assignments: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM settings WHERE key = ?`, envelopesStartKey); err != nil {
		return fmt.Errorf("clear envelopes start: %w", err)
	}
	return tx.Commit()
}

// ListEnvelopes expects month ("YYYY-MM") to be validated already (see
// package api). It returns every expense category, sorted by name.
func (s *Store) ListEnvelopes(ctx context.Context, month string) (Envelopes, error) {
	cats, err := s.expenseCategories(ctx)
	if err != nil {
		return Envelopes{}, err
	}
	caps, err := s.capHistory(ctx, month)
	if err != nil {
		return Envelopes{}, err
	}
	start, err := s.EnvelopesStart(ctx)
	if err != nil {
		return Envelopes{}, err
	}
	// Before the start, only the caps and uncategorized spending are shown.
	active := start != nil && *start <= month

	from := month
	if active {
		from = *start
	}
	spent, err := s.spendingByMonth(ctx, from, month)
	if err != nil {
		return Envelopes{}, err
	}

	var assigned map[spendKey]int64
	var assignedTotal int64
	if active {
		if assigned, assignedTotal, err = s.assignmentsSince(ctx, *start); err != nil {
			return Envelopes{}, err
		}
	}

	envs := Envelopes{Month: month, StartMonth: start, Envelopes: []Envelope{}}
	for _, c := range cats {
		e := Envelope{CategoryID: c.ID, Name: c.Name}
		if row := capAt(caps[c.ID], month); row != nil && row.Cents != nil {
			e.CapCents = row.Cents
		}
		if active {
			for m := *start; m < month; m = nextMonth(m) {
				e.CarryoverCents += assigned[spendKey{c.ID, m}] - spent[spendKey{c.ID, m}]
			}
			e.AssignedCents = assigned[spendKey{c.ID, month}]
			e.SpentCents = spent[spendKey{c.ID, month}]
			e.BalanceCents = e.CarryoverCents + e.AssignedCents - e.SpentCents
			envs.AssignedCents += e.AssignedCents
		}
		envs.Envelopes = append(envs.Envelopes, e)
	}
	envs.UncategorizedCents = spent[spendKey{0, month}]

	if active {
		funds, income, err := s.funds(ctx, *start, month)
		if err != nil {
			return Envelopes{}, err
		}
		envs.IncomeCents = income
		envs.ReadyToAssignCents = funds - assignedTotal
	}
	return envs, nil
}

// SetAssignment sets categoryID's assignment for month; 0 removes it. It
// expects month to be validated already (see package api), and returns the
// month's envelopes.
func (s *Store) SetAssignment(ctx context.Context, categoryID int64, month string, cents int64) (Envelopes, error) {
	if err := s.checkEnvelopeMonth(ctx, month); err != nil {
		return Envelopes{}, err
	}
	c, err := s.getCategory(ctx, categoryID)
	if err != nil {
		return Envelopes{}, err
	}
	if c.IsIncome {
		return Envelopes{}, errorf(Invalid, "%q is an income category, so it has no envelope", c.Name)
	}

	if cents == 0 {
		_, err = s.db.ExecContext(ctx, `DELETE FROM assignments WHERE category_id = ? AND month = ?`, categoryID, month)
	} else {
		_, err = s.db.ExecContext(ctx, `
			INSERT INTO assignments (category_id, month, amount_cents) VALUES (?, ?, ?)
			ON CONFLICT (category_id, month) DO UPDATE SET amount_cents = excluded.amount_cents`,
			categoryID, month, cents)
	}
	// Backstop for a category deleted since the check above.
	if isForeignKey(err) {
		return Envelopes{}, errorf(NotFound, "no category with id %d", categoryID)
	}
	if err != nil {
		return Envelopes{}, fmt.Errorf("set assignment: %w", err)
	}
	return s.ListEnvelopes(ctx, month)
}

// FillFromCaps assigns each expense category its cap for month, unless the
// category has no cap or already has an assignment that month. It expects
// month to be validated already (see package api), and returns the month's
// envelopes.
func (s *Store) FillFromCaps(ctx context.Context, month string) (Envelopes, error) {
	if err := s.checkEnvelopeMonth(ctx, month); err != nil {
		return Envelopes{}, err
	}
	cats, err := s.expenseCategories(ctx)
	if err != nil {
		return Envelopes{}, err
	}
	caps, err := s.capHistory(ctx, month)
	if err != nil {
		return Envelopes{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Envelopes{}, err
	}
	defer tx.Rollback() // no-op after Commit
	for _, c := range cats {
		row := capAt(caps[c.ID], month)
		if row == nil || row.Cents == nil {
			continue
		}
		_, err := tx.ExecContext(ctx, `
			INSERT INTO assignments (category_id, month, amount_cents) VALUES (?, ?, ?)
			ON CONFLICT (category_id, month) DO NOTHING`, c.ID, month, *row.Cents)
		if err != nil {
			return Envelopes{}, fmt.Errorf("fill from caps: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return Envelopes{}, err
	}
	return s.ListEnvelopes(ctx, month)
}

// checkEnvelopeMonth returns an error unless envelopes have been started
// and month is on or after the start.
func (s *Store) checkEnvelopeMonth(ctx context.Context, month string) error {
	start, err := s.EnvelopesStart(ctx)
	if err != nil {
		return err
	}
	if start == nil {
		return errorf(Conflict, "envelope budgeting hasn't been started")
	}
	if month < *start {
		return errorf(Invalid, "envelopes start in %s, so %s can't have assignments", *start, month)
	}
	return nil
}

// assignmentsSince returns each category's assignment by month from start
// on, and their total over every month, including months after the one
// being shown.
func (s *Store) assignmentsSince(ctx context.Context, start string) (map[spendKey]int64, int64, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT category_id, month, amount_cents FROM assignments WHERE month >= ?`, start)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	assigned := map[spendKey]int64{}
	var total int64
	for rows.Next() {
		var k spendKey
		var cents int64
		if err := rows.Scan(&k.CategoryID, &k.Month, &cents); err != nil {
			return nil, 0, err
		}
		assigned[k] = cents
		total += cents
	}
	return assigned, total, rows.Err()
}

// funds returns the money envelopes can be filled from through month: the
// balance of every account except investments just before start, plus their
// income from start through month. income is month's income alone.
func (s *Store) funds(ctx context.Context, start, month string) (funds, income int64, err error) {
	err = s.db.QueryRowContext(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN t.date < ? OR c.is_income = 1 THEN t.amount_cents END), 0),
			COALESCE(SUM(CASE WHEN t.date >= ? AND c.is_income = 1 THEN t.amount_cents END), 0)
		FROM transactions t
		JOIN accounts a ON a.id = t.account_id
		LEFT JOIN categories c ON c.id = t.category_id
		WHERE a.type <> 'investment' AND t.date < ?`,
		start+"-01", month+"-01", nextMonth(month)+"-01").Scan(&funds, &income)
	return funds, income, err
}
