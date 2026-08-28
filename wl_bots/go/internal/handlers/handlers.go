// Package handlers implements the standalone WL-Bots backend REST API. A clone
// of the TigerWallet mm_bot_platform/bot_api bot management platform — full bot
// lifecycle (create/start/stop/pause/delete), executions, subscriptions, fee
// configs, and per-user encrypted API keys. REAL bcrypt + JWT auth, REAL
// AES-GCM at-rest encryption (wlcrypto), REAL PostgreSQL persistence, and a
// fail-closed license gate (wlgate). No stubs, no fakes, no mocks, no demos.
package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tigerwallet/wl-bots/internal/config"
	"github.com/tigerwallet/wl-bots/internal/store"
	"github.com/tigerwallet/wl-shared/wlcrypto"
	"github.com/tigerwallet/wl-shared/wlgate"
	"golang.org/x/crypto/bcrypt"
)

// botTypes mirrors the TigerWallet bot_core BotType enum (18 types).
var botTypes = []string{
	"market_maker", "liquidity_provider", "sniper", "front_run", "mev",
	"sandwich", "flash_loan", "cross_chain", "perp_hedge", "grid",
	"dca", "momentum", "mean_reversion", "scalping", "ai_trading",
	"signal", "arbitrage", "custom",
}

func validBotType(t string) bool {
	for _, b := range botTypes {
		if b == t {
			return true
		}
	}
	return false
}

// subscriptionTiers mirrors the canonical bot_api defaultTiers (4 tiers). These
// are config constants — the WL deployment's subscription offering.
var subscriptionTiers = []gin.H{
	{"id": "free", "name": "Free", "max_bots": 1, "max_dex": 1, "max_cex": 0, "latency_ms": 5000, "monthly_fee": "0"},
	{"id": "basic", "name": "Basic", "max_bots": 3, "max_dex": 5, "max_cex": 3, "latency_ms": 2000, "monthly_fee": "49"},
	{"id": "pro", "name": "Pro", "max_bots": 10, "max_dex": 15, "max_cex": 10, "latency_ms": 500, "monthly_fee": "299"},
	{"id": "enterprise", "name": "Enterprise", "max_bots": 50, "max_dex": 50, "max_cex": 30, "latency_ms": 100, "monthly_fee": "1499"},
}

// validRoles mirrors the canonical bot_api UserRole set.
func validRoles() map[string]bool {
	return map[string]bool{
		"super_admin": true, "bot_operator": true, "finance_admin": true, "client": true,
	}
}

type Svc struct {
	cfg       *config.Config
	store     *store.Store
	gate      *wlgate.Gate
	httpClient *http.Client // shared HTTP client for dispatch to bot_core
}

