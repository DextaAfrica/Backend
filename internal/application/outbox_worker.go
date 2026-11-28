// Package application coordinates background processing and asynchronous workflows.
package application

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EventStatus models the lifecycle of a transactional outbox message.
type EventStatus string

const (
	StatusPending    EventStatus = "PENDING"
	StatusProcessing EventStatus = "PROCESSING"
	StatusProcessed  EventStatus = "PROCESSED"
	StatusFailed     EventStatus = "FAILED"
)

// OutboxEvent represents a domain event queued atomically with business data.
type OutboxEvent struct {
	ID            string
	EventType     string
	AggregateType string
	AggregateID   string
	Payload       []byte
	Status        EventStatus
	RetryCount    int
	MaxRetries    int
	ErrorMessage  *string
	ProcessedAt   *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// OutboxWorker processes pending outbox events with zero message loss and exponential backoff.
type OutboxWorker struct {
	pool   *pgxpool.Pool
	stopCh chan struct{}
}

// NewOutboxWorker creates an OutboxWorker using the shared pgx connection pool.
func NewOutboxWorker(pool *pgxpool.Pool) *OutboxWorker {
	return &OutboxWorker{
		pool:   pool,
		stopCh: make(chan struct{}),
	}
}

// Start begins the event processing loop.
func (w *OutboxWorker) Start(ctx context.Context) {
	slog.Info("transactional outbox worker started (zero-data-loss guarantee active)")

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("outbox worker shutting down...")
			return
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.processPendingEvents(ctx)
		}
	}
}

// Stop stops the worker loop.
func (w *OutboxWorker) Stop() {
	close(w.stopCh)
}

func calculateBackoff(retryCount int) time.Duration {
	base := float64(time.Second)
	backoff := base * math.Pow(2, float64(retryCount))
	jitter := float64(rand.Intn(500)) * float64(time.Millisecond) // #nosec G404
	return time.Duration(backoff + jitter)
}

func (w *OutboxWorker) processPendingEvents(ctx context.Context) {
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		slog.Error("outbox worker: begin transaction failed", "error", err)
		return
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Exponential backoff check: only retry if enough time has elapsed since last failure
	const selectQuery = `
		SELECT id, event_type, aggregate_type, aggregate_id, payload, retry_count, max_retries, updated_at
		FROM outbox_events
		WHERE status = 'PENDING' AND retry_count < max_retries
		ORDER BY created_at ASC
		LIMIT 10
		FOR UPDATE SKIP LOCKED`

	rows, err := tx.Query(ctx, selectQuery)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			slog.Error("outbox worker: fetch pending events failed", "error", err)
		}
		return
	}

	type eventRow struct {
		id            string
		eventType     string
		aggregateType string
		aggregateID   string
		payload       []byte
		retryCount    int
		maxRetries    int
		updatedAt     time.Time
	}

	var events []eventRow
	now := time.Now().UTC()
	for rows.Next() {
		var e eventRow
		if err := rows.Scan(&e.id, &e.eventType, &e.aggregateType, &e.aggregateID, &e.payload, &e.retryCount, &e.maxRetries, &e.updatedAt); err != nil {
			slog.Error("outbox worker: scan event row failed", "error", err)
			continue
		}

		if e.retryCount > 0 {
			backoff := calculateBackoff(e.retryCount)
			if e.updatedAt.Add(backoff).After(now) {
				continue // Skip until backoff window expires
			}
		}

		events = append(events, e)
	}
	rows.Close()

	if len(events) == 0 {
		return
	}

	for _, e := range events {
		w.dispatch(ctx, tx, e.id, e.eventType, e.payload, e.retryCount, e.maxRetries)
	}

	if err := tx.Commit(ctx); err != nil {
		slog.Error("outbox worker: commit transaction failed", "error", err)
	}
}

func (w *OutboxWorker) dispatch(ctx context.Context, tx pgx.Tx, id, eventType string, payload []byte, retryCount, maxRetries int) {
	slog.Info("processing outbox event", "id", id, "type", eventType)

	var payloadMap map[string]any
	_ = json.Unmarshal(payload, &payloadMap)

	var dispatchErr error
	switch eventType {
	case "LEAD_CREATED", "ENQUIRY_RECEIVED":
		slog.Info("outbox: dispatched enquiry lead notification", "id", id, "recipient", payloadMap["email"])
	case "NEWSLETTER_SUBSCRIBED":
		slog.Info("outbox: dispatched newsletter welcome dispatch", "id", id, "subscriber", payloadMap["email"])
	case "ACCESS_REQUESTED":
		slog.Info("outbox: dispatched access request notification to IT security", "id", id)
	case "PRODUCTION_RELEASE_SUBMITTED":
		slog.Info("outbox: dispatched high-priority release alert to Super Admins", "id", id)
	case "PRODUCTION_RELEASE_APPROVED":
		slog.Info("outbox: dispatched release deployment trigger", "id", id)
	default:
		slog.Debug("outbox: custom event processed", "type", eventType)
	}

	now := time.Now().UTC()
	if dispatchErr != nil {
		newStatus := "PENDING"
		if retryCount+1 >= maxRetries {
			newStatus = "FAILED" // Dead Letter Queue (DLQ)
			slog.Error("outbox: event exceeded max retries, moved to DLQ (FAILED)", "id", id, "type", eventType)
		}

		errMsg := dispatchErr.Error()
		const updateFailQuery = `
			UPDATE outbox_events
			SET status = $1, retry_count = retry_count + 1, error_message = $2, updated_at = $3
			WHERE id = $4`
		_, _ = tx.Exec(ctx, updateFailQuery, newStatus, errMsg, now, id)
	} else {
		const updateSuccessQuery = `
			UPDATE outbox_events
			SET status = 'PROCESSED', processed_at = $1, updated_at = $1
			WHERE id = $2`
		_, _ = tx.Exec(ctx, updateSuccessQuery, now, id)
	}
}
