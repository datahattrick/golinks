package db

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"golinks/internal/models"
)

// CreateNotification inserts a single notification.
func (d *DB) CreateNotification(ctx context.Context, n *models.Notification) error {
	query := `
		INSERT INTO notifications (user_id, type, title, body, action_url, link_id)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := d.Pool.Exec(ctx, query, n.UserID, n.Type, n.Title, n.Body, n.ActionURL, n.LinkID)
	return err
}

// CreateNotifications bulk-inserts multiple notifications using a pgx batch.
func (d *DB) CreateNotifications(ctx context.Context, ns []models.Notification) error {
	if len(ns) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, n := range ns {
		batch.Queue(
			`INSERT INTO notifications (user_id, type, title, body, action_url, link_id) VALUES ($1, $2, $3, $4, $5, $6)`,
			n.UserID, n.Type, n.Title, n.Body, n.ActionURL, n.LinkID,
		)
	}
	results := d.Pool.SendBatch(ctx, batch)
	defer results.Close()
	for range ns {
		if _, err := results.Exec(); err != nil {
			return err
		}
	}
	return nil
}

// GetNotificationsForUser returns the latest notifications for a user (most recent first).
func (d *DB) GetNotificationsForUser(ctx context.Context, userID uuid.UUID, limit int) ([]models.Notification, error) {
	query := `
		SELECT id, user_id, type, title, body, action_url, link_id, read, created_at
		FROM notifications
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	rows, err := d.Pool.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []models.Notification
	for rows.Next() {
		var n models.Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body, &n.ActionURL, &n.LinkID, &n.Read, &n.CreatedAt); err != nil {
			return nil, err
		}
		notifications = append(notifications, n)
	}
	return notifications, rows.Err()
}

// CountUnreadNotifications returns the number of unread notifications for a user.
func (d *DB) CountUnreadNotifications(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := d.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read = FALSE`, userID).Scan(&count)
	return count, err
}

// MarkNotificationRead marks a single notification as read, scoped to the owning user.
func (d *DB) MarkNotificationRead(ctx context.Context, id, userID uuid.UUID) error {
	_, err := d.Pool.Exec(ctx, `UPDATE notifications SET read = TRUE WHERE id = $1 AND user_id = $2`, id, userID)
	return err
}

// MarkAllNotificationsRead marks all notifications for a user as read.
func (d *DB) MarkAllNotificationsRead(ctx context.Context, userID uuid.UUID) error {
	_, err := d.Pool.Exec(ctx, `UPDATE notifications SET read = TRUE WHERE user_id = $1 AND read = FALSE`, userID)
	return err
}

// DeleteAllNotifications removes all notifications for a user.
func (d *DB) DeleteAllNotifications(ctx context.Context, userID uuid.UUID) error {
	_, err := d.Pool.Exec(ctx, `DELETE FROM notifications WHERE user_id = $1`, userID)
	return err
}

// DeleteNotification removes a single notification, scoped to the owning user.
func (d *DB) DeleteNotification(ctx context.Context, id, userID uuid.UUID) error {
	_, err := d.Pool.Exec(ctx, `DELETE FROM notifications WHERE id = $1 AND user_id = $2`, id, userID)
	return err
}

// DeleteNotificationsForLink removes all notifications of a given type tied to a link.
// Used to clean up pending-review notifications for moderators once a link is actioned.
func (d *DB) DeleteNotificationsForLink(ctx context.Context, linkID uuid.UUID, notifType string) error {
	_, err := d.Pool.Exec(ctx,
		`DELETE FROM notifications WHERE link_id = $1 AND type = $2`,
		linkID, notifType,
	)
	return err
}
