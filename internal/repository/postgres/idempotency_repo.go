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

type IdempotencyRepository struct {
	pool *pgxpool.Pool
}

func NewIdempotencyRepository(pool *pgxpool.Pool) *IdempotencyRepository {
	return &IdempotencyRepository{pool: pool}
}

func (r *IdempotencyRepository) LockOrGet(ctx context.Context, key, path, method, bodyHash string, ttl time.Duration) (*domain.IdempotencyRecord, bool, error) {
	now := time.Now().UTC()
	lockedUntil := now.Add(ttl)

	const insertQ = `
		INSERT INTO idempotency_keys (key, request_path, request_method, request_body_hash, locked_until)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (key) DO NOTHING`

	cmd, err := r.pool.Exec(ctx, insertQ, key, path, method, bodyHash, lockedUntil)
	if err != nil {
		return nil, false, fmt.Errorf("idempotency_repo: insert: %w", err)
	}

	// If successfully inserted, we acquired the lock!
	if cmd.RowsAffected() > 0 {
		return &domain.IdempotencyRecord{
			Key:             key,
			RequestPath:     path,
			RequestMethod:   method,
			RequestBodyHash: bodyHash,
			LockedUntil:     lockedUntil,
			CreatedAt:       now,
		}, false, nil
	}

	// Key already exists — fetch existing record
	const selectQ = `
		SELECT key, request_path, request_method, request_body_hash, status_code, response_body, locked_until, created_at
		FROM idempotency_keys
		WHERE key = $1`

	var rec domain.IdempotencyRecord
	var sc *int
	err = r.pool.QueryRow(ctx, selectQ, key).Scan(
		&rec.Key,
		&rec.RequestPath,
		&rec.RequestMethod,
		&rec.RequestBodyHash,
		&sc,
		&rec.ResponseBody,
		&rec.LockedUntil,
		&rec.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, domain.ErrNotFound
		}
		return nil, false, fmt.Errorf("idempotency_repo: fetch: %w", err)
	}
	if sc != nil {
		rec.StatusCode = *sc
	}

	// If body hash does not match, client is reusing key for different request payload!
	if rec.RequestBodyHash != bodyHash {
		return nil, false, errors.New("idempotency key reused with different request payload")
	}

	return &rec, true, nil
}

func (r *IdempotencyRepository) SaveResponse(ctx context.Context, key string, statusCode int, body []byte) error {
	const updateQ = `
		UPDATE idempotency_keys
		SET status_code = $1, response_body = $2
		WHERE key = $3`

	_, err := r.pool.Exec(ctx, updateQ, statusCode, body, key)
	if err != nil {
		return fmt.Errorf("idempotency_repo: save response: %w", err)
	}
	return nil
}
