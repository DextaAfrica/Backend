package handlers

import (
	"context"
	"runtime"
	"time"

	"github.com/DextaAfrica/Backend/internal/infrastructure/cache"
	"github.com/DextaAfrica/Backend/internal/infrastructure/database"
	"github.com/DextaAfrica/Backend/internal/interfaces/http/response"

	"github.com/gofiber/fiber/v2"
)

type HealthHandler struct {
	db          *database.PostgresDB
	redisClient *cache.RedisClient
	startTime   time.Time
}

func NewHealthHandler(db *database.PostgresDB, redisClient *cache.RedisClient) *HealthHandler {
	return &HealthHandler{
		db:          db,
		redisClient: redisClient,
		startTime:   time.Now(),
	}
}

type HealthResponse struct {
	Status    string            `json:"status"`
	Uptime    string            `json:"uptime"`
	Timestamp string            `json:"timestamp"`
	Version   string            `json:"version"`
	Services  map[string]string `json:"services"`
	System    SystemStats       `json:"system"`
}

type SystemStats struct {
	GoVersion   string `json:"go_version"`
	NumCPU      int    `json:"num_cpu"`
	NumGoroutine int   `json:"num_goroutine"`
	MemoryAllocMB float64 `json:"memory_alloc_mb"`
}

func (h *HealthHandler) Check(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	services := make(map[string]string)
	isHealthy := true

	// Check Database
	if h.db != nil {
		if err := h.db.Ping(ctx); err != nil {
			services["database"] = "degraded: " + err.Error()
			isHealthy = false
		} else {
			services["database"] = "healthy"
		}
	} else {
		services["database"] = "not_configured"
	}

	// Check Redis
	if h.redisClient != nil {
		if err := h.redisClient.Ping(ctx); err != nil {
			services["redis"] = "degraded: " + err.Error()
			isHealthy = false
		} else {
			services["redis"] = "healthy"
		}
	} else {
		services["redis"] = "not_configured"
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	overallStatus := "healthy"
	statusCode := fiber.StatusOK
	if !isHealthy {
		overallStatus = "degraded"
		statusCode = fiber.StatusServiceUnavailable
	}

	healthData := HealthResponse{
		Status:    overallStatus,
		Uptime:    time.Since(h.startTime).String(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   "1.0.0-enterprise",
		Services:  services,
		System: SystemStats{
			GoVersion:    runtime.Version(),
			NumCPU:       runtime.NumCPU(),
			NumGoroutine: runtime.NumGoroutine(),
			MemoryAllocMB: float64(m.Alloc) / 1024 / 1024,
		},
	}

	return response.Success(c, statusCode, "Health check status", healthData, nil)
}

