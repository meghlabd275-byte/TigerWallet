// Package handlers implements the standalone WL-ProjectParty backend REST API.
// REAL PostgreSQL persistence + fail-closed license gate (wlgate) + real
// bcrypt/JWT auth. A clone of the TigerWallet project_party token-listing /
// launchpad platform: tokens, listings, launchpad, participations,
// market-making configs, fee configs, favorites. No stubs, no fakes, no
// TigerWallet cloud dependency at request time.
package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tigerwallet/wl-project-party/internal/config"
	"github.com/tigerwallet/wl-project-party/internal/store"
	"github.com/tigerwallet/wl-shared/wlgate"
	"golang.org/x/crypto/bcrypt"
)

// Handlers serves the WL-ProjectParty REST API. It embeds the license gate
// (fail-closed) so no protected request is served without a valid license.
type Handlers struct {
	cfg        *config.Config
	store      *store.Store
	gate       *wlgate.Gate
	wlClientID uuid.UUID
}

// New builds Handlers bound to a fail-closed license gate.
func New(cfg *config.Config, st *store.Store, gate *wlgate.Gate) *Handlers {
	wlID, err := uuid.Parse(cfg.WLClientID)
	if err != nil {
		wlID = uuid.Nil
	}
	return &Handlers{
		cfg:        cfg,
		store:      st,
		gate:       gate,
		wlClientID: wlID,
	}
}

// ==================== Auth (real bcrypt + JWT via wlgate.IssueJWT) ====================

func (h *Handlers) Register(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=8"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), h.cfg.BCryptCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "hash failed"})
		return
	}
	u, err := h.store.CreateUser(c.Request.Context(), req.Email, string(hash), req.Role)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": u.ID, "email": u.Email, "role": u.Role})
}

func (h *Handlers) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	u, err := h.store.GetUserByEmail(c.Request.Context(), req.Email)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	scopes := []string{"project_party"}
	tok, err := wlgate.IssueJWT(h.cfg.JWTSecret, u.ID, u.Email, h.wlClientID, scopes, h.cfg.JWTExpiry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token issue failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": tok, "user_id": u.ID, "email": u.Email, "wl_client_id": h.cfg.WLClientID})
}

// ==================== Tokens ====================

func (h *Handlers) CreateToken(c *gin.Context) {
	var req struct {
		Name            string `json:"name" binding:"required"`
		Symbol          string `json:"symbol" binding:"required"`
		ContractAddress string `json:"contract_address"`
		ChainID         int64  `json:"chain_id"`
		Decimals        int    `json:"decimals"`
		LogoURL         string `json:"logo_url"`
		Description     string `json:"description"`
		Website         string `json:"website"`
		Status          string `json:"status"`
		ListingType     string `json:"listing_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.ChainID == 0 {
		req.ChainID = 1
	}
	t, err := h.store.CreateToken(c.Request.Context(), &store.Token{
		Name: req.Name, Symbol: req.Symbol, ContractAddress: req.ContractAddress,
		ChainID: req.ChainID, Decimals: req.Decimals, LogoURL: req.LogoURL,
		Description: req.Description, Website: req.Website, Status: req.Status,
		ListingType: req.ListingType,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, tokenToJSON(t))
}

func (h *Handlers) ListTokens(c *gin.Context) {
	status := c.Query("status")
	ts, err := h.store.ListTokens(c.Request.Context(), status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(ts))
	for _, t := range ts {
		out = append(out, tokenToJSON(&t))
	}
	c.JSON(http.StatusOK, gin.H{"tokens": out})
}

func (h *Handlers) GetToken(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	t, err := h.store.GetToken(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "token not found"})
		return
	}
	c.JSON(http.StatusOK, tokenToJSON(t))
}

func (h *Handlers) UpdateToken(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		Name            string `json:"name" binding:"required"`
		Symbol          string `json:"symbol" binding:"required"`
		ContractAddress string `json:"contract_address"`
		ChainID         int64  `json:"chain_id"`
		Decimals        int    `json:"decimals"`
		LogoURL         string `json:"logo_url"`
		Description     string `json:"description"`
		Website         string `json:"website"`
		Status          string `json:"status"`
		ListingType     string `json:"listing_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	t, err := h.store.UpdateToken(c.Request.Context(), id, &store.Token{
		Name: req.Name, Symbol: req.Symbol, ContractAddress: req.ContractAddress,
		ChainID: req.ChainID, Decimals: req.Decimals, LogoURL: req.LogoURL,
		Description: req.Description, Website: req.Website, Status: req.Status,
		ListingType: req.ListingType,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "token not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tokenToJSON(t))
}

func (h *Handlers) DeleteToken(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.store.DeleteToken(c.Request.Context(), id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "token not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": id})
}

// ==================== Listings ====================

