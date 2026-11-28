// Package http assembles the chi router: every route in the API is
// registered here, and only here, so the full surface of the API — what's
// public, what's admin-only, what's rate-limited — is visible in one file.
package http

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DextaAfrica/Backend/internal/apperror"
	"github.com/DextaAfrica/Backend/internal/config"
	"github.com/DextaAfrica/Backend/internal/domain"
	"github.com/DextaAfrica/Backend/internal/transport/http/handlers"
	"github.com/DextaAfrica/Backend/internal/transport/http/middleware"
	"github.com/DextaAfrica/Backend/internal/transport/http/response"
)

// Handlers bundles every handler the router needs. Built once in main.go
// after all services are constructed.
type Handlers struct {
	Health     *handlers.HealthHandler
	Auth       *handlers.AuthHandler
	Content    *handlers.ContentHandler
	Portfolio  *handlers.PortfolioHandler
	Journal    *handlers.JournalHandler
	Careers    *handlers.CareerHandler
	Enquiries  *handlers.EnquiryHandler
	Newsletter *handlers.NewsletterHandler
	Forms      *handlers.FormHandler
	Visitors   *handlers.VisitorHandler
	Users      *handlers.UserHandler
	Audit              *handlers.AuditHandler
	Storage            *handlers.StorageHandler
	ProductionApproval *handlers.ProductionApprovalHandler
	DashboardAccess    *handlers.DashboardAccessHandler
}

