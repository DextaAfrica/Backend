package handlers

import (
	"net/http"
	"time"

	"github.com/DextaAfrica/Backend/internal/domain"
	"github.com/DextaAfrica/Backend/internal/service"
	"github.com/DextaAfrica/Backend/internal/transport/http/request"
	"github.com/DextaAfrica/Backend/internal/transport/http/response"
)

type NewsletterHandler struct {
	newsletter *service.NewsletterService
}

func NewNewsletterHandler(newsletter *service.NewsletterService) *NewsletterHandler {
	return &NewsletterHandler{newsletter: newsletter}
}

type subscriberResponse struct {
	Email        string    `json:"email"`
	Status       string    `json:"status"`
	SubscribedAt time.Time `json:"subscribedAt"`
}

func toSubscriberResponse(s *domain.Subscriber) subscriberResponse {
	return subscriberResponse{Email: s.Email, Status: string(s.Status), SubscribedAt: s.SubscribedAt}
}

func (h *NewsletterHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	var req request.SubscribeRequest
	if err := request.Decode(w, r, &req); err != nil {
		response.Error(w, r, err)
		return
	}

	sub, err := h.newsletter.Subscribe(r.Context(), req.Email)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.Created(w, toSubscriberResponse(sub))
}

func (h *NewsletterHandler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	var req request.UnsubscribeRequest
	if err := request.Decode(w, r, &req); err != nil {
		response.Error(w, r, err)
		return
	}

	if err := h.newsletter.Unsubscribe(r.Context(), req.Email); err != nil {
		response.Error(w, r, err)
		return
	}
	response.NoContent(w)
}

func (h *NewsletterHandler) ListAdmin(w http.ResponseWriter, r *http.Request) {
	page := queryInt(r, "page", 1)
	perPage := queryInt(r, "perPage", 20)

	result, err := h.newsletter.List(r.Context(), page, perPage)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	out := make([]subscriberResponse, len(result.Items))
	for i := range result.Items {
		out[i] = toSubscriberResponse(&result.Items[i])
	}
	response.Paginated(w, out, response.Meta{
		Page: result.Page, PerPage: result.PerPage,
		TotalItems: result.TotalItems, TotalPages: result.TotalPages(),
	})
}