func (h *Handlers) CreateListing(c *gin.Context) {
	var req struct {
		TokenID    string `json:"token_id" binding:"required"`
		Pair       string `json:"pair" binding:"required"`
		BaseToken  string `json:"base_token"`
		QuoteToken string `json:"quote_token"`
		Status     string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	tokenID, err := uuid.Parse(req.TokenID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token_id"})
		return
	}
	l, err := h.store.CreateListing(c.Request.Context(), &store.Listing{
		TokenID: tokenID, Pair: req.Pair, BaseToken: req.BaseToken,
		QuoteToken: req.QuoteToken, Status: req.Status,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, listingToJSON(l))
}

func (h *Handlers) ListListings(c *gin.Context) {
	status := c.Query("status")
	ls, err := h.store.ListListings(c.Request.Context(), status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(ls))
	for _, l := range ls {
		out = append(out, listingToJSON(&l))
	}
	c.JSON(http.StatusOK, gin.H{"listings": out})
}

// ==================== Launchpad ====================

func (h *Handlers) CreateLaunchpadProject(c *gin.Context) {
	var req struct {
		TokenID       string  `json:"token_id" binding:"required"`
		Name          string  `json:"name" binding:"required"`
		Description   string  `json:"description"`
		StartTime     *string `json:"start_time"`
		EndTime       *string `json:"end_time"`
		TotalSupply   string  `json:"total_supply"`
		PricePerToken string  `json:"price_per_token"`
		Status        string  `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	tokenID, err := uuid.Parse(req.TokenID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token_id"})
		return
	}
	var startT, endT *time.Time
	if req.StartTime != nil && *req.StartTime != "" {
		t, err := time.Parse(time.RFC3339, *req.StartTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_time (use RFC3339)"})
			return
		}
		startT = &t
	}
	if req.EndTime != nil && *req.EndTime != "" {
		t, err := time.Parse(time.RFC3339, *req.EndTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_time (use RFC3339)"})
			return
		}
		endT = &t
	}
	p, err := h.store.CreateLaunchpadProject(c.Request.Context(), &store.LaunchpadProject{
		TokenID: tokenID, Name: req.Name, Description: req.Description,
		StartTime: startT, EndTime: endT, TotalSupply: req.TotalSupply,
		PricePerToken: req.PricePerToken, Status: req.Status,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, launchpadToJSON(p))
}

func (h *Handlers) ListLaunchpadProjects(c *gin.Context) {
	status := c.Query("status")
	ps, err := h.store.ListLaunchpadProjects(c.Request.Context(), status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(ps))
	for _, p := range ps {
		out = append(out, launchpadToJSON(&p))
	}
	c.JSON(http.StatusOK, gin.H{"launchpad_projects": out})
}

func (h *Handlers) GetLaunchpadProject(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	p, err := h.store.GetLaunchpadProject(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "launchpad project not found"})
		return
	}
	c.JSON(http.StatusOK, launchpadToJSON(p))
}

func (h *Handlers) ParticipateInLaunchpad(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	userID := wlgate.UserID(c)
	var req struct {
		Amount string `json:"amount" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Validate the launchpad exists and is active.
	p, err := h.store.GetLaunchpadProject(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "launchpad project not found"})
		return
	}
	if p.Status != "active" && p.Status != "upcoming" {
		c.JSON(http.StatusConflict, gin.H{"error": "launchpad not accepting participations", "status": p.Status})
		return
	}
	// Enforce end_time if set.
	if p.EndTime != nil && time.Now().After(*p.EndTime) {
		c.JSON(http.StatusConflict, gin.H{"error": "launchpad has ended"})
		return
	}
	part, err := h.store.CreateParticipation(c.Request.Context(), id, userID, req.Amount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, participationToJSON(part))
}

func (h *Handlers) ListParticipations(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	ps, err := h.store.ListParticipations(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(ps))
	for _, p := range ps {
		out = append(out, participationToJSON(&p))
	}
	c.JSON(http.StatusOK, gin.H{"participations": out})
}

// ==================== Market-making configs ====================

func (h *Handlers) CreateMarketMakingConfig(c *gin.Context) {
	var req struct {
		TokenID string `json:"token_id" binding:"required"`
		Pair    string `json:"pair" binding:"required"`
		Spread  string `json:"spread"`
		Enabled *bool  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	tokenID, err := uuid.Parse(req.TokenID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token_id"})
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	m, err := h.store.CreateMarketMakingConfig(c.Request.Context(), &store.MarketMakingConfig{
		TokenID: tokenID, Pair: req.Pair, Spread: req.Spread, Enabled: enabled,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, marketMakingToJSON(m))
}

func (h *Handlers) ListMarketMakingConfigs(c *gin.Context) {
	ms, err := h.store.ListMarketMakingConfigs(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(ms))
	for _, m := range ms {
		out = append(out, marketMakingToJSON(&m))
	}
	c.JSON(http.StatusOK, gin.H{"market_making_configs": out})
}

// ==================== Fee configs ====================

func (h *Handlers) CreateFeeConfig(c *gin.Context) {
	var req struct {
		Name       string  `json:"name" binding:"required"`
		Percentage float64 `json:"percentage"`
		Enabled    *bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	// Store as a string to preserve precision; NUMERIC(10,4).
	percentage := formatFloat(req.Percentage)
	f, err := h.store.CreateFeeConfig(c.Request.Context(), req.Name, percentage, enabled)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, feeConfigToJSON(f))
}

func (h *Handlers) ListFeeConfigs(c *gin.Context) {
	fs, err := h.store.ListFeeConfigs(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(fs))
	for _, f := range fs {
		out = append(out, feeConfigToJSON(&f))
	}
	c.JSON(http.StatusOK, gin.H{"fee_configs": out})
}

// ==================== Favorites ====================

func (h *Handlers) AddFavorite(c *gin.Context) {
	userID := wlgate.UserID(c)
	var req struct {
		TokenID string `json:"token_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	tokenID, err := uuid.Parse(req.TokenID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token_id"})
		return
	}
	f, err := h.store.AddFavorite(c.Request.Context(), userID, tokenID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, favoriteToJSON(f))
}

func (h *Handlers) ListFavorites(c *gin.Context) {
	userID := wlgate.UserID(c)
	fs, err := h.store.ListFavorites(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(fs))
	for _, f := range fs {
		out = append(out, favoriteToJSON(&f))
	}
	c.JSON(http.StatusOK, gin.H{"favorites": out})
}

func (h *Handlers) RemoveFavorite(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	userID := wlgate.UserID(c)
	if err := h.store.RemoveFavorite(c.Request.Context(), id, userID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "favorite not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": id})
}

// ==================== Health ====================

func (h *Handlers) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":       "healthy",
		"service":      "wl-project-party",
		"licensed":     h.gate.IsAlive(),
		"reason":       h.gate.Reason(),
		"wl_client_id": h.cfg.WLClientID,
		"product":      h.cfg.Product,
		"instance_id":  h.cfg.InstanceID,
	})
}

// ==================== helpers ====================

func parseID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return uuid.Nil, false
	}
	return id, true
}

func tokenToJSON(t *store.Token) gin.H {
	return gin.H{
		"id": t.ID, "name": t.Name, "symbol": t.Symbol,
		"contract_address": t.ContractAddress, "chain_id": t.ChainID,
		"decimals": t.Decimals, "logo_url": t.LogoURL, "description": t.Description,
		"website": t.Website, "status": t.Status, "listing_type": t.ListingType,
		"created_at": t.CreatedAt,
	}
}

func listingToJSON(l *store.Listing) gin.H {
	return gin.H{
		"id": l.ID, "token_id": l.TokenID, "pair": l.Pair,
		"base_token": l.BaseToken, "quote_token": l.QuoteToken,
		"status": l.Status, "created_at": l.CreatedAt,
	}
}

func launchpadToJSON(p *store.LaunchpadProject) gin.H {
	return gin.H{
		"id": p.ID, "token_id": p.TokenID, "name": p.Name,
		"description": p.Description, "start_time": p.StartTime,
		"end_time": p.EndTime, "total_supply": p.TotalSupply,
		"sold_amount": p.SoldAmount, "price_per_token": p.PricePerToken,
		"status": p.Status, "created_at": p.CreatedAt,
	}
}

func participationToJSON(p *store.Participation) gin.H {
	return gin.H{
		"id": p.ID, "project_id": p.ProjectID, "user_id": p.UserID,
		"amount": p.Amount, "allocated": p.Allocated, "status": p.Status,
		"created_at": p.CreatedAt,
	}
}

func marketMakingToJSON(m *store.MarketMakingConfig) gin.H {
	return gin.H{
		"id": m.ID, "token_id": m.TokenID, "pair": m.Pair,
		"spread": m.Spread, "enabled": m.Enabled, "created_at": m.CreatedAt,
	}
}

func feeConfigToJSON(f *store.FeeConfig) gin.H {
	return gin.H{
		"id": f.ID, "name": f.Name, "percentage": f.Percentage,
		"enabled": f.Enabled, "created_at": f.CreatedAt,
	}
}

func favoriteToJSON(f *store.Favorite) gin.H {
	return gin.H{
		"id": f.ID, "user_id": f.UserID, "token_id": f.TokenID,
		"created_at": f.CreatedAt,
	}
}

// formatFloat renders a float as a plain decimal string without scientific
// notation, so it stores cleanly into NUMERIC columns.
func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
