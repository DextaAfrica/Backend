package domain

import (
	"context"
	"time"
)

type Article struct {
	ID             string
	Slug           string
	Title          string
	Excerpt        string
	Body           map[string]any
	CoverImageURL  string
	Author         string
	Category       string
	SEOTitle       string
	SEODescription string
	Published      bool
	PublishedAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ArticleRepository interface {
	Create(ctx context.Context, a *Article) error
	Update(ctx context.Context, a *Article) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*Article, error)
	GetBySlug(ctx context.Context, slug string) (*Article, error)
	List(ctx context.Context, params ListParams) (PageResult[Article], error)
	SlugExists(ctx context.Context, slug string, excludeID string) (bool, error)
}
