package request

// PresignUploadRequest defines validation constraints for presigning an upload URL.
type PresignUploadRequest struct {
	FileName    string `json:"fileName" validate:"required,min=1,max=255"`
	ContentType string `json:"contentType" validate:"required"`
	Folder      string `json:"folder" validate:"omitempty,oneof=portfolio journal pages forms resumes media"`
	SizeBytes   int64  `json:"sizeBytes" validate:"omitempty,min=1"`
}

// DeleteUploadRequest defines constraints for removing an asset.
type DeleteUploadRequest struct {
	Key string `json:"key" validate:"required,min=3,max=500"`
}
