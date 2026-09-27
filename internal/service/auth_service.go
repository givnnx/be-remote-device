package service

import (
	"errors"
	"time"

	"be-remote-device/internal/config"
	"be-remote-device/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrInvalidToken       = errors.New("invalid or expired token")
)

type AuthService interface {
	Login(req models.LoginRequest) (*models.LoginResponse, error)
	ValidateToken(tokenStr string) (*models.JWTClaims, error)
	ValidateAPIKey(apiKey string) bool
}

type authService struct {
	cfg *config.Config
}

func NewAuthService(cfg *config.Config) AuthService {
	return &authService{cfg: cfg}
}

func (s *authService) Login(req models.LoginRequest) (*models.LoginResponse, error) {
	// 1. Verify username
	if req.Username != s.cfg.AdminUsername {
		return nil, ErrInvalidCredentials
	}

	// 2. Verify password (support both bcrypt hash and plain text fallback)
	passwordMatch := false
	if err := bcrypt.CompareHashAndPassword([]byte(s.cfg.AdminPassword), []byte(req.Password)); err == nil {
		passwordMatch = true
	} else if req.Password == s.cfg.AdminPassword {
		passwordMatch = true
	}

	if !passwordMatch {
		return nil, ErrInvalidCredentials
	}

	// 3. Generate JWT token
	expirationDuration := 24 * time.Hour
	expiresAt := time.Now().Add(expirationDuration)

	claims := &models.JWTClaims{
		Username: req.Username,
		Role:     "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    s.cfg.AppName,
			Subject:   req.Username,
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
		Username:  req.Username,
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

func (s *authService) ValidateAPIKey(apiKey string) bool {
	if s.cfg.MasterAPIKey == "" || apiKey == "" {
		return false
	}
	return s.cfg.MasterAPIKey == apiKey
}
