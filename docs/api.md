# API contract

JSON in and out. Every error response is `{"error": "message"}`; the frontend
shows `message` to the user, so write it for a person.

Requests that change data must send `Content-Type: application/json` (the
middleware rejects anything else with 415). With curl:

```sh
curl -s localhost:7420/api/accounts -H 'Content-Type: application/json' \
  -d '{"name":"Checking","type":"checking"}'
```

The frontend's copy of these shapes is `web/src/api/types.ts`. Keep the two
in sync.

## Rules for `{id}` routes (M2)

These apply to every `PATCH` and `DELETE` below.

- `{id}` must be a positive integer, otherwise 400 (`invalid id "abc"`).
- No row with that id is a 404 (`no account with id 7`).
- `PATCH` returns 200 with the updated object, in the same shape as the list endpoint returns it.
- `DELETE` returns 204 with no body.
- `DELETE` is a mutation, so it needs `Content-Type: application/json` too, even though it has no body.
- Fields in a `PATCH` body follow the same validation as `POST`, with the same 400 messages.

## Accounts

Code: `internal/store/accounts.go`, `internal/api/accounts.go`, and their tests. The M1 routes are the reference pattern for the rest of the API.

```ts
type Account = { id: number; name: string; type: 'checking' | 'savings' | 'credit' | 'cash'; balance_cents: number }
```

| Request | Success | Errors |
|---|---|---|
| `GET /api/accounts` | 200, `Account[]` sorted by name (case-insensitive) | |
| `POST /api/accounts` `{name, type, opening_balance_cents?}` | 201, the new `Account` | 400 blank name, bad type, unknown field, bad JSON. 409 duplicate name (case-insensitive) |
| `PATCH /api/accounts/{id}` `{name?, type?}` (M2) | 200, the updated `Account` | 400 as for `POST`, or an empty body `{}`. 404. 409 duplicate name |
| `DELETE /api/accounts/{id}` (M2) | 204 | 404. 409 the account has transactions |

Opening balance (M2):
- `opening_balance_cents` is optional. Omitted or `0` means no opening balance.
- If it's non-zero, the same request also creates a transaction in that account:
  - category: the seeded `Opening balance` category
  - date: today, in the server's local time
  - payee `Opening balance`, note `""`
  - amount: `opening_balance_cents` as given. Positive for money you have; negative for a credit card that starts out owing.
- The account and its opening transaction are created in one SQL transaction, so a failure leaves neither behind.
- The response's `balance_cents` includes the opening balance.

PATCH (M2):
- Only the fields sent are changed. Omitted fields keep their current value.
- `name` is trimmed. Renaming to a different case of the current name (`checking` → `Checking`) is allowed and isn't a conflict.
- At least one field is required; `{}` is a 400 (`nothing to update`).

DELETE (M2):
- An account with any transactions can't be deleted. The 409 message gives the count, e.g. `"Checking" has 12 transactions; delete or move them first`.
- Archiving instead of deleting is listed under "later" in the plan.

## Categories

```ts
type Category = {
  id: number
  name: string
  is_income: boolean
  rollover_since: string | null  // (M4) "YYYY-MM" rollover starts; null = off
}
```

| Request | Success | Errors |
|---|---|---|
| `GET /api/categories` | 200, `Category[]` with expense categories first, then income, each sorted by name (case-insensitive) | |
| `POST /api/categories` `{name, is_income?}` | 201, the new `Category`. `is_income` defaults to `false` | 400 blank name, unknown field, bad JSON. 409 duplicate name (case-insensitive) |
| `PATCH /api/categories/{id}` `{name?, is_income?, rollover_since?}` (M2, M4) | 200, the updated `Category` | 400 as for `POST`, `{}`, or a rollover rule below. 404. 409 duplicate name, or the category is `Opening balance` |
| `DELETE /api/categories/{id}` (M2) | 204 | 404. 409 the category has transactions, or is `Opening balance` |

- Trim `name`.
- Migration 0001 seeds `Income` and `Opening balance` as income categories, so a fresh database returns two categories, not `[]`.
- PATCH changes only the fields sent, as for accounts. Changing `is_income` is allowed. It moves the category's transactions in or out of spending in reports from then on.
- DELETE of a category with transactions is a 409 with the count, as for accounts. Uncategorizing those transactions automatically is not part of M2.
- **Moving a category to income (M4)** deletes its goals and turns rollover off, since income categories have no caps. Moving it back starts with no cap.
- **Rollover (M4):** `rollover_since` is a `"YYYY-MM"` month to start rolling over from, or `null` to turn it off. Omitted leaves it as it is.
  - Only expense categories can roll over (400 `an income category can't roll over`), including when the same `PATCH` sets `is_income: true`.
  - Setting it again, or turning it off and back on, starts fresh from the new month. Carryover from before is dropped.
- **`Opening balance` is locked (M2):** renaming or deleting it is a 409 (`"Opening balance" is used for account opening balances and can't be renamed or deleted`). The add-account flow finds it by name. `Income` is an ordinary category.

