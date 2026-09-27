package models

import "time"

type TelemetryData struct {
	DeviceID     string    `json:"device_id"`
	CPUUsage     float64   `json:"cpu_usage_pct"`
	MemoryUsage  float64   `json:"memory_usage_pct"`
	DiskUsage    float64   `json:"disk_usage_pct"`
	BatteryLevel *float64  `json:"battery_level,omitempty"`
	Temperature  *float64  `json:"temperature_celsius,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}
