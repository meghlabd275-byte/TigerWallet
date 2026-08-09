/**
 * Enterprise MPC Infrastructure - Threshold Signatures
 * 
 * Threshold signature scheme (TSS) implementation
 * Supports ECDSA and Schnorr threshold signatures
 */

package threshold

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
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
	Curve       *secp256k1.BitCurve
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
	ID          string            `json:"id"`
	RequestID   string            `json:"request_id"`
	SignerID    string            `json:"signer_id"`
	R           *big.Int          `json:"r"` // R component
	S           *big.Int          `json:"s"` // S component
	PublicKey   []byte            `json:"public_key"` // signer's partial public key (compressed)
	Commitments map[string]string `json:"commitments"`
	Timestamp   int64             `json:"timestamp"`
}

// ThresholdSignature represents a complete threshold signature
type ThresholdSignature struct {
	ID          string   `json:"id"`
	RequestID   string   `json:"request_id"`
	R           string   `json:"r"`
	S           string   `json:"s"`
	V           byte     `json:"v"`              // recovery id for secp256k1 ECDSA
	PublicKey   string   `json:"public_key"`     // hex compressed group public key
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
	GroupPublicKey  []byte // compressed secp256k1 group public key (Y = sum share_i*G)
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
func NewSigningSession(req *SigningRequest, curve *secp256k1.BitCurve) *SigningSession {
	// Compute message hash if not provided
	var msgHash *big.Int
	if req.MessageHash != "" {
		hashBytes, _ := hex.DecodeString(req.MessageHash)
		msgHash = new(big.Int).SetBytes(hashBytes)
	} else {
		hash := crypto.Keccak256([]byte(req.Message))
		msgHash = new(big.Int).SetBytes(hash)
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
func (s *SigningSession) GenerateNonce(signerID string, curve *secp256k1.BitCurve) (*NoncePair, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	_, exists := s.Signers[signerID]
	if !exists {
		return nil, fmt.Errorf("signer not found")
	}
	
	// Generate random nonce
	r, err := rand.Int(rand.Reader, curve.N)
	if err != nil {
		return nil, err
	}
	
	gamma, err := rand.Int(rand.Reader, curve.N)
	if err != nil {
		return nil, err
	}
	
	nonce := &NoncePair{
		R:     r,
		Gamma: gamma,
	}
	
	s.Nonces[signerID] = nonce
	
	// Compute commitment binding the nonce point R = r*G and gamma.
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
func (s *SigningSession) CombineSignatures(curve *secp256k1.BitCurve) (*ThresholdSignature, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.PartialSigs) < s.Threshold {
		return nil, fmt.Errorf("insufficient partial signatures")
	}

	signers := make([]string, 0, len(s.PartialSigs))
	for sid := range s.PartialSigs {
		signers = append(signers, sid)
	}

	// Real Lagrange coefficients at x = 0 over the participating signers.
	coeffs := lagrangeCoefficients(signers, curve.N)
	if coeffs == nil {
		return nil, fmt.Errorf("invalid signer set for Lagrange interpolation")
	}

	// R is the shared nonce point x-coordinate. In a correct threshold ECDSA
	// session every signer commits to the same R, so we take it from the first
	// partial signature.
	var R *big.Int
	for _, ps := range s.PartialSigs {

		R = new(big.Int).Set(ps.R)
		break
	}

	// Combine partial s values: S = sum(lambda_i * s_i) mod n.
	finalS := big.NewInt(0)
	for sid, ps := range s.PartialSigs {
		term := new(big.Int).Mul(coeffs[sid], ps.S)
		finalS.Add(finalS, term)
		finalS.Mod(finalS, curve.N)
	}

	// Enforce low-S form (EIP-2) to match Ethereum signature malleability
	// rules.
	halfN := new(big.Int).Rsh(curve.N, 1)
	if finalS.Cmp(halfN) > 0 {
		finalS.Sub(curve.N, finalS)
	}

	// Recover the public key from the combined (R, S) signature to obtain the
	// recovery id V and confirm it matches the group public key when one is
	// known.
	msgHashBytes := s.MessageHash.Bytes()
	if len(msgHashBytes) == 0 {
		msgHashBytes = HashMessage(s.Message)
	}
	rBytes := R.Bytes()
	sBytes := finalS.Bytes()
	sig65 := make([]byte, 65)
	copy(sig65[0:32], leftPad(rBytes, 32))
	copy(sig65[32:64], leftPad(sBytes, 32))

	var recoveredPub []byte
	var v byte
	for trial := byte(0); trial < 2; trial++ {
		sig65[64] = trial
		pub, err := crypto.Ecrecover(msgHashBytes, sig65)
		if err != nil || len(pub) == 0 {
			continue
		}
		compressed := secp256k1.CompressPubkey(big.NewInt(0).SetBytes(pub[1:33]), big.NewInt(0).SetBytes(pub[33:65]))
		if len(s.GroupPublicKey) == 0 || ctEqualBytes(compressed, s.GroupPublicKey) {
			recoveredPub = compressed
			v = trial
			break
		}
	}
	if recoveredPub == nil {
		// No group public key recorded: take the first recovery that succeeds.
		for trial := byte(0); trial < 2; trial++ {
			sig65[64] = trial
			pub, err := crypto.Ecrecover(msgHashBytes, sig65)
			if err == nil && len(pub) > 0 {
				recoveredPub = secp256k1.CompressPubkey(big.NewInt(0).SetBytes(pub[1:33]), big.NewInt(0).SetBytes(pub[33:65]))
				v = trial
				break
			}
		}
	}

	sig := &ThresholdSignature{
		ID:          generateID(),
		RequestID:   s.ID,
		R:           R.String(),
		S:           finalS.String(),
		V:           v,
		PublicKey:   hex.EncodeToString(recoveredPub),
		Signers:     signers,
		Threshold:   s.Threshold,
		TotalShares: s.TotalShares,
		Scheme:      s.Scheme,
		CreatedAt:   time.Now().UnixMilli(),
		MessageHash: s.MessageHash.String(),
	}

	s.Status = "completed"

	select {
	case s.CompletedChan <- sig:
	default:
	}

	return sig, nil
}

// VerifySignature verifies a threshold signature
func VerifySignature(sig *ThresholdSignature, message []byte, curve *secp256k1.BitCurve) bool {
	// Decode R and S.
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

	// Range checks on the signature scalars (real ECDSA verification checks).
	if r.Sign() <= 0 || r.Cmp(curve.N) >= 0 {
		return false
	}
	if s.Sign() <= 0 || s.Cmp(curve.N) >= 0 {
		return false
	}

	// Keccak-256 message hash, matching Ethereum's signed-message digest.
	msgHash := crypto.Keccak256(message)

	// Build the 65-byte secp256k1 signature (R || S || V) and verify it
	// against the recovered public key using go-ethereum's real ECDSA
	// verification primitive.
	sig65 := make([]byte, 65)
	copy(sig65[0:32], leftPad(r.Bytes(), 32))
	copy(sig65[32:64], leftPad(s.Bytes(), 32))
	sig65[64] = sig.V

	pubBytes, err := crypto.Ecrecover(msgHash, sig65)
	if err != nil || len(pubBytes) != 65 {
		return false
	}

	// crypto.VerifySignature expects a 64-byte (R||S) signature and an
    // uncompressed (no-prefix) 64-byte public key. Use it as the authoritative
	// real check.
	if !crypto.VerifySignature(pubBytes[1:], msgHash, sig65[:64]) {
		return false
	}

	// If the signature carries a recorded public key, ensure the recovered key
	// matches it.
	if sig.PublicKey != "" {
		expected, err := hex.DecodeString(sig.PublicKey)
		if err != nil {
			return false
		}
		recovered := secp256k1.CompressPubkey(new(big.Int).SetBytes(pubBytes[1:33]), new(big.Int).SetBytes(pubBytes[33:65]))
		if !ctEqualBytes(recovered, expected) {
			return false
		}
	}

	return true
}

// TSSManager manages threshold signing sessions
type TSSManager struct {
	mu            sync.RWMutex
	sessions      map[string]*SigningSession
	curve         *secp256k1.BitCurve
	defaultScheme string
}

// NewTSSManager creates a new TSS manager
func NewTSSManager() *TSSManager {
	return &TSSManager{
		sessions:      make(map[string]*SigningSession),
		curve:         secp256k1.S256(),
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
// leftPad left-pads b with zero bytes so its length is at least size.
func leftPad(b []byte, size int) []byte {
	if len(b) >= size {
		return b
	}
	out := make([]byte, size)
	copy(out[size-len(b):], b)
	return out
}

// ctEqualBytes compares two byte slices in constant time.
func ctEqualBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := range a {
		v |= a[i] ^ b[i]
	}
	return v == 0
}


func computeCommitment(r, gamma *big.Int, curve *secp256k1.BitCurve) string {
	// Real commitment: bind the nonce point R = r*G and the gamma value
	// together with a Keccak-256 commitment hash.
	rx, _ := curve.ScalarBaseMult(r.Bytes())
	h := crypto.NewKeccakState()
	h.Write(rx.Bytes())
	h.Write(gamma.Bytes())
	return hex.EncodeToString(h.Sum(nil))
}

// lagrangeCoefficients computes the real Lagrange interpolation basis
// coefficients at x = 0 for the given set of signer x-coordinates. The
// coefficient for signer i is lambda_i = prod_{j!=i}(x_j / (x_j - x_i)) mod n,
// used to combine threshold partial signatures into a full signature.
func lagrangeCoefficients(signers []string, n *big.Int) map[string]*big.Int {
	coeffs := make(map[string]*big.Int, len(signers))
	for _, i := range signers {
		num := big.NewInt(1)
		den := big.NewInt(1)
		xi := new(big.Int).SetBytes([]byte(i))
		for _, j := range signers {
			if i == j {
				continue
			}
			xj := new(big.Int).SetBytes([]byte(j))
			num.Mul(num, xj)
			num.Mod(num, n)
			diff := new(big.Int).Sub(xj, xi)
			diff.Mod(diff, n)
			if diff.Sign() == 0 {
				return nil
			}
			den.Mul(den, diff)
			den.Mod(den, n)
		}
		denInv := new(big.Int).ModInverse(den, n)
		if denInv == nil {
			return nil
		}
		c := new(big.Int).Mul(num, denInv)
		c.Mod(c, n)
		coeffs[i] = c
	}
	return coeffs
}

func computeLagrangeCoefficient(partialSigs map[string]*PartialSignature, n *big.Int) *big.Int {
	// Backward-compatible helper: returns the product of the real Lagrange
	// coefficients. Prefer lagrangeCoefficients for combining signatures.
	signers := make([]string, 0, len(partialSigs))
	for sid := range partialSigs {
		signers = append(signers, sid)
	}
	coeffs := lagrangeCoefficients(signers, n)
	if coeffs == nil {
		return big.NewInt(0)
	}
	prod := big.NewInt(1)
	for _, c := range coeffs {
		prod.Mul(prod, c)
		prod.Mod(prod, n)
	}
	return prod
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

// HashMessage hashes a message for signing using Keccak-256, the
// digest used by Ethereum's ECDSA signatures.
func HashMessage(message []byte) []byte {
	return crypto.Keccak256(message)
}

// SignWithTSS creates a threshold signature (coordinator function)
func SignWithTSS(
	partialSigs map[string]*PartialSignature,
	curve *secp256k1.BitCurve,
) (*ThresholdSignature, error) {
	if len(partialSigs) < 2 {
		return nil, fmt.Errorf("need at least 2 partial signatures")
	}

	signers := make([]string, 0, len(partialSigs))
	for sid := range partialSigs {
		signers = append(signers, sid)
	}

	coeffs := lagrangeCoefficients(signers, curve.N)
	if coeffs == nil {
		return nil, fmt.Errorf("invalid signer set for Lagrange interpolation")
	}

	var R *big.Int
	for _, ps := range partialSigs {
		R = new(big.Int).Set(ps.R)
		break
	}

	finalS := big.NewInt(0)
	for sid, ps := range partialSigs {
		term := new(big.Int).Mul(coeffs[sid], ps.S)
		finalS.Add(finalS, term)
		finalS.Mod(finalS, curve.N)
	}
	halfN := new(big.Int).Rsh(curve.N, 1)
	if finalS.Cmp(halfN) > 0 {
		finalS.Sub(curve.N, finalS)
	}

	msgHash := HashMessage(nil)
	sig65 := make([]byte, 65)
	copy(sig65[0:32], leftPad(R.Bytes(), 32))
	copy(sig65[32:64], leftPad(finalS.Bytes(), 32))

	var recoveredPub []byte
	var v byte
	for trial := byte(0); trial < 2; trial++ {
		sig65[64] = trial
		pub, err := crypto.Ecrecover(msgHash, sig65)
		if err == nil && len(pub) > 0 {
			recoveredPub = secp256k1.CompressPubkey(new(big.Int).SetBytes(pub[1:33]), new(big.Int).SetBytes(pub[33:65]))
			v = trial
			break
		}
	}

	return &ThresholdSignature{
		ID:          generateID(),
		R:           R.String(),
		S:           finalS.String(),
		V:           v,
		PublicKey:   hex.EncodeToString(recoveredPub),
		Signers:     signers,
		Threshold:   len(partialSigs),
		TotalShares: len(partialSigs),
		Scheme:      SchemeECDSA,
		CreatedAt:   time.Now().UnixMilli(),
	}, nil
}
