package handlers

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/DextaAfrica/Backend/internal/domain"
	"github.com/DextaAfrica/Backend/internal/service"
	"github.com/DextaAfrica/Backend/internal/transport/http/request"
	"github.com/DextaAfrica/Backend/internal/transport/http/response"
)

type EnquiryHandler struct {
	enquiries *service.EnquiryService
}

func NewEnquiryHandler(enquiries *service.EnquiryService) *EnquiryHandler {
	return &EnquiryHandler{enquiries: enquiries}
}

type enquiryResponse struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Email      string    `json:"email"`
	Phone      string    `json:"phone"`
	Subject    string    `json:"subject"`
	Message    string    `json:"message"`
	SourcePage string    `json:"sourcePage"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"createdAt"`
}

func toEnquiryResponse(e *domain.Enquiry) enquiryResponse {
	return enquiryResponse{
		ID: e.ID, Name: e.Name, Email: e.Email, Phone: e.Phone, Subject: e.Subject,
		Message: e.Message, SourcePage: e.SourcePage, Status: string(e.Status), CreatedAt: e.CreatedAt,
	}
}

// Submit is the public contact-form endpoint. It is rate-limited per IP at
// the router level (see routes.go) since it takes unauthenticated input.
func (h *EnquiryHandler) Submit(w http.ResponseWriter, r *http.Request) {
	var req request.SubmitEnquiryRequest
	if err := request.Decode(w, r, &req); err != nil {
		response.Error(w, r, err)
		return
	}

	enquiry := &domain.Enquiry{
		Name: req.Name, Email: req.Email, Phone: req.Phone,
		Subject: req.Subject, Message: req.Message, SourcePage: req.SourcePage,
	}

	created, err := h.enquiries.Submit(r.Context(), enquiry)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.Created(w, toEnquiryResponse(created))
}

func (h *EnquiryHandler) ListAdmin(w http.ResponseWriter, r *http.Request) {
	page := queryInt(r, "page", 1)
	perPage := queryInt(r, "perPage", 20)

	result, err := h.enquiries.List(r.Context(), page, perPage)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	out := make([]enquiryResponse, len(result.Items))
	for i := range result.Items {
		out[i] = toEnquiryResponse(&result.Items[i])
	}
	response.Paginated(w, out, response.Meta{
		Page: result.Page, PerPage: result.PerPage,
		TotalItems: result.TotalItems, TotalPages: result.TotalPages(),
	})
}

func (h *EnquiryHandler) GetAdmin(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	e, err := h.enquiries.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.OK(w, toEnquiryResponse(e))
}

func (h *EnquiryHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req request.UpdateEnquiryStatusRequest
	if err := request.Decode(w, r, &req); err != nil {
		response.Error(w, r, err)
		return
	}

	if err := h.enquiries.UpdateStatus(r.Context(), id, domain.EnquiryStatus(req.Status)); err != nil {
		response.Error(w, r, err)
		return
	}
	response.NoContent(w)
}
