package request

type SaveArticleRequest struct {
	Slug           string         `json:"slug" validate:"required,slug,max=120"`
	Title          string         `json:"title" validate:"required,max=200"`
	Excerpt        string         `json:"excerpt" validate:"max=500"`
	Body           map[string]any `json:"body" validate:"required"`
	CoverImageURL  string         `json:"coverImageUrl" validate:"omitempty,url"`
	Author         string         `json:"author" validate:"max=120"`
	Category       string         `json:"category" validate:"max=120"`
	SEOTitle       string         `json:"seoTitle" validate:"max=200"`
	SEODescription string         `json:"seoDescription" validate:"max=500"`
	Published      bool           `json:"published"`
}
