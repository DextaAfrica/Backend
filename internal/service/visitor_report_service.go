package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/csv"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/DextaAfrica/Backend/internal/apperror"
	"github.com/DextaAfrica/Backend/internal/config"
	"github.com/DextaAfrica/Backend/internal/domain"
)

type VisitorReportService struct {
	reportRepo  domain.VisitorReportRepository
	visitorRepo domain.VisitorRepository
	smtpCfg     config.SMTP
}

func NewVisitorReportService(
	reportRepo domain.VisitorReportRepository,
	visitorRepo domain.VisitorRepository,
	smtpCfg config.SMTP,
) *VisitorReportService {
	return &VisitorReportService{
		reportRepo:  reportRepo,
		visitorRepo: visitorRepo,
		smtpCfg:     smtpCfg,
	}
}

func (s *VisitorReportService) GetSchedule(ctx context.Context) (*domain.VisitorReportSchedule, error) {
	return s.reportRepo.GetOrCreateSchedule(ctx)
}

func calculateNextRun(cadence domain.ReportCadence, from time.Time) time.Time {
	switch cadence {
	case domain.ReportCadenceMonthly:
		return from.AddDate(0, 1, 0)
	case domain.ReportCadenceWeekly:
		fallthrough
	default:
		return from.AddDate(0, 0, 7)
	}
}

func (s *VisitorReportService) UpdateSchedule(
	ctx context.Context,
	title string,
	cadence domain.ReportCadence,
	recipients []string,
	department string,
	includeAll bool,
	isActive bool,
) (*domain.VisitorReportSchedule, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "Visitor Check-in Digest"
	}

	if cadence != domain.ReportCadenceWeekly && cadence != domain.ReportCadenceMonthly {
		cadence = domain.ReportCadenceWeekly
	}

	cleanedRecipients := make([]string, 0, len(recipients))
	for _, r := range recipients {
		r = strings.TrimSpace(strings.ToLower(r))
		if r != "" && strings.Contains(r, "@") {
			cleanedRecipients = append(cleanedRecipients, r)
		}
	}
	if len(cleanedRecipients) == 0 {
		return nil, apperror.Validation("at least one valid recipient email is required", map[string]string{
			"recipientEmails": "Please specify at least one recipient email address",
		})
	}

	current, err := s.reportRepo.GetOrCreateSchedule(ctx)
	if err != nil {
		return nil, fmt.Errorf("visitor_report_service: get schedule: %w", err)
	}

	now := time.Now().UTC()
	current.Title = title
	current.Cadence = cadence
	current.RecipientEmails = cleanedRecipients
	current.Department = strings.TrimSpace(department)
	current.IncludeAllDepts = includeAll
	current.IsActive = isActive
	if current.NextRunAt.Before(now) {
		current.NextRunAt = calculateNextRun(cadence, now)
	}

	if err := s.reportRepo.UpdateSchedule(ctx, current); err != nil {
		return nil, fmt.Errorf("visitor_report_service: update schedule: %w", err)
	}
	return current, nil
}

func (s *VisitorReportService) ListDeliveries(ctx context.Context, page, perPage int) (domain.PageResult[domain.VisitorReportDelivery], error) {
	return s.reportRepo.ListDeliveries(ctx, page, perPage)
}

func generateDeliveryID() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 12)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			b[i] = charset[i%len(charset)]
			continue
		}
		b[i] = charset[n.Int64()]
	}
	return "vrd-" + string(b)
}

func sanitizeCSV(s string) string {
	if s == "" {
		return s
	}
	first := s[0]
	if first == '=' || first == '@' || first == '\t' || first == '\r' {
		return "'" + s
	}
	if (first == '+' || first == '-') && !strings.HasPrefix(s, "+") {
		return "'" + s
	}
	return s
}

