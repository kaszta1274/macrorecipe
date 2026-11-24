package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/kaszta1274/macrorecipe/internal/auth"
	"github.com/kaszta1274/macrorecipe/internal/database"
)

func TestRateLimitByIP(t *testing.T) {
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := chimw.ClientIPFromRemoteAddr(RateLimitByIP(2, time.Minute)(okHandler))

	send := func(remoteAddr string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = remoteAddr
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	for i := 1; i <= 2; i++ {
		if rec := send("10.0.0.1:1234"); rec.Code != http.StatusOK {
			t.Fatalf("req %d status = %d, want 200", i, rec.Code)
		}
	}

	limited := send("10.0.0.1:5678")
	if limited.Code != http.StatusTooManyRequests {
		t.Fatalf("3rd req status = %d, want 429", limited.Code)
	}
	if limited.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header on 429")
	}
	var body struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(limited.Body).Decode(&body); err != nil || body.Error == "" {
		t.Fatalf("expected JSON error body, got %q (err=%v)", limited.Body.String(), err)
	}

	// Different IP gets its own bucket.
	if rec := send("10.0.0.2:1234"); rec.Code != http.StatusOK {
		t.Fatalf("other IP status = %d, want 200", rec.Code)
	}
}

func TestRateLimitByUser(t *testing.T) {
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := chimw.ClientIPFromRemoteAddr(RateLimitByUser(2, time.Minute)(okHandler))

	userA := uuid.New()
	userB := uuid.New()

	sendAs := func(userID uuid.UUID) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		ctx := auth.WithAuthContext(req.Context(), auth.AuthContext{UserID: userID, Role: database.UserRoleUser})
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req.WithContext(ctx))
		return rec
	}

	for i := 1; i <= 2; i++ {
		if rec := sendAs(userA); rec.Code != http.StatusOK {
			t.Fatalf("userA req %d status = %d, want 200", i, rec.Code)
		}
	}
	if rec := sendAs(userA); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("userA 3rd req status = %d, want 429", rec.Code)
	}
	// Different user from the same IP gets their own bucket.
	if rec := sendAs(userB); rec.Code != http.StatusOK {
		t.Fatalf("userB req status = %d, want 200", rec.Code)
	}
}
