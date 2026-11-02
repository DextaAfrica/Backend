package property

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DevelopmentStatus string

const (
	DevStatusPlanning          DevelopmentStatus = "PLANNING"
	DevStatusUnderConstruction DevelopmentStatus = "UNDER_CONSTRUCTION"
	DevStatusCompleted         DevelopmentStatus = "COMPLETED"
)

type UnitStatus string

const (
	UnitStatusAvailable     UnitStatus = "AVAILABLE"
	UnitStatusReserved      UnitStatus = "RESERVED"
	UnitStatusUnderContract UnitStatus = "UNDER_CONTRACT"
	UnitStatusSold          UnitStatus = "SOLD"
)

type UnitType string

const (
	UnitTypeApartment UnitType = "APARTMENT"
	UnitTypeVilla     UnitType = "VILLA"
	UnitTypeDuplex    UnitType = "DUPLEX"
	UnitTypePenthouse UnitType = "PENTHOUSE"
	UnitTypeMansion   UnitType = "MANSION"
	UnitTypeLandLot   UnitType = "LAND_LOT"
)

type MediaType string

const (
	MediaTypeImage       MediaType = "IMAGE"
	MediaTypeVideo       MediaType = "VIDEO"
	MediaTypeFloorPlan   MediaType = "FLOOR_PLAN"
	MediaTypeVirtualTour MediaType = "VIRTUAL_TOUR_360"
	MediaTypeBrochure    MediaType = "BROCHURE"
)

// Development represents a full estate or residential project.
type Development struct {
	ID             uuid.UUID         `gorm:"type:uuid;primaryKey" json:"id"`
	Name           string            `gorm:"type:varchar(255);not null;index" json:"name"`
	Slug           string            `gorm:"type:varchar(255);uniqueIndex;not null" json:"slug"`
	Tagline        string            `gorm:"type:varchar(255)" json:"tagline"`
	Description    string            `gorm:"type:text;not null" json:"description"`
	Country        string            `gorm:"type:varchar(100);not null;index" json:"country"`
	StateProvince  string            `gorm:"type:varchar(100);not null;index" json:"state_province"`
	City           string            `gorm:"type:varchar(100);not null;index" json:"city"`
	Address        string            `gorm:"type:text;not null" json:"address"`
	Latitude       float64           `gorm:"type:decimal(10,8)" json:"latitude"`
	Longitude      float64           `gorm:"type:decimal(11,8)" json:"longitude"`
	Status         DevelopmentStatus `gorm:"type:varchar(50);not null;default:'PLANNING';index" json:"status"`
	TotalUnits     int               `gorm:"default:0" json:"total_units"`
	AvailableUnits int               `gorm:"default:0" json:"available_units"`
	StartingPrice  float64           `gorm:"type:decimal(15,2);index" json:"starting_price"`
	Currency       string            `gorm:"type:varchar(10);default:'USD'" json:"currency"`
	IsFeatured     bool              `gorm:"default:false;index" json:"is_featured"`
	CompletionDate *time.Time        `json:"completion_date"`
	Units          []Unit            `gorm:"foreignKey:DevelopmentID" json:"units,omitempty"`
	MediaAssets    []MediaAsset      `gorm:"polymorphic:Entity;polymorphicValue:development" json:"media_assets,omitempty"`
	Milestones     []Milestone       `gorm:"foreignKey:DevelopmentID" json:"milestones,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	DeletedAt      gorm.DeletedAt    `gorm:"index" json:"-"`
}

func (d *Development) BeforeCreate(tx *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return nil
}

// Unit represents an individual home, apartment, or plot in a development.
type Unit struct {
	ID                   uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	DevelopmentID        uuid.UUID      `gorm:"type:uuid;not null;index" json:"development_id"`
	UnitNumber           string         `gorm:"type:varchar(100);not null" json:"unit_number"`
	Title                string         `gorm:"type:varchar(255);not null" json:"title"`
	Type                 UnitType       `gorm:"type:varchar(50);not null;index" json:"type"`
	Bedrooms             int            `gorm:"default:1;index" json:"bedrooms"`
	Bathrooms            float32        `gorm:"type:decimal(3,1);default:1" json:"bathrooms"`
	FloorAreaSqm         float64        `gorm:"type:decimal(10,2);not null" json:"floor_area_sqm"`
	Price                float64        `gorm:"type:decimal(15,2);not null;index" json:"price"`
	Currency             string         `gorm:"type:varchar(10);default:'USD'" json:"currency"`
	Status               UnitStatus     `gorm:"type:varchar(50);not null;default:'AVAILABLE';index" json:"status"`
	ReservationDeposit   float64        `gorm:"type:decimal(15,2);default:0" json:"reservation_deposit"`
	ReservedUntil        *time.Time     `json:"reserved_until,omitempty"`
	ReservedByClientID   *uuid.UUID     `gorm:"type:uuid;index" json:"reserved_by_client_id,omitempty"`
	FloorNumber          int            `gorm:"default:1" json:"floor_number"`
	Features             string         `gorm:"type:jsonb" json:"features,omitempty"`
	MediaAssets          []MediaAsset   `gorm:"polymorphic:Entity;polymorphicValue:unit" json:"media_assets,omitempty"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
	DeletedAt            gorm.DeletedAt `gorm:"index" json:"-"`
}

func (u *Unit) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

// MediaAsset stores polymorphic photos, videos, 360-tours, and brochures.
type MediaAsset struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	EntityID     uuid.UUID `gorm:"type:uuid;index;not null" json:"entity_id"`
	EntityType   string    `gorm:"type:varchar(50);index;not null" json:"entity_type"`
	MediaType    MediaType `gorm:"type:varchar(50);not null" json:"media_type"`
	URL          string    `gorm:"type:text;not null" json:"url"`
	ThumbnailURL string    `gorm:"type:text" json:"thumbnail_url"`
	Caption      string    `gorm:"type:varchar(255)" json:"caption"`
	SortOrder    int       `gorm:"default:0" json:"sort_order"`
	CreatedAt    time.Time `json:"created_at"`
}

func (m *MediaAsset) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

// Milestone tracks off-plan construction progress for investor transparency.
type Milestone struct {
	ID                  uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	DevelopmentID       uuid.UUID  `gorm:"type:uuid;not null;index" json:"development_id"`
	Title               string     `gorm:"type:varchar(255);not null" json:"title"`
	Description         string     `gorm:"type:text" json:"description"`
	PercentageComplete  int        `gorm:"default:0" json:"percentage_complete"`
	TargetDate          *time.Time `json:"target_date"`
	CompletedDate       *time.Time `json:"completed_date,omitempty"`
	ProofImageURL       string     `gorm:"type:text" json:"proof_image_url"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

func (m *Milestone) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

