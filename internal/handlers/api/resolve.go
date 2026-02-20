package api

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"golinks/internal/config"
	"golinks/internal/db"
	"golinks/internal/models"
	"golinks/internal/validation"
)

// ResolveHandler handles keyword resolution via JSON API.
type ResolveHandler struct {
	db  *db.DB
	cfg *config.Config
}

// NewResolveHandler creates a new API resolve handler.
func NewResolveHandler(database *db.DB, cfg *config.Config) *ResolveHandler {
	return &ResolveHandler{db: database, cfg: cfg}
}

// Resolve resolves a keyword to its URL without performing a redirect.
func (h *ResolveHandler) Resolve(c fiber.Ctx) error {
	keyword := validation.NormalizeKeyword(c.Params("keyword"))

	if !validation.ValidateKeyword(keyword) {
		return jsonError(c, fiber.StatusBadRequest, "invalid keyword")
	}

	user, _ := c.Locals("user").(*models.User)

	var userID *uuid.UUID
	var orgID *uuid.UUID
	if user != nil {
		userID = &user.ID
		orgID = user.OrganizationID
	}

	resolved, err := h.db.ResolveKeywordForUser(c.Context(), userID, orgID, keyword)
	if err != nil {
		if errors.Is(err, db.ErrLinkNotFound) {
			return jsonError(c, fiber.StatusNotFound, "keyword not found")
		}
		return jsonError(c, fiber.StatusInternalServerError, "failed to resolve keyword")
	}

	return jsonSuccess(c, models.ResolveResponse{
		Keyword: keyword,
		URL:     resolved.URL,
		Source:  resolved.Source,
	})
}
