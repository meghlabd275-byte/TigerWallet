package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/tigerwallet/admin/internal/models"
	"github.com/tigerwallet/admin/pkg/database"

	"github.com/gin-gonic/gin"
)

// PairHandler handles trading pair-related requests - COMPLETE IMPLEMENTATION
type PairHandler struct {
	db *database.PostgresDB
}

// NewPairHandler creates a new pair handler
func NewPairHandler(db *database.PostgresDB) *PairHandler {
	return &PairHandler{db: db}
}

// ListPairs lists all trading pairs
func (h *PairHandler) ListPairs(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	// Accept both snake_case and camelCase pagination params.
	pageSize := firstNonEmpty(c.Query("page_size"), c.Query("pageSize"))
	if pageSize == "" {
		pageSize = "20"
	}
	// Accept both snake_case and camelCase filter params.
	baseToken := firstNonEmpty(c.Query("base_token"), c.Query("baseToken"))
	quoteToken := firstNonEmpty(c.Query("quote_token"), c.Query("quoteToken"))
	chain := c.Query("chain")
	status := c.Query("status")
	active := c.Query("active")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)
	if pageInt < 1 {
		pageInt = 1
	}
	if pageSizeInt < 1 {
		pageSizeInt = 20
	}

	var pairs []models.TradingPair
	var total int64

	query := h.db.Model(&models.TradingPair{})

	if baseToken != "" {
		query = query.Where("base_token = ?", baseToken)
	}
	if quoteToken != "" {
		query = query.Where("quote_token = ?", quoteToken)
	}
	if chain != "" {
		query = query.Where("chain = ?", chain)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if active != "" {
		isActive := active == "true" || active == "1"
		query = query.Where("is_active = ?", isActive)
	}

	query.Count(&total)

	offset := (pageInt - 1) * pageSizeInt
	query = query.Offset(offset).Limit(pageSizeInt).Order("volume_24h DESC")

	if err := query.Find(&pairs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch pairs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        pairs,
		"total":       total,
		"page":        pageInt,
		"page_size":   pageSizeInt,
		"total_pages": (total + int64(pageSizeInt) - 1) / int64(pageSizeInt),
	})
}

// GetPair gets a trading pair by ID
func (h *PairHandler) GetPair(c *gin.Context) {
	pairID := c.Param("id")

	var pair models.TradingPair
	if err := h.db.Preload("Token0").Preload("Token1").First(&pair, pairID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pair not found"})
		return
	}

	c.JSON(http.StatusOK, pair)
}

