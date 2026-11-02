package request

type SavePageRequest struct {
	Title          string         `json:"title" validate:"required,max=200"`
	Content        map[string]any `json:"content" validate:"required"`
	SEOTitle       string         `json:"seoTitle" validate:"max=200"`
	SEODescription string         `json:"seoDescription" validate:"max=500"`
	Published      bool           `json:"published"`
}
