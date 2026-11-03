// Command seeder initializes baseline CMS editorial pages and bootstrap data.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/DextaAfrica/Backend/internal/config"
	"github.com/DextaAfrica/Backend/internal/db"
	"github.com/DextaAfrica/Backend/internal/domain"
	"github.com/DextaAfrica/Backend/internal/logging"
	"github.com/DextaAfrica/Backend/internal/repository/postgres"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal seeder error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logging.Setup(cfg.App)
	slog.Info("starting dexta seeder", "env", cfg.App.Env)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if cfg.Database.AutoMigrate {
		if err := db.Migrate(cfg.Database.URL, cfg.Database.MigrationsPath); err != nil {
			return fmt.Errorf("run migrations: %w", err)
		}
		slog.Info("database migrations verified")
	}

	pool, err := db.Connect(ctx, cfg.Database)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer pool.Close()

	pageRepo := postgres.NewPageRepository(pool)
	seedBaselinePages(ctx, pageRepo)

	slog.Info("seeder completed successfully")
	return nil
}

func seedBaselinePages(ctx context.Context, repo *postgres.PageRepository) {
	baselinePages := []struct {
		Key         string
		Title       string
		SEOTitle    string
		SEODesc     string
		DefaultBody map[string]any
	}{
		{
			Key:      "home",
			Title:    "Home",
			SEOTitle: "Dexta Africa | Luxury Continental Real Estate",
			SEODesc:  "Pioneering ultra-luxury architectural developments across Africa.",
			DefaultBody: map[string]any{
				"tagline": "Architectural Excellence Across the Continent",
			},
		},
		{
			Key:      "about",
			Title:    "About Us",
			SEOTitle: "About Dexta Africa",
			SEODesc:  "Learn about our vision, leadership, and continental development footprint.",
			DefaultBody: map[string]any{
				"mission": "To set the benchmark for modern luxury real estate in Africa.",
			},
		},
		{
			Key:      "lifestyle",
			Title:    "Lifestyle",
			SEOTitle: "The Dexta Lifestyle",
			SEODesc:  "Curated living spaces engineered for discerning global clients.",
			DefaultBody: map[string]any{
				"highlights": []string{"Biophilic Architecture", "Smart Home Automation", "Private Marinas"},
			},
		},
		{
			Key:      "contact",
			Title:    "Contact",
			SEOTitle: "Contact Dexta Africa",
			SEODesc:  "Reach out to our advisory team for bespoke acquisitions.",
			DefaultBody: map[string]any{
				"email": "info@dexta.africa",
			},
		},
	}

	for _, p := range baselinePages {
		existing, err := repo.GetByKey(ctx, p.Key)
		if err == nil && existing != nil {
			slog.Debug("page already exists, skipping", "key", p.Key)
			continue
		}

		page := &domain.Page{
			Key:            p.Key,
			Title:          p.Title,
			SEOTitle:       p.SEOTitle,
			SEODescription: p.SEODesc,
			Published:      true,
			Content:        p.DefaultBody,
		}

		if _, err := repo.Upsert(ctx, page); err != nil {
			slog.Warn("could not upsert page", "key", p.Key, "error", err)
		} else {
			slog.Info("seeded baseline page", "key", p.Key)
		}
	}
}
