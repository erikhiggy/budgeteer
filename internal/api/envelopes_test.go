package api

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"budgeteer/internal/store"
)

func getTestEnvelopes(t *testing.T, h http.Handler, query string) store.Envelopes {
	t.Helper()
	res, raw := doJSON(t, h, "GET", "/api/envelopes"+query, nil)
	wantStatus(t, res, raw, http.StatusOK)
	return decode[store.Envelopes](t, raw)
}

func TestEnvelopes(t *testing.T) {
	h := newTestServer(t, builtDist)
	a := createTestAccount(t, h, "Checking")
	food := createTestCategory(t, h, "Food")
	createTestTransaction(t, h, map[string]any{"account_id": a.ID, "category_id": 1, "date": "2026-10-01", "amount_cents": 100000})
	createTestTransaction(t, h, map[string]any{"account_id": a.ID, "category_id": food.ID, "date": "2026-10-03", "amount_cents": -1234})
	createTestTransaction(t, h, map[string]any{"account_id": a.ID, "date": "2026-10-04", "amount_cents": -500})

	// Check the wire format the frontend relies on, before starting.
	res, raw := doJSON(t, h, "GET", "/api/envelopes?month=2026-10", nil)
	wantStatus(t, res, raw, http.StatusOK)
	body := decode[map[string]any](t, raw)
	for _, key := range []string{"month", "start_month", "ready_to_assign_cents", "income_cents", "assigned_cents", "envelopes", "uncategorized_cents"} {
		if _, ok := body[key]; !ok {
			t.Errorf("missing key %q in %v", key, body)
		}
	}
	if body["start_month"] != nil || body["uncategorized_cents"] != float64(500) {
		t.Errorf("not started = %v", body)
	}
	envs, _ := body["envelopes"].([]any)
	if len(envs) != 1 {
		t.Fatalf("envelopes = %v, want only Food", envs)
	}
	for _, key := range []string{"category_id", "name", "cap_cents", "carryover_cents", "assigned_cents", "spent_cents", "balance_cents"} {
		if _, ok := envs[0].(map[string]any)[key]; !ok {
			t.Errorf("missing key %q in %v", key, envs[0])
		}
	}

	res, raw = doJSON(t, h, "PUT", "/api/envelopes/start", map[string]any{"month": "2026-10"})
	wantStatus(t, res, raw, http.StatusOK)
	if got := decode[map[string]any](t, raw); got["start_month"] != "2026-10" {
		t.Errorf("start = %v", got)
	}

	res, raw = doJSON(t, h, "PUT", fmt.Sprintf("/api/envelopes/%d?month=2026-10", food.ID), map[string]any{"amount_cents": 40000})
	wantStatus(t, res, raw, http.StatusOK)
	got := decode[store.Envelopes](t, raw)
	if got.Month != "2026-10" || got.ReadyToAssignCents != 60000 || got.IncomeCents != 100000 || got.AssignedCents != 40000 {
		t.Errorf("after assigning = %+v", got)
	}
	if e := got.Envelopes[0]; e.AssignedCents != 40000 || e.SpentCents != 1234 || e.BalanceCents != 40000-1234 {
		t.Errorf("food = %+v", e)
	}

	// Fill from caps leaves Food's assignment alone.
	res, raw = doJSON(t, h, "PUT", fmt.Sprintf("/api/goals/%d?month=2026-10", food.ID), map[string]any{"amount_cents": 50000})
	wantStatus(t, res, raw, http.StatusOK)
	res, raw = doJSON(t, h, "POST", "/api/envelopes/fill-from-caps?month=2026-11", nil)
	wantStatus(t, res, raw, http.StatusOK)
	if got := decode[store.Envelopes](t, raw); got.Envelopes[0].AssignedCents != 50000 || got.ReadyToAssignCents != 10000 {
		t.Errorf("filled Nov = %+v", got)
	}

	// Stopping clears the start and the assignments.
	res, raw = doJSON(t, h, "PUT", "/api/envelopes/start", map[string]any{"month": nil})
	wantStatus(t, res, raw, http.StatusOK)
	if got := getTestEnvelopes(t, h, "?month=2026-11"); got.StartMonth != nil || got.AssignedCents != 0 {
		t.Errorf("after stopping = %+v", got)
	}
}

func TestEnvelopesMonthDefaultsToNow(t *testing.T) {
	h := newTestServer(t, builtDist)
	if got := getTestEnvelopes(t, h, ""); got.Month != time.Now().Format("2006-01") {
		t.Errorf("month = %q, want the current month", got.Month)
	}
}

