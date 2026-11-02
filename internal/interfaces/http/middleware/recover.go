package middleware

import (
	"fmt"
	"runtime/debug"

	"github.com/DextaAfrica/Backend/internal/infrastructure/logger"
	"github.com/DextaAfrica/Backend/internal/interfaces/http/response"

	"github.com/gofiber/fiber/v2"
)

// Recover catches any panic and returns a standardized 500 error envelope.
func Recover() fiber.Handler {
	return func(c *fiber.Ctx) (err error) {
		defer func() {
			if r := recover(); r != nil {
				traceID, _ := c.Locals("trace_id").(string)
				stack := string(debug.Stack())

				logger.Log.Error().
					Str("trace_id", traceID).
					Interface("panic", r).
					Str("stack", stack).
					Msg("Unhandled panic recovered")

				err = response.Error(
					c,
					fiber.StatusInternalServerError,
					"An unexpected internal error occurred. Our engineering team has been notified.",
					"INTERNAL_SERVER_ERROR",
					[]response.ErrorDetail{
						{
							Field: "server",
							Issue: fmt.Sprintf("Panic: %v", r),
						},
					},
				)
			}
		}()
		return c.Next()
	}
}

