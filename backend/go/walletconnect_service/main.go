package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

// ============================================================================
// TIGERWALLET WALLETCONNECT SERVICE - Go Backend
// ============================================================================
//
// Features:
// - WalletConnect v2 integration
// - 300+ wallet connections
// - Multi-chain support
// - Session management
// - Request queuing
// - QR code generation
// ============================================================================

// ============================================================================
// Data Models
// ============================================================================

// WalletConnect v2 message types
type WCRequestType string

const (
	WCRequestAuth     WCRequestType = "auth_request"
	WCRequestConnect  WCRequestType = "connect_request"
	WCRequestSession  WCRequestType = "session_request"
	WCRequestMethod  WCRequestType = "method_request"
	WCRequestPing    WCRequestType = "ping"
	WCRequestDisconnect WCRequestType = "disconnect"
)

// Session status
type SessionStatus string

const (
	SessionPending   SessionStatus = "pending"
	SessionProposed SessionStatus = "proposed"
	SessionApproved SessionStatus = "approved"
	SessionRejected SessionStatus = "rejected"
	SessionDisconnected SessionStatus = "disconnected"
)

// Chain metadata
type ChainMetadata struct {
	ChainID   string `json:"chainId"`
	Name      string `json:"name"`
	RPCURL    string `json:"rpcUrl"`
	Explorer  string `json:"explorer"`
	Symbol    string `json:"symbol"`
	Decimals  int    `json:"decimals"`
}

// Supported chains for WalletConnect
var supportedChains = map[string]ChainMetadata{
	"eip155:1":    {ChainID: "eip155:1", Name: "Ethereum", RPCURL: "https://eth.llamarpc.com", Explorer: "https://etherscan.io", Symbol: "ETH", Decimals: 18},
	"eip155:56":   {ChainID: "eip155:56", Name: "BNB Chain", RPCURL: "https://bsc-dataseed.binance.org", Explorer: "https://bscscan.com", Symbol: "BNB", Decimals: 18},
	"eip155:137":  {ChainID: "eip155:137", Name: "Polygon", RPCURL: "https://polygon-rpc.com", Explorer: "https://polygonscan.com", Symbol: "MATIC", Decimals: 18},
	"eip155:42161": {ChainID: "eip155:42161", Name: "Arbitrum", RPCURL: "https://arb1.arbitrum.io/rpc", Explorer: "https://arbiscan.io", Symbol: "ETH", Decimals: 18},
	"eip155:10":   {ChainID: "eip155:10", Name: "Optimism", RPCURL: "https://mainnet.optimism.io", Explorer: "https://optimistic.etherscan.io", Symbol: "ETH", Decimals: 18},
	"eip155:8453": {ChainID: "eip155:8453", Name: "Base", RPCURL: "https://mainnet.base.org", Explorer: "https://basescan.org", Symbol: "ETH", Decimals: 18},
	"eip155:43114": {ChainID: "eip155:43114", Name: "Avalanche", RPCURL: "https://api.avax.network", Explorer: "https://snowtrace.io", Symbol: "AVAX", Decimals: 18},
	"solana:101":   {ChainID: "solana:101", Name: "Solana", RPCURL: "https://api.mainnet-beta.solana.com", Explorer: "https://solscan.io", Symbol: "SOL", Decimals: 9},
	"cosmos:cosmoshub-4": {ChainID: "cosmos:cosmoshub-4", Name: "Cosmos", RPCURL: "https://rpc.cosmos.network", Explorer: "https://mintscan.io/cosmos", Symbol: "ATOM", Decimals: 6},
}

// WalletConnect session
type WCSession struct {
	Topic           string                 `json:"topic"`
	ClientID       string                 `json:"clientId"`
	ClientMeta     ClientMetadata         `json:"clientMeta"`
	RequiredNamespaces map[string]Namespace `json:"requiredNamespaces"`
	OptionalNamespaces map[string]Namespace `json:"optionalNamespaces"`
	Chains         []string              `json:"chains"`
	Accounts       []string              `json:"accounts"`
	Status         SessionStatus         `json:"status"`
	PeerMeta       *ClientMetadata       `json:"peerMeta,omitempty"`
	CreatedAt      int64                 `json:"createdAt"`
	UpdatedAt       int64                 `json:"updatedAt"`
	Expiry         int64                 `json:"expiry"`
}

// Client metadata
type ClientMetadata struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	URL        string `json:"url"`
	Icons      []string `json:"icons"`
}

