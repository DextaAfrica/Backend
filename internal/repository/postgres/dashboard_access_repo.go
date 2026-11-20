package postgres

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DextaAfrica/Backend/internal/domain"
)

type DashboardAccessRepository struct {
	pool *pgxpool.Pool
}

func NewDashboardAccessRepository(pool *pgxpool.Pool) *DashboardAccessRepository {
	return &DashboardAccessRepository{pool: pool}
}

const dashAccessColumns = `
	id, ticket_number, user_id, requester_name, requester_email, department, job_title,
	requested_modules, justification, signature_data, signature_type, signed_at, status,
	reviewed_by_id, approved_at, approved_modules, it_notes, created_at, updated_at`

func scanDashAccess(row pgx.Row, r *domain.DashboardAccessRequest) error {
	var sigType string
	var status string
	err := row.Scan(
		&r.ID,
		&r.TicketNumber,
		&r.UserID,
		&r.RequesterName,
		&r.RequesterEmail,
		&r.Department,
		&r.JobTitle,
		&r.RequestedModules,
		&r.Justification,
		&r.SignatureData,
		&sigType,
		&r.SignedAt,
		&status,
		&r.ReviewedByID,
		&r.ApprovedAt,
		&r.ApprovedModules,
		&r.ITNotes,
		&r.CreatedAt,
		&r.UpdatedAt,
	)
	if err != nil {
		return err
	}
	r.SignatureType = domain.DigitalSignatureType(sigType)
	r.Status = domain.DashboardRequestStatus(status)
	return nil
}

func generateTicketNumber() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("DAR-%s-%s", time.Now().UTC().Format("20060102"), hex.EncodeToString(b))
}

func (repo *DashboardAccessRepository) Create(ctx context.Context, req *domain.DashboardAccessRequest) error {
	if req.ID == "" {
		req.ID = uuid.New().String()
	}
	if req.TicketNumber == "" {
		req.TicketNumber = generateTicketNumber()
	}
	if req.Status == "" {
		req.Status = domain.DashboardStatusPendingITApproval
	}
	if req.SignatureType == "" {
		req.SignatureType = domain.SignatureTypeDrawn
	}
	if req.SignedAt.IsZero() {
		req.SignedAt = time.Now().UTC()
	}

	const query = `
		INSERT INTO dashboard_access_requests (
			id, ticket_number, user_id, requester_name, requester_email, department, job_title,
			requested_modules, justification, signature_data, signature_type, signed_at, status,
			reviewed_by_id, approved_at, approved_modules, it_notes, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12, $13,
			$14, $15, $16, $17, now(), now()
		)
		RETURNING created_at, updated_at`

	err := repo.pool.QueryRow(ctx, query,
		req.ID,
		req.TicketNumber,
		req.UserID,
		req.RequesterName,
		req.RequesterEmail,
		req.Department,
		req.JobTitle,
		req.RequestedModules,
		req.Justification,
		req.SignatureData,
		string(req.SignatureType),
		req.SignedAt,
		string(req.Status),
		req.ReviewedByID,
		req.ApprovedAt,
		req.ApprovedModules,
		req.ITNotes,
	).Scan(&req.CreatedAt, &req.UpdatedAt)

	if err != nil {
		return fmt.Errorf("dashboard_access_repo: create: %w", err)
	}
	return nil
}

func (repo *DashboardAccessRepository) GetByID(ctx context.Context, id string) (*domain.DashboardAccessRequest, error) {
	query := fmt.Sprintf(`SELECT %s FROM dashboard_access_requests WHERE id = $1`, dashAccessColumns)
	var item domain.DashboardAccessRequest
	err := scanDashAccess(repo.pool.QueryRow(ctx, query, id), &item)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("dashboard_access_repo: get by id: %w", err)
	}
	return &item, nil
}

func (repo *DashboardAccessRepository) GetByTicket(ctx context.Context, ticket string) (*domain.DashboardAccessRequest, error) {
	query := fmt.Sprintf(`SELECT %s FROM dashboard_access_requests WHERE ticket_number = $1`, dashAccessColumns)
	var item domain.DashboardAccessRequest
	err := scanDashAccess(repo.pool.QueryRow(ctx, query, ticket), &item)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("dashboard_access_repo: get by ticket: %w", err)
	}
	return &item, nil
}

func (repo *DashboardAccessRepository) List(ctx context.Context, status *domain.DashboardRequestStatus) ([]*domain.DashboardAccessRequest, error) {
	var rows pgx.Rows
	var err error

	if status != nil && *status != "" {
		query := fmt.Sprintf(`SELECT %s FROM dashboard_access_requests WHERE status = $1 ORDER BY created_at DESC`, dashAccessColumns)
		rows, err = repo.pool.Query(ctx, query, string(*status))
	} else {
		query := fmt.Sprintf(`SELECT %s FROM dashboard_access_requests ORDER BY created_at DESC`, dashAccessColumns)
		rows, err = repo.pool.Query(ctx, query)
	}

	if err != nil {
		return nil, fmt.Errorf("dashboard_access_repo: list: %w", err)
	}
	defer rows.Close()

	items := make([]*domain.DashboardAccessRequest, 0)
	for rows.Next() {
		var item domain.DashboardAccessRequest
		if err := scanDashAccess(rows, &item); err != nil {
			return nil, fmt.Errorf("dashboard_access_repo: scan list: %w", err)
		}
		items = append(items, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dashboard_access_repo: list rows: %w", err)
	}
	return items, nil
}

func (repo *DashboardAccessRepository) ListByUser(ctx context.Context, userID string) ([]*domain.DashboardAccessRequest, error) {
	query := fmt.Sprintf(`SELECT %s FROM dashboard_access_requests WHERE user_id = $1 ORDER BY created_at DESC`, dashAccessColumns)
	rows, err := repo.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("dashboard_access_repo: list by user: %w", err)
	}
	defer rows.Close()

	items := make([]*domain.DashboardAccessRequest, 0)
	for rows.Next() {
		var item domain.DashboardAccessRequest
		if err := scanDashAccess(rows, &item); err != nil {
			return nil, fmt.Errorf("dashboard_access_repo: scan user list: %w", err)
		}
		items = append(items, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dashboard_access_repo: list by user rows: %w", err)
	}
	return items, nil
}

func (repo *DashboardAccessRepository) Update(ctx context.Context, req *domain.DashboardAccessRequest) error {
	const query = `
		UPDATE dashboard_access_requests
		SET status = $1,
		    reviewed_by_id = $2,
		    approved_at = $3,
		    approved_modules = $4,
		    it_notes = $5,
		    updated_at = now()
		WHERE id = $6
		RETURNING updated_at`

	err := repo.pool.QueryRow(ctx, query,
		string(req.Status),
		req.ReviewedByID,
		req.ApprovedAt,
		req.ApprovedModules,
		req.ITNotes,
		req.ID,
	).Scan(&req.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("dashboard_access_repo: update: %w", err)
	}
	return nil
}
