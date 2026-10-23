package api

import (
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
