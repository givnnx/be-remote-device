package service

import (
	"testing"

	"be-remote-device/internal/models"
	"be-remote-device/internal/repository"
)

func TestDeviceService_Lifecycle(t *testing.T) {
	repo := repository.NewMemoryDeviceRepository()
	svc := NewDeviceService(repo)

	// Create
	created, err := svc.CreateDevice(models.CreateDeviceRequest{
		Name:      "Test-Device-01",
		Type:      "sensor",
		IPAddress: "192.168.1.50",
	})
	if err != nil {
		t.Fatalf("expected no error creating device, got: %v", err)
	}
	if created.ID == "" {
		t.Errorf("expected non-empty device ID")
	}

	// Read
	found, err := svc.GetDeviceByID(created.ID)
	if err != nil {
		t.Fatalf("expected to find device, got error: %v", err)
	}
	if found.Name != "Test-Device-01" {
		t.Errorf("expected name Test-Device-01, got %s", found.Name)
	}

	// Update
	newName := "Updated-Device-01"
	updated, err := svc.UpdateDevice(created.ID, models.UpdateDeviceRequest{
		Name: &newName,
	})
	if err != nil {
		t.Fatalf("expected update to succeed, got: %v", err)
	}
	if updated.Name != newName {
		t.Errorf("expected name to be %s, got %s", newName, updated.Name)
	}

	// Heartbeat
	if err := svc.RecordHeartbeat(created.ID); err != nil {
		t.Errorf("expected heartbeat to succeed, got: %v", err)
	}

	// Send Command
	cmd, err := svc.SendCommand(created.ID, models.SendCommandRequest{Payload: "reboot"})
	if err != nil {
		t.Fatalf("expected command dispatch to succeed, got: %v", err)
	}
	if cmd.Status != models.CommandPending {
		t.Errorf("expected pending status, got: %s", cmd.Status)
	}

	// Telemetry
	err = svc.RecordTelemetry(models.TelemetryData{
		DeviceID:    created.ID,
		CPUUsage:    15.5,
		MemoryUsage: 45.2,
		DiskUsage:   30.0,
	})
	if err != nil {
		t.Errorf("expected telemetry recording to succeed, got: %v", err)
	}

	telem, err := svc.GetLatestTelemetry(created.ID)
	if err != nil || telem == nil {
		t.Fatalf("expected to get telemetry, got: %v", err)
	}
	if telem.CPUUsage != 15.5 {
		t.Errorf("expected CPU 15.5, got %f", telem.CPUUsage)
	}

	// Delete
	if err := svc.DeleteDevice(created.ID); err != nil {
		t.Errorf("expected delete to succeed, got: %v", err)
	}

	_, err = svc.GetDeviceByID(created.ID)
	if err == nil {
		t.Errorf("expected error getting deleted device")
	}
}
