package handlers

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"golinks/internal/config"
	"golinks/internal/db"
	"golinks/internal/models"
	"golinks/internal/validation"
)

// UserLinkHandler handles user-specific link management.
type UserLinkHandler struct {
	db  *db.DB
	cfg *config.Config
}

// NewUserLinkHandler creates a new user link handler.
func NewUserLinkHandler(database *db.DB, cfg *config.Config) *UserLinkHandler {
	return &UserLinkHandler{db: database, cfg: cfg}
}

// List renders the my links page with all user link overrides, pending submissions, and shares.
func (h *UserLinkHandler) List(c fiber.Ctx) error {
	user := c.Locals("user").(*models.User)

	// Get personal links
	personalLinks, err := h.db.GetUserLinks(c.Context(), user.ID)
	if err != nil {
		return err
	}

	// Get pending submissions (org/global links awaiting approval)
	pendingLinks, err := h.db.GetPendingLinksByUser(c.Context(), user.ID)
	if err != nil {
		return err
	}

	// Get incoming and outgoing shared links
	incomingShares, err := h.db.GetIncomingShares(c.Context(), user.ID)
	if err != nil {
		return err
	}

	outgoingShares, err := h.db.GetOutgoingShares(c.Context(), user.ID)
	if err != nil {
		return err
	}

	return c.Render("my_links", MergeBranding(fiber.Map{
		"UserLinks":      personalLinks,
		"PendingLinks":   pendingLinks,
		"IncomingShares": incomingShares,
		"OutgoingShares": outgoingShares,
		"User":           user,
	}, h.cfg))
}

// Create creates new user link overrides. Supports comma-separated keywords.
func (h *UserLinkHandler) Create(c fiber.Ctx) error {
	user := c.Locals("user").(*models.User)

	url := c.FormValue("url")
	description := c.FormValue("description")

	keywords := splitKeywords(c.FormValue("keyword"))
	if len(keywords) == 0 || url == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Keyword and URL are required")
	}

	// Validate URL scheme
	if valid, msg := validation.ValidateURL(url); !valid {
		return fiber.NewError(fiber.StatusBadRequest, msg)
	}

	// Validate all keywords first
	for _, kw := range keywords {
		if !validation.ValidateKeyword(kw) {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid keyword: "+kw)
		}
	}

	// Create each keyword
	var errMsgs []string
	for _, kw := range keywords {
		link := &models.UserLink{
			UserID:      user.ID,
			Keyword:     kw,
			URL:         url,
			Description: description,
		}
		if err := h.db.CreateUserLink(c.Context(), link); err != nil {
			if errors.Is(err, db.ErrDuplicateKeyword) {
				errMsgs = append(errMsgs, kw+": duplicate")
			} else {
				errMsgs = append(errMsgs, kw+": "+err.Error())
			}
		}
	}

	if len(errMsgs) == len(keywords) {
		return fiber.NewError(fiber.StatusConflict, "Failed: "+strings.Join(errMsgs, "; "))
	}

	// Return the updated list for HTMX
	links, err := h.db.GetUserLinks(c.Context(), user.ID)
	if err != nil {
		return err
	}

	return c.Render("partials/user_links_list", fiber.Map{
		"UserLinks": links,
	}, "")
}

// Delete removes a user link override.
func (h *UserLinkHandler) Delete(c fiber.Ctx) error {
	user := c.Locals("user").(*models.User)

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid link ID")
	}

	if err := h.db.DeleteUserLink(c.Context(), id, user.ID); err != nil {
		if errors.Is(err, db.ErrUserLinkNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Link not found")
		}
		return err
	}

	// Return empty for HTMX to remove the element
	return c.SendString("")
}
