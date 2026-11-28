package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/DextaAfrica/Backend/internal/apperror"
	"github.com/DextaAfrica/Backend/internal/authtoken"
	"github.com/DextaAfrica/Backend/internal/domain"
)

const (
	MaxFailedLoginAttempts = 5
	AccountLockoutDuration = 15 * time.Minute
	DefaultAccessTTL       = 15 * time.Minute
	DefaultRefreshTTL      = 7 * 24 * time.Hour
)

type AuthResult struct {
	AccessToken  string        `json:"accessToken"`
	RefreshToken string        `json:"refreshToken"`
	TokenType    string        `json:"tokenType"`
	ExpiresIn    int64         `json:"expiresIn"`
	Admin        *domain.Admin `json:"admin"`
}

type AuthService struct {
	admins    domain.AdminRepository
	sessions  domain.SessionRepository
	audit     domain.AuditRepository
	jwtSecret string
	accessTTL time.Duration
}

func NewAuthService(
	admins domain.AdminRepository,
	sessions domain.SessionRepository,
	audit domain.AuditRepository,
	jwtSecret string,
	accessTTL time.Duration,
) *AuthService {
	if accessTTL <= 0 {
		accessTTL = DefaultAccessTTL
	}
	return &AuthService{
		admins:    admins,
		sessions:  sessions,
		audit:     audit,
		jwtSecret: jwtSecret,
		accessTTL: accessTTL,
	}
}

func parseModules(modulesJSON string) []string {
	if modulesJSON == "" {
		return []string{"operations"}
	}
	var res []string
	if err := json.Unmarshal([]byte(modulesJSON), &res); err != nil {
		return []string{"operations"}
	}
	return res
}

