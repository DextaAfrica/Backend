package outbox

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EventStatus string

const (
	StatusPending    EventStatus = "PENDING"
	StatusProcessing EventStatus = "PROCESSING"
	StatusProcessed  EventStatus = "PROCESSED"
	StatusFailed     EventStatus = "FAILED"
)

// OutboxEvent records events atomically with domain operations to guarantee zero message loss.
type OutboxEvent struct {
	ID            uuid.UUID   `gorm:"type:uuid;primaryKey" json:"id"`
	EventType     string      `gorm:"type:varchar(100);not null;index" json:"event_type"`
	AggregateType string      `gorm:"type:varchar(100);not null;index" json:"aggregate_type"`
	AggregateID   string      `gorm:"type:varchar(255);not null;index" json:"aggregate_id"`
	Payload       string      `gorm:"type:jsonb;not null" json:"payload"`
	Status        EventStatus `gorm:"type:varchar(50);default:'PENDING';index;not null" json:"status"`
	RetryCount    int         `gorm:"default:0" json:"retry_count"`
	MaxRetries    int         `gorm:"default:5" json:"max_retries"`
	ErrorMessage  string      `gorm:"type:text" json:"error_message,omitempty"`
	ProcessedAt   *time.Time  `json:"processed_at,omitempty"`
	CreatedAt     time.Time   `gorm:"index" json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

func (o *OutboxEvent) BeforeCreate(tx *gorm.DB) error {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	return nil
}

