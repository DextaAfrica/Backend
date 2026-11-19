package handlers

import (
	"net/http"
	"strings"
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
	ID                  string         `json:"id"`
	Slug                string         `json:"slug"`
	Title               string         `json:"title"`
	Department          string         `json:"department"`
	Location            string         `json:"location"`
	EmploymentType      string         `json:"employmentType"`
	Type                string         `json:"type"`
	Summary             string         `json:"summary,omitempty"`
	Requirements        []string       `json:"requirements,omitempty"`
	Benefits            []string       `json:"benefits,omitempty"`
	Description         map[string]any `json:"description"`
	Status              string         `json:"status"`
	Published           bool           `json:"published"`
	ApplicantClickCount int            `json:"applicantClickCount"`
	UpdatedAt           time.Time      `json:"updatedAt"`
}

func toCareerListingResponse(c *domain.CareerListing) careerListingResponse {
	resp := careerListingResponse{
		ID: c.ID, Slug: c.Slug, Title: c.Title, Department: c.Department,
		Location: c.Location, EmploymentType: string(c.EmploymentType),
		Type: string(c.EmploymentType), Description: c.Description,
		Status: c.Status, Published: c.Published,
		ApplicantClickCount: c.ApplicantClickCount, UpdatedAt: c.UpdatedAt,
	}
	if c.Description != nil {
		if summary, ok := c.Description["summary"].(string); ok {
			resp.Summary = summary
		}
		if reqs, ok := c.Description["requirements"].([]any); ok {
			for _, r := range reqs {
				if s, ok := r.(string); ok {
					resp.Requirements = append(resp.Requirements, s)
				}
			}
		}
		if bens, ok := c.Description["benefits"].([]any); ok {
			for _, b := range bens {
				if s, ok := b.(string); ok {
					resp.Benefits = append(resp.Benefits, s)
				}
			}
		}
	}
	return resp
}

func (h *CareerHandler) ListPublic(w http.ResponseWriter, r *http.Request) {
	h.list(w, r, true)
}

func (h *CareerHandler) ListAdmin(w http.ResponseWriter, r *http.Request) {
	h.list(w, r, false)
}

func (h *CareerHandler) list(w http.ResponseWriter, r *http.Request, publishedOnly bool) {
	page := queryInt(r, "page", 1)
	perPage := queryInt(r, "perPage", 50)

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
	empTypeStr := req.EmploymentType
	if empTypeStr == "" {
		empTypeStr = req.Type
	}
	empType := normalizeEmploymentType(empTypeStr)

	desc := req.Description
	if desc == nil {
		desc = make(map[string]any)
	}
	if req.Summary != "" {
		desc["summary"] = req.Summary
	}
	if len(req.Requirements) > 0 {
		desc["requirements"] = req.Requirements
	}
	if len(req.Benefits) > 0 {
		desc["benefits"] = req.Benefits
	}

	status := req.Status
	if status == "" {
		status = "open"
	}

	return &domain.CareerListing{
		Slug: req.Slug, Title: req.Title, Department: req.Department, Location: req.Location,
		EmploymentType: empType, Description: desc,
		Status: status, Published: req.Published,
	}
}

func normalizeEmploymentType(val string) domain.EmploymentType {
	clean := strings.ToLower(strings.TrimSpace(val))
	clean = strings.ReplaceAll(clean, "-", "_")
	switch clean {
	case "full_time", "fulltime", "full time":
		return domain.EmploymentTypeFullTime
	case "part_time", "parttime", "part time":
		return domain.EmploymentTypePartTime
	case "contract":
		return domain.EmploymentTypeContract
	case "internship", "intern":
		return domain.EmploymentTypeInternship
	default:
		return domain.EmploymentTypeFullTime
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

func (h *CareerHandler) SubmitInquiry(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	listing, err := h.careers.GetBySlug(r.Context(), slug)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	var req request.SubmitCareerInquiryRequest
	if err := request.Decode(w, r, &req); err != nil {
		response.Error(w, r, err)
		return
	}

	inq := &domain.CareerInquiry{
		CareerListingID: listing.ID,
		ApplicantName:   req.ApplicantName,
		ApplicantEmail:  req.ApplicantEmail,
		ApplicantPhone:  req.ApplicantPhone,
		ResumeURL:       req.ResumeURL,
		CoverLetter:     req.CoverLetter,
		Source:          req.Source,
	}
	if inq.Source == "" {
		inq.Source = "website"
	}

	if err := h.careers.RecordInquiry(r.Context(), inq); err != nil {
		response.Error(w, r, err)
		return
	}

	response.Created(w, map[string]any{
		"success": true,
		"message": "application submitted successfully",
		"id":      inq.ID,
	})
}

func (h *CareerHandler) ListInquiries(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	inquiries, err := h.careers.ListInquiries(r.Context(), id)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.OK(w, map[string]any{
		"data": inquiries,
	})
}
