package handlers

import (
	"net/http"

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

type loginResponse struct {
	AccessToken string `json:"accessToken"`
	TokenType   string `json:"tokenType"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req request.LoginRequest
	if err := request.Decode(w, r, &req); err != nil {
		response.Error(w, r, err)
		return
	}

	token, err := h.auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.OK(w, loginResponse{AccessToken: token, TokenType: "Bearer"})
}
