package api

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"budgeteer/internal/store"
)

// accountInput is the JSON body for POST /api/accounts.
type accountInput struct {
	Name                string `json:"name"`
	Type                string `json:"type"`
	OpeningBalanceCents int64  `json:"opening_balance_cents"`
}

// validate normalizes the input in place and reports the first problem.
func (in *accountInput) validate() error {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return errors.New("name is required")
	}
	return validateAccountType(in.Type)
}

// accountUpdate is the JSON body for PATCH /api/accounts/{id}. A nil field
// wasn't sent and stays as it is.
type accountUpdate struct {
	Name *string `json:"name"`
	Type *string `json:"type"`
}

// validate normalizes the input in place and reports the first problem.
func (in *accountUpdate) validate() error {
	if in.Name == nil && in.Type == nil {
		return errors.New("nothing to update")
	}
	if in.Name != nil {
		*in.Name = strings.TrimSpace(*in.Name)
		if *in.Name == "" {
			return errors.New("name is required")
		}
	}
	if in.Type != nil {
		return validateAccountType(*in.Type)
	}
	return nil
}

func validateAccountType(t string) error {
	if !slices.Contains(store.AccountTypes, t) {
		return fmt.Errorf("type must be one of %s", strings.Join(store.AccountTypes, ", "))
	}
	return nil
}

func (h *handlers) listAccounts(w http.ResponseWriter, r *http.Request) {
	accounts, err := h.store.ListAccounts(r.Context())
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, accounts)
}

func (h *handlers) createAccount(w http.ResponseWriter, r *http.Request) {
	var in accountInput
	if err := readJSON(w, r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := in.validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	account, err := h.store.CreateAccount(r.Context(), store.AccountInput{
		Name:                in.Name,
		Type:                in.Type,
		OpeningBalanceCents: in.OpeningBalanceCents,
		OpeningBalanceDate:  time.Now().Format(time.DateOnly),
	})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, account)
}

func (h *handlers) updateAccount(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var in accountUpdate
	if err := readJSON(w, r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := in.validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	account, err := h.store.UpdateAccount(r.Context(), id, store.AccountUpdate{Name: in.Name, Type: in.Type})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, account)
}

func (h *handlers) deleteAccount(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.store.DeleteAccount(r.Context(), id); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
