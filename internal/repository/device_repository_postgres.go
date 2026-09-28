package repository

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"be-remote-device/internal/models"
)

type postgresDeviceRepository struct {
	db *sql.DB
}

func NewPostgresDeviceRepository(db *sql.DB) DeviceRepository {
	return &postgresDeviceRepository{db: db}
}

func (r *postgresDeviceRepository) GetAll(userID string) ([]*models.Device, error) {
	query := `
		SELECT id, user_id, COALESCE(machine_id, ''), api_key, name, type, 
		       COALESCE(ip_address, ''), COALESCE(mac_address, ''), status, 
		       last_seen, metadata, created_at, updated_at
		FROM devices
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	devices := make([]*models.Device, 0)
	for rows.Next() {
		var dev models.Device
		var metaJSON []byte
		err := rows.Scan(
			&dev.ID, &dev.UserID, &dev.MachineID, &dev.APIKey, &dev.Name, &dev.Type,
			&dev.IPAddress, &dev.MACAddress, &dev.Status, &dev.LastSeen, &metaJSON,
			&dev.CreatedAt, &dev.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		if len(metaJSON) > 0 {
			_ = json.Unmarshal(metaJSON, &dev.Metadata)
		}
		devices = append(devices, &dev)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return devices, nil
}

func (r *postgresDeviceRepository) GetByID(userID, id string) (*models.Device, error) {
	query := `
		SELECT id, user_id, COALESCE(machine_id, ''), api_key, name, type, 
		       COALESCE(ip_address, ''), COALESCE(mac_address, ''), status, 
		       last_seen, metadata, created_at, updated_at
		FROM devices
		WHERE user_id = $1 AND id = $2
	`
	var dev models.Device
	var metaJSON []byte
	err := r.db.QueryRow(query, userID, id).Scan(
		&dev.ID, &dev.UserID, &dev.MachineID, &dev.APIKey, &dev.Name, &dev.Type,
		&dev.IPAddress, &dev.MACAddress, &dev.Status, &dev.LastSeen, &metaJSON,
		&dev.CreatedAt, &dev.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDeviceNotFound
		}
		return nil, err
	}
	if len(metaJSON) > 0 {
		_ = json.Unmarshal(metaJSON, &dev.Metadata)
	}
	return &dev, nil
}

func (r *postgresDeviceRepository) GetByMachineID(userID, machineID string) (*models.Device, error) {
	if machineID == "" {
		return nil, ErrDeviceNotFound
	}

	query := `
		SELECT id, user_id, COALESCE(machine_id, ''), api_key, name, type, 
		       COALESCE(ip_address, ''), COALESCE(mac_address, ''), status, 
		       last_seen, metadata, created_at, updated_at
		FROM devices
		WHERE user_id = $1 AND machine_id = $2
	`
	var dev models.Device
	var metaJSON []byte
	err := r.db.QueryRow(query, userID, machineID).Scan(
		&dev.ID, &dev.UserID, &dev.MachineID, &dev.APIKey, &dev.Name, &dev.Type,
		&dev.IPAddress, &dev.MACAddress, &dev.Status, &dev.LastSeen, &metaJSON,
		&dev.CreatedAt, &dev.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDeviceNotFound
		}
		return nil, err
	}
	if len(metaJSON) > 0 {
		_ = json.Unmarshal(metaJSON, &dev.Metadata)
	}
	return &dev, nil
}

func (r *postgresDeviceRepository) GetByAPIKey(apiKey string) (*models.Device, error) {
	query := `
		SELECT id, user_id, COALESCE(machine_id, ''), api_key, name, type, 
		       COALESCE(ip_address, ''), COALESCE(mac_address, ''), status, 
		       last_seen, metadata, created_at, updated_at
		FROM devices
		WHERE api_key = $1
	`
	var dev models.Device
	var metaJSON []byte
	err := r.db.QueryRow(query, apiKey).Scan(
		&dev.ID, &dev.UserID, &dev.MachineID, &dev.APIKey, &dev.Name, &dev.Type,
		&dev.IPAddress, &dev.MACAddress, &dev.Status, &dev.LastSeen, &metaJSON,
		&dev.CreatedAt, &dev.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDeviceNotFound
		}
		return nil, err
	}
	if len(metaJSON) > 0 {
		_ = json.Unmarshal(metaJSON, &dev.Metadata)
	}
	return &dev, nil
}

func (r *postgresDeviceRepository) Create(dev *models.Device) (*models.Device, error) {
	var metaBytes []byte
	if dev.Metadata != nil {
		var err error
		metaBytes, err = json.Marshal(dev.Metadata)
		if err != nil {
			return nil, err
		}
	}

	query := `
		INSERT INTO devices (
			id, user_id, machine_id, api_key, name, type, 
			ip_address, mac_address, status, last_seen, metadata, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := r.db.Exec(
		query,
		dev.ID, dev.UserID, dev.MachineID, dev.APIKey, dev.Name, dev.Type,
		dev.IPAddress, dev.MACAddress, dev.Status, dev.LastSeen, metaBytes,
		dev.CreatedAt, dev.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return dev, nil
}

func (r *postgresDeviceRepository) Update(userID, id string, req *models.UpdateDeviceRequest) (*models.Device, error) {
	if _, err := r.GetByID(userID, id); err != nil {
		return nil, err
	}

	setClauses := []string{"updated_at = $1"}
	args := []interface{}{time.Now().UTC()}
	argIdx := 2

	if req.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
	if req.Type != nil {
		setClauses = append(setClauses, fmt.Sprintf("type = $%d", argIdx))
		args = append(args, *req.Type)
		argIdx++
	}
	if req.IPAddress != nil {
		setClauses = append(setClauses, fmt.Sprintf("ip_address = $%d", argIdx))
		args = append(args, *req.IPAddress)
		argIdx++
	}
	if req.MACAddress != nil {
		setClauses = append(setClauses, fmt.Sprintf("mac_address = $%d", argIdx))
		args = append(args, *req.MACAddress)
		argIdx++
	}
	if req.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *req.Status)
		argIdx++
	}
	if req.Metadata != nil {
		metaBytes, err := json.Marshal(req.Metadata)
		if err != nil {
			return nil, err
		}
		setClauses = append(setClauses, fmt.Sprintf("metadata = $%d", argIdx))
		args = append(args, metaBytes)
		argIdx++
	}

	args = append(args, userID, id)
	query := fmt.Sprintf(
		"UPDATE devices SET %s WHERE user_id = $%d AND id = $%d",
		strings.Join(setClauses, ", "),
		argIdx,
		argIdx+1,
	)

	_, err := r.db.Exec(query, args...)
	if err != nil {
		return nil, err
	}

	return r.GetByID(userID, id)
}

