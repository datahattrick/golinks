package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"log"

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
	db           *db.DB
	cfg          *config.Config
}

// NewAuthHandler creates a new auth handler with OIDC configuration.
func NewAuthHandler(ctx context.Context, cfg *config.Config, database *db.DB) (*AuthHandler, error) {
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
		db:           database,
		cfg:          cfg,
	}, nil
}

// Login initiates the OIDC login flow.
func (h *AuthHandler) Login(c fiber.Ctx) error {
	state := generateState()

	sess := session.FromContext(c)
	if sess == nil {
		return fiber.NewError(fiber.StatusInternalServerError, "session not available")
	}
	sess.Set("oauth_state", state)

	url := h.oauth2Config.AuthCodeURL(state)
	return c.Redirect().To(url)
}

// Callback handles the OIDC callback after authentication.
func (h *AuthHandler) Callback(c fiber.Ctx) error {
	sess := session.FromContext(c)
	if sess == nil {
		return fiber.NewError(fiber.StatusInternalServerError, "session not available")
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

	// Extract claims from ID token first
	claimsMap := make(map[string]any)
	if err := idToken.Claims(&claimsMap); err != nil {
		return err
	}

	// Also fetch userinfo endpoint to get additional claims (email, org, etc.)
	// Some OIDC providers only include minimal claims in the ID token
	userInfo, err := h.provider.UserInfo(c.Context(), oauth2.StaticTokenSource(oauth2Token))
	if err == nil {
		var userInfoClaims map[string]any
		if err := userInfo.Claims(&userInfoClaims); err == nil {
			// Merge userinfo claims into claimsMap (userinfo takes precedence)
			for k, v := range userInfoClaims {
				claimsMap[k] = v
			}
		}
	} else {
		log.Printf("Warning: Failed to fetch userinfo: %v", err)
	}

	// Debug: log received claims
	if h.cfg.IsDev() {
		log.Printf("OIDC claims received: %v", claimsMap)
	}

	// Extract standard claims
	sub, _ := claimsMap["sub"].(string)
	email, _ := claimsMap["email"].(string)
	name, _ := claimsMap["name"].(string)
	picture, _ := claimsMap["picture"].(string)

	// Upsert user first
	user := &models.User{
		Sub:     sub,
		Email:   email,
		Name:    name,
		Picture: picture,
	}
	if err := h.db.UpsertUser(c.Context(), user); err != nil {
		return err
	}

	// Handle organization claim if configured
	if h.cfg.OIDCOrgClaim != "" {
		if orgValue, ok := claimsMap[h.cfg.OIDCOrgClaim]; ok {
			var orgSlug string
			switch v := orgValue.(type) {
			case string:
				orgSlug = v
			case []any:
				// If it's an array, take the first value
				if len(v) > 0 {
					orgSlug, _ = v[0].(string)
				}
			}

			if orgSlug != "" {
				// Get or create the organization
				org, err := h.db.GetOrCreateOrganization(c.Context(), orgSlug)
				if err == nil {
					// Update user's organization
					h.db.UpdateUserOrganization(c.Context(), user.ID, &org.ID)
				}
			}
		}
	}

	// Store session
	sess.Set("user_sub", sub)

	// Redirect to original URL if stored, otherwise home
	redirectURL := "/"
	if savedRedirect := sess.Get("redirect_after_login"); savedRedirect != nil {
		if url, ok := savedRedirect.(string); ok && url != "" {
			redirectURL = url
		}
		sess.Delete("redirect_after_login")
	}

	return c.Redirect().To(redirectURL)
}

// Logout clears the user session.
func (h *AuthHandler) Logout(c fiber.Ctx) error {
	sess := session.FromContext(c)
	if sess != nil {
		sess.Destroy()
	}
	return c.Redirect().To("/")
}

func generateState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
