package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"golinks/internal/db"
	"golinks/internal/models"
)

// HealthHandler handles link health check operations.
type HealthHandler struct {
	db     *db.DB
	client *http.Client
}

// NewHealthHandler creates a new health handler.
func NewHealthHandler(database *db.DB) *HealthHandler {
	return &HealthHandler{
		db: database,
		client: &http.Client{
			Timeout: 5 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Allow up to 10 redirects
				if len(via) >= 10 {
					return errors.New("too many redirects")
				}
				return nil
			},
		},
	}
}

// CheckLink performs an on-demand health check for a link.
func (h *HealthHandler) CheckLink(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	if !user.IsOrgMod() {
		return fiber.NewError(fiber.StatusForbidden, "you do not have permission to check link health")
	}

	idStr := c.Params("id")
	linkID, err := uuid.Parse(idStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid link id")
	}

	link, err := h.db.GetLinkByID(c.Context(), linkID)
	if err != nil {
		if errors.Is(err, db.ErrLinkNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "link not found")
		}
		return err
	}

	// Check permissions
	if !canManageLink(user, link) {
		return fiber.NewError(fiber.StatusForbidden, "you do not have permission to check this link")
	}

	// Perform health check
	status, errorMsg := h.checkURL(c.Context(), link.URL)

	// Update link health status
	if err := h.db.UpdateLinkHealthStatus(c.Context(), linkID, status, errorMsg); err != nil {
		return err
	}

	// Update link object for template
	link.HealthStatus = status
	now := time.Now()
	link.HealthCheckedAt = &now
	link.HealthError = errorMsg

	return c.Render("partials/health_status", fiber.Map{
		"Link": link,
	}, "")
}

// checkURL performs a HEAD request to check if a URL is healthy.
func (h *HealthHandler) checkURL(ctx context.Context, url string) (string, *string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		errMsg := "invalid URL: " + err.Error()
		return models.HealthUnhealthy, &errMsg
	}

	// Set a reasonable User-Agent
	req.Header.Set("User-Agent", "GoLinks-HealthChecker/1.0")

	resp, err := h.client.Do(req)
	if err != nil {
		errMsg := "connection failed: " + err.Error()
		return models.HealthUnhealthy, &errMsg
	}
	defer resp.Body.Close()

	// Check for successful status codes (2xx and 3xx are considered healthy)
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return models.HealthHealthy, nil
	}

	errMsg := "HTTP " + resp.Status
	return models.HealthUnhealthy, &errMsg
}
