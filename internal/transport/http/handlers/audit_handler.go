package handlers

import (
	"net/http"

	"github.com/DextaAfrica/Backend/internal/domain"
	"github.com/DextaAfrica/Backend/internal/service"
	"github.com/DextaAfrica/Backend/internal/transport/http/request"
	"github.com/DextaAfrica/Backend/internal/transport/http/response"
)

type AuditHandler struct {
	audit *service.AuditService
}

func NewAuditHandler(audit *service.AuditService) *AuditHandler {
	return &AuditHandler{audit: audit}
}

func (h *AuditHandler) Record(w http.ResponseWriter, r *http.Request) {
	var req request.RecordAuditRequest
	if err := request.Decode(w, r, &req); err != nil {
		response.Error(w, r, err)
		return
	}

	log := &domain.AuditLog{
		ActorID:    req.ActorID,
		ActorEmail: req.ActorEmail,
		Action:     req.Action,
		Entity:     req.Entity,
		EntityID:   req.EntityID,
		IPAddress:  req.IPAddress,
		UserAgent:  req.UserAgent,
		Metadata:   req.Metadata,
	}

	if err := h.audit.Record(r.Context(), log); err != nil {
		response.Error(w, r, err)
		return
	}

	response.Created(w, map[string]any{
		"success": true,
		"id":      log.ID,
	})
}

func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	page := queryInt(r, "page", 1)
	perPage := queryInt(r, "perPage", 50)

	result, err := h.audit.List(r.Context(), page, perPage)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.Paginated(w, result.Items, response.Meta{
		Page:       result.Page,
		PerPage:    result.PerPage,
		TotalItems: result.TotalItems,
		TotalPages: result.TotalPages(),
	})
}
