package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"golinks/internal/config"
	"golinks/internal/db"
	"golinks/internal/models"
)

// FallbackRedirectHandler handles admin management of fallback redirect options.
type FallbackRedirectHandler struct {
	db  *db.DB
	cfg *config.Config
}

// NewFallbackRedirectHandler creates a new fallback redirect handler.
func NewFallbackRedirectHandler(database *db.DB, cfg *config.Config) *FallbackRedirectHandler {
	return &FallbackRedirectHandler{db: database, cfg: cfg}
}

// List renders the admin page for managing fallback redirect options.
func (h *FallbackRedirectHandler) List(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok || !user.IsAdmin() {
		return fiber.NewError(fiber.StatusForbidden, "admin access required")
	}

	orgs, err := h.db.GetAllOrganizations(c.Context())
	if err != nil {
		return err
	}

	// Collect all fallback redirects grouped by org
	type orgWithFallbacks struct {
		Org       models.Organization
		Fallbacks []models.FallbackRedirect
	}
	var data []orgWithFallbacks
	for _, org := range orgs {
		fallbacks, err := h.db.ListFallbackRedirectsByOrg(c.Context(), org.ID)
		if err != nil {
			return err
		}
		data = append(data, orgWithFallbacks{Org: org, Fallbacks: fallbacks})
	}

	return c.Render("fallback_redirects", MergeBranding(fiber.Map{
		"User":             user,
		"Orgs":             orgs,
		"OrgWithFallbacks": data,
	}, h.cfg, c.Path()))
}

// Create creates a new fallback redirect option (admin only).
func (h *FallbackRedirectHandler) Create(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok || !user.IsAdmin() {
		return fiber.NewError(fiber.StatusForbidden, "admin access required")
	}

	orgIDStr := c.FormValue("organization_id")
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		return htmxError(c, "Invalid organization")
	}

	name := c.FormValue("name")
	url := c.FormValue("url")
	if name == "" || url == "" {
		return htmxError(c, "Name and URL are required")
	}

	r := &models.FallbackRedirect{
		OrganizationID: orgID,
		Name:           name,
		URL:            url,
	}
	if err := h.db.CreateFallbackRedirect(c.Context(), r); err != nil {
		return htmxError(c, "Failed to create fallback redirect: "+err.Error())
	}

	// Return the updated list for this org
	return h.renderOrgFallbacks(c, orgID)
}

// Update updates an existing fallback redirect option (admin only).
func (h *FallbackRedirectHandler) Update(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok || !user.IsAdmin() {
		return fiber.NewError(fiber.StatusForbidden, "admin access required")
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return htmxError(c, "Invalid fallback redirect ID")
	}

	name := c.FormValue("name")
	url := c.FormValue("url")
	if name == "" || url == "" {
		return htmxError(c, "Name and URL are required")
	}

	// Get the existing record to know which org to re-render
	existing, err := h.db.GetFallbackRedirectByID(c.Context(), id)
	if err != nil {
		return htmxError(c, "Fallback redirect not found")
	}

	if err := h.db.UpdateFallbackRedirect(c.Context(), id, name, url); err != nil {
		return htmxError(c, "Failed to update: "+err.Error())
	}

	return h.renderOrgFallbacks(c, existing.OrganizationID)
}

// Delete deletes a fallback redirect option (admin only).
func (h *FallbackRedirectHandler) Delete(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok || !user.IsAdmin() {
		return fiber.NewError(fiber.StatusForbidden, "admin access required")
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return htmxError(c, "Invalid fallback redirect ID")
	}

	// Get the existing record to know which org to re-render
	existing, err := h.db.GetFallbackRedirectByID(c.Context(), id)
	if err != nil {
		return htmxError(c, "Fallback redirect not found")
	}

	if err := h.db.DeleteFallbackRedirect(c.Context(), id); err != nil {
		return htmxError(c, "Failed to delete: "+err.Error())
	}

	return h.renderOrgFallbacks(c, existing.OrganizationID)
}

// renderOrgFallbacks re-renders the fallback list partial for a specific org.
func (h *FallbackRedirectHandler) renderOrgFallbacks(c fiber.Ctx, orgID uuid.UUID) error {
	fallbacks, err := h.db.ListFallbackRedirectsByOrg(c.Context(), orgID)
	if err != nil {
		return err
	}

	org, err := h.db.GetOrganizationByID(c.Context(), orgID)
	if err != nil {
		return err
	}

	return c.Render("partials/fallback_redirect_list", fiber.Map{
		"Org":       org,
		"Fallbacks": fallbacks,
	}, "")
}
