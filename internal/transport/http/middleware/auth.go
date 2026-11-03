package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/DextaAfrica/Backend/internal/apperror"
	"github.com/DextaAfrica/Backend/internal/authtoken"
	"github.com/DextaAfrica/Backend/internal/transport/http/response"
)

type adminContextKey string

const adminIDKey adminContextKey = "admin_id"

// RequireAdmin protects CMS write endpoints with a bearer JWT issued by the
// auth service at login. It is the only gate between the public internet
// and content mutation — every admin route in routes.go must be wrapped
// with this.
func RequireAdmin(jwtSecret string) func(http.Handler) http.Handler {
	secret := []byte(jwtSecret)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			token, ok := strings.CutPrefix(header, "Bearer ")
			if !ok || token == "" {
				response.Error(w, r, apperror.Unauthorized("missing bearer token"))
				return
			}

			claims, err := authtoken.Parse(secret, token)
			if err != nil {
				response.Error(w, r, apperror.Unauthorized("invalid or expired session"))
				return
			}

			ctx := context.WithValue(r.Context(), adminIDKey, claims.AdminID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func AdminIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(adminIDKey).(string)
	return id
}

// RequireSecret gates service-to-service endpoints (e.g. an inbound webhook)
// with a static bearer secret rather than a user session — appropriate when
// the caller is a machine, not a logged-in admin.
func RequireSecret(expected string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			token, ok := strings.CutPrefix(header, "Bearer ")
			if !ok || token == "" || expected == "" || token != expected {
				response.Error(w, r, apperror.Unauthorized("invalid credentials"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
