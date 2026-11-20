package api_test

import (
	"net/http"
	"testing"

	"github.com/kaszta1274/macrorecipe/internal/api/handler"
	"github.com/kaszta1274/macrorecipe/internal/database"
	"github.com/kaszta1274/macrorecipe/internal/seeder"
)

func TestLoginSeededUsers(t *testing.T) {
	requireTestDB(t)

	cases := []struct {
		name       string
		email      string
		password   string
		wantStatus int
		wantRole   database.UserRole
	}{
		{"admin", testAdminEmail, testAdminPassword, http.StatusOK, database.UserRoleAdmin},
		{"alice", seeder.TestUser1Email, seeder.TestUserPassword, http.StatusOK, database.UserRoleUser},
		{"wrong password", seeder.TestUser1Email, "wrongpassword", http.StatusUnauthorized, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router := newTestRouter(t)

			rec := doJSON(t, router, http.MethodPost, "/api/auth/login", "", handler.UserLoginRequest{
				Email:    tc.email,
				Password: tc.password,
			})
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantStatus >= 400 {
				assertErrorJSON(t, rec)
				return
			}

			resp := decodeJSON[handler.UserLoginResponse](t, rec)
			if resp.Token == "" {
				t.Fatal("expected non-empty JWT token")
			}
			if resp.Email != tc.email {
				t.Fatalf("email = %q, want %q", resp.Email, tc.email)
			}
			if resp.Role != tc.wantRole {
				t.Fatalf("role = %q, want %q", resp.Role, tc.wantRole)
			}
		})
	}
}
