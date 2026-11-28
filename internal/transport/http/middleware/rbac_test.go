package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DextaAfrica/Backend/internal/authtoken"
)

func TestRequireRoleAndModule(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	roleMiddleware := RequireRole("SUPER_ADMIN")
	moduleMiddleware := RequireModule("approvals")

	// 1. Unauthorized actor (no claims)
	req := httptest.NewRequest("GET", "/admin/test", nil)
	rr := httptest.NewRecorder()
	roleMiddleware(handler).ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", rr.Code)
	}

	// 2. Insufficient Role
	regularClaims := &authtoken.Claims{
		AdminID:         "admin-1",
		Role:            "EDITOR",
		AssignedModules: []string{"operations"},
	}
	ctxWithRegular := authtoken.ContextWithClaims(req.Context(), regularClaims)
	rr = httptest.NewRecorder()
	roleMiddleware(handler).ServeHTTP(rr, req.WithContext(ctxWithRegular))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for EDITOR on SUPER_ADMIN gate, got %d", rr.Code)
	}

	// 3. Module Denial
	rr = httptest.NewRecorder()
	moduleMiddleware(handler).ServeHTTP(rr, req.WithContext(ctxWithRegular))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for missing 'approvals' module, got %d", rr.Code)
	}

	// 4. Authorized Super Admin
	superClaims := &authtoken.Claims{
		AdminID:         "super-1",
		Role:            "SUPER_ADMIN",
		AssignedModules: []string{"operations", "approvals"},
	}
	ctxWithSuper := authtoken.ContextWithClaims(req.Context(), superClaims)

	rr = httptest.NewRecorder()
	roleMiddleware(handler).ServeHTTP(rr, req.WithContext(ctxWithSuper))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for SUPER_ADMIN, got %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	moduleMiddleware(handler).ServeHTTP(rr, req.WithContext(ctxWithSuper))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for SUPER_ADMIN on module gate, got %d", rr.Code)
	}
}
