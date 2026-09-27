package http

import (
	"context"
	"net/http"
	"strings"

	"be-remote-device/internal/config"
	"be-remote-device/internal/service"
	"be-remote-device/pkg/response"
)

type contextKey string

const (
	UserIDContextKey   contextKey = "user_id"
	DeviceIDContextKey contextKey = "device_id"
	ActorContextKey    contextKey = "actor_identity"
)

func AuthMiddleware(cfg *config.Config, authSvc service.AuthService, deviceSvc service.DeviceService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path

			// Public routes: no auth required
			if path == "/health" || path == "/api/v1/health" ||
				path == "/api/v1/auth/login" || path == "/api/v1/auth/register" {
				next.ServeHTTP(w, r)
				return
			}

			ctx := r.Context()

			// 1. Check Device API Key (from laptop or smartphone agent)
			apiKey := r.Header.Get("X-Device-Key")
			if apiKey == "" {
				apiKey = r.Header.Get("X-API-Key")
			}
			if apiKey == "" {
				apiKey = r.URL.Query().Get("api_key")
			}

			if apiKey != "" {
				// Check against Master API Key if set
				if cfg.MasterAPIKey != "" && apiKey == cfg.MasterAPIKey {
					r.Header.Set("X-Actor", "apikey:master")
					ctx = context.WithValue(ctx, ActorContextKey, "apikey:master")
					// Master key has superuser access, leave userID empty or master
					ctx = context.WithValue(ctx, UserIDContextKey, "master")
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}

				// Check per-device API key
				dev, err := deviceSvc.GetDeviceByAPIKey(apiKey)
				if err == nil && dev != nil {
					identity := "device:" + dev.Name + " (" + dev.ID + ")"
					r.Header.Set("X-Actor", identity)
					ctx = context.WithValue(ctx, UserIDContextKey, dev.UserID)
					ctx = context.WithValue(ctx, DeviceIDContextKey, dev.ID)
					ctx = context.WithValue(ctx, ActorContextKey, identity)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			// 2. Check Bearer Token (JWT for user / admin dashboard)
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
				// Also check if Master API Key passed as Bearer
				if cfg.MasterAPIKey != "" && tokenStr == cfg.MasterAPIKey {
					r.Header.Set("X-Actor", "apikey:master")
					ctx = context.WithValue(ctx, ActorContextKey, "apikey:master")
					ctx = context.WithValue(ctx, UserIDContextKey, "master")
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}

				claims, err := authSvc.ValidateToken(tokenStr)
				if err == nil && claims != nil {
					identity := "user:" + claims.Username + " (" + claims.UserID + ")"
					r.Header.Set("X-Actor", identity)
					ctx = context.WithValue(ctx, UserIDContextKey, claims.UserID)
					ctx = context.WithValue(ctx, ActorContextKey, identity)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			// 3. Unauthorized
			response.Error(w, http.StatusUnauthorized, "Unauthorized: valid user Bearer token or device X-Device-Key required")
		})
	}
}

func GetUserIDFromContext(ctx context.Context) string {
	if val, ok := ctx.Value(UserIDContextKey).(string); ok {
		return val
	}
	return ""
}
