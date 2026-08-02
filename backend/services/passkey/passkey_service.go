/**
 * TigerWallet Passkey Authentication Service
 * 
 * WebAuthn/FIDO2 implementation for passwordless authentication
 * Supports biometric authentication across all platforms
 */

package passkey

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// ============================================================================
// Types
// ============================================================================

// Credential represents a passkey credential
type Credential struct {
	ID                string             `json:"id"`
	UserID            string             `json:"user_id"`
	PublicKey         string             `json:"public_key"`
	AttestationType   string             `json:"attestation_type"`
	Transport         []string           `json:"transport"`
	Counter           uint32             `json:"counter"`
	CreatedAt         int64              `json:"created_at"`
	LastUsedAt        int64              `json:"last_used_at"`
	BackedUp          bool               `json:"backed_up"`
	Discoverable      bool               `json:"discoverable"`
}

// User represents a user with passkey credentials
type User struct {
	ID                string         `json:"id"`
	Username          string         `json:"username"`
	DisplayName       string         `json:"display_name"`
	Credentials       []*Credential  `json:"credentials"`
	CreatedAt         int64          `json:"created_at"`
	UpdatedAt         int64          `json:"updated_at"`
	Verified          bool           `json:"verified"`
	MFAMethods        []string       `json:"mfa_methods"`
}

// AuthenticationSession represents a WebAuthn authentication session
type AuthenticationSession struct {
	ID             string          `json:"id"`
	Challenge      string          `json:"challenge"`
	UserID         string          `json:"user_id"`
	RpID           string          `json:"rp_id"`
	RpOrigin       string          `json:"rp_origin"`
	Timeout        int64           `json:"timeout"`
	CreatedAt      int64           `json:"created_at"`
	ExpiresAt      int64           `json:"expires_at"`
	Used           bool             `json:"used"`
}

// RegistrationSession represents a WebAuthn registration session
type RegistrationSession struct {
	ID             string          `json:"id"`
	Challenge      string          `json:"challenge"`
	UserID         string          `json:"user_id"`
	RpID           string          `json:"rp_id"`
	RpOrigin       string          `json:"rp_origin"`
	Timeout        int64           `json:"timeout"`
	CreatedAt      int64           `json:"created_at"`
	ExpiresAt      int64           `json:"expires_at"`
	Used           bool             `json:"used"`
}

// PasskeyConfig represents passkey configuration
type PasskeyConfig struct {
	RPID            string   `json:"rp_id"`            // Relying Party ID
	RPName          string   `json:"rp_name"`          // Relying Party Name
	RPOrigin        string   `json:"rp_origin"`       // Relying Party Origin
	Timeout         int64    `json:"timeout"`           // Timeout in milliseconds
	ChallengeSize   int      `json:"challenge_size"`   // Challenge size in bytes
	CredentialSize  int      `json:"credential_size"`  // Credential ID size
	Authenticator   AuthenticatorConfig `json:"authenticator"` // Authenticator config
}

// AuthenticatorConfig represents authenticator configuration
type AuthenticatorConfig struct {
	RequireResidentKey bool     `json:"require_resident_key"`
	RequireUserVerification bool `json:"require_user_verification"`
	CredProtect        string   `json:"cred_protect"` // "low", "medium", "high"
}

// WebAuthnRequest represents a WebAuthn request
type WebAuthnRequest struct {
	Challenge    string   `json:"challenge"`
	Timeout      int64    `json:"timeout"`
	RPID         string   `json:"rp_id"`
	AllowCredentials []CredentialDescriptor `json:"allow_credentials"`
	UserVerification string `json:"user_verification"`
}

// WebAuthnResponse represents a WebAuthn response
type WebAuthnResponse struct {
	ID           string `json:"id"`
	RawID       string `json:"raw_id"`
	ClientDataJSON string `json:"client_data_json"`
	AuthenticatorData string `json:"authenticator_data"`
	Signature    string `json:"signature"`
	UserHandle   string `json:"user_handle"`
}

// CredentialDescriptor represents a credential descriptor
type CredentialDescriptor struct {
	Type       string `json:"type"`
	ID         string `json:"id"`
	Transports []string `json:"transports"`
}

