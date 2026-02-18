package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"log"
	"log/slog"
	"strings"

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

	scopes := []string{oidc.ScopeOpenID, "profile", "email"}
	if cfg.HasGroupRoleMapping() {
		scopes = append(scopes, "groups")
	}

	oauth2Config := oauth2.Config{
		ClientID:     cfg.OIDCClientID,
		ClientSecret: cfg.OIDCClientSecret,
		RedirectURL:  cfg.OIDCRedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       scopes,
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

	verifier := oauth2.GenerateVerifier()
	sess.Set("pkce_verifier", verifier)

	url := h.oauth2Config.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))
	return c.Redirect().To(url)
}

// Callback handles the OIDC callback after authentication.
func (h *AuthHandler) Callback(c fiber.Ctx) error {
	sess := session.FromContext(c)
	if sess == nil {
		return fiber.NewError(fiber.StatusInternalServerError, "session not available")
	}

	// Verify state
	savedState, ok := sess.Get("oauth_state").(string)
	if !ok || savedState == "" || savedState != c.Query("state") {
		return fiber.NewError(fiber.StatusBadRequest, "invalid state")
	}
	sess.Delete("oauth_state")

	// Retrieve PKCE verifier
	verifier, _ := sess.Get("pkce_verifier").(string)
	sess.Delete("pkce_verifier")

	// Exchange code for token (with PKCE verifier)
	var exchangeOpts []oauth2.AuthCodeOption
	if verifier != "" {
		exchangeOpts = append(exchangeOpts, oauth2.VerifierOption(verifier))
	}
	oauth2Token, err := h.oauth2Config.Exchange(c.Context(), c.Query("code"), exchangeOpts...)
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

	// Preserve groups from the ID token before merging userinfo claims.
	// Many providers (Keycloak, Azure AD, etc.) include groups only in the
	// ID token, not the userinfo endpoint response.
	idTokenGroups := extractGroups(claimsMap, h.cfg.OIDCGroupsClaim)

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
				org, created, err := h.db.GetOrCreateOrganization(c.Context(), orgSlug)
				if err == nil {
					h.db.UpdateUserOrganization(c.Context(), user.ID, &org.ID)
					user.OrganizationID = &org.ID

					// New org + active group mapping → promote any existing users
					// in this org who were previously mapped to moderator
					if created && h.cfg.HasGroupRoleMapping() {
						if promErr := h.db.PromoteOrgModerators(c.Context(), org.ID); promErr != nil {
							log.Printf("Warning: failed to promote org moderators for new org %s: %v", orgSlug, promErr)
						}
					}
				}
			}
		}
	}

	// Apply OIDC group-based role mapping when configured.
	// Admin > moderator > user.  Moderator-mapped users become org_mod when they
	// belong to an organisation, global_mod otherwise.
	if h.cfg.HasGroupRoleMapping() {
		groups := extractGroups(claimsMap, h.cfg.OIDCGroupsClaim)
		// Fall back to ID token groups if the userinfo merge overwrote them
		if len(groups) == 0 {
			groups = idTokenGroups
		}
		if len(groups) == 0 && h.cfg.IsDev() {
			log.Printf("Warning: OIDC group role mapping is configured but no groups found in claim '%s'", h.cfg.OIDCGroupsClaim)
		}
		mappedRole := resolveRoleFromGroups(groups, h.cfg)
		finalRole := finalRoleFromMapped(mappedRole, user.OrganizationID != nil)
		if err := h.db.UpdateUserRoleFromOIDC(c.Context(), user.ID, mappedRole, finalRole); err != nil {
			log.Printf("Warning: failed to update role from OIDC groups for user %s: %v", sub, err)
		}
	}

	// Store session and regenerate ID to prevent session fixation
	sess.Set("user_sub", sub)
	if err := sess.Regenerate(); err != nil {
		slog.Error("failed to regenerate session", "error", err)
	}

	// Redirect to original URL if stored, otherwise home.
	// Validate that the redirect is a safe relative path to prevent open redirects.
	redirectURL := "/"
	if savedRedirect := sess.Get("redirect_after_login"); savedRedirect != nil {
		if url, ok := savedRedirect.(string); ok && isSafeRedirect(url) {
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
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// isSafeRedirect validates that a redirect URL is a relative path on this server.
// Rejects empty strings, protocol-relative URLs (//evil.com), absolute URLs,
// and paths containing backslashes or control characters.
func isSafeRedirect(url string) bool {
	if url == "" || url[0] != '/' {
		return false
	}
	// Block protocol-relative URLs (//evil.com) and backslash variants
	if strings.HasPrefix(url, "//") || strings.HasPrefix(url, "/\\") {
		return false
	}
	// Block URLs containing control characters
	for _, c := range url {
		if c < 0x20 || c == 0x7f {
			return false
		}
	}
	return true
}

// extractGroups pulls a string slice out of a claims map value that may be
// a []any (most providers) or a bare string.
func extractGroups(claimsMap map[string]any, claimName string) []string {
	val, ok := claimsMap[claimName]
	if !ok {
		return nil
	}
	switch v := val.(type) {
	case []any:
		groups := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				groups = append(groups, s)
			}
		}
		return groups
	case string:
		if v != "" {
			return []string{v}
		}
	}
	return nil
}

// resolveRoleFromGroups returns the highest role implied by the user's OIDC
// groups: "admin", "moderator", or "user".  This is the intermediate value —
// the final DB role is determined by finalRoleFromMapped.
func resolveRoleFromGroups(groups []string, cfg *config.Config) string {
	groupSet := make(map[string]struct{}, len(groups))
	for _, g := range groups {
		groupSet[g] = struct{}{}
	}

	for _, ag := range cfg.OIDCAdminGroups {
		if _, ok := groupSet[ag]; ok {
			return "admin"
		}
	}
	for _, mg := range cfg.OIDCModeratorGroups {
		if _, ok := groupSet[mg]; ok {
			return "moderator"
		}
	}
	return "user"
}

// finalRoleFromMapped converts the intermediate mapped role into the actual
// role constant stored in the database.  Moderator-mapped users become org_mod
// when they belong to an organisation (scoped to that org's keywords only) or
// global_mod when they do not.
func finalRoleFromMapped(mappedRole string, hasOrg bool) string {
	switch mappedRole {
	case "admin":
		return models.RoleAdmin
	case "moderator":
		if hasOrg {
			return models.RoleOrgMod
		}
		return models.RoleGlobalMod
	default:
		return models.RoleUser
	}
}
