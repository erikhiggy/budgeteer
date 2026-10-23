package api

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"budgeteer/internal/store"
)

type transactionInput struct {
	AccountID   int64  `json:"account_id"`
	CategoryID  *int64 `json:"category_id"`
	Date        string `json:"date"`
	Payee       string `json:"payee"`
	AmountCents int64  `json:"amount_cents"`
	Note        string `json:"note"`
}

// validate normalizes the input in place and reports the first problem.
func (in *transactionInput) validate() error {
	if in.Date == "" {
		return errors.New("date is required")
	}
	if _, err := time.Parse(time.DateOnly, in.Date); err != nil {
		return fmt.Errorf("date must be a date like 2026-10-31, got %q", in.Date)
	}
	if in.AmountCents == 0 {
		return errors.New("amount_cents must be non-zero")
	}
	in.Payee = strings.TrimSpace(in.Payee)
	in.Note = strings.TrimSpace(in.Note)
	return nil
}

func (in transactionInput) toStore() store.TransactionInput {
	return store.TransactionInput{
		AccountID:   in.AccountID,
		CategoryID:  in.CategoryID,
		Date:        in.Date,
		Payee:       in.Payee,
		AmountCents: in.AmountCents,
		Note:        in.Note,
	}
}

// parseTransactionFilter reads the query string for GET /api/transactions and
// reports the first problem, naming the parameter.
func parseTransactionFilter(q url.Values) (store.TransactionFilter, error) {
	var f store.TransactionFilter

	raw := q.Get("account_id")
	if raw == "" {
		return f, errors.New("account_id is required")
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return f, fmt.Errorf("account_id must be a positive integer, got %q", raw)
	}
	f.AccountID = id

	for _, p := range []struct {
		name string
		dst  *string
	}{{"from", &f.From}, {"to", &f.To}} {
		v := q.Get(p.name)
		if v == "" {
			continue
		}
		// time.Parse checks the format and rejects impossible dates like 2026-02-30.
		if _, err := time.Parse(time.DateOnly, v); err != nil {
			return f, fmt.Errorf("%s must be a date like 2026-10-31, got %q", p.name, v)
		}
		*p.dst = v
	}
	return f, nil
}

func (h *handlers) listTransactions(w http.ResponseWriter, r *http.Request) {
	f, err := parseTransactionFilter(r.URL.Query())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	transactions, err := h.store.ListTransactions(r.Context(), f)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, transactions)
}

func (h *handlers) createTransaction(w http.ResponseWriter, r *http.Request) {
	var in transactionInput
	if err := readJSON(w, r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := in.validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	transaction, err := h.store.CreateTransaction(r.Context(), in.toStore())
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, transaction)
}

// updateTransaction takes the same full body as createTransaction.
func (h *handlers) updateTransaction(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var in transactionInput
	if err := readJSON(w, r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := in.validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	transaction, err := h.store.UpdateTransaction(r.Context(), id, in.toStore())
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, transaction)
}

func (h *handlers) deleteTransaction(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.store.DeleteTransaction(r.Context(), id); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
