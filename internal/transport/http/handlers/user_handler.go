package handlers

import (
	"net/http"

	"github.com/DextaAfrica/Backend/internal/domain"
	"github.com/DextaAfrica/Backend/internal/service"
	"github.com/DextaAfrica/Backend/internal/transport/http/request"
	"github.com/DextaAfrica/Backend/internal/transport/http/response"
)

type UserHandler struct {
	users *service.UserService
}

func NewUserHandler(users *service.UserService) *UserHandler {
	return &UserHandler{users: users}
}

type adminUserResponse struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	FullName   string `json:"fullName"`
	Role       string `json:"role"`
	Department string `json:"department"`
	JobTitle   string `json:"jobTitle"`
	Phone      string `json:"phone"`
	Status     string `json:"status"`
	CreatedAt  string `json:"createdAt"`
}

func toAdminUserResponse(a *domain.Admin) adminUserResponse {
	return adminUserResponse{
		ID:         a.ID,
		Email:      a.Email,
		FullName:   a.FullName,
		Role:       a.Role,
		Department: a.Department,
		JobTitle:   a.JobTitle,
		Phone:      a.Phone,
		Status:     a.Status,
		CreatedAt:  a.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func (h *UserHandler) Sync(w http.ResponseWriter, r *http.Request) {
	var req request.SyncUserRequest
	if err := request.Decode(w, r, &req); err != nil {
		response.Error(w, r, err)
		return
	}

	admin := &domain.Admin{
		Email:        req.Email,
		FullName:     req.FullName,
		Role:         req.Role,
		Department:   req.Department,
		JobTitle:     req.JobTitle,
		Phone:        req.Phone,
		Status:       req.Status,
		PasswordHash: "", // Delegated to Next.js enterprise auth / SSO
	}

	if err := h.users.SyncUser(r.Context(), admin); err != nil {
		response.Error(w, r, err)
		return
	}

	response.OK(w, map[string]any{
		"success": true,
		"synced":  true,
		"data":    toAdminUserResponse(admin),
	})
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	page := queryInt(r, "page", 1)
	perPage := queryInt(r, "perPage", 20)

	result, err := h.users.List(r.Context(), page, perPage)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	out := make([]adminUserResponse, len(result.Items))
	for i := range result.Items {
		out[i] = toAdminUserResponse(&result.Items[i])
	}

	response.Paginated(w, out, response.Meta{
		Page:       result.Page,
		PerPage:    result.PerPage,
		TotalItems: result.TotalItems,
		TotalPages: result.TotalPages(),
	})
}
