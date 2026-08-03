package request

type SubscribeRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type UnsubscribeRequest struct {
	Email string `json:"email" validate:"required,email"`
}
