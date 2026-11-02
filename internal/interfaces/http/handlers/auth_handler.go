package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/DextaAfrica/Backend/internal/config"
	"github.com/DextaAfrica/Backend/internal/domain/identity"
	"github.com/DextaAfrica/Backend/internal/infrastructure/database"
	"github.com/DextaAfrica/Backend/internal/interfaces/http/response"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	db  *database.PostgresDB
	cfg *config.Config
}

func NewAuthHandler(db *database.PostgresDB, cfg *config.Config) *AuthHandler {
	return &AuthHandler{db: db, cfg: cfg}
}

type RegisterRequest struct {
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	PhoneNumber string `json:"phone_number"`
	CountryCode string `json:"country_code"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type TokenResponse struct {
	AccessToken  string         `json:"access_token"`
	RefreshToken string         `json:"refresh_token"`
	ExpiresIn    int64          `json:"expires_in"` // seconds
	TokenType    string         `json:"token_type"`
	User         *identity.User `json:"user"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Register creates a new client investor account.
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", "INVALID_INPUT", nil)
	}

	if req.Email == "" || req.Password == "" || req.FirstName == "" || req.LastName == "" {
		return response.Error(c, fiber.StatusUnprocessableEntity, "Required registration fields missing", "VALIDATION_ERROR", nil)
	}

	// Check if email already registered
	var existing identity.User
	if err := h.db.DB.Where("email = ?", req.Email).First(&existing).Error; err == nil {
		return response.Error(c, fiber.StatusConflict, "Email address is already registered", "EMAIL_EXISTS", nil)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to process credentials", "ENCRYPTION_ERROR", nil)
	}

	user := identity.User{
		Email:        req.Email,
		PasswordHash: string(hash),
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		PhoneNumber:  req.PhoneNumber,
		CountryCode:  req.CountryCode,
		Role:         identity.RoleClient,
		Status:       identity.StatusActive,
	}

	if err := h.db.DB.Create(&user).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to create user account", "DATABASE_ERROR", nil)
	}

	tokens, err := h.generateTokens(&user, c.IP(), c.Get("User-Agent"))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to generate authorization tokens", "AUTH_ERROR", nil)
	}

	return response.Success(c, fiber.StatusCreated, "Account registered successfully", tokens, nil)
}

// Login authenticates staff or clients and issues JWT + Refresh Token.
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", "INVALID_INPUT", nil)
	}

	var user identity.User
	if err := h.db.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "Invalid email or password", "INVALID_CREDENTIALS", nil)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "Invalid email or password", "INVALID_CREDENTIALS", nil)
	}

	if user.Status != identity.StatusActive {
		return response.Error(c, fiber.StatusForbidden, "Account is suspended or pending verification", "ACCOUNT_INACTIVE", nil)
	}

	now := time.Now().UTC()
	user.LastLoginAt = &now
	h.db.DB.Model(&user).Update("last_login_at", now)

	tokens, err := h.generateTokens(&user, c.IP(), c.Get("User-Agent"))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to generate session tokens", "AUTH_ERROR", nil)
	}

	return response.Success(c, fiber.StatusOK, "Authentication successful", tokens, nil)
}

// RefreshToken exchanges an active refresh token for a renewed access token.
func (h *AuthHandler) RefreshToken(c *fiber.Ctx) error {
	var req RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", "INVALID_INPUT", nil)
	}

	hashedToken := hashToken(req.RefreshToken)

	var tokenRecord identity.RefreshToken
	err := h.db.DB.Where("token_hash = ? AND revoked_at IS NULL AND expires_at > ?", hashedToken, time.Now()).First(&tokenRecord).Error
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "Refresh token is invalid or expired", "INVALID_REFRESH_TOKEN", nil)
	}

	var user identity.User
	if err := h.db.DB.Where("id = ?", tokenRecord.UserID).First(&user).Error; err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "User account not found", "USER_NOT_FOUND", nil)
	}

	// Revoke old refresh token (rotating pattern)
	now := time.Now()
	tokenRecord.RevokedAt = &now
	h.db.DB.Save(&tokenRecord)

	tokens, err := h.generateTokens(&user, c.IP(), c.Get("User-Agent"))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to rotate tokens", "AUTH_ERROR", nil)
	}

	return response.Success(c, fiber.StatusOK, "Token rotated successfully", tokens, nil)
}

// GetMe returns current authenticated user details.
func (h *AuthHandler) GetMe(c *fiber.Ctx) error {
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok || userIDStr == "" {
		return response.Error(c, fiber.StatusUnauthorized, "Authentication required", "UNAUTHORIZED", nil)
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid user ID", "INVALID_ID", nil)
	}

	var user identity.User
	if err := h.db.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "User profile not found", "NOT_FOUND", nil)
	}

	return response.Success(c, fiber.StatusOK, "User profile retrieved", user, nil)
}

func (h *AuthHandler) generateTokens(user *identity.User, ip string, userAgent string) (*TokenResponse, error) {
	// Access Token
	claims := jwt.MapClaims{
		"user_id": user.ID.String(),
		"email":   user.Email,
		"role":    string(user.Role),
		"exp":     time.Now().Add(h.cfg.JWT.AccessTokenExpiry).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedAccessToken, err := token.SignedString([]byte(h.cfg.JWT.Secret))
	if err != nil {
		return nil, err
	}

	// Refresh Token
	rawRefreshToken := uuid.New().String() + "-" + uuid.New().String()
	refreshTokenRecord := identity.RefreshToken{
		UserID:            user.ID,
		TokenHash:         hashToken(rawRefreshToken),
		DeviceFingerprint: hashToken(ip + "|" + userAgent),
		IPAddress:         ip,
		ExpiresAt:         time.Now().Add(h.cfg.JWT.RefreshTokenExpiry),
	}

	if err := h.db.DB.Create(&refreshTokenRecord).Error; err != nil {
		return nil, err
	}

	return &TokenResponse{
		AccessToken:  signedAccessToken,
		RefreshToken: rawRefreshToken,
		ExpiresIn:    int64(h.cfg.JWT.AccessTokenExpiry.Seconds()),
		TokenType:    "Bearer",
		User:         user,
	}, nil
}

func hashToken(token string) string {
	hasher := sha256.New()
	hasher.Write([]byte(token))
	return hex.EncodeToString(hasher.Sum(nil))
}

