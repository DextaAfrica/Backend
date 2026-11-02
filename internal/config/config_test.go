package config_test

import (
	"os"
	"testing"
	"time"

	"github.com/DextaAfrica/Backend/internal/config"
)

func TestLoadConfigDefaults(t *testing.T) {
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig returned unexpected error: %v", err)
	}

	if cfg.App.Name != "DextaAfrica-API" {
		t.Errorf("Expected App.Name to be DextaAfrica-API, got %s", cfg.App.Name)
	}
	if cfg.App.Port != "8080" {
		t.Errorf("Expected default Port 8080, got %s", cfg.App.Port)
	}
	if cfg.Database.Name != "dexta_africa" && cfg.Database.Name != "dexta_db" {
		t.Errorf("Expected database name to be configured, got %s", cfg.Database.Name)
	}
}

func TestLoadConfigWithEnvOverride(t *testing.T) {
	os.Setenv("APP_PORT", "9090")
	os.Setenv("RATE_LIMITER_MAX_REQS", "250")
	defer func() {
		os.Unsetenv("APP_PORT")
		os.Unsetenv("RATE_LIMITER_MAX_REQS")
	}()

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if cfg.App.Port != "9090" {
		t.Errorf("Expected overridden port 9090, got %s", cfg.App.Port)
	}
	if cfg.RateLimiter.MaxReqs != 250 {
		t.Errorf("Expected overridden max reqs 250, got %d", cfg.RateLimiter.MaxReqs)
	}
	if cfg.App.ReadTimeout != 15*time.Second {
		t.Errorf("Expected default read timeout 15s, got %v", cfg.App.ReadTimeout)
	}
}

