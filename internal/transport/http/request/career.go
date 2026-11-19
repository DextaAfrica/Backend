package request

type SaveCareerListingRequest struct {
	Slug           string         `json:"slug" validate:"required,slug,max=120"`
	Title          string         `json:"title" validate:"required,max=200"`
	Department     string         `json:"department" validate:"max=120"`
	Location       string         `json:"location" validate:"max=200"`
	EmploymentType string         `json:"employmentType"`
	Type           string         `json:"type"`
	Summary        string         `json:"summary"`
	Requirements   []string       `json:"requirements"`
	Benefits       []string       `json:"benefits"`
	Description    map[string]any `json:"description"`
	Status         string         `json:"status"`
	Published      bool           `json:"published"`
}

type SubmitCareerInquiryRequest struct {
	ApplicantName  string `json:"applicantName" validate:"required,max=200"`
	ApplicantEmail string `json:"applicantEmail" validate:"required,email"`
	ApplicantPhone string `json:"applicantPhone" validate:"max=50"`
	ResumeURL      string `json:"resumeUrl" validate:"omitempty,url"`
	CoverLetter    string `json:"coverLetter" validate:"max=3000"`
	Source         string `json:"source" validate:"max=50"`
}
