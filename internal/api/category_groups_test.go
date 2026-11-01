package api

import (
	"fmt"
	"net/http"
	"testing"

	"budgeteer/internal/store"
)

func createTestGroup(t *testing.T, h http.Handler, name string) store.CategoryGroup {
	t.Helper()
	res, raw := doJSON(t, h, "POST", "/api/category-groups", map[string]string{"name": name})
	wantStatus(t, res, raw, http.StatusCreated)
	return decode[store.CategoryGroup](t, raw)
}

func listTestGroups(t *testing.T, h http.Handler) []store.CategoryGroup {
	t.Helper()
	res, raw := doJSON(t, h, "GET", "/api/category-groups", nil)
	wantStatus(t, res, raw, http.StatusOK)
	return decode[[]store.CategoryGroup](t, raw)
}

func wantGroupNames(t *testing.T, got []store.CategoryGroup, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("groups = %+v, want names %v", got, want)
	}
	for i, name := range want {
		if got[i].Name != name {
			t.Fatalf("groups = %+v, want names %v", got, want)
		}
	}
}

func TestCreateAndListCategoryGroups(t *testing.T) {
	h := newTestServer(t, builtDist)

	// A fresh database has none, and says so with [] rather than null.
	res, raw := doJSON(t, h, "GET", "/api/category-groups", nil)
	wantStatus(t, res, raw, http.StatusOK)
	if string(raw) != "[]\n" {
		t.Errorf("fresh list = %q, want []", raw)
	}

	res, raw = doJSON(t, h, "POST", "/api/category-groups", map[string]string{"name": "  Housing "})
	wantStatus(t, res, raw, http.StatusCreated)
	housing := decode[store.CategoryGroup](t, raw)
	if housing.ID == 0 || housing.Name != "Housing" || housing.SortOrder != 0 {
		t.Errorf("created = %+v", housing)
	}
	wealth := createTestGroup(t, h, "Wealth building")
	if wealth.SortOrder != 1 {
		t.Errorf("second = %+v, want sort_order 1", wealth)
	}

	list := listTestGroups(t, h)
	wantGroupNames(t, list, "Housing", "Wealth building")
	if list[0] != housing || list[1] != wealth {
		t.Errorf("list = %+v, want %+v and %+v", list, housing, wealth)
	}
}

func TestCreateCategoryGroupErrors(t *testing.T) {
	h := newTestServer(t, builtDist)
	createTestGroup(t, h, "Housing")

	cases := []struct {
		name string
		body any
		want int
	}{
		{"duplicate name, different case", map[string]string{"name": "HOUSING"}, http.StatusConflict},
		{"duplicate after trimming", map[string]string{"name": " Housing "}, http.StatusConflict},
		{"blank name", map[string]string{"name": "  "}, http.StatusBadRequest},
		{"missing name", map[string]any{}, http.StatusBadRequest},
		{"unknown field", map[string]string{"name": "Fun", "color": "red"}, http.StatusBadRequest},
		{"wrong type for name", `{"name":5}`, http.StatusBadRequest},
		{"not an object", `[]`, http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wantError(t, h, "POST", "/api/category-groups", tc.body, tc.want)
		})
	}
	wantGroupNames(t, listTestGroups(t, h), "Housing")
}

func TestUpdateCategoryGroup(t *testing.T) {
	h := newTestServer(t, builtDist)
	g := createTestGroup(t, h, "housing")
	createTestGroup(t, h, "Fun")
	target := fmt.Sprintf("/api/category-groups/%d", g.ID)

	res, raw := doJSON(t, h, "PATCH", target, map[string]string{"name": " Housing "})
	wantStatus(t, res, raw, http.StatusOK)
	got := decode[store.CategoryGroup](t, raw)
	if got.ID != g.ID || got.Name != "Housing" || got.SortOrder != g.SortOrder {
		t.Errorf("renamed = %+v", got)
	}

	res, raw = doJSON(t, h, "PATCH", target, map[string]any{"sort_order": 9})
	wantStatus(t, res, raw, http.StatusOK)
	got = decode[store.CategoryGroup](t, raw)
	if got.Name != "Housing" || got.SortOrder != 9 {
		t.Errorf("reordered = %+v", got)
	}

	wantError(t, h, "PATCH", target, map[string]any{}, http.StatusBadRequest)
	wantError(t, h, "PATCH", target, map[string]string{"name": "  "}, http.StatusBadRequest)
	wantError(t, h, "PATCH", target, map[string]string{"name": "FUN"}, http.StatusConflict)
	wantError(t, h, "PATCH", target, map[string]any{"sort_order": "first"}, http.StatusBadRequest)
	wantError(t, h, "PATCH", target, map[string]any{"color": "red"}, http.StatusBadRequest)
	wantError(t, h, "PATCH", "/api/category-groups/999", map[string]string{"name": "X"}, http.StatusNotFound)
	wantError(t, h, "PATCH", "/api/category-groups/abc", map[string]string{"name": "X"}, http.StatusBadRequest)
}