// CreatePair creates a new trading pair
func (h *PairHandler) CreatePair(c *gin.Context) {
	adminID := c.GetUint("admin_id")

	// Permissive binding: accept both snake_case (pair_name/base_token/...) and
	// camelCase (pairName/baseToken/...) keys, and accept amounts/fees as either
	// JSON numbers or numeric strings (the admin frontend sends strings).
	var req struct {
		PairName          string  `json:"pair_name"`
		PairNameCamel     string  `json:"pairName"`
		BaseToken         string  `json:"base_token"`
		BaseTokenCamel    string  `json:"baseToken"`
		QuoteToken        string  `json:"quote_token"`
		QuoteTokenCamel   string  `json:"quoteToken"`
		Chain             string  `json:"chain"`
		ChainID           string  `json:"chainId"`
		MinTradeAmount    float64 `json:"min_trade_amount"`
		MinTradeAmountCam float64 `json:"minTradeAmount"`
		MaxTradeAmount    float64 `json:"max_trade_amount"`
		MaxTradeAmountCam float64 `json:"maxTradeAmount"`
		MakerFee          float64 `json:"maker_fee"`
		MakerFeeCamel     float64 `json:"makerFee"`
		TakerFee          float64 `json:"taker_fee"`
		TakerFeeCamel     float64 `json:"takerFee"`
		PricePrecision    int     `json:"price_precision"`
		QuantityPrecision int     `json:"quantity_precision"`
		MinPrice          float64 `json:"min_price"`
		MaxPrice          float64 `json:"max_price"`
		IsActive          bool    `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	baseToken := firstNonEmpty(req.BaseToken, req.BaseTokenCamel)
	quoteToken := firstNonEmpty(req.QuoteToken, req.QuoteTokenCamel)
	chain := firstNonEmpty(req.Chain, req.ChainID)
	if baseToken == "" || quoteToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "base_token and quote_token are required"})
		return
	}
	// The admin frontend's create-pair modal does not expose a chain selector,
	// so default to ethereum when omitted rather than rejecting the request.
	if chain == "" {
		chain = "ethereum"
	}

	pairName := firstNonEmpty(req.PairName, req.PairNameCamel)
	if pairName == "" {
		pairName = baseToken + "/" + quoteToken
	}

	// Check if pair already exists
	var existingPair models.TradingPair
	if err := h.db.Where("base_token = ? AND quote_token = ? AND chain = ?",
		baseToken, quoteToken, chain).First(&existingPair).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Pair already exists"})
		return
	}

	minTrade := req.MinTradeAmount
	if minTrade == 0 {
		minTrade = req.MinTradeAmountCam
	}
	maxTrade := req.MaxTradeAmount
	if maxTrade == 0 {
		maxTrade = req.MaxTradeAmountCam
	}
	makerFee := req.MakerFee
	if makerFee == 0 {
		makerFee = req.MakerFeeCamel
	}
	takerFee := req.TakerFee
	if takerFee == 0 {
		takerFee = req.TakerFeeCamel
	}

	pair := models.TradingPair{
		PairName:          pairName,
		BaseToken:         baseToken,
		QuoteToken:        quoteToken,
		Chain:             chain,
		MinTradeAmount:    minTrade,
		MaxTradeAmount:    maxTrade,
		MakerFee:          makerFee,
		TakerFee:          takerFee,
		PricePrecision:    req.PricePrecision,
		QuantityPrecision: req.QuantityPrecision,
		MinPrice:          req.MinPrice,
		MaxPrice:          req.MaxPrice,
		IsActive:          req.IsActive,
		CreatedBy:         adminID,
	}

	if err := h.db.Create(&pair).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create pair"})
		return
	}

	logAdminActivity(h.db, adminID, "create_pair", "pair", strconv.FormatUint(uint64(pair.ID), 10), "Created pair: "+pair.PairName, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusCreated, pair)
}

// UpdatePair updates a trading pair
func (h *PairHandler) UpdatePair(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	pairID := c.Param("id")

	// Permissive binding: accept both snake_case and camelCase keys (the admin
	// frontend sends camelCase such as minTradeAmount/maxTradeAmount).
	var req struct {
		PairName           string  `json:"pair_name"`
		PairNameCamel      string  `json:"pairName"`
		MinTradeAmount     float64 `json:"min_trade_amount"`
		MinTradeAmountCam  float64 `json:"minTradeAmount"`
		MaxTradeAmount     float64 `json:"max_trade_amount"`
		MaxTradeAmountCam  float64 `json:"maxTradeAmount"`
		MakerFee           float64 `json:"maker_fee"`
		MakerFeeCamel      float64 `json:"makerFee"`
		TakerFee           float64 `json:"taker_fee"`
		TakerFeeCamel      float64 `json:"takerFee"`
		PricePrecision     int     `json:"price_precision"`
		QuantityPrecision  int     `json:"quantity_precision"`
		MinPrice           float64 `json:"min_price"`
		MaxPrice           float64 `json:"max_price"`
		IsActive           *bool   `json:"is_active"`
		Active             *bool   `json:"active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	var pair models.TradingPair
	if err := h.db.First(&pair, pairID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pair not found"})
		return
	}

	updates := map[string]interface{}{}

	pairName := firstNonEmpty(req.PairName, req.PairNameCamel)
	if pairName != "" {
		updates["pair_name"] = pairName
	}
	minTrade := req.MinTradeAmount
	if minTrade == 0 {
		minTrade = req.MinTradeAmountCam
	}
	if minTrade > 0 {
		updates["min_trade_amount"] = minTrade
	}
	maxTrade := req.MaxTradeAmount
	if maxTrade == 0 {
		maxTrade = req.MaxTradeAmountCam
	}
	if maxTrade > 0 {
		updates["max_trade_amount"] = maxTrade
	}
	makerFee := req.MakerFee
	if makerFee == 0 {
		makerFee = req.MakerFeeCamel
	}
	if makerFee > 0 {
		updates["maker_fee"] = makerFee
	}
	takerFee := req.TakerFee
	if takerFee == 0 {
		takerFee = req.TakerFeeCamel
	}
	if takerFee > 0 {
		updates["taker_fee"] = takerFee
	}
	if req.PricePrecision > 0 {
		updates["price_precision"] = req.PricePrecision
	}
	if req.QuantityPrecision > 0 {
		updates["quantity_precision"] = req.QuantityPrecision
	}
	if req.MinPrice > 0 {
		updates["min_price"] = req.MinPrice
	}
	if req.MaxPrice > 0 {
		updates["max_price"] = req.MaxPrice
	}
	active := req.IsActive
	if active == nil {
		active = req.Active
	}
	if active != nil {
		updates["is_active"] = *active
		updates["status"] = map[bool]string{true: "active", false: "suspended"}[*active]
	}

	if err := h.db.Model(&pair).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update pair"})
		return
	}

	logAdminActivity(h.db, adminID, "update_pair", "pair", pairID, "Updated pair: "+pair.PairName, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, pair)
}

