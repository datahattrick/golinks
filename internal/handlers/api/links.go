package api

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"golinks/internal/config"
	"golinks/internal/db"
	"golinks/internal/email"
	"golinks/internal/models"
	"golinks/internal/validation"
)

// LinkHandler handles link CRUD operations via JSON API.
type LinkHandler struct {
	db       *db.DB
	cfg      *config.Config
	notifier *email.Notifier
}

// NewLinkHandler creates a new API link handler.
func NewLinkHandler(database *db.DB, cfg *config.Config, notifier *email.Notifier) *LinkHandler {
	return &LinkHandler{db: database, cfg: cfg, notifier: notifier}
}

// List returns links, optionally filtered by search query.
func (h *LinkHandler) List(c fiber.Ctx) error {
	user, _ := c.Locals("user").(*models.User)

	var orgID *uuid.UUID
	if user != nil && user.OrganizationID != nil {
		orgID = user.OrganizationID
	}

	query := c.Query("q", "")
	links, err := h.db.SearchApprovedLinks(c.Context(), query, orgID, 100)
	if err != nil {
		return jsonError(c, fiber.StatusInternalServerError, "failed to fetch links")
	}

	return jsonSuccess(c, links)
}

// Get returns a single link by ID.
func (h *LinkHandler) Get(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return jsonError(c, fiber.StatusBadRequest, "invalid link id")
	}

	link, err := h.db.GetLinkByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, db.ErrLinkNotFound) {
			return jsonError(c, fiber.StatusNotFound, "link not found")
		}
		return jsonError(c, fiber.StatusInternalServerError, "failed to fetch link")
	}

	return jsonSuccess(c, link)
}

