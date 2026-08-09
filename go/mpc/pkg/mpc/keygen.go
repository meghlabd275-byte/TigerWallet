/**
 * Enterprise MPC Infrastructure - Key Generation
 * 
 * Multi-party computation for distributed key generation
 * Supports threshold signatures and HSM integration
 */

package mpc

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto/secp256k1"
)

const (
	// Threshold parameters
	DefaultThreshold   = 2 // Minimum signers required
	DefaultTotalShares = 3 // Total number of shares
	
	// Key sizes
	KeySize256 = 32
	KeySize512 = 64
	
	// Protocol versions
	ProtocolV1 = "1.0"
	ProtocolV2 = "2.0"
)

// CurveParameters represents the elliptic curve for MPC
type CurveParameters struct {
	Curve  *secp256k1.BitCurve
	Name   string
	Prefix string
	P      *big.Int
	N      *big.Int
	Gx, Gy *big.Int
}

// GetSecp256k1Curve returns secp256k1 curve parameters (the curve used by Ethereum).
func GetSecp256k1Curve() *CurveParameters {
	c := secp256k1.S256()
	return &CurveParameters{
		Curve:  c,
		Name:   "secp256k1",
		Prefix: "02",
		P:      new(big.Int).Set(c.Params().P),
		N:      new(big.Int).Set(c.Params().N),
		Gx:     new(big.Int).Set(c.Params().Gx),
		Gy:     new(big.Int).Set(c.Params().Gy),
	}
}

// GetP256Curve returns secp256k1 curve parameters (P-256 is not used for
// Ethereum-compatible signing; secp256k1 is used instead for real ECDSA).
func GetP256Curve() *CurveParameters {
	return GetSecp256k1Curve()
}

// GetP384Curve returns secp256k1 curve parameters (kept for API compatibility).
func GetP384Curve() *CurveParameters {
	return GetSecp256k1Curve()
}

// KeyGenerationRequest represents a request for distributed key generation
type KeyGenerationRequest struct {
	ID          string                 `json:"id"`
	Protocol    string                 `json:"protocol"`
	Threshold   int                    `json:"threshold"`
	TotalShares int                    `json:"total_shares"`
	Curve       string                 `json:"curve"`
	Metadata    map[string]interface{} `json:"metadata"`
	Timestamp   int64                  `json:"timestamp"`
	SignerIDs   []string               `json:"signer_ids"`
}

// KeyGenerationResponse represents the response from key generation
type KeyGenerationResponse struct {
	ID             string   `json:"id"`
	PublicKey      string   `json:"public_key"`
	KeyShare       string   `json:"key_share,omitempty"`
	VerifierShare  string   `json:"verifier_share"`
	Threshold      int      `json:"threshold"`
	TotalShares    int      `json:"total_shares"`
	SignerID       string   `json:"signer_id"`
	ChainCode      string   `json:"chain_code"`
	Protocol       string   `json:"protocol"`
	CreatedAt      int64    `json:"created_at"`
	Commitment     string   `json:"commitment"`
}

// KeyShare represents a party's share of the secret key
type KeyShare struct {
	ID            string   `json:"id"`
	ShareID       string   `json:"share_id"`
	SignerID      string   `json:"signer_id"`
	Share         *big.Int `json:"share"`
	Verifier      *big.Int `json:"verifier"`
	ChainCode     *big.Int `json:"chain_code"`
	PublicKey     string   `json:"public_key"`
	Index         int      `json:"index"`
	Threshold     int      `json:"threshold"`
	TotalShares   int      `json:"total_shares"`
	CreatedAt     int64    `json:"created_at"`
	Encrypted     bool     `json:"encrypted"`
	EncryptionKey string   `json:"encryption_key,omitempty"`
}

// KeyGenerationSession manages the distributed key generation process
type KeyGenerationSession struct {
	mu                sync.RWMutex
	ID                string
	Protocol          string
	Threshold         int
	TotalShares       int
	Curve             *CurveParameters
	Status            string
	Participants      map[string]*Participant
	PublicKey         *big.Int
	Commitments       map[string]string
	ReceivedShares    map[string]*KeyShare
	CompletedChannels map[string]chan *KeyGenerationResponse
	CreatedAt         int64
	ExpiresAt         int64
	Metadata          map[string]interface{}
}

// Participant represents a party in the MPC protocol
type Participant struct {
	ID        string    `json:"id"`
	Index     int       `json:"index"`
	Status    string    `json:"status"`
	JoinedAt  int64     `json:"joined_at"`
	LastSeen  int64     `json:"last_seen"`
	IPAddress string    `json:"ip_address"`
	PubKey    string    `json:"pub_key"`
}

