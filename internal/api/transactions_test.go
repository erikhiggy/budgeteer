package api

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"budgeteer/internal/store"
)

func createTestAccount(t *testing.T, h http.Handler, name string) store.Account {
	t.Helper()
	res, raw := doJSON(t, h, "POST", "/api/accounts", map[string]string{"name": name, "type": "checking"})
	wantStatus(t, res, raw, http.StatusCreated)
	return decode[store.Account](t, raw)
}

func createTestTransaction(t *testing.T, h http.Handler, body map[string]any) store.Transaction {
	t.Helper()
	res, raw := doJSON(t, h, "POST", "/api/transactions", body)
	wantStatus(t, res, raw, http.StatusCreated)
	return decode[store.Transaction](t, raw)
}

func TestCreateTransaction(t *testing.T) {
	h := newTestServer(t, builtDist)
	a := createTestAccount(t, h, "Checking")
	res, raw := doJSON(t, h, "POST", "/api/categories", map[string]string{"name": "Coffee"})
	wantStatus(t, res, raw, http.StatusCreated)
	coffee := decode[store.Category](t, raw)

	created := createTestTransaction(t, h, map[string]any{
		"account_id": a.ID, "category_id": coffee.ID, "date": "2026-10-05",
		"payee": "  Cafe ", "amount_cents": -450, "note": " latte  ",
	})
	if created.ID == 0 || created.AccountID != a.ID || created.CategoryID == nil || *created.CategoryID != coffee.ID ||
		created.Date != "2026-10-05" || created.Payee != "Cafe" || created.AmountCents != -450 || created.Note != "latte" {
		t.Errorf("created = %+v", created)
	}

	// payee, note and category_id are optional; null and omitted both mean uncategorized.
	for _, body := range []map[string]any{
		{"account_id": a.ID, "date": "2026-10-06", "amount_cents": 100000},
		{"account_id": a.ID, "category_id": nil, "date": "2026-10-06", "amount_cents": 100000},
	} {
		tx := createTestTransaction(t, h, body)
		if tx.CategoryID != nil || tx.Payee != "" || tx.Note != "" {
			t.Errorf("created = %+v, want uncategorized with empty payee and note", tx)
		}
	}

	// The new transactions show up in the list and the account balance.
	res, raw = doJSON(t, h, "GET", fmt.Sprintf("/api/transactions?account_id=%d", a.ID), nil)
	wantStatus(t, res, raw, http.StatusOK)
	if list := decode[[]store.Transaction](t, raw); len(list) != 3 || list[2].ID != created.ID {
		t.Errorf("list = %+v", list)
	}
	res, raw = doJSON(t, h, "GET", "/api/accounts", nil)
	wantStatus(t, res, raw, http.StatusOK)
	if accounts := decode[[]store.Account](t, raw); accounts[0].BalanceCents != 199550 {
		t.Errorf("balance = %d, want 199550", accounts[0].BalanceCents)
	}
}

func TestCreateTransactionErrors(t *testing.T) {
	h := newTestServer(t, builtDist)
	a := createTestAccount(t, h, "Checking")

	// valid returns a good body with the given fields overridden; a nil value
	// removes the field.
	valid := func(overrides map[string]any) map[string]any {
		body := map[string]any{"account_id": a.ID, "date": "2026-10-05", "amount_cents": -100}
		for k, v := range overrides {
			if v == nil {
				delete(body, k)
			} else {
				body[k] = v
			}
		}
		return body
	}

	cases := []struct {
		name      string
		body      any
		wantField string // the field the message must name
	}{
		{"missing date", valid(map[string]any{"date": nil}), "date"},
		{"impossible date", valid(map[string]any{"date": "2026-02-30"}), "date"},
		{"badly formatted date", valid(map[string]any{"date": "10/05/2026"}), "date"},
		{"missing amount", valid(map[string]any{"amount_cents": nil}), "amount_cents"},
		{"zero amount", valid(map[string]any{"amount_cents": 0}), "amount_cents"},
		{"missing account", valid(map[string]any{"account_id": nil}), "account_id"},
		{"unknown account", valid(map[string]any{"account_id": 999}), "account_id"},
		{"unknown category", valid(map[string]any{"category_id": 42}), "category_id"},
		{"fractional amount", valid(map[string]any{"amount_cents": 1.5}), "amount_cents"},
		{"unknown field", valid(map[string]any{"memo": "hi"}), "memo"},
		{"malformed JSON", `{"date":`, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res, raw := doJSON(t, h, "POST", "/api/transactions", c.body)
			wantStatus(t, res, raw, http.StatusBadRequest)
			msg := decode[map[string]string](t, raw)["error"]
			if msg == "" || !strings.Contains(msg, c.wantField) {
				t.Errorf("error = %q, want a message naming %q", msg, c.wantField)
			}
		})
	}

	// None of the rejected requests created anything.
	res, raw := doJSON(t, h, "GET", fmt.Sprintf("/api/transactions?account_id=%d", a.ID), nil)
	wantStatus(t, res, raw, http.StatusOK)
	if string(raw) != "[]\n" {
		t.Errorf("body = %s, want []", raw)
	}
}

