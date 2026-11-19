package request

type SaveCustomFormRequest struct {
	Slug          string `json:"slug" validate:"max=100"`
	Title         string `json:"title" validate:"required,max=200"`
	Description   string `json:"description" validate:"max=2000"`
	FormType      string `json:"formType" validate:"max=50"`
	Status        string `json:"status" validate:"max=50"`
	CoverImageURL string `json:"coverImageUrl" validate:"max=500"`
	EventDate     string `json:"eventDate"`
	EventLocation string `json:"eventLocation" validate:"max=300"`
	ActiveFrom    string `json:"activeFrom"`
	ActiveUntil   string `json:"activeUntil"`
	Fields        any    `json:"fields"`
	Metadata      any    `json:"metadata"`
}

type BatchEmailSubmissionsRequest struct {
	SubmissionIDs []string `json:"submissionIds" validate:"required,min=1"`
	Subject       string   `json:"subject" validate:"required,max=200"`
	Message       string   `json:"message" validate:"required,max=10000"`
}

type SubmitFormRequest struct {
	FullName       string `json:"fullName" validate:"required,max=200"`
	Email          string `json:"email" validate:"required,email"`
	Phone          string `json:"phone" validate:"max=50"`
	Company        string `json:"company" validate:"max=200"`
	JobTitle       string `json:"jobTitle" validate:"max=200"`
	Category       string `json:"category" validate:"max=100"`
	AttendanceMode string `json:"attendanceMode" validate:"max=50"`
	Answers        any    `json:"answers"`
	Notes          string `json:"notes" validate:"max=2000"`
}
