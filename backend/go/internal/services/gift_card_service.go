package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tigerwallet/backend/internal/models"
)

type GiftCardService struct {
	db *sql.DB
}

func NewGiftCardService(db *sql.DB) *GiftCardService {
	return &GiftCardService{db: db}
}

// Create gift card
func (s *GiftCardService) CreateGiftCard(ctx context.Context, userID uuid.UUID, token string, amount float64, templateID string) (*models.GiftCard, error) {
	// Generate unique code
	code := generateGiftCardCode()

	giftCard := &models.GiftCard{
		ID:          uuid.New(),
		Code:        code,
		Token:       token,
		Amount:      amount,
		TemplateID:  templateID,
		Status:      "ACTIVE",
		CreatedBy:   userID,
		RedeemedBy:  nil,
		RedeemedAt:  nil,
		ExpiresAt:   time.Now().Add(365 * 24 * time.Hour), // 1 year
		CreatedAt:   time.Now(),
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO gift_cards (id, code, token, amount, template_id, status, created_by, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, giftCard.ID, giftCard.Code, giftCard.Token, giftCard.Amount, giftCard.TemplateID, 
		giftCard.Status, giftCard.CreatedBy, giftCard.ExpiresAt, giftCard.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create gift card: %w", err)
	}

	return giftCard, nil
}

// Redeem gift card
func (s *GiftCardService) RedeemGiftCard(ctx context.Context, userID uuid.UUID, code string) (*models.GiftCard, error) {
	var giftCard models.GiftCard
	err := s.db.QueryRowContext(ctx, `
		SELECT id, code, token, amount, template_id, status, created_by, redeemed_by, redeemed_at, expires_at, created_at
		FROM gift_cards WHERE code = $1
	`, code).Scan(&giftCard.ID, &giftCard.Code, &giftCard.Token, &giftCard.Amount, 
		&giftCard.TemplateID, &giftCard.Status, &giftCard.CreatedBy, &giftCard.RedeemedBy, 
		&giftCard.RedeemedAt, &giftCard.ExpiresAt, &giftCard.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("gift card not found")
	}
	if err != nil {
		return nil, err
	}

	if giftCard.Status == "REDEEMED" {
		return nil, fmt.Errorf("gift card already redeemed")
	}

	if time.Now().After(giftCard.ExpiresAt) {
		return nil, fmt.Errorf("gift card expired")
	}

	// Redeem the card
	_, err = s.db.ExecContext(ctx, `
		UPDATE gift_cards 
		SET status = 'REDEEMED', redeemed_by = $1, redeemed_at = NOW()
		WHERE id = $2
	`, userID, giftCard.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to redeem gift card: %w", err)
	}

	giftCard.Status = "REDEEMED"
	now := time.Now()
	giftCard.RedeemedBy = &userID
	giftCard.RedeemedAt = &now

	return &giftCard, nil
}

// Get gift card balance
func (s *GiftCardService) GetGiftCardBalance(ctx context.Context, code string) (*models.GiftCard, error) {
	var giftCard models.GiftCard
	err := s.db.QueryRowContext(ctx, `
		SELECT id, code, token, amount, template_id, status, expires_at
		FROM gift_cards WHERE code = $1
	`, code).Scan(&giftCard.ID, &giftCard.Code, &giftCard.Token, &giftCard.Amount, 
		&giftCard.TemplateID, &giftCard.Status, &giftCard.ExpiresAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("gift card not found")
	}

	return &giftCard, err
}

// Get user's created gift cards
func (s *GiftCardService) GetUserCreatedCards(ctx context.Context, userID uuid.UUID) ([]models.GiftCard, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, code, token, amount, template_id, status, redeemed_by, redeemed_at, expires_at, created_at
		FROM gift_cards WHERE created_by = $1 ORDER BY created_at DESC
	`, userID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cards []models.GiftCard
	for rows.Next() {
		var c models.GiftCard
		err := rows.Scan(&c.ID, &c.Code, &c.Token, &c.Amount, &c.TemplateID, &c.Status, 
			&c.RedeemedBy, &c.RedeemedAt, &c.ExpiresAt, &c.CreatedAt)
		if err != nil {
			continue
		}
		cards = append(cards, c)
	}

	return cards, nil
}

// Get user's redeemed gift cards
func (s *GiftCardService) GetUserRedeemedCards(ctx context.Context, userID uuid.UUID) ([]models.GiftCard, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, code, token, amount, template_id, status, created_by, expires_at, redeemed_at, created_at
		FROM gift_cards WHERE redeemed_by = $1 ORDER BY redeemed_at DESC
	`, userID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cards []models.GiftCard
	for rows.Next() {
		var c models.GiftCard
		err := rows.Scan(&c.ID, &c.Code, &c.Token, &c.Amount, &c.TemplateID, &c.Status, 
			&c.CreatedBy, &c.ExpiresAt, &c.RedeemedAt, &c.CreatedAt)
		if err != nil {
			continue
		}
		cards = append(cards, c)
	}

	return cards, nil
}

// Get gift card templates
func (s *GiftCardService) GetTemplates(ctx context.Context) ([]models.GiftCardTemplate, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, image_url, is_active FROM gift_card_templates WHERE is_active = true
	`)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []models.GiftCardTemplate
	for rows.Next() {
		var t models.GiftCardTemplate
		err := rows.Scan(&t.ID, &t.Name, &t.ImageURL, &t.IsActive)
		if err != nil {
			continue
		}
		templates = append(templates, t)
	}

	return templates, nil
}

func generateGiftCardCode() string {
	// Generate a unique 16-character code
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	code := make([]byte, 16)
	for i := range code {
		code[i] = chars[time.Now().UnixNano()%int64(len(chars))]
		time.Sleep(time.Nanosecond)
	}
	return string(code)
}
