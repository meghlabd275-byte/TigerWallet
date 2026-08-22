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
	// Issue the user's assigned scopes (set by the WL client via UpdateUserScopes)
	// in the JWT. wlgate.RequireScope enforces these on admin routes. wl_client is
	// NOT granted by default — only the WL client owner has it assigned. The
	// canonical scope for THIS product is 'listing_admin' (coin/token listing +
	// trading pairs). Legacy local role strings are still honored via RequireRole
	// fallback so existing deployments don't break.
	scopes := u.Scopes
	if len(scopes) == 0 {
		scopes = []string{"user"}
	}
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

// ==================== Discovery (public reads) ====================

// ListCoins is the /coins alias of /tokens — same listed tokens, "coins" shape.
func (h *Handlers) ListCoins(c *gin.Context) {
	status := c.Query("status")
	if status == "" {
		status = "listed"
	}
	ts, err := h.store.ListTokens(c.Request.Context(), status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(ts))
	for _, t := range ts {
		out = append(out, tokenToJSON(&t))
	}
	c.JSON(http.StatusOK, gin.H{"coins": out})
}

// SearchTokens runs a real ILIKE query over name/symbol/contract_address.
func (h *Handlers) SearchTokens(c *gin.Context) {
	q := c.Query("q")
	if q == "" {
		c.JSON(http.StatusOK, gin.H{"tokens": []gin.H{}})
		return
	}
	ts, err := h.store.SearchTokens(c.Request.Context(), q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(ts))
	for _, t := range ts {
		out = append(out, tokenToJSON(&t))
	}
	c.JSON(http.StatusOK, gin.H{"tokens": out, "query": q})
}

// FeaturedTokens returns tokens where is_featured=true (real WHERE).
func (h *Handlers) FeaturedTokens(c *gin.Context) {
	ts, err := h.store.FeaturedTokens(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(ts))
	for _, t := range ts {
		out = append(out, tokenToJSON(&t))
	}
	c.JSON(http.StatusOK, gin.H{"featured": out})
}

// TrendingTokens returns listed tokens ordered by 24h volume (real ORDER BY).
func (h *Handlers) TrendingTokens(c *gin.Context) {
	ts, err := h.store.TrendingTokens(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(ts))
	for _, t := range ts {
		out = append(out, tokenToJSON(&t))
	}
	c.JSON(http.StatusOK, gin.H{"trending": out})
}

// MarketOverview returns a real aggregate: counts, summed volume, top gainers.
func (h *Handlers) MarketOverview(c *gin.Context) {
	m, err := h.store.MarketOverview(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	gainers := make([]gin.H, 0, len(m.TopGainers))
	for _, g := range m.TopGainers {
		gainers = append(gainers, gin.H{"token_id": g.TokenID, "symbol": g.Symbol, "change_24h": g.Change24h})
	}
	c.JSON(http.StatusOK, gin.H{
		"total_tokens": m.TotalTokens, "total_listings": m.TotalListings,
		"total_launchpads": m.TotalLaunch, "total_volume": m.TotalVolume,
		"top_gainers": gainers,
	})
}

// ==================== Token listing workflow ====================

// SubmitToken transitions a token to pending review (set status='pending').
func (h *Handlers) SubmitToken(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.store.SubmitToken(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "token not found or already submitted"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "status": "pending"})
}

// ApproveToken transitions a token to listed (admin gate).
func (h *Handlers) ApproveToken(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.store.ApproveToken(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "token not found or not in a reviewable state"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "status": "listed"})
}

// RejectToken transitions a token to rejected with a reason (admin gate).
func (h *Handlers) RejectToken(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.store.RejectToken(c.Request.Context(), id, req.Reason); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "token not found or not in a reviewable state"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "status": "rejected", "reason": req.Reason})
}

// TokenListingStatus returns the listing-review state for a token.
func (h *Handlers) TokenListingStatus(c *gin.Context) {
	tokenID, ok := parseTokenID(c)
	if !ok {
		return
	}
	ts, err := h.store.GetTokenStatus(c.Request.Context(), tokenID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "token not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id": ts.ID, "status": ts.Status, "is_featured": ts.IsFeatured,
		"submission_date": ts.SubmissionDate, "reviewed_at": ts.ReviewedAt,
		"rejection_reason": ts.RejectionReason,
	})
}

