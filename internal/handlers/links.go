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

// LinkHandler handles link CRUD operations.
type LinkHandler struct {
	db  *db.DB
	cfg *config.Config
}

// NewLinkHandler creates a new link handler.
func NewLinkHandler(database *db.DB, cfg *config.Config) *LinkHandler {
	return &LinkHandler{db: database, cfg: cfg}
}

// htmxError returns an error message as HTML that HTMX will display.
// Uses 200 status so HTMX processes the swap (HTMX ignores non-2xx by default).
func htmxError(c fiber.Ctx, message string) error {
	return c.SendString(
		`<div class="p-3 rounded-lg bg-red-50 dark:bg-red-900/30 text-red-700 dark:text-red-300 text-sm">` + message + `</div>`,
	)
}

// Index renders the home page with search box.
func (h *LinkHandler) Index(c fiber.Ctx) error {
	user, _ := c.Locals("user").(*models.User)

	data := MergeBranding(fiber.Map{
		"User":                 user,
		"EnableRandomKeywords": h.cfg.EnableRandomKeywords,
		"EnablePersonalLinks":  h.cfg.EnablePersonalLinks,
		"EnableOrgLinks":       h.cfg.EnableOrgLinks,
		"IsSimpleMode":         h.cfg.IsSimpleMode(),
	}, h.cfg)

	// Fetch top used, newest, and random keywords
	var orgID *uuid.UUID
	if user != nil && user.OrganizationID != nil {
		orgID = user.OrganizationID
	}

	// Top used links (global/org only, no personal)
	topUsed, err := h.db.GetTopApprovedLinks(c.Context(), orgID, 5)
	if err == nil {
		data["TopUsedLinks"] = topUsed
	}

	// Newest links
	newestLinks, err := h.db.GetNewestApprovedLinks(c.Context(), orgID, 5)
	if err == nil {
		data["NewestLinks"] = newestLinks
	}

	// Random links (always fetch for display, config only controls the /random API endpoint)
	randomLinks, err := h.db.GetRandomApprovedLinks(c.Context(), orgID, 5)
	if err == nil {
		data["RandomLinks"] = randomLinks
	}

	return c.Render("index", data)
}

// Search renders the search results page.
func (h *LinkHandler) Search(c fiber.Ctx) error {
	query := c.Query("q", "")
	user, _ := c.Locals("user").(*models.User)

	// Get the user's org ID for search filtering
	var orgID *uuid.UUID
	if user != nil && user.OrganizationID != nil {
		orgID = user.OrganizationID
	}

	links, err := h.db.SearchApprovedLinks(c.Context(), query, orgID, 50)
	if err != nil {
		return err
	}

	return c.Render("search", MergeBranding(fiber.Map{
		"Links": links,
		"Query": query,
		"User":  user,
	}, h.cfg))
}

// Suggest returns autocomplete suggestions for HTMX.
func (h *LinkHandler) Suggest(c fiber.Ctx) error {
	query := c.Query("q", "")
	if query == "" {
		return c.SendString("")
	}

	user, _ := c.Locals("user").(*models.User)
	var orgID *uuid.UUID
	if user != nil && user.OrganizationID != nil {
		orgID = user.OrganizationID
	}

	links, err := h.db.SearchApprovedLinks(c.Context(), query, orgID, 5)
	if err != nil {
		return err
	}

	return c.Render("partials/suggestions", fiber.Map{
		"Links": links,
		"Query": query,
	}, "")
}

// Browse renders the browse all links page.
func (h *LinkHandler) Browse(c fiber.Ctx) error {
	query := c.Query("q", "")
	user, _ := c.Locals("user").(*models.User)

	var orgID *uuid.UUID
	if user != nil && user.OrganizationID != nil {
		orgID = user.OrganizationID
	}

	links, err := h.db.SearchApprovedLinks(c.Context(), query, orgID, 100)
	if err != nil {
		return err
	}

	// If HTMX request, return just the list
	if c.Get("HX-Request") == "true" {
		return c.Render("partials/links_list", fiber.Map{
			"Links": links,
			"User":  user,
		}, "")
	}

	return c.Render("browse", MergeBranding(fiber.Map{
		"Links": links,
		"User":  user,
	}, h.cfg))
}

