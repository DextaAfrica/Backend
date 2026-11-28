package domain

import (
	"context"
	"time"
)

type IdempotencyRecord struct {
	Key             string
	RequestPath     string
	RequestMethod   string
	RequestBodyHash string
	StatusCode      int
	ResponseBody    []byte
	LockedUntil     time.Time
	CreatedAt       time.Time
}

type IdempotencyRepository interface {
	LockOrGet(ctx context.Context, key, path, method, bodyHash string, ttl time.Duration) (*IdempotencyRecord, bool, error)
	SaveResponse(ctx context.Context, key string, statusCode int, body []byte) error
}