// DeletePair deletes a trading pair
func (h *PairHandler) DeletePair(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	pairID := c.Param("id")

	var pair models.TradingPair
	if err := h.db.First(&pair, pairID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pair not found"})
		return
	}

	if err := h.db.Delete(&pair).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete pair"})
		return
	}

	logAdminActivity(h.db, adminID, "delete_pair", "pair", pairID, "Deleted pair: "+pair.PairName, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Pair deleted successfully"})
}

// UpdatePairStatus updates pair status (activate/deactivate).
// Accepts the admin frontend's {active: true|false} payload, and also the
// legacy {status: "active"|"suspended"|"halted"} form.
func (h *PairHandler) UpdatePairStatus(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	pairID := c.Param("id")

	var req struct {
		Status string `json:"status"`
		Active *bool  `json:"active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	var pair models.TradingPair
	if err := h.db.First(&pair, pairID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pair not found"})
		return
	}

	updates := map[string]interface{}{}
	description := ""
	if req.Active != nil {
		updates["is_active"] = *req.Active
		if *req.Active {
			updates["status"] = "active"
		} else {
			updates["status"] = "suspended"
		}
		description = "Updated pair active flag to: " + strconv.FormatBool(*req.Active)
	} else if req.Status != "" {
		if req.Status != "active" && req.Status != "suspended" && req.Status != "halted" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status"})
			return
		}
		updates["status"] = req.Status
		updates["is_active"] = req.Status == "active"
		description = "Updated pair status to: " + req.Status
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "active or status is required"})
		return
	}

	if err := h.db.Model(&pair).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
		return
	}

	logAdminActivity(h.db, adminID, "update_pair_status", "pair", pairID, description, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Status updated successfully"})
}

// UpdatePairPrice updates pair price
func (h *PairHandler) UpdatePairPrice(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	pairID := c.Param("id")

	var req struct {
		Price       string `json:"price" binding:"required"`
		Volume24h   string `json:"volume_24h"`
		PriceChange string `json:"price_change_24h"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	var pair models.TradingPair
	if err := h.db.First(&pair, pairID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pair not found"})
		return
	}

	updates := map[string]interface{}{
		"last_price": req.Price,
	}

	if req.Volume24h != "" {
		updates["volume_24h"] = req.Volume24h
	}
	if req.PriceChange != "" {
		updates["price_change_24h"] = req.PriceChange
	}

	if err := h.db.Model(&pair).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update price"})
		return
	}

	logAdminActivity(h.db, adminID, "update_pair_price", "pair", pairID, "Updated price: "+req.Price, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, pair)
}

// GetPairStats gets trading pair statistics
func (h *PairHandler) GetPairStats(c *gin.Context) {
	var stats struct {
		TotalPairs     int64   `json:"total_pairs"`
		ActivePairs    int64   `json:"active_pairs"`
		SuspendedPairs int64   `json:"suspended_pairs"`
		HaltingPairs   int64   `json:"halting_pairs"`
		TotalVolume    float64 `json:"total_volume"`
		Volume24h      float64 `json:"volume_24h"`
	}

	h.db.Model(&models.TradingPair{}).Count(&stats.TotalPairs)
	h.db.Model(&models.TradingPair{}).Where("status = ?", "active").Count(&stats.ActivePairs)
	h.db.Model(&models.TradingPair{}).Where("status = ?", "suspended").Count(&stats.SuspendedPairs)
	h.db.Model(&models.TradingPair{}).Where("status = ?", "halted").Count(&stats.HaltingPairs)

	// Get volume from all active pairs
	h.db.Model(&models.TradingPair{}).Where("status = ?", "active").Select("COALESCE(SUM(volume_24h), 0)").Scan(&stats.Volume24h)

	c.JSON(http.StatusOK, stats)
}

