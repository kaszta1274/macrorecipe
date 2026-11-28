package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/kaszta1274/macrorecipe/internal/api/handler"
)

func TestFollows(t *testing.T) {
	requireTestDB(t)

	alice := testSeed.Users[0] // seeded: alice follows bob
	bob := testSeed.Users[1]

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
			"bob follow alice",
			http.MethodPost, "/api/users/" + alice.ID.String() + "/follow", bobToken,
			http.StatusCreated,
			assertFollow(bob.ID, alice.ID),
		},
		{
			"alice duplicate follow bob",
			http.MethodPost, "/api/users/" + bob.ID.String() + "/follow", aliceToken,
			http.StatusConflict, nil,
		},
		{
			"alice self follow rejected",
			http.MethodPost, "/api/users/" + alice.ID.String() + "/follow", aliceToken,
			http.StatusBadRequest, nil,
		},
		{
			"follow missing user",
			http.MethodPost, "/api/users/" + uuid.NewString() + "/follow", aliceToken,
			http.StatusNotFound, nil,
		},
		{
			"follow unauthenticated",
			http.MethodPost, "/api/users/" + bob.ID.String() + "/follow", "",
			http.StatusUnauthorized, nil,
		},
		{
			"list bob followers",
			http.MethodGet, "/api/users/" + bob.ID.String() + "/followers", bobToken,
			http.StatusOK,
			assertUserListSingle(alice.ID, alice.Username, false),
		},
		{
			"list alice following as alice sees followed_by_me true",
			http.MethodGet, "/api/users/" + alice.ID.String() + "/following", aliceToken,
			http.StatusOK,
			assertUserListSingle(bob.ID, bob.Username, true),
		},
		{
			"list alice following as bob sees followed_by_me false",
			http.MethodGet, "/api/users/" + alice.ID.String() + "/following", bobToken,
			http.StatusOK,
			assertUserListSingle(bob.ID, bob.Username, false),
		},
		{
			"list followers missing user",
			http.MethodGet, "/api/users/" + uuid.NewString() + "/followers", aliceToken,
			http.StatusNotFound, nil,
		},
		{
			"list following missing user",
			http.MethodGet, "/api/users/" + uuid.NewString() + "/following", aliceToken,
			http.StatusNotFound, nil,
		},
		{
			"alice unfollow bob",
			http.MethodDelete, "/api/users/" + bob.ID.String() + "/follow", aliceToken,
			http.StatusNoContent, nil,
		},
		{
			"bob unfollow alice idempotent",
			http.MethodDelete, "/api/users/" + alice.ID.String() + "/follow", bobToken,
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

func assertFollow(wantFollowerID, wantFollowingID uuid.UUID) func(*testing.T, *httptest.ResponseRecorder) {
	return func(t *testing.T, rec *httptest.ResponseRecorder) {
		t.Helper()
		got := decodeJSON[handler.Follow](t, rec)
		if got.FollowerID != wantFollowerID || got.FollowingID != wantFollowingID {
			t.Fatalf("follow = (%s -> %s), want (%s -> %s)", got.FollowerID, got.FollowingID, wantFollowerID, wantFollowingID)
		}
	}
}

func assertUserListSingle(wantID uuid.UUID, wantUsername string, wantFollowedByMe bool) func(*testing.T, *httptest.ResponseRecorder) {
	return func(t *testing.T, rec *httptest.ResponseRecorder) {
		t.Helper()
		got := decodeJSON[handler.Page[handler.UserListItem]](t, rec)
		if len(got.Items) != 1 || got.HasMore {
			t.Fatalf("got (len=%d has_more=%v), want (1, false)", len(got.Items), got.HasMore)
		}
		item := got.Items[0]
		if item.ID != wantID || item.Username != wantUsername || item.FollowedByMe != wantFollowedByMe || item.FollowedAt.IsZero() {
			t.Fatalf("item = %+v, want (%s, %q, followed_by_me=%v)", item, wantID, wantUsername, wantFollowedByMe)
		}
	}
}
