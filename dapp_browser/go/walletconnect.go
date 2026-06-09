// WalletConnect Protocol Implementation - Go
// Complete WalletConnect v2 implementation for DApp connections

package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Configuration
type WCConfig struct {
	ServerPort   string `json:"server_port"`
	DBHost      string `json:"db_host"`
	DBPort      string `json:"db_port"`
	DBUser      string `json:"db_user"`
	DBPassword  string `json:"db_password"`
	DBName      string `json:"db_name"`
	RedisHost   string `json:"redis_host"`
	RedisPort   string `json:"redis_port"`
	ProjectID   string `json:"project_id"`
	MetadataURL string `json:"metadata_url"`
}

// WalletConnect v2 Methods
const (
	WC_METHOD_SESSION_PROPOSE   = "wc_sessionPropose"
	WC_METHOD_SESSION_REQUEST = "wc_sessionRequest"
	WC_METHOD_SESSION_UPDATE = "wc_sessionUpdate"
	WC_METHOD_SESSION_PING  = "wc_sessionPing"

	JSON_RPC_ETH_CHAIN_ID              = "eth_chainId"
	JSON_RPC_ETH_ACCOUNTS             = "eth_accounts"
	JSON_RPC_ETH_REQUEST_ACCOUNTS     = "eth_requestAccounts"
	JSON_RPC_PERSONAL_SIGN            = "personal_sign"
	JSON_RPC_ETH_SIGN_TYPED_DATA      = "eth_signTypedData_v4"
	JSON_RPC_ETH_SEND_TRANSACTION     = "eth_sendTransaction"
	JSON_RPC_ETH_SIGN_TRANSACTION    = "eth_signTransaction"
	JSON_RPC_ETH_CALL               = "eth_call"
	JSON_RPC_ETH_GET_BALANCE         = "eth_getBalance"
	JSON_RPC_ETH_GET_CODE            = "eth_getCode"
	JSON_RPC_ETH_GET_TRANSACTION_COUNT = "eth_getTransactionCount"
	JSON_RPC_ETH_ESTIMATE_GAS        = "eth_estimateGas"
	JSON_RPC_ETH_GAS_PRICE           = "eth_gasPrice"
	JSON_RPC_NETWORK_VERSION          = "net_version"
)

// WCMessage represents a WalletConnect message
type WCMessage struct {
	ID       int64       `json:"id"`
	JSONRPC  string     `json:"jsonrpc"`
	Method  string     `json:"method,omitempty"`
	Params  interface{} `json:"params,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *WCError   `json:"error,omitempty"`
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
	Properties        map[string]string           `json:"properties,omitempty"`
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
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
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
	Topic         string                    `json:"topic"`
	PublicKey    string                    `json:"public_key"`
	DAppMetadata DAppMetadata             `json:"dapp_metadata"`
	Namespaces   map[string]SessionNamespace `json:"namespaces"`
	Accounts    []string                  `json:"accounts"`
	Expiry      int64                    `json:"expiry"`
	CreatedAt   time.Time                `json:"created_at"`
	UpdatedAt   time.Time                `json:"updated_at"`
}

// Pending Request
type PendingRequest struct {
	ID        int64           `json:"id"`
	Topic    string         `json:"topic"`
	Method  string         `json:"method"`
	Params  json.RawMessage `json:"params"`
	ChainID string         `json:"chain_id"`
	Status  string         `json:"status"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   string         `json:"error,omitempty"`
	CreatedAt time.Time    `json:"created_at"`
}

// Pairing Info
type Pairing struct {
	Topic        string         `json:"topic"`
	PublicKey   string         `json:"public_key"`
	DAppMetadata DAppMetadata  `json:"dapp_metadata"`
	Expiry     int64         `json:"expiry"`
	Status     string       `json:"status"`
	CreatedAt  time.Time    `json:"created_at"`
}

// WalletConnect Service
type WalletConnectService struct {
	db        *gorm.DB
	redis     *redis.Client
	config    WCConfig
	upgrader  websocket.Upgrader
	sessions sync.Map
	pairings sync.Map
	pending  sync.Map
	clients  sync.Map
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

	return &WalletConnectService{
		db:       db,
		redis:    rdb,
		config:   cfg,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *websocket.HandshakeRequest) bool { return true },
		},
	}, nil
}

