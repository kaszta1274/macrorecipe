package api_test

import (
	"net/http"
	"testing"
)

func TestRateLimits(t *testing.T) {
	requireTestDB(t)

	aliceToken := mustToken(t, testSeed.Users[0])

	cases := []struct {
		name   string
		method string
		path   string
		token  string
		body   any
		limit  int
	}{
		// Empty JSON bodies fail validation (400) before Argon2id, keeping the test fast.
		{"auth login limited by IP", http.MethodPost, "/api/auth/login", "", map[string]any{}, 10},
		{"sensitive password limited by user", http.MethodPatch, "/api/users/password", aliceToken, map[string]any{}, 5},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router := newTestRouter(t)

			for i := 1; i <= tc.limit; i++ {
				rec := doJSON(t, router, tc.method, tc.path, tc.token, tc.body)
				if rec.Code == http.StatusTooManyRequests {
					t.Fatalf("req %d/%d unexpectedly rate-limited", i, tc.limit)
				}
			}

			rec := doJSON(t, router, tc.method, tc.path, tc.token, tc.body)
			if rec.Code != http.StatusTooManyRequests {
				t.Fatalf("req %d status = %d, want 429; body = %s", tc.limit+1, rec.Code, rec.Body.String())
			}
			if rec.Header().Get("Retry-After") == "" {
				t.Fatal("expected Retry-After header on 429")
			}
			assertErrorJSON(t, rec)
		})
	}
}
