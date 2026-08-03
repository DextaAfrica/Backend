// Package config loads and validates all runtime configuration from the
// environment. No package in this codebase should read os.Getenv directly or
// embed a literal default for a secret, URL, or credential — everything
// flows through the Config struct built here, so every tunable is visible in
// one place and every required value fails fast at boot instead of panicking
// deep in a request handler.
package config

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/caarlos0/env/v10"
	"github.com/joho/godotenv"
)

// Config is the fully-resolved application configuration. Fields are grouped
// by the subsystem that owns them.
type Config struct {
	App      App
	Database Database
	Auth     Auth
	CORS     CORS
	RateLimit RateLimit
	Frontend Frontend
	Newsletter Newsletter
}

type App struct {
	Env             string        `env:"APP_ENV" envDefault:"development"`
	Port            int           `env:"APP_PORT" envDefault:"8080"`
	LogLevel        string        `env:"LOG_LEVEL" envDefault:"info"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"15s"`
	RequestTimeout  time.Duration `env:"REQUEST_TIMEOUT" envDefault:"30s"`
}

func (a App) IsProduction() bool { return a.Env == "production" }

type Database struct {
	URL             string        `env:"DATABASE_URL,required"`
	MaxConns        int32         `env:"DATABASE_MAX_CONNS" envDefault:"10"`
	MinConns        int32         `env:"DATABASE_MIN_CONNS" envDefault:"2"`
	ConnMaxLifetime time.Duration `env:"DATABASE_CONN_MAX_LIFETIME" envDefault:"1h"`
	MigrationsPath  string        `env:"DATABASE_MIGRATIONS_PATH" envDefault:"file://internal/db/migrations"`
	AutoMigrate     bool          `env:"DATABASE_AUTO_MIGRATE" envDefault:"true"`
}

type Auth struct {
	JWTSecret        string        `env:"JWT_SECRET,required"`
	AccessTokenTTL   time.Duration `env:"JWT_ACCESS_TOKEN_TTL" envDefault:"24h"`
	AdminBootstrapEmail    string `env:"ADMIN_BOOTSTRAP_EMAIL"`
	AdminBootstrapPassword string `env:"ADMIN_BOOTSTRAP_PASSWORD"`
}

// CORS origins are always explicit and environment-supplied — never a
// hardcoded frontend domain — because the same binary serves local dev,
// staging, and production frontends from different origins.
type CORS struct {
	AllowedOrigins []string `env:"CORS_ALLOWED_ORIGINS,required" envSeparator:","`
}

type RateLimit struct {
	PublicRPS   float64 `env:"RATE_LIMIT_PUBLIC_RPS" envDefault:"2"`
	PublicBurst int     `env:"RATE_LIMIT_PUBLIC_BURST" envDefault:"10"`
}

// Frontend holds the settings needed to call back into the Next.js frontend
// when content changes, so editors see updates without waiting for the
// frontend's own cache TTL to expire.
type Frontend struct {
	RevalidateURL    string        `env:"FRONTEND_REVALIDATE_URL"`
	RevalidateSecret string        `env:"FRONTEND_REVALIDATE_SECRET"`
	RevalidateTimeout time.Duration `env:"FRONTEND_REVALIDATE_TIMEOUT" envDefault:"5s"`
}

type Newsletter struct {
	WebhookURL   string        `env:"NEWSLETTER_WEBHOOK_URL"`
	WebhookToken string        `env:"NEWSLETTER_WEBHOOK_TOKEN"`
	WebhookTimeout time.Duration `env:"NEWSLETTER_WEBHOOK_TIMEOUT" envDefault:"5s"`
}

// Load reads a .env file when present (local development convenience only —
// production deployments are expected to inject real environment variables)
// then parses and validates the environment into a Config.
func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		slog.Debug("no .env file loaded", "detail", err.Error())
	}

	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("config: parse environment: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if len(c.Auth.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}
	if c.App.Port <= 0 || c.App.Port > 65535 {
		return fmt.Errorf("APP_PORT must be between 1 and 65535")
	}
	for _, origin := range c.CORS.AllowedOrigins {
		if origin == "" {
			return fmt.Errorf("CORS_ALLOWED_ORIGINS contains an empty origin")
		}
	}
	if (c.Auth.AdminBootstrapEmail == "") != (c.Auth.AdminBootstrapPassword == "") {
		return fmt.Errorf("ADMIN_BOOTSTRAP_EMAIL and ADMIN_BOOTSTRAP_PASSWORD must be set together")
	}
	return nil
}
