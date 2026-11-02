package handlers

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/DextaAfrica/Backend/internal/apperror"
	"github.com/DextaAfrica/Backend/internal/domain"
	"github.com/DextaAfrica/Backend/internal/service"
	"github.com/DextaAfrica/Backend/internal/transport/http/request"
	"github.com/DextaAfrica/Backend/internal/transport/http/response"
)

type ContentHandler struct {
	content *service.ContentService
}

func NewContentHandler(content *service.ContentService) *ContentHandler {
	return &ContentHandler{content: content}
}

type pageResponse struct {
	Key            string         `json:"key"`
	Title          string         `json:"title"`
	Content        map[string]any `json:"content"`
	SEOTitle       string         `json:"seoTitle"`
	SEODescription string         `json:"seoDescription"`
	Published      bool           `json:"published"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

func toPageResponse(p *domain.Page) pageResponse {
	return pageResponse{
		Key: p.Key, Title: p.Title, Content: p.Content,
		SEOTitle: p.SEOTitle, SEODescription: p.SEODescription,
		Published: p.Published, UpdatedAt: p.UpdatedAt,
	}
}

// GetPublic serves a single page by key for the public frontend. Unpublished
// pages 404 for anonymous callers so draft content in the CMS is never
// accidentally exposed.
func (h *ContentHandler) GetPublic(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	page, err := h.content.GetPage(r.Context(), key)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	if !page.Published {
		response.Error(w, r, apperror.NotFound("page"))
		return
	}
	response.OK(w, toPageResponse(page))
}

func (h *ContentHandler) GetAdmin(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	page, err := h.content.GetPage(r.Context(), key)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.OK(w, toPageResponse(page))
}

func (h *ContentHandler) ListAdmin(w http.ResponseWriter, r *http.Request) {
	pages, err := h.content.ListPages(r.Context())
	if err != nil {
		response.Error(w, r, err)
		return
	}
	out := make([]pageResponse, len(pages))
	for i, p := range pages {
		out[i] = toPageResponse(&p)
	}
	response.OK(w, out)
}

func (h *ContentHandler) Save(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")

	var req request.SavePageRequest
	if err := request.Decode(w, r, &req); err != nil {
		response.Error(w, r, err)
		return
	}

	page := &domain.Page{
		Key: key, Title: req.Title, Content: req.Content,
		SEOTitle: req.SEOTitle, SEODescription: req.SEODescription, Published: req.Published,
	}

	saved, err := h.content.SavePage(r.Context(), page)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.OK(w, toPageResponse(saved))
}
