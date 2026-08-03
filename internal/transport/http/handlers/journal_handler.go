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

type JournalHandler struct {
	journal *service.JournalService
}

func NewJournalHandler(journal *service.JournalService) *JournalHandler {
	return &JournalHandler{journal: journal}
}

type articleResponse struct {
	ID             string         `json:"id"`
	Slug           string         `json:"slug"`
	Title          string         `json:"title"`
	Excerpt        string         `json:"excerpt"`
	Body           map[string]any `json:"body"`
	CoverImageURL  string         `json:"coverImageUrl"`
	Author         string         `json:"author"`
	Category       string         `json:"category"`
	SEOTitle       string         `json:"seoTitle"`
	SEODescription string         `json:"seoDescription"`
	Published      bool           `json:"published"`
	PublishedAt    *time.Time     `json:"publishedAt,omitempty"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

func toArticleResponse(a *domain.Article) articleResponse {
	return articleResponse{
		ID: a.ID, Slug: a.Slug, Title: a.Title, Excerpt: a.Excerpt, Body: a.Body,
		CoverImageURL: a.CoverImageURL, Author: a.Author, Category: a.Category,
		SEOTitle: a.SEOTitle, SEODescription: a.SEODescription,
		Published: a.Published, PublishedAt: a.PublishedAt, UpdatedAt: a.UpdatedAt,
	}
}

func (h *JournalHandler) ListPublic(w http.ResponseWriter, r *http.Request) {
	h.list(w, r, true)
}

func (h *JournalHandler) ListAdmin(w http.ResponseWriter, r *http.Request) {
	h.list(w, r, false)
}

func (h *JournalHandler) list(w http.ResponseWriter, r *http.Request, publishedOnly bool) {
	page := queryInt(r, "page", 1)
	perPage := queryInt(r, "perPage", 20)

	result, err := h.journal.List(r.Context(), page, perPage, publishedOnly)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	out := make([]articleResponse, len(result.Items))
	for i := range result.Items {
		out[i] = toArticleResponse(&result.Items[i])
	}
	response.Paginated(w, out, response.Meta{
		Page: result.Page, PerPage: result.PerPage,
		TotalItems: result.TotalItems, TotalPages: result.TotalPages(),
	})
}

func (h *JournalHandler) GetPublic(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	a, err := h.journal.GetBySlug(r.Context(), slug)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.OK(w, toArticleResponse(a))
}

func (h *JournalHandler) GetAdmin(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	a, err := h.journal.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.OK(w, toArticleResponse(a))
}

func fromSaveArticleRequest(req request.SaveArticleRequest) *domain.Article {
	return &domain.Article{
		Slug: req.Slug, Title: req.Title, Excerpt: req.Excerpt, Body: req.Body,
		CoverImageURL: req.CoverImageURL, Author: req.Author, Category: req.Category,
		SEOTitle: req.SEOTitle, SEODescription: req.SEODescription, Published: req.Published,
	}
}

func (h *JournalHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req request.SaveArticleRequest
	if err := request.Decode(w, r, &req); err != nil {
		response.Error(w, r, err)
		return
	}

	created, err := h.journal.Create(r.Context(), fromSaveArticleRequest(req))
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.Created(w, toArticleResponse(created))
}

func (h *JournalHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req request.SaveArticleRequest
	if err := request.Decode(w, r, &req); err != nil {
		response.Error(w, r, err)
		return
	}

	a := fromSaveArticleRequest(req)
	a.ID = id

	updated, err := h.journal.Update(r.Context(), a)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.OK(w, toArticleResponse(updated))
}

func (h *JournalHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.journal.Delete(r.Context(), id); err != nil {
		response.Error(w, r, err)
		return
	}
	response.NoContent(w)
}
