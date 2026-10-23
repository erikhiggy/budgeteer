-- Monthly spending caps, stored by the month they take effect. The cap for
-- month M is the latest row with effective_month <= M; a NULL amount removes
-- the cap from that month on. A category's goals go with it when it's deleted.
CREATE TABLE goals (
  category_id     INTEGER NOT NULL REFERENCES categories (id) ON DELETE CASCADE,
  effective_month TEXT NOT NULL CHECK (effective_month GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]'),
  amount_cents    INTEGER CHECK (amount_cents > 0),
  PRIMARY KEY (category_id, effective_month)
);

-- The month rollover starts for a category ('YYYY-MM'); NULL means off.
ALTER TABLE categories ADD COLUMN rollover_since TEXT;
