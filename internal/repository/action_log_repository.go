package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"

	"be-remote-device/internal/models"
)

type ActionLogRepository interface {
	Save(log *models.ActionLog) error
	GetAll(filter models.ActionLogFilter) ([]*models.ActionLog, int64, error)
	GetByID(id string) (*models.ActionLog, error)
}

// --- Postgres Implementation ---

type postgresActionLogRepository struct {
	db *sql.DB
}

func NewPostgresActionLogRepository(db *sql.DB) ActionLogRepository {
	return &postgresActionLogRepository{db: db}
}

func (r *postgresActionLogRepository) Save(log *models.ActionLog) error {
	query := `
		INSERT INTO action_logs (
			id, timestamp, actor, action, method, path, client_ip, 
			user_agent, status_code, duration_ms, request_body, response_body, error_message, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	_, err := r.db.Exec(
		query,
		log.ID, log.Timestamp, log.Actor, log.Action, log.Method, log.Path, log.ClientIP,
		log.UserAgent, log.StatusCode, log.DurationMs, log.RequestBody, log.ResponseBody, log.ErrorMessage, log.Status,
	)
	return err
}

func (r *postgresActionLogRepository) GetAll(filter models.ActionLogFilter) ([]*models.ActionLog, int64, error) {
	var conditions []string
	var args []interface{}
	idx := 1

	if filter.Actor != "" {
		conditions = append(conditions, fmt.Sprintf("actor ILIKE $%d", idx))
		args = append(args, "%"+filter.Actor+"%")
		idx++
	}
	if filter.Action != "" {
		conditions = append(conditions, fmt.Sprintf("action ILIKE $%d", idx))
		args = append(args, "%"+filter.Action+"%")
		idx++
	}
	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", idx))
		args = append(args, filter.Status)
		idx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM action_logs %s", whereClause)
	var total int64
	if err := r.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Query data
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	dataQuery := fmt.Sprintf(`
		SELECT id, timestamp, actor, action, method, path, client_ip, 
		       user_agent, status_code, duration_ms, request_body, response_body, error_message, status
		FROM action_logs
		%s
		ORDER BY timestamp DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, idx, idx+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []*models.ActionLog
	for rows.Next() {
		var l models.ActionLog
		if err := rows.Scan(
			&l.ID, &l.Timestamp, &l.Actor, &l.Action, &l.Method, &l.Path, &l.ClientIP,
			&l.UserAgent, &l.StatusCode, &l.DurationMs, &l.RequestBody, &l.ResponseBody, &l.ErrorMessage, &l.Status,
		); err != nil {
			return nil, 0, err
		}
		logs = append(logs, &l)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

func (r *postgresActionLogRepository) GetByID(id string) (*models.ActionLog, error) {
	query := `
		SELECT id, timestamp, actor, action, method, path, client_ip, 
		       user_agent, status_code, duration_ms, request_body, response_body, error_message, status
		FROM action_logs
		WHERE id = $1
	`
	var l models.ActionLog
	err := r.db.QueryRow(query, id).Scan(
		&l.ID, &l.Timestamp, &l.Actor, &l.Action, &l.Method, &l.Path, &l.ClientIP,
		&l.UserAgent, &l.StatusCode, &l.DurationMs, &l.RequestBody, &l.ResponseBody, &l.ErrorMessage, &l.Status,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrDeviceNotFound
		}
		return nil, err
	}
	return &l, nil
}

// --- In-Memory Fallback Implementation ---

type memoryActionLogRepository struct {
	mu   sync.RWMutex
	logs []*models.ActionLog
}

func NewMemoryActionLogRepository() ActionLogRepository {
	return &memoryActionLogRepository{
		logs: make([]*models.ActionLog, 0),
	}
}

func (m *memoryActionLogRepository) Save(log *models.ActionLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append([]*models.ActionLog{log}, m.logs...) // prepend
	return nil
}

func (m *memoryActionLogRepository) GetAll(filter models.ActionLogFilter) ([]*models.ActionLog, int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var filtered []*models.ActionLog
	for _, l := range m.logs {
		if filter.Actor != "" && !strings.Contains(strings.ToLower(l.Actor), strings.ToLower(filter.Actor)) {
			continue
		}
		if filter.Action != "" && !strings.Contains(strings.ToLower(l.Action), strings.ToLower(filter.Action)) {
			continue
		}
		if filter.Status != "" && string(l.Status) != filter.Status {
			continue
		}
		filtered = append(filtered, l)
	}

	total := int64(len(filtered))
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	start := offset
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + limit
	if end > len(filtered) {
		end = len(filtered)
	}

	return filtered[start:end], total, nil
}

func (m *memoryActionLogRepository) GetByID(id string) (*models.ActionLog, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, l := range m.logs {
		if l.ID == id {
			return l, nil
		}
	}
	return nil, ErrDeviceNotFound
}
