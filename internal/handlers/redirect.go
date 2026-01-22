package handlers

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v3"

	"golinks/internal/db"
)

// RedirectHandler handles keyword-to-URL redirects.
type RedirectHandler struct {
	db *db.DB
}

// NewRedirectHandler creates a new redirect handler.
func NewRedirectHandler(database *db.DB) *RedirectHandler {
	return &RedirectHandler{db: database}
}

// Redirect looks up a keyword and redirects to the associated URL.
func (h *RedirectHandler) Redirect(c fiber.Ctx) error {
	keyword := c.Params("keyword")

	link, err := h.db.GetLinkByKeyword(c.Context(), keyword)
	if err != nil {
		if errors.Is(err, db.ErrLinkNotFound) {
			return c.Status(fiber.StatusNotFound).Render("error", fiber.Map{
				"Title":   "Not Found",
				"Message": "The link 'go/" + keyword + "' does not exist.",
			})
		}
		return err
	}

	// Increment click count asynchronously
	go h.db.IncrementClickCount(context.Background(), keyword)

	return c.Redirect().To(link.URL)
}
