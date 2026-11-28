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

type VisitorReportRepository struct {
	pool *pgxpool.Pool
}

func NewVisitorReportRepository(pool *pgxpool.Pool) *VisitorReportRepository {
	return &VisitorReportRepository{pool: pool}
}

const scheduleColumns = `
	id, title, cadence, recipient_emails, department, include_all_depts,
	is_active, last_run_at, next_run_at, created_at, updated_at
`

func scanSchedule(row pgx.Row) (*domain.VisitorReportSchedule, error) {
	var s domain.VisitorReportSchedule
	var cadence string
	err := row.Scan(
		&s.ID, &s.Title, &cadence, &s.RecipientEmails, &s.Department, &s.IncludeAllDepts,
		&s.IsActive, &s.LastRunAt, &s.NextRunAt, &s.CreatedAt, &s.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("visitor_report_repo: scan schedule: %w", err)
	}
	s.Cadence = domain.ReportCadence(cadence)
	return &s, nil
}

func (r *VisitorReportRepository) GetOrCreateSchedule(ctx context.Context) (*domain.VisitorReportSchedule, error) {
	q := `SELECT ` + scheduleColumns + ` FROM visitor_report_schedules WHERE id = 'default'`
	s, err := scanSchedule(r.pool.QueryRow(ctx, q))
	if err == nil {
		return s, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}

	// Create default schedule for weekly reporting
	now := time.Now().UTC()
	nextRun := now.Add(7 * 24 * time.Hour)
	insertQ := `
		INSERT INTO visitor_report_schedules (
			id, title, cadence, recipient_emails, department, include_all_depts,
			is_active, last_run_at, next_run_at, created_at, updated_at
		) VALUES (
			'default', 'Visitor Check-in Digest', 'WEEKLY', $1, 'ALL', true,
			true, NULL, $2, $3, $3
		)
		ON CONFLICT (id) DO UPDATE SET updated_at = now()
		RETURNING ` + scheduleColumns
	return scanSchedule(r.pool.QueryRow(ctx, insertQ, []string{"operations@dextaafrica.com"}, nextRun, now))
}

func (r *VisitorReportRepository) UpdateSchedule(ctx context.Context, schedule *domain.VisitorReportSchedule) error {
	q := `
		INSERT INTO visitor_report_schedules (
			id, title, cadence, recipient_emails, department, include_all_depts,
			is_active, last_run_at, next_run_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, now(), now()
		)
		ON CONFLICT (id) DO UPDATE SET
			title = EXCLUDED.title,
			cadence = EXCLUDED.cadence,
			recipient_emails = EXCLUDED.recipient_emails,
			department = EXCLUDED.department,
			include_all_depts = EXCLUDED.include_all_depts,
			is_active = EXCLUDED.is_active,
			last_run_at = EXCLUDED.last_run_at,
			next_run_at = EXCLUDED.next_run_at,
			updated_at = now()
	`
	_, err := r.pool.Exec(
		ctx, q,
		schedule.ID, schedule.Title, string(schedule.Cadence), schedule.RecipientEmails,
		schedule.Department, schedule.IncludeAllDepts, schedule.IsActive,
		schedule.LastRunAt, schedule.NextRunAt,
	)
	if err != nil {
		return fmt.Errorf("visitor_report_repo: update schedule: %w", err)
	}
	return nil
}

func (r *VisitorReportRepository) GetDueSchedules(ctx context.Context, now time.Time) ([]domain.VisitorReportSchedule, error) {
	q := `
		SELECT ` + scheduleColumns + `
		FROM visitor_report_schedules
		WHERE is_active = true AND next_run_at <= $1
	`
	rows, err := r.pool.Query(ctx, q, now)
	if err != nil {
		return nil, fmt.Errorf("visitor_report_repo: get due schedules: %w", err)
	}
	defer rows.Close()

	var schedules []domain.VisitorReportSchedule
	for rows.Next() {
		var s domain.VisitorReportSchedule
		var cadence string
		if err := rows.Scan(
			&s.ID, &s.Title, &cadence, &s.RecipientEmails, &s.Department, &s.IncludeAllDepts,
			&s.IsActive, &s.LastRunAt, &s.NextRunAt, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		s.Cadence = domain.ReportCadence(cadence)
		schedules = append(schedules, s)
	}
	return schedules, nil
}

func (r *VisitorReportRepository) RecordDelivery(ctx context.Context, delivery *domain.VisitorReportDelivery) error {
	q := `
		INSERT INTO visitor_report_deliveries (
			id, schedule_id, title, cadence, recipient_emails,
			period_start, period_end, total_visitors, status, error_message, sent_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10, $11
		)
	`
	_, err := r.pool.Exec(
		ctx, q,
		delivery.ID, delivery.ScheduleID, delivery.Title, string(delivery.Cadence), delivery.RecipientEmails,
		delivery.PeriodStart, delivery.PeriodEnd, delivery.TotalVisitors, delivery.Status, delivery.ErrorMessage, delivery.SentAt,
	)
	if err != nil {
		return fmt.Errorf("visitor_report_repo: record delivery: %w", err)
	}
	return nil
}

func (r *VisitorReportRepository) ListDeliveries(ctx context.Context, page, perPage int) (domain.PageResult[domain.VisitorReportDelivery], error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	var total int
	countQ := `SELECT count(*) FROM visitor_report_deliveries`
	if err := r.pool.QueryRow(ctx, countQ).Scan(&total); err != nil {
		return domain.PageResult[domain.VisitorReportDelivery]{}, fmt.Errorf("visitor_report_repo: count deliveries: %w", err)
	}

	q := `
		SELECT id, schedule_id, title, cadence, recipient_emails,
		       period_start, period_end, total_visitors, status, error_message, sent_at
		FROM visitor_report_deliveries
		ORDER BY sent_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.pool.Query(ctx, q, perPage, offset)
	if err != nil {
		return domain.PageResult[domain.VisitorReportDelivery]{}, fmt.Errorf("visitor_report_repo: list deliveries: %w", err)
	}
	defer rows.Close()

	var items []domain.VisitorReportDelivery
	for rows.Next() {
		var d domain.VisitorReportDelivery
		var cadence string
		if err := rows.Scan(
			&d.ID, &d.ScheduleID, &d.Title, &cadence, &d.RecipientEmails,
			&d.PeriodStart, &d.PeriodEnd, &d.TotalVisitors, &d.Status, &d.ErrorMessage, &d.SentAt,
		); err != nil {
			return domain.PageResult[domain.VisitorReportDelivery]{}, err
		}
		d.Cadence = domain.ReportCadence(cadence)
		items = append(items, d)
	}
	if items == nil {
		items = []domain.VisitorReportDelivery{}
	}

	return domain.PageResult[domain.VisitorReportDelivery]{
		Items:      items,
		TotalItems: total,
		Page:       page,
		PerPage:    perPage,
	}, nil
}
