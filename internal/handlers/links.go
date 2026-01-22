package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"golinks/internal/db"
	"golinks/internal/models"
)

// LinkHandler handles link CRUD operations.
type LinkHandler struct {
	db *db.DB
}

// NewLinkHandler creates a new link handler.
func NewLinkHandler(database *db.DB) *LinkHandler {
	return &LinkHandler{db: database}
}

// Index renders the home page with all links.
func (h *LinkHandler) Index(c fiber.Ctx) error {
	links, err := h.db.SearchLinks(c.Context(), "", 50)
	if err != nil {
		return err
	}

	return c.Render("index", fiber.Map{
		"Links": links,
		"User":  c.Locals("user"),
	})
}

// Create handles creating a new link via HTMX.
func (h *LinkHandler) Create(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	keyword := c.FormValue("keyword")
	url := c.FormValue("url")
	description := c.FormValue("description")

	if keyword == "" || url == "" {
		return c.Status(fiber.StatusBadRequest).Render("partials/link_row", fiber.Map{
			"Error": "Keyword and URL are required",
		})
	}

	link := &models.Link{
		Keyword:     keyword,
		URL:         url,
		Description: description,
		CreatedBy:   &user.ID,
	}

	if err := h.db.CreateLink(c.Context(), link); err != nil {
		if errors.Is(err, db.ErrDuplicateKeyword) {
			return c.Status(fiber.StatusConflict).Render("partials/link_row", fiber.Map{
				"Error": "Keyword already exists",
			})
		}
		return err
	}

	return c.Render("partials/link_row", fiber.Map{
		"Link": link,
		"User": user,
	})
}

// Search handles searching for links via HTMX.
func (h *LinkHandler) Search(c fiber.Ctx) error {
	query := c.Query("q", "")

	links, err := h.db.SearchLinks(c.Context(), query, 50)
	if err != nil {
		return err
	}

	return c.Render("partials/search_results", fiber.Map{
		"Links": links,
		"Query": query,
		"User":  c.Locals("user"),
	})
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

	if err := h.db.DeleteLink(c.Context(), id, user.ID); err != nil {
		if errors.Is(err, db.ErrLinkNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "link not found")
		}
		return err
	}

	// Return empty response for HTMX to remove the row
	return c.SendString("")
}
