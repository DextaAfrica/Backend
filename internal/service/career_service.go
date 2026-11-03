package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/DextaAfrica/Backend/internal/apperror"
	"github.com/DextaAfrica/Backend/internal/domain"
)

type CareerService struct {
	repo        domain.CareerListingRepository
	revalidator Revalidator
}

func NewCareerService(repo domain.CareerListingRepository, revalidator Revalidator) *CareerService {
	return &CareerService{repo: repo, revalidator: revalidator}
}

func (s *CareerService) List(ctx context.Context, page, perPage int, publishedOnly bool) (domain.PageResult[domain.CareerListing], error) {
	result, err := s.repo.List(ctx, NormalizeListParams(page, perPage, publishedOnly))
	if err != nil {
		return domain.PageResult[domain.CareerListing]{}, apperror.Internal(fmt.Errorf("career: list: %w", err))
	}
	return result, nil
}

func (s *CareerService) GetBySlug(ctx context.Context, slug string) (*domain.CareerListing, error) {
	c, err := s.repo.GetBySlug(ctx, slug)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, apperror.NotFound("career listing")
	}
	if err != nil {
		return nil, apperror.Internal(fmt.Errorf("career: get by slug: %w", err))
	}
	return c, nil
}

func (s *CareerService) GetByID(ctx context.Context, id string) (*domain.CareerListing, error) {
	c, err := s.repo.GetByID(ctx, id)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, apperror.NotFound("career listing")
	}
	if err != nil {
		return nil, apperror.Internal(fmt.Errorf("career: get by id: %w", err))
	}
	return c, nil
}

func (s *CareerService) Create(ctx context.Context, c *domain.CareerListing) (*domain.CareerListing, error) {
	if !c.EmploymentType.Valid() {
		return nil, apperror.Validation("invalid career listing", map[string]string{"employmentType": "is not a recognized employment type"})
	}
	if err := s.assertSlugAvailable(ctx, c.Slug, ""); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, c); err != nil {
		if errors.Is(err, domain.ErrConflict) {
			return nil, apperror.Conflict("a listing with this slug already exists")
		}
		return nil, apperror.Internal(fmt.Errorf("career: create: %w", err))
	}
	s.revalidator.Trigger(ctx, "careers")
	return c, nil
}

func (s *CareerService) Update(ctx context.Context, c *domain.CareerListing) (*domain.CareerListing, error) {
	if !c.EmploymentType.Valid() {
		return nil, apperror.Validation("invalid career listing", map[string]string{"employmentType": "is not a recognized employment type"})
	}
	if err := s.assertSlugAvailable(ctx, c.Slug, c.ID); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, c); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, apperror.NotFound("career listing")
		}
		if errors.Is(err, domain.ErrConflict) {
			return nil, apperror.Conflict("a listing with this slug already exists")
		}
		return nil, apperror.Internal(fmt.Errorf("career: update: %w", err))
	}
	s.revalidator.Trigger(ctx, "careers")
	return c, nil
}

func (s *CareerService) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return apperror.NotFound("career listing")
		}
		return apperror.Internal(fmt.Errorf("career: delete: %w", err))
	}
	s.revalidator.Trigger(ctx, "careers")
	return nil
}

func (s *CareerService) assertSlugAvailable(ctx context.Context, slug, excludeID string) error {
	exists, err := s.repo.SlugExists(ctx, slug, excludeID)
	if err != nil {
		return apperror.Internal(fmt.Errorf("career: slug exists: %w", err))
	}
	if exists {
		return apperror.Conflict("a listing with this slug already exists")
	}
	return nil
}
