package api

import (
	"fmt"
	"net/http"
	"time"
)

// maxReportMonths caps monthly-spending's range: ten years of bars.
const maxReportMonths = 120

func (h *handlers) spendingByCategory(w http.ResponseWriter, r *http.Request) {
	month := r.URL.Query().Get("month")
	if month == "" {
		month = time.Now().Format("2006-01")
	} else if err := parseMonth("month", month); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	report, err := h.store.SpendingByCategory(r.Context(), month)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (h *handlers) monthlySpending(w http.ResponseWriter, r *http.Request) {
	from, to := r.URL.Query().Get("from"), r.URL.Query().Get("to")
	for _, p := range []struct{ name, value string }{{"from", from}, {"to", to}} {
		if p.value == "" {
			writeError(w, http.StatusBadRequest, p.name+" is required")
			return
		}
		if err := parseMonth(p.name, p.value); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	if from > to {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("from (%s) is after to (%s)", from, to))
		return
	}
	if n := monthsBetween(from, to) + 1; n > maxReportMonths {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("that's %d months; the most is %d", n, maxReportMonths))
		return
	}
	report, err := h.store.MonthlySpending(r.Context(), from, to)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

// monthsBetween returns how many months "to" is after "from". Both must be
// valid "YYYY-MM".
func monthsBetween(from, to string) int {
	f, _ := time.Parse("2006-01", from)
	t, _ := time.Parse("2006-01", to)
	return (t.Year()-f.Year())*12 + int(t.Month()-f.Month())
}