func TestListTransactions(t *testing.T) {
	h := newTestServer(t, builtDist)
	a := createTestAccount(t, h, "Checking")
	other := createTestAccount(t, h, "Savings")
	for _, row := range []struct {
		accountID int64
		date      string
		cents     int64
	}{{a.ID, "2026-09-30", -450}, {a.ID, "2026-10-05", -1000}, {a.ID, "2026-10-31", 250000}, {other.ID, "2026-10-15", 500}} {
		createTestTransaction(t, h, map[string]any{"account_id": row.accountID, "date": row.date, "payee": "Payee", "amount_cents": row.cents})
	}

	res, raw := doJSON(t, h, "GET", fmt.Sprintf("/api/transactions?account_id=%d&from=2026-10-01&to=2026-10-31", a.ID), nil)
	wantStatus(t, res, raw, http.StatusOK)
	list := decode[[]map[string]any](t, raw)
	if len(list) != 2 || list[0]["date"] != "2026-10-31" || list[1]["date"] != "2026-10-05" {
		t.Fatalf("list = %v", list)
	}
	// Check the wire format the frontend relies on, including a null category_id.
	first := list[0]
	for _, key := range []string{"id", "account_id", "category_id", "date", "payee", "amount_cents", "note"} {
		if _, ok := first[key]; !ok {
			t.Errorf("missing key %q in %v", key, first)
		}
	}
	if first["category_id"] != nil || first["amount_cents"] != float64(250000) || first["note"] != "" {
		t.Errorf("first = %v", first)
	}
}

func TestListTransactionsEmptyIsArray(t *testing.T) {
	h := newTestServer(t, builtDist)
	a := createTestAccount(t, h, "Checking")
	res, raw := doJSON(t, h, "GET", fmt.Sprintf("/api/transactions?account_id=%d", a.ID), nil)
	wantStatus(t, res, raw, http.StatusOK)
	if string(raw) != "[]\n" {
		t.Errorf("body = %q, want []", raw)
	}
}

func TestListTransactionsErrors(t *testing.T) {
	h := newTestServer(t, builtDist)
	a := createTestAccount(t, h, "Checking")

	cases := []struct {
		name  string
		query string
		want  int
	}{
		{"missing account_id", "", http.StatusBadRequest},
		{"non-numeric account_id", "?account_id=abc", http.StatusBadRequest},
		{"zero account_id", "?account_id=0", http.StatusBadRequest},
		{"negative account_id", "?account_id=-1", http.StatusBadRequest},
		{"bad from format", fmt.Sprintf("?account_id=%d&from=10/01/2026", a.ID), http.StatusBadRequest},
		{"impossible to date", fmt.Sprintf("?account_id=%d&to=2026-02-30", a.ID), http.StatusBadRequest},
		{"unknown account", "?account_id=999", http.StatusNotFound},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res, raw := doJSON(t, h, "GET", "/api/transactions"+c.query, nil)
			wantStatus(t, res, raw, c.want)
			if e := decode[map[string]string](t, raw); e["error"] == "" {
				t.Errorf("missing error message: %s", raw)
			}
		})
	}
}

func TestUpdateTransaction(t *testing.T) {
	h := newTestServer(t, builtDist)
	a := createTestAccount(t, h, "Checking")
	b := createTestAccount(t, h, "Savings")
	c := createTestCategory(t, h, "Coffee")
	tx := createTestTransaction(t, h, map[string]any{"account_id": a.ID, "category_id": c.ID, "date": "2026-10-01", "payee": "Cafe", "amount_cents": -450})
	target := fmt.Sprintf("/api/transactions/%d", tx.ID)

	res, raw := doJSON(t, h, "PATCH", target, map[string]any{
		"account_id": b.ID, "category_id": nil, "date": "2026-10-02", "payee": " Bakery ", "amount_cents": -600, "note": "bread",
	})
	wantStatus(t, res, raw, http.StatusOK)
	got := decode[store.Transaction](t, raw)
	if got.ID != tx.ID || got.AccountID != b.ID || got.CategoryID != nil || got.Date != "2026-10-02" ||
		got.Payee != "Bakery" || got.AmountCents != -600 || got.Note != "bread" {
		t.Errorf("updated = %+v", got)
	}

	// Both balances follow the move.
	res, raw = doJSON(t, h, "GET", "/api/accounts", nil)
	wantStatus(t, res, raw, http.StatusOK)
	for _, acct := range decode[[]store.Account](t, raw) {
		want := map[int64]int64{a.ID: 0, b.ID: -600}[acct.ID]
		if acct.BalanceCents != want {
			t.Errorf("%s balance = %d, want %d", acct.Name, acct.BalanceCents, want)
		}
	}

	// with returns a valid body with field k set to v.
	with := func(k string, v any) map[string]any {
		m := map[string]any{"account_id": a.ID, "date": "2026-10-02", "amount_cents": -1}
		m[k] = v
		return m
	}
	cases := []struct {
		name string
		body any
		want int
	}{
		{"impossible date", with("date", "2026-02-30"), http.StatusBadRequest},
		{"zero amount", with("amount_cents", 0), http.StatusBadRequest},
		{"unknown account", with("account_id", 999), http.StatusBadRequest},
		{"unknown category", with("category_id", 42), http.StatusBadRequest},
		{"unknown field", with("memo", "x"), http.StatusBadRequest},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			wantError(t, h, "PATCH", target, c.body, c.want)
		})
	}
	wantError(t, h, "PATCH", "/api/transactions/999", with("note", ""), http.StatusNotFound)
}

func TestDeleteTransaction(t *testing.T) {
	h := newTestServer(t, builtDist)
	a := createTestAccount(t, h, "Checking")
	tx := createTestTransaction(t, h, map[string]any{"account_id": a.ID, "date": "2026-10-01", "amount_cents": -100})
	target := fmt.Sprintf("/api/transactions/%d", tx.ID)

	wantNoContent(t, h, "DELETE", target)
	wantError(t, h, "DELETE", target, nil, http.StatusNotFound)
}
