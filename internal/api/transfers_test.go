package api

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"budgeteer/internal/store"
)

// createTestTransferLegs moves cents into account a from account b (negative
// cents is money out of a into b) and returns the legs' ids.
func createTestTransferLegs(t *testing.T, h http.Handler, a, b int64, cents int64) (legA, legB int64) {
	t.Helper()
	res, raw := doJSON(t, h, "POST", "/api/transfers", map[string]any{
		"account_id": a, "other_account_id": b, "date": "2026-10-01", "payee": "Savings plan", "amount_cents": cents,
	})
	wantStatus(t, res, raw, http.StatusCreated)
	tr := decode[store.Transfer](t, raw)
	return tr.Legs[0].ID, tr.Legs[1].ID
}

func listTestTransactions(t *testing.T, h http.Handler, accountID int64) []store.Transaction {
	t.Helper()
	res, raw := doJSON(t, h, "GET", fmt.Sprintf("/api/transactions?account_id=%d", accountID), nil)
	wantStatus(t, res, raw, http.StatusOK)
	return decode[[]store.Transaction](t, raw)
}

func TestTransferLegWireFormat(t *testing.T) {
	h := newTestServer(t, builtDist)
	checking := createTestAccount(t, h, "Checking")
	savings := createTestAccount(t, h, "Savings")
	createTestTransferLegs(t, h, checking.ID, savings.ID, -20000)
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
	h := newTestServer(t, builtDist)
	checking := createTestAccount(t, h, "Checking")
	savings := createTestAccount(t, h, "Savings")
	c := createTestCategory(t, h, "Coffee")
	out, _ := createTestTransferLegs(t, h, checking.ID, savings.ID, -20000)
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
	h := newTestServer(t, builtDist)
	checking := createTestAccount(t, h, "Checking")
	savings := createTestAccount(t, h, "Savings")
	_, in := createTestTransferLegs(t, h, checking.ID, savings.ID, -20000)

	wantNoContent(t, h, "DELETE", fmt.Sprintf("/api/transactions/%d", in))
	if got := listTestTransactions(t, h, checking.ID); len(got) != 0 {
		t.Errorf("checking = %+v, want the other leg gone too", got)
	}
}

func TestCreateTransfer(t *testing.T) {
	h := newTestServer(t, builtDist)
	checking := createTestAccount(t, h, "Checking")
	savings := createTestAccount(t, h, "Savings")

	res, raw := doJSON(t, h, "POST", "/api/transfers", map[string]any{
		"account_id": checking.ID, "other_account_id": savings.ID, "date": "2026-10-23",
		"payee": "  Rainy day ", "amount_cents": -20000, "note": " monthly ",
	})
	wantStatus(t, res, raw, http.StatusCreated)

	// Check the wire format the frontend relies on.
	body := decode[map[string]any](t, raw)
	legs, ok := body["legs"].([]any)
	if body["id"] == nil || !ok || len(legs) != 2 {
		t.Fatalf("body = %v", body)
	}
	tr := decode[store.Transfer](t, raw)
	out, in := tr.Legs[0], tr.Legs[1]
	if out.AccountID != checking.ID || out.AmountCents != -20000 || in.AccountID != savings.ID || in.AmountCents != 20000 {
		t.Errorf("legs = %+v", tr.Legs)
	}
	if out.Payee != "Rainy day" || out.Note != "monthly" || out.CategoryID != nil || *out.TransferAccountID != savings.ID {
		t.Errorf("first leg = %+v", out)
	}

	// Both balances move; the payee isn't offered for autofill.
	res, raw = doJSON(t, h, "GET", "/api/accounts", nil)
	wantStatus(t, res, raw, http.StatusOK)
	for _, a := range decode[[]store.Account](t, raw) {
		want := map[int64]int64{checking.ID: -20000, savings.ID: 20000}[a.ID]
		if a.BalanceCents != want {
			t.Errorf("%s balance = %d, want %d", a.Name, a.BalanceCents, want)
		}
	}
	res, raw = doJSON(t, h, "GET", "/api/payees", nil)
	wantStatus(t, res, raw, http.StatusOK)
	if string(raw) != "[]\n" {
		t.Errorf("payees = %s, want []", raw)
	}
}

func TestCreateTransferErrors(t *testing.T) {
	h := newTestServer(t, builtDist)
	checking := createTestAccount(t, h, "Checking")
	savings := createTestAccount(t, h, "Savings")

	// with returns a valid body with field k set to v; a nil v removes it.
	with := func(k string, v any) map[string]any {
		m := map[string]any{"account_id": checking.ID, "other_account_id": savings.ID, "date": "2026-10-23", "amount_cents": -100}
		if v == nil {
			delete(m, k)
		} else {
			m[k] = v
		}
		return m
	}
	cases := []struct {
		name      string
		body      any
		wantField string // the field the message must name
	}{
		{"missing date", with("date", nil), "date"},
		{"impossible date", with("date", "2026-02-30"), "date"},
		{"zero amount", with("amount_cents", 0), "amount_cents"},
		{"same account", with("other_account_id", checking.ID), "other_account_id"},
		{"unknown account", with("account_id", 999), "account_id"},
		{"unknown other account", with("other_account_id", 999), "other_account_id"},
		{"missing other account", with("other_account_id", nil), "other_account_id"},
		{"category given", with("category_id", 1), "category_id"},
		{"malformed JSON", `{"date":`, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res, raw := doJSON(t, h, "POST", "/api/transfers", c.body)
			wantStatus(t, res, raw, http.StatusBadRequest)
			msg := decode[map[string]string](t, raw)["error"]
			if msg == "" || !strings.Contains(msg, c.wantField) {
				t.Errorf("error = %q, want a message naming %q", msg, c.wantField)
			}
		})
	}

	// None of the rejected requests created anything.
	for _, a := range []store.Account{checking, savings} {
		if got := listTestTransactions(t, h, a.ID); len(got) != 0 {
			t.Errorf("%s has %+v, want nothing", a.Name, got)
		}
	}
}