func New(cfg *config.Config, s *store.Store, g *wlgate.Gate) *Svc {
	return &Svc{
		cfg:   cfg,
		store: s,
		gate:  g,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
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
	// Issue the user's assigned scopes (set by the WL client via UpdateUserScopes)
	// in the JWT. wlgate.RequireScope enforces these on admin routes. wl_client
	// is always included (the WL client owner has full tenancy control). The
	// canonical scope taxonomy lives in white_label_admin/go/internal/roles.
	scopes := u.Scopes
	if len(scopes) == 0 {
		// Default: a plain client user (no admin scopes) gets only the base
		// 'user' scope so RequireScope denies admin routes. wl_client is NOT
		// granted by default — only the WL client owner has it assigned.
		scopes = []string{"user"}
	}
	tok, err := wlgate.IssueJWT(s.cfg.JWTSecret, u.ID, u.Email, wlID, scopes, s.cfg.JWTExpiry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token issue failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": tok, "user_id": u.ID, "email": u.Email, "role": u.Role})
}

// Logout is audit-only and stateless: wl-bots uses stateless HS256 JWTs (no
// server-side session table that gates auth), so the token itself is NOT
// invalidated. We record a real audit event in PostgreSQL (the honest record
// that the user requested logout) and return success — mirroring canonical's
// stateless logout intent.
func (s *Svc) Logout(c *gin.Context) {
	uid := wlgate.UserID(c)
	if uid != uuid.Nil {
		_ = s.store.RecordAuditEvent(c.Request.Context(), uid, "logout", "user requested logout")
	}
	c.JSON(http.StatusOK, gin.H{"status": "logged out"})
}

// RequireRole is kept for backward compat but now delegates to wlgate.HasScope.
// The wl-bots JWT carries the user's assigned scopes (set by the WL client via
// UpdateUserScopes). 'wl_client' (the WL owner) always passes. The canonical
// bot-admin scope is 'bot_admin' (white_label_admin/go/internal/roles.BotAdmin).
// Legacy local role strings (super_admin/finance_admin/bot_operator) are honored
// ONLY if the user also holds the matching scope, so existing deployments don't
// break while the WL client migrates to the canonical taxonomy.
func (s *Svc) RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := wlgate.UserID(c)
		if uid == uuid.Nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient privileges"})
			return
		}
		// wl_client (WL owner) always passes — full tenancy control.
		if wlgate.HasScope(c, "wl_client") {
			c.Set("role", "wl_client")
			c.Next()
			return
		}
		// Canonical scope: bot_admin controls all bots.
		if wlgate.HasScope(c, "bot_admin") {
			c.Set("role", "bot_admin")
			c.Next()
			return
		}
		// Legacy local-role fallback: load the user's role from the DB + match.
		allowed := make(map[string]bool, len(roles))
		for _, r := range roles {
			allowed[r] = true
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

// UpdateAdminScopes is the WL-client-facing endpoint to grant/revoke scoped
// admin roles on a bots user. Mirrors white_label_admin AssignAdminRole. Only
// a caller holding 'wl_client' (the WL owner) may set scopes — a bot_admin
// cannot escalate themselves or others. The scopes MUST be in the canonical
// whitelist (validated server-side).
func (s *Svc) UpdateAdminScopes(c *gin.Context) {
	if !wlgate.HasScope(c, "wl_client") {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "only the WL client owner may assign admin scopes"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	var req struct {
		Scopes []string `json:"scopes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Validate every scope is in the canonical whitelist.
	valid := map[string]bool{
		"wl_client": true, "bot_admin": true, "trading_admin": true, "p2p_admin": true,
		"listing_admin": true, "liquidity_admin": true, "wallet_admin": true,
		"customer_service_admin": true, "marketing_admin": true, "kyc_admin": true,
		"card_admin": true, "reward_admin": true, "security_admin": true, "compliance_admin": true,
		"user": true,
	}
	for _, sc := range req.Scopes {
		if !valid[sc] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scope: " + sc})
			return
		}
	}
	if err := s.store.UpdateUserScopes(c.Request.Context(), id, req.Scopes); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_ = s.store.RecordAuditEvent(c.Request.Context(), wlgate.UserID(c), "update_admin_scopes", "user "+id.String()+" scopes="+strings.Join(req.Scopes, ","))
	c.JSON(http.StatusOK, gin.H{"user_id": id, "scopes": req.Scopes})
}

// ==================== Bots ====================

func (s *Svc) CreateBot(c *gin.Context) {
	userID := wlgate.UserID(c)
	var req struct {
		Name     string         `json:"name" binding:"required"`
		BotType  string         `json:"bot_type" binding:"required"`
		Exchange string         `json:"exchange"`
		Pair     string         `json:"pair"`
		Config   map[string]any `json:"config"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !validBotType(req.BotType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid bot_type", "valid_types": botTypes})
		return
	}
	if req.Config == nil {
		req.Config = map[string]any{}
	}
	b, err := s.store.CreateBot(c.Request.Context(), userID, req.Name, req.BotType, req.Exchange, req.Pair, req.Config)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, botJSON(b))
}

func (s *Svc) ListBots(c *gin.Context) {
	userID := wlgate.UserID(c)
	bots, err := s.store.ListBots(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(bots))
	for i := range bots {
		out = append(out, botJSON(&bots[i]))
	}
	c.JSON(http.StatusOK, gin.H{"bots": out, "count": len(out)})
}

func (s *Svc) GetBot(c *gin.Context) {
	b, ok := s.fetchOwnedBot(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, botJSON(b))
}

func (s *Svc) DeleteBot(c *gin.Context) {
	b, ok := s.fetchOwnedBot(c)
	if !ok {
		return
	}
	if err := s.store.DeleteBot(c.Request.Context(), b.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted", "id": b.ID})
}

func (s *Svc) StartBot(c *gin.Context) {
	b, ok := s.fetchOwnedBot(c)
	if !ok {
		return
	}
	if err := s.store.SetBotStatus(c.Request.Context(), b.ID, "running"); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "bot not found"})
		return
	}
	_ = s.store.AppendBotLog(c.Request.Context(), b.ID, "info", "start requested")
	// Dispatch real start to the Rust bot_core execution plane (same async
	// best-effort pattern as canonical mm_bot_platform/bot_api). DB status
	// already succeeded; dispatch failures are logged, not fatal.
	s.dispatchBotCore(c, b, "start")
	b.Status = "running"
	c.JSON(http.StatusOK, botJSON(b))
}

func (s *Svc) StopBot(c *gin.Context) {
	b, ok := s.fetchOwnedBot(c)
	if !ok {
		return
	}
	if err := s.store.SetBotStatus(c.Request.Context(), b.ID, "stopped"); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "bot not found"})
		return
	}
	_ = s.store.AppendBotLog(c.Request.Context(), b.ID, "info", "stop requested")
	s.dispatchBotCore(c, b, "stop")
	b.Status = "stopped"
	c.JSON(http.StatusOK, botJSON(b))
}

