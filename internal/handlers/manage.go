package handlers

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"golinks/internal/config"
	"golinks/internal/db"
	"golinks/internal/models"
	"golinks/internal/validation"
)

// orgColorPalette provides distinct badge colors for each organization.
var orgColorPalette = []string{
	"bg-emerald-100 text-emerald-700 dark:bg-emerald-900/50 dark:text-emerald-300",
	"bg-amber-100 text-amber-700 dark:bg-amber-900/50 dark:text-amber-300",
	"bg-rose-100 text-rose-700 dark:bg-rose-900/50 dark:text-rose-300",
	"bg-violet-100 text-violet-700 dark:bg-violet-900/50 dark:text-violet-300",
	"bg-teal-100 text-teal-700 dark:bg-teal-900/50 dark:text-teal-300",
	"bg-orange-100 text-orange-700 dark:bg-orange-900/50 dark:text-orange-300",
	"bg-indigo-100 text-indigo-700 dark:bg-indigo-900/50 dark:text-indigo-300",
	"bg-pink-100 text-pink-700 dark:bg-pink-900/50 dark:text-pink-300",
}

// ManageHandler handles link management operations.
type ManageHandler struct {
	db  *db.DB
	cfg *config.Config
}

// NewManageHandler creates a new manage handler.
func NewManageHandler(database *db.DB, cfg *config.Config) *ManageHandler {
	return &ManageHandler{db: database, cfg: cfg}
}

// buildOrgMaps returns name and color maps for all organizations, keyed by ID string.
func (h *ManageHandler) buildOrgMaps(ctx context.Context) (map[string]string, map[string]string) {
	orgNames := make(map[string]string)
	orgColors := make(map[string]string)
	orgs, err := h.db.GetAllOrganizations(ctx)
	if err != nil {
		return orgNames, orgColors
	}
	for i, org := range orgs {
		orgNames[org.ID.String()] = org.Name
		orgColors[org.ID.String()] = orgColorPalette[i%len(orgColorPalette)]
	}
	return orgNames, orgColors
}

// Index renders the management page.
func (h *ManageHandler) Index(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	filter := c.Query("filter", "all")
	isModerator := user.IsOrgMod()

	links, err := h.db.GetLinksForManagement(c.Context(), user, filter, 100)
	if err != nil {
		return err
	}

	orgNames, orgColors := h.buildOrgMaps(c.Context())

	// Collect link IDs and check which have pending edit requests
	linkIDs := make([]uuid.UUID, len(links))
	for i, l := range links {
		linkIDs[i] = l.ID
	}
	pendingEdits, _ := h.db.GetLinkIDsWithPendingEdits(c.Context(), linkIDs)
	if pendingEdits == nil {
		pendingEdits = make(map[string]bool)
	}

	data := fiber.Map{
		"Links":        links,
		"Filter":       filter,
		"User":         user,
		"OrgNames":     orgNames,
		"OrgColors":    orgColors,
		"IsModerator":  isModerator,
		"PendingEdits": pendingEdits,
	}

	// Check if this is an HTMX request
	if c.Get("HX-Request") == "true" {
		return c.Render("partials/manage_links_list", data, "")
	}

	return c.Render("manage", MergeBranding(data, h.cfg))
}

// Edit renders the inline edit form for a link.
func (h *ManageHandler) Edit(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
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
		"Link":        link,
		"User":        user,
		"IsModerator": user.IsOrgMod(),
	}, "")
}

// Update saves changes to a link (moderators only — direct edit).
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

	// Validate URL scheme
	if valid, msg := validation.ValidateURL(newURL); !valid {
		return fiber.NewError(fiber.StatusBadRequest, msg)
	}

	// Update link
	link.URL = newURL
	link.Description = newDescription

	// If URL changed, reset health status
	if err := h.db.UpdateLinkAndResetHealth(c.Context(), link); err != nil {
		return err
	}

	orgNames, orgColors := h.buildOrgMaps(c.Context())

	return c.Render("partials/manage_link_row", fiber.Map{
		"Link":        link,
		"User":        user,
		"OrgNames":    orgNames,
		"OrgColors":   orgColors,
		"IsModerator": true,
	}, "")
}

