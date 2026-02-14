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

// SharedLinkHandler handles personal link sharing between users.
type SharedLinkHandler struct {
	db  *db.DB
	cfg *config.Config
}

// NewSharedLinkHandler creates a new shared link handler.
func NewSharedLinkHandler(database *db.DB, cfg *config.Config) *SharedLinkHandler {
	return &SharedLinkHandler{db: database, cfg: cfg}
}

// SearchUsers returns an HTML partial of matching users for autocomplete.
func (h *SharedLinkHandler) SearchUsers(c fiber.Ctx) error {
	user := c.Locals("user").(*models.User)
	query := c.Query("q")

	if len(query) < 2 {
		return c.SendString("")
	}

	users, err := h.db.SearchUsers(c.Context(), query, user.ID, 5)
	if err != nil {
		return err
	}

	return c.Render("partials/user_suggestions", fiber.Map{
		"Users": users,
	}, "")
}

// Create creates shared link offers to one or more recipients.
func (h *SharedLinkHandler) Create(c fiber.Ctx) error {
	user := c.Locals("user").(*models.User)

	keyword := c.FormValue("keyword")
	url := c.FormValue("url")
	description := c.FormValue("description")

	// Collect all recipient_id values from the form
	var recipientIDs []string
	for _, v := range c.Request().PostArgs().PeekMulti("recipient_id") {
		if len(v) > 0 {
			recipientIDs = append(recipientIDs, string(v))
		}
	}

	if len(recipientIDs) == 0 || keyword == "" || url == "" {
		return htmxError(c, "At least one recipient, keyword, and URL are required")
	}

	if !validation.ValidateKeyword(keyword) {
		return htmxError(c, "Invalid keyword: "+keyword)
	}

	if valid, msg := validation.ValidateURL(url); !valid {
		return htmxError(c, msg)
	}

	var errMsgs []string
	for _, ridStr := range recipientIDs {
		recipientID, err := uuid.Parse(ridStr)
		if err != nil {
			errMsgs = append(errMsgs, "invalid recipient ID")
			continue
		}

		if recipientID == user.ID {
			errMsgs = append(errMsgs, "cannot share with yourself")
			continue
		}

		// Check if the recipient already has a personal link with this keyword
		if _, err := h.db.GetUserLinkByKeyword(c.Context(), recipientID, keyword); err == nil {
			// Recipient already has this keyword — look up their name for the error
			recipient, _ := h.db.GetUserByID(c.Context(), recipientID)
			name := ridStr
			if recipient != nil {
				name = recipient.Name
				if name == "" {
					name = recipient.Username
				}
				if name == "" {
					name = recipient.Sub
				}
			}
			errMsgs = append(errMsgs, fmt.Sprintf("%s already has keyword '%s'", name, keyword))
			continue
		}

		link := &models.SharedLink{
			SenderID:    user.ID,
			RecipientID: recipientID,
			Keyword:     keyword,
			URL:         url,
			Description: description,
		}

		if err := h.db.CreateSharedLink(c.Context(), link); err != nil {
			if errors.Is(err, db.ErrShareLimitReached) ||
				errors.Is(err, db.ErrRecipientLimitReached) ||
				errors.Is(err, db.ErrDuplicateShare) {
				errMsgs = append(errMsgs, err.Error())
			} else {
				return err
			}
		}
	}

	// If all recipients failed, show the errors
	if len(errMsgs) == len(recipientIDs) {
		return htmxError(c, strings.Join(errMsgs, "; "))
	}

	// Return updated outgoing shares list
	shares, err := h.db.GetOutgoingShares(c.Context(), user.ID)
	if err != nil {
		return err
	}

	return c.Render("partials/outgoing_shares_list", fiber.Map{
		"OutgoingShares": shares,
	}, "")
}

