package service

import (
	"testing"

	"be-remote-device/internal/config"
	"be-remote-device/internal/models"
	"be-remote-device/internal/repository"
)

func TestAuthService_RegisterAndLogin(t *testing.T) {
	cfg := &config.Config{
		AppName:      "be-remote-device",
		JWTSecret:    "secret-jwt-key",
		MasterAPIKey: "my-secret-api-key",
	}

	userRepo := repository.NewMemoryUserRepository()
	authSvc := NewAuthService(cfg, userRepo)

	// 1. Register new user
	regResp, err := authSvc.Register(models.RegisterRequest{
		Username: "giovanni",
		Email:    "giovanni@example.com",
		Password: "strongpassword123",
		FullName: "Giovanni Agung",
	})
	if err != nil {
		t.Fatalf("expected registration to succeed, got: %v", err)
	}
	if regResp.ID == "" {
		t.Errorf("expected generated user ID")
	}

	// Duplicate registration should fail
	_, err = authSvc.Register(models.RegisterRequest{
		Username: "giovanni",
		Email:    "different@example.com",
		Password: "strongpassword123",
	})
	if err != repository.ErrUserAlreadyExists {
		t.Errorf("expected ErrUserAlreadyExists on duplicate username, got: %v", err)
	}

	// 2. Login with correct credentials (username)
	loginResp, err := authSvc.Login(models.LoginRequest{
		Username: "giovanni",
		Password: "strongpassword123",
	})
	if err != nil {
		t.Fatalf("expected login to succeed, got: %v", err)
	}
	if loginResp.Token == "" {
		t.Errorf("expected non-empty token")
	}

	// 3. Login with email
	loginWithEmailResp, err := authSvc.Login(models.LoginRequest{
		Username: "giovanni@example.com",
		Password: "strongpassword123",
	})
	if err != nil {
		t.Fatalf("expected login with email to succeed, got: %v", err)
	}
	if loginWithEmailResp.Token == "" {
		t.Errorf("expected non-empty token")
	}

	// 4. Login with invalid password
	_, err = authSvc.Login(models.LoginRequest{
		Username: "giovanni",
		Password: "wrongpassword",
	})
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got: %v", err)
	}

	// 5. Validate Token
	claims, err := authSvc.ValidateToken(loginResp.Token)
	if err != nil {
		t.Fatalf("expected token validation to succeed, got: %v", err)
	}
	if claims.UserID != regResp.ID {
		t.Errorf("expected claims UserID %s, got %s", regResp.ID, claims.UserID)
	}
}
