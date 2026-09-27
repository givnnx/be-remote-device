package models

import "time"

type LogStatus string

const (
	LogStatusSuccess LogStatus = "SUCCESS"
	LogStatusFailed  LogStatus = "FAILED"
	LogStatusError   LogStatus = "ERROR"
)

type ActionLog struct {
	ID           string    `json:"id"`
	Timestamp    time.Time `json:"timestamp"`
	Actor        string    `json:"actor"`         // Pelaku: username, user ID, API key, device ID, atau IP
	Action       string    `json:"action"`        // Nama aksi (e.g. CREATE_DEVICE, SEND_COMMAND, HEARTBEAT)
	Method       string    `json:"method"`        // GET, POST, PUT, DELETE
	Path         string    `json:"path"`          // URL / Endpoint
	ClientIP     string    `json:"client_ip"`     // IP address pelaku request
	UserAgent    string    `json:"user_agent"`    // Browser / client agent
	StatusCode   int       `json:"status_code"`   // HTTP status code
	DurationMs   int64     `json:"duration_ms"`   // Waktu eksekusi dalam millisecond
	RequestBody  string    `json:"request_body"`  // Payload request (vital info)
	ResponseBody string    `json:"response_body"` // Response atau potongan data balikan
	ErrorMessage string    `json:"error_message"` // Pesan error jika request gagal
	Status       LogStatus `json:"status"`        // SUCCESS, FAILED, ERROR
}

type ActionLogFilter struct {
	Actor  string
	Action string
	Status string
	Limit  int
	Offset int
}
