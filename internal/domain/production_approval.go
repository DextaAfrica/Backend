package domain

import (
	"context"
	"time"
)

type ProductionReleaseStatus string

const (
	ProductionStatusPendingSuperAdmin ProductionReleaseStatus = "PENDING_SUPER_ADMIN"
	ProductionStatusApprovedLive      ProductionReleaseStatus = "APPROVED_LIVE"
	ProductionStatusRejected          ProductionReleaseStatus = "REJECTED"
)

type ProductionReleaseRiskLevel string

const (
	ProductionRiskLow      ProductionReleaseRiskLevel = "LOW"
	ProductionRiskMedium   ProductionReleaseRiskLevel = "MEDIUM"
	ProductionRiskHigh     ProductionReleaseRiskLevel = "HIGH"
	ProductionRiskCritical ProductionReleaseRiskLevel = "CRITICAL"
)

type ProductionReleaseRequest struct {
	ID               string                     `json:"id"`
	TicketNumber     string                     `json:"ticket_number"`
	Title            string                     `json:"title"`
	TargetSystem     string                     `json:"target_system"`
	Stage            int                        `json:"stage"` // 1 = Staged by Admin, 2 = Live / Authorized
	Status           ProductionReleaseStatus    `json:"status"`
	AdminID          string                     `json:"admin_id"`
	AdminName        string                     `json:"admin_name"`
	AdminEmail       string                     `json:"admin_email"`
	RiskLevel        ProductionReleaseRiskLevel `json:"risk_level"`
	Justification    string                     `json:"justification"`
	RollBackPlan     string                     `json:"rollback_plan"`
	DiffSummary      *string                    `json:"diff_summary,omitempty"`
	SuperAdminID     *string                    `json:"super_admin_id,omitempty"`
	SuperAdminName   *string                    `json:"super_admin_name,omitempty"`
	SuperAdminEmail  *string                    `json:"super_admin_email,omitempty"`
	SuperAdminNotes  *string                    `json:"super_admin_notes,omitempty"`
	ApprovedAt       *time.Time                 `json:"approved_at,omitempty"`
	CreatedAt        time.Time                  `json:"created_at"`
	UpdatedAt        time.Time                  `json:"updated_at"`
}

type ProductionReleaseStats struct {
	PendingCount  int `json:"pending_count"`
	ApprovedCount int `json:"approved_count"`
	RejectedCount int `json:"rejected_count"`
	TotalCount    int `json:"total_count"`
}

type ProductionApprovalRepository interface {
	Create(ctx context.Context, req *ProductionReleaseRequest) error
	GetByID(ctx context.Context, id string) (*ProductionReleaseRequest, error)
	List(ctx context.Context, params ListParams) (PageResult[ProductionReleaseRequest], error)
	GetStats(ctx context.Context) (*ProductionReleaseStats, error)
	UpdateDecision(ctx context.Context, id string, status ProductionReleaseStatus, stage int, superAdminID, superAdminName, superAdminEmail string, notes string) (*ProductionReleaseRequest, error)
}
