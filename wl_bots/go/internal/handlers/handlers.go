// Package handlers implements the standalone WL-Bots backend REST API. A clone
// of the TigerWallet mm_bot_platform/bot_api bot management platform — full bot
// lifecycle (create/start/stop/pause/delete), executions, subscriptions, fee
// configs, and per-user encrypted API keys. REAL bcrypt + JWT auth, REAL
// AES-GCM at-rest encryption (wlcrypto), REAL PostgreSQL persistence, and a
// fail-closed license gate (wlgate). No stubs, no fakes, no mocks, no demos.
package handlers

import (
	"net/http"
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
	if err != nil || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	wlID, err := uuid.Parse(s.cfg.WLClientID)
	if err != nil {
		wlID = uuid.Nil
	}
	scopes := []string{"wl_client", "bots"}
	tok, err := wlgate.IssueJWT(s.cfg.JWTSecret, u.ID, u.Email, wlID, scopes, s.cfg.JWTExpiry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token issue failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": tok, "user_id": u.ID, "email": u.Email, "role": u.Role})
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
	s.transitionBot(c, "running", "start requested")
}

func (s *Svc) StopBot(c *gin.Context) {
	s.transitionBot(c, "stopped", "stop requested")
}

func (s *Svc) PauseBot(c *gin.Context) {
	s.transitionBot(c, "paused", "pause requested")
}

func (s *Svc) transitionBot(c *gin.Context, status, logMsg string) {
	b, ok := s.fetchOwnedBot(c)
	if !ok {
		return
	}
	if err := s.store.SetBotStatus(c.Request.Context(), b.ID, status); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "bot not found"})
		return
	}
	_ = s.store.AppendBotLog(c.Request.Context(), b.ID, "info", logMsg)
	b.Status = status
	c.JSON(http.StatusOK, botJSON(b))
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

// previewKey returns a redacted preview of an API key (first 4 + last 4 chars).
func previewKey(k string) string {
	if len(k) <= 8 {
		return "****"
	}
	return k[:4] + "****" + k[len(k)-4:]
}
