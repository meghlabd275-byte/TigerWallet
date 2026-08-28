// Package handlers implements the standalone WL-Liquidity backend REST API. A
// standalone clone of the TigerWallet liquidity aggregator — but
// PostgreSQL-persisted (real liquidity sources + routes). REAL bcrypt + JWT
// auth, REAL PostgreSQL persistence, real constant-product (x*y=k) quote math
// across persisted sources, the P2P trade surface (orders, trade lifecycle,
// trade messages, user profiles), and a fail-closed license gate (wlgate). No
// stubs, no fakes, no mocks, no demos. No fabricated pool data: starts empty,
// populated by admin CRUD.
package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tigerwallet/wl-liquidity/internal/config"
	"github.com/tigerwallet/wl-liquidity/internal/store"
	"github.com/tigerwallet/wl-shared/wlgate"
	"golang.org/x/crypto/bcrypt"
)

type Svc struct {
	cfg   *config.Config
	store *store.Store
	gate  *wlgate.Gate
}

func New(cfg *config.Config, s *store.Store, g *wlgate.Gate) *Svc {
	return &Svc{cfg: cfg, store: s, gate: g}
}

// ==================== Auth (real bcrypt + JWT) ====================

func (s *Svc) Register(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=8"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), s.cfg.BCryptCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "hash failed"})
		return
	}
	u, err := s.store.CreateUser(c.Request.Context(), req.Email, string(hash), req.Role)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": u.ID, "email": u.Email, "role": u.Role})
}

func (s *Svc) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	u, err := s.store.GetUserByEmail(c.Request.Context(), req.Email)
	if err != nil || !u.IsActive || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	wlID, err := uuid.Parse(s.cfg.WLClientID)
	if err != nil {
		wlID = uuid.Nil
	}
	scopes := []string{"wl_client", "liquidity"}
	tok, err := wlgate.IssueJWT(s.cfg.JWTSecret, u.ID, u.Email, wlID, scopes, s.cfg.JWTExpiry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token issue failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": tok, "user_id": u.ID, "email": u.Email, "role": u.Role})
}

// RequireRole is the admin gate. The wlgate JWT does not carry a role, so the
// role is loaded fresh from the users table on each request — fail-closed
// (403) if the user is missing or lacks one of the allowed roles.
func (s *Svc) RequireRole(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(c *gin.Context) {
		uid := wlgate.UserID(c)
		if uid == uuid.Nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient privileges"})
			return
		}
		u, err := s.store.GetUserByID(c.Request.Context(), uid)
		if err != nil || !u.IsActive || !allowed[u.Role] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient privileges"})
			return
		}
		c.Set("role", u.Role)
		c.Next()
	}
}

// ==================== Liquidity sources CRUD ====================

func (s *Svc) ListSources(c *gin.Context) {
	chain := c.Query("chain")
	srcs, err := s.store.ListSources(c.Request.Context(), chain)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(srcs))
	for i := range srcs {
		out = append(out, sourceJSON(&srcs[i]))
	}
	c.JSON(http.StatusOK, gin.H{"sources": out, "count": len(out)})
}

func (s *Svc) CreateSource(c *gin.Context) {
	src, ok := bindSource(c)
	if !ok {
		return
	}
	created, err := s.store.CreateSource(c.Request.Context(), src)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, sourceJSON(created))
}

func (s *Svc) UpdateSource(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	src, ok := bindSource(c)
	if !ok {
		return
	}
	updated, err := s.store.UpdateSource(c.Request.Context(), id, src)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "source not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sourceJSON(updated))
}

