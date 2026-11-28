package crypto

import (
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := DeriveKey("test-ultra-secure-key-32-chars-long-123456")
	input := "Sensitive PII: Applicant phone +2348012345678, signature SVG"

	encrypted, err := EncryptText(key, input)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	if encrypted == input {
		t.Fatal("encrypted text must not match plaintext")
	}

	decrypted, err := DecryptText(key, encrypted)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}

	if decrypted != input {
		t.Fatalf("expected '%s', got '%s'", input, decrypted)
	}
}

func TestDecryptTamperedCiphertext(t *testing.T) {
	key := DeriveKey("test-ultra-secure-key-32-chars-long-123456")
	input := "Secret banking details"

	encrypted, err := EncryptText(key, input)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	// Tamper with the ciphertext
	tampered := []byte(encrypted)
	tampered[len(tampered)-2] ^= 0xFF

	_, err = DecryptText(key, string(tampered))
	if err == nil {
		t.Fatal("expected decryption error on tampered ciphertext, got nil")
	}
}