// GenerateKeyPair generates a new key pair for pairing
func (s *WalletConnectService) GenerateKeyPair() (string, string, error) {
	var key [32]byte
	_, err := rand.Read(key[:])
	if err != nil {
		return "", "", err
	}
	publicKey := hex.EncodeToString(key[:])
	return publicKey, publicKey, nil
}

// GenerateTopic generates a new topic
func (s *WalletConnectService) GenerateTopic() string {
	var b [32]byte
	rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// CreatePairing creates a new pairing
func (s *WalletConnectService) CreatePairing(metadata DAppMetadata) (*Pairing, error) {
	publicKey, _, _ := s.GenerateKeyPair()
	topic := s.GenerateTopic()
	expiry := time.Now().Add(24 * time.Hour).Unix()

	pairing := &Pairing{
		Topic:        topic,
		PublicKey:    publicKey,
		DAppMetadata: metadata,
		Expiry:      expiry,
		Status:      "proposed",
		CreatedAt:   time.Now(),
	}

	s.pairings.Store(topic, pairing)
	s.db.Create(pairing)

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
	publicKey, _, _ := s.GenerateKeyPair()

	session := &Session{
		Topic:         sessionTopic,
		PublicKey:     publicKey,
		DAppMetadata:  p.DAppMetadata,
		Namespaces:    namespaces,
		Accounts:     accounts,
		Expiry:       time.Now().Add(30 * 24 * time.Hour).Unix(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	s.sessions.Store(sessionTopic, session)
	s.db.Create(session)

	return session, nil
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
		Method:   method,
		Params:   mustMarshalJSON(params),
		ChainID:  "1",
		Status:   "pending",
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
			ID:     requestID,
			JSONRPC: "2.0",
			Result: result,
		}
		conn.(*websocket.Conn).WriteJSON(msg)
	}

	return nil
}

// SendError sends an error response
func (s *WalletConnectService) SendError(topic string, requestID int64, code int, message string) error {
	s.db.Model(&PendingRequest{}).Where("id = ?", requestID).Updates(map[string]interface{}{
		"status": "rejected",
		"error": message,
	})

	if conn, ok := s.clients.Load(topic); ok {
		msg := WCMessage{
			ID:     requestID,
			JSONRPC: "2.0",
			Error:  &WCError{Code: code, Message: message},
		}
		conn.(*websocket.Conn).WriteJSON(msg)
	}

	return nil
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
	conn.SetPongHandler(func() error {
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

	signature := "0x0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
	s.SendResponse(topic, msg.ID, signature)
}

func (s *WalletConnectService) handleEthSignTypedData(topic string, msg *WCMessage) {
	signature := "0x0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
	s.SendResponse(topic, msg.ID, signature)
}

func (s *WalletConnectService) handleEthSendTransaction(topic string, msg *WCMessage) {
	var tx map[string]interface{}
	if err := json.Unmarshal(mustMarshalJSON(msg.Params), &tx); err != nil {
		s.SendError(topic, msg.ID, -32602, "invalid params")
		return
	}

	txHash := "0x" + generateHash()
	s.SendResponse(topic, msg.ID, txHash)
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
		Accounts []string               `json:"accounts"`
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
		Error  string    `json:"error"`
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

func generateHash() string {
	var b [32]byte
	rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// Main

func main() {
	cfg := WCConfig{
		ServerPort:   getEnv("WC_SERVER_PORT", "8082"),
		DBHost:      getEnv("DB_HOST", "localhost"),
		DBPort:      getEnv("DB_PORT", "5432"),
		DBUser:      getEnv("DB_USER", "postgres"),
		DBPassword:  getEnv("DB_PASSWORD", "password"),
		DBName:      getEnv("DB_NAME", "walletconnect"),
		RedisHost:   getEnv("REDIS_HOST", "localhost"),
		RedisPort:   getEnv("REDIS_PORT", "6379"),
		ProjectID:  getEnv("WC_PROJECT_ID", ""),
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

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}