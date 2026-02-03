package api

import (
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"golinks/internal/config"
	"golinks/internal/db"
	"golinks/internal/models"
)

// UserHandler handles user management operations via JSON API.
type UserHandler struct {
	db  *db.DB
	cfg *config.Config
}

// NewUserHandler creates a new API user handler.
func NewUserHandler(database *db.DB, cfg *config.Config) *UserHandler {
	return &UserHandler{db: database, cfg: cfg}
}

// List returns all users with their organization info (admin only).
func (h *UserHandler) List(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok || !user.IsAdmin() {
		return jsonError(c, fiber.StatusForbidden, "admin access required")
	}

	users, err := h.db.GetAllUsersWithOrgs(c.Context())
	if err != nil {
		return jsonError(c, fiber.StatusInternalServerError, "failed to fetch users")
	}

	type userResponse struct {
		ID               uuid.UUID  `json:"id"`
		Sub              string     `json:"sub"`
		Username         string     `json:"username"`
		Email            string     `json:"email"`
		Name             string     `json:"name"`
		Role             string     `json:"role"`
		OrganizationID   *uuid.UUID `json:"organization_id"`
		OrganizationName string     `json:"organization_name"`
		OrganizationSlug string     `json:"organization_slug"`
		CreatedAt        time.Time  `json:"created_at"`
		UpdatedAt        time.Time  `json:"updated_at"`
	}

	resp := make([]userResponse, len(users))
	for i, u := range users {
		resp[i] = userResponse{
			ID:               u.ID,
			Sub:              u.Sub,
			Username:         u.Username,
			Email:            u.Email,
			Name:             u.Name,
			Role:             u.Role,
			OrganizationID:   u.OrganizationID,
			OrganizationName: u.OrganizationName,
			OrganizationSlug: u.OrganizationSlug,
			CreatedAt:        u.CreatedAt,
			UpdatedAt:        u.UpdatedAt,
		}
	}

	return jsonSuccess(c, resp)
}

// UpdateRole updates a user's role (admin only).
func (h *UserHandler) UpdateRole(c fiber.Ctx) error {
	currentUser, ok := c.Locals("user").(*models.User)
	if !ok || !currentUser.IsAdmin() {
		return jsonError(c, fiber.StatusForbidden, "admin access required")
	}

	userID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return jsonError(c, fiber.StatusBadRequest, "invalid user id")
	}

	var body struct {
		Role string `json:"role"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return jsonError(c, fiber.StatusBadRequest, "invalid request body")
	}

	if body.Role == "" {
		return jsonError(c, fiber.StatusBadRequest, "role is required")
	}

	validRoles := map[string]bool{
		models.RoleUser:      true,
		models.RoleOrgMod:    true,
		models.RoleGlobalMod: true,
		models.RoleAdmin:     true,
	}
	if !validRoles[body.Role] {
		return jsonError(c, fiber.StatusBadRequest, "invalid role")
	}

	if userID == currentUser.ID && body.Role != models.RoleAdmin {
		return jsonError(c, fiber.StatusBadRequest, "cannot change your own role")
	}

	if err := h.db.UpdateUserRole(c.Context(), userID, body.Role); err != nil {
		return jsonError(c, fiber.StatusInternalServerError, "failed to update role")
	}

	return jsonSuccess(c, fiber.Map{
		"message": "role updated successfully",
	})
}

// UpdateOrg updates a user's organization (admin only).
func (h *UserHandler) UpdateOrg(c fiber.Ctx) error {
	currentUser, ok := c.Locals("user").(*models.User)
	if !ok || !currentUser.IsAdmin() {
		return jsonError(c, fiber.StatusForbidden, "admin access required")
	}

	userID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return jsonError(c, fiber.StatusBadRequest, "invalid user id")
	}

	var body struct {
		OrganizationID *string `json:"organization_id"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return jsonError(c, fiber.StatusBadRequest, "invalid request body")
	}

	var orgID *uuid.UUID
	if body.OrganizationID != nil && *body.OrganizationID != "" {
		id, err := uuid.Parse(*body.OrganizationID)
		if err != nil {
			return jsonError(c, fiber.StatusBadRequest, "invalid organization id")
		}
		orgID = &id
	}

	if err := h.db.UpdateUserOrganization(c.Context(), userID, orgID); err != nil {
		return jsonError(c, fiber.StatusInternalServerError, "failed to update organization")
	}

	return jsonSuccess(c, fiber.Map{
		"message": "organization updated successfully",
	})
}

// Delete removes a user (admin only).
func (h *UserHandler) Delete(c fiber.Ctx) error {
	currentUser, ok := c.Locals("user").(*models.User)
	if !ok || !currentUser.IsAdmin() {
		return jsonError(c, fiber.StatusForbidden, "admin access required")
	}

	userID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return jsonError(c, fiber.StatusBadRequest, "invalid user id")
	}

	if userID == currentUser.ID {
		return jsonError(c, fiber.StatusBadRequest, "cannot delete your own account")
	}

	if err := h.db.DeleteUser(c.Context(), userID); err != nil {
		return jsonError(c, fiber.StatusInternalServerError, "failed to delete user")
	}

	return jsonSuccess(c, fiber.Map{
		"message": "user deleted successfully",
	})
}
