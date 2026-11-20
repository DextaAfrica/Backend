package domain

import (
	"context"
	"time"
)

type DashboardRequestStatus string

const (
	DashboardStatusPendingITApproval DashboardRequestStatus = "PENDING_IT_APPROVAL"
	DashboardStatusApproved          DashboardRequestStatus = "APPROVED"
	DashboardStatusRejected          DashboardRequestStatus = "REJECTED"
)

type DigitalSignatureType string

const (
	SignatureTypeDrawn    DigitalSignatureType = "DRAWN"
	SignatureTypeTyped    DigitalSignatureType = "TYPED"
	SignatureTypeUploaded DigitalSignatureType = "UPLOADED"
)

type DashboardAccessRequest struct {
	ID               string                 `json:"id"`
	TicketNumber     string                 `json:"ticket_number"`
	UserID           string                 `json:"user_id"`
	RequesterName    string                 `json:"requester_name"`
	RequesterEmail   string                 `json:"requester_email"`
	Department       string                 `json:"department"`
	JobTitle         string                 `json:"job_title"`
	RequestedModules string                 `json:"requested_modules"` // Serialized JSON array: ["campaigns", "treasury"]
	Justification    string                 `json:"justification"`
	SignatureData    string                 `json:"signature_data"`
	SignatureType    DigitalSignatureType   `json:"signature_type"`
	SignedAt         time.Time              `json:"signed_at"`
	Status           DashboardRequestStatus `json:"status"`
	ReviewedByID     *string                `json:"reviewed_by_id,omitempty"`
	ApprovedAt       *time.Time             `json:"approved_at,omitempty"`
	ApprovedModules  *string                `json:"approved_modules,omitempty"` // Serialized JSON array
	ITNotes          *string                `json:"it_notes,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

type DashboardAccessRepository interface {
	Create(ctx context.Context, req *DashboardAccessRequest) error
	GetByID(ctx context.Context, id string) (*DashboardAccessRequest, error)
	GetByTicket(ctx context.Context, ticket string) (*DashboardAccessRequest, error)
	List(ctx context.Context, status *DashboardRequestStatus) ([]*DashboardAccessRequest, error)
	ListByUser(ctx context.Context, userID string) ([]*DashboardAccessRequest, error)
	Update(ctx context.Context, req *DashboardAccessRequest) error
}
