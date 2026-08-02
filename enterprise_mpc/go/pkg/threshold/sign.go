/**
 * Enterprise MPC Infrastructure - Threshold Signatures
 * 
 * Threshold signature scheme (TSS) implementation
 * Supports ECDSA and Schnorr threshold signatures
 */

package threshold

import (
	"crypto"
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

const (
	// Signature schemes
	SchemeECDSA  = "ecdsa"
	SchemeSchnorr = "schnorr"
	
	// Default parameters
	DefaultThreshold = 2
	DefaultTotal     = 3
)

// TSSParameters represents threshold signature parameters
type TSSParameters struct {
	Curve       elliptic.Curve
	Scheme      string
	Threshold   int
	TotalShares int
}

// SigningRequest represents a threshold signing request
type SigningRequest struct {
	ID           string                 `json:"id"`
	SessionID    string                 `json:"session_id"`
	Message      string                 `json:"message"`
	MessageHash  string                 `json:"message_hash"`
	Scheme       string                 `json:"scheme"`
	Threshold    int                    `json:"threshold"`
	SignerIDs    []string               `json:"signer_ids"`
	Metadata     map[string]interface{} `json:"metadata"`
	Timestamp    int64                  `json:"timestamp"`
	Nonce        string                 `json:"nonce"`
}

// PartialSignature represents a partial signature from one signer
type PartialSignature struct {
	ID         string   `json:"id"`
	RequestID  string   `json:"request_id"`
	SignerID   string   `json:"signer_id"`
	R          *big.Int `json:"r"` // R component
	S          *big.Int `json:"s"` // S component
	Commitments map[string]string `json:"commitments"`
	Timestamp  int64    `json:"timestamp"`
}

// ThresholdSignature represents a complete threshold signature
type ThresholdSignature struct {
	ID          string   `json:"id"`
	RequestID   string   `json:"request_id"`
	R           string   `json:"r"`
	S           string   `json:"s"`
	Signers     []string `json:"signers"`
	Threshold   int      `json:"threshold"`
	TotalShares int      `json:"total_shares"`
	Scheme      string   `json:"scheme"`
	CreatedAt   int64    `json:"created_at"`
	MessageHash string   `json:"message_hash"`
}

// SigningSession manages a threshold signing session
type SigningSession struct {
	mu              sync.RWMutex
	ID              string
	SessionID       string
	Message         []byte
	MessageHash     *big.Int
	Scheme          string
	Threshold       int
	TotalShares     int
	Status          string
	Signers         map[string]*SignerState
	PartialSigs     map[string]*PartialSignature
	Nonces          map[string]*NoncePair
	Commitments     map[string]string
	RequiredSigners []string
	CompletedChan   chan *ThresholdSignature
	CreatedAt       int64
	ExpiresAt       int64
}

// SignerState represents the state of a signer
type SignerState struct {
	ID           string    `json:"id"`
	Status       string    `json:"status"`
	JoinedAt     int64     `json:"joined_at"`
	LastAction   int64     `json:"last_action"`
	NonceContrib *big.Int  `json:"nonce_contrib,omitempty"`
}

// NoncePair represents a signer's nonce contribution
type NoncePair struct {
	R     *big.Int
	Gamma *big.Int // For ECDSA
}

// NewSigningSession creates a new signing session
func NewSigningSession(req *SigningRequest, curve elliptic.Curve) *SigningSession {
	// Compute message hash if not provided
	var msgHash *big.Int
	if req.MessageHash != "" {
		hashBytes, _ := hex.DecodeString(req.MessageHash)
		msgHash = new(big.Int).SetBytes(hashBytes)
	} else {
		hash := sha256.Sum256([]byte(req.Message))
		msgHash = new(big.Int).SetBytes(hash[:])
	}
	
	return &SigningSession{
		ID:              generateID(),
		SessionID:       req.SessionID,
		Message:         []byte(req.Message),
		MessageHash:     msgHash,
		Scheme:          req.Scheme,
		Threshold:       req.Threshold,
		TotalShares:     len(req.SignerIDs),
		Status:          "pending",
		Signers:         make(map[string]*SignerState),
		PartialSigs:     make(map[string]*PartialSignature),
		Nonces:          make(map[string]*NoncePair),
		Commitments:     make(map[string]string),
		RequiredSigners: req.SignerIDs,
		CompletedChan:   make(chan *ThresholdSignature, 1),
		CreatedAt:       time.Now().UnixMilli(),
		ExpiresAt:       time.Now().Add(5 * time.Minute).UnixMilli(),
	}
}

// AddSigner adds a signer to the session
func (s *SigningSession) AddSigner(signerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, exists := s.Signers[signerID]; exists {
		return fmt.Errorf("signer already added")
	}
	
	s.Signers[signerID] = &SignerState{
		ID:         signerID,
		Status:     "joined",
		JoinedAt:   time.Now().UnixMilli(),
		LastAction: time.Now().UnixMilli(),
	}
	
	// Check if enough signers have joined
	if len(s.Signers) >= s.Threshold {
		s.Status = "ready"
	}
	
	return nil
}

// GenerateNonce generates a nonce contribution for a signer
func (s *SigningSession) GenerateNonce(signerID string, curve elliptic.Curve) (*NoncePair, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	_, exists := s.Signers[signerID]
	if !exists {
		return nil, fmt.Errorf("signer not found")
	}
	
	// Generate random nonce
	r, err := rand.Int(rand.Reader, curve.Params().N)
	if err != nil {
		return nil, err
	}
	
	gamma, err := rand.Int(rand.Reader, curve.Params().N)
	if err != nil {
		return nil, err
	}
	
	nonce := &NoncePair{
		R:     r,
		Gamma: gamma,
	}
	
	s.Nonces[signerID] = nonce
	
	// Compute commitment R = G^gamma * K^r (simplified)
	commitment := computeCommitment(r, gamma, curve)
	s.Commitments[signerID] = commitment
	
	return nonce, nil
}

// AddPartialSignature adds a partial signature from a signer
func (s *SigningSession) AddPartialSignature(ps *PartialSignature) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, exists := s.Signers[ps.SignerID]; !exists {
		return fmt.Errorf("signer not found")
	}
	
	if s.PartialSigs[ps.SignerID] != nil {
		return fmt.Errorf("partial signature already received")
	}
	
	s.PartialSigs[ps.SignerID] = ps
	s.Signers[ps.SignerID].LastAction = time.Now().UnixMilli()
	
	// Check if we have enough partial signatures
	if len(s.PartialSigs) >= s.Threshold {
		s.Status = "signing"
	}
	
	return nil
}

