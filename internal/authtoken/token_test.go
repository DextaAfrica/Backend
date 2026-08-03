package authtoken

import (
	"testing"
	"time"
)

func TestGenerateAndParse_RoundTrip(t *testing.T) {
	secret := []byte("test-secret-key-thats-long-enough")

	token, err := Generate(secret, "admin-123", "admin@example.com", time.Hour)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	claims, err := Parse(secret, token)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if claims.AdminID != "admin-123" {
		t.Errorf("AdminID = %q, want %q", claims.AdminID, "admin-123")
	}
	if claims.Email != "admin@example.com" {
		t.Errorf("Email = %q, want %q", claims.Email, "admin@example.com")
	}
}

func TestParse_RejectsExpiredToken(t *testing.T) {
	secret := []byte("test-secret-key-thats-long-enough")

	token, err := Generate(secret, "admin-123", "admin@example.com", -time.Hour)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if _, err := Parse(secret, token); err == nil {
		t.Fatal("expected Parse() to reject an expired token")
	}
}

func TestParse_RejectsWrongSecret(t *testing.T) {
	token, err := Generate([]byte("secret-one-thats-long-enough-ok"), "admin-123", "admin@example.com", time.Hour)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if _, err := Parse([]byte("secret-two-thats-also-long-enough"), token); err == nil {
		t.Fatal("expected Parse() to reject a token signed with a different secret")
	}
}

func TestParse_RejectsGarbageToken(t *testing.T) {
	if _, err := Parse([]byte("any-secret"), "not.a.token"); err == nil {
		t.Fatal("expected Parse() to reject a malformed token")
	}
}
