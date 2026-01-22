package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"golang.org/x/oauth2"

	"golinks/internal/config"
	"golinks/internal/db"
	"golinks/internal/models"
)

// AuthHandler handles OIDC authentication flows.
type AuthHandler struct {
	provider     *oidc.Provider
	oauth2Config oauth2.Config
	verifier     *oidc.IDTokenVerifier
	store        *session.Store
	db           *db.DB
}

// NewAuthHandler creates a new auth handler with OIDC configuration.
func NewAuthHandler(ctx context.Context, cfg *config.Config, store *session.Store, database *db.DB) (*AuthHandler, error) {
	provider, err := oidc.NewProvider(ctx, cfg.OIDCIssuer)
	if err != nil {
		return nil, err
	}

	oauth2Config := oauth2.Config{
		ClientID:     cfg.OIDCClientID,
		ClientSecret: cfg.OIDCClientSecret,
		RedirectURL:  cfg.OIDCRedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: cfg.OIDCClientID})

	return &AuthHandler{
		provider:     provider,
		oauth2Config: oauth2Config,
		verifier:     verifier,
		store:        store,
		db:           database,
	}, nil
}

// Login initiates the OIDC login flow.
func (h *AuthHandler) Login(c fiber.Ctx) error {
	state := generateState()

	sess, err := h.store.Get(c)
	if err != nil {
		return err
	}
	sess.Set("oauth_state", state)
	if err := sess.Save(); err != nil {
		return err
	}

	url := h.oauth2Config.AuthCodeURL(state)
	return c.Redirect().To(url)
}

// Callback handles the OIDC callback after authentication.
func (h *AuthHandler) Callback(c fiber.Ctx) error {
	sess, err := h.store.Get(c)
	if err != nil {
		return err
	}

	// Verify state
	savedState := sess.Get("oauth_state")
	if savedState == nil || savedState.(string) != c.Query("state") {
		return fiber.NewError(fiber.StatusBadRequest, "invalid state")
	}
	sess.Delete("oauth_state")

	// Exchange code for token
	oauth2Token, err := h.oauth2Config.Exchange(c.Context(), c.Query("code"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "failed to exchange code")
	}

	// Extract and verify ID token
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		return fiber.NewError(fiber.StatusBadRequest, "missing id_token")
	}

	idToken, err := h.verifier.Verify(c.Context(), rawIDToken)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id_token")
	}

	// Extract claims
	var claims struct {
		Sub     string `json:"sub"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return err
	}

	// Upsert user
	user := &models.User{
		Sub:     claims.Sub,
		Email:   claims.Email,
		Name:    claims.Name,
		Picture: claims.Picture,
	}
	if err := h.db.UpsertUser(c.Context(), user); err != nil {
		return err
	}

	// Store session
	sess.Set("user_sub", claims.Sub)
	if err := sess.Save(); err != nil {
		return err
	}

	return c.Redirect().To("/")
}

// Logout clears the user session.
func (h *AuthHandler) Logout(c fiber.Ctx) error {
	sess, err := h.store.Get(c)
	if err != nil {
		return c.Redirect().To("/")
	}

	sess.Destroy()
	return c.Redirect().To("/")
}

func generateState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
