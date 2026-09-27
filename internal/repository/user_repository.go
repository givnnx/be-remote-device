package repository

import (
	"database/sql"
	"errors"
	"strings"
	"sync"

	"be-remote-device/internal/models"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("username or email already in use")
)

type UserRepository interface {
	Create(user *models.User) (*models.User, error)
	GetByID(id string) (*models.User, error)
	GetByUsernameOrEmail(identifier string) (*models.User, error)
}

// --- Postgres Implementation ---

type postgresUserRepository struct {
	db *sql.DB
}

func NewPostgresUserRepository(db *sql.DB) UserRepository {
	return &postgresUserRepository{db: db}
}

func (r *postgresUserRepository) Create(u *models.User) (*models.User, error) {
	query := `
		INSERT INTO users (id, username, email, password_hash, full_name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.Exec(query, u.ID, u.Username, u.Email, u.PasswordHash, u.FullName, u.CreatedAt, u.UpdatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			return nil, ErrUserAlreadyExists
		}
		return nil, err
	}
	return u, nil
}

func (r *postgresUserRepository) GetByID(id string) (*models.User, error) {
	query := `
		SELECT id, username, email, password_hash, full_name, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	var u models.User
	err := r.db.QueryRow(query, id).Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.FullName, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *postgresUserRepository) GetByUsernameOrEmail(identifier string) (*models.User, error) {
	query := `
		SELECT id, username, email, password_hash, full_name, created_at, updated_at
		FROM users
		WHERE LOWER(username) = LOWER($1) OR LOWER(email) = LOWER($1)
		LIMIT 1
	`
	var u models.User
	err := r.db.QueryRow(query, identifier).Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.FullName, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

// --- In-Memory Implementation ---

type memoryUserRepository struct {
	mu    sync.RWMutex
	users map[string]*models.User
}

func NewMemoryUserRepository() UserRepository {
	return &memoryUserRepository{
		users: make(map[string]*models.User),
	}
}

func (m *memoryUserRepository) Create(u *models.User) (*models.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, existing := range m.users {
		if strings.EqualFold(existing.Username, u.Username) || strings.EqualFold(existing.Email, u.Email) {
			return nil, ErrUserAlreadyExists
		}
	}

	m.users[u.ID] = u
	return u, nil
}

func (m *memoryUserRepository) GetByID(id string) (*models.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	u, exists := m.users[id]
	if !exists {
		return nil, ErrUserNotFound
	}
	return u, nil
}

func (m *memoryUserRepository) GetByUsernameOrEmail(identifier string) (*models.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, u := range m.users {
		if strings.EqualFold(u.Username, identifier) || strings.EqualFold(u.Email, identifier) {
			return u, nil
		}
	}
	return nil, ErrUserNotFound
}
