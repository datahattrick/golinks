package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"golinks/internal/config"
	"golinks/internal/db"
	"golinks/internal/models"
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

// Index renders the home page with search box.
func (h *LinkHandler) Index(c fiber.Ctx) error {
	user, _ := c.Locals("user").(*models.User)

	data := MergeBranding(fiber.Map{
		"User":                 user,
		"EnableRandomKeywords": h.cfg.EnableRandomKeywords,
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
	return c.Render("new", MergeBranding(fiber.Map{
		"User": c.Locals("user"),
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
		return c.Status(fiber.StatusBadRequest).SendString(
			`<div class="p-3 rounded-lg bg-red-50 dark:bg-red-900/30 text-red-700 dark:text-red-300 text-sm">Keyword and URL are required</div>`,
		)
	}

	// Default to personal scope if not specified
	if scope == "" {
		scope = "personal"
	}

	switch scope {
	case "personal":
		return h.createPersonalLink(c, user, keyword, url, description)
	case "org":
		return h.createOrgLink(c, user, keyword, url, description)
	case "global":
		return h.createGlobalLink(c, user, keyword, url, description)
	default:
		return c.Status(fiber.StatusBadRequest).SendString(
			`<div class="p-3 rounded-lg bg-red-50 dark:bg-red-900/30 text-red-700 dark:text-red-300 text-sm">Invalid scope</div>`,
		)
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
			return c.Status(fiber.StatusConflict).SendString(
				`<div class="p-3 rounded-lg bg-red-50 dark:bg-red-900/30 text-red-700 dark:text-red-300 text-sm">You already have a personal link with this keyword</div>`,
			)
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
		return c.Status(fiber.StatusBadRequest).SendString(
			`<div class="p-3 rounded-lg bg-red-50 dark:bg-red-900/30 text-red-700 dark:text-red-300 text-sm">You must be a member of an organization to create org links</div>`,
		)
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
				return c.Status(fiber.StatusConflict).SendString(
					`<div class="p-3 rounded-lg bg-red-50 dark:bg-red-900/30 text-red-700 dark:text-red-300 text-sm">An org link with this keyword already exists</div>`,
				)
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
			return c.Status(fiber.StatusConflict).SendString(
				`<div class="p-3 rounded-lg bg-red-50 dark:bg-red-900/30 text-red-700 dark:text-red-300 text-sm">An org link with this keyword already exists or is pending approval</div>`,
			)
		}
		return err
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
				return c.Status(fiber.StatusConflict).SendString(
					`<div class="p-3 rounded-lg bg-red-50 dark:bg-red-900/30 text-red-700 dark:text-red-300 text-sm">A global link with this keyword already exists</div>`,
				)
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
			return c.Status(fiber.StatusConflict).SendString(
				`<div class="p-3 rounded-lg bg-red-50 dark:bg-red-900/30 text-red-700 dark:text-red-300 text-sm">A global link with this keyword already exists or is pending approval</div>`,
			)
		}
		return err
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
