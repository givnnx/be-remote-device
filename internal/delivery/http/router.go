package http

import (
	"log/slog"
	"net/http"
	"time"

	"be-remote-device/internal/service"
)

func NewRouter(
	handler *DeviceHandler,
	logHandler *LogHandler,
	authHandler *AuthHandler,
	logSvc service.ActionLogService,
	authSvc service.AuthService,
) http.Handler {
	mux := http.NewServeMux()

	// Base / Health (Public)
	mux.HandleFunc("GET /health", handler.HealthCheck)
	mux.HandleFunc("GET /api/v1/health", handler.HealthCheck)

	// Auth (Public)
	mux.HandleFunc("POST /api/v1/auth/login", authHandler.Login)

	// Device Management (Protected)
	mux.HandleFunc("GET /api/v1/devices", handler.ListDevices)
	mux.HandleFunc("POST /api/v1/devices", handler.CreateDevice)
	mux.HandleFunc("GET /api/v1/devices/{id}", handler.GetDevice)
	mux.HandleFunc("PUT /api/v1/devices/{id}", handler.UpdateDevice)
	mux.HandleFunc("DELETE /api/v1/devices/{id}", handler.DeleteDevice)

	// Heartbeat (Protected)
	mux.HandleFunc("POST /api/v1/devices/{id}/heartbeat", handler.Heartbeat)

	// Remote Commands (Protected)
	mux.HandleFunc("POST /api/v1/devices/{id}/commands", handler.SendCommand)
	mux.HandleFunc("GET /api/v1/devices/{id}/commands", handler.ListCommands)
	mux.HandleFunc("POST /api/v1/commands/{id}/result", handler.UpdateCommandResult)

	// Telemetry & Metrics (Protected)
	mux.HandleFunc("POST /api/v1/devices/{id}/telemetry", handler.RecordTelemetry)
	mux.HandleFunc("GET /api/v1/devices/{id}/telemetry", handler.GetTelemetry)

	// Action / Audit Logs (Protected)
	mux.HandleFunc("GET /api/v1/logs", logHandler.ListLogs)
	mux.HandleFunc("GET /api/v1/logs/{id}", logHandler.GetLog)

	return applyMiddlewares(mux, logSvc, authSvc)
}

func applyMiddlewares(next http.Handler, logSvc service.ActionLogService, authSvc service.AuthService) http.Handler {
	// Recovery -> CORS -> Audit (records everything including 401s) -> Console Log -> Auth
	return recoveryMiddleware(corsMiddleware(AuditMiddleware(logSvc)(loggingMiddleware(AuthMiddleware(authSvc)(next)))))
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("Request handled",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"duration", time.Since(start).String(),
		)
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("Panic recovered", "error", rec)
				http.Error(w, `{"success":false,"error":"Internal Server Error"}`, http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