func NewRouter(cfg *config.Config, pool *pgxpool.Pool, h Handlers, idempotencyRepo domain.IdempotencyRepository) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Recover)
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.Logging)
	r.Use(chimiddleware.Timeout(cfg.App.RequestTimeout))
	r.Use(middleware.CORS(cfg.CORS.AllowedOrigins))

	publicLimiter := middleware.NewIPRateLimiter(cfg.RateLimit.PublicRPS, cfg.RateLimit.PublicBurst)
	// Stricter rate limiter for authentication to prevent brute force / credential stuffing attacks:
	// 1 request every 5 seconds per IP with a burst of 5.
	authLimiter := middleware.NewIPRateLimiter(0.2, 5)
	requireAdmin := middleware.RequireAdmin(cfg.Auth.JWTSecret)
	idempotency := middleware.RequireIdempotency(idempotencyRepo, 24*time.Hour)

	// Canonical JSON 404 and 405 error boundary handlers
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		response.Error(w, r, apperror.NotFound("endpoint"))
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		response.Error(w, r, apperror.New(apperror.CodeBadRequest, "method not allowed"))
	})

	// Root operational status endpoint
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		response.JSON(w, http.StatusOK, map[string]any{
			"status":  "operational",
			"service": "Dexta Africa Enterprise API",
			"version": "1.0.0",
			"docs": map[string]string{
				"health": "/healthz",
				"ready":  "/readyz",
			},
		})
	})

	// Browser favicon request handler (suppresses 404 noise and satisfies CSP)
	r.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/x-icon")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		response.NoContent(w)
	})

	r.Get("/healthz", h.Health.Live)
	r.Get("/readyz", h.Health.Ready)

	// Authentication endpoint protected by strict brute-force rate limiter
	r.Group(func(r chi.Router) {
		r.Use(authLimiter.Middleware)
		r.Post("/auth/login", h.Auth.Login)
		r.Post("/admin/auth/login", h.Auth.Login)
		r.Post("/admin/auth/refresh", h.Auth.Refresh)
	})

	// Public read surface consumed by the Next.js frontend.
	r.Route("/content", func(r chi.Router) {
		r.Get("/{key}", h.Content.GetPublic)
	})

	r.Route("/portfolio", func(r chi.Router) {
		r.Get("/", h.Portfolio.ListPublic)
		r.Get("/{slug}", h.Portfolio.GetPublic)
	})

	r.Route("/journal", func(r chi.Router) {
		r.Get("/", h.Journal.ListPublic)
		r.Get("/{slug}", h.Journal.GetPublic)
	})

	r.Route("/careers/listings", func(r chi.Router) {
		r.Get("/", h.Careers.ListPublic)
		r.Get("/{slug}", h.Careers.GetPublic)
		r.With(idempotency).Post("/{slug}/inquiries", h.Careers.SubmitInquiry)
	})

	r.Route("/forms/{slug}", func(r chi.Router) {
		r.Get("/", h.Forms.GetPublic)
	})

	r.Get("/visitors/pass/{code}", h.Visitors.GetByPassCode)

	// Public write surface: unauthenticated but rate-limited per IP since
	// it accepts arbitrary visitor input. Protected by enterprise Idempotency-Key validation.
	r.Group(func(r chi.Router) {
		r.Use(publicLimiter.Middleware)
		r.With(idempotency).Post("/enquiries", h.Enquiries.Submit)
		r.With(idempotency).Post("/newsletter/subscribe", h.Newsletter.Subscribe)
		r.Post("/newsletter/unsubscribe", h.Newsletter.Unsubscribe)
		r.With(idempotency).Post("/forms/{slug}/submit", h.Forms.Submit)
		r.With(idempotency).Post("/visitors/check-in", h.Visitors.CheckIn)
		r.Post("/visitors/check-out", h.Visitors.SelfCheckOut)
		r.Post("/uploads/resume/presign", h.Storage.PresignPublicResume)
		r.Post("/uploads/resume", h.Storage.DirectPublicResume)
	})

	// Admin surface: every route here requires a valid admin session JWT.
	r.Route("/admin", func(r chi.Router) {
		r.Use(requireAdmin)

		// Session management endpoints
		r.Post("/auth/logout", h.Auth.Logout)
		r.Get("/auth/me", h.Auth.Me)

		r.Route("/content", func(r chi.Router) {
			r.Use(middleware.RequireModule("content"))
			r.Get("/", h.Content.ListAdmin)
			r.Get("/{key}", h.Content.GetAdmin)
			r.Put("/{key}", h.Content.Save)
		})

		r.Route("/portfolio", func(r chi.Router) {
			r.Use(middleware.RequireModule("content"))
			r.Get("/", h.Portfolio.ListAdmin)
			r.With(idempotency).Post("/", h.Portfolio.Create)
			r.Get("/{id}", h.Portfolio.GetAdmin)
			r.Put("/{id}", h.Portfolio.Update)
			r.With(middleware.RequireRole(string(domain.RoleSuperAdmin))).Delete("/{id}", h.Portfolio.Delete)
		})

		r.Route("/journal", func(r chi.Router) {
			r.Use(middleware.RequireModule("content"))
			r.Get("/", h.Journal.ListAdmin)
			r.With(idempotency).Post("/", h.Journal.Create)
			r.Get("/{id}", h.Journal.GetAdmin)
			r.Put("/{id}", h.Journal.Update)
			r.With(middleware.RequireRole(string(domain.RoleSuperAdmin))).Delete("/{id}", h.Journal.Delete)
		})

		r.Route("/careers/listings", func(r chi.Router) {
			r.Use(middleware.RequireModule("operations"))
			r.Get("/", h.Careers.ListAdmin)
			r.With(idempotency).Post("/", h.Careers.Create)
			r.Get("/{id}", h.Careers.GetAdmin)
			r.Put("/{id}", h.Careers.Update)
			r.With(middleware.RequireRole(string(domain.RoleSuperAdmin))).Delete("/{id}", h.Careers.Delete)
			r.Get("/{id}/inquiries", h.Careers.ListInquiries)
		})

		r.Route("/users", func(r chi.Router) {
			r.Use(middleware.RequireRole(string(domain.RoleSuperAdmin)))
			r.Post("/sync", h.Users.Sync)
			r.Post("/", h.Users.Sync)
			r.Get("/", h.Users.List)
		})

		r.Route("/audit-logs", func(r chi.Router) {
			r.Use(middleware.RequireRole(string(domain.RoleSuperAdmin)))
			r.Post("/", h.Audit.Record)
			r.Get("/", h.Audit.List)
		})

		r.Route("/enquiries", func(r chi.Router) {
			r.Use(middleware.RequireModule("operations"))
			r.Get("/", h.Enquiries.ListAdmin)
			r.Get("/{id}", h.Enquiries.GetAdmin)
			r.Patch("/{id}/status", h.Enquiries.UpdateStatus)
		})

		r.Route("/newsletter", func(r chi.Router) {
			r.Use(middleware.RequireModule("operations"))
			r.Get("/subscribers", h.Newsletter.ListAdmin)
		})

		r.Route("/forms", func(r chi.Router) {
			r.Use(middleware.RequireModule("operations"))
			r.Get("/", h.Forms.ListAdmin)
			r.With(idempotency).Post("/", h.Forms.Create)
			r.Get("/{id}", h.Forms.GetAdmin)
			r.Put("/{id}", h.Forms.Update)
			r.With(middleware.RequireRole(string(domain.RoleSuperAdmin))).Delete("/{id}", h.Forms.Delete)
			r.Get("/{id}/submissions", h.Forms.ListSubmissions)
			r.Get("/{id}/demographics", h.Forms.GetDemographics)
			r.Get("/{id}/export/csv", h.Forms.ExportCSV)
			r.Post("/{id}/batch-email", h.Forms.BatchEmail)
		})

		r.Route("/visitors", func(r chi.Router) {
			r.Use(middleware.RequireModule("operations"))
			r.Get("/", h.Visitors.ListAdmin)
			r.With(idempotency).Post("/", h.Visitors.CheckIn)
			r.Get("/analytics", h.Visitors.GetAnalytics)
			r.Get("/export/csv", h.Visitors.ExportCSV)
			r.Patch("/{id}/check-out", h.Visitors.AdminCheckOut)
			r.Get("/report-schedule", h.Visitors.GetReportSchedule)
			r.Put("/report-schedule", h.Visitors.UpdateReportSchedule)
			r.With(idempotency).Post("/report-schedule/trigger", h.Visitors.TriggerReport)
			r.Get("/report-deliveries", h.Visitors.ListReportDeliveries)
		})

		r.Route("/uploads", func(r chi.Router) {
			r.Use(middleware.RequireModule("content"))
			r.Post("/presign", h.Storage.PresignAdminUpload)
			r.Post("/", h.Storage.DirectAdminUpload)
			r.Delete("/", h.Storage.DeleteAdminUpload)
			r.Get("/private", h.Storage.PresignDownloadPrivate)
		})

		r.Route("/production-releases", func(r chi.Router) {
			r.Use(middleware.RequireModule("approvals"))
			r.Get("/", h.ProductionApproval.List)
			r.With(idempotency).Post("/", h.ProductionApproval.Create)
			r.Get("/{id}", h.ProductionApproval.Get)
			r.With(middleware.RequireRole(string(domain.RoleSuperAdmin))).Patch("/{id}/decision", h.ProductionApproval.Decide)
		})

		r.Route("/access-requests", func(r chi.Router) {
			r.Use(middleware.RequireModule("approvals"))
			r.Get("/", h.DashboardAccess.List)
			r.With(idempotency).Post("/", h.DashboardAccess.Create)
			r.Get("/{id}", h.DashboardAccess.Get)
			r.With(middleware.RequireRole(string(domain.RoleSuperAdmin))).Patch("/{id}/review", h.DashboardAccess.Review)
		})
	})

	return r
}
