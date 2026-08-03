package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/DextaAfrica/Backend/internal/apperror"
	"github.com/DextaAfrica/Backend/internal/domain"
)

type NewsletterService struct {
	repo      domain.SubscriberRepository
	forwarder NewsletterForwarder
}

func NewNewsletterService(repo domain.SubscriberRepository, forwarder NewsletterForwarder) *NewsletterService {
	return &NewsletterService{repo: repo, forwarder: forwarder}
}

func (s *NewsletterService) Subscribe(ctx context.Context, email string) (*domain.Subscriber, error) {
	sub, err := s.repo.Upsert(ctx, email)
	if err != nil {
		return nil, apperror.Internal(fmt.Errorf("newsletter: subscribe: %w", err))
	}
	s.forwarder.Forward(ctx, email)
	return sub, nil
}

func (s *NewsletterService) Unsubscribe(ctx context.Context, email string) error {
	if err := s.repo.Unsubscribe(ctx, email); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return apperror.NotFound("subscriber")
		}
		return apperror.Internal(fmt.Errorf("newsletter: unsubscribe: %w", err))
	}
	return nil
}

func (s *NewsletterService) List(ctx context.Context, page, perPage int) (domain.PageResult[domain.Subscriber], error) {
	result, err := s.repo.List(ctx, NormalizeListParams(page, perPage, false))
	if err != nil {
		return domain.PageResult[domain.Subscriber]{}, apperror.Internal(fmt.Errorf("newsletter: list: %w", err))
	}
	return result, nil
}
