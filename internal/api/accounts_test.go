package api

import (
	"fmt"
	"net/http"
	"testing"

	"budgeteer/internal/store"
)

func TestCreateAndListAccounts(t *testing.T) {
	h := newTestServer(t, builtDist)

	res, raw := doJSON(t, h, "POST", "/api/accounts", map[string]string{"name": "  Checking ", "type": "checking"})
	wantStatus(t, res, raw, http.StatusCreated)
	created := decode[store.Account](t, raw)
	if created.ID == 0 || created.Name != "Checking" || created.Type != "checking" || created.BalanceCents != 0 {
		t.Errorf("created = %+v", created)
	}

	res, raw = doJSON(t, h, "GET", "/api/accounts", nil)
	wantStatus(t, res, raw, http.StatusOK)
	list := decode[[]store.Account](t, raw)
	if len(list) != 1 || list[0] != created {
		t.Errorf("list = %+v, want [%+v]", list, created)
	}
}

func TestListAccountsEmptyIsArray(t *testing.T) {
	h := newTestServer(t, builtDist)
	res, raw := doJSON(t, h, "GET", "/api/accounts", nil)
	wantStatus(t, res, raw, http.StatusOK)
	if string(raw) != "[]\n" {
		t.Errorf("body = %q, want []", raw)
	}
}

func TestCreateAccountErrors(t *testing.T) {
	h := newTestServer(t, builtDist)
	res, raw := doJSON(t, h, "POST", "/api/accounts", map[string]string{"name": "Checking", "type": "checking"})
	wantStatus(t, res, raw, http.StatusCreated)

	cases := []struct {
		name string
		body any
		want int
	}{
		{"duplicate name, different case", map[string]string{"name": "checking", "type": "savings"}, http.StatusConflict},
		{"blank name", map[string]string{"name": "  ", "type": "cash"}, http.StatusBadRequest},
		{"unknown type", map[string]string{"name": "Wallet", "type": "wallet"}, http.StatusBadRequest},
		{"unknown field", map[string]string{"name": "Wallet", "type": "cash", "color": "red"}, http.StatusBadRequest},
		{"malformed JSON", `{"name":`, http.StatusBadRequest},
		{"two objects", `{"name":"A","type":"cash"} {}`, http.StatusBadRequest},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res, raw := doJSON(t, h, "POST", "/api/accounts", c.body)
			wantStatus(t, res, raw, c.want)
			if e := decode[map[string]string](t, raw); e["error"] == "" {
				t.Errorf("missing error message: %s", raw)
			}
		})
	}
}

func TestCreateAccountWithOpeningBalance(t *testing.T) {
	h := newTestServer(t, builtDist)
	res, raw := doJSON(t, h, "POST", "/api/accounts", map[string]any{"name": "Visa", "type": "credit", "opening_balance_cents": -50000})
	wantStatus(t, res, raw, http.StatusCreated)
	a := decode[store.Account](t, raw)
	if a.BalanceCents != -50000 {
		t.Errorf("created = %+v, want balance -50000", a)
	}

	res, raw = doJSON(t, h, "GET", fmt.Sprintf("/api/transactions?account_id=%d", a.ID), nil)
	wantStatus(t, res, raw, http.StatusOK)
	list := decode[[]store.Transaction](t, raw)
	if len(list) != 1 || list[0].Payee != "Opening balance" || list[0].AmountCents != -50000 || list[0].CategoryID == nil {
		t.Errorf("list = %+v", list)
	}
}

func TestUpdateAccount(t *testing.T) {
	h := newTestServer(t, builtDist)
	a := createTestAccount(t, h, "checking")
	createTestAccount(t, h, "Savings")
	target := fmt.Sprintf("/api/accounts/%d", a.ID)

	res, raw := doJSON(t, h, "PATCH", target, map[string]string{"name": "  Checking "})
	wantStatus(t, res, raw, http.StatusOK)
	if got := decode[store.Account](t, raw); got != (store.Account{ID: a.ID, Name: "Checking", Type: "checking"}) {
		t.Errorf("updated = %+v", got)
	}

	res, raw = doJSON(t, h, "PATCH", target, map[string]string{"type": "cash"})
	wantStatus(t, res, raw, http.StatusOK)
	if got := decode[store.Account](t, raw); got.Name != "Checking" || got.Type != "cash" {
		t.Errorf("updated = %+v", got)
	}

	cases := []struct {
		name string
		body any
		want int
	}{
		{"empty body", map[string]string{}, http.StatusBadRequest},
		{"blank name", map[string]string{"name": " "}, http.StatusBadRequest},
		{"bad type", map[string]string{"type": "wallet"}, http.StatusBadRequest},
		{"unknown field", map[string]string{"color": "red"}, http.StatusBadRequest},
		{"duplicate name", map[string]string{"name": "SAVINGS"}, http.StatusConflict},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			wantError(t, h, "PATCH", target, c.body, c.want)
		})
	}
	wantError(t, h, "PATCH", "/api/accounts/999", map[string]string{"name": "X"}, http.StatusNotFound)
}

func TestDeleteAccount(t *testing.T) {
	h := newTestServer(t, builtDist)
	used := createTestAccount(t, h, "Checking")
	createTestTransaction(t, h, map[string]any{"account_id": used.ID, "date": "2026-10-01", "amount_cents": -100})
	empty := createTestAccount(t, h, "Spare")

	wantError(t, h, "DELETE", fmt.Sprintf("/api/accounts/%d", used.ID), nil, http.StatusConflict)
	wantNoContent(t, h, "DELETE", fmt.Sprintf("/api/accounts/%d", empty.ID))
	wantError(t, h, "DELETE", fmt.Sprintf("/api/accounts/%d", empty.ID), nil, http.StatusNotFound)
}
