package request

type GalleryImageRequest struct {
	URL string `json:"url" validate:"required,url"`
	Alt string `json:"alt" validate:"max=200"`
}

type SaveDevelopmentRequest struct {
	Slug           string                `json:"slug" validate:"required,slug,max=120"`
	Name           string                `json:"name" validate:"required,max=200"`
	Summary        string                `json:"summary" validate:"max=500"`
	Body           map[string]any        `json:"body" validate:"required"`
	HeroImageURL   string                `json:"heroImageUrl" validate:"omitempty,url"`
	Gallery        []GalleryImageRequest `json:"gallery"`
	Location       string                `json:"location" validate:"max=200"`
	Status         string                `json:"status" validate:"required,oneof=planning under_construction completed"`
	Featured       bool                  `json:"featured"`
	SEOTitle       string                `json:"seoTitle" validate:"max=200"`
	SEODescription string                `json:"seoDescription" validate:"max=500"`
	Published      bool                  `json:"published"`
	DisplayOrder   int                   `json:"displayOrder"`
}
