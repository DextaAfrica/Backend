package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration parameters for the Dexta Africa platform.
type Config struct {
	App         AppConfig
	Database    DatabaseConfig
	Redis       RedisConfig
	JWT         JWTConfig
	Mail        MailConfig
	Storage     StorageConfig
	RateLimiter RateLimiterConfig
}

type AppConfig struct {
	Name            string
	Environment     string
	Port            string
	BaseURL         string
	FrontendWebURL  string
	FrontendAdminURL string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
}

type DatabaseConfig struct {
	Host               string
	Port               string
	User               string
	Password           string
	Name               string
	SSLMode            string
	MaxOpenConnections int
	MaxIdleConnections int
	MaxLifetimeMinutes int
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type JWTConfig struct {
	Secret            string
	AccessTokenExpiry time.Duration
	RefreshTokenExpiry time.Duration
}

type MailConfig struct {
	Provider string // resend, sendgrid, smtp
	APIKey   string
	FromEmail string
	FromName  string
	Host     string
	Port     int
	Username string
	Password string
}

type StorageConfig struct {
	Provider        string // s3, r2, local
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	Region          string
	Endpoint        string // For Cloudflare R2 or MinIO
}

type RateLimiterConfig struct {
	Enabled   bool
	MaxReqs   int
	WindowSec int
}

// LoadConfig reads configuration from .env file and environment variables.
func LoadConfig() (*Config, error) {
	// Load .env if present (ignore error if file doesn't exist)
	_ = godotenv.Load()

	cfg := &Config{
		App: AppConfig{
			Name:             getEnv("APP_NAME", "DextaAfrica-API"),
			Environment:      getEnv("APP_ENV", "development"),
			Port:             getEnv("APP_PORT", "8080"),
			BaseURL:          getEnv("APP_BASE_URL", "http://localhost:8080"),
			FrontendWebURL:   getEnv("FRONTEND_WEB_URL", "http://localhost:3000"),
			FrontendAdminURL: getEnv("FRONTEND_ADMIN_URL", "http://localhost:3001"),
			ReadTimeout:      getEnvDuration("APP_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:     getEnvDuration("APP_WRITE_TIMEOUT", 15*time.Second),
		},
		Database: DatabaseConfig{
			Host:               getEnv("DB_HOST", "localhost"),
			Port:               getEnv("DB_PORT", "5432"),
			User:               getEnv("DB_USER", "postgres"),
			Password:           getEnv("DB_PASSWORD", "postgres"),
			Name:               getEnv("DB_NAME", "dexta_db"),
			SSLMode:            getEnv("DB_SSLMODE", "disable"),
			MaxOpenConnections: getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConnections: getEnvInt("DB_MAX_IDLE_CONNS", 10),
			MaxLifetimeMinutes: getEnvInt("DB_MAX_LIFETIME_MINS", 30),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret:             getEnv("JWT_SECRET", "super-secret-dexta-africa-enterprise-key-2026"),
			AccessTokenExpiry:  getEnvDuration("JWT_ACCESS_EXPIRY", 15*time.Minute),
			RefreshTokenExpiry: getEnvDuration("JWT_REFRESH_EXPIRY", 7*24*time.Hour),
		},
		Mail: MailConfig{
			Provider:  getEnv("MAIL_PROVIDER", "smtp"),
			APIKey:    getEnv("MAIL_API_KEY", ""),
			FromEmail: getEnv("MAIL_FROM_EMAIL", "notifications@dextaafrica.com"),
			FromName:  getEnv("MAIL_FROM_NAME", "Dexta Africa"),
			Host:      getEnv("MAIL_HOST", "localhost"),
			Port:      getEnvInt("MAIL_PORT", 1025), // Mailpit default
			Username:  getEnv("MAIL_USERNAME", ""),
			Password:  getEnv("MAIL_PASSWORD", ""),
		},
		Storage: StorageConfig{
			Provider:        getEnv("STORAGE_PROVIDER", "local"),
			AccessKeyID:     getEnv("STORAGE_ACCESS_KEY", ""),
			SecretAccessKey: getEnv("STORAGE_SECRET_KEY", ""),
			BucketName:      getEnv("STORAGE_BUCKET", "dexta-assets"),
			Region:          getEnv("STORAGE_REGION", "auto"),
			Endpoint:        getEnv("STORAGE_ENDPOINT", ""),
		},
		RateLimiter: RateLimiterConfig{
			Enabled:   getEnvBool("RATE_LIMITER_ENABLED", true),
			MaxReqs:   getEnvInt("RATE_LIMITER_MAX_REQS", 120),
			WindowSec: getEnvInt("RATE_LIMITER_WINDOW_SEC", 60),
		},
	}

	return cfg, nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		if boolVal, err := strconv.ParseBool(val); err == nil {
			return boolVal
		}
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if dur, err := time.ParseDuration(val); err == nil {
			return dur
		}
	}
	return defaultVal
}

