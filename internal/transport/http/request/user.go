package request

type SyncUserRequest struct {
	ID         string `json:"id"`
	Email      string `json:"email" validate:"required,email"`
	FullName   string `json:"fullName"`
	Role       string `json:"role"`
	Department string `json:"department"`
	JobTitle   string `json:"jobTitle"`
	Phone      string `json:"phone"`
	Status     string `json:"status"`
}
