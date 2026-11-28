package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/DextaAfrica/Backend/internal/config"
	"github.com/DextaAfrica/Backend/internal/domain"
	"github.com/DextaAfrica/Backend/internal/service"
)

type mockVisitorReportRepo struct {
	schedule   *domain.VisitorReportSchedule
	deliveries []domain.VisitorReportDelivery
}

func (m *mockVisitorReportRepo) GetOrCreateSchedule(ctx context.Context) (*domain.VisitorReportSchedule, error) {
	if m.schedule == nil {
		m.schedule = &domain.VisitorReportSchedule{
			ID:              "default",
			Title:           "Visitor Check-in Digest",
			Cadence:         domain.ReportCadenceWeekly,
			RecipientEmails: []string{"security@dextaafrica.com"},
			Department:      "ALL",
			IncludeAllDepts: true,
			IsActive:        true,
			NextRunAt:       time.Now().UTC().Add(7 * 24 * time.Hour),
		}
	}
	return m.schedule, nil
}

func (m *mockVisitorReportRepo) UpdateSchedule(ctx context.Context, s *domain.VisitorReportSchedule) error {
	m.schedule = s
	return nil
}

func (m *mockVisitorReportRepo) GetDueSchedules(ctx context.Context, now time.Time) ([]domain.VisitorReportSchedule, error) {
	if m.schedule != nil && m.schedule.IsActive && !m.schedule.NextRunAt.After(now) {
		return []domain.VisitorReportSchedule{*m.schedule}, nil
	}
	return nil, nil
}

func (m *mockVisitorReportRepo) RecordDelivery(ctx context.Context, d *domain.VisitorReportDelivery) error {
	m.deliveries = append(m.deliveries, *d)
	return nil
}

func (m *mockVisitorReportRepo) ListDeliveries(ctx context.Context, page, perPage int) (domain.PageResult[domain.VisitorReportDelivery], error) {
	return domain.PageResult[domain.VisitorReportDelivery]{
		Items:      m.deliveries,
		TotalItems: len(m.deliveries),
		Page:       page,
		PerPage:    perPage,
	}, nil
}

type mockVisitorRepoForReports struct {
	visitors []domain.Visitor
}

func (m *mockVisitorRepoForReports) Create(ctx context.Context, v *domain.Visitor) error {
	m.visitors = append(m.visitors, *v)
	return nil
}
func (m *mockVisitorRepoForReports) GetByID(ctx context.Context, id string) (*domain.Visitor, error) {
	return nil, nil
}
func (m *mockVisitorRepoForReports) GetByPassCode(ctx context.Context, code string) (*domain.Visitor, error) {
	return nil, nil
}
func (m *mockVisitorRepoForReports) FindActiveByContact(ctx context.Context, id string) (*domain.Visitor, error) {
	return nil, nil
}
func (m *mockVisitorRepoForReports) CheckOut(ctx context.Context, id, by string) (*domain.Visitor, error) {
	return nil, nil
}
func (m *mockVisitorRepoForReports) List(ctx context.Context, filter domain.VisitorFilter) (domain.PageResult[domain.Visitor], error) {
	return domain.PageResult[domain.Visitor]{
		Items:      m.visitors,
		TotalItems: len(m.visitors),
		Page:       filter.Page,
		PerPage:    filter.PerPage,
	}, nil
}
func (m *mockVisitorRepoForReports) GetAnalytics(ctx context.Context) (*domain.VisitorAnalytics, error) {
	return &domain.VisitorAnalytics{TotalToday: len(m.visitors)}, nil
}

