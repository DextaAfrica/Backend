package domain

import (
	"context"
	"time"
)

const (
	RoleSuperAdmin = "SUPER_ADMIN"
	RoleAdmin      = "ADMIN"
)

type Admin struct {
	ID                  string     `json:"id"`
	Email               string     `json:"email"`
	PasswordHash        string     `json:"-"`
	FullName            string     `json:"fullName"`
	Role                string     `json:"role"`
	Department          string     `json:"department"`
	JobTitle            string     `json:"jobTitle"`
	Phone               string     `json:"phone"`
	Status              string     `json:"status"`
	AssignedModules     string     `json:"assignedModules"`
	SignatureURL        *string    `json:"signatureUrl,omitempty"`
	FailedLoginAttempts int        `json:"failedLoginAttempts"`
	LockedUntil         *time.Time `json:"lockedUntil,omitempty"`
	LastLoginAt         *time.Time `json:"lastLoginAt,omitempty"`
	LastLoginIP         string     `json:"lastLoginIp,omitempty"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
}

type AdminRepository interface {
	Create(ctx context.Context, admin *Admin) error
	GetByEmail(ctx context.Context, email string) (*Admin, error)
	GetByID(ctx context.Context, id string) (*Admin, error)
	Count(ctx context.Context) (int, error)
	List(ctx context.Context, params ListParams) (PageResult[Admin], error)
	Upsert(ctx context.Context, admin *Admin) error
	UpdateLoginSecurity(ctx context.Context, id string, failedAttempts int, lockedUntil *time.Time, lastLoginAt *time.Time, lastLoginIP string) error
}
