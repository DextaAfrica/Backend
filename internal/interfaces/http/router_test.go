package http_test

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/DextaAfrica/Backend/internal/config"
	httpInterface "github.com/DextaAfrica/Backend/internal/interfaces/http"
	"github.com/DextaAfrica/Backend/internal/interfaces/http/response"

	"github.com/gofiber/fiber/v2"
)

func TestHealthEndpoint(t *testing.T) {
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	app := fiber.New()
	httpInterface.SetupRouter(app, cfg, nil, nil)

	req := httptest.NewRequest("GET", "/health", nil)
	req.Header.Set("X-Trace-ID", "req_health_trace_999")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var envelope response.ResponseEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("Failed to parse response envelope: %v", err)
	}

	if !envelope.Success {
		t.Errorf("Expected success true, got false")
	}

	if envelope.Meta.TraceID != "req_health_trace_999" {
		t.Errorf("Expected trace ID req_health_trace_999, got %s", envelope.Meta.TraceID)
	}
}

func TestRootEndpoint(t *testing.T) {
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	app := fiber.New()
	httpInterface.SetupRouter(app, cfg, nil, nil)

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		t.Fatalf("Failed to parse json: %v", err)
	}

	if data["status"] != "OPERATIONAL" {
		t.Errorf("Expected status OPERATIONAL, got %v", data["status"])
	}
}
