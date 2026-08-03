package domain

import (
	"context"
	"time"
)

type EnquiryStatus string

const (
	EnquiryStatusNew      EnquiryStatus = "new"
	EnquiryStatusRead     EnquiryStatus = "read"
	EnquiryStatusArchived EnquiryStatus = "archived"
)

type Enquiry struct {
	ID         string
	Name       string
	Email      string
	Phone      string
	Subject    string
	Message    string
	SourcePage string
	Status     EnquiryStatus
	CreatedAt  time.Time
}

type EnquiryRepository interface {
	Create(ctx context.Context, e *Enquiry) error
	GetByID(ctx context.Context, id string) (*Enquiry, error)
	UpdateStatus(ctx context.Context, id string, status EnquiryStatus) error
	List(ctx context.Context, params ListParams) (PageResult[Enquiry], error)
}
