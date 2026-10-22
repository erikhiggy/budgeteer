package api

import (
	"net/http"
	"testing"

	"budgeteer/internal/store"
)

func TestCreateAndListAccounts(t *testing.T) {
	h := newTestServer(t, builtDist)

	res, raw := doJSON(t, h, "POST", "/api/accounts", map[string]string{"name": "  Checking ", "type": "checking"})
	wantStatus(t, res, raw, http.StatusCreated)
	created := decode[store.Account](t, raw)
	if created.ID == 0 || created.Name != "Checking" || created.Type != "checking" || created.BalanceCents != 0 {
		t.Errorf("created = %+v", created)
	}

	res, raw = doJSON(t, h, "GET", "/api/accounts", nil)
	wantStatus(t, res, raw, http.StatusOK)
	list := decode[[]store.Account](t, raw)
	if len(list) != 1 || list[0] != created {
		t.Errorf("list = %+v, want [%+v]", list, created)
	}
}

func TestListAccountsEmptyIsArray(t *testing.T) {
	h := newTestServer(t, builtDist)
	res, raw := doJSON(t, h, "GET", "/api/accounts", nil)
	wantStatus(t, res, raw, http.StatusOK)
	if string(raw) != "[]\n" {
		t.Errorf("body = %q, want []", raw)
	}
}

func TestCreateAccountErrors(t *testing.T) {
	h := newTestServer(t, builtDist)
	res, raw := doJSON(t, h, "POST", "/api/accounts", map[string]string{"name": "Checking", "type": "checking"})
	wantStatus(t, res, raw, http.StatusCreated)

	cases := []struct {
		name string
		body any
		want int
	}{
		{"duplicate name, different case", map[string]string{"name": "checking", "type": "savings"}, http.StatusConflict},
		{"blank name", map[string]string{"name": "  ", "type": "cash"}, http.StatusBadRequest},
		{"unknown type", map[string]string{"name": "Wallet", "type": "wallet"}, http.StatusBadRequest},
		{"unknown field", map[string]string{"name": "Wallet", "type": "cash", "color": "red"}, http.StatusBadRequest},
		{"malformed JSON", `{"name":`, http.StatusBadRequest},
		{"two objects", `{"name":"A","type":"cash"} {}`, http.StatusBadRequest},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res, raw := doJSON(t, h, "POST", "/api/accounts", c.body)
			wantStatus(t, res, raw, c.want)
			if e := decode[map[string]string](t, raw); e["error"] == "" {
				t.Errorf("missing error message: %s", raw)
			}
		})
	}
}
