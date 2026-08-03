package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/DextaAfrica/Backend/internal/apperror"
	"github.com/DextaAfrica/Backend/internal/domain"
)

type ContentService struct {
	pages       domain.PageRepository
	revalidator Revalidator
}

func NewContentService(pages domain.PageRepository, revalidator Revalidator) *ContentService {
	return &ContentService{pages: pages, revalidator: revalidator}
}

func (s *ContentService) GetPage(ctx context.Context, key string) (*domain.Page, error) {
	page, err := s.pages.GetByKey(ctx, key)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, apperror.NotFound("page")
	}
	if err != nil {
		return nil, apperror.Internal(fmt.Errorf("content: get page %q: %w", key, err))
	}
	return page, nil
}

func (s *ContentService) ListPages(ctx context.Context) ([]domain.Page, error) {
	pages, err := s.pages.List(ctx)
	if err != nil {
		return nil, apperror.Internal(fmt.Errorf("content: list pages: %w", err))
	}
	return pages, nil
}

func (s *ContentService) SavePage(ctx context.Context, page *domain.Page) (*domain.Page, error) {
	saved, err := s.pages.Upsert(ctx, page)
	if err != nil {
		return nil, apperror.Internal(fmt.Errorf("content: save page %q: %w", page.Key, err))
	}
	s.revalidator.Trigger(ctx, saved.Key)
	return saved, nil
}
