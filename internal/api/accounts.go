package api

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"budgeteer/internal/store"
)

// accountInput is the JSON body for POST /api/accounts.
type accountInput struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// validate normalizes the input in place and reports the first problem.
func (in *accountInput) validate() error {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return errors.New("name is required")
	}
	if !slices.Contains(store.AccountTypes, in.Type) {
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
	account, err := h.store.CreateAccount(r.Context(), store.AccountInput{Name: in.Name, Type: in.Type})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, account)
}
