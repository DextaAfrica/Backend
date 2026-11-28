package middleware

import (
	"net/http"
	"slices"

	"github.com/DextaAfrica/Backend/internal/apperror"
	"github.com/DextaAfrica/Backend/internal/authtoken"
	"github.com/DextaAfrica/Backend/internal/transport/http/response"
)

// RequireRole ensures the authenticated actor holds one of the required roles.
func RequireRole(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := authtoken.ClaimsFromContext(r.Context())
			if claims == nil {
				response.Error(w, r, apperror.Unauthorized("authentication required"))
				return
			}
			if !slices.Contains(allowedRoles, claims.Role) {
				response.Error(w, r, apperror.Forbidden("insufficient role privileges"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireModule ensures the authenticated admin is authorized for the target operational module.
func RequireModule(moduleName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := authtoken.ClaimsFromContext(r.Context())
			if claims == nil {
				response.Error(w, r, apperror.Unauthorized("authentication required"))
				return
			}
			// SUPER_ADMIN has access to every module by definition
			if claims.Role == "SUPER_ADMIN" || slices.Contains(claims.AssignedModules, moduleName) {
				next.ServeHTTP(w, r)
				return
			}
			response.Error(w, r, apperror.Forbidden("access to module '"+moduleName+"' is restricted"))
		})
	}
}
