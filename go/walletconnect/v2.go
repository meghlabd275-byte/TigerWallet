package walletconnect

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ============================================================================
// WalletConnect v2 Protocol Implementation
//
// This implementation communicates with the real WalletConnect v2 relay
// (wss://relay.walletconnect.com) over WebSocket. It implements the relay
// publish/subscribe protocol on topics, and encrypts/decrypts payloads with
// AES-256-GCM using the symKey carried in the wc: URI.
// ============================================================================

// relayDialer is the WebSocket dialer used to reach the relay.
var relayDialer = &websocket.Dialer{
	HandshakeTimeout: 10 * time.Second,
}

// Client implements WalletConnect v2 protocol
type Client struct {
	mu         sync.RWMutex
	projectID  string
	metadata   Metadata
	sessions   map[string]*Session
	relayURL   string
	httpClient *http.Client
	timeout    time.Duration

	// Relay connection state. A single WebSocket is shared across all
	// sessions because the relay multiplexes topics over one connection.
	connMu       sync.Mutex
	conn         *websocket.Conn
	connCancel   context.CancelFunc
	connCtx      context.Context
	connDone     chan struct{}
	subscribed   map[string]bool     // topics currently subscribed on the relay
	symKeys      map[string][]byte   // topic -> 32-byte symKey
	pending      map[int64]chan *rpcEnvelope // request id -> response waiter
	pendingMutex sync.Mutex
	handlers     map[string]func(*rpcEnvelope) // inbound peer request/notification handlers
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
	Topic      string
	Metadata   Metadata
	Chains     []uint64
	Methods    []string
	Events     []string
	Expiry     time.Time
	PublicKey  string
	symKey     []byte // 32-byte symmetric key used for relay encryption
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
		projectID:  projectID,
		metadata:   metadata,
		sessions:   make(map[string]*Session),
		relayURL:   "wss://relay.walletconnect.com",
		httpClient: &http.Client{Timeout: 30 * time.Second},
		timeout:    30 * time.Second,
		subscribed: make(map[string]bool),
		symKeys:    make(map[string][]byte),
		pending:    make(map[int64]chan *rpcEnvelope),
		handlers:   make(map[string]func(*rpcEnvelope)),
	}
}

// SetHandler registers a callback for inbound peer requests/notifications
// delivered on the given session topic. Pass nil to clear.
func (c *Client) SetHandler(topic string, handler func(*rpcEnvelope)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers[topic] = handler
}

