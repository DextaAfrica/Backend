// Package authtoken is the single place that issues and verifies admin
// session JWTs. Both the auth service (issuing, on login) and the auth
// middleware (verifying, on every protected request) depend on this package
// instead of duplicating claim shapes or signing logic.
package authtoken

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("authtoken: invalid or expired token")

// Claims identifies the authenticated admin. Kept intentionally small: this
// is a single-role admin system today, so there is no roles/permissions
// array to keep in sync with a future RBAC model — that can be added here
// without touching callers, since they only ever read AdminID/Email.
type Claims struct {
	AdminID string `json:"admin_id"`
	Email   string `json:"email"`
	jwt.RegisteredClaims
}

func Generate(secret []byte, adminID, email string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		AdminID: adminID,
		Email:   email,
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
