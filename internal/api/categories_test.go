package api

import (
	"fmt"
	"net/http"
	"testing"

	"budgeteer/internal/store"
)

func TestCreateAndListCategories(t *testing.T) {
	h := newTestServer(t, builtDist)

	res, raw := doJSON(t, h, "POST", "/api/categories", map[string]any{"name": "  Rent ", "is_income": false})
	wantStatus(t, res, raw, http.StatusCreated)
	rent := decode[store.Category](t, raw)
	if rent.ID == 0 || rent.Name != "Rent" || rent.IsIncome {
		t.Errorf("created = %+v", rent)
	}

	res, raw = doJSON(t, h, "POST", "/api/categories", map[string]any{"name": "Bonus", "is_income": true})
	wantStatus(t, res, raw, http.StatusCreated)
	bonus := decode[store.Category](t, raw)
	if !bonus.IsIncome {
		t.Errorf("created = %+v, want is_income", bonus)
	}

	// is_income defaults to false.
	res, raw = doJSON(t, h, "POST", "/api/categories", map[string]string{"name": "groceries"})
	wantStatus(t, res, raw, http.StatusCreated)
	groceries := decode[store.Category](t, raw)
	if groceries.IsIncome {
		t.Errorf("created = %+v, want is_income false", groceries)
	}

	// Expense first, then income, each sorted case-insensitively by name.
	res, raw = doJSON(t, h, "GET", "/api/categories", nil)
	wantStatus(t, res, raw, http.StatusOK)
	list := decode[[]store.Category](t, raw)
	want := []string{"groceries", "Rent", "Bonus", "Income", "Opening balance"}
	if len(list) != len(want) {
		t.Fatalf("list = %+v, want names %v", list, want)
	}
	for i, name := range want {
		if list[i].Name != name {
			t.Fatalf("list = %+v, want names %v", list, want)
		}
	}
	if list[1] != rent || list[2] != bonus {
		t.Errorf("list = %+v, want it to contain %+v and %+v", list, rent, bonus)
	}
}

func TestListCategoriesFreshDatabaseHasSeeds(t *testing.T) {
	h := newTestServer(t, builtDist)
	res, raw := doJSON(t, h, "GET", "/api/categories", nil)
	wantStatus(t, res, raw, http.StatusOK)
	list := decode[[]store.Category](t, raw)
	if len(list) != 2 || list[0].Name != "Income" || list[1].Name != "Opening balance" || !list[0].IsIncome || !list[1].IsIncome {
		t.Errorf("list = %+v, want the two seeded income categories", list)
	}
}

func TestCreateCategoryErrors(t *testing.T) {
	h := newTestServer(t, builtDist)
	res, raw := doJSON(t, h, "POST", "/api/categories", map[string]string{"name": "Coffee"})
	wantStatus(t, res, raw, http.StatusCreated)

	cases := []struct {
		name string
		body any
		want int
	}{
		{"duplicate name, different case", map[string]string{"name": "COFFEE"}, http.StatusConflict},
		{"duplicate of seeded category", map[string]any{"name": "income", "is_income": true}, http.StatusConflict},
		{"duplicate after trimming", map[string]string{"name": " Coffee "}, http.StatusConflict},
		{"blank name", map[string]string{"name": "  "}, http.StatusBadRequest},
		{"missing name", map[string]any{"is_income": true}, http.StatusBadRequest},
		{"unknown field", map[string]string{"name": "Gas", "color": "red"}, http.StatusBadRequest},
		{"wrong type for is_income", `{"name":"Gas","is_income":"yes"}`, http.StatusBadRequest},
		{"malformed JSON", `{"name":`, http.StatusBadRequest},
		{"two objects", `{"name":"Gas"} {}`, http.StatusBadRequest},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res, raw := doJSON(t, h, "POST", "/api/categories", c.body)
			wantStatus(t, res, raw, c.want)
			if e := decode[map[string]string](t, raw); e["error"] == "" {
				t.Errorf("missing error message: %s", raw)
			}
		})
	}
}

func createTestCategory(t *testing.T, h http.Handler, name string) store.Category {
	t.Helper()
	res, raw := doJSON(t, h, "POST", "/api/categories", map[string]string{"name": name})
	wantStatus(t, res, raw, http.StatusCreated)
	return decode[store.Category](t, raw)
}

func TestUpdateCategory(t *testing.T) {
	h := newTestServer(t, builtDist)
	c := createTestCategory(t, h, "coffee")
	target := fmt.Sprintf("/api/categories/%d", c.ID)

	res, raw := doJSON(t, h, "PATCH", target, map[string]any{"name": " Coffee ", "is_income": true})
	wantStatus(t, res, raw, http.StatusOK)
	if got := decode[store.Category](t, raw); got != (store.Category{ID: c.ID, Name: "Coffee", IsIncome: true}) {
		t.Errorf("updated = %+v", got)
	}

	wantError(t, h, "PATCH", target, map[string]string{}, http.StatusBadRequest)
	wantError(t, h, "PATCH", target, map[string]string{"name": ""}, http.StatusBadRequest)
	wantError(t, h, "PATCH", target, map[string]string{"name": "INCOME"}, http.StatusConflict)
	wantError(t, h, "PATCH", "/api/categories/999", map[string]string{"name": "X"}, http.StatusNotFound)
}

func TestOpeningBalanceCategoryIsLocked(t *testing.T) {
	h := newTestServer(t, builtDist)
	res, raw := doJSON(t, h, "GET", "/api/categories", nil)
	wantStatus(t, res, raw, http.StatusOK)
	var id int64
	for _, c := range decode[[]store.Category](t, raw) {
		if c.Name == "Opening balance" {
			id = c.ID
		}
	}
	target := fmt.Sprintf("/api/categories/%d", id)
	wantError(t, h, "PATCH", target, map[string]string{"name": "Start"}, http.StatusConflict)
	wantError(t, h, "DELETE", target, nil, http.StatusConflict)
}

func TestDeleteCategory(t *testing.T) {
	h := newTestServer(t, builtDist)
	a := createTestAccount(t, h, "Checking")
	used := createTestCategory(t, h, "Coffee")
	createTestTransaction(t, h, map[string]any{"account_id": a.ID, "category_id": used.ID, "date": "2026-10-01", "amount_cents": -100})
	unused := createTestCategory(t, h, "Gifts")

	wantError(t, h, "DELETE", fmt.Sprintf("/api/categories/%d", used.ID), nil, http.StatusConflict)
	wantNoContent(t, h, "DELETE", fmt.Sprintf("/api/categories/%d", unused.ID))
	wantError(t, h, "DELETE", fmt.Sprintf("/api/categories/%d", unused.ID), nil, http.StatusNotFound)
}
