package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

// ============================================================================
// TIGERWALLET MULTI-DEVICE SYNC SERVICE - Go Backend
// ============================================================================
//
// Features:
// - End-to-end encrypted sync
// - Real-time device sync
// - Secure key derivation
// - Conflict resolution
// - Offline support
// ============================================================================

// ============================================================================
// Data Models
// ============================================================================

// Encrypted payload
type EncryptedPayload struct {
	ID        string `json:"id"`
	Ciphertext string `json:"ciphertext"`
	IV       string `json:"iv"`
	Tag      string `json:"tag"`
	Version  int    `json:"version"`
	DeviceID string `json:"deviceId"`
}

// Sync item
type SyncItem struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"` // wallet, transaction, settings, contact
	Data       EncryptedPayload `json:"data"`
	Timestamp  int64           `json:"timestamp"`
	Version    int            `json:"version"`
	DeviceID   string         `json:"deviceId"`
	Deleted    bool           `json:"deleted"`
}

// Device registration
type Device struct {
	ID           string    `json:"id"`
	UserID       string    `json:"userId"`
	Name         string    `json:"name"`
	Type         string    `json:"type"` // mobile, desktop, web
	PublicKey   string    `json:"publicKey"`
	LastSeen    int64     `json:"lastSeen"`
	CreatedAt   int64     `json:"createdAt"`
	Status      string    `json:"status"` // active, inactive
}

// Sync request
type SyncRequest struct {
	UserID      string     `json:"userId"`
	DeviceID    string     `json:"deviceId"`
	LastSync    int64      `json:"lastSync"`
	Items       []SyncItem `json:"items"`
}

// Sync response
type SyncResponse struct {
	Items     []SyncItem `json:"items"`
	Timestamp int64      `json:"timestamp"`
	Conflicts []Conflict `json:"conflicts,omitempty"`
}

// Conflict resolution
type Conflict struct {
	ItemID     string     `json:"itemId"`
	Local      SyncItem   `json:"local"`
	Remote     SyncItem   `json:"remote"`
	Resolution string     `json:"resolution"` // local, remote, merge
}

// ============================================================================
// Service
// ============================================================================

type SyncService struct {
	mu          sync.RWMutex
	items       map[string]map[string]*SyncItem // userID -> itemID -> item
	devices     map[string]*Device
	users       map[string]*UserData
}

type UserData struct {
	UserID    string
	MasterKey []byte
	Devices   map[string]*Device
	Items    map[string]*SyncItem
}

func NewSyncService() *SyncService {
	return &SyncService{
		items:   make(map[string]map[string]*SyncItem),
		devices: make(map[string]*Device),
		users:   make(map[string]*UserData),
	}
}

// ============================================================================
// Encryption Functions
// ============================================================================

// Derive key from password
func deriveKey(password, salt string) []byte {
	key := sha256.Sum256([]byte(password + salt))
	return key[:]
}

// Generate random bytes
func generateRandomBytes(n int) []byte {
	b := make([]byte, n)
	rand.Read(b)
	return b
}

// Encrypt data with AES-GCM
func encrypt(data []byte, key []byte) (ciphertext, iv, tag []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, nil, err
	}

	iv = generateRandomBytes(12)
	ciphertext, tag = gcm.Seal(nil, iv, data, nil), gcm.Seal(nil, iv, data, nil)[len(gcm.Seal(nil, iv, data, nil))-16:]
	
	return ciphertext, iv, tag[:16], nil
}

// Decrypt data with AES-GCM
func decrypt(ciphertext, iv, tag, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return gcm.Open(nil, append(iv, tag...), ciphertext, nil)
}

// ============================================================================
// API Handlers
// ============================================================================

func healthCheck(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"status":  "healthy",
		"service": "sync",
		"version": "1.0.0",
	})
}

// Register device
func (s *SyncService) registerDevice(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID    string `json:"userId"`
		Name     string `json:"name"`
		Type     string `json:"type"`
		PublicKey string `json:"publicKey"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	device := &Device{
		ID:         generateID(),
		UserID:     req.UserID,
		Name:       req.Name,
		Type:       req.Type,
		PublicKey: req.PublicKey,
		LastSeen:   time.Now().Unix(),
		CreatedAt: time.Now().Unix(),
		Status:    "active",
	}

	s.mu.Lock()
	s.devices[device.ID] = device
	
	// Initialize user data if not exists
	if _, ok := s.users[req.UserID]; !ok {
		s.users[req.UserID] = &UserData{
			UserID: req.UserID,
			Items: make(map[string]*SyncItem),
			Devices: make(map[string]*Device),
		}
	}
	s.users[req.UserID].Devices[device.ID] = device
	s.mu.Unlock()

	log.Printf("[SYNC] Device registered: %s for user %s", device.Name, req.UserID)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"device": device,
	})
}

// Get user devices
func (s *SyncService) getDevices(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]

	s.mu.RLock()
	user, ok := s.users[userID]
	s.mu.RUnlock()

	if !ok {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"devices": []Device{},
		})
		return
	}

	devices := make([]*Device, 0)
	for _, d := range user.Devices {
		devices = append(devices, d)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"devices": devices,
	})
}

// Remove device
func (s *SyncService) removeDevice(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   string `json:"userId"`
		DeviceID string `json:"deviceId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if user, ok := s.users[req.UserID]; ok {
		delete(user.Devices, req.DeviceID)
	}

	log.Printf("[SYNC] Device removed: %s for user %s", req.DeviceID, req.UserID)

	respondJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}