func TestReorderCategoryGroups(t *testing.T) {
	h := newTestServer(t, builtDist)
	a := createTestGroup(t, h, "A")
	b := createTestGroup(t, h, "B")
	c := createTestGroup(t, h, "C")

	res, raw := doJSON(t, h, "PUT", "/api/category-groups/order", map[string]any{"ids": []int64{c.ID, a.ID, b.ID}})
	wantStatus(t, res, raw, http.StatusOK)
	got := decode[[]store.CategoryGroup](t, raw)
	wantGroupNames(t, got, "C", "A", "B")
	wantGroupNames(t, listTestGroups(t, h), "C", "A", "B")

	order := "/api/category-groups/order"
	wantError(t, h, "PUT", order, map[string]any{"ids": []int64{a.ID, b.ID}}, http.StatusBadRequest)             // missing one
	wantError(t, h, "PUT", order, map[string]any{"ids": []int64{a.ID, a.ID, b.ID, c.ID}}, http.StatusBadRequest) // repeat
	wantError(t, h, "PUT", order, map[string]any{"ids": []int64{a.ID, b.ID, 999}}, http.StatusBadRequest)        // unknown
	wantError(t, h, "PUT", order, map[string]any{"ids": []int64{}}, http.StatusBadRequest)
	wantError(t, h, "PUT", order, map[string]any{}, http.StatusBadRequest)
	wantError(t, h, "PUT", order, map[string]any{"ids": "abc"}, http.StatusBadRequest)
	wantError(t, h, "PUT", order, map[string]any{"ids": []int64{a.ID, b.ID, c.ID}, "extra": 1}, http.StatusBadRequest)
	wantGroupNames(t, listTestGroups(t, h), "C", "A", "B") // failed requests changed nothing
}

func TestDeleteCategoryGroup(t *testing.T) {
	h := newTestServer(t, builtDist)
	g := createTestGroup(t, h, "Housing")
	res, raw := doJSON(t, h, "POST", "/api/categories", map[string]any{"name": "Rent", "group_id": g.ID})
	wantStatus(t, res, raw, http.StatusCreated)
	rent := decode[store.Category](t, raw)

	target := fmt.Sprintf("/api/category-groups/%d", g.ID)
	wantNoContent(t, h, "DELETE", target)
	wantGroupNames(t, listTestGroups(t, h))

	// Its category is still there, now ungrouped.
	res, raw = doJSON(t, h, "GET", "/api/categories", nil)
	wantStatus(t, res, raw, http.StatusOK)
	found := false
	for _, c := range decode[[]store.Category](t, raw) {
		if c.ID == rent.ID {
			found = true
			if c.GroupID != nil {
				t.Errorf("rent group_id = %d, want null", *c.GroupID)
			}
		}
	}
	if !found {
		t.Error("rent was deleted along with its group")
	}

	wantError(t, h, "DELETE", target, nil, http.StatusNotFound)
	wantError(t, h, "DELETE", "/api/category-groups/abc", nil, http.StatusBadRequest)
}

