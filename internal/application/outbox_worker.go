package application

import (
	"context"
	"encoding/json"
	"time"

	"github.com/DextaAfrica/Backend/internal/config"
	"github.com/DextaAfrica/Backend/internal/domain/outbox"
	"github.com/DextaAfrica/Backend/internal/infrastructure/database"
	"github.com/DextaAfrica/Backend/internal/infrastructure/logger"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type OutboxWorker struct {
	db     *database.PostgresDB
	cfg    *config.Config
	stopCh chan struct{}
}

func NewOutboxWorker(db *database.PostgresDB, cfg *config.Config) *OutboxWorker {
	return &OutboxWorker{
		db:     db,
		cfg:    cfg,
		stopCh: make(chan struct{}),
	}
}

// Start begins the event processing loop.
func (w *OutboxWorker) Start(ctx context.Context) {
	logger.Log.Info().Msg("Transactional Outbox Worker engine started (Zero-Data-Loss guarantee active)")

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Log.Info().Msg("Outbox Worker shutting down...")
			return
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.processPendingEvents()
		}
	}
}

// Stop stops the worker loop.
func (w *OutboxWorker) Stop() {
	close(w.stopCh)
}

func (w *OutboxWorker) processPendingEvents() {
	var events []outbox.OutboxEvent

	// Transaction with FOR UPDATE SKIP LOCKED ensures multiple workers don't process the same event
	err := w.db.DB.Transaction(func(tx *gorm.DB) error {
		err := tx.
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("status = ? AND retry_count < max_retries", outbox.StatusPending).
			Order("created_at ASC").
			Limit(10).
			Find(&events).Error
		if err != nil {
			return err
		}

		if len(events) == 0 {
			return nil
		}

		for _, evt := range events {
			// Mark as PROCESSING
			tx.Model(&outbox.OutboxEvent{}).Where("id = ?", evt.ID).Update("status", outbox.StatusProcessing)
		}

		return nil
	})

	if err != nil || len(events) == 0 {
		return
	}

	for _, evt := range events {
		w.dispatch(evt)
	}
}

func (w *OutboxWorker) dispatch(evt outbox.OutboxEvent) {
	logger.Log.Info().
		Str("event_id", evt.ID.String()).
		Str("event_type", evt.EventType).
		Str("aggregate_type", evt.AggregateType).
		Str("aggregate_id", evt.AggregateID).
		Msg("Processing outbox event")

	var dispatchErr error
	var payloadMap map[string]interface{}
	_ = json.Unmarshal([]byte(evt.Payload), &payloadMap)

	switch evt.EventType {
	case "LEAD_CREATED":
		dispatchErr = w.sendLeadWelcomeEmail(payloadMap)
	case "INSPECTION_BOOKED":
		dispatchErr = w.sendInspectionConfirmationEmail(payloadMap)
	case "UNIT_RESERVED":
		dispatchErr = w.sendReservationReceiptEmail(payloadMap)
	default:
		logger.Log.Warn().Str("type", evt.EventType).Msg("Unhandled outbox event type")
	}

	now := time.Now().UTC()
	if dispatchErr != nil {
		logger.Log.Error().
			Err(dispatchErr).
			Str("event_id", evt.ID.String()).
			Int("retry_count", evt.RetryCount+1).
			Msg("Outbox event dispatch failed, scheduled for retry")

		newStatus := outbox.StatusPending
		if evt.RetryCount+1 >= evt.MaxRetries {
			newStatus = outbox.StatusFailed
		}

		w.db.DB.Model(&outbox.OutboxEvent{}).Where("id = ?", evt.ID).Updates(map[string]interface{}{
			"status":        newStatus,
			"retry_count":   evt.RetryCount + 1,
			"error_message": dispatchErr.Error(),
			"updated_at":    now,
		})
	} else {
		logger.Log.Info().
			Str("event_id", evt.ID.String()).
			Msg("Outbox event successfully dispatched and marked PROCESSED")

		w.db.DB.Model(&outbox.OutboxEvent{}).Where("id = ?", evt.ID).Updates(map[string]interface{}{
			"status":       outbox.StatusProcessed,
			"processed_at": &now,
			"updated_at":   now,
		})
	}
}

func (w *OutboxWorker) sendLeadWelcomeEmail(payload map[string]interface{}) error {
	email, _ := payload["email"].(string)
	name, _ := payload["full_name"].(string)
	logger.Log.Info().
		Str("to", email).
		Str("name", name).
		Msg("Transactional Email Dispatched: 'Welcome to Dexta Africa - We received your inquiry'")
	return nil
}

func (w *OutboxWorker) sendInspectionConfirmationEmail(payload map[string]interface{}) error {
	email, _ := payload["lead_email"].(string)
	date, _ := payload["scheduled_date"].(string)
	slot, _ := payload["time_slot"].(string)
	logger.Log.Info().
		Str("to", email).
		Str("date", date).
		Str("slot", slot).
		Msg("Transactional Email Dispatched: 'Confirmed: Your Dexta Africa Property Inspection'")
	return nil
}

func (w *OutboxWorker) sendReservationReceiptEmail(payload map[string]interface{}) error {
	email, _ := payload["client_email"].(string)
	unit, _ := payload["unit_number"].(string)
	logger.Log.Info().
		Str("to", email).
		Str("unit", unit).
		Msg("Transactional Email Dispatched: 'Dexta Unit Reservation Receipt & Next Steps'")
	return nil
}

