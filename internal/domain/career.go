package domain

import (
	"context"
	"time"
)

type EmploymentType string

const (
	EmploymentTypeFullTime   EmploymentType = "full_time"
	EmploymentTypePartTime   EmploymentType = "part_time"
	EmploymentTypeContract   EmploymentType = "contract"
	EmploymentTypeInternship EmploymentType = "internship"
)

func (t EmploymentType) Valid() bool {
	switch t {
	case EmploymentTypeFullTime, EmploymentTypePartTime, EmploymentTypeContract, EmploymentTypeInternship:
		return true
	default:
		return false
	}
}

type CareerListing struct {
	ID                  string
	Slug                string
	Title               string
	Department          string
	Location            string
	EmploymentType      EmploymentType
	Description         map[string]any
	Status              string
	Published           bool
	ApplicantClickCount int
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type CareerInquiry struct {
	ID              string
	CareerListingID string
	ApplicantName   string
	ApplicantEmail  string
	ApplicantPhone  string
	ResumeURL       string
	CoverLetter     string
	Source          string
	CreatedAt       time.Time
}

type CareerListingRepository interface {
	Create(ctx context.Context, c *CareerListing) error
	Update(ctx context.Context, c *CareerListing) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*CareerListing, error)
	GetBySlug(ctx context.Context, slug string) (*CareerListing, error)
	List(ctx context.Context, params ListParams) (PageResult[CareerListing], error)
	SlugExists(ctx context.Context, slug string, excludeID string) (bool, error)
	IncrementClickCount(ctx context.Context, id string) error
	RecordInquiry(ctx context.Context, inq *CareerInquiry) error
	ListInquiries(ctx context.Context, listingID string) ([]CareerInquiry, error)
}
