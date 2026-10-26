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
- **Funds through month M**: the on-budget balance just before `S`, plus
  income-category transactions in on-budget accounts from `S` through `M`.
  Opening balances count, since "Opening balance" is an income category.
- **Ready to assign (M)**: funds through `M` minus everything assigned in
  any month from `S` on. Like YNAB, money assigned to a later month is
  already spoken for. A negative number means more is assigned than you have.
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
- **Caps as targets.** The month's cap appears beside each envelope.
  "Assign caps" fills every capped envelope that has nothing assigned yet
  this month.

Known gaps, acceptable for an experiment:

- Uncategorized spending in a month appears on its own line, but it doesn't
  come out of any envelope or out of Ready to assign.
- A transfer between an on-budget account and an investment account moves
  money on or off the budget without showing up anywhere. Use an income or
  expense transaction instead if it should count.
- Assignments made before a later start month are kept but ignored. Moving
  the start back brings them back.

## Schema (migration 0005)

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
  "uncategorized_cents": 0
}
```

- `start_month` is null until envelope budgeting is started.
- `income_cents` and `assigned_cents` are this month's totals.
- `envelopes` lists every expense category, sorted by name. `cap_cents` is
  null when the category has no cap.
- If `start_month` is null or later than `month`, every number is 0 except
  `cap_cents` and `uncategorized_cents`.

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

### `POST /api/envelopes/fill-from-caps?month=YYYY-MM`

For every expense category that has a cap in `month` and no assignment yet,
assigns the cap. Returns the month's envelopes response. The errors are the
same as for the PUT above, except for the category ones.

## Frontend

A new **Envelopes** tab at `/envelopes/:month`:

- If the budget isn't started: a short explanation and a "Start envelopes in
  <month>" button.
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
