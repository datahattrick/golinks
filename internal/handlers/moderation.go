package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"golinks/internal/config"
	"golinks/internal/db"
	"golinks/internal/email"
	"golinks/internal/models"
)

// ModerationHandler handles link moderation operations.
type ModerationHandler struct {
	db       *db.DB
	cfg      *config.Config
	notifier *email.Notifier
}

// NewModerationHandler creates a new moderation handler.
func NewModerationHandler(database *db.DB, cfg *config.Config, notifier *email.Notifier) *ModerationHandler {
	return &ModerationHandler{db: database, cfg: cfg, notifier: notifier}
}

// Index renders the moderation dashboard.
func (h *ModerationHandler) Index(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	// Check if user has moderation permissions
	if !user.IsOrgMod() {
		return fiber.NewError(fiber.StatusForbidden, "you do not have moderation permissions")
	}

	var globalPending, orgPending []models.Link
	var err error

	// Global mods and admins see all pending links (global + all orgs)
	if user.IsGlobalMod() {
		globalPending, err = h.db.GetPendingGlobalLinks(c.Context())
		if err != nil {
			return err
		}
		orgPending, err = h.db.GetAllPendingOrgLinks(c.Context())
		if err != nil {
			return err
		}
	} else if user.OrganizationID != nil {
		// Org mods only see their org's pending links
		orgPending, err = h.db.GetPendingOrgLinks(c.Context(), *user.OrganizationID)
		if err != nil {
			return err
		}
	}

	// Fetch deletion requests and edit requests
	deletionRequests, err := h.db.GetPendingDeletionRequests(c.Context(), user)
	if err != nil {
		return err
	}

	editRequests, err := h.db.GetPendingEditRequests(c.Context(), user)
	if err != nil {
		return err
	}

	// Build a map of org IDs to names for the template
	orgNames := make(map[string]string)
	if len(orgPending) > 0 || len(deletionRequests) > 0 {
		orgs, err := h.db.GetAllOrganizations(c.Context())
		if err == nil {
			for _, org := range orgs {
				orgNames[org.ID.String()] = org.Name
			}
		}
	}

	return c.Render("moderation", MergeBranding(fiber.Map{
		"User":             user,
		"GlobalPending":    globalPending,
		"OrgPending":       orgPending,
		"DeletionRequests": deletionRequests,
		"EditRequests":     editRequests,
		"OrgNames":         orgNames,
	}, h.cfg, c.Path()))
}

// Approve approves a pending link.
func (h *ModerationHandler) Approve(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	idStr := c.Params("id")
	linkID, err := uuid.Parse(idStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid link id")
	}

	// Get the link to check permissions
	link, err := h.db.GetLinkByID(c.Context(), linkID)
	if err != nil {
		if errors.Is(err, db.ErrLinkNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "link not found")
		}
		return err
	}

	// Check permissions
	if !canModerate(user, link) {
		return fiber.NewError(fiber.StatusForbidden, "you do not have permission to moderate this link")
	}

	if err := h.db.ApproveLink(c.Context(), linkID, user.ID); err != nil {
		if errors.Is(err, db.ErrLinkNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "link not found or already processed")
		}
		return err
	}

	// Send email notification to the link creator
	h.notifier.NotifyUserLinkApproved(c.Context(), link, user)

	// Return success message for HTMX
	return c.Render("partials/moderation_success", fiber.Map{
		"Action":  "approved",
		"Keyword": link.Keyword,
	}, "")
}

// Reject rejects a pending link.
func (h *ModerationHandler) Reject(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	idStr := c.Params("id")
	linkID, err := uuid.Parse(idStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid link id")
	}

	// Get the link to check permissions
	link, err := h.db.GetLinkByID(c.Context(), linkID)
	if err != nil {
		if errors.Is(err, db.ErrLinkNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "link not found")
		}
		return err
	}

	// Check permissions
	if !canModerate(user, link) {
		return fiber.NewError(fiber.StatusForbidden, "you do not have permission to moderate this link")
	}

	if err := h.db.RejectLink(c.Context(), linkID, user.ID); err != nil {
		if errors.Is(err, db.ErrLinkNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "link not found or already processed")
		}
		return err
	}

	// Send email notification to the link creator
	reason := c.FormValue("reason") // Optional rejection reason
	h.notifier.NotifyUserLinkRejected(c.Context(), link, reason)

	// Return success message for HTMX
	return c.Render("partials/moderation_success", fiber.Map{
		"Action":  "rejected",
		"Keyword": link.Keyword,
	}, "")
}

