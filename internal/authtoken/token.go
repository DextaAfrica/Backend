// Package authtoken is the single place that issues and verifies admin
// session JWTs and cryptographically secure refresh tokens.
package authtoken

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("authtoken: invalid or expired token")

type contextKey string

const claimsContextKey contextKey = "authtoken_claims"

// ContextWithClaims stores parsed claims in context.
func ContextWithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey, claims)
}

// ClaimsFromContext extracts parsed claims from context if present.
func ClaimsFromContext(ctx context.Context) *Claims {
	if ctx == nil {
		return nil
	}
	claims, _ := ctx.Value(claimsContextKey).(*Claims)
	return claims
}

// Claims identifies the authenticated admin with role and module permissions.
type Claims struct {
	AdminID         string   `json:"admin_id"`
	Email           string   `json:"email"`
	Role            string   `json:"role,omitempty"`
	AssignedModules []string `json:"modules,omitempty"`
	SessionID       string   `json:"session_id,omitempty"`
	jwt.RegisteredClaims
}

// Generate creates a standard access JWT (compatible with legacy callers).
func Generate(secret []byte, adminID, email string, ttl time.Duration) (string, error) {
	return GenerateWithDetails(secret, adminID, email, "ADMIN", []string{"operations"}, "", ttl)
}

// GenerateWithDetails creates an enterprise access JWT with RBAC and session linkage.
func GenerateWithDetails(secret []byte, adminID, email, role string, modules []string, sessionID string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		AdminID:         adminID,
		Email:           email,
		Role:            role,
		AssignedModules: modules,
		SessionID:       sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			Subject:   adminID,
			Issuer:    "dexta-backend",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(secret)
	if err != nil {
		return "", fmt.Errorf("authtoken: sign token: %w", err)
	}
	return signed, nil
}

// GenerateRefreshToken creates a 32-byte cryptographically secure random token
// and returns both the raw token (to return to client) and its SHA-256 hash (to store in DB).
func GenerateRefreshToken() (rawToken string, tokenHash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("authtoken: generate refresh bytes: %w", err)
	}
	rawToken = hex.EncodeToString(b)
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash = hex.EncodeToString(hash[:])
	return rawToken, tokenHash, nil
}

// HashRefreshToken calculates the SHA-256 hash of a raw refresh token.
func HashRefreshToken(rawToken string) string {
	hash := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(hash[:])
}

// Parse validates and extracts claims from an access JWT.
func Parse(secret []byte, tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("authtoken: unexpected signing method %v", t.Header["alg"])
		}
		return secret, nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
