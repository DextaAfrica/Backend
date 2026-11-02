package domain

import (
	"context"
	"time"
)

type EmploymentType string

const (
	EmploymentTypeFullTime EmploymentType = "full_time"
	EmploymentTypePartTime EmploymentType = "part_time"
	EmploymentTypeContract EmploymentType = "contract"
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
	ID             string
	Slug           string
	Title          string
	Department     string
	Location       string
	EmploymentType EmploymentType
	Description    map[string]any
	Published      bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type CareerListingRepository interface {
	Create(ctx context.Context, c *CareerListing) error
	Update(ctx context.Context, c *CareerListing) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*CareerListing, error)
	GetBySlug(ctx context.Context, slug string) (*CareerListing, error)
	List(ctx context.Context, params ListParams) (PageResult[CareerListing], error)
	SlugExists(ctx context.Context, slug string, excludeID string) (bool, error)
}
