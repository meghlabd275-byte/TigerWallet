// SessionService - Session management service
package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/tigerwallet/admin_panel/internal/database"
	"github.com/tigerwallet/admin_panel/internal/models"
)

type SessionService struct{}

func NewSessionService() *SessionService {
	return &SessionService{}
}

func (s *SessionService) ListSessions(ctx context.Context, adminID uuid.UUID) ([]models.Session, error) {
	rows, err := database.Query(ctx, `
		SELECT id, admin_id, token_hash, ip_address, user_agent, expires_at, created_at
		FROM sessions WHERE admin_id = $1 AND expires_at > NOW() ORDER BY created_at DESC
	`, adminID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []models.Session
	for rows.Next() {
		var session models.Session
		err := rows.Scan(&session.ID, &session.AdminID, &session.TokenHash, &session.IPAddress, &session.UserAgent, &session.ExpiresAt, &session.CreatedAt)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}
	return sessions, nil
}

func (s *SessionService) RevokeSession(ctx context.Context, id uuid.UUID) error {
	_, err := database.Exec(ctx, "DELETE FROM sessions WHERE id = $1", id)
	return err
}

func (s *SessionService) RevokeAllSessions(ctx context.Context, adminID uuid.UUID) error {
	_, err := database.Exec(ctx, "DELETE FROM sessions WHERE admin_id = $1", adminID)
	return err
}

func (s *SessionService) CleanupExpiredSessions(ctx context.Context) error {
	_, err := database.Exec(ctx, "DELETE FROM sessions WHERE expires_at < $1", time.Now())
	return err
}
