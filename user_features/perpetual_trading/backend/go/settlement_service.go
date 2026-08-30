package main

import (
	"context"
	"log"
	"math/big"
	"sync"
	"time"
)

// SettlementService handles settlement operations
type SettlementService struct {
	mu                 sync.RWMutex
	pendingSettlements map[string]*Settlement
	liquidations       map[string]*Liquidation
	fundingPayments    map[string][]*FundingPayment
}

// NewSettlementService creates a new settlement service
func NewSettlementService() *SettlementService {
	return &SettlementService{
		pendingSettlements: make(map[string]*Settlement),
		liquidations:       make(map[string]*Liquidation),
		fundingPayments:    make(map[string][]*FundingPayment),
	}
}

// Settlement represents a settlement
type Settlement struct {
	ID         string `json:"id"`
	UserID     string `json:"userId"`
	Symbol     string `json:"symbol"`
	Side       string `json:"side"`
	Quantity   string `json:"quantity"`
	Price      string `json:"price"`
	SettleType string `json:"settleType"`
	Amount     string `json:"amount"`
	Status     string `json:"status"`
	Timestamp  int64  `json:"timestamp"`
}

// Liquidation represents a liquidation event
type Liquidation struct {
	ID        string `json:"id"`
	UserID    string `json:"userId"`
	Symbol    string `json:"symbol"`
	Side      string `json:"side"`
	Quantity  string `json:"quantity"`
	Price     string `json:"price"`
	LiqType   string `json:"liqType"`
	Status    string `json:"status"`
	Timestamp int64  `json:"timestamp"`
}

// FundingPayment represents a funding payment
type FundingPayment struct {
	ID        string `json:"id"`
	UserID    string `json:"userId"`
	Symbol    string `json:"symbol"`
	Side      string `json:"side"`
	Quantity  string `json:"quantity"`
	Rate      string `json:"rate"`
	Amount    string `json:"amount"`
	Timestamp int64  `json:"timestamp"`
}

// StartFundingProcessor starts the funding processor
func (s *SettlementService) StartFundingProcessor(hub *WSHub) {
	ticker := time.NewTicker(8 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.processFunding(hub)
		}
	}
}

func (s *SettlementService) processFunding(hub *WSHub) {
	log.Println("Processing funding payments...")

	s.mu.Lock()
	// Process all funding payments
	s.mu.Unlock()

	if hub != nil {
		hub.Broadcast(map[string]interface{}{
			"type": "funding",
			"data": map[string]string{
				"status": "processed",
			},
		})
	}
}

// StartLiquidationWatcher starts the liquidation watcher
func (s *SettlementService) StartLiquidationWatcher(positionSvc *PositionService, hub *WSHub) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.checkLiquidations(positionSvc, hub)
		}
	}
}

func (s *SettlementService) checkLiquidations(positionSvc *PositionService, hub *WSHub) {
	// Check for liquidatable positions
	// This would integrate with the Rust liquidation engine
}

// NotificationService handles notifications
type NotificationService struct {
	mu            sync.RWMutex
	notifications map[string][]*Notification
}

// NewNotificationService creates a new notification service
func NewNotificationService() *NotificationService {
	return &NotificationService{
		notifications: make(map[string][]*Notification),
	}
}

// Notification represents a notification
type Notification struct {
	ID        string `json:"id"`
	UserID    string `json:"userId"`
	Type      string `json:"type"`
	Title     string `json:"title"`
	Message   string `json:"message"`
	Read      bool   `json:"read"`
	Timestamp int64  `json:"timestamp"`
}

// SendNotification sends a notification to a user
func (n *NotificationService) SendNotification(ctx context.Context, userID, notifType, title, message string) {
	notif := &Notification{
		ID:        generateID(),
		UserID:    userID,
		Type:      notifType,
		Title:     title,
		Message:   message,
		Read:      false,
		Timestamp: time.Now().Unix(),
	}

	n.mu.Lock()
	n.notifications[userID] = append(n.notifications[userID], notif)
	n.mu.Unlock()
}

// GetNotifications gets notifications for a user
func (n *NotificationService) GetNotifications(ctx context.Context, userID string) ([]*Notification, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return n.notifications[userID], nil
}

// MarkRead marks a notification as read
func (n *NotificationService) MarkRead(ctx context.Context, userID, notifID string) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	for _, notif := range n.notifications[userID] {
		if notif.ID == notifID {
			notif.Read = true
			return nil
		}
	}

	return nil
}

// generateID generates a unique ID
func generateID() string {
	return time.Now().Format("20060102150405") + "_" + string(big.NewInt(time.Now().UnixNano()).String())
}
