-- A transfer is a pair of transactions ("legs") in two accounts that share
-- a transfer_id. Deleting the transfers row deletes both legs. Legs have no
-- category; the store enforces that, since ALTER TABLE can't add a CHECK.
CREATE TABLE transfers (
  id INTEGER PRIMARY KEY
);

ALTER TABLE transactions
  ADD COLUMN transfer_id INTEGER REFERENCES transfers (id) ON DELETE CASCADE;

CREATE INDEX tx_transfer ON transactions (transfer_id);