func TestEnvelopesErrors(t *testing.T) {
	h := newTestServer(t, builtDist)
	food := createTestCategory(t, h, "Food")
	put := fmt.Sprintf("/api/envelopes/%d", food.ID)

	wantError(t, h, "GET", "/api/envelopes?month=2026-13", nil, http.StatusBadRequest)
	// Not started yet.
	wantError(t, h, "PUT", put+"?month=2026-10", map[string]any{"amount_cents": 100}, http.StatusConflict)
	wantError(t, h, "POST", "/api/envelopes/fill-from-caps?month=2026-10", nil, http.StatusConflict)

	wantError(t, h, "PUT", "/api/envelopes/start", map[string]any{}, http.StatusBadRequest)
	wantError(t, h, "PUT", "/api/envelopes/start", map[string]any{"month": "2026-1"}, http.StatusBadRequest)
	res, raw := doJSON(t, h, "PUT", "/api/envelopes/start", map[string]any{"month": "2026-10"})
	wantStatus(t, res, raw, http.StatusOK)

	wantError(t, h, "PUT", put, map[string]any{"amount_cents": 100}, http.StatusBadRequest)
	wantError(t, h, "PUT", put+"?month=2026-10", map[string]any{}, http.StatusBadRequest)
	wantError(t, h, "PUT", put+"?month=2026-10", map[string]any{"amount_cents": 1.5}, http.StatusBadRequest)
	wantError(t, h, "PUT", put+"?month=2026-09", map[string]any{"amount_cents": 100}, http.StatusBadRequest)
	wantError(t, h, "PUT", "/api/envelopes/1?month=2026-10", map[string]any{"amount_cents": 100}, http.StatusBadRequest) // income
	wantError(t, h, "PUT", "/api/envelopes/999?month=2026-10", map[string]any{"amount_cents": 100}, http.StatusNotFound)
	wantError(t, h, "PUT", "/api/envelopes/abc?month=2026-10", map[string]any{"amount_cents": 100}, http.StatusBadRequest)
	wantError(t, h, "POST", "/api/envelopes/fill-from-caps", nil, http.StatusBadRequest)
	wantError(t, h, "POST", "/api/envelopes/fill-from-caps?month=2026-09", nil, http.StatusBadRequest)
}

func TestCardEnvelopesAPI(t *testing.T) {
	h := newTestServer(t, builtDist)
	res, raw := doJSON(t, h, "POST", "/api/accounts", map[string]any{"name": "Visa", "type": "credit"})
	wantStatus(t, res, raw, http.StatusCreated)
	visa := decode[store.Account](t, raw)
	checking := createTestAccount(t, h, "Checking")
	put := fmt.Sprintf("/api/envelopes/cards/%d", visa.ID)

	wantError(t, h, "PUT", put+"?month=2026-11", map[string]any{"amount_cents": 100}, http.StatusConflict)
	res, raw = doJSON(t, h, "PUT", "/api/envelopes/start", map[string]any{"month": "2026-11"})
	wantStatus(t, res, raw, http.StatusOK)

	res, raw = doJSON(t, h, "PUT", put+"?month=2026-11", map[string]any{"amount_cents": 5000})
	wantStatus(t, res, raw, http.StatusOK)
	body := decode[map[string]any](t, raw)
	cards, _ := body["cards"].([]any)
	if len(cards) != 1 {
		t.Fatalf("cards = %v", body["cards"])
	}
	card := cards[0].(map[string]any)
	for _, key := range []string{"account_id", "name", "carryover_cents", "assigned_cents", "purchases_cents", "paid_cents", "balance_cents", "owed_cents"} {
		if _, ok := card[key]; !ok {
			t.Errorf("missing key %q in %v", key, card)
		}
	}
	if card["assigned_cents"] != float64(5000) || body["ready_to_assign_cents"] != float64(-5000) {
		t.Errorf("after assigning = %v", body)
	}

	wantError(t, h, "PUT", put+"?month=2026-11", map[string]any{}, http.StatusBadRequest)
	wantError(t, h, "PUT", put, map[string]any{"amount_cents": 100}, http.StatusBadRequest)
	wantError(t, h, "PUT", fmt.Sprintf("/api/envelopes/cards/%d?month=2026-11", checking.ID), map[string]any{"amount_cents": 100}, http.StatusBadRequest)
	wantError(t, h, "PUT", "/api/envelopes/cards/999?month=2026-11", map[string]any{"amount_cents": 100}, http.StatusNotFound)
	wantError(t, h, "PUT", "/api/envelopes/cards/x?month=2026-11", map[string]any{"amount_cents": 100}, http.StatusBadRequest)
}
