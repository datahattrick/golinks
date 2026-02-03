package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"golinks/internal/db"
	"golinks/internal/models"
	"golinks/internal/validation"
)

// HealthHandler handles link health check operations via JSON API.
type HealthHandler struct {
	db     *db.DB
	client *http.Client
}

// NewHealthHandler creates a new API health handler.
func NewHealthHandler(database *db.DB) *HealthHandler {
	return &HealthHandler{
		db: database,
		client: &http.Client{
			Timeout: 5 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return errors.New("too many redirects")
				}
				return nil
			},
		},
	}
}

// CheckLink performs a health check and returns JSON results.
func (h *HealthHandler) CheckLink(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return jsonError(c, fiber.StatusUnauthorized, "unauthorized")
	}

	if !user.IsOrgMod() {
		return jsonError(c, fiber.StatusForbidden, "moderator access required")
	}

	linkID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return jsonError(c, fiber.StatusBadRequest, "invalid link id")
	}

	link, err := h.db.GetLinkByID(c.Context(), linkID)
	if err != nil {
		if errors.Is(err, db.ErrLinkNotFound) {
			return jsonError(c, fiber.StatusNotFound, "link not found")
		}
		return jsonError(c, fiber.StatusInternalServerError, "failed to fetch link")
	}

	if !canManageLink(user, link) {
		return jsonError(c, fiber.StatusForbidden, "you do not have permission to check this link")
	}

	var status string
	var errorMsg *string

	if valid, msg := validation.ValidateURLForHealthCheck(link.URL); !valid {
		status = models.HealthUnhealthy
		errorMsg = &msg
	} else {
		status, errorMsg = h.checkURL(c.Context(), link.URL)
	}

	if err := h.db.UpdateLinkHealthStatus(c.Context(), linkID, status, errorMsg); err != nil {
		return jsonError(c, fiber.StatusInternalServerError, "failed to update health status")
	}

	now := time.Now()
	resp := models.HealthCheckAPIResponse{
		LinkID:    linkID,
		Status:    status,
		CheckedAt: &now,
	}
	if errorMsg != nil {
		resp.Error = *errorMsg
	}

	return jsonSuccess(c, resp)
}

func (h *HealthHandler) checkURL(ctx context.Context, url string) (string, *string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		errMsg := "invalid URL: " + err.Error()
		return models.HealthUnhealthy, &errMsg
	}

	req.Header.Set("User-Agent", "GoLinks-HealthChecker/1.0")

	resp, err := h.client.Do(req)
	if err != nil {
		errMsg := "connection failed: " + err.Error()
		return models.HealthUnhealthy, &errMsg
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return models.HealthHealthy, nil
	}

	errMsg := "HTTP " + resp.Status
	return models.HealthUnhealthy, &errMsg
}