func (s *AuthService) Login(ctx context.Context, email, password, ip, userAgent string) (*AuthResult, error) {
	admin, err := s.admins.GetByEmail(ctx, email)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, apperror.Unauthorized("invalid email or password")
	}
	if err != nil {
		return nil, apperror.Internal(fmt.Errorf("auth: lookup admin: %w", err))
	}

	now := time.Now().UTC()

	// 1. Account Lockout Protection (Brute Force / ATO defense)
	if admin.LockedUntil != nil && admin.LockedUntil.After(now) {
		remaining := time.Until(*admin.LockedUntil).Round(time.Second)
		if s.audit != nil {
			_ = s.audit.Record(ctx, &domain.AuditLog{
				Action:     "LOGIN_ATTEMPT_LOCKED_ACCOUNT",
				Entity:     "admins",
				EntityID:   admin.ID,
				ActorEmail: admin.Email,
				IPAddress:  ip,
				UserAgent:  userAgent,
				Metadata:   map[string]any{"remaining_seconds": remaining.Seconds()},
			})
		}
		return nil, apperror.New(apperror.CodeForbidden, fmt.Sprintf("account temporarily locked due to repeated failed login attempts; retry in %s", remaining))
	}

	// 2. Validate Password
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password)); err != nil {
		failed := admin.FailedLoginAttempts + 1
		var lockUntil *time.Time
		if failed >= MaxFailedLoginAttempts {
			l := now.Add(AccountLockoutDuration)
			lockUntil = &l
			slog.Warn("auth: account locked due to repeated failed logins", "email", email, "ip", ip)
		}
		_ = s.admins.UpdateLoginSecurity(ctx, admin.ID, failed, lockUntil, nil, ip)

		if s.audit != nil {
			_ = s.audit.Record(ctx, &domain.AuditLog{
				Action:     "LOGIN_FAILED_CREDENTIALS",
				Entity:     "admins",
				EntityID:   admin.ID,
				ActorEmail: admin.Email,
				IPAddress:  ip,
				UserAgent:  userAgent,
				Metadata:   map[string]any{"failed_attempts": failed, "locked": lockUntil != nil},
			})
		}
		return nil, apperror.Unauthorized("invalid email or password")
	}

	// 3. Reset security counters on success
	_ = s.admins.UpdateLoginSecurity(ctx, admin.ID, 0, nil, &now, ip)

	// 4. Issue Dual Tokens & Session
	rawRefresh, tokenHash, err := authtoken.GenerateRefreshToken()
	if err != nil {
		return nil, apperror.Internal(fmt.Errorf("auth: generate refresh token: %w", err))
	}

	session := &domain.AdminSession{
		AdminID:   admin.ID,
		TokenHash: tokenHash,
		IPAddress: ip,
		UserAgent: userAgent,
		Revoked:   false,
		ExpiresAt: now.Add(DefaultRefreshTTL),
	}
	if s.sessions != nil {
		if err := s.sessions.Create(ctx, session); err != nil {
			return nil, apperror.Internal(fmt.Errorf("auth: save session: %w", err))
		}
	}

	modules := parseModules(admin.AssignedModules)
	accessToken, err := authtoken.GenerateWithDetails(
		[]byte(s.jwtSecret),
		admin.ID,
		admin.Email,
		admin.Role,
		modules,
		session.ID,
		s.accessTTL,
	)
	if err != nil {
		return nil, apperror.Internal(fmt.Errorf("auth: generate access token: %w", err))
	}

	if s.audit != nil {
		_ = s.audit.Record(ctx, &domain.AuditLog{
			Action:     "LOGIN_SUCCESS",
			Entity:     "admins",
			EntityID:   admin.ID,
			ActorEmail: admin.Email,
			IPAddress:  ip,
			UserAgent:  userAgent,
			Metadata:   map[string]any{"session_id": session.ID, "role": admin.Role},
		})
	}

	return &AuthResult{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.accessTTL.Seconds()),
		Admin:        admin,
	}, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, rawRefreshToken, ip, userAgent string) (*AuthResult, error) {
	if rawRefreshToken == "" {
		return nil, apperror.Unauthorized("refresh token required")
	}

	tokenHash := authtoken.HashRefreshToken(rawRefreshToken)
	session, err := s.sessions.GetByTokenHash(ctx, tokenHash)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, apperror.Unauthorized("invalid refresh token")
	}
	if err != nil {
		return nil, apperror.Internal(fmt.Errorf("auth: get session: %w", err))
	}

	now := time.Now().UTC()

	// Token Reuse Detection: If an already revoked refresh token is presented,
	// assume theft and revoke ALL active sessions for this admin!
	if session.Revoked {
		_ = s.sessions.RevokeAllForAdmin(ctx, session.AdminID)
		if s.audit != nil {
			_ = s.audit.Record(ctx, &domain.AuditLog{
				Action:    "SECURITY_TOKEN_REUSE_DETECTED",
				Entity:    "admin_sessions",
				EntityID:  session.ID,
				ActorID:   session.AdminID,
				IPAddress: ip,
				UserAgent: userAgent,
				Metadata:  map[string]any{"alert": "all sessions revoked due to reuse detection"},
			})
		}
		return nil, apperror.Unauthorized("invalid or reused refresh token; session revoked")
	}

	if session.ExpiresAt.Before(now) {
		return nil, apperror.Unauthorized("expired refresh token")
	}

	admin, err := s.admins.GetByID(ctx, session.AdminID)
	if err != nil {
		return nil, apperror.Unauthorized("associated user account not found")
	}

	// Rotate refresh token (revoke current, create new)
	_ = s.sessions.Revoke(ctx, session.ID)

	newRawRefresh, newTokenHash, err := authtoken.GenerateRefreshToken()
	if err != nil {
		return nil, apperror.Internal(fmt.Errorf("auth: generate new refresh: %w", err))
	}

	newSession := &domain.AdminSession{
		AdminID:   admin.ID,
		TokenHash: newTokenHash,
		IPAddress: ip,
		UserAgent: userAgent,
		Revoked:   false,
		ExpiresAt: now.Add(DefaultRefreshTTL),
	}
	if err := s.sessions.Create(ctx, newSession); err != nil {
		return nil, apperror.Internal(fmt.Errorf("auth: create rotated session: %w", err))
	}

	modules := parseModules(admin.AssignedModules)
	accessToken, err := authtoken.GenerateWithDetails(
		[]byte(s.jwtSecret),
		admin.ID,
		admin.Email,
		admin.Role,
		modules,
		newSession.ID,
		s.accessTTL,
	)
	if err != nil {
		return nil, apperror.Internal(fmt.Errorf("auth: generate new access token: %w", err))
	}

	return &AuthResult{
		AccessToken:  accessToken,
		RefreshToken: newRawRefresh,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.accessTTL.Seconds()),
		Admin:        admin,
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, sessionID string) error {
	if s.sessions != nil && sessionID != "" {
		return s.sessions.Revoke(ctx, sessionID)
	}
	return nil
}

func (s *AuthService) RevokeAllSessions(ctx context.Context, adminID string) error {
	if s.sessions != nil && adminID != "" {
		return s.sessions.RevokeAllForAdmin(ctx, adminID)
	}
	return nil
}

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

	if err := s.admins.Create(ctx, &domain.Admin{
		Email:           email,
		PasswordHash:    string(hash),
		Role:            "SUPER_ADMIN",
		Department:      "SYSTEMS_OPERATIONS",
		JobTitle:        "System Super Administrator",
		AssignedModules: `["operations","approvals","personal","security"]`,
		Status:          "ACTIVE",
	}); err != nil {
		return fmt.Errorf("auth: create bootstrap admin: %w", err)
	}

	slog.Info("auth: bootstrap super administrator created", "email", email)
	return nil
}
