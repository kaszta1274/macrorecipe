package middleware

import (
	"context"
	"net/http"

	"github.com/kaszta1274/macrorecipe/internal/api/response"
	"github.com/kaszta1274/macrorecipe/internal/auth"
	"github.com/kaszta1274/macrorecipe/internal/database"
)

func AuthMiddleware(tokenSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString, err := auth.GetBearerToken(r.Header)
			if err != nil {
				response.RespondWithError(w, http.StatusUnauthorized, "missing or malformed authorization header", err)
				return
			}

			userId, role, err := auth.ValidateJWT(tokenString, tokenSecret)
			if err != nil {
				response.RespondWithError(w, http.StatusUnauthorized, "invalid jwt", err)
				return
			}

			ctx := context.WithValue(r.Context(), "user_id", userId)
			ctx = context.WithValue(r.Context(), "role", role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, ok := r.Context().Value("role").(database.UserRole)
		if !ok || role != database.UserRoleAdmin {
			response.RespondWithError(w, http.StatusForbidden, "requires admin", nil)
			return
		}

		next.ServeHTTP(w, r)
	})
}
