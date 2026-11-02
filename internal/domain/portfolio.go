package domain

import (
	"context"
	"time"
)

type DevelopmentStatus string

const (
	DevelopmentStatusPlanning           DevelopmentStatus = "planning"
	DevelopmentStatusUnderConstruction  DevelopmentStatus = "under_construction"
	DevelopmentStatusCompleted          DevelopmentStatus = "completed"
)

func (s DevelopmentStatus) Valid() bool {
	switch s {
	case DevelopmentStatusPlanning, DevelopmentStatusUnderConstruction, DevelopmentStatusCompleted:
		return true
	default:
		return false
	}
}

type GalleryImage struct {
	URL string `json:"url"`
	Alt string `json:"alt"`
}

type Development struct {
	ID             string
	Slug           string
	Name           string
	Summary        string
	Body           map[string]any
	HeroImageURL   string
	Gallery        []GalleryImage
	Location       string
	Status         DevelopmentStatus
	Featured       bool
	SEOTitle       string
	SEODescription string
	Published      bool
	PublishedAt    *time.Time
	DisplayOrder   int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type DevelopmentRepository interface {
	Create(ctx context.Context, d *Development) error
	Update(ctx context.Context, d *Development) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*Development, error)
	GetBySlug(ctx context.Context, slug string) (*Development, error)
	List(ctx context.Context, params ListParams) (PageResult[Development], error)
	SlugExists(ctx context.Context, slug string, excludeID string) (bool, error)
}
