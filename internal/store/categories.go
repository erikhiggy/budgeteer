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
}

type CategoryInput struct {
	Name     string
	IsIncome bool
}

// CategoryUpdate changes the non-nil fields.
type CategoryUpdate struct {
	Name     *string
	IsIncome *bool
}

func (s *Store) ListCategories(ctx context.Context) ([]Category, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT c.id, c.name, c.is_income
		FROM categories c
		ORDER BY c.is_income, c.name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categories := []Category{} // non-nil so it encodes as [] rather than null
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.IsIncome); err != nil {
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
		`SELECT id, name, is_income FROM categories WHERE id = ?`, id).Scan(&c.ID, &c.Name, &c.IsIncome)
	if errors.Is(err, sql.ErrNoRows) {
		return Category{}, errorf(NotFound, "no category with id %d", id)
	}
	return c, err
}

// CreateCategory expects in to be validated already (see package api).
func (s *Store) CreateCategory(ctx context.Context, in CategoryInput) (Category, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO categories (name, is_income) VALUES (?, ?)`, in.Name, in.IsIncome)
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
	return Category{ID: id, Name: in.Name, IsIncome: in.IsIncome}, nil
}

// UpdateCategory expects in to be validated already (see package api), with
// at least one field set. Renaming OpeningBalanceCategory is a Conflict.
func (s *Store) UpdateCategory(ctx context.Context, id int64, in CategoryUpdate) (Category, error) {
	current, err := s.getCategory(ctx, id)
	if err != nil {
		return Category{}, err
	}
	if in.Name != nil && *in.Name != current.Name && current.Name == OpeningBalanceCategory {
		return Category{}, errLockedCategory()
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

	res, err := s.db.ExecContext(ctx,
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