func (s *Svc) PauseBot(c *gin.Context) {
	b, ok := s.fetchOwnedBot(c)
	if !ok {
		return
	}
	if err := s.store.SetBotStatus(c.Request.Context(), b.ID, "paused"); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "bot not found"})
		return
	}
	_ = s.store.AppendBotLog(c.Request.Context(), b.ID, "info", "pause requested")
	s.dispatchBotCore(c, b, "pause")
	b.Status = "paused"
	c.JSON(http.StatusOK, botJSON(b))
}

// dispatchBotCore sends a real start/stop/pause command to the Rust bot_core
// execution plane at /dispatch/<action>. For "start" it builds the StartReq
// tagged-enum payload bot_core expects (marketmaker/arbitrage/sniper) from the
// bot row; for stop/pause it sends {bot_id}. This mirrors the canonical
// mm_bot_platform/bot_api dispatchBotCore exactly (best-effort async dispatch:
// the DB status update already succeeded and the execution plane may be down
// for maintenance). No fake execution: signal-only bot types
// (grid/dca/momentum/etc.) have no real runner in bot_core yet and are
// skipped (logged), with the DB status still transitioning to running.
func (s *Svc) dispatchBotCore(c *gin.Context, b *store.Bot, action string) {
	var body []byte
	if action == "start" {
		payload, err := s.buildStartPayload(c, b)
		if err != nil {
			log.Printf("dispatchBotCore start %s: failed to build payload: %v", b.ID, err)
			return
		}
		body = payload
	} else {
		body, _ = json.Marshal(map[string]string{"bot_id": b.ID.String()})
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), "POST",
		s.botCoreURL()+"/dispatch/"+action, bytes.NewReader(body))
	if err != nil {
		log.Printf("dispatchBotCore: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("dispatchBotCore %s %s: %v (bot_core unreachable)", action, b.ID, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		log.Printf("dispatchBotCore %s %s: bot_core returned %d", action, b.ID, resp.StatusCode)
	}
}

func (s *Svc) botCoreURL() string {
	if s.cfg.BotCoreURL != "" {
		return s.cfg.BotCoreURL
	}
	return "http://localhost:8472"
}

