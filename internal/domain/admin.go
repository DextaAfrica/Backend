package domain

import (
	"context"
	"time"
)

type Admin struct {
	ID           string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// AdminRepository is the persistence port for admin accounts. It has no
// Update/Delete/List beyond what login and bootstrap need — this is a
// single-role admin system by design (see ADR in docs/ARCHITECTURE.md);
// broader admin management is deliberately deferred, not an oversight.
type AdminRepository interface {
	Create(ctx context.Context, admin *Admin) error
	GetByEmail(ctx context.Context, email string) (*Admin, error)
	Count(ctx context.Context) (int, error)
}
