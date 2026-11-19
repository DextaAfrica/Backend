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

type CareerListingRepository struct {
	pool *pgxpool.Pool
}

func NewCareerListingRepository(pool *pgxpool.Pool) *CareerListingRepository {
	return &CareerListingRepository{pool: pool}
}

func (r *CareerListingRepository) Create(ctx context.Context, c *domain.CareerListing) error {
	description, err := json.Marshal(c.Description)
	if err != nil {
		return fmt.Errorf("career_repo: encode description: %w", err)
	}

	if c.Status == "" {
		c.Status = "open"
	}

	const q = `
		INSERT INTO career_listings (slug, title, department, location, employment_type, description, status, published, applicant_click_count)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id, created_at, updated_at`

	err = r.pool.QueryRow(ctx, q, c.Slug, c.Title, c.Department, c.Location, c.EmploymentType, description, c.Status, c.Published, c.ApplicantClickCount).
		Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
	if isUniqueViolation(err) {
		return domain.ErrConflict
	}
	if err != nil {
		return fmt.Errorf("career_repo: create: %w", err)
	}
	return nil
}

func (r *CareerListingRepository) Update(ctx context.Context, c *domain.CareerListing) error {
	description, err := json.Marshal(c.Description)
	if err != nil {
		return fmt.Errorf("career_repo: encode description: %w", err)
	}

	if c.Status == "" {
		c.Status = "open"
	}

	const q = `
		UPDATE career_listings SET
			slug = $1, title = $2, department = $3, location = $4,
			employment_type = $5, description = $6, status = $7, published = $8
		WHERE id = $9
		RETURNING updated_at`

	err = r.pool.QueryRow(ctx, q, c.Slug, c.Title, c.Department, c.Location, c.EmploymentType, description, c.Status, c.Published, c.ID).
		Scan(&c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	if isUniqueViolation(err) {
		return domain.ErrConflict
	}
	if err != nil {
		return fmt.Errorf("career_repo: update: %w", err)
	}
	return nil
}

func (r *CareerListingRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM career_listings WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("career_repo: delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

const careerListingColumns = `
	id, slug, title, department, location, employment_type, description, status, published, applicant_click_count, created_at, updated_at`

func (r *CareerListingRepository) scanRow(row pgx.Row) (*domain.CareerListing, error) {
	c := &domain.CareerListing{}
	var description []byte
	err := row.Scan(&c.ID, &c.Slug, &c.Title, &c.Department, &c.Location, &c.EmploymentType,
		&description, &c.Status, &c.Published, &c.ApplicantClickCount, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("career_repo: scan: %w", err)
	}
	if err := json.Unmarshal(description, &c.Description); err != nil {
		return nil, fmt.Errorf("career_repo: decode description: %w", err)
	}
	return c, nil
}

func (r *CareerListingRepository) GetByID(ctx context.Context, id string) (*domain.CareerListing, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+careerListingColumns+` FROM career_listings WHERE id = $1`, id)
	return r.scanRow(row)
}

func (r *CareerListingRepository) GetBySlug(ctx context.Context, slug string) (*domain.CareerListing, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+careerListingColumns+` FROM career_listings WHERE slug = $1`, slug)
	return r.scanRow(row)
}

func (r *CareerListingRepository) List(ctx context.Context, params domain.ListParams) (domain.PageResult[domain.CareerListing], error) {
	whereClause := ""
	if params.PublishedOnly {
		whereClause = "WHERE published = true"
	}

	var total int
	countQ := `SELECT count(*) FROM career_listings ` + whereClause
	if err := r.pool.QueryRow(ctx, countQ).Scan(&total); err != nil {
		return domain.PageResult[domain.CareerListing]{}, fmt.Errorf("career_repo: count: %w", err)
	}

	q := `SELECT ` + careerListingColumns + ` FROM career_listings ` + whereClause + `
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.pool.Query(ctx, q, params.PerPage, params.Offset())
	if err != nil {
		return domain.PageResult[domain.CareerListing]{}, fmt.Errorf("career_repo: list: %w", err)
	}
	defer rows.Close()

	var items []domain.CareerListing
	for rows.Next() {
		c, err := r.scanRow(rows)
		if err != nil {
			return domain.PageResult[domain.CareerListing]{}, err
		}
		items = append(items, *c)
	}
	if err := rows.Err(); err != nil {
		return domain.PageResult[domain.CareerListing]{}, fmt.Errorf("career_repo: rows: %w", err)
	}

	return domain.PageResult[domain.CareerListing]{
		Items: items, TotalItems: total, Page: params.Page, PerPage: params.PerPage,
	}, nil
}

func (r *CareerListingRepository) SlugExists(ctx context.Context, slug string, excludeID string) (bool, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT count(*) FROM career_listings WHERE slug = $1 AND id <> $2`, slug, excludeID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("career_repo: check slug: %w", err)
	}
	return count > 0, nil
}

func (r *CareerListingRepository) IncrementClickCount(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `UPDATE career_listings SET applicant_click_count = applicant_click_count + 1 WHERE id = $1`, id)
	return err
}

func (r *CareerListingRepository) RecordInquiry(ctx context.Context, inq *domain.CareerInquiry) error {
	const q = `
		INSERT INTO career_inquiries (career_listing_id, applicant_name, applicant_email, applicant_phone, resume_url, cover_letter, source)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`
	return r.pool.QueryRow(ctx, q, inq.CareerListingID, inq.ApplicantName, inq.ApplicantEmail, inq.ApplicantPhone, inq.ResumeURL, inq.CoverLetter, inq.Source).
		Scan(&inq.ID, &inq.CreatedAt)
}

func (r *CareerListingRepository) ListInquiries(ctx context.Context, listingID string) ([]domain.CareerInquiry, error) {
	const q = `
		SELECT id, career_listing_id, applicant_name, applicant_email, applicant_phone, resume_url, cover_letter, source, created_at
		FROM career_inquiries
		WHERE career_listing_id = $1
		ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, q, listingID)
	if err != nil {
		return nil, fmt.Errorf("career_repo: list inquiries: %w", err)
	}
	defer rows.Close()

	var inquiries []domain.CareerInquiry
	for rows.Next() {
		var inq domain.CareerInquiry
		if err := rows.Scan(&inq.ID, &inq.CareerListingID, &inq.ApplicantName, &inq.ApplicantEmail, &inq.ApplicantPhone, &inq.ResumeURL, &inq.CoverLetter, &inq.Source, &inq.CreatedAt); err != nil {
			return nil, fmt.Errorf("career_repo: scan inquiry: %w", err)
		}
		inquiries = append(inquiries, inq)
	}
	return inquiries, nil
}