func (s *VisitorReportService) GenerateAndSendReport(
	ctx context.Context,
	schedule *domain.VisitorReportSchedule,
	triggerSource string,
) (*domain.VisitorReportDelivery, error) {
	now := time.Now().UTC()
	var periodStart time.Time
	if schedule.Cadence == domain.ReportCadenceMonthly {
		periodStart = now.AddDate(0, -1, 0)
	} else {
		periodStart = now.AddDate(0, 0, -7)
	}

	filter := domain.VisitorFilter{
		StartDate: &periodStart,
		EndDate:   &now,
		Page:      1,
		PerPage:   1000,
	}
	if !schedule.IncludeAllDepts && schedule.Department != "" && schedule.Department != "ALL" {
		filter.Department = schedule.Department
	}

	result, err := s.visitorRepo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("visitor_report_service: query visitors: %w", err)
	}

	visitors := result.Items
	totalVisitors := len(visitors)
	checkedInCount := 0
	checkedOutCount := 0
	deptMap := make(map[string]int)
	purposeMap := make(map[string]int)

	for _, v := range visitors {
		if v.Status == domain.VisitorStatusCheckedIn {
			checkedInCount++
		} else {
			checkedOutCount++
		}
		dept := v.Department
		if dept == "" {
			dept = "Unassigned"
		}
		deptMap[dept]++

		purpose := v.Purpose
		if purpose == "" {
			purpose = "General"
		}
		purposeMap[purpose]++
	}

	// Build CSV Buffer
	var csvBuf bytes.Buffer
	cw := csv.NewWriter(&csvBuf)
	_ = cw.Write([]string{
		"Pass Code", "Full Name", "Phone", "Email", "Company", "Position",
		"Purpose", "Host Name", "Department", "Appointment", "Check In Time",
		"Check Out Time", "Status",
	})
	for _, v := range visitors {
		checkOut := ""
		if v.CheckOutAt != nil {
			checkOut = v.CheckOutAt.Format(time.RFC3339)
		}
		_ = cw.Write([]string{
			sanitizeCSV(v.PassCode),
			sanitizeCSV(v.FullName),
			sanitizeCSV(v.Phone),
			sanitizeCSV(v.Email),
			sanitizeCSV(v.Company),
			sanitizeCSV(v.Position),
			sanitizeCSV(v.Purpose),
			sanitizeCSV(v.HostName),
			sanitizeCSV(v.Department),
			fmt.Sprintf("%t", v.HasAppointment),
			v.CheckInAt.Format(time.RFC3339),
			checkOut,
			string(v.Status),
		})
	}
	cw.Flush()

	// Build HTML Email
	cadenceLabel := "Weekly"
	if schedule.Cadence == domain.ReportCadenceMonthly {
		cadenceLabel = "Monthly"
	}

	dateRangeStr := fmt.Sprintf("%s to %s", periodStart.Format("Jan 02, 2006"), now.Format("Jan 02, 2006"))
	subject := fmt.Sprintf("Dexta Africa - %s Visitor Check-In Report (%s)", cadenceLabel, dateRangeStr)

	deptListHTML := ""
	for dept, count := range deptMap {
		deptListHTML += fmt.Sprintf(`<tr><td style="padding: 6px 12px; border-bottom: 1px solid #1e293b; color: #cbd5e1;">%s</td><td style="padding: 6px 12px; border-bottom: 1px solid #1e293b; color: #f8fafc; text-align: right; font-weight: 700;">%d</td></tr>`, dept, count)
	}
	if deptListHTML == "" {
		deptListHTML = `<tr><td colspan="2" style="padding: 12px; text-align: center; color: #94a3b8;">No visitor arrivals recorded during this period.</td></tr>`
	}

	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"><title>%s</title></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #070A0F; color: #f8fafc; padding: 24px; margin: 0;">
  <div style="max-width: 640px; margin: 0 auto; background: #0d111a; border: 1px solid #1e293b; border-radius: 16px; overflow: hidden;">
    <div style="background: linear-gradient(135deg, #ED2618 0%%, #991b1b 100%%); padding: 24px 28px;">
      <p style="margin: 0; font-size: 11px; text-transform: uppercase; letter-spacing: 0.15em; color: #fecaca; font-weight: 700;">Dexta Africa Reception &amp; Security</p>
      <h1 style="margin: 6px 0 0; font-size: 22px; color: #ffffff;">%s Visitor Check-In Digest</h1>
      <p style="margin: 4px 0 0; font-size: 13px; color: #fee2e2;">Period: %s</p>
    </div>
    <div style="padding: 28px;">
      <div style="display: flex; gap: 12px; margin-bottom: 24px;">
        <div style="flex: 1; background: #141923; border: 1px solid #263042; border-radius: 12px; padding: 16px; text-align: center;">
          <div style="font-size: 11px; text-transform: uppercase; color: #94a3b8; font-weight: 700;">Total Arrivals</div>
          <div style="font-size: 28px; font-weight: 800; color: #ffffff; margin-top: 4px;">%d</div>
        </div>
        <div style="flex: 1; background: #141923; border: 1px solid #263042; border-radius: 12px; padding: 16px; text-align: center;">
          <div style="font-size: 11px; text-transform: uppercase; color: #94a3b8; font-weight: 700;">Departed</div>
          <div style="font-size: 28px; font-weight: 800; color: #10b981; margin-top: 4px;">%d</div>
        </div>
        <div style="flex: 1; background: #141923; border: 1px solid #263042; border-radius: 12px; padding: 16px; text-align: center;">
          <div style="font-size: 11px; text-transform: uppercase; color: #94a3b8; font-weight: 700;">Active On-Site</div>
          <div style="font-size: 28px; font-weight: 800; color: #ED2618; margin-top: 4px;">%d</div>
        </div>
      </div>

      <h3 style="font-size: 14px; text-transform: uppercase; letter-spacing: 0.1em; color: #cbd5e1; margin: 24px 0 12px; border-bottom: 1px solid #1e293b; padding-bottom: 8px;">Visits by Department</h3>
      <table style="width: 100%%; border-collapse: collapse; font-size: 13px; margin-bottom: 24px;">
        <thead>
          <tr style="background: #141923;">
            <th style="padding: 8px 12px; text-align: left; color: #94a3b8; border-bottom: 1px solid #263042;">Department</th>
            <th style="padding: 8px 12px; text-align: right; color: #94a3b8; border-bottom: 1px solid #263042;">Visits</th>
          </tr>
        </thead>
        <tbody>
          %s
        </tbody>
      </table>

      <div style="background: #141923; border: 1px dashed #334155; border-radius: 12px; padding: 16px; font-size: 12px; color: #94a3b8;">
        📎 A full CSV spreadsheet containing every visitor pass, arrival timestamp, host, and contact details has been attached to this email (<strong>dexta-visitors-%s.csv</strong>).
      </div>

      <div style="margin-top: 28px; padding-top: 16px; border-top: 1px solid #1e293b; font-size: 11px; color: #64748b; text-align: center;">
        Automated Enterprise Report • Triggered by %s • Dexta Africa Security Operations
      </div>
    </div>
  </div>
