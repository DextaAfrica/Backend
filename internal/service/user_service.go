package service

import (
	"context"
	"fmt"

	"github.com/DextaAfrica/Backend/internal/apperror"
	"github.com/DextaAfrica/Backend/internal/domain"
)

type UserService struct {
	repo domain.AdminRepository
}

func NewUserService(repo domain.AdminRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) SyncUser(ctx context.Context, admin *domain.Admin) error {
	if admin.Email == "" {
		return apperror.New(apperror.CodeBadRequest, "email is required")
	}
	if err := s.repo.Upsert(ctx, admin); err != nil {
		return apperror.Internal(fmt.Errorf("user_service: sync: %w", err))
	}
	return nil
}

func (s *UserService) List(ctx context.Context, page, perPage int) (domain.PageResult[domain.Admin], error) {
	result, err := s.repo.List(ctx, NormalizeListParams(page, perPage, false))
	if err != nil {
		return domain.PageResult[domain.Admin]{}, apperror.Internal(fmt.Errorf("user_service: list: %w", err))
	}
	return result, nil
}
