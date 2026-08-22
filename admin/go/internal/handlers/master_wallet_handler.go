package handlers

import (
	"context"
	"math/big"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MasterWalletHandler struct {
	db        *gorm.DB
	ethClient interface {
		BalanceAt(ctx context.Context, account string, blockNum *big.Int) (*big.Int, error)
		TransactionByHash(ctx context.Context, hash string) (interface{}, error)
	}
	btcClient interface {
		GetBalance(address string) (int64, error)
		GetTransactions(address string) ([]interface{}, error)
	}
}

type MasterWallet struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Address   string    `json:"address"`
	Chain     string    `json:"chain"`
	Balance   float64   `json:"balance"`
	Currency  string    `json:"currency"`
	Status    string    `json:"status"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type WalletTransaction struct {
	ID            uuid.UUID `json:"id"`
	WalletID      uuid.UUID `json:"wallet_id"`
	TxHash        string    `json:"tx_hash"`
	FromAddress   string    `json:"from_address"`
	ToAddress     string    `json:"to_address"`
	Amount        float64   `json:"amount"`
	Currency      string    `json:"currency"`
	Status        string    `json:"status"`
	Confirmations int       `json:"confirmations"`
	CreatedAt     time.Time `json:"created_at"`
}

type WalletStats struct {
	TotalWallets int     `json:"total_wallets"`
	HotWallets   int     `json:"hot_wallets"`
	ColdWallets  int     `json:"cold_wallets"`
	WarmWallets  int     `json:"warm_wallets"`
	TotalBalance float64 `json:"total_balance"`
}

// MasterWalletRecord is the GORM-backed representation of a master wallet
// persisted in the database. It is the source of truth for balance and
// wallet-metadata queries, mirroring the in-memory MasterWallet DTO.
type MasterWalletRecord struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Name      string    `json:"name"`
	Address   string    `gorm:"index" json:"address"`
	Chain     string    `gorm:"index" json:"chain"`
	Balance   float64   `json:"balance"`
	Currency  string    `json:"currency"`
	Status    string    `gorm:"default:'active'" json:"status"`
	Type      string    `json:"type"` // hot, cold, warm
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (MasterWalletRecord) TableName() string {
	return "master_wallets"
}

// MasterWalletTransaction is the GORM-backed representation of a master
// wallet movement persisted in the database.
type MasterWalletTransaction struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	WalletID      uuid.UUID `gorm:"type:uuid;index" json:"wallet_id"`
	TxHash        string    `gorm:"index" json:"tx_hash"`
	FromAddress   string    `json:"from_address"`
	ToAddress     string    `json:"to_address"`
	Amount        float64   `json:"amount"`
	Currency      string    `json:"currency"`
	Status        string    `gorm:"default:'pending'" json:"status"`
	Confirmations int       `json:"confirmations"`
	CreatedAt     time.Time `json:"created_at"`
}

func (MasterWalletTransaction) TableName() string {
	return "master_wallet_transactions"
}

func NewMasterWalletHandler(db *gorm.DB) *MasterWalletHandler {
	return &MasterWalletHandler{db: db}
}

// GetWallets lists master wallet governance records from PostgreSQL.
func (h *MasterWalletHandler) GetWallets(c *gin.Context) {
	var records []MasterWalletRecord
	if err := h.db.Order("created_at DESC").Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch wallets"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": records})
}

// GetWalletByID returns one wallet record; 404 when it does not exist.
func (h *MasterWalletHandler) GetWalletByID(c *gin.Context) {
	walletID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid wallet_id"})
		return
	}
	var rec MasterWalletRecord
	if err := h.db.First(&rec, "id = ?", walletID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": rec})
}

// CreateWallet registers a watch-only master-wallet governance record. No key
// material is accepted or stored — the admin panel never holds private keys.
func (h *MasterWalletHandler) CreateWallet(c *gin.Context) {
	var req struct {
		Name    string `json:"name" binding:"required"`
		Address string `json:"address" binding:"required"`
		Chain   string `json:"chain" binding:"required"`
		Type    string `json:"type" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	rec := MasterWalletRecord{
		ID:       uuid.New(),
		Name:     req.Name,
		Address:  req.Address,
		Chain:    req.Chain,
		Currency: getCurrencyForChain(req.Chain),
		Status:   "active",
		Type:     req.Type,
	}
	if err := h.db.Create(&rec).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create wallet record"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": rec})
}

// UpdateWallet updates mutable metadata (name, status) on a wallet record.
func (h *MasterWalletHandler) UpdateWallet(c *gin.Context) {
	walletID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid wallet_id"})
		return
	}
	var req struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no updatable fields supplied"})
		return
	}
	res := h.db.Model(&MasterWalletRecord{}).Where("id = ?", walletID).Updates(updates)
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update wallet"})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "wallet updated"})
}

// DeleteWallet removes a wallet governance record.
func (h *MasterWalletHandler) DeleteWallet(c *gin.Context) {
	walletID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid wallet_id"})
		return
	}
	res := h.db.Delete(&MasterWalletRecord{}, "id = ?", walletID)
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete wallet"})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "wallet deleted"})
}

