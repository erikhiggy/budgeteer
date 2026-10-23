package store

import (
	"context"
	"fmt"
)

type Category struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	IsIncome bool   `json:"is_income"`
}

type CategoryInput struct {
	Name     string
	IsIncome bool
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
