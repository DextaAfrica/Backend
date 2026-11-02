package response_test

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/DextaAfrica/Backend/internal/interfaces/http/response"

	"github.com/gofiber/fiber/v2"
)

func TestSuccessResponseEnvelope(t *testing.T) {
	app := fiber.New()

	app.Get("/test-success", func(c *fiber.Ctx) error {
		c.Locals("trace_id", "req_test_123")
		return response.Success(c, fiber.StatusOK, "Operation completed", map[string]string{"foo": "bar"}, &response.PaginationMeta{
			Page:       1,
			Limit:      10,
			TotalItems: 1,
			TotalPages: 1,
		})
	})

	req := httptest.NewRequest("GET", "/test-success", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to execute test request: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var envelope response.ResponseEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("Failed to parse response JSON: %v", err)
	}

	if !envelope.Success {
		t.Errorf("Expected success to be true")
	}
	if envelope.Meta.TraceID != "req_test_123" {
		t.Errorf("Expected trace_id req_test_123, got %s", envelope.Meta.TraceID)
	}
	if envelope.Error != nil {
		t.Errorf("Expected error to be nil")
	}
}

func TestErrorResponseEnvelope(t *testing.T) {
	app := fiber.New()

	app.Get("/test-error", func(c *fiber.Ctx) error {
		c.Locals("trace_id", "req_err_456")
		return response.Error(c, fiber.StatusUnprocessableEntity, "Validation failed", "VALIDATION_ERROR", []response.ErrorDetail{
			{Field: "email", Issue: "Must be a valid email format"},
		})
	})

	req := httptest.NewRequest("GET", "/test-error", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to execute test request: %v", err)
	}

	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("Expected status 422, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var envelope response.ResponseEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("Failed to parse response JSON: %v", err)
	}

	if envelope.Success {
		t.Errorf("Expected success to be false")
	}
	if envelope.Error == nil || envelope.Error.Code != "VALIDATION_ERROR" {
		t.Errorf("Expected VALIDATION_ERROR code, got %v", envelope.Error)
	}
	if len(envelope.Error.Details) != 1 || envelope.Error.Details[0].Field != "email" {
		t.Errorf("Expected email validation error detail, got %v", envelope.Error.Details)
	}
}

