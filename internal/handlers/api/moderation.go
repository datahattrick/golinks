package api

import (
	"encoding/json"
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"golinks/internal/config"
	"golinks/internal/db"
	"golinks/internal/email"
	"golinks/internal/models"
)

// ModerationHandler handles link moderation via JSON API.
type ModerationHandler struct {
	db       *db.DB
	cfg      *config.Config
	notifier *email.Notifier
}

// NewModerationHandler creates a new API moderation handler.
func NewModerationHandler(database *db.DB, cfg *config.Config, notifier *email.Notifier) *ModerationHandler {
	return &ModerationHandler{db: database, cfg: cfg, notifier: notifier}
}

// ListPending returns all pending links visible to the current moderator.
func (h *ModerationHandler) ListPending(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return jsonError(c, fiber.StatusUnauthorized, "unauthorized")
	}

	if !user.IsOrgMod() {
		return jsonError(c, fiber.StatusForbidden, "moderator access required")
	}

	var globalPending, orgPending []models.Link

	if user.IsGlobalMod() {
		var err error
		globalPending, err = h.db.GetPendingGlobalLinks(c.Context())
		if err != nil {
			return jsonError(c, fiber.StatusInternalServerError, "failed to fetch pending links")
		}
		orgPending, err = h.db.GetAllPendingOrgLinks(c.Context())
		if err != nil {
			return jsonError(c, fiber.StatusInternalServerError, "failed to fetch pending links")
		}
	} else if user.OrganizationID != nil {
		var err error
		orgPending, err = h.db.GetPendingOrgLinks(c.Context(), *user.OrganizationID)
		if err != nil {
			return jsonError(c, fiber.StatusInternalServerError, "failed to fetch pending links")
		}
	}

	// Ensure non-null arrays in JSON
	if globalPending == nil {
		globalPending = []models.Link{}
	}
	if orgPending == nil {
		orgPending = []models.Link{}
	}

	return jsonSuccess(c, fiber.Map{
		"global": globalPending,
		"org":    orgPending,
	})
}

// Approve approves a pending link.
func (h *ModerationHandler) Approve(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return jsonError(c, fiber.StatusUnauthorized, "unauthorized")
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

	if !canModerate(user, link) {
		return jsonError(c, fiber.StatusForbidden, "you do not have permission to moderate this link")
	}

	if err := h.db.ApproveLink(c.Context(), linkID, user.ID); err != nil {
		if errors.Is(err, db.ErrLinkNotFound) {
			return jsonError(c, fiber.StatusNotFound, "link not found or already processed")
		}
		return jsonError(c, fiber.StatusInternalServerError, "failed to approve link")
	}

	if h.notifier != nil {
		h.notifier.NotifyUserLinkApproved(c.Context(), link, user)
	}

	return jsonSuccess(c, fiber.Map{
		"message": "link approved",
		"keyword": link.Keyword,
	})
}

// Reject rejects a pending link with an optional reason.
func (h *ModerationHandler) Reject(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return jsonError(c, fiber.StatusUnauthorized, "unauthorized")
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

	if !canModerate(user, link) {
		return jsonError(c, fiber.StatusForbidden, "you do not have permission to moderate this link")
	}

	// Parse optional reason from body
	var body struct {
		Reason string `json:"reason"`
	}
	json.Unmarshal(c.Body(), &body) // Body and reason are both optional

	if err := h.db.RejectLink(c.Context(), linkID, user.ID); err != nil {
		if errors.Is(err, db.ErrLinkNotFound) {
			return jsonError(c, fiber.StatusNotFound, "link not found or already processed")
		}
		return jsonError(c, fiber.StatusInternalServerError, "failed to reject link")
	}

	if h.notifier != nil {
		h.notifier.NotifyUserLinkRejected(c.Context(), link, body.Reason)
	}

	return jsonSuccess(c, fiber.Map{
		"message": "link rejected",
		"keyword": link.Keyword,
	})
}

// canModerate checks if a user can moderate a specific link.
func canModerate(user *models.User, link *models.Link) bool {
	if user.IsGlobalMod() {
		return true
	}
	if link.Scope == models.ScopeOrg && link.OrganizationID != nil {
		return user.CanModerateOrg(*link.OrganizationID)
	}
	return false
}
