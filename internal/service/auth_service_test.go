package service

import (
	"context"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/DextaAfrica/Backend/internal/domain"
)

type mockAdminRepo struct {
	admins map[string]*domain.Admin
}

func (m *mockAdminRepo) Create(ctx context.Context, admin *domain.Admin) error {
	m.admins[admin.Email] = admin
	return nil
}

func (m *mockAdminRepo) GetByEmail(ctx context.Context, email string) (*domain.Admin, error) {
	if a, ok := m.admins[email]; ok {
		return a, nil
	}
	return nil, domain.ErrNotFound
}

func (m *mockAdminRepo) GetByID(ctx context.Context, id string) (*domain.Admin, error) {
	for _, a := range m.admins {
		if a.ID == id {
			return a, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *mockAdminRepo) Count(ctx context.Context) (int, error) {
	return len(m.admins), nil
}

func (m *mockAdminRepo) List(ctx context.Context, params domain.ListParams) (domain.PageResult[domain.Admin], error) {
	items := make([]domain.Admin, 0)
	for _, a := range m.admins {
		items = append(items, *a)
	}
	return domain.PageResult[domain.Admin]{Items: items, TotalItems: len(items)}, nil
}

func (m *mockAdminRepo) Upsert(ctx context.Context, admin *domain.Admin) error {
	m.admins[admin.Email] = admin
	return nil
}

func (m *mockAdminRepo) UpdateLoginSecurity(ctx context.Context, id string, failedAttempts int, lockedUntil *time.Time, lastLoginAt *time.Time, lastLoginIP string) error {
	for _, a := range m.admins {
		if a.ID == id {
			a.FailedLoginAttempts = failedAttempts
			a.LockedUntil = lockedUntil
			if lastLoginAt != nil {
				a.LastLoginAt = lastLoginAt
			}
			if lastLoginIP != "" {
				a.LastLoginIP = lastLoginIP
			}
			return nil
		}
	}
	return domain.ErrNotFound
}

type mockSessionRepo struct {
	sessions map[string]*domain.AdminSession
}

func (m *mockSessionRepo) Create(ctx context.Context, s *domain.AdminSession) error {
	s.ID = "session-" + s.TokenHash[:8]
	m.sessions[s.TokenHash] = s
	return nil
}

func (m *mockSessionRepo) GetByTokenHash(ctx context.Context, hash string) (*domain.AdminSession, error) {
	if s, ok := m.sessions[hash]; ok {
		return s, nil
	}
	return nil, domain.ErrNotFound
}

func (m *mockSessionRepo) Revoke(ctx context.Context, id string) error {
	for _, s := range m.sessions {
		if s.ID == id {
			s.Revoked = true
			return nil
		}
	}
	return domain.ErrNotFound
}

func (m *mockSessionRepo) RevokeAllForAdmin(ctx context.Context, adminID string) error {
	for _, s := range m.sessions {
		if s.AdminID == adminID {
			s.Revoked = true
		}
	}
	return nil
}

func (m *mockSessionRepo) PruneExpired(ctx context.Context) (int64, error) {
	return 0, nil
}

func TestAuthService_DualTokensAndLockout(t *testing.T) {
	ctx := context.Background()
	hash, _ := bcrypt.GenerateFromPassword([]byte("StrongPassword2026!"), bcrypt.DefaultCost)
	admin := &domain.Admin{
		ID:              "admin-uuid-1",
		Email:           "security@dextaafrica.com",
		PasswordHash:    string(hash),
		Role:            "SUPER_ADMIN",
		AssignedModules: `["operations","approvals"]`,
	}

	adminRepo := &mockAdminRepo{admins: map[string]*domain.Admin{admin.Email: admin}}
	sessionRepo := &mockSessionRepo{sessions: make(map[string]*domain.AdminSession)}
	authSvc := NewAuthService(adminRepo, sessionRepo, nil, "super-secret-key-at-least-32-chars", 15*time.Minute)

	// 1. Successful Login returns dual tokens
	result, err := authSvc.Login(ctx, "security@dextaafrica.com", "StrongPassword2026!", "127.0.0.1", "TestAgent")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if result.AccessToken == "" || result.RefreshToken == "" {
		t.Fatalf("expected both access and refresh tokens, got access=%v, refresh=%v", result.AccessToken, result.RefreshToken)
	}

	// 2. Rotate refresh token
	refreshed, err := authSvc.RefreshToken(ctx, result.RefreshToken, "127.0.0.1", "TestAgent")
	if err != nil {
		t.Fatalf("token refresh failed: %v", err)
	}
	if refreshed.AccessToken == "" || refreshed.RefreshToken == "" {
		t.Fatalf("refresh result missing tokens")
	}

	// 3. Token Reuse Detection: using old refresh token triggers complete invalidation
	_, err = authSvc.RefreshToken(ctx, result.RefreshToken, "127.0.0.1", "TestAgent")
	if err == nil {
		t.Fatal("expected error on reused refresh token, got nil")
	}

	// 4. Brute force lockout: 5 consecutive failures locks account
	for i := 1; i <= 5; i++ {
		_, _ = authSvc.Login(ctx, "security@dextaafrica.com", "WrongPassword!", "127.0.0.1", "Attacker")
	}
	// 6th attempt should be rejected due to account lockout
	_, err = authSvc.Login(ctx, "security@dextaafrica.com", "StrongPassword2026!", "127.0.0.1", "Attacker")
	if err == nil {
		t.Fatal("expected account lockout rejection, got success")
	}
}
