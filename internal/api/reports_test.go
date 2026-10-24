package api

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"budgeteer/internal/store"
)

func TestReports(t *testing.T) {
	h := newTestServer(t, builtDist)
	a := createTestAccount(t, h, "Checking")
	food := createTestCategory(t, h, "Food")
	createTestTransaction(t, h, map[string]any{"account_id": a.ID, "category_id": food.ID, "date": "2026-10-03", "amount_cents": -1234})
	createTestTransaction(t, h, map[string]any{"account_id": a.ID, "date": "2026-10-04", "amount_cents": -500})
	createTestTransaction(t, h, map[string]any{"account_id": a.ID, "category_id": food.ID, "date": "2026-08-30", "amount_cents": -100})

	// Check the wire format the frontend relies on.
	res, raw := doJSON(t, h, "GET", "/api/reports/spending-by-category?month=2026-10", nil)
	wantStatus(t, res, raw, http.StatusOK)
	body := decode[map[string]any](t, raw)
	cats, _ := body["categories"].([]any)
	if body["month"] != "2026-10" || body["total_cents"] != float64(1734) || len(cats) != 2 {
		t.Fatalf("body = %v", body)
	}
	first, last := cats[0].(map[string]any), cats[1].(map[string]any)
	if first["category_id"] != float64(food.ID) || first["name"] != "Food" || first["spent_cents"] != float64(1234) {
		t.Errorf("first = %v", first)
	}
	if v, ok := last["category_id"]; !ok || v != nil || last["name"] != "Uncategorized" || last["spent_cents"] != float64(500) {
		t.Errorf("uncategorized = %v", last)
	}

	res, raw = doJSON(t, h, "GET", "/api/reports/monthly-spending?from=2026-08&to=2026-10", nil)
	wantStatus(t, res, raw, http.StatusOK)
	trend := decode[store.MonthlySpending](t, raw)
	want := []store.MonthSpending{{Month: "2026-08", SpentCents: 100}, {Month: "2026-09"}, {Month: "2026-10", SpentCents: 1734}}
	if trend.From != "2026-08" || trend.To != "2026-10" || len(trend.Months) != 3 {
		t.Fatalf("trend = %+v", trend)
	}
	for i, w := range want {
		if trend.Months[i] != w {
			t.Errorf("months[%d] = %+v, want %+v", i, trend.Months[i], w)
		}
	}

	// The total matches the goals for the month.
	g := getTestGoals(t, h, "?month=2026-10")
	if total := g.Goals[0].SpentCents + g.UncategorizedCents; total != trend.Months[2].SpentCents {
		t.Errorf("goals total %d, trend %d", total, trend.Months[2].SpentCents)
	}

	// One month is fine, and so is the most.
	res, raw = doJSON(t, h, "GET", "/api/reports/monthly-spending?from=2026-10&to=2026-10", nil)
	wantStatus(t, res, raw, http.StatusOK)
	res, raw = doJSON(t, h, "GET", "/api/reports/monthly-spending?from=2017-01&to=2026-12", nil)
	wantStatus(t, res, raw, http.StatusOK)
	if got := decode[store.MonthlySpending](t, raw); len(got.Months) != 120 {
		t.Errorf("months = %d, want 120", len(got.Months))
	}
}

func TestSpendingByCategoryMonthDefaultsToNow(t *testing.T) {
	h := newTestServer(t, builtDist)
	res, raw := doJSON(t, h, "GET", "/api/reports/spending-by-category", nil)
	wantStatus(t, res, raw, http.StatusOK)
	if got := decode[store.SpendingByCategory](t, raw); got.Month != time.Now().Format("2006-01") || got.Categories == nil {
		t.Errorf("report = %+v, want the current month and []", got)
	}
}

func TestReportsErrors(t *testing.T) {
	h := newTestServer(t, builtDist)
	cases := []struct {
		name   string
		target string
		field  string
	}{
		{"bad month", "/api/reports/spending-by-category?month=2026-13", "month"},
		{"missing from", "/api/reports/monthly-spending?to=2026-10", "from"},
		{"missing to", "/api/reports/monthly-spending?from=2026-10", "to"},
		{"bad from", "/api/reports/monthly-spending?from=2026-1&to=2026-10", "from"},
		{"bad to", "/api/reports/monthly-spending?from=2026-01&to=2026-10-01", "to"},
		{"backwards", "/api/reports/monthly-spending?from=2026-11&to=2026-10", "after"},
		{"too long", "/api/reports/monthly-spending?from=2016-12&to=2026-12", "121 months"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res, raw := doJSON(t, h, "GET", c.target, nil)
			wantStatus(t, res, raw, http.StatusBadRequest)
			if msg := decode[map[string]string](t, raw)["error"]; !strings.Contains(msg, c.field) {
				t.Errorf("error = %q, want it to mention %q", msg, c.field)
			}
		})
	}
}
