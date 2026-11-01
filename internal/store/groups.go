package store

import (
	"context"
	"errors"
	"fmt"
)

// errGroupsTODO is returned by the methods still to be written. It isn't an
// *Error, so the API reports it as a 500.
var errGroupsTODO = errors.New("TODO(groups): not implemented")

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
//
// TODO(groups): write this.
func (s *Store) ListCategoryGroups(ctx context.Context) ([]CategoryGroup, error) {
	return nil, errGroupsTODO
}

// UpdateCategoryGroup expects in to be validated already, with at least one
// field set. NotFound if there's no such group, Conflict if the name is taken.
//
// TODO(groups): write this.
func (s *Store) UpdateCategoryGroup(ctx context.Context, id int64, in CategoryGroupUpdate) (CategoryGroup, error) {
	return CategoryGroup{}, errGroupsTODO
}

// ReorderCategoryGroups sets each group's sort_order to its position in ids
// and returns the groups in the new order. ids must list every group exactly
// once, otherwise it's Invalid and nothing changes.
//
// TODO(groups): write this.
func (s *Store) ReorderCategoryGroups(ctx context.Context, ids []int64) ([]CategoryGroup, error) {
	return nil, errGroupsTODO
}

// DeleteCategoryGroup ungroups the group's categories, then deletes it.
// NotFound if there's no such group.
//
// TODO(groups): write this.
func (s *Store) DeleteCategoryGroup(ctx context.Context, id int64) error {
	return errGroupsTODO
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
