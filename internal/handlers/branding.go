package handlers

import (
	"github.com/gofiber/fiber/v3"

	"golinks/internal/config"
)

// BrandingData contains site branding information for templates.
type BrandingData struct {
	SiteTitle            string
	SiteTagline          string
	SiteFooter           string
	SiteLogoURL          string
	EnableAnimatedBackground bool
}

// GetBrandingData returns branding data from config for template rendering.
func GetBrandingData(cfg *config.Config) BrandingData {
	return BrandingData{
		SiteTitle:            cfg.SiteTitle,
		SiteTagline:          cfg.SiteTagline,
		SiteFooter:           cfg.SiteFooter,
		SiteLogoURL:          cfg.SiteLogoURL,
		EnableAnimatedBackground: cfg.EnableAnimatedBackground,
	}
}

// MergeBranding adds branding data to a fiber.Map for template rendering.
func MergeBranding(data fiber.Map, cfg *config.Config) fiber.Map {
	branding := GetBrandingData(cfg)
	data["SiteTitle"] = branding.SiteTitle
	data["SiteTagline"] = branding.SiteTagline
	data["SiteFooter"] = branding.SiteFooter
	data["SiteLogoURL"] = branding.SiteLogoURL
	data["EnableAnimatedBackground"] = branding.EnableAnimatedBackground
	return data
}
