package middleware

import (
	"strings"

	"github.com/DextaAfrica/Backend/internal/config"
	"github.com/DextaAfrica/Backend/internal/domain/identity"
	"github.com/DextaAfrica/Backend/internal/interfaces/http/response"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// Authenticate verifies the Bearer JWT token and sets user context.
func Authenticate(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			return response.Error(c, fiber.StatusUnauthorized, "Missing or invalid authorization header", "UNAUTHORIZED", nil)
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.ErrUnauthorized
			}
			return []byte(cfg.JWT.Secret), nil
		})

		if err != nil || !token.Valid {
			return response.Error(c, fiber.StatusUnauthorized, "Invalid or expired access token", "INVALID_TOKEN", nil)
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return response.Error(c, fiber.StatusUnauthorized, "Invalid token claims", "INVALID_TOKEN", nil)
		}

		userID, _ := claims["user_id"].(string)
		role, _ := claims["role"].(string)
		email, _ := claims["email"].(string)

		c.Locals("user_id", userID)
		c.Locals("user_role", role)
		c.Locals("user_email", email)

		return c.Next()
	}
}

// RequireRoles restricts access to users possessing at least one of the specified roles.
func RequireRoles(allowedRoles ...identity.Role) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userRoleStr, ok := c.Locals("user_role").(string)
		if !ok || userRoleStr == "" {
			return response.Error(c, fiber.StatusForbidden, "Access denied. Insufficient permissions.", "FORBIDDEN", nil)
		}

		userRole := identity.Role(userRoleStr)

		// SuperAdmin has full access to all protected resources
		if userRole == identity.RoleSuperAdmin {
			return c.Next()
		}

		for _, role := range allowedRoles {
			if userRole == role {
				return c.Next()
			}
		}

		return response.Error(c, fiber.StatusForbidden, "Access denied. You do not have permission to access this resource.", "FORBIDDEN", nil)
	}
}

