package main

// live_feed.go — public WebSocket live price feed.
//
// Every UserWallet client (web/android/ios/desktop/extension/rust) previously
// had to poll /api/v1/terminal/ticker/:symbol for price updates. This hub
// provides a single push channel instead:
//
//      GET /api/v1/ws            (WebSocket upgrade; public, read-only)
//      -> {"action":"subscribe","symbols":["BTC","ETH"]}
//      -> {"action":"unsubscribe","symbols":["BTC"]}
//      <- {"type":"subscribed","symbols":["BTC","ETH"]}
//      <- {"type":"ticker","symbol":"BTC","coin":"bitcoin","last_price":...,
//          "change_24h_pct":...,"volume_24h":...,"market_cap":...,"updated":...}
//      <- {"type":"error","message":"price feed unavailable"}
//
// The hub batches the union of all subscribed symbols into ONE upstream
// CoinGecko markets call per tick (LIVE_FEED_INTERVAL_MS, default 5000) and
// fans out to subscribers, so N clients watching BTC cost 1 upstream request
// per tick total. The feed is fail-closed: when the upstream price provider
// is down an error frame is sent and no price is ever fabricated.

import (
        "context"
        "encoding/json"
        "fmt"
        "io"
        "net/http"
        "os"
        "strconv"
        "strings"
        "sync"
        "time"

        "github.com/gin-gonic/gin"
        "github.com/gorilla/websocket"
)

type liveFeedHub struct {
        mu       sync.Mutex
        clients  map[*liveFeedClient]struct{}
        // symbol (upper) -> set of clients subscribed
        subs     map[string]map[*liveFeedClient]struct{}
        // latest ticker per symbol, shared with late subscribers
        latest   map[string]gin.H
        stopOnce sync.Once
        done     chan struct{}
}

type liveFeedClient struct {
        conn *websocket.Conn
        send chan []byte
}

type liveFeedRequest struct {
        Action  string   `json:"action"`
        Symbols []string `json:"symbols"`
}

var liveFeed = newLiveFeedHub()

func newLiveFeedHub() *liveFeedHub {
        return &liveFeedHub{
                clients: map[*liveFeedClient]struct{}{},
                subs:    map[string]map[*liveFeedClient]struct{}{},
                latest:  map[string]gin.H{},
                done:    make(chan struct{}),
        }
}

func liveFeedInterval() time.Duration {
        if v := strings.TrimSpace(os.Getenv("LIVE_FEED_INTERVAL_MS")); v != "" {
                if ms, err := strconv.Atoi(v); err == nil && ms >= 500 {
                        return time.Duration(ms) * time.Millisecond
                }
        }
        return 5 * time.Second
}

var liveFeedUpgrader = websocket.Upgrader{
        ReadBufferSize:  1024,
        WriteBufferSize: 4096,
        // Public read-only price feed consumed by mobile/desktop/extension
        // clients that send no Origin header; data exposed is identical to the
        // public /api/v1/terminal/* endpoints, so there is no origin policy.
        CheckOrigin: func(r *http.Request) bool { return true },
}

// handleLiveFeed upgrades the connection and registers the client.
func handleLiveFeed(c *gin.Context) {
        conn, err := liveFeedUpgrader.Upgrade(c.Writer, c.Request, nil)
        if err != nil {
                return // upgrade writes its own error response
        }
        client := &liveFeedClient{conn: conn, send: make(chan []byte, 32)}
        liveFeed.add(client)
        liveFeed.start()
        go client.writePump(liveFeed)
        client.readPump(liveFeed)
}

func (h *liveFeedHub) start() {
        h.stopOnce.Do(func() {
                go h.tickLoop()
        })
}

func (h *liveFeedHub) add(c *liveFeedClient) {
        h.mu.Lock()
        h.clients[c] = struct{}{}
        h.mu.Unlock()
}

func (h *liveFeedHub) remove(c *liveFeedClient) {
        h.mu.Lock()
        delete(h.clients, c)
        for sym, set := range h.subs {
                delete(set, c)
                if len(set) == 0 {
                        delete(h.subs, sym)
                }
        }
        h.mu.Unlock()
        close(c.send)
}

func (h *liveFeedHub) subscribe(c *liveFeedClient, symbols []string) {
        clean := make([]string, 0, len(symbols))
        h.mu.Lock()
        for _, s := range symbols {
                sym := strings.ToUpper(strings.TrimSpace(s))
                if sym == "" || len(sym) > 20 {
                        continue
                }
                if h.subs[sym] == nil {
                        h.subs[sym] = map[*liveFeedClient]struct{}{}
                }
                h.subs[sym][c] = struct{}{}
                clean = append(clean, sym)
        }
        latest := make([]gin.H, 0, len(clean))
        for _, sym := range clean {
                if t, ok := h.latest[sym]; ok {
                        latest = append(latest, t)
                }
        }
        h.mu.Unlock()
        h.sendJSON(c, gin.H{"type": "subscribed", "symbols": clean})
        for _, t := range latest {
                h.sendJSON(c, t)
        }
}

func (h *liveFeedHub) unsubscribe(c *liveFeedClient, symbols []string) {
        h.mu.Lock()
        for _, s := range symbols {
                sym := strings.ToUpper(strings.TrimSpace(s))
                if set, ok := h.subs[sym]; ok {
                        delete(set, c)
                        if len(set) == 0 {
                                delete(h.subs, sym)
                        }
                }
        }
        h.mu.Unlock()
}