// ClientData represents client data from WebAuthn
type ClientData struct {
	Type        string `json:"type"`
	Challenge   string `json:"challenge"`
	Origin      string `json:"origin"`
	CrossOrigin bool   `json:"crossOrigin"`
}

// AuthenticatorData represents authenticator data
type AuthenticatorData struct {
	RPIDHash   [32]byte
	Flags      byte
	Counter    uint32
	AAGUID     [16]byte
	// Credential data follows if flag bit 6 is set
}

// ============================================================================
// Passkey Service
// ============================================================================

// PasskeyService represents the passkey authentication service
type PasskeyService struct {
	mu            sync.RWMutex
	config        *PasskeyConfig
	users         map[string]*User
	authSessions  map[string]*AuthenticationSession
	regSessions   map[string]*RegistrationSession
}

// NewPasskeyService creates a new passkey service
func NewPasskeyService(config *PasskeyConfig) *PasskeyService {
	if config == nil {
		config = DefaultPasskeyConfig()
	}
	
	return &PasskeyService{
		config:       config,
		users:        make(map[string]*User),
		authSessions: make(map[string]*AuthenticationSession),
		regSessions:  make(map[string]*RegistrationSession),
	}
}

// DefaultPasskeyConfig returns default configuration
func DefaultPasskeyConfig() *PasskeyConfig {
	return &PasskeyConfig{
		RPID:           "tigerwallet.com",
		RPName:         "TigerWallet",
		RPOrigin:       "https://tigerwallet.com",
		Timeout:        60000,
		ChallengeSize:  32,
		CredentialSize: 64,
		Authenticator: AuthenticatorConfig{
			RequireResidentKey:     false,
			RequireUserVerification: true,
			CredProtect:           "high",
		},
	}
}

// ============================================================================
// User Management
// ============================================================================

// CreateUser creates a new user
func (s *PasskeyService) CreateUser(ctx context.Context, username, displayName string) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Check if user exists
	if _, exists := s.users[username]; exists {
		return nil, fmt.Errorf("user already exists")
	}
	
	user := &User{
		ID:          generateID(),
		Username:    username,
		DisplayName: displayName,
		Credentials: make([]*Credential, 0),
		CreatedAt:   time.Now().UnixMilli(),
		UpdatedAt:   time.Now().UnixMilli(),
		MFAMethods:  []string{"passkey"},
	}
	
	s.users[username] = user
	return user, nil
}

// GetUser gets a user by username
func (s *PasskeyService) GetUser(ctx context.Context, username string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	user, exists := s.users[username]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}
	
	return user, nil
}

// GetUserByID gets a user by ID
func (s *PasskeyService) GetUserByID(ctx context.Context, userID string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	for _, user := range s.users {
		if user.ID == userID {
			return user, nil
		}
	}
	
	return nil, fmt.Errorf("user not found")
}

// DeleteUser deletes a user
func (s *PasskeyService) DeleteUser(ctx context.Context, username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, exists := s.users[username]; !exists {
		return fmt.Errorf("user not found")
	}
	
	delete(s.users, username)
	return nil
}

// ============================================================================
// Registration
// ============================================================================

// BeginRegistration begins a passkey registration ceremony
func (s *PasskeyService) BeginRegistration(ctx context.Context, username string) (*RegistrationSession, *WebAuthnRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	user, exists := s.users[username]
	if !exists {
		return nil, nil, fmt.Errorf("user not found")
	}
	
	// Generate challenge
	challenge := generateChallenge(s.config.ChallengeSize)
	
	// Create registration session
	session := &RegistrationSession{
		ID:        generateID(),
		Challenge: challenge,
		UserID:    user.ID,
		RpID:     s.config.RPID,
		RpOrigin: s.config.RPOrigin,
		Timeout:  s.config.Timeout,
		CreatedAt: time.Now().UnixMilli(),
		ExpiresAt: time.Now().Add(time.Duration(s.config.Timeout) * time.Millisecond).UnixMilli(),
	}
	
	// Create WebAuthn request
	request := &WebAuthnRequest{
		Challenge: base64.RawURLEncoding.EncodeToString([]byte(challenge)),
		Timeout:   s.config.Timeout,
		RPID:      s.config.RPID,
		AllowCredentials: []CredentialDescriptor{},
		UserVerification: "preferred",
	}
	
	s.regSessions[session.ID] = session
	
	return session, request, nil
}

