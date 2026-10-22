package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

// doJSON sends body (encoded as JSON unless it's a string, which is sent
// verbatim) and returns the response with its raw body.
func doJSON(t *testing.T, h http.Handler, method, target string, body any) (*http.Response, []byte) {
	t.Helper()
	var r io.Reader
	switch b := body.(type) {
	case nil:
	case string:
		r = bytes.NewBufferString(b)
	default:
		buf, err := json.Marshal(b)
		if err != nil {
			t.Fatal(err)
		}
		r = bytes.NewReader(buf)
	}
	req := localRequest(method, target, r)
	if method != http.MethodGet {
		req.Header.Set("Content-Type", "application/json")
	}
	res, raw := do(t, h, req)
	return res, []byte(raw)
}

// decode unmarshals a response body, failing the test on bad JSON.
func decode[T any](t *testing.T, raw []byte) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("decode %s: %v", raw, err)
	}
	return v
}

// wantStatus fails the test if res doesn't have the expected status code.
func wantStatus(t *testing.T, res *http.Response, raw []byte, want int) {
	t.Helper()
	if res.StatusCode != want {
		t.Fatalf("status = %d, want %d; body: %s", res.StatusCode, want, raw)
	}
}
