package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"path/filepath"
	"testing"

	"budgeteer/internal/db"
	"budgeteer/internal/store"
)

// newTestServerWithDB is newTestServer plus the connection, so tests can
// insert transfers with raw SQL until POST /api/transfers exists.
func newTestServerWithDB(t *testing.T) (http.Handler, *sql.DB) {
	t.Helper()
	conn, err := db.Open(context.Background(), filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return New(conn, builtDist), conn
}

// insertTestTransfer adds a transfer's legs with raw SQL: cents in account a
// and -cents in account b. It returns the legs' ids.
func insertTestTransfer(t *testing.T, conn *sql.DB, a, b int64, cents int64) (legA, legB int64) {
	t.Helper()
	res, err := conn.Exec(`INSERT INTO transfers DEFAULT VALUES`)
	if err != nil {
		t.Fatal(err)
	}
	transferID, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	var ids [2]int64
	for i, leg := range [][2]int64{{a, cents}, {b, -cents}} {
		res, err := conn.Exec(
			`INSERT INTO transactions (account_id, date, payee, amount_cents, transfer_id) VALUES (?, '2026-10-01', 'Savings plan', ?, ?)`,
			leg[0], leg[1], transferID)
		if err != nil {
			t.Fatal(err)
		}
		if ids[i], err = res.LastInsertId(); err != nil {
			t.Fatal(err)
		}
	}
	return ids[0], ids[1]
}

func listTestTransactions(t *testing.T, h http.Handler, accountID int64) []store.Transaction {
	t.Helper()
	res, raw := doJSON(t, h, "GET", fmt.Sprintf("/api/transactions?account_id=%d", accountID), nil)
	wantStatus(t, res, raw, http.StatusOK)
	return decode[[]store.Transaction](t, raw)
}

func TestTransferLegWireFormat(t *testing.T) {
	h, conn := newTestServerWithDB(t)
	checking := createTestAccount(t, h, "Checking")
	savings := createTestAccount(t, h, "Savings")
	insertTestTransfer(t, conn, checking.ID, savings.ID, -20000)
	createTestTransaction(t, h, map[string]any{"account_id": checking.ID, "date": "2026-10-02", "amount_cents": -500})

	res, raw := doJSON(t, h, "GET", fmt.Sprintf("/api/transactions?account_id=%d", checking.ID), nil)
	wantStatus(t, res, raw, http.StatusOK)
	list := decode[[]map[string]any](t, raw)
	if len(list) != 2 {
		t.Fatalf("list = %v", list)
	}
	if list[0]["transfer_id"] != nil || list[0]["transfer_account_id"] != nil {
		t.Errorf("ordinary = %v, want null transfer fields", list[0])
	}
	if list[1]["transfer_id"] == nil || list[1]["transfer_account_id"] != float64(savings.ID) || list[1]["category_id"] != nil {
		t.Errorf("leg = %v", list[1])
	}
}

func TestPatchTransferLeg(t *testing.T) {
	h, conn := newTestServerWithDB(t)
	checking := createTestAccount(t, h, "Checking")
	savings := createTestAccount(t, h, "Savings")
	c := createTestCategory(t, h, "Coffee")
	out, _ := insertTestTransfer(t, conn, checking.ID, savings.ID, -20000)
	target := fmt.Sprintf("/api/transactions/%d", out)

	res, raw := doJSON(t, h, "PATCH", target, map[string]any{
		"account_id": checking.ID, "category_id": nil, "date": "2026-10-05", "payee": " Rainy day ", "amount_cents": -12500, "note": "",
	})
	wantStatus(t, res, raw, http.StatusOK)
	if got := decode[store.Transaction](t, raw); got.AmountCents != -12500 || got.Payee != "Rainy day" ||
		got.TransferAccountID == nil || *got.TransferAccountID != savings.ID {
		t.Errorf("updated = %+v", got)
	}
	other := listTestTransactions(t, h, savings.ID)
	if len(other) != 1 || other[0].AmountCents != 12500 || other[0].Date != "2026-10-05" || other[0].Payee != "Rainy day" {
		t.Errorf("other leg = %+v", other)
	}

	valid := func(k string, v any) map[string]any {
		m := map[string]any{"account_id": checking.ID, "date": "2026-10-05", "amount_cents": -1}
		m[k] = v
		return m
	}
	wantError(t, h, "PATCH", target, valid("account_id", savings.ID), http.StatusBadRequest)
	wantError(t, h, "PATCH", target, valid("category_id", c.ID), http.StatusBadRequest)
}

func TestDeleteTransferLeg(t *testing.T) {
	h, conn := newTestServerWithDB(t)
	checking := createTestAccount(t, h, "Checking")
	savings := createTestAccount(t, h, "Savings")
	_, in := insertTestTransfer(t, conn, checking.ID, savings.ID, -20000)

	wantNoContent(t, h, "DELETE", fmt.Sprintf("/api/transactions/%d", in))
	if got := listTestTransactions(t, h, checking.ID); len(got) != 0 {
		t.Errorf("checking = %+v, want the other leg gone too", got)
	}
}
