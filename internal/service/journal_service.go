package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/DextaAfrica/Backend/internal/apperror"
	"github.com/DextaAfrica/Backend/internal/domain"
)

type JournalService struct {
	repo        domain.ArticleRepository
	revalidator Revalidator
}

func NewJournalService(repo domain.ArticleRepository, revalidator Revalidator) *JournalService {
	return &JournalService{repo: repo, revalidator: revalidator}
}

func (s *JournalService) List(ctx context.Context, page, perPage int, publishedOnly bool) (domain.PageResult[domain.Article], error) {
	result, err := s.repo.List(ctx, NormalizeListParams(page, perPage, publishedOnly))
	if err != nil {
		return domain.PageResult[domain.Article]{}, apperror.Internal(fmt.Errorf("journal: list: %w", err))
	}
	return result, nil
}

func (s *JournalService) GetBySlug(ctx context.Context, slug string) (*domain.Article, error) {
	a, err := s.repo.GetBySlug(ctx, slug)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, apperror.NotFound("article")
	}
	if err != nil {
		return nil, apperror.Internal(fmt.Errorf("journal: get by slug: %w", err))
	}
	return a, nil
}

func (s *JournalService) GetByID(ctx context.Context, id string) (*domain.Article, error) {
	a, err := s.repo.GetByID(ctx, id)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, apperror.NotFound("article")
	}
	if err != nil {
		return nil, apperror.Internal(fmt.Errorf("journal: get by id: %w", err))
	}
	return a, nil
}

func (s *JournalService) Create(ctx context.Context, a *domain.Article) (*domain.Article, error) {
	if err := s.assertSlugAvailable(ctx, a.Slug, ""); err != nil {
		return nil, err
	}
	if a.Published {
		now := time.Now()
		a.PublishedAt = &now
	}
	if err := s.repo.Create(ctx, a); err != nil {
		if errors.Is(err, domain.ErrConflict) {
			return nil, apperror.Conflict("an article with this slug already exists")
		}
		return nil, apperror.Internal(fmt.Errorf("journal: create: %w", err))
	}
	s.revalidator.Trigger(ctx, "journal")
	return a, nil
}

func (s *JournalService) Update(ctx context.Context, a *domain.Article) (*domain.Article, error) {
	if err := s.assertSlugAvailable(ctx, a.Slug, a.ID); err != nil {
		return nil, err
	}

	existing, err := s.GetByID(ctx, a.ID)
	if err != nil {
		return nil, err
	}
	if a.Published && !existing.Published {
		now := time.Now()
		a.PublishedAt = &now
	} else {
		a.PublishedAt = existing.PublishedAt
	}

	if err := s.repo.Update(ctx, a); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, apperror.NotFound("article")
		}
		if errors.Is(err, domain.ErrConflict) {
			return nil, apperror.Conflict("an article with this slug already exists")
		}
		return nil, apperror.Internal(fmt.Errorf("journal: update: %w", err))
	}
	s.revalidator.Trigger(ctx, "journal")
	s.revalidator.Trigger(ctx, "journal:"+a.Slug)
	return a, nil
}

func (s *JournalService) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return apperror.NotFound("article")
		}
		return apperror.Internal(fmt.Errorf("journal: delete: %w", err))
	}
	s.revalidator.Trigger(ctx, "journal")
	return nil
}

func (s *JournalService) assertSlugAvailable(ctx context.Context, slug, excludeID string) error {
	exists, err := s.repo.SlugExists(ctx, slug, excludeID)
	if err != nil {
		return apperror.Internal(fmt.Errorf("journal: slug exists: %w", err))
	}
	if exists {
		return apperror.Conflict("an article with this slug already exists")
	}
	return nil
}
