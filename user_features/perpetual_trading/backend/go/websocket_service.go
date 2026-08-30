package main

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WSHub maintains the set of active clients
type WSHub struct {
	clients    map[*WSClient]bool
	broadcast  chan []byte
	register   chan *WSClient
	unregister chan *WSClient
	mu         sync.RWMutex
}

// NewWSHub creates a new WebSocket hub
func NewWSHub() *WSHub {
	return &WSHub{
		clients:    make(map[*WSClient]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *WSClient),
		unregister: make(chan *WSClient),
	}
}

// Run runs the hub
func (h *WSHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Register registers a client
func (h *WSHub) Register(client *WSClient) {
	h.register <- client
}

// Unregister unregisters a client
func (h *WSHub) Unregister(client *WSClient) {
	h.unregister <- client
}

// Broadcast broadcasts a message to all clients
func (h *WSHub) Broadcast(msg interface{}) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	h.broadcast <- data
}

// WSClient represents a WebSocket client
type WSClient struct {
	hub  *WSHub
	conn *websocket.Conn
	send chan []byte
}

// NewWSClient creates a new WebSocket client
func NewWSClient(conn *websocket.Conn, hub *WSHub) *WSClient {
	return &WSClient{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, 256),
	}
}

// ReadPump reads messages from the WebSocket connection
func (c *WSClient) ReadPump() {
	defer func() {
		c.hub.Unregister(c)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		c.handleMessage(msg)
	}
}

// WritePump writes messages to the WebSocket connection
func (c *WSClient) WritePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Write pending messages
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *WSClient) handleMessage(msg map[string]interface{}) {
	msgType, ok := msg["type"].(string)
	if !ok {
		return
	}

	switch msgType {
	case "subscribe":
		if channels, ok := msg["channels"].([]interface{}); ok {
			for _, ch := range channels {
				if channel, ok := ch.(string); ok {
					c.subscribe(channel)
				}
			}
		}
	case "unsubscribe":
		if channels, ok := msg["channels"].([]interface{}); ok {
			for _, ch := range channels {
				if channel, ok := ch.(string); ok {
					c.unsubscribe(channel)
				}
			}
		}
	case "ping":
		c.send <- []byte(`{"type":"pong"}`)
	}
}

func (c *WSClient) subscribe(channel string) {
	log.Printf("Client subscribed to %s", channel)
}

func (c *WSClient) unsubscribe(channel string) {
	log.Printf("Client unsubscribed from %s", channel)
}

// WebSocketMessage represents a WebSocket message
type WebSocketMessage struct {
	Type    string          `json:"type"`
	Channel string          `json:"channel,omitempty"`
	Data    json.RawMessage `json:"data"`
}

// SubscriptionRequest represents a subscription request
type SubscriptionRequest struct {
	Type     string   `json:"type"`
	Channels []string `json:"channels"`
}

// SubscriptionResponse represents a subscription response
type SubscriptionResponse struct {
	Type     string   `json:"type"`
	Channels []string `json:"channels"`
	Status   string   `json:"status"`
}