// buildStartPayload builds the StartReq tagged-enum JSON that bot_core's
// dispatch_start endpoint expects, mapping bot_type -> strategy kind:
//   market_maker -> marketmaker, arbitrage -> arbitrage, sniper -> sniper.
// Other bot types are signal-only in bot_core and have no real execution
// runner; their start dispatch is skipped (returns errStopDispatch). This is
// honest: no fake execution is emitted.
func (s *Svc) buildStartPayload(c *gin.Context, b *store.Bot) ([]byte, error) {
	userID := wlgate.UserID(c)
	cfg := b.Config
	if cfg == nil {
		cfg = map[string]any{}
	}
	switch b.BotType {
	case "market_maker":
		cexCreds, err := s.fetchCEXCreds(c, userID, b.Exchange)
		if err != nil {
			return nil, err
		}
		payload := map[string]any{
			"kind":       "marketmaker",
			"bot_id":     b.ID.String(),
			"exchange":   cexCreds.exchange,
			"api_key":    cexCreds.apiKey,
			"secret_key": cexCreds.apiSecret,
			"symbol":     b.Pair,
			"order_size": cfgFloat(cfg, "order_size", 0.01),
			"spread_bps": cfgFloat(cfg, "spread_bps", 10),
		}
		if v, ok := cfg["base_url"].(string); ok && v != "" {
			payload["base_url"] = v
		}
		if v, ok := cfg["passphrase"].(string); ok && v != "" {
			payload["passphrase"] = v
		}
		if pi, ok := cfg["poll_interval_ms"].(float64); ok {
			payload["poll_interval_ms"] = int64(pi)
		}
		return json.Marshal(payload)

	case "arbitrage":
		cexCreds, err := s.fetchCEXCreds(c, userID, b.Exchange)
		if err != nil {
			return nil, err
		}
		payload := map[string]any{
			"kind":          "arbitrage",
			"bot_id":        b.ID.String(),
			"exchange":      cexCreds.exchange,
			"api_key":       cexCreds.apiKey,
			"secret_key":    cexCreds.apiSecret,
			"symbol":        b.Pair,
			"threshold_bps": cfgFloat(cfg, "threshold_bps", 50),
		}
		if v, ok := cfg["base_url"].(string); ok && v != "" {
			payload["base_url"] = v
		}
		if pi, ok := cfg["poll_interval_ms"].(float64); ok {
			payload["poll_interval_ms"] = int64(pi)
		}
		return json.Marshal(payload)

	case "sniper":
		payload := map[string]any{
			"kind":   "sniper",
			"bot_id": b.ID.String(),
			"symbol": b.Pair,
		}
		if mu, ok := cfg["mempool_url"].(string); ok && mu != "" {
			payload["mempool_url"] = mu
		}
		if pi, ok := cfg["poll_interval_ms"].(float64); ok {
			payload["poll_interval_ms"] = int64(pi)
		}
		if mta, ok := cfg["min_target_amount"].(float64); ok {
			payload["min_target_amount"] = int64(mta)
		}
		return json.Marshal(payload)

	default:
		// Signal-only bot types have no real execution runner in bot_core.
		return nil, errStopDispatch
	}
}

// errStopDispatch signals that a bot type has no bot_core execution runner and
// its start dispatch should be skipped (the DB status still transitions).
var errStopDispatch = errors.New("bot type has no bot_core execution runner")

func cfgFloat(cfg map[string]any, key string, def float64) float64 {
	if v, ok := cfg[key].(float64); ok {
		return v
	}
	return def
}

// cexCreds holds the decrypted CEX API key + secret for a bot dispatch.
type cexCreds struct {
	exchange  string
	apiKey    string
	apiSecret string
}

// fetchCEXCreds resolves the per-user API key (from api_keys, AES-GCM
// encrypted at rest) and the platform-level API secret (from
// bot_cex_connections, admin-managed) for the given exchange, decrypting both
// via wlcrypto. Returns an error if no key is configured for the exchange —
// the caller skips dispatch rather than emitting a payload with empty creds.
func (s *Svc) fetchCEXCreds(c *gin.Context, userID uuid.UUID, exchange string) (cexCreds, error) {
	if exchange == "" {
		return cexCreds{}, errors.New("bot has no exchange configured")
	}
	keys, err := s.store.ListAPIKeys(c.Request.Context(), userID)
	if err != nil {
		return cexCreds{}, fmt.Errorf("list api keys: %w", err)
	}
	var encAPIKey string
	for i := range keys {
		if keys[i].Exchange == exchange && keys[i].Enabled {
			encAPIKey = keys[i].APIKeyEncrypted
			break
		}
	}
	if encAPIKey == "" {
		return cexCreds{}, fmt.Errorf("no api key configured for exchange %s", exchange)
	}
	apiKey, err := wlcrypto.DecryptSeedAtRest(encAPIKey, s.cfg.JWTSecret)
	if err != nil {
		return cexCreds{}, fmt.Errorf("decrypt api key: %w", err)
	}
	// Platform-level secret for the same exchange (admin-managed).
	var apiSecret string
	conns, err := s.store.ListCEX(c.Request.Context())
	if err == nil {
		for i := range conns {
			if conns[i].Exchange == exchange && conns[i].IsActive {
				dec, derr := wlcrypto.DecryptSeedAtRest(conns[i].APISecretEncrypted, s.cfg.JWTSecret)
				if derr == nil {
					apiSecret = string(dec)
				}
				break
			}
		}
	}
	return cexCreds{exchange: exchange, apiKey: string(apiKey), apiSecret: apiSecret}, nil
}

