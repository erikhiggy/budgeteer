package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// OpeningBalanceCategory is seeded by migration 0001. CreateAccount finds it
// by name, so it can't be renamed or deleted.
const OpeningBalanceCategory = "Opening balance"

type Category struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	IsIncome bool   `json:"is_income"`
	// RolloverSince is the "YYYY-MM" rollover starts from; nil means off.
	RolloverSince *string `json:"rollover_since"`
	// GroupID is the category's group; nil means ungrouped. Income
	// categories are never grouped.
	GroupID *int64 `json:"group_id"`
}

type CategoryInput struct {
	Name     string
	IsIncome bool
	GroupID  *int64
}

// CategoryUpdate changes the non-nil fields. RolloverSince is applied only
// when SetRollover is true, since nil there means "turn rollover off".
type CategoryUpdate struct {
	Name          *string
	IsIncome      *bool
	SetRollover   bool
	RolloverSince *string
	// GroupID is applied only when SetGroup is true, since nil there means
	// "ungroup".
	SetGroup bool
	GroupID  *int64
}

func (s *Store) ListCategories(ctx context.Context) ([]Category, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT c.id, c.name, c.is_income, c.rollover_since, c.group_id
		FROM categories c
		ORDER BY c.is_income, c.name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categories := []Category{} // non-nil so it encodes as [] rather than null
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.IsIncome, &c.RolloverSince, &c.GroupID); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, rows.Err()
}

// getCategory returns the category with id, or a NotFound error.
func (s *Store) getCategory(ctx context.Context, id int64) (Category, error) {
	var c Category
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, is_income, rollover_since, group_id FROM categories WHERE id = ?`, id).Scan(&c.ID, &c.Name, &c.IsIncome, &c.RolloverSince, &c.GroupID)
	if errors.Is(err, sql.ErrNoRows) {
		return Category{}, errorf(NotFound, "no category with id %d", id)
	}
	return c, err
}

// CreateCategory expects in to be validated already (see package api). An
// income category can't have a group (Invalid), and an unknown group is a
// NotFound.
func (s *Store) CreateCategory(ctx context.Context, in CategoryInput) (Category, error) {
	if in.GroupID != nil {
		if in.IsIncome {
			return Category{}, errIncomeGroup()
		}
		if _, err := s.getCategoryGroup(ctx, *in.GroupID); err != nil {
			return Category{}, err
		}
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO categories (name, is_income, group_id) VALUES (?, ?, ?)`, in.Name, in.IsIncome, in.GroupID)
	if isUnique(err) {
		return Category{}, errorf(Conflict, "a category named %q already exists", in.Name)
	}
	if err != nil {
		return Category{}, fmt.Errorf("insert category: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Category{}, err
	}
	return Category{ID: id, Name: in.Name, IsIncome: in.IsIncome, GroupID: in.GroupID}, nil
}

// UpdateCategory expects in to be validated already (see package api), with
// at least one field set. Renaming OpeningBalanceCategory is a Conflict, and
// an income category can't roll over or have a group (Invalid), and an
// unknown group is a NotFound. Moving a category to income deletes its goals
// and envelope assignments and turns rollover and its group off.
func (s *Store) UpdateCategory(ctx context.Context, id int64, in CategoryUpdate) (Category, error) {
	current, err := s.getCategory(ctx, id)
	if err != nil {
		return Category{}, err
	}
	if in.Name != nil && *in.Name != current.Name && current.Name == OpeningBalanceCategory {
		return Category{}, errLockedCategory()
	}
	income := current.IsIncome
	if in.IsIncome != nil {
		income = *in.IsIncome
	}
	if in.SetRollover && in.RolloverSince != nil && income {
		return Category{}, errorf(Invalid, "an income category can't roll over")
	}
	toIncome := income && !current.IsIncome
	if in.SetGroup && in.GroupID != nil {
		if income {
			return Category{}, errIncomeGroup()
		}
		if _, err := s.getCategoryGroup(ctx, *in.GroupID); err != nil {
			return Category{}, err
		}
	}

	var set []string
	var args []any
	if in.Name != nil {
		set = append(set, "name = ?")
		args = append(args, *in.Name)
	}
	if in.IsIncome != nil {
		set = append(set, "is_income = ?")
		args = append(args, *in.IsIncome)
	}
	if in.SetRollover || toIncome {
		set = append(set, "rollover_since = ?")
		args = append(args, in.RolloverSince) // nil when moving to income
	}

	if in.SetGroup || toIncome {
		set = append(set, "group_id = ?")
		if toIncome {
			args = append(args, nil)
		} else {
			args = append(args, in.GroupID)
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Category{}, err
	}
	defer tx.Rollback() // no-op after Commit

	res, err := tx.ExecContext(ctx,
		`UPDATE categories SET `+strings.Join(set, ", ")+` WHERE id = ?`, append(args, id)...)
	if isUnique(err) {
		return Category{}, errorf(Conflict, "a category named %q already exists", *in.Name)
	}
	if err != nil {
		return Category{}, fmt.Errorf("update category: %w", err)
	}
	if err := wantOneRow(res, "category", id); err != nil {
		return Category{}, err
	}
	if toIncome {
		if _, err := tx.ExecContext(ctx, `DELETE FROM goals WHERE category_id = ?`, id); err != nil {
			return Category{}, fmt.Errorf("delete goals: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM assignments WHERE category_id = ?`, id); err != nil {
			return Category{}, fmt.Errorf("delete assignments: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return Category{}, err
	}
	return s.getCategory(ctx, id)
}

// DeleteCategory returns a Conflict error if the category has transactions
// or is OpeningBalanceCategory.
func (s *Store) DeleteCategory(ctx context.Context, id int64) error {
	name, n, err := s.usage(ctx, "categories", "category_id", id)
	if errors.Is(err, sql.ErrNoRows) {
		return errorf(NotFound, "no category with id %d", id)
	}
	if err != nil {
		return err
	}
	if name == OpeningBalanceCategory {
		return errLockedCategory()
	}
	if n > 0 {
		return errorf(Conflict, "%q has %s; delete or recategorize them first", name, plural(n, "transaction"))
	}

	res, err := s.db.ExecContext(ctx, `DELETE FROM categories WHERE id = ?`, id)
	// Backstop for a transaction added since the count above.
	if isForeignKey(err) {
		return errorf(Conflict, "%q has transactions; delete or recategorize them first", name)
	}
	if err != nil {
		return fmt.Errorf("delete category: %w", err)
	}
	return wantOneRow(res, "category", id)
}

func errLockedCategory() error {
	return errorf(Conflict, "%q is used for account opening balances and can't be renamed or deleted", OpeningBalanceCategory)
}

func errIncomeGroup() error {
	return errorf(Invalid, "an income category can't be in a group")
}
