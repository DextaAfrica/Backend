package handlers

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/DextaAfrica/Backend/internal/domain/crm"
	"github.com/DextaAfrica/Backend/internal/domain/outbox"
	"github.com/DextaAfrica/Backend/internal/infrastructure/database"
	"github.com/DextaAfrica/Backend/internal/interfaces/http/response"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CRMHandler struct {
	db *database.PostgresDB
}

func NewCRMHandler(db *database.PostgresDB) *CRMHandler {
	return &CRMHandler{db: db}
}

type InquiryRequest struct {
	FullName          string  `json:"full_name"`
	Email             string  `json:"email"`
	PhoneNumber       string  `json:"phone_number"`
	Country           string  `json:"country"`
	BudgetMin         float64 `json:"budget_min"`
	BudgetMax         float64 `json:"budget_max"`
	Currency          string  `json:"currency"`
	PreferredProperty string  `json:"preferred_property"`
	Message           string  `json:"message"`
	Source            string  `json:"source"`
}

// SubmitInquiry captures leads and writes to Transactional Outbox for auto-reply and sales assignment.
func (h *CRMHandler) SubmitInquiry(c *fiber.Ctx) error {
	var req InquiryRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", "INVALID_INPUT", nil)
	}

	if req.FullName == "" || req.Email == "" || req.PhoneNumber == "" {
		return response.Error(c, fiber.StatusUnprocessableEntity, "Name, email and phone number are required", "VALIDATION_ERROR", []response.ErrorDetail{
			{Field: "full_name", Issue: "Full name is required"},
			{Field: "email", Issue: "Email address is required"},
			{Field: "phone_number", Issue: "Phone number is required"},
		})
	}

	lead := crm.Lead{
		FullName:          req.FullName,
		Email:             req.Email,
		PhoneNumber:       req.PhoneNumber,
		Country:           req.Country,
		BudgetMin:         req.BudgetMin,
		BudgetMax:         req.BudgetMax,
		Currency:          req.Currency,
		PreferredProperty: req.PreferredProperty,
		Source:            req.Source,
		Status:            crm.LeadStatusNew,
		Notes:             req.Message,
	}
	if lead.Currency == "" {
		lead.Currency = "USD"
	}
	if lead.Source == "" {
		lead.Source = "WEBSITE"
	}

	err := h.db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&lead).Error; err != nil {
			return err
		}

		payload, _ := json.Marshal(map[string]interface{}{
			"lead_id":      lead.ID,
			"full_name":    lead.FullName,
			"email":        lead.Email,
			"phone_number": lead.PhoneNumber,
			"message":      req.Message,
		})

		outboxEntry := outbox.OutboxEvent{
			EventType:     "LEAD_CREATED",
			AggregateType: "LEAD",
			AggregateID:   lead.ID.String(),
			Payload:       string(payload),
			Status:        outbox.StatusPending,
		}

		return tx.Create(&outboxEntry).Error
	})

	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to record inquiry", "TRANSACTION_FAILED", nil)
	}

	return response.Success(c, fiber.StatusCreated, "Inquiry submitted successfully. A Dexta Africa advisor will reach out shortly.", lead, nil)
}

type BookInspectionRequest struct {
	FullName      string             `json:"full_name"`
	Email         string             `json:"email"`
	PhoneNumber   string             `json:"phone_number"`
	DevelopmentID *uuid.UUID         `json:"development_id,omitempty"`
	UnitID        *uuid.UUID         `json:"unit_id,omitempty"`
	Type          crm.InspectionType `json:"type"`
	ScheduledDate string             `json:"scheduled_date"` // YYYY-MM-DD
	TimeSlot      string             `json:"time_slot"`      // e.g. "10:00 AM - 11:30 AM"
	Notes         string             `json:"notes,omitempty"`
}

// BookInspection schedules physical or virtual video tours.
func (h *CRMHandler) BookInspection(c *fiber.Ctx) error {
	var req BookInspectionRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", "INVALID_INPUT", nil)
	}

	if req.FullName == "" || req.Email == "" || req.PhoneNumber == "" || req.ScheduledDate == "" || req.TimeSlot == "" {
		return response.Error(c, fiber.StatusUnprocessableEntity, "All inspection booking fields are required", "VALIDATION_ERROR", nil)
	}

	scheduledTime, err := time.Parse("2006-01-02", req.ScheduledDate)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid scheduled_date format. Expected YYYY-MM-DD", "INVALID_DATE_FORMAT", nil)
	}

	var inspection crm.Inspection
	txErr := h.db.DB.Transaction(func(tx *gorm.DB) error {
		// Find or create lead
		var lead crm.Lead
		err := tx.Where("email = ?", req.Email).First(&lead).Error
		if err != nil {
			lead = crm.Lead{
				FullName:    req.FullName,
				Email:       req.Email,
				PhoneNumber: req.PhoneNumber,
				Status:      crm.LeadStatusInspectionScheduled,
			}
			if err := tx.Create(&lead).Error; err != nil {
				return err
			}
		}

		inspection = crm.Inspection{
			LeadID:        lead.ID,
			DevelopmentID: req.DevelopmentID,
			UnitID:        req.UnitID,
			Type:          req.Type,
			ScheduledAt:   scheduledTime,
			TimeSlot:      req.TimeSlot,
			Status:        crm.InspectionStatusConfirmed,
			Feedback:      req.Notes,
		}
		if inspection.Type == "" {
			inspection.Type = crm.InspectionPhysical
		}

		if err := tx.Create(&inspection).Error; err != nil {
			return err
		}

		payload, _ := json.Marshal(map[string]interface{}{
			"inspection_id":  inspection.ID,
			"lead_name":      lead.FullName,
			"lead_email":     lead.Email,
			"lead_phone":     lead.PhoneNumber,
			"type":           inspection.Type,
			"scheduled_date": req.ScheduledDate,
			"time_slot":      inspection.TimeSlot,
		})

		outboxEntry := outbox.OutboxEvent{
			EventType:     "INSPECTION_BOOKED",
			AggregateType: "INSPECTION",
			AggregateID:   inspection.ID.String(),
			Payload:       string(payload),
			Status:        outbox.StatusPending,
		}

		return tx.Create(&outboxEntry).Error
	})

	if txErr != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to book inspection", "TRANSACTION_FAILED", nil)
	}

	return response.Success(c, fiber.StatusCreated, "Inspection scheduled successfully. Calendar invite and confirmation email queued.", inspection, nil)
}

// ListLeads retrieves leads for internal admin dashboard.
func (h *CRMHandler) ListLeads(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	query := h.db.DB.Model(&crm.Lead{})
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	var totalItems int64
	query.Count(&totalItems)

	var leads []crm.Lead
	if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&leads).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch leads", "DATABASE_ERROR", nil)
	}

	totalPages := int((totalItems + int64(limit) - 1) / int64(limit))
	pagination := &response.PaginationMeta{
		Page:       page,
		Limit:      limit,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}

	return response.Success(c, fiber.StatusOK, "Leads retrieved successfully", leads, pagination)
}

