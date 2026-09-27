package service

import (
	"testing"

	"be-remote-device/internal/config"
	"be-remote-device/internal/models"
)

func TestAuthService(t *testing.T) {
	cfg := &config.Config{
		AppName:       "be-remote-device",
		JWTSecret:     "secret-jwt-key",
		MasterAPIKey:  "my-secret-api-key",
		AdminUsername: "admin",
		AdminPassword: "mypassword123",
	}

	authSvc := NewAuthService(cfg)

	// 1. Invalid login
	_, err := authSvc.Login(models.LoginRequest{Username: "admin", Password: "wrong"})
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}

	// 2. Successful login
	resp, err := authSvc.Login(models.LoginRequest{Username: "admin", Password: "mypassword123"})
	if err != nil {
		t.Fatalf("expected login to succeed, got %v", err)
	}
	if resp.Token == "" {
		t.Errorf("expected token not to be empty")
	}

	// 3. Validate Token
	claims, err := authSvc.ValidateToken(resp.Token)
	if err != nil {
		t.Fatalf("expected token to be valid, got %v", err)
	}
	if claims.Username != "admin" {
		t.Errorf("expected username admin, got %s", claims.Username)
	}

	// 4. Validate API Key
	if !authSvc.ValidateAPIKey("my-secret-api-key") {
		t.Errorf("expected valid API key")
	}
	if authSvc.ValidateAPIKey("invalid-key") {
		t.Errorf("expected invalid API key to fail")
	}
}