func (s *Svc) DeleteSource(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := s.store.DeleteSource(c.Request.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "source not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "status": "deleted"})
}

// bindSource parses a source body. Reserves default to "0" if omitted so the
// constant-product math has a real (zero) input rather than an empty string.
func bindSource(c *gin.Context) (*store.Source, bool) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Chain       string `json:"chain" binding:"required"`
		DEX         string `json:"dex" binding:"required"`
		PoolAddress string `json:"pool_address"`
		TokenA      string `json:"token_a" binding:"required"`
		TokenB      string `json:"token_b" binding:"required"`
		ReserveA    string `json:"reserve_a"`
		ReserveB    string `json:"reserve_b"`
		FeePct       string `json:"fee_pct"`
		APY         string `json:"apy"`
		IsActive    *bool  `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil, false
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	if req.ReserveA == "" {
		req.ReserveA = "0"
	}
	if req.ReserveB == "" {
		req.ReserveB = "0"
	}
	if req.FeePct == "" {
		req.FeePct = "0"
	}
	if req.APY == "" {
		req.APY = "0"
	}
	return &store.Source{
		Name: req.Name, Chain: req.Chain, DEX: req.DEX, PoolAddress: req.PoolAddress,
		TokenA: req.TokenA, TokenB: req.TokenB, ReserveA: req.ReserveA, ReserveB: req.ReserveB,
		FeePct: req.FeePct, APY: req.APY, IsActive: active,
	}, true
}

// ==================== Liquidity routes CRUD ====================

func (s *Svc) ListRoutes(c *gin.Context) {
	routes, err := s.store.ListRoutes(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(routes))
	for i := range routes {
		r := &routes[i]
		out = append(out, gin.H{
			"id": r.ID, "source_id": r.SourceID, "from_token": r.FromToken,
			"to_token": r.ToToken, "share_pct": r.SharePct, "created_at": r.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"routes": out, "count": len(out)})
}

func (s *Svc) CreateRoute(c *gin.Context) {
	var req struct {
		SourceID  string `json:"source_id" binding:"required"`
		FromToken string `json:"from_token" binding:"required"`
		ToToken   string `json:"to_token" binding:"required"`
		SharePct  string `json:"share_pct"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sourceID, err := uuid.Parse(req.SourceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid source_id"})
		return
	}
	if req.SharePct == "" {
		req.SharePct = "0"
	}
	if _, err := s.store.GetSource(c.Request.Context(), sourceID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "source not found"})
		return
	}
	r, err := s.store.CreateRoute(c.Request.Context(), sourceID, req.FromToken, req.ToToken, req.SharePct)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id": r.ID, "source_id": r.SourceID, "from_token": r.FromToken,
		"to_token": r.ToToken, "share_pct": r.SharePct, "created_at": r.CreatedAt,
	})
}

func (s *Svc) DeleteRoute(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := s.store.DeleteRoute(c.Request.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "route not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "status": "deleted"})
}

// ==================== Quote / depth / pools / best_dex (real x*y=k math) ====================

// SwapQuote is the best quote across all matching persisted sources, computed
// with the real constant-product formula.
type SwapQuote struct {
	FromToken   string      `json:"from_token"`
	ToToken     string      `json:"to_token"`
	FromAmount  float64     `json:"from_amount"`
	ToAmount    float64     `json:"to_amount"`
	PriceImpact float64     `json:"price_impact"`
	Slippage    float64     `json:"slippage"`
	Route       []RouteStep `json:"route"`
	DEX         string      `json:"dex"`
	SourceID    string      `json:"source_id"`
	GasEstimate float64     `json:"gas_estimate"`
	ValidUntil  int64       `json:"valid_until"`
}

type RouteStep struct {
	DEX       string `json:"dex"`
	SourceID  string `json:"source_id"`
	FromToken string `json:"from_token"`
	ToToken   string `json:"to_token"`
}

// Quote computes the best output across all matching persisted sources using
// the real constant-product formula x*y=k. Honest empty result (404) when no
// sources match — never fabricated.
func (s *Svc) Quote(c *gin.Context) {
	from := c.Query("from")
	to := c.Query("to")
	chain := c.Query("chain")
	amount := parseFloat(c.Query("amount"), 1.0)

	srcs, err := s.store.MatchingSources(c.Request.Context(), from, to, chain)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var best *SwapQuote
	for i := range srcs {
		q := calculateQuote(&srcs[i], from, to, amount)
		if q == nil {
			continue
		}
		if best == nil || q.ToAmount > best.ToAmount {
			best = q
		}
	}
	if best == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no liquidity found"})
		return
	}
	c.JSON(http.StatusOK, best)
}

// BestDEX returns the source with the highest output for the given amount — a
// real comparison across all matching persisted sources (not a liquidity sort
// like the canonical stub).
func (s *Svc) BestDEX(c *gin.Context) {
	from := c.Query("from")
	to := c.Query("to")
	chain := c.Query("chain")
	amount := parseFloat(c.Query("amount"), 1.0)

	srcs, err := s.store.MatchingSources(c.Request.Context(), from, to, chain)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var best *SwapQuote
	var bestSource *store.Source
	for i := range srcs {
		q := calculateQuote(&srcs[i], from, to, amount)
		if q == nil {
			continue
		}
		if best == nil || q.ToAmount > best.ToAmount {
			best = q
			bestSource = &srcs[i]
		}
	}
	if bestSource == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no pools found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"source":     sourceJSON(bestSource),
		"output":     best.ToAmount,
		"price_impact": best.PriceImpact,
	})
}

