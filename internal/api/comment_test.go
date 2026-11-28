package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/kaszta1274/macrorecipe/internal/api/handler"
)

func TestComments(t *testing.T) {
	requireTestDB(t)

	alice := testSeed.Users[0]
	bob := testSeed.Users[1]
	aliceRecipe := testSeed.Recipes[0]   // Chicken and Rice Bowl (2 seeded comments)
	oatmealRecipe := testSeed.Recipes[1] // Protein Oatmeal (0 seeded comments)
	aliceComment := testSeed.Comments[0] // alice's comment on aliceRecipe

	aliceToken := mustToken(t, alice)
	bobToken := mustToken(t, bob)
	adminToken := mustToken(t, testSeed.Admin)

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
			http.MethodPost, "/api/recipes/" + aliceRecipe.ID.String() + "/comments", aliceToken,
			handler.CommentRequest{Body: "  Delicious!  "},
			http.StatusCreated,
			assertComment(uuid.Nil, alice.ID, aliceRecipe.ID, "Delicious!"),
		},
		{
			"create whitespace body",
			http.MethodPost, "/api/recipes/" + aliceRecipe.ID.String() + "/comments", aliceToken,
			handler.CommentRequest{Body: "   "},
			http.StatusBadRequest, nil,
		},
		{
			"create missing recipe",
			http.MethodPost, "/api/recipes/" + uuid.NewString() + "/comments", aliceToken,
			handler.CommentRequest{Body: "Nice!"},
			http.StatusNotFound, nil,
		},
		{
			"create unauthenticated",
			http.MethodPost, "/api/recipes/" + aliceRecipe.ID.String() + "/comments", "",
			handler.CommentRequest{Body: "Nice!"},
			http.StatusUnauthorized, nil,
		},
		{
			"list recipe comments",
			http.MethodGet, "/api/recipes/" + aliceRecipe.ID.String() + "/comments", aliceToken, nil,
			http.StatusOK,
			func(t *testing.T, rec *httptest.ResponseRecorder) {
				got := decodeJSON[handler.Page[handler.CommentsByRecipeId]](t, rec)
				if len(got.Items) != 2 || got.HasMore {
					t.Fatalf("got (len=%d has_more=%v), want (2, false)", len(got.Items), got.HasMore)
				}
				for _, c := range got.Items {
					if c.RecipeID != aliceRecipe.ID || c.Username == "" || c.Body == "" {
						t.Fatalf("invalid listed comment: %+v", c)
					}
				}
			},
		},
		{
			"list empty recipe comments",
			http.MethodGet, "/api/recipes/" + oatmealRecipe.ID.String() + "/comments", aliceToken, nil,
			http.StatusOK,
			func(t *testing.T, rec *httptest.ResponseRecorder) {
				got := decodeJSON[handler.Page[handler.CommentsByRecipeId]](t, rec)
				if len(got.Items) != 0 || got.HasMore {
					t.Fatalf("got (len=%d has_more=%v), want (0, false)", len(got.Items), got.HasMore)
				}
			},
		},
		{
			"list missing recipe",
			http.MethodGet, "/api/recipes/" + uuid.NewString() + "/comments", aliceToken, nil,
			http.StatusNotFound, nil,
		},
		{
			"alice update own",
			http.MethodPut, "/api/comments/" + aliceComment.ID.String(), aliceToken,
			handler.CommentRequest{Body: "  Updated meal prep tip  "},
			http.StatusOK,
			assertComment(aliceComment.ID, alice.ID, aliceRecipe.ID, "Updated meal prep tip"),
		},
		{
			"alice update whitespace body",
			http.MethodPut, "/api/comments/" + aliceComment.ID.String(), aliceToken,
			handler.CommentRequest{Body: "   "},
			http.StatusBadRequest, nil,
		},
		{
			"bob update forbidden",
			http.MethodPut, "/api/comments/" + aliceComment.ID.String(), bobToken,
			handler.CommentRequest{Body: "Hijacked"},
			http.StatusForbidden, nil,
		},
		{
			"admin update forbidden",
			http.MethodPut, "/api/comments/" + aliceComment.ID.String(), adminToken,
			handler.CommentRequest{Body: "Admin edit"},
			http.StatusForbidden, nil,
		},
		{
			"update missing comment",
			http.MethodPut, "/api/comments/" + uuid.NewString(), aliceToken,
			handler.CommentRequest{Body: "Updated"},
			http.StatusNotFound, nil,
		},
		{
			"bob delete forbidden",
			http.MethodDelete, "/api/comments/" + aliceComment.ID.String(), bobToken, nil,
			http.StatusForbidden, nil,
		},
		{
			"alice delete own",
			http.MethodDelete, "/api/comments/" + aliceComment.ID.String(), aliceToken, nil,
			http.StatusNoContent, nil,
		},
		{
			"admin delete moderation",
			http.MethodDelete, "/api/comments/" + aliceComment.ID.String(), adminToken, nil,
			http.StatusNoContent, nil,
		},
		{
			"delete missing comment",
			http.MethodDelete, "/api/comments/" + uuid.NewString(), aliceToken, nil,
			http.StatusNotFound, nil,
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

func assertComment(wantID, wantUserID, wantRecipeID uuid.UUID, wantBody string) func(*testing.T, *httptest.ResponseRecorder) {
	return func(t *testing.T, rec *httptest.ResponseRecorder) {
		t.Helper()
		got := decodeJSON[handler.Comment](t, rec)
		if wantID != uuid.Nil && got.ID != wantID {
			t.Fatalf("id = %s, want %s", got.ID, wantID)
		}
		if got.ID == uuid.Nil {
			t.Fatal("expected non-nil comment id")
		}
		if got.UserID != wantUserID {
			t.Fatalf("user_id = %s, want %s", got.UserID, wantUserID)
		}
		if got.RecipeID != wantRecipeID {
			t.Fatalf("recipe_id = %s, want %s", got.RecipeID, wantRecipeID)
		}
		if got.Body != wantBody {
			t.Fatalf("body = %q, want %q", got.Body, wantBody)
		}
	}
}
