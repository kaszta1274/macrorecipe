package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/kaszta1274/macrorecipe/internal/api/handler"
	"github.com/kaszta1274/macrorecipe/internal/database"
	"github.com/kaszta1274/macrorecipe/internal/seeder"
)

func TestUsers(t *testing.T) {
	requireTestDB(t)

	alice := testSeed.Users[0]
	bob := testSeed.Users[1]
	aliceToken := mustToken(t, alice)

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
			"register new user",
			http.MethodPost, "/api/auth/register", "",
			handler.UserCreateRequest{Email: "charlie@example.com", Username: "charlie", Password: "password123"},
			http.StatusCreated,
			assertUser(uuid.Nil, "charlie@example.com", "charlie", database.UserRoleUser),
		},
		{
			"register duplicate email",
			http.MethodPost, "/api/auth/register", "",
			handler.UserCreateRequest{Email: alice.Email, Username: "charlie", Password: "password123"},
			http.StatusConflict, nil,
		},
		{
			"register duplicate username",
			http.MethodPost, "/api/auth/register", "",
			handler.UserCreateRequest{Email: "charlie@example.com", Username: alice.Username, Password: "password123"},
			http.StatusConflict, nil,
		},
		{
			"get user",
			http.MethodGet, "/api/users/" + alice.ID.String(), aliceToken, nil,
			http.StatusOK,
			assertUser(alice.ID, alice.Email, alice.Username, database.UserRoleUser),
		},
		{
			"get missing user",
			http.MethodGet, "/api/users/" + uuid.NewString(), aliceToken, nil,
			http.StatusNotFound, nil,
		},
		{
			"get unauthenticated",
			http.MethodGet, "/api/users/" + alice.ID.String(), "", nil,
			http.StatusUnauthorized, nil,
		},
		{
			"update username",
			http.MethodPatch, "/api/users/profile", aliceToken,
			handler.UserUpdateRequest{Username: "aliceNew"},
			http.StatusOK,
			assertUser(alice.ID, alice.Email, "aliceNew", database.UserRoleUser),
		},
		{
			"update username conflict",
			http.MethodPatch, "/api/users/profile", aliceToken,
			handler.UserUpdateRequest{Username: bob.Username},
			http.StatusConflict, nil,
		},
		{
			"update password",
			http.MethodPatch, "/api/users/password", aliceToken,
			handler.UserUpdatePasswordRequest{Password: seeder.TestUserPassword, NewPassword: "newpassword123"},
			http.StatusOK,
			assertUser(alice.ID, alice.Email, alice.Username, database.UserRoleUser),
		},
		{
			"update password wrong current",
			http.MethodPatch, "/api/users/password", aliceToken,
			handler.UserUpdatePasswordRequest{Password: "wrongpassword", NewPassword: "newpassword123"},
			http.StatusUnauthorized, nil,
		},
		{
			"delete user",
			http.MethodDelete, "/api/users", aliceToken,
			handler.UserDeleteRequest{Password: seeder.TestUserPassword},
			http.StatusNoContent, nil,
		},
		{
			"delete user wrong password",
			http.MethodDelete, "/api/users", aliceToken,
			handler.UserDeleteRequest{Password: "wrongpassword"},
			http.StatusUnauthorized, nil,
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

func assertUser(wantID uuid.UUID, wantEmail, wantUsername string, wantRole database.UserRole) func(*testing.T, *httptest.ResponseRecorder) {
	return func(t *testing.T, rec *httptest.ResponseRecorder) {
		t.Helper()
		got := decodeJSON[handler.User](t, rec)
		if wantID != uuid.Nil && got.ID != wantID {
			t.Fatalf("id = %s, want %s", got.ID, wantID)
		}
		if got.ID == uuid.Nil {
			t.Fatal("expected non-nil user id")
		}
		if got.Email != wantEmail {
			t.Fatalf("email = %q, want %q", got.Email, wantEmail)
		}
		if got.Username != wantUsername {
			t.Fatalf("username = %q, want %q", got.Username, wantUsername)
		}
		if got.Role != wantRole {
			t.Fatalf("role = %q, want %q", got.Role, wantRole)
		}
	}
}
