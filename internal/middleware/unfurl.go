package middleware

import (
	"errors"
	"fmt"
	"html"
	"strings"

	"github.com/gofiber/fiber/v3"

	"golinks/internal/config"
	"golinks/internal/db"
	"golinks/internal/validation"
)

// unfurlBots contains User-Agent substrings for known link-preview crawlers.
var unfurlBots = []string{
	"Slackbot",
	"SkypeUriPreview",
	"Discordbot",
	"TelegramBot",
	"WhatsApp",
	"facebookexternalhit",
	"Twitterbot",
	"LinkedInBot",
	"Applebot",
}

func isUnfurlBot(ua string) bool {
	for _, bot := range unfurlBots {
		if strings.Contains(ua, bot) {
			return true
		}
	}
	return false
}

// UnfurlMiddleware intercepts requests from link-preview bots and returns an
// HTML page with Open Graph meta tags instead of redirecting or requiring auth.
//
// For /go/:keyword paths the keyword is resolved globally (no user context) so
// the destination URL can be included in the description. For all other paths a
// generic site preview is returned.
//
// This must be registered before auth middleware so that bots never reach the
// OIDC login flow (which shows "page expired" to headless clients).
func UnfurlMiddleware(database *db.DB, cfg *config.Config) fiber.Handler {
	siteName := cfg.SiteTitle
	if siteName == "" {
		siteName = "GoLinks"
	}

	return func(c fiber.Ctx) error {
		if !isUnfurlBot(c.Get("User-Agent")) {
			return c.Next()
		}

		title := siteName
		description := cfg.SiteTagline
		if description == "" {
			description = "Fast URL shortcuts for your team"
		}
		canonicalURL := cfg.BaseURL + c.Path()

		// For /go/:keyword, try to resolve the keyword without user context.
		path := c.Path()
		if rest, ok := strings.CutPrefix(path, "/go/"); ok {
			keyword := validation.NormalizeKeyword(rest)
			if validation.ValidateKeyword(keyword) {
				title = fmt.Sprintf("%s: %s", siteName, keyword)
				resolved, err := database.ResolveKeywordForUser(c.Context(), nil, nil, keyword)
				switch {
				case err == nil:
					description = "\u2192 " + resolved.URL
				case errors.Is(err, db.ErrLinkNotFound):
					description = "Keyword not found"
				default:
					description = "Shortcut not available"
				}
			}
		}

		e := html.EscapeString
		body := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>%s</title>
<meta property="og:title" content="%s">
<meta property="og:description" content="%s">
<meta property="og:url" content="%s">
<meta property="og:site_name" content="%s">
<meta property="og:type" content="website">
</head>
<body><p>%s</p></body>
</html>`, e(title), e(title), e(description), e(canonicalURL), e(siteName), e(description))

		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.Status(fiber.StatusOK).SendString(body)
	}
}