// ParseURI parses a WalletConnect v2 URI of the form:
//
//	wc:<topic>@2?publicKey=<hex>&symKey=<hex>&relay-protocol=<p>&[relayUrl=<url>]
//
// The query parameters may use either "symKey"/"symkey" or "relayUrl"/"relay-protocol".
func ParseURI(uri string) (*URI, error) {
	if len(uri) < 4 || uri[:3] != "wc:" {
		return nil, fmt.Errorf("invalid URI: missing wc: scheme")
	}
	rest := uri[3:]
	at := -1
	for i := 0; i < len(rest); i++ {
		if rest[i] == '@' {
			at = i
			break
		}
		if rest[i] == '?' {
			break
		}
	}
	if at < 0 {
		return nil, fmt.Errorf("invalid URI: missing version segment")
	}
	topic := rest[:at]
	// version segment runs until '?' or end
	verEnd := len(rest)
	for i := at; i < len(rest); i++ {
		if rest[i] == '?' {
			verEnd = i
			break
		}
	}
	version := rest[at+1 : verEnd]

	wcu := &URI{
		Protocol: "wc",
		Version:  version,
		Topic:    topic,
		RelayURL: "wss://relay.walletconnect.com",
	}
	if verEnd < len(rest) {
		q, err := url.ParseQuery(rest[verEnd+1:])
		if err != nil {
			return nil, fmt.Errorf("invalid URI query: %w", err)
		}
		wcu.PublicKey = q.Get("publicKey")
		if k := q.Get("symKey"); k != "" {
			wcu.SymKey = k
		} else {
			wcu.SymKey = q.Get("symkey")
		}
		if r := q.Get("relayUrl"); r != "" {
			wcu.RelayURL = r
		}
		// relay-protocol= only names the protocol (e.g. "waku"); the relay
		// URL stays at the WalletConnect default.
	}
	if wcu.Topic == "" {
		return nil, fmt.Errorf("invalid URI: missing topic")
	}
	if wcu.SymKey == "" {
		return nil, fmt.Errorf("invalid URI: missing symKey")
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

// Connect initiates a real connection with the peer via the WalletConnect
// relay. It dials the relay WebSocket (reusing an existing connection if one
// is already open), subscribes to the session topic, stores the symmetric key
// for encryption, and publishes an encrypted wc_sessionProposal payload.
func (c *Client) Connect(ctx context.Context, uri *URI) (*Session, error) {
	if uri == nil {
		return nil, fmt.Errorf("nil URI")
	}
	if uri.Topic == "" {
		return nil, fmt.Errorf("URI missing topic")
	}
	symKey, err := decodeSymKey(uri.SymKey)
	if err != nil {
		return nil, fmt.Errorf("invalid symKey: %w", err)
	}

	if err := c.ensureRelay(ctx, uri.RelayURL); err != nil {
		return nil, fmt.Errorf("relay connect: %w", err)
	}

	if err := c.subscribe(ctx, uri.Topic); err != nil {
		return nil, fmt.Errorf("subscribe: %w", err)
	}

	c.connMu.Lock()
	c.symKeys[uri.Topic] = symKey
	c.connMu.Unlock()

	// Publish the session proposal so the peer wallet can pair on the topic.
	proposal := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      generateID(),
		"method":  "wc_sessionProposal",
		"params": map[string]interface{}{
			"topic":     uri.Topic,
			"publicKey": uri.PublicKey,
			"metadata":  c.metadata,
			"ttl":       2592000, // 30 days, in seconds
		},
	}
	if err := c.publish(ctx, uri.Topic, proposal); err != nil {
		return nil, fmt.Errorf("publish proposal: %w", err)
	}

	session := &Session{
		Topic:     uri.Topic,
		Metadata:  c.metadata,
		Expiry:    time.Now().Add(30 * 24 * time.Hour),
		PublicKey: uri.PublicKey,
		symKey:    symKey,
	}

	c.mu.Lock()
	c.sessions[uri.Topic] = session
	c.mu.Unlock()

	return session, nil
}

// Request sends a JSON-RPC request to the peer over the relay and waits for
// the matching response (or until the context is cancelled / the client
// timeout elapses). The fake "0x0" placeholder has been removed.
func (c *Client) Request(ctx context.Context, sessionTopic string, method string, params interface{}) (interface{}, error) {
	c.mu.RLock()
	_, ok := c.sessions[sessionTopic]
	c.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("session not found")
	}

	id := generateID()
	envelope := &rpcEnvelope{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	wait := make(chan *rpcEnvelope, 1)
	c.pendingMutex.Lock()
	c.pending[id] = wait
	c.pendingMutex.Unlock()
	defer func() {
		c.pendingMutex.Lock()
		delete(c.pending, id)
		c.pendingMutex.Unlock()
	}()

	if err := c.publish(ctx, sessionTopic, envelope); err != nil {
		return nil, fmt.Errorf("publish request: %w", err)
	}

	// Resolve the deadline: the earlier of ctx and the client default timeout.
	timeout := c.timeout
	if dl, ok := ctx.Deadline(); ok {
		if remaining := time.Until(dl); remaining < timeout {
			timeout = remaining
		}
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case resp := <-wait:
		if resp == nil {
			// Channel was closed because the relay connection dropped.
			return nil, errors.New("relay connection closed")
		}
		if resp.Error != nil {
			return nil, fmt.Errorf("rpc error %d: %s", resp.Error.Code, resp.Error.Message)
		}
		return resp.Result, nil
	case <-timer.C:
		return nil, fmt.Errorf("request timed out after %s", timeout)
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.relayDone():
		return nil, errors.New("relay connection closed")
	}
}

// Respond sends a JSON-RPC response to the peer over the relay.
func (c *Client) Respond(ctx context.Context, sessionTopic string, requestID int64, result interface{}) error {
	c.mu.RLock()
	_, ok := c.sessions[sessionTopic]
	c.mu.RUnlock()
	if !ok {
		return fmt.Errorf("session not found")
	}
	envelope := &rpcEnvelope{
		JSONRPC: "2.0",
		ID:      requestID,
		Result:  result,
	}
	return c.publish(ctx, sessionTopic, envelope)
}

// Emit sends a JSON-RPC notification (no id) to the peer over the relay.
func (c *Client) Emit(ctx context.Context, sessionTopic, event string, data interface{}) error {
	c.mu.RLock()
	_, ok := c.sessions[sessionTopic]
	c.mu.RUnlock()
	if !ok {
		return fmt.Errorf("session not found")
	}
	envelope := &rpcEnvelope{
		JSONRPC: "2.0",
		Method:  event,
		Params:  data,
	}
	return c.publish(ctx, sessionTopic, envelope)
}

// Disconnect ends the session: unsubscribes from the relay topic and removes
// local state. The shared relay WebSocket is torn down once no sessions
// remain.
func (c *Client) Disconnect(ctx context.Context, sessionTopic string) error {
	c.mu.Lock()
	delete(c.sessions, sessionTopic)
	c.mu.Unlock()

	c.connMu.Lock()
	delete(c.symKeys, sessionTopic)
	wasSubscribed := c.subscribed[sessionTopic]
	delete(c.subscribed, sessionTopic)
	noSessionsLeft := len(c.symKeys) == 0
	c.connMu.Unlock()

	if wasSubscribed {
		_ = c.unsubscribe(sessionTopic)
	}

	if noSessionsLeft {
		c.closeRelay()
	}
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

// generateID returns a JSON-RPC id drawn from crypto/rand so that concurrent
// requests never collide (time-based ids could overlap under high load).
func generateID() int64 {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Fall back to a time-based id if the system RNG is unavailable.
		return time.Now().UnixNano()
	}
	return int64(binary.BigEndian.Uint64(b[:]))
}

func hash(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

// ============================================================================
// Relay protocol types
// ============================================================================

// rpcEnvelope is the JSON-RPC 2.0 envelope carried (encrypted) inside relay
// publish messages. Either Method/Params (request/notification) or
// Result/Error (response) is set.
type rpcEnvelope struct {
	JSONRPC string     `json:"jsonrpc"`
	ID      int64      `json:"id,omitempty"`
	Method  string     `json:"method,omitempty"`
	Params  interface{} `json:"params,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *rpcError  `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// relayMessage is the WalletConnect relay wire envelope. The "type" field
// selects the operation: "sub" (subscribe), "unsub" (unsubscribe) or
// "pub" (publish). For publishes, "message" carries the hex-encoded
// AES-256-GCM ciphertext of the JSON-RPC envelope.
type relayMessage struct {
	Topic   string `json:"topic"`
	Type    string `json:"type"`
	Message string `json:"message,omitempty"`
	TTL     int64  `json:"ttl,omitempty"`
}

const (
	relayDefaultTTL = 2592000 // 30 days in seconds
	relayProtocol   = "waku"
)

// ============================================================================
// Relay connection management
// ============================================================================

// ensureRelay opens the shared WebSocket to the relay if it is not already
// open, then starts the read pump that decrypts inbound messages and
// dispatches JSON-RPC responses to waiting Request callers.
func (c *Client) ensureRelay(ctx context.Context, relayURL string) error {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	if c.conn != nil {
		return nil
	}
	if relayURL == "" {
		relayURL = c.relayURL
	}

	conn, err := c.dialRelay(ctx, relayURL)
	if err != nil {
		return err
	}

	rctx, cancel := context.WithCancel(context.Background())
	c.conn = conn
	c.connCtx = rctx
	c.connCancel = cancel
	c.connDone = make(chan struct{})

	go c.readPump(rctx)
	return nil
}

// dialRelay opens the WebSocket to the WalletConnect relay. The relay
// requires the project id as the "projectId" query parameter and the relay
// protocol ("waku") as "relay-protocol".
func (c *Client) dialRelay(ctx context.Context, relayURL string) (*websocket.Conn, error) {
	u, err := url.Parse(relayURL)
	if err != nil {
		return nil, fmt.Errorf("invalid relay URL: %w", err)
	}
	q := u.Query()
	if c.projectID != "" {
		q.Set("projectId", c.projectID)
	}
	q.Set("relay-protocol", relayProtocol)
	u.RawQuery = q.Encode()

	conn, resp, err := relayDialer.DialContext(ctx, u.String(), nil)
	if err != nil {
		if resp != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("relay dial failed (HTTP %d): %w", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("relay dial failed: %w", err)
	}
	return conn, nil
}

// relayDone returns a channel that is closed when the read pump exits. If no
// relay is currently open, a nil channel is returned which blocks forever in
// a select (so callers fall through to ctx/timer instead).
func (c *Client) relayDone() chan struct{} {
	c.connMu.Lock()
	defer c.connMu.Unlock()
	return c.connDone
}

// closeRelay tears down the shared relay connection. Safe to call when no
// connection is open.
func (c *Client) closeRelay() {
	c.connMu.Lock()
	conn := c.conn
	cancel := c.connCancel
	done := c.connDone
	c.conn = nil
	c.connCancel = nil
	c.connDone = nil
	c.subscribed = make(map[string]bool)
	c.pendingMutex.Lock()
	for id, ch := range c.pending {
		close(ch)
		delete(c.pending, id)
	}
	c.pendingMutex.Unlock()
	c.connMu.Unlock()

	if cancel != nil {
		cancel()
	}
	if conn != nil {
		_ = conn.Close()
	}
	if done != nil {
		<-done
	}
}

// ============================================================================
// Relay publish / subscribe
// ============================================================================

// writeRelay serializes and sends a relay envelope over the WebSocket. All
// writes are serialized through connMu to keep the gorilla/websocket
// connection's single-writer invariant.
func (c *Client) writeRelay(msg *relayMessage) error {
	c.connMu.Lock()
	conn := c.conn
	c.connMu.Unlock()
	if conn == nil {
		return errors.New("relay not connected")
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal relay message: %w", err)
	}
	c.connMu.Lock()
	defer c.connMu.Unlock()
	if c.conn == nil {
		return errors.New("relay closed during write")
	}
	if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("relay write: %w", err)
	}
	return nil
}

// subscribe tells the relay to deliver publishes on the given topic to this
// connection. Re-subscription is a no-op.
func (c *Client) subscribe(ctx context.Context, topic string) error {
	c.connMu.Lock()
	if c.subscribed[topic] {
		c.connMu.Unlock()
		return nil
	}
	c.connMu.Unlock()

	if err := c.writeRelay(&relayMessage{
		Topic: topic,
		Type:  "sub",
		TTL:   relayDefaultTTL,
	}); err != nil {
		return err
	}
	c.connMu.Lock()
	c.subscribed[topic] = true
	c.connMu.Unlock()
	return nil
}

// unsubscribe stops the relay from delivering publishes on the topic.
func (c *Client) unsubscribe(topic string) error {
	return c.writeRelay(&relayMessage{
		Topic: topic,
		Type:  "unsub",
	})
}

// publish encrypts the JSON-RPC envelope with the topic's symKey and sends a
// "pub" relay message. The peer receives it, decrypts, and responds on the
// same topic.
func (c *Client) publish(ctx context.Context, topic string, envelope interface{}) error {
	c.connMu.Lock()
	key, ok := c.symKeys[topic]
	c.connMu.Unlock()
	if !ok {
		return fmt.Errorf("no symKey for topic %s", topic)
	}

	payload, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("marshal rpc envelope: %w", err)
	}
	ciphertext, err := encryptPayload(key, payload)
	if err != nil {
		return fmt.Errorf("encrypt payload: %w", err)
	}

	return c.writeRelay(&relayMessage{
		Topic:   topic,
		Type:    "pub",
		Message: hex.EncodeToString(ciphertext),
		TTL:     relayDefaultTTL,
	})
}

// readPump reads inbound relay messages for the lifetime of the connection.
// Delivered publishes are decrypted and, if they are JSON-RPC responses
// matching a pending Request id, dispatched to the waiting caller.
func (c *Client) readPump(ctx context.Context) {
	defer func() {
		c.connMu.Lock()
		if c.connDone != nil {
			close(c.connDone)
			c.connDone = nil
		}
		c.conn = nil
		c.connMu.Unlock()
	}()

	c.connMu.Lock()
	conn := c.conn
	c.connMu.Unlock()
	if conn == nil {
		return
	}

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var msg relayMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}
		if msg.Type != "pub" || msg.Message == "" {
			continue
		}

		c.connMu.Lock()
		key, ok := c.symKeys[msg.Topic]
		c.connMu.Unlock()
		if !ok {
			continue
		}

		ciphertext, err := hex.DecodeString(msg.Message)
		if err != nil {
			continue
		}
		plaintext, err := decryptPayload(key, ciphertext)
		if err != nil {
			continue
		}

		var env rpcEnvelope
		if err := json.Unmarshal(plaintext, &env); err != nil {
			continue
		}
		c.dispatch(msg.Topic, &env)
	}
}

