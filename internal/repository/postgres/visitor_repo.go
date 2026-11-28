package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DextaAfrica/Backend/internal/domain"
)

type VisitorRepository struct {
	pool *pgxpool.Pool
}

func NewVisitorRepository(pool *pgxpool.Pool) *VisitorRepository {
	return &VisitorRepository{pool: pool}
}

const visitorColumns = `
	id, pass_code, full_name, phone, email, company, position,
	purpose, purpose_other, host_name, department, has_appointment,
	invited_by, visit_description, visit_date, check_in_at, check_out_at,
	status, agreed_guidelines, consented_data, declared_at,
	ip_address, user_agent, checked_out_by, created_at, updated_at
`

func scanVisitor(row pgx.Row) (*domain.Visitor, error) {
	var v domain.Visitor
	var status string
	err := row.Scan(
		&v.ID, &v.PassCode, &v.FullName, &v.Phone, &v.Email, &v.Company, &v.Position,
		&v.Purpose, &v.PurposeOther, &v.HostName, &v.Department, &v.HasAppointment,
		&v.InvitedBy, &v.VisitDescription, &v.VisitDate, &v.CheckInAt, &v.CheckOutAt,
		&status, &v.AgreedGuidelines, &v.ConsentedData, &v.DeclaredAt,
		&v.IPAddress, &v.UserAgent, &v.CheckedOutBy, &v.CreatedAt, &v.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("visitor_repo: scan: %w", err)
	}
	v.Status = domain.VisitorStatus(status)
	return &v, nil
}

func (r *VisitorRepository) Create(ctx context.Context, v *domain.Visitor) error {
	q := `
		INSERT INTO visitors (
			pass_code, full_name, phone, email, company, position,
			purpose, purpose_other, host_name, department, has_appointment,
			invited_by, visit_description, visit_date, check_in_at,
			status, agreed_guidelines, consented_data, declared_at,
			ip_address, user_agent
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11,
			$12, $13, CURRENT_DATE, now(),
			'checked_in', $14, $15, now(),
			$16, $17
		)
		RETURNING id, visit_date, check_in_at, declared_at, created_at, updated_at
	`
	err := r.pool.QueryRow(ctx, q,
		v.PassCode, v.FullName, v.Phone, v.Email, v.Company, v.Position,
		v.Purpose, v.PurposeOther, v.HostName, v.Department, v.HasAppointment,
		v.InvitedBy, v.VisitDescription,
		v.AgreedGuidelines, v.ConsentedData,
		v.IPAddress, v.UserAgent,
	).Scan(&v.ID, &v.VisitDate, &v.CheckInAt, &v.DeclaredAt, &v.CreatedAt, &v.UpdatedAt)
	if isUniqueViolation(err) {
		return domain.ErrConflict
	}
	if err != nil {
		return fmt.Errorf("visitor_repo: create: %w", err)
	}
	v.Status = domain.VisitorStatusCheckedIn
	return nil
}

func (r *VisitorRepository) GetByID(ctx context.Context, id string) (*domain.Visitor, error) {
	q := `SELECT ` + visitorColumns + ` FROM visitors WHERE id = $1`
	return scanVisitor(r.pool.QueryRow(ctx, q, id))
}

func (r *VisitorRepository) GetByPassCode(ctx context.Context, passCode string) (*domain.Visitor, error) {
	q := `SELECT ` + visitorColumns + ` FROM visitors WHERE UPPER(pass_code) = UPPER($1)`
	return scanVisitor(r.pool.QueryRow(ctx, q, strings.TrimSpace(passCode)))
}

func (r *VisitorRepository) FindActiveByContact(ctx context.Context, identifier string) (*domain.Visitor, error) {
	clean := strings.TrimSpace(identifier)
	q := `
		SELECT ` + visitorColumns + `
		FROM visitors
		WHERE status = 'checked_in'
		  AND (UPPER(pass_code) = UPPER($1) OR LOWER(email) = LOWER($1) OR phone = $1)
		ORDER BY check_in_at DESC
		LIMIT 1
	`
	return scanVisitor(r.pool.QueryRow(ctx, q, clean))
}

func (r *VisitorRepository) CheckOut(ctx context.Context, id string, checkedOutBy string) (*domain.Visitor, error) {
	q := `
		UPDATE visitors
		SET status = 'checked_out',
		    check_out_at = COALESCE(check_out_at, now()),
		    checked_out_by = CASE WHEN checked_out_by = '' THEN $2 ELSE checked_out_by END
		WHERE id = $1
		RETURNING ` + visitorColumns
	return scanVisitor(r.pool.QueryRow(ctx, q, id, checkedOutBy))
}

