package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/kaszta1274/macrorecipe/internal/api/handler"
)

func TestIngredients(t *testing.T) {
	requireTestDB(t)

	aliceToken := mustToken(t, testSeed.Users[0])
	adminToken := mustToken(t, testSeed.Admin)
	chicken := testSeed.Ingredients["Chicken Breast"]

	tofuReq := handler.IngredientRequest{
		Name:          "Tofu",
		Protein:       15.0,
		Fat:           8.0,
		Carbohydrates: 2.0,
	}
	updateReq := handler.IngredientRequest{
		Name:          "Chicken Thigh",
		Protein:       26.0,
		Fat:           10.0,
		Carbohydrates: 0.0,
	}

	cases := []struct {
		name       string
		method     string
		path       string
		token      string
		body       any
		wantStatus int
		check      func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			"user get",
			http.MethodGet, "/api/ingredients/" + chicken.ID.String(), aliceToken, nil,
			http.StatusOK,
			assertIngredient(chicken.ID, chicken.Name, chicken.Protein, chicken.Fat, chicken.Carbohydrates),
		},
		{
			"get missing",
			http.MethodGet, "/api/ingredients/" + uuid.NewString(), aliceToken, nil,
			http.StatusNotFound, nil,
		},
		{
			"list all",
			http.MethodGet, "/api/ingredients?limit=50", aliceToken, nil,
			http.StatusOK,
			func(t *testing.T, rec *httptest.ResponseRecorder) {
				got := decodeJSON[[]handler.Ingredient](t, rec)
				if len(got) != len(testSeed.Ingredients) {
					t.Fatalf("ingredient count = %d, want %d", len(got), len(testSeed.Ingredients))
				}
			},
		},
		{
			"list filtered by query",
			http.MethodGet, "/api/ingredients?query=Chicken", aliceToken, nil,
			http.StatusOK,
			func(t *testing.T, rec *httptest.ResponseRecorder) {
				got := decodeJSON[[]handler.Ingredient](t, rec)
				if len(got) != 1 || got[0].Name != chicken.Name {
					t.Fatalf("filtered ingredients = %+v, want [%q]", got, chicken.Name)
				}
			},
		},
		{
			"admin create",
			http.MethodPost, "/api/admin/ingredients", adminToken, tofuReq,
			http.StatusCreated,
			assertIngredient(uuid.Nil, tofuReq.Name, tofuReq.Protein, tofuReq.Fat, tofuReq.Carbohydrates),
		},
		{
			"admin create duplicate",
			http.MethodPost, "/api/admin/ingredients", adminToken,
			handler.IngredientRequest{Name: chicken.Name, Protein: 30, Fat: 3, Carbohydrates: 0},
			http.StatusConflict, nil,
		},
		{
			"user create forbidden",
			http.MethodPost, "/api/admin/ingredients", aliceToken, tofuReq,
			http.StatusForbidden, nil,
		},
		{
			"admin update",
			http.MethodPut, "/api/admin/ingredients/" + chicken.ID.String(), adminToken, updateReq,
			http.StatusOK,
			assertIngredient(chicken.ID, updateReq.Name, updateReq.Protein, updateReq.Fat, updateReq.Carbohydrates),
		},
		{
			"admin update duplicate name",
			http.MethodPut, "/api/admin/ingredients/" + chicken.ID.String(), adminToken,
			handler.IngredientRequest{Name: "White Rice", Protein: 10, Fat: 1, Carbohydrates: 50},
			http.StatusConflict, nil,
		},
		{
			"admin update missing",
			http.MethodPut, "/api/admin/ingredients/" + uuid.NewString(), adminToken, updateReq,
			http.StatusNotFound, nil,
		},
		{
			"user update forbidden",
			http.MethodPut, "/api/admin/ingredients/" + chicken.ID.String(), aliceToken, updateReq,
			http.StatusForbidden, nil,
		},
		{
			"admin delete",
			http.MethodDelete, "/api/admin/ingredients/" + chicken.ID.String(), adminToken, nil,
			http.StatusNoContent, nil,
		},
		{
			"user delete forbidden",
			http.MethodDelete, "/api/admin/ingredients/" + chicken.ID.String(), aliceToken, nil,
			http.StatusForbidden, nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router := newTestRouter(t)

			rec := doJSON(t, router, tc.method, tc.path, tc.token, tc.body)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantStatus >= 400 {
				assertErrorJSON(t, rec)
			} else if tc.wantStatus == http.StatusNoContent && rec.Body.Len() != 0 {
				t.Fatalf("expected empty body for 204, got %q", rec.Body.String())
			}
			if tc.check != nil {
				tc.check(t, rec)
			}
		})
	}
}

func assertIngredient(wantID uuid.UUID, wantName string, wantProtein, wantFat, wantCarbs float32) func(*testing.T, *httptest.ResponseRecorder) {
	return func(t *testing.T, rec *httptest.ResponseRecorder) {
		t.Helper()
		got := decodeJSON[handler.Ingredient](t, rec)
		if wantID != uuid.Nil && got.ID != wantID {
			t.Fatalf("id = %s, want %s", got.ID, wantID)
		}
		if got.ID == uuid.Nil {
			t.Fatal("expected non-nil ingredient id")
		}
		if got.Name != wantName {
			t.Fatalf("name = %q, want %q", got.Name, wantName)
		}
		if got.Protein != wantProtein || got.Fat != wantFat || got.Carbohydrates != wantCarbs {
			t.Fatalf("macros = (%v, %v, %v), want (%v, %v, %v)", got.Protein, got.Fat, got.Carbohydrates, wantProtein, wantFat, wantCarbs)
		}
	}
}