// NewKeyGenerationSession creates a new key generation session
func NewKeyGenerationSession(req *KeyGenerationRequest) *KeyGenerationSession {
	curve := GetP256Curve()
	if req.Curve == "P-384" {
		curve = GetP384Curve()
	}
	
	return &KeyGenerationSession{
		ID:                req.ID,
		Protocol:          req.Protocol,
		Threshold:         req.Threshold,
		TotalShares:       req.TotalShares,
		Curve:             curve,
		Status:            "pending",
		Participants:      make(map[string]*Participant),
		Commitments:       make(map[string]string),
		ReceivedShares:    make(map[string]*KeyShare),
		CompletedChannels: make(map[string]chan *KeyGenerationResponse),
		CreatedAt:         time.Now().UnixMilli(),
		ExpiresAt:         time.Now().Add(10 * time.Minute).UnixMilli(),
		Metadata:          req.Metadata,
	}
}

// AddParticipant adds a participant to the session
func (s *KeyGenerationSession) AddParticipant(p *Participant) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if len(s.Participants) >= s.TotalShares {
		return fmt.Errorf("session is full")
	}
	
	if _, exists := s.Participants[p.ID]; exists {
		return fmt.Errorf("participant already exists")
	}
	
	p.Index = len(s.Participants) + 1
	p.Status = "active"
	p.JoinedAt = time.Now().UnixMilli()
	p.LastSeen = time.Now().UnixMilli()
	
	s.Participants[p.ID] = p
	
	// Create completion channel
	s.CompletedChannels[p.ID] = make(chan *KeyGenerationResponse, 1)
	
	// Check if all participants joined
	if len(s.Participants) == s.TotalShares {
		s.Status = "ready"
	}
	
	return nil
}

// GenerateShare generates a key share for a participant
func (s *KeyGenerationSession) GenerateShare(signerID string) (*KeyShare, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	participant, exists := s.Participants[signerID]
	if !exists {
		return nil, fmt.Errorf("participant not found")
	}
	
	// Generate random share in [1, N-1]
	share, err := rand.Int(rand.Reader, new(big.Int).Sub(s.Curve.N, big.NewInt(1)))
	if err != nil {
		return nil, fmt.Errorf("failed to generate share: %w", err)
	}
	share.Add(share, big.NewInt(1))
	
	// Verifier is the public point (public key) corresponding to this share:
	// V = share * G, computed on the real secp256k1 curve.
	vx, vy := s.Curve.Curve.ScalarBaseMult(share.Bytes())
	verifierBytes := secp256k1.CompressPubkey(vx, vy)
	verifier := new(big.Int).SetBytes(verifierBytes)
	
	// Generate chain code
	chainCode, err := rand.Int(rand.Reader, big.NewInt(0).Lsh(big.NewInt(1), 256))
	if err != nil {
		return nil, fmt.Errorf("failed to generate chain code: %w", err)
	}
	
	keyShare := &KeyShare{
		ID:           generateID(),
		ShareID:      fmt.Sprintf("%s-share-%d", s.ID, participant.Index),
		SignerID:     signerID,
		Share:        share,
		Verifier:     verifier,
		ChainCode:    chainCode,
		PublicKey:    s.PublicKey.String(),
		Index:        participant.Index,
		Threshold:    s.Threshold,
		TotalShares:  s.TotalShares,
		CreatedAt:    time.Now().UnixMilli(),
		Encrypted:    false,
	}
	
	s.ReceivedShares[signerID] = keyShare
	
	return keyShare, nil
}

// ComputePublicKey computes the collective public key from commitments
func (s *KeyGenerationSession) ComputePublicKey(commitments map[string]string) (*big.Int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Verify all commitments are present
	if len(commitments) < s.Threshold {
		return nil, fmt.Errorf("insufficient commitments")
	}
	
	// Compute the collective public key by summing each participant's public
	// share point (V_i = share_i * G). The resulting point is the group public
	// key Y = sum(V_i), which is what reconstructed secrets sign against.
	var sumX, sumY *big.Int
	for _, share := range s.ReceivedShares {
		sx, sy := s.Curve.Curve.ScalarBaseMult(share.Share.Bytes())
		if sumX == nil {
			sumX, sumY = sx, sy
		} else {
			sumX, sumY = s.Curve.Curve.Add(sumX, sumY, sx, sy)
		}
	}
	if sumX == nil {
		// No shares yet: fall back to the generator point.
		sumX, sumY = s.Curve.Curve.ScalarBaseMult(big.NewInt(1).Bytes())
	}
	pubBytes := secp256k1.CompressPubkey(sumX, sumY)
	s.PublicKey = new(big.Int).SetBytes(pubBytes)
	
	return s.PublicKey, nil
}

// GetCompletionChannel returns the channel for a participant's completion
func (s *KeyGenerationSession) GetCompletionChannel(signerID string) chan *KeyGenerationResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if ch, exists := s.CompletedChannels[signerID]; exists {
		return ch
	}
	
	return nil
}

