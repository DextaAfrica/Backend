package http

import (
	"github.com/DextaAfrica/Backend/internal/config"
	"github.com/DextaAfrica/Backend/internal/domain/identity"
	"github.com/DextaAfrica/Backend/internal/infrastructure/cache"
	"github.com/DextaAfrica/Backend/internal/infrastructure/database"
	"github.com/DextaAfrica/Backend/internal/interfaces/http/handlers"
	"github.com/DextaAfrica/Backend/internal/interfaces/http/middleware"

	"github.com/gofiber/fiber/v2"
)

// SetupRouter configures all middlewares and HTTP routes for Dexta Africa.
func SetupRouter(
	app *fiber.App,
	cfg *config.Config,
	db *database.PostgresDB,
	redisClient *cache.RedisClient,
) {
	// Global Middlewares
	app.Use(middleware.Recover())
	app.Use(middleware.RequestID())
	app.Use(middleware.HTTPLogger())
	app.Use(middleware.CORS(cfg))

	if cfg.RateLimiter.Enabled && redisClient != nil {
		app.Use(middleware.RateLimiter(redisClient, cfg.RateLimiter.MaxReqs, cfg.RateLimiter.WindowSec))
	}

	// Initialize Handlers
	healthHandler := handlers.NewHealthHandler(db, redisClient)
	propertyHandler := handlers.NewPropertyHandler(db)
	crmHandler := handlers.NewCRMHandler(db)
	authHandler := handlers.NewAuthHandler(db, cfg)

	// Base & Health check routes
	app.Get("/health", healthHandler.Check)
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"platform": "Dexta Africa Enterprise Real Estate Engine",
			"status":   "OPERATIONAL",
			"version":  "1.0.0",
			"docs":     "/docs",
		})
	})

	// API v1 Group
	api := app.Group("/api/v1")

	// Authentication Endpoints
	auth := api.Group("/auth")
	auth.Post("/register", authHandler.Register)
	auth.Post("/login", authHandler.Login)
	auth.Post("/refresh", authHandler.RefreshToken)
	auth.Get("/me", middleware.Authenticate(cfg), authHandler.GetMe)

	// Public Real Estate Discovery Endpoints (For Dynamic Customer Web Portal)
	properties := api.Group("/properties")
	properties.Get("/developments", propertyHandler.ListDevelopments)
	properties.Get("/developments/:slug", propertyHandler.GetDevelopment)
	properties.Get("/units", propertyHandler.ListUnits)
	properties.Get("/units/:id", propertyHandler.GetUnit)
	properties.Post("/units/:id/reserve", propertyHandler.ReserveUnit)

	// Public CRM & Inspection Pipelines
	crmRoutes := api.Group("/crm")
	crmRoutes.Post("/inquiries", crmHandler.SubmitInquiry)
	crmRoutes.Post("/inspections", crmHandler.BookInspection)

	// Protected Internal Operations & Admin Portal Endpoints
	admin := api.Group("/admin", middleware.Authenticate(cfg))
	{
		// CRM & Leads Management
		admin.Get("/leads", middleware.RequireRoles(
			identity.RoleSuperAdmin,
			identity.RoleOperationsDirector,
			identity.RoleSalesLead,
		), crmHandler.ListLeads)
	}
}

