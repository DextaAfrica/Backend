package response

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

// PaginationMeta carries pagination information
type PaginationMeta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
}

// Meta carries request tracing and metadata
type Meta struct {
	TraceID    string          `json:"trace_id"`
	Timestamp  string          `json:"timestamp"`
	Pagination *PaginationMeta `json:"pagination,omitempty"`
}

// ErrorDetail describes an issue with a specific payload field
type ErrorDetail struct {
	Field string `json:"field"`
	Issue string `json:"issue"`
}

// ErrorPayload describes the error in a uniform structure
type ErrorPayload struct {
	Code    string        `json:"code"`
	Details []ErrorDetail `json:"details,omitempty"`
}

// ResponseEnvelope represents the enterprise universal API response envelope
type ResponseEnvelope struct {
	Success    bool          `json:"success"`
	StatusCode int           `json:"status_code"`
	Message    string        `json:"message"`
	Data       interface{}   `json:"data"`
	Meta       Meta          `json:"meta"`
	Error      *ErrorPayload `json:"error"`
}

func getTraceID(c *fiber.Ctx) string {
	if traceID, ok := c.Locals("trace_id").(string); ok && traceID != "" {
		return traceID
	}
	return c.Get("X-Trace-ID", "req_unknown")
}

// Success returns a standardized HTTP 200/201/204 response
func Success(c *fiber.Ctx, statusCode int, message string, data interface{}, pagination *PaginationMeta) error {
	envelope := ResponseEnvelope{
		Success:    true,
		StatusCode: statusCode,
		Message:    message,
		Data:       data,
		Meta: Meta{
			TraceID:    getTraceID(c),
			Timestamp:  time.Now().UTC().Format(time.RFC3339),
			Pagination: pagination,
		},
		Error: nil,
	}
	return c.Status(statusCode).JSON(envelope)
}

// Error returns a standardized HTTP 4xx/5xx error response
func Error(c *fiber.Ctx, statusCode int, message string, errorCode string, details []ErrorDetail) error {
	envelope := ResponseEnvelope{
		Success:    false,
		StatusCode: statusCode,
		Message:    message,
		Data:       nil,
		Meta: Meta{
			TraceID:   getTraceID(c),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		},
		Error: &ErrorPayload{
			Code:    errorCode,
			Details: details,
		},
	}
	return c.Status(statusCode).JSON(envelope)
}

