package api

import (
	"net/http"
	"testing"
)

func TestListPayees(t *testing.T) {
	h := newTestServer(t, builtDist)
	res, raw := doJSON(t, h, "GET", "/api/payees", nil)
	wantStatus(t, res, raw, http.StatusOK)
	if string(raw) != "[]\n" {
		t.Errorf("body = %q, want []", raw)
	}

	a := createTestAccount(t, h, "Checking")
	c := createTestCategory(t, h, "Coffee")
	createTestTransaction(t, h, map[string]any{"account_id": a.ID, "date": "2026-10-01", "payee": "Rent", "amount_cents": -100})
	createTestTransaction(t, h, map[string]any{"account_id": a.ID, "category_id": c.ID, "date": "2026-10-02", "payee": "cafe", "amount_cents": -100})

	res, raw = doJSON(t, h, "GET", "/api/payees", nil)
	wantStatus(t, res, raw, http.StatusOK)
	got := decode[[]map[string]any](t, raw)
	if len(got) != 2 || got[0]["payee"] != "cafe" || got[0]["category_id"] != float64(c.ID) ||
		got[1]["payee"] != "Rent" || got[1]["category_id"] != nil {
		t.Errorf("payees = %v", got)
	}
}
