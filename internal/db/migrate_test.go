package db

import (
	"context"
	"database/sql"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func openTemp(t *testing.T) *sql.DB {
	t.Helper()
	// The space mirrors "Application Support" in the real default path.
	path := filepath.Join(t.TempDir(), "with space", "test.db")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func userVersion(t *testing.T, db *sql.DB) int {
	t.Helper()
	var v int
	if err := db.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

// upTo returns the first n embedded migrations.
func upTo(t *testing.T, n int) fstest.MapFS {
	t.Helper()
	migrations, err := fs.Sub(embedded, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	names, err := fs.Glob(migrations, "*.sql")
	if err != nil {
		t.Fatal(err)
	}
	m := fstest.MapFS{}
	for _, name := range names[:n] {
		data, err := fs.ReadFile(migrations, name)
		if err != nil {
			t.Fatal(err)
		}
		m[name] = &fstest.MapFile{Data: data}
	}
	return m
}

func TestOpenSetsPragmas(t *testing.T) {
	db := openTemp(t)

	var fk int
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&fk); err != nil {
		t.Fatal(err)
	}
	if fk != 1 {
		t.Errorf("foreign_keys = %d, want 1", fk)
	}

	var mode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatal(err)
	}
	if mode != "wal" {
		t.Errorf("journal_mode = %q, want wal", mode)
	}
}

func TestEmbeddedMigrationsApply(t *testing.T) {
	db, err := Open(context.Background(), filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if v := userVersion(t, db); v < 1 {
		t.Errorf("user_version = %d, want >= 1", v)
	}
}

func TestMigrateAppliesInOrderAndIsIdempotent(t *testing.T) {
	db := openTemp(t)
	ctx := context.Background()
	fsys := fstest.MapFS{
		"0001_a.sql": {Data: []byte("CREATE TABLE a (id INTEGER PRIMARY KEY);")},
		"0002_b.sql": {Data: []byte("CREATE TABLE b (id INTEGER PRIMARY KEY); INSERT INTO a DEFAULT VALUES;")},
	}

	if err := Migrate(ctx, db, fsys); err != nil {
		t.Fatal(err)
	}
	if v := userVersion(t, db); v != 2 {
		t.Fatalf("user_version = %d, want 2", v)
	}

	// Running again must not re-apply 0002 (which would insert a second row).
	if err := Migrate(ctx, db, fsys); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := db.QueryRow("SELECT count(*) FROM a").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("rows in a = %d, want 1", n)
	}

	// A later migration is picked up on the next run.
	fsys["0003_c.sql"] = &fstest.MapFile{Data: []byte("CREATE TABLE c (id INTEGER PRIMARY KEY);")}
	if err := Migrate(ctx, db, fsys); err != nil {
		t.Fatal(err)
	}
	if v := userVersion(t, db); v != 3 {
		t.Errorf("user_version = %d, want 3", v)
	}
}

func TestMigrateRollsBackFailedMigration(t *testing.T) {
	db := openTemp(t)
	fsys := fstest.MapFS{
		"0001_ok.sql":  {Data: []byte("CREATE TABLE a (id INTEGER PRIMARY KEY);")},
		"0002_bad.sql": {Data: []byte("CREATE TABLE b (id INTEGER PRIMARY KEY); NOT VALID SQL;")},
	}

	if err := Migrate(context.Background(), db, fsys); err == nil {
		t.Fatal("expected error")
	}
	if v := userVersion(t, db); v != 1 {
		t.Errorf("user_version = %d, want 1", v)
	}
	var n int
	if err := db.QueryRow("SELECT count(*) FROM sqlite_master WHERE name = 'b'").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Error("table b exists after failed migration")
	}
}

func TestMigrateRunsWithForeignKeysOff(t *testing.T) {
	db := openTemp(t)
	fsys := fstest.MapFS{
		"0001_a.sql": {Data: []byte("CREATE TABLE a (id INTEGER PRIMARY KEY); CREATE TABLE b (a_id INTEGER REFERENCES a (id) ON DELETE RESTRICT); INSERT INTO a (id) VALUES (1); INSERT INTO b VALUES (1);")},
		// Rebuilding a referenced table only works with foreign keys off.
		"0002_rebuild.sql": {Data: []byte("CREATE TABLE a_new (id INTEGER PRIMARY KEY); INSERT INTO a_new SELECT id FROM a; DROP TABLE a; ALTER TABLE a_new RENAME TO a;")},
	}
	if err := Migrate(context.Background(), db, fsys); err != nil {
		t.Fatal(err)
	}

	var fk int
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&fk); err != nil {
		t.Fatal(err)
	}
	if fk != 1 {
		t.Errorf("foreign_keys = %d after migrating, want 1", fk)
	}
	if _, err := db.Exec("DELETE FROM a"); err == nil {
		t.Error("deleting a referenced row succeeded, want the RESTRICT to hold after the rebuild")
	}
}

func TestMigrateRejectsForeignKeyViolation(t *testing.T) {
	db := openTemp(t)
	fsys := fstest.MapFS{
		"0001_a.sql":   {Data: []byte("CREATE TABLE a (id INTEGER PRIMARY KEY); CREATE TABLE b (a_id INTEGER REFERENCES a (id)); INSERT INTO a (id) VALUES (1); INSERT INTO b VALUES (1);")},
		"0002_bad.sql": {Data: []byte("DELETE FROM a;")},
	}
	err := Migrate(context.Background(), db, fsys)
	if err == nil || !strings.Contains(err.Error(), "foreign key violation") {
		t.Fatalf("err = %v, want a foreign key violation", err)
	}
	if v := userVersion(t, db); v != 1 {
		t.Errorf("user_version = %d, want 1", v)
	}
	var n int
	if err := db.QueryRow("SELECT count(*) FROM a").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("rows in a = %d, want the delete rolled back", n)
	}
}

