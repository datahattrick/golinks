package handlers

import (
	"context"
	"errors"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"golinks/internal/config"
	"golinks/internal/db"
	"golinks/internal/models"
	"golinks/internal/validation"
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
// API clients (Accept: application/json) receive JSON instead of a redirect.
func (h *RedirectHandler) Redirect(c fiber.Ctx) error {
	keyword := c.Params("keyword")
	wantsJSON := strings.Contains(c.Get("Accept"), "application/json")

	// Validate keyword format to prevent path traversal or injection attacks
	if !validation.ValidateKeyword(keyword) {
		if wantsJSON {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status": "error",
				"error":  "invalid keyword",
			})
		}
		return c.Status(fiber.StatusBadRequest).Render("error", MergeBranding(fiber.Map{
			"Title":   "Invalid Keyword",
			"Message": "The keyword contains invalid characters.",
		}, h.cfg))
	}

	user, _ := c.Locals("user").(*models.User)

	// Use tier-based resolution
	var userID *uuid.UUID
	if user != nil {
		userID = &user.ID
	}

	resolved, err := h.db.ResolveKeywordForUser(c.Context(), userID, keyword)
	if err != nil {
		if errors.Is(err, db.ErrLinkNotFound) {
			if wantsJSON {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"status": "error",
					"error":  "keyword not found",
				})
			}
			// Check for organization fallback URL (browser only)
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

	// Return JSON for API clients
	if wantsJSON {
		return c.JSON(fiber.Map{
			"status": "ok",
			"data": fiber.Map{
				"keyword": keyword,
				"url":     resolved.URL,
				"tier":    resolved.Tier,
				"source":  resolved.Source,
			},
		})
	}

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