// dispatch routes a decoded JSON-RPC envelope. Responses (envelopes with an
// id but no method) are delivered to the pending Request waiter; requests
// and notifications from the peer are forwarded to the session callback if
// one is registered.
func (c *Client) dispatch(topic string, env *rpcEnvelope) {
	// A response carries an id and no method.
	if env.Method == "" && env.ID != 0 {
		c.pendingMutex.Lock()
		ch, ok := c.pending[env.ID]
		if ok {
			delete(c.pending, env.ID)
		}
		c.pendingMutex.Unlock()
		if ok {
			select {
			case ch <- env:
			default:
			}
			return
		}
	}

	c.mu.RLock()
	handler := c.handlers[topic]
	c.mu.RUnlock()
	if handler != nil {
		handler(env)
	}
}

// ============================================================================
// Symmetric encryption (AES-256-GCM)
// ============================================================================
//
// WalletConnect v2 encrypts relay payloads with AES-256-GCM using the symKey
// from the wc: URI as the raw key. The on-the-wire ciphertext layout is:
//
//	iv (12 bytes, random) || ciphertext || tag (16 bytes)
//
// encoded as a hex string in the relay "message" field.

func decodeSymKey(hexKey string) ([]byte, error) {
	// The symKey may optionally be 0x-prefixed.
	if len(hexKey) >= 2 && hexKey[:2] == "0x" {
		hexKey = hexKey[2:]
	}
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, err
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("symKey must be 32 bytes, got %d", len(key))
	}
	return key, nil
}

