package store

import "context"

// Payee is a payee and the category of its most recent transaction, for the
// entry row's autofill.
type Payee struct {
	Payee      string `json:"payee"`
	CategoryID *int64 `json:"category_id"` // nil if the latest was uncategorized
}

// ListPayees returns one entry per payee across all accounts, matched
// case-insensitively, sorted by name. The spelling and category both come
// from the payee's most recent transaction. Empty payees and transfer legs
// are left out.
func (s *Store) ListPayees(ctx context.Context) ([]Payee, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT payee, category_id FROM (
			SELECT payee, category_id,
				ROW_NUMBER() OVER (PARTITION BY payee COLLATE NOCASE ORDER BY date DESC, id DESC) AS n
			FROM transactions
			-- Transfer legs never have a category, so they'd only clutter autofill.
			WHERE payee <> '' AND transfer_id IS NULL
		)
		WHERE n = 1
		ORDER BY payee COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	payees := []Payee{} // non-nil so it encodes as [] rather than null
	for rows.Next() {
		var p Payee
		if err := rows.Scan(&p.Payee, &p.CategoryID); err != nil {
			return nil, err
		}
		payees = append(payees, p)
	}
	return payees, rows.Err()
}
