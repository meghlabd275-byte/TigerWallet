// WebhookService - Webhook management service
package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/tigerwallet/admin_panel/internal/database"
	"github.com/tigerwallet/admin_panel/internal/models"
)

type WebhookService struct{}

func NewWebhookService() *WebhookService {
	return &WebhookService{}
}

func (s *WebhookService) ListWebhooks(ctx context.Context) ([]models.Webhook, error) {
	rows, err := database.Query(ctx, `
		SELECT id, name, url, secret, events, is_active, created_at, created_by
		FROM webhooks ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var webhooks []models.Webhook
	for rows.Next() {
		var wh models.Webhook
		err := rows.Scan(&wh.ID, &wh.Name, &wh.URL, &wh.Secret, &wh.Events, &wh.IsActive, &wh.CreatedAt, &wh.CreatedBy)
		if err != nil {
			return nil, err
		}
		webhooks = append(webhooks, wh)
	}
	return webhooks, nil
}

func (s *WebhookService) CreateWebhook(ctx context.Context, wh *models.Webhook, adminID uuid.UUID) (*models.Webhook, error) {
	err := database.QueryRow(ctx, `
		INSERT INTO webhooks (name, url, secret, events, is_active, created_at, created_by)
		VALUES ($1, $2, $3, $4, $5, NOW(), $6)
		RETURNING id, created_at
	`, wh.Name, wh.URL, wh.Secret, wh.Events, wh.IsActive, adminID).Scan(&wh.ID, &wh.CreatedAt)
	return wh, err
}

func (s *WebhookService) TestWebhook(ctx context.Context, id uuid.UUID) error {
	// Create a test delivery record
	_, err := database.Exec(ctx, `
		INSERT INTO webhook_deliveries (webhook_id, event, payload, status, attempts, created_at)
		VALUES ($1, 'test', '{}', 'pending', 0, NOW())
	`, id)
	return err
}

func (s *WebhookService) DeleteWebhook(ctx context.Context, id uuid.UUID) error {
	_, err := database.Exec(ctx, "DELETE FROM webhooks WHERE id = $1", id)
	return err
}

// UpdateWebhookDelivery updates webhook delivery status
func (s *WebhookService) UpdateWebhookDelivery(ctx context.Context, id uuid.UUID, status string, responseCode int) error {
	_, err := database.Exec(ctx, `
		UPDATE webhook_deliveries SET status = $1, response_code = $2, attempts = attempts + 1, last_attempt_at = $3 WHERE id = $4
	`, status, responseCode, time.Now(), id)
	return err
}
