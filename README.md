# Budgeteer

A small, local-first budgeting app for macOS. You enter expenses by hand, set
a monthly cap per category (optionally rolling over what's left), and look at
your spending in charts. It's one Go binary with the web app built in, serving
on localhost. There's no account, no sync, and no cloud.

## Requirements

- [Go](https://go.dev/dl/) 1.26.2 or later
- [Node.js](https://nodejs.org/) 22.12 or later (only to build the frontend)

## Build and run

```sh
make build
./bin/budgeteer
```

Then open <http://127.0.0.1:7420>. `make build` installs the npm packages the
first time, builds the frontend, and embeds it in `bin/budgeteer`. The binary
has no other files to install, so you can move it anywhere.

| Flag | Default | |
|---|---|---|
| `--addr` | `127.0.0.1:7420` | Address to listen on. Keep it on loopback: there's no login. |
| `--db` | `~/Library/Application Support/Budgeteer/budget.db` | The SQLite database. It's created on first run. |
| `--version` | | Print the version and exit. |

## Your data

Everything is in the one database file above. To back it up, quit Budgeteer
and copy `budget.db`. While it's running, use
`sqlite3 budget.db ".backup budget-backup.db"` instead, since a plain copy can
miss recent writes.

To try a new build without touching your real data, point it at another file:
`./bin/budgeteer --db ./dev.db`.

## Development

```sh
make dev        # Go API on :7420 (using ./dev.db) + Vite on :5173; open http://localhost:5173
make test       # Go vet and tests, Prettier check, typecheck, Vitest
make fmt        # gofmt and Prettier
make reset-dev  # delete ./dev.db; the next `make dev` recreates it
make help       # list every target
```

The code is Go (`cmd/`, `internal/`) with SQLite, and React with TypeScript
and Vite (`web/`). [`docs/plan.md`](docs/plan.md) has the decisions, schema
and milestones, and [`docs/api.md`](docs/api.md) is the API contract.
