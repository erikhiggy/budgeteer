package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"budgeteer/internal/store"
)

// transferInput is the JSON body for POST /api/transfers.
type transferInput struct {
	AccountID      int64  `json:"account_id"`
	OtherAccountID int64  `json:"other_account_id"`
	Date           string `json:"date"`
	Payee          string `json:"payee"`
	AmountCents    int64  `json:"amount_cents"`
	Note           string `json:"note"`
}

// validate normalizes the input in place and reports the first problem.
func (in *transferInput) validate() error {
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

func (in transferInput) toStore() store.TransferInput {
	return store.TransferInput{
		AccountID:      in.AccountID,
		OtherAccountID: in.OtherAccountID,
		Date:           in.Date,
		Payee:          in.Payee,
		AmountCents:    in.AmountCents,
		Note:           in.Note,
	}
}

func (h *handlers) createTransfer(w http.ResponseWriter, r *http.Request) {
	var in transferInput
	if err := readJSON(w, r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := in.validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	transfer, err := h.store.CreateTransfer(r.Context(), in.toStore())
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, transfer)
}