func TestVisitorReportService_ScheduleAndUpdate(t *testing.T) {
	reportRepo := &mockVisitorReportRepo{}
	visitorRepo := &mockVisitorRepoForReports{}
	smtpCfg := config.SMTP{} // unconfigured -> simulation in tests

	svc := service.NewVisitorReportService(reportRepo, visitorRepo, smtpCfg)
	ctx := context.Background()

	// 1. Get default schedule
	sched, err := svc.GetSchedule(ctx)
	if err != nil {
		t.Fatalf("failed to get schedule: %v", err)
	}
	if sched.Cadence != domain.ReportCadenceWeekly {
		t.Errorf("expected WEEKLY cadence by default, got %s", sched.Cadence)
	}

	// 2. Update schedule to Monthly with multiple recipients
	updated, err := svc.UpdateSchedule(
		ctx,
		"Monthly Executive Visitor Log",
		domain.ReportCadenceMonthly,
		[]string{"director@dextaafrica.com", "operations@dextaafrica.com"},
		"Executive",
		false,
		true,
	)
	if err != nil {
		t.Fatalf("failed to update schedule: %v", err)
	}

	if updated.Cadence != domain.ReportCadenceMonthly {
		t.Errorf("expected MONTHLY cadence, got %s", updated.Cadence)
	}
	if len(updated.RecipientEmails) != 2 {
		t.Errorf("expected 2 recipients, got %d", len(updated.RecipientEmails))
	}
	if updated.Department != "Executive" {
		t.Errorf("expected Executive department, got %s", updated.Department)
	}

	// 3. Validation failure for empty recipients
	_, err = svc.UpdateSchedule(
		ctx,
		"Monthly Executive Visitor Log",
		domain.ReportCadenceMonthly,
		[]string{"invalid-email"},
		"Executive",
		false,
		true,
	)
	if err == nil {
		t.Errorf("expected validation error for invalid email list, got nil")
	}
}

func TestVisitorReportService_TriggerAndGenerateReport(t *testing.T) {
	now := time.Now().UTC()
	visitorRepo := &mockVisitorRepoForReports{
		visitors: []domain.Visitor{
			{
				ID:             "v-1",
				PassCode:       "DXV-ABC123",
				FullName:       "John Executive",
				Phone:          "+2348011223344",
				Email:          "john@client.com",
				Company:        "Client Co",
				Purpose:        "Strategic Partnership",
				HostName:       "Jane Lead",
				Department:     "Executive",
				HasAppointment: true,
				VisitDate:      now,
				CheckInAt:      now.Add(-2 * time.Hour),
				Status:         domain.VisitorStatusCheckedIn,
			},
			{
				ID:             "v-2",
				PassCode:       "DXV-XYZ789",
				FullName:       "Alice Contractor",
				Phone:          "+2348099887766",
				Email:          "alice@contractor.com",
				Company:        "Build Squad",
				Purpose:        "Site Inspection",
				HostName:       "Bob Operations",
				Department:     "Operations",
				HasAppointment: false,
				VisitDate:      now,
				CheckInAt:      now.Add(-4 * time.Hour),
				Status:         domain.VisitorStatusCheckedOut,
			},
		},
	}

	reportRepo := &mockVisitorReportRepo{}
	smtpCfg := config.SMTP{} // unconfigured -> simulates delivery

	svc := service.NewVisitorReportService(reportRepo, visitorRepo, smtpCfg)
	ctx := context.Background()

	delivery, err := svc.TriggerReport(ctx, "admin@dextaafrica.com")
	if err != nil {
		t.Fatalf("failed to trigger report: %v", err)
	}

	if delivery.TotalVisitors != 2 {
		t.Errorf("expected 2 visitors in report, got %d", delivery.TotalVisitors)
	}
	if delivery.Status != "SENT" {
		t.Errorf("expected status SENT, got %s", delivery.Status)
	}

	// Check delivery listing
	deliveries, err := svc.ListDeliveries(ctx, 1, 10)
	if err != nil {
		t.Fatalf("failed to list deliveries: %v", err)
	}
	if deliveries.TotalItems != 1 {
		t.Errorf("expected 1 delivery in history, got %d", deliveries.TotalItems)
	}
}
