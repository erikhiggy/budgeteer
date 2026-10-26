# Envelope budgeting (experiment)

Branch: `feat/envelope-style-budgeting`.

Try YNAB/Buckets-style envelopes next to the existing caps, without changing
caps. Goals, rollover and reports work as before. Envelopes get their own
tab, their own tables, and a start month. Before the start month is set,
nothing changes. Stopping deletes only the envelope data.

## The model

- **Start month** (`S`). Envelopes count money from `S` on. The experiment
  can start in any month, the real database included, without old spending
  showing up as overspent envelopes.
- **On-budget accounts**: every account except `investment`.
- **Funds through month M**: the balance just before `S` of every on-budget
  account except credit cards, plus income-category transactions in
  on-budget accounts from `S` through `M`. Opening balances count, since
  "Opening balance" is an income category, except on credit cards. Card
  debt isn't money to assign: it shows as owed in the card's envelope.
- **Ready to assign (M)**: funds through `M` minus everything assigned in
  any month from `S` on, to categories and cards. Like YNAB, money assigned
  to a later month is already spoken for. A negative number means more is
  assigned than you have.
- **Envelope** (one per expense category):
  - `assigned`: the money put in this month. It can be negative, which takes
    money back out.
  - `spent`: the same figure Goals uses: categorized money out minus refunds,
    in any account, not counting transfers.
  - `carryover`: assigned minus spent over every month from `S` up to the
    month before `M`.
  - `balance` = `carryover + assigned − spent`. Balances always roll over,
    and overspending carries into the next month as a negative balance (the
    Buckets way). It doesn't come out of Ready to assign, as YNAB's cash
    overspending does.
