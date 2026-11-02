package finance

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TransactionStatus string

const (
	TxStatusInitiated  TransactionStatus = "INITIATED"
	TxStatusSuccessful TransactionStatus = "SUCCESSFUL"
	TxStatusFailed     TransactionStatus = "FAILED"
	TxStatusRefunded   TransactionStatus = "REFUNDED"
)

type InvoiceStatus string

const (
	InvoiceStatusDraft   InvoiceStatus = "DRAFT"
	InvoiceStatusIssued  InvoiceStatus = "ISSUED"
	InvoiceStatusPaid    InvoiceStatus = "PAID"
	InvoiceStatusOverdue InvoiceStatus = "OVERDUE"
	InvoiceStatusVoid    InvoiceStatus = "VOID"
)

// Transaction tracks financial payments, escrow deposits, and reservations.
type Transaction struct {
	ID             uuid.UUID         `gorm:"type:uuid;primaryKey" json:"id"`
	Reference      string            `gorm:"type:varchar(100);uniqueIndex;not null" json:"reference"`
	IdempotencyKey string            `gorm:"type:varchar(255);uniqueIndex" json:"idempotency_key,omitempty"`
	ClientID       uuid.UUID         `gorm:"type:uuid;index;not null" json:"client_id"`
	UnitID         *uuid.UUID        `gorm:"type:uuid;index" json:"unit_id,omitempty"`
	Amount         float64           `gorm:"type:decimal(15,2);not null" json:"amount"`
	Currency       string            `gorm:"type:varchar(10);default:'USD';not null" json:"currency"`
	PaymentMethod  string            `gorm:"type:varchar(50);not null" json:"payment_method"`
	Gateway        string            `gorm:"type:varchar(50);not null" json:"gateway"`
	GatewayRef     string            `gorm:"type:varchar(255);index" json:"gateway_ref,omitempty"`
	Status         TransactionStatus `gorm:"type:varchar(50);default:'INITIATED';index" json:"status"`
	Description    string            `gorm:"type:text" json:"description"`
	PaidAt         *time.Time        `json:"paid_at,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

func (t *Transaction) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

// Invoice tracks milestone contracts, reservation invoices, and escrow releases.
type Invoice struct {
	ID            uuid.UUID     `gorm:"type:uuid;primaryKey" json:"id"`
	InvoiceNumber string        `gorm:"type:varchar(100);uniqueIndex;not null" json:"invoice_number"`
	ClientID      uuid.UUID     `gorm:"type:uuid;index;not null" json:"client_id"`
	UnitID        *uuid.UUID    `gorm:"type:uuid;index" json:"unit_id,omitempty"`
	Title         string        `gorm:"type:varchar(255);not null" json:"title"`
	TotalAmount   float64       `gorm:"type:decimal(15,2);not null" json:"total_amount"`
	Currency      string        `gorm:"type:varchar(10);default:'USD';not null" json:"currency"`
	Status        InvoiceStatus `gorm:"type:varchar(50);default:'ISSUED';index" json:"status"`
	DueDate       time.Time     `gorm:"not null" json:"due_date"`
	PaidAt        *time.Time    `json:"paid_at,omitempty"`
	PdfURL        string        `gorm:"type:text" json:"pdf_url,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

func (inv *Invoice) BeforeCreate(tx *gorm.DB) error {
	if inv.ID == uuid.Nil {
		inv.ID = uuid.New()
	}
	return nil
}

