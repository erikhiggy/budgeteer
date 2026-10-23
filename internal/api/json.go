package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"budgeteer/internal/store"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// readJSON decodes a single JSON object from the request body into dst,
// rejecting unknown fields so typos in field names fail loudly.
func readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("invalid JSON body: %v", err)
	}
	if dec.More() {
		return errors.New("invalid JSON body: expected a single object")
	}
	return nil
}

// writeStoreError turns an error from package store into a response. Errors
// of type *store.Error carry a user-facing message; anything else is logged
// and reported as a 500 without details.
func writeStoreError(w http.ResponseWriter, err error) {
	var se *store.Error
	if !errors.As(err, &se) {
		log.Printf("internal error: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	status := http.StatusInternalServerError
	switch se.Kind {
	case store.NotFound:
		status = http.StatusNotFound
	case store.Conflict:
		status = http.StatusConflict
	case store.Invalid:
		status = http.StatusBadRequest
	}
	writeError(w, status, se.Msg)
}

// optional is a JSON field that records whether it was present at all, which
// a pointer alone can't: with optional[*string], an omitted field leaves Set
// false, while null sets it with a nil Value.
type optional[T any] struct {
	Set   bool
	Value T
}

func (o *optional[T]) UnmarshalJSON(data []byte) error {
	o.Set = true
	return json.Unmarshal(data, &o.Value)
}

// parseMonth checks that s is a real "YYYY-MM" month. name is the field or
// parameter, for the message.
func parseMonth(name, s string) error {
	if _, err := time.Parse("2006-01", s); err != nil || len(s) != len("2006-01") {
		return fmt.Errorf("%s must be a month like 2026-10, got %q", name, s)
	}
	return nil
}
