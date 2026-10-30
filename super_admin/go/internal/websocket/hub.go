// WebSocket - Real-time WebSocket support for Admin Panel
package websocket

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/tigerwallet/super-admin/internal/config"
)

// upgrader configures WebSocket connections
// wsAllowedOrigins is the browser-origin allowlist for WebSocket handshakes,
// parsed once from WS_ALLOWED_ORIGINS. Returning true unconditionally would
// allow cross-site WebSocket hijacking against a logged-in super admin.
var wsAllowedOrigins = func() map[string]bool {
	allowed := make(map[string]bool)
	for _, part := range strings.Split(os.Getenv("WS_ALLOWED_ORIGINS"), ",") {
		if o := strings.TrimSpace(part); o != "" {
			allowed[o] = true
		}
	}
	return allowed
}()

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		// Non-browser clients send no Origin and are not CSWSH candidates.
		if origin == "" {
			return true
		}
		return wsAllowedOrigins[origin]
	},
}

// Message types
const (
	MsgTypeAuth         = "auth"
	MsgTypeAuthResponse = "auth_response"
	MsgTypeSubscribe    = "subscribe"
	MsgTypeUnsubscribe  = "unsubscribe"
	MsgTypeEvent        = "event"
	MsgTypePing         = "ping"
	MsgTypePong         = "pong"
	MsgTypeError        = "error"
)

// Event types
const (
	EventUserCreated     = "user.created"
	EventUserUpdated     = "user.updated"
	EventUserDeleted     = "user.deleted"
	EventTransactionNew = "transaction.new"
	EventTransactionUpdate = "transaction.update"
	EventWithdrawalNew  = "withdrawal.new"
	EventWithdrawalUpdate = "withdrawal.update"
	EventKycSubmitted   = "kyc.submitted"
	EventKycApproved    = "kyc.approved"
	EventKycRejected    = "kyc.rejected"
	EventAlert          = "alert"
	EventNotification   = "notification"
)

// WebSocket message
type Message struct {
	Type      string          `json:"type"`
	Channel   string          `json:"channel,omitempty"`
	Event     string          `json:"event,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Timestamp int64           `json:"timestamp"`
}

// Client represents a WebSocket client
type Client struct {
	ID       string
	Conn     *websocket.Conn
	Send     chan []byte
	Hub      *Hub
	Channels map[string]bool
	mu       sync.Mutex
	// JWTSecret is used to re-verify the token sent in an auth frame.
	JWTSecret string
	// AdminID and Role are populated only from a verified token.
	AdminID string
	Role    string
	// authed guards channel subscriptions.
	authed bool
}

// Hub maintains the set of active clients
type Hub struct {
	clients    map[string]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan *BroadcastMessage
	mutex      sync.RWMutex
}

// BroadcastMessage is a message to broadcast to a channel
type BroadcastMessage struct {
	Channel string
	Event   string
	Message []byte
}

// NewHub creates a new Hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *BroadcastMessage, 256),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client.ID] = client
			h.mutex.Unlock()
			log.Printf("Client connected: %s (total: %d)", client.ID, len(h.clients))

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client.ID]; ok {
				close(client.Send)
				delete(h.clients, client.ID)
				log.Printf("Client disconnected: %s (total: %d)", client.ID, len(h.clients))
			}
			h.mutex.Unlock()

		case broadcast := <-h.broadcast:
			h.mutex.RLock()
			for _, client := range h.clients {
				client.mu.Lock()
				if client.Channels[broadcast.Channel] {
					select {
					case client.Send <- broadcast.Message:
					default:
						close(client.Send)
						delete(h.clients, client.ID)
					}
				}
				client.mu.Unlock()
			}
			h.mutex.RUnlock()
		}
	}
}

// Register adds a new client
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister removes a client
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// Broadcast sends a message to all clients subscribed to a channel
func (h *Hub) Broadcast(channel, event string, payload interface{}) {
	msg := Message{
		Type:      MsgTypeEvent,
		Event:     event,
		Channel:   channel,
		Timestamp: time.Now().UnixMilli(),
	}

	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			log.Printf("Error marshaling WebSocket message: %v", err)
			return
		}
		msg.Payload = data
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling WebSocket message: %v", err)
		return
	}

	h.broadcast <- &BroadcastMessage{
		Channel: channel,
		Event:   event,
		Message: data,
	}
}

// GetClientCount returns the number of connected clients
func (h *Hub) GetClientCount() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.clients)
}

// Client methods

// ReadPump reads messages from the WebSocket connection
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister(c)
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(512 * 1024) // 512KB max message size
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		c.handleMessage(message)
	}
}

// WritePump writes messages to the WebSocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current WebSocket message
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) handleMessage(data []byte) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		c.sendError("Invalid message format")
		return
	}

	switch msg.Type {
	case MsgTypeAuth:
		// Re-authenticate with the token carried in the frame. This previously
		// replied "authenticated" to every caller regardless of any credential.
		var authPayload struct {
			Token string `json:"token"`
		}
		_ = json.Unmarshal(msg.Payload, &authPayload)
		claims, ok := wsVerifyToken(c.JWTSecret, strings.TrimSpace(authPayload.Token))
		if !ok {
			c.sendAuthResponse(false, "invalid token")
			return
		}
		c.mu.Lock()
		c.AdminID = claims.AdminID.String()
		c.Role = claims.Role
		c.authed = true
		c.mu.Unlock()
		c.sendAuthResponse(true, "authenticated")

	case MsgTypeSubscribe:
		// Subscriptions require a verified identity.
		if !c.isAuthed() {
			c.sendError("authentication required")
			return
		}
		if msg.Channel != "" {
			c.mu.Lock()
			c.Channels[msg.Channel] = true
			c.mu.Unlock()
			log.Printf("Client %s subscribed to %s", c.ID, msg.Channel)
		}

	case MsgTypeUnsubscribe:
		if msg.Channel != "" {
			c.mu.Lock()
			delete(c.Channels, msg.Channel)
			c.mu.Unlock()
			log.Printf("Client %s unsubscribed from %s", c.ID, msg.Channel)
		}

	case MsgTypePing:
		c.sendPong()
	}
}

func (c *Client) sendAuthResponse(success bool, message string) {
	resp := Message{
		Type:      MsgTypeAuthResponse,
		Timestamp: time.Now().UnixMilli(),
	}

	payload := map[string]interface{}{
		"success": success,
		"message": message,
	}

	data, _ := json.Marshal(payload)
	resp.Payload = data

	msg, _ := json.Marshal(resp)
	c.Send <- msg
}

func (c *Client) sendPong() {
	msg := Message{
		Type:      MsgTypePong,
		Timestamp: time.Now().UnixMilli(),
	}

	data, _ := json.Marshal(msg)
	c.Send <- data
}

func (c *Client) sendError(message string) {
	msg := Message{
		Type:      MsgTypeError,
		Timestamp: time.Now().UnixMilli(),
		Payload:   json.RawMessage(`{"error": "` + message + `"}`),
	}

	data, _ := json.Marshal(msg)
	c.Send <- data
}

// HandleWebSocket handles WebSocket connections. The caller must present a
// valid super-admin JWT (Authorization header, or a `token` query parameter
// because browsers cannot set headers on a WebSocket handshake); otherwise the
// connection is refused before the upgrade.
func HandleWebSocket(hub *Hub, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.TrimSpace(cfg.JWTSecret) == "" {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "websocket authentication is not configured"})
			return
		}
		claims, ok := wsVerifyToken(cfg.JWTSecret, bearerOrQueryToken(c))
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "valid token required"})
			return
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("WebSocket upgrade error: %v", err)
			return
		}

		clientID := c.Query("client_id")
		if clientID == "" {
			clientID = generateClientID()
		}

		client := &Client{
			ID:        clientID,
			Conn:      conn,
			Send:      make(chan []byte, 256),
			Hub:       hub,
			Channels:  make(map[string]bool),
			JWTSecret: cfg.JWTSecret,
			AdminID:   claims.AdminID.String(),
			Role:      claims.Role,
			authed:    true,
		}

		// Subscribe to default channels
		client.Channels["events"] = true
		client.Channels["notifications"] = true

		hub.Register(client)

		go client.WritePump()
		go client.ReadPump()
	}
}

// wsClaims mirrors the super-admin JWT payload. Declared here so the
// transport layer does not depend on the HTTP middleware package.
type wsClaims struct {
	AdminID  uuid.UUID `json:"admin_id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
	jwt.RegisteredClaims
}