// Accept accepts a shared link, copying it into the recipient's personal links.
func (h *SharedLinkHandler) Accept(c fiber.Ctx) error {
	user := c.Locals("user").(*models.User)

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid share ID")
	}

	share, err := h.db.GetSharedLinkByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, db.ErrSharedLinkNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Share not found")
		}
		return err
	}

	if share.RecipientID != user.ID {
		return fiber.NewError(fiber.StatusForbidden, "Not authorized")
	}

	// Create the personal link for the recipient
	userLink := &models.UserLink{
		UserID:      user.ID,
		Keyword:     share.Keyword,
		URL:         share.URL,
		Description: share.Description,
	}

	if err := h.db.CreateUserLink(c.Context(), userLink); err != nil {
		if errors.Is(err, db.ErrDuplicateKeyword) {
			return htmxError(c, "You already have a personal link with keyword '"+share.Keyword+"'")
		}
		return err
	}

	// Delete the share after successful accept
	if err := h.db.DeleteSharedLink(c.Context(), id); err != nil {
		return err
	}

	return h.renderAcceptDeclineResponse(c, user.ID)
}

// Decline removes a shared link (recipient action).
func (h *SharedLinkHandler) Decline(c fiber.Ctx) error {
	user := c.Locals("user").(*models.User)

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid share ID")
	}

	share, err := h.db.GetSharedLinkByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, db.ErrSharedLinkNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Share not found")
		}
		return err
	}

	if share.RecipientID != user.ID {
		return fiber.NewError(fiber.StatusForbidden, "Not authorized")
	}

	if err := h.db.DeleteSharedLink(c.Context(), id); err != nil {
		return err
	}

	return h.renderAcceptDeclineResponse(c, user.ID)
}

// Withdraw removes a shared link (sender action).
func (h *SharedLinkHandler) Withdraw(c fiber.Ctx) error {
	user := c.Locals("user").(*models.User)

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid share ID")
	}

	share, err := h.db.GetSharedLinkByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, db.ErrSharedLinkNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Share not found")
		}
		return err
	}

	if share.SenderID != user.ID {
		return fiber.NewError(fiber.StatusForbidden, "Not authorized")
	}

	if err := h.db.DeleteSharedLink(c.Context(), id); err != nil {
		return err
	}

	return c.SendString("")
}

// renderAcceptDeclineResponse returns OOB swaps to update both the incoming
// shares section and the personal links list after an accept or decline.
func (h *SharedLinkHandler) renderAcceptDeclineResponse(c fiber.Ctx, userID uuid.UUID) error {
	var html strings.Builder

	// Render updated incoming shares section (or empty if none remain)
	incomingShares, err := h.db.GetIncomingShares(c.Context(), userID)
	if err != nil {
		return err
	}

	if len(incomingShares) > 0 {
		// Render the shares list partial
		var sharesBuf strings.Builder
		if err := c.App().Config().Views.Render(&sharesBuf, "partials/incoming_shares_list", fiber.Map{
			"IncomingShares": incomingShares,
		}); err != nil {
			return err
		}
		html.WriteString(fmt.Sprintf(`<div id="incoming-shares-section" hx-swap-oob="innerHTML">`+
			`<div class="mb-8 p-5 rounded-2xl bg-blue-50/50 dark:bg-blue-900/10 border border-blue-200 dark:border-blue-800/30">`+
			`<h2 class="text-sm font-semibold mb-3 flex items-center gap-2 text-blue-600 dark:text-blue-400">`+
			`<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4"/></svg>`+
			`Shared With You `+
			`<span class="text-xs bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 px-2 py-0.5 rounded-full">%d</span>`+
			`</h2>`+
			`<div class="space-y-2" id="incoming-shares-list">%s</div>`+
			`</div></div>`, len(incomingShares), sharesBuf.String()))
	} else {
		// Empty — remove the section entirely
		html.WriteString(`<div id="incoming-shares-section" hx-swap-oob="innerHTML"></div>`)
	}

	// Render updated personal links list
	links, err := h.db.GetUserLinks(c.Context(), userID)
	if err != nil {
		return err
	}

	var linksBuf strings.Builder
	if err := c.App().Config().Views.Render(&linksBuf, "partials/user_links_list", fiber.Map{
		"UserLinks": links,
	}); err != nil {
		return err
	}
	html.WriteString(`<div id="user-links-list" hx-swap-oob="innerHTML">` + linksBuf.String() + `</div>`)

	return c.SendString(html.String())
}
