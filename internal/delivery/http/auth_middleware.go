package http

import (
	"context"
	"net/http"
	"strings"

	"be-remote-device/internal/service"
	"be-remote-device/pkg/response"
)

type contextKey string

const (
	UserContextKey contextKey = "user_identity"
)

func AuthMiddleware(authSvc service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path

			// Public routes: no auth required
			if path == "/health" || path == "/api/v1/health" || path == "/api/v1/auth/login" {
				next.ServeHTTP(w, r)
				return
			}

			// 1. Check Master / Device API Key via Header
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				apiKey = r.URL.Query().Get("api_key")
			}
			if apiKey != "" && authSvc.ValidateAPIKey(apiKey) {
				r.Header.Set("X-Actor", "apikey:master")
				ctx := context.WithValue(r.Context(), UserContextKey, "apikey:master")
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// 2. Check Bearer Token (JWT or API Key)
			var tokenStr string
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				parts := strings.SplitN(authHeader, " ", 2)
				if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
					tokenStr = strings.TrimSpace(parts[1])
				}
			}
			if tokenStr == "" {
				tokenStr = r.URL.Query().Get("token")
			}

			if tokenStr != "" {
				// Allow Master API Key passed as Bearer token
				if authSvc.ValidateAPIKey(tokenStr) {
					r.Header.Set("X-Actor", "apikey:master")
					ctx := context.WithValue(r.Context(), UserContextKey, "apikey:master")
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}

				// Validate JWT
				claims, err := authSvc.ValidateToken(tokenStr)
				if err == nil && claims != nil {
					identity := "user:" + claims.Username
					r.Header.Set("X-Actor", identity)
					ctx := context.WithValue(r.Context(), UserContextKey, identity)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			// 3. Unauthorized
			response.Error(w, http.StatusUnauthorized, "Unauthorized: valid Bearer token or X-API-Key required")
		})
	}
}