func TestCategoryGroupID(t *testing.T) {
	h := newTestServer(t, builtDist)
	housing := createTestGroup(t, h, "Housing")
	fun := createTestGroup(t, h, "Fun")

	// Ungrouped by default, and the field is there as null.
	res, raw := doJSON(t, h, "POST", "/api/categories", map[string]string{"name": "Plain"})
	wantStatus(t, res, raw, http.StatusCreated)
	plain := decode[map[string]any](t, raw)
	if v, ok := plain["group_id"]; !ok || v != nil {
		t.Errorf("plain = %v, want group_id: null", plain)
	}

	res, raw = doJSON(t, h, "POST", "/api/categories", map[string]any{"name": "Rent", "group_id": housing.ID})
	wantStatus(t, res, raw, http.StatusCreated)
	rent := decode[store.Category](t, raw)
	if rent.GroupID == nil || *rent.GroupID != housing.ID {
		t.Fatalf("created = %+v, want group %d", rent, housing.ID)
	}
	target := fmt.Sprintf("/api/categories/%d", rent.ID)

	// Move, leave alone, ungroup.
	res, raw = doJSON(t, h, "PATCH", target, map[string]any{"group_id": fun.ID})
	wantStatus(t, res, raw, http.StatusOK)
	if got := decode[store.Category](t, raw); got.GroupID == nil || *got.GroupID != fun.ID {
		t.Errorf("moved = %+v, want group %d", got, fun.ID)
	}
	res, raw = doJSON(t, h, "PATCH", target, map[string]string{"name": "Rent (home)"})
	wantStatus(t, res, raw, http.StatusOK)
	if got := decode[store.Category](t, raw); got.GroupID == nil || *got.GroupID != fun.ID {
		t.Errorf("renamed = %+v, group_id omitted should leave it alone", got)
	}
	res, raw = doJSON(t, h, "PATCH", target, map[string]any{"group_id": nil})
	wantStatus(t, res, raw, http.StatusOK)
	if got := decode[store.Category](t, raw); got.GroupID != nil {
		t.Errorf("ungrouped = %+v", got)
	}

	// GET shows it.
	res, raw = doJSON(t, h, "PATCH", target, map[string]any{"group_id": housing.ID})
	wantStatus(t, res, raw, http.StatusOK)
	res, raw = doJSON(t, h, "GET", "/api/categories", nil)
	wantStatus(t, res, raw, http.StatusOK)
	for _, c := range decode[[]store.Category](t, raw) {
		if c.ID == rent.ID && (c.GroupID == nil || *c.GroupID != housing.ID) {
			t.Errorf("listed rent = %+v, want group %d", c, housing.ID)
		}
	}

	// Errors: unknown group is 404, income can't be grouped, wrong type is 400.
	wantError(t, h, "POST", "/api/categories", map[string]any{"name": "Lost", "group_id": 999}, http.StatusNotFound)
	wantError(t, h, "PATCH", target, map[string]any{"group_id": 999}, http.StatusNotFound)
	wantError(t, h, "PATCH", target, map[string]any{"group_id": "housing"}, http.StatusBadRequest)
	wantError(t, h, "POST", "/api/categories", map[string]any{"name": "Bonus", "is_income": true, "group_id": housing.ID}, http.StatusBadRequest)
	wantError(t, h, "PATCH", target, map[string]any{"is_income": true, "group_id": housing.ID}, http.StatusBadRequest)

	// Moving a grouped category to income ungroups it.
	res, raw = doJSON(t, h, "PATCH", target, map[string]any{"is_income": true})
	wantStatus(t, res, raw, http.StatusOK)
	if got := decode[store.Category](t, raw); !got.IsIncome || got.GroupID != nil {
		t.Errorf("moved to income = %+v, want ungrouped income", got)
	}
}

func TestEnvelopesGroupsAPI(t *testing.T) {
	h := newTestServer(t, builtDist)
	housing := createTestGroup(t, h, "Housing")
	res, raw := doJSON(t, h, "POST", "/api/categories", map[string]any{"name": "Rent", "group_id": housing.ID})
	wantStatus(t, res, raw, http.StatusCreated)
	rent := decode[store.Category](t, raw)
	coffee := createTestCategory(t, h, "Coffee")

	res, raw = doJSON(t, h, "PUT", "/api/envelopes/start", map[string]any{"month": "2026-11"})
	wantStatus(t, res, raw, http.StatusOK)
	res, raw = doJSON(t, h, "PUT", fmt.Sprintf("/api/envelopes/%d?month=2026-11", rent.ID), map[string]any{"amount_cents": 5000})
	wantStatus(t, res, raw, http.StatusOK)

	res, raw = doJSON(t, h, "GET", "/api/envelopes?month=2026-11", nil)
	wantStatus(t, res, raw, http.StatusOK)
	body := decode[map[string]any](t, raw)

	groups, _ := body["groups"].([]any)
	if len(groups) != 1 {
		t.Fatalf("groups = %v, want one", body["groups"])
	}
	g := groups[0].(map[string]any)
	for key, want := range map[string]any{"id": float64(housing.ID), "name": "Housing", "assigned_cents": float64(5000), "spent_cents": float64(0), "balance_cents": float64(5000)} {
		if g[key] != want {
			t.Errorf("group %s = %v, want %v", key, g[key], want)
		}
	}

	// The envelopes stay flat, each with its group_id (null when ungrouped).
	envs, _ := body["envelopes"].([]any)
	if len(envs) != 2 {
		t.Fatalf("envelopes = %v, want two", body["envelopes"])
	}
	for _, e := range envs {
		e := e.(map[string]any)
		v, ok := e["group_id"]
		if !ok {
			t.Errorf("envelope %v has no group_id key", e)
		}
		switch e["category_id"] {
		case float64(rent.ID):
			if v != float64(housing.ID) {
				t.Errorf("rent group_id = %v, want %d", v, housing.ID)
			}
		case float64(coffee.ID):
			if v != nil {
				t.Errorf("coffee group_id = %v, want null", v)
			}
		}
	}

	// No groups at all is [] rather than null.
	h2 := newTestServer(t, builtDist)
	res, raw = doJSON(t, h2, "GET", "/api/envelopes?month=2026-11", nil)
	wantStatus(t, res, raw, http.StatusOK)
	if got := decode[map[string]any](t, raw); got["groups"] == nil {
		t.Errorf("groups = null, want []")
	}
}