// New renders the create link form.
func (h *LinkHandler) New(c fiber.Ctx) error {
	user, _ := c.Locals("user").(*models.User)

	var orgName string
	if user != nil && user.OrganizationID != nil {
		if org, err := h.db.GetOrganizationByID(c.Context(), *user.OrganizationID); err == nil {
			orgName = org.Name
		}
	}

	return c.Render("new", MergeBranding(fiber.Map{
		"User":               user,
		"OrgName":            orgName,
		"EnablePersonalLinks": h.cfg.EnablePersonalLinks,
		"EnableOrgLinks":     h.cfg.EnableOrgLinks,
	}, h.cfg))
}

// Create handles creating a new link based on scope.
func (h *LinkHandler) Create(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	keyword := c.FormValue("keyword")
	url := c.FormValue("url")
	description := c.FormValue("description")
	scope := c.FormValue("scope")

	if keyword == "" || url == "" {
		return htmxError(c, "Keyword and URL are required")
	}

	// Validate keyword format (alphanumeric, hyphens, underscores only)
	if !validation.ValidateKeyword(keyword) {
		return htmxError(c, "Keyword must contain only letters, numbers, hyphens, and underscores")
	}

	if keyword == "random" {
		return htmxError(c, `The keyword "random" is reserved and cannot be used`)
	}

	// Validate URL scheme (http/https only, prevents javascript: XSS)
	if valid, msg := validation.ValidateURL(url); !valid {
		return htmxError(c, msg)
	}

	// Default scope based on what's enabled
	if scope == "" {
		if h.cfg.EnablePersonalLinks {
			scope = "personal"
		} else {
			scope = "global"
		}
	}

	switch scope {
	case "personal":
		if !h.cfg.EnablePersonalLinks {
			return htmxError(c, "Personal links are not enabled")
		}
		return h.createPersonalLink(c, user, keyword, url, description)
	case "org":
		if !h.cfg.EnableOrgLinks {
			return htmxError(c, "Organization links are not enabled")
		}
		return h.createOrgLink(c, user, keyword, url, description)
	case "global":
		return h.createGlobalLink(c, user, keyword, url, description)
	default:
		return htmxError(c, "Invalid scope")
	}
}

// createPersonalLink creates a personal link (user_links table).
func (h *LinkHandler) createPersonalLink(c fiber.Ctx, user *models.User, keyword, url, description string) error {
	userLink := &models.UserLink{
		UserID:      user.ID,
		Keyword:     keyword,
		URL:         url,
		Description: description,
	}

	if err := h.db.CreateUserLink(c.Context(), userLink); err != nil {
		if errors.Is(err, db.ErrDuplicateKeyword) {
			return htmxError(c, "You already have a personal link with this keyword")
		}
		return err
	}

	return c.Render("partials/form_success", fiber.Map{
		"Keyword": keyword,
		"Message": "Personal link created successfully!",
	}, "")
}

// createOrgLink creates an org-scoped link.
func (h *LinkHandler) createOrgLink(c fiber.Ctx, user *models.User, keyword, url, description string) error {
	if user.OrganizationID == nil {
		return htmxError(c, "You must be a member of an organization to create org links")
	}

	link := &models.Link{
		Keyword:        keyword,
		URL:            url,
		Description:    description,
		Scope:          models.ScopeOrg,
		OrganizationID: user.OrganizationID,
	}

	// Org mods can create links directly, others need approval
	if user.CanModerateOrg(*user.OrganizationID) {
		link.CreatedBy = &user.ID
		link.Status = models.StatusApproved
		if err := h.db.CreateLink(c.Context(), link); err != nil {
			if errors.Is(err, db.ErrDuplicateKeyword) {
				return htmxError(c, "An org link with this keyword already exists")
			}
			return err
		}
		return c.Render("partials/form_success", fiber.Map{
			"Keyword": keyword,
			"Message": "Organization link created successfully!",
		}, "")
	}

	// Regular users submit for approval
	link.SubmittedBy = &user.ID
	if err := h.db.SubmitLinkForApproval(c.Context(), link); err != nil {
		if errors.Is(err, db.ErrDuplicateKeyword) {
			return htmxError(c, "An org link with this keyword already exists or is pending approval")
		}
		return err
	}

	// Send email notification to moderators
	if Notifier != nil {
		go Notifier.NotifyLinkSubmitted(c.Context(), link, user)
	}

	return c.Render("partials/form_success", fiber.Map{
		"Keyword": keyword,
		"Message": "Organization link submitted for approval. A moderator will review it shortly.",
		"Pending": true,
	}, "")
}