// GetBalance returns the recorded balance of one wallet record.
func (h *MasterWalletHandler) GetBalance(c *gin.Context) {
	walletID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid wallet_id"})
		return
	}
	var rec MasterWalletRecord
	if err := h.db.First(&rec, "id = ?", walletID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"balance":    rec.Balance,
			"currency":   rec.Currency,
			"updated_at": rec.UpdatedAt,
		},
	})
}

// GetTransactions lists master-wallet movement records. wallet_id is optional:
// when omitted (the registered /master-wallet/transactions route has no :id
// param) all records are returned, newest first.
func (h *MasterWalletHandler) GetTransactions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	query := h.db.Model(&MasterWalletTransaction{})
	if walletID := c.Param("id"); walletID != "" {
		walletUUID, err := uuid.Parse(walletID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid wallet_id"})
			return
		}
		query = query.Where("wallet_id = ?", walletUUID)
	}

	var total int64
	query.Count(&total)

	var records []MasterWalletTransaction
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch wallet transactions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"data":        records,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

// Transfer is fail-closed: admin panels never move crypto. Fund movement is
// the wallet owner's action via the canonical wallet backend only. The admin
// surface is strictly governance/read-only for master wallets.
func (h *MasterWalletHandler) Transfer(c *gin.Context) {
	c.JSON(http.StatusForbidden, gin.H{
		"error": "admin fund transfer is prohibited; crypto asset movement is performed only by the wallet owner via the canonical wallet backend",
	})
}

// RefreshBalance re-reads the persisted record (the balance is written by the
// settlement/indexing pipeline, not computed here).
func (h *MasterWalletHandler) RefreshBalance(c *gin.Context) {
	h.GetBalance(c)
}

// GetStats aggregates real counts and balances from the master_wallets table.
func (h *MasterWalletHandler) GetStats(c *gin.Context) {
	var stats WalletStats
	var total int64
	h.db.Model(&MasterWalletRecord{}).Count(&total)
	stats.TotalWallets = int(total)

	countByType := func(t string) int {
		var n int64
		h.db.Model(&MasterWalletRecord{}).Where("type = ?", t).Count(&n)
		return int(n)
	}
	stats.HotWallets = countByType("hot")
	stats.ColdWallets = countByType("cold")
	stats.WarmWallets = countByType("warm")

	var sum *float64
	h.db.Model(&MasterWalletRecord{}).Select("SUM(balance)").Scan(&sum)
	if sum != nil {
		stats.TotalBalance = *sum
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": stats})
}

// GetByChain filters wallet records by chain.
func (h *MasterWalletHandler) GetByChain(c *gin.Context) {
	chain := c.Param("chain")
	if chain == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chain required"})
		return
	}
	var records []MasterWalletRecord
	if err := h.db.Where("chain = ?", chain).Order("created_at DESC").Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch wallets"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": records})
}

// GetByType filters wallet records by custody type (hot/cold/warm).
func (h *MasterWalletHandler) GetByType(c *gin.Context) {
	walletType := c.Param("type")
	if walletType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type required"})
		return
	}
	var records []MasterWalletRecord
	if err := h.db.Where("type = ?", walletType).Order("created_at DESC").Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch wallets"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": records})
}

// FreezeWallet marks a wallet record frozen (governance flag, no fund action).
func (h *MasterWalletHandler) FreezeWallet(c *gin.Context) {
	h.setWalletFrozen(c, true)
}

// UnfreezeWallet clears the frozen governance flag.
func (h *MasterWalletHandler) UnfreezeWallet(c *gin.Context) {
	h.setWalletFrozen(c, false)
}

func (h *MasterWalletHandler) setWalletFrozen(c *gin.Context, frozen bool) {
	walletID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid wallet_id"})
		return
	}
	status := "active"
	if frozen {
		status = "frozen"
	}
	res := h.db.Model(&MasterWalletRecord{}).Where("id = ?", walletID).Update("status", status)
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update wallet status"})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "status": status})
}

func getCurrencyForChain(chain string) string {
	switch chain {
	case "Ethereum", "Polygon", "BSC", "Avalanche":
		return "USDT"
	case "Bitcoin", "Litecoin", "Dogecoin":
		return "BTC"
	default:
		return "USDT"
	}
}

// GetBalances returns balances across all master wallets, aggregated by
// currency and per wallet, sourced from the master_wallets table.
func (h *MasterWalletHandler) GetBalances(c *gin.Context) {
	chain := c.Query("chain")
	walletType := c.Query("type")

	query := h.db.Model(&MasterWalletRecord{})
	if chain != "" {
		query = query.Where("chain = ?", chain)
	}
	if walletType != "" {
		query = query.Where("type = ?", walletType)
	}

	var wallets []MasterWalletRecord
	if err := query.Order("updated_at DESC").Find(&wallets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch wallet balances"})
		return
	}

	type currencyBalance struct {
		Currency string  `json:"currency"`
		Total    float64 `json:"total"`
	}
	byCurrency := map[string]float64{}
	for _, w := range wallets {
		byCurrency[w.Currency] += w.Balance
	}
	totals := make([]currencyBalance, 0, len(byCurrency))
	for currency, total := range byCurrency {
		totals = append(totals, currencyBalance{Currency: currency, Total: total})
	}

	var grandTotal float64
	for _, t := range totals {
		grandTotal += t.Total
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"data":         wallets,
		"totals":       totals,
		"grand_total":  grandTotal,
		"wallet_count": len(wallets),
		"updated_at":   time.Now(),
	})
}