func TestMigrateRejectsBadSequences(t *testing.T) {
	cases := map[string]fstest.MapFS{
		"gap":       {"0001_a.sql": {}, "0003_c.sql": {}},
		"duplicate": {"0001_a.sql": {}, "0001_b.sql": {}},
		"bad name":  {"init.sql": {}},
	}
	for name, fsys := range cases {
		t.Run(name, func(t *testing.T) {
			if err := Migrate(context.Background(), openTemp(t), fsys); err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestMigrateRejectsNewerDatabase(t *testing.T) {
	db := openTemp(t)
	if _, err := db.Exec("PRAGMA user_version = 5"); err != nil {
		t.Fatal(err)
	}
	err := Migrate(context.Background(), db, fstest.MapFS{"0001_a.sql": {}})
	if err == nil || !strings.Contains(err.Error(), "newer build") {
		t.Errorf("err = %v, want newer-build error", err)
	}
}

func TestTransfersMigrationKeepsExistingData(t *testing.T) {
	db := openTemp(t)
	ctx := context.Background()
	migrations, err := fs.Sub(embedded, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	only := func(names ...string) fstest.MapFS {
		m := fstest.MapFS{}
		for _, name := range names {
			data, err := fs.ReadFile(migrations, name)
			if err != nil {
				t.Fatal(err)
			}
			m[name] = &fstest.MapFile{Data: data}
		}
		return m
	}

	// A database from before M3, with a transaction in it.
	if err := Migrate(ctx, db, only("0001_init.sql")); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO accounts (name, type) VALUES ('Checking', 'checking'), ('Savings', 'savings')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO transactions (account_id, date, amount_cents) VALUES (1, '2026-10-01', -500)`); err != nil {
		t.Fatal(err)
	}

	if err := Migrate(ctx, db, only("0001_init.sql", "0002_transfers.sql")); err != nil {
		t.Fatal(err)
	}
	var amount int
	var transferID sql.NullInt64
	if err := db.QueryRow(`SELECT amount_cents, transfer_id FROM transactions`).Scan(&amount, &transferID); err != nil {
		t.Fatal(err)
	}
	if amount != -500 || transferID.Valid {
		t.Errorf("existing row = %d, %v; want -500 with no transfer", amount, transferID)
	}

	// Deleting a transfer deletes both legs.
	if _, err := db.Exec(`INSERT INTO transfers (id) VALUES (1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO transactions (account_id, date, amount_cents, transfer_id) VALUES (1, '2026-10-02', -100, 1), (2, '2026-10-02', 100, 1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM transfers WHERE id = 1`); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM transactions`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("%d transactions after deleting the transfer, want 1", n)
	}
}

func TestGoalsMigrationKeepsExistingData(t *testing.T) {
	db := openTemp(t)
	ctx := context.Background()

	// A database from before M4, with a category in use.
	if err := Migrate(ctx, db, upTo(t, 2)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO categories (name) VALUES ('Coffee')`); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(ctx, db, upTo(t, 3)); err != nil {
		t.Fatal(err)
	}

	var rollover sql.NullString
	if err := db.QueryRow(`SELECT rollover_since FROM categories WHERE name = 'Coffee'`).Scan(&rollover); err != nil {
		t.Fatal(err)
	}
	if rollover.Valid {
		t.Errorf("rollover_since = %q, want NULL", rollover.String)
	}

	// Goals go with their category.
	if _, err := db.Exec(`INSERT INTO goals (category_id, effective_month, amount_cents) SELECT id, '2026-10', 5000 FROM categories WHERE name = 'Coffee'`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM categories WHERE name = 'Coffee'`); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM goals`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("%d goals left after deleting the category, want 0", n)
	}
}

func TestInvestmentAccountsMigrationKeepsExistingData(t *testing.T) {
	db := openTemp(t)
	ctx := context.Background()

	// A database from before M7, with a transfer between two accounts.
	if err := Migrate(ctx, db, upTo(t, 3)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO accounts (id, name, type) VALUES (3, 'Checking', 'checking'), (7, 'Visa', 'credit')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO transfers (id) VALUES (1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO transactions (account_id, date, amount_cents, transfer_id) VALUES (3, '2026-10-02', -100, 1), (7, '2026-10-02', 100, 1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO accounts (name, type) VALUES ('Brokerage', 'investment')`); err == nil {
		t.Fatal("investment account accepted before 0004")
	}

	if err := Migrate(ctx, db, upTo(t, 4)); err != nil {
		t.Fatal(err)
	}

	var got string
	if err := db.QueryRow(`SELECT group_concat(row, ' ') FROM (SELECT a.name || ':' || a.type || ':' || t.amount_cents AS row FROM transactions t JOIN accounts a ON a.id = t.account_id ORDER BY t.id)`).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if want := "Checking:checking:-100 Visa:credit:100"; got != want {
		t.Errorf("transactions joined to accounts = %q, want %q", got, want)
	}

	if _, err := db.Exec(`INSERT INTO accounts (name, type) VALUES ('Brokerage', 'investment')`); err != nil {
		t.Errorf("investment account: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO accounts (name, type) VALUES ('Piggy', 'piggy')`); err == nil {
		t.Error("unknown type accepted after 0004")
	}
	if _, err := db.Exec(`INSERT INTO accounts (name, type) VALUES ('checking', 'cash')`); err == nil {
		t.Error("case-insensitive duplicate name accepted after 0004")
	}
	if _, err := db.Exec(`DELETE FROM accounts WHERE id = 3`); err == nil {
		t.Error("deleted an account with transactions, want the RESTRICT to hold")
	}
}