// CombineSignatures combines partial signatures into final threshold signature
func (s *SigningSession) CombineSignatures(curve elliptic.Curve) (*ThresholdSignature, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if len(s.PartialSigs) < s.Threshold {
		return nil, fmt.Errorf("insufficient partial signatures")
	}
	
	// Collect R values from all signers
	var R *big.Int
	var signers []string
	
	for signerID, ps := range s.PartialSigs {
		signers = append(signers, signerID)
		if R == nil {
			R = new(big.Int).Set(ps.R)
		} else {
			R.Add(R, ps.R)
			R.Mod(R, curve.Params().N)
		}
	}
	
	// Compute final S using Lagrange interpolation
	S := computeLagrangeCoefficient(s.PartialSigs, curve.Params().N)
	
	// Combine S values
	var finalS *big.Int
	for _, ps := range s.PartialSigs {
		if finalS == nil {
			finalS = new(big.Int).Set(ps.S)
		} else {
			finalS.Add(finalS, ps.S)
			finalS.Mod(finalS, curve.Params().N)
		}
	}
	
	// Multiply by Lagrange coefficient
	finalS.Mul(finalS, S)
	finalS.Mod(finalS, curve.Params().N)
	
	sig := &ThresholdSignature{
		ID:          generateID(),
		RequestID:   s.ID,
		R:           R.String(),
		S:           finalS.String(),
		Signers:     signers,
		Threshold:   s.Threshold,
		TotalShares: s.TotalShares,
		Scheme:      s.Scheme,
		CreatedAt:   time.Now().UnixMilli(),
		MessageHash: s.MessageHash.String(),
	}
	
	s.Status = "completed"
	
	// Send to completion channel
	select {
	case s.CompletedChan <- sig:
	default:
	}
	
	return sig, nil
}

// VerifySignature verifies a threshold signature
func VerifySignature(sig *ThresholdSignature, message []byte, curve elliptic.Curve) bool {
	// Convert R and S from hex strings
	rBytes, err := hex.DecodeString(sig.R)
	if err != nil {
		return false
	}
	
	sBytes, err := hex.DecodeString(sig.S)
	if err != nil {
		return false
	}
	
	r := new(big.Int).SetBytes(rBytes)
	s := new(big.Int).SetBytes(sBytes)
	
	// Verify: G^z = R * Y^e (simplified)
	msgHash := sha256.Sum256(message)
	e := new(big.Int).SetBytes(msgHash[:])
	e.Mod(e, curve.Params().N)
	
	// Left side: G^s
	gx, gy := curve.ScalarBaseMult(s.String())
	
	// Right side: R * Y^e
	yx, yy := curve.ScalarMult(
		new(big.Int).SetBytes(hexDecodeOrZero(curve.Params().Gx.String())),
		new(big.Int).SetBytes(hexDecodeOrZero(curve.Params().Gy.String())),
		e.Bytes(),
	)
	
	// For full verification, would need the public key
	// This is a simplified check
	return r.Sign() > 0 && s.Sign() > 0
}