func (r *VisitorRepository) List(ctx context.Context, filter domain.VisitorFilter) (domain.PageResult[domain.Visitor], error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PerPage <= 0 || filter.PerPage > 500 {
		filter.PerPage = 50
	}

	var conditions []string
	var args []any
	argIdx := 1

	if filter.Status != "" && filter.Status != "all" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}
	if filter.Department != "" && filter.Department != "all" {
		conditions = append(conditions, fmt.Sprintf("department = $%d", argIdx))
		args = append(args, filter.Department)
		argIdx++
	}
	if filter.Purpose != "" && filter.Purpose != "all" {
		conditions = append(conditions, fmt.Sprintf("purpose = $%d", argIdx))
		args = append(args, filter.Purpose)
		argIdx++
	}
	if filter.Date != "" {
		conditions = append(conditions, fmt.Sprintf("visit_date = $%d::date", argIdx))
		args = append(args, filter.Date)
		argIdx++
	}
	if filter.StartDate != nil {
		conditions = append(conditions, fmt.Sprintf("check_in_at >= $%d", argIdx))
		args = append(args, *filter.StartDate)
		argIdx++
	}
	if filter.EndDate != nil {
		conditions = append(conditions, fmt.Sprintf("check_in_at <= $%d", argIdx))
		args = append(args, *filter.EndDate)
		argIdx++
	}
	if q := strings.TrimSpace(filter.Query); q != "" {
		like := "%" + q + "%"
		conditions = append(conditions, fmt.Sprintf("(full_name ILIKE $%d OR email ILIKE $%d OR phone ILIKE $%d OR company ILIKE $%d OR host_name ILIKE $%d OR pass_code ILIKE $%d)", argIdx, argIdx, argIdx, argIdx, argIdx, argIdx))
		args = append(args, like)
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	var total int
	countQ := `SELECT count(*) FROM visitors ` + whereClause
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return domain.PageResult[domain.Visitor]{}, fmt.Errorf("visitor_repo: count: %w", err)
	}

	offset := (filter.Page - 1) * filter.PerPage
	listQ := fmt.Sprintf(`
		SELECT %s FROM visitors %s
		ORDER BY CASE WHEN status = 'checked_in' THEN 0 ELSE 1 END, check_in_at DESC
		LIMIT $%d OFFSET $%d
	`, visitorColumns, whereClause, argIdx, argIdx+1)
	args = append(args, filter.PerPage, offset)

	rows, err := r.pool.Query(ctx, listQ, args...)
	if err != nil {
		return domain.PageResult[domain.Visitor]{}, fmt.Errorf("visitor_repo: list: %w", err)
	}
	defer rows.Close()

	var items []domain.Visitor
	for rows.Next() {
		v, err := scanVisitor(rows)
		if err != nil {
			return domain.PageResult[domain.Visitor]{}, err
		}
		items = append(items, *v)
	}
	if items == nil {
		items = []domain.Visitor{}
	}

	return domain.PageResult[domain.Visitor]{
		Items:      items,
		TotalItems: total,
		Page:       filter.Page,
		PerPage:    filter.PerPage,
	}, nil
}

func (r *VisitorRepository) GetAnalytics(ctx context.Context) (*domain.VisitorAnalytics, error) {
	analytics := &domain.VisitorAnalytics{
		ByDepartment:   []domain.DemographicCount{},
		ByPurpose:      []domain.DemographicCount{},
		HourlyArrivals: []domain.DemographicCount{},
	}

	kpiQ := `
		SELECT
			COUNT(*) FILTER (WHERE visit_date = CURRENT_DATE) AS total_today,
			COUNT(*) FILTER (WHERE status = 'checked_in') AS currently_on_site,
			COUNT(*) FILTER (WHERE visit_date = CURRENT_DATE AND status = 'checked_out') AS checked_out_today,
			COUNT(*) FILTER (WHERE visit_date = CURRENT_DATE AND has_appointment = true) AS scheduled_today,
			COUNT(*) FILTER (WHERE visit_date = CURRENT_DATE AND has_appointment = false) AS walk_in_today,
			COUNT(*) AS total_all_time
		FROM visitors
	`
	if err := r.pool.QueryRow(ctx, kpiQ).Scan(
		&analytics.TotalToday,
		&analytics.CurrentlyOnSite,
		&analytics.CheckedOutToday,
		&analytics.ScheduledToday,
		&analytics.WalkInToday,
		&analytics.TotalAllTime,
	); err != nil {
		return nil, fmt.Errorf("visitor_repo: analytics kpis: %w", err)
	}

	deptRows, err := r.pool.Query(ctx, `
		SELECT department, COUNT(*) AS cnt
		FROM visitors
		GROUP BY department
		ORDER BY cnt DESC
	`)
	if err == nil {
		defer deptRows.Close()
		for deptRows.Next() {
			var c domain.DemographicCount
			if err := deptRows.Scan(&c.Label, &c.Count); err == nil {
				analytics.ByDepartment = append(analytics.ByDepartment, c)
			}
		}
	}

	purposeRows, err := r.pool.Query(ctx, `
		SELECT purpose, COUNT(*) AS cnt
		FROM visitors
		GROUP BY purpose
		ORDER BY cnt DESC
	`)
	if err == nil {
		defer purposeRows.Close()
		for purposeRows.Next() {
			var c domain.DemographicCount
			if err := purposeRows.Scan(&c.Label, &c.Count); err == nil {
				analytics.ByPurpose = append(analytics.ByPurpose, c)
			}
		}
	}

	hourRows, err := r.pool.Query(ctx, `
		SELECT to_char(date_trunc('hour', check_in_at), 'HH24:00') AS hr, COUNT(*) AS cnt
		FROM visitors
		WHERE visit_date >= CURRENT_DATE - INTERVAL '7 days'
		GROUP BY hr
		ORDER BY hr ASC
	`)
	if err == nil {
		defer hourRows.Close()
		for hourRows.Next() {
			var c domain.DemographicCount
			if err := hourRows.Scan(&c.Label, &c.Count); err == nil {
				analytics.HourlyArrivals = append(analytics.HourlyArrivals, c)
			}
		}
	}

	return analytics, nil
}
