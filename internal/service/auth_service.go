package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/DextaAfrica/Backend/internal/apperror"
	"github.com/DextaAfrica/Backend/internal/authtoken"
	"github.com/DextaAfrica/Backend/internal/domain"
)

type AuthService struct {
	admins      domain.AdminRepository
	jwtSecret   string
	accessTTL   time.Duration
}

func NewAuthService(admins domain.AdminRepository, jwtSecret string, accessTTL time.Duration) *AuthService {
	return &AuthService{admins: admins, jwtSecret: jwtSecret, accessTTL: accessTTL}
}

// Login verifies credentials and issues a session JWT. The same generic
// error is returned whether the email doesn't exist or the password is
// wrong, so the endpoint never leaks which admin emails are registered.
func (s *AuthService) Login(ctx context.Context, email, password string) (string, error) {
	admin, err := s.admins.GetByEmail(ctx, email)
	if errors.Is(err, domain.ErrNotFound) {
		return "", apperror.Unauthorized("invalid email or password")
	}
	if err != nil {
		return "", apperror.Internal(fmt.Errorf("auth: lookup admin: %w", err))
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password)); err != nil {
		return "", apperror.Unauthorized("invalid email or password")
	}

	token, err := authtoken.Generate([]byte(s.jwtSecret), admin.ID, admin.Email, s.accessTTL)
	if err != nil {
		return "", apperror.Internal(fmt.Errorf("auth: generate token: %w", err))
	}
	return token, nil
}

// BootstrapAdmin seeds the first admin account from ADMIN_BOOTSTRAP_EMAIL /
// ADMIN_BOOTSTRAP_PASSWORD on startup if (and only if) no admin exists yet.
// This is how the CMS gets its first login without a public registration
// endpoint — there is deliberately no self-service admin signup.
func (s *AuthService) BootstrapAdmin(ctx context.Context, email, password string) error {
	if email == "" || password == "" {
		return nil
	}

	count, err := s.admins.Count(ctx)
	if err != nil {
		return fmt.Errorf("auth: count admins: %w", err)
	}
	if count > 0 {
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("auth: hash bootstrap password: %w", err)
	}

	if err := s.admins.Create(ctx, &domain.Admin{Email: email, PasswordHash: string(hash)}); err != nil {
		return fmt.Errorf("auth: create bootstrap admin: %w", err)
	}

	slog.Info("auth: bootstrap admin created", "email", email)
	return nil
}
