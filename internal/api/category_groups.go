package api

import (
	"errors"
	"net/http"
	"strings"

	"budgeteer/internal/store"
)

// categoryGroupInput is the JSON body for POST /api/category-groups.
type categoryGroupInput struct {
	Name string `json:"name"`
}

// validate normalizes the input in place and reports the first problem.
func (in *categoryGroupInput) validate() error {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return errors.New("name is required")
	}
	return nil
}

func (h *handlers) createCategoryGroup(w http.ResponseWriter, r *http.Request) {
	var in categoryGroupInput
	if err := readJSON(w, r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := in.validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	group, err := h.store.CreateCategoryGroup(r.Context(), store.CategoryGroupInput{Name: in.Name})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, group)
}

// TODO(groups): the handlers below follow createCategoryGroup above (and
// updateCategory in categories.go for the PATCH, with its optional fields).
// Until they're written, each answers 501.

func (h *handlers) listCategoryGroups(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "TODO(groups): not implemented")
}

func (h *handlers) updateCategoryGroup(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "TODO(groups): not implemented")
}

func (h *handlers) reorderCategoryGroups(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "TODO(groups): not implemented")
}

func (h *handlers) deleteCategoryGroup(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "TODO(groups): not implemented")
}
