package handlers

import (
	"github.com/gofiber/fiber/v3"

	"golinks/internal/config"
)

// BrandingData contains site branding information for templates.
type BrandingData struct {
	SiteTitle                string
	SiteTagline              string
	SiteFooter               string
	SiteLogoURL              string
	EnableAnimatedBackground bool
	BannerText               string
	BannerTextColor          string
	BannerBGColor            string
}

// GetBrandingData returns branding data from config for template rendering.
func GetBrandingData(cfg *config.Config) BrandingData {
	return BrandingData{
		SiteTitle:                cfg.SiteTitle,
		SiteTagline:              cfg.SiteTagline,
		SiteFooter:               cfg.SiteFooter,
		SiteLogoURL:              cfg.SiteLogoURL,
		EnableAnimatedBackground: cfg.EnableAnimatedBackground,
		BannerText:               cfg.BannerText,
		BannerTextColor:          cfg.BannerTextColor,
		BannerBGColor:            cfg.BannerBGColor,
	}
}

// MergeBranding adds branding data to a fiber.Map for template rendering.
// Pass c.Path() as the currentPath argument to enable server-side nav active state.
func MergeBranding(data fiber.Map, cfg *config.Config, currentPath ...string) fiber.Map {
	branding := GetBrandingData(cfg)
	data["SiteTitle"] = branding.SiteTitle
	data["SiteTagline"] = branding.SiteTagline
	data["SiteFooter"] = branding.SiteFooter
	data["SiteLogoURL"] = branding.SiteLogoURL
	data["EnableAnimatedBackground"] = branding.EnableAnimatedBackground
	data["BannerText"] = branding.BannerText
	data["BannerTextColor"] = branding.BannerTextColor
	data["BannerBGColor"] = branding.BannerBGColor
	if len(currentPath) > 0 {
		data["CurrentPath"] = currentPath[0]
	}
	return data
}
