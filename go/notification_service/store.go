package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// store persists notifications and subscriptions.
//
// The service previously kept both in process memory, so every notification
// vanished on restart while the API still reported status "sent". Nothing is
// marked sent now unless a delivery channel actually accepted it.
type store struct {
	pool *pgxpool.Pool
}

func newStore(ctx context.Context, dsn string) (*store, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, errors.New("DATABASE_URL is required: notifications must survive a restart")
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	if v := os.Getenv("PG_MAX_CONNS"); v != "" {
		if n, err := parseInt32(v); err == nil && n > 0 {
			cfg.MaxConns = n
		}
	}
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	s := &store{pool: pool}
	if err := s.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return s, nil
}

func (s *store) Close() { s.pool.Close() }

func (s *store) migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS notifications (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL,
    type        TEXT NOT NULL,
    channel     TEXT NOT NULL,
    title       TEXT NOT NULL,
    body        TEXT NOT NULL,
    data        JSONB NOT NULL DEFAULT '{}'::jsonb,
    priority    TEXT NOT NULL DEFAULT 'normal',
    status      TEXT NOT NULL DEFAULT 'pending',
    error       TEXT NOT NULL DEFAULT '',
    is_read     BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    sent_at     TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_status ON notifications(status) WHERE status = 'pending';

CREATE TABLE IF NOT EXISTS notification_subscriptions (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL,
    channels    TEXT[] NOT NULL,
    events      TEXT[] NOT NULL,
    enabled     BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_subscriptions_user ON notification_subscriptions(user_id);

CREATE TABLE IF NOT EXISTS notification_templates (
    id          TEXT PRIMARY KEY,
    title       TEXT NOT NULL,
    body        TEXT NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO notification_templates (id, title, body) VALUES
    ('tx_sent',       'Transaction Sent',    'Your transaction of {{amount}} {{token}} has been sent.'),
    ('tx_received',   'Transaction Received','You received {{amount}} {{token}} from {{from}}.'),
    ('swap_complete', 'Swap Complete',       'Your swap of {{from_amount}} {{from_token}} to {{to_amount}} {{to_token}} is complete.'),
    ('staking_reward','Staking Reward',      'You received {{amount}} {{token}} in staking rewards.'),
    ('nft_sold',      'NFT Sold',            'Your NFT {{nft_name}} was sold for {{price}} {{token}}.'),
    ('price_alert',   'Price Alert',         '{{token}} has reached {{price}}.')
ON CONFLICT (id) DO NOTHING;
`)
	return err
}

// insert stores a notification and returns it.
func (s *store) insert(ctx context.Context, n *Notification) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO notifications (id, user_id, type, channel, title, body, data, priority, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, n.ID, n.UserID, n.Type, n.Channel, n.Title, n.Body, n.Data, n.Priority, n.Status, n.CreatedAt)
	return err
}

// markResult records the outcome of a delivery attempt. A notification is only
// ever marked sent after a channel accepted it; otherwise the provider's error
// is stored so the failure is visible instead of silently reported as success.
func (s *store) markResult(ctx context.Context, id, status, errMsg string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE notifications SET status = $2, error = $3, sent_at = CASE WHEN $2 = 'sent' THEN now() ELSE sent_at END
		WHERE id = $1
	`, id, status, errMsg)
	return err
}

func (s *store) listForUser(ctx context.Context, userID string, limit int) ([]Notification, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, type, channel, title, body, data, priority, status, is_read, created_at, sent_at
		FROM notifications WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Notification, 0, limit)
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Channel, &n.Title, &n.Body, &n.Data,
			&n.Priority, &n.Status, &n.Read, &n.CreatedAt, &n.SentAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// markRead marks one notification read, scoped to its owner so a caller cannot
// touch another user's record by guessing an id.
func (s *store) markRead(ctx context.Context, id, userID string) (bool, error) {
	tag, err := s.pool.Exec(ctx,
		`UPDATE notifications SET is_read = true WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (s *store) markAllRead(ctx context.Context, userID string) (int64, error) {
	tag, err := s.pool.Exec(ctx,
		`UPDATE notifications SET is_read = true WHERE user_id = $1 AND is_read = false`, userID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (s *store) delete(ctx context.Context, id, userID string) (bool, error) {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM notifications WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (s *store) clearForUser(ctx context.Context, userID string) (int64, error) {
	tag, err := s.pool.Exec(ctx, `DELETE FROM notifications WHERE user_id = $1`, userID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (s *store) createSubscription(ctx context.Context, sub *Subscription) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO notification_subscriptions (id, user_id, channels, events, enabled, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, sub.ID, sub.UserID, sub.Channels, sub.Events, sub.Enabled, sub.CreatedAt)
	return err
}

func (s *store) deleteSubscription(ctx context.Context, id, userID string) (bool, error) {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM notification_subscriptions WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// subscribersFor returns the user ids subscribed to an event on a channel.
func (s *store) subscribersFor(ctx context.Context, event, channel string) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT user_id FROM notification_subscriptions
		WHERE enabled = true AND $1 = ANY(events) AND $2 = ANY(channels)
	`, event, channel)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var u string
		if err := rows.Scan(&u); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *store) templates(ctx context.Context) ([]map[string]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, title, body FROM notification_templates ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]map[string]string, 0)
	for rows.Next() {
		var id, title, body string
		if err := rows.Scan(&id, &title, &body); err != nil {
			return nil, err
		}
		out = append(out, map[string]string{"id": id, "title": title, "body": body})
	}
	return out, rows.Err()
}

// pendingIDs returns notifications whose delivery attempt never completed, so
// a restart can resume them instead of leaving them stuck.
func (s *store) pendingIDs(ctx context.Context, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 500
	}
	rows, err := s.pool.Query(ctx,
		`SELECT id FROM notifications WHERE status = 'pending' ORDER BY created_at LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// channelFor returns the delivery channels a user subscribed to for an event.
func (s *store) channelsFor(ctx context.Context, userID, event string) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT channels FROM notification_subscriptions
		WHERE user_id = $1 AND enabled = true AND $2 = ANY(events)
	`, userID, event)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var chans []string
		if err := rows.Scan(&chans); err != nil {
			return nil, err
		}
		out = append(out, chans...)
	}
	return out, rows.Err()
}
