package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/DextaAfrica/Backend/internal/apperror"
	"github.com/DextaAfrica/Backend/internal/domain"
)

type EnquiryService struct {
	repo domain.EnquiryRepository
}

func NewEnquiryService(repo domain.EnquiryRepository) *EnquiryService {
	return &EnquiryService{repo: repo}
}

func (s *EnquiryService) Submit(ctx context.Context, e *domain.Enquiry) (*domain.Enquiry, error) {
	e.Status = domain.EnquiryStatusNew
	if err := s.repo.Create(ctx, e); err != nil {
		return nil, apperror.Internal(fmt.Errorf("enquiry: submit: %w", err))
	}
	return e, nil
}

func (s *EnquiryService) List(ctx context.Context, page, perPage int) (domain.PageResult[domain.Enquiry], error) {
	result, err := s.repo.List(ctx, NormalizeListParams(page, perPage, false))
	if err != nil {
		return domain.PageResult[domain.Enquiry]{}, apperror.Internal(fmt.Errorf("enquiry: list: %w", err))
	}
	return result, nil
}

func (s *EnquiryService) GetByID(ctx context.Context, id string) (*domain.Enquiry, error) {
	e, err := s.repo.GetByID(ctx, id)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, apperror.NotFound("enquiry")
	}
	if err != nil {
		return nil, apperror.Internal(fmt.Errorf("enquiry: get by id: %w", err))
	}
	return e, nil
}

func (s *EnquiryService) UpdateStatus(ctx context.Context, id string, status domain.EnquiryStatus) error {
	switch status {
	case domain.EnquiryStatusNew, domain.EnquiryStatusRead, domain.EnquiryStatusArchived:
	default:
		return apperror.Validation("invalid status", map[string]string{"status": "is not a recognized status"})
	}

	if err := s.repo.UpdateStatus(ctx, id, status); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return apperror.NotFound("enquiry")
		}
		return apperror.Internal(fmt.Errorf("enquiry: update status: %w", err))
	}
	return nil
}