## Transactions

```ts
type Transaction = {
  id: number
  account_id: number
  category_id: number | null
  date: string          // "YYYY-MM-DD"
  payee: string
  amount_cents: number  // negative = money out, positive = money in
  note: string
  transfer_id: number | null          // (M3) set on both legs of a transfer
  transfer_account_id: number | null  // (M3) the other leg's account
}
```

| Request | Success | Errors |
|---|---|---|
| `GET /api/transactions?account_id=1&from=2026-10-01&to=2026-10-31` | 200, `Transaction[]`, newest first (`date DESC, id DESC`) | 400 missing or invalid `account_id`, or invalid `from`/`to`. 404 no such account |
| `POST /api/transactions` `{account_id, category_id, date, payee?, amount_cents, note?}` | 201, the new `Transaction` | 400 for any of the rules below |
| `PATCH /api/transactions/{id}` `{account_id, category_id, date, payee?, amount_cents, note?}` (M2) | 200, the updated `Transaction` | 400 as for `POST`, or a transfer leg rule below. 404 |
| `DELETE /api/transactions/{id}` (M2) | 204 | 404 |

GET:
- `account_id` is required.
- `from` and `to` are optional and inclusive.
- No matching rows returns `[]`, never `null`.

POST validation (all 400):
- `date` is required and must be a real calendar date. `2026-02-30` is rejected.
- `amount_cents` is required and must be non-zero.
- `account_id` must refer to an existing account.
- `category_id` may be `null` or omitted (uncategorized). If present, it must refer to an existing category.
- `payee` and `note` default to `""` and are trimmed.
- The message should name the field, e.g. `unknown category_id 42`, since the UI shows it next to the entry row.

PATCH (M2):
- **Unlike accounts and categories, a transaction `PATCH` takes the full body**, with the same fields and rules as `POST`. The inline editor always has every field. This also avoids the difference between an omitted `category_id` (unchanged) and `null` (uncategorized).
- Changing `account_id` moves the transaction to that account.
- An unknown `account_id` or `category_id` in the body is a 400, as for `POST`. An unknown `{id}` in the path is a 404.

Transfer legs (M3):
- A leg is an ordinary row in `GET` results, with `category_id: null` and both transfer fields set. The table shows it as "Transfer → Savings" (money out) or "Transfer ← Savings" (money in).
- **`PATCH` on a leg updates the whole transfer.** `date`, `payee` and `note` are copied to both legs. `amount_cents` is this leg's amount, and the other leg gets the negation, so the money still balances.
- The accounts can't change: `account_id` must be the leg's current account (400 `a transfer's accounts can't be changed`). `category_id` must be `null` (400 `a transfer has no category`). To turn a transfer into an ordinary transaction or the reverse, delete it and add it again.
- **`DELETE` on a leg deletes the whole transfer**, both legs.
- `POST /api/transactions` never creates a leg; use `POST /api/transfers`.

## Transfers (M3)

```ts
type Transfer = { id: number; legs: [Transaction, Transaction] }  // legs[0] is in account_id
```

| Request | Success | Errors |
|---|---|---|
| `POST /api/transfers` `{account_id, other_account_id, date, payee?, amount_cents, note?}` | 201, the new `Transfer` | 400 for any of the rules below |

- The request is written from `account_id`'s side, with the entry row's sign convention:
  - negative `amount_cents` (a plain number in the UI) is money out of `account_id` and into `other_account_id`
  - positive (`+` in the UI) is money into `account_id` from `other_account_id`
- The two legs get `amount_cents` and its negation, the same `date`, `payee` and `note`, and `category_id` NULL. Both are created in one SQL transaction.
- Validation (all 400), with messages naming the field:
  - `date` and `amount_cents` follow the transaction rules: a real date, non-zero amount.
  - `account_id` and `other_account_id` must both exist, and must differ (`other_account_id must be a different account`).
  - `payee` and `note` default to `""` and are trimmed. The payee is the user's text, on both legs.
- Editing and deleting go through the leg's `/api/transactions/{id}` routes above, so there's one edit path for every row in the table.
- Transfers never count as spending. They have no category, and M5's reports must also leave them out of "Uncategorized" (`transfer_id IS NULL`).

## Goals (M4)

```ts
type Goal = {
  category_id: number
  name: string
  cap_cents: number | null        // the cap in effect for the month; null = no cap
  cap_since: string | null        // "YYYY-MM" the cap was set, which may be earlier
  rollover_since: string | null   // as on the category
  carryover_cents: number         // leftover from earlier months; 0 without rollover
  available_cents: number | null  // cap + carryover; null with no cap and no rollover
  spent_cents: number
}
type Goals = {
  month: string                   // "YYYY-MM", echoed or defaulted
  goals: Goal[]                   // every expense category, sorted by name (case-insensitive)
  uncategorized_cents: number     // uncategorized spending in the month, not counting transfers
}
```

