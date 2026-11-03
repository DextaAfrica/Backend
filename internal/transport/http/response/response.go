// Package response centralizes every shape of HTTP response the API emits so
// handlers never hand-roll JSON encoding. One envelope for success, one for
// errors, everywhere.
package response

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/DextaAfrica/Backend/internal/apperror"
	"github.com/DextaAfrica/Backend/internal/requestid"
)

// Envelope is the success response shape: {"data": ...}. A stable wrapper
// lets the frontend add response metadata (pagination, etc.) later without
// a breaking change to every existing consumer.
type Envelope struct {
	Data any   `json:"data"`
	Meta *Meta `json:"meta,omitempty"`
}

type Meta struct {
	Page       int `json:"page,omitempty"`
	PerPage    int `json:"per_page,omitempty"`
	TotalItems int `json:"total_items,omitempty"`
	TotalPages int `json:"total_pages,omitempty"`
}

// ErrorBody is the error response shape returned for every failed request,
// regardless of which layer produced the failure.
type ErrorBody struct {
	Error struct {
		Code    apperror.Code     `json:"code"`
		Message string            `json:"message"`
		Fields  map[string]string `json:"fields,omitempty"`
	} `json:"error"`
	RequestID string `json:"request_id,omitempty"`
}

func JSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("response: failed to encode JSON body", "error", err)
	}
}

func OK(w http.ResponseWriter, data any) {
	JSON(w, http.StatusOK, Envelope{Data: data})
}

func Created(w http.ResponseWriter, data any) {
	JSON(w, http.StatusCreated, Envelope{Data: data})
}

func Paginated(w http.ResponseWriter, data any, meta Meta) {
	JSON(w, http.StatusOK, Envelope{Data: data, Meta: &meta})
}

func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// Error translates any error into the canonical ErrorBody, logging server
// faults (5xx) at error level with the request ID for correlation and
// client faults (4xx) at debug level to keep logs signal-heavy.
func Error(w http.ResponseWriter, r *http.Request, err error) {
	appErr := apperror.As(err)
	requestID := requestid.FromContext(r.Context())

	logAttrs := []any{
		"request_id", requestID,
		"code", appErr.Code,
		"path", r.URL.Path,
		"method", r.Method,
	}
	if appErr.HTTPStatus() >= 500 {
		slog.Error("request failed", append(logAttrs, "cause", appErr.Unwrap())...)
	} else {
		slog.Debug("request rejected", logAttrs...)
	}

	body := ErrorBody{RequestID: requestID}
	body.Error.Code = appErr.Code
	body.Error.Message = appErr.Message
	body.Error.Fields = appErr.Fields

	JSON(w, appErr.HTTPStatus(), body)
}
