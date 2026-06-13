package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

// User represents a user
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Username     string    `json:"username"`
	PasswordHash string   `json:"-"`
	CreatedAt   int64     `json:"createdAt"`
	LastLogin   int64     `json:"lastLogin"`
	Verified    bool      `json:"verified"`
	TwoFactor   bool      `json:"twoFactor"`
	Role       string    `json:"role"`
}

// Session represents an active session
type Session struct {
	SessionID   string    `json:"sessionId"`
	UserID     string    `json:"userId"`
	Token      string    `json:"token"`
	ExpiresAt  int64     `json:"expiresAt"`
	CreatedAt  int64     `json:"createdAt"`
	IPAddress  string    `json:"ipAddress"`
	Device     string    `json:"device"`
}

// Auth service
type AuthService struct {
	mu       sync.RWMutex
	users    map[string]*User
	sessions map[string]*Session
}

func NewAuthService() *AuthService {
	return &AuthService{
		users:    make(map[string]*User),
		sessions: make(map[string]*Session),
	}
}

func main() {
	router := mux.NewRouter()
	svc := NewAuthService()

	router.HandleFunc("/api/v1/auth/register", svc.register).Methods("POST")
	router.HandleFunc("/api/v1/auth/login", svc.login).Methods("POST")
	router.HandleFunc("/api/v1/auth/logout", svc.logout).Methods("POST")
	router.HandleFunc("/api/v1/auth/refresh", svc.refreshToken).Methods("POST")
	router.HandleFunc("/api/v1/auth/verify", svc.verifyEmail).Methods("POST")
	router.HandleFunc("/api/v1/auth/2fa/enable", svc.enable2FA).Methods("POST")
	router.HandleFunc("/api/v1/auth/2fa/verify", svc.verify2FA).Methods("POST")
	router.HandleFunc("/api/v1/auth/me", svc.getMe).Methods("GET")
	router.HandleFunc("/api/v1/auth/password/reset", svc.resetPassword).Methods("POST")

	router.HandleFunc("/health", healthCheck).Methods("GET")

	log.Println("Starting Auth Service on port 8002")
	log.Fatal(http.ListenAndServe(":8002", router))
}

func (s *AuthService) register(w http.ResponseWriter, r *http.Request) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Hash password
	hashed, err := bcrypt.GenerateFromPassword([]byte(user.PasswordHash), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	user.ID = fmt.Sprintf("user_%d", time.Now().UnixNano())
	user.PasswordHash = string(hashed)
	user.CreatedAt = time.Now().Unix()
	user.Verified = false
	user.Role = "user"

	s.mu.Lock()
	s.users[user.ID] = &user
	s.mu.Unlock()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"userId": user.ID,
		"status": "registered",
	})
}

func (s *AuthService) login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	var user *User
	for _, u := range s.users {
		if u.Email == req.Email {
			user = u
			break
		}
	}
	s.mu.RUnlock()

	if user == nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Create session
	session := Session{
		SessionID: fmt.Sprintf("sess_%d", time.Now().UnixNano()),
		UserID:   user.ID,
		Token:    fmt.Sprintf("tok_%d", time.Now().UnixNano()),
		ExpiresAt: time.Now().Add(24*time.Hour).Unix(),
		CreatedAt: time.Now().Unix(),
	}

	s.mu.Lock()
	s.sessions[session.SessionID] = &session
	s.mu.Unlock()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"token": session.Token,
		"user": map[string]interface{}{
			"id": user.ID,
			"email": user.Email,
			"username": user.Username,
		},
	})
}

func (s *AuthService) logout(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionID string `json:"sessionId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	delete(s.sessions, req.SessionID)
	s.mu.Unlock()

	json.NewEncoder(w).Encode(map[string]string{"status": "logged out"})
}

func (s *AuthService) refreshToken(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"token": fmt.Sprintf("tok_%d", time.Now().UnixNano())})
}

func (s *AuthService) verifyEmail(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "verified"})
}

func (s *AuthService) enable2FA(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]interface{}{
		"secret": "JBSWY3DPEHPK3PXP",
		"qr": "otpauth://totp/TigerWallet:user@example.com",
	})
}

func (s *AuthService) verify2FA(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "2FA enabled"})
}

func (s *AuthService) getMe(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id": "user_1",
		"email": "user@example.com",
		"username": "user",
	})
}

func (s *AuthService) resetPassword(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "password reset sent"})
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "healthy"})
}