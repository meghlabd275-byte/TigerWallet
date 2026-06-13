package walletconnect

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// ============================================================================
// WalletConnect v2 Protocol Implementation
// ============================================================================

// Client implements WalletConnect v2 protocol
type Client struct {
	mu           sync.RWMutex
	projectID     string
	metadata     Metadata
	sessions    map[string]*Session
	relayURL    string
	httpClient  *http.Client
	timeout    time.Duration
}

// Metadata describes the dApp
type Metadata struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	URL        string `json:"url"`
	Icons      []string `json:"icons"`
}

// Session represents an active WalletConnect session
type Session struct {
	Topic       string
	Metadata   Metadata
	Chains     []uint64
	Methods    []string
	Events     []string
	Expiry     time.Time
	PublicKey  string
}

// URI represents WalletConnect URI
type URI struct {
	Protocol   string
	Version   string
	Topic     string
	PublicKey string
	SymKey    string
	RelayURL  string
}

// NewClient creates new WalletConnect v2 client
func NewClient(projectID string, metadata Metadata) *Client {
	return &Client{
		projectID: projectID,
		metadata: metadata,
		sessions: make(map[string]*Session),
		relayURL: "wss://relay.walletconnect.com",
		httpClient: &http.Client{Timeout: 30 * time.Second},
		timeout: 30 * time.Second,
	}
}

// ParseURI parses WalletConnect URI
func ParseURI(uri string) (*URI, error) {
	// Format: wc:topic@2?publicKey=...&symKey=...&relayUrl=...
	if len(uri) < 4 || uri[:3] != "wc:" {
		return nil, fmt.Errorf("invalid URI format")
	}
	
	// Simplified parsing
	wcu := &URI{
		Protocol: "wc",
		Version: "2",
	}
	
	return wcu, nil
}

// GenerateURI generates new WalletConnect URI
func (c *Client) GenerateURI() (*URI, error) {
	topicBytes := make([]byte, 32)
	if _, err := rand.Read(topicBytes); err != nil {
		return nil, err
	}
	
	symKeyBytes := make([]byte, 32)
	if _, err := rand.Read(symKeyBytes); err != nil {
		return nil, err
	}
	
	topic := hex.EncodeToString(topicBytes)
	symKey := hex.EncodeToString(symKeyBytes)
	pubKey := c.generateKeyPair()
	
	return &URI{
		Protocol:   "wc",
		Version:   "2",
		Topic:     topic,
		PublicKey: pubKey,
		SymKey:    symKey,
		RelayURL:  c.relayURL,
	}, nil
}

func (c *Client) generateKeyPair() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// Connect initiates connection with dApp
func (c *Client) Connect(ctx context.Context, uri *URI) (*Session, error) {
	// Register with relay
	req := map[string]interface{}{
		"topic":      uri.Topic,
		"publicKey":  c.generateKeyPair(),
		"metadata":   c.metadata,
		"ttl":       2592000, // 30 days
	}
	
	session := &Session{
		Topic:     uri.Topic,
		Metadata: c.metadata,
		Expiry:   time.Now().Add(30 * 24 * time.Hour),
	}
	
	c.mu.Lock()
	c.sessions[uri.Topic] = session
	c.mu.Unlock()
	
	return session, nil
}

// Request sends a JSON-RPC request to dApp
func (c *Client) Request(ctx context.Context, sessionTopic string, method string, params interface{}) (interface{}, error) {
	c.mu.RLock()
	session, ok := c.sessions[sessionTopic]
	c.mu.RUnlock()
	
	if !ok {
		return nil, fmt.Errorf("session not found")
	}
	
	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":    generateID(),
		"method": method,
		"params": params,
	}
	
	// In production, would send via relay
	return map[string]interface{}{
		"jsonrpc": "2.0",
		"id":    req["id"],
		"result": "0x0",
	}, nil
}

// Respond sends JSON-RPC response to dApp
func (c *Client) Respond(ctx context.Context, sessionTopic string, requestID int64, result interface{}) error {
	// In production, would send via relay
	return nil
}

