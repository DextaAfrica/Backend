package middleware

import (
	"fmt"

	"github.com/DextaAfrica/Backend/internal/infrastructure/cache"
	"github.com/DextaAfrica/Backend/internal/interfaces/http/response"

	"github.com/gofiber/fiber/v2"
)

// RateLimiter enforces sliding-window rate limits backed by Redis.
func RateLimiter(redisClient *cache.RedisClient, maxReqs int, windowSec int) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if redisClient == nil || maxReqs <= 0 {
			return c.Next()
		}

		// Identify by client IP or authenticated user ID
		clientKey := c.IP()
		if userID, ok := c.Locals("user_id").(string); ok && userID != "" {
			clientKey = "user:" + userID
		}

		key := fmt.Sprintf("rl:%s:%s", c.Path(), clientKey)
		ctx := c.Context()

		allowed, remaining, resetSec, err := redisClient.CheckRateLimit(ctx, key, maxReqs, windowSec)
		if err != nil {
			// Fail open on Redis connectivity issue so we never drop traffic
			return c.Next()
		}

		c.Set("X-RateLimit-Limit", fmt.Sprintf("%d", maxReqs))
		c.Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetSec))

		if !allowed {
			return response.Error(
				c,
				fiber.StatusTooManyRequests,
				"Rate limit exceeded. Please throttle your requests.",
				"RATE_LIMIT_EXCEEDED",
				[]response.ErrorDetail{
					{
						Field: "ip",
						Issue: fmt.Sprintf("Exceeded %d requests per %d seconds", maxReqs, windowSec),
					},
				},
			)
		}

		return c.Next()
	}
}

