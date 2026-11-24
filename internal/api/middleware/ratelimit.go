package middleware

import (
	"net"
	"net/http"
	"time"

	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/kaszta1274/macrorecipe/internal/api/response"
	"github.com/kaszta1274/macrorecipe/internal/auth"
)

func RateLimitByIP(requestLimit int, window time.Duration) func(http.Handler) http.Handler {
	return newRateLimiter(requestLimit, window, clientIPKey)
}

func RateLimitByUser(requestLimit int, window time.Duration) func(http.Handler) http.Handler {
	return newRateLimiter(requestLimit, window, func(r *http.Request) (string, error) {
		if userID, ok := auth.UserIDFromContext(r.Context()); ok {
			return userID.String(), nil
		}
		return clientIPKey(r)
	})
}

func clientIPKey(r *http.Request) (string, error) {
	ip := chimw.GetClientIP(r.Context())
	if ip == "" {
		ip = r.RemoteAddr
		if host, _, err := net.SplitHostPort(ip); err == nil {
			ip = host
		}
	}
	return httprate.CanonicalizeIP(ip), nil
}

func newRateLimiter(requestLimit int, window time.Duration, keyFn httprate.KeyFunc) func(http.Handler) http.Handler {
	return httprate.LimitBy(
		requestLimit,
		window,
		keyFn,
		httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
			response.RespondWithError(w, r, http.StatusTooManyRequests, "too many requests", nil)
		}),
	)
}
