package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"golinks/internal/config"
	"golinks/internal/db"
	"golinks/internal/models"
)

// ManageHandler handles link management operations for moderators.
type ManageHandler struct {
	db  *db.DB
	cfg *config.Config
}

// NewManageHandler creates a new manage handler.
func NewManageHandler(database *db.DB, cfg *config.Config) *ManageHandler {
	return &ManageHandler{db: database, cfg: cfg}
}

// Index renders the management page.
func (h *ManageHandler) Index(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	// Check if user has management permissions
	if !user.IsOrgMod() {
		return fiber.NewError(fiber.StatusForbidden, "you do not have management permissions")
	}

	filter := c.Query("filter", "all")
	links, err := h.db.GetLinksForManagement(c.Context(), user, filter, 100)
	if err != nil {
		return err
	}

	// Check if this is an HTMX request
	if c.Get("HX-Request") == "true" {
		return c.Render("partials/manage_links_list", fiber.Map{
			"Links":  links,
			"Filter": filter,
			"User":   user,
		}, "")
	}

	return c.Render("manage", MergeBranding(fiber.Map{
		"User":   user,
		"Links":  links,
		"Filter": filter,
	}, h.cfg))
}

// Edit renders the inline edit form for a link.
func (h *ManageHandler) Edit(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	if !user.IsOrgMod() {
		return fiber.NewError(fiber.StatusForbidden, "you do not have management permissions")
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
		return fiber.NewError(fiber.StatusForbidden, "you do not have permission to manage this link")
	}

	return c.Render("partials/manage_edit_form", fiber.Map{
		"Link": link,
		"User": user,
	}, "")
}

// Update saves changes to a link.
func (h *ManageHandler) Update(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	if !user.IsOrgMod() {
		return fiber.NewError(fiber.StatusForbidden, "you do not have management permissions")
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
		return fiber.NewError(fiber.StatusForbidden, "you do not have permission to manage this link")
	}

	// Parse form data
	newURL := c.FormValue("url")
	newDescription := c.FormValue("description")

	if newURL == "" {
		return fiber.NewError(fiber.StatusBadRequest, "URL is required")
	}

	// Update link
	link.URL = newURL
	link.Description = newDescription

	// If URL changed, reset health status
	if err := h.db.UpdateLinkAndResetHealth(c.Context(), link); err != nil {
		return err
	}

	return c.Render("partials/manage_link_row", fiber.Map{
		"Link": link,
		"User": user,
	}, "")
}

// canManageLink checks if a user can manage a specific link.
func canManageLink(user *models.User, link *models.Link) bool {
	// Admins can manage anything
	if user.IsAdmin() {
		return true
	}

	// Global mods can manage global links
	if user.IsGlobalMod() && link.Scope == models.ScopeGlobal {
		return true
	}

	// Global mods can also manage org links
	if user.IsGlobalMod() && link.Scope == models.ScopeOrg {
		return true
	}

	// Org mods can manage links for their org
	if link.Scope == models.ScopeOrg && link.OrganizationID != nil {
		return user.CanModerateOrg(*link.OrganizationID)
	}

	return false
}
