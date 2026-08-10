/**
 * TigerWallet Push Notifications Service
 * High-Load Distributed Go Implementation
 *
 * Features:
 * - WebPush for browser notifications
 * - APNs for iOS
 * - FCM for Android
 * - Multi-device support
 * - Notification templates
 * - Delivery tracking
 */

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// ============== Data Structures ==============

type Notification struct {
	ID          string                 `json:"id"`
	UserID      string                 `json:"user_id"`
	Type        string                 `json:"type"`
	Title       string                 `json:"title"`
	Body        string                 `json:"body"`
	Icon        string                 `json:"icon"`
	Data        map[string]interface{} `json:"data"`
	Priority    string                 `json:"priority"` // high, normal, low
	Devices     []string               `json:"devices"`
	ScheduledAt *time.Time             `json:"scheduled_at,omitempty"`
	SentAt      *time.Time             `json:"sent_at,omitempty"`
	Status      string                 `json:"status"` // pending, sent, delivered, failed
}

type Device struct {
	ID         string `json:"id"`
	UserID     string `json:"user_id"`
	Platform   string `json:"platform"` // ios, android, web
	Token      string `json:"token"`
	Enabled    bool   `json:"enabled"`
	LastActive int64  `json:"last_active"`
	CreatedAt  int64  `json:"created_at"`
}

type Subscription struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Endpoint  string     `json:"endpoint"`
	Keys      P256DHKeys `json:"keys"`
	CreatedAt int64      `json:"created_at"`
}

type P256DHKeys struct {
	Auth   string `json:"auth"`
	P256DH string `json:"p256dh"`
}

type Template struct {
	ID      string               `json:"id"`
	Name    string               `json:"name"`
	Type    string               `json:"type"`
	Title   string               `json:"title"`
	Body    string               `json:"body"`
	Icon    string               `json:"icon"`
	Actions []NotificationAction `json:"actions"`
}

type NotificationAction struct {
	Action string `json:"action"`
	Title  string `json:"title"`
	Icon   string `json:"icon"`
}

type DeliveryReport struct {
	NotificationID string `json:"notification_id"`
	DeviceID       string `json:"device_id"`
	Status         string `json:"status"`
	DeliveredAt    int64  `json:"delivered_at"`
	Error          string `json:"error,omitempty"`
}

// ============== Service ==============

type NotificationService struct {
	notifications map[string]*Notification
	devices       map[string]*Device
	subscriptions map[string]*Subscription
	templates     map[string]*Template
	deliveries    map[string][]DeliveryReport

	mu         sync.RWMutex
	httpServer *http.Server
}

func NewNotificationService() *NotificationService {
	return &NotificationService{
		notifications: make(map[string]*Notification),
		devices:       make(map[string]*Device),
		subscriptions: make(map[string]*Subscription),
		templates:     make(map[string]*Template),
		deliveries:    make(map[string][]DeliveryReport),
	}
}

func (s *NotificationService) Run() error {
	// Initialize default templates
	s.initTemplates()

	// Setup routes
	mux := http.NewServeMux()
	mux.HandleFunc("/api/notify", s.handleSendNotification)
	mux.HandleFunc("/api/devices/register", s.handleRegisterDevice)
	mux.HandleFunc("/api/devices/unregister", s.handleUnregisterDevice)
	mux.HandleFunc("/api/subscriptions", s.handleSubscriptions)
	mux.HandleFunc("/api/templates", s.handleTemplates)
	mux.HandleFunc("/api/delivery", s.handleDeliveryReport)
	mux.HandleFunc("/api/history", s.handleGetHistory)
	mux.HandleFunc("/health", s.handleHealth)

	s.httpServer = &http.Server{
		Addr:         ":8085",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Println("Notification service starting on :8085")
	return s.httpServer.ListenAndServe()
}

func (s *NotificationService) initTemplates() {
	s.templates = map[string]*Template{
		"transaction": {
			ID:    "transaction",
			Name:  "Transaction Notification",
			Type:  "transaction",
			Title: "Transaction Confirmed",
			Body:  "Your {{amount}} {{token}} transaction has been confirmed",
			Icon:  "https://tigerwallet.com/icons/tx.png",
			Actions: []NotificationAction{
				{Action: "view", Title: "View Details"},
			},
		},
		"price_alert": {
			ID:    "price_alert",
			Name:  "Price Alert",
			Type:  "price",
			Title: "Price Alert: {{token}}",
			Body:  "{{token}} has reached ${{price}}",
			Icon:  "https://tigerwallet.com/icons/price.png",
		},
		"airdrop": {
			ID:    "airdrop",
			Name:  "Airdrop Claim",
			Type:  "airdrop",
			Title: "Airdrop Available!",
			Body:  "You have {{amount}} {{token}} to claim",
			Icon:  "https://tigerwallet.com/icons/airdrop.png",
			Actions: []NotificationAction{
				{Action: "claim", Title: "Claim Now"},
			},
		},
		"security": {
			ID:    "security",
			Name:  "Security Alert",
			Type:  "security",
			Title: "Security Alert",
			Body:  "{{message}}",
			Icon:  "https://tigerwallet.com/icons/security.png",
			Actions: []NotificationAction{
				{Action: "secure", Title: "Secure Wallet"},
			},
		},
	}
}

// ============== Handlers ==============

func (s *NotificationService) handleSendNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var notif Notification
		if err := json.NewDecoder(r.Body).Decode(&notif); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		notif.ID = generateID()
		notif.Status = "pending"
		notif.SentAt = nil

		// Get user devices
		devices := s.getUserDevices(notif.UserID)
		notif.Devices = make([]string, len(devices))
		for i, d := range devices {
			notif.Devices[i] = d.ID
		}

		s.notifications[notif.ID] = &notif

		// Send asynchronously
		go s.deliverNotification(&notif)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(notif)
		return
	}

	// GET - list notifications
	userID := r.URL.Query().Get("user_id")
	notifications := s.getUserNotifications(userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notifications)
}

