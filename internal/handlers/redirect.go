package handlers

import (
	"context"
	"errors"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"golinks/internal/config"
	"golinks/internal/db"
	"golinks/internal/metrics"
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
// Resolution order: personal > org > global.
// API clients (Accept: application/json) receive JSON instead of a redirect.
func (h *RedirectHandler) Redirect(c fiber.Ctx) error {
	keyword := validation.NormalizeKeyword(c.Params("keyword"))
	wantsJSON := strings.Contains(c.Get("Accept"), "application/json")

	// Validate keyword format to prevent path traversal or injection attacks
	if !validation.ValidateKeyword(keyword) {
		if wantsJSON {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status": "error",
				"error":  "invalid keyword",
			})
		}
		user, _ := c.Locals("user").(*models.User)
		return c.Status(fiber.StatusBadRequest).Render("error", MergeBranding(fiber.Map{
			"Title":   "Invalid Keyword",
			"Message": "The keyword contains invalid characters.",
			"User":    user,
		}, h.cfg))
	}

	user, _ := c.Locals("user").(*models.User)

	var userID *uuid.UUID
	var orgID *uuid.UUID
	if user != nil {
		userID = &user.ID
		orgID = user.OrganizationID
	}

	resolved, err := h.db.ResolveKeywordForUser(c.Context(), userID, orgID, keyword)
	if err != nil {
		if errors.Is(err, db.ErrLinkNotFound) {
			if wantsJSON {
				metrics.RecordKeywordLookup(keyword, models.OutcomeNotFound)
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"status": "error",
					"error":  "keyword not found",
				})
			}
			// Check for user's fallback redirect preference (browser only)
			if user != nil && user.FallbackRedirectID != nil {
				fb, fbErr := h.db.GetFallbackRedirectByID(c.Context(), *user.FallbackRedirectID)
				if fbErr == nil {
					metrics.RecordKeywordLookup(keyword, models.OutcomeFallback)
					return c.Redirect().To(fb.URL + keyword)
				}
			}
			metrics.RecordKeywordLookup(keyword, models.OutcomeNotFound)
			// Look up similar keywords for "did you mean?" suggestions
			suggestions, _ := h.db.GetSimilarKeywords(c.Context(), keyword, orgID, 5)
			return c.Status(fiber.StatusNotFound).Render("not_found", MergeBranding(fiber.Map{
				"Title":       "Not Found",
				"Keyword":     keyword,
				"Suggestions": suggestions,
				"User":        user,
			}, h.cfg))
		}
		return err
	}

	// Record successful resolution and increment click count asynchronously
	metrics.RecordKeywordLookup(keyword, models.OutcomeResolved)
	go h.db.IncrementResolvedLinkClickCount(context.Background(), resolved, userID)

	// Return JSON for API clients
	if wantsJSON {
		return c.JSON(fiber.Map{
			"status": "ok",
			"data": fiber.Map{
				"keyword": keyword,
				"url":     resolved.URL,
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
				"User":    user,
			}, h.cfg))
		}
		return err
	}

	go h.db.IncrementClickCount(context.Background(), link.ID)
	return c.Redirect().To(link.URL)
}