// Create creates a new link.
func (h *LinkHandler) Create(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return jsonError(c, fiber.StatusUnauthorized, "unauthorized")
	}

	var body struct {
		Keyword     string `json:"keyword"`
		URL         string `json:"url"`
		Description string `json:"description"`
		Scope       string `json:"scope"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return jsonError(c, fiber.StatusBadRequest, "invalid request body")
	}

	body.Keyword = validation.NormalizeKeyword(body.Keyword)

	if body.Keyword == "" || body.URL == "" {
		return jsonError(c, fiber.StatusBadRequest, "keyword and url are required")
	}

	if !validation.ValidateKeyword(body.Keyword) {
		return jsonError(c, fiber.StatusBadRequest, "keyword must contain only letters, numbers, hyphens, and underscores")
	}

	if body.Keyword == "random" {
		return jsonError(c, fiber.StatusBadRequest, `the keyword "random" is reserved`)
	}

	if valid, msg := validation.ValidateURL(body.URL); !valid {
		return jsonError(c, fiber.StatusBadRequest, msg)
	}

	if body.Scope == "" {
		if h.cfg.EnablePersonalLinks {
			body.Scope = "personal"
		} else {
			body.Scope = "global"
		}
	}

	switch body.Scope {
	case "personal":
		if !h.cfg.EnablePersonalLinks {
			return jsonError(c, fiber.StatusBadRequest, "personal links are not enabled")
		}
		return h.createPersonalLink(c, user, body.Keyword, body.URL, body.Description)
	case "org":
		if !h.cfg.EnableOrgLinks {
			return jsonError(c, fiber.StatusBadRequest, "organization links are not enabled")
		}
		return h.createOrgLink(c, user, body.Keyword, body.URL, body.Description)
	case "global":
		return h.createGlobalLink(c, user, body.Keyword, body.URL, body.Description)
	default:
		return jsonError(c, fiber.StatusBadRequest, "invalid scope")
	}
}

func (h *LinkHandler) createPersonalLink(c fiber.Ctx, user *models.User, keyword, url, description string) error {
	userLink := &models.UserLink{
		UserID:      user.ID,
		Keyword:     keyword,
		URL:         url,
		Description: description,
	}

	if err := h.db.CreateUserLink(c.Context(), userLink); err != nil {
		if errors.Is(err, db.ErrDuplicateKeyword) {
			return jsonError(c, fiber.StatusConflict, "you already have a personal link with this keyword")
		}
		return jsonError(c, fiber.StatusInternalServerError, "failed to create link")
	}

	return jsonSuccess(c, fiber.Map{
		"link":    userLink,
		"pending": false,
		"message": "personal link created successfully",
	})
}

func (h *LinkHandler) createOrgLink(c fiber.Ctx, user *models.User, keyword, url, description string) error {
	var orgID *uuid.UUID

	// Admins can create org links for any organization via organization_id in body
	if user.IsAdmin() {
		var bodyMap map[string]any
		if err := json.Unmarshal(c.Body(), &bodyMap); err == nil {
			if oidStr, ok := bodyMap["organization_id"].(string); ok && oidStr != "" {
				parsed, err := uuid.Parse(oidStr)
				if err != nil {
					return jsonError(c, fiber.StatusBadRequest, "invalid organization_id")
				}
				orgID = &parsed
			}
		}
		if orgID == nil {
			if user.OrganizationID != nil {
				orgID = user.OrganizationID
			} else {
				return jsonError(c, fiber.StatusBadRequest, "organization_id is required for admins without an org")
			}
		}
	} else {
		if user.OrganizationID == nil {
			return jsonError(c, fiber.StatusBadRequest, "you must be a member of an organization to create org links")
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
				return jsonError(c, fiber.StatusConflict, "an org link with this keyword already exists")
			}
			return jsonError(c, fiber.StatusInternalServerError, "failed to create link")
		}
		return jsonSuccess(c, fiber.Map{
			"link":    link,
			"pending": false,
			"message": "organization link created successfully",
		})
	}

	link.SubmittedBy = &user.ID
	if err := h.db.SubmitLinkForApproval(c.Context(), link); err != nil {
		if errors.Is(err, db.ErrDuplicateKeyword) {
			return jsonError(c, fiber.StatusConflict, "an org link with this keyword already exists or is pending approval")
		}
		return jsonError(c, fiber.StatusInternalServerError, "failed to submit link")
	}

	if h.notifier != nil {
		go h.notifier.NotifyModeratorsLinkSubmitted(context.Background(), link, user)
	}

	return jsonSuccess(c, fiber.Map{
		"link":    link,
		"pending": true,
		"message": "organization link submitted for approval",
	})
}

func (h *LinkHandler) createGlobalLink(c fiber.Ctx, user *models.User, keyword, url, description string) error {
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
				return jsonError(c, fiber.StatusConflict, "a global link with this keyword already exists")
			}
			return jsonError(c, fiber.StatusInternalServerError, "failed to create link")
		}
		return jsonSuccess(c, fiber.Map{
			"link":    link,
			"pending": false,
			"message": "global link created successfully",
		})
	}

	link.SubmittedBy = &user.ID
	if err := h.db.SubmitLinkForApproval(c.Context(), link); err != nil {
		if errors.Is(err, db.ErrDuplicateKeyword) {
			return jsonError(c, fiber.StatusConflict, "a global link with this keyword already exists or is pending approval")
		}
		return jsonError(c, fiber.StatusInternalServerError, "failed to submit link")
	}

	if h.notifier != nil {
		go h.notifier.NotifyModeratorsLinkSubmitted(context.Background(), link, user)
	}

	return jsonSuccess(c, fiber.Map{
		"link":    link,
		"pending": true,
		"message": "global link submitted for approval",
	})
}

// Update updates a link's URL and description (moderators only).
func (h *LinkHandler) Update(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return jsonError(c, fiber.StatusUnauthorized, "unauthorized")
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return jsonError(c, fiber.StatusBadRequest, "invalid link id")
	}

	link, err := h.db.GetLinkByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, db.ErrLinkNotFound) {
			return jsonError(c, fiber.StatusNotFound, "link not found")
		}
		return jsonError(c, fiber.StatusInternalServerError, "failed to fetch link")
	}

	if !canManageLink(user, link) {
		return jsonError(c, fiber.StatusForbidden, "you do not have permission to edit this link")
	}

	var body struct {
		URL         string `json:"url"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return jsonError(c, fiber.StatusBadRequest, "invalid request body")
	}

	if body.URL == "" {
		return jsonError(c, fiber.StatusBadRequest, "url is required")
	}

	if valid, msg := validation.ValidateURL(body.URL); !valid {
		return jsonError(c, fiber.StatusBadRequest, msg)
	}

	link.URL = body.URL
	link.Description = body.Description
	if err := h.db.UpdateLinkAndResetHealth(c.Context(), link); err != nil {
		return jsonError(c, fiber.StatusInternalServerError, "failed to update link")
	}

	return jsonSuccess(c, link)
}

