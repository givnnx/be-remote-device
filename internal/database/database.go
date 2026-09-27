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
