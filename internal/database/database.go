package database

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"be-remote-device/internal/config"

	_ "github.com/lib/pq"
)

func InitDB(cfg *config.Config) (*sql.DB, error) {
	hosts := []string{cfg.DBHost}
	if lower := strings.ToLower(cfg.DBHost); lower != cfg.DBHost {
		hosts = append(hosts, lower)
	}

	var lastErr error
	maxAttempts := 5

	for _, host := range hosts {
		dsn := fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s connect_timeout=5",
			host, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode,
		)

		for attempt := 1; attempt <= maxAttempts; attempt++ {
			// Auto create database if it doesn't exist yet
			if err := ensureDatabaseExists(host, cfg); err != nil {
				slog.Debug("Database auto-create check skipped or failed", "error", err)
			}

			db, err := sql.Open("postgres", dsn)
			if err == nil {
				db.SetMaxOpenConns(25)
				db.SetMaxIdleConns(5)
				db.SetConnMaxLifetime(5 * time.Minute)

				if pingErr := db.Ping(); pingErr == nil {
					slog.Info("Connected to PostgreSQL database successfully", "host", host, "dbname", cfg.DBName)
					if err := migrate(db); err != nil {
						_ = db.Close()
						return nil, fmt.Errorf("migration failed: %w", err)
					}
					return db, nil
				} else {
					lastErr = pingErr
					_ = db.Close()
				}
			} else {
				lastErr = err
			}

			if attempt < maxAttempts {
				slog.Warn("Waiting for database connection...", "host", host, "attempt", attempt, "max", maxAttempts, "error", lastErr)
				time.Sleep(2 * time.Second)
			}
		}
	}

	return nil, fmt.Errorf("failed to ping db: %w", lastErr)
}

func ensureDatabaseExists(host string, cfg *config.Config) error {
	if cfg.DBName == "" || cfg.DBName == "postgres" {
		return nil
	}

	defaultDSN := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=postgres sslmode=%s connect_timeout=5",
		host, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBSSLMode,
	)

	adminDB, err := sql.Open("postgres", defaultDSN)
	if err != nil {
		return err
	}
	defer adminDB.Close()

	if err := adminDB.Ping(); err != nil {
		return err
	}

	var exists bool
	checkQuery := "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)"
	if err := adminDB.QueryRow(checkQuery, cfg.DBName).Scan(&exists); err != nil {
		return err
	}

	if !exists {
		slog.Info("Database does not exist yet. Auto-creating database...", "dbname", cfg.DBName)
		safeDBName := strings.ReplaceAll(cfg.DBName, `"`, `""`)
		createQuery := fmt.Sprintf(`CREATE DATABASE "%s"`, safeDBName)
		if _, err := adminDB.Exec(createQuery); err != nil {
			return fmt.Errorf("failed to auto-create database %s: %w", cfg.DBName, err)
		}
		slog.Info("Database auto-created successfully", "dbname", cfg.DBName)
	}

	return nil
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
