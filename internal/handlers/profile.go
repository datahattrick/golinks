package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"golinks/internal/config"
	"golinks/internal/db"
	"golinks/internal/models"
)

// ProfileHandler handles user profile pages.
type ProfileHandler struct {
	db  *db.DB
	cfg *config.Config
}

// NewProfileHandler creates a new profile handler.
func NewProfileHandler(database *db.DB, cfg *config.Config) *ProfileHandler {
	return &ProfileHandler{db: database, cfg: cfg}
}

// Show renders the user's profile page with their links.
func (h *ProfileHandler) Show(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return c.Redirect().To("/login")
	}

	links, err := h.db.GetLinksByUser(c.Context(), user.ID)
	if err != nil {
		return err
	}

	data := fiber.Map{
		"User":  user,
		"Links": links,
	}

	// Load fallback redirect options if user belongs to an org
	if user.OrganizationID != nil {
		fallbacks, err := h.db.ListFallbackRedirectsByOrg(c.Context(), *user.OrganizationID)
		if err == nil && len(fallbacks) > 0 {
			data["FallbackOptions"] = fallbacks
		}
	}

	return c.Render("profile", MergeBranding(data, h.cfg))
}

// UpdateFallbackPreference updates the user's fallback redirect preference.
func (h *ProfileHandler) UpdateFallbackPreference(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "authentication required")
	}

	fallbackIDStr := c.FormValue("fallback_redirect_id")

	var fallbackID *uuid.UUID
	if fallbackIDStr != "" && fallbackIDStr != "none" {
		id, err := uuid.Parse(fallbackIDStr)
		if err != nil {
			return htmxError(c, "Invalid fallback redirect ID")
		}

		// Verify the fallback belongs to the user's org
		fb, err := h.db.GetFallbackRedirectByID(c.Context(), id)
		if err != nil {
			return htmxError(c, "Fallback redirect not found")
		}
		if user.OrganizationID == nil || *user.OrganizationID != fb.OrganizationID {
			return htmxError(c, "Fallback redirect does not belong to your organization")
		}

		fallbackID = &id
	}

	if err := h.db.UpdateUserFallback(c.Context(), user.ID, fallbackID); err != nil {
		return htmxError(c, "Failed to update preference")
	}

	// Re-render the preference partial with updated state
	user.FallbackRedirectID = fallbackID
	var fallbacks []models.FallbackRedirect
	if user.OrganizationID != nil {
		fallbacks, _ = h.db.ListFallbackRedirectsByOrg(c.Context(), *user.OrganizationID)
	}

	return c.Render("partials/fallback_preference", fiber.Map{
		"User":            user,
		"FallbackOptions": fallbacks,
		"SavedMessage":    true,
	}, "")
}
