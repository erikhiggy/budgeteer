// Package api serves the JSON API under /api and the built frontend at /.
package api

import (
	"database/sql"
	"fmt"
	"io/fs"
	"mime"
	"net"
	"net/http"
	"strconv"
	"strings"

	"budgeteer/internal/store"
)

// handlers holds what the API handlers need. Each entity's handlers live in
// their own file (accounts.go, …) as methods on this type.
type handlers struct {
	store *store.Store
}

// New returns the app's root handler. dist is the built frontend (the
// contents of web/dist); it may lack index.html when the frontend hasn't
// been built, e.g. during Vite-based development.
func New(db *sql.DB, dist fs.FS) http.Handler {
	h := &handlers{store: store.New(db)}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		if err := db.PingContext(r.Context()); err != nil {
			writeError(w, http.StatusServiceUnavailable, "database unavailable")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("GET /api/accounts", h.listAccounts)
	mux.HandleFunc("POST /api/accounts", h.createAccount)
	mux.HandleFunc("PATCH /api/accounts/{id}", h.updateAccount)
	mux.HandleFunc("DELETE /api/accounts/{id}", h.deleteAccount)
	mux.HandleFunc("GET /api/categories", h.listCategories)
	mux.HandleFunc("POST /api/categories", h.createCategory)
	mux.HandleFunc("PATCH /api/categories/{id}", h.updateCategory)
	mux.HandleFunc("DELETE /api/categories/{id}", h.deleteCategory)
	mux.HandleFunc("GET /api/transactions", h.listTransactions)
	mux.HandleFunc("POST /api/transactions", h.createTransaction)
	mux.HandleFunc("PATCH /api/transactions/{id}", h.updateTransaction)
	mux.HandleFunc("DELETE /api/transactions/{id}", h.deleteTransaction)
	mux.HandleFunc("GET /api/payees", h.listPayees)

	// TODO(M3): POST /api/transfers. The contract is in docs/api.md
	// ("Transfers (M3)"). Editing and deleting legs already work through the
	// transactions routes above.

	// Keep unknown API paths from falling through to the frontend.
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "not found")
	})

	mux.Handle("/", staticHandler(dist))

	return guard(mux)
}

// pathID reads the {id} path value, which must be a positive integer.
func pathID(r *http.Request) (int64, error) {
	raw := r.PathValue("id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid id %q", raw)
	}
	return id, nil
}

// guard rejects requests that could come from another website the user has
// open in the browser:
//   - Host must be a loopback name, which defeats DNS rebinding.
//   - Mutations must send Content-Type: application/json. Browsers won't send
//     that cross-origin without a CORS preflight, which we never approve.
func guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		switch host {
		case "localhost", "127.0.0.1", "::1":
		default:
			writeError(w, http.StatusForbidden, "forbidden host")
			return
		}

		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
		default:
			mt, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if mt != "application/json" {
				writeError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// staticHandler serves files from dist and falls back to index.html for any
// path that isn't a file, so client-side routes survive a reload.
func staticHandler(dist fs.FS) http.Handler {
	if _, err := fs.Stat(dist, "index.html"); err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Frontend not built. Run `make build`, or use the Vite dev server at http://localhost:5173.", http.StatusNotFound)
		})
	}

	files := http.FileServerFS(dist)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Registered as "/" rather than "GET /" so it can coexist with the
		// "/api/" catch-all, so the method check lives here.
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/")
		if name != "" && name != "index.html" {
			if info, err := fs.Stat(dist, name); err == nil && !info.IsDir() {
				// Vite content-hashes everything under assets/.
				if strings.HasPrefix(name, "assets/") {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				}
				files.ServeHTTP(w, r)
				return
			}
			// A missing asset is a stale page or a bad link, not a route.
			if strings.HasPrefix(name, "assets/") {
				http.NotFound(w, r)
				return
			}
		}
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFileFS(w, r, dist, "index.html")
	})
}
