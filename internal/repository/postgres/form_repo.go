package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DextaAfrica/Backend/internal/domain"
)

type FormRepository struct {
	pool *pgxpool.Pool
}

func NewFormRepository(pool *pgxpool.Pool) *FormRepository {
	return &FormRepository{pool: pool}
}

const formColumns = `
	f.id, f.slug, f.title, f.description, f.form_type, f.status,
	f.cover_image_url, f.event_date, f.event_location, f.active_from, f.active_until, f.fields, f.metadata,
	COALESCE((SELECT COUNT(*) FROM form_submissions fs WHERE fs.form_id = f.id), 0) AS submissions_count,
	f.created_at, f.updated_at`

func (r *FormRepository) scanForm(row pgx.Row) (*domain.CustomForm, error) {
	f := &domain.CustomForm{}
	err := row.Scan(
		&f.ID, &f.Slug, &f.Title, &f.Description, &f.FormType, &f.Status,
		&f.CoverImageURL, &f.EventDate, &f.EventLocation, &f.ActiveFrom, &f.ActiveUntil, &f.Fields, &f.Metadata,
		&f.SubmissionsCount, &f.CreatedAt, &f.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("form_repo: scan form: %w", err)
	}
	return f, nil
}

func (r *FormRepository) Create(ctx context.Context, f *domain.CustomForm) error {
	fieldsJSON := f.Fields
	if len(fieldsJSON) == 0 {
		fieldsJSON = []byte("[]")
	}
	metaJSON := f.Metadata
	if len(metaJSON) == 0 {
		metaJSON = []byte("{}")
	}

	const q = `
		INSERT INTO custom_forms (
			slug, title, description, form_type, status, cover_image_url,
			event_date, event_location, active_from, active_until, fields, metadata
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		RETURNING id, created_at, updated_at`

	err := r.pool.QueryRow(ctx, q,
		f.Slug, f.Title, f.Description, f.FormType, f.Status, f.CoverImageURL,
		f.EventDate, f.EventLocation, f.ActiveFrom, f.ActiveUntil, fieldsJSON, metaJSON,
	).Scan(&f.ID, &f.CreatedAt, &f.UpdatedAt)

	if isUniqueViolation(err) {
		return domain.ErrConflict
	}
	if err != nil {
		return fmt.Errorf("form_repo: create: %w", err)
	}
	return nil
}

