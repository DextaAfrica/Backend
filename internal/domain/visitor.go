package domain

import (
	"context"
	"time"
)

type VisitorStatus string

const (
	VisitorStatusCheckedIn  VisitorStatus = "checked_in"
	VisitorStatusCheckedOut VisitorStatus = "checked_out"
)

type Visitor struct {
	ID               string
	PassCode         string
	FullName         string
	Phone            string
	Email            string
	Company          string
	Position         string
	Purpose          string
	PurposeOther     string
	HostName         string
	Department       string
	HasAppointment   bool
	InvitedBy        string
	VisitDescription string
	VisitDate        time.Time
	CheckInAt        time.Time
	CheckOutAt       *time.Time
	Status           VisitorStatus
	AgreedGuidelines bool
	ConsentedData    bool
	DeclaredAt       time.Time
	IPAddress        string
	UserAgent        string
	CheckedOutBy     string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type VisitorFilter struct {
	Status     string
	Department string
	Purpose    string
	Date       string // YYYY-MM-DD
	Query      string
	Page       int
	PerPage    int
}

type VisitorAnalytics struct {
	TotalToday        int                `json:"totalToday"`
	CurrentlyOnSite   int                `json:"currentlyOnSite"`
	CheckedOutToday   int                `json:"checkedOutToday"`
	ScheduledToday    int                `json:"scheduledToday"`
	WalkInToday       int                `json:"walkInToday"`
	TotalAllTime      int                `json:"totalAllTime"`
	ByDepartment      []DemographicCount `json:"byDepartment"`
	ByPurpose         []DemographicCount `json:"byPurpose"`
	HourlyArrivals    []DemographicCount `json:"hourlyArrivals"`
}

type VisitorRepository interface {
	Create(ctx context.Context, visitor *Visitor) error
	GetByID(ctx context.Context, id string) (*Visitor, error)
	GetByPassCode(ctx context.Context, passCode string) (*Visitor, error)
	FindActiveByContact(ctx context.Context, identifier string) (*Visitor, error)
	CheckOut(ctx context.Context, id string, checkedOutBy string) (*Visitor, error)
	List(ctx context.Context, filter VisitorFilter) (PageResult[Visitor], error)
	GetAnalytics(ctx context.Context) (*VisitorAnalytics, error)
}
