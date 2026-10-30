package main

// handlers_markets.go — market, trade and position handlers backed by the
// store, with a short Redis cache in front of the read paths.
//
// Nothing here is synthesised: a market list is what the providers returned, a
// trade is a row that was written, and a price is the last observed quote. When
// the database is unavailable the endpoints report that rather than serving an
// empty list that reads as "no markets exist".

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// marketCacheTTL bounds how stale a market list may be.
const marketCacheTTL = 30 * time.Second

// outcomesCacheTTL bounds how stale an outcome quote may be.
const outcomesCacheTTL = 10 * time.Second

func getMarkets(c *gin.Context) {
	ctx := c.Request.Context()

	source := c.Query("source")
	status := c.Query("status")
	limit := getIntParam(c, "limit", 50)
	offset := getIntParam(c, "offset", 0)

	cacheKey := fmt.Sprintf("markets:%s:%s:%d:%d", source, status, limit, offset)
	var cached []Market
	if cacheGet(ctx, cacheKey, &cached) {
		c.JSON(http.StatusOK, cached)
		return
	}

	markets, err := store.ListMarkets(ctx, source, status, limit, offset)
	if err != nil {
		logger.Error().Err(err).Msg("list markets failed")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "markets are unavailable"})
		return
	}

	cacheSet(ctx, cacheKey, markets, marketCacheTTL)
	c.JSON(http.StatusOK, markets)
}

func getMarket(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	cacheKey := "market:" + id
	var cached Market
	if cacheGet(ctx, cacheKey, &cached) {
		c.JSON(http.StatusOK, cached)
		return
	}

	market, err := store.GetMarket(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Market not found"})
			return
		}
		logger.Error().Err(err).Str("marketId", id).Msg("get market failed")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "the market is unavailable"})
		return
	}

	cacheSet(ctx, cacheKey, market, marketCacheTTL)
	c.JSON(http.StatusOK, market)
}

func getOutcomes(c *gin.Context) {
	ctx := c.Request.Context()
	marketID := c.Param("id")

	cacheKey := "outcomes:" + marketID
	var cached []Outcome
	if cacheGet(ctx, cacheKey, &cached) {
		c.JSON(http.StatusOK, cached)
		return
	}

	outcomes, err := store.ListOutcomes(ctx, marketID)
	if err != nil {
		logger.Error().Err(err).Str("marketId", marketID).Msg("list outcomes failed")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "outcomes are unavailable"})
		return
	}

	cacheSet(ctx, cacheKey, outcomes, outcomesCacheTTL)
	c.JSON(http.StatusOK, outcomes)
}

func getTrendingMarkets(c *gin.Context) {
	ctx := c.Request.Context()
	limit := getIntParam(c, "limit", 10)

	markets, err := store.ListTrendingMarkets(ctx, limit)
	if err != nil {
		logger.Error().Err(err).Msg("list trending markets failed")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "markets are unavailable"})
		return
	}
	c.JSON(http.StatusOK, markets)
}

func getRecentMarkets(c *gin.Context) {
	ctx := c.Request.Context()
	limit := getIntParam(c, "limit", 20)

	markets, err := store.ListRecentMarkets(ctx, limit)
	if err != nil {
		logger.Error().Err(err).Msg("list recent markets failed")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "markets are unavailable"})
		return
	}
	c.JSON(http.StatusOK, markets)
}

// getCurrentPrice returns the last observed price for an outcome.
//
// The price is the probability the market itself quotes, which is the price a
// fill can actually be expected at. It is never defaulted: an unknown outcome
// has no price, and the caller must refuse the trade rather than assume one.
func getCurrentPrice(ctx context.Context, marketID, outcomeID string) (float64, error) {
	outcomes, err := store.ListOutcomes(ctx, marketID)
	if err != nil {
		return 0, err
	}
	for _, o := range outcomes {
		if o.ID == outcomeID {
			return o.Price, nil
		}
	}
	return 0, fmt.Errorf("outcome %s is not part of market %s", outcomeID, marketID)
}

