package models

import "time"

type DeviceStatus string

const (
	StatusOnline  DeviceStatus = "online"
	StatusOffline DeviceStatus = "offline"
	StatusBusy    DeviceStatus = "busy"
	StatusError   DeviceStatus = "error"
)

type Device struct {
	ID         string                 `json:"id"`
	UserID     string                 `json:"user_id"`
	MachineID  string                 `json:"machine_id,omitempty"` // Unique Hardware GUID / BIOS UUID
	APIKey     string                 `json:"api_key"`              // Secret key unik untuk background agent di laptop/smartphone
	Name       string                 `json:"name"`
	Type       string                 `json:"type"` // e.g. "laptop", "pc", "smartphone"
	IPAddress  string                 `json:"ip_address"`
	MACAddress string                 `json:"mac_address"`
	Status     DeviceStatus           `json:"status"`
	LastSeen   time.Time              `json:"last_seen"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

type CreateDeviceRequest struct {
	MachineID  string                 `json:"machine_id,omitempty"`
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	IPAddress  string                 `json:"ip_address"`
	MACAddress string                 `json:"mac_address"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

type EnrollDeviceRequest struct {
	MachineID  string                 `json:"machine_id"` // Hardware unique ID
	Name       string                 `json:"name"`       // Hostname
	Type       string                 `json:"type"`       // laptop or pc
	IPAddress  string                 `json:"ip_address"`
	MACAddress string                 `json:"mac_address"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

type UpdateDeviceRequest struct {
	Name       *string                `json:"name,omitempty"`
	Type       *string                `json:"type,omitempty"`
	IPAddress  *string                `json:"ip_address,omitempty"`
	MACAddress *string                `json:"mac_address,omitempty"`
	Status     *DeviceStatus          `json:"status,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}
