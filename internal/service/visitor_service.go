package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/DextaAfrica/Backend/internal/apperror"
	"github.com/DextaAfrica/Backend/internal/domain"
)

type VisitorService struct {
	repo domain.VisitorRepository
}

func NewVisitorService(repo domain.VisitorRepository) *VisitorService {
	return &VisitorService{repo: repo}
}

func generatePassCode() string {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 6)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			b[i] = charset[i%len(charset)]
			continue
		}
		b[i] = charset[n.Int64()]
	}
	return "DXV-" + string(b)
}

func (s *VisitorService) CheckIn(ctx context.Context, v *domain.Visitor) (*domain.Visitor, error) {
	v.FullName = strings.TrimSpace(v.FullName)
	v.Phone = strings.TrimSpace(v.Phone)
	v.Email = strings.TrimSpace(strings.ToLower(v.Email))
	v.Company = strings.TrimSpace(v.Company)
	v.Position = strings.TrimSpace(v.Position)
	v.Purpose = strings.TrimSpace(v.Purpose)
	v.PurposeOther = strings.TrimSpace(v.PurposeOther)
	v.HostName = strings.TrimSpace(v.HostName)
	v.Department = strings.TrimSpace(v.Department)
	v.InvitedBy = strings.TrimSpace(v.InvitedBy)
	v.VisitDescription = strings.TrimSpace(v.VisitDescription)

	fields := map[string]string{}
	if v.FullName == "" {
		fields["fullName"] = "Full name is required"
	}
	if v.Phone == "" {
		fields["phone"] = "Phone number is required"
	}
	if v.Email == "" || !strings.Contains(v.Email, "@") {
		fields["email"] = "A valid email address is required"
	}
	if v.Purpose == "" {
		fields["purpose"] = "Purpose of visit is required"
	}
	if strings.EqualFold(v.Purpose, "Other") && v.PurposeOther == "" {
		fields["purposeOther"] = "Please specify your purpose of visit"
	}
	if v.HostName == "" {
		fields["hostName"] = "Please specify who you are here to see"
	}
	if v.Department == "" {
		fields["department"] = "Please select the department you are visiting"
	}
	if !v.AgreedGuidelines {
		fields["agreedGuidelines"] = "You must agree to comply with company visitor guidelines and office regulations"
	}
	if !v.ConsentedData {
		fields["consentedData"] = "You must consent to the collection and processing of your registration information"
	}
	if len(fields) > 0 {
		return nil, apperror.Validation("please complete all required visitor registration fields", fields)
	}

	// Retry up to 3 times if a generated pass code collides
	var err error
	for i := 0; i < 3; i++ {
		v.PassCode = generatePassCode()
		err = s.repo.Create(ctx, v)
		if !errors.Is(err, domain.ErrConflict) {
			break
		}
	}
	if err != nil {
		return nil, apperror.Internal(fmt.Errorf("visitor_service: check in: %w", err))
	}
	return v, nil
}

func (s *VisitorService) GetByPassCode(ctx context.Context, passCode string) (*domain.Visitor, error) {
	v, err := s.repo.GetByPassCode(ctx, passCode)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, apperror.NotFound("visitor pass")
	}
	if err != nil {
		return nil, apperror.Internal(fmt.Errorf("visitor_service: get by pass code: %w", err))
	}
	return v, nil
}

func (s *VisitorService) SelfCheckOut(ctx context.Context, identifier string) (*domain.Visitor, error) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return nil, apperror.Validation("pass code, email, or phone number is required", map[string]string{
			"identifier": "Please enter your Visitor Pass Code, Email, or Phone Number",
		})
	}

	v, err := s.repo.FindActiveByContact(ctx, identifier)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, apperror.NotFound("active checked-in visitor record")
	}
	if err != nil {
		return nil, apperror.Internal(fmt.Errorf("visitor_service: find active: %w", err))
	}

	updated, err := s.repo.CheckOut(ctx, v.ID, "self-checkout")
	if err != nil {
		return nil, apperror.Internal(fmt.Errorf("visitor_service: self check out: %w", err))
	}
	return updated, nil
}

func (s *VisitorService) AdminCheckOut(ctx context.Context, id string, adminEmail string) (*domain.Visitor, error) {
	if strings.TrimSpace(adminEmail) == "" {
		adminEmail = "reception-admin"
	}
	updated, err := s.repo.CheckOut(ctx, id, adminEmail)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, apperror.NotFound("visitor")
	}
	if err != nil {
		return nil, apperror.Internal(fmt.Errorf("visitor_service: admin check out: %w", err))
	}
	return updated, nil
}

func (s *VisitorService) List(ctx context.Context, filter domain.VisitorFilter) (domain.PageResult[domain.Visitor], error) {
	res, err := s.repo.List(ctx, filter)
	if err != nil {
		return domain.PageResult[domain.Visitor]{}, apperror.Internal(fmt.Errorf("visitor_service: list: %w", err))
	}
	return res, nil
}

func (s *VisitorService) GetAnalytics(ctx context.Context) (*domain.VisitorAnalytics, error) {
	res, err := s.repo.GetAnalytics(ctx)
	if err != nil {
		return nil, apperror.Internal(fmt.Errorf("visitor_service: analytics: %w", err))
	}
	return res, nil
}
