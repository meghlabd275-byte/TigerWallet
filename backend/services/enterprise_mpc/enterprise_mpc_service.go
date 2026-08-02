/**
 * TigerWallet Enterprise MPC Service
 * 
 * Multi-party computation service for enterprise-grade key management
 * Supports threshold signatures, HSM integration, and distributed signing
 */

package enterprise_mpc

import (
	"context"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"time"
)

// ============================================================================
// Types
// ============================================================================

// MPCConfig represents MPC configuration
type MPCConfig struct {
	Threshold     int      `json:"threshold"`
	TotalShares  int      `json:"total_shares"`
	Curve        string   `json:"curve"` // P-256, P-384
	Protocol     string   `json:"protocol"` // ECDSA, Schnorr
	HSMEnabled   bool     `json:"hsm_enabled"`
	AuditEnabled bool     `json:"audit_enabled"`
}

// KeyGenerationRequest represents a key generation request
type KeyGenerationRequest struct {
	ID          string   `json:"id"`
	Threshold   int      `json:"threshold"`
	TotalShares int      `json:"total_shares"`
	SignerIDs   []string `json:"signer_ids"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// SigningRequest represents a signing request
type SigningRequest struct {
	ID          string `json:"id"`
	SessionID  string `json:"session_id"`
	Message    string `json:"message"`
	SignerIDs  []string `json:"signer_ids"`
	Threshold  int    `json:"threshold"`
}

// KeyShare represents a key share
type KeyShare struct {
	ID            string   `json:"id"`
	ShareID      string   `json:"share_id"`
	SignerID     string   `json:"signer_id"`
	ShareData    string   `json:"share_data"`
	VerifierData string   `json:"verifier_data"`
	PublicKey    string   `json:"public_key"`
	Index        int      `json:"index"`
	CreatedAt    int64    `json:"created_at"`
}

// ThresholdSignature represents a threshold signature
type ThresholdSignature struct {
	ID          string   `json:"id"`
	R           string   `json:"r"`
	S           string   `json:"s"`
	Signers     []string `json:"signers"`
	Threshold   int      `json:"threshold"`
	MessageHash string   `json:"message_hash"`
	CreatedAt   int64    `json:"created_at"`
}

// KeyGenerationSession represents a key generation session
type KeyGenerationSession struct {
	ID            string                `json:"id"`
	Status       string                `json:"status"`
	Threshold    int                   `json:"threshold"`
	TotalShares  int                   `json:"total_shares"`
	PublicKey    string                `json:"public_key"`
	Shares       map[string]*KeyShare  `json:"shares"`
	Signers      map[string]bool       `json:"signers"`
	CreatedAt    int64                 `json:"created_at"`
	ExpiresAt    int64                 `json:"expires_at"`
}

// SigningSession represents a signing session
type SigningSession struct {
	ID           string                 `json:"id"`
	SessionID    string                 `json:"session_id"`
	Status       string                 `json:"status"`
	MessageHash  string                 `json:"message_hash"`
	Threshold   int                    `json:"threshold"`
	PartialSigs map[string]string      `json:"partial_sigs"`
	Signers      map[string]bool        `json:"signers"`
	Signature    *ThresholdSignature     `json:"signature,omitempty"`
	CreatedAt   int64                  `json:"created_at"`
	ExpiresAt   int64                  `json:"expires_at"`
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID          string                 `json:"id"`
	SessionID   string                 `json:"session_id"`
	Action      string                 `json:"action"`
	ActorID     string                 `json:"actor_id"`
	Details     map[string]interface{} `json:"details"`
	IPAddress   string                 `json:"ip_address"`
	Timestamp   int64                 `json:"timestamp"`
}

// ============================================================================
// Enterprise MPC Service
// ============================================================================

// EnterpriseMPCService represents the enterprise MPC service
type EnterpriseMPCService struct {
	mu              sync.RWMutex
	config          *MPCConfig
	keyGenSessions  map[string]*KeyGenerationSession
	signingSessions map[string]*SigningSession
	auditLogs      []*AuditLog
	keyShares      map[string]*KeyShare
}

// NewEnterpriseMPCService creates a new enterprise MPC service
func NewEnterpriseMPCService(config *MPCConfig) *EnterpriseMPCService {
	if config == nil {
		config = DefaultMPCConfig()
	}
	
	return &EnterpriseMPCService{
		config:          config,
		keyGenSessions:  make(map[string]*KeyGenerationSession),
		signingSessions: make(map[string]*SigningSession),
		auditLogs:      make([]*AuditLog, 0),
		keyShares:      make(map[string]*KeyShare),
	}
}

// DefaultMPCConfig returns default configuration
func DefaultMPCConfig() *MPCConfig {
	return &MPCConfig{
		Threshold:    2,
		TotalShares: 3,
		Curve:       "P-256",
		Protocol:    "ECDSA",
		HSMEnabled:  true,
		AuditEnabled: true,
	}
}

// ============================================================================
// Key Generation
// ============================================================================

// CreateKeyGenerationSession creates a new key generation session
func (s *EnterpriseMPCService) CreateKeyGenerationSession(ctx context.Context, req *KeyGenerationRequest) (*KeyGenerationSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	session := &KeyGenerationSession{
		ID:           generateID(),
		Status:       "pending",
		Threshold:    req.Threshold,
		TotalShares:  req.TotalShares,
		Shares:       make(map[string]*KeyShare),
		Signers:     make(map[string]bool),
		CreatedAt:   time.Now().UnixMilli(),
		ExpiresAt:   time.Now().Add(10 * time.Minute).UnixMilli(),
	}
	
	// Initialize signers
	for _, signerID := range req.SignerIDs {
		session.Signers[signerID] = false
	}
	
	s.keyGenSessions[session.ID] = session
	
	// Audit log
	if s.config.AuditEnabled {
		s.auditLogs = append(s.auditLogs, &AuditLog{
			ID:        generateID(),
			SessionID: session.ID,
			Action:    "key_generation_started",
			Details:   map[string]interface{}{"threshold": req.Threshold, "total": req.TotalShares},
			Timestamp: time.Now().UnixMilli(),
		})
	}
	
	return session, nil
}

// AddKeyShare adds a key share to a key generation session
func (s *EnterpriseMPCService) AddKeyShare(ctx context.Context, sessionID, signerID string, shareData, verifierData string) (*KeyShare, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	session, exists := s.keyGenSessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}
	
	if session.Status != "pending" {
		return nil, fmt.Errorf("session not pending")
	}
	
	// Generate key share (simplified - in production use proper MPC)
	share := &KeyShare{
		ID:            generateID(),
		ShareID:       fmt.Sprintf("%s-share-%s", sessionID, signerID),
		SignerID:      signerID,
		ShareData:     shareData,
		VerifierData:  verifierData,
		PublicKey:     session.PublicKey,
		Index:         len(session.Shares) + 1,
		CreatedAt:     time.Now().UnixMilli(),
	}
	
	session.Shares[signerID] = share
	session.Signers[signerID] = true
	s.keyShares[share.ID] = share
	
	// Check if all shares received
	allReceived := true
	for _, received := range session.Signers {
		if !received {
			allReceived = false
			break
		}
	}
	
	if allReceived {
		session.Status = "completed"
		
		// Generate public key (simplified)
		session.PublicKey = generatePublicKey(shareData)
		
		// Audit log
		if s.config.AuditEnabled {
			s.auditLogs = append(s.auditLogs, &AuditLog{
				ID:        generateID(),
				SessionID: sessionID,
				Action:    "key_generation_completed",
				Details:   map[string]interface{}{"public_key": session.PublicKey},
				Timestamp: time.Now().UnixMilli(),
			})
		}
	}
	
	return share, nil
}

// GetKeyGenerationSession gets a key generation session
func (s *EnterpriseMPCService) GetKeyGenerationSession(ctx context.Context, sessionID string) (*KeyGenerationSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	session, exists := s.keyGenSessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}
	
	return session, nil
}

// GetKeyShare gets a key share by ID
func (s *EnterpriseMPCService) GetKeyShare(ctx context.Context, shareID string) (*KeyShare, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	share, exists := s.keyShares[shareID]
	if !exists {
		return nil, fmt.Errorf("share not found")
	}
	
	return share, nil
}

// ============================================================================
// Signing
// ============================================================================

// CreateSigningSession creates a new signing session
func (s *EnterpriseMPCService) CreateSigningSession(ctx context.Context, req *SigningRequest) (*SigningSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Hash message
	msgHash := sha256.Sum256([]byte(req.Message))
	msgHashHex := hex.EncodeToString(msgHash[:])
	
	session := &SigningSession{
		ID:           generateID(),
		SessionID:    req.SessionID,
		Status:       "pending",
		MessageHash:  msgHashHex,
		Threshold:    req.Threshold,
		PartialSigs:  make(map[string]string),
		Signers:      make(map[string]bool),
		CreatedAt:    time.Now().UnixMilli(),
		ExpiresAt:    time.Now().Add(5 * time.Minute).UnixMilli(),
	}
	
	// Initialize signers
	for _, signerID := range req.SignerIDs {
		session.Signers[signerID] = false
	}
	
	s.signingSessions[session.ID] = session
	
	// Audit log
	if s.config.AuditEnabled {
		s.auditLogs = append(s.auditLogs, &AuditLog{
			ID:         generateID(),
			SessionID:  session.ID,
			Action:     "signing_started",
			Details:    map[string]interface{}{"message_hash": msgHashHex},
			Timestamp:  time.Now().UnixMilli(),
		})
	}
	
	return session, nil
}

// AddPartialSignature adds a partial signature to a signing session
func (s *EnterpriseMPCService) AddPartialSignature(ctx context.Context, sessionID, signerID, partialSig string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	session, exists := s.signingSessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found")
	}
	
	if session.Status != "pending" {
		return fmt.Errorf("session not pending")
	}
	
	session.PartialSigs[signerID] = partialSig
	session.Signers[signerID] = true
	
	// Check if we have enough partial signatures
	if len(session.PartialSigs) >= session.Threshold {
		// Combine signatures (simplified)
		sig, err := combineSignatures(session.PartialSigs, session.MessageHash)
		if err != nil {
			return fmt.Errorf("failed to combine signatures: %w", err)
		}
		
		session.Signature = sig
		session.Status = "completed"
		
		// Audit log
		if s.config.AuditEnabled {
			s.auditLogs = append(s.auditLogs, &AuditLog{
				ID:        generateID(),
				SessionID: sessionID,
				Action:    "signing_completed",
				Details:   map[string]interface{}{"signature": fmt.Sprintf("%s:%s", sig.R, sig.S)},
				Timestamp: time.Now().UnixMilli(),
			})
		}
	}
	
	return nil
}

// GetSigningSession gets a signing session
func (s *EnterpriseMPCService) GetSigningSession(ctx context.Context, sessionID string) (*SigningSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	session, exists := s.signingSessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}
	
	return session, nil
}

// ============================================================================
// Audit
// ============================================================================

// GetAuditLogs gets audit logs
func (s *EnterpriseMPCService) GetAuditLogs(ctx context.Context, sessionID string) ([]*AuditLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	logs := make([]*AuditLog, 0)
	for _, log := range s.auditLogs {
		if log.SessionID == sessionID {
			logs = append(logs, log)
		}
	}
	
	return logs, nil
}

// GetAllAuditLogs gets all audit logs
func (s *EnterpriseMPCService) GetAllAuditLogs(ctx context.Context) ([]*AuditLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return s.auditLogs, nil
}

// ============================================================================
// HSM Integration (Placeholder)
// ============================================================================

// HSMKey represents an HSM key
type HSMKey struct {
	ID           string `json:"id"`
	KeyType      string `json:"key_type"`
	PublicKey    string `json:"public_key"`
	CreatedAt    int64  `json:"created_at"`
	Algorithm    string `json:"algorithm"`
}

// GenerateHSMKey generates a key in HSM
func (s *EnterpriseMPCService) GenerateHSMKey(ctx context.Context, keyType string) (*HSMKey, error) {
	// In production, integrate with HSM (Thales, Utimaco, etc.)
	// This is a placeholder implementation
	key := &HSMKey{
		ID:        generateID(),
		KeyType:   keyType,
		CreatedAt: time.Now().UnixMilli(),
		Algorithm: s.config.Curve,
	}
	
	// Generate key in HSM (simulated)
	key.PublicKey = generatePublicKey(generateID())
	
	return key, nil
}

// SignWithHSM signs data with HSM key
func (s *EnterpriseMPCService) SignWithHSM(ctx context.Context, keyID, data string) (string, error) {
	// In production, use actual HSM signing
	// This is a placeholder
	return hex.EncodeToString(sha256.Sum256([]byte(data))), nil
}

// ============================================================================
// Helper Functions
// ============================================================================

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func generatePublicKey(privateKey string) string {
	// Simplified - in production use proper EC operations
	curve := elliptic.P256()
	_, _, x, y := curve.ScalarBaseMult([]byte(privateKey))
	
	pubKey := append(x.Bytes(), y.Bytes()...)
	return hex.EncodeToString(pubKey)
}

func combineSignatures(partialSigs map[string]string, messageHash string) (*ThresholdSignature, error) {
	// Simplified signature combination
	// In production, use proper threshold signature combination
	
	var r, s *big.Int
	for id, sig := range partialSigs {
		sigBytes, _ := hex.DecodeString(sig)
		if len(sigBytes) >= 64 {
			r = new(big.Int).SetBytes(sigBytes[:32])
			s = new(big.Int).SetBytes(sigBytes[32:64])
			break
		}
	}
	
	if r == nil || s == nil {
		return nil, fmt.Errorf("failed to parse signatures")
	}
	
	signers := make([]string, 0, len(partialSigs))
	for signerID := range partialSigs {
		signers = append(signers, signerID)
	}
	
	return &ThresholdSignature{
		ID:          generateID(),
		R:           r.String(),
		S:           s.String(),
		Signers:     signers,
		Threshold:   len(partialSigs),
		MessageHash: messageHash,
		CreatedAt:   time.Now().UnixMilli(),
	}, nil
}

// ============================================================================
// EnterpriseMPCServiceServer (gRPC integration)
// ============================================================================

// EnterpriseMPCServiceServer represents the gRPC server
type EnterpriseMPCServiceServer struct {
	service *EnterpriseMPCService
}

// NewEnterpriseMPCServiceServer creates a new server
func NewEnterpriseMPCServiceServer(service *EnterpriseMPCService) *EnterpriseMPCServiceServer {
	return &EnterpriseMPCServiceServer{
		service: service,
	}
}

// RegisterService registers the service with gRPC
func (s *EnterpriseMPCServiceServer) RegisterService() {
	// In production, register with gRPC server
}

// JSON serialization helpers

func (s *KeyGenerationSession) Serialize() (string, error) {
	data, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

func DeserializeKeyGenerationSession(data string) (*KeyGenerationSession, error) {
	decoded, err := hex.DecodeString(data)
	if err != nil {
		return nil, err
	}
	
	var session KeyGenerationSession
	if err := json.Unmarshal(decoded, &session); err != nil {
		return nil, err
	}
	
	return &session, nil
}

func (s *SigningSession) Serialize() (string, error) {
	data, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

func DeserializeSigningSession(data string) (*SigningSession, error) {
	decoded, err := hex.DecodeString(data)
	if err != nil {
		return nil, err
	}
	
	var session SigningSession
	if err := json.Unmarshal(decoded, &session); err != nil {
		return nil, err
	}
	
	return &session, nil
}
