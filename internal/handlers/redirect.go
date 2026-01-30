package handlers

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"golinks/internal/config"
	"golinks/internal/db"
	"golinks/internal/models"
)

// RedirectHandler handles keyword-to-URL redirects.
type RedirectHandler struct {
	db  *db.DB
	cfg *config.Config
}

// NewRedirectHandler creates a new redirect handler.
func NewRedirectHandler(database *db.DB, cfg *config.Config) *RedirectHandler {
	return &RedirectHandler{db: database, cfg: cfg}
}

// Redirect looks up a keyword and redirects to the associated URL.
// Uses tier-based resolution: personal (100) > group (1-99) > global (0)
// At the same tier, user's primary group wins.
func (h *RedirectHandler) Redirect(c fiber.Ctx) error {
	keyword := c.Params("keyword")
	user, _ := c.Locals("user").(*models.User)

	// Use tier-based resolution
	var userID *uuid.UUID
	if user != nil {
		userID = &user.ID
	}

	resolved, err := h.db.ResolveKeywordForUser(c.Context(), userID, keyword)
	if err != nil {
		if errors.Is(err, db.ErrLinkNotFound) {
			// Check for organization fallback URL
			if user != nil && user.OrganizationID != nil {
				org, orgErr := h.db.GetOrganizationByID(c.Context(), *user.OrganizationID)
				if orgErr == nil && org.FallbackRedirectURL != nil && *org.FallbackRedirectURL != "" {
					return c.Redirect().To(*org.FallbackRedirectURL + keyword)
				}
			}
			return c.Status(fiber.StatusNotFound).Render("error", MergeBranding(fiber.Map{
				"Title":   "Not Found",
				"Message": "The link '" + keyword + "' does not exist.",
			}, h.cfg))
		}
		return err
	}

	// Increment click count asynchronously
	go h.db.IncrementResolvedLinkClickCount(context.Background(), resolved, userID)

	return c.Redirect().To(resolved.URL)
}

// Random redirects to a random link ("I'm Feeling Lucky" feature).
func (h *RedirectHandler) Random(c fiber.Ctx) error {
	if !h.cfg.EnableRandomKeywords {
		return fiber.NewError(fiber.StatusNotFound, "Random links feature is not enabled")
	}

	user, _ := c.Locals("user").(*models.User)

	var orgID *uuid.UUID
	if user != nil && user.OrganizationID != nil {
		orgID = user.OrganizationID
	}

	link, err := h.db.GetRandomApprovedLink(c.Context(), orgID)
	if err != nil {
		if errors.Is(err, db.ErrLinkNotFound) {
			return c.Status(fiber.StatusNotFound).Render("error", MergeBranding(fiber.Map{
				"Title":   "No Links",
				"Message": "There are no links available.",
			}, h.cfg))
		}
		return err
	}

	go h.db.IncrementClickCount(context.Background(), link.ID)
	return c.Redirect().To(link.URL)
}
