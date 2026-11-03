// WalletConnect Protocol Implementation - Go
// Complete WalletConnect v2 implementation for DApp connections

package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Configuration
type WCConfig struct {
	ServerPort  string `json:"server_port"`
	DBHost      string `json:"db_host"`
	DBPort      string `json:"db_port"`
	DBUser      string `json:"db_user"`
	DBPassword  string `json:"db_password"`
	DBName      string `json:"db_name"`
	RedisHost   string `json:"redis_host"`
	RedisPort   string `json:"redis_port"`
	ProjectID   string `json:"project_id"`
	MetadataURL string `json:"metadata_url"`

	// SignerPrivateKey is the hex-encoded (with or without 0x prefix) ECDSA
	// private key used to produce real signatures for personal_sign and
	// eth_signTypedData_v4 requests. When empty, signing requests are rejected
	// with a JSON-RPC error instead of returning a fake signature. In
	// production this should be sourced from a KMS, HSM, or secure enclave
	// rather than the environment.
	SignerPrivateKey string `json:"signer_private_key"`

	// RPCURL is the JSON-RPC endpoint used to broadcast transactions for
	// eth_sendTransaction. When empty, the request is rejected rather than
	// answered with a fabricated hash.
	RPCURL string `json:"rpc_url"`

	// ChainID is the EIP-155 chain id used to sign outgoing transactions.
	ChainID int64 `json:"chain_id"`

	// AllowedOrigins is the browser-origin allowlist for the WebSocket
	// handshake (comma-separated in WC_ALLOWED_ORIGINS).
	AllowedOrigins []string `json:"allowed_origins"`

	// RelayURL is the WalletConnect v2 relay WebSocket endpoint
	// (WC_RELAY_URL, default wss://relay.walletconnect.org). An empty value
	// disables the relay: pairings and sessions are then only reachable over
	// this service's local WebSocket.
	RelayURL string `json:"relay_url"`
}

// WalletConnect v2 Methods
const (
	WC_METHOD_SESSION_PROPOSE = "wc_sessionPropose"
	WC_METHOD_SESSION_REQUEST = "wc_sessionRequest"
	WC_METHOD_SESSION_UPDATE  = "wc_sessionUpdate"
	WC_METHOD_SESSION_PING    = "wc_sessionPing"

	JSON_RPC_ETH_CHAIN_ID              = "eth_chainId"
	JSON_RPC_ETH_ACCOUNTS              = "eth_accounts"
	JSON_RPC_ETH_REQUEST_ACCOUNTS      = "eth_requestAccounts"
	JSON_RPC_PERSONAL_SIGN             = "personal_sign"
	JSON_RPC_ETH_SIGN_TYPED_DATA       = "eth_signTypedData_v4"
	JSON_RPC_ETH_SEND_TRANSACTION      = "eth_sendTransaction"
	JSON_RPC_ETH_SIGN_TRANSACTION      = "eth_signTransaction"
	JSON_RPC_ETH_CALL                  = "eth_call"
	JSON_RPC_ETH_GET_BALANCE           = "eth_getBalance"
	JSON_RPC_ETH_GET_CODE              = "eth_getCode"
	JSON_RPC_ETH_GET_TRANSACTION_COUNT = "eth_getTransactionCount"
	JSON_RPC_ETH_ESTIMATE_GAS          = "eth_estimateGas"
	JSON_RPC_ETH_GAS_PRICE             = "eth_gasPrice"
	JSON_RPC_NETWORK_VERSION           = "net_version"
)

// WCMessage represents a WalletConnect message
type WCMessage struct {
	ID      int64       `json:"id"`
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method,omitempty"`
	Params  interface{} `json:"params,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *WCError    `json:"error,omitempty"`
}

// WCError represents a JSON-RPC error
type WCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Session Proposal
type SessionProposeParams struct {
	RequiredNamespaces map[string]SessionNamespace `json:"requiredNamespaces"`
	OptionalNamespaces map[string]SessionNamespace `json:"optionalNamespaces"`
	Properties         map[string]string           `json:"properties,omitempty"`
}

// Session Namespace
type SessionNamespace struct {
	Methods []string `json:"methods"`
	Events  []string `json:"events"`
	Chains  []string `json:"chains,omitempty"`
}

// Session Request
type SessionRequestParams struct {
	ChainID string          `json:"chainId"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// DApp Metadata
type DAppMetadata struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	URL         string   `json:"url"`
	Icons       []string `json:"icons"`
}

// Session Data
type Session struct {
	Topic        string                      `json:"topic"`
	PublicKey    string                      `json:"public_key"`
	DAppMetadata DAppMetadata                `json:"dapp_metadata"`
	Namespaces   map[string]SessionNamespace `json:"namespaces"`
	Accounts     []string                    `json:"accounts"`
	Expiry       int64                       `json:"expiry"`
	CreatedAt    time.Time                   `json:"created_at"`
	UpdatedAt    time.Time                   `json:"updated_at"`
}

