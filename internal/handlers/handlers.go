package handlers

import (
	"html"
	"strconv"

	"github.com/gofiber/fiber/v3"

	"golinks/internal/email"
)

// PageInfo holds computed pagination state for templates.
type PageInfo struct {
	Page      int
	PerPage   int
	Total     int
	TotalPages int
	HasPrev   bool
	HasNext   bool
	PrevPage  int
	NextPage  int
	From      int // first item number on this page (1-based)
	To        int // last item number on this page
}

// parsePagination reads page and per_page from query params with sane defaults/bounds.
func parsePagination(c fiber.Ctx) (page, perPage int) {
	perPage, _ = strconv.Atoi(c.Query("per_page", "20"))
	if perPage != 20 && perPage != 30 && perPage != 50 {
		perPage = 20
	}
	page, _ = strconv.Atoi(c.Query("page", "1"))
	if page < 1 {
		page = 1
	}
	return
}

// buildPagination computes all derived pagination values for a template.
func buildPagination(page, perPage, total int) PageInfo {
	totalPages := total / perPage
	if total%perPage != 0 {
		totalPages++
	}
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	from := (page-1)*perPage + 1
	to := page * perPage
	if to > total {
		to = total
	}
	if total == 0 {
		from = 0
	}
	return PageInfo{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
		PrevPage:   page - 1,
		NextPage:   page + 1,
		From:       from,
		To:         to,
	}
}

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
