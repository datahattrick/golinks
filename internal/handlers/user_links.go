package handlers

import (
	"errors"
	"fmt"
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
	}, h.cfg, c.Path()))
}

// PendingCount returns an HTML badge showing the number of pending submissions for the current user.
// Used by the navbar to lazily load the count via HTMX.
func (h *UserLinkHandler) PendingCount(c fiber.Ctx) error {
	user := c.Locals("user").(*models.User)
	links, err := h.db.GetPendingLinksByUser(c.Context(), user.ID)
	if err != nil || len(links) == 0 {
		return c.SendString("")
	}
	return c.SendString(fmt.Sprintf(
		`<span class="ml-1 inline-flex items-center justify-center min-w-[1rem] h-4 px-1 text-xs font-bold rounded-full bg-amber-500 text-white">%d</span>`,
		len(links),
	))
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

// Edit renders the inline edit form for a personal link.
func (h *UserLinkHandler) Edit(c fiber.Ctx) error {
	user := c.Locals("user").(*models.User)

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid link ID")
	}

	link, err := h.db.GetUserLinkByID(c.Context(), id, user.ID)
	if err != nil {
		if errors.Is(err, db.ErrUserLinkNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Link not found")
		}
		return err
	}

	return c.Render("partials/user_link_edit_form", fiber.Map{
		"Link": link,
		"User": user,
	}, "")
}

// Update saves changes to a personal link.
func (h *UserLinkHandler) Update(c fiber.Ctx) error {
	user := c.Locals("user").(*models.User)

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid link ID")
	}

	link, err := h.db.GetUserLinkByID(c.Context(), id, user.ID)
	if err != nil {
		if errors.Is(err, db.ErrUserLinkNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Link not found")
		}
		return err
	}

	newURL := c.FormValue("url")
	newDescription := c.FormValue("description")

	if newURL == "" {
		return htmxError(c, "URL is required")
	}

	if valid, msg := validation.ValidateURL(newURL); !valid {
		return htmxError(c, msg)
	}

	link.URL = newURL
	link.Description = newDescription

	if err := h.db.UpdateUserLink(c.Context(), link); err != nil {
		return err
	}

	return c.Render("partials/user_link_card", fiber.Map{
		"Link": link,
		"User": user,
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
