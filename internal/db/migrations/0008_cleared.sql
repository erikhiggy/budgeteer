-- A transaction is uncleared until it has posted at the bank. Each transfer
-- leg has its own flag, since the two accounts are checked against different
-- statements. The budget ignores this: it only splits an account's balance
-- into cleared and uncleared.
ALTER TABLE transactions
  ADD COLUMN cleared INTEGER NOT NULL DEFAULT 0 CHECK (cleared IN (0, 1));
