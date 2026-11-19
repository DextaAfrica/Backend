package authtoken_test

import (
	"testing"
	"time"

	"github.com/DextaAfrica/Backend/internal/authtoken"
)

func TestTokenGenerateAndParse(t *testing.T) {
	secret := []byte("test-super-secret-key-at-least-32-chars-long")
	adminID := "admin-123"
	email := "admin@dexta.africa"

	token, err := authtoken.Generate(secret, adminID, email, time.Hour)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	claims, err := authtoken.Parse(secret, token)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if claims.AdminID != adminID {
		t.Errorf("got AdminID %q, want %q", claims.AdminID, adminID)
	}
	if claims.Email != email {
		t.Errorf("got Email %q, want %q", claims.Email, email)
	}
}

func TestTokenExpired(t *testing.T) {
	secret := []byte("test-super-secret-key-at-least-32-chars-long")
	token, err := authtoken.Generate(secret, "admin-123", "admin@dexta.africa", -time.Hour)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	_, err = authtoken.Parse(secret, token)
	if err == nil {
		t.Fatal("Parse() expected error for expired token, got nil")
	}
}
