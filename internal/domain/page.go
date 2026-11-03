package domain

import (
	"context"
	"time"
)

// Page is a flexible, JSON-bodied content document for every editorial page
// that isn't a repeatable collection: home, about, lifestyle, careers
// landing, and the static legal pages. Content shape is owned by the
// frontend contract for that key, not by this backend — the API stores and
// serves it opaquely.
type Page struct {
	ID             string
	Key            string
	Title          string
	Content        map[string]any
	SEOTitle       string
	SEODescription string
	Published      bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type PageRepository interface {
	GetByKey(ctx context.Context, key string) (*Page, error)
	Upsert(ctx context.Context, page *Page) (*Page, error)
	List(ctx context.Context) ([]Page, error)
}