// createGlobalLink creates a global-scoped link.
func (h *LinkHandler) createGlobalLink(c fiber.Ctx, user *models.User, keyword, url, description string) error {
	link := &models.Link{
		Keyword:     keyword,
		URL:         url,
		Description: description,
		Scope:       models.ScopeGlobal,
	}

	// Global mods can create links directly, others need approval
	if user.IsGlobalMod() {
		link.CreatedBy = &user.ID
		link.Status = models.StatusApproved
		if err := h.db.CreateLink(c.Context(), link); err != nil {
			if errors.Is(err, db.ErrDuplicateKeyword) {
				return htmxError(c, "A global link with this keyword already exists")
			}
			return err
		}
		return c.Render("partials/form_success", fiber.Map{
			"Keyword": keyword,
			"Message": "Global link created successfully!",
		}, "")
	}

	// Regular users submit for approval
	link.SubmittedBy = &user.ID
	if err := h.db.SubmitLinkForApproval(c.Context(), link); err != nil {
		if errors.Is(err, db.ErrDuplicateKeyword) {
			return htmxError(c, "A global link with this keyword already exists or is pending approval")
		}
		return err
	}

	// Send email notification to moderators
	if Notifier != nil {
		go Notifier.NotifyLinkSubmitted(c.Context(), link, user)
	}

	return c.Render("partials/form_success", fiber.Map{
		"Keyword": keyword,
		"Message": "Global link submitted for approval. A moderator will review it shortly.",
		"Pending": true,
	}, "")
}

// Delete handles deleting a link via HTMX.
func (h *LinkHandler) Delete(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	// Get the link to check permissions
	link, err := h.db.GetLinkByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, db.ErrLinkNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "link not found")
		}
		return err
	}

	// Check permissions: admins can delete anything, moderators can delete their scope, users can delete their pending submissions
	canDelete := false
	if user.IsAdmin() {
		canDelete = true
	} else if link.Scope == models.ScopeGlobal && user.IsGlobalMod() {
		canDelete = true
	} else if link.Scope == models.ScopeOrg && link.OrganizationID != nil && user.CanModerateOrg(*link.OrganizationID) {
		canDelete = true
	} else if link.Status == models.StatusPending && link.SubmittedBy != nil && *link.SubmittedBy == user.ID {
		// Users can delete their own pending submissions
		canDelete = true
	}

	if !canDelete {
		return fiber.NewError(fiber.StatusForbidden, "you do not have permission to delete this link")
	}

	if err := h.db.DeleteLink(c.Context(), id); err != nil {
		if errors.Is(err, db.ErrLinkNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "link not found")
		}
		return err
	}

	// Return empty response for HTMX to remove the element
	return c.SendString("")
}

// CheckKeyword checks if a keyword already exists for the given scope.
// Returns HTML for HTMX to display conflict warnings.
func (h *LinkHandler) CheckKeyword(c fiber.Ctx) error {
	keyword := c.Query("keyword")
	scope := c.Query("scope", "personal")

	if keyword == "" {
		return c.SendString("")
	}

	// Check for reserved keywords
	if keyword == "random" {
		return c.SendString(`<div class="flex items-center gap-2 p-2 rounded-lg bg-red-50 dark:bg-red-900/30 text-red-700 dark:text-red-300 text-sm mt-1">
			<svg class="w-4 h-4 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/>
			</svg>
			<span>The keyword "random" is reserved and cannot be used</span>
		</div>`)
	}

	user, _ := c.Locals("user").(*models.User)

	var exists bool
	var conflictType string

	switch scope {
	case "personal":
		// Check user's personal links
		if user != nil {
			_, err := h.db.GetUserLinkByKeyword(c.Context(), user.ID, keyword)
			exists = err == nil
			conflictType = "personal"
		}
	case "org":
		// Check org links
		if user != nil && user.OrganizationID != nil {
			_, err := h.db.GetApprovedOrgLinkByKeyword(c.Context(), keyword, *user.OrganizationID)
			exists = err == nil
			conflictType = "organization"
		}
	case "global":
		// Check global links
		_, err := h.db.GetApprovedGlobalLinkByKeyword(c.Context(), keyword)
		exists = err == nil
		conflictType = "global"
	}

	if exists {
		return c.SendString(`<div class="flex items-center gap-2 p-2 rounded-lg bg-amber-50 dark:bg-amber-900/30 text-amber-700 dark:text-amber-300 text-sm mt-1">
			<svg class="w-4 h-4 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/>
			</svg>
			<span>A ` + conflictType + ` link with this keyword already exists</span>
		</div>`)
	}

	return c.SendString("")
}
