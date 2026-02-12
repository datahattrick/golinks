package handlers

import (
	"errors"

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

// List renders the my links page with all user link overrides and pending submissions.
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

	return c.Render("my_links", MergeBranding(fiber.Map{
		"UserLinks":    personalLinks,
		"PendingLinks": pendingLinks,
		"User":         user,
	}, h.cfg))
}

// Create creates a new user link override.
func (h *UserLinkHandler) Create(c fiber.Ctx) error {
	user := c.Locals("user").(*models.User)

	link := &models.UserLink{
		UserID:      user.ID,
		Keyword:     c.FormValue("keyword"),
		URL:         c.FormValue("url"),
		Description: c.FormValue("description"),
	}

	if link.Keyword == "" || link.URL == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Keyword and URL are required")
	}

	// Validate keyword format
	if !validation.ValidateKeyword(link.Keyword) {
		return fiber.NewError(fiber.StatusBadRequest, "Keyword must contain only letters, numbers, hyphens, and underscores")
	}

	// Validate URL scheme
	if valid, msg := validation.ValidateURL(link.URL); !valid {
		return fiber.NewError(fiber.StatusBadRequest, msg)
	}

	if err := h.db.CreateUserLink(c.Context(), link); err != nil {
		if errors.Is(err, db.ErrDuplicateKeyword) {
			return fiber.NewError(fiber.StatusConflict, "You already have a link with this keyword")
		}
		return err
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
