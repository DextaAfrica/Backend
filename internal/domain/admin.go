package domain

import (
	"context"
	"time"
)

type Admin struct {
	ID           string
	Email        string
	PasswordHash string
	FullName     string
	Role         string
	Department   string
	JobTitle     string
	Phone        string
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type AdminRepository interface {
	Create(ctx context.Context, admin *Admin) error
	GetByEmail(ctx context.Context, email string) (*Admin, error)
	GetByID(ctx context.Context, id string) (*Admin, error)
	Count(ctx context.Context) (int, error)
	List(ctx context.Context, params ListParams) (PageResult[Admin], error)
	Upsert(ctx context.Context, admin *Admin) error
}