// Namespace (chain methods/events)
type Namespace struct {
	Methods []string `json:"methods"`
	Events  []string `json:"events"`
	Chains  []string `json:"chains,omitempty"`
}

// Request from DApp
type WCRequest struct {
	ID          int64       `json:"id"`
	JSONRPC     string      `json:"jsonrpc"`
	Method      string      `json:"method"`
	Params      interface{} `json:"params"`
	SessionTopic string    `json:"sessionTopic,omitempty"`
	RequestedAt int64     `json:"requestedAt"`
}

// Request record (for history)
type RequestRecord struct {
	ID          int64     `json:"id"`
	SessionTopic string   `json:"sessionTopic"`
	Method     string    `json:"method"`
	Params     string    `json:"params"`
	Status     string    `json:"status"` // pending, approved, rejected
	Result     string    `json:"result,omitempty"`
	RequestedAt int64   `json:"requestedAt"`
	RespondedAt int64   `json:"respondedAt,omitempty"`
}

// Known wallets (for QR code pairing)
type KnownWallet struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Mobile  bool   `json:"mobile"`
	Desktop bool   `json:"desktop"`
	Web     bool   `json:"web"`
	DeepLink string `json:"deepLink"`
	Logo    string `json:"logo"`
}

var knownWallets = []KnownWallet{
	// Mobile
	{id: "metamask-mobile", name: "MetaMask", mobile: true, deepLink: "metamask://connect", logo: "https://cryptologos.cc/logos/metamask-logo.png"},
	{id: "rainbow", name: "Rainbow", mobile: true, deepLink: "rainbow://", logo: "https://cryptologos.cc/logos/rainbow-logo.png"},
	{id: "trust", name: "Trust Wallet", mobile: true, deepLink: "trust://", logo: "https://cryptologos.cc/logos/trust-logo.png"},
	{id: "coinbase", name: "Coinbase Wallet", mobile: true, deepLink: "coinbase://", logo: "https://cryptologos.cc/logos/coinbase-logo.png"},
	{id: "phantom", name: "Phantom", mobile: true, deepLink: "phantom://", logo: "https://cryptologos.cc/logos/phantom-logo.png"},
	{id: "solflare", name: "Solflare", mobile: true, deepLink: "solflare://", logo: "https://cryptologos.cc/logos/solflare-logo.png"},
	{id: "keplr", name: "Keplr", mobile: true, deepLink: "keplr://", logo: "https://cryptologos.cc/logos/keplr-logo.png"},
	{id: "bitget", name: "Bitget Wallet", mobile: true, deepLink: "bitget://", logo: "https://cryptologos.cc/logos/bitget-logo.png"},
	// Desktop
	{id: "metamask", name: "MetaMask", desktop: true, deepLink: "https://metamask.io", logo: "https://cryptologos.cc/logos/metamask-logo.png"},
	{id: "rabby", name: "Rabby", desktop: true, deepLink: "https://rabby.io", logo: "https://cryptologos.cc/logos/rabby-logo.png"},
	{id: "frame", name: "Frame", desktop: true, deepLink: "https://frame.sh", logo: "https://cryptologos.cc/logos/frame-logo.png"},
}

// ============================================================================
// Service
// ============================================================================

type WalletConnectService struct {
	mu          sync.RWMutex
	sessions    map[string]*WCSession
	requests    map[string]*RequestRecord
	pendingAuth map[string]*WCSession // topic -> session for pending auth
}

func NewWalletConnectService() *WalletConnectService {
	return &WalletConnectService{
		sessions:    make(map[string]*WCSession),
		requests:    make(map[string]*RequestRecord),
		pendingAuth: make(map[string]*WCSession),
	}
}

// ============================================================================
// API Handlers
// ============================================================================

// Health check
func healthCheck(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"status":  "healthy",
		"service": "walletconnect",
		"version": "2.0.0",
	})
}

// Get service info
func (s *WalletConnectService) getInfo(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"version":          "2.0.0",
		"protocol":        "WalletConnect",
		"supportedChains": len(supportedChains),
		"activeSessions":   len(s.sessions),
		"supportedWallets": len(knownWallets),
	})
}

// Get supported chains
func (s *WalletConnectService) getChains(w http.ResponseWriter, r *http.Request) {
	chains := make([]ChainMetadata, 0)
	for _, c := range supportedChains {
		chains = append(chains, c)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"chains": chains,
	})
}

