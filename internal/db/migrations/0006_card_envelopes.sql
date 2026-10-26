-- Credit card envelopes (see docs/envelopes.md). Each credit account has an
-- envelope that card purchases fill and payments to the card draw down;
-- these are the months' assignments to it. They go with the account.
CREATE TABLE card_assignments (
  account_id   INTEGER NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
  month        TEXT NOT NULL CHECK (month GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]'),
  amount_cents INTEGER NOT NULL CHECK (amount_cents <> 0),
  PRIMARY KEY (account_id, month)
);