// CompleteSession finalizes the key generation session
func (s *KeyGenerationSession) CompleteSession() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if len(s.ReceivedShares) < s.Threshold {
		return fmt.Errorf("insufficient shares collected")
	}
	
	s.Status = "completed"
	
	// Notify all participants
	for _, ch := range s.CompletedChannels {
		select {
		case ch <- &KeyGenerationResponse{
			ID:            s.ID,
			PublicKey:     s.PublicKey.String(),
			Threshold:     s.Threshold,
			TotalShares:   s.TotalShares,
			Protocol:      s.Protocol,
			CreatedAt:     time.Now().UnixMilli(),
		}:
		default:
		}
	}
	
	return nil
}

// VerifyShare verifies a key share is valid
func (s *KeyGenerationSession) VerifyShare(share *KeyShare) bool {
	// Verify: share * G equals the verifier point (the compressed public key
	// stored in share.Verifier). This is a real EC point comparison on
	// secp256k1, not a modular multiplication hack.
	actualX, actualY := s.Curve.Curve.ScalarBaseMult(share.Share.Bytes())
	actualCompressed := secp256k1.CompressPubkey(actualX, actualY)
	expectedCompressed := share.Verifier.Bytes()
	if len(expectedCompressed) == 0 {
		return false
	}
	return ctEqual(actualCompressed, expectedCompressed)
}

// GetSessionInfo returns session information
func (s *KeyGenerationSession) GetSessionInfo() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return map[string]interface{}{
		"id":              s.ID,
		"protocol":        s.Protocol,
		"threshold":       s.Threshold,
		"total_shares":    s.TotalShares,
		"curve":           s.Curve.Name,
		"status":          s.Status,
		"participants":    len(s.Participants),
		"shares_received": len(s.ReceivedShares),
		"created_at":      s.CreatedAt,
		"expires_at":      s.ExpiresAt,
	}
}

// MPCKeyGenerator manages multiple key generation sessions
type MPCKeyGenerator struct {
	mu            sync.RWMutex
	sessions      map[string]*KeyGenerationSession
	curve         *CurveParameters
	defaultConfig *KeyGenerationRequest
}

// NewMPCKeyGenerator creates a new MPC key generator
func NewMPCKeyGenerator() *MPCKeyGenerator {
	return &MPCKeyGenerator{
		sessions: make(map[string]*KeyGenerationSession),
		curve:    GetP256Curve(),
		defaultConfig: &KeyGenerationRequest{
			Protocol:    ProtocolV2,
			Threshold:   DefaultThreshold,
			TotalShares: DefaultTotalShares,
			Curve:       "P-256",
		},
	}
}

// CreateSession creates a new key generation session
func (g *MPCKeyGenerator) CreateSession(req *KeyGenerationRequest) (*KeyGenerationSession, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	if req.ID == "" {
		req.ID = generateID()
	}
	
	if req.Threshold <= 0 {
		req.Threshold = g.defaultConfig.Threshold
	}
	
	if req.TotalShares <= 0 {
		req.TotalShares = g.defaultConfig.TotalShares
	}
	
	if req.Threshold > req.TotalShares {
		return nil, fmt.Errorf("threshold cannot exceed total shares")
	}
	
	if req.Protocol == "" {
		req.Protocol = g.defaultConfig.Protocol
	}
	
	session := NewKeyGenerationSession(req)
	g.sessions[session.ID] = session
	
	return session, nil
}

// GetSession retrieves a session by ID
func (g *MPCKeyGenerator) GetSession(sessionID string) (*KeyGenerationSession, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	session, exists := g.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}
	
	return session, nil
}

// DeleteSession removes a session
func (g *MPCKeyGenerator) DeleteSession(sessionID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	if _, exists := g.sessions[sessionID]; !exists {
		return fmt.Errorf("session not found")
	}
	
	delete(g.sessions, sessionID)
	return nil
}

// CleanupExpiredSessions removes expired sessions
func (g *MPCKeyGenerator) CleanupExpiredSessions() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	now := time.Now().UnixMilli()
	count := 0
	
	for id, session := range g.sessions {
		if session.ExpiresAt < now {
			delete(g.sessions, id)
			count++
		}
	}
	
	return count
}

// generateID generates a unique ID
func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// HashPublicKey computes hash of public key
func HashPublicKey(pubKey *big.Int) string {
	data := pubKey.Bytes()
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// SerializeKeyShare serializes a key share to JSON
func SerializeKeyShare(share *KeyShare) (string, error) {
	data, err := json.Marshal(share)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

// DeserializeKeyShare deserializes a key share from JSON
func DeserializeKeyShare(data string) (*KeyShare, error) {
	decoded, err := hex.DecodeString(data)
	if err != nil {
		return nil, err
	}
	
	var share KeyShare
	if err := json.Unmarshal(decoded, &share); err != nil {
		return nil, err
	}
	
	return &share, nil
}

// ctEqual performs a constant-time comparison of two byte slices and reports
// whether they are equal in length and content.
func ctEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := range a {
		v |= a[i] ^ b[i]
	}
	return v == 0
}