// ToggleFeatured flips is_featured for a token (admin gate).
func (h *Handlers) ToggleFeatured(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	featured, err := h.store.ToggleFeatured(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "token not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "is_featured": featured})
}

// ==================== Launchpad contributions ====================

// Contribute atomically records a contribution and increments sold_amount.
func (h *Handlers) Contribute(c *gin.Context) {
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
	// Pre-validate the launchpad exists (store.CreateContribution also checks
	// active/end_time atomically inside its tx, but a 404 here is friendlier).
	if _, err := h.store.GetLaunchpadProject(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "launchpad project not found"})
		return
	}
	contrib, err := h.store.CreateContribution(c.Request.Context(), id, userID, req.Amount)
	if err != nil {
		if errors.Is(err, store.ErrNotAccepting) {
			c.JSON(http.StatusConflict, gin.H{"error": "launchpad not accepting contributions"})
			return
		}
		if errors.Is(err, store.ErrEnded) {
			c.JSON(http.StatusConflict, gin.H{"error": "launchpad has ended"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, contributionToJSON(contrib))
}

// Claim marks a user's pending contribution as claimed (fail-closed if none).
func (h *Handlers) Claim(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	userID := wlgate.UserID(c)
	if err := h.store.ClaimContribution(c.Request.Context(), id, userID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no claimable contribution found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"project_id": id, "user_id": userID, "status": "claimed"})
}

// CancelContribution marks a user's pending contribution as refunded.
func (h *Handlers) CancelContribution(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	userID := wlgate.UserID(c)
	if err := h.store.CancelContribution(c.Request.Context(), id, userID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no cancellable contribution found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"project_id": id, "user_id": userID, "status": "refunded"})
}

// ContributionHistory returns launchpad contributions for a token (real join).
func (h *Handlers) ContributionHistory(c *gin.Context) {
	tokenID, ok := parseTokenID(c)
	if !ok {
		return
	}
	cs, err := h.store.ListContributionsByToken(c.Request.Context(), tokenID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(cs))
	for _, c2 := range cs {
		out = append(out, contributionToJSON(&c2))
	}
	c.JSON(http.StatusOK, gin.H{"contributions": out})
}

// ==================== Market-maker orders ====================

// ListMMOrders lists market-maker orders (optionally filtered by token_id).
func (h *Handlers) ListMMOrders(c *gin.Context) {
	tokenIDStr := c.Query("token_id")
	var tokenID uuid.UUID
	filter := false
	if tokenIDStr != "" {
		parsed, err := uuid.Parse(tokenIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token_id"})
			return
		}
		tokenID = parsed
		filter = true
	}
	os, err := h.store.ListMMOrders(c.Request.Context(), tokenID, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(os))
	for _, o := range os {
		out = append(out, mmOrderToJSON(&o))
	}
	c.JSON(http.StatusOK, gin.H{"orders": out})
}

// CreateMMOrder creates a market-maker order (real PG insert).
func (h *Handlers) CreateMMOrder(c *gin.Context) {
	var req struct {
		TokenID   string `json:"token_id" binding:"required"`
		Side      string `json:"side" binding:"required"`
		Price     string `json:"price" binding:"required"`
		Quantity  string `json:"quantity" binding:"required"`
		ExpiresAt string `json:"expires_at"`
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
	if req.Side != "buy" && req.Side != "sell" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "side must be buy or sell"})
		return
	}
	var expiresAt *time.Time
	if req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid expires_at (use RFC3339)"})
			return
		}
		expiresAt = &t
	}
	o, err := h.store.CreateMMOrder(c.Request.Context(), &store.MMOrder{
		TokenID: tokenID, Side: req.Side, Price: req.Price,
		Quantity: req.Quantity, Status: "pending", ExpiresAt: expiresAt,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, mmOrderToJSON(o))
}

// UpdateMMOrderStatus updates an order's status (pending/filled/cancelled).
func (h *Handlers) UpdateMMOrderStatus(c *gin.Context) {
	id, ok := parseID(c)
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
	if req.Status != "pending" && req.Status != "filled" && req.Status != "cancelled" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be pending, filled or cancelled"})
		return
	}
	if err := h.store.UpdateMMOrderStatus(c.Request.Context(), id, req.Status); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "status": req.Status})
}

// MarketMakerStatus returns the real MM aggregate for a token.
func (h *Handlers) MarketMakerStatus(c *gin.Context) {
	tokenID, ok := parseTokenID(c)
	if !ok {
		return
	}
	st, err := h.store.MMStatus(c.Request.Context(), tokenID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"token_id": st.TokenID, "total_orders": st.TotalOrders,
		"filled_orders": st.FilledOrders, "buy_high": st.BuyHigh,
		"sell_low": st.SellLow, "spread": st.Spread,
	})
}

