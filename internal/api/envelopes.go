package api

import (
	"errors"
	"net/http"
	"strconv"
	"time"
)

// envelopesStartInput is the JSON body for PUT /api/envelopes/start.
type envelopesStartInput struct {
	// The month envelopes count from, or null to stop. Required, so {}
	// can't stop envelopes (and delete every assignment) by accident.
	Month optional[*string] `json:"month"`
}

func (in *envelopesStartInput) validate() error {
	if !in.Month.Set {
		return errors.New("month is required; use null to stop envelope budgeting")
	}
	if m := in.Month.Value; m != nil {
		return parseMonth("month", *m)
	}
	return nil
}

// assignmentInput is the JSON body for PUT /api/envelopes/{category_id} and
// PUT /api/envelopes/cards/{account_id}.
type assignmentInput struct {
	// The month's assignment; 0 removes it. Required, so {} can't clear one.
	AmountCents *int64 `json:"amount_cents"`
}

func (h *handlers) listEnvelopes(w http.ResponseWriter, r *http.Request) {
	month := r.URL.Query().Get("month")
	if month == "" {
		month = time.Now().Format("2006-01")
	} else if err := parseMonth("month", month); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	envs, err := h.store.ListEnvelopes(r.Context(), month)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envs)
}

func (h *handlers) setEnvelopesStart(w http.ResponseWriter, r *http.Request) {
	var in envelopesStartInput
	if err := readJSON(w, r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := in.validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.store.SetEnvelopesStart(r.Context(), in.Month.Value); err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]*string{"start_month": in.Month.Value})
}

func (h *handlers) setAssignment(w http.ResponseWriter, r *http.Request) {
	raw := r.PathValue("category_id")
	categoryID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || categoryID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid category_id "+strconv.Quote(raw))
		return
	}
	month, ok := requiredMonth(w, r)
	if !ok {
		return
	}
	var in assignmentInput
	if err := readJSON(w, r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if in.AmountCents == nil {
		writeError(w, http.StatusBadRequest, "amount_cents is required; use 0 to remove the assignment")
		return
	}
	envs, err := h.store.SetAssignment(r.Context(), categoryID, month, *in.AmountCents)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envs)
}

func (h *handlers) setCardAssignment(w http.ResponseWriter, r *http.Request) {
	raw := r.PathValue("account_id")
	accountID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || accountID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid account_id "+strconv.Quote(raw))
		return
	}
	month, ok := requiredMonth(w, r)
	if !ok {
		return
	}
	var in assignmentInput
	if err := readJSON(w, r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if in.AmountCents == nil {
		writeError(w, http.StatusBadRequest, "amount_cents is required; use 0 to remove the assignment")
		return
	}
	envs, err := h.store.SetCardAssignment(r.Context(), accountID, month, *in.AmountCents)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envs)
}

func (h *handlers) fillFromCaps(w http.ResponseWriter, r *http.Request) {
	month, ok := requiredMonth(w, r)
	if !ok {
		return
	}
	envs, err := h.store.FillFromCaps(r.Context(), month)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envs)
}

// requiredMonth reads the ?month= parameter, writing a 400 and returning
// false if it's missing or invalid.
func requiredMonth(w http.ResponseWriter, r *http.Request) (string, bool) {
	month := r.URL.Query().Get("month")
	if month == "" {
		writeError(w, http.StatusBadRequest, "month is required")
		return "", false
	}
	if err := parseMonth("month", month); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return "", false
	}
	return month, true
}
