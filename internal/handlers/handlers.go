package handlers

import (
	"html"

	"github.com/gofiber/fiber/v3"

	"golinks/internal/email"
)

// Notifier is the global email notifier instance.
// Set during application initialization.
var Notifier *email.Notifier

// SetNotifier sets the global email notifier.
func SetNotifier(n *email.Notifier) {
	Notifier = n
}

// htmxError returns an error message as HTML that HTMX will display.
// Uses 200 status so HTMX processes the swap (HTMX ignores non-2xx by default).
func htmxError(c fiber.Ctx, message string) error {
	return c.SendString(
		`<div class="p-3 rounded-lg bg-red-50 dark:bg-red-900/30 text-red-700 dark:text-red-300 text-sm">` + html.EscapeString(message) + `</div>`,
	)
}
