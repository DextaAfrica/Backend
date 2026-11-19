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
		INSERT INTO admins (email, password_hash, full_name, role, department, job_title, phone, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`

	err := r.pool.QueryRow(ctx, q, admin.Email, admin.PasswordHash, admin.FullName, admin.Role, admin.Department, admin.JobTitle, admin.Phone, admin.Status).
		Scan(&admin.ID, &admin.CreatedAt, &admin.UpdatedAt)
	if err != nil {
		return fmt.Errorf("admin_repo: create: %w", err)
	}
	return nil
}

func (r *AdminRepository) Upsert(ctx context.Context, admin *domain.Admin) error {
	const q = `
		INSERT INTO admins (email, password_hash, full_name, role, department, job_title, phone, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (email) DO UPDATE SET
			full_name = EXCLUDED.full_name,
			role = EXCLUDED.role,
			department = EXCLUDED.department,
			job_title = EXCLUDED.job_title,
			phone = EXCLUDED.phone,
			status = EXCLUDED.status,
			updated_at = now()
		RETURNING id, created_at, updated_at`

	err := r.pool.QueryRow(ctx, q, admin.Email, admin.PasswordHash, admin.FullName, admin.Role, admin.Department, admin.JobTitle, admin.Phone, admin.Status).
		Scan(&admin.ID, &admin.CreatedAt, &admin.UpdatedAt)
	if err != nil {
		return fmt.Errorf("admin_repo: upsert: %w", err)
	}
	return nil
}

const adminColumns = `id, email, password_hash, full_name, role, department, job_title, phone, status, created_at, updated_at`

func (r *AdminRepository) GetByEmail(ctx context.Context, email string) (*domain.Admin, error) {
	q := `SELECT ` + adminColumns + ` FROM admins WHERE email = $1`

	a := &domain.Admin{}
	err := r.pool.QueryRow(ctx, q, email).
		Scan(&a.ID, &a.Email, &a.PasswordHash, &a.FullName, &a.Role, &a.Department, &a.JobTitle, &a.Phone, &a.Status, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("admin_repo: get by email: %w", err)
	}
	return a, nil
}

func (r *AdminRepository) GetByID(ctx context.Context, id string) (*domain.Admin, error) {
	q := `SELECT ` + adminColumns + ` FROM admins WHERE id = $1`

	a := &domain.Admin{}
	err := r.pool.QueryRow(ctx, q, id).
		Scan(&a.ID, &a.Email, &a.PasswordHash, &a.FullName, &a.Role, &a.Department, &a.JobTitle, &a.Phone, &a.Status, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("admin_repo: get by id: %w", err)
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

func (r *AdminRepository) List(ctx context.Context, params domain.ListParams) (domain.PageResult[domain.Admin], error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM admins`).Scan(&total); err != nil {
		return domain.PageResult[domain.Admin]{}, fmt.Errorf("admin_repo: count: %w", err)
	}

	q := `SELECT ` + adminColumns + ` FROM admins
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.pool.Query(ctx, q, params.PerPage, params.Offset())
	if err != nil {
		return domain.PageResult[domain.Admin]{}, fmt.Errorf("admin_repo: list: %w", err)
	}
	defer rows.Close()

	var items []domain.Admin
	for rows.Next() {
		var a domain.Admin
		if err := rows.Scan(&a.ID, &a.Email, &a.PasswordHash, &a.FullName, &a.Role, &a.Department, &a.JobTitle, &a.Phone, &a.Status, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return domain.PageResult[domain.Admin]{}, fmt.Errorf("admin_repo: scan: %w", err)
		}
		items = append(items, a)
	}
	if err := rows.Err(); err != nil {
		return domain.PageResult[domain.Admin]{}, fmt.Errorf("admin_repo: rows: %w", err)
	}
	return domain.PageResult[domain.Admin]{
		Items:      items,
		TotalItems: total,
		Page:       params.Page,
		PerPage:    params.PerPage,
	}, nil
}
