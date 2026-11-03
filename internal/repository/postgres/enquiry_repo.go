package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DextaAfrica/Backend/internal/domain"
)

type EnquiryRepository struct {
	pool *pgxpool.Pool
}

func NewEnquiryRepository(pool *pgxpool.Pool) *EnquiryRepository {
	return &EnquiryRepository{pool: pool}
}

func (r *EnquiryRepository) Create(ctx context.Context, e *domain.Enquiry) error {
	const q = `
		INSERT INTO enquiries (name, email, phone, subject, message, source_page, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, created_at`

	err := r.pool.QueryRow(ctx, q, e.Name, e.Email, e.Phone, e.Subject, e.Message, e.SourcePage, e.Status).
		Scan(&e.ID, &e.CreatedAt)
	if err != nil {
		return fmt.Errorf("enquiry_repo: create: %w", err)
	}
	return nil
}

const enquiryColumns = `id, name, email, phone, subject, message, source_page, status, created_at`

func (r *EnquiryRepository) GetByID(ctx context.Context, id string) (*domain.Enquiry, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+enquiryColumns+` FROM enquiries WHERE id = $1`, id)
	e := &domain.Enquiry{}
	err := row.Scan(&e.ID, &e.Name, &e.Email, &e.Phone, &e.Subject, &e.Message, &e.SourcePage, &e.Status, &e.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("enquiry_repo: get by id: %w", err)
	}
	return e, nil
}

func (r *EnquiryRepository) UpdateStatus(ctx context.Context, id string, status domain.EnquiryStatus) error {
	tag, err := r.pool.Exec(ctx, `UPDATE enquiries SET status = $1 WHERE id = $2`, status, id)
	if err != nil {
		return fmt.Errorf("enquiry_repo: update status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *EnquiryRepository) List(ctx context.Context, params domain.ListParams) (domain.PageResult[domain.Enquiry], error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM enquiries`).Scan(&total); err != nil {
		return domain.PageResult[domain.Enquiry]{}, fmt.Errorf("enquiry_repo: count: %w", err)
	}

	q := `SELECT ` + enquiryColumns + ` FROM enquiries ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.pool.Query(ctx, q, params.PerPage, params.Offset())
	if err != nil {
		return domain.PageResult[domain.Enquiry]{}, fmt.Errorf("enquiry_repo: list: %w", err)
	}
	defer rows.Close()

	var items []domain.Enquiry
	for rows.Next() {
		e := domain.Enquiry{}
		if err := rows.Scan(&e.ID, &e.Name, &e.Email, &e.Phone, &e.Subject, &e.Message, &e.SourcePage, &e.Status, &e.CreatedAt); err != nil {
			return domain.PageResult[domain.Enquiry]{}, fmt.Errorf("enquiry_repo: scan: %w", err)
		}
		items = append(items, e)
	}
	if err := rows.Err(); err != nil {
		return domain.PageResult[domain.Enquiry]{}, fmt.Errorf("enquiry_repo: rows: %w", err)
	}

	return domain.PageResult[domain.Enquiry]{
		Items: items, TotalItems: total, Page: params.Page, PerPage: params.PerPage,
	}, nil
}
