package auth

import (
	"context"
	"fmt"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ospiem/gophermart/internal/models"
)

type ContextKey string

const ContextLoginKey ContextKey = "login"

func JWTAuthorization(key string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			login, err := getUserLogin(r.Header.Get("Authorization"), key)
			if err != nil {
				http.Error(w, "", http.StatusUnauthorized)
			}

			ctx := context.WithValue(r.Context(), ContextLoginKey, login)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}

func getUserLogin(tokenString string, key string) (string, error) {
	claims := &models.Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(key), nil
	})
	if err != nil {
		return "", fmt.Errorf("cannot parse claims: %w", err)
	}
	if !token.Valid {
		return "", fmt.Errorf("token invalid")
	}
	return claims.Login, nil
}
