package middleware

import (
	"time"

	"github.com/DextaAfrica/Backend/internal/infrastructure/logger"

	"github.com/gofiber/fiber/v2"
)

// HTTPLogger logs structured request and response metrics.
func HTTPLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		err := c.Next()

		latency := time.Since(start)
		statusCode := c.Response().StatusCode()
		traceID, _ := c.Locals("trace_id").(string)

		event := logger.Log.Info()
		if statusCode >= 500 {
			event = logger.Log.Error()
		} else if statusCode >= 400 {
			event = logger.Log.Warn()
		}

		event.
			Str("trace_id", traceID).
			Str("method", c.Method()).
			Str("path", c.Path()).
			Int("status", statusCode).
			Dur("latency", latency).
			Str("ip", c.IP()).
			Str("user_agent", c.Get("User-Agent")).
			Msg("HTTP Request")

		return err
	}
}