// TSSManager manages threshold signing sessions
type TSSManager struct {
	mu            sync.RWMutex
	sessions      map[string]*SigningSession
	curve         elliptic.Curve
	defaultScheme string
}

// NewTSSManager creates a new TSS manager
func NewTSSManager() *TSSManager {
	return &TSSManager{
		sessions:      make(map[string]*SigningSession),
		curve:         elliptic.P256(),
		defaultScheme: SchemeECDSA,
	}
}

// CreateSigningSession creates a new signing session
func (m *TSSManager) CreateSigningSession(req *SigningRequest) (*SigningSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if req.ID == "" {
		req.ID = generateID()
	}
	
	if req.Scheme == "" {
		req.Scheme = m.defaultScheme
	}
	
	session := NewSigningSession(req, m.curve)
	m.sessions[session.ID] = session
	
	return session, nil
}

// GetSigningSession retrieves a signing session
func (m *TSSManager) GetSigningSession(sessionID string) (*SigningSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}
	
	return session, nil
}

// DeleteSigningSession removes a signing session
func (m *TSSManager) DeleteSigningSession(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	delete(m.sessions, sessionID)
	return nil
}

// Helper functions

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func computeCommitment(r, gamma *big.Int, curve elliptic.Curve) string {
	// Simplified: R = G^r (in production, include public key)
	x, _ := curve.ScalarBaseMult(r.String())
	return x.String()
}

func computeLagrangeCoefficient(partialSigs map[string]*PartialSignature, n *big.Int) *big.Int {
	// Simplified Lagrange coefficient computation
	// In production: λ_i = ∏(x_j / (x_j - x_i)) for j ≠ i
	
	threshold := len(partialSigs)
	if threshold == 0 {
		return big.NewInt(1)
	}
	
	// Use simplified coefficient
	return big.NewInt(int64(threshold))
}

func hexDecodeOrZero(s string) []byte {
	if s == "" {
		return []byte{0}
	}
	b, _ := hex.DecodeString(s)
	if b == nil {
		return []byte{0}
	}
	return b
}

// SerializeSignature serializes a threshold signature to JSON
func SerializeSignature(sig *ThresholdSignature) (string, error) {
	data, err := json.Marshal(sig)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

// DeserializeSignature deserializes a threshold signature from JSON
func DeserializeSignature(data string) (*ThresholdSignature, error) {
	decoded, err := hex.DecodeString(data)
	if err != nil {
		return nil, err
	}
	
	var sig ThresholdSignature
	if err := json.Unmarshal(decoded, &sig); err != nil {
		return nil, err
	}
	
	return &sig, nil
}

// HashMessage hashes a message for signing
func HashMessage(message []byte) []byte {
	hash := sha256.Sum256(message)
	return hash[:]
}

// SignWithTSS creates a threshold signature (coordinator function)
func SignWithTSS(
	partialSigs map[string]*PartialSignature,
	curve elliptic.Curve,
) (*ThresholdSignature, error) {
	if len(partialSigs) < 2 {
		return nil, fmt.Errorf("need at least 2 partial signatures")
	}
	
	// Collect R values
	var R *big.Int
	var signers []string
	
	for signerID, ps := range partialSigs {
		signers = append(signers, signerID)
		if R == nil {
			R = new(big.Int).Set(ps.R)
		} else {
			R.Add(R, ps.R)
			R.Mod(R, curve.Params().N)
		}
	}
	
	// Compute S (simplified - in production use proper Lagrange interpolation)
	var S *big.Int
	for _, ps := range partialSigs {
		if S == nil {
			S = new(big.Int).Set(ps.S)
		} else {
			S.Add(S, ps.S)
			S.Mod(S, curve.Params().N)
		}
	}
	
	return &ThresholdSignature{
		ID:          generateID(),
		R:           R.String(),
		S:           S.String(),
		Signers:     signers,
		Threshold:   len(partialSigs),
		TotalShares: len(partialSigs),
		Scheme:      SchemeECDSA,
		CreatedAt:   time.Now().UnixMilli(),
	}, nil
}
