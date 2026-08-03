// WebSocket Hub for Real-Time Communication
// Production-ready WebSocket server with room management and event broadcasting

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

// ============================================================================
// TYPES
// ============================================================================

type Client struct {
	ID        string
	Hub       *WebSocketHub
	Conn      *websocket.Conn
	Send      chan []byte
	Rooms     map[string]bool
	UserID    string
	AdminID   string
	IsAdmin   bool
	Metadata  map[string]interface{}
	mu       sync.RWMutex
}

type WebSocketHub struct {
	Clients    map[string]*Client
	Rooms      map[string]map[string]*Client
	Broadcast  chan *Message
	Register   chan *Client
	Unregister chan *Client
	mu         sync.RWMutex
}

type Message struct {
	Type      string          `json:"type"`
	Event     string          `json:"event"`
	Payload   json.RawMessage `json:"payload"`
	Room      string          `json:"room,omitempty"`
	TargetID  string          `json:"targetId,omitempty"`
	Timestamp int64           `json:"timestamp"`
}

type RoomEvent struct {
	Type    string      `json:"type"`
	Room    string      `json:"room"`
	Payload interface{} `json:"payload"`
}

// ============================================================================
// HUB MANAGEMENT
// ============================================================================

func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		Clients:    make(map[string]*Client),
		Rooms:      make(map[string]map[string]*Client),
		Broadcast:  make(chan *Message, 256),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
	}
}

func (h *WebSocketHub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.Clients[client.ID] = client
			h.mu.Unlock()
			log.Printf("Client connected: %s", client.ID)

		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.Clients[client.ID]; ok {
				close(client.Send)
				delete(h.Clients, client.ID)
				
				// Remove from all rooms
				for room := range client.Rooms {
					h.LeaveRoom(client, room)
				}
			}
			h.mu.Unlock()
			log.Printf("Client disconnected: %s", client.ID)

		case message := <-h.Broadcast:
			h.mu.RLock()
			for _, client := range h.Clients {
				select {
				case client.Send <- message.toBytes():
				default:
					// Client buffer full, skip
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *WebSocketHub) GetClient(id string) *Client {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.Clients[id]
}

func (h *WebSocketHub) GetRoomClients(room string) []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	clients := make([]*Client, 0)
	if roomClients, ok := h.Rooms[room]; ok {
		for _, client := range roomClients {
			clients = append(clients, client)
		}
	}
	return clients
}

func (h *WebSocketHub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.Clients)
}

// ============================================================================
// ROOM MANAGEMENT
// ============================================================================

func (h *WebSocketHub) JoinRoom(client *Client, room string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.Rooms[room] == nil {
		h.Rooms[room] = make(map[string]*Client)
	}

	h.Rooms[room][client.ID] = client
	client.Rooms[room] = true

	log.Printf("Client %s joined room %s", client.ID, room)
}

func (h *WebSocketHub) LeaveRoom(client *Client, room string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.Rooms[room] != nil {
		delete(h.Rooms[room], client.ID)
		delete(client.Rooms, room)
		
		if len(h.Rooms[room]) == 0 {
			delete(h.Rooms, room)
		}
	}

	log.Printf("Client %s left room %s", client.ID, room)
}

func (h *WebSocketHub) BroadcastToRoom(room string, message *Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	messageBytes := message.toBytes()
	
	if roomClients, ok := h.Rooms[room]; ok {
		for _, client := range roomClients {
			select {
			case client.Send <- messageBytes:
			default:
			}
		}
	}
}

func (h *WebSocketHub) SendToUser(userID string, message *Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	messageBytes := message.toBytes()
	
	for _, client := range h.Clients {
		if client.UserID == userID {
			select {
			case client.Send <- messageBytes:
			default:
			}
		}
	}
}

// ============================================================================
// CLIENT METHODS
// ============================================================================

func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
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

		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Error unmarshaling message: %v", err)
			continue
		}

		msg.Timestamp = time.Now().Unix()
		c.handleMessage(&msg)
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(30 * time.Second)
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

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
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

func (c *Client) handleMessage(msg *Message) {
	switch msg.Type {
	case "join_room":
		var payload struct {
			Room string `json:"room"`
		}
		json.Unmarshal(msg.Payload, &payload)
		if payload.Room != "" {
			c.Hub.JoinRoom(c, payload.Room)
		}

	case "leave_room":
		var payload struct {
			Room string `json:"room"`
		}
		json.Unmarshal(msg.Payload, &payload)
		if payload.Room != "" {
			c.Hub.LeaveRoom(c, payload.Room)
		}

	case "broadcast":
		var payload struct {
			Message string `json:"message"`
			Room   string `json:"room,omitempty"`
		}
		json.Unmarshal(msg.Payload, &payload)
		if payload.Room != "" {
			broadcastMsg := &Message{
				Type:      "broadcast",
				Event:    msg.Event,
				Payload:  msg.Payload,
				Room:     payload.Room,
				Timestamp: time.Now().Unix(),
			}
			c.Hub.BroadcastToRoom(payload.Room, broadcastMsg)
		}

	case "ping":
		c.Send <- []byte(`{"type":"pong","timestamp":` + fmt.Sprintf("%d", time.Now().Unix()) + `}`)

	default:
		log.Printf("Unknown message type: %s", msg.Type)
	}
}

// ============================================================================
// MESSAGE HELPERS
// ============================================================================

