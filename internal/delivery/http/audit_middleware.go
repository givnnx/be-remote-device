package http

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"be-remote-device/internal/models"
	"be-remote-device/internal/service"
)

type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

func (rw *responseWriterWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriterWrapper) Write(b []byte) (int, error) {
	// Limit captured response body to 4KB
	if rw.body.Len() < 4096 {
		remaining := 4096 - rw.body.Len()
		if len(b) > remaining {
			rw.body.Write(b[:remaining])
		} else {
			rw.body.Write(b)
		}
	}
	return rw.ResponseWriter.Write(b)
}

func AuditMiddleware(logSvc service.ActionLogService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip logging for health checks to prevent filling logs with frequent polling
			if r.URL.Path == "/health" || r.URL.Path == "/api/v1/health" {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()

			// 1. Capture Client IP
			clientIP := extractClientIP(r)

			// 2. Identify Actor (pelaku request)
			actor := extractActor(r, clientIP)

			// 3. Capture Request Body (without consuming it permanently)
			var reqBodyStr string
			if r.Body != nil && r.ContentLength != 0 {
				bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 8192)) // Limit to 8KB
				if err == nil {
					reqBodyStr = string(bodyBytes)
					r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				}
			}

			// 4. Wrap ResponseWriter to capture status code & response
			wrappedWriter := &responseWriterWrapper{
				ResponseWriter: w,
				statusCode:     http.StatusOK, // Default if WriteHeader is not called
			}

			// 5. Execute downstream handlers
			next.ServeHTTP(wrappedWriter, r)

			duration := time.Since(start)

			// 6. Determine Action Name
			actionName := resolveActionName(r.Method, r.URL.Path)

			// 7. Determine Log Status & Error Message
			statusCode := wrappedWriter.statusCode
			respBodyStr := wrappedWriter.body.String()
			logStatus := models.LogStatusSuccess
			var errorMessage string

			if statusCode >= 500 {
				logStatus = models.LogStatusError
				errorMessage = extractErrorFromResponse(respBodyStr)
			} else if statusCode >= 400 {
				logStatus = models.LogStatusFailed
				errorMessage = extractErrorFromResponse(respBodyStr)
			}

			// 8. Construct ActionLog record
			logEntry := &models.ActionLog{
				Timestamp:    start.UTC(),
				Actor:        actor,
				Action:       actionName,
				Method:       r.Method,
				Path:         r.URL.Path,
				ClientIP:     clientIP,
				UserAgent:    r.UserAgent(),
				StatusCode:   statusCode,
				DurationMs:   duration.Milliseconds(),
				RequestBody:  reqBodyStr,
				ResponseBody: respBodyStr,
				ErrorMessage: errorMessage,
				Status:       logStatus,
			}

			// 9. Asynchronously save log to DB
			logSvc.LogAction(logEntry)
		})
	}
}

func extractClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func extractActor(r *http.Request, clientIP string) string {
	if actor := r.Header.Get("X-User-ID"); actor != "" {
		return actor
	}
	if actor := r.Header.Get("X-Actor"); actor != "" {
		return actor
	}
	if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
		return "api-key:" + maskString(apiKey)
	}
	if auth := r.Header.Get("Authorization"); auth != "" {
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) == 2 {
			return "auth:" + parts[0]
		}
	}
	// Fallback to client IP
	return "ip:" + clientIP
}

func maskString(s string) string {
	if len(s) <= 6 {
		return "***"
	}
	return s[:3] + "..." + s[len(s)-3:]
}

func resolveActionName(method, path string) string {
	switch {
	case method == http.MethodGet && path == "/api/v1/devices":
		return "LIST_DEVICES"
	case method == http.MethodPost && path == "/api/v1/devices":
		return "REGISTER_DEVICE"
	case strings.HasPrefix(path, "/api/v1/devices/") && strings.HasSuffix(path, "/heartbeat"):
		return "DEVICE_HEARTBEAT"
	case strings.HasPrefix(path, "/api/v1/devices/") && strings.HasSuffix(path, "/commands"):
		if method == http.MethodPost {
			return "DISPATCH_COMMAND"
		}
		return "LIST_DEVICE_COMMANDS"
	case strings.HasPrefix(path, "/api/v1/commands/") && strings.HasSuffix(path, "/result"):
		return "UPDATE_COMMAND_RESULT"
	case strings.HasPrefix(path, "/api/v1/devices/") && strings.HasSuffix(path, "/telemetry"):
		if method == http.MethodPost {
			return "RECORD_TELEMETRY"
		}
		return "GET_TELEMETRY"
	case strings.HasPrefix(path, "/api/v1/devices/"):
		switch method {
		case http.MethodGet:
			return "GET_DEVICE_DETAIL"
		case http.MethodPut:
			return "UPDATE_DEVICE"
		case http.MethodDelete:
			return "DELETE_DEVICE"
		}
	case strings.HasPrefix(path, "/api/v1/logs"):
		return "QUERY_ACTION_LOGS"
	}

	return method + " " + path
}

func extractErrorFromResponse(resp string) string {
	if resp == "" {
		return ""
	}
	var res map[string]interface{}
	if err := json.Unmarshal([]byte(resp), &res); err == nil {
		if errMsg, ok := res["error"].(string); ok && errMsg != "" {
			return errMsg
		}
		if msg, ok := res["message"].(string); ok && msg != "" {
			return msg
		}
	}
	if len(resp) > 200 {
		return resp[:200]
	}
	return resp
}
