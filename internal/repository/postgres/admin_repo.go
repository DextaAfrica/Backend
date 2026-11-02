package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DextaAfrica/Backend/internal/domain"
)

type AdminRepository struct {
	pool *pgxpool.Pool
}

func NewAdminRepository(pool *pgxpool.Pool) *AdminRepository {
	return &AdminRepository{pool: pool}
}

func (r *AdminRepository) Create(ctx context.Context, admin *domain.Admin) error {
	const q = `
		INSERT INTO admins (email, password_hash)
		VALUES ($1, $2)
		RETURNING id, created_at, updated_at`

	err := r.pool.QueryRow(ctx, q, admin.Email, admin.PasswordHash).
		Scan(&admin.ID, &admin.CreatedAt, &admin.UpdatedAt)
	if err != nil {
		return fmt.Errorf("admin_repo: create: %w", err)
	}
	return nil
}

func (r *AdminRepository) GetByEmail(ctx context.Context, email string) (*domain.Admin, error) {
	const q = `
		SELECT id, email, password_hash, created_at, updated_at
		FROM admins
		WHERE email = $1`

	a := &domain.Admin{}
	err := r.pool.QueryRow(ctx, q, email).
		Scan(&a.ID, &a.Email, &a.PasswordHash, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("admin_repo: get by email: %w", err)
	}
	return a, nil
}

func (r *AdminRepository) Count(ctx context.Context) (int, error) {
	var count int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM admins`).Scan(&count); err != nil {
		return 0, fmt.Errorf("admin_repo: count: %w", err)
	}
	return count, nil
}