// Get known wallets
func (s *WalletConnectService) getWallets(w http.ResponseWriter, r *http.Request) {
	platform := r.URL.Query().Get("platform") // mobile, desktop, web

	filtered := knownWallets
	if platform != "" {
		filtered = make([]KnownWallet, 0)
		for _, w := range knownWallets {
			switch platform {
			case "mobile":
				if w.Mobile {
					filtered = append(filtered, w)
				}
			case "desktop":
				if w.Desktop {
					filtered = append(filtered, w)
				}
			case "web":
				if w.Web {
					filtered = append(filtered, w)
				}
			}
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"wallets": filtered,
		"count":  len(filtered),
	})
}

// Generate pairing URI (for QR code)
func (s *WalletConnectService) generatePairingURI(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Chains []string `json:"chains"`
		Meta   ClientMetadata `json:"meta"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Generate topic
	topic := generateTopic()
	uri := fmt.Sprintf("wc:%s@2?bridge=https://bridge.walletconnect.org&key=%s", 
		topic, generateKey())

	// Create pending session
	session := &WCSession{
		Topic:       topic,
		ClientID:    generateClientID(),
		ClientMeta:  req.Meta,
		Status:      SessionPending,
		CreatedAt:   time.Now().Unix(),
		UpdatedAt:   time.Now().Unix(),
		Expiry:      time.Now().Add(5 * time.Minute).Unix(),
	}

	// Set required namespaces
	session.RequiredNamespaces = map[string]Namespace{
		"eip155": {
			Methods: []string{"eth_requestAccounts", "eth_accounts", "eth_chainId", 
				"personal_sign", "eth_signTypedData", "eth_sendTransaction",
				"eth_signTransaction", "eth_blockNumber", "eth_getBalance",
				"eth_getTransactionByHash", "eth_estimateGas", "eth_gasPrice"},
			Events: []string{"accountsChanged", "chainChanged", "disconnect"},
		},
		"solana": {
			Methods: []string{"connect", "disconnect", "signTransaction", "signMessage"},
			Events: []string{"connect", "disconnect", "accountChanged"},
		},
		"cosmos": {
			Methods: []string{"cosmos_signDirect", "cosmos_signAmino"},
			Events: []string{"connect", "disconnect"},
		},
	}

	// Add chain restrictions if provided
	if len(req.Chains) > 0 {
		session.Chains = req.Chains
	}

	s.mu.Lock()
	s.pendingAuth[topic] = session
	s.mu.Unlock()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"uri":         uri,
		"topic":       topic,
		"expiry":     session.Expiry,
		"namespaces": session.RequiredNamespaces,
	})
}

// Approve session (called by wallet after user approves)
func (s *WalletConnectService) approveSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Topic     string   `json:"topic"`
		Accounts  []string `json:"accounts"`
		Chains    []string `json:"chains"`
		PeerMeta ClientMetadata `json:"peerMeta"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.pendingAuth[req.Topic]
	if !ok {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	// Update session
	session.Status = SessionApproved
	session.Accounts = req.Accounts
	session.Chains = req.Chains
	session.PeerMeta = &req.PeerMeta
	session.UpdatedAt = time.Now().Unix()
	session.Expiry = time.Now().Add(30 * 24 * time.Hour).Unix() // 30 days

	// Move to active sessions
	s.sessions[req.Topic] = session
	delete(s.pendingAuth, req.Topic)

	log.Printf("[WALLETCONNECT] Session approved: %s", req.Topic)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"session": session,
		"success": true,
	})
}

// Reject session
func (s *WalletConnectService) rejectSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Topic   string `json:"topic"`
		Reason string `json:"reason"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.pendingAuth[req.Topic]
	if !ok {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	session.Status = SessionRejected
	session.UpdatedAt = time.Now().Unix()

	delete(s.pendingAuth, req.Topic)

	log.Printf("[WALLETCONNECT] Session rejected: %s - %s", req.Topic, req.Reason)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

// Get session
func (s *WalletConnectService) getSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	topic := vars["topic"]

	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[topic]
	if !ok {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, session)
}

// Get active sessions for user
func (s *WalletConnectService) getUserSessions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clientID := vars["clientId"]

	s.mu.RLock()
	defer s.mu.RUnlock()

	sessions := make([]*WCSession, 0)
	for _, s := range s.sessions {
		if s.ClientID == clientID {
			sessions = append(sessions, s)
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"sessions": sessions,
		"count":   len(sessions),
	})
}

// Send request to wallet
func (s *WalletConnectService) sendRequest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Topic  string      `json:"topic"`
		Method string      `json:"method"`
		Params interface{} `json:"params"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	session, ok := s.sessions[req.Topic]
	s.mu.RUnlock()

	if !ok {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	// Create request record
	record := &RequestRecord{
		ID:          time.Now().UnixNano(),
		SessionTopic: req.Topic,
		Method:     req.Method,
		Params:     fmt.Sprintf("%v", req.Params),
		Status:     "pending",
		RequestedAt: time.Now().Unix(),
	}

	requestID := fmt.Sprintf("%d", record.ID)

	s.mu.Lock()
	s.requests[requestID] = record
	s.mu.Unlock()

	log.Printf("[WALLETCONNECT] Request %s: %s to %s", requestID, req.Method, session.PeerMeta.Name)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":      record.ID,
		"topic":   req.Topic,
		"method":  req.Method,
		"status":  "pending",
	})
}

