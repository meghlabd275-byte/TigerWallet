package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// Hardware Wallet Service - Ledger/Trezor/Air-gapped Integration
// ============================================================================

const (
	HardwareServicePort = 8086
)

// ============================================================================
// Types
// ============================================================================

// HardwareWallet represents a connected hardware wallet
type HardwareWallet struct {
	ID           string    `json:"id"`
	Type        string    `json:"type"` // ledger, trezor, keystone, tangem, airgapped
	Model       string    `json:"model"`
	Serial      string    `json:"serial"`
	Firmware    string    `json:"firmware"`
	PublicKey   string    `json:"public_key"`
	Address     string    `json:"address"`
	Connected   bool      `json:"connected"`
	LastSeen   time.Time `json:"last_seen"`
	Features   []string  `json:"features"`
}

// DeviceSession represents an active hardware wallet session
type DeviceSession struct {
	ID           string    `json:"id"`
	WalletID    string    `json:"wallet_id"`
	DeviceID    string    `json:"device_id"`
	PublicKey   string    `json:"public_key"`
	SessionKey  string    `json:"session_key_encrypted"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	LastActive  time.Time `json:"last_active"`
}

// SigningRequest represents a transaction signing request
type SigningRequest struct {
	ID          string    `json:"id"`
	WalletID    string    `json:"wallet_id"`
	SessionID   string    `json:"session_id"`
	ChainID     int       `json:"chain_id"`
	Type        string    `json:"type"` // transaction, message, typed_data
	Payload    string    `json:"payload"`
	Status      string    `json:"status"` // pending, approved, rejected, signed
	Signature   string    `json:"signature"`
	CreatedAt   time.Time `json:"created_at"`
	CompletedAt time.Time `json:"completed_at"`
}

// AirGapTransaction represents an air-gapped transaction
type AirGapTransaction struct {
	ID           string    `json:"id"`
	WalletID    string    `json:"wallet_id"`
	UnsignedTX  string    `json:"unsigned_tx"`
	Signature   string    `json:"signature"`
	ChainID     int       `json:"chain_id"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	SignedAt    time.Time `json:"signed_at"`
}

// ============================================================================
// Storage
// ============================================================================

var (
	hwMux       sync.RWMutex
	wallets     = make(map[string]*HardwareWallet)
	sessions    = make(map[string]*DeviceSession)
	signingReqs = make(map[string]*SigningRequest)
	airGapTXs   = make(map[string]*AirGapTransaction)
)

// ============================================================================
// Hardware Wallet Functions
// ============================================================================

// ConnectWallet connects to a hardware wallet
func ConnectWallet(walletType, model, serial string) (*HardwareWallet, error) {
	hw := &HardwareWallet{
		ID:        uuid.New().String(),
		Type:     walletType,
		Model:   model,
		Serial:  serial,
		Connected: true,
		LastSeen: time.Now(),
		Features: getFeaturesForType(walletType),
	}
	
	// Generate keypair
	publicKey, address := generateKeyPair(walletType)
	hw.PublicKey = publicKey
	hw.Address = address
	
	// Get firmware version (simulated)
	hw.Firmware = getFirmwareVersion(walletType, model)
	
	hwMux.Lock()
	wallets[hw.ID] = hw
	hwMux.Unlock()
	
	return hw, nil
}

func getFeaturesForType(walletType string) []string {
	switch walletType {
	case "ledger":
		return []string{"sign_tx", "sign_message", "sign_typed_data", "get_public_key", " attestation"}
	case "trezor":
		return []string{"sign_tx", "sign_message", "sign_typed_data", "get_public_key", "matrix", " passphrase"}
	case "keystone":
		return []string{"sign_tx", "sign_message", "get_public_key", "qr_code", "air_gapped"}
	case "tangem":
		return []string{"sign_tx", "nfc", "tap_to_pay", "get_public_key"}
	default:
		return []string{"sign_tx", "get_public_key"}
	}
}

func getFirmwareVersion(walletType, model string) string {
	switch walletType {
	case "ledger":
		return "2.1.0"
	case "trezor":
		return "2.5.3"
	case "keystone":
		return "3.0.1"
	case "tangem":
		return "1.0.5"
	default:
		return "1.0.0"
	}
}

func generateKeyPair(walletType string) (string, string) {
	// Generate random keypair based on wallet type
	randomBytes := make([]byte, 64)
	rand.Read(randomBytes)
	
	publicKey := hex.EncodeToString(randomBytes[:32])
	
	// Derive address
	hash := sha256.Sum256(randomBytes[:32])
	address := "0x" + hex.EncodeToString(hash[:20])
	
	return publicKey, address
}

// DisconnectWallet disconnects a hardware wallet
func DisconnectWallet(walletID string) error {
	hwMux.Lock()
	defer hwMux.Unlock()
	
	if hw, ok := wallets[walletID]; ok {
		hw.Connected = false
		hw.LastSeen = time.Now()
	}
	
	return nil
}

