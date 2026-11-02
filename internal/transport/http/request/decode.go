// Package request holds every inbound request DTO and the shared JSON
// decoding helper handlers use to populate and validate them.
package request

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/DextaAfrica/Backend/internal/apperror"
	"github.com/DextaAfrica/Backend/internal/validator"
)

const maxBodyBytes = 1 << 20 // 1MB — generous for a JSON form/CMS payload, small enough to bound abuse.

// Decode reads r.Body into dto, rejecting unknown fields and oversized
// bodies, then runs struct validation. Handlers get one call that covers
// both "is this valid JSON" and "does it satisfy our business rules."
func Decode(w http.ResponseWriter, r *http.Request, dto any) *apperror.Error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dto); err != nil {
		if err == io.EOF {
			return apperror.New(apperror.CodeBadRequest, "request body is required")
		}
		return apperror.New(apperror.CodeBadRequest, "request body is not valid JSON: "+err.Error())
	}

	return validator.Struct(dto)
}
