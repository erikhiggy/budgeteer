# Budgeteer plan

A small, local-first personal budgeting app for macOS, meant as a lean
replacement for Budget with Buckets. Expenses are entered by hand, there's one
user, and everything runs on localhost from a single Go binary.

There are three views:

1. **Transactions:** enter and browse transactions per account.
2. **Goals:** a monthly spending cap per category, with spent vs. cap. A category can roll over, so unspent money carries into next month and overspending comes out of it.
3. **Charts:** spending visualizations.

The API contract lives in [`api.md`](api.md).

## Status

| Milestone | State |
|---|---|
| M0 Skeleton | Done |
| M1 Thin slice | Done |
| M2 Transactions view | Done |
| M3 Transfers | Done |
| M4 Goals | Done |
| M5 Charts | Done |
| M6 Polish | Done |
| M7 Investment accounts | Done |
| M8 Sidebar groups | In progress |

## Decisions

These are settled and shouldn't be reopened without a reason.

- **Stack:**
  - Go with SQLite via `modernc.org/sqlite` (pure Go, no cgo).
  - React 19, TypeScript and Vite.
  - CSS Modules, not Tailwind.
  - Recharts, React Router and TanStack Query.
  - Prettier, using the editor's settings: tabs, single quotes, single quotes in JSX.
