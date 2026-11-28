package handlers

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/DextaAfrica/Backend/internal/domain"
	"github.com/DextaAfrica/Backend/internal/service"
	"github.com/DextaAfrica/Backend/internal/transport/http/request"
	"github.com/DextaAfrica/Backend/internal/transport/http/response"
)

type VisitorHandler struct {
	visitors *service.VisitorService
}

func NewVisitorHandler(visitors *service.VisitorService) *VisitorHandler {
	return &VisitorHandler{visitors: visitors}
}

type visitorCheckInRequest struct {
	FullName         string `json:"fullName"`
	Phone            string `json:"phone"`
	Email            string `json:"email"`
	Company          string `json:"company"`
	Position         string `json:"position"`
	Purpose          string `json:"purpose"`
	PurposeOther     string `json:"purposeOther"`
	HostName         string `json:"hostName"`
	Department       string `json:"department"`
	HasAppointment   bool   `json:"hasAppointment"`
	InvitedBy        string `json:"invitedBy"`
	VisitDescription string `json:"visitDescription"`
	AgreedGuidelines bool   `json:"agreedGuidelines"`
	ConsentedData    bool   `json:"consentedData"`
}

type visitorCheckOutRequest struct {
	Identifier string `json:"identifier"`
	PassCode   string `json:"passCode"`
}

type visitorResponse struct {
	ID               string  `json:"id"`
	PassCode         string  `json:"passCode"`
	FullName         string  `json:"fullName"`
	Phone            string  `json:"phone"`
	Email            string  `json:"email"`
	Company          string  `json:"company"`
	Position         string  `json:"position"`
	Purpose          string  `json:"purpose"`
	PurposeOther     string  `json:"purposeOther"`
	HostName         string  `json:"hostName"`
	Department       string  `json:"department"`
	HasAppointment   bool    `json:"hasAppointment"`
	InvitedBy        string  `json:"invitedBy"`
	VisitDescription string  `json:"visitDescription"`
	VisitDate        string  `json:"visitDate"`
	CheckInAt        string  `json:"checkInAt"`
	CheckOutAt       *string `json:"checkOutAt"`
	Status           string  `json:"status"`
	AgreedGuidelines bool    `json:"agreedGuidelines"`
	ConsentedData    bool    `json:"consentedData"`
	DeclaredAt       string  `json:"declaredAt"`
	CheckedOutBy     string  `json:"checkedOutBy"`
	CreatedAt        string  `json:"createdAt"`
}

func toVisitorResponse(v *domain.Visitor) visitorResponse {
	var checkOutStr *string
	if v.CheckOutAt != nil {
		s := v.CheckOutAt.Format(time.RFC3339)
		checkOutStr = &s
	}
	return visitorResponse{
		ID:               v.ID,
		PassCode:         v.PassCode,
		FullName:         v.FullName,
		Phone:            v.Phone,
		Email:            v.Email,
		Company:          v.Company,
		Position:         v.Position,
		Purpose:          v.Purpose,
		PurposeOther:     v.PurposeOther,
		HostName:         v.HostName,
		Department:       v.Department,
		HasAppointment:   v.HasAppointment,
		InvitedBy:        v.InvitedBy,
		VisitDescription: v.VisitDescription,
		VisitDate:        v.VisitDate.Format("2006-01-02"),
		CheckInAt:        v.CheckInAt.Format(time.RFC3339),
		CheckOutAt:       checkOutStr,
		Status:           string(v.Status),
		AgreedGuidelines: v.AgreedGuidelines,
		ConsentedData:    v.ConsentedData,
		DeclaredAt:       v.DeclaredAt.Format(time.RFC3339),
		CheckedOutBy:     v.CheckedOutBy,
		CreatedAt:        v.CreatedAt.Format(time.RFC3339),
	}
}

// Public: POST /visitors/check-in
func (h *VisitorHandler) CheckIn(w http.ResponseWriter, r *http.Request) {
	var req visitorCheckInRequest
	if err := request.Decode(w, r, &req); err != nil {
		response.Error(w, r, err)
		return
	}

	v := &domain.Visitor{
		FullName:         req.FullName,
		Phone:            req.Phone,
		Email:            req.Email,
		Company:          req.Company,
		Position:         req.Position,
		Purpose:          req.Purpose,
		PurposeOther:     req.PurposeOther,
		HostName:         req.HostName,
		Department:       req.Department,
		HasAppointment:   req.HasAppointment,
		InvitedBy:        req.InvitedBy,
		VisitDescription: req.VisitDescription,
		AgreedGuidelines: req.AgreedGuidelines,
		ConsentedData:    req.ConsentedData,
		IPAddress:        r.RemoteAddr,
		UserAgent:        r.UserAgent(),
	}

	created, err := h.visitors.CheckIn(r.Context(), v)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.Created(w, toVisitorResponse(created))
}

