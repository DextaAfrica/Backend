package validator

import "testing"

type sampleDTO struct {
	Email string `json:"email" validate:"required,email"`
	Slug  string `json:"slug" validate:"required,slug"`
	Bio   string `json:"bio" validate:"max=10"`
}

func TestStruct_Valid(t *testing.T) {
	dto := sampleDTO{Email: "person@example.com", Slug: "seren-redwood", Bio: "short"}
	if err := Struct(dto); err != nil {
		t.Fatalf("expected no validation error, got %v", err)
	}
}

func TestStruct_InvalidEmail(t *testing.T) {
	dto := sampleDTO{Email: "not-an-email", Slug: "valid-slug"}
	err := Struct(dto)
	if err == nil {
		t.Fatal("expected validation error for invalid email")
	}
	if _, ok := err.Fields["email"]; !ok {
		t.Fatalf("expected field error for 'email', got fields: %v", err.Fields)
	}
}

func TestStruct_InvalidSlug(t *testing.T) {
	cases := []string{"Not Lowercase", "trailing-", "under_score", "double--hyphen "}
	for _, slug := range cases {
		dto := sampleDTO{Email: "person@example.com", Slug: slug}
		err := Struct(dto)
		if err == nil {
			t.Errorf("slug %q: expected validation error, got none", slug)
			continue
		}
		if _, ok := err.Fields["slug"]; !ok {
			t.Errorf("slug %q: expected field error for 'slug', got fields: %v", slug, err.Fields)
		}
	}
}

func TestStruct_MissingRequiredFields(t *testing.T) {
	err := Struct(sampleDTO{})
	if err == nil {
		t.Fatal("expected validation error for empty struct")
	}
	if len(err.Fields) < 2 {
		t.Fatalf("expected at least 2 field errors, got %v", err.Fields)
	}
}

func TestStruct_MaxLength(t *testing.T) {
	dto := sampleDTO{Email: "person@example.com", Slug: "ok-slug", Bio: "this bio is way too long"}
	err := Struct(dto)
	if err == nil {
		t.Fatal("expected validation error for bio exceeding max length")
	}
	if _, ok := err.Fields["bio"]; !ok {
		t.Fatalf("expected field error for 'bio', got fields: %v", err.Fields)
	}
}
