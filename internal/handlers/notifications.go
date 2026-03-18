package handlers

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"golinks/internal/db"
	"golinks/internal/models"
)

// NotificationHandler handles in-app notification operations.
type NotificationHandler struct {
	db *db.DB
}

// NewNotificationHandler creates a new notification handler.
func NewNotificationHandler(database *db.DB) *NotificationHandler {
	return &NotificationHandler{db: database}
}

// Count returns an unread badge span, or empty string if zero.
// Designed for HTMX polling: hx-trigger="load, every 60s"
func (h *NotificationHandler) Count(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return c.SendString("")
	}

	count, err := h.db.CountUnreadNotifications(c.Context(), user.ID)
	if err != nil || count == 0 {
		return c.SendString(`<span id="notif-badge"></span>`)
	}

	label := fmt.Sprintf("%d", count)
	if count > 99 {
		label = "99+"
	}
	return c.SendString(fmt.Sprintf(
		`<span id="notif-badge" class="absolute -top-1 -right-1 inline-flex items-center justify-center min-w-[1.1rem] h-[1.1rem] px-1 text-[10px] font-bold rounded-full bg-red-500 text-white leading-none">%s</span>`,
		label,
	))
}

// List renders the notifications panel partial.
func (h *NotificationHandler) List(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return c.SendString("")
	}

	notifications, err := h.db.GetNotificationsForUser(c.Context(), user.ID, 20)
	if err != nil {
		return c.SendString("")
	}

	return c.Render("partials/notifications_panel", fiber.Map{
		"Notifications": notifications,
		"Now":           time.Now(),
	}, "")
}

// MarkRead marks a single notification as read and returns the updated item HTML.
func (h *NotificationHandler) MarkRead(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	_ = h.db.MarkNotificationRead(c.Context(), id, user.ID)

	// Return updated badge HTML; JS caller handles navigation and panel close.
	count, _ := h.db.CountUnreadNotifications(c.Context(), user.ID)
	badgeHTML := `<span id="notif-badge"></span>`
	if count > 0 {
		label := fmt.Sprintf("%d", count)
		if count > 99 {
			label = "99+"
		}
		badgeHTML = fmt.Sprintf(
			`<span id="notif-badge" class="absolute -top-1 -right-1 inline-flex items-center justify-center min-w-[1.1rem] h-[1.1rem] px-1 text-[10px] font-bold rounded-full bg-red-500 text-white leading-none">%s</span>`,
			label,
		)
	}

	// Return OOB badge update alongside empty body (navigation handled by JS)
	return c.SendString(badgeHTML)
}

// Delete removes a single notification from the user's feed.
// The row is removed via outerHTML swap; badge is updated via OOB swap.
func (h *NotificationHandler) Delete(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	_ = h.db.DeleteNotification(c.Context(), id, user.ID)

	// Build updated badge HTML for OOB swap
	count, _ := h.db.CountUnreadNotifications(c.Context(), user.ID)
	var badgeInner string
	if count > 0 {
		label := fmt.Sprintf("%d", count)
		if count > 99 {
			label = "99+"
		}
		badgeInner = fmt.Sprintf(
			`<span id="notif-badge" hx-swap-oob="outerHTML:#notif-badge" class="absolute -top-1 -right-1 inline-flex items-center justify-center min-w-[1.1rem] h-[1.1rem] px-1 text-[10px] font-bold rounded-full bg-red-500 text-white leading-none">%s</span>`,
			label,
		)
	} else {
		badgeInner = `<span id="notif-badge" hx-swap-oob="outerHTML:#notif-badge"></span>`
	}

	// Primary swap (outerHTML on the row) → empty removes the row.
	// OOB swap updates the badge.
	return c.SendString(badgeInner)
}

// DeleteAll removes all notifications for the user and re-renders the panel.
func (h *NotificationHandler) DeleteAll(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	_ = h.db.DeleteAllNotifications(c.Context(), user.ID)

	return c.Render("partials/notifications_panel", fiber.Map{
		"Notifications": nil,
		"Now":           time.Now(),
		"ClearBadge":    true,
	}, "")
}

// MarkAllRead marks all notifications as read and re-renders the panel.
func (h *NotificationHandler) MarkAllRead(c fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	_ = h.db.MarkAllNotificationsRead(c.Context(), user.ID)

	notifications, err := h.db.GetNotificationsForUser(c.Context(), user.ID, 20)
	if err != nil {
		notifications = nil
	}

	return c.Render("partials/notifications_panel", fiber.Map{
		"Notifications": notifications,
		"Now":           time.Now(),
		"ClearBadge":    true,
	}, "")
}