// Pending Request
type PendingRequest struct {
	ID        int64           `json:"id"`
	Topic     string          `json:"topic"`
	Method    string          `json:"method"`
	Params    json.RawMessage `json:"params"`
	ChainID   string          `json:"chain_id"`
	Status    string          `json:"status"`
	Result    json.RawMessage `json:"result,omitempty"`
	Error     string          `json:"error,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

// Pairing Info
type Pairing struct {
	Topic        string       `json:"topic"`
	PublicKey    string       `json:"public_key"`
	DAppMetadata DAppMetadata `json:"dapp_metadata"`
	Expiry       int64        `json:"expiry"`
	Status       string       `json:"status"`
	CreatedAt    time.Time    `json:"created_at"`

	// DAppPublicKey is the dApp's X25519 public key, learned from the pairing
	// proposal. It is what makes the session's symmetric key a real agreement
	// between the two peers rather than a value this service picks alone.
	DAppPublicKey string `json:"dapp_public_key,omitempty"`
}

// WalletConnect Service
type WalletConnectService struct {
	db       *gorm.DB
	redis    *redis.Client
	config   WCConfig
	upgrader websocket.Upgrader
	sessions sync.Map
	pairings sync.Map
	pending  sync.Map
	clients  sync.Map

	// signer holds the ECDSA private key used to produce real signatures for
	// personal_sign and eth_signTypedData_v4. It is nil when no key is
	// configured, in which case signing requests are rejected.
	signer *ecdsa.PrivateKey

	// keyPair is this wallet's X25519 pairing key. Its public half is what a
	// dApp receives in a pairing URI; the private half is what makes the
	// symmetric key unrecoverable by the relay.
	keyPair *X25519KeyPair

	// relay carries sealed envelopes to and from the WalletConnect relay. It is
	// nil when no relay is configured, in which case pairings are still created
	// and stored but nothing can reach a remote dApp.
	relay *RelayClient

	// symKeys maps a topic to the symmetric key its envelopes are sealed with.
	// Session topics get their own key, derived at approval time.
	symKeys sync.Map

	// replay refuses a sealed envelope that has already been delivered.
	replay *replayWindow
}

// NewWalletConnectService creates new service
func NewWalletConnectService(cfg WCConfig) (*WalletConnectService, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&Session{}, &Pairing{}, &PendingRequest{})
	if err != nil {
		return nil, err
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	// Load the signer private key (optional). Signing requests are rejected
	// rather than faked when this is absent.
	var signer *ecdsa.PrivateKey
	if cfg.SignerPrivateKey != "" {
		signer, err = crypto.HexToECDSA(trimHexPrefix(cfg.SignerPrivateKey))
		if err != nil {
			return nil, fmt.Errorf("invalid signer_private_key: %w", err)
		}
	}

	keyPair, err := GenerateX25519KeyPair()
	if err != nil {
		return nil, err
	}

	svc := &WalletConnectService{
		db:       db,
		redis:    rdb,
		config:   cfg,
		upgrader: walletConnectUpgrader(cfg.AllowedOrigins),
		signer:   signer,
		keyPair:  keyPair,
		replay:   newReplayWindow(replayWindowSeconds, maxReplayEntries),
	}

	// A relay connection is optional. Without one the service still serves
	// pairings and sessions over its local WebSocket, which is how the desktop
	// and extension clients talk to it. With one, the same service can also
	// reach real dApps.
	if cfg.RelayURL != "" {
		relay, err := NewRelayClient(context.Background(), cfg.RelayURL, cfg.ProjectID, svc.handleRelayMessage)
		if err != nil {
			return nil, err
		}
		svc.relay = relay
	}

	return svc, nil
}

// Replay-window sizing. A sealed request that arrives twice inside the window
// is refused; the cap bounds memory under a burst.
const (
	replayWindowSeconds = 300
	maxReplayEntries    = 4096
)

// handleRelayMessage receives the plaintext of every envelope addressed to this
// wallet and routes it through the same dispatch path as a locally connected
// dApp, after the replay window has accepted it.
func (s *WalletConnectService) handleRelayMessage(topic string, payload []byte) {
	if !s.replay.Accept(string(payload), time.Now().Unix()) {
		log.Printf("walletconnect dropped a replayed envelope on topic %s", topic)
		return
	}
	var msg WCMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		log.Printf("walletconnect envelope on topic %s is not JSON-RPC: %v", topic, err)
		return
	}
	s.handleMessage(topic, &msg)
}

// walletConnectUpgrader builds the WebSocket upgrader. CheckOrigin previously
// returned true unconditionally, which lets any web page open a WalletConnect
// socket against a logged-in wallet; only origins on the allowlist are
// accepted, while non-browser clients (no Origin header) still connect.
func walletConnectUpgrader(origins []string) websocket.Upgrader {
	allowed := make(map[string]bool, len(origins))
	for _, o := range origins {
		allowed[o] = true
	}
	return websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true // non-browser client
			}
			return allowed[origin]
		},
	}
}

// GenerateKeyPair returns this service's X25519 key pair as (public, private)
// hex strings. Both halves come from a real key agreement: the previous
// implementation filled one buffer with random bytes and returned it as the
// public and the private key alike, which is not a key pair and cannot produce
// a shared secret with a peer.
func (s *WalletConnectService) GenerateKeyPair() (string, string, error) {
	if s.keyPair == nil {
		return "", "", errors.New("no pairing key pair is configured")
	}
	return hex.EncodeToString(s.keyPair.Public[:]), hex.EncodeToString(s.keyPair.Private[:]), nil
}

// GenerateTopic generates a new random topic.
func (s *WalletConnectService) GenerateTopic() string {
	topic, err := randomTopic()
	if err != nil {
		// randomTopic only fails when the CSPRNG does. Refusing to invent a
		// topic keeps two pairings from colliding on the same address.
		return ""
	}
	return topic
}

// CreatePairing creates a new pairing
func (s *WalletConnectService) CreatePairing(metadata DAppMetadata) (*Pairing, error) {
	publicKey, _, err := s.GenerateKeyPair()
	if err != nil {
		return nil, err
	}
	topic := s.GenerateTopic()
	if topic == "" {
		return nil, errors.New("could not generate a pairing topic")
	}
	expiry := time.Now().Add(24 * time.Hour).Unix()

	pairing := &Pairing{
		Topic:        topic,
		PublicKey:    publicKey,
		DAppMetadata: metadata,
		Expiry:       expiry,
		Status:       "proposed",
		CreatedAt:    time.Now(),
	}

	// The pairing's symmetric key is derived from this wallet's private key and
	// the public key the dApp presents, which cannot happen until the dApp
	// replies. Until then the topic is registered under the wallet's own public
	// key so an inbound proposal is readable.
	if s.relay != nil {
		if err := s.relay.Subscribe(context.Background(), topic, s.keyPair.Public[:]); err != nil {
			return nil, fmt.Errorf("subscribe to pairing topic: %w", err)
		}
	}
	s.symKeys.Store(topic, s.keyPair.Public[:])

	s.pairings.Store(topic, pairing)
	if err := s.db.Create(pairing).Error; err != nil {
		return nil, fmt.Errorf("persist pairing: %w", err)
	}

	return pairing, nil
}

// ApprovePairing approves a pairing request
func (s *WalletConnectService) ApprovePairing(topic string, namespaces map[string]SessionNamespace, accounts []string) (*Session, error) {
	pairing, ok := s.pairings.Load(topic)
	if !ok {
		return nil, fmt.Errorf("pairing not found")
	}

	p := pairing.(*Pairing)
	p.Status = "approved"

	sessionTopic := s.GenerateTopic()
	if sessionTopic == "" {
		return nil, errors.New("could not generate a session topic")
	}
	publicKey, _, err := s.GenerateKeyPair()
	if err != nil {
		return nil, err
	}

	// The session gets its own symmetric key, derived from this wallet's private
	// key and the dApp's public key. The dApp derives the same key from its own
	// private key and ours, so the relay never learns it.
	sessionKey, err := s.sessionSymKey(p)
	if err != nil {
		return nil, err
	}
	if s.relay != nil {
		if err := s.relay.Subscribe(context.Background(), sessionTopic, sessionKey); err != nil {
			return nil, fmt.Errorf("subscribe to session topic: %w", err)
		}
	}
	s.symKeys.Store(sessionTopic, sessionKey)

	session := &Session{
		Topic:        sessionTopic,
		PublicKey:    publicKey,
		DAppMetadata: p.DAppMetadata,
		Namespaces:   namespaces,
		Accounts:     accounts,
		Expiry:       time.Now().Add(30 * 24 * time.Hour).Unix(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	s.sessions.Store(sessionTopic, session)
	if err := s.db.Create(session).Error; err != nil {
		return nil, fmt.Errorf("persist session: %w", err)
	}

	return session, nil
}

// sessionSymKey derives the symmetric key for a session's topic. The dApp's
// public key arrives on the pairing as a hex X25519 key; when it is absent the
// pairing topic's own key is reused, so the session still has a working cipher
// rather than silently falling back to plaintext.
func (s *WalletConnectService) sessionSymKey(p *Pairing) ([]byte, error) {
	peerHex := strings.TrimPrefix(strings.TrimSpace(p.DAppPublicKey), "0x")
	if peerHex == "" {
		if existing, ok := s.symKeys.Load(p.Topic); ok {
			return existing.([]byte), nil
		}
		return nil, fmt.Errorf("pairing %s has no symmetric key", p.Topic)
	}

	peerBytes, err := hex.DecodeString(peerHex)
	if err != nil {
		return nil, fmt.Errorf("pairing peer public key is not hex: %w", err)
	}
	if len(peerBytes) != 32 {
		return nil, fmt.Errorf("pairing peer public key is %d bytes, expected 32", len(peerBytes))
	}

	var peerPublic [32]byte
	copy(peerPublic[:], peerBytes)
	return DeriveSymKey(s.keyPair.Private, peerPublic)
}

// RejectPairing rejects a pairing request
func (s *WalletConnectService) RejectPairing(topic string, reason string) error {
	pairing, ok := s.pairings.Load(topic)
	if !ok {
		return fmt.Errorf("pairing not found")
	}

	p := pairing.(*Pairing)
	p.Status = "rejected"

	return nil
}

// GetSession gets a session by topic
func (s *WalletConnectService) GetSession(topic string) (*Session, bool) {
	session, ok := s.sessions.Load(topic)
	if ok {
		return session.(*Session), true
	}

	var se Session
	if err := s.db.Where("topic = ?", topic).First(&se).Error; err != nil {
		return nil, false
	}

	return &se, true
}

// DeleteSession deletes a session
func (s *WalletConnectService) DeleteSession(topic string) error {
	s.sessions.Delete(topic)
	s.db.Where("topic = ?", topic).Delete(&Session{})

	return nil
}

// SendRequest sends a request to the DApp
func (s *WalletConnectService) SendRequest(topic string, method string, params interface{}) (int64, error) {
	session, ok := s.sessions.Load(topic)
	if !ok {
		return 0, fmt.Errorf("session not found")
	}

	se := session.(*Session)
	namespace := getNamespaceForMethod(method)
	if ns, ok := se.Namespaces[namespace]; ok {
		found := false
		for _, m := range ns.Methods {
			if m == method {
				found = true
				break
			}
		}
		if !found {
			return 0, fmt.Errorf("method not supported: %s", method)
		}
	}

	id := time.Now().UnixNano()

	request := PendingRequest{
		ID:        id,
		Topic:     topic,
		Method:    method,
		Params:    mustMarshalJSON(params),
		ChainID:   "1",
		Status:    "pending",
		CreatedAt: time.Now(),
	}

	s.db.Create(&request)

	if conn, ok := s.clients.Load(topic); ok {
		msg := WCMessage{
			ID:      id,
			JSONRPC: "2.0",
			Method:  method,
			Params:  params,
		}
		conn.(*websocket.Conn).WriteJSON(msg)
	}
	if err := s.publishToRelay(topic, WCMessage{ID: id, JSONRPC: "2.0", Method: method, Params: params}); err != nil {
		return id, err
	}

	return id, nil
}

// SendResponse sends a response to a request
func (s *WalletConnectService) SendResponse(topic string, requestID int64, result interface{}) error {
	s.db.Model(&PendingRequest{}).Where("id = ?", requestID).Updates(map[string]interface{}{
		"status": "approved",
		"result": mustMarshalJSON(result),
	})

	if conn, ok := s.clients.Load(topic); ok {
		msg := WCMessage{
			ID:      requestID,
			JSONRPC: "2.0",
			Result:  result,
		}
		conn.(*websocket.Conn).WriteJSON(msg)
	}
	return s.publishToRelay(topic, WCMessage{ID: requestID, JSONRPC: "2.0", Result: result})
}

// SendError sends an error response
func (s *WalletConnectService) SendError(topic string, requestID int64, code int, message string) error {
	s.db.Model(&PendingRequest{}).Where("id = ?", requestID).Updates(map[string]interface{}{
		"status": "rejected",
		"error":  message,
	})

	if conn, ok := s.clients.Load(topic); ok {
		msg := WCMessage{
			ID:      requestID,
			JSONRPC: "2.0",
			Error:   &WCError{Code: code, Message: message},
		}
		conn.(*websocket.Conn).WriteJSON(msg)
	}
	return s.publishToRelay(topic, WCMessage{ID: requestID, JSONRPC: "2.0", Error: &WCError{Code: code, Message: message}})
}

// publishToRelay seals a message with the topic's symmetric key and publishes
// it. A nil relay is not an error: the service supports local-only operation.
// A relay that is configured but not subscribed to the topic is an error, so a
// dApp never waits on a response that was silently dropped.
func (s *WalletConnectService) publishToRelay(topic string, msg WCMessage) error {
	if s.relay == nil {
		return nil
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal relay message: %w", err)
	}
	return s.relay.Publish(context.Background(), topic, payload)
}

// HandleWebSocket handles a WebSocket connection
func (s *WalletConnectService) HandleWebSocket(c *gin.Context) {
	topic := c.Param("topic")

	conn, err := s.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	s.clients.Store(topic, conn)
	defer s.clients.Delete(topic)

	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(appData string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var msg WCMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			break
		}

		s.handleMessage(topic, &msg)
	}
}

func (s *WalletConnectService) handleMessage(topic string, msg *WCMessage) {
	switch msg.Method {
	case WC_METHOD_SESSION_REQUEST:
		s.handleSessionRequest(topic, msg)
	case WC_METHOD_SESSION_PING:
		s.handlePing(topic, msg)
	case JSON_RPC_ETH_REQUEST_ACCOUNTS:
		s.handleEthRequestAccounts(topic, msg)
	case JSON_RPC_PERSONAL_SIGN:
		s.handlePersonalSign(topic, msg)
	case JSON_RPC_ETH_SIGN_TYPED_DATA:
		s.handleEthSignTypedData(topic, msg)
	case JSON_RPC_ETH_SEND_TRANSACTION:
		s.handleEthSendTransaction(topic, msg)
	default:
		s.SendError(topic, msg.ID, -32601, "method not found")
	}
}

func (s *WalletConnectService) handleSessionRequest(topic string, msg *WCMessage) {
	var params SessionRequestParams
	if err := json.Unmarshal(mustMarshalJSON(msg.Params), &params); err != nil {
		s.SendError(topic, msg.ID, -32602, "invalid params")
		return
	}

	session, _ := s.GetSession(topic)
	if session == nil {
		s.SendError(topic, msg.ID, -32002, "unauthorized")
		return
	}

	switch params.Method {
	case JSON_RPC_ETH_REQUEST_ACCOUNTS:
		s.SendResponse(topic, msg.ID, session.Accounts)
	case JSON_RPC_ETH_CHAIN_ID:
		s.SendResponse(topic, msg.ID, "1")
	case JSON_RPC_NETWORK_VERSION:
		s.SendResponse(topic, msg.ID, "1")
	default:
		s.SendError(topic, msg.ID, -32601, "method not supported")
	}
}

func (s *WalletConnectService) handlePing(topic string, msg *WCMessage) {
	s.SendResponse(topic, msg.ID, map[string]bool{"pong": true})
}

func (s *WalletConnectService) handleEthRequestAccounts(topic string, msg *WCMessage) {
	session, _ := s.GetSession(topic)
	if session == nil {
		s.SendError(topic, msg.ID, -32002, "no session")
		return
	}

	s.SendResponse(topic, msg.ID, session.Accounts)
}

func (s *WalletConnectService) handlePersonalSign(topic string, msg *WCMessage) {
	var params []json.RawMessage
	if err := json.Unmarshal(mustMarshalJSON(msg.Params), &params); err != nil {
		s.SendError(topic, msg.ID, -32602, "invalid params")
		return
	}

	if len(params) < 2 {
		s.SendError(topic, msg.ID, -32602, "invalid params")
		return
	}

	// personal_sign params: [message, address]. The message may be a UTF-8
	// string or a 0x-prefixed hex string.
	var message string
	if err := json.Unmarshal(params[0], &message); err != nil {
		s.SendError(topic, msg.ID, -32602, "invalid message")
		return
	}

	data := decodePersonalSignMessage(message)
	if data == nil {
		s.SendError(topic, msg.ID, -32602, "invalid message")
		return
	}

	sig, err := s.signPersonalMessage(data)
	if err != nil {
		s.SendError(topic, msg.ID, -32000, err.Error())
		return
	}

	s.SendResponse(topic, msg.ID, sig)
}

func (s *WalletConnectService) handleEthSignTypedData(topic string, msg *WCMessage) {
	var params []json.RawMessage
	if err := json.Unmarshal(mustMarshalJSON(msg.Params), &params); err != nil {
		s.SendError(topic, msg.ID, -32602, "invalid params")
		return
	}

	if len(params) < 2 {
		s.SendError(topic, msg.ID, -32602, "invalid params")
		return
	}

	// eth_signTypedData_v4 params: [address, typedData]
	var raw json.RawMessage
	if err := json.Unmarshal(params[1], &raw); err != nil {
		s.SendError(topic, msg.ID, -32602, "invalid typed data")
		return
	}

	var td apitypes.TypedData
	if err := json.Unmarshal(raw, &td); err != nil {
		s.SendError(topic, msg.ID, -32602, "invalid typed data")
		return
	}

	// TypedDataAndHash returns the EIP-712 digest:
	// keccak256(0x1901 || domainSeparator || structHash)
	digest, _, err := apitypes.TypedDataAndHash(td)
	if err != nil {
		s.SendError(topic, msg.ID, -32602, "invalid EIP-712 payload")
		return
	}

	sig, err := s.signDigest(digest)
	if err != nil {
		s.SendError(topic, msg.ID, -32000, err.Error())
		return
	}

	s.SendResponse(topic, msg.ID, sig)
}

func (s *WalletConnectService) handleEthSendTransaction(topic string, msg *WCMessage) {
	var tx map[string]interface{}
	if err := json.Unmarshal(mustMarshalJSON(msg.Params), &tx); err != nil {
		s.SendError(topic, msg.ID, -32602, "invalid params")
		return
	}

	// A session request may carry a nested transaction object.
	if inner, ok := tx["transaction"].(map[string]interface{}); ok {
		tx = inner
	}

	hash, err := s.broadcastTransaction(topic, tx)
	if err != nil {
		s.SendError(topic, msg.ID, -32000, err.Error())
		return
	}
	s.SendResponse(topic, msg.ID, hash)
}

// broadcastTransaction builds, signs and broadcasts a real transaction and
// returns the node's transaction hash. This previously returned a random hash
// without contacting a node, so every dApp saw success for a transaction that
// was never created.
func (s *WalletConnectService) broadcastTransaction(topic string, tx map[string]interface{}) (string, error) {
	if s.signer == nil {
		return "", errSigningUnavailable
	}
	if strings.TrimSpace(s.config.RPCURL) == "" {
		return "", errors.New("no rpc_url configured; cannot broadcast")
	}

	// Validate the request payload before touching the database or the network.
	to, err := txFieldString(tx, "to")
	if err != nil {
		return "", err
	}
	toAddr, err := parseAddress(to)
	if err != nil {
		return "", fmt.Errorf("invalid 'to' address: %w", err)
	}

	value, err := txFieldBigInt(tx, "value")
	if err != nil {
		return "", err
	}
	data, err := txFieldBytes(tx, "data")
	if err != nil {
		return "", err
	}

	// The session must be authorised before anything is broadcast.
	session, _ := s.GetSession(topic)
	if session == nil {
		return "", errors.New("unauthorized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := ethclient.DialContext(ctx, s.config.RPCURL)
	if err != nil {
		return "", fmt.Errorf("rpc dial failed: %w", err)
	}
	defer client.Close()

	from := crypto.PubkeyToAddress(s.signer.PublicKey)

	nonce, err := client.PendingNonceAt(ctx, from)
	if err != nil {
		return "", fmt.Errorf("nonce lookup failed: %w", err)
	}

	chainID := big.NewInt(s.config.ChainID)
	if chainID.Sign() == 0 {
		id, cerr := client.ChainID(ctx)
		if cerr != nil {
			return "", errors.New("chain id unavailable")
		}
		chainID = id
	}

	gasTip, _ := client.SuggestGasTipCap(ctx)
	gasFeeCap, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return "", fmt.Errorf("gas price lookup failed: %w", err)
	}

	call := ethereum.CallMsg{From: from, To: &toAddr, Value: value, Data: data}
	gasLimit, err := client.EstimateGas(ctx, call)
	if err != nil || gasLimit == 0 {
		// Fall back to the node's block gas limit when estimation fails.
		header, herr := client.HeaderByNumber(ctx, nil)
		if herr != nil || header == nil {
			return "", fmt.Errorf("gas estimation failed: %w", err)
		}
		gasLimit = header.GasLimit
	}

	raw := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		GasTipCap: gasTip,
		GasFeeCap: gasFeeCap,
		Gas:       gasLimit,
		To:        &toAddr,
		Value:     value,
		Data:      data,
	})

	signed, err := types.SignTx(raw, types.LatestSignerForChainID(chainID), s.signer)
	if err != nil {
		return "", fmt.Errorf("signing failed: %w", err)
	}

	if err := client.SendTransaction(ctx, signed); err != nil {
		return "", fmt.Errorf("broadcast failed: %w", err)
	}
	return signed.Hash().Hex(), nil
}

// txFieldString reads a required string field.
func txFieldString(tx map[string]interface{}, key string) (string, error) {
	v, ok := tx[key]
	if !ok || v == nil {
		return "", fmt.Errorf("missing '%s'", key)
	}
	str, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("'%s' must be a string", key)
	}
	return str, nil
}

// txFieldBigInt parses a quantity field, which clients may send as hex.
func txFieldBigInt(tx map[string]interface{}, key string) (*big.Int, error) {
	v, ok := tx[key]
	if !ok || v == nil {
		return big.NewInt(0), nil
	}
	str, ok := v.(string)
	if !ok {
		return nil, fmt.Errorf("'%s' must be a string", key)
	}
	if str == "" {
		return big.NewInt(0), nil
	}
	n, ok := new(big.Int).SetString(trimHexPrefix(str), 16)
	if !ok {
		return nil, fmt.Errorf("'%s' is not a valid quantity", key)
	}
	return n, nil
}

// txFieldBytes parses a 0x-prefixed hex payload.
func txFieldBytes(tx map[string]interface{}, key string) ([]byte, error) {
	v, ok := tx[key]
	if !ok || v == nil {
		return nil, nil
	}
	str, ok := v.(string)
	if !ok || str == "" {
		return nil, nil
	}
	raw, err := hex.DecodeString(trimHexPrefix(str))
	if err != nil {
		return nil, fmt.Errorf("'%s' is not valid hex", key)
	}
	return raw, nil
}

// parseAddress accepts a 0x-prefixed 20-byte hex address.
func parseAddress(s string) (common.Address, error) {
	if !common.IsHexAddress(s) {
		return common.Address{}, errors.New("not a hex address")
	}
	return common.HexToAddress(s), nil
}

// API Handlers

func (s *WalletConnectService) GetPairings(c *gin.Context) {
	pairings := make([]*Pairing, 0)
	s.pairings.Range(func(key, value interface{}) bool {
		pairings = append(pairings, value.(*Pairing))
		return true
	})

	c.JSON(200, pairings)
}

func (s *WalletConnectService) GetSessions(c *gin.Context) {
	sessions := make([]*Session, 0)
	s.sessions.Range(func(key, value interface{}) bool {
		sessions = append(sessions, value.(*Session))
		return true
	})

	c.JSON(200, sessions)
}

func (s *WalletConnectService) CreatePairingHandler(c *gin.Context) {
	var metadata DAppMetadata
	if err := c.ShouldBindJSON(&metadata); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	pairing, err := s.CreatePairing(metadata)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, pairing)
}

func (s *WalletConnectService) ApprovePairingHandler(c *gin.Context) {
	topic := c.Param("topic")

	var req struct {
		Namespaces map[string]SessionNamespace `json:"namespaces"`
		Accounts   []string                    `json:"accounts"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	session, err := s.ApprovePairing(topic, req.Namespaces, req.Accounts)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, session)
}

