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

type PortfolioHandler struct {
	portfolio *service.PortfolioService
}

func NewPortfolioHandler(portfolio *service.PortfolioService) *PortfolioHandler {
	return &PortfolioHandler{portfolio: portfolio}
}

type galleryImageResponse struct {
	URL string `json:"url"`
	Alt string `json:"alt"`
}

type developmentResponse struct {
	ID             string                  `json:"id"`
	Slug           string                  `json:"slug"`
	Name           string                  `json:"name"`
	Summary        string                  `json:"summary"`
	Body           map[string]any          `json:"body"`
	HeroImageURL   string                  `json:"heroImageUrl"`
	Gallery        []galleryImageResponse  `json:"gallery"`
	Location       string                  `json:"location"`
	Status         string                  `json:"status"`
	Featured       bool                    `json:"featured"`
	SEOTitle       string                  `json:"seoTitle"`
	SEODescription string                  `json:"seoDescription"`
	Published      bool                    `json:"published"`
	PublishedAt    *time.Time              `json:"publishedAt,omitempty"`
	DisplayOrder   int                     `json:"displayOrder"`
	UpdatedAt      time.Time               `json:"updatedAt"`
}

func toDevelopmentResponse(d *domain.Development) developmentResponse {
	gallery := make([]galleryImageResponse, len(d.Gallery))
	for i, g := range d.Gallery {
		gallery[i] = galleryImageResponse{URL: g.URL, Alt: g.Alt}
	}
	return developmentResponse{
		ID: d.ID, Slug: d.Slug, Name: d.Name, Summary: d.Summary, Body: d.Body,
		HeroImageURL: d.HeroImageURL, Gallery: gallery, Location: d.Location,
		Status: string(d.Status), Featured: d.Featured, SEOTitle: d.SEOTitle,
		SEODescription: d.SEODescription, Published: d.Published, PublishedAt: d.PublishedAt,
		DisplayOrder: d.DisplayOrder, UpdatedAt: d.UpdatedAt,
	}
}

func (h *PortfolioHandler) ListPublic(w http.ResponseWriter, r *http.Request) {
	h.list(w, r, true)
}

func (h *PortfolioHandler) ListAdmin(w http.ResponseWriter, r *http.Request) {
	h.list(w, r, false)
}

func (h *PortfolioHandler) list(w http.ResponseWriter, r *http.Request, publishedOnly bool) {
	page := queryInt(r, "page", 1)
	perPage := queryInt(r, "perPage", 20)

	result, err := h.portfolio.List(r.Context(), page, perPage, publishedOnly)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	out := make([]developmentResponse, len(result.Items))
	for i := range result.Items {
		out[i] = toDevelopmentResponse(&result.Items[i])
	}
	response.Paginated(w, out, response.Meta{
		Page: result.Page, PerPage: result.PerPage,
		TotalItems: result.TotalItems, TotalPages: result.TotalPages(),
	})
}

func (h *PortfolioHandler) GetPublic(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	d, err := h.portfolio.GetBySlug(r.Context(), slug)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.OK(w, toDevelopmentResponse(d))
}

func (h *PortfolioHandler) GetAdmin(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	d, err := h.portfolio.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.OK(w, toDevelopmentResponse(d))
}

func fromSaveDevelopmentRequest(req request.SaveDevelopmentRequest) *domain.Development {
	gallery := make([]domain.GalleryImage, len(req.Gallery))
	for i, g := range req.Gallery {
		gallery[i] = domain.GalleryImage{URL: g.URL, Alt: g.Alt}
	}
	return &domain.Development{
		Slug: req.Slug, Name: req.Name, Summary: req.Summary, Body: req.Body,
		HeroImageURL: req.HeroImageURL, Gallery: gallery, Location: req.Location,
		Status: domain.DevelopmentStatus(req.Status), Featured: req.Featured,
		SEOTitle: req.SEOTitle, SEODescription: req.SEODescription,
		Published: req.Published, DisplayOrder: req.DisplayOrder,
	}
}

func (h *PortfolioHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req request.SaveDevelopmentRequest
	if err := request.Decode(w, r, &req); err != nil {
		response.Error(w, r, err)
		return
	}

	created, err := h.portfolio.Create(r.Context(), fromSaveDevelopmentRequest(req))
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.Created(w, toDevelopmentResponse(created))
}

func (h *PortfolioHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req request.SaveDevelopmentRequest
	if err := request.Decode(w, r, &req); err != nil {
		response.Error(w, r, err)
		return
	}

	d := fromSaveDevelopmentRequest(req)
	d.ID = id

	updated, err := h.portfolio.Update(r.Context(), d)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.OK(w, toDevelopmentResponse(updated))
}

func (h *PortfolioHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.portfolio.Delete(r.Context(), id); err != nil {
		response.Error(w, r, err)
		return
	}
	response.NoContent(w)
}
