package service

import (
	"context"
	"fmt"

	"github.com/DextaAfrica/Backend/internal/apperror"
	"github.com/DextaAfrica/Backend/internal/domain"
)

type AuditService struct {
	repo domain.AuditRepository
}

func NewAuditService(repo domain.AuditRepository) *AuditService {
	return &AuditService{repo: repo}
}

func (s *AuditService) Record(ctx context.Context, log *domain.AuditLog) error {
	if log.Action == "" || log.Entity == "" {
		return apperror.New(apperror.CodeBadRequest, "action and entity are required")
	}
	if err := s.repo.Record(ctx, log); err != nil {
		return apperror.Internal(fmt.Errorf("audit_service: record: %w", err))
	}
	return nil
}

func (s *AuditService) List(ctx context.Context, page, perPage int) (domain.PageResult[domain.AuditLog], error) {
	result, err := s.repo.List(ctx, NormalizeListParams(page, perPage, false))
	if err != nil {
		return domain.PageResult[domain.AuditLog]{}, apperror.Internal(fmt.Errorf("audit_service: list: %w", err))
	}
	return result, nil
}
