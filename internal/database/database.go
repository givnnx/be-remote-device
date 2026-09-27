package database

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"be-remote-device/internal/config"

	_ "github.com/lib/pq"
)

func InitDB(cfg *config.Config) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s connect_timeout=5",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open db connection: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}

	slog.Info("Connected to PostgreSQL database successfully", "host", cfg.DBHost, "dbname", cfg.DBName)

	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	return db, nil
}

func migrate(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id VARCHAR(64) PRIMARY KEY,
		username VARCHAR(100) UNIQUE NOT NULL,
		email VARCHAR(255) UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		full_name VARCHAR(255),
		created_at TIMESTAMPTZ NOT NULL,
		updated_at TIMESTAMPTZ NOT NULL
	);

	CREATE TABLE IF NOT EXISTS devices (
		id VARCHAR(64) PRIMARY KEY,
		user_id VARCHAR(64) NOT NULL,
		machine_id VARCHAR(128),
		api_key VARCHAR(128) UNIQUE NOT NULL,
		name VARCHAR(255) NOT NULL,
		type VARCHAR(100) NOT NULL,
		ip_address VARCHAR(50),
		mac_address VARCHAR(50),
		status VARCHAR(50) NOT NULL,
		last_seen TIMESTAMPTZ NOT NULL,
		metadata JSONB,
		created_at TIMESTAMPTZ NOT NULL,
		updated_at TIMESTAMPTZ NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_devices_user_id ON devices(user_id);
	CREATE INDEX IF NOT EXISTS idx_devices_machine_id ON devices(machine_id);
	CREATE INDEX IF NOT EXISTS idx_devices_api_key ON devices(api_key);

	CREATE TABLE IF NOT EXISTS remote_commands (
		id VARCHAR(64) PRIMARY KEY,
		device_id VARCHAR(64) NOT NULL,
		payload TEXT NOT NULL,
		status VARCHAR(50) NOT NULL,
		result TEXT,
		created_at TIMESTAMPTZ NOT NULL,
		executed_at TIMESTAMPTZ
	);
	CREATE INDEX IF NOT EXISTS idx_commands_device_id ON remote_commands(device_id);

	CREATE TABLE IF NOT EXISTS telemetries (
		id BIGSERIAL PRIMARY KEY,
		device_id VARCHAR(64) NOT NULL,
		cpu_usage_pct DOUBLE PRECISION NOT NULL,
		memory_usage_pct DOUBLE PRECISION NOT NULL,
		disk_usage_pct DOUBLE PRECISION NOT NULL,
		battery_level DOUBLE PRECISION,
		temperature_celsius DOUBLE PRECISION,
		timestamp TIMESTAMPTZ NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_telemetries_device_id ON telemetries(device_id);

	CREATE TABLE IF NOT EXISTS action_logs (
		id VARCHAR(64) PRIMARY KEY,
		timestamp TIMESTAMPTZ NOT NULL,
		actor VARCHAR(255) NOT NULL,
		action VARCHAR(100) NOT NULL,
		method VARCHAR(10) NOT NULL,
		path TEXT NOT NULL,
		client_ip VARCHAR(50) NOT NULL,
		user_agent TEXT,
		status_code INT NOT NULL,
		duration_ms BIGINT NOT NULL,
		request_body TEXT,
		response_body TEXT,
		error_message TEXT,
		status VARCHAR(20) NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_action_logs_timestamp ON action_logs(timestamp DESC);
	CREATE INDEX IF NOT EXISTS idx_action_logs_actor ON action_logs(actor);
	CREATE INDEX IF NOT EXISTS idx_action_logs_action ON action_logs(action);
	CREATE INDEX IF NOT EXISTS idx_action_logs_status ON action_logs(status);
	`

	_, err := db.Exec(query)
	if err != nil {
		return err
	}

	slog.Info("Database tables verified/migrated")
	return nil
}
