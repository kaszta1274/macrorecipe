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
	aliceRecipe := testSeed.Recipes[0] // Chicken and Rice Bowl: 2 portions, 4 ingredients, 1 like (bob), 2 comments

	aliceToken := mustToken(t, alice)
	bobToken := mustToken(t, bob)
	adminToken := mustToken(t, testSeed.Admin)

	validRecipe := handler.RecipeRequest{
		Name:         "Egg Scramble",
		Description:  "Quick scrambled eggs with olive oil.",
		Instructions: "Whisk eggs and cook in pan over medium heat.",
		Portions:     1,
	}

	wantBowlTotal := handler.Macros{Calories: 819.3, Protein: 73.3, Fat: 18.5, Carbohydrates: 89.9}
	wantBowlPerPortion := handler.Macros{Calories: 409.7, Protein: 36.7, Fat: 9.3, Carbohydrates: 45.0}

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
			"alice create",
			http.MethodPost, "/api/recipes", aliceToken, validRecipe,
			http.StatusCreated,
			assertRecipe(uuid.Nil, alice.ID, validRecipe.Name, validRecipe.Portions),
		},
		{
			"alice get detail",
			http.MethodGet, "/api/recipes/" + aliceRecipe.ID.String(), aliceToken, nil,
			http.StatusOK,
			func(t *testing.T, rec *httptest.ResponseRecorder) {
				got := decodeJSON[handler.RecipeDetail](t, rec)
				if got.ID != aliceRecipe.ID || got.Name != aliceRecipe.Name || got.Instructions != aliceRecipe.Instructions {
					t.Fatalf("unexpected recipe detail metadata: %+v", got)
				}
				if got.Author.ID != alice.ID || got.Author.Username != alice.Username {
					t.Fatalf("author = %+v, want (%s, %q)", got.Author, alice.ID, alice.Username)
				}
				if len(got.Ingredients) != 4 {
					t.Fatalf("ingredients count = %d, want 4", len(got.Ingredients))
				}
				if got.MacrosTotal != wantBowlTotal {
					t.Fatalf("macros_total = %+v, want %+v", got.MacrosTotal, wantBowlTotal)
				}
				if got.MacrosPerPortion != wantBowlPerPortion {
					t.Fatalf("macros_per_portion = %+v, want %+v", got.MacrosPerPortion, wantBowlPerPortion)
				}
				if got.LikesCount != 1 || got.LikedByMe {
					t.Fatalf("likes = (%d, %v), want (1, false)", got.LikesCount, got.LikedByMe)
				}
				if got.CommentsCount != 2 {
					t.Fatalf("comments_count = %d, want 2", got.CommentsCount)
				}
			},
		},
		{
			"bob get detail sees liked_by_me true",
			http.MethodGet, "/api/recipes/" + aliceRecipe.ID.String(), bobToken, nil,
			http.StatusOK,
			func(t *testing.T, rec *httptest.ResponseRecorder) {
				got := decodeJSON[handler.RecipeDetail](t, rec)
				if got.LikesCount != 1 || !got.LikedByMe {
					t.Fatalf("likes = (%d, %v), want (1, true)", got.LikesCount, got.LikedByMe)
				}
			},
		},
		{
			"get missing",
			http.MethodGet, "/api/recipes/" + uuid.NewString(), aliceToken, nil,
			http.StatusNotFound, nil,
		},
		{
			"list recipes by name returns summaries with macros and counts",
			http.MethodGet, "/api/recipes?query=Chicken", bobToken, nil,
			http.StatusOK,
			func(t *testing.T, rec *httptest.ResponseRecorder) {
				got := decodeJSON[[]handler.RecipeSummary](t, rec)
				if len(got) != 1 {
					t.Fatalf("recipe count = %d, want 1", len(got))
				}
				r := got[0]
				if r.ID != aliceRecipe.ID || r.Author.Username != alice.Username {
					t.Fatalf("unexpected recipe summary: %+v", r)
				}
				if r.MacrosPerPortion != wantBowlPerPortion {
					t.Fatalf("macros_per_portion = %+v, want %+v", r.MacrosPerPortion, wantBowlPerPortion)
				}
				if r.LikesCount != 1 || !r.LikedByMe || r.CommentsCount != 2 {
					t.Fatalf("counts = (likes=%d liked=%v comments=%d), want (1, true, 2)", r.LikesCount, r.LikedByMe, r.CommentsCount)
				}
			},
		},
		{
			"list by user id",
			http.MethodGet, "/api/users/" + alice.ID.String() + "/recipes", aliceToken, nil,
			http.StatusOK,
			func(t *testing.T, rec *httptest.ResponseRecorder) {
				got := decodeJSON[[]handler.RecipeSummary](t, rec)
				if len(got) != 2 {
					t.Fatalf("recipe count = %d, want 2", len(got))
				}
				for _, r := range got {
					if r.Author.ID != alice.ID || r.Author.Username != alice.Username {
						t.Fatalf("unexpected author = %+v, want %s", r.Author, alice.ID)
					}
				}
			},
		},
		{
			"list my recipes",
			http.MethodGet, "/api/users/recipes", aliceToken, nil,
			http.StatusOK,
			func(t *testing.T, rec *httptest.ResponseRecorder) {
				got := decodeJSON[[]handler.RecipeSummary](t, rec)
				if len(got) != 2 {
					t.Fatalf("my recipes count = %d, want 2", len(got))
				}
			},
		},
		{
			"alice update own",
			http.MethodPut, "/api/recipes/" + aliceRecipe.ID.String(), aliceToken, validRecipe,
			http.StatusOK,
			assertRecipe(aliceRecipe.ID, alice.ID, validRecipe.Name, validRecipe.Portions),
		},
		{
			"bob update forbidden",
			http.MethodPut, "/api/recipes/" + aliceRecipe.ID.String(), bobToken, validRecipe,
			http.StatusForbidden, nil,
		},
		{
			"bob delete forbidden",
			http.MethodDelete, "/api/recipes/" + aliceRecipe.ID.String(), bobToken, nil,
			http.StatusForbidden, nil,
		},
		{
			"admin update",
			http.MethodPut, "/api/recipes/" + aliceRecipe.ID.String(), adminToken, validRecipe,
			http.StatusOK,
			assertRecipe(aliceRecipe.ID, alice.ID, validRecipe.Name, validRecipe.Portions),
		},
		{
			"admin delete",
			http.MethodDelete, "/api/recipes/" + aliceRecipe.ID.String(), adminToken, nil,
			http.StatusNoContent, nil,
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