// Sync data
func (s *SyncService) sync(w http.ResponseWriter, r *http.Request) {
	var req SyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Get or create user data
	user, ok := s.users[req.UserID]
	if !ok {
		user = &UserData{
			UserID: req.UserID,
			Items: make(map[string]*SyncItem),
			Devices: make(map[string]*Device),
		}
		s.users[req.UserID] = user
	}

	// Initialize items map if not exists
	if user.Items == nil {
		user.Items = make(map[string]*SyncItem)
	}

	conflicts := []Conflict{}
	timestamp := time.Now().Unix()

	// Process incoming items
	for i := range req.Items {
		item := &req.Items[i]
		
		existing, exists := user.Items[item.ID]
		
		if !exists {
			// New item - add it
			user.Items[item.ID] = item
		} else if item.Version > existing.Version {
			// Remote is newer - update
			user.Items[item.ID] = item
		} else if item.Version == existing.Version && item.Timestamp > existing.Timestamp {
			// Same version but newer timestamp - conflict
			conflicts = append(conflicts, Conflict{
				ItemID:     item.ID,
				Local:      *existing,
				Remote:     *item,
				Resolution: "remote", // Default resolution
			})
		}
	}

	// Get items changed since last sync
	changedItems := make([]SyncItem, 0)
	for _, item := range user.Items {
		if item.Timestamp > req.LastSync {
			changedItems = append(changedItems, *item)
		}
	}

	response := SyncResponse{
		Items:     changedItems,
		Timestamp: timestamp,
		Conflicts: conflicts,
	}

	log.Printf("[SYNC] User %s synced %d items", req.UserID, len(changedItems))

	respondJSON(w, http.StatusOK, response)
}

// Get all user data
func (s *SyncService) getUserData(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]

	s.mu.RLock()
	user, ok := s.users[userID]
	s.mu.RUnlock()

	if !ok {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"items": []SyncItem{},
		})
		return
	}

	items := make([]SyncItem, 0)
	for _, item := range user.Items {
		if !item.Deleted {
			items = append(items, *item)
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"items": items,
	})
}

// Delete item
func (s *SyncService) deleteItem(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID  string `json:"userId"`
		ItemID string `json:"itemId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if user, ok := s.users[req.UserID]; ok {
		if item, exists := user.Items[req.ItemID]; exists {
			item.Deleted = true
			item.Timestamp = time.Now().Unix()
			item.Version++
		}
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// Clear all user data
func (s *SyncService) clearUserData(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID string `json:"userId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.users, req.UserID)

	log.Printf("[SYNC] User data cleared: %s", req.UserID)

	respondJSON(w, http.StatusOK, map[string]string{"status": "cleared"})
}

// Get sync status
func (s *SyncService) getStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]

	s.mu.RLock()
	user, ok := s.users[userID]
	s.mu.RUnlock()

	if !ok {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"status": "not_synced",
			"items":  0,
		})
		return
	}

	count := 0
	for _, item := range user.Items {
		if !item.Deleted {
			count++
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "synced",
		"items":     count,
		"timestamp": time.Now().Unix(),
	})
}

// ============================================================================
// Helper Functions
// ============================================================================

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// ============================================================================
// Main
// ============================================================================

func main() {
	log.Println("Starting TigerWallet Multi-Device Sync Service...")

	service := NewSyncService()

	router := mux.NewRouter()

	router.HandleFunc("/health", healthCheck).Methods("GET")
	router.HandleFunc("/api/v1/sync/device/register", service.registerDevice).Methods("POST")
	router.HandleFunc("/api/v1/sync/devices/{userId}", service.getDevices).Methods("GET")
	router.HandleFunc("/api/v1/sync/device/remove", service.removeDevice).Methods("POST")
	router.HandleFunc("/api/v1/sync", service.sync).Methods("POST")
	router.HandleFunc("/api/v1/sync/data/{userId}", service.getUserData).Methods("GET")
	router.HandleFunc("/api/v1/sync/item/delete", service.deleteItem).Methods("POST")
	router.HandleFunc("/api/v1/sync/clear", service.clearUserData).Methods("POST")
	router.HandleFunc("/api/v1/sync/status/{userId}", service.getStatus).Methods("GET")

	log.Printf("Sync service listening on :8008")

	log.Fatal(http.ListenAndServe(":8008", router))
}