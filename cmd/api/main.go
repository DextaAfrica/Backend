// Command api is the entrypoint for the Dexta Africa backend. It wires
// configuration, the database, every repository/service/handler, and starts
// an HTTP server with graceful shutdown. No business logic lives here — this
// file only constructs and connects things defined elsewhere.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DextaAfrica/Backend/internal/config"
	"github.com/DextaAfrica/Backend/internal/db"
	"github.com/DextaAfrica/Backend/internal/logging"
	"github.com/DextaAfrica/Backend/internal/repository/postgres"
	"github.com/DextaAfrica/Backend/internal/service"
	"github.com/DextaAfrica/Backend/internal/storage"
	transporthttp "github.com/DextaAfrica/Backend/internal/transport/http"
	"github.com/DextaAfrica/Backend/internal/transport/http/handlers"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal startup error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logging.Setup(cfg.App)
	slog.Info("starting dexta backend", "env", cfg.App.Env, "port", cfg.App.Port)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Dedicated migration execution mode for DevOps container orchestration
	// (e.g. `make migrate` or ephemeral migration jobs in docker-compose-prod.yml).
	isMigrateOnly := os.Getenv("MIGRATE_ONLY") == "true"
	for _, arg := range os.Args[1:] {
		if arg == "migrate" || arg == "-migrate-only" || arg == "--migrate-only" {
			isMigrateOnly = true
			break
		}
	}
	if isMigrateOnly {
		slog.Info("running database migrations in standalone mode", "migrations_path", cfg.Database.MigrationsPath)
		if err := db.Migrate(cfg.Database.URL, cfg.Database.MigrationsPath); err != nil {
			return fmt.Errorf("standalone migrations failed: %w", err)
		}
		slog.Info("database migrations applied successfully")
		return nil
	}

	if cfg.Database.AutoMigrate {
		if err := db.Migrate(cfg.Database.URL, cfg.Database.MigrationsPath); err != nil {
			return fmt.Errorf("run migrations: %w", err)
		}
		slog.Info("database migrations applied")
	}

	pool, err := db.Connect(ctx, cfg.Database)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer pool.Close()

	repos := buildRepositories(pool)
	services := buildServices(cfg, repos)

	if err := services.Auth.BootstrapAdmin(ctx, cfg.Auth.AdminBootstrapEmail, cfg.Auth.AdminBootstrapPassword); err != nil {
		return fmt.Errorf("bootstrap admin: %w", err)
	}

	h := transporthttp.Handlers{
		Health:     handlers.NewHealthHandler(pool),
		Auth:       handlers.NewAuthHandler(services.Auth),
		Content:    handlers.NewContentHandler(services.Content),
		Portfolio:  handlers.NewPortfolioHandler(services.Portfolio),
		Journal:    handlers.NewJournalHandler(services.Journal),
		Careers:    handlers.NewCareerHandler(services.Career),
		Enquiries:  handlers.NewEnquiryHandler(services.Enquiry),
		Newsletter: handlers.NewNewsletterHandler(services.Newsletter),
		Forms:      handlers.NewFormHandler(services.Forms),
		Users:      handlers.NewUserHandler(services.User),
		Audit:              handlers.NewAuditHandler(services.Audit),
		Storage:            handlers.NewStorageHandler(services.Storage),
		ProductionApproval: handlers.NewProductionApprovalHandler(services.ProductionApproval),
		DashboardAccess:    handlers.NewDashboardAccessHandler(services.DashboardAccess),
	}

	router := transporthttp.NewRouter(cfg, pool, h)

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.App.Port),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       cfg.App.RequestTimeout,
		WriteTimeout:      cfg.App.RequestTimeout + 5*time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		slog.Info("http server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()

	select {
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	case err := <-serveErr:
		return fmt.Errorf("http server: %w", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.App.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	slog.Info("server shut down cleanly")
	return nil
}

// repositories groups every repository implementation constructed against
// the shared connection pool.
type repositories struct {
	Admins      *postgres.AdminRepository
	Pages       *postgres.PageRepository
	Portfolio   *postgres.DevelopmentRepository
	Journal     *postgres.ArticleRepository
	Careers     *postgres.CareerListingRepository
	Enquiries   *postgres.EnquiryRepository
	Subscribers *postgres.SubscriberRepository
	Forms       *postgres.FormRepository
	Audit               *postgres.AuditRepository
	ProductionApprovals *postgres.ProductionApprovalRepository
	DashboardAccess     *postgres.DashboardAccessRepository
}

func buildRepositories(pool *pgxpool.Pool) repositories {
	return repositories{
		Admins:              postgres.NewAdminRepository(pool),
		Pages:               postgres.NewPageRepository(pool),
		Portfolio:           postgres.NewDevelopmentRepository(pool),
		Journal:             postgres.NewArticleRepository(pool),
		Careers:             postgres.NewCareerListingRepository(pool),
		Enquiries:           postgres.NewEnquiryRepository(pool),
		Subscribers:         postgres.NewSubscriberRepository(pool),
		Forms:               postgres.NewFormRepository(pool),
		Audit:               postgres.NewAuditRepository(pool),
		ProductionApprovals: postgres.NewProductionApprovalRepository(pool),
		DashboardAccess:     postgres.NewDashboardAccessRepository(pool),
	}
}

// services groups every service constructed against the repositories above
// plus cross-cutting collaborators (revalidation, newsletter forwarding).
type services struct {
	Auth       *service.AuthService
	Content    *service.ContentService
	Portfolio  *service.PortfolioService
	Journal    *service.JournalService
	Career     *service.CareerService
	Enquiry    *service.EnquiryService
	Newsletter *service.NewsletterService
	Forms      *service.FormService
	User       *service.UserService
	Audit              *service.AuditService
	Storage            storage.Service
	ProductionApproval *service.ProductionApprovalService
	DashboardAccess    *service.DashboardAccessService
}

func buildServices(cfg *config.Config, repos repositories) services {
	revalidator := service.NewRevalidator(cfg.Frontend)
	forwarder := service.NewNewsletterForwarder(cfg.Newsletter)
	storageService := storage.NewS3Service(cfg.Storage)

	return services{
		Auth:               service.NewAuthService(repos.Admins, cfg.Auth.JWTSecret, cfg.Auth.AccessTokenTTL),
		Content:            service.NewContentService(repos.Pages, revalidator),
		Portfolio:          service.NewPortfolioService(repos.Portfolio, revalidator),
		Journal:            service.NewJournalService(repos.Journal, revalidator),
		Career:             service.NewCareerService(repos.Careers, revalidator),
		Enquiry:            service.NewEnquiryService(repos.Enquiries),
		Newsletter:         service.NewNewsletterService(repos.Subscribers, forwarder),
		Forms:              service.NewFormService(repos.Forms),
		User:               service.NewUserService(repos.Admins),
		Audit:              service.NewAuditService(repos.Audit),
		Storage:            storageService,
		ProductionApproval: service.NewProductionApprovalService(repos.ProductionApprovals),
		DashboardAccess:    service.NewDashboardAccessService(repos.DashboardAccess, repos.Audit, repos.Admins),
	}
}
