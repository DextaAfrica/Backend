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

type DashboardAccessHandler struct {
	service *service.DashboardAccessService
}

func NewDashboardAccessHandler(svc *service.DashboardAccessService) *DashboardAccessHandler {
	return &DashboardAccessHandler{service: svc}
}

func (h *DashboardAccessHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body request.CreateDashboardAccessRequest
	if err := request.Decode(w, r, &body); err != nil {
		response.Error(w, r, err)
		return
	}

	userID := body.UserID
	if userID == "" {
		userID = middleware.AdminIDFromContext(r.Context())
	}
	email := body.RequesterEmail
	if email == "" {
		email = r.Header.Get("X-Actor-Email")
	}

	req := &domain.DashboardAccessRequest{
		UserID:           userID,
		RequesterName:    body.RequesterName,
		RequesterEmail:   email,
		Department:       body.Department,
		JobTitle:         body.JobTitle,
		RequestedModules: body.RequestedModules,
		Justification:    body.Justification,
		SignatureData:    body.SignatureData,
		SignatureType:    domain.DigitalSignatureType(body.SignatureType),
	}

	if err := h.service.Create(r.Context(), req); err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusCreated, req)
}

func (h *DashboardAccessHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	item, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, item)
}

func (h *DashboardAccessHandler) List(w http.ResponseWriter, r *http.Request) {
	var statusFilter *domain.DashboardRequestStatus
	if s := r.URL.Query().Get("status"); s != "" {
		st := domain.DashboardRequestStatus(s)
		statusFilter = &st
	}

	items, err := h.service.List(r.Context(), statusFilter)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"data":  items,
		"total": len(items),
	})
}

func (h *DashboardAccessHandler) Review(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body request.ReviewDashboardAccessRequest
	if err := request.Decode(w, r, &body); err != nil {
		response.Error(w, r, err)
		return
	}

	reviewerID := middleware.AdminIDFromContext(r.Context())
	if reviewerID == "" {
		reviewerID = "admin-system"
	}

	item, err := h.service.Review(r.Context(), id, reviewerID, body.Approved, body.ApprovedModules, body.ITNotes)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, item)
}
