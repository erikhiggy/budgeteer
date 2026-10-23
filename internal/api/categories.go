package api

import (
	"errors"
	"net/http"
	"strings"

	"budgeteer/internal/store"
)

// categoryInput is the JSON body for POST /api/categories.
type categoryInput struct {
	Name     string `json:"name"`
	IsIncome bool   `json:"is_income"`
}

// validate normalizes the input in place and reports the first problem.
func (in *categoryInput) validate() error {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return errors.New("name is required")
	}
	return nil
}

// categoryUpdate is the JSON body for PATCH /api/categories/{id}. A nil field
// wasn't sent and stays as it is.
type categoryUpdate struct {
	Name     *string `json:"name"`
	IsIncome *bool   `json:"is_income"`
	// A "YYYY-MM" month to start rolling over from, or null to turn it off.
	RolloverSince optional[*string] `json:"rollover_since"`
}

// validate normalizes the input in place and reports the first problem.
func (in *categoryUpdate) validate() error {
	if in.Name == nil && in.IsIncome == nil && !in.RolloverSince.Set {
		return errors.New("nothing to update")
	}
	if in.Name != nil {
		*in.Name = strings.TrimSpace(*in.Name)
		if *in.Name == "" {
			return errors.New("name is required")
		}
	}
	if since := in.RolloverSince.Value; since != nil {
		return parseMonth("rollover_since", *since)
	}
	return nil
}

func (h *handlers) listCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.store.ListCategories(r.Context())
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, categories)
}

func (h *handlers) createCategory(w http.ResponseWriter, r *http.Request) {
	var in categoryInput
	if err := readJSON(w, r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := in.validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	category, err := h.store.CreateCategory(r.Context(), store.CategoryInput{Name: in.Name, IsIncome: in.IsIncome})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, category)
}

func (h *handlers) updateCategory(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var in categoryUpdate
	if err := readJSON(w, r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := in.validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	category, err := h.store.UpdateCategory(r.Context(), id, store.CategoryUpdate{
		Name:          in.Name,
		IsIncome:      in.IsIncome,
		SetRollover:   in.RolloverSince.Set,
		RolloverSince: in.RolloverSince.Value,
	})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, category)
}

func (h *handlers) deleteCategory(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.store.DeleteCategory(r.Context(), id); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
