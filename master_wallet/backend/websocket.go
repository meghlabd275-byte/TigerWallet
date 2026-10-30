package main

// websocket.go — Real WebSocket hub for the MasterWallet backend. Broadcasts
// live balance updates, transaction confirmations, and (optionally) market
// ticker/orderbook events to connected clients. No loopback fakes.

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// wsAllowedOrigins is the browser-origin allowlist for WebSocket handshakes,
// parsed once from MASTER_WALLET_CORS_ORIGINS. Empty means no browser origin
// is accepted; returning true unconditionally would allow cross-site
// WebSocket hijacking against an authenticated operator session.
var wsAllowedOrigins = func() map[string]bool {
	allowed := make(map[string]bool)
	for _, o := range splitAndTrim(os.Getenv("MASTER_WALLET_CORS_ORIGINS")) {
		allowed[o] = true
	}
	return allowed
}()

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		// Non-browser clients send no Origin and are not CSWSH candidates.
		if origin == "" {
			return true
		}
		return wsAllowedOrigins[origin]
	},
}

// wsClient is one connected WebSocket client.
type wsClient struct {
	conn       *websocket.Conn
	masterID   string
	send       chan []byte
}

// wsHub manages connected clients + broadcasts.
type wsHub struct {
	mu      sync.RWMutex
	clients map[*wsClient]bool
}

func newWSHub() *wsHub {
	return &wsHub{clients: make(map[*wsClient]bool)}
}

func (h *wsHub) register(c *wsClient) {
	h.mu.Lock()
	h.clients[c] = true
	h.mu.Unlock()
}

func (h *wsHub) unregister(c *wsClient) {
	h.mu.Lock()
	if _, ok := h.clients[c]; ok {
		delete(h.clients, c)
		close(c.send)
	}
	h.mu.Unlock()
}

// broadcast sends a message to all clients (optionally filtered by masterID).
func (h *wsHub) broadcast(masterID string, msg interface{}) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for client := range h.clients {
		if masterID != "" && client.masterID != masterID {
			continue
		}
		select {
		case client.send <- data:
		default:
			// drop slow clients
		}
	}
}

// readPump reads messages from a client (handles ping/pong + close).
func (c *wsClient) readPump(hub *wsHub) {
	defer func() {
		hub.unregister(c)
		c.conn.Close()
	}()
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// writePump writes messages from the send channel to the client.
func (c *wsClient) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
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

// HandleWebSocket upgrades the HTTP connection and registers a client.
//
// The caller must present a valid JWT (Authorization header, or the `token`
// query parameter because browsers cannot set headers on a WebSocket
// handshake), and the requested master_wallet_id must belong to that user
// unless they are an admin. Without both checks any anonymous client could
// subscribe to another operator's balance and transaction stream.
func (svc *Service) HandleWebSocket(c *gin.Context) {
	claims, ok := wsAuthenticate(c, svc.cfg.JWTSecret)
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "valid token required"})
		return
	}

	masterID := c.Query("master_wallet_id")
	if masterID != "" {
		if !svc.wsMayAccessWallet(c.Request.Context(), masterID, claims.UserID, claims.Role) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "wallet not accessible to this user"})
			return
		}
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("ws upgrade failed: %v", err)
		return
	}
	client := &wsClient{conn: conn, masterID: masterID, send: make(chan []byte, 64)}
	svc.hub.register(client)
	go client.writePump()
	go client.readPump(svc.hub)
	// Send an immediate connection-acknowledgement with the wallet's live balance.
	if masterID != "" {
		go svc.pushInitialBalance(client, masterID)
	}
}

// wsAuthenticate validates a JWT from the Authorization header or the token
// query parameter.
func wsAuthenticate(c *gin.Context, secret string) (*MasterClaims, bool) {
	if strings.TrimSpace(secret) == "" {
		return nil, false
	}
	tokenStr := strings.TrimPrefix(strings.TrimSpace(c.GetHeader("Authorization")), "Bearer ")
	if tokenStr == "" {
		tokenStr = strings.TrimSpace(c.Query("token"))
	}
	if tokenStr == "" {
		return nil, false
	}
	claims, err := ParseJWT(secret, tokenStr)
	if err != nil {
		return nil, false
	}
	return claims, true
}

// wsMayAccessWallet reports whether the caller owns the wallet, or holds an
// operator role. A wallet with no recorded owner is reachable only by admins.
func (svc *Service) wsMayAccessWallet(ctx context.Context, masterID, userID, role string) bool {
	if role == "admin" || role == "super_admin" {
		return true
	}
	var createdBy string
	if err := svc.store.db.QueryRow(ctx,
		`SELECT COALESCE(created_by::text, '') FROM master_wallets WHERE id = $1`, masterID).Scan(&createdBy); err != nil {
		return false
	}
	return createdBy != "" && createdBy == userID
}

// pushInitialBalance sends the live balance once on connect.
func (svc *Service) pushInitialBalance(client *wsClient, masterID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	var address string
	var chainID int64
	err := svc.store.db.QueryRow(ctx,
		`SELECT address, chain_id FROM master_wallets WHERE id = $1`, masterID).
		Scan(&address, &chainID)
	if err != nil {
		return
	}
	rpc := rpcEndpointForChain(chainID)
	if rpc == "" {
		return
	}
	bal, err := FetchNativeBalance(ctx, rpc, common.HexToAddress(address))
	if err != nil {
		return
	}
	chain, _ := chainByID(chainID)
	msg := gin.H{
		"type": "balance", "master_wallet_id": masterID, "address": address,
		"chain_id": chainID, "balance": weiToFloat(bal, chain.Decimals), "symbol": chain.Symbol, "timestamp": time.Now().UTC(),
	}
	data, _ := json.Marshal(msg)
	select {
	case client.send <- data:
	default:
	}
}

// startMarketTicker broadcasts live CoinGecko prices for tracked chains every
// 60 seconds to all connected clients (ticker events).
func (svc *Service) startMarketTicker(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, chain := range supportedChains {
				coinID := chainCoinGeckoID(chain.ChainID)
				if coinID == "" {
					continue
				}
				p, err := svc.FetchTokenPriceCached(ctx, coinID)
				if err != nil || p == nil {
					continue
				}
				svc.hub.broadcast("", gin.H{
					"type": "ticker", "symbol": chain.Symbol, "chain_id": chain.ChainID,
					"price_usd": p.USD, "change_24h": p.USD24h, "market_cap": p.MarketCap, "timestamp": time.Now().UTC(),
				})
			}
		}
	}
}

// notifyEvent pushes a wallet-scoped event to connected clients (used by other
// handlers when a transaction confirms, an approval is requested, etc.).
func (svc *Service) notifyEvent(masterID, eventType string, payload gin.H) {
	payload["type"] = eventType
	payload["master_wallet_id"] = masterID
	payload["timestamp"] = time.Now().UTC()
	svc.hub.broadcast(masterID, payload)
	// Cluster fanout: deliver to clients connected to OTHER replicas (no-op
	// without Redis; receivers skip this replica's own message via Origin).
	svc.publishWSEvent(masterID, payload)
}
