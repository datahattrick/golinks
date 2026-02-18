package handlers

import (
	"errors"
	"html"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"golinks/internal/config"
	"golinks/internal/db"
	"golinks/internal/models"
	"golinks/internal/validation"
)

// linkWithSparkline wraps a Link with pre-computed sparkline data for the home page.
type linkWithSparkline struct {
	models.Link
	SparklineData string // comma-separated 24 hourly click counts
}

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
		`<div class="p-3 rounded-lg bg-red-50 dark:bg-red-900/30 text-red-700 dark:text-red-300 text-sm">` + html.EscapeString(message) + `</div>`,
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

	// Top used links (global/org only, no personal) with 24h sparkline data
	topUsed, err := h.db.GetTopApprovedLinks(c.Context(), orgID, 5)
	if err == nil {
		sparklines := make([]linkWithSparkline, len(topUsed))
		for i, link := range topUsed {
			history, hErr := h.db.GetClickHistory24h(c.Context(), link.ID)
			if hErr != nil || len(history) == 0 {
				history = make([]int, 24)
			}
			parts := make([]string, len(history))
			for j, v := range history {
				parts[j] = strconv.Itoa(v)
			}
			sparklines[i] = linkWithSparkline{Link: link, SparklineData: strings.Join(parts, ",")}
		}
		data["TopUsedLinks"] = sparklines
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

	// Build org name map for displaying org names on links
	orgNames := make(map[string]string)
	if orgs, err := h.db.GetAllOrganizations(c.Context()); err == nil {
		for _, org := range orgs {
			orgNames[org.ID.String()] = org.Name
		}
	}

	// If HTMX request, return just the list
	if c.Get("HX-Request") == "true" {
		return c.Render("partials/links_list", fiber.Map{
			"Links":    links,
			"User":     user,
			"OrgNames": orgNames,
		}, "")
	}

	return c.Render("browse", MergeBranding(fiber.Map{
		"Links":    links,
		"User":     user,
		"OrgNames": orgNames,
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

	data := fiber.Map{
		"User":                user,
		"OrgName":             orgName,
		"EnablePersonalLinks": h.cfg.EnablePersonalLinks,
		"EnableOrgLinks":      h.cfg.EnableOrgLinks,
	}

	// Admins can create org links for any organization
	if user != nil && user.IsAdmin() && h.cfg.EnableOrgLinks {
		if allOrgs, err := h.db.GetAllOrganizations(c.Context()); err == nil {
			data["AllOrgs"] = allOrgs
		}
	}

	return c.Render("new", MergeBranding(data, h.cfg))
}

// splitKeywords splits a comma-separated keyword string into normalized, unique keywords.
func splitKeywords(raw string) []string {
	parts := strings.Split(raw, ",")
	seen := make(map[string]bool, len(parts))
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		kw := validation.NormalizeKeyword(strings.TrimSpace(p))
		if kw != "" && !seen[kw] {
			seen[kw] = true
			result = append(result, kw)
		}
	}
	return result
}

// Create handles creating a new link based on scope.
// Supports comma-separated keywords to create multiple links for the same URL.
func (h *LinkHandler) Create(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	rawKeywords := c.FormValue("keyword")
	url := c.FormValue("url")
	description := c.FormValue("description")
	scope := c.FormValue("scope")

	if rawKeywords == "" || url == "" {
		return htmxError(c, "Keyword and URL are required")
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

	// Split comma-separated keywords
	keywords := splitKeywords(rawKeywords)
	if len(keywords) == 0 {
		return htmxError(c, "At least one keyword is required")
	}

	// Validate all keywords first
	for _, kw := range keywords {
		if !validation.ValidateKeyword(kw) {
			return htmxError(c, "Invalid keyword: "+kw)
		}
		if kw == "random" {
			return htmxError(c, `The keyword "random" is reserved and cannot be used`)
		}
	}

	// Single keyword — use the original direct path (renders response directly)
	if len(keywords) == 1 {
		switch scope {
		case "personal":
			if !h.cfg.EnablePersonalLinks {
				return htmxError(c, "Personal links are not enabled")
			}
			return h.createPersonalLink(c, user, keywords[0], url, description)
		case "org":
			if !h.cfg.EnableOrgLinks {
				return htmxError(c, "Organization links are not enabled")
			}
			return h.createOrgLink(c, user, keywords[0], url, description)
		case "global":
			return h.createGlobalLink(c, user, keywords[0], url, description)
		default:
			return htmxError(c, "Invalid scope")
		}
	}

	// Multiple keywords — create each using DB-only helpers, collect results
	var created []string
	var errMsgs []string
	for _, kw := range keywords {
		if errMsg := h.saveLinkForKeyword(c, user, kw, url, description, scope); errMsg != "" {
			errMsgs = append(errMsgs, kw+": "+errMsg)
		} else {
			created = append(created, kw)
		}
	}

	// Build combined result message
	var msg string
	if len(created) > 0 {
		msg = "Created links: " + strings.Join(created, ", ")
	}
	if len(errMsgs) > 0 {
		if msg != "" {
			msg += ". "
		}
		msg += "Errors: " + strings.Join(errMsgs, "; ")
	}

	if len(created) > 0 {
		return c.Render("partials/form_success", fiber.Map{
			"Keywords": created,
			"Message":  msg,
		}, "")
	}
	return htmxError(c, msg)
}

// saveLinkForKeyword performs the DB work for creating a link without rendering a response.
// Returns an error message string (empty on success).
func (h *LinkHandler) saveLinkForKeyword(c fiber.Ctx, user *models.User, keyword, url, description, scope string) string {
	switch scope {
	case "personal":
		if !h.cfg.EnablePersonalLinks {
			return "personal links are not enabled"
		}
		userLink := &models.UserLink{
			UserID:      user.ID,
			Keyword:     keyword,
			URL:         url,
			Description: description,
		}
		if err := h.db.CreateUserLink(c.Context(), userLink); err != nil {
			if errors.Is(err, db.ErrDuplicateKeyword) {
				return "duplicate keyword"
			}
			return err.Error()
		}
		return ""
	case "org":
		if !h.cfg.EnableOrgLinks {
			return "organization links are not enabled"
		}
		var orgID *uuid.UUID
		if user.IsAdmin() {
			orgIDStr := c.FormValue("organization_id")
			if orgIDStr != "" {
				parsed, err := uuid.Parse(orgIDStr)
				if err != nil {
					return "invalid organization ID"
				}
				orgID = &parsed
			} else if user.OrganizationID != nil {
				orgID = user.OrganizationID
			} else {
				return "organization required"
			}
		} else {
			if user.OrganizationID == nil {
				return "you must be a member of an organization"
			}
			orgID = user.OrganizationID
		}
		link := &models.Link{
			Keyword:        keyword,
			URL:            url,
			Description:    description,
			Scope:          models.ScopeOrg,
			OrganizationID: orgID,
		}
		if user.IsAdmin() || user.CanModerateOrg(*orgID) {
			link.CreatedBy = &user.ID
			link.Status = models.StatusApproved
			if err := h.db.CreateLink(c.Context(), link); err != nil {
				if errors.Is(err, db.ErrDuplicateKeyword) {
					return "duplicate keyword"
				}
				return err.Error()
			}
		} else {
			link.SubmittedBy = &user.ID
			if err := h.db.SubmitLinkForApproval(c.Context(), link); err != nil {
				if errors.Is(err, db.ErrDuplicateKeyword) {
					return "duplicate keyword"
				}
				return err.Error()
			}
			if Notifier != nil {
				go Notifier.NotifyModeratorsLinkSubmitted(c.Context(), link, user)
			}
		}
		return ""
	case "global":
		link := &models.Link{
			Keyword:     keyword,
			URL:         url,
			Description: description,
			Scope:       models.ScopeGlobal,
		}
		if user.IsGlobalMod() {
			link.CreatedBy = &user.ID
			link.Status = models.StatusApproved
			if err := h.db.CreateLink(c.Context(), link); err != nil {
				if errors.Is(err, db.ErrDuplicateKeyword) {
					return "duplicate keyword"
				}
				return err.Error()
			}
		} else {
			link.SubmittedBy = &user.ID
			if err := h.db.SubmitLinkForApproval(c.Context(), link); err != nil {
				if errors.Is(err, db.ErrDuplicateKeyword) {
					return "duplicate keyword"
				}
				return err.Error()
			}
			if Notifier != nil {
				go Notifier.NotifyModeratorsLinkSubmitted(c.Context(), link, user)
			}
		}
		return ""
	default:
		return "invalid scope"
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
	var orgID *uuid.UUID

	// Admins can create org links for any organization
	if user.IsAdmin() {
		orgIDStr := c.FormValue("organization_id")
		if orgIDStr != "" {
			parsed, err := uuid.Parse(orgIDStr)
			if err != nil {
				return htmxError(c, "Invalid organization ID")
			}
			orgID = &parsed
		} else if user.OrganizationID != nil {
			orgID = user.OrganizationID
		} else {
			return htmxError(c, "Please select an organization")
		}
	} else {
		if user.OrganizationID == nil {
			return htmxError(c, "You must be a member of an organization to create org links")
		}
		orgID = user.OrganizationID
	}

	link := &models.Link{
		Keyword:        keyword,
		URL:            url,
		Description:    description,
		Scope:          models.ScopeOrg,
		OrganizationID: orgID,
	}

	// Admins and org mods can create links directly, others need approval
	if user.IsAdmin() || user.CanModerateOrg(*orgID) {
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
		go Notifier.NotifyModeratorsLinkSubmitted(c.Context(), link, user)
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
		go Notifier.NotifyModeratorsLinkSubmitted(c.Context(), link, user)
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
	keyword := validation.NormalizeKeyword(c.Query("keyword"))
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
