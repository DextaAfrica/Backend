package crm

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LeadStatus string

const (
	LeadStatusNew                 LeadStatus = "NEW"
	LeadStatusContacted           LeadStatus = "CONTACTED"
	LeadStatusQualified           LeadStatus = "QUALIFIED"
	LeadStatusInspectionScheduled LeadStatus = "INSPECTION_SCHEDULED"
	LeadStatusNegotiating         LeadStatus = "NEGOTIATING"
	LeadStatusConverted           LeadStatus = "CONVERTED"
	LeadStatusLost                LeadStatus = "LOST"
)

type InspectionStatus string

const (
	InspectionStatusPending   InspectionStatus = "PENDING"
	InspectionStatusConfirmed InspectionStatus = "CONFIRMED"
	InspectionStatusCompleted InspectionStatus = "COMPLETED"
	InspectionStatusCancelled InspectionStatus = "CANCELLED"
	InspectionStatusNoShow    InspectionStatus = "NO_SHOW"
)

type InspectionType string

const (
	InspectionPhysical InspectionType = "PHYSICAL"
	InspectionVirtual  InspectionType = "VIRTUAL_VIDEO"
)

// Lead represents any prospective buyer or investor inquiring via Web or Admin.
type Lead struct {
	ID                uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	FullName          string         `gorm:"type:varchar(255);not null" json:"full_name"`
	Email             string         `gorm:"type:varchar(255);index;not null" json:"email"`
	PhoneNumber       string         `gorm:"type:varchar(50);index;not null" json:"phone_number"`
	Country           string         `gorm:"type:varchar(100);index" json:"country"`
	BudgetMin         float64        `gorm:"type:decimal(15,2)" json:"budget_min"`
	BudgetMax         float64        `gorm:"type:decimal(15,2)" json:"budget_max"`
	Currency          string         `gorm:"type:varchar(10);default:'USD'" json:"currency"`
	PreferredProperty string         `gorm:"type:varchar(255)" json:"preferred_property"`
	Source            string         `gorm:"type:varchar(100);default:'WEBSITE'" json:"source"`
	Status            LeadStatus     `gorm:"type:varchar(50);default:'NEW';index" json:"status"`
	AssignedToAgentID *uuid.UUID     `gorm:"type:uuid;index" json:"assigned_to_agent_id,omitempty"`
	Notes             string         `gorm:"type:text" json:"notes,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

func (l *Lead) BeforeCreate(tx *gorm.DB) error {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	return nil
}

// Inspection coordinates physical and virtual site visits with auto-email confirmations.
type Inspection struct {
	ID              uuid.UUID        `gorm:"type:uuid;primaryKey" json:"id"`
	LeadID          uuid.UUID        `gorm:"type:uuid;not null;index" json:"lead_id"`
	DevelopmentID   *uuid.UUID       `gorm:"type:uuid;index" json:"development_id,omitempty"`
	UnitID          *uuid.UUID       `gorm:"type:uuid;index" json:"unit_id,omitempty"`
	Type            InspectionType   `gorm:"type:varchar(50);not null" json:"type"`
	ScheduledAt     time.Time        `gorm:"not null;index" json:"scheduled_at"`
	TimeSlot        string           `gorm:"type:varchar(50);not null" json:"time_slot"`
	Status          InspectionStatus `gorm:"type:varchar(50);default:'PENDING';index" json:"status"`
	MeetingLink     string           `gorm:"type:text" json:"meeting_link,omitempty"`
	AssignedAgentID *uuid.UUID       `gorm:"type:uuid;index" json:"assigned_agent_id,omitempty"`
	Feedback        string           `gorm:"type:text" json:"feedback,omitempty"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

func (i *Inspection) BeforeCreate(tx *gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	return nil
}

