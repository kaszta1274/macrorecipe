package auth

import (
	"context"

	"github.com/google/uuid"
	"github.com/kaszta1274/macrorecipe/internal/database"
)

type contextKey struct{}

var authKey = contextKey{}

type AuthContext struct {
	UserID uuid.UUID
	Role   database.UserRole
}

func WithAuthContext(ctx context.Context, authCtx AuthContext) context.Context {
	return context.WithValue(ctx, authKey, authCtx)
}

func FromContext(ctx context.Context) (AuthContext, bool) {
	authCtx, ok := ctx.Value(authKey).(AuthContext)
	return authCtx, ok
}

func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	authCtx, ok := FromContext(ctx)
	if !ok {
		return uuid.Nil, false
	}
	return authCtx.UserID, true
}
