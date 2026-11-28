package domain

import (
	"context"
	"time"
)

type ReportCadence string

const (
	ReportCadenceWeekly  ReportCadence = "WEEKLY"
	ReportCadenceMonthly ReportCadence = "MONTHLY"
)

type VisitorReportSchedule struct {
	ID              string        `json:"id"`
	Title           string        `json:"title"`
	Cadence         ReportCadence `json:"cadence"`
	RecipientEmails []string      `json:"recipientEmails"`
	Department      string        `json:"department"`
	IncludeAllDepts bool          `json:"includeAllDepts"`
	IsActive        bool          `json:"isActive"`
	LastRunAt       *time.Time    `json:"lastRunAt,omitempty"`
	NextRunAt       time.Time     `json:"nextRunAt"`
	CreatedAt       time.Time     `json:"createdAt"`
	UpdatedAt       time.Time     `json:"updatedAt"`
}

type VisitorReportDelivery struct {
	ID              string        `json:"id"`
	ScheduleID      string        `json:"scheduleId"`
	Title           string        `json:"title"`
	Cadence         ReportCadence `json:"cadence"`
	RecipientEmails []string      `json:"recipientEmails"`
	PeriodStart     time.Time     `json:"periodStart"`
	PeriodEnd       time.Time     `json:"periodEnd"`
	TotalVisitors   int           `json:"totalVisitors"`
	Status          string        `json:"status"` // "SENT", "FAILED"
	ErrorMessage    *string       `json:"errorMessage,omitempty"`
	SentAt          time.Time     `json:"sentAt"`
}

type VisitorReportRepository interface {
	GetOrCreateSchedule(ctx context.Context) (*VisitorReportSchedule, error)
	UpdateSchedule(ctx context.Context, schedule *VisitorReportSchedule) error
	GetDueSchedules(ctx context.Context, now time.Time) ([]VisitorReportSchedule, error)
	RecordDelivery(ctx context.Context, delivery *VisitorReportDelivery) error
	ListDeliveries(ctx context.Context, page, perPage int) (PageResult[VisitorReportDelivery], error)
}
