package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/DextaAfrica/Backend/internal/apperror"
	"github.com/DextaAfrica/Backend/internal/transport/http/response"
)

// Recover is the outermost error boundary: any panic in a handler or a
// deeper middleware is caught here, logged with a full stack trace, and
// turned into a well-formed 500 JSON response instead of crashing the
// process or leaking a bare stack trace to the client.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic recovered",
					"request_id", RequestIDFromContext(r.Context()),
					"path", r.URL.Path,
					"panic", rec,
					"stack", string(debug.Stack()),
				)
				response.Error(w, r, apperror.New(apperror.CodeInternal, "an unexpected error occurred"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
