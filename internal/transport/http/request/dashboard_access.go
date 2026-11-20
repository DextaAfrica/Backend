package request

type CreateDashboardAccessRequest struct {
	UserID           string `json:"userId"`
	RequesterName    string `json:"requesterName" validate:"required"`
	RequesterEmail   string `json:"requesterEmail" validate:"required,email"`
	Department       string `json:"department" validate:"required"`
	JobTitle         string `json:"jobTitle" validate:"required"`
	RequestedModules string `json:"requestedModules" validate:"required"`
	Justification    string `json:"justification" validate:"required"`
	SignatureData    string `json:"signatureData" validate:"required"`
	SignatureType    string `json:"signatureType"`
}

type ReviewDashboardAccessRequest struct {
	Approved        bool    `json:"approved"`
	ApprovedModules *string `json:"approvedModules"`
	ITNotes         *string `json:"itNotes"`
}
