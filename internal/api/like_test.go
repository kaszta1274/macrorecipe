package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/kaszta1274/macrorecipe/internal/api/handler"
)

func TestLikes(t *testing.T) {
	requireTestDB(t)

	alice := testSeed.Users[0]
	aliceRecipe := testSeed.Recipes[0] // not liked by alice
	bobRecipe := testSeed.Recipes[2]   // already liked by alice in seeder

	aliceToken := mustToken(t, alice)

	cases := []struct {
		name       string
		method     string
		path       string
		token      string
		wantStatus int
		check      func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			"alice create like",
			http.MethodPost, "/api/recipes/" + aliceRecipe.ID.String() + "/likes", aliceToken,
			http.StatusCreated,
			assertLike(alice.ID, aliceRecipe.ID),
		},
		{
			"alice duplicate like",
			http.MethodPost, "/api/recipes/" + bobRecipe.ID.String() + "/likes", aliceToken,
			http.StatusConflict, nil,
		},
		{
			"create like missing recipe",
			http.MethodPost, "/api/recipes/" + uuid.NewString() + "/likes", aliceToken,
			http.StatusNotFound, nil,
		},
		{
			"create like unauthenticated",
			http.MethodPost, "/api/recipes/" + aliceRecipe.ID.String() + "/likes", "",
			http.StatusUnauthorized, nil,
		},
		{
			"list alice likes",
			http.MethodGet, "/api/users/" + alice.ID.String() + "/likes", aliceToken,
			http.StatusOK,
			func(t *testing.T, rec *httptest.ResponseRecorder) {
				got := decodeJSON[[]handler.LikeByUserId](t, rec)
				if len(got) != 1 {
					t.Fatalf("likes count = %d, want 1", len(got))
				}
				if got[0].UserID != alice.ID || got[0].RecipeID != bobRecipe.ID {
					t.Fatalf("like ids = (%s, %s), want (%s, %s)", got[0].UserID, got[0].RecipeID, alice.ID, bobRecipe.ID)
				}
				if got[0].RecipeName != bobRecipe.Name || got[0].RecipeDescription != bobRecipe.Description {
					t.Fatalf("like recipe = (%q, %q), want (%q, %q)", got[0].RecipeName, got[0].RecipeDescription, bobRecipe.Name, bobRecipe.Description)
				}
			},
		},
		{
			"list likes empty user",
			http.MethodGet, "/api/users/" + testSeed.Admin.ID.String() + "/likes", aliceToken,
			http.StatusOK,
			func(t *testing.T, rec *httptest.ResponseRecorder) {
				got := decodeJSON[[]handler.LikeByUserId](t, rec)
				if len(got) != 0 {
					t.Fatalf("likes count = %d, want 0", len(got))
				}
			},
		},
		{
			"alice delete seeded like",
			http.MethodDelete, "/api/recipes/" + bobRecipe.ID.String() + "/likes", aliceToken,
			http.StatusNoContent, nil,
		},
		{
			"alice delete unliked idempotent",
			http.MethodDelete, "/api/recipes/" + aliceRecipe.ID.String() + "/likes", aliceToken,
			http.StatusNoContent, nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router := newTestRouter(t)

			rec := doJSON(t, router, tc.method, tc.path, tc.token, nil)
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

func assertLike(wantUserID, wantRecipeID uuid.UUID) func(*testing.T, *httptest.ResponseRecorder) {
	return func(t *testing.T, rec *httptest.ResponseRecorder) {
		t.Helper()
		got := decodeJSON[handler.Like](t, rec)
		if got.UserID != wantUserID {
			t.Fatalf("user_id = %s, want %s", got.UserID, wantUserID)
		}
		if got.RecipeID != wantRecipeID {
			t.Fatalf("recipe_id = %s, want %s", got.RecipeID, wantRecipeID)
		}
	}
}