// CompleteRegistration completes a passkey registration
func (s *PasskeyService) CompleteRegistration(ctx context.Context, sessionID string, response *WebAuthnResponse) (*Credential, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	session, exists := s.regSessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}
	
	if session.Used {
		return nil, fmt.Errorf("session already used")
	}
	
	if time.Now().UnixMilli() > session.ExpiresAt {
		return nil, fmt.Errorf("session expired")
	}
	
	// Verify client data
	clientData, err := verifyClientData(response.ClientDataJSON, session.Challenge, session.RpOrigin)
	if err != nil {
		return nil, fmt.Errorf("client data verification failed: %w", err)
	}
	
	if clientData.Type != "webauthn.create" {
		return nil, fmt.Errorf("invalid client data type")
	}
	
	// Get user
	user, exists := s.users[session.UserID]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}
	
	// Create credential
	credential := &Credential{
		ID:              generateID(),
		UserID:          user.ID,
		PublicKey:       response.AuthenticatorData, // Simplified
		AttestationType:  "none",
		Transport:       []string{"hybrid"},
		Counter:         0,
		CreatedAt:       time.Now().UnixMilli(),
		LastUsedAt:      time.Now().UnixMilli(),
		BackedUp:        false,
		Discoverable:    true,
	}
	
	user.Credentials = append(user.Credentials, credential)
	user.UpdatedAt = time.Now().UnixMilli()
	
	session.Used = true
	
	return credential, nil
}

// ============================================================================
// Authentication
// ============================================================================

// BeginAuthentication begins a passkey authentication ceremony
func (s *PasskeyService) BeginAuthentication(ctx context.Context, username string) (*AuthenticationSession, *WebAuthnRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	user, exists := s.users[username]
	if !exists {
		return nil, nil, fmt.Errorf("user not found")
	}
	
	// Generate challenge
	challenge := generateChallenge(s.config.ChallengeSize)
	
	// Create authentication session
	session := &AuthenticationSession{
		ID:        generateID(),
		Challenge: challenge,
		UserID:    user.ID,
		RpID:     s.config.RPID,
		RpOrigin: s.config.RPOrigin,
		Timeout:  s.config.Timeout,
		CreatedAt: time.Now().UnixMilli(),
		ExpiresAt: time.Now().Add(time.Duration(s.config.Timeout) * time.Millisecond).UnixMilli(),
	}
	
	// Create credential descriptors
	allowCredentials := make([]CredentialDescriptor, len(user.Credentials))
	for i, cred := range user.Credentials {
		allowCredentials[i] = CredentialDescriptor{
			Type:       "public-key",
			ID:         cred.ID,
			Transports: cred.Transport,
		}
	}
	
	// Create WebAuthn request
	request := &WebAuthnRequest{
		Challenge: base64.RawURLEncoding.EncodeToString([]byte(challenge)),
		Timeout:   s.config.Timeout,
		RPID:      s.config.RPID,
		AllowCredentials: allowCredentials,
		UserVerification: "preferred",
	}
	
	s.authSessions[session.ID] = session
	
	return session, request, nil
}

// CompleteAuthentication completes a passkey authentication
func (s *PasskeyService) CompleteAuthentication(ctx context.Context, sessionID string, response *WebAuthnResponse) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	session, exists := s.authSessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}
	
	if session.Used {
		return nil, fmt.Errorf("session already used")
	}
	
	if time.Now().UnixMilli() > session.ExpiresAt {
		return nil, fmt.Errorf("session expired")
	}
	
	// Verify client data
	clientData, err := verifyClientData(response.ClientDataJSON, session.Challenge, session.RpOrigin)
	if err != nil {
		return nil, fmt.Errorf("client data verification failed: %w", err)
	}
	
	if clientData.Type != "webauthn.get" {
		return nil, fmt.Errorf("invalid client data type")
	}
	
	// Get user
	user, exists := s.users[session.UserID]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}
	
	// Find credential
	var credential *Credential
	for _, cred := range user.Credentials {
		if cred.ID == response.ID {
			credential = cred
			break
		}
	}
	
	if credential == nil {
		return nil, fmt.Errorf("credential not found")
	}
	
	// Update credential
	credential.LastUsedAt = time.Now().UnixMilli()
	credential.Counter++
	
	session.Used = true
	
	return user, nil
}

