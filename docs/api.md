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

## Accounts (done, use as the reference)

Code: `internal/store/accounts.go`, `internal/api/accounts.go`, and their tests.

```ts
type Account = { id: number; name: string; type: 'checking' | 'savings' | 'credit' | 'cash'; balance_cents: number }
```

| Request | Success | Errors |
|---|---|---|
| `GET /api/accounts` | 200, `Account[]` sorted by name (case-insensitive) | |
| `POST /api/accounts` `{name, type}` | 201, the new `Account` | 400 blank name, bad type, unknown field, bad JSON. 409 duplicate name (case-insensitive) |

## Categories (M1, yours)

```ts
type Category = { id: number; name: string; is_income: boolean }
```

| Request | Success | Errors |
|---|---|---|
| `GET /api/categories` | 200, `Category[]` with expense categories first, then income, each sorted by name (case-insensitive) | |
| `POST /api/categories` `{name, is_income?}` | 201, the new `Category`. `is_income` defaults to `false` | 400 blank name, unknown field, bad JSON. 409 duplicate name (case-insensitive) |

- Trim `name`.
- Migration 0001 seeds `Income` and `Opening balance` as income categories, so a fresh database returns two categories, not `[]`.

## Transactions (M1, yours)

```ts
type Transaction = {
  id: number
  account_id: number
  category_id: number | null
  date: string          // "YYYY-MM-DD"
  payee: string
  amount_cents: number  // negative = money out, positive = money in
  note: string
}
```

| Request | Success | Errors |
|---|---|---|
| `GET /api/transactions?account_id=1&from=2026-10-01&to=2026-10-31` | 200, `Transaction[]`, newest first (`date DESC, id DESC`) | 400 missing or invalid `account_id`, or invalid `from`/`to`. 404 no such account |
| `POST /api/transactions` `{account_id, category_id, date, payee?, amount_cents, note?}` | 201, the new `Transaction` | 400 for any of the rules below |

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

## Hints, if you want them

- Copy the accounts files as a starting point: input struct with `validate()`, store method, `writeStoreError`.
- A nullable JSON number decodes cleanly into `*int64`. When writing it to SQLite, `nil` becomes NULL. When reading it back, scan into `sql.NullInt64` or `*int64`.
- `time.Parse(time.DateOnly, s)` both checks the format and rejects impossible dates.
- A bad `account_id` or `category_id` makes the insert fail with a foreign key error (`isForeignKey` in `store.go`). SQLite doesn't say which column caused it. Either check existence first, or turn the error into `errorf(Invalid, …)` with a message covering both.
- The test helpers `doJSON`, `decode[T]`, `wantStatus` (api) and `newTestStore`, `wantKind` (store) are ready to use.
- `TestAccountBalanceSumsTransactions` inserts transactions with raw SQL. Once you have `CreateTransaction`, switching it over is a nice check that your method works.