func (h *liveFeedHub) sendJSON(c *liveFeedClient, v gin.H) {
        data, err := json.Marshal(v)
        if err != nil {
                return
        }
        select {
        case c.send <- data:
        default: // slow consumer; drop rather than block the hub
        }
}

func (h *liveFeedHub) broadcast(symbol string, v gin.H) {
        h.mu.Lock()
        targets := make([]*liveFeedClient, 0, len(h.subs[symbol]))
        for c := range h.subs[symbol] {
                targets = append(targets, c)
        }
        h.latest[symbol] = v
        h.mu.Unlock()
        data, err := json.Marshal(v)
        if err != nil {
                return
        }
        for _, c := range targets {
                select {
                case c.send <- data:
                default:
                }
        }
}

func (h *liveFeedHub) subscribedSymbols() []string {
        h.mu.Lock()
        defer h.mu.Unlock()
        out := make([]string, 0, len(h.subs))
        for sym := range h.subs {
                out = append(out, sym)
        }
        return out
}

func (h *liveFeedHub) clientCount() int {
        h.mu.Lock()
        defer h.mu.Unlock()
        return len(h.clients)
}

// tickLoop fetches the batched markets quote for all subscribed symbols once
// per interval and fans tickers out. Idle (zero subscriptions) costs zero
// upstream calls.
func (h *liveFeedHub) tickLoop() {
        ticker := time.NewTicker(liveFeedInterval())
        defer ticker.Stop()
        for {
                select {
                case <-h.done:
                        return
                case <-ticker.C:
                        symbols := h.subscribedSymbols()
                        if len(symbols) == 0 {
                                continue
                        }
                        h.fetchAndBroadcast(symbols)
                }
        }
}

// fetchAndBroadcast performs ONE batched upstream markets call for every
// subscribed symbol, then fans out per-symbol tickers. On upstream failure it
// notifies subscribers with an error frame and never fabricates a price.
func (h *liveFeedHub) fetchAndBroadcast(symbols []string) {
        ids := make([]string, 0, len(symbols))
        idToSymbol := map[string]string{}
        seen := map[string]bool{}
        for _, sym := range symbols {
                id := coinIDForSymbol(sym)
                if id == "" || seen[id] {
                        continue
                }
                seen[id] = true
                ids = append(ids, id)
                idToSymbol[id] = sym
        }
        if len(ids) == 0 {
                return
        }
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        u := fmt.Sprintf("https://api.coingecko.com/api/v3/coins/markets?vs_currency=usd&ids=%s", strings.Join(ids, ","))
        req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
        if err != nil {
                return
        }
        if appConfig != nil && appConfig.CoinGeckoAPIKey != "" {
                req.Header.Set("x-cg-pro-api-key", appConfig.CoinGeckoAPIKey)
        }
        client := &http.Client{Timeout: 10 * time.Second}
        resp, err := client.Do(req)
        if err != nil {
                for _, sym := range symbols {
                        h.broadcast(sym, gin.H{"type": "error", "symbol": sym, "message": "price feed unavailable"})
                }
                return
        }
        defer resp.Body.Close()
        body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
        if err != nil || resp.StatusCode != http.StatusOK {
                for _, sym := range symbols {
                        h.broadcast(sym, gin.H{"type": "error", "symbol": sym, "message": "price feed unavailable"})
                }
                return
        }
        var markets []CoinGeckoMarket
        if err := json.Unmarshal(body, &markets); err != nil {
                return
        }
        // The markets endpoint returns lowercase symbols; resolve the coin id
        // through the reverse lookup so each subscriber gets its own symbol.
        for _, m := range markets {
                id := coinIDForSymbol(strings.ToUpper(m.Symbol))
                sym := idToSymbol[id]
                if sym == "" {
                        continue
                }
                h.broadcast(sym, gin.H{
                        "type": "ticker", "symbol": sym, "coin": id,
                        "last_price": m.CurrentPrice, "high_24h": m.High24, "low_24h": m.Low24,
                        "change_24h_pct": m.PriceChange24, "volume_24h": m.TotalVolume,
                        "market_cap": m.MarketCap, "updated": m.LastUpdated,
                })
        }
}

func (c *liveFeedClient) readPump(h *liveFeedHub) {
        defer func() {
                h.remove(c)
                c.conn.Close()
        }()
        c.conn.SetReadLimit(4096)
        _ = c.conn.SetReadDeadline(time.Now().Add(90 * time.Second))
        c.conn.SetPongHandler(func(string) error {
                _ = c.conn.SetReadDeadline(time.Now().Add(90 * time.Second))
                return nil
        })
        for {
                _, data, err := c.conn.ReadMessage()
                if err != nil {
                        return
                }
                var req liveFeedRequest
                if err := json.Unmarshal(data, &req); err != nil {
                        h.sendJSON(c, gin.H{"type": "error", "message": "invalid request"})
                        continue
                }
                switch strings.ToLower(req.Action) {
                case "subscribe":
                        h.subscribe(c, req.Symbols)
                case "unsubscribe":
                        h.unsubscribe(c, req.Symbols)
                case "ping":
                        h.sendJSON(c, gin.H{"type": "pong"})
                default:
                        h.sendJSON(c, gin.H{"type": "error", "message": "unknown action"})
                }
        }
}

func (c *liveFeedClient) writePump(h *liveFeedHub) {
        ping := time.NewTicker(30 * time.Second)
        defer ping.Stop()
        for {
                select {
                case data, ok := <-c.send:
                        if !ok {
                                _ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
                                return
                        }
                        _ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
                        if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
                                return
                        }
                case <-ping.C:
                        _ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
                        if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                                return
                        }
                }
        }
}
