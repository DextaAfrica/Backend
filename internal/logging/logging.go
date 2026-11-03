// Package logging configures the process-wide slog default logger. Called
// once at startup; every other package just uses log/slog directly.
package logging

import (
	"log/slog"
	"os"

	"github.com/DextaAfrica/Backend/internal/config"
)

func Setup(cfg config.App) {
	level := parseLevel(cfg.LogLevel)

	var handler slog.Handler
	opts := &slog.HandlerOptions{Level: level}

	if cfg.IsProduction() {
		// Structured JSON in production so logs are directly ingestible by
		// a log aggregator (CloudWatch, Loki, Datadog, etc.).
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	slog.SetDefault(slog.New(handler))
}

func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
