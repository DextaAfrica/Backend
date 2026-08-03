// Package validator is the single entry point for validating inbound
// request DTOs. Handlers call Struct(dto) and get back a ready-to-return
// *apperror.Error with field-level messages — no handler hand-rolls its own
// validation logic or error formatting.
package validator

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	pv "github.com/go-playground/validator/v10"

	"github.com/DextaAfrica/Backend/internal/apperror"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

var instance *pv.Validate

func init() {
	instance = pv.New(pv.WithRequiredStructEnabled())
	if err := instance.RegisterValidation("slug", func(fl pv.FieldLevel) bool {
		return slugPattern.MatchString(fl.Field().String())
	}); err != nil {
		panic(fmt.Sprintf("validator: register slug rule: %v", err))
	}

	// Report field names using their `json` tag so error responses match
	// the request body shape the frontend actually sent, not Go's exported
	// field names.
	instance.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]
		if name == "-" || name == "" {
			return field.Name
		}
		return name
	})
}

// Struct validates dto against its `validate:"..."` tags and returns a
// single *apperror.Error with one human-readable message per invalid field,
// or nil when the struct is valid.
func Struct(dto any) *apperror.Error {
	err := instance.Struct(dto)
	if err == nil {
		return nil
	}

	fieldErrs, ok := err.(pv.ValidationErrors)
	if !ok {
		return apperror.Validation("invalid request", nil)
	}

	fields := make(map[string]string, len(fieldErrs))
	for _, fe := range fieldErrs {
		fields[fe.Field()] = message(fe)
	}
	return apperror.Validation("validation failed", fields)
}

func message(fe pv.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email address"
	case "min":
		return fmt.Sprintf("must be at least %s characters", fe.Param())
	case "max":
		return fmt.Sprintf("must be at most %s characters", fe.Param())
	case "slug":
		return "must be lowercase letters, numbers, and hyphens only"
	case "oneof":
		return fmt.Sprintf("must be one of: %s", fe.Param())
	case "url":
		return "must be a valid URL"
	default:
		return fmt.Sprintf("failed validation: %s", fe.Tag())
	}
}