</body>
</html>
`, subject, cadenceLabel, dateRangeStr, totalVisitors, checkedOutCount, checkedInCount, deptListHTML, now.Format("2006-01-02"), triggerSource)

	// Send email or simulate
	deliveryStatus := "SENT"
	var deliveryError *string

	sendErr := s.sendReportEmail(schedule.RecipientEmails, subject, htmlBody, csvBuf.Bytes(), fmt.Sprintf("dexta-visitors-%s.csv", now.Format("2006-01-02")))
	if sendErr != nil {
		slog.Warn("visitor report delivery notice", "error", sendErr)
		errStr := sendErr.Error()
		deliveryError = &errStr
		if !s.smtpCfg.IsConfigured() {
			deliveryStatus = "SENT" // Simulated success in local development
			simMsg := "Simulated delivery (SMTP relay not configured for local environment)"
			deliveryError = &simMsg
		} else {
			deliveryStatus = "FAILED"
		}
	}

	delivery := &domain.VisitorReportDelivery{
		ID:              generateDeliveryID(),
		ScheduleID:      schedule.ID,
		Title:           schedule.Title,
		Cadence:         schedule.Cadence,
		RecipientEmails: schedule.RecipientEmails,
		PeriodStart:     periodStart,
		PeriodEnd:       now,
		TotalVisitors:   totalVisitors,
		Status:          deliveryStatus,
		ErrorMessage:    deliveryError,
		SentAt:          now,
	}

	if err := s.reportRepo.RecordDelivery(ctx, delivery); err != nil {
		slog.Error("failed to record report delivery", "error", err)
	}

	// Advance NextRunAt
	schedule.LastRunAt = &now
	schedule.NextRunAt = calculateNextRun(schedule.Cadence, now)
	if err := s.reportRepo.UpdateSchedule(ctx, schedule); err != nil {
		slog.Error("failed to update next run for schedule", "error", err)
	}

	return delivery, nil
}

func (s *VisitorReportService) sendReportEmail(
	recipients []string,
	subject string,
	htmlBody string,
	csvAttachment []byte,
	attachmentFilename string,
) error {
	if !s.smtpCfg.IsConfigured() {
		return errors.New("smtp relay not configured")
	}

	boundary := "==DEXTA_MULTIPART_BOUNDARY_987654321=="
	var msg bytes.Buffer

	from := s.smtpCfg.From
	if from == "" {
		from = "operations@dextaafrica.com"
	}

	msg.WriteString(fmt.Sprintf("From: %s\r\n", from))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(recipients, ", ")))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n", boundary))
	msg.WriteString("\r\n")

	// HTML Body Part
	msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	msg.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	msg.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(htmlBody)
	msg.WriteString("\r\n\r\n")

	// CSV Attachment Part
	msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	msg.WriteString("Content-Type: text/csv; charset=UTF-8\r\n")
	msg.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n", attachmentFilename))
	msg.WriteString("Content-Transfer-Encoding: base64\r\n")
	msg.WriteString("\r\n")

	b64Data := base64.StdEncoding.EncodeToString(csvAttachment)
	for i := 0; i < len(b64Data); i += 76 {
		end := i + 76
		if end > len(b64Data) {
			end = len(b64Data)
		}
		msg.WriteString(b64Data[i:end] + "\r\n")
	}
	msg.WriteString(fmt.Sprintf("--%s--\r\n", boundary))

	addr := net.JoinHostPort(s.smtpCfg.Host, fmt.Sprintf("%d", s.smtpCfg.Port))
	var auth smtp.Auth
	if s.smtpCfg.User != "" && s.smtpCfg.Password != "" {
		auth = smtp.PlainAuth("", s.smtpCfg.User, s.smtpCfg.Password, s.smtpCfg.Host)
	}

	// Dial with timeout to prevent hanging connections
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("connect to smtp %s: %w", addr, err)
	}
	_ = conn.Close()

	return smtp.SendMail(addr, auth, from, recipients, msg.Bytes())
}

func (s *VisitorReportService) TriggerReport(ctx context.Context, adminEmail string) (*domain.VisitorReportDelivery, error) {
	schedule, err := s.reportRepo.GetOrCreateSchedule(ctx)
	if err != nil {
		return nil, fmt.Errorf("visitor_report_service: get schedule: %w", err)
	}
	triggerSource := "Admin Console"
	if adminEmail != "" {
		triggerSource = fmt.Sprintf("Admin (%s)", adminEmail)
	}
	return s.GenerateAndSendReport(ctx, schedule, triggerSource)
}

func (s *VisitorReportService) ProcessDueSchedules(ctx context.Context) error {
	now := time.Now().UTC()
	schedules, err := s.reportRepo.GetDueSchedules(ctx, now)
	if err != nil {
		return fmt.Errorf("visitor_report_service: fetch due schedules: %w", err)
	}

	for i := range schedules {
		sched := &schedules[i]
		slog.Info("processing scheduled visitor report", "schedule_id", sched.ID, "cadence", sched.Cadence)
		if _, err := s.GenerateAndSendReport(ctx, sched, "Automated Scheduler"); err != nil {
			slog.Error("failed to generate scheduled visitor report", "schedule_id", sched.ID, "error", err)
		}
	}
	return nil
}
