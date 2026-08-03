package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DextaAfrica/Backend/internal/domain"
)

type PageRepository struct {
	pool *pgxpool.Pool
}

func NewPageRepository(pool *pgxpool.Pool) *PageRepository {
	return &PageRepository{pool: pool}
}

func (r *PageRepository) GetByKey(ctx context.Context, key string) (*domain.Page, error) {
	const q = `
		SELECT id, key, title, content, seo_title, seo_description, published, created_at, updated_at
		FROM pages
		WHERE key = $1`

	p := &domain.Page{}
	var content []byte
	err := r.pool.QueryRow(ctx, q, key).Scan(
		&p.ID, &p.Key, &p.Title, &content, &p.SEOTitle, &p.SEODescription,
		&p.Published, &p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("page_repo: get by key: %w", err)
	}
	if err := json.Unmarshal(content, &p.Content); err != nil {
		return nil, fmt.Errorf("page_repo: decode content: %w", err)
	}
	return p, nil
}

// Upsert creates or replaces the page identified by Key. Pages are edited as
// a whole document by the CMS admin rather than patched field-by-field,
// which keeps this a single atomic write instead of a partial-update API.
func (r *PageRepository) Upsert(ctx context.Context, page *domain.Page) (*domain.Page, error) {
	content, err := json.Marshal(page.Content)
	if err != nil {
		return nil, fmt.Errorf("page_repo: encode content: %w", err)
	}

	const q = `
		INSERT INTO pages (key, title, content, seo_title, seo_description, published)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (key) DO UPDATE SET
			title = EXCLUDED.title,
			content = EXCLUDED.content,
			seo_title = EXCLUDED.seo_title,
			seo_description = EXCLUDED.seo_description,
			published = EXCLUDED.published
		RETURNING id, created_at, updated_at`

	err = r.pool.QueryRow(ctx, q, page.Key, page.Title, content, page.SEOTitle, page.SEODescription, page.Published).
		Scan(&page.ID, &page.CreatedAt, &page.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("page_repo: upsert: %w", err)
	}
	return page, nil
}

func (r *PageRepository) List(ctx context.Context) ([]domain.Page, error) {
	const q = `
		SELECT id, key, title, content, seo_title, seo_description, published, created_at, updated_at
		FROM pages
		ORDER BY key`

	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("page_repo: list: %w", err)
	}
	defer rows.Close()

	var pages []domain.Page
	for rows.Next() {
		p := domain.Page{}
		var content []byte
		if err := rows.Scan(&p.ID, &p.Key, &p.Title, &content, &p.SEOTitle, &p.SEODescription,
			&p.Published, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("page_repo: scan: %w", err)
		}
		if err := json.Unmarshal(content, &p.Content); err != nil {
			return nil, fmt.Errorf("page_repo: decode content: %w", err)
		}
		pages = append(pages, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("page_repo: rows: %w", err)
	}
	return pages, nil
}
