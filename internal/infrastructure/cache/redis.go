package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/DextaAfrica/Backend/internal/config"
	"github.com/DextaAfrica/Backend/internal/infrastructure/logger"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	Client *redis.Client
}

// Connect initializes the Redis client.
func Connect(cfg *config.RedisConfig) (*RedisClient, error) {
	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)

	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  3 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping Redis at %s: %w", addr, err)
	}

	logger.Log.Info().Str("addr", addr).Msg("Redis connected successfully")
	return &RedisClient{Client: rdb}, nil
}

// Ping checks if Redis is responsive.
func (r *RedisClient) Ping(ctx context.Context) error {
	if r == nil || r.Client == nil {
		return fmt.Errorf("redis client is not initialized")
	}
	return r.Client.Ping(ctx).Err()
}

// CheckRateLimit implements a sliding window counter using Redis sorted sets.
// Returns (allowed, remaining, resetSeconds, error).
func (r *RedisClient) CheckRateLimit(ctx context.Context, key string, limit int, windowSec int) (bool, int, int64, error) {
	if r == nil || r.Client == nil {
		// If Redis is not initialized, allow request
		return true, limit, 0, nil
	}

	now := time.Now().UnixNano()
	windowNano := int64(windowSec) * int64(time.Second)
	clearBefore := now - windowNano

	pipe := r.Client.TxPipeline()
	// Remove events older than current window
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", clearBefore))
	// Add current request timestamp
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: fmt.Sprintf("%d", now)})
	// Count elements in window
	countCmd := pipe.ZCard(ctx, key)
	// Refresh TTL on the key
	pipe.Expire(ctx, key, time.Duration(windowSec)*time.Second)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return true, limit, 0, err
	}

	count := int(countCmd.Val())
	remaining := limit - count
	if remaining < 0 {
		remaining = 0
	}

	allowed := count <= limit
	resetSeconds := int64(windowSec)

	return allowed, remaining, resetSeconds, nil
}

