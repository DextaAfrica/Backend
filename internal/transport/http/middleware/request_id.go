package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type contextKey string

const requestIDKey contextKey = "request_id"

const RequestIDHeader = "X-Request-ID"

// RequestID assigns a unique ID to every request — reusing one supplied by
// an upstream proxy/load balancer when present — so a single failure can be
// traced end to end across logs, error responses, and downstream calls.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(RequestIDHeader)
		if id == "" {
			id = uuid.NewString()
		}
		w.Header().Set(RequestIDHeader, id)
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}