func (m *Message) toBytes() []byte {
	data, _ := json.Marshal(m)
	return data
}

// ============================================================================
// HTTP HANDLERS
// ============================================================================

type WebSocketServer struct {
	Hub *WebSocketHub
}

func NewWebSocketServer() *WebSocketServer {
	hub := NewWebSocketHub()
	go hub.Run()
	
	return &WebSocketServer{Hub: hub}
}

func (s *WebSocketServer) HandleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	clientID := c.Query("client_id")
	if clientID == "" {
		clientID = fmt.Sprintf("client_%d", time.Now().UnixNano())
	}

	client := &Client{
		ID:      clientID,
		Hub:     s.Hub,
		Conn:    conn,
		Send:    make(chan []byte, 256),
		Rooms:   make(map[string]bool),
		Metadata: make(map[string]interface{}),
	}

	s.Hub.Register <- client

	go client.WritePump()
	client.ReadPump()
}

func (s *WebSocketServer) HandleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":       "healthy",
		"clients":      s.Hub.GetClientCount(),
		"rooms":        len(s.Hub.Rooms),
		"timestamp":    time.Now().Unix(),
	})
}

// ============================================================================
// EVENT BROADCAST HELPERS
// ============================================================================

// BroadcastUserUpdate broadcasts user-related updates
func (h *WebSocketHub) BroadcastUserUpdate(userID string, event string, data interface{}) {
	payload, _ := json.Marshal(data)
	msg := &Message{
		Type:      "user_update",
		Event:    event,
		Payload:   payload,
		TargetID: userID,
		Timestamp: time.Now().Unix(),
	}
	h.Broadcast <- msg
}

// BroadcastTradeUpdate broadcasts trade/exchange updates
func (h *WebSocketHub) BroadcastTradeUpdate(room string, event string, data interface{}) {
	payload, _ := json.Marshal(data)
	msg := &Message{
		Type:      "trade_update",
		Event:    event,
		Payload:  payload,
		Room:     room,
		Timestamp: time.Now().Unix(),
	}
	h.BroadcastToRoom(room, msg)
}

// BroadcastAdminBroadcast broadcasts admin messages
func (h *WebSocketHub) BroadcastAdminBroadcast(event string, data interface{}) {
	payload, _ := json.Marshal(data)
	msg := &Message{
		Type:      "admin_broadcast",
		Event:    event,
		Payload:  payload,
		Timestamp: time.Now().Unix(),
	}
	h.Broadcast <- msg
}

// BroadcastPriceUpdate broadcasts price updates to all subscribers
func (h *WebSocketHub) BroadcastPriceUpdate(symbol string, price float64, change24h float64) {
	data := map[string]interface{}{
		"symbol":    symbol,
		"price":     price,
		"change24h": change24h,
	}
	h.BroadcastTradeUpdate("prices", "price_update", data)
}

// BroadcastNewTransaction broadcasts new transaction
func (h *WebSocketHub) BroadcastNewTransaction(txHash string, from string, to string, amount string) {
	data := map[string]interface{}{
		"txHash": txHash,
		"from":   from,
		"to":     to,
		"amount": amount,
	}
	h.BroadcastTradeUpdate("transactions", "new_transaction", data)
}

// BroadcastKYCUpdate broadcasts KYC status changes
func (h *WebSocketHub) BroadcastKYCUpdate(userID string, status string) {
	data := map[string]interface{}{
		"userID": userID,
		"status": status,
	}
	h.BroadcastUserUpdate(userID, "kyc_update", data)
}

// BroadcastWithdrawalUpdate broadcasts withdrawal status
func (h *WebSocketHub) BroadcastWithdrawalUpdate(userID string, txHash string, status string) {
	data := map[string]interface{}{
		"userID": userID,
		"txHash": txHash,
		"status": status,
	}
	h.BroadcastUserUpdate(userID, "withdrawal_update", data)
}

// ============================================================================
// WEBSOCKET MIDDLEWARE FOR GIN
// ============================================================================

func WebSocketMiddleware(hub *WebSocketHub) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("ws_hub", hub)
		c.Next()
	}
}

// ============================================================================
// SSE (Server-Sent Events) SUPPORT
// ============================================================================

type SSEManager struct {
	clients map[string]chan []byte
	mu      sync.RWMutex
}

func NewSSEManager() *SSEManager {
	return &SSEManager{
		clients: make(map[string]chan []byte),
	}
}

func (m *SSEManager) Subscribe(id string) chan []byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	ch := make(chan []byte, 10)
	m.clients[id] = ch
	return ch
}

func (m *SSEManager) Unsubscribe(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if ch, ok := m.clients[id]; ok {
		close(ch)
		delete(m.clients, id)
	}
}

func (m *SSEManager) Publish(event string, data interface{}) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	payload, _ := json.Marshal(map[string]interface{}{
		"event": event,
		"data":  data,
	})
	
	for _, ch := range m.clients {
		select {
		case ch <- payload:
		default:
		}
	}
}

func (m *SSEManager) HandleSSE(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		id = fmt.Sprintf("sse_%d", time.Now().UnixNano())
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	notify := c.Request.Context().Done()

	ch := m.Subscribe(id)
	defer m.Unsubscribe(id)

	for {
		select {
		case <-notify:
			return
		case data := <-ch:
			if len(data) > 0 {
				c.SSEvent("message", data)
				c.Writer.Flush()
			}
		}
	}
}
