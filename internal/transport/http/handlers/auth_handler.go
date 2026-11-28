package handlers

import (
	"net/http"

	"github.com/DextaAfrica/Backend/internal/authtoken"
	"github.com/DextaAfrica/Backend/internal/service"
	"github.com/DextaAfrica/Backend/internal/transport/http/request"
	"github.com/DextaAfrica/Backend/internal/transport/http/response"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

func getClientIP(r *http.Request) string {
	if ip := r.Header.Get("CF-Connecting-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req request.LoginRequest
	if err := request.Decode(w, r, &req); err != nil {
		response.Error(w, r, err)
		return
	}

	ip := getClientIP(r)
	userAgent := r.UserAgent()

	result, err := h.auth.Login(r.Context(), req.Email, req.Password, ip, userAgent)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.OK(w, result)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req request.RefreshTokenRequest
	if err := request.Decode(w, r, &req); err != nil {
		response.Error(w, r, err)
		return
	}

	ip := getClientIP(r)
	userAgent := r.UserAgent()

	result, err := h.auth.RefreshToken(r.Context(), req.RefreshToken, ip, userAgent)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.OK(w, result)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	claims := authtoken.ClaimsFromContext(r.Context())
	if claims != nil && claims.SessionID != "" {
		_ = h.auth.Logout(r.Context(), claims.SessionID)
	}

	response.OK(w, map[string]any{"loggedOut": true})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims := authtoken.ClaimsFromContext(r.Context())
	if claims == nil {
		response.Error(w, r, authtoken.ErrInvalidToken)
		return
	}

	response.OK(w, map[string]any{
		"adminId":  claims.AdminID,
		"email":    claims.Email,
		"role":     claims.Role,
		"modules":  claims.AssignedModules,
		"sessionId": claims.SessionID,
	})
}
