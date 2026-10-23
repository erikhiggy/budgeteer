package api

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"budgeteer/internal/db"
)

var builtDist = fstest.MapFS{
	"index.html":        {Data: []byte("<!doctype html><title>app</title>")},
	"assets/app-abc.js": {Data: []byte("console.log(1)")},
	"favicon.svg":       {Data: []byte("<svg/>")},
}

func newTestServer(t *testing.T, dist fstest.MapFS) http.Handler {
	t.Helper()
	conn, err := db.Open(context.Background(), filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return New(conn, dist)
}

func do(t *testing.T, h http.Handler, req *http.Request) (*http.Response, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	res := rec.Result()
	body, _ := io.ReadAll(res.Body)
	return res, string(body)
}

// httptest.NewRequest defaults Host to example.com, which guard rejects.
func localRequest(method, target string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, target, body)
	req.Host = "127.0.0.1:7420"
	return req
}

func TestHealth(t *testing.T) {
	h := newTestServer(t, builtDist)
	res, body := do(t, h, localRequest("GET", "/api/health", nil))
	if res.StatusCode != 200 {
		t.Fatalf("status = %d", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q", ct)
	}
	if strings.TrimSpace(body) != `{"status":"ok"}` {
		t.Errorf("body = %s", body)
	}
}

func TestUnknownAPIPathIsJSON404(t *testing.T) {
	h := newTestServer(t, builtDist)
	res, body := do(t, h, localRequest("GET", "/api/nope", nil))
	if res.StatusCode != 404 || !strings.Contains(body, `"error"`) {
		t.Errorf("status = %d, body = %s", res.StatusCode, body)
	}
}

func TestGuardHost(t *testing.T) {
	h := newTestServer(t, builtDist)
	for host, want := range map[string]int{
		"127.0.0.1:7420":  200,
		"localhost:5173":  200, // Vite dev proxy forwards the browser's Host
		"[::1]:7420":      200,
		"evil.example":    403,
		"evil.example:80": 403,
	} {
		req := localRequest("GET", "/api/health", nil)
		req.Host = host
		if res, _ := do(t, h, req); res.StatusCode != want {
			t.Errorf("Host %q: status = %d, want %d", host, res.StatusCode, want)
		}
	}
}

func TestGuardRequiresJSONForMutations(t *testing.T) {
	h := newTestServer(t, builtDist)

	req := localRequest("POST", "/api/nope", strings.NewReader("a=b"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if res, _ := do(t, h, req); res.StatusCode != http.StatusUnsupportedMediaType {
		t.Errorf("form POST: status = %d, want 415", res.StatusCode)
	}

	req = localRequest("DELETE", "/api/nope", nil)
	if res, _ := do(t, h, req); res.StatusCode != http.StatusUnsupportedMediaType {
		t.Errorf("bare DELETE: status = %d, want 415", res.StatusCode)
	}

	// Passes the guard and reaches the API's 404.
	req = localRequest("POST", "/api/nope", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	if res, _ := do(t, h, req); res.StatusCode != 404 {
		t.Errorf("JSON POST: status = %d, want 404", res.StatusCode)
	}
}

func TestStaticFilesAndFallback(t *testing.T) {
	h := newTestServer(t, builtDist)

	cases := []struct {
		path, wantBody, wantCache string
	}{
		{"/", "<!doctype html>", "no-cache"},
		{"/transactions/3", "<!doctype html>", "no-cache"},
		{"/assets/app-abc.js", "console.log(1)", "public, max-age=31536000, immutable"},
		{"/favicon.svg", "<svg/>", ""},
	}
	for _, c := range cases {
		res, body := do(t, h, localRequest("GET", c.path, nil))
		if res.StatusCode != 200 {
			t.Errorf("%s: status = %d", c.path, res.StatusCode)
			continue
		}
		if !strings.HasPrefix(body, c.wantBody) {
			t.Errorf("%s: body = %q", c.path, body)
		}
		if got := res.Header.Get("Cache-Control"); got != c.wantCache {
			t.Errorf("%s: Cache-Control = %q, want %q", c.path, got, c.wantCache)
		}
	}
}

func TestMissingAssetIs404(t *testing.T) {
	h := newTestServer(t, builtDist)
	if res, _ := do(t, h, localRequest("GET", "/assets/missing.js", nil)); res.StatusCode != 404 {
		t.Errorf("status = %d, want 404", res.StatusCode)
	}
}

func TestStaticRejectsMutations(t *testing.T) {
	h := newTestServer(t, builtDist)
	req := localRequest("POST", "/", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	if res, _ := do(t, h, req); res.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", res.StatusCode)
	}
}

func TestFrontendNotBuilt(t *testing.T) {
	h := newTestServer(t, fstest.MapFS{".gitkeep": {}})
	res, body := do(t, h, localRequest("GET", "/", nil))
	if res.StatusCode != 404 || !strings.Contains(body, "make build") {
		t.Errorf("status = %d, body = %q", res.StatusCode, body)
	}
}

func TestIDRoutesRejectBadIDs(t *testing.T) {
	h := newTestServer(t, builtDist)
	for _, entity := range []string{"accounts", "categories", "transactions"} {
		for _, id := range []string{"abc", "0", "-1", "1.5"} {
			t.Run(entity+"/"+id, func(t *testing.T) {
				wantError(t, h, "PATCH", "/api/"+entity+"/"+id, map[string]string{"name": "x"}, http.StatusBadRequest)
				wantError(t, h, "DELETE", "/api/"+entity+"/"+id, nil, http.StatusBadRequest)
			})
		}
	}
}

func TestDeleteNeedsJSONContentType(t *testing.T) {
	h := newTestServer(t, builtDist)
	res, raw := doJSON(t, h, "POST", "/api/accounts", map[string]string{"name": "Spare", "type": "cash"})
	wantStatus(t, res, raw, http.StatusCreated)
	res, body := do(t, h, localRequest("DELETE", "/api/accounts/1", nil))
	if res.StatusCode != http.StatusUnsupportedMediaType {
		t.Errorf("status = %d, want 415; body: %s", res.StatusCode, body)
	}
}
