CREATE TABLE accounts (
  id   INTEGER PRIMARY KEY,
  name TEXT NOT NULL UNIQUE COLLATE NOCASE,
  type TEXT NOT NULL CHECK (type IN ('checking', 'savings', 'credit', 'cash'))
);

CREATE TABLE categories (
  id        INTEGER PRIMARY KEY,
  name      TEXT NOT NULL UNIQUE COLLATE NOCASE,
  is_income INTEGER NOT NULL DEFAULT 0 CHECK (is_income IN (0, 1))
);

INSERT INTO categories (name, is_income) VALUES ('Income', 1), ('Opening balance', 1);

CREATE TABLE transactions (
  id           INTEGER PRIMARY KEY,
  account_id   INTEGER NOT NULL REFERENCES accounts (id) ON DELETE RESTRICT,
  category_id  INTEGER REFERENCES categories (id) ON DELETE RESTRICT,
  date         TEXT NOT NULL CHECK (date GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'),
  payee        TEXT NOT NULL DEFAULT '',
  amount_cents INTEGER NOT NULL, -- negative = money out, positive = money in
  note         TEXT NOT NULL DEFAULT '',
  created_at   TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX tx_account_date ON transactions (account_id, date DESC, id DESC);
CREATE INDEX tx_date_cat ON transactions (date, category_id);
CREATE INDEX tx_payee ON transactions (payee COLLATE NOCASE);
