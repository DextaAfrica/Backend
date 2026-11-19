package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/DextaAfrica/Backend/internal/domain"
	"github.com/DextaAfrica/Backend/internal/service"
	"github.com/DextaAfrica/Backend/internal/transport/http/middleware"
	"github.com/DextaAfrica/Backend/internal/transport/http/request"
	"github.com/DextaAfrica/Backend/internal/transport/http/response"
)

type ProductionApprovalHandler struct {
	service *service.ProductionApprovalService
}

func NewProductionApprovalHandler(s *service.ProductionApprovalService) *ProductionApprovalHandler {
	return &ProductionApprovalHandler{service: s}
}

func (h *ProductionApprovalHandler) List(w http.ResponseWriter, r *http.Request) {
	page := queryInt(r, "page", 1)
	perPage := queryInt(r, "per_page", 20)

	result, err := h.service.List(r.Context(), domain.ListParams{
		Page:    page,
		PerPage: perPage,
	})
	if err != nil {
		response.Error(w, r, err)
		return
	}

	stats, _ := h.service.GetStats(r.Context())

	response.JSON(w, http.StatusOK, map[string]any{
		"items":       result.Items,
		"stats":       stats,
		"page":        result.Page,
		"per_page":    result.PerPage,
		"total_items": result.TotalItems,
		"total_pages": result.TotalPages(),
	})
}

func (h *ProductionApprovalHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req request.CreateProductionReleaseRequest
	if err := request.Decode(w, r, &req); err != nil {
		response.Error(w, r, err)
		return
	}

	adminID := middleware.AdminIDFromContext(r.Context())
	adminName := r.Header.Get("X-Actor-Name")
	if adminName == "" {
		adminName = "Enterprise Admin"
	}
	adminEmail := r.Header.Get("X-Actor-Email")
	if adminEmail == "" {
		adminEmail = "admin@dextaafrica.com"
	}

	res, err := h.service.Create(r.Context(), service.CreateProductionReleaseInput{
		TicketNumber:  req.TicketNumber,
		Title:         req.Title,
		TargetSystem:  req.TargetSystem,
		AdminID:       adminID,
		AdminName:     adminName,
		AdminEmail:    adminEmail,
		RiskLevel:     req.RiskLevel,
		Justification: req.Justification,
		RollBackPlan:  req.RollBackPlan,
		DiffSummary:   req.DiffSummary,
	})
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.Created(w, res)
}

func (h *ProductionApprovalHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	res, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, res)
}

func (h *ProductionApprovalHandler) Decide(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req request.DecideProductionReleaseRequest
	if err := request.Decode(w, r, &req); err != nil {
		response.Error(w, r, err)
		return
	}

	superAdminID := middleware.AdminIDFromContext(r.Context())
	superAdminName := r.Header.Get("X-Actor-Name")
	if superAdminName == "" {
		superAdminName = "Super Admin Authority"
	}
	superAdminEmail := r.Header.Get("X-Actor-Email")
	if superAdminEmail == "" {
		superAdminEmail = "superadmin@dextaafrica.com"
	}

	res, err := h.service.Decide(r.Context(), id, req.Action, superAdminID, superAdminName, superAdminEmail, req.Notes)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, res)
}
