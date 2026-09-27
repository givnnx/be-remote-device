package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"be-remote-device/internal/config"
	"be-remote-device/internal/models"
	"be-remote-device/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid username/email or password")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrRegistrationFailed = errors.New("failed to register user")
)

type AuthService interface {
	Register(req models.RegisterRequest) (*models.UserResponse, error)
	Login(req models.LoginRequest) (*models.LoginResponse, error)
	ValidateToken(tokenStr string) (*models.JWTClaims, error)
	GetUserByID(id string) (*models.UserResponse, error)
}

type authService struct {
	cfg      *config.Config
	userRepo repository.UserRepository
}

func NewAuthService(cfg *config.Config, userRepo repository.UserRepository) AuthService {
	return &authService{
		cfg:      cfg,
		userRepo: userRepo,
	}
}

func (s *authService) Register(req models.RegisterRequest) (*models.UserResponse, error) {
	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	if req.Username == "" || req.Email == "" || len(req.Password) < 6 {
		return nil, errors.New("username, email, and password (min 6 chars) are required")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, ErrRegistrationFailed
	}

	b := make([]byte, 8)
	_, _ = rand.Read(b)
	userID := "usr-" + hex.EncodeToString(b)
	now := time.Now().UTC()

	user := &models.User{
		ID:           userID,
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hash),
		FullName:     req.FullName,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	created, err := s.userRepo.Create(user)
	if err != nil {
		return nil, err
	}

	return &models.UserResponse{
		ID:        created.ID,
		Username:  created.Username,
		Email:     created.Email,
		FullName:  created.FullName,
		CreatedAt: created.CreatedAt,
	}, nil
}

func (s *authService) Login(req models.LoginRequest) (*models.LoginResponse, error) {
	identifier := strings.TrimSpace(req.Username)
	if identifier == "" || req.Password == "" {
		return nil, ErrInvalidCredentials
	}

	user, err := s.userRepo.GetByUsernameOrEmail(identifier)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	expirationDuration := 24 * 7 * time.Hour // 7 days
	expiresAt := time.Now().Add(expirationDuration)

	claims := &models.JWTClaims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     "user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    s.cfg.AppName,
			Subject:   user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, err
	}

	return &models.LoginResponse{
		Token:     tokenString,
		TokenType: "Bearer",
		ExpiresIn: int64(expirationDuration.Seconds()),
		User: &models.UserResponse{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			FullName:  user.FullName,
			CreatedAt: user.CreatedAt,
		},
	}, nil
}

func (s *authService) ValidateToken(tokenStr string) (*models.JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &models.JWTClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(s.cfg.JWTSecret), nil
	})

	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*models.JWTClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

func (s *authService) GetUserByID(id string) (*models.UserResponse, error) {
	u, err := s.userRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	return &models.UserResponse{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		FullName:  u.FullName,
		CreatedAt: u.CreatedAt,
	}, nil
}
