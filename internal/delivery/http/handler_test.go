package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"be-remote-device/internal/models"
	"be-remote-device/internal/repository"
	"be-remote-device/internal/service"
)

func TestDeviceHandler_EndpointsAndAuditLogging(t *testing.T) {
	deviceRepo := repository.NewMemoryDeviceRepository()
	deviceSvc := service.NewDeviceService(deviceRepo)
	deviceHandler := NewDeviceHandler(deviceSvc)

	logRepo := repository.NewMemoryActionLogRepository()
	logSvc := service.NewActionLogService(logRepo)
	defer logSvc.Stop()

	logHandler := NewLogHandler(logSvc)
	router := NewRouter(deviceHandler, logHandler, logSvc)

	// 1. Health check (health check should not be logged to avoid log bloat)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	// 2. Create Device with custom Actor header
	payload := models.CreateDeviceRequest{
		Name:      "Edge-Device-001",
		Type:      "iot-gateway",
		IPAddress: "192.168.1.10",
	}
	body, _ := json.Marshal(payload)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/devices", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Actor", "admin-user-42")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d, body: %s", rr.Code, rr.Body.String())
	}

	// Wait briefly for asynchronous worker to persist the action log
	time.Sleep(50 * time.Millisecond)

	// 3. Query Action Logs
	req = httptest.NewRequest(http.MethodGet, "/api/v1/logs?actor=admin-user-42", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var logResp struct {
		Success bool `json:"success"`
		Data    struct {
			Total int                 `json:"total"`
			Items []*models.ActionLog `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &logResp); err != nil {
		t.Fatalf("failed to decode log response: %v", err)
	}

	if logResp.Data.Total != 1 {
		t.Fatalf("expected 1 logged action, got %d", logResp.Data.Total)
	}

	loggedItem := logResp.Data.Items[0]
	if loggedItem.Actor != "admin-user-42" {
		t.Errorf("expected actor 'admin-user-42', got '%s'", loggedItem.Actor)
	}
	if loggedItem.Action != "REGISTER_DEVICE" {
		t.Errorf("expected action 'REGISTER_DEVICE', got '%s'", loggedItem.Action)
	}
	if loggedItem.Status != models.LogStatusSuccess {
		t.Errorf("expected status 'SUCCESS', got '%s'", loggedItem.Status)
	}
	if loggedItem.StatusCode != http.StatusCreated {
		t.Errorf("expected status code 201, got %d", loggedItem.StatusCode)
	}
}
