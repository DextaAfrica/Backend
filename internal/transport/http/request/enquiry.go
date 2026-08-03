package request

type SubmitEnquiryRequest struct {
	Name       string `json:"name" validate:"required,max=200"`
	Email      string `json:"email" validate:"required,email"`
	Phone      string `json:"phone" validate:"max=40"`
	Subject    string `json:"subject" validate:"max=200"`
	Message    string `json:"message" validate:"required,max=5000"`
	SourcePage string `json:"sourcePage" validate:"max=200"`
}

type UpdateEnquiryStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=new read archived"`
}
