package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/DextaAfrica/Backend/internal/apperror"
	"github.com/DextaAfrica/Backend/internal/domain"
)

type DashboardAccessService struct {
	repo  domain.DashboardAccessRepository
	audit domain.AuditRepository
	admin domain.AdminRepository
}

func NewDashboardAccessService(
	repo domain.DashboardAccessRepository,
	audit domain.AuditRepository,
	admin domain.AdminRepository,
) *DashboardAccessService {
	return &DashboardAccessService{
		repo:  repo,
		audit: audit,
		admin: admin,
	}
}

func (s *DashboardAccessService) Create(ctx context.Context, req *domain.DashboardAccessRequest) error {
	if err := s.repo.Create(ctx, req); err != nil {
		return apperror.Internal(fmt.Errorf("dashboard_access_service: create: %w", err))
	}

	if s.audit != nil {
		_ = s.audit.Record(ctx, &domain.AuditLog{
			Action:     "DASHBOARD_ACCESS_REQUESTED",
			Entity:     "dashboard_access_requests",
			EntityID:   req.ID,
			ActorEmail: req.RequesterEmail,
			Metadata: map[string]any{
				"ticket_number": req.TicketNumber,
				"modules":       req.RequestedModules,
				"signature":     req.SignatureType,
			},
		})
	}
	return nil
}

func (s *DashboardAccessService) GetByID(ctx context.Context, id string) (*domain.DashboardAccessRequest, error) {
	req, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, apperror.NotFound("dashboard access request")
		}
		return nil, apperror.Internal(fmt.Errorf("dashboard_access_service: get by id: %w", err))
	}
	return req, nil
}

func (s *DashboardAccessService) List(ctx context.Context, status *domain.DashboardRequestStatus) ([]*domain.DashboardAccessRequest, error) {
	items, err := s.repo.List(ctx, status)
	if err != nil {
		return nil, apperror.Internal(fmt.Errorf("dashboard_access_service: list: %w", err))
	}
	return items, nil
}

func (s *DashboardAccessService) Review(
	ctx context.Context,
	id string,
	reviewerAdminID string,
	approved bool,
	approvedModules *string,
	itNotes *string,
) (*domain.DashboardAccessRequest, error) {
	req, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, apperror.NotFound("dashboard access request")
		}
		return nil, apperror.Internal(fmt.Errorf("dashboard_access_service: review lookup: %w", err))
	}

	now := time.Now().UTC()
	req.ReviewedByID = &reviewerAdminID
	req.ApprovedAt = &now
	req.ITNotes = itNotes

	if approved {
		req.Status = domain.DashboardStatusApproved
		req.ApprovedModules = approvedModules
		if req.ApprovedModules == nil || *req.ApprovedModules == "" {
			req.ApprovedModules = &req.RequestedModules
		}
	} else {
		req.Status = domain.DashboardStatusRejected
	}

	if err := s.repo.Update(ctx, req); err != nil {
		return nil, apperror.Internal(fmt.Errorf("dashboard_access_service: update review: %w", err))
	}

	if s.audit != nil {
		action := "DASHBOARD_ACCESS_APPROVED"
		if !approved {
			action = "DASHBOARD_ACCESS_REJECTED"
		}
		_ = s.audit.Record(ctx, &domain.AuditLog{
			Action:   action,
			Entity:   "dashboard_access_requests",
			EntityID: req.ID,
			ActorID:  reviewerAdminID,
			Metadata: map[string]any{
				"ticket_number": req.TicketNumber,
				"user_id":       req.UserID,
				"notes":         itNotes,
			},
		})
	}

	return req, nil
}
