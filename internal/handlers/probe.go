package handlers

import (
	"github.com/gofiber/fiber/v3"

	"golinks/internal/db"
)

// ProbeHandler handles Kubernetes health probe endpoints.
type ProbeHandler struct {
	db *db.DB
}

// NewProbeHandler creates a new probe handler.
func NewProbeHandler(database *db.DB) *ProbeHandler {
	return &ProbeHandler{db: database}
}

// Liveness handles the /healthz endpoint for Kubernetes liveness probes.
// Returns 200 OK if the application is running.
func (h *ProbeHandler) Liveness(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "ok",
	})
}

// Readiness handles the /readyz endpoint for Kubernetes readiness probes.
// Returns 200 OK if the application can serve traffic (database is reachable).
func (h *ProbeHandler) Readiness(c fiber.Ctx) error {
	if err := h.db.Ping(c.Context()); err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"status": "error",
			"error":  "database unavailable",
		})
	}

	return c.JSON(fiber.Map{
		"status": "ok",
	})
}