// Public: GET /visitors/pass/{code}
func (h *VisitorHandler) GetByPassCode(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	v, err := h.visitors.GetByPassCode(r.Context(), code)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.OK(w, toVisitorResponse(v))
}

// Public: POST /visitors/check-out
func (h *VisitorHandler) SelfCheckOut(w http.ResponseWriter, r *http.Request) {
	var req visitorCheckOutRequest
	if err := request.Decode(w, r, &req); err != nil {
		response.Error(w, r, err)
		return
	}
	id := req.Identifier
	if id == "" {
		id = req.PassCode
	}
	updated, err := h.visitors.SelfCheckOut(r.Context(), id)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.OK(w, toVisitorResponse(updated))
}

// Admin: GET /admin/visitors
func (h *VisitorHandler) ListAdmin(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := domain.VisitorFilter{
		Status:     q.Get("status"),
		Department: q.Get("department"),
		Purpose:    q.Get("purpose"),
		Date:       q.Get("date"),
		Query:      q.Get("q"),
		Page:       queryInt(r, "page", 1),
		PerPage:    queryInt(r, "perPage", 50),
	}

	page, err := h.visitors.List(r.Context(), filter)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	items := make([]visitorResponse, 0, len(page.Items))
	for i := range page.Items {
		items = append(items, toVisitorResponse(&page.Items[i]))
	}
	response.Paginated(w, items, response.Meta{
		Page:       page.Page,
		PerPage:    page.PerPage,
		TotalItems: page.TotalItems,
		TotalPages: page.TotalPages(),
	})
}

// Admin: PATCH /admin/visitors/{id}/check-out
func (h *VisitorHandler) AdminCheckOut(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	adminEmail := r.Header.Get("X-Admin-Email")
	updated, err := h.visitors.AdminCheckOut(r.Context(), id, adminEmail)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.OK(w, toVisitorResponse(updated))
}

// Admin: GET /admin/visitors/analytics
func (h *VisitorHandler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	analytics, err := h.visitors.GetAnalytics(r.Context())
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.OK(w, analytics)
}

func safeCSVField(s string) string {
	if s == "" {
		return s
	}
	first := s[0]
	if first == '=' || first == '@' || first == '\t' || first == '\r' {
		return "'" + s
	}
	if (first == '+' || first == '-') && !strings.HasPrefix(s, "+") {
		return "'" + s
	}
	return s
}

// Admin: GET /admin/visitors/export/csv
func (h *VisitorHandler) ExportCSV(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := domain.VisitorFilter{
		Status:     q.Get("status"),
		Department: q.Get("department"),
		Purpose:    q.Get("purpose"),
		Date:       q.Get("date"),
		Query:      q.Get("q"),
		Page:       1,
		PerPage:    500,
	}

	page, err := h.visitors.List(r.Context(), filter)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"dexta-visitors-%s.csv\"", time.Now().Format("2006-01-02")))

	cw := csv.NewWriter(w)
	_ = cw.Write([]string{
		"Pass Code", "Full Name", "Phone", "Email", "Company", "Position",
		"Purpose", "Purpose Detail", "Host", "Department", "Scheduled",
		"Invited By", "Date of Visit", "Time of Arrival", "Time of Departure",
		"Status", "Guidelines Agreed", "Data Consent",
	})

	for _, v := range page.Items {
		checkOut := ""
		if v.CheckOutAt != nil {
			checkOut = v.CheckOutAt.Format(time.RFC3339)
		}
		_ = cw.Write([]string{
			safeCSVField(v.PassCode),
			safeCSVField(v.FullName),
			safeCSVField(v.Phone),
			safeCSVField(v.Email),
			safeCSVField(v.Company),
			safeCSVField(v.Position),
			safeCSVField(v.Purpose),
			safeCSVField(v.PurposeOther),
			safeCSVField(v.HostName),
			safeCSVField(v.Department),
			fmt.Sprintf("%t", v.HasAppointment),
			safeCSVField(v.InvitedBy),
			v.VisitDate.Format("2006-01-02"),
			v.CheckInAt.Format(time.RFC3339),
			checkOut,
			string(v.Status),
			fmt.Sprintf("%t", v.AgreedGuidelines),
			fmt.Sprintf("%t", v.ConsentedData),
		})
	}
	cw.Flush()
}