- **Money** is stored as integer cents. A plain amount is money out (stored negative). A leading `+` means money in (stored positive), and a leading `-` is also accepted as money out.
- **Income** is a flag on categories (`is_income`). Income categories are excluded from spending. Paychecks and opening balances use income categories.
- **Transfers** are tracked on both sides, as a linked pair of transactions (a `transfers` row with two legs). They have no category, so they never count as spending.
- **Goal caps** are stored by effective month. The cap for month M is the latest row with `effective_month <= M`. Changing a cap never rewrites earlier months.
- **Rollover** is caps plus carryover, not envelope budgeting. It's off by default and turned on per category from a start month (`categories.rollover_since`). For such a category, each month's leftover (cap minus spending, negative if overspent) carries into the next. There's no assigning income to categories, and credit cards need no special handling: spending is still just categorized transactions.
- **Dates** are `YYYY-MM-DD` text with no time zone.
- **Server** binds to `127.0.0.1:7420` only. Middleware requires a loopback `Host` (blocks DNS rebinding) and `Content-Type: application/json` on mutations (forces a CORS preflight, so other sites can't post to the API).

## Project structure

```
budgeteer/
├── Makefile                    # help, dev, reset-dev, build, test, fmt, clean
├── README.md                   # setup, usage, backups
├── cmd/budgeteer/main.go       # flags: --addr (127.0.0.1:7420), --db, --version
├── docs/                       # this plan, api.md
├── internal/
│   ├── db/                     # open + pragmas, migration runner, migrations/*.sql
│   ├── store/                  # SQL, one file per entity; store.Error{Kind, Msg}
│   └── api/                    # routes, guard middleware, JSON helpers, handlers
└── web/                        # Vite project
    ├── embed.go                # //go:embed all:dist
    ├── vite.config.ts          # /api proxy, Vitest (jsdom)
    └── src/
        ├── api/                # types.ts (mirrors api.md), client.ts, hooks.ts
        ├── components/         # CategorySelect, PayeeCombobox, ConfirmButton, MonthPicker, Notice, Button.module.css
        ├── lib/                # money, date, month, cx, chartColors (+ tests)
        ├── styles/             # reset, tokens (CSS variables), global
        └── views/              # transactions/, categories/, goals/, charts/
```

- **Dev:** `make dev` runs the Go API against `./dev.db` and Vite on :5173, which proxies `/api` to Go. `make reset-dev` deletes the dev database, and refuses while :7420 is in use.
- **Production:** `make build` builds the frontend, then the Go binary with the frontend embedded. With no flags the database is `~/Library/Application Support/Budgeteer/budget.db`.
- **Tests:** `make test` runs Go vet and tests, Prettier check, typecheck and Vitest. Go tests each get a fresh migrated database in `t.TempDir()`.
- **Frontend serving:** paths that aren't files fall back to `index.html`. A missing `/assets/*` file returns 404. Hashed assets are cached as immutable; `index.html` is served `no-cache`.
- **Code layout:** handlers validate input, store methods run the SQL, and `writeStoreError` maps `store.Error` kinds to status codes (NotFound → 404, Conflict → 409, Invalid → 400; anything else → 500 and gets logged).

## Schema

Migrations are forward-only, numbered `NNNN_name.sql` files. They're applied by
`PRAGMA user_version`, each in its own transaction, with no gaps allowed.
Foreign keys are off while a migration runs, so a migration can rebuild a
table; `PRAGMA foreign_key_check` must come back clean before it commits.
**Don't edit a migration after it has shipped.** In development, `make reset-dev`
instead.

- **0001_init (done):**
  - `accounts(id, name UNIQUE NOCASE, type ∈ checking|savings|credit|cash)`
  - `categories(id, name UNIQUE NOCASE, is_income)`, seeded with "Income" and "Opening balance"
  - `transactions(id, account_id, category_id NULL, date, payee, amount_cents, note, created_at)`
  - Indexes: (account_id, date, id), (date, category_id), payee
- **0002_transfers (M3, done):**
  - New table `transfers(id)`
  - `transactions.transfer_id` references it with `ON DELETE CASCADE`
  - "A transfer has no category" is enforced in the store, since `ALTER TABLE` can't add a CHECK
- **0003_goals (M4, done):** `goals(category_id, effective_month 'YYYY-MM', amount_cents NULL)`
- **0004_investment_accounts (M7, done):** rebuilds `accounts` to add `investment` to the type CHECK, since SQLite can't change a CHECK in place. Ids are kept, so transactions still point at the same accounts
  - Primary key is (category_id, effective_month); rows go with their category (`ON DELETE CASCADE`), and moving a category to income deletes them
  - NULL means the cap was removed from that month on
  - `categories.rollover_since 'YYYY-MM' NULL` added with `ALTER TABLE`; NULL means rollover is off

**Derived numbers:**
- **Account balance** is `SUM(amount_cents)` for the account.
- **Spending** for category C in month M is `-SUM(amount_cents)` over that category's transactions in M, for expense categories only. Refunds reduce spending.
- **Uncategorized** money going out shows as "Uncategorized".
- **Carryover** for a rollover category in month M is the sum of (cap − spending) over every month from `rollover_since` up to, but not including, M. A month with no cap adds nothing but still subtracts its spending. Without rollover, carryover is 0.
- **Available** for month M is the cap for M plus the carryover. It's what the Goals view compares spending against, and it can be negative.
- **Opening balance:** a transaction in the "Opening balance" category, created by the add-account form (M2).

## API

See [`api.md`](api.md) for request and response shapes. Planned routes:

| Route | Milestone |
|---|---|
| `GET /api/health` | M0 ✓ |
| `GET / POST /api/accounts` | M1 ✓ |
| `GET / POST /api/categories` | M1 ✓ |
| `GET / POST /api/transactions?account_id=&from=&to=` | M1 ✓ |
| `PATCH / DELETE /api/accounts/{id}`, `/api/categories/{id}`, `/api/transactions/{id}`. `opening_balance_cents` on account create | M2 ✓ |
| `GET /api/payees`: `[{payee, category_id}]` with the latest category per payee | M2 ✓ |
| `POST /api/transfers`; editing and deleting a leg through `/api/transactions/{id}` applies to the whole transfer | M3 ✓ |
| `GET /api/goals?month=` (cap, spent, carryover, available per expense category), `PUT /api/goals/{category_id}?month=`, `rollover_since` on category `PATCH` | M4 ✓ |
| `GET /api/reports/spending-by-category?month=`, `GET /api/reports/monthly-spending?from=&to=` | M5 ✓ |

## Frontend

- **State:**
  - Server data lives in TanStack Query. Window-focus refetch is off and data never goes stale on its own; 4xx responses aren't retried.
  - Each mutation invalidates only what it touched, and only queries on screen refetch.
  - The URL holds the selected account and month.
  - Form fields are local state.
  - No global store.
- **Entry row** (`views/transactions/EntryRow.tsx`):
  - Fields in Tab order: date, payee, category, amount, note.
  - Required: date, amount, and a category or a transfer target.
  - **Rule 1:** Enter on an incomplete row sends nothing, focuses the first missing field, and highlights every missing field.
  - **Rule 2:** Enter on a complete row saves it from any field. Repeated Enter is ignored while the save is in flight. After saving, the row clears but keeps the date, and focus returns to payee.
  - **Rule 3 (M2):** Enter while a payee suggestion is highlighted picks the suggestion instead of saving. This needs a custom `PayeeCombobox`, because a native `<datalist>` can't tell the page whether its popup is open.
  - Esc closes the suggestions if they're open, otherwise clears the row.
  - Picking or typing a known payee fills in its latest category, unless one is already chosen.
  - Category is a native `<select>` grouped into Spending / Income, plus a "Transfer" group listing the other accounts (`⇄ Savings`). A transfer target saves through `POST /api/transfers`.
- **Table:** inline edit (the entry row in edit mode, same validation and Enter rules), a two-click delete (`ConfirmButton`), and transfer legs shown as "Transfer → X" / "Transfer ← X". Editing a leg keeps its other account fixed.
- **Accounts:** a sidebar lists accounts with balances and a total. The header above the ledger renames, retypes and deletes the selected account. The add-account form takes an optional opening balance, where a plain number is money you have and `-` is money owed.
  - **Grouped by type (M8):** the sidebar shows a group per account type, in the order Checking, Savings, Cash, Credit cards, Investments. Each group has a heading with its subtotal, and its accounts are sorted by name, as the API returns them. Types with no accounts are left out. The total at the bottom stays the sum of every account. Frontend only: the API doesn't change.
- **Goals (M4):**
  - Route `/goals/:month` (`YYYY-MM`), with a month picker at the top.
  - A row per expense category: inline-editable cap (Enter saves, empty removes, Esc cancels), spent, remaining, and a progress bar that turns `--color-over-cap` past 100%.
  - A "since Aug 2026" label when the cap comes from an earlier month.
  - A rollover toggle per category, starting from the month on screen. Rollover rows measure against available instead of the cap, and show the carryover, e.g. "+200.00 rolled over" or "−35.00 from last month".
  - An "Uncategorized" row when the month has uncategorized spending.
- **Charts (M5):**
  - Route `/charts/:month`, with the same `MonthPicker` as Goals. The view is lazy-loaded, since Recharts is about half the bundle.
  - Spending by category: horizontal bars, biggest first, amount at each tip, "Uncategorized" as its own bar.
  - Last 12 months: a column per month ending at the URL month, the selected month solid and the rest lighter. Clicking a column goes to that month.
  - Goal vs. actual: from `GET /api/goals`, grouped bars of goal and spent for each category with a cap or rollover. The goal is available when rollover applies, otherwise the cap, as on the Goals progress bar.
  - Colors are tokens (`--color-chart-1`, `--color-chart-2`, `--color-chart-grid`) used through `lib/chartColors.ts`; the pair was checked for colorblind separation and 3:1 contrast. Axis ticks come from `niceTicks` in `views/charts/chartData.ts`.
- **Styling:**
  - Every color, spacing and type value is a variable in `tokens.css`.
  - `global.css` has the base look for form controls.
  - Each component has a `.module.css` next to it.

## Milestones

Each milestone is runnable on its own and ends with `make test` passing.

| Milestone | Scope | Done when |
|---|---|---|
| **M0 Skeleton** ✓ | Go server, health check, migration runner, Vite/React/TS, CSS tokens, proxy, embed, Makefile, Vitest, Prettier | `make build && ./bin/budgeteer` serves the app |
| **M1 Thin slice** ✓ | 0001, accounts, categories and transactions APIs. Frontend: account picker and balance, first-run account form, entry row (rules 1–2), table, Categories view | Add a transaction in the browser; it survives a restart. Handler, store and EntryRow tests |
| **M2 Transactions view** ✓ | Edit and delete everywhere, 409 when deleting something in use, `PayeeCombobox` (rule 3) and `/api/payees`, category autofill, sidebar with balances, opening balance | Component tests for rule 3 and the Esc order. Ten rows entered without the mouse |
| **M3 Transfers** ✓ | 0002, transfer endpoints, "Transfer to…", legs edit and delete together | A transfer moves both balances and leaves spending unchanged |
| **M4 Goals** ✓ | 0003, goals endpoints, rollover, Goals view | A cap set in Oct and changed in Dec: Nov shows the Oct cap. A $100 rollover category from Oct with no spending shows $300 available in Dec, and overspending in Nov lowers it |
| **M5 Charts** ✓ | Report endpoints, three charts | Report totals equal the `/api/goals` spent values |
| **M6 Polish** ✓ | `--version`, README, dev-script cleanup | A fresh clone builds with `make build` |
| **M7 Investment accounts** ✓ | 0004, an `investment` account type (a label only), migrations run with foreign keys off | An existing database migrates with its accounts and transactions intact, and an investment account can be added |
| **M8 Sidebar groups** | Sidebar grouped by account type, with a subtotal per group | Accounts of every type show under the right heading with correct subtotals, and empty types are hidden. AccountSidebar component test |

## Next up

All the planned milestones are done. New work comes from "Later" below; give
each item a milestone in the table, with the same loop as before: contract in
`api.md` first, then backend, frontend, tests, and a browser check against a
scratch database.

- **Versions:** `make build` stamps `--version` from `git describe --tags --always --dirty`, so it's a commit hash until the first tag. Tag a release with `git tag v1.0.0`.
- **Still to check by hand:** chart tooltips and clicking a trend column (M5 was checked by screenshot only).

## Risks and notes

- **Deleting in-use items:** deleting an account or category that has transactions returns a 409. Archiving them instead is listed under "later".
- **Embed needs a placeholder:** `go build` needs `web/dist/.gitkeep`. Vite deletes it on every build, so the build script recreates it.
- **No pagination:** years of manual entry is well under 10k rows, and the list endpoint already accepts `from`/`to`.
- **Date input:** the native date input looks different in each browser. A custom date field is listed under "later".
- **Real data:** a built binary run without `--db` writes to the real database. Try release builds with `--db ./dev.db`.
- **Ignored directory:** `go.mod` has `ignore ./web/node_modules`, so `go test ./...` never looks inside npm packages.
- **Vitest in a sandbox:** in a sandboxed shell, Vitest's default forked workers can time out at startup. `npx vitest run --pool=threads` works; plain `make test` is fine on a normal machine.
- **Enter in plain forms:** the add-account, edit-account and rename-category forms submit through the browser's built-in Enter handling. Browser automation can't trigger that, so check Enter on those forms by hand after changing them.
- **Rollover before its start month:** a month earlier than `rollover_since` still shows the category's rollover box ticked (labelled "rolls over from …"), and unticking it there turns rollover off entirely.

## Out of scope

Bank sync or aggregators, importing from Budget with Buckets, backups,
phone/mobile entry, investment tracking (holdings, prices; an `investment`
account is only a label), net worth, forecasting, split transactions,
envelope budgeting (assigning income to categories), multi-user, auth.

## Later

Dark mode (redefine the tokens), search and filter, running-balance column,
undo instead of a delete confirm, archiving accounts and categories, trend
split by category, CSV export, recurring transactions, a custom category
combobox, a custom date field, typed CSS Module class names, start at login
via launchd, opening the browser on launch, pagination, leaving "Opening balance"
out of payee suggestions.