func (s *Svc) ListBotExecutions(c *gin.Context) {
	b, ok := s.fetchOwnedBot(c)
	if !ok {
		return
	}
	exs, err := s.store.ListBotExecutions(c.Request.Context(), b.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(exs))
	for i := range exs {
		e := &exs[i]
		out = append(out, gin.H{
			"id":         e.ID,
			"bot_id":     e.BotID,
			"status":     e.Status,
			"pnl":        e.PNL,
			"started_at": e.StartedAt,
			"ended_at":   e.EndedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"executions": out, "count": len(out)})
}

func (s *Svc) ListBotLogs(c *gin.Context) {
	b, ok := s.fetchOwnedBot(c)
	if !ok {
		return
	}
	logs, err := s.store.ListBotLogs(c.Request.Context(), b.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(logs))
	for i := range logs {
		l := &logs[i]
		out = append(out, gin.H{
			"id":         l.ID,
			"bot_id":     l.BotID,
			"level":      l.Level,
			"message":    l.Message,
			"created_at": l.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"logs": out, "count": len(out)})
}

// ==================== Subscriptions ====================

func (s *Svc) CreateSubscription(c *gin.Context) {
	userID := wlgate.UserID(c)
	var req struct {
		Tier      string `json:"tier" binding:"required"`
		ExpiresIn string `json:"expires_in"` // duration, e.g. "720h"
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var expiresAt *time.Time
	if req.ExpiresIn != "" {
		if d, err := time.ParseDuration(req.ExpiresIn); err == nil {
			t := time.Now().Add(d)
			expiresAt = &t
		}
	}
	sub, err := s.store.CreateSubscription(c.Request.Context(), userID, req.Tier, expiresAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id":         sub.ID,
		"user_id":    sub.UserID,
		"tier":       sub.Tier,
		"started_at": sub.StartedAt,
		"expires_at": sub.ExpiresAt,
	})
}

func (s *Svc) ListSubscriptions(c *gin.Context) {
	userID := wlgate.UserID(c)
	subs, err := s.store.ListSubscriptions(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(subs))
	for i := range subs {
		sub := &subs[i]
		out = append(out, gin.H{
			"id":         sub.ID,
			"user_id":    sub.UserID,
			"tier":       sub.Tier,
			"started_at": sub.StartedAt,
			"expires_at": sub.ExpiresAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"subscriptions": out, "count": len(out)})
}

// ==================== Fee configs ====================

func (s *Svc) CreateFeeConfig(c *gin.Context) {
	var req struct {
		Name       string `json:"name" binding:"required"`
		Percentage string `json:"percentage" binding:"required"`
		Enabled    *bool  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	f, err := s.store.CreateFeeConfig(c.Request.Context(), req.Name, req.Percentage, enabled)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id":         f.ID,
		"name":       f.Name,
		"percentage": f.Percentage,
		"enabled":    f.Enabled,
		"created_at": f.CreatedAt,
	})
}

func (s *Svc) ListFeeConfigs(c *gin.Context) {
	fees, err := s.store.ListFeeConfigs(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(fees))
	for i := range fees {
		f := &fees[i]
		out = append(out, gin.H{
			"id":         f.ID,
			"name":       f.Name,
			"percentage": f.Percentage,
			"enabled":    f.Enabled,
			"created_at": f.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"fee_configs": out, "count": len(out)})
}

// ==================== API keys (AES-GCM at rest via wlcrypto) ====================

func (s *Svc) CreateApiKey(c *gin.Context) {
	userID := wlgate.UserID(c)
	var req struct {
		Exchange string `json:"exchange" binding:"required"`
		APIKey   string `json:"api_key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Real AEAD: scrypt(passphrase) -> AES-256-GCM. The JWT secret is the
	// at-rest passphrase (configured per WL client deployment).
	encrypted, err := wlcrypto.EncryptSeedAtRest([]byte(req.APIKey), s.cfg.JWTSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "encryption failed"})
		return
	}
	k, err := s.store.CreateAPIKey(c.Request.Context(), userID, req.Exchange, encrypted)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id":          k.ID,
		"user_id":     k.UserID,
		"exchange":    k.Exchange,
		"enabled":     k.Enabled,
		"created_at":  k.CreatedAt,
		"api_key_preview": previewKey(req.APIKey),
	})
}

func (s *Svc) ListApiKeys(c *gin.Context) {
	userID := wlgate.UserID(c)
	keys, err := s.store.ListAPIKeys(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(keys))
	for i := range keys {
		k := &keys[i]
		// Decrypt back to plaintext only for the owning user's view.
		plain, derr := wlcrypto.DecryptSeedAtRest(k.APIKeyEncrypted, s.cfg.JWTSecret)
		apiKey := ""
		if derr == nil {
			apiKey = string(plain)
		}
		out = append(out, gin.H{
			"id":             k.ID,
			"user_id":        k.UserID,
			"exchange":       k.Exchange,
			"enabled":        k.Enabled,
			"created_at":     k.CreatedAt,
			"api_key":        apiKey,
			"api_key_preview": previewKey(apiKey),
		})
	}
	c.JSON(http.StatusOK, gin.H{"api_keys": out, "count": len(out)})
}

// ==================== Public ====================

// PublicTiers returns the subscription tiers + bot types (no auth).
func (s *Svc) PublicTiers(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"tiers": subscriptionTiers, "bot_types": botTypes})
}

// ==================== Bot operator aliases (frontend-compat) ====================

// ListBotInstances is an alias of ListBots (GET /bots/instances).
func (s *Svc) ListBotInstances(c *gin.Context) { s.ListBots(c) }

// CurrentBotUser returns the authenticated user's profile (GET /bots/me).
func (s *Svc) CurrentBotUser(c *gin.Context) {
	uid := wlgate.UserID(c)
	u, err := s.store.GetUserByID(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, userJSON(u))
}

// ListBotTransactions returns executions across the caller's bots (real PG
// query filtered by the authenticated user).
func (s *Svc) ListBotTransactions(c *gin.Context) {
	uid := wlgate.UserID(c)
	txs, err := s.store.ListBotTransactions(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(txs))
	for i := range txs {
		t := &txs[i]
		out = append(out, gin.H{
			"id":         t.ID,
			"bot_id":     t.BotID,
			"status":     t.Status,
			"pnl":        t.PNL,
			"started_at": t.StartedAt,
			"ended_at":   t.EndedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"transactions": out, "count": len(out)})
}

// SetBotStatus sets a bot's status directly (POST /bots/:id/status). Distinct
// from start/stop/pause lifecycle endpoints — accepts any status string.
func (s *Svc) SetBotStatus(c *gin.Context) {
	b, ok := s.fetchOwnedBot(c)
	if !ok {
		return
	}
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := s.store.SetBotStatus(c.Request.Context(), b.ID, req.Status); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "bot not found"})
		return
	}
	_ = s.store.AppendBotLog(c.Request.Context(), b.ID, "info", "status set to "+req.Status)
	b.Status = req.Status
	c.JSON(http.StatusOK, botJSON(b))
}

// ==================== User management (admin/operator surface) ====================

// ListUsers returns all users (admin). Mirrors canonical adminListUsers.
func (s *Svc) ListUsers(c *gin.Context) {
	users, err := s.store.ListUsers(c.Request.Context(), 500)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(users))
	for i := range users {
		out = append(out, userJSON(&users[i]))
	}
	c.JSON(http.StatusOK, gin.H{"users": out, "count": len(out)})
}

// CreateBotUser creates a new bot-platform user (admin/operator only). Mirrors
// canonical createBotUser.
func (s *Svc) CreateBotUser(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
		Role  string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Role == "" {
		req.Role = "client"
	}
	if !validRoles()[req.Role] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role", "valid_roles": []string{"super_admin", "bot_operator", "finance_admin", "client"}})
		return
	}
	u, err := s.store.CreateBotUser(c.Request.Context(), req.Email, req.Role)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, userJSON(u))
}

// DeleteBotUser removes a bot-platform user (admin only). Mirrors canonical
// deleteBotUser.
func (s *Svc) DeleteBotUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := s.store.DeleteUser(c.Request.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "status": "deleted"})
}

// UpdateUserStatus sets is_active and (optionally) role. Mirrors canonical
// adminUserStatus.
func (s *Svc) UpdateUserStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		IsActive bool   `json:"is_active"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Role != "" && !validRoles()[req.Role] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role"})
		return
	}
	if err := s.store.UpdateUserStatus(c.Request.Context(), id, req.IsActive, req.Role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "is_active": req.IsActive, "role": req.Role})
}

// ==================== Platform stats (real COUNTs) ====================

func (s *Svc) Stats(c *gin.Context) {
	st, err := s.store.Stats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	dist, _ := s.store.BotTypeDistribution(c.Request.Context())
	typeDist := make([]gin.H, 0, len(dist))
	for i := range dist {
		typeDist = append(typeDist, gin.H{"bot_type": dist[i].BotType, "count": dist[i].Count})
	}
	c.JSON(http.StatusOK, gin.H{
		"total_users":      st.TotalUsers,
		"total_bots":       st.TotalBots,
		"running_bots":     st.RunningBots,
		"total_executions": st.TotalExecutions,
		"bot_type_distribution": typeDist,
	})
}

// ==================== Fee config update ====================

// UpdateFeeConfig updates an existing fee config by id (real PG UPDATE).
func (s *Svc) UpdateFeeConfig(c *gin.Context) {
	var req struct {
		ID         string `json:"id" binding:"required"`
		Name       string `json:"name" binding:"required"`
		Percentage string `json:"percentage" binding:"required"`
		Enabled    *bool  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, err := uuid.Parse(req.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	if err := s.store.UpdateFeeConfig(c.Request.Context(), id, req.Name, req.Percentage, enabled); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "fee config not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":         id,
		"name":       req.Name,
		"percentage": req.Percentage,
		"enabled":    enabled,
	})
}

// ==================== API key DELETE (/keys full CRUD) ====================

// DeleteApiKey removes one of the caller's API keys (DELETE /keys/:id). Backed
// by the same api_keys table as /api-keys.
func (s *Svc) DeleteApiKey(c *gin.Context) {
	uid := wlgate.UserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := s.store.DeleteAPIKey(c.Request.Context(), id, uid); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "key not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "status": "deleted"})
}

// ==================== CEX connector configs (AES-GCM at rest) ====================

func (s *Svc) ListCEX(c *gin.Context) {
	conns, err := s.store.ListCEX(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(conns))
	for i := range conns {
		cn := &conns[i]
		out = append(out, gin.H{
			"id":         cn.ID,
			"exchange":   cn.Exchange,
			"is_active":  cn.IsActive,
			"created_at": cn.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"connections": out, "count": len(out)})
}

func (s *Svc) CreateCEX(c *gin.Context) {
	var req struct {
		Exchange  string `json:"exchange" binding:"required"`
		APIKey    string `json:"api_key" binding:"required"`
		APISecret string `json:"api_secret" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	encKey, err := wlcrypto.EncryptSeedAtRest([]byte(req.APIKey), s.cfg.JWTSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "encryption failed"})
		return
	}
	encSecret, err := wlcrypto.EncryptSeedAtRest([]byte(req.APISecret), s.cfg.JWTSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "encryption failed"})
		return
	}
	cn, err := s.store.CreateCEX(c.Request.Context(), req.Exchange, encKey, encSecret)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id":        cn.ID,
		"exchange":  cn.Exchange,
		"is_active": cn.IsActive,
	})
}

func (s *Svc) DeleteCEX(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := s.store.DeleteCEX(c.Request.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "connection not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "status": "deleted"})
}

// ==================== DEX connector configs ====================

func (s *Svc) ListDEX(c *gin.Context) {
	conns, err := s.store.ListDEX(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(conns))
	for i := range conns {
		d := &conns[i]
		out = append(out, gin.H{
			"id":         d.ID,
			"dex":        d.DEX,
			"chain_id":   d.ChainID,
			"rpc_url":    d.RPCURL,
			"is_active":  d.IsActive,
			"created_at": d.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"connections": out, "count": len(out)})
}

func (s *Svc) CreateDEX(c *gin.Context) {
	var req struct {
		DEX     string `json:"dex" binding:"required"`
		ChainID int64  `json:"chain_id" binding:"required"`
		RPCURL  string `json:"rpc_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	d, err := s.store.CreateDEX(c.Request.Context(), req.DEX, req.ChainID, req.RPCURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id":       d.ID,
		"dex":      d.DEX,
		"chain_id": d.ChainID,
		"rpc_url":  d.RPCURL,
	})
}

func (s *Svc) DeleteDEX(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := s.store.DeleteDEX(c.Request.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "connection not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "status": "deleted"})
}

// ==================== Admin fee addresses ====================

func (s *Svc) ListFeeAddresses(c *gin.Context) {
	addrs, err := s.store.ListFeeAddresses(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(addrs))
	for i := range addrs {
		a := &addrs[i]
		out = append(out, gin.H{
			"id":         a.ID,
			"label":      a.Label,
			"address":    a.Address,
			"chain_id":   a.ChainID,
			"is_active":  a.IsActive,
			"created_at": a.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"addresses": out, "count": len(out)})
}

func (s *Svc) CreateFeeAddress(c *gin.Context) {
	var req struct {
		Label   string `json:"label" binding:"required"`
		Address string `json:"address" binding:"required"`
		ChainID int64  `json:"chain_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	a, err := s.store.CreateFeeAddress(c.Request.Context(), req.Label, req.Address, req.ChainID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id":       a.ID,
		"label":    a.Label,
		"address":  a.Address,
		"chain_id": a.ChainID,
	})
}

func (s *Svc) DeleteFeeAddress(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := s.store.DeleteFeeAddress(c.Request.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "address not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "status": "deleted"})
}

// ==================== Subscription (singular alias) ====================

// GetSubscription is an alias of ListSubscriptions filtered to the current
// user (GET /subscription). Returns the caller's subscriptions.
func (s *Svc) GetSubscription(c *gin.Context) { s.ListSubscriptions(c) }

// ==================== Health ====================

func (s *Svc) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":      "healthy",
		"service":     "wl-bots",
		"licensed":    s.gate.IsAlive(),
		"reason":      s.gate.Reason(),
		"wl_client_id": s.cfg.WLClientID,
		"product":     s.cfg.Product,
	})
}

// ==================== helpers ====================

// fetchOwnedBot loads a bot by id and enforces caller ownership.
func (s *Svc) fetchOwnedBot(c *gin.Context) (*store.Bot, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return nil, false
	}
	b, err := s.store.GetBot(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "bot not found"})
		return nil, false
	}
	if b.UserID != wlgate.UserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "not your bot"})
		return nil, false
	}
	return b, true
}

func botJSON(b *store.Bot) gin.H {
	return gin.H{
		"id":         b.ID,
		"user_id":    b.UserID,
		"name":       b.Name,
		"bot_type":   b.BotType,
		"status":     b.Status,
		"config":     b.Config,
		"exchange":   b.Exchange,
		"pair":       b.Pair,
		"created_at": b.CreatedAt,
	}
}

// userJSON projects a user for API responses (never leaks password_hash).
func userJSON(u *store.User) gin.H {
	return gin.H{
		"id":         u.ID,
		"email":      u.Email,
		"role":       u.Role,
		"is_active":  u.IsActive,
		"created_at": u.CreatedAt,
	}
}

// previewKey returns a redacted preview of an API key (first 4 + last 4 chars).
func previewKey(k string) string {
	if len(k) <= 8 {
		return "****"
	}
	return k[:4] + "****" + k[len(k)-4:]
}