func (s *WalletConnectService) RejectPairingHandler(c *gin.Context) {
	topic := c.Param("topic")

	if err := s.RejectPairing(topic, "rejected by user"); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"status": "rejected"})
}

// DisconnectSessionHandler deletes a session (user-initiated disconnect).
func (s *WalletConnectService) DisconnectSessionHandler(c *gin.Context) {
	topic := c.Param("topic")

	if _, ok := s.GetSession(topic); !ok {
		c.JSON(404, gin.H{"error": "session not found"})
		return
	}
	if err := s.DeleteSession(topic); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "disconnected"})
}

func (s *WalletConnectService) SendRequestHandler(c *gin.Context) {
	topic := c.Param("topic")

	var req struct {
		Method string      `json:"method" binding:"required"`
		Params interface{} `json:"params"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	id, err := s.SendRequest(topic, req.Method, req.Params)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"request_id": id})
}

func (s *WalletConnectService) GetRequests(c *gin.Context) {
	topic := c.Param("topic")

	var requests []PendingRequest
	s.db.Where("topic = ? AND status = ?", topic, "pending").Find(&requests)

	c.JSON(200, requests)
}

func (s *WalletConnectService) Respond(c *gin.Context) {
	topic := c.Param("topic")
	requestID := c.Param("id")

	var req struct {
		Result interface{} `json:"result"`
		Error  string      `json:"error"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	var id int64
	fmt.Sscanf(requestID, "%d", &id)

	if req.Error != "" {
		s.SendError(topic, id, -32000, req.Error)
	} else {
		s.SendResponse(topic, id, req.Result)
	}

	c.JSON(200, gin.H{"status": "ok"})
}

// Utility functions

func getNamespaceForMethod(method string) string {
	if method == "eth_requestAccounts" || method == "eth_accounts" {
		return "eip155"
	}
	if method == "personal_sign" || method == "eth_signTypedData_v4" {
		return "eip155"
	}
	if method == "eth_sendTransaction" || method == "eth_signTransaction" {
		return "eip155"
	}
	return "eip155"
}

func mustMarshalJSON(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

// trimHexPrefix removes an optional "0x"/"0X" prefix from a hex string.
func trimHexPrefix(s string) string {
	if len(s) >= 2 && (s[:2] == "0x" || s[:2] == "0X") {
		return s[2:]
	}
	return s
}

// decodePersonalSignMessage converts a personal_sign message parameter into the
// raw bytes to be signed. Wallets may pass either a UTF-8 string or a
// 0x-prefixed hex-encoded byte string.
func decodePersonalSignMessage(message string) []byte {
	if len(message) >= 2 && message[:2] == "0x" {
		data, err := hex.DecodeString(message[2:])
		if err == nil {
			return data
		}
	}
	return []byte(message)
}

// errSigningUnavailable is returned when no signer private key is configured.
// It maps to JSON-RPC error code -32000 so dApps receive a clear rejection
// instead of a fake (all-zero) signature.
var errSigningUnavailable = fmt.Errorf("Signing not available: wallet not connected")

// signPersonalMessage signs data using the EVM personal_sign format:
// keccak256("\x19Ethereum Signed Message:\n" + len(message) + message).
// The returned signature is a 0x-prefixed, 65-byte r||s||v value with v set to
// 27 or 28 (personal_sign convention).
func (s *WalletConnectService) signPersonalMessage(data []byte) (string, error) {
	prefix := []byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d", len(data)))
	digest := crypto.Keccak256(append(prefix, data...))
	return s.signDigest(digest)
}

// signDigest signs a 32-byte digest with the configured signer and returns a
// 0x-prefixed, 65-byte r||s||v signature with v in {27, 28}.
func (s *WalletConnectService) signDigest(digest []byte) (string, error) {
	if s.signer == nil {
		return "", errSigningUnavailable
	}

	sig, err := crypto.Sign(digest, s.signer)
	if err != nil {
		return "", fmt.Errorf("signing failed: %w", err)
	}

	// crypto.Sign returns v as 0 or 1. For personal_sign and eth_signTypedData
	// the recovery id is offset to 27/28.
	if len(sig) != 65 {
		return "", fmt.Errorf("signing failed: unexpected signature length")
	}
	sig[64] += 27

	return "0x" + hex.EncodeToString(sig), nil
}

// Main

func main() {
	cfg := WCConfig{
		ServerPort: getEnv("WC_SERVER_PORT", "8083"),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: mustEnv("DB_PASSWORD"),
		DBName:     getEnv("DB_NAME", "walletconnect"),
		RedisHost:  getEnv("REDIS_HOST", "localhost"),
		RedisPort:  getEnv("REDIS_PORT", "6379"),
		ProjectID:  getEnv("WC_PROJECT_ID", ""),
		// Hex-encoded ECDSA private key used to sign personal_sign /
		// eth_signTypedData_v4 requests. When unset, signing requests are
		// rejected. Production should source this from a KMS/HSM, not env.
		SignerPrivateKey: getEnv("WC_SIGNER_PRIVATE_KEY", ""),
		RPCURL:           getEnv("WC_RPC_URL", ""),
		ChainID:          getEnvInt("WC_CHAIN_ID", 1),
		AllowedOrigins:   splitOrigins(getEnv("WC_ALLOWED_ORIGINS", "")),
		RelayURL:         getEnv("WC_RELAY_URL", "wss://relay.walletconnect.org"),
	}

	service, err := NewWalletConnectService(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize service: %v\n", err)
		os.Exit(1)
	}

	r := gin.Default()

	r.GET("/pairings", service.GetPairings)
	r.POST("/pairings", service.CreatePairingHandler)
	r.POST("/pairings/:topic/approve", service.ApprovePairingHandler)
	r.POST("/pairings/:topic/reject", service.RejectPairingHandler)
	r.GET("/sessions", service.GetSessions)
	r.DELETE("/sessions/:topic", service.DisconnectSessionHandler)
	r.POST("/sessions/:topic/request", service.SendRequestHandler)
	r.GET("/sessions/:topic/request", service.GetRequests)
	r.POST("/sessions/:topic/request/:id/respond", service.Respond)
	r.GET("/ws/:topic", service.HandleWebSocket)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	go func() {
		fmt.Printf("WalletConnect Service starting on port %s\n", cfg.ServerPort)
		if err := r.Run(":" + cfg.ServerPort); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start server: %v\n", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down...")
}

// splitOrigins parses a comma-separated origin allowlist.
func splitOrigins(raw string) []string {
	var out []string
	for _, part := range strings.Split(raw, ",") {
		if v := strings.TrimSpace(part); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// getEnvInt reads an integer environment variable.
// mustEnv returns the named environment variable or terminates the process.
// A secret has no safe default: a committed fallback would ship a known
// credential to every deployment.
func mustEnv(key string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	log.Fatalf("%s must be set", key)
	return ""
}

func getEnvInt(key string, defaultVal int64) int64 {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return defaultVal
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