// AddLiquidity records a real liquidity position.
func (h *Handlers) AddLiquidity(c *gin.Context) {
	var req struct {
		TokenID    string `json:"token_id" binding:"required"`
		QuoteToken string `json:"quote_token"`
		Amount     string `json:"amount" binding:"required"`
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
	userID := wlgate.UserID(c)
	var uid *uuid.UUID
	if userID != uuid.Nil {
		uid = &userID
	}
	// LP tokens: constant-product proxy = amount * 1000.
	amt, perr := strconv.ParseFloat(req.Amount, 64)
	if perr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount"})
		return
	}
	pos, err := h.store.AddLiquidity(c.Request.Context(), &store.LiquidityPosition{
		TokenID: tokenID, UserID: uid, QuoteToken: req.QuoteToken,
		LPTokens: formatFloat(amt * 1000),
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, liquidityToJSON(pos))
}

// RemoveLiquidity removes the most recent matching liquidity position.
func (h *Handlers) RemoveLiquidity(c *gin.Context) {
	var req struct {
		TokenID  string `json:"token_id" binding:"required"`
		LPAmount string `json:"lp_amount" binding:"required"`
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
	if err := h.store.RemoveLiquidity(c.Request.Context(), tokenID, req.LPAmount); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no matching liquidity position"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"removed": true, "token_id": tokenID})
}

// ==================== Pricing ====================

// SetTokenPrice records a new price point for a token.
func (h *Handlers) SetTokenPrice(c *gin.Context) {
	var req struct {
		TokenID string `json:"token_id" binding:"required"`
		Price   string `json:"price" binding:"required"`
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
	p, err := h.store.SetTokenPrice(c.Request.Context(), tokenID, req.Price)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, tokenPriceToJSON(p))
}

// GetTokenPrice returns the latest price for a token.
func (h *Handlers) GetTokenPrice(c *gin.Context) {
	tokenID, ok := parseTokenID(c)
	if !ok {
		return
	}
	p, err := h.store.GetTokenPrice(c.Request.Context(), tokenID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no price found for token"})
		return
	}
	c.JSON(http.StatusOK, tokenPriceToJSON(p))
}

// PriceHistory returns recent price points for a token.
func (h *Handlers) PriceHistory(c *gin.Context) {
	tokenID, ok := parseTokenID(c)
	if !ok {
		return
	}
	ps, err := h.store.ListTokenPriceHistory(c.Request.Context(), tokenID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(ps))
	for _, p := range ps {
		out = append(out, tokenPriceToJSON(&p))
	}
	c.JSON(http.StatusOK, gin.H{"prices": out})
}

// UpdatePrice is the alias form of SetTokenPrice (admin gate).
func (h *Handlers) UpdatePrice(c *gin.Context) {
	h.SetTokenPrice(c)
}

// ==================== Analytics ====================

// VolumeStats returns real SUM(volume_24h) over rolling windows.
func (h *Handlers) VolumeStats(c *gin.Context) {
	v, err := h.store.VolumeStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"volume_24h": v.Total24h, "volume_7d": v.Total7d, "volume_30d": v.Total30d})
}

// LiquidityState returns the real total liquidity across all positions.
func (h *Handlers) LiquidityState(c *gin.Context) {
	total, err := h.store.TotalLiquidity(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"total_liquidity": total})
}

// HolderCount returns the real distinct-contributor count for a token.
func (h *Handlers) HolderCount(c *gin.Context) {
	tokenID, ok := parseTokenID(c)
	if !ok {
		return
	}
	count, err := h.store.HolderCount(c.Request.Context(), tokenID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token_id": tokenID, "holders": count})
}

// TransactionCount returns real contribution counts over rolling windows.
func (h *Handlers) TransactionCount(c *gin.Context) {
	ts, err := h.store.TransactionStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"transactions_24h": ts.Total24h, "transactions_7d": ts.Total7d})
}

// ==================== Fees ====================

// ListFees returns the real fee_schedule rows.
func (h *Handlers) ListFees(c *gin.Context) {
	sched, err := h.store.FeeSchedule(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"fees": sched})
}

