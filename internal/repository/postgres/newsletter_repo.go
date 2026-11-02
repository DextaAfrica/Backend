package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DextaAfrica/Backend/internal/domain"
)

type SubscriberRepository struct {
	pool *pgxpool.Pool
}

func NewSubscriberRepository(pool *pgxpool.Pool) *SubscriberRepository {
	return &SubscriberRepository{pool: pool}
}

// Upsert (re)subscribes an email. Signing up again after unsubscribing
// simply flips the status back rather than erroring, which matches how a
// newsletter form should behave for a returning visitor.
func (r *SubscriberRepository) Upsert(ctx context.Context, email string) (*domain.Subscriber, error) {
	const q = `
		INSERT INTO newsletter_subscribers (email, status, subscribed_at)
		VALUES ($1, 'subscribed', now())
		ON CONFLICT (email) DO UPDATE SET
			status = 'subscribed',
			subscribed_at = now(),
			unsubscribed_at = NULL
		RETURNING id, email, status, subscribed_at, unsubscribed_at, created_at`

	s := &domain.Subscriber{}
	err := r.pool.QueryRow(ctx, q, email).
		Scan(&s.ID, &s.Email, &s.Status, &s.SubscribedAt, &s.UnsubscribedAt, &s.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("newsletter_repo: upsert: %w", err)
	}
	return s, nil
}

func (r *SubscriberRepository) Unsubscribe(ctx context.Context, email string) error {
	const q = `
		UPDATE newsletter_subscribers
		SET status = 'unsubscribed', unsubscribed_at = now()
		WHERE email = $1`

	tag, err := r.pool.Exec(ctx, q, email)
	if err != nil {
		return fmt.Errorf("newsletter_repo: unsubscribe: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *SubscriberRepository) GetByEmail(ctx context.Context, email string) (*domain.Subscriber, error) {
	const q = `
		SELECT id, email, status, subscribed_at, unsubscribed_at, created_at
		FROM newsletter_subscribers WHERE email = $1`

	s := &domain.Subscriber{}
	err := r.pool.QueryRow(ctx, q, email).
		Scan(&s.ID, &s.Email, &s.Status, &s.SubscribedAt, &s.UnsubscribedAt, &s.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("newsletter_repo: get by email: %w", err)
	}
	return s, nil
}

func (r *SubscriberRepository) List(ctx context.Context, params domain.ListParams) (domain.PageResult[domain.Subscriber], error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM newsletter_subscribers`).Scan(&total); err != nil {
		return domain.PageResult[domain.Subscriber]{}, fmt.Errorf("newsletter_repo: count: %w", err)
	}

	const q = `
		SELECT id, email, status, subscribed_at, unsubscribed_at, created_at
		FROM newsletter_subscribers ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	rows, err := r.pool.Query(ctx, q, params.PerPage, params.Offset())
	if err != nil {
		return domain.PageResult[domain.Subscriber]{}, fmt.Errorf("newsletter_repo: list: %w", err)
	}
	defer rows.Close()

	var items []domain.Subscriber
	for rows.Next() {
		s := domain.Subscriber{}
		if err := rows.Scan(&s.ID, &s.Email, &s.Status, &s.SubscribedAt, &s.UnsubscribedAt, &s.CreatedAt); err != nil {
			return domain.PageResult[domain.Subscriber]{}, fmt.Errorf("newsletter_repo: scan: %w", err)
		}
		items = append(items, s)
	}
	if err := rows.Err(); err != nil {
		return domain.PageResult[domain.Subscriber]{}, fmt.Errorf("newsletter_repo: rows: %w", err)
	}

	return domain.PageResult[domain.Subscriber]{
		Items: items, TotalItems: total, Page: params.Page, PerPage: params.PerPage,
	}, nil
}