// ============================================================================
// Credential Management
// ============================================================================

// GetCredentials gets all credentials for a user
func (s *PasskeyService) GetCredentials(ctx context.Context, username string) ([]*Credential, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	user, exists := s.users[username]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}
	
	return user.Credentials, nil
}

// DeleteCredential deletes a credential
func (s *PasskeyService) DeleteCredential(ctx context.Context, username, credentialID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	user, exists := s.users[username]
	if !exists {
		return fmt.Errorf("user not found")
	}
	
	for i, cred := range user.Credentials {
		if cred.ID == credentialID {
			user.Credentials = append(user.Credentials[:i], user.Credentials[i+1:]...)
			user.UpdatedAt = time.Now().UnixMilli()
			return nil
		}
	}
	
	return fmt.Errorf("credential not found")
}

// UpdateCredential updates a credential
func (s *PasskeyService) UpdateCredential(ctx context.Context, username, credentialID string, updates map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	user, exists := s.users[username]
	if !exists {
		return fmt.Errorf("user not found")
	}
	
	for _, cred := range user.Credentials {
		if cred.ID == credentialID {
			if backedUp, ok := updates["backed_up"].(bool); ok {
				cred.BackedUp = backedUp
			}
			cred.LastUsedAt = time.Now().UnixMilli()
			user.UpdatedAt = time.Now().UnixMilli()
			return nil
		}
	}
	
	return fmt.Errorf("credential not found")
}

// ============================================================================
// Helper Functions
// ============================================================================

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func generateChallenge(size int) string {
	b := make([]byte, size)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func verifyClientData(clientDataJSON, expectedChallenge, expectedOrigin string) (*ClientData, error) {
	// Decode client data JSON
	decoded, err := base64.RawURLEncoding.DecodeString(clientDataJSON)
	if err != nil {
		return nil, fmt.Errorf("invalid client data encoding: %w", err)
	}
	
	var clientData ClientData
	if err := json.Unmarshal(decoded, &clientData); err != nil {
		return nil, fmt.Errorf("invalid client data JSON: %w", err)
	}
	
	// Verify challenge
	if subtle.ConstantTimeCompare([]byte(clientData.Challenge), []byte(expectedChallenge)) != 1 {
		return nil, fmt.Errorf("challenge mismatch")
	}
	
	// Verify origin
	if clientData.Origin != expectedOrigin {
		return nil, fmt.Errorf("origin mismatch: got %s, expected %s", clientData.Origin, expectedOrigin)
	}
	
	return &clientData, nil
}

func verifyAuthenticatorData(authData []byte, rpID string) (*AuthenticatorData, error) {
	if len(authData) < 37 {
		return nil, fmt.Errorf("authenticator data too short")
	}
	
	var auth AuthenticatorData
	copy(auth.RPIDHash[:], authData[:32])
	auth.Flags = authData[32]
	auth.Counter = uint32(authData[33])<<24 | uint32(authData[34])<<16 | uint32(authData[35])<<8 | uint32(authData[36])
	
	return &auth, nil
}

func computeSHA256(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

// ============================================================================
// PasskeyServiceServer (gRPC integration)
// ============================================================================

// PasskeyServiceServer represents the gRPC server
type PasskeyServiceServer struct {
	UnimplementedPasskeyServiceServer
	service *PasskeyService
}

// NewPasskeyServiceServer creates a new server
func NewPasskeyServiceServer(service *PasskeyService) *PasskeyServiceServer {
	return &PasskeyServiceServer{
		service: service,
	}
}

// RegisterService registers the service with gRPC
func (s *PasskeyServiceServer) RegisterService() {
	// In production, register with gRPC server
	// pb.RegisterPasskeyServiceServer(grpcServer, s)
}
