package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/DextaAfrica/Backend/internal/apperror"
	"github.com/DextaAfrica/Backend/internal/domain"
)

type FormService struct {
	repo domain.FormRepository
}

func NewFormService(repo domain.FormRepository) *FormService {
	return &FormService{repo: repo}
}

func sanitizeSlug(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.ReplaceAll(s, " ", "-")
	return s
}

func (s *FormService) CreateForm(ctx context.Context, f *domain.CustomForm) (*domain.CustomForm, error) {
	f.Slug = sanitizeSlug(f.Slug)
	if f.Slug == "" {
		f.Slug = sanitizeSlug(f.Title)
	}
	if f.Status == "" {
		f.Status = domain.FormStatusActive
	}
	if f.FormType == "" {
		f.FormType = domain.FormTypeEvent
	}

	err := s.repo.Create(ctx, f)
	if errors.Is(err, domain.ErrConflict) {
		return nil, apperror.Conflict("a registration link with this slug already exists")
	}
	if err != nil {
		return nil, apperror.Internal(fmt.Errorf("form_service: create: %w", err))
	}
	return f, nil
}

func (s *FormService) UpdateForm(ctx context.Context, f *domain.CustomForm) (*domain.CustomForm, error) {
	f.Slug = sanitizeSlug(f.Slug)
	err := s.repo.Update(ctx, f)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, apperror.NotFound("form")
	}
	if errors.Is(err, domain.ErrConflict) {
		return nil, apperror.Conflict("slug is already in use by another form")
	}
	if err != nil {
		return nil, apperror.Internal(fmt.Errorf("form_service: update: %w", err))
	}
	return f, nil
}

func (s *FormService) DeleteForm(ctx context.Context, id string) error {
	err := s.repo.Delete(ctx, id)
	if errors.Is(err, domain.ErrNotFound) {
		return apperror.NotFound("form")
	}
	if err != nil {
		return apperror.Internal(fmt.Errorf("form_service: delete: %w", err))
	}
	return nil
}

func (s *FormService) GetByID(ctx context.Context, id string) (*domain.CustomForm, error) {
	f, err := s.repo.GetByID(ctx, id)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, apperror.NotFound("form")
	}
	if err != nil {
		return nil, apperror.Internal(fmt.Errorf("form_service: get by id: %w", err))
	}
	return f, nil
}

func (s *FormService) GetBySlug(ctx context.Context, slug string) (*domain.CustomForm, error) {
	f, err := s.repo.GetBySlug(ctx, sanitizeSlug(slug))
	if errors.Is(err, domain.ErrNotFound) {
		return nil, apperror.NotFound("form")
	}
	if err != nil {
		return nil, apperror.Internal(fmt.Errorf("form_service: get by slug: %w", err))
	}
	return f, nil
}

func (s *FormService) ListForms(ctx context.Context, page, perPage int) (domain.PageResult[domain.CustomForm], error) {
	params := NormalizeListParams(page, perPage, false)
	res, err := s.repo.List(ctx, params)
	if err != nil {
		return domain.PageResult[domain.CustomForm]{}, apperror.Internal(fmt.Errorf("form_service: list: %w", err))
	}
	return res, nil
}

func (s *FormService) Submit(ctx context.Context, slug string, sub *domain.FormSubmission) (*domain.FormSubmission, error) {
	form, err := s.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if form.Status != domain.FormStatusActive {
		return nil, apperror.New(apperror.CodeBadRequest, "this registration portal is currently closed")
	}

	sub.FormID = form.ID
	if sub.Category == "" {
		sub.Category = "Attendee"
	}
	if sub.AttendanceMode == "" {
		sub.AttendanceMode = "in-person"
	}
	if sub.Status == "" {
		sub.Status = "registered"
	}

	if err := s.repo.CreateSubmission(ctx, sub); err != nil {
		return nil, apperror.Internal(fmt.Errorf("form_service: submit: %w", err))
	}
	return sub, nil
}

func (s *FormService) ListSubmissions(ctx context.Context, formID string, page, perPage int) (domain.PageResult[domain.FormSubmission], error) {
	params := NormalizeListParams(page, perPage, false)
	res, err := s.repo.ListSubmissions(ctx, formID, params)
	if err != nil {
		return domain.PageResult[domain.FormSubmission]{}, apperror.Internal(fmt.Errorf("form_service: list submissions: %w", err))
	}
	return res, nil
}

func (s *FormService) GetDemographics(ctx context.Context, formID string) (*domain.FormDemographics, error) {
	demo, err := s.repo.GetDemographics(ctx, formID)
	if err != nil {
		return nil, apperror.Internal(fmt.Errorf("form_service: demographics: %w", err))
	}
	return demo, nil
}
