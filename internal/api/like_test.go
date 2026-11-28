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
	bob := testSeed.Users[1]
	aliceRecipe := testSeed.Recipes[0] // not liked by alice
	bobRecipe := testSeed.Recipes[2]   // already liked by alice in seeder (1 comment by alice)

	aliceToken := mustToken(t, alice)
	bobToken := mustToken(t, bob)

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
			"list alice likes as alice",
			http.MethodGet, "/api/users/" + alice.ID.String() + "/likes", aliceToken,
			http.StatusOK,
			func(t *testing.T, rec *httptest.ResponseRecorder) {
				got := decodeJSON[handler.Page[handler.LikedRecipe]](t, rec)
				if len(got.Items) != 1 || got.HasMore {
					t.Fatalf("got (len=%d has_more=%v), want (1, false)", len(got.Items), got.HasMore)
				}
				r := got.Items[0]
				if r.ID != bobRecipe.ID || r.Name != bobRecipe.Name || r.Description != bobRecipe.Description {
					t.Fatalf("unexpected liked recipe: %+v", r)
				}
				if r.Author.ID != bob.ID || r.Author.Username != bob.Username {
					t.Fatalf("author = %+v, want (%s, %q)", r.Author, bob.ID, bob.Username)
				}
				if r.LikesCount != 1 || !r.LikedByMe || r.CommentsCount != 1 {
					t.Fatalf("counts = (likes=%d liked=%v comments=%d), want (1, true, 1)", r.LikesCount, r.LikedByMe, r.CommentsCount)
				}
				if r.MacrosPerPortion.Calories <= 0 || r.LikedAt.IsZero() {
					t.Fatalf("expected positive calories and liked_at, got %+v", r)
				}
			},
		},
		{
			"list alice likes as bob sees liked_by_me false",
			http.MethodGet, "/api/users/" + alice.ID.String() + "/likes", bobToken,
			http.StatusOK,
			func(t *testing.T, rec *httptest.ResponseRecorder) {
				got := decodeJSON[handler.Page[handler.LikedRecipe]](t, rec)
				if len(got.Items) != 1 || got.Items[0].LikedByMe || got.HasMore {
					t.Fatalf("expected 1 liked recipe with liked_by_me=false and has_more=false for bob, got %+v", got)
				}
			},
		},
		{
			"list likes empty user",
			http.MethodGet, "/api/users/" + testSeed.Admin.ID.String() + "/likes", aliceToken,
			http.StatusOK,
			func(t *testing.T, rec *httptest.ResponseRecorder) {
				got := decodeJSON[handler.Page[handler.LikedRecipe]](t, rec)
				if len(got.Items) != 0 || got.HasMore {
					t.Fatalf("got (len=%d has_more=%v), want (0, false)", len(got.Items), got.HasMore)
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