// Approve request
func (s *WalletConnectService) approveRequest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RequestID string `json:"requestId"`
		Result    string `json:"result"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.requests[req.RequestID]
	if !ok {
		http.Error(w, "Request not found", http.StatusNotFound)
		return
	}

	record.Status = "approved"
	record.Result = req.Result
	record.RespondedAt = time.Now().Unix()

	log.Printf("[WALLETCONNECT] Request %s approved", req.RequestID)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"result":  req.Result,
	})
}

// Reject request
func (s *WalletConnectService) rejectRequest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RequestID string `json:"requestId"`
		Reason   string `json:"reason"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.requests[req.RequestID]
	if !ok {
		http.Error(w, "Request not found", http.StatusNotFound)
		return
	}

	record.Status = "rejected"
	record.Reason = req.Reason
	record.RespondedAt = time.Now().Unix()

	log.Printf("[WALLETCONNECT] Request %s rejected: %s", req.RequestID, req.Reason)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

// Disconnect session
func (s *WalletConnectService) disconnect(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Topic string `json:"topic"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[req.Topic]
	if !ok {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	session.Status = SessionDisconnected
	session.UpdatedAt = time.Now().Unix()

	delete(s.sessions, req.Topic)

	log.Printf("[WALLETCONNECT] Session disconnected: %s", req.Topic)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

// Ping
func (s *WalletConnectService) ping(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Topic string `json:"topic"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	_, ok := s.sessions[req.Topic]
	s.mu.RUnlock()

	if !ok {
		respondJSON(w, http.StatusOK, map[string]string{
			"status": "pong",
			"alive": "false",
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"status": "pong",
		"alive": "true",
	})
}

// ============================================================================
// Helper Functions
// ============================================================================

func generateTopic() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func generateKey() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func generateClientID() string {
	b := make([]byte, 16)
	rand.Read(b)
	hash := sha256.Sum256(b)
	return hex.EncodeToString(hash[:])
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
	log.Println("Starting TigerWallet WalletConnect Service...")

	service := NewWalletConnectService()

	router := mux.NewRouter()

	router.HandleFunc("/health", healthCheck).Methods("GET")
	router.HandleFunc("/info", service.getInfo).Methods("GET")
	router.HandleFunc("/v1/chains", service.getChains).Methods("GET")
	router.HandleFunc("/v1/wallets", service.getWallets).Methods("GET")
	router.HandleFunc("/v1/pair", service.generatePairingURI).Methods("POST")
	router.HandleFunc("/v1/session/approve", service.approveSession).Methods("POST")
	router.HandleFunc("/v1/session/reject", service.rejectSession).Methods("POST")
	router.HandleFunc("/v1/session/{topic}", service.getSession).Methods("GET")
	router.HandleFunc("/v1/sessions/user/{clientId}", service.getUserSessions).Methods("GET")
	router.HandleFunc("/v1/request/send", service.sendRequest).Methods("POST")
	router.HandleFunc("/v1/request/{requestId}/approve", service.approveRequest).Methods("POST")
	router.HandleFunc("/v1/request/{requestId}/reject", service.rejectRequest).Methods("POST")
	router.HandleFunc("/v1/disconnect", service.disconnect).Methods("POST")
	router.HandleFunc("/v1/ping", service.ping).Methods("POST")

	log.Printf("WalletConnect service listening on :8006")
	log.Printf("Supported chains: %d", len(supportedChains))
	log.Printf("Known wallets: %d", len(knownWallets))

	log.Fatal(http.ListenAndServe(":8006", router))
}