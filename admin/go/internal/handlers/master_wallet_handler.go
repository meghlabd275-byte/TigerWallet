package handlers

import (
	"context"
	"encoding/hex"
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

func (h *MasterWalletHandler) GetWallets(c *gin.Context) {
	wallets := []MasterWallet{
		{
			ID:       uuid.New(),
			Name:     "Hot Wallet Main",
			Address:  "0x742d35Cc6634C0532925a3b844Bc9e7595f6eB2E",
			Chain:    "Ethereum",
			Balance:  1500000.0,
			Currency: "USDT",
			Status:   "active",
			Type:     "hot",
		},
		{
			ID:       uuid.New(),
			Name:     "Cold Wallet Primary",
			Address:  "0x8ba1f109551bD432803012645Ac136ddd64DBA72",
			Chain:    "Ethereum",
			Balance:  10000000.0,
			Currency: "USDT",
			Status:   "active",
			Type:     "cold",
		},
		{
			ID:       uuid.New(),
			Name:     "Hot Wallet Fee",
			Address:  "bc1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh",
			Chain:    "Bitcoin",
			Balance:  50000.0,
			Currency: "BTC",
			Status:   "active",
			Type:     "hot",
		},
	}

	c.JSON(200, gin.H{
		"success": true,
		"data":    wallets,
	})
}

func (h *MasterWalletHandler) GetWalletByID(c *gin.Context) {
	walletID := c.Param("id")
	if walletID == "" {
		c.JSON(400, gin.H{"error": "wallet_id required"})
		return
	}

	wallet := MasterWallet{
		ID:       uuid.MustParse(walletID),
		Name:     "Hot Wallet Main",
		Address:  "0x742d35Cc6634C0532925a3b844Bc9e7595f6eB2E",
		Chain:    "Ethereum",
		Balance:  1500000.0,
		Currency: "USDT",
		Status:   "active",
		Type:     "hot",
	}

	c.JSON(200, gin.H{
		"success": true,
		"data":    wallet,
	})
}

func (h *MasterWalletHandler) CreateWallet(c *gin.Context) {
	var req struct {
		Name    string `json:"name" binding:"required"`
		Address string `json:"address" binding:"required"`
		Chain   string `json:"chain" binding:"required"`
		Type    string `json:"type" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	wallet := MasterWallet{
		ID:       uuid.New(),
		Name:     req.Name,
		Address:  req.Address,
		Chain:    req.Chain,
		Balance:  0.0,
		Currency: getCurrencyForChain(req.Chain),
		Status:   "active",
		Type:     req.Type,
	}

	c.JSON(201, gin.H{
		"success": true,
		"data":    wallet,
	})
}

func (h *MasterWalletHandler) UpdateWallet(c *gin.Context) {
	walletID := c.Param("id")
	if walletID == "" {
		c.JSON(400, gin.H{"error": "wallet_id required"})
		return
	}

	var req struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "wallet updated",
	})
}

func (h *MasterWalletHandler) DeleteWallet(c *gin.Context) {
	walletID := c.Param("id")
	if walletID == "" {
		c.JSON(400, gin.H{"error": "wallet_id required"})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "wallet deleted",
	})
}

func (h *MasterWalletHandler) GetBalance(c *gin.Context) {
	walletID := c.Param("id")
	if walletID == "" {
		c.JSON(400, gin.H{"error": "wallet_id required"})
		return
	}

	wallet := MasterWallet{
		ID:       uuid.MustParse(walletID),
		Name:     "Hot Wallet Main",
		Address:  "0x742d35Cc6634C0532925a3b844Bc9e7595f6eB2E",
		Chain:    "Ethereum",
		Balance:  1500000.0,
		Currency: "USDT",
		Status:   "active",
		Type:     "hot",
	}

	c.JSON(200, gin.H{
		"success": true,
		"data": gin.H{
			"balance":    wallet.Balance,
			"currency":   wallet.Currency,
			"updated_at": time.Now(),
		},
	})
}

func (h *MasterWalletHandler) GetTransactions(c *gin.Context) {
	walletID := c.Param("id")
	if walletID == "" {
		c.JSON(400, gin.H{"error": "wallet_id required"})
		return
	}

	walletUUID, err := uuid.Parse(walletID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid wallet_id"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int64
	query := h.db.Model(&MasterWalletTransaction{}).Where("wallet_id = ?", walletUUID)
	query.Count(&total)

	var records []MasterWalletTransaction
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch wallet transactions"})
		return
	}

	c.JSON(200, gin.H{
		"success":      true,
		"data":         records,
		"total":        total,
		"page":         page,
		"page_size":    pageSize,
		"total_pages":  (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

func (h *MasterWalletHandler) Transfer(c *gin.Context) {
	walletID := c.Param("id")
	if walletID == "" {
		c.JSON(400, gin.H{"error": "wallet_id required"})
		return
	}

	var req struct {
		ToAddress string  `json:"to_address" binding:"required"`
		Amount    float64 `json:"amount" binding:"required"`
		Currency  string  `json:"currency"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	tx := WalletTransaction{
		ID:            uuid.New(),
		WalletID:      uuid.MustParse(walletID),
		TxHash:        generateTxHash(),
		FromAddress:   "0x742d35Cc6634C0532925a3b844Bc9e7595f6eB2E",
		ToAddress:     req.ToAddress,
		Amount:        req.Amount,
		Currency:      req.Currency,
		Status:        "pending",
		Confirmations: 0,
		CreatedAt:     time.Now(),
	}

	c.JSON(201, gin.H{
		"success": true,
		"data":    tx,
	})
}

func (h *MasterWalletHandler) RefreshBalance(c *gin.Context) {
	walletID := c.Param("id")
	if walletID == "" {
		c.JSON(400, gin.H{"error": "wallet_id required"})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"data": gin.H{
			"balance":    1500000.0,
			"currency":   "USDT",
			"updated_at": time.Now(),
		},
	})
}

func (h *MasterWalletHandler) GetStats(c *gin.Context) {
	stats := WalletStats{
		TotalWallets: 5,
		HotWallets:   2,
		ColdWallets:  2,
		WarmWallets:  1,
		TotalBalance: 15000000.0,
	}

	c.JSON(200, gin.H{
		"success": true,
		"data":    stats,
	})
}

func (h *MasterWalletHandler) GetByChain(c *gin.Context) {
	chain := c.Param("chain")
	if chain == "" {
		c.JSON(400, gin.H{"error": "chain required"})
		return
	}

	wallets := []MasterWallet{
		{
			ID:       uuid.New(),
			Name:     "Hot Wallet Main",
			Address:  "0x742d35Cc6634C0532925a3b844Bc9e7595f6eB2E",
			Chain:    chain,
			Balance:  1500000.0,
			Currency: "USDT",
			Status:   "active",
			Type:     "hot",
		},
	}

	c.JSON(200, gin.H{
		"success": true,
		"data":    wallets,
	})
}

func (h *MasterWalletHandler) GetByType(c *gin.Context) {
	walletType := c.Param("type")
	if walletType == "" {
		c.JSON(400, gin.H{"error": "type required"})
		return
	}

	wallets := []MasterWallet{
		{
			ID:       uuid.New(),
			Name:     "Hot Wallet Main",
			Address:  "0x742d35Cc6634C0532925a3b844Bc9e7595f6eB2E",
			Chain:    "Ethereum",
			Balance:  1500000.0,
			Currency: "USDT",
			Status:   "active",
			Type:     walletType,
		},
	}

	c.JSON(200, gin.H{
		"success": true,
		"data":    wallets,
	})
}

func (h *MasterWalletHandler) FreezeWallet(c *gin.Context) {
	walletID := c.Param("id")
	if walletID == "" {
		c.JSON(400, gin.H{"error": "wallet_id required"})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "wallet frozen",
	})
}

func (h *MasterWalletHandler) UnfreezeWallet(c *gin.Context) {
	walletID := c.Param("id")
	if walletID == "" {
		c.JSON(400, gin.H{"error": "wallet_id required"})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "wallet unfrozen",
	})
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

func generateTxHash() string {
	bytes := make([]byte, 32)
	for i := range bytes {
		bytes[i] = byte(i % 16)
	}
	return "0x" + hex.EncodeToString(bytes)
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

	c.JSON(200, gin.H{
		"success":       true,
		"data":          wallets,
		"totals":        totals,
		"grand_total":   grandTotal,
		"wallet_count":  len(wallets),
		"updated_at":    time.Now(),
	})
}
