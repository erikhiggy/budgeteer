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
