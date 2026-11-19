package domain

import (
	"context"
	"time"
)

type FormType string

const (
	FormTypeEvent      FormType = "event"
	FormTypeOnboarding FormType = "onboarding"
	FormTypePartner    FormType = "partner"
	FormTypeSurvey     FormType = "survey"
)

type FormStatus string

const (
	FormStatusActive FormStatus = "active"
	FormStatusClosed FormStatus = "closed"
	FormStatusDraft  FormStatus = "draft"
)

type CustomForm struct {
	ID               string
	Slug             string
	Title            string
	Description      string
	FormType         FormType
	Status           FormStatus
	CoverImageURL    string
	EventDate        *time.Time
	EventLocation    string
	Fields           []byte
	Metadata         []byte
	SubmissionsCount int
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type FormSubmission struct {
	ID             string
	FormID         string
	FullName       string
	Email          string
	Phone          string
	Company        string
	JobTitle       string
	Category       string
	AttendanceMode string
	Answers        []byte
	Notes          string
	Status         string
	CreatedAt      time.Time
}

type DemographicCount struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

type FormDemographics struct {
	TotalSubmissions  int                `json:"total_submissions"`
	ByCategory        []DemographicCount `json:"by_category"`
	ByAttendanceMode  []DemographicCount `json:"by_attendance_mode"`
	TopOrganizations  []DemographicCount `json:"top_organizations"`
	RecentSubmissions []FormSubmission   `json:"recent_submissions"`
}

type FormRepository interface {
	Create(ctx context.Context, form *CustomForm) error
	GetByID(ctx context.Context, id string) (*CustomForm, error)
	GetBySlug(ctx context.Context, slug string) (*CustomForm, error)
	Update(ctx context.Context, form *CustomForm) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, params ListParams) (PageResult[CustomForm], error)

	CreateSubmission(ctx context.Context, sub *FormSubmission) error
	ListSubmissions(ctx context.Context, formID string, params ListParams) (PageResult[FormSubmission], error)
	GetDemographics(ctx context.Context, formID string) (*FormDemographics, error)
}
