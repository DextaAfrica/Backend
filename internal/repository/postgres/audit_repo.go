package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DextaAfrica/Backend/internal/domain"
)

type AuditRepository struct {
	pool *pgxpool.Pool
}

func NewAuditRepository(pool *pgxpool.Pool) *AuditRepository {
	return &AuditRepository{pool: pool}
}

func (r *AuditRepository) Record(ctx context.Context, log *domain.AuditLog) error {
	metaJSON, err := json.Marshal(log.Metadata)
	if err != nil {
		metaJSON = []byte("{}")
	}

	const q = `
		INSERT INTO audit_logs (actor_id, actor_email, action, entity, entity_id, ip_address, user_agent, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at`

	return r.pool.QueryRow(ctx, q, log.ActorID, log.ActorEmail, log.Action, log.Entity, log.EntityID, log.IPAddress, log.UserAgent, metaJSON).
		Scan(&log.ID, &log.CreatedAt)
}

func (r *AuditRepository) List(ctx context.Context, params domain.ListParams) (domain.PageResult[domain.AuditLog], error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM audit_logs`).Scan(&total); err != nil {
		return domain.PageResult[domain.AuditLog]{}, fmt.Errorf("audit_repo: count: %w", err)
	}

	const q = `
		SELECT id, actor_id, actor_email, action, entity, entity_id, ip_address, user_agent, metadata, created_at
		FROM audit_logs
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.pool.Query(ctx, q, params.PerPage, params.Offset())
	if err != nil {
		return domain.PageResult[domain.AuditLog]{}, fmt.Errorf("audit_repo: list: %w", err)
	}
	defer rows.Close()

	var items []domain.AuditLog
	for rows.Next() {
		var l domain.AuditLog
		var metaBytes []byte
		if err := rows.Scan(&l.ID, &l.ActorID, &l.ActorEmail, &l.Action, &l.Entity, &l.EntityID, &l.IPAddress, &l.UserAgent, &metaBytes, &l.CreatedAt); err != nil {
			return domain.PageResult[domain.AuditLog]{}, fmt.Errorf("audit_repo: scan: %w", err)
		}
		_ = json.Unmarshal(metaBytes, &l.Metadata)
		items = append(items, l)
	}
	if err := rows.Err(); err != nil {
		return domain.PageResult[domain.AuditLog]{}, fmt.Errorf("audit_repo: rows: %w", err)
	}

	return domain.PageResult[domain.AuditLog]{
		Items:      items,
		TotalItems: total,
		Page:       params.Page,
		PerPage:    params.PerPage,
	}, nil
}