// CalculateFees computes the real fee for a listing from fee_schedule.
func (h *Handlers) CalculateFees(c *gin.Context) {
	var req struct {
		FeeType string `json:"fee_type" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sched, err := h.store.FeeSchedule(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	amount, ok := sched[req.FeeType]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "unknown fee_type", "fee_type": req.FeeType})
		return
	}
	c.JSON(http.StatusOK, gin.H{"fee_type": req.FeeType, "amount": amount, "currency": "USD"})
}

// PayFees records a real fee payment with its on-chain tx_hash.
func (h *Handlers) PayFees(c *gin.Context) {
	var req struct {
		TokenID       string `json:"token_id"`
		Amount        string `json:"amount" binding:"required"`
		Currency      string `json:"currency"`
		PaymentMethod string `json:"payment_method"`
		TxHash        string `json:"tx_hash" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var tokenID *uuid.UUID
	if req.TokenID != "" {
		parsed, err := uuid.Parse(req.TokenID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token_id"})
			return
		}
		tokenID = &parsed
	}
	userID := wlgate.UserID(c)
	var uid *uuid.UUID
	if userID != uuid.Nil {
		uid = &userID
	}
	currency := req.Currency
	if currency == "" {
		currency = "USD"
	}
	p, err := h.store.RecordFeePayment(c.Request.Context(), &store.FeePayment{
		TokenID: tokenID, UserID: uid, Amount: req.Amount, Currency: currency,
		PaymentMethod: req.PaymentMethod, TxHash: req.TxHash, Status: "completed",
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, feePaymentToJSON(p))
}

// SetFeeConfig upserts a fee_schedule row (admin gate).
func (h *Handlers) SetFeeConfig(c *gin.Context) {
	var req struct {
		FeeType string  `json:"fee_type" binding:"required"`
		Amount  float64 `json:"amount"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.store.SetFeeConfig(c.Request.Context(), req.FeeType, req.Amount); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"fee_type": req.FeeType, "amount": req.Amount})
}

// UpdateFeeConfig is the alias form of SetFeeConfig (admin gate).
func (h *Handlers) UpdateFeeConfig(c *gin.Context) {
	h.SetFeeConfig(c)
}

// ==================== Compliance: audit + KYC ====================

// CreateAuditLog creates an audit entry for a token (admin gate).
func (h *Handlers) CreateAuditLog(c *gin.Context) {
	var req struct {
		TokenID   string `json:"token_id" binding:"required"`
		AuditType string `json:"audit_type" binding:"required"`
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
	a, err := h.store.CreateAuditLog(c.Request.Context(), tokenID, req.AuditType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, auditLogToJSON(a))
}

// AuditStatus returns the audit log for a token (real PG).
func (h *Handlers) AuditStatus(c *gin.Context) {
	tokenID, ok := parseTokenID(c)
	if !ok {
		return
	}
	as, err := h.store.ListAuditLogs(c.Request.Context(), tokenID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(as))
	for _, a := range as {
		out = append(out, auditLogToJSON(&a))
	}
	c.JSON(http.StatusOK, gin.H{"audit_logs": out})
}

// SubmitKYC submits a KYC record for a token listing.
func (h *Handlers) SubmitKYC(c *gin.Context) {
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
	k, err := h.store.SubmitKYC(c.Request.Context(), tokenID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, kycToJSON(k))
}

// KYCStatus returns the KYC record for a token listing.
func (h *Handlers) KYCStatus(c *gin.Context) {
	tokenID, ok := parseTokenID(c)
	if !ok {
		return
	}
	k, err := h.store.GetKYC(c.Request.Context(), tokenID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no kyc record for token"})
		return
	}
	c.JSON(http.StatusOK, kycToJSON(k))
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

// RequireRole is a gin middleware that gates a route on the caller's admin
// privileges. It now prefers the canonical scoped-role taxonomy carried in the
// JWT (set via UpdateAdminScopes by the WL client owner): 'wl_client' (the WL
// owner) always passes, then the canonical scope for THIS product —
// 'listing_admin' (coin/token listing + trading pairs). The legacy local-role
// DB check (users.role ∈ allowed) is kept as a fallback so existing deployments
// don't break while the WL client migrates to the canonical taxonomy. Requires
// JWTAuth to have run first (sets user_id + scopes).
func (h *Handlers) RequireRole(allowed ...string) gin.HandlerFunc {
	allow := make(map[string]struct{}, len(allowed))
	for _, r := range allowed {
		allow[r] = struct{}{}
	}
	return func(c *gin.Context) {
		uid := wlgate.UserID(c)
		if uid == uuid.Nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			c.Abort()
			return
		}
		// wl_client (WL owner) always passes — full tenancy control.
		if wlgate.HasScope(c, "wl_client") {
			c.Set("role", "wl_client")
			c.Next()
			return
		}
		// Canonical scope: listing_admin controls all token listing/trading pairs.
		if wlgate.HasScope(c, "listing_admin") {
			c.Set("role", "listing_admin")
			c.Next()
			return
		}
		// Legacy local-role fallback: load the user's role from the DB + match.
		u, err := h.store.GetUserByID(c.Request.Context(), uid)
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "user not found"})
			c.Abort()
			return
		}
		if _, ok := allow[u.Role]; !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "admin role required"})
			c.Abort()
			return
		}
		c.Set("role", u.Role)
		c.Next()
	}
}

// adminScopeWhitelist is the canonical scoped-role taxonomy (from
// white_label_admin/go/internal/roles). UpdateAdminScopes validates every
// requested scope against this set.
var adminScopeWhitelist = map[string]bool{
	"wl_client": true, "trading_admin": true, "p2p_admin": true, "bot_admin": true,
	"listing_admin": true, "liquidity_admin": true, "wallet_admin": true,
	"customer_service_admin": true, "marketing_admin": true, "kyc_admin": true,
	"card_admin": true, "reward_admin": true, "security_admin": true,
	"compliance_admin": true, "user": true,
}

// UpdateAdminScopes is the WL-client-facing endpoint to grant/revoke scoped
// admin roles on a project-party user. Mirrors white_label_admin
// AssignAdminRole. Only a caller holding 'wl_client' (the WL owner) may set
// scopes — a listing_admin cannot escalate themselves or others. The scopes
// MUST be in the canonical whitelist (validated server-side). wl_project_party
// has no user-level audit-event table, so the change is simply persisted.
func (h *Handlers) UpdateAdminScopes(c *gin.Context) {
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
	for _, sc := range req.Scopes {
		if !adminScopeWhitelist[sc] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scope: " + sc})
			return
		}
	}
	if err := h.store.UpdateUserScopes(c.Request.Context(), id, req.Scopes); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user_id": id, "scopes": req.Scopes})
}

func parseID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return uuid.Nil, false
	}
	return id, true
}

// parseTokenID reads the :token_id path param used by status/history/pricing/
// analytics/compliance routes.
func parseTokenID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("token_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token_id"})
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

func contributionToJSON(c *store.LaunchpadContribution) gin.H {
	return gin.H{
		"id": c.ID, "project_id": c.ProjectID, "user_id": c.UserID,
		"amount": c.Amount, "token_amount": c.TokenAmount, "status": c.Status,
		"claimed_at": c.ClaimedAt, "refunded_at": c.RefundedAt,
		"created_at": c.CreatedAt,
	}
}

func mmOrderToJSON(o *store.MMOrder) gin.H {
	return gin.H{
		"id": o.ID, "token_id": o.TokenID, "side": o.Side,
		"price": o.Price, "quantity": o.Quantity, "remaining": o.Remaining,
		"status": o.Status, "filled_at": o.FilledAt, "expires_at": o.ExpiresAt,
		"created_at": o.CreatedAt,
	}
}

func liquidityToJSON(p *store.LiquidityPosition) gin.H {
	return gin.H{
		"id": p.ID, "token_id": p.TokenID, "user_id": p.UserID,
		"quote_token": p.QuoteToken, "lp_tokens": p.LPTokens,
		"created_at": p.CreatedAt,
	}
}

func tokenPriceToJSON(p *store.TokenPrice) gin.H {
	return gin.H{
		"id": p.ID, "token_id": p.TokenID, "price": p.Price,
		"change_24h": p.Change24h, "volume_24h": p.Volume24h,
		"timestamp": p.Timestamp,
	}
}

func feePaymentToJSON(p *store.FeePayment) gin.H {
	return gin.H{
		"id": p.ID, "token_id": p.TokenID, "user_id": p.UserID,
		"amount": p.Amount, "currency": p.Currency,
		"payment_method": p.PaymentMethod, "tx_hash": p.TxHash,
		"status": p.Status, "created_at": p.CreatedAt,
	}
}

func auditLogToJSON(a *store.AuditLog) gin.H {
	return gin.H{
		"id": a.ID, "token_id": a.TokenID, "audit_type": a.AuditType,
		"status": a.Status, "report_url": a.ReportURL, "auditor": a.Auditor,
		"completed_at": a.CompletedAt, "requested_at": a.RequestedAt,
	}
}

func kycToJSON(k *store.KYCRecord) gin.H {
	return gin.H{
		"id": k.ID, "token_id": k.TokenID, "status": k.Status,
		"submitted_at": k.SubmittedAt, "expires_at": k.ExpiresAt,
		"reviewed_at": k.ReviewedAt, "created_at": k.CreatedAt,
	}
}

// formatFloat renders a float as a plain decimal string without scientific
// notation, so it stores cleanly into NUMERIC columns.
func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