// RequestEdit creates an edit request for a link (regular users).
func (h *ManageHandler) RequestEdit(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	idStr := c.Params("id")
	linkID, err := uuid.Parse(idStr)
	if err != nil {
		return htmxError(c, "Invalid link ID")
	}

	link, err := h.db.GetLinkByID(c.Context(), linkID)
	if err != nil {
		if errors.Is(err, db.ErrLinkNotFound) {
			return htmxError(c, "Link not found")
		}
		return err
	}

	if !canManageLink(user, link) {
		return htmxError(c, "You do not have permission to edit this link")
	}

	newURL := c.FormValue("url")
	newDescription := c.FormValue("description")
	reason := c.FormValue("reason")

	if newURL == "" {
		return htmxError(c, "URL is required")
	}
	if reason == "" {
		return htmxError(c, "A reason is required for edit requests")
	}
	if valid, msg := validation.ValidateURL(newURL); !valid {
		return htmxError(c, msg)
	}

	req := &models.LinkEditRequest{
		LinkID:      linkID,
		UserID:      user.ID,
		URL:         newURL,
		Description: newDescription,
		Reason:      reason,
	}

	if err := h.db.CreateEditRequest(c.Context(), req); err != nil {
		if errors.Is(err, db.ErrPendingRequestLimit) {
			return htmxError(c, err.Error())
		}
		if errors.Is(err, db.ErrDuplicateEditRequest) {
			return htmxError(c, "You already have a pending edit request for this link")
		}
		return err
	}

	orgNames, orgColors := h.buildOrgMaps(c.Context())

	return c.Render("partials/manage_link_row", fiber.Map{
		"Link":        link,
		"User":        user,
		"OrgNames":    orgNames,
		"OrgColors":   orgColors,
		"IsModerator": false,
		"EditMessage": "Edit request submitted for review",
	}, "")
}

// RequestDeletion creates a deletion request for a link (regular users).
func (h *ManageHandler) RequestDeletion(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	idStr := c.Params("id")
	linkID, err := uuid.Parse(idStr)
	if err != nil {
		return htmxError(c, "Invalid link ID")
	}

	link, err := h.db.GetLinkByID(c.Context(), linkID)
	if err != nil {
		if errors.Is(err, db.ErrLinkNotFound) {
			return htmxError(c, "Link not found")
		}
		return err
	}

	if !canManageLink(user, link) {
		return htmxError(c, "You do not have permission to manage this link")
	}

	reason := c.FormValue("reason")
	if reason == "" {
		return htmxError(c, "A reason is required for deletion requests")
	}

	// Check pending request limit
	count, err := h.db.CountPendingRequestsByUser(c.Context(), user.ID)
	if err != nil {
		return err
	}
	if count >= 5 {
		return htmxError(c, db.ErrPendingRequestLimit.Error())
	}

	if err := h.db.RequestLinkDeletion(c.Context(), linkID, reason); err != nil {
		if errors.Is(err, db.ErrLinkNotFound) {
			return htmxError(c, "Link not found or not eligible for deletion request")
		}
		return err
	}

	// Re-fetch the link to get updated status
	link, _ = h.db.GetLinkByID(c.Context(), linkID)

	orgNames, orgColors := h.buildOrgMaps(c.Context())

	return c.Render("partials/manage_link_row", fiber.Map{
		"Link":        link,
		"User":        user,
		"OrgNames":    orgNames,
		"OrgColors":   orgColors,
		"IsModerator": false,
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
		if user.CanModerateOrg(*link.OrganizationID) {
			return true
		}
	}

	// Users can manage links they authored
	if link.CreatedBy != nil && *link.CreatedBy == user.ID {
		return true
	}

	return false
}