// GetConnectedWallets returns all connected hardware wallets
func GetConnectedWallets() ([]HardwareWallet, error) {
	result := make([]HardwareWallet, 0)
	
	hwMux.RLock()
	for _, hw := range wallets {
		if hw.Connected {
			result = append(result, *hw)
		}
	}
	hwMux.RUnlock()
	
	return result, nil
}

// ============================================================================
// Session Management
// ============================================================================

// CreateSession creates a new device session
func CreateSession(walletID string) (*DeviceSession, error) {
	hwMux.RLock()
	hw, ok := wallets[walletID]
	hwMux.RUnlock()
	
	if !ok || !hw.Connected {
		return nil, fmt.Errorf("wallet not connected")
	}
	
	session := &DeviceSession{
		ID:         uuid.New().String(),
		WalletID:  walletID,
		DeviceID:  hw.ID,
		PublicKey: hw.PublicKey,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(10 * time.Minute),
		LastActive: time.Now(),
	}
	
	// Generate session key
	sessionKey := make([]byte, 32)
	rand.Read(sessionKey)
	session.SessionKey = encryptSessionKey(sessionKey)
	
	hwMux.Lock()
	sessions[session.ID] = session
	hwMux.Unlock()
	
	return session, nil
}

func encryptSessionKey(key []byte) string {
	// Simplified encryption for demo
	return hex.EncodeToString(key)
}

// ValidateSession validates a session
func ValidateSession(sessionID string) error {
	hwMux.RLock()
	session, ok := sessions[sessionID]
	hwMux.RUnlock()
	
	if !ok {
		return fmt.Errorf("session not found")
	}
	
	if time.Now().After(session.ExpiresAt) {
		return fmt.Errorf("session expired")
	}
	
	// Update last active
	hwMux.Lock()
	session.LastActive = time.Now()
	session.ExpiresAt = time.Now().Add(10 * time.Minute)
	hwMux.Unlock()
	
	return nil
}

// ============================================================================
// Signing
// ============================================================================

// CreateSigningRequest creates a new signing request
func CreateSigningRequest(walletID, sessionID, chainID, signType, payload string) (*SigningRequest, error) {
	if err := ValidateSession(sessionID); err != nil {
		return nil, err
	}
	
	req := &SigningRequest{
		ID:        uuid.New().String(),
		WalletID: walletID,
		SessionID: sessionID,
		ChainID:  parseInt(chainID),
		Type:     signType,
		Payload:  payload,
		Status:   "pending",
		CreatedAt: time.Now(),
	}
	
	hwMux.Lock()
	signingReqs[req.ID] = req
	hwMux.Unlock()
	
	return req, nil
}

// ApproveSigning approves a signing request
func ApproveSigning(requestID string) error {
	hwMux.Lock()
	defer hwMux.Unlock()
	
	if req, ok := signingReqs[requestID]; ok {
		req.Status = "approved"
		
		// In production, would send to hardware device
		// For now, generate mock signature
		sig := generateMockSignature(req.Payload)
		req.Signature = sig
		req.Status = "signed"
		req.CompletedAt = time.Now()
		
		return nil
	}
	
	return fmt.Errorf("request not found")
}

// RejectSigning rejects a signing request
func RejectSigning(requestID string) error {
	hwMux.Lock()
	defer hwMux.Unlock()
	
	if req, ok := signingReqs[requestID]; ok {
		req.Status = "rejected"
		req.CompletedAt = time.Now()
		return nil
	}
	
	return fmt.Errorf("request not found")
}

// GetSigningRequest returns a signing request
func GetSigningRequest(requestID string) (*SigningRequest, error) {
	hwMux.RLock()
	defer hwMux.RUnlock()
	
	if req, ok := signingReqs[requestID]; ok {
		return req, nil
	}
	
	return nil, fmt.Errorf("request not found")
}

// ============================================================================
// Air-Gapped Signing
// ============================================================================

// CreateAirGapTransaction creates a transaction for air-gapped signing
func CreateAirGapTransaction(walletID, unsignedTX string, chainID int) (*AirGapTransaction, error) {
	tx := &AirGapTransaction{
		ID:          uuid.New().String(),
		WalletID:   walletID,
		UnsignedTX: unsignedTX,
		ChainID:    chainID,
		Status:      "pending",
		CreatedAt:  time.Now(),
	}
	
	hwMux.Lock()
	airGapTXs[tx.ID] = tx
	hwMux.Unlock()
	
	return tx, nil
}

// SignAirGapTransaction signs an air-gapped transaction
func SignAirGapTransaction(txID, signature string) error {
	hwMux.Lock()
	defer hwMux.Unlock()
	
	if tx, ok := airGapTXs[txID]; ok {
		tx.Signature = signature
		tx.Status = "signed"
		tx.SignedAt = time.Now()
		return nil
	}
	
	return fmt.Errorf("transaction not found")
}

