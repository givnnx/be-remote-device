package service

import (
	"testing"
	"time"

	"be-remote-device/internal/models"
	"be-remote-device/internal/repository"
)

func TestActionLogService(t *testing.T) {
	repo := repository.NewMemoryActionLogRepository()
	svc := NewActionLogService(repo)
	defer svc.Stop()

	// 1. Log a success action
	svc.LogAction(&models.ActionLog{
		Actor:       "operator-1",
		Action:      "SEND_COMMAND",
		Method:      "POST",
		Path:        "/api/v1/devices/dev-123/commands",
		ClientIP:    "127.0.0.1",
		StatusCode:  202,
		DurationMs:  12,
		RequestBody: `{"payload":"restart"}`,
		Status:      models.LogStatusSuccess,
	})

	// 2. Log a failed action
	svc.LogAction(&models.ActionLog{
		Actor:        "operator-2",
		Action:       "DELETE_DEVICE",
		Method:       "DELETE",
		Path:         "/api/v1/devices/dev-999",
		ClientIP:     "192.168.1.10",
		StatusCode:   404,
		DurationMs:   5,
		ErrorMessage: "Device not found",
		Status:       models.LogStatusFailed,
	})

	// Wait for async queue
	time.Sleep(50 * time.Millisecond)

	// Test Query
	logs, total, err := svc.GetLogs(models.ActionLogFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Errorf("expected 2 logs, got %d", total)
	}

	// Filter by Actor
	logs, total, err = svc.GetLogs(models.ActionLogFilter{Actor: "operator-1"})
	if err != nil || total != 1 {
		t.Fatalf("expected 1 log for operator-1, got %d", total)
	}
	if logs[0].Action != "SEND_COMMAND" {
		t.Errorf("expected SEND_COMMAND, got %s", logs[0].Action)
	}

	// Filter by Status
	logs, total, err = svc.GetLogs(models.ActionLogFilter{Status: "FAILED"})
	if err != nil || total != 1 {
		t.Fatalf("expected 1 failed log, got %d", total)
	}
	if logs[0].ErrorMessage != "Device not found" {
		t.Errorf("expected error message 'Device not found', got '%s'", logs[0].ErrorMessage)
	}
}