// ApproveDeletion approves a deletion request (deletes the link).
func (h *ModerationHandler) ApproveDeletion(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	idStr := c.Params("id")
	linkID, err := uuid.Parse(idStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid link id")
	}

	link, err := h.db.GetLinkByID(c.Context(), linkID)
	if err != nil {
		if errors.Is(err, db.ErrLinkNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "link not found")
		}
		return err
	}

	if !canModerate(user, link) {
		return fiber.NewError(fiber.StatusForbidden, "you do not have permission to moderate this link")
	}

	if err := h.db.ApproveDeletion(c.Context(), linkID); err != nil {
		if errors.Is(err, db.ErrLinkNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "link not found or already processed")
		}
		return err
	}

	return c.Render("partials/moderation_success", fiber.Map{
		"Action":  "deletion approved",
		"Keyword": link.Keyword,
	}, "")
}

// RejectDeletion rejects a deletion request (restores the link to approved).
func (h *ModerationHandler) RejectDeletion(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	idStr := c.Params("id")
	linkID, err := uuid.Parse(idStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid link id")
	}

	link, err := h.db.GetLinkByID(c.Context(), linkID)
	if err != nil {
		if errors.Is(err, db.ErrLinkNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "link not found")
		}
		return err
	}

	if !canModerate(user, link) {
		return fiber.NewError(fiber.StatusForbidden, "you do not have permission to moderate this link")
	}

	if err := h.db.RejectDeletion(c.Context(), linkID, user.ID); err != nil {
		if errors.Is(err, db.ErrLinkNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "link not found or already processed")
		}
		return err
	}

	return c.Render("partials/moderation_success", fiber.Map{
		"Action":  "deletion rejected",
		"Keyword": link.Keyword,
	}, "")
}

// ApproveEdit approves an edit request (applies changes to the link).
func (h *ModerationHandler) ApproveEdit(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	if !user.IsOrgMod() {
		return fiber.NewError(fiber.StatusForbidden, "you do not have moderation permissions")
	}

	idStr := c.Params("id")
	reqID, err := uuid.Parse(idStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request id")
	}

	editReq, err := h.db.GetEditRequestByID(c.Context(), reqID)
	if err != nil {
		if errors.Is(err, db.ErrEditRequestNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "edit request not found")
		}
		return err
	}

	if err := h.db.ApproveEditRequest(c.Context(), reqID, user.ID); err != nil {
		if errors.Is(err, db.ErrEditRequestNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "edit request not found or already processed")
		}
		return err
	}

	return c.Render("partials/moderation_success", fiber.Map{
		"Action":  "edit approved",
		"Keyword": editReq.Keyword,
	}, "")
}

// RejectEdit rejects an edit request.
func (h *ModerationHandler) RejectEdit(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	if !user.IsOrgMod() {
		return fiber.NewError(fiber.StatusForbidden, "you do not have moderation permissions")
	}

	idStr := c.Params("id")
	reqID, err := uuid.Parse(idStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request id")
	}

	editReq, err := h.db.GetEditRequestByID(c.Context(), reqID)
	if err != nil {
		if errors.Is(err, db.ErrEditRequestNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "edit request not found")
		}
		return err
	}

	if err := h.db.RejectEditRequest(c.Context(), reqID, user.ID); err != nil {
		if errors.Is(err, db.ErrEditRequestNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "edit request not found or already processed")
		}
		return err
	}

	return c.Render("partials/moderation_success", fiber.Map{
		"Action":  "edit rejected",
		"Keyword": editReq.Keyword,
	}, "")
}

// canModerate checks if a user can moderate a specific link.
func canModerate(user *models.User, link *models.Link) bool {
	// Admins and global mods can moderate anything (global and org links)
	if user.IsGlobalMod() {
		return true
	}

	// Org mods can moderate links for their org
	if link.Scope == models.ScopeOrg && link.OrganizationID != nil {
		return user.CanModerateOrg(*link.OrganizationID)
	}

	return false
}