- **Card envelope** (one per `credit` account, the YNAB way):
  - `assigned`: money put toward paying the card, e.g. toward old debt.
  - `purchases`: categorized spending on the card, minus refunds (income
    and uncategorized transactions don't count). That money already left
    its category's envelope, so it moves here, set aside to pay the card.
  - `paid`: what transfers from on-budget accounts paid toward the card. A
    transfer out of the card, such as a cash advance, makes it negative.
  - `balance` = `carryover + assigned + purchases − paid`. It carries over
    like any envelope. A negative balance means you paid more than was set
    aside.
  - `owed`: what's owed on the card at the end of the month. It's shown
    next to the envelope. Debt from before `S` or from a card's opening
    balance is owed until you assign toward it.

  If you pay in full every month, the envelope fills with the month's
  purchases and empties when you pay, with nothing to assign. The sum of
  Ready to assign and every envelope equals the cash in the non-card
  on-budget accounts (plus any income categorized on a card).
- **Caps as targets.** The month's cap appears beside each envelope.
  "Assign caps" fills every capped envelope that has nothing assigned yet
  this month.

Known gaps, acceptable for an experiment:

- Uncategorized spending in a month appears on its own line, but it doesn't
  come out of any envelope or out of Ready to assign.
- A transfer between an on-budget account and an investment account moves
  money on or off the budget without showing up anywhere. A payment to a
  card from an investment account isn't counted as paid. Use an income or
  expense transaction instead if it should count.
- Assignments made before a later start month are kept but ignored. Moving
  the start back brings them back.

## Schema (migrations 0005 and 0006)

```sql
CREATE TABLE assignments (
  category_id  INTEGER NOT NULL REFERENCES categories (id) ON DELETE CASCADE,
  month        TEXT NOT NULL CHECK (month GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]'),
  amount_cents INTEGER NOT NULL CHECK (amount_cents <> 0),
  PRIMARY KEY (category_id, month)
);

CREATE TABLE settings (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
```

```sql
-- 0006: assignments to credit card envelopes.
CREATE TABLE card_assignments (
  account_id   INTEGER NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
  month        TEXT NOT NULL CHECK (month GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]'),
  amount_cents INTEGER NOT NULL CHECK (amount_cents <> 0),
  PRIMARY KEY (account_id, month)
);
```

The start month is stored under the `settings` key `envelopes_start`.
Moving a category to income deletes its assignments, the same way it
deletes its goals.

## API

### `GET /api/envelopes?month=YYYY-MM`

`month` defaults to the current month.

```json
{
  "month": "2026-11",
  "start_month": "2026-11",
  "ready_to_assign_cents": 12000,
  "income_cents": 450000,
  "assigned_cents": 438000,
  "envelopes": [
    {
      "category_id": 3,
      "name": "Groceries",
      "cap_cents": 60000,
      "carryover_cents": 0,
      "assigned_cents": 60000,
      "spent_cents": 21000,
      "balance_cents": 39000
    }
  ],
  "cards": [
    {
      "account_id": 7,
      "name": "Visa",
      "carryover_cents": 0,
      "assigned_cents": 50000,
      "purchases_cents": 10000,
      "paid_cents": 60000,
      "balance_cents": 0,
      "owed_cents": 150000
    }
  ],
  "uncategorized_cents": 0
}
```

- `start_month` is null until envelope budgeting is started.
- `income_cents` and `assigned_cents` are this month's totals.
  `assigned_cents` includes the cards.
- `cards` lists every credit account, sorted by name. `owed_cents` is
  always filled in.
- `envelopes` lists every expense category, sorted by name. `cap_cents` is
  null when the category has no cap.
- If `start_month` is null or later than `month`, every number is 0 except
  `cap_cents`, `owed_cents` and `uncategorized_cents`.

### `PUT /api/envelopes/start`

Body: `{"month": "YYYY-MM"}` starts envelopes or moves the start month.
`{"month": null}` stops: it clears the start month and deletes every
assignment. Returns `{"start_month": "2026-11"}` or `{"start_month": null}`.

### `PUT /api/envelopes/{category_id}?month=YYYY-MM`

Body: `{"amount_cents": 60000}` sets the month's assignment. The amount can
be negative, and 0 removes it. Returns the month's whole envelopes response,
since Ready to assign changes too.

Errors:

- 400 if `month` is missing or invalid, `amount_cents` is missing, the
  category is an income category, or `month` is before the start month.
- 404 if there's no such category.
- 409 if envelope budgeting hasn't been started.

### `PUT /api/envelopes/cards/{account_id}?month=YYYY-MM`

The same as above, for a credit account's envelope. 400 if the account
isn't a credit account, and 404 if there's no such account.

### `POST /api/envelopes/fill-from-caps?month=YYYY-MM`

For every expense category that has a cap in `month` and no assignment yet,
assigns the cap. Returns the month's envelopes response. The errors are the
same as for the PUT above, except for the category ones.

## Frontend

A new **Envelopes** tab at `/envelopes/:month`:

- If the budget isn't started: a short explanation and a "Start envelopes in
  <month>" button.
- A "Credit cards" section under the categories, with the same columns.
  Spent is what was paid, and the small print shows the month's purchases
  and what's owed, including how much of it isn't covered yet.
- A month picker, then a summary bar with Ready to assign (shown in red with
  "over-assigned" when it's negative) and the month's income and assigned
  totals, plus an "Assign caps" button.
- A table with Category (with the carryover note), Assigned (click to edit,
  as the Goals cap is edited; a leading "-" takes money out), Spent, and
  Balance (red when negative). The cap appears as small print.
- For months before the start: a notice with a link to the start month.
- At the bottom, "Stop envelopes", a two-click button that deletes all
  assignments.

## Steps

1. This plan and contract.
2. Backend: migration, store (`envelopes.go`), API handlers and routes, and
   tests.
3. Frontend: types, client, hooks, view, tests.
4. Browser check against a scratch DB on port 7421.

## Later, if the experiment sticks

- A "move money" action between two envelopes.
- Count uncategorized spending against Ready to assign.
- Decide whether envelopes replace caps, and if so, migrate caps into
  monthly assignments and remove the Goals tab.
