package auth

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/kaszta1274/macrorecipe/internal/database"
)

type TokenType string

const (
	TokenTypeAccess TokenType = "macrorecipe-access"
)

type CustomClaims struct {
	Role database.UserRole `json:"role"`
	jwt.RegisteredClaims
}

var ErrNoAuthHeaderIncluded = errors.New("no auth header included in request")

func MakeJWTWithRole(
	userID uuid.UUID,
	role database.UserRole,
	tokenSecret string,
	expiresIn time.Duration,
) (string, error) {
	signingKey := []byte(tokenSecret)

	claims := CustomClaims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    string(TokenTypeAccess),
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
			ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(expiresIn)),
			Subject:   userID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(signingKey)
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, database.UserRole, error) {
	claims := &CustomClaims{}
	_, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(t *jwt.Token) (any, error) { return []byte(tokenSecret), nil },
	)
	if err != nil {
		return uuid.Nil, "", err
	}

	userIDString, err := claims.GetSubject()
	if err != nil {
		return uuid.Nil, "", err
	}

	issuer, err := claims.GetIssuer()
	if err != nil {
		return uuid.Nil, "", err
	}
	if issuer != string(TokenTypeAccess) {
		return uuid.Nil, "", errors.New("invalid issuer")
	}

	id, err := uuid.Parse(userIDString)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("invalid user ID: %w", err)
	}
	return id, claims.Role, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		return "", ErrNoAuthHeaderIncluded
	}

	splitAuth := strings.Split(authHeader, " ")
	if len(splitAuth) < 2 || splitAuth[0] != "Bearer" {
		return "", errors.New("malformed authorization header")
	}

	return splitAuth[1], nil
}
