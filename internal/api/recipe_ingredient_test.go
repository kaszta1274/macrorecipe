package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/kaszta1274/macrorecipe/internal/api/handler"
)

func TestRecipeIngredients(t *testing.T) {
	requireTestDB(t)

	alice := testSeed.Users[0]
	bob := testSeed.Users[1]
	aliceRecipe := testSeed.Recipes[0] // Chicken and Rice Bowl

	aliceToken := mustToken(t, alice)
	bobToken := mustToken(t, bob)
	adminToken := mustToken(t, testSeed.Admin)

	chicken := testSeed.Ingredients["Chicken Breast"] // already on aliceRecipe
	eggs := testSeed.Ingredients["Whole Eggs"]        // not on aliceRecipe
	salmon := testSeed.Ingredients["Salmon Fillet"]   // not on aliceRecipe

	basePath := "/api/recipes/" + aliceRecipe.ID.String() + "/ingredients"

	t.Run("list seeded ingredients", func(t *testing.T) {
		router := newTestRouter(t)

		rec := doJSON(t, router, http.MethodGet, basePath, aliceToken, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
		}
		listed := decodeJSON[[]handler.RecipeIngredientsByRecipe](t, rec)
		if len(listed) != 4 {
			t.Fatalf("seeded ingredient count = %d, want 4", len(listed))
		}
		for _, item := range listed {
			if item.RecipeID != aliceRecipe.ID || item.IngredientName == "" || item.Amount <= 0 {
				t.Fatalf("invalid listed recipe ingredient: %+v", item)
			}
		}
	})

	cases := []struct {
		name       string
		method     string
		path       string
		token      string
		body       any
		wantStatus int
		check      func(*testing.T, *httptest.ResponseRecorder)
	}{
		{"alice add", http.MethodPost, basePath, aliceToken, handler.RecipeIngredientCreateRequest{IngredientID: eggs.ID, Amount: 100}, http.StatusCreated, assertRecipeIngredient(aliceRecipe.ID, eggs.ID, 100)},
		{"alice add duplicate", http.MethodPost, basePath, aliceToken, handler.RecipeIngredientCreateRequest{IngredientID: chicken.ID, Amount: 150}, http.StatusConflict, nil},
		{"bob add forbidden", http.MethodPost, basePath, bobToken, handler.RecipeIngredientCreateRequest{IngredientID: eggs.ID, Amount: 100}, http.StatusForbidden, nil},
		{"admin add", http.MethodPost, basePath, adminToken, handler.RecipeIngredientCreateRequest{IngredientID: eggs.ID, Amount: 100}, http.StatusCreated, assertRecipeIngredient(aliceRecipe.ID, eggs.ID, 100)},
		{"alice update", http.MethodPut, basePath + "/" + chicken.ID.String(), aliceToken, handler.RecipeIngredientUpdateRequest{Amount: 250}, http.StatusOK, assertRecipeIngredient(aliceRecipe.ID, chicken.ID, 250)},
		{"alice update missing", http.MethodPut, basePath + "/" + salmon.ID.String(), aliceToken, handler.RecipeIngredientUpdateRequest{Amount: 100}, http.StatusNotFound, nil},
		{"bob update forbidden", http.MethodPut, basePath + "/" + chicken.ID.String(), bobToken, handler.RecipeIngredientUpdateRequest{Amount: 250}, http.StatusForbidden, nil},
		{"bob delete forbidden", http.MethodDelete, basePath + "/" + chicken.ID.String(), bobToken, nil, http.StatusForbidden, nil},
		{"alice delete", http.MethodDelete, basePath + "/" + chicken.ID.String(), aliceToken, nil, http.StatusNoContent, nil},
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

func assertRecipeIngredient(wantRecipeID, wantIngredientID uuid.UUID, wantAmount float32) func(*testing.T, *httptest.ResponseRecorder) {
	return func(t *testing.T, rec *httptest.ResponseRecorder) {
		t.Helper()
		got := decodeJSON[handler.RecipeIngredient](t, rec)
		if got.RecipeID != wantRecipeID {
			t.Fatalf("recipe_id = %s, want %s", got.RecipeID, wantRecipeID)
		}
		if got.IngredientID != wantIngredientID {
			t.Fatalf("ingredient_id = %s, want %s", got.IngredientID, wantIngredientID)
		}
		if got.Amount != wantAmount {
			t.Fatalf("amount = %v, want %v", got.Amount, wantAmount)
		}
	}
}
