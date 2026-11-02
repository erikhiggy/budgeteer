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

// categoryGroupUpdate is the JSON body for PATCH /api/category-groups/{id}. A
// nil field wasn't sent and stays as it is.
type categoryGroupUpdate struct {
	Name      *string `json:"name"`
	SortOrder *int64  `json:"sort_order"`
}

// validate normalizes the input in place and reports the first problem.
func (in *categoryGroupUpdate) validate() error {
	if in.Name == nil && in.SortOrder == nil {
		return errors.New("nothing to update")
	}
	if in.Name != nil {
		*in.Name = strings.TrimSpace(*in.Name)
		if *in.Name == "" {
			return errors.New("name is required")
		}
	}
	return nil
}

func (h *handlers) listCategoryGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := h.store.ListCategoryGroups(r.Context())
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, groups)
}

func (h *handlers) updateCategoryGroup(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var in categoryGroupUpdate
	if err := readJSON(w, r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := in.validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	group, err := h.store.UpdateCategoryGroup(r.Context(), id, store.CategoryGroupUpdate{Name: in.Name, SortOrder: in.SortOrder})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, group)
}

// categoryGroupOrder is the JSON body for PUT /api/category-groups/order.
type categoryGroupOrder struct {
	IDs []int64 `json:"ids"`
}

func (h *handlers) reorderCategoryGroups(w http.ResponseWriter, r *http.Request) {
	var in categoryGroupOrder
	if err := readJSON(w, r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if in.IDs == nil {
		writeError(w, http.StatusBadRequest, "ids is required")
		return
	}
	groups, err := h.store.ReorderCategoryGroups(r.Context(), in.IDs)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, groups)
}

func (h *handlers) deleteCategoryGroup(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.store.DeleteCategoryGroup(r.Context(), id); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