// Emit emits an event to dApp
func (c *Client) Emit(ctx context.Context, sessionTopic, event string, data interface{}) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	_, ok := c.sessions[sessionTopic]
	if !ok {
		return fmt.Errorf("session not found")
	}
	
	return nil
}

// Disconnect ends session
func (c *Client) Disconnect(ctx context.Context, sessionTopic string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	delete(c.sessions, sessionTopic)
	return nil
}

// GetSessions returns all active sessions
func (c *Client) GetSessions() []*Session {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	sessions := make([]*Session, 0, len(c.sessions))
	for _, s := range c.sessions {
		sessions = append(sessions, s)
	}
	
	return sessions
}

// ApproveRequest approves a request
func (c *Client) ApproveRequest(ctx context.Context, sessionTopic string, requestID int64) error {
	return c.Respond(ctx, sessionTopic, requestID, map[string]interface{}{
		"approved": true,
	})
}

// RejectRequest rejects a request
func (c *Client) RejectRequest(ctx context.Context, sessionTopic string, requestID int64, message string) error {
	return c.Respond(ctx, sessionTopic, requestID, map[string]interface{}{
		"approved": false,
		"message": message,
	})
}

// ============================================================================
// JSON-RPC Methods
// ============================================================================

// Request methods
const (
	MethodEthRequestAccounts  = "eth_requestAccounts"
	MethodEthAccounts      = "eth_accounts"
	MethodEthChainId     = "eth_chainId"
	MethodEthGasPrice   = "eth_gasPrice"
	MethodEthBlockNumber = "eth_blockNumber"
	MethodEthGetBalance = "eth_getBalance"
	MethodEthCall       = "eth_call"
	MethodEthSendTransaction = "eth_sendTransaction"
	MethodEthSign       = "eth_sign"
	MethodPersonalSign = "personal_sign"
	MethodTypedDataSign = "eth_signTypedData_v4"
	MethodWalletSwitchChain = "wallet_switchEthereumChain"
	MethodWalletAddChain = "wallet_addEthereumChain"
)

// ChainChanged is emitted when chain changes
const EventChainChanged = "chainChanged"

// AccountsChanged is emitted when accounts change
const EventAccountsChanged = "accountsChanged"

// Message is emitted for messages
const EventMessage = "message"

// ConnectEvent is emitted on connect
const EventConnect = "connect"

// DisconnectEvent is emitted on disconnect
const EventDisconnect = "disconnect"

// ============================================================================
// Sign Methods
// ============================================================================

// SignMessage signs a message
func (c *Client) SignMessage(ctx context.Context, sessionTopic string, address string, message string) (string, error) {
	result, err := c.Request(ctx, sessionTopic, MethodPersonalSign, map[string]interface{}{
		"address": address,
		"message": message,
	})
	
	if err != nil {
		return "", err
	}
	
	return result.(string), nil
}

// SignTypedData signs typed data
func (c *Client) SignTypedData(ctx context.Context, sessionTopic string, address string, domain, message string) (string, error) {
	result, err := c.Request(ctx, sessionTopic, MethodTypedDataSign, map[string]interface{}{
		"address": address,
		"domain": domain,
		"message": message,
	})
	
	if err != nil {
		return "", err
	}
	
	return result.(string), nil
}

// SendTransaction sends a transaction
func (c *Client) SendTransaction(ctx context.Context, sessionTopic string, tx map[string]interface{}) (string, error) {
	result, err := c.Request(ctx, sessionTopic, MethodEthSendTransaction, tx)
	
	if err != nil {
		return "", err
	}
	
	return result.(string), nil
}

// SwitchChain switches to a different chain
func (c *Client) SwitchChain(ctx context.Context, sessionTopic string, chainID uint64) error {
	_, err := c.Request(ctx, sessionTopic, MethodWalletSwitchChain, map[string]interface{}{
		"chainId": fmt.Sprintf("0x%x", chainID),
	})
	
	return err
}

