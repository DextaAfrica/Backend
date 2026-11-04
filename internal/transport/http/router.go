// Package http assembles the chi router: every route in the API is
// registered here, and only here, so the full surface of the API — what's
// public, what's admin-only, what's rate-limited — is visible in one file.
package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DextaAfrica/Backend/internal/apperror"
	"github.com/DextaAfrica/Backend/internal/config"
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
}

func NewRouter(cfg *config.Config, pool *pgxpool.Pool, h Handlers) http.Handler {
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
	})

	// Public write surface: unauthenticated but rate-limited per IP since
	// it accepts arbitrary visitor input.
	r.Group(func(r chi.Router) {
		r.Use(publicLimiter.Middleware)
		r.Post("/enquiries", h.Enquiries.Submit)
		r.Post("/newsletter/subscribe", h.Newsletter.Subscribe)
		r.Post("/newsletter/unsubscribe", h.Newsletter.Unsubscribe)
	})

	// Admin surface: every route here requires a valid admin session JWT.
	r.Route("/admin", func(r chi.Router) {
		r.Use(requireAdmin)

		r.Route("/content", func(r chi.Router) {
			r.Get("/", h.Content.ListAdmin)
			r.Get("/{key}", h.Content.GetAdmin)
			r.Put("/{key}", h.Content.Save)
		})

		r.Route("/portfolio", func(r chi.Router) {
			r.Get("/", h.Portfolio.ListAdmin)
			r.Post("/", h.Portfolio.Create)
			r.Get("/{id}", h.Portfolio.GetAdmin)
			r.Put("/{id}", h.Portfolio.Update)
			r.Delete("/{id}", h.Portfolio.Delete)
		})

		r.Route("/journal", func(r chi.Router) {
			r.Get("/", h.Journal.ListAdmin)
			r.Post("/", h.Journal.Create)
			r.Get("/{id}", h.Journal.GetAdmin)
			r.Put("/{id}", h.Journal.Update)
			r.Delete("/{id}", h.Journal.Delete)
		})

		r.Route("/careers/listings", func(r chi.Router) {
			r.Get("/", h.Careers.ListAdmin)
			r.Post("/", h.Careers.Create)
			r.Get("/{id}", h.Careers.GetAdmin)
			r.Put("/{id}", h.Careers.Update)
			r.Delete("/{id}", h.Careers.Delete)
		})

		r.Route("/enquiries", func(r chi.Router) {
			r.Get("/", h.Enquiries.ListAdmin)
			r.Get("/{id}", h.Enquiries.GetAdmin)
			r.Patch("/{id}/status", h.Enquiries.UpdateStatus)
		})

		r.Route("/newsletter", func(r chi.Router) {
			r.Get("/subscribers", h.Newsletter.ListAdmin)
		})
	})

	return r
}
