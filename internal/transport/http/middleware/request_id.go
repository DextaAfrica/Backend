package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/DextaAfrica/Backend/internal/requestid"
)

// RequestID assigns a unique ID to every request — reusing one supplied by
// an upstream proxy/load balancer when present — so a single failure can be
// traced end to end across logs, error responses, and downstream calls.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(requestid.Header)
		if id == "" {
			id = uuid.NewString()
		}
		w.Header().Set(requestid.Header, id)
		next.ServeHTTP(w, r.WithContext(requestid.Set(r.Context(), id)))
	})
}

// RequestIDFromContext is kept here as a thin alias so existing call sites
// in this package's sibling files don't need to import requestid directly.
func RequestIDFromContext(ctx context.Context) string {
	return requestid.FromContext(ctx)
}
