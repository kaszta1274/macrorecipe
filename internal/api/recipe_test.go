package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/kaszta1274/macrorecipe/internal/api/handler"
)

func TestRecipes(t *testing.T) {
	requireTestDB(t)

	alice := testSeed.Users[0]
	bob := testSeed.Users[1]
	aliceRecipe := testSeed.Recipes[0]

	aliceToken := mustToken(t, alice)
	bobToken := mustToken(t, bob)
	adminToken := mustToken(t, testSeed.Admin)

	validRecipe := handler.RecipeRequest{
		Name:         "Egg Scramble",
		Description:  "Quick scrambled eggs with olive oil.",
		Instructions: "Whisk eggs and cook in pan over medium heat.",
		Portions:     1,
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
		{"alice create", http.MethodPost, "/api/recipes", aliceToken, validRecipe, http.StatusCreated, assertRecipe(uuid.Nil, alice.ID, validRecipe.Name, validRecipe.Portions)},
		{"alice get", http.MethodGet, "/api/recipes/" + aliceRecipe.ID.String(), aliceToken, nil, http.StatusOK, assertRecipe(aliceRecipe.ID, alice.ID, aliceRecipe.Name, int(aliceRecipe.Portions))},
		{"get missing", http.MethodGet, "/api/recipes/" + uuid.NewString(), aliceToken, nil, http.StatusNotFound, nil},
		{"list by user id", http.MethodGet, "/api/users/" + alice.ID.String() + "/recipes", aliceToken, nil, http.StatusOK, func(t *testing.T, rec *httptest.ResponseRecorder) {
			got := decodeJSON[[]handler.Recipe](t, rec)
			if len(got) != 2 {
				t.Fatalf("recipe count = %d, want 2", len(got))
			}
			for _, r := range got {
				if r.UserID != alice.ID {
					t.Fatalf("unexpected user_id = %s, want %s", r.UserID, alice.ID)
				}
			}
		}},
		{"alice update own", http.MethodPut, "/api/recipes/" + aliceRecipe.ID.String(), aliceToken, validRecipe, http.StatusOK, assertRecipe(aliceRecipe.ID, alice.ID, validRecipe.Name, validRecipe.Portions)},
		{"bob update forbidden", http.MethodPut, "/api/recipes/" + aliceRecipe.ID.String(), bobToken, validRecipe, http.StatusForbidden, nil},
		{"bob delete forbidden", http.MethodDelete, "/api/recipes/" + aliceRecipe.ID.String(), bobToken, nil, http.StatusForbidden, nil},
		{"admin update", http.MethodPut, "/api/recipes/" + aliceRecipe.ID.String(), adminToken, validRecipe, http.StatusOK, assertRecipe(aliceRecipe.ID, alice.ID, validRecipe.Name, validRecipe.Portions)},
		{"admin delete", http.MethodDelete, "/api/recipes/" + aliceRecipe.ID.String(), adminToken, nil, http.StatusNoContent, nil},
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

func assertRecipe(wantID, wantUserID uuid.UUID, wantName string, wantPortions int) func(*testing.T, *httptest.ResponseRecorder) {
	return func(t *testing.T, rec *httptest.ResponseRecorder) {
		t.Helper()
		got := decodeJSON[handler.Recipe](t, rec)
		if wantID != uuid.Nil && got.ID != wantID {
			t.Fatalf("id = %s, want %s", got.ID, wantID)
		}
		if got.ID == uuid.Nil {
			t.Fatal("expected non-nil recipe id")
		}
		if got.UserID != wantUserID {
			t.Fatalf("user_id = %s, want %s", got.UserID, wantUserID)
		}
		if got.Name != wantName {
			t.Fatalf("name = %q, want %q", got.Name, wantName)
		}
		if got.Portions != wantPortions {
			t.Fatalf("portions = %d, want %d", got.Portions, wantPortions)
		}
	}
}
