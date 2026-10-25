-- Adds 'investment' to the account types. SQLite can't change a CHECK in
-- place, so the table is rebuilt. Ids are copied, so transactions still point
-- at the same accounts. This relies on the migrator turning foreign keys off.
CREATE TABLE accounts_new (
  id   INTEGER PRIMARY KEY,
  name TEXT NOT NULL UNIQUE COLLATE NOCASE,
  type TEXT NOT NULL CHECK (type IN ('checking', 'savings', 'credit', 'cash', 'investment'))
);

INSERT INTO accounts_new (id, name, type) SELECT id, name, type FROM accounts;

DROP TABLE accounts;

ALTER TABLE accounts_new RENAME TO accounts;
