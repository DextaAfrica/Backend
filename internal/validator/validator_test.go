package validator_test

import (
	"testing"

	"github.com/DextaAfrica/Backend/internal/validator"
)

type TestDTO struct {
	Slug  string `json:"slug" validate:"required,slug"`
	Email string `json:"email" validate:"required,email"`
}

func TestValidatorSlugAndEmailSuccess(t *testing.T) {
	dto := TestDTO{
		Slug:  "prime-estate-lagos",
		Email: "info@dextaafrica.com",
	}

	appErr := validator.Struct(dto)
	if appErr != nil {
		t.Fatalf("unexpected validation error: %v", appErr)
	}
}

func TestValidatorSlugFailure(t *testing.T) {
	dto := TestDTO{
		Slug:  "Invalid_Slug!123",
		Email: "info@dextaafrica.com",
	}

	appErr := validator.Struct(dto)
	if appErr == nil {
		t.Fatal("expected error for invalid slug, got nil")
	}
	if appErr.Fields["slug"] == "" {
		t.Errorf("expected field error for 'slug', got %v", appErr.Fields)
	}
}
