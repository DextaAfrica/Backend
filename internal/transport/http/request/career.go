package request

type SaveCareerListingRequest struct {
	Slug           string         `json:"slug" validate:"required,slug,max=120"`
	Title          string         `json:"title" validate:"required,max=200"`
	Department     string         `json:"department" validate:"max=120"`
	Location       string         `json:"location" validate:"max=200"`
	EmploymentType string         `json:"employmentType" validate:"required,oneof=full_time part_time contract internship"`
	Description    map[string]any `json:"description" validate:"required"`
	Published      bool           `json:"published"`
}
