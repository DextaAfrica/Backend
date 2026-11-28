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

type SessionRepository struct {
	pool *pgxpool.Pool
}

func NewSessionRepository(pool *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{pool: pool}
}

const sessionColumns = `id, admin_id, token_hash, ip_address, user_agent, revoked, expires_at, created_at, updated_at`

func (r *SessionRepository) Create(ctx context.Context, session *domain.AdminSession) error {
	const q = `
		INSERT INTO admin_sessions (admin_id, token_hash, ip_address, user_agent, revoked, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, q,
		session.AdminID,
		session.TokenHash,
		session.IPAddress,
		session.UserAgent,
		session.Revoked,
		session.ExpiresAt,
	).Scan(&session.ID, &session.CreatedAt, &session.UpdatedAt)
}

func (r *SessionRepository) GetByTokenHash(ctx context.Context, hash string) (*domain.AdminSession, error) {
	q := `SELECT ` + sessionColumns + ` FROM admin_sessions WHERE token_hash = $1`

	var s domain.AdminSession
	err := r.pool.QueryRow(ctx, q, hash).Scan(
		&s.ID,
		&s.AdminID,
		&s.TokenHash,
		&s.IPAddress,
		&s.UserAgent,
		&s.Revoked,
		&s.ExpiresAt,
		&s.CreatedAt,
		&s.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("session_repo: get by hash: %w", err)
	}
	return &s, nil
}

func (r *SessionRepository) Revoke(ctx context.Context, id string) error {
	const q = `UPDATE admin_sessions SET revoked = true, updated_at = now() WHERE id = $1`
	cmd, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("session_repo: revoke: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *SessionRepository) RevokeAllForAdmin(ctx context.Context, adminID string) error {
	const q = `UPDATE admin_sessions SET revoked = true, updated_at = now() WHERE admin_id = $1 AND revoked = false`
	if _, err := r.pool.Exec(ctx, q, adminID); err != nil {
		return fmt.Errorf("session_repo: revoke all for admin: %w", err)
	}
	return nil
}

func (r *SessionRepository) PruneExpired(ctx context.Context) (int64, error) {
	const q = `DELETE FROM admin_sessions WHERE expires_at < $1`
	cmd, err := r.pool.Exec(ctx, q, time.Now())
	if err != nil {
		return 0, fmt.Errorf("session_repo: prune: %w", err)
	}
	return cmd.RowsAffected(), nil
}
