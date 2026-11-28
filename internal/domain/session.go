package domain

import (
	"context"
	"time"
)

type AdminSession struct {
	ID        string    `json:"id"`
	AdminID   string    `json:"adminId"`
	TokenHash string    `json:"-"`
	IPAddress string    `json:"ipAddress"`
	UserAgent string    `json:"userAgent"`
	Revoked   bool      `json:"revoked"`
	ExpiresAt time.Time `json:"expiresAt"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type SessionRepository interface {
	Create(ctx context.Context, session *AdminSession) error
	GetByTokenHash(ctx context.Context, hash string) (*AdminSession, error)
	Revoke(ctx context.Context, id string) error
	RevokeAllForAdmin(ctx context.Context, adminID string) error
	PruneExpired(ctx context.Context) (int64, error)
}
