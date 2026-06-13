package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

// Notification type
type Notification struct {
	ID        string `json:"id"`
	UserID   string `json:"userId"`
	Type    string `json:"type"`
	Title   string `json:"title"`
	Message string `json:"message"`
	Read    bool   `json:"read"`
	Data    map[string]interface{} `json:"data,omitempty"`
	CreatedAt int64 `json:"createdAt"`
}

// Push subscription
type PushSubscription struct {
	ID        string `json:"id"`
	UserID   string `json:"userId"`
	Endpoint string `json:"endpoint"`
	Keys     map[string]string `json:"keys"`
	Enabled  bool   `json:"enabled"`
}

// Notification service
type NotificationService struct {
	mu            sync.RWMutex
	notifications map[string][]Notification
	subscriptions  map[string][]PushSubscription
}

func NewNotificationService() *NotificationService {
	return &NotificationService{
		notifications: make(map[string][]Notification),
		subscriptions:  make(map[string][]PushSubscription),
	}
}

func main() {
	router := mux.NewRouter()
	svc := NewNotificationService()

	router.HandleFunc("/api/v1/notifications", svc.getNotifications).Methods("GET")
	router.HandleFunc("/api/v1/notifications/{id}/read", svc.markRead).Methods("PUT")
	router.HandleFunc("/api/v1/notifications/read-all", svc.markAllRead).Methods("PUT")
	router.HandleFunc("/api/v1/notifications/subscribe", svc.subscribePush).Methods("POST")
	router.HandleFunc("/api/v1/notifications/unsubscribe", svc.unsubscribePush).Methods("POST")
	router.HandleFunc("/api/v1/notifications/send", svc.sendNotification).Methods("POST")
	router.HandleFunc("/api/v1/notifications/settings", svc.getSettings).Methods("GET")
	router.HandleFunc("/api/v1/notifications/settings", svc.updateSettings).Methods("PUT")

	router.HandleFunc("/health", healthCheck).Methods("GET")

	log.Println("Starting Notification Service on port 8004")
	log.Fatal(http.ListenAndServe(":8004", router))
}

func (s *NotificationService) getNotifications(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")
	unreadOnly := r.URL.Query().Get("unread") == "true"

	s.mu.RLock()
	defer s.mu.RUnlock()

	notifications := s.notifications[userID]
	if unreadOnly {
		var unread []Notification
		for _, n := range notifications {
			if !n.Read {
				unread = append(unread, n)
			}
		}
		notifications = unread
	}

	json.NewEncoder(w).Encode(notifications)
}

func (s *NotificationService) markRead(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req struct {
		UserID string `json:"userId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	notifications := s.notifications[req.UserID]
	for i, n := range notifications {
		if n.ID == id {
			notifications[i].Read = true
			json.NewEncoder(w).Encode(map[string]string{"status": "marked"})
			return
		}
	}

	http.Error(w, "Notification not found", http.StatusNotFound)
}

func (s *NotificationService) markAllRead(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID string `json:"userId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	notifications := s.notifications[req.UserID]
	for i := range notifications {
		notifications[i].Read = true
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "all marked"})
}

func (s *NotificationService) subscribePush(w http.ResponseWriter, r *http.Request) {
	var sub PushSubscription
	if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sub.ID = fmt.Sprintf("sub_%d", time.Now().UnixNano())

	s.mu.Lock()
	s.subscriptions[sub.UserID] = append(s.subscriptions[sub.UserID], sub)
	s.mu.Unlock()

	json.NewEncoder(w).Encode(sub)
}

func (s *NotificationService) unsubscribePush(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID string `json:"userId"`
		Endpoint string `json:"endpoint"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	subs := s.subscriptions[req.UserID]
	for i, sub := range subs {
		if sub.Endpoint == req.Endpoint {
			s.subscriptions[req.UserID] = append(subs[:i], subs[i+1:]...)
			break
		}
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "unsubscribed"})
}

func (s *NotificationService) sendNotification(w http.ResponseWriter, r *http.Request) {
	var notification Notification
	if err := json.NewDecoder(r.Body).Decode(&notification); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	notification.ID = fmt.Sprintf("notif_%d", time.Now().UnixNano())
	notification.CreatedAt = time.Now().Unix()

	s.mu.Lock()
	s.notifications[notification.UserID] = append(s.notifications[notification.UserID], notification)
	s.mu.Unlock()

	json.NewEncoder(w).Encode(notification)
}

func (s *NotificationService) getSettings(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]interface{}{
		"push": true,
		"email": true,
		"sms": false,
		"priceAlerts": true,
		"news": true,
		"security": true,
	})
}

func (s *NotificationService) updateSettings(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "healthy"})
}