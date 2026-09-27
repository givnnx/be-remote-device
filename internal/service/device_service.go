package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"be-remote-device/internal/models"
	"be-remote-device/internal/repository"
)

type DeviceService interface {
	GetAllDevices() ([]*models.Device, error)
	GetDeviceByID(id string) (*models.Device, error)
	CreateDevice(req models.CreateDeviceRequest) (*models.Device, error)
	UpdateDevice(id string, req models.UpdateDeviceRequest) (*models.Device, error)
	DeleteDevice(id string) error
	RecordHeartbeat(deviceID string) error

	SendCommand(deviceID string, req models.SendCommandRequest) (*models.RemoteCommand, error)
	GetDeviceCommands(deviceID string) ([]*models.RemoteCommand, error)
	UpdateCommandExecution(cmdID string, status models.CommandStatus, result string) (*models.RemoteCommand, error)

	RecordTelemetry(data models.TelemetryData) error
	GetLatestTelemetry(deviceID string) (*models.TelemetryData, error)
}

type deviceService struct {
	repo repository.DeviceRepository
}

func NewDeviceService(repo repository.DeviceRepository) DeviceService {
	return &deviceService{repo: repo}
}

func generateID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return prefix + "-" + hex.EncodeToString(b)
}

func (s *deviceService) GetAllDevices() ([]*models.Device, error) {
	return s.repo.GetAll()
}

func (s *deviceService) GetDeviceByID(id string) (*models.Device, error) {
	return s.repo.GetByID(id)
}

func (s *deviceService) CreateDevice(req models.CreateDeviceRequest) (*models.Device, error) {
	if req.Name == "" {
		return nil, errors.New("device name is required")
	}

	now := time.Now().UTC()
	dev := &models.Device{
		ID:         generateID("dev"),
		Name:       req.Name,
		Type:       req.Type,
		IPAddress:  req.IPAddress,
		MACAddress: req.MACAddress,
		Status:     models.StatusOffline,
		LastSeen:   now,
		Metadata:   req.Metadata,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	return s.repo.Create(dev)
}

func (s *deviceService) UpdateDevice(id string, req models.UpdateDeviceRequest) (*models.Device, error) {
	return s.repo.Update(id, &req)
}

func (s *deviceService) DeleteDevice(id string) error {
	return s.repo.Delete(id)
}

func (s *deviceService) RecordHeartbeat(deviceID string) error {
	return s.repo.UpdateLastSeen(deviceID)
}

func (s *deviceService) SendCommand(deviceID string, req models.SendCommandRequest) (*models.RemoteCommand, error) {
	// Check if device exists
	_, err := s.repo.GetByID(deviceID)
	if err != nil {
		return nil, err
	}

	if req.Payload == "" {
		return nil, errors.New("command payload is required")
	}

	cmd := &models.RemoteCommand{
		ID:        generateID("cmd"),
		DeviceID:  deviceID,
		Payload:   req.Payload,
		Status:    models.CommandPending,
		CreatedAt: time.Now().UTC(),
	}

	return s.repo.SaveCommand(cmd)
}

func (s *deviceService) GetDeviceCommands(deviceID string) ([]*models.RemoteCommand, error) {
	return s.repo.GetCommandsByDeviceID(deviceID)
}

func (s *deviceService) UpdateCommandExecution(cmdID string, status models.CommandStatus, result string) (*models.RemoteCommand, error) {
	return s.repo.UpdateCommandStatus(cmdID, status, result)
}

func (s *deviceService) RecordTelemetry(data models.TelemetryData) error {
	if data.Timestamp.IsZero() {
		data.Timestamp = time.Now().UTC()
	}
	// Also mark device as seen
	_ = s.repo.UpdateLastSeen(data.DeviceID)
	return s.repo.SaveTelemetry(&data)
}

func (s *deviceService) GetLatestTelemetry(deviceID string) (*models.TelemetryData, error) {
	return s.repo.GetLatestTelemetry(deviceID)
}
