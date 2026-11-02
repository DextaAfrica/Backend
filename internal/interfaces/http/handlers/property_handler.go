package handlers

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/DextaAfrica/Backend/internal/domain/outbox"
	"github.com/DextaAfrica/Backend/internal/domain/property"
	"github.com/DextaAfrica/Backend/internal/infrastructure/database"
	"github.com/DextaAfrica/Backend/internal/interfaces/http/response"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PropertyHandler struct {
	db *database.PostgresDB
}

func NewPropertyHandler(db *database.PostgresDB) *PropertyHandler {
	return &PropertyHandler{db: db}
}

// ListDevelopments returns developments with filtering and pagination.
func (h *PropertyHandler) ListDevelopments(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	query := h.db.DB.Model(&property.Development{})

	if country := c.Query("country"); country != "" {
		query = query.Where("country = ?", country)
	}
	if city := c.Query("city"); city != "" {
		query = query.Where("city = ?", city)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if featured := c.Query("featured"); featured != "" {
		query = query.Where("is_featured = ?", featured == "true")
	}

	var totalItems int64
	if err := query.Count(&totalItems).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to count developments", "DATABASE_ERROR", nil)
	}

	var developments []property.Development
	err := query.
		Preload("MediaAssets").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&developments).Error

	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch developments", "DATABASE_ERROR", nil)
	}

	totalPages := int((totalItems + int64(limit) - 1) / int64(limit))
	pagination := &response.PaginationMeta{
		Page:       page,
		Limit:      limit,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}

	return response.Success(c, fiber.StatusOK, "Developments retrieved successfully", developments, pagination)
}

// GetDevelopment retrieves a single development by slug.
func (h *PropertyHandler) GetDevelopment(c *fiber.Ctx) error {
	slug := c.Params("slug")

	var dev property.Development
	err := h.db.DB.
		Preload("Units").
		Preload("MediaAssets").
		Preload("Milestones").
		Where("slug = ?", slug).
		First(&dev).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.Error(c, fiber.StatusNotFound, "Development not found", "NOT_FOUND", nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch development", "DATABASE_ERROR", nil)
	}

	return response.Success(c, fiber.StatusOK, "Development retrieved successfully", dev, nil)
}

// ListUnits returns units matching search filters.
func (h *PropertyHandler) ListUnits(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	query := h.db.DB.Model(&property.Unit{})

	if devID := c.Query("development_id"); devID != "" {
		query = query.Where("development_id = ?", devID)
	}
	if unitType := c.Query("type"); unitType != "" {
		query = query.Where("type = ?", unitType)
	}
	if bedrooms := c.Query("bedrooms"); bedrooms != "" {
		query = query.Where("bedrooms >= ?", bedrooms)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	} else {
		// Default to available units for public listings
		query = query.Where("status = ?", property.UnitStatusAvailable)
	}
	if minPrice := c.Query("min_price"); minPrice != "" {
		query = query.Where("price >= ?", minPrice)
	}
	if maxPrice := c.Query("max_price"); maxPrice != "" {
		query = query.Where("price <= ?", maxPrice)
	}

	var totalItems int64
	if err := query.Count(&totalItems).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to count units", "DATABASE_ERROR", nil)
	}

	var units []property.Unit
	err := query.
		Preload("MediaAssets").
		Order("price ASC").
		Limit(limit).
		Offset(offset).
		Find(&units).Error

	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch units", "DATABASE_ERROR", nil)
	}

	totalPages := int((totalItems + int64(limit) - 1) / int64(limit))
	pagination := &response.PaginationMeta{
		Page:       page,
		Limit:      limit,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}

	return response.Success(c, fiber.StatusOK, "Units retrieved successfully", units, pagination)
}

// GetUnit retrieves unit by UUID.
func (h *PropertyHandler) GetUnit(c *fiber.Ctx) error {
	idStr := c.Params("id")
	unitID, err := uuid.Parse(idStr)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid unit UUID format", "INVALID_INPUT", nil)
	}

	var unit property.Unit
	err = h.db.DB.Preload("MediaAssets").Where("id = ?", unitID).First(&unit).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.Error(c, fiber.StatusNotFound, "Unit not found", "NOT_FOUND", nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch unit", "DATABASE_ERROR", nil)
	}

	return response.Success(c, fiber.StatusOK, "Unit retrieved successfully", unit, nil)
}

type ReserveUnitRequest struct {
	ClientName  string `json:"client_name"`
	ClientEmail string `json:"client_email"`
	ClientPhone string `json:"client_phone"`
	Notes       string `json:"notes"`
}

// ReserveUnit atomically reserves a unit with pessimistic locking and writes to Transactional Outbox.
func (h *PropertyHandler) ReserveUnit(c *fiber.Ctx) error {
	idStr := c.Params("id")
	unitID, err := uuid.Parse(idStr)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid unit UUID format", "INVALID_INPUT", nil)
	}

	var req ReserveUnitRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", "INVALID_INPUT", nil)
	}

	if req.ClientEmail == "" || req.ClientName == "" {
		return response.Error(c, fiber.StatusUnprocessableEntity, "Client name and email are required", "VALIDATION_ERROR", []response.ErrorDetail{
			{Field: "client_name", Issue: "Name is required"},
			{Field: "client_email", Issue: "Valid email is required"},
		})
	}

	// Begin atomic database transaction with pessimistic locking
	var reservedUnit property.Unit
	txErr := h.db.DB.Transaction(func(tx *gorm.DB) error {
		// SELECT FOR UPDATE locks the row against simultaneous concurrent reservations
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", unitID).First(&reservedUnit).Error; err != nil {
			return err
		}

		if reservedUnit.Status != property.UnitStatusAvailable {
			return fmt.Errorf("unit_not_available")
		}

		// Update unit status to RESERVED with 48h expiration window
		reserveUntil := time.Now().Add(48 * time.Hour)
		reservedUnit.Status = property.UnitStatusReserved
		reservedUnit.ReservedUntil = &reserveUntil

		if err := tx.Save(&reservedUnit).Error; err != nil {
			return err
		}

		// Atomically record event in Outbox table
		eventPayload, _ := json.Marshal(map[string]interface{}{
			"unit_id":         reservedUnit.ID,
			"unit_number":     reservedUnit.UnitNumber,
			"development_id":  reservedUnit.DevelopmentID,
			"client_name":     req.ClientName,
			"client_email":    req.ClientEmail,
			"client_phone":    req.ClientPhone,
			"price":           reservedUnit.Price,
			"currency":        reservedUnit.Currency,
			"reserved_until":  reserveUntil,
		})

		outboxEntry := outbox.OutboxEvent{
			EventType:     "UNIT_RESERVED",
			AggregateType: "UNIT",
			AggregateID:   reservedUnit.ID.String(),
			Payload:       string(eventPayload),
			Status:        outbox.StatusPending,
		}

		if err := tx.Create(&outboxEntry).Error; err != nil {
			return err
		}

		return nil
	})

	if txErr != nil {
		if txErr.Error() == "unit_not_available" {
			return response.Error(c, fiber.StatusConflict, "Unit is no longer available for reservation", "UNIT_UNAVAILABLE", nil)
		}
		if txErr == gorm.ErrRecordNotFound {
			return response.Error(c, fiber.StatusNotFound, "Unit not found", "NOT_FOUND", nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, "Failed to reserve unit", "TRANSACTION_FAILED", nil)
	}

	return response.Success(c, fiber.StatusCreated, "Unit successfully reserved. Reservation confirmation email queued.", reservedUnit, nil)
}

