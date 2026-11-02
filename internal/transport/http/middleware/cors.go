package middleware

import (
	"github.com/go-chi/cors"
	"net/http"
)

// CORS builds the CORS middleware from configured allowed origins — never a
// hardcoded frontend domain — so the same binary works against local,
// staging, and production frontends by configuration alone.
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	return cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	})
}
