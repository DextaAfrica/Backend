package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DextaAfrica/Backend/internal/domain"
)

type ProductionApprovalRepository struct {
	pool *pgxpool.Pool
}

func NewProductionApprovalRepository(pool *pgxpool.Pool) *ProductionApprovalRepository {
	return &ProductionApprovalRepository{pool: pool}
}

const prodReleaseColumns = `
	id, ticket_number, title, target_system, stage, status, admin_id, admin_name, admin_email,
	risk_level, justification, rollback_plan, diff_summary, super_admin_id, super_admin_name,
	super_admin_email, super_admin_notes, approved_at, created_at, updated_at`

func scanProdRelease(row pgx.Row, r *domain.ProductionReleaseRequest) error {
	return row.Scan(
		&r.ID,
		&r.TicketNumber,
		&r.Title,
		&r.TargetSystem,
		&r.Stage,
		&r.Status,
		&r.AdminID,
		&r.AdminName,
		&r.AdminEmail,
		&r.RiskLevel,
		&r.Justification,
		&r.RollBackPlan,
		&r.DiffSummary,
		&r.SuperAdminID,
		&r.SuperAdminName,
		&r.SuperAdminEmail,
		&r.SuperAdminNotes,
		&r.ApprovedAt,
		&r.CreatedAt,
		&r.UpdatedAt,
	)
}

func (repo *ProductionApprovalRepository) Create(ctx context.Context, req *domain.ProductionReleaseRequest) error {
	if req.ID == "" {
		req.ID = uuid.New().String()
	}
	if req.Stage <= 0 {
		req.Stage = 1
	}
	if req.Status == "" {
		req.Status = domain.ProductionStatusPendingSuperAdmin
	}
	if req.RiskLevel == "" {
		req.RiskLevel = domain.ProductionRiskMedium
	}

	const q = `
		INSERT INTO production_release_requests (
			id, ticket_number, title, target_system, stage, status, admin_id, admin_name, admin_email,
			risk_level, justification, rollback_plan, diff_summary, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, now(), now())
		RETURNING created_at, updated_at`

	err := repo.pool.QueryRow(ctx, q,
		req.ID,
		req.TicketNumber,
		req.Title,
		req.TargetSystem,
		req.Stage,
		req.Status,
		req.AdminID,
		req.AdminName,
		req.AdminEmail,
		req.RiskLevel,
		req.Justification,
		req.RollBackPlan,
		req.DiffSummary,
	).Scan(&req.CreatedAt, &req.UpdatedAt)

	if err != nil {
		return fmt.Errorf("production_approval_repo: create: %w", err)
	}
	return nil
}

func (repo *ProductionApprovalRepository) GetByID(ctx context.Context, id string) (*domain.ProductionReleaseRequest, error) {
	const q = `SELECT ` + prodReleaseColumns + ` FROM production_release_requests WHERE id = $1`
	row := repo.pool.QueryRow(ctx, q, id)

	r := &domain.ProductionReleaseRequest{}
	if err := scanProdRelease(row, r); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("production_approval_repo: get by id: %w", err)
	}
	return r, nil
}

func (repo *ProductionApprovalRepository) List(ctx context.Context, params domain.ListParams) (domain.PageResult[domain.ProductionReleaseRequest], error) {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PerPage <= 0 || params.PerPage > 100 {
		params.PerPage = 20
	}

	var total int
	err := repo.pool.QueryRow(ctx, `SELECT count(*) FROM production_release_requests`).Scan(&total)
	if err != nil {
		return domain.PageResult[domain.ProductionReleaseRequest]{}, fmt.Errorf("production_approval_repo: count: %w", err)
	}

	const q = `SELECT ` + prodReleaseColumns + ` FROM production_release_requests ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := repo.pool.Query(ctx, q, params.PerPage, params.Offset())
	if err != nil {
		return domain.PageResult[domain.ProductionReleaseRequest]{}, fmt.Errorf("production_approval_repo: list: %w", err)
	}
	defer rows.Close()

	items := make([]domain.ProductionReleaseRequest, 0)
	for rows.Next() {
		var item domain.ProductionReleaseRequest
		if err := scanProdRelease(rows, &item); err != nil {
			return domain.PageResult[domain.ProductionReleaseRequest]{}, fmt.Errorf("production_approval_repo: scan: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return domain.PageResult[domain.ProductionReleaseRequest]{}, fmt.Errorf("production_approval_repo: rows err: %w", err)
	}

	return domain.PageResult[domain.ProductionReleaseRequest]{
		Items:      items,
		TotalItems: total,
		Page:       params.Page,
		PerPage:    params.PerPage,
	}, nil
}

func (repo *ProductionApprovalRepository) GetStats(ctx context.Context) (*domain.ProductionReleaseStats, error) {
	const q = `
		SELECT 
			COALESCE(COUNT(*) FILTER (WHERE status = 'PENDING_SUPER_ADMIN'), 0) AS pending_count,
			COALESCE(COUNT(*) FILTER (WHERE status = 'APPROVED_LIVE'), 0) AS approved_count,
			COALESCE(COUNT(*) FILTER (WHERE status = 'REJECTED'), 0) AS rejected_count,
			COUNT(*) AS total_count
		FROM production_release_requests`

	stats := &domain.ProductionReleaseStats{}
	err := repo.pool.QueryRow(ctx, q).Scan(
		&stats.PendingCount,
		&stats.ApprovedCount,
		&stats.RejectedCount,
		&stats.TotalCount,
	)
	if err != nil {
		return nil, fmt.Errorf("production_approval_repo: get stats: %w", err)
	}
	return stats, nil
}

func (repo *ProductionApprovalRepository) UpdateDecision(
	ctx context.Context,
	id string,
	status domain.ProductionReleaseStatus,
	stage int,
	superAdminID, superAdminName, superAdminEmail string,
	notes string,
) (*domain.ProductionReleaseRequest, error) {
	var approvedAt *time.Time
	if status == domain.ProductionStatusApprovedLive {
		now := time.Now().UTC()
		approvedAt = &now
	}

	const q = `
		UPDATE production_release_requests
		SET status = $2,
			stage = $3,
			super_admin_id = $4,
			super_admin_name = $5,
			super_admin_email = $6,
			super_admin_notes = $7,
			approved_at = COALESCE($8, approved_at),
			updated_at = now()
		WHERE id = $1
		RETURNING ` + prodReleaseColumns

	row := repo.pool.QueryRow(ctx, q,
		id,
		status,
		stage,
		superAdminID,
		superAdminName,
		superAdminEmail,
		notes,
		approvedAt,
	)

	updated := &domain.ProductionReleaseRequest{}
	if err := scanProdRelease(row, updated); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("production_approval_repo: update decision: %w", err)
	}
	return updated, nil
}
