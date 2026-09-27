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

func TestUserRegistrationLoginAndDeviceScoping(t *testing.T) {
	cfg := &config.Config{
		AppName:      "be-remote-device",
		JWTSecret:    "test-jwt-secret-12345",
		MasterAPIKey: "master-backup-key-123",
	}

	userRepo := repository.NewMemoryUserRepository()
	authSvc := service.NewAuthService(cfg, userRepo)
	authHandler := NewAuthHandler(authSvc)

	deviceRepo := repository.NewMemoryDeviceRepository()
	deviceSvc := service.NewDeviceService(deviceRepo)
	deviceHandler := NewDeviceHandler(deviceSvc)

	logRepo := repository.NewMemoryActionLogRepository()
	logSvc := service.NewActionLogService(logRepo)
	defer logSvc.Stop()

	logHandler := NewLogHandler(logSvc)

	router := NewRouter(cfg, deviceHandler, logHandler, authHandler, logSvc, authSvc, deviceSvc)

	// 1. Health check (Public)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	// 2. Register User 1
	regPayload := models.RegisterRequest{
		Username: "giovanni",
		Email:    "giovanni@personal.com",
		Password: "password12345",
		FullName: "Giovanni Agung",
	}
	body, _ := json.Marshal(regPayload)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created on registration, got %d, body: %s", rr.Code, rr.Body.String())
	}

	// 3. Login User 1
	loginPayload := models.LoginRequest{
		Username: "giovanni",
		Password: "password12345",
	}
	body, _ = json.Marshal(loginPayload)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on login, got %d", rr.Code)
	}

	var loginResp struct {
		Data models.LoginResponse `json:"data"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &loginResp)
	tokenUser1 := loginResp.Data.Token
	userID1 := loginResp.Data.User.ID

	// 4. Create Device for User 1 (Laptop)
	devPayload := models.CreateDeviceRequest{
		Name:      "My-Personal-Laptop",
		Type:      "laptop",
		IPAddress: "192.168.1.20",
	}
	body, _ = json.Marshal(devPayload)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/devices", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenUser1)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created device, got %d, body: %s", rr.Code, rr.Body.String())
	}

	var devResp struct {
		Data models.Device `json:"data"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &devResp)
	laptopDevice := devResp.Data

	if laptopDevice.UserID != userID1 {
		t.Errorf("expected device UserID %s, got %s", userID1, laptopDevice.UserID)
	}
	if laptopDevice.APIKey == "" {
		t.Errorf("expected generated device APIKey")
	}

	// 5. Device Daemon (Laptop) sends Heartbeat using its X-Device-Key
	req = httptest.NewRequest(http.MethodPost, "/api/v1/devices/"+laptopDevice.ID+"/heartbeat", nil)
	req.Header.Set("X-Device-Key", laptopDevice.APIKey)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on heartbeat via device key, got %d", rr.Code)
	}

	// 6. User 2 registers and shouldn't see User 1's device
	regPayload2 := models.RegisterRequest{
		Username: "otheruser",
		Email:    "other@example.com",
		Password: "password987",
	}
	body, _ = json.Marshal(regPayload2)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Login User 2
	loginPayload2 := models.LoginRequest{
		Username: "otheruser",
		Password: "password987",
	}
	body, _ = json.Marshal(loginPayload2)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	var loginResp2 struct {
		Data models.LoginResponse `json:"data"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &loginResp2)
	tokenUser2 := loginResp2.Data.Token

	// User 2 lists devices: should be empty
	req = httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
	req.Header.Set("Authorization", "Bearer "+tokenUser2)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	var listResp struct {
		Data []*models.Device `json:"data"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &listResp)
	if len(listResp.Data) != 0 {
		t.Errorf("expected 0 devices for user 2, got %d", len(listResp.Data))
	}

	// User 1 lists devices: should see 1 device
	req = httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
	req.Header.Set("Authorization", "Bearer "+tokenUser1)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	_ = json.Unmarshal(rr.Body.Bytes(), &listResp)
	if len(listResp.Data) != 1 {
		t.Errorf("expected 1 device for user 1, got %d", len(listResp.Data))
	}

	// Wait briefly for asynchronous action logs
	time.Sleep(50 * time.Millisecond)

	// Query Action Logs
	req = httptest.NewRequest(http.MethodGet, "/api/v1/logs", nil)
	req.Header.Set("Authorization", "Bearer "+tokenUser1)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 querying logs, got %d", rr.Code)
	}
}