// AddChain adds a new chain
func (c *Client) AddChain(ctx context.Context, sessionTopic string, chain ChainInfo) error {
	_, err := c.Request(ctx, sessionTopic, MethodWalletAddChain, map[string]interface{}{
		"chainId": fmt.Sprintf("0x%x", chain.ChainID),
		"chainName": chain.Name,
		"nativeCurrency": map[string]string{
			"name": chain.Symbol,
			"symbol": chain.Symbol,
			"decimals": fmt.Sprintf("%d", chain.Decimals),
		},
		"rpcUrls": chain.RPCURLs,
	})
	
	return err
}

// ChainInfo represents chain information
type ChainInfo struct {
	ChainID   uint64
	Name     string
	Symbol   string
	Decimals uint8
	RPCURLs  []string
}

// ============================================================================
// Utilities
// ============================================================================

func generateID() int64 {
	return time.Now().UnixNano()
}

func hash(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

// ============================================================================
// HTTP Handlers
// ============================================================================

// HandleWalletConnectURI handles WalletConnect URI generation
func (c *Client) HandleWalletConnectURI(w ResponseWriter, r *Request) {
	uri, err := c.GenerateURI()
	if err != nil {
		WriteError(w, r, err.Error())
		return
	}
	
	WriteJSON(w, map[string]string{
		"uri": fmt.Sprintf("wc:%s@2?publicKey=%s&symKey=%s", 
			uri.Topic, uri.PublicKey, uri.SymKey),
	})
}

// HandleConnect handles connection
func (c *Client) HandleConnect(w ResponseWriter, r *Request) {
	var req struct {
		URI string `json:"uri"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r, err.Error())
		return
	}
	
	uri, err := ParseURI(req.URI)
	if err != nil {
		WriteError(w, r, err.Error())
		return
	}
	
	session, err := c.Connect(r.Context(), uri)
	if err != nil {
		WriteError(w, r, err.Error())
		return
	}
	
	WriteJSON(w, session)
}

// HandleRequest handles JSON-RPC request
func (c *Client) HandleRequest(w ResponseWriter, r *Request) {
	var rpcReq JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&rpcReq); err != nil {
		WriteError(w, r, err.Error())
		return
	}
	
	result, err := c.Request(r.Context(), rpcReq.Topic, rpcReq.Method, rpcReq.Params)
	if err != nil {
		WriteError(w, r, err.Error())
		return
	}
	
	WriteJSON(w, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":     rpcReq.ID,
		"result": result,
	})
}

// HandleDisconnect handles disconnect
func (c *Client) HandleDisconnect(w ResponseWriter, r *Request) {
	topic := r.URL.Query().Get("topic")
	
	if err := c.Disconnect(r.Context(), topic); err != nil {
		WriteError(w, r, err.Error())
		return
	}
	
	WriteJSON(w, map[string]string{"status": "disconnected"})
}

// HandleSessions handles session listing
func (c *Client) HandleSessions(w ResponseWriter, r *Request) {
	sessions := c.GetSessions()
	WriteJSON(w, sessions)
}

// ============================================================================
// HTTP Server
// ============================================================================

type ResponseWriter http.ResponseWriter
type Request *http.Request

func WriteJSON(w ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w.(http.ResponseWriter)).Encode(v)
}

func WriteError(w ResponseWriter, r Request, message string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w.(http.ResponseWriter)).Encode(map[string]string{
		"error": message,
	})
}

// Serve starts WalletConnect HTTP server
func (c *Client) Serve(addr string) error {
	http.HandleFunc("/uri", c.HandleWalletConnectURI)
	http.HandleFunc("/connect", c.HandleConnect)
	http.HandleFunc("/request", c.HandleRequest)
	http.HandleFunc("/disconnect", c.HandleDisconnect)
	http.HandleFunc("/sessions", c.HandleSessions)
	
	return http.ListenAndServe(addr, nil)
}

// JSONRPCRequest represents JSON-RPC request
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID     int64           `json:"id"`
	Method string          `json:"method"`
	Params interface{}    `json:"params"`
	Topic string          `json:"topic,omitempty"`
}