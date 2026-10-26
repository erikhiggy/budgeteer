-- Envelope budgeting (an experiment next to caps; see docs/envelopes.md).
-- An assignment is money put into a category's envelope in a month; it can
-- be negative to take money back out. A category's assignments go with it.
CREATE TABLE assignments (
  category_id  INTEGER NOT NULL REFERENCES categories (id) ON DELETE CASCADE,
  month        TEXT NOT NULL CHECK (month GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]'),
  amount_cents INTEGER NOT NULL CHECK (amount_cents <> 0),
  PRIMARY KEY (category_id, month)
);

-- App-wide settings. 'envelopes_start' is the 'YYYY-MM' envelopes count from.
CREATE TABLE settings (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
