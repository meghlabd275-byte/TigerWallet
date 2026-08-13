package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/tigerwallet/notifications/notifications"
)

// In-memory store of delivered notifications so the per-user GET and the
// per-notification "read" endpoints are operational rather than stubbed.
type notificationStore struct {
	notifications map[string]*storedNotification
	byUser        map[string][]string
}

type storedNotification struct {
	ID        string                        `json:"id"`
	UserID    string                        `json:"userId"`
	Title     string                        `json:"title"`
	Body      string                        `json:"body"`
	Read      bool                          `json:"read"`
	CreatedAt int64                         `json:"createdAt"`
	Data      map[string]string             `json:"data,omitempty"`
	Responses []*notifications.SendResponse `json:"responses,omitempty"`
}

func main() {
	cfg := &notifications.Config{Timeout: 30 * time.Second}
	svc := notifications.NewService(cfg)
	// Register built-in providers so /send is operational end-to-end.
	svc.RegisterProvider("fcm", notifications.NewFCMProvider(""))
	svc.RegisterProvider("apns", notifications.NewAPNSProvider("", "", ""))
	svc.RegisterProvider("webpush", notifications.NewWebPushProvider("", ""))

	store := &notificationStore{
		notifications: make(map[string]*storedNotification),
		byUser:        make(map[string][]string),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler("notifications"))

	mux.HandleFunc("POST /api/v1/notifications/register", func(w http.ResponseWriter, r *http.Request) {
		var sub notifications.Subscriber
		if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if sub.ID == "" {
			writeError(w, http.StatusBadRequest, "subscriber id is required")
			return
		}
		svc.Subscribe(&sub)
		writeJSON(w, http.StatusOK, map[string]string{"status": "registered", "subscriberId": sub.ID})
	})

	mux.HandleFunc("POST /api/v1/notifications/send", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			UserID string `json:"userId"`
			notifications.SendRequest
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if req.UserID == "" {
			writeError(w, http.StatusBadRequest, "userId is required")
			return
		}
		responses, err := svc.Send(r.Context(), req.UserID, &req.SendRequest)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		n := &storedNotification{
			ID:        "notif_" + time.Now().Format("20060102150405.000000"),
			UserID:    req.UserID,
			Title:     req.Title,
			Body:      req.Body,
			CreatedAt: time.Now().Unix(),
			Data:      req.Data,
			Responses: responses,
		}
		store.add(n)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"notificationId": n.ID,
			"responses":      responses,
		})
	})

	mux.HandleFunc("GET /api/v1/notifications/{userId}", func(w http.ResponseWriter, r *http.Request) {
		userID := r.PathValue("userId")
		ids := store.byUser[userID]
		out := make([]*storedNotification, 0, len(ids))
		for _, id := range ids {
			if n := store.notifications[id]; n != nil {
				out = append(out, n)
			}
		}
		writeJSON(w, http.StatusOK, out)
	})

	mux.HandleFunc("POST /api/v1/notifications/{id}/read", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		n, ok := store.notifications[id]
		if !ok {
			writeError(w, http.StatusNotFound, "notification not found")
			return
		}
		n.Read = true
		writeJSON(w, http.StatusOK, map[string]string{"status": "read", "id": id})
	})

	addr := ":" + port("8472")
	log.Printf("notifications service listening on %s", addr)
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func (s *notificationStore) add(n *storedNotification) {
	s.notifications[n.ID] = n
	s.byUser[n.UserID] = append(s.byUser[n.UserID], n.ID)
}

func port(def string) string {
	if p := os.Getenv("PORT"); p != "" {
		return p
	}
	return def
}

func healthHandler(service string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": service})
	}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
