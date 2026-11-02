package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

type CategoryGroup struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	SortOrder int64  `json:"sort_order"`
}

type CategoryGroupInput struct {
	Name string
}

// CategoryGroupUpdate changes the non-nil fields.
type CategoryGroupUpdate struct {
	Name      *string
	SortOrder *int64
}

func scanCategoryGroup(row interface{ Scan(...any) error }) (CategoryGroup, error) {
	var g CategoryGroup
	err := row.Scan(&g.ID, &g.Name, &g.SortOrder)
	return g, err
}

func (s *Store) getCategoryGroup(ctx context.Context, id int64) (CategoryGroup, error) {
	g, err := scanCategoryGroup(s.db.QueryRowContext(ctx, `SELECT g.id, g.name, g.sort_order FROM category_groups g WHERE g.id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return CategoryGroup{}, errorf(NotFound, "no group with id %d", id)
	}
	return g, err
}

// CreateCategoryGroup expects in to be validated already (see package api).
// The new group goes last. A duplicate name is a Conflict.
func (s *Store) CreateCategoryGroup(ctx context.Context, in CategoryGroupInput) (CategoryGroup, error) {
	// MAX is NULL on an empty table, so the first group gets 0.
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO category_groups (name, sort_order)
		SELECT ?, COALESCE(MAX(sort_order) + 1, 0) FROM category_groups`, in.Name)
	if isUnique(err) {
		return CategoryGroup{}, errorf(Conflict, "a group named %q already exists", in.Name)
	}
	if err != nil {
		return CategoryGroup{}, fmt.Errorf("insert category group: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return CategoryGroup{}, err
	}
	g := CategoryGroup{ID: id, Name: in.Name}
	err = s.db.QueryRowContext(ctx,
		`SELECT sort_order FROM category_groups WHERE id = ?`, id).Scan(&g.SortOrder)
	return g, err
}

// ListCategoryGroups returns the groups by sort_order, then name. It's never
// nil, so it encodes as [].
func (s *Store) ListCategoryGroups(ctx context.Context) ([]CategoryGroup, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT g.id, g.name, g.sort_order
		 FROM category_groups g
		  ORDER BY g.sort_order, g.name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	groups := []CategoryGroup{}
	for rows.Next() {
		var g CategoryGroup
		if err := rows.Scan(&g.ID, &g.Name, &g.SortOrder); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

// UpdateCategoryGroup expects in to be validated already, with at least one
// field set. NotFound if there's no such group, Conflict if the name is taken.
func (s *Store) UpdateCategoryGroup(ctx context.Context, id int64, in CategoryGroupUpdate) (CategoryGroup, error) {
	var set []string
	var args []any
	if in.Name != nil {
		set = append(set, "name = ?")
		args = append(args, *in.Name)
	}
	if in.SortOrder != nil {
		set = append(set, "sort_order = ?")
		args = append(args, *in.SortOrder)
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE category_groups SET `+strings.Join(set, ", ")+` WHERE id = ?`, append(args, id)...)
	if isUnique(err) {
		return CategoryGroup{}, errorf(Conflict, "a group named %q already exists", *in.Name)
	}
	if err != nil {
		return CategoryGroup{}, fmt.Errorf("update category group: %w", err)
	}
	if err := wantOneRow(res, "group", id); err != nil {
		return CategoryGroup{}, err
	}
	return s.getCategoryGroup(ctx, id)
}

// ReorderCategoryGroups sets each group's sort_order to its position in ids
// and returns the groups in the new order. ids must list every group exactly
// once, otherwise it's Invalid and nothing changes.
func (s *Store) ReorderCategoryGroups(ctx context.Context, ids []int64) ([]CategoryGroup, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() // no-op after Commit

	// Checked inside the transaction so a group added meanwhile can't slip by.
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM category_groups`).Scan(&count); err != nil {
		return nil, err
	}
	if len(ids) != count {
		return nil, errorf(Invalid, "ids must list every group exactly once (%d groups, got %d ids)", count, len(ids))
	}
	seen := make(map[int64]bool, len(ids))
	for i, id := range ids {
		if seen[id] {
			return nil, errorf(Invalid, "group %d is listed more than once", id)
		}
		seen[id] = true
		// With the count and no repeats checked, a missing id means an unknown one.
		res, err := tx.ExecContext(ctx, `UPDATE category_groups SET sort_order = ? WHERE id = ?`, i, id)
		if err != nil {
			return nil, fmt.Errorf("reorder category groups: %w", err)
		}
		if n, err := res.RowsAffected(); err != nil {
			return nil, err
		} else if n == 0 {
			return nil, errorf(Invalid, "no group with id %d", id)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.ListCategoryGroups(ctx)
}

// DeleteCategoryGroup deletes the group; its categories become ungrouped
// (the group_id foreign key is ON DELETE SET NULL). NotFound if there's no
// such group.
func (s *Store) DeleteCategoryGroup(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM category_groups WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete category group: %w", err)
	}
	return wantOneRow(res, "group", id)
}

// EnvelopeGroup is one group's totals in the Envelopes view: the sums over
// its envelopes. See docs/groups.md.
type EnvelopeGroup struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	AssignedCents int64  `json:"assigned_cents"`
	SpentCents    int64  `json:"spent_cents"`
	BalanceCents  int64  `json:"balance_cents"`
}
