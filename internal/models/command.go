package models

import "time"

type CommandStatus string

const (
	CommandPending   CommandStatus = "pending"
	CommandExecuting CommandStatus = "executing"
	CommandSuccess   CommandStatus = "success"
	CommandFailed    CommandStatus = "failed"
)

type RemoteCommand struct {
	ID          string        `json:"id"`
	DeviceID    string        `json:"device_id"`
	Payload     string        `json:"payload"`      // e.g. "reboot", "restart_service", "update_firmware"
	Status      CommandStatus `json:"status"`
	Result      string        `json:"result,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	ExecutedAt  *time.Time    `json:"executed_at,omitempty"`
}

type SendCommandRequest struct {
	Payload string `json:"payload"`
}
