package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"golinks/internal/config"
	"golinks/internal/db"
	"golinks/internal/models"
)

// UserHandler handles user management operations.
type UserHandler struct {
	db  *db.DB
	cfg *config.Config
}

// NewUserHandler creates a new user handler.
func NewUserHandler(database *db.DB, cfg *config.Config) *UserHandler {
	return &UserHandler{db: database, cfg: cfg}
}

// ListUsers renders the user management page (admin only).
func (h *UserHandler) ListUsers(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok || !user.IsAdmin() {
		return fiber.NewError(fiber.StatusForbidden, "admin access required")
	}

	// Get all users with org info
	users, err := h.db.GetAllUsersWithOrgs(c.Context())
	if err != nil {
		return err
	}

	// Get all organizations for the dropdown
	orgs, err := h.db.GetAllOrganizations(c.Context())
	if err != nil {
		return err
	}

	// Get user counts by org
	orgCounts, err := h.db.GetUserCountByOrg(c.Context())
	if err != nil {
		return err
	}

	return c.Render("users", MergeBranding(fiber.Map{
		"User":      user,
		"Users":     users,
		"Orgs":      orgs,
		"OrgCounts": orgCounts,
		"Roles":     []string{models.RoleUser, models.RoleOrgMod, models.RoleGlobalMod, models.RoleAdmin},
	}, h.cfg))
}

// UpdateUserRole updates a user's role (admin only).
func (h *UserHandler) UpdateUserRole(c fiber.Ctx) error {
	currentUser, ok := c.Locals("user").(*models.User)
	if !ok || !currentUser.IsAdmin() {
		return fiber.NewError(fiber.StatusForbidden, "admin access required")
	}

	userID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid user ID")
	}

	role := c.FormValue("role")
	if role == "" {
		return fiber.NewError(fiber.StatusBadRequest, "role is required")
	}

	// Validate role
	validRoles := map[string]bool{
		models.RoleUser:      true,
		models.RoleOrgMod:    true,
		models.RoleGlobalMod: true,
		models.RoleAdmin:     true,
	}
	if !validRoles[role] {
		return fiber.NewError(fiber.StatusBadRequest, "invalid role")
	}

	// Prevent admins from demoting themselves
	if userID == currentUser.ID && role != models.RoleAdmin {
		return fiber.NewError(fiber.StatusBadRequest, "cannot change your own role")
	}

	if err := h.db.UpdateUserRole(c.Context(), userID, role); err != nil {
		return err
	}

	// Return updated user row
	users, err := h.db.GetAllUsersWithOrgs(c.Context())
	if err != nil {
		return err
	}

	// Find the updated user
	for _, u := range users {
		if u.ID == userID {
			orgs, _ := h.db.GetAllOrganizations(c.Context())
			return c.Render("partials/user_row", fiber.Map{
				"UserRow":     u,
				"CurrentUser": currentUser,
				"Orgs":        orgs,
				"Roles":       []string{models.RoleUser, models.RoleOrgMod, models.RoleGlobalMod, models.RoleAdmin},
			}, "")
		}
	}

	return c.SendStatus(fiber.StatusOK)
}

// UpdateUserOrg updates a user's organization (admin only).
func (h *UserHandler) UpdateUserOrg(c fiber.Ctx) error {
	currentUser, ok := c.Locals("user").(*models.User)
	if !ok || !currentUser.IsAdmin() {
		return fiber.NewError(fiber.StatusForbidden, "admin access required")
	}

	userID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid user ID")
	}

	orgIDStr := c.FormValue("organization_id")
	var orgID *uuid.UUID
	if orgIDStr != "" && orgIDStr != "none" {
		id, err := uuid.Parse(orgIDStr)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid organization ID")
		}
		orgID = &id
	}

	if err := h.db.UpdateUserOrganization(c.Context(), userID, orgID); err != nil {
		return err
	}

	// Get all organizations for the dropdown
	orgs, err := h.db.GetAllOrganizations(c.Context())
	if err != nil {
		return err
	}

	// Return updated user row
	users, err := h.db.GetAllUsersWithOrgs(c.Context())
	if err != nil {
		return err
	}

	// Find the updated user
	for _, u := range users {
		if u.ID == userID {
			return c.Render("partials/user_row", fiber.Map{
				"UserRow":     u,
				"CurrentUser": currentUser,
				"Orgs":        orgs,
				"Roles":       []string{models.RoleUser, models.RoleOrgMod, models.RoleGlobalMod, models.RoleAdmin},
			}, "")
		}
	}

	return c.SendStatus(fiber.StatusOK)
}

// DeleteUser deletes a user (admin only).
func (h *UserHandler) DeleteUser(c fiber.Ctx) error {
	currentUser, ok := c.Locals("user").(*models.User)
	if !ok || !currentUser.IsAdmin() {
		return fiber.NewError(fiber.StatusForbidden, "admin access required")
	}

	userID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid user ID")
	}

	// Prevent admins from deleting themselves
	if userID == currentUser.ID {
		return fiber.NewError(fiber.StatusBadRequest, "cannot delete your own account")
	}

	if err := h.db.DeleteUser(c.Context(), userID); err != nil {
		return err
	}

	// Return empty response - HTMX will remove the row
	return c.SendStatus(fiber.StatusOK)
}
