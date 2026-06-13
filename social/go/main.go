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

// User profile
type Profile struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	DisplayName string   `json:"displayName"`
	Avatar      string   `json:"avatar"`
	Bio         string   `json:"bio"`
	Followers   int     `json:"followers"`
	Following   int     `json:"following"`
	Wallet      string   `json:"wallet"`
	Verified    bool    `json:"verified"`
	CreatedAt   int64   `json:"createdAt"`
}

// Follow relationship
type Follow struct {
	Follower string `json:"follower"`
	Followee string `json:"followee"`
	Time    int64  `json:"time"`
}

// Group chat
type Group struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Members   []string `json:"members"`
	Admin    string   `json:"admin"`
	CreatedAt int64    `json:"createdAt"`
}

// Message
type Message struct {
	ID        string `json:"id"`
	GroupID  string `json:"groupId"`
	Sender   string `json:"sender"`
	Content  string `json:"content"`
	Time     int64  `json:"time"`
}

// SocialService
type SocialService struct {
	mu       sync.RWMutex
	profiles map[string]*Profile
	groups   map[string]*Group
	messages map[string][]*Message
}

func NewSocialService() *SocialService {
	return &SocialService{
		profiles: make(map[string]*Profile),
		groups:   make(map[string]*Group),
		messages: make(map[string][]*Message),
	}
}

func main() {
	router := mux.NewRouter()
	svc := NewSocialService()

	// Profile routes
	router.HandleFunc("/api/v1/profiles", svc.createProfile).Methods("POST")
	router.HandleFunc("/api/v1/profiles/{id}", svc.getProfile).Methods("GET")
	router.HandleFunc("/api/v1/profiles/{id}", svc.updateProfile).Methods("PUT")

	// Follow routes
	router.HandleFunc("/api/v1/follow", svc.follow).Methods("POST")
	router.HandleFunc("/api/v1/follow/{id}", svc.unfollow).Methods("DELETE")
	router.HandleFunc("/api/v1/followers/{id}", svc.getFollowers).Methods("GET")
	router.HandleFunc("/api/v1/following/{id}", svc.getFollowing).Methods("GET")

	// Group routes
	router.HandleFunc("/api/v1/groups", svc.createGroup).Methods("POST")
	router.HandleFunc("/api/v1/groups/{id}", svc.getGroup).Methods("GET")
	router.HandleFunc("/api/v1/groups/{id}/messages", svc.sendMessage).Methods("POST")
	router.HandleFunc("/api/v1/groups/{id}/messages", svc.getMessages).Methods("GET")

	router.HandleFunc("/health", healthCheck).Methods("GET")

	log.Println("Starting Social Layer on port 8091")
	log.Fatal(http.ListenAndServe(":8091", router))
}

func (s *SocialService) createProfile(w http.ResponseWriter, r *http.Request) {
	var profile Profile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	profile.ID = fmt.Sprintf("user_%d", time.Now().UnixNano())
	profile.CreatedAt = time.Now().Unix()

	s.mu.Lock()
	s.profiles[profile.ID] = &profile
	s.mu.Unlock()

	json.NewEncoder(w).Encode(profile)
}

func (s *SocialService) getProfile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	s.mu.RLock()
	defer s.mu.RUnlock()

	if profile, ok := s.profiles[id]; ok {
		json.NewEncoder(w).Encode(profile)
	} else {
		http.Error(w, "Profile not found", http.StatusNotFound)
	}
}

func (s *SocialService) updateProfile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var profile Profile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	if existing, ok := s.profiles[id]; ok {
		existing.DisplayName = profile.DisplayName
		existing.Avatar = profile.Avatar
		existing.Bio = profile.Bio
		json.NewEncoder(w).Encode(existing)
	} else {
		http.Error(w, "Profile not found", http.StatusNotFound)
	}
	s.mu.Unlock()
}

func (s *SocialService) follow(w http.ResponseWriter, r *http.Request) {
	var follow Follow
	if err := json.NewDecoder(r.Body).Decode(&follow); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	follow.Time = time.Now().Unix()

	s.mu.Lock()
	if follower, ok := s.profiles[follow.Follower]; ok {
		follower.Following++
	}
	if followee, ok := s.profiles[follow.Followee]; ok {
		followee.Followers++
	}
	s.mu.Unlock()

	json.NewEncoder(w).Encode(map[string]string{"status": "followed"})
}

func (s *SocialService) unfollow(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "unfollowed"})
}

func (s *SocialService) getFollowers(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode([]string{})
}

func (s *SocialService) getFollowing(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode([]string{})
}

func (s *SocialService) createGroup(w http.ResponseWriter, r *http.Request) {
	var group Group
	if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	group.ID = fmt.Sprintf("group_%d", time.Now().UnixNano())
	group.CreatedAt = time.Now().Unix()

	s.mu.Lock()
	s.groups[group.ID] = &group
	s.mu.Unlock()

	json.NewEncoder(w).Encode(group)
}

func (s *SocialService) getGroup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	s.mu.RLock()
	defer s.mu.RUnlock()

	if group, ok := s.groups[id]; ok {
		json.NewEncoder(w).Encode(group)
	} else {
		http.Error(w, "Group not found", http.StatusNotFound)
	}
}

func (s *SocialService) sendMessage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupID := vars["id"]

	var msg Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	msg.ID = fmt.Sprintf("msg_%d", time.Now().UnixNano())
	msg.GroupID = groupID
	msg.Time = time.Now().Unix()

	s.mu.Lock()
	s.messages[groupID] = append(s.messages[groupID], &msg)
	s.mu.Unlock()

	json.NewEncoder(w).Encode(msg)
}

func (s *SocialService) getMessages(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupID := vars["id"]

	s.mu.RLock()
	defer s.mu.RUnlock()

	json.NewEncoder(w).Encode(s.messages[groupID])
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "healthy"})
}