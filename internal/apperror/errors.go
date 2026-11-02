// Package apperror defines the single error vocabulary used across every
// layer of the application (repository, service, transport). Handlers never
// invent ad-hoc status codes or messages — they translate a *Error into an
// HTTP response using the Code field, so behavior is identical no matter
// which endpoint produced the failure. This is the app's error boundary:
// every handler is wrapped so a panic or an unmapped error still produces a
// well-formed JSON response instead of leaking a stack trace or hanging.
package apperror

import (
	"errors"
	"fmt"
	"net/http"
)

// Code is a stable, machine-readable error classification. Clients (the
// Next.js frontend) can safely switch on Code without parsing Message text.
type Code string

const (
	CodeValidation      Code = "VALIDATION_ERROR"
	CodeNotFound        Code = "NOT_FOUND"
	CodeConflict        Code = "CONFLICT"
	CodeUnauthorized    Code = "UNAUTHORIZED"
	CodeForbidden       Code = "FORBIDDEN"
	CodeRateLimited     Code = "RATE_LIMITED"
	CodeInternal        Code = "INTERNAL_ERROR"
	CodeUnavailable     Code = "SERVICE_UNAVAILABLE"
	CodeBadRequest      Code = "BAD_REQUEST"
)

// Error is the canonical application error. Message is always safe to show
// to an API consumer; internal details belong in Err, which is logged but
// never serialized.
type Error struct {
	Code    Code
	Message string
	Fields  map[string]string // field-level validation messages, when applicable
	Err     error             // wrapped internal cause, never exposed to clients
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error { return e.Err }

// HTTPStatus maps a Code to its HTTP status. Centralizing this mapping means
// adding a new error kind never requires touching handler code.
func (e *Error) HTTPStatus() int {
	switch e.Code {
	case CodeValidation, CodeBadRequest:
		return http.StatusBadRequest
	case CodeUnauthorized:
		return http.StatusUnauthorized
	case CodeForbidden:
		return http.StatusForbidden
	case CodeNotFound:
		return http.StatusNotFound
	case CodeConflict:
		return http.StatusConflict
	case CodeRateLimited:
		return http.StatusTooManyRequests
	case CodeUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

func New(code Code, message string) *Error {
	return &Error{Code: code, Message: message}
}

func Wrap(code Code, message string, err error) *Error {
	return &Error{Code: code, Message: message, Err: err}
}

func Validation(message string, fields map[string]string) *Error {
	return &Error{Code: CodeValidation, Message: message, Fields: fields}
}

func NotFound(resource string) *Error {
	return &Error{Code: CodeNotFound, Message: fmt.Sprintf("%s not found", resource)}
}

func Internal(err error) *Error {
	return &Error{Code: CodeInternal, Message: "an unexpected error occurred", Err: err}
}

func Unauthorized(message string) *Error {
	if message == "" {
		message = "authentication required"
	}
	return &Error{Code: CodeUnauthorized, Message: message}
}

func Forbidden(message string) *Error {
	if message == "" {
		message = "you do not have access to this resource"
	}
	return &Error{Code: CodeForbidden, Message: message}
}

func Conflict(message string) *Error {
	return &Error{Code: CodeConflict, Message: message}
}

func RateLimited() *Error {
	return &Error{Code: CodeRateLimited, Message: "too many requests, please try again shortly"}
}

// As extracts an *Error from err, wrapping unknown errors as internal errors
// so every code path — including third-party library errors — resolves to a
// well-formed *Error before reaching the transport layer.
func As(err error) *Error {
	if err == nil {
		return nil
	}
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr
	}
	return Internal(err)
}
