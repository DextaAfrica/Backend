package postgres

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DextaAfrica/Backend/internal/domain"
)

type AuditRepository struct {
	pool   *pgxpool.Pool
	secret []byte
}

func NewAuditRepository(pool *pgxpool.Pool, secret string) *AuditRepository {
	if secret == "" {
		secret = "dexta-enterprise-audit-default-ledger-secret"
	}
	return &AuditRepository{
		pool:   pool,
		secret: []byte(secret),
	}
}

func (r *AuditRepository) computeHash(prevHash, actorID, action, entity, entityID string, metaJSON []byte) string {
	mac := hmac.New(sha256.New, r.secret)
	mac.Write([]byte(prevHash))
	mac.Write([]byte(actorID))
	mac.Write([]byte(action))
	mac.Write([]byte(entity))
	mac.Write([]byte(entityID))
	mac.Write(metaJSON)
	return hex.EncodeToString(mac.Sum(nil))
}

func (r *AuditRepository) Record(ctx context.Context, log *domain.AuditLog) error {
	metaJSON, err := json.Marshal(log.Metadata)
	if err != nil {
		metaJSON = []byte("{}")
	}

	// Fetch previous record's hash to maintain cryptographic ledger chain
	var prevHash string
	const prevQuery = `SELECT record_hash FROM audit_logs ORDER BY created_at DESC LIMIT 1`
	err = r.pool.QueryRow(ctx, prevQuery).Scan(&prevHash)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("audit_repo: fetch prev hash: %w", err)
	}
	if prevHash == "" {
		prevHash = "GENESIS_BLOCK_DEXTA_LEDGER"
	}
	log.PrevHash = prevHash
	log.RecordHash = r.computeHash(prevHash, log.ActorID, log.Action, log.Entity, log.EntityID, metaJSON)

	const q = `
		INSERT INTO audit_logs (actor_id, actor_email, action, entity, entity_id, ip_address, user_agent, metadata, prev_hash, record_hash)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at`

	return r.pool.QueryRow(ctx, q, log.ActorID, log.ActorEmail, log.Action, log.Entity, log.EntityID, log.IPAddress, log.UserAgent, metaJSON, log.PrevHash, log.RecordHash).
		Scan(&log.ID, &log.CreatedAt)
}

func (r *AuditRepository) List(ctx context.Context, params domain.ListParams) (domain.PageResult[domain.AuditLog], error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM audit_logs`).Scan(&total); err != nil {
		return domain.PageResult[domain.AuditLog]{}, fmt.Errorf("audit_repo: count: %w", err)
	}

	const q = `
		SELECT id, actor_id, actor_email, action, entity, entity_id, ip_address, user_agent, metadata, prev_hash, record_hash, created_at
		FROM audit_logs
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.pool.Query(ctx, q, params.PerPage, params.Offset())
	if err != nil {
		return domain.PageResult[domain.AuditLog]{}, fmt.Errorf("audit_repo: list: %w", err)
	}
	defer rows.Close()

	items := make([]domain.AuditLog, 0)
	for rows.Next() {
		var l domain.AuditLog
		var metaBytes []byte
		if err := rows.Scan(&l.ID, &l.ActorID, &l.ActorEmail, &l.Action, &l.Entity, &l.EntityID, &l.IPAddress, &l.UserAgent, &metaBytes, &l.PrevHash, &l.RecordHash, &l.CreatedAt); err != nil {
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

func (r *AuditRepository) VerifyChain(ctx context.Context) (bool, error) {
	const q = `
		SELECT id, actor_id, action, entity, entity_id, metadata, prev_hash, record_hash
		FROM audit_logs
		ORDER BY created_at ASC`

	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return false, fmt.Errorf("audit_repo: verify query: %w", err)
	}
	defer rows.Close()

	expectedPrev := "GENESIS_BLOCK_DEXTA_LEDGER"
	for rows.Next() {
		var id, actorID, action, entity, entityID, prevHash, recordHash string
		var metaBytes []byte
		if err := rows.Scan(&id, &actorID, &action, &entity, &entityID, &metaBytes, &prevHash, &recordHash); err != nil {
			return false, fmt.Errorf("audit_repo: verify scan: %w", err)
		}

		if prevHash != expectedPrev {
			return false, fmt.Errorf("audit_repo: chain broken at id %s: expected prev %s, got %s", id, expectedPrev, prevHash)
		}

		calculated := r.computeHash(prevHash, actorID, action, entity, entityID, metaBytes)
		if calculated != recordHash {
			return false, fmt.Errorf("audit_repo: tamper detected at id %s: hash mismatch", id)
		}
		expectedPrev = recordHash
	}
	return true, nil
}
