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
	GetAllDevices(userID string) ([]*models.Device, error)
	GetDeviceByID(userID, id string) (*models.Device, error)
	GetDeviceByAPIKey(apiKey string) (*models.Device, error)
	CreateDevice(userID string, req models.CreateDeviceRequest) (*models.Device, error)
	EnrollDevice(userID string, req models.EnrollDeviceRequest) (*models.Device, bool, error)
	UpdateDevice(userID, id string, req models.UpdateDeviceRequest) (*models.Device, error)
	DeleteDevice(userID, id string) error
	RecordHeartbeat(deviceID string) error

	SendCommand(userID, deviceID string, req models.SendCommandRequest) (*models.RemoteCommand, error)
	GetDeviceCommands(userID, deviceID string) ([]*models.RemoteCommand, error)
	UpdateCommandExecution(cmdID string, status models.CommandStatus, result string) (*models.RemoteCommand, error)

	RecordTelemetry(data models.TelemetryData) error
	GetLatestTelemetry(userID, deviceID string) (*models.TelemetryData, error)
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

func generateAPIKey() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return "devkey_" + hex.EncodeToString(b)
}

func (s *deviceService) GetAllDevices(userID string) ([]*models.Device, error) {
	return s.repo.GetAll(userID)
}

func (s *deviceService) GetDeviceByID(userID, id string) (*models.Device, error) {
	return s.repo.GetByID(userID, id)
}

func (s *deviceService) GetDeviceByAPIKey(apiKey string) (*models.Device, error) {
	return s.repo.GetByAPIKey(apiKey)
}

func (s *deviceService) CreateDevice(userID string, req models.CreateDeviceRequest) (*models.Device, error) {
	if req.Name == "" {
		return nil, errors.New("device name is required")
	}

	now := time.Now().UTC()
	dev := &models.Device{
		ID:         generateID("dev"),
		UserID:     userID,
		APIKey:     generateAPIKey(),
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

func (s *deviceService) EnrollDevice(userID string, req models.EnrollDeviceRequest) (*models.Device, bool, error) {
	if req.MachineID == "" {
		return nil, false, errors.New("machine_id is required for enrollment")
	}

	// 1. Check if device with this MachineID already exists for this user
	existing, err := s.repo.GetByMachineID(userID, req.MachineID)
	if err == nil && existing != nil {
		now := time.Now().UTC()
		existing.LastSeen = now
		existing.Status = models.StatusOnline
		if req.Name != "" {
			existing.Name = req.Name
		}
		if req.IPAddress != "" {
			existing.IPAddress = req.IPAddress
		}
		if req.MACAddress != "" {
			existing.MACAddress = req.MACAddress
		}
		existing.UpdatedAt = now
		return existing, false, nil // false = already enrolled
	}

	// 2. New enrollment
	now := time.Now().UTC()
	devName := req.Name
	if devName == "" {
		devName = "Device-" + req.MachineID
		if len(devName) > 16 {
			devName = devName[:16]
		}
	}
	devType := req.Type
	if devType == "" {
		devType = "laptop"
	}

	dev := &models.Device{
		ID:         generateID("dev"),
		UserID:     userID,
		MachineID:  req.MachineID,
		APIKey:     generateAPIKey(),
		Name:       devName,
		Type:       devType,
		IPAddress:  req.IPAddress,
		MACAddress: req.MACAddress,
		Status:     models.StatusOnline,
		LastSeen:   now,
		Metadata:   req.Metadata,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	created, err := s.repo.Create(dev)
	if err != nil {
		return nil, false, err
	}

	return created, true, nil // true = newly created
}

func (s *deviceService) UpdateDevice(userID, id string, req models.UpdateDeviceRequest) (*models.Device, error) {
	return s.repo.Update(userID, id, &req)
}

func (s *deviceService) DeleteDevice(userID, id string) error {
	return s.repo.Delete(userID, id)
}

func (s *deviceService) RecordHeartbeat(deviceID string) error {
	return s.repo.UpdateLastSeen(deviceID)
}

func (s *deviceService) SendCommand(userID, deviceID string, req models.SendCommandRequest) (*models.RemoteCommand, error) {
	// Verify device belongs to user
	_, err := s.repo.GetByID(userID, deviceID)
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

func (s *deviceService) GetDeviceCommands(userID, deviceID string) ([]*models.RemoteCommand, error) {
	// Verify ownership
	_, err := s.repo.GetByID(userID, deviceID)
	if err != nil {
		return nil, err
	}
	return s.repo.GetCommandsByDeviceID(deviceID)
}

func (s *deviceService) UpdateCommandExecution(cmdID string, status models.CommandStatus, result string) (*models.RemoteCommand, error) {
	return s.repo.UpdateCommandStatus(cmdID, status, result)
}

func (s *deviceService) RecordTelemetry(data models.TelemetryData) error {
	if data.Timestamp.IsZero() {
		data.Timestamp = time.Now().UTC()
	}
	// Mark device as seen
	_ = s.repo.UpdateLastSeen(data.DeviceID)
	return s.repo.SaveTelemetry(&data)
}

func (s *deviceService) GetLatestTelemetry(userID, deviceID string) (*models.TelemetryData, error) {
	// Verify ownership
	_, err := s.repo.GetByID(userID, deviceID)
	if err != nil {
		return nil, err
	}
	return s.repo.GetLatestTelemetry(deviceID)
}