func (r *FormRepository) Update(ctx context.Context, f *domain.CustomForm) error {
	fieldsJSON := f.Fields
	if len(fieldsJSON) == 0 {
		fieldsJSON = []byte("[]")
	}
	metaJSON := f.Metadata
	if len(metaJSON) == 0 {
		metaJSON = []byte("{}")
	}

	const q = `
		UPDATE custom_forms SET
			slug = $1, title = $2, description = $3, form_type = $4,
			status = $5, cover_image_url = $6, event_date = $7,
			event_location = $8, active_from = $9, active_until = $10,
			fields = $11, metadata = $12
		WHERE id = $13
		RETURNING updated_at`

	err := r.pool.QueryRow(ctx, q,
		f.Slug, f.Title, f.Description, f.FormType, f.Status, f.CoverImageURL,
		f.EventDate, f.EventLocation, f.ActiveFrom, f.ActiveUntil, fieldsJSON, metaJSON, f.ID,
	).Scan(&f.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	if isUniqueViolation(err) {
		return domain.ErrConflict
	}
	if err != nil {
		return fmt.Errorf("form_repo: update: %w", err)
	}
	return nil
}

func (r *FormRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM custom_forms WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("form_repo: delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *FormRepository) GetByID(ctx context.Context, id string) (*domain.CustomForm, error) {
	q := `SELECT ` + formColumns + ` FROM custom_forms f WHERE f.id = $1`
	return r.scanForm(r.pool.QueryRow(ctx, q, id))
}

func (r *FormRepository) GetBySlug(ctx context.Context, slug string) (*domain.CustomForm, error) {
	q := `SELECT ` + formColumns + ` FROM custom_forms f WHERE f.slug = $1`
	return r.scanForm(r.pool.QueryRow(ctx, q, slug))
}

func (r *FormRepository) List(ctx context.Context, params domain.ListParams) (domain.PageResult[domain.CustomForm], error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM custom_forms`).Scan(&total); err != nil {
		return domain.PageResult[domain.CustomForm]{}, fmt.Errorf("form_repo: count: %w", err)
	}

	limit := params.PerPage
	if limit <= 0 {
		limit = 50
	}

	q := `SELECT ` + formColumns + ` FROM custom_forms f
		ORDER BY f.created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.pool.Query(ctx, q, limit, params.Offset())
	if err != nil {
		return domain.PageResult[domain.CustomForm]{}, fmt.Errorf("form_repo: list: %w", err)
	}
	defer rows.Close()

	var items []domain.CustomForm
	for rows.Next() {
		f, err := r.scanForm(rows)
		if err != nil {
			return domain.PageResult[domain.CustomForm]{}, err
		}
		items = append(items, *f)
	}

	return domain.PageResult[domain.CustomForm]{
		Items:      items,
		TotalItems: total,
		Page:       params.Page,
		PerPage:    limit,
	}, nil
}

const submissionColumns = `
	id, form_id, full_name, email, phone, company, job_title,
	category, attendance_mode, answers, notes, status, created_at`

func (r *FormRepository) scanSubmission(row pgx.Row) (*domain.FormSubmission, error) {
	s := &domain.FormSubmission{}
	err := row.Scan(
		&s.ID, &s.FormID, &s.FullName, &s.Email, &s.Phone, &s.Company, &s.JobTitle,
		&s.Category, &s.AttendanceMode, &s.Answers, &s.Notes, &s.Status, &s.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("form_repo: scan submission: %w", err)
	}
	return s, nil
}

func (r *FormRepository) CreateSubmission(ctx context.Context, s *domain.FormSubmission) error {
	answersJSON := s.Answers
	if len(answersJSON) == 0 {
		answersJSON = []byte("{}")
	}

	const q = `
		INSERT INTO form_submissions (
			form_id, full_name, email, phone, company, job_title,
			category, attendance_mode, answers, notes, status
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id, created_at`

	err := r.pool.QueryRow(ctx, q,
		s.FormID, s.FullName, s.Email, s.Phone, s.Company, s.JobTitle,
		s.Category, s.AttendanceMode, answersJSON, s.Notes, s.Status,
	).Scan(&s.ID, &s.CreatedAt)

	if err != nil {
		return fmt.Errorf("form_repo: create submission: %w", err)
	}
	return nil
}

func (r *FormRepository) ListSubmissions(ctx context.Context, formID string, params domain.ListParams) (domain.PageResult[domain.FormSubmission], error) {
	var total int
	countQ := `SELECT count(*) FROM form_submissions WHERE form_id = $1`
	if err := r.pool.QueryRow(ctx, countQ, formID).Scan(&total); err != nil {
		return domain.PageResult[domain.FormSubmission]{}, fmt.Errorf("form_repo: count submissions: %w", err)
	}

	limit := params.PerPage
	if limit <= 0 {
		limit = 100
	}

	q := `SELECT ` + submissionColumns + ` FROM form_submissions
		WHERE form_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, q, formID, limit, params.Offset())
	if err != nil {
		return domain.PageResult[domain.FormSubmission]{}, fmt.Errorf("form_repo: list submissions: %w", err)
	}
	defer rows.Close()

	var items []domain.FormSubmission
	for rows.Next() {
		s, err := r.scanSubmission(rows)
		if err != nil {
			return domain.PageResult[domain.FormSubmission]{}, err
		}
		items = append(items, *s)
	}

	return domain.PageResult[domain.FormSubmission]{
		Items:      items,
		TotalItems: total,
		Page:       params.Page,
		PerPage:    limit,
	}, nil
}

func (r *FormRepository) GetDemographics(ctx context.Context, formID string) (*domain.FormDemographics, error) {
	demo := &domain.FormDemographics{
		ByCategory:        make([]domain.DemographicCount, 0),
		ByAttendanceMode:  make([]domain.DemographicCount, 0),
		TopOrganizations:  make([]domain.DemographicCount, 0),
		RecentSubmissions: make([]domain.FormSubmission, 0),
	}

	// Total count
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM form_submissions WHERE form_id = $1`, formID).Scan(&demo.TotalSubmissions); err != nil {
		return nil, fmt.Errorf("form_repo: demo total: %w", err)
	}

	// By category
	catRows, err := r.pool.Query(ctx, `
		SELECT COALESCE(NULLIF(category, ''), 'Attendee') as label, COUNT(*) as cnt
		FROM form_submissions
		WHERE form_id = $1
		GROUP BY label
		ORDER BY cnt DESC`, formID)
	if err == nil {
		defer catRows.Close()
		for catRows.Next() {
			var dc domain.DemographicCount
			if err := catRows.Scan(&dc.Label, &dc.Count); err == nil {
				demo.ByCategory = append(demo.ByCategory, dc)
			}
		}
	}

	// By attendance mode
	modeRows, err := r.pool.Query(ctx, `
		SELECT COALESCE(NULLIF(attendance_mode, ''), 'in-person') as label, COUNT(*) as cnt
		FROM form_submissions
		WHERE form_id = $1
		GROUP BY label
		ORDER BY cnt DESC`, formID)
	if err == nil {
		defer modeRows.Close()
		for modeRows.Next() {
			var dc domain.DemographicCount
			if err := modeRows.Scan(&dc.Label, &dc.Count); err == nil {
				demo.ByAttendanceMode = append(demo.ByAttendanceMode, dc)
			}
		}
	}

	// Top organizations
	orgRows, err := r.pool.Query(ctx, `
		SELECT company as label, COUNT(*) as cnt
		FROM form_submissions
		WHERE form_id = $1 AND company != ''
		GROUP BY company
		ORDER BY cnt DESC
		LIMIT 6`, formID)
	if err == nil {
		defer orgRows.Close()
		for orgRows.Next() {
			var dc domain.DemographicCount
			if err := orgRows.Scan(&dc.Label, &dc.Count); err == nil {
				demo.TopOrganizations = append(demo.TopOrganizations, dc)
			}
		}
	}

	// Recent submissions (last 10)
	recentRows, err := r.pool.Query(ctx, `
		SELECT `+submissionColumns+` FROM form_submissions
		WHERE form_id = $1
		ORDER BY created_at DESC
		LIMIT 10`, formID)
	if err == nil {
		defer recentRows.Close()
		for recentRows.Next() {
			s, err := r.scanSubmission(recentRows)
			if err == nil {
				demo.RecentSubmissions = append(demo.RecentSubmissions, *s)
			}
		}
	}

	return demo, nil
}
