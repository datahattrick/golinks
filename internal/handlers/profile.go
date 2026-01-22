package handlers

import (
	"github.com/gofiber/fiber/v3"

	"golinks/internal/db"
	"golinks/internal/models"
)

// ProfileHandler handles user profile pages.
type ProfileHandler struct {
	db *db.DB
}

// NewProfileHandler creates a new profile handler.
func NewProfileHandler(database *db.DB) *ProfileHandler {
	return &ProfileHandler{db: database}
}

// Show renders the user's profile page with their links.
func (h *ProfileHandler) Show(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return c.Redirect().To("/login")
	}

	links, err := h.db.GetLinksByUser(c.Context(), user.ID)
	if err != nil {
		return err
	}

	return c.Render("profile", fiber.Map{
		"User":  user,
		"Links": links,
	})
}