// bearerOrQueryToken returns the caller's JWT from the Authorization header,
// falling back to the token query parameter.
func bearerOrQueryToken(c *gin.Context) string {
	if h := strings.TrimSpace(c.GetHeader("Authorization")); h != "" {
		return strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
	}
	return strings.TrimSpace(c.Query("token"))
}

// wsVerifyToken validates an HS256 super-admin token. The signing method is
// pinned so that "none" or an asymmetric algorithm cannot be substituted.
func wsVerifyToken(secret, tokenString string) (*wsClaims, bool) {
	if tokenString == "" {
		return nil, false
	}
	claims := &wsClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, false
	}
	return claims, true
}

// Helper functions

// isAuthed reports whether this connection carries a verified identity.
func (c *Client) isAuthed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.authed
}

func generateClientID() string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		panic("websocket: crypto/rand unavailable: " + err.Error())
	}
	return time.Now().UTC().Format("20060102150405.000000") + "." + hex.EncodeToString(buf[:])
}

// randomString returns n characters drawn from crypto/rand. It previously
// derived every character from time.Now().UnixNano(), which made identifiers
// predictable and prone to collision when called in a tight loop.
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		v, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			panic("websocket: crypto/rand unavailable: " + err.Error())
		}
		b[i] = letters[v.Int64()]
	}
	return string(b)
}

// Broadcast helpers

func BroadcastUserEvent(hub *Hub, event string, user interface{}) {
	hub.Broadcast("users", event, user)
}

func BroadcastTransactionEvent(hub *Hub, event string, tx interface{}) {
	hub.Broadcast("transactions", event, tx)
}

func BroadcastWithdrawalEvent(hub *Hub, event string, withdrawal interface{}) {
	hub.Broadcast("withdrawals", event, withdrawal)
}

func BroadcastKycEvent(hub *Hub, event string, kyc interface{}) {
	hub.Broadcast("kyc", event, kyc)
}

func BroadcastNotification(hub *Hub, notification interface{}) {
	hub.Broadcast("notifications", EventNotification, notification)
}

func BroadcastAlert(hub *Hub, alert interface{}) {
	hub.Broadcast("alerts", EventAlert, alert)
}