| Request | Success | Errors |
|---|---|---|
| `GET /api/goals?month=2026-11` | 200, `Goals` | 400 invalid `month` |
| `PUT /api/goals/{category_id}?month=2026-12` `{amount_cents}` | 200, the category's `Goal` for that month | 400 invalid `month` or body, or an income category. 404 no such category |

- `month` is optional on `GET` and defaults to the current month (server local time). It's required on `PUT`.
- **Caps** are stored by effective month. The cap for month M is the latest one set for a month `<= M`, so changing December's cap leaves November alone.
- `PUT` sets the cap from `month` on. `amount_cents` is required: a positive number sets the cap, and `null` removes it from that month on. Zero or negative is a 400, and so is `{}`.
- **Spent** for month M is `-SUM(amount_cents)` over the category's transactions dated in M. Refunds reduce it, so it can be negative.
- **Carryover**, when rollover is on and `rollover_since <= M`, is the sum of (cap − spent) for each month from `rollover_since` up to, but not including, M. A month with no cap adds nothing but still subtracts what was spent. Before `rollover_since`, carryover is 0.
- **Available** is the month's cap (0 if none) plus carryover when rollover applies; otherwise just the cap. It can be negative.
- Deleting a category deletes its goals.
- Transfer legs have no category, so they're never spending, and they're left out of `uncategorized_cents` too.

## Payees (M2)

```ts
type Payee = { payee: string; category_id: number | null }
```

| Request | Success | Errors |
|---|---|---|
| `GET /api/payees` | 200, `Payee[]` sorted by payee (case-insensitive) | |

- One entry per payee across all accounts, matched case-insensitively. Empty payees are left out.
- Both `payee` and `category_id` come from that payee's most recent transaction (`date DESC, id DESC`). So the spelling shown is the latest one, and the category is the one autofill should pick.
- `category_id` is `null` if the latest transaction was uncategorized. The entry row then leaves the category alone.
- No transactions returns `[]`.
- Transfer legs are left out (M3). Their category is always `null`, so they'd only clutter autofill.
- The frontend filters the list as you type. There's no `?q=`, since years of manual entry stay small (see "No pagination" in the plan).

## Hints, if you want them

- The M1 handlers are the pattern: an input struct with `validate()`, a store method, and `writeStoreError`. The test helpers `doJSON`, `decode[T]`, `wantStatus` (api) and `newTestStore`, `wantKind` (store) are ready to use.
- A nullable JSON number decodes cleanly into `*int64`. When writing it to SQLite, `nil` becomes NULL. When reading it back, scan into `sql.NullInt64` or `*int64`.
- **Path ids:** register routes like `"PATCH /api/accounts/{id}"` and read the id with `r.PathValue("id")`, then `strconv.ParseInt`. One small `pathID(r) (int64, error)` helper covers all six routes.
- **Partial PATCH:** use pointer fields (`Name *string`, `Type *string`, `IsIncome *bool`); `nil` means not sent. Build the `SET` clause from the non-nil fields, the same way `ListTransactions` builds its `WHERE`.
- **404 vs no-op:** `res.RowsAffected()` is 0 when no row matched the id. That's simpler than an existence check before an `UPDATE` or `DELETE`. Then read the row back for the `PATCH` response.
- **In-use deletes:** the foreign keys are `ON DELETE RESTRICT`, so deleting a used account or category already fails with a foreign key error. `isForeignKey` alone can't give the count for the message, so count first (`SELECT COUNT(*) FROM transactions WHERE account_id = ?`). Keep the `isForeignKey` check as a backstop.
- **Opening balance:** use `s.db.BeginTx`, run both inserts on the `*sql.Tx`, and `defer tx.Rollback()` (it's a no-op after `Commit`). Find the category with `SELECT id FROM categories WHERE name = 'Opening balance'`. Use `time.Now().Format(time.DateOnly)` for the date.
- **Payees:** SQLite window functions do the "latest per payee" in one query:
  ```sql
  SELECT payee, category_id FROM (
    SELECT payee, category_id,
           ROW_NUMBER() OVER (PARTITION BY payee COLLATE NOCASE ORDER BY date DESC, id DESC) AS n
    FROM transactions WHERE payee <> ''
  ) WHERE n = 1 ORDER BY payee COLLATE NOCASE
  ```
- **Transfers (M3):** add the column with `ALTER TABLE transactions ADD COLUMN transfer_id INTEGER REFERENCES transfers (id) ON DELETE CASCADE`. Then deleting the `transfers` row deletes both legs, so a leg's `DELETE` is just `DELETE FROM transfers WHERE id = (SELECT transfer_id FROM transactions WHERE id = ?)`. Get `transfer_account_id` with a self-join on `transfer_id` and `id <>`.
- **Optional fields (M4):** a pointer can't tell an omitted field from `null`, which `rollover_since` and the goal `amount_cents` need. `optional[T]` in `internal/api/json.go` records whether the field was present at all.