// Depth aggregates reserve depth across all matching persisted sources — a real
// SUM of reserve_a + reserve_b (per-pool) plus a per-pool share.
func (s *Svc) Depth(c *gin.Context) {
	tokenA := c.Query("token_a")
	tokenB := c.Query("token_b")
	chain := c.Query("chain")

	srcs, err := s.store.DepthSources(c.Request.Context(), tokenA, tokenB, chain)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	type poolDepth struct {
		SourceID string  `json:"source_id"`
		DEX      string  `json:"dex"`
		ReserveA float64 `json:"reserve_a"`
		ReserveB float64 `json:"reserve_b"`
		Depth    float64 `json:"depth"`
		Share    float64 `json:"share"`
	}
	pools := make([]poolDepth, 0, len(srcs))
	var totalDepth float64
	for i := range srcs {
		ra := parseFloat(srcs[i].ReserveA, 0)
		rb := parseFloat(srcs[i].ReserveB, 0)
		depth := ra + rb
		totalDepth += depth
		pools = append(pools, poolDepth{
			SourceID: srcs[i].ID.String(), DEX: srcs[i].DEX,
			ReserveA: ra, ReserveB: rb, Depth: depth,
		})
	}
	for i := range pools {
		if totalDepth > 0 {
			pools[i].Share = pools[i].Depth / totalDepth * 100
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"token_a": tokenA, "token_b": tokenB, "chain": chain,
		"total_depth": totalDepth, "pools": pools, "count": len(pools),
	})
}

// Pools lists the persisted active pools (sources) for a chain.
func (s *Svc) Pools(c *gin.Context) {
	chain := c.Query("chain")
	srcs, err := s.store.PoolSources(c.Request.Context(), chain)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(srcs))
	for i := range srcs {
		out = append(out, sourceJSON(&srcs[i]))
	}
	c.JSON(http.StatusOK, gin.H{"pools": out, "count": len(out)})
}

// ==================== P2P orders ====================

