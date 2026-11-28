package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

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

const adminColumns = `
	id, email, password_hash, full_name, role, department, job_title, phone, status,
	assigned_modules, signature_url, failed_login_attempts, locked_until, last_login_at,
	last_login_ip, created_at, updated_at`

func scanAdmin(row pgx.Row, a *domain.Admin) error {
	return row.Scan(
		&a.ID,
		&a.Email,
		&a.PasswordHash,
		&a.FullName,
		&a.Role,
		&a.Department,
		&a.JobTitle,
		&a.Phone,
		&a.Status,
		&a.AssignedModules,
		&a.SignatureURL,
		&a.FailedLoginAttempts,
		&a.LockedUntil,
		&a.LastLoginAt,
		&a.LastLoginIP,
		&a.CreatedAt,
		&a.UpdatedAt,
	)
}

func (r *AdminRepository) Create(ctx context.Context, admin *domain.Admin) error {
	if admin.AssignedModules == "" {
		admin.AssignedModules = `["operations","approvals","personal"]`
	}
	if admin.Role == "" {
		admin.Role = "ADMIN"
	}
	if admin.Status == "" {
		admin.Status = "ACTIVE"
	}

	const q = `
		INSERT INTO admins (
			email, password_hash, full_name, role, department, job_title, phone, status,
			assigned_modules, signature_url
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at`

	err := r.pool.QueryRow(ctx, q,
		admin.Email,
		admin.PasswordHash,
		admin.FullName,
		admin.Role,
		admin.Department,
		admin.JobTitle,
		admin.Phone,
		admin.Status,
		admin.AssignedModules,
		admin.SignatureURL,
	).Scan(&admin.ID, &admin.CreatedAt, &admin.UpdatedAt)
	if err != nil {
		return fmt.Errorf("admin_repo: create: %w", err)
	}
	return nil
}

func (r *AdminRepository) Upsert(ctx context.Context, admin *domain.Admin) error {
	if admin.AssignedModules == "" {
		admin.AssignedModules = `["operations","approvals","personal"]`
	}
	if admin.Role == "" {
		admin.Role = "ADMIN"
	}
	if admin.Status == "" {
		admin.Status = "ACTIVE"
	}

	const q = `
		INSERT INTO admins (
			email, password_hash, full_name, role, department, job_title, phone, status,
			assigned_modules, signature_url
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (email) DO UPDATE SET
			full_name = EXCLUDED.full_name,
			role = EXCLUDED.role,
			department = EXCLUDED.department,
			job_title = EXCLUDED.job_title,
			phone = EXCLUDED.phone,
			status = EXCLUDED.status,
			assigned_modules = CASE WHEN EXCLUDED.assigned_modules <> '' THEN EXCLUDED.assigned_modules ELSE admins.assigned_modules END,
			signature_url = COALESCE(EXCLUDED.signature_url, admins.signature_url),
			updated_at = now()
		RETURNING id, created_at, updated_at`

	err := r.pool.QueryRow(ctx, q,
		admin.Email,
		admin.PasswordHash,
		admin.FullName,
		admin.Role,
		admin.Department,
		admin.JobTitle,
		admin.Phone,
		admin.Status,
		admin.AssignedModules,
		admin.SignatureURL,
	).Scan(&admin.ID, &admin.CreatedAt, &admin.UpdatedAt)
	if err != nil {
		return fmt.Errorf("admin_repo: upsert: %w", err)
	}
	return nil
}

func (r *AdminRepository) GetByEmail(ctx context.Context, email string) (*domain.Admin, error) {
	q := `SELECT ` + adminColumns + ` FROM admins WHERE email = $1`

	a := &domain.Admin{}
	err := scanAdmin(r.pool.QueryRow(ctx, q, email), a)
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
	err := scanAdmin(r.pool.QueryRow(ctx, q, id), a)
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

	items := make([]domain.Admin, 0)
	for rows.Next() {
		var a domain.Admin
		if err := scanAdmin(rows, &a); err != nil {
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

func (r *AdminRepository) UpdateLoginSecurity(ctx context.Context, id string, failedAttempts int, lockedUntil *time.Time, lastLoginAt *time.Time, lastLoginIP string) error {
	const q = `
		UPDATE admins
		SET failed_login_attempts = $1,
		    locked_until = $2,
		    last_login_at = COALESCE($3, last_login_at),
		    last_login_ip = CASE WHEN $4 <> '' THEN $4 ELSE last_login_ip END,
		    updated_at = now()
		WHERE id = $5`
	_, err := r.pool.Exec(ctx, q, failedAttempts, lockedUntil, lastLoginAt, lastLoginIP, id)
	if err != nil {
		return fmt.Errorf("admin_repo: update login security: %w", err)
	}
	return nil
}