func encryptPayload(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	iv := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(iv); err != nil {
		return nil, err
	}
	// Seal appends the authenticated ciphertext+tag to iv.
	return gcm.Seal(iv, iv, plaintext, nil), nil
}

func decryptPayload(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}
	iv, ct := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	return gcm.Open(nil, iv, ct, nil)
}

// ============================================================================
// HTTP Handlers
// ============================================================================

// HandleWalletConnectURI handles WalletConnect URI generation
func (c *Client) HandleWalletConnectURI(w ResponseWriter, r Request) {
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
func (c *Client) HandleConnect(w ResponseWriter, r Request) {
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
func (c *Client) HandleRequest(w ResponseWriter, r Request) {
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
func (c *Client) HandleDisconnect(w ResponseWriter, r Request) {
	topic := r.URL.Query().Get("topic")
	
	if err := c.Disconnect(r.Context(), topic); err != nil {
		WriteError(w, r, err.Error())
		return
	}
	
	WriteJSON(w, map[string]string{"status": "disconnected"})
}

// HandleSessions handles session listing
func (c *Client) HandleSessions(w ResponseWriter, r Request) {
	sessions := c.GetSessions()
	WriteJSON(w, sessions)
}

// ============================================================================
// HTTP Server
// ============================================================================

type ResponseWriter = http.ResponseWriter
type Request = *http.Request

func WriteJSON(w ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func WriteError(w ResponseWriter, r Request, message string) {
	_ = r
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
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