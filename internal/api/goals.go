package api

import (
	"errors"
	"net/http"
	"strconv"
	"time"
)

// goalInput is the JSON body for PUT /api/goals/{category_id}.
type goalInput struct {
	// A positive cap, or null to remove the cap. Required, so {} can't
	// remove a cap by accident.
	AmountCents optional[*int64] `json:"amount_cents"`
}

func (in *goalInput) validate() error {
	if !in.AmountCents.Set {
		return errors.New("amount_cents is required; use null to remove the cap")
	}
	if c := in.AmountCents.Value; c != nil && *c <= 0 {
		return errors.New("amount_cents must be positive, or null to remove the cap")
	}
	return nil
}

func (h *handlers) listGoals(w http.ResponseWriter, r *http.Request) {
	month := r.URL.Query().Get("month")
	if month == "" {
		month = time.Now().Format("2006-01")
	} else if err := parseMonth("month", month); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	goals, err := h.store.ListGoals(r.Context(), month)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, goals)
}

func (h *handlers) setGoal(w http.ResponseWriter, r *http.Request) {
	raw := r.PathValue("category_id")
	categoryID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || categoryID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid category_id "+strconv.Quote(raw))
		return
	}
	month := r.URL.Query().Get("month")
	if month == "" {
		writeError(w, http.StatusBadRequest, "month is required")
		return
	}
	if err := parseMonth("month", month); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var in goalInput
	if err := readJSON(w, r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := in.validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	goal, err := h.store.SetGoal(r.Context(), categoryID, month, in.AmountCents.Value)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, goal)
}
