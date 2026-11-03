package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/DextaAfrica/Backend/internal/apperror"
	"github.com/DextaAfrica/Backend/internal/domain"
)

type PortfolioService struct {
	repo        domain.DevelopmentRepository
	revalidator Revalidator
}

func NewPortfolioService(repo domain.DevelopmentRepository, revalidator Revalidator) *PortfolioService {
	return &PortfolioService{repo: repo, revalidator: revalidator}
}

func (s *PortfolioService) List(ctx context.Context, page, perPage int, publishedOnly bool) (domain.PageResult[domain.Development], error) {
	result, err := s.repo.List(ctx, NormalizeListParams(page, perPage, publishedOnly))
	if err != nil {
		return domain.PageResult[domain.Development]{}, apperror.Internal(fmt.Errorf("portfolio: list: %w", err))
	}
	return result, nil
}

func (s *PortfolioService) GetBySlug(ctx context.Context, slug string) (*domain.Development, error) {
	d, err := s.repo.GetBySlug(ctx, slug)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, apperror.NotFound("development")
	}
	if err != nil {
		return nil, apperror.Internal(fmt.Errorf("portfolio: get by slug: %w", err))
	}
	return d, nil
}

func (s *PortfolioService) GetByID(ctx context.Context, id string) (*domain.Development, error) {
	d, err := s.repo.GetByID(ctx, id)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, apperror.NotFound("development")
	}
	if err != nil {
		return nil, apperror.Internal(fmt.Errorf("portfolio: get by id: %w", err))
	}
	return d, nil
}

func (s *PortfolioService) Create(ctx context.Context, d *domain.Development) (*domain.Development, error) {
	if !d.Status.Valid() {
		return nil, apperror.Validation("invalid development", map[string]string{"status": "is not a recognized status"})
	}
	if err := s.assertSlugAvailable(ctx, d.Slug, ""); err != nil {
		return nil, err
	}

	if d.Published {
		now := time.Now()
		d.PublishedAt = &now
	}

	if err := s.repo.Create(ctx, d); err != nil {
		if errors.Is(err, domain.ErrConflict) {
			return nil, apperror.Conflict("a development with this slug already exists")
		}
		return nil, apperror.Internal(fmt.Errorf("portfolio: create: %w", err))
	}
	s.revalidator.Trigger(ctx, "portfolio")
	return d, nil
}

func (s *PortfolioService) Update(ctx context.Context, d *domain.Development) (*domain.Development, error) {
	if !d.Status.Valid() {
		return nil, apperror.Validation("invalid development", map[string]string{"status": "is not a recognized status"})
	}
	if err := s.assertSlugAvailable(ctx, d.Slug, d.ID); err != nil {
		return nil, err
	}

	existing, err := s.GetByID(ctx, d.ID)
	if err != nil {
		return nil, err
	}
	if d.Published && !existing.Published {
		now := time.Now()
		d.PublishedAt = &now
	} else {
		d.PublishedAt = existing.PublishedAt
	}

	if err := s.repo.Update(ctx, d); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, apperror.NotFound("development")
		}
		if errors.Is(err, domain.ErrConflict) {
			return nil, apperror.Conflict("a development with this slug already exists")
		}
		return nil, apperror.Internal(fmt.Errorf("portfolio: update: %w", err))
	}
	s.revalidator.Trigger(ctx, "portfolio")
	s.revalidator.Trigger(ctx, "portfolio:"+d.Slug)
	return d, nil
}

func (s *PortfolioService) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return apperror.NotFound("development")
		}
		return apperror.Internal(fmt.Errorf("portfolio: delete: %w", err))
	}
	s.revalidator.Trigger(ctx, "portfolio")
	return nil
}

func (s *PortfolioService) assertSlugAvailable(ctx context.Context, slug, excludeID string) error {
	exists, err := s.repo.SlugExists(ctx, slug, excludeID)
	if err != nil {
		return apperror.Internal(fmt.Errorf("portfolio: slug exists: %w", err))
	}
	if exists {
		return apperror.Conflict("a development with this slug already exists")
	}
	return nil
}
