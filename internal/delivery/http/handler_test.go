package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"be-remote-device/internal/config"
	"be-remote-device/internal/models"
	"be-remote-device/internal/repository"
	"be-remote-device/internal/service"
)

func TestDeviceHandler_EndpointsAuthAndAuditLogging(t *testing.T) {
	cfg := &config.Config{
		AppName:       "be-remote-device",
		JWTSecret:     "test-secret-key-12345",
		MasterAPIKey:  "my-laptop-key-abcde",
		AdminUsername: "admin",
		AdminPassword: "secretpassword",
	}

	deviceRepo := repository.NewMemoryDeviceRepository()
	deviceSvc := service.NewDeviceService(deviceRepo)
	deviceHandler := NewDeviceHandler(deviceSvc)

	logRepo := repository.NewMemoryActionLogRepository()
	logSvc := service.NewActionLogService(logRepo)
	defer logSvc.Stop()

	authSvc := service.NewAuthService(cfg)
	authHandler := NewAuthHandler(authSvc)
	logHandler := NewLogHandler(logSvc)

	router := NewRouter(deviceHandler, logHandler, authHandler, logSvc, authSvc)

	// 1. Health check (Public: Should work without any auth)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	// 2. Protected endpoint WITHOUT credentials -> Should return 401 Unauthorized
	req = httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 Unauthorized, got %d", rr.Code)
	}

	// 3. Login to get JWT Token
	loginBody, _ := json.Marshal(models.LoginRequest{
		Username: "admin",
		Password: "secretpassword",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected login status 200, got %d", rr.Code)
	}

	var loginResp struct {
		Data models.LoginResponse `json:"data"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &loginResp)
	jwtToken := loginResp.Data.Token
	if jwtToken == "" {
		t.Fatalf("expected valid JWT token from login")
	}

	// 4. Create Device using JWT Bearer Token
	devicePayload := models.CreateDeviceRequest{
		Name:      "My-Personal-Laptop",
		Type:      "laptop",
		IPAddress: "192.168.1.15",
	}
	body, _ := json.Marshal(devicePayload)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/devices", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+jwtToken)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201 Created with JWT, got %d, body: %s", rr.Code, rr.Body.String())
	}

	// 5. Access with Master API Key (e.g., from laptop/smartphone daemon)
	req = httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
	req.Header.Set("X-API-Key", "my-laptop-key-abcde")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK with API Key, got %d", rr.Code)
	}

	// Wait briefly for asynchronous log worker
	time.Sleep(50 * time.Millisecond)

	// 6. Query Action Logs (using API Key)
	req = httptest.NewRequest(http.MethodGet, "/api/v1/logs", nil)
	req.Header.Set("X-API-Key", "my-laptop-key-abcde")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200 reading logs, got %d", rr.Code)
	}

	var logResp struct {
		Data struct {
			Total int                 `json:"total"`
			Items []*models.ActionLog `json:"items"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &logResp)

	// Logs should have captured the unauthorized attempt, the registration, and the listing!
	if logResp.Data.Total < 2 {
		t.Errorf("expected at least 2 logs captured, got %d", logResp.Data.Total)
	}
}