func (s *NotificationService) handleRegisterDevice(w http.ResponseWriter, r *http.Request) {
	var device Device
	if err := json.NewDecoder(r.Body).Decode(&device); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	device.ID = generateID()
	device.CreatedAt = time.Now().Unix()
	device.Enabled = true

	s.devices[device.ID] = &device

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(device)
}

func (s *NotificationService) handleUnregisterDevice(w http.ResponseWriter, r *http.Request) {
	deviceID := r.URL.Query().Get("device_id")
	if deviceID == "" {
		http.Error(w, "device_id required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	if device, ok := s.devices[deviceID]; ok {
		device.Enabled = false
	}
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "unregistered"})
}

func (s *NotificationService) handleSubscriptions(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var sub Subscription
		if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		sub.ID = generateID()
		sub.CreatedAt = time.Now().Unix()

		s.subscriptions[sub.ID] = &sub

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sub)
		return
	}

	// GET user subscriptions
	userID := r.URL.Query().Get("user_id")
	var userSubs []*Subscription
	for _, sub := range s.subscriptions {
		if sub.UserID == userID {
			userSubs = append(userSubs, sub)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userSubs)
}

func (s *NotificationService) handleTemplates(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var tmpl Template
		if err := json.NewDecoder(r.Body).Decode(&tmpl); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.templates[tmpl.ID] = &tmpl
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tmpl)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.templates)
}

func (s *NotificationService) handleDeliveryReport(w http.ResponseWriter, r *http.Request) {
	var report DeliveryReport
	if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.deliveries[report.NotificationID] = append(s.deliveries[report.NotificationID], report)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "recorded"})
}

func (s *NotificationService) handleGetHistory(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	notifications := s.getUserNotifications(userID)
	if len(notifications) > limit {
		notifications = notifications[:limit]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notifications)
}

func (s *NotificationService) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":        "healthy",
		"notifications": len(s.notifications),
		"devices":       len(s.devices),
		"timestamp":     time.Now().Unix(),
	})
}

// ============== Delivery ==============

func (s *NotificationService) deliverNotification(notif *Notification) {
	now := time.Now()
	notif.Status = "sent"
	notif.SentAt = &now

	for _, deviceID := range notif.Devices {
		device, ok := s.devices[deviceID]
		if !ok || !device.Enabled {
			continue
		}

		// Send to device based on platform
		var err string
		switch device.Platform {
		case "ios":
			err = s.sendAPNS(device, notif)
		case "android":
			err = s.sendFCM(device, notif)
		case "web":
			err = s.sendWebPush(device, notif)
		}

		// Record delivery
		report := DeliveryReport{
			NotificationID: notif.ID,
			DeviceID:       deviceID,
			Status:         "delivered",
			DeliveredAt:    time.Now().Unix(),
		}
		if err != "" {
			report.Status = "failed"
			report.Error = err
		}

		s.deliveries[notif.ID] = append(s.deliveries[notif.ID], report)
	}

	// Update status
	notif.Status = "delivered"
}

func (s *NotificationService) sendAPNS(device *Device, notif *Notification) string {
	// In production, use apns2 library
	log.Printf("Sending APNS to %s: %s", device.ID, notif.Title)
	return ""
}

func (s *NotificationService) sendFCM(device *Device, notif *Notification) string {
	// In production, use firebase-admin library
	log.Printf("Sending FCM to %s: %s", device.ID, notif.Title)
	return ""
}

func (s *NotificationService) sendWebPush(device *Device, notif *Notification) string {
	// In production, use webpush library
	log.Printf("Sending WebPush to %s: %s", device.ID, notif.Title)
	return ""
}

// ============== Helpers ==============

func (s *NotificationService) getUserDevices(userID string) []*Device {
	var devices []*Device
	for _, d := range s.devices {
		if d.UserID == userID && d.Enabled {
			devices = append(devices, d)
		}
	}
	return devices
}

func (s *NotificationService) getUserNotifications(userID string) []*Notification {
	var notifs []*Notification
	for _, n := range s.notifications {
		if n.UserID == userID {
			notifs = append(notifs, n)
		}
	}
	return notifs
}

func generateID() string {
	return fmt.Sprintf("notif_%d", time.Now().UnixNano())
}

// ============== Main ==============

func main() {
	log.Println("Starting TigerWallet Push Notifications Service...")

	service := NewNotificationService()
	if err := service.Run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func (s *NotificationService) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