func (r *postgresDeviceRepository) Delete(userID, id string) error {
	query := "DELETE FROM devices WHERE user_id = $1 AND id = $2"
	res, err := r.db.Exec(query, userID, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrDeviceNotFound
	}
	return nil
}

func (r *postgresDeviceRepository) UpdateStatus(id string, status models.DeviceStatus) error {
	query := "UPDATE devices SET status = $1, updated_at = $2 WHERE id = $3"
	res, err := r.db.Exec(query, status, time.Now().UTC(), id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrDeviceNotFound
	}
	return nil
}

func (r *postgresDeviceRepository) UpdateLastSeen(id string) error {
	now := time.Now().UTC()
	query := "UPDATE devices SET last_seen = $1, status = $2, updated_at = $1 WHERE id = $3"
	res, err := r.db.Exec(query, now, models.StatusOnline, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrDeviceNotFound
	}
	return nil
}

func (r *postgresDeviceRepository) SaveCommand(cmd *models.RemoteCommand) (*models.RemoteCommand, error) {
	query := `
		INSERT INTO remote_commands (id, device_id, payload, status, result, created_at, executed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.Exec(
		query,
		cmd.ID, cmd.DeviceID, cmd.Payload, cmd.Status, cmd.Result, cmd.CreatedAt, cmd.ExecutedAt,
	)
	if err != nil {
		return nil, err
	}
	return cmd, nil
}

func (r *postgresDeviceRepository) GetCommandsByDeviceID(deviceID string) ([]*models.RemoteCommand, error) {
	query := `
		SELECT id, device_id, payload, status, COALESCE(result, ''), created_at, executed_at
		FROM remote_commands
		WHERE device_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(query, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	commands := make([]*models.RemoteCommand, 0)
	for rows.Next() {
		var cmd models.RemoteCommand
		err := rows.Scan(
			&cmd.ID, &cmd.DeviceID, &cmd.Payload, &cmd.Status, &cmd.Result,
			&cmd.CreatedAt, &cmd.ExecutedAt,
		)
		if err != nil {
			return nil, err
		}
		commands = append(commands, &cmd)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return commands, nil
}

func (r *postgresDeviceRepository) GetCommandByID(id string) (*models.RemoteCommand, error) {
	query := `
		SELECT id, device_id, payload, status, COALESCE(result, ''), created_at, executed_at
		FROM remote_commands
		WHERE id = $1
	`
	var cmd models.RemoteCommand
	err := r.db.QueryRow(query, id).Scan(
		&cmd.ID, &cmd.DeviceID, &cmd.Payload, &cmd.Status, &cmd.Result,
		&cmd.CreatedAt, &cmd.ExecutedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCommandNotFound
		}
		return nil, err
	}
	return &cmd, nil
}

func (r *postgresDeviceRepository) UpdateCommandStatus(id string, status models.CommandStatus, result string) (*models.RemoteCommand, error) {
	now := time.Now().UTC()
	query := `
		UPDATE remote_commands
		SET status = $1, result = $2, executed_at = $3
		WHERE id = $4
	`
	res, err := r.db.Exec(query, status, result, now, id)
	if err != nil {
		return nil, err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		return nil, ErrCommandNotFound
	}

	return r.GetCommandByID(id)
}

func (r *postgresDeviceRepository) SaveTelemetry(t *models.TelemetryData) error {
	query := `
		INSERT INTO telemetries (
			device_id, cpu_usage_pct, memory_usage_pct, disk_usage_pct, 
			battery_level, temperature_celsius, timestamp
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.Exec(
		query,
		t.DeviceID, t.CPUUsage, t.MemoryUsage, t.DiskUsage,
		t.BatteryLevel, t.Temperature, t.Timestamp,
	)
	return err
}

func (r *postgresDeviceRepository) GetLatestTelemetry(deviceID string) (*models.TelemetryData, error) {
	query := `
		SELECT device_id, cpu_usage_pct, memory_usage_pct, disk_usage_pct, 
		       battery_level, temperature_celsius, timestamp
		FROM telemetries
		WHERE device_id = $1
		ORDER BY timestamp DESC
		LIMIT 1
	`
	var t models.TelemetryData
	err := r.db.QueryRow(query, deviceID).Scan(
		&t.DeviceID, &t.CPUUsage, &t.MemoryUsage, &t.DiskUsage,
		&t.BatteryLevel, &t.Temperature, &t.Timestamp,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}