// BroadcastAirGapTransaction broadcasts a signed air-gapped transaction
func BroadcastAirGapTransaction(txID string) (string, error) {
	hwMux.Lock()
	tx, ok := airGapTXs[txID]
	hwMux.Unlock()
	
	if !ok {
		return "", fmt.Errorf("transaction not found")
	}
	
	if tx.Status != "signed" {
		return "", fmt.Errorf("transaction not signed")
	}
	
	// Generate mock transaction hash
	txHash := "0x" + hex.EncodeToString([]byte(tx.Signature))[:64]
	
	return txHash, nil
}

// ============================================================================
// HTTP Handlers
// ============================================================================

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "hardware"})
}

func connectHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var req struct {
		Type  string `json:"type"`
		Model string `json:"model"`
		Serial string `json:"serial"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	
	hw, err := ConnectWallet(req.Type, req.Model, req.Serial)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	
	json.NewEncoder(w).Encode(hw)
}

func disconnectHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var req struct {
		WalletID string `json:"wallet_id"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	
	if err := DisconnectWallet(req.WalletID); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	
	json.NewEncoder(w).Encode(map[string]string{"status": "disconnected"})
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	wallets, err := GetConnectedWallets()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	
	json.NewEncoder(w).Encode(wallets)
}

func sessionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var req struct {
		WalletID string `json:"wallet_id"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	
	session, err := CreateSession(req.WalletID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	
	json.NewEncoder(w).Encode(session)
}

func signRequestHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var req struct {
		WalletID string `json:"wallet_id"`
		SessionID string `json:"session_id"`
		ChainID string `json:"chain_id"`
		Type   string `json:"type"`
		Payload string `json:"payload"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	
	signingReq, err := CreateSigningRequest(req.WalletID, req.SessionID, req.ChainID, req.Type, req.Payload)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	
	json.NewEncoder(w).Encode(signingReq)
}

func approveHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var req struct {
		RequestID string `json:"request_id"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	
	if err := ApproveSigning(req.RequestID); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	
	req, _ := GetSigningRequest(req.RequestID)
	json.NewEncoder(w).Encode(req)
}

func rejectHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var req struct {
		RequestID string `json:"request_id"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	
	if err := RejectSigning(req.RequestID); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	
	json.NewEncoder(w).Encode(map[string]string{"status": "rejected"})
}

func airgapCreateHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var req struct {
		WalletID string `json:"wallet_id"`
		UnsignedTX string `json:"unsigned_tx"`
		ChainID int `json:"chain_id"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	
	tx, err := CreateAirGapTransaction(req.WalletID, req.UnsignedTX, req.ChainID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	
	json.NewEncoder(w).Encode(tx)
}

func airgapSignHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var req struct {
		TXID string `json:"tx_id"`
		Signature string `json:"signature"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	
	if err := SignAirGapTransaction(req.TXID, req.Signature); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	
	json.NewEncoder(w).Encode(map[string]string{"status": "signed"})
}

func airgapBroadcastHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var req struct {
		TXID string `json:"tx_id"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	
	hash, err := BroadcastAirGapTransaction(req.TXID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	
	json.NewEncoder(w).Encode(map[string]string{"hash": hash})
}

// ============================================================================
// Router
// ============================================================================

func router() http.Handler {
	mux := http.NewServeMux()
	
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/api/hardware/connect", connectHandler)
	mux.HandleFunc("/api/hardware/disconnect", disconnectHandler)
	mux.HandleFunc("/api/hardware/list", listHandler)
	mux.HandleFunc("/api/hardware/session", sessionHandler)
	mux.HandleFunc("/api/hardware/sign/request", signRequestHandler)
	mux.HandleFunc("/api/hardware/sign/approve", approveHandler)
	mux.HandleFunc("/api/hardware/sign/reject", rejectHandler)
	mux.HandleFunc("/api/airgap/create", airgapCreateHandler)
	mux.HandleFunc("/api/airgap/sign", airgapSignHandler)
	mux.HandleFunc("/api/airgap/broadcast", airgapBroadcastHandler)
	
	return mux
}

// ============================================================================
// Helpers
// ============================================================================

func parseInt(s string) int {
	if s == "" {
		return 0
	}
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

func generateMockSignature(payload string) string {
	randomBytes := make([]byte, 64)
	rand.Read(randomBytes)
	return hex.EncodeToString(randomBytes)
}

// ============================================================================
// Main
// ============================================================================

func main() {
	fmt.Printf("Hardware Wallet Service starting on port %d\n", HardwareServicePort)
	
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", HardwareServicePort),
		Handler:      router(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	
	fmt.Printf("Hardware Wallet Service ready on :%d\n", HardwareServicePort)
	if err := server.ListenAndServe(); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
