package repository

import (
	"errors"
	"sync"
	"time"

	"be-remote-device/internal/models"
)

var (
	ErrDeviceNotFound  = errors.New("device not found")
	ErrCommandNotFound = errors.New("command not found")
)

type DeviceRepository interface {
	GetAll(userID string) ([]*models.Device, error)
	GetByID(userID, id string) (*models.Device, error)
	GetByMachineID(userID, machineID string) (*models.Device, error)
	GetByAPIKey(apiKey string) (*models.Device, error)
	Create(dev *models.Device) (*models.Device, error)
	Update(userID, id string, req *models.UpdateDeviceRequest) (*models.Device, error)
	Delete(userID, id string) error
	UpdateStatus(id string, status models.DeviceStatus) error
	UpdateLastSeen(id string) error

	SaveCommand(cmd *models.RemoteCommand) (*models.RemoteCommand, error)
	GetCommandsByDeviceID(deviceID string) ([]*models.RemoteCommand, error)
	GetCommandByID(id string) (*models.RemoteCommand, error)
	UpdateCommandStatus(id string, status models.CommandStatus, result string) (*models.RemoteCommand, error)

	SaveTelemetry(t *models.TelemetryData) error
	GetLatestTelemetry(deviceID string) (*models.TelemetryData, error)
}

type memoryDeviceRepository struct {
	mu          sync.RWMutex
	devices     map[string]*models.Device
	commands    map[string]*models.RemoteCommand
	telemetries map[string]*models.TelemetryData
}

func NewMemoryDeviceRepository() DeviceRepository {
	return &memoryDeviceRepository{
		devices:     make(map[string]*models.Device),
		commands:    make(map[string]*models.RemoteCommand),
		telemetries: make(map[string]*models.TelemetryData),
	}
}

func (r *memoryDeviceRepository) GetAll(userID string) ([]*models.Device, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]*models.Device, 0)
	for _, dev := range r.devices {
		if dev.UserID == userID {
			list = append(list, dev)
		}
	}
	return list, nil
}

func (r *memoryDeviceRepository) GetByID(userID, id string) (*models.Device, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	dev, exists := r.devices[id]
	if !exists || (userID != "" && dev.UserID != userID) {
		return nil, ErrDeviceNotFound
	}
	return dev, nil
}

func (r *memoryDeviceRepository) GetByMachineID(userID, machineID string) (*models.Device, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if machineID == "" {
		return nil, ErrDeviceNotFound
	}

	for _, dev := range r.devices {
		if dev.UserID == userID && dev.MachineID == machineID {
			return dev, nil
		}
	}
	return nil, ErrDeviceNotFound
}

func (r *memoryDeviceRepository) GetByAPIKey(apiKey string) (*models.Device, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if apiKey == "" {
		return nil, ErrDeviceNotFound
	}

	for _, dev := range r.devices {
		if dev.APIKey == apiKey {
			return dev, nil
		}
	}
	return nil, ErrDeviceNotFound
}

func (r *memoryDeviceRepository) Create(dev *models.Device) (*models.Device, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.devices[dev.ID] = dev
	return dev, nil
}

func (r *memoryDeviceRepository) Update(userID, id string, req *models.UpdateDeviceRequest) (*models.Device, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	dev, exists := r.devices[id]
	if !exists || (userID != "" && dev.UserID != userID) {
		return nil, ErrDeviceNotFound
	}

	if req.Name != nil {
		dev.Name = *req.Name
	}
	if req.Type != nil {
		dev.Type = *req.Type
	}
	if req.IPAddress != nil {
		dev.IPAddress = *req.IPAddress
	}
	if req.MACAddress != nil {
		dev.MACAddress = *req.MACAddress
	}
	if req.Status != nil {
		dev.Status = *req.Status
	}
	if req.Metadata != nil {
		dev.Metadata = req.Metadata
	}
	dev.UpdatedAt = time.Now().UTC()

	return dev, nil
}

func (r *memoryDeviceRepository) Delete(userID, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	dev, exists := r.devices[id]
	if !exists || (userID != "" && dev.UserID != userID) {
		return ErrDeviceNotFound
	}
	delete(r.devices, id)
	return nil
}

func (r *memoryDeviceRepository) UpdateStatus(id string, status models.DeviceStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	dev, exists := r.devices[id]
	if !exists {
		return ErrDeviceNotFound
	}
	dev.Status = status
	dev.UpdatedAt = time.Now().UTC()
	return nil
}

func (r *memoryDeviceRepository) UpdateLastSeen(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	dev, exists := r.devices[id]
	if !exists {
		return ErrDeviceNotFound
	}
	dev.LastSeen = time.Now().UTC()
	dev.Status = models.StatusOnline
	dev.UpdatedAt = dev.LastSeen
	return nil
}

func (r *memoryDeviceRepository) SaveCommand(cmd *models.RemoteCommand) (*models.RemoteCommand, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.commands[cmd.ID] = cmd
	return cmd, nil
}

func (r *memoryDeviceRepository) GetCommandsByDeviceID(deviceID string) ([]*models.RemoteCommand, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]*models.RemoteCommand, 0)
	for _, cmd := range r.commands {
		if cmd.DeviceID == deviceID {
			list = append(list, cmd)
		}
	}
	return list, nil
}

func (r *memoryDeviceRepository) GetCommandByID(id string) (*models.RemoteCommand, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cmd, exists := r.commands[id]
	if !exists {
		return nil, ErrCommandNotFound
	}
	return cmd, nil
}

func (r *memoryDeviceRepository) UpdateCommandStatus(id string, status models.CommandStatus, result string) (*models.RemoteCommand, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	cmd, exists := r.commands[id]
	if !exists {
		return nil, ErrCommandNotFound
	}

	cmd.Status = status
	cmd.Result = result
	now := time.Now().UTC()
	cmd.ExecutedAt = &now

	return cmd, nil
}

func (r *memoryDeviceRepository) SaveTelemetry(t *models.TelemetryData) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.telemetries[t.DeviceID] = t
	return nil
}

func (r *memoryDeviceRepository) GetLatestTelemetry(deviceID string) (*models.TelemetryData, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	t, exists := r.telemetries[deviceID]
	if !exists {
		return nil, nil
	}
	return t, nil
}
