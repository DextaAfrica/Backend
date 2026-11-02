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

type CareerHandler struct {
	careers *service.CareerService
}

func NewCareerHandler(careers *service.CareerService) *CareerHandler {
	return &CareerHandler{careers: careers}
}

type careerListingResponse struct {
	ID             string         `json:"id"`
	Slug           string         `json:"slug"`
	Title          string         `json:"title"`
	Department     string         `json:"department"`
	Location       string         `json:"location"`
	EmploymentType string         `json:"employmentType"`
	Description    map[string]any `json:"description"`
	Published      bool           `json:"published"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

func toCareerListingResponse(c *domain.CareerListing) careerListingResponse {
	return careerListingResponse{
		ID: c.ID, Slug: c.Slug, Title: c.Title, Department: c.Department,
		Location: c.Location, EmploymentType: string(c.EmploymentType),
		Description: c.Description, Published: c.Published, UpdatedAt: c.UpdatedAt,
	}
}

func (h *CareerHandler) ListPublic(w http.ResponseWriter, r *http.Request) {
	h.list(w, r, true)
}

func (h *CareerHandler) ListAdmin(w http.ResponseWriter, r *http.Request) {
	h.list(w, r, false)
}

func (h *CareerHandler) list(w http.ResponseWriter, r *http.Request, publishedOnly bool) {
	page := queryInt(r, "page", 1)
	perPage := queryInt(r, "perPage", 20)

	result, err := h.careers.List(r.Context(), page, perPage, publishedOnly)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	out := make([]careerListingResponse, len(result.Items))
	for i := range result.Items {
		out[i] = toCareerListingResponse(&result.Items[i])
	}
	response.Paginated(w, out, response.Meta{
		Page: result.Page, PerPage: result.PerPage,
		TotalItems: result.TotalItems, TotalPages: result.TotalPages(),
	})
}

func (h *CareerHandler) GetPublic(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	c, err := h.careers.GetBySlug(r.Context(), slug)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.OK(w, toCareerListingResponse(c))
}

func (h *CareerHandler) GetAdmin(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	c, err := h.careers.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.OK(w, toCareerListingResponse(c))
}

func fromSaveCareerListingRequest(req request.SaveCareerListingRequest) *domain.CareerListing {
	return &domain.CareerListing{
		Slug: req.Slug, Title: req.Title, Department: req.Department, Location: req.Location,
		EmploymentType: domain.EmploymentType(req.EmploymentType), Description: req.Description,
		Published: req.Published,
	}
}

func (h *CareerHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req request.SaveCareerListingRequest
	if err := request.Decode(w, r, &req); err != nil {
		response.Error(w, r, err)
		return
	}

	created, err := h.careers.Create(r.Context(), fromSaveCareerListingRequest(req))
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.Created(w, toCareerListingResponse(created))
}

func (h *CareerHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req request.SaveCareerListingRequest
	if err := request.Decode(w, r, &req); err != nil {
		response.Error(w, r, err)
		return
	}

	c := fromSaveCareerListingRequest(req)
	c.ID = id

	updated, err := h.careers.Update(r.Context(), c)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.OK(w, toCareerListingResponse(updated))
}

func (h *CareerHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.careers.Delete(r.Context(), id); err != nil {
		response.Error(w, r, err)
		return
	}
	response.NoContent(w)
}
