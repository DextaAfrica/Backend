package handlers

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/DextaAfrica/Backend/internal/domain"
	"github.com/DextaAfrica/Backend/internal/service"
	"github.com/DextaAfrica/Backend/internal/transport/http/request"
	"github.com/DextaAfrica/Backend/internal/transport/http/response"
)

type FormHandler struct {
	forms *service.FormService
}

func NewFormHandler(forms *service.FormService) *FormHandler {
	return &FormHandler{forms: forms}
}

type customFormResponse struct {
	ID               string    `json:"id"`
	Slug             string    `json:"slug"`
	Title            string    `json:"title"`
	Description      string    `json:"description"`
	FormType         string    `json:"formType"`
	Status           string    `json:"status"`
	CoverImageURL    string    `json:"coverImageUrl"`
	EventDate        *string   `json:"eventDate"`
	EventLocation    string    `json:"eventLocation"`
	Fields           any       `json:"fields"`
	Metadata         any       `json:"metadata"`
	SubmissionsCount int       `json:"submissionsCount"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

func toCustomFormResponse(f *domain.CustomForm) customFormResponse {
	var fields any
	if len(f.Fields) > 0 {
		_ = json.Unmarshal(f.Fields, &fields)
	}
	if fields == nil {
		fields = []any{}
	}

	var meta any
	if len(f.Metadata) > 0 {
		_ = json.Unmarshal(f.Metadata, &meta)
	}
	if meta == nil {
		meta = map[string]any{}
	}

	var eventDateStr *string
	if f.EventDate != nil {
		s := f.EventDate.Format(time.RFC3339)
		eventDateStr = &s
	}

	return customFormResponse{
		ID:               f.ID,
		Slug:             f.Slug,
		Title:            f.Title,
		Description:      f.Description,
		FormType:         string(f.FormType),
		Status:           string(f.Status),
		CoverImageURL:    f.CoverImageURL,
		EventDate:        eventDateStr,
		EventLocation:    f.EventLocation,
		Fields:           fields,
		Metadata:         meta,
		SubmissionsCount: f.SubmissionsCount,
		CreatedAt:        f.CreatedAt,
		UpdatedAt:        f.UpdatedAt,
	}
}

type formSubmissionResponse struct {
	ID             string    `json:"id"`
	FormID         string    `json:"formId"`
	FullName       string    `json:"fullName"`
	Email          string    `json:"email"`
	Phone          string    `json:"phone"`
	Company        string    `json:"company"`
	JobTitle       string    `json:"jobTitle"`
	Category       string    `json:"category"`
	AttendanceMode string    `json:"attendanceMode"`
	Answers        any       `json:"answers"`
	Notes          string    `json:"notes"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"createdAt"`
}

func toSubmissionResponse(s *domain.FormSubmission) formSubmissionResponse {
	var answers any
	if len(s.Answers) > 0 {
		_ = json.Unmarshal(s.Answers, &answers)
	}
	if answers == nil {
		answers = map[string]any{}
	}

	return formSubmissionResponse{
		ID:             s.ID,
		FormID:         s.FormID,
		FullName:       s.FullName,
		Email:          s.Email,
		Phone:          s.Phone,
		Company:        s.Company,
		JobTitle:       s.JobTitle,
		Category:       s.Category,
		AttendanceMode: s.AttendanceMode,
		Answers:        answers,
		Notes:          s.Notes,
		Status:         s.Status,
		CreatedAt:      s.CreatedAt,
	}
}

// Public: GET /forms/{slug}
func (h *FormHandler) GetPublic(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	f, err := h.forms.GetBySlug(r.Context(), slug)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.OK(w, toCustomFormResponse(f))
}

// Public: POST /forms/{slug}/submit
func (h *FormHandler) Submit(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	var req request.SubmitFormRequest
	if err := request.Decode(w, r, &req); err != nil {
		response.Error(w, r, err)
		return
	}

	answersBytes, _ := json.Marshal(req.Answers)

	sub := &domain.FormSubmission{
		FullName:       req.FullName,
		Email:          req.Email,
		Phone:          req.Phone,
		Company:        req.Company,
		JobTitle:       req.JobTitle,
		Category:       req.Category,
		AttendanceMode: req.AttendanceMode,
		Answers:        answersBytes,
		Notes:          req.Notes,
	}

	created, err := h.forms.Submit(r.Context(), slug, sub)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.Created(w, toSubmissionResponse(created))
}

// Admin: GET /admin/forms
func (h *FormHandler) ListAdmin(w http.ResponseWriter, r *http.Request) {
	page := queryInt(r, "page", 1)
	perPage := queryInt(r, "perPage", 50)

	result, err := h.forms.ListForms(r.Context(), page, perPage)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	out := make([]customFormResponse, len(result.Items))
	for i := range result.Items {
		out[i] = toCustomFormResponse(&result.Items[i])
	}

	response.Paginated(w, out, response.Meta{
		Page:       result.Page,
		PerPage:    result.PerPage,
		TotalItems: result.TotalItems,
		TotalPages: result.TotalPages(),
	})
}

// Admin: POST /admin/forms
func (h *FormHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req request.SaveCustomFormRequest
	if err := request.Decode(w, r, &req); err != nil {
		response.Error(w, r, err)
		return
	}

	fieldsBytes, _ := json.Marshal(req.Fields)
	metaBytes, _ := json.Marshal(req.Metadata)

	var evDate *time.Time
	if req.EventDate != "" {
		if t, err := time.Parse(time.RFC3339, req.EventDate); err == nil {
			evDate = &t
		}
	}

	form := &domain.CustomForm{
		Slug:          req.Slug,
		Title:         req.Title,
		Description:   req.Description,
		FormType:      domain.FormType(req.FormType),
		Status:        domain.FormStatus(req.Status),
		CoverImageURL: req.CoverImageURL,
		EventDate:     evDate,
		EventLocation: req.EventLocation,
		Fields:        fieldsBytes,
		Metadata:      metaBytes,
	}

	created, err := h.forms.CreateForm(r.Context(), form)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.Created(w, toCustomFormResponse(created))
}

// Admin: GET /admin/forms/{id}
func (h *FormHandler) GetAdmin(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	f, err := h.forms.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.OK(w, toCustomFormResponse(f))
}

// Admin: PUT /admin/forms/{id}
func (h *FormHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req request.SaveCustomFormRequest
	if err := request.Decode(w, r, &req); err != nil {
		response.Error(w, r, err)
		return
	}

	fieldsBytes, _ := json.Marshal(req.Fields)
	metaBytes, _ := json.Marshal(req.Metadata)

	var evDate *time.Time
	if req.EventDate != "" {
		if t, err := time.Parse(time.RFC3339, req.EventDate); err == nil {
			evDate = &t
		}
	}

	form := &domain.CustomForm{
		ID:            id,
		Slug:          req.Slug,
		Title:         req.Title,
		Description:   req.Description,
		FormType:      domain.FormType(req.FormType),
		Status:        domain.FormStatus(req.Status),
		CoverImageURL: req.CoverImageURL,
		EventDate:     evDate,
		EventLocation: req.EventLocation,
		Fields:        fieldsBytes,
		Metadata:      metaBytes,
	}

	updated, err := h.forms.UpdateForm(r.Context(), form)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.OK(w, toCustomFormResponse(updated))
}

// Admin: DELETE /admin/forms/{id}
func (h *FormHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.forms.DeleteForm(r.Context(), id); err != nil {
		response.Error(w, r, err)
		return
	}
	response.NoContent(w)
}

// Admin: GET /admin/forms/{id}/submissions
func (h *FormHandler) ListSubmissions(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	page := queryInt(r, "page", 1)
	perPage := queryInt(r, "perPage", 50)

	result, err := h.forms.ListSubmissions(r.Context(), id, page, perPage)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	out := make([]formSubmissionResponse, len(result.Items))
	for i := range result.Items {
		out[i] = toSubmissionResponse(&result.Items[i])
	}

	response.Paginated(w, out, response.Meta{
		Page:       result.Page,
		PerPage:    result.PerPage,
		TotalItems: result.TotalItems,
		TotalPages: result.TotalPages(),
	})
}

// Admin: GET /admin/forms/{id}/demographics
func (h *FormHandler) GetDemographics(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	demo, err := h.forms.GetDemographics(r.Context(), id)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.OK(w, demo)
}

// Admin: GET /admin/forms/{id}/export/csv
func (h *FormHandler) ExportCSV(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	form, err := h.forms.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	result, err := h.forms.ListSubmissions(r.Context(), id, 1, 10000)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	filename := fmt.Sprintf("%s-registrations-%s.csv", form.Slug, time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))

	writer := csv.NewWriter(w)
	defer writer.Flush()

	_ = writer.Write([]string{
		"ID", "Full Name", "Email", "Phone", "Company", "Job Title",
		"Category", "Attendance Mode", "Status", "Notes", "Registration Date",
	})

	for _, s := range result.Items {
		_ = writer.Write([]string{
			s.ID,
			s.FullName,
			s.Email,
			s.Phone,
			s.Company,
			s.JobTitle,
			s.Category,
			s.AttendanceMode,
			s.Status,
			s.Notes,
			s.CreatedAt.Format(time.RFC3339),
		})
	}
}
