package api

import "net/http"

func (h *handlers) listPayees(w http.ResponseWriter, r *http.Request) {
	payees, err := h.store.ListPayees(r.Context())
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, payees)
}
