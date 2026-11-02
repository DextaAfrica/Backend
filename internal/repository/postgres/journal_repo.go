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

type ArticleRepository struct {
	pool *pgxpool.Pool
}

func NewArticleRepository(pool *pgxpool.Pool) *ArticleRepository {
	return &ArticleRepository{pool: pool}
}

func (r *ArticleRepository) Create(ctx context.Context, a *domain.Article) error {
	body, err := json.Marshal(a.Body)
	if err != nil {
		return fmt.Errorf("journal_repo: encode body: %w", err)
	}

	const q = `
		INSERT INTO journal_articles
			(slug, title, excerpt, body, cover_image_url, author, category,
			 seo_title, seo_description, published, published_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id, created_at, updated_at`

	err = r.pool.QueryRow(ctx, q,
		a.Slug, a.Title, a.Excerpt, body, a.CoverImageURL, a.Author, a.Category,
		a.SEOTitle, a.SEODescription, a.Published, a.PublishedAt,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
	if isUniqueViolation(err) {
		return domain.ErrConflict
	}
	if err != nil {
		return fmt.Errorf("journal_repo: create: %w", err)
	}
	return nil
}

func (r *ArticleRepository) Update(ctx context.Context, a *domain.Article) error {
	body, err := json.Marshal(a.Body)
	if err != nil {
		return fmt.Errorf("journal_repo: encode body: %w", err)
	}

	const q = `
		UPDATE journal_articles SET
			slug = $1, title = $2, excerpt = $3, body = $4, cover_image_url = $5,
			author = $6, category = $7, seo_title = $8, seo_description = $9,
			published = $10, published_at = $11
		WHERE id = $12
		RETURNING updated_at`

	err = r.pool.QueryRow(ctx, q,
		a.Slug, a.Title, a.Excerpt, body, a.CoverImageURL, a.Author, a.Category,
		a.SEOTitle, a.SEODescription, a.Published, a.PublishedAt, a.ID,
	).Scan(&a.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	if isUniqueViolation(err) {
		return domain.ErrConflict
	}
	if err != nil {
		return fmt.Errorf("journal_repo: update: %w", err)
	}
	return nil
}

func (r *ArticleRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM journal_articles WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("journal_repo: delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

const articleColumns = `
	id, slug, title, excerpt, body, cover_image_url, author, category,
	seo_title, seo_description, published, published_at, created_at, updated_at`

func (r *ArticleRepository) scanRow(row pgx.Row) (*domain.Article, error) {
	a := &domain.Article{}
	var body []byte
	err := row.Scan(
		&a.ID, &a.Slug, &a.Title, &a.Excerpt, &body, &a.CoverImageURL, &a.Author, &a.Category,
		&a.SEOTitle, &a.SEODescription, &a.Published, &a.PublishedAt, &a.CreatedAt, &a.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("journal_repo: scan: %w", err)
	}
	if err := json.Unmarshal(body, &a.Body); err != nil {
		return nil, fmt.Errorf("journal_repo: decode body: %w", err)
	}
	return a, nil
}

func (r *ArticleRepository) GetByID(ctx context.Context, id string) (*domain.Article, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+articleColumns+` FROM journal_articles WHERE id = $1`, id)
	return r.scanRow(row)
}

func (r *ArticleRepository) GetBySlug(ctx context.Context, slug string) (*domain.Article, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+articleColumns+` FROM journal_articles WHERE slug = $1`, slug)
	return r.scanRow(row)
}

func (r *ArticleRepository) List(ctx context.Context, params domain.ListParams) (domain.PageResult[domain.Article], error) {
	whereClause := ""
	if params.PublishedOnly {
		whereClause = "WHERE published = true"
	}

	var total int
	countQ := `SELECT count(*) FROM journal_articles ` + whereClause
	if err := r.pool.QueryRow(ctx, countQ).Scan(&total); err != nil {
		return domain.PageResult[domain.Article]{}, fmt.Errorf("journal_repo: count: %w", err)
	}

	q := `SELECT ` + articleColumns + ` FROM journal_articles ` + whereClause + `
		ORDER BY COALESCE(published_at, created_at) DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.pool.Query(ctx, q, params.PerPage, params.Offset())
	if err != nil {
		return domain.PageResult[domain.Article]{}, fmt.Errorf("journal_repo: list: %w", err)
	}
	defer rows.Close()

	var items []domain.Article
	for rows.Next() {
		a, err := r.scanRow(rows)
		if err != nil {
			return domain.PageResult[domain.Article]{}, err
		}
		items = append(items, *a)
	}
	if err := rows.Err(); err != nil {
		return domain.PageResult[domain.Article]{}, fmt.Errorf("journal_repo: rows: %w", err)
	}

	return domain.PageResult[domain.Article]{
		Items: items, TotalItems: total, Page: params.Page, PerPage: params.PerPage,
	}, nil
}

func (r *ArticleRepository) SlugExists(ctx context.Context, slug string, excludeID string) (bool, error) {
	var exists bool
	const q = `SELECT EXISTS(SELECT 1 FROM journal_articles WHERE slug = $1 AND id != $2)`
	if err := r.pool.QueryRow(ctx, q, slug, nullableID(excludeID)).Scan(&exists); err != nil {
		return false, fmt.Errorf("journal_repo: slug exists: %w", err)
	}
	return exists, nil
}
