package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"golinks/internal/config"
	"golinks/internal/db"
	"golinks/internal/models"
)

// ModerationHandler handles link moderation operations.
type ModerationHandler struct {
	db  *db.DB
	cfg *config.Config
}

// NewModerationHandler creates a new moderation handler.
func NewModerationHandler(database *db.DB, cfg *config.Config) *ModerationHandler {
	return &ModerationHandler{db: database, cfg: cfg}
}

// Index renders the moderation dashboard.
func (h *ModerationHandler) Index(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	// Check if user has moderation permissions
	if !user.IsOrgMod() {
		return fiber.NewError(fiber.StatusForbidden, "you do not have moderation permissions")
	}

	var globalPending, orgPending []models.Link
	var err error

	// Global mods see global pending links
	if user.IsGlobalMod() {
		globalPending, err = h.db.GetPendingGlobalLinks(c.Context())
		if err != nil {
			return err
		}
	}

	// Org mods see their org's pending links
	if user.OrganizationID != nil {
		orgPending, err = h.db.GetPendingOrgLinks(c.Context(), *user.OrganizationID)
		if err != nil {
			return err
		}
	}

	return c.Render("moderation", MergeBranding(fiber.Map{
		"User":          user,
		"GlobalPending": globalPending,
		"OrgPending":    orgPending,
	}, h.cfg))
}

// Approve approves a pending link.
func (h *ModerationHandler) Approve(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	idStr := c.Params("id")
	linkID, err := uuid.Parse(idStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid link id")
	}

	// Get the link to check permissions
	link, err := h.db.GetLinkByID(c.Context(), linkID)
	if err != nil {
		if errors.Is(err, db.ErrLinkNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "link not found")
		}
		return err
	}

	// Check permissions
	if !canModerate(user, link) {
		return fiber.NewError(fiber.StatusForbidden, "you do not have permission to moderate this link")
	}

	if err := h.db.ApproveLink(c.Context(), linkID, user.ID); err != nil {
		if errors.Is(err, db.ErrLinkNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "link not found or already processed")
		}
		return err
	}

	// Return success message for HTMX
	return c.Render("partials/moderation_success", fiber.Map{
		"Action":  "approved",
		"Keyword": link.Keyword,
	}, "")
}

// Reject rejects a pending link.
func (h *ModerationHandler) Reject(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	idStr := c.Params("id")
	linkID, err := uuid.Parse(idStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid link id")
	}

	// Get the link to check permissions
	link, err := h.db.GetLinkByID(c.Context(), linkID)
	if err != nil {
		if errors.Is(err, db.ErrLinkNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "link not found")
		}
		return err
	}

	// Check permissions
	if !canModerate(user, link) {
		return fiber.NewError(fiber.StatusForbidden, "you do not have permission to moderate this link")
	}

	if err := h.db.RejectLink(c.Context(), linkID, user.ID); err != nil {
		if errors.Is(err, db.ErrLinkNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "link not found or already processed")
		}
		return err
	}

	// Return success message for HTMX
	return c.Render("partials/moderation_success", fiber.Map{
		"Action":  "rejected",
		"Keyword": link.Keyword,
	}, "")
}

// canModerate checks if a user can moderate a specific link.
func canModerate(user *models.User, link *models.Link) bool {
	// Admins can moderate anything
	if user.IsAdmin() {
		return true
	}

	// Global mods can moderate global links
	if user.IsGlobalMod() && link.Scope == models.ScopeGlobal {
		return true
	}

	// Org mods can moderate links for their org
	if link.Scope == models.ScopeOrg && link.OrganizationID != nil {
		return user.CanModerateOrg(*link.OrganizationID)
	}

	return false
}