// CreateOrder posts a P2P buy/sell advertisement. The caller may pin one side
// (buyer_id or seller_id); the authenticated user fills the other side. With
// neither side given the caller is the seller of an open advertisement.
func (s *Svc) CreateOrder(c *gin.Context) {
	var req struct {
		BuyerID  string `json:"buyer_id"`
		SellerID string `json:"seller_id"`
		Asset    string `json:"asset" binding:"required"`
		Amount   string `json:"amount"`
		Price    string `json:"price"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	buyer, ok := parseOptionalUUID(c, req.BuyerID)
	if !ok {
		return
	}
	seller, ok := parseOptionalUUID(c, req.SellerID)
	if !ok {
		return
	}
	caller := wlgate.UserID(c)
	switch {
	case buyer == nil && seller == nil:
		seller = &caller
	case buyer == nil:
		buyer = &caller
	case seller == nil:
		seller = &caller
	}
	if *buyer == *seller {
		c.JSON(http.StatusBadRequest, gin.H{"error": "buyer and seller must differ"})
		return
	}
	if req.Amount == "" {
		req.Amount = "0"
	}
	if req.Price == "" {
		req.Price = "0"
	}
	created, err := s.store.CreateOrder(c.Request.Context(), &store.Order{
		BuyerID: buyer, SellerID: seller, Asset: req.Asset, Amount: req.Amount, Price: req.Price,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, orderJSON(created))
}

// ListOrders lists orders, with optional ?asset= and ?status= filters.
func (s *Svc) ListOrders(c *gin.Context) {
	orders, err := s.store.ListOrders(c.Request.Context(), c.Query("asset"), c.Query("status"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(orders))
	for i := range orders {
		out = append(out, orderJSON(&orders[i]))
	}
	c.JSON(http.StatusOK, gin.H{"orders": out, "count": len(out)})
}

func (s *Svc) GetOrder(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	o, err := s.store.GetOrder(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, orderJSON(o))
}

// ==================== P2P trade lifecycle ====================

// CreateTrade initiates a trade against an open order. The open side of the
// order is filled by the authenticated taker; the trade starts 'open' and the
// parent order moves to 'pending'.
func (s *Svc) CreateTrade(c *gin.Context) {
	var req struct {
		OrderID string `json:"order_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	orderID, err := uuid.Parse(req.OrderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order_id"})
		return
	}
	o, err := s.store.GetOrder(c.Request.Context(), orderID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if o.Status != "open" {
		c.JSON(http.StatusConflict, gin.H{"error": "order is not open"})
		return
	}
	caller := wlgate.UserID(c)
	buyer, seller := caller, caller
	if o.BuyerID != nil {
		buyer = *o.BuyerID
	}
	if o.SellerID != nil {
		seller = *o.SellerID
	}
	if buyer == seller {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot trade with yourself"})
		return
	}
	t, err := s.store.CreateTrade(c.Request.Context(), &store.Trade{
		OrderID: o.ID, BuyerID: buyer, SellerID: seller,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, tradeJSON(t))
}

func (s *Svc) GetTrade(c *gin.Context) {
	t, ok := s.loadTrade(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, tradeJSON(t))
}

// ConfirmTrade: buyer marks the fiat/payment leg as done (open -> confirmed).
func (s *Svc) ConfirmTrade(c *gin.Context) {
	s.transitionTrade(c, "open", "confirmed", "buyer")
}

// ReleaseTrade: seller releases the asset to the buyer (confirmed -> released).
// The parent order is completed at the same time.
func (s *Svc) ReleaseTrade(c *gin.Context) {
	s.transitionTrade(c, "confirmed", "released", "seller")
}

// DisputeTrade flags a non-terminal trade as disputed. Either party may open a
// dispute; an optional reason is persisted as a trade message.
func (s *Svc) DisputeTrade(c *gin.Context) {
	t, ok := s.loadTrade(c)
	if !ok {
		return
	}
	caller := wlgate.UserID(c)
	if caller != t.BuyerID && caller != t.SellerID {
		c.JSON(http.StatusForbidden, gin.H{"error": "only a trade party can open a dispute"})
		return
	}
	if t.Status != "open" && t.Status != "confirmed" {
		c.JSON(http.StatusConflict, gin.H{"error": "trade cannot be disputed in status " + t.Status})
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)
	if err := s.store.UpdateTradeStatus(c.Request.Context(), t.ID, "disputed"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if req.Reason != "" {
		_, _ = s.store.CreateMessage(c.Request.Context(), &store.Message{
			TradeID: t.ID, FromUser: caller, Body: "dispute: " + req.Reason,
		})
	}
	t.Status = "disputed"
	c.JSON(http.StatusOK, tradeJSON(t))
}

// transitionTrade applies a buyer/seller-gated status transition and reports
// the updated trade.
func (s *Svc) transitionTrade(c *gin.Context, from, to, actor string) {
	t, ok := s.loadTrade(c)
	if !ok {
		return
	}
	caller := wlgate.UserID(c)
	party := t.BuyerID
	if actor == "seller" {
		party = t.SellerID
	}
	if caller != party {
		c.JSON(http.StatusForbidden, gin.H{"error": "only the " + actor + " can " + to + " this trade"})
		return
	}
	if t.Status != from {
		c.JSON(http.StatusConflict, gin.H{"error": "trade is not " + from})
		return
	}
	if err := s.store.UpdateTradeStatus(c.Request.Context(), t.ID, to); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if to == "released" {
		if err := s.store.UpdateOrderStatus(c.Request.Context(), t.OrderID, "completed"); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	t.Status = to
	c.JSON(http.StatusOK, tradeJSON(t))
}

// loadTrade parses :id and loads the trade, writing the error response on
// failure.
func (s *Svc) loadTrade(c *gin.Context) (*store.Trade, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return nil, false
	}
	t, err := s.store.GetTrade(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "trade not found"})
			return nil, false
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return nil, false
	}
	return t, true
}

// ==================== P2P trade messages ====================

func (s *Svc) ListMessages(c *gin.Context) {
	t, ok := s.loadTrade(c)
	if !ok {
		return
	}
	msgs, err := s.store.ListMessages(c.Request.Context(), t.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(msgs))
	for i := range msgs {
		out = append(out, messageJSON(&msgs[i]))
	}
	c.JSON(http.StatusOK, gin.H{"messages": out, "count": len(out)})
}

