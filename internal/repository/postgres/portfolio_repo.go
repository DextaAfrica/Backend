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

type DevelopmentRepository struct {
	pool *pgxpool.Pool
}

func NewDevelopmentRepository(pool *pgxpool.Pool) *DevelopmentRepository {
	return &DevelopmentRepository{pool: pool}
}

func (r *DevelopmentRepository) Create(ctx context.Context, d *domain.Development) error {
	body, err := json.Marshal(d.Body)
	if err != nil {
		return fmt.Errorf("portfolio_repo: encode body: %w", err)
	}
	gallery, err := json.Marshal(d.Gallery)
	if err != nil {
		return fmt.Errorf("portfolio_repo: encode gallery: %w", err)
	}

	const q = `
		INSERT INTO portfolio_developments
			(slug, name, summary, body, hero_image_url, gallery, location, status,
			 featured, seo_title, seo_description, published, published_at, display_order)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING id, created_at, updated_at`

	err = r.pool.QueryRow(ctx, q,
		d.Slug, d.Name, d.Summary, body, d.HeroImageURL, gallery, d.Location, d.Status,
		d.Featured, d.SEOTitle, d.SEODescription, d.Published, d.PublishedAt, d.DisplayOrder,
	).Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt)
	if isUniqueViolation(err) {
		return domain.ErrConflict
	}
	if err != nil {
		return fmt.Errorf("portfolio_repo: create: %w", err)
	}
	return nil
}

func (r *DevelopmentRepository) Update(ctx context.Context, d *domain.Development) error {
	body, err := json.Marshal(d.Body)
	if err != nil {
		return fmt.Errorf("portfolio_repo: encode body: %w", err)
	}
	gallery, err := json.Marshal(d.Gallery)
	if err != nil {
		return fmt.Errorf("portfolio_repo: encode gallery: %w", err)
	}

	const q = `
		UPDATE portfolio_developments SET
			slug = $1, name = $2, summary = $3, body = $4, hero_image_url = $5,
			gallery = $6, location = $7, status = $8, featured = $9,
			seo_title = $10, seo_description = $11, published = $12,
			published_at = $13, display_order = $14
		WHERE id = $15
		RETURNING updated_at`

	err = r.pool.QueryRow(ctx, q,
		d.Slug, d.Name, d.Summary, body, d.HeroImageURL, gallery, d.Location, d.Status,
		d.Featured, d.SEOTitle, d.SEODescription, d.Published, d.PublishedAt, d.DisplayOrder, d.ID,
	).Scan(&d.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	if isUniqueViolation(err) {
		return domain.ErrConflict
	}
	if err != nil {
		return fmt.Errorf("portfolio_repo: update: %w", err)
	}
	return nil
}

func (r *DevelopmentRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM portfolio_developments WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("portfolio_repo: delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *DevelopmentRepository) scanRow(row pgx.Row) (*domain.Development, error) {
	d := &domain.Development{}
	var body, gallery []byte
	err := row.Scan(
		&d.ID, &d.Slug, &d.Name, &d.Summary, &body, &d.HeroImageURL, &gallery,
		&d.Location, &d.Status, &d.Featured, &d.SEOTitle, &d.SEODescription,
		&d.Published, &d.PublishedAt, &d.DisplayOrder, &d.CreatedAt, &d.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("portfolio_repo: scan: %w", err)
	}
	if err := json.Unmarshal(body, &d.Body); err != nil {
		return nil, fmt.Errorf("portfolio_repo: decode body: %w", err)
	}
	if err := json.Unmarshal(gallery, &d.Gallery); err != nil {
		return nil, fmt.Errorf("portfolio_repo: decode gallery: %w", err)
	}
	return d, nil
}

const developmentColumns = `
	id, slug, name, summary, body, hero_image_url, gallery, location, status,
	featured, seo_title, seo_description, published, published_at, display_order, created_at, updated_at`

func (r *DevelopmentRepository) GetByID(ctx context.Context, id string) (*domain.Development, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+developmentColumns+` FROM portfolio_developments WHERE id = $1`, id)
	return r.scanRow(row)
}

func (r *DevelopmentRepository) GetBySlug(ctx context.Context, slug string) (*domain.Development, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+developmentColumns+` FROM portfolio_developments WHERE slug = $1`, slug)
	return r.scanRow(row)
}

func (r *DevelopmentRepository) List(ctx context.Context, params domain.ListParams) (domain.PageResult[domain.Development], error) {
	whereClause := ""
	if params.PublishedOnly {
		whereClause = "WHERE published = true"
	}

	var total int
	countQ := `SELECT count(*) FROM portfolio_developments ` + whereClause
	if err := r.pool.QueryRow(ctx, countQ).Scan(&total); err != nil {
		return domain.PageResult[domain.Development]{}, fmt.Errorf("portfolio_repo: count: %w", err)
	}

	q := `SELECT ` + developmentColumns + ` FROM portfolio_developments ` + whereClause + `
		ORDER BY display_order ASC, created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.pool.Query(ctx, q, params.PerPage, params.Offset())
	if err != nil {
		return domain.PageResult[domain.Development]{}, fmt.Errorf("portfolio_repo: list: %w", err)
	}
	defer rows.Close()

	var items []domain.Development
	for rows.Next() {
		d, err := r.scanRow(rows)
		if err != nil {
			return domain.PageResult[domain.Development]{}, err
		}
		items = append(items, *d)
	}
	if err := rows.Err(); err != nil {
		return domain.PageResult[domain.Development]{}, fmt.Errorf("portfolio_repo: rows: %w", err)
	}

	return domain.PageResult[domain.Development]{
		Items: items, TotalItems: total, Page: params.Page, PerPage: params.PerPage,
	}, nil
}

func (r *DevelopmentRepository) SlugExists(ctx context.Context, slug string, excludeID string) (bool, error) {
	var exists bool
	const q = `SELECT EXISTS(SELECT 1 FROM portfolio_developments WHERE slug = $1 AND id != $2)`
	if err := r.pool.QueryRow(ctx, q, slug, nullableID(excludeID)).Scan(&exists); err != nil {
		return false, fmt.Errorf("portfolio_repo: slug exists: %w", err)
	}
	return exists, nil
}