// Delete removes a link.
func (h *LinkHandler) Delete(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return jsonError(c, fiber.StatusUnauthorized, "unauthorized")
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return jsonError(c, fiber.StatusBadRequest, "invalid link id")
	}

	link, err := h.db.GetLinkByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, db.ErrLinkNotFound) {
			return jsonError(c, fiber.StatusNotFound, "link not found")
		}
		return jsonError(c, fiber.StatusInternalServerError, "failed to fetch link")
	}

	canDelete := user.IsAdmin() ||
		(link.Scope == models.ScopeGlobal && user.IsGlobalMod()) ||
		(link.Scope == models.ScopeOrg && link.OrganizationID != nil && user.CanModerateOrg(*link.OrganizationID)) ||
		(link.Status == models.StatusPending && link.SubmittedBy != nil && *link.SubmittedBy == user.ID)

	if !canDelete {
		return jsonError(c, fiber.StatusForbidden, "you do not have permission to delete this link")
	}

	if err := h.db.DeleteLink(c.Context(), id); err != nil {
		if errors.Is(err, db.ErrLinkNotFound) {
			return jsonError(c, fiber.StatusNotFound, "link not found")
		}
		return jsonError(c, fiber.StatusInternalServerError, "failed to delete link")
	}

	return jsonSuccess(c, fiber.Map{
		"message": "link deleted successfully",
	})
}

// CheckKeyword checks if a keyword is available for the given scope.
func (h *LinkHandler) CheckKeyword(c fiber.Ctx) error {
	keyword := validation.NormalizeKeyword(c.Params("keyword"))
	scope := c.Query("scope", "personal")

	if keyword == "" {
		return jsonError(c, fiber.StatusBadRequest, "keyword is required")
	}

	if keyword == "random" {
		return jsonSuccess(c, models.KeywordCheckResponse{
			Available:    false,
			ConflictType: "reserved",
		})
	}

	user, _ := c.Locals("user").(*models.User)

	var exists bool
	var conflictType string

	switch scope {
	case "personal":
		if user != nil {
			_, err := h.db.GetUserLinkByKeyword(c.Context(), user.ID, keyword)
			exists = err == nil
			conflictType = "personal"
		}
	case "org":
		if user != nil && user.OrganizationID != nil {
			_, err := h.db.GetApprovedOrgLinkByKeyword(c.Context(), keyword, *user.OrganizationID)
			exists = err == nil
			conflictType = "organization"
		}
	case "global":
		_, err := h.db.GetApprovedGlobalLinkByKeyword(c.Context(), keyword)
		exists = err == nil
		conflictType = "global"
	}

	resp := models.KeywordCheckResponse{Available: !exists}
	if exists {
		resp.ConflictType = conflictType
	}
	return jsonSuccess(c, resp)
}

// canManageLink checks if a user can manage a specific link.
func canManageLink(user *models.User, link *models.Link) bool {
	if user.IsAdmin() {
		return true
	}
	if user.IsGlobalMod() {
		return true
	}
	if link.Scope == models.ScopeOrg && link.OrganizationID != nil {
		return user.CanModerateOrg(*link.OrganizationID)
	}
	return false
}

