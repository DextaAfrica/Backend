package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/DextaAfrica/Backend/internal/apperror"
	"github.com/DextaAfrica/Backend/internal/domain"
)

type ProductionApprovalService struct {
	repo domain.ProductionApprovalRepository
}

func NewProductionApprovalService(repo domain.ProductionApprovalRepository) *ProductionApprovalService {
	return &ProductionApprovalService{repo: repo}
}

type CreateProductionReleaseInput struct {
	TicketNumber  string
	Title         string
	TargetSystem  string
	AdminID       string
	AdminName     string
	AdminEmail    string
	RiskLevel     string
	Justification string
	RollBackPlan  string
	DiffSummary   *string
}

func (s *ProductionApprovalService) Create(ctx context.Context, input CreateProductionReleaseInput) (*domain.ProductionReleaseRequest, error) {
	if strings.TrimSpace(input.Title) == "" {
		return nil, apperror.Validation("invalid input", map[string]string{"title": "title is required"})
	}
	if strings.TrimSpace(input.TargetSystem) == "" {
		return nil, apperror.Validation("invalid input", map[string]string{"target_system": "target system is required"})
	}
	if strings.TrimSpace(input.Justification) == "" {
		return nil, apperror.Validation("invalid input", map[string]string{"justification": "justification is required"})
	}
	if strings.TrimSpace(input.RollBackPlan) == "" {
		return nil, apperror.Validation("invalid input", map[string]string{"rollback_plan": "rollback plan is required"})
	}

	ticket := strings.TrimSpace(input.TicketNumber)
	if ticket == "" {
		b := make([]byte, 3)
		_, _ = rand.Read(b)
		ticket = fmt.Sprintf("REL-%s-%s", time.Now().Format("20060102"), strings.ToUpper(hex.EncodeToString(b)))
	}

	risk := domain.ProductionReleaseRiskLevel(strings.ToUpper(strings.TrimSpace(input.RiskLevel)))
	switch risk {
	case domain.ProductionRiskLow, domain.ProductionRiskMedium, domain.ProductionRiskHigh, domain.ProductionRiskCritical:
	default:
		risk = domain.ProductionRiskMedium
	}

	req := &domain.ProductionReleaseRequest{
		TicketNumber:  ticket,
		Title:         input.Title,
		TargetSystem:  input.TargetSystem,
		Stage:         1,
		Status:        domain.ProductionStatusPendingSuperAdmin,
		AdminID:       input.AdminID,
		AdminName:     input.AdminName,
		AdminEmail:    input.AdminEmail,
		RiskLevel:     risk,
		Justification: input.Justification,
		RollBackPlan:  input.RollBackPlan,
		DiffSummary:   input.DiffSummary,
	}

	if err := s.repo.Create(ctx, req); err != nil {
		return nil, err
	}
	return req, nil
}

func (s *ProductionApprovalService) GetByID(ctx context.Context, id string) (*domain.ProductionReleaseRequest, error) {
	req, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, apperror.NotFound("production release request")
		}
		return nil, err
	}
	return req, nil
}

func (s *ProductionApprovalService) List(ctx context.Context, params domain.ListParams) (domain.PageResult[domain.ProductionReleaseRequest], error) {
	return s.repo.List(ctx, params)
}

func (s *ProductionApprovalService) GetStats(ctx context.Context) (*domain.ProductionReleaseStats, error) {
	return s.repo.GetStats(ctx)
}

func (s *ProductionApprovalService) Decide(
	ctx context.Context,
	id string,
	action string,
	superAdminID, superAdminName, superAdminEmail string,
	notes string,
) (*domain.ProductionReleaseRequest, error) {
	var status domain.ProductionReleaseStatus
	var stage int

	switch strings.ToUpper(strings.TrimSpace(action)) {
	case "APPROVE", "APPROVED", "APPROVED_LIVE":
		status = domain.ProductionStatusApprovedLive
		stage = 2
	case "REJECT", "REJECTED":
		status = domain.ProductionStatusRejected
		stage = 1
	default:
		return nil, apperror.Validation("invalid action", map[string]string{
			"action": "action must be either APPROVE or REJECT",
		})
	}

	req, err := s.repo.UpdateDecision(ctx, id, status, stage, superAdminID, superAdminName, superAdminEmail, notes)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, apperror.NotFound("production release request")
		}
		return nil, err
	}
	return req, nil
}