func placeTrade(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		// UserID is optional. When present it is checked against the token, so
		// naming another account is an error rather than an override.
		UserID    string  `json:"userId"`
		MarketID  string  `json:"marketId" binding:"required"`
		OutcomeID string  `json:"outcomeId" binding:"required"`
		Side      string  `json:"side" binding:"required,oneof=yes no"`
		Amount    float64 `json:"amount" binding:"required,gt=0"`
		ChainID   uint64  `json:"chainId"`
		TxHash    string  `json:"txHash"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, ok := resolveUser(c, req.UserID)
	if !ok {
		return
	}

	// The market must exist and still be open.
	market, err := store.GetMarket(ctx, req.MarketID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Market not found"})
			return
		}
		logger.Error().Err(err).Msg("lookup market failed")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "the market is unavailable"})
		return
	}
	if market.Closed || market.Resolved {
		c.JSON(http.StatusConflict, gin.H{"error": "Market is closed"})
		return
	}

	price, err := getCurrentPrice(ctx, req.MarketID, req.OutcomeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if price <= 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Outcome has no quoted price"})
		return
	}

	// A trade is confirmed only when a transaction hash is supplied. Without
	// one the trade is recorded as pending, because nothing has settled it.
	status := "pending"
	if req.TxHash != "" {
		status = "confirmed"
	}

	trade := Trade{
		ID:        generateID(),
		UserID:    userID,
		MarketID:  req.MarketID,
		OutcomeID: req.OutcomeID,
		Side:      req.Side,
		Amount:    req.Amount,
		Price:     price,
		Total:     req.Amount * price,
		ChainID:   req.ChainID,
		TxHash:    req.TxHash,
		Status:    status,
		CreatedAt: time.Now().Unix(),
	}

	if err := store.RecordTrade(ctx, trade); err != nil {
		if errors.Is(err, ErrDuplicateTrade) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		logger.Error().Err(err).Str("tradeId", trade.ID).Msg("record trade failed")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "the trade could not be recorded"})
		return
	}

	wsHub.BroadcastMarketUpdate(trade.MarketID, map[string]interface{}{
		"type":   "trade",
		"trade":  trade,
		"price":  price,
		"volume": trade.Total,
	})

	c.JSON(http.StatusOK, trade)
}

func getTradeHistory(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := resolveUser(c, c.Param("userId"))
	if !ok {
		return
	}
	limit := getIntParam(c, "limit", 50)

	trades, err := store.ListTrades(ctx, userID, limit)
	if err != nil {
		logger.Error().Err(err).Str("userId", userID).Msg("list trades failed")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "trade history is unavailable"})
		return
	}
	c.JSON(http.StatusOK, trades)
}

func getPositions(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := resolveUser(c, c.Param("userId"))
	if !ok {
		return
	}

	positions, err := store.ListPositions(ctx, userID)
	if err != nil {
		logger.Error().Err(err).Str("userId", userID).Msg("list positions failed")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "positions are unavailable"})
		return
	}
	c.JSON(http.StatusOK, positions)
}

func getMarketPrices(c *gin.Context) {
	ctx := c.Request.Context()
	marketID := c.Param("marketId")

	outcomes, err := store.ListOutcomes(ctx, marketID)
	if err != nil {
		logger.Error().Err(err).Str("marketId", marketID).Msg("list prices failed")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "prices are unavailable"})
		return
	}

	prices := make(map[string]float64, len(outcomes))
	for _, o := range outcomes {
		prices[o.ID] = o.Price
	}
	c.JSON(http.StatusOK, prices)
}

// getVolumeAnalytics reports traded volume over a period.
func getVolumeAnalytics(c *gin.Context) {
	ctx := c.Request.Context()
	period := c.DefaultQuery("period", "24h")

	since, err := periodStart(period)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var trades int64
	var volume, fees float64
	err = store.pool.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(SUM(total), 0)::float8, COALESCE(SUM(total), 0)::float8
		FROM prediction_trades
		WHERE created_at >= $1 AND status = 'confirmed'`, since).
		Scan(&trades, &volume, &fees)
	if err != nil {
		logger.Error().Err(err).Msg("volume analytics failed")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "analytics are unavailable"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"period":    period,
		"trades":    trades,
		"volumeUSD": volume,
		"feesUSD":   fees,
		"since":     since,
	})
}

// getUserAnalytics reports a user's trading activity.
func getUserAnalytics(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := resolveUser(c, c.Param("userId"))
	if !ok {
		return
	}

	var trades int64
	var volume, realised float64
	err := store.pool.QueryRow(ctx, `
		SELECT COUNT(*),
		       COALESCE(SUM(total), 0)::float8,
		       COALESCE(SUM(CASE WHEN side = 'no' THEN total ELSE 0 END), 0)::float8
		FROM prediction_trades
		WHERE user_id = $1 AND status = 'confirmed'`, userID).
		Scan(&trades, &volume, &realised)
	if err != nil {
		logger.Error().Err(err).Str("userId", userID).Msg("user analytics failed")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "analytics are unavailable"})
		return
	}

	positions, err := store.ListPositions(ctx, userID)
	if err != nil {
		logger.Error().Err(err).Str("userId", userID).Msg("user positions failed")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "analytics are unavailable"})
		return
	}

	var unrealised float64
	for _, p := range positions {
		unrealised += p.ProfitLoss
	}

	c.JSON(http.StatusOK, gin.H{
		"userId":        userID,
		"trades":        trades,
		"volumeUSD":     volume,
		"noVolumeUSD":   realised,
		"openPositions": len(positions),
		"unrealisedPnl": unrealised,
	})
}

// periodStart converts a period label to a unix start time.
func periodStart(period string) (int64, error) {
	now := time.Now()
	switch period {
	case "24h":
		return now.Add(-24 * time.Hour).Unix(), nil
	case "7d":
		return now.Add(-7 * 24 * time.Hour).Unix(), nil
	case "30d":
		return now.Add(-30 * 24 * time.Hour).Unix(), nil
	case "all":
		return 0, nil
	default:
		return 0, fmt.Errorf("unknown period %q; expected 24h, 7d, 30d or all", period)
	}
}

// streamPrices sends the current outcome prices to a subscriber.
func streamPrices(c *gin.Context) {
	ctx := c.Request.Context()
	marketID := c.Param("marketId")

	interval := 5 * time.Second
	if raw := c.Query("intervalMs"); raw != "" {
		if ms, err := strconv.Atoi(raw); err == nil && ms >= 500 && ms <= 60000 {
			interval = time.Duration(ms) * time.Millisecond
		}
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-store")
	c.Header("Connection", "keep-alive")

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		outcomes, err := store.ListOutcomes(ctx, marketID)
		if err != nil {
			logger.Error().Err(err).Str("marketId", marketID).Msg("price stream failed")
			return
		}

		prices := make(map[string]float64, len(outcomes))
		for _, o := range outcomes {
			prices[o.ID] = o.Price
		}

		payload := mustJSON(map[string]interface{}{
			"marketId": marketID,
			"prices":   prices,
			"ts":       time.Now().Unix(),
		})
		fmt.Fprintf(c.Writer, "data: %s\n\n", payload)
		c.Writer.Flush()

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