// ImportPairs bulk-imports trading pairs from an external system.
// Accepts the admin frontend's {pairs: [...]} envelope and is permissive on
// field names: both snake_case (base_token/quote_token/chain) and camelCase
// (baseToken/quoteToken/chainId) keys are accepted. A bare array is also
// accepted for compatibility with older callers.
func (h *PairHandler) ImportPairs(c *gin.Context) {
	adminID := c.GetUint("admin_id")

	type pairItem struct {
		PairName          string  `json:"pair_name"`
		PairNameCamel     string  `json:"pairName"`
		BaseToken         string  `json:"base_token"`
		BaseTokenCamel    string  `json:"baseToken"`
		QuoteToken        string  `json:"quote_token"`
		QuoteTokenCamel   string  `json:"quoteToken"`
		Chain             string  `json:"chain"`
		ChainID           string  `json:"chainId"`
		Fee               float64 `json:"fee"`
		MinTradeAmount    float64 `json:"min_trade_amount"`
		MinTradeAmountCam float64 `json:"minTradeAmount"`
		MaxTradeAmount    float64 `json:"max_trade_amount"`
		MaxTradeAmountCam float64 `json:"maxTradeAmount"`
		MakerFee          float64 `json:"maker_fee"`
		TakerFee          float64 `json:"taker_fee"`
		PricePrecision    int     `json:"price_precision"`
		QuantityPrecision int     `json:"quantity_precision"`
	}

	// Read the body once so we can try both the {pairs:[...]} envelope and a
	// bare array payload without consuming the body twice.
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	var items []pairItem
	// First try the frontend's {pairs: [...]} envelope.
	var envelope struct {
		Pairs []pairItem `json:"pairs"`
	}
	if err := json.Unmarshal(rawBody, &envelope); err == nil && len(envelope.Pairs) > 0 {
		items = envelope.Pairs
	} else if err := json.Unmarshal(rawBody, &items); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body, expected {pairs: [...]} or an array of pairs"})
		return
	}

	imported := 0
	failed := 0
	for _, item := range items {
		baseToken := firstNonEmpty(item.BaseToken, item.BaseTokenCamel)
		quoteToken := firstNonEmpty(item.QuoteToken, item.QuoteTokenCamel)
		chain := firstNonEmpty(item.Chain, item.ChainID)
		if baseToken == "" || quoteToken == "" {
			failed++
			continue
		}
		if chain == "" {
			chain = "ethereum"
		}
		pairName := firstNonEmpty(item.PairName, item.PairNameCamel)
		if pairName == "" {
			pairName = baseToken + "/" + quoteToken
		}
		// Skip duplicates
		var existing models.TradingPair
		if err := h.db.Where("base_token = ? AND quote_token = ? AND chain = ?", baseToken, quoteToken, chain).First(&existing).Error; err == nil {
			failed++
			continue
		}
		fee := item.MakerFee
		if fee == 0 {
			fee = item.Fee
		}
		minTrade := item.MinTradeAmount
		if minTrade == 0 {
			minTrade = item.MinTradeAmountCam
		}
		maxTrade := item.MaxTradeAmount
		if maxTrade == 0 {
			maxTrade = item.MaxTradeAmountCam
		}
		pair := models.TradingPair{
			PairName:          pairName,
			BaseToken:         baseToken,
			QuoteToken:        quoteToken,
			Chain:             chain,
			MinTradeAmount:    minTrade,
			MaxTradeAmount:    maxTrade,
			MakerFee:          fee,
			TakerFee:          item.TakerFee,
			PricePrecision:    item.PricePrecision,
			QuantityPrecision: item.QuantityPrecision,
			IsActive:          true,
			Status:            "active",
			CreatedBy:         adminID,
		}
		if err := h.db.Create(&pair).Error; err != nil {
			failed++
			continue
		}
		imported++
	}
	logAdminActivity(h.db, adminID, "import_pairs", "pair", "", "Imported trading pairs from external system", c.ClientIP(), c.Request.UserAgent())
	c.JSON(http.StatusOK, gin.H{"imported": imported, "failed": failed})
}

// firstNonEmpty returns the first non-empty string argument, or "" if all are empty.
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
