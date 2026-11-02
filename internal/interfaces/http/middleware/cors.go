package middleware

import (
	"github.com/DextaAfrica/Backend/internal/config"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// CORS configures cross-origin resource sharing for Web and Admin clients.
func CORS(cfg *config.Config) fiber.Handler {
	origins := cfg.App.FrontendWebURL + ", " + cfg.App.FrontendAdminURL
	if cfg.App.Environment == "development" {
		origins += ", http://localhost:3000, http://localhost:3001, http://localhost:5173"
	}

	return cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Trace-ID, X-Request-ID, Idempotency-Key",
		AllowMethods:     "GET, POST, PUT, PATCH, DELETE, OPTIONS",
		AllowCredentials: true,
		MaxAge:           86400,
	})
}