func (s *Svc) CreateMessage(c *gin.Context) {
	t, ok := s.loadTrade(c)
	if !ok {
		return
	}
	var req struct {
		Body string `json:"body" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	m, err := s.store.CreateMessage(c.Request.Context(), &store.Message{
		TradeID: t.ID, FromUser: wlgate.UserID(c), Body: req.Body,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, messageJSON(m))
}

// ==================== P2P user profile ====================

// GetUserProfile looks up a user by :address (a user UUID or email) and returns
// the profile with real P2P trade counters from p2p_trades.
func (s *Svc) GetUserProfile(c *gin.Context) {
	address := c.Param("address")
	var u *store.User
	var err error
	if id, perr := uuid.Parse(address); perr == nil {
		u, err = s.store.GetUserByID(c.Request.Context(), id)
	} else {
		u, err = s.store.GetUserByEmail(c.Request.Context(), address)
	}
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	stats, err := s.store.TradeStats(c.Request.Context(), u.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"address": address, "id": u.ID, "email": u.Email, "role": u.Role,
		"is_active": u.IsActive, "created_at": u.CreatedAt,
		"total_trades": stats.Total, "completed_trades": stats.Completed,
		"disputed_trades": stats.Disputed,
	})
}

// ==================== Health ====================

func (s *Svc) Health(c *gin.Context) {
	total, active, _ := s.store.SourceCount(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{
		"status":        "healthy",
		"service":       "wl-liquidity",
		"licensed":      s.gate.IsAlive(),
		"reason":        s.gate.Reason(),
		"wl_client_id":  s.cfg.WLClientID,
		"product":       s.cfg.Product,
		"total_sources": total,
		"active_sources": active,
	})
}

// ==================== calculations ====================

// calculateQuote applies the real constant-product formula:
//
//	output = (input * (1-fee) * reserveOut) / (reserveIn + input * (1-fee))
//
// Returns nil if the pool cannot serve this direction (reserves <= 0).
func calculateQuote(src *store.Source, fromToken, toToken string, amount float64) *SwapQuote {
	var reserveIn, reserveOut float64
	if fromToken == src.TokenA {
		reserveIn = parseFloat(src.ReserveA, 0)
		reserveOut = parseFloat(src.ReserveB, 0)
	} else if fromToken == src.TokenB {
		reserveIn = parseFloat(src.ReserveB, 0)
		reserveOut = parseFloat(src.ReserveA, 0)
	} else {
		return nil
	}
	if reserveIn <= 0 || reserveOut <= 0 || amount <= 0 {
		return nil
	}
	fee := parseFloat(src.FeePct, 0) / 100.0
	inputWithFee := amount * (1 - fee)
	numerator := inputWithFee * reserveOut
	denominator := reserveIn + inputWithFee
	if denominator == 0 {
		return nil
	}
	output := numerator / denominator
	priceImpact := (amount / (reserveIn + amount)) * 100
	return &SwapQuote{
		FromToken:   fromToken,
		ToToken:     toToken,
		FromAmount:  amount,
		ToAmount:    output,
		PriceImpact: priceImpact,
		Slippage:    priceImpact * 1.5,
		Route: []RouteStep{
			{DEX: src.DEX, SourceID: src.ID.String(), FromToken: fromToken, ToToken: toToken},
		},
		DEX:         src.DEX,
		SourceID:    src.ID.String(),
		GasEstimate: 150000,
		ValidUntil:  time.Now().Add(30 * time.Second).UnixMilli(),
	}
}

// ==================== helpers ====================

func orderJSON(o *store.Order) gin.H {
	return gin.H{
		"id": o.ID, "buyer_id": o.BuyerID, "seller_id": o.SellerID,
		"asset": o.Asset, "amount": o.Amount, "price": o.Price,
		"status": o.Status, "created_at": o.CreatedAt,
	}
}

func tradeJSON(t *store.Trade) gin.H {
	return gin.H{
		"id": t.ID, "order_id": t.OrderID, "buyer_id": t.BuyerID,
		"seller_id": t.SellerID, "status": t.Status, "created_at": t.CreatedAt,
	}
}

func messageJSON(m *store.Message) gin.H {
	return gin.H{
		"id": m.ID, "trade_id": m.TradeID, "from_user": m.FromUser,
		"body": m.Body, "created_at": m.CreatedAt,
	}
}

// parseOptionalUUID parses an optional UUID request field ("" -> nil). On a
// parse failure it writes the 400 response and reports ok=false.
func parseOptionalUUID(c *gin.Context, s string) (*uuid.UUID, bool) {
	if s == "" {
		return nil, true
	}
	id, err := uuid.Parse(s)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid uuid: " + s})
		return nil, false
	}
	return &id, true
}

func sourceJSON(s *store.Source) gin.H {
	return gin.H{
		"id": s.ID, "name": s.Name, "chain": s.Chain, "dex": s.DEX,
		"pool_address": s.PoolAddress, "token_a": s.TokenA, "token_b": s.TokenB,
		"reserve_a": s.ReserveA, "reserve_b": s.ReserveB, "fee_pct": s.FeePct,
		"apy": s.APY, "is_active": s.IsActive, "created_at": s.CreatedAt,
	}
}

// parseFloat parses a NUMERIC-as-string column; falls back to def on parse error.
func parseFloat(s string, def float64) float64 {
	if s == "" {
		return def
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return def
	}
	return v
}
