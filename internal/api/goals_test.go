package api

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"budgeteer/internal/store"
)

func getTestGoals(t *testing.T, h http.Handler, query string) store.Goals {
	t.Helper()
	res, raw := doJSON(t, h, "GET", "/api/goals"+query, nil)
	wantStatus(t, res, raw, http.StatusOK)
	return decode[store.Goals](t, raw)
}

func TestGoals(t *testing.T) {
	h := newTestServer(t, builtDist)
	a := createTestAccount(t, h, "Checking")
	food := createTestCategory(t, h, "Food")
	createTestTransaction(t, h, map[string]any{"account_id": a.ID, "category_id": food.ID, "date": "2026-10-03", "amount_cents": -1234})
	createTestTransaction(t, h, map[string]any{"account_id": a.ID, "date": "2026-10-04", "amount_cents": -500})

	res, raw := doJSON(t, h, "PUT", fmt.Sprintf("/api/goals/%d?month=2026-10", food.ID), map[string]any{"amount_cents": 40000})
	wantStatus(t, res, raw, http.StatusOK)
	if g := decode[store.Goal](t, raw); g.CategoryID != food.ID || g.CapCents == nil || *g.CapCents != 40000 || g.SpentCents != 1234 {
		t.Errorf("set = %+v", g)
	}

	// Check the wire format the frontend relies on.
	res, raw = doJSON(t, h, "GET", "/api/goals?month=2026-11", nil)
	wantStatus(t, res, raw, http.StatusOK)
	body := decode[map[string]any](t, raw)
	goals, _ := body["goals"].([]any)
	if body["month"] != "2026-11" || len(goals) != 1 || body["uncategorized_cents"] != float64(0) {
		t.Fatalf("body = %v", body)
	}
	g := goals[0].(map[string]any)
	for _, key := range []string{"category_id", "name", "cap_cents", "cap_since", "rollover_since", "carryover_cents", "available_cents", "spent_cents"} {
		if _, ok := g[key]; !ok {
			t.Errorf("missing key %q in %v", key, g)
		}
	}
	if g["cap_cents"] != float64(40000) || g["cap_since"] != "2026-10" || g["rollover_since"] != nil {
		t.Errorf("Nov goal = %v", g)
	}

	if got := getTestGoals(t, h, "?month=2026-10"); got.UncategorizedCents != 500 {
		t.Errorf("uncategorized = %d, want 500", got.UncategorizedCents)
	}

	// Removing the cap.
	res, raw = doJSON(t, h, "PUT", fmt.Sprintf("/api/goals/%d?month=2026-12", food.ID), map[string]any{"amount_cents": nil})
	wantStatus(t, res, raw, http.StatusOK)
	if g := decode[store.Goal](t, raw); g.CapCents != nil {
		t.Errorf("removed = %+v", g)
	}
}

func TestGoalsMonthDefaultsToNow(t *testing.T) {
	h := newTestServer(t, builtDist)
	if got := getTestGoals(t, h, ""); got.Month != time.Now().Format("2006-01") {
		t.Errorf("month = %q, want the current month", got.Month)
	}
}

func TestGoalsErrors(t *testing.T) {
	h := newTestServer(t, builtDist)
	food := createTestCategory(t, h, "Food")
	put := fmt.Sprintf("/api/goals/%d", food.ID)
	// The seeded income categories aren't listed.
	if listed := getTestGoals(t, h, "?month=2026-10"); len(listed.Goals) != 1 {
		t.Fatalf("goals = %+v, want only Food", listed.Goals)
	}

	for _, q := range []string{"2026-13", "2026-1", "10/2026", "2026-10-01"} {
		wantError(t, h, "GET", "/api/goals?month="+q, nil, http.StatusBadRequest)
	}

	cases := []struct {
		name   string
		target string
		body   any
		want   int
		field  string
	}{
		{"missing month", put, map[string]any{"amount_cents": 100}, http.StatusBadRequest, "month"},
		{"bad month", put + "?month=2026-13", map[string]any{"amount_cents": 100}, http.StatusBadRequest, "month"},
		{"empty body", put + "?month=2026-10", map[string]any{}, http.StatusBadRequest, "amount_cents"},
		{"zero", put + "?month=2026-10", map[string]any{"amount_cents": 0}, http.StatusBadRequest, "amount_cents"},
		{"negative", put + "?month=2026-10", map[string]any{"amount_cents": -5}, http.StatusBadRequest, "amount_cents"},
		{"unknown field", put + "?month=2026-10", map[string]any{"amount_cents": 5, "x": 1}, http.StatusBadRequest, "x"},
		{"bad category id", "/api/goals/abc?month=2026-10", map[string]any{"amount_cents": 5}, http.StatusBadRequest, "category_id"},
		{"income category", "/api/goals/1?month=2026-10", map[string]any{"amount_cents": 5}, http.StatusBadRequest, "income"},
		{"unknown category", "/api/goals/999?month=2026-10", map[string]any{"amount_cents": 5}, http.StatusNotFound, "999"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res, raw := doJSON(t, h, "PUT", c.target, c.body)
			wantStatus(t, res, raw, c.want)
			if msg := decode[map[string]string](t, raw)["error"]; !strings.Contains(msg, c.field) {
				t.Errorf("error = %q, want it to mention %q", msg, c.field)
			}
		})
	}
}

func TestCategoryRollover(t *testing.T) {
	h := newTestServer(t, builtDist)
	ins := createTestCategory(t, h, "Insurance")
	target := fmt.Sprintf("/api/categories/%d", ins.ID)

	res, raw := doJSON(t, h, "PATCH", target, map[string]any{"rollover_since": "2026-10"})
	wantStatus(t, res, raw, http.StatusOK)
	if got := decode[store.Category](t, raw); got.RolloverSince == nil || *got.RolloverSince != "2026-10" {
		t.Errorf("updated = %+v", got)
	}

	// Omitting it leaves it alone.
	res, raw = doJSON(t, h, "PATCH", target, map[string]any{"name": "Car insurance"})
	wantStatus(t, res, raw, http.StatusOK)
	if got := decode[store.Category](t, raw); got.RolloverSince == nil {
		t.Errorf("rename turned rollover off: %+v", got)
	}

	// null turns it off.
	res, raw = doJSON(t, h, "PATCH", target, map[string]any{"rollover_since": nil})
	wantStatus(t, res, raw, http.StatusOK)
	if got := decode[store.Category](t, raw); got.RolloverSince != nil {
		t.Errorf("updated = %+v, want rollover off", got)
	}

	wantError(t, h, "PATCH", target, map[string]any{"rollover_since": "2026-13"}, http.StatusBadRequest)
	wantError(t, h, "PATCH", target, map[string]any{"rollover_since": 5}, http.StatusBadRequest)
	wantError(t, h, "PATCH", target, map[string]any{"is_income": true, "rollover_since": "2026-10"}, http.StatusBadRequest)
	wantError(t, h, "PATCH", "/api/categories/1", map[string]any{"rollover_since": "2026-10"}, http.StatusBadRequest)
}
