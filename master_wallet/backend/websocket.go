package main

// websocket.go — Real WebSocket hub for the MasterWallet backend. Broadcasts
// live balance updates, transaction confirmations, and (optionally) market
// ticker/orderbook events to connected clients. No loopback fakes.

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
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

// HandleWebSocket upgrades the HTTP connection and registers a client. The
// master_wallet_id query param scopes broadcasts to the relevant wallet.
func (svc *Service) HandleWebSocket(c *gin.Context) {
	masterID := c.Query("master_wallet_id")
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
				p, err := FetchTokenPrice(ctx, coinID)
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
}
