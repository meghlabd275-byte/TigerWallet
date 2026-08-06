// NotificationService - Notification management service
package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/tigerwallet/admin_panel/internal/database"
	"github.com/tigerwallet/admin_panel/internal/models"
)

type NotificationService struct{}

func NewNotificationService() *NotificationService {
	return &NotificationService{}
}

func (s *NotificationService) ListNotifications(ctx context.Context, adminID uuid.UUID, limit, offset int) ([]models.Notification, int, error) {
	var total int
	database.QueryRow(ctx, "SELECT COUNT(*) FROM notifications WHERE admin_id = $1", adminID).Scan(&total)

	rows, err := database.Query(ctx, `
		SELECT id, admin_id, title, message, notification_type, is_read, created_at
		FROM notifications WHERE admin_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3
	`, adminID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var notifications []models.Notification
	for rows.Next() {
		var n models.Notification
		err := rows.Scan(&n.ID, &n.AdminID, &n.Title, &n.Message, &n.NotificationType, &n.IsRead, &n.CreatedAt)
		if err != nil {
			return nil, 0, err
		}
		notifications = append(notifications, n)
	}
	return notifications, total, nil
}

func (s *NotificationService) SendNotification(ctx context.Context, adminID uuid.UUID, title, message, notifType string) error {
	_, err := database.Exec(ctx, `
		INSERT INTO notifications (admin_id, title, message, notification_type, is_read, created_at)
		VALUES ($1, $2, $3, $4, false, NOW())
	`, adminID, title, message, notifType)
	return err
}

func (s *NotificationService) Broadcast(ctx context.Context, title, message, notifType string) error {
	_, err := database.Exec(ctx, `
		INSERT INTO notifications (admin_id, title, message, notification_type, is_read, created_at)
		SELECT id, $1, $2, $3, false, NOW() FROM admin_users WHERE is_active = true
	`, title, message, notifType)
	return err
}

func (s *NotificationService) MarkAsRead(ctx context.Context, id uuid.UUID) error {
	_, err := database.Exec(ctx, "UPDATE notifications SET is_read = true WHERE id = $1", id)
	return err
}

func (s *NotificationService) MarkAllAsRead(ctx context.Context, adminID uuid.UUID) error {
	_, err := database.Exec(ctx, "UPDATE notifications SET is_read = true WHERE admin_id = $1", adminID)
	return err
}
