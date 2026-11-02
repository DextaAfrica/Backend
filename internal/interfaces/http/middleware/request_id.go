package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// RequestID attaches a unique trace ID to each request and in the response header.
func RequestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		traceID := c.Get("X-Trace-ID")
		if traceID == "" {
			traceID = c.Get("X-Request-ID")
		}
		if traceID == "" {
			traceID = "req_" + uuid.New().String()
		}

		c.Locals("trace_id", traceID)
		c.Set("X-Trace-ID", traceID)

		return c.Next()
	}
}

