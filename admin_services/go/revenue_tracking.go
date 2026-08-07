package main

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// ============================================================================
// TIGERWALLET REVENUE TRACKING SYSTEM
// Comprehensive revenue tracking for fees, swaps, transfers, and white labels
// ============================================================================

// RevenueType represents the type of revenue
type RevenueType string

const (
	RevenueTypeSwap       RevenueType = "swap"
	RevenueTypeTransfer   RevenueType = "transfer"
	RevenueTypeWithdraw   RevenueType = "withdraw"
	RevenueTypeDeposit    RevenueType = "deposit"
	RevenueTypeBridge     RevenueType = "bridge"
	RevenueTypeStaking    RevenueType = "staking"
	RevenueTypeNFT        RevenueType = "nft"
	RevenueTypeWhiteLabel RevenueType = "white_label"
	RevenueTypeNetworkFee RevenueType = "network_fee"
)

// Currency represents the currency of revenue
type Currency string

const (
	CurrencyUSD   Currency = "USD"
	CurrencyETH   Currency = "ETH"
	CurrencyBNB   Currency = "BNB"
	CurrencyMATIC Currency = "MATIC"
	CurrencyAVAX  Currency = "AVAX"
	CurrencyFTM   Currency = "FTM"
	CurrencyCRO   Currency = "CRO"
)

// RevenueRecord represents a single revenue record
type RevenueRecord struct {
	ID            string                 `json:"id"`
	WhiteLabelID  string                 `json:"whiteLabelId"`
	UserID        string                 `json:"userId"`
	Type          RevenueType            `json:"type"`
	Amount        float64                `json:"amount"`
	Currency      Currency               `json:"currency"`
	USDValue      float64                `json:"usdValue"`
	FeePercentage float64                `json:"feePercentage"`
	FeeAmount     float64                `json:"feeAmount"`
	NetworkFee    float64                `json:"networkFee"`
	ChainID       int64                  `json:"chainId"`
	TokenSymbol   string                 `json:"tokenSymbol"`
	TxHash        string                 `json:"txHash"`
	Timestamp     int64                  `json:"timestamp"`
	Status        string                 `json:"status"` // completed, pending, failed
	Metadata      map[string]interface{} `json:"metadata"`
}

// DailyRevenue represents daily revenue summary
type DailyRevenue struct {
	Date         string             `json:"date"`
	TotalRevenue float64            `json:"totalRevenue"`
	ByType       map[string]float64 `json:"byType"`
	ByCurrency   map[string]float64 `json:"byCurrency"`
	ByChain      map[int64]float64  `json:"byChain"`
	ByWhiteLabel map[string]float64 `json:"byWhiteLabel"`
	TxCount      int                `json:"txCount"`
}

// RevenueStats represents revenue statistics
type RevenueStats struct {
	TotalRevenue    float64            `json:"totalRevenue"`
	TotalFees       float64            `json:"totalFees"`
	TotalNetworkFee float64            `json:"totalNetworkFee"`
	ByType          map[string]float64 `json:"byType"`
	ByCurrency      map[string]float64 `json:"byCurrency"`
	ByChain         map[int64]float64  `json:"byChain"`
	ByWhiteLabel    map[string]float64 `json:"byWhiteLabel"`
	TxCount         int                `json:"txCount"`
	AverageTxValue  float64            `json:"averageTxValue"`
}

// RevenueTrackingService manages revenue tracking
type RevenueTrackingService struct {
	mu                sync.RWMutex
	records           map[string]*RevenueRecord
	dailyCache        map[string]*DailyRevenue
	whiteLabelRevenue map[string]float64
}

// Global revenue tracking instance
var (
	revenueService     *RevenueTrackingService
	revenueServiceOnce sync.Once
)

// GetRevenueService returns the singleton revenue tracking service
func GetRevenueService() *RevenueTrackingService {
	revenueServiceOnce.Do(func() {
		revenueService = &RevenueTrackingService{
			records:           make(map[string]*RevenueRecord),
			dailyCache:        make(map[string]*DailyRevenue),
			whiteLabelRevenue: make(map[string]float64),
		}
	})
	return revenueService
}

// RecordRevenue records a new revenue transaction
func (r *RevenueTrackingService) RecordRevenue(record *RevenueRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Validate record
	if record.Type == "" {
		return fmt.Errorf("revenue type is required")
	}
	if record.Amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	// Set timestamp if not set
	if record.Timestamp == 0 {
		record.Timestamp = time.Now().Unix()
	}

	// Set status if not set
	if record.Status == "" {
		record.Status = "completed"
	}

	// Calculate fee amount
	if record.FeePercentage > 0 {
		record.FeeAmount = record.Amount * record.FeePercentage / 100.0
	}

	// Store record
	r.records[record.ID] = record

	// Update white label revenue
	if record.WhiteLabelID != "" {
		r.whiteLabelRevenue[record.WhiteLabelID] += record.FeeAmount
	}

	// Update daily cache
	r.updateDailyCache(record)

	return nil
}

// updateDailyCache updates the daily revenue cache
func (r *RevenueTrackingService) updateDailyCache(record *RevenueRecord) {
	date := time.Unix(record.Timestamp, 0).Format("2006-01-02")

	daily, ok := r.dailyCache[date]
	if !ok {
		daily = &DailyRevenue{
			Date:         date,
			ByType:       make(map[string]float64),
			ByCurrency:   make(map[string]float64),
			ByChain:      make(map[int64]float64),
			ByWhiteLabel: make(map[string]float64),
		}
		r.dailyCache[date] = daily
	}

	// Update totals
	daily.TotalRevenue += record.FeeAmount
	daily.ByType[string(record.Type)] += record.FeeAmount
	daily.ByCurrency[string(record.Currency)] += record.FeeAmount
	daily.ByChain[record.ChainID] += record.FeeAmount
	if record.WhiteLabelID != "" {
		daily.ByWhiteLabel[record.WhiteLabelID] += record.FeeAmount
	}
	daily.TxCount++
}

// GetRevenueRecord returns a revenue record by ID
func (r *RevenueTrackingService) GetRevenueRecord(id string) (*RevenueRecord, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	record, ok := r.records[id]
	return record, ok
}

// GetRevenueByDateRange returns revenue records within a date range
func (r *RevenueTrackingService) GetRevenueByDateRange(startDate, endDate string) []*RevenueRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []*RevenueRecord
	start := parseDate(startDate)
	end := parseDate(endDate)

	for _, record := range r.records {
		recordDate := time.Unix(record.Timestamp, 0)
		if recordDate.After(start) && recordDate.Before(end.Add(24*time.Hour)) {
			results = append(results, record)
		}
	}

	return results
}

// GetRevenueByUser returns revenue records for a specific user
func (r *RevenueTrackingService) GetRevenueByUser(userID string) []*RevenueRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []*RevenueRecord
	for _, record := range r.records {
		if record.UserID == userID {
			results = append(results, record)
		}
	}

	return results
}

// GetRevenueByWhiteLabel returns revenue records for a specific white label
func (r *RevenueTrackingService) GetRevenueByWhiteLabel(whiteLabelID string) []*RevenueRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []*RevenueRecord
	for _, record := range r.records {
		if record.WhiteLabelID == whiteLabelID {
			results = append(results, record)
		}
	}

	return results
}

// GetRevenueByType returns revenue records by type
func (r *RevenueTrackingService) GetRevenueByType(revenueType RevenueType) []*RevenueRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []*RevenueRecord
	for _, record := range r.records {
		if record.Type == revenueType {
			results = append(results, record)
		}
	}

	return results
}

// GetDailyRevenue returns daily revenue for a specific date
func (r *RevenueTrackingService) GetDailyRevenue(date string) (*DailyRevenue, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	daily, ok := r.dailyCache[date]
	return daily, ok
}

// GetRevenueStats returns overall revenue statistics
func (r *RevenueTrackingService) GetRevenueStats() *RevenueStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := &RevenueStats{
		ByType:       make(map[string]float64),
		ByCurrency:   make(map[string]float64),
		ByChain:      make(map[int64]float64),
		ByWhiteLabel: make(map[string]float64),
	}

	for _, record := range r.records {
		stats.TotalRevenue += record.FeeAmount
		stats.TotalFees += record.FeeAmount
		stats.TotalNetworkFee += record.NetworkFee
		stats.ByType[string(record.Type)] += record.FeeAmount
		stats.ByCurrency[string(record.Currency)] += record.FeeAmount
		stats.ByChain[record.ChainID] += record.FeeAmount
		if record.WhiteLabelID != "" {
			stats.ByWhiteLabel[record.WhiteLabelID] += record.FeeAmount
		}
		stats.TxCount++
	}

	if stats.TxCount > 0 {
		stats.AverageTxValue = stats.TotalRevenue / float64(stats.TxCount)
	}

	return stats
}

// GetRevenueByPeriod returns revenue statistics for a specific period
func (r *RevenueTrackingService) GetRevenueByPeriod(startDate, endDate string) *RevenueStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := &RevenueStats{
		ByType:       make(map[string]float64),
		ByCurrency:   make(map[string]float64),
		ByChain:      make(map[int64]float64),
		ByWhiteLabel: make(map[string]float64),
	}

	start := parseDate(startDate)
	end := parseDate(endDate)

	for _, record := range r.records {
		recordDate := time.Unix(record.Timestamp, 0)
		if recordDate.After(start) && recordDate.Before(end.Add(24*time.Hour)) {
			stats.TotalRevenue += record.FeeAmount
			stats.TotalFees += record.FeeAmount
			stats.TotalNetworkFee += record.NetworkFee
			stats.ByType[string(record.Type)] += record.FeeAmount
			stats.ByCurrency[string(record.Currency)] += record.FeeAmount
			stats.ByChain[record.ChainID] += record.FeeAmount
			if record.WhiteLabelID != "" {
				stats.ByWhiteLabel[record.WhiteLabelID] += record.FeeAmount
			}
			stats.TxCount++
		}
	}

	if stats.TxCount > 0 {
		stats.AverageTxValue = stats.TotalRevenue / float64(stats.TxCount)
	}

	return stats
}

// GetWhiteLabelRevenue returns total revenue for a specific white label
func (r *RevenueTrackingService) GetWhiteLabelRevenue(whiteLabelID string) float64 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.whiteLabelRevenue[whiteLabelID]
}

// GetAllWhiteLabelRevenue returns all white label revenues
func (r *RevenueTrackingService) GetAllWhiteLabelRevenue() map[string]float64 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]float64)
	for k, v := range r.whiteLabelRevenue {
		result[k] = v
	}

	return result
}

// GetTopTokensByVolume returns top tokens by transaction volume
func (r *RevenueTrackingService) GetTopTokensByVolume(limit int) []map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tokenVolume := make(map[string]float64)
	tokenCount := make(map[string]int)

	for _, record := range r.records {
		tokenVolume[record.TokenSymbol] += record.Amount
		tokenCount[record.TokenSymbol]++
	}

	// Sort by volume
	type tokenData struct {
		symbol string
		volume float64
		count  int
	}

	var tokens []tokenData
	for symbol, volume := range tokenVolume {
		tokens = append(tokens, tokenData{symbol: symbol, volume: volume, count: tokenCount[symbol]})
	}

	for i := 0; i < len(tokens); i++ {
		for j := i + 1; j < len(tokens); j++ {
			if tokens[j].volume > tokens[i].volume {
				tokens[i], tokens[j] = tokens[j], tokens[i]
			}
		}
	}

	// Limit results
	if limit > 0 && len(tokens) > limit {
		tokens = tokens[:limit]
	}

	// Convert to result format
	var result []map[string]interface{}
	for _, t := range tokens {
		result = append(result, map[string]interface{}{
			"symbol": t.symbol,
			"volume": t.volume,
			"count":  t.count,
		})
	}

	return result
}

// GetTopChainsByVolume returns top chains by transaction volume
func (r *RevenueTrackingService) GetTopChainsByVolume(limit int) []map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	chainVolume := make(map[int64]float64)
	chainCount := make(map[int64]int)

	for _, record := range r.records {
		chainVolume[record.ChainID] += record.Amount
		chainCount[record.ChainID]++
	}

	// Sort by volume
	type chainData struct {
		chainID int64
		volume  float64
		count   int
	}

	var chains []chainData
	for chainID, volume := range chainVolume {
		chains = append(chains, chainData{chainID: chainID, volume: volume, count: chainCount[chainID]})
	}

	for i := 0; i < len(chains); i++ {
		for j := i + 1; j < len(chains); j++ {
			if chains[j].volume > chains[i].volume {
				chains[i], chains[j] = chains[j], chains[i]
			}
		}
	}

	// Limit results
	if limit > 0 && len(chains) > limit {
		chains = chains[:limit]
	}

	// Convert to result format
	var result []map[string]interface{}
	for _, c := range chains {
		result = append(result, map[string]interface{}{
			"chainId": c.chainID,
			"volume":  c.volume,
			"count":   c.count,
		})
	}

	return result
}

// ExportRevenue exports revenue data as JSON
func (r *RevenueTrackingService) ExportRevenue(format string) ([]byte, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	switch format {
	case "json":
		return json.MarshalIndent(r.records, "", "  ")
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// Helper function to parse date
func parseDate(dateStr string) time.Time {
	loc, _ := time.LoadLocation("UTC")
	t, _ := time.ParseInLocation("2006-01-02", dateStr, loc)
	return t
}

// ============================================================================
// FEE CALCULATION HELPERS
// ============================================================================

// CalculateSwapFee calculates the fee for a swap transaction
func CalculateSwapFee(amount float64, whiteLabelFee float64, dexFee float64) (totalFee, whiteLabelShare, platformShare float64) {
	totalFee = amount * (whiteLabelFee + dexFee) / 100.0
	whiteLabelShare = amount * whiteLabelFee / 100.0
	platformShare = amount * dexFee / 100.0
	return
}

// CalculateTransferFee calculates the fee for a transfer transaction
func CalculateTransferFee(amount float64, whiteLabelFee float64, networkFee float64) (totalFee, whiteLabelShare, networkCost float64) {
	totalFee = amount*whiteLabelFee/100.0 + networkFee
	whiteLabelShare = amount * whiteLabelFee / 100.0
	networkCost = networkFee
	return
}

// CalculateWithdrawFee calculates the fee for a withdrawal
func CalculateWithdrawFee(amount float64, whiteLabelFee float64, networkFee float64) (totalFee, whiteLabelShare, networkCost float64) {
	totalFee = amount*whiteLabelFee/100.0 + networkFee
	whiteLabelShare = amount * whiteLabelFee / 100.0
	networkCost = networkFee
	return
}

// ============================================================================
// DEFAULT FEE CONFIGURATION
// ============================================================================

// DefaultFeeConfig represents the default fee configuration
type DefaultFeeConfig struct {
	SwapFee        float64 `json:"swapFee"`        // 0.3%
	TransferFee    float64 `json:"transferFee"`    // 0.1%
	WithdrawFee    float64 `json:"withdrawFee"`    // 0.1%
	DepositFee     float64 `json:"depositFee"`     // 0%
	BridgeFee      float64 `json:"bridgeFee"`      // 0.5%
	StakingFee     float64 `json:"stakingFee"`     // 0%
	NFTFee         float64 `json:"nftFee"`         // 2.5%
	NetworkFeeBump float64 `json:"networkFeeBump"` // 10%
}

// GetDefaultFeeConfig returns the default fee configuration
func GetDefaultFeeConfig() *DefaultFeeConfig {
	return &DefaultFeeConfig{
		SwapFee:        0.3,
		TransferFee:    0.1,
		WithdrawFee:    0.1,
		DepositFee:     0.0,
		BridgeFee:      0.5,
		StakingFee:     0.0,
		NFTFee:         2.5,
		NetworkFeeBump: 10.0,
	}
}

// ============================================================================
// EXAMPLE USAGE
// ============================================================================

func main() {
	// Get revenue service instance
	rs := GetRevenueService()

	// Record some sample revenue
	sampleRecords := []*RevenueRecord{
		{
			ID:            "rev_001",
			WhiteLabelID:  "wl_partner1",
			UserID:        "user_123",
			Type:          RevenueTypeSwap,
			Amount:        1000.0,
			Currency:      CurrencyETH,
			USDValue:      2500.0,
			FeePercentage: 0.3,
			FeeAmount:     3.0,
			ChainID:       1,
			TokenSymbol:   "ETH",
			TxHash:        "0x1234567890abcdef",
			Timestamp:     time.Now().Unix(),
			Status:        "completed",
		},
		{
			ID:            "rev_002",
			WhiteLabelID:  "wl_partner1",
			UserID:        "user_456",
			Type:          RevenueTypeTransfer,
			Amount:        500.0,
			Currency:      CurrencyUSDT,
			USDValue:      500.0,
			FeePercentage: 0.1,
			FeeAmount:     0.5,
			ChainID:       56,
			TokenSymbol:   "USDT",
			TxHash:        "0xabcdef1234567890",
			Timestamp:     time.Now().Unix(),
			Status:        "completed",
		},
		{
			ID:            "rev_003",
			WhiteLabelID:  "wl_partner2",
			UserID:        "user_789",
			Type:          RevenueTypeWithdraw,
			Amount:        2000.0,
			Currency:      CurrencyBNB,
			USDValue:      600.0,
			FeePercentage: 0.1,
			FeeAmount:     2.0,
			ChainID:       56,
			TokenSymbol:   "BNB",
			TxHash:        "0x9876543210fedcba",
			Timestamp:     time.Now().Unix(),
			Status:        "completed",
		},
	}

	// Record all sample revenues
	for _, record := range sampleRecords {
		if err := rs.RecordRevenue(record); err != nil {
			fmt.Printf("Error recording revenue: %v\n", err)
		}
	}

	// Get overall statistics
	stats := rs.GetRevenueStats()
	fmt.Printf("\n=== Revenue Statistics ===\n")
	fmt.Printf("Total Revenue: $%.2f\n", stats.TotalRevenue)
	fmt.Printf("Total Fees: $%.2f\n", stats.TotalFees)
	fmt.Printf("Transaction Count: %d\n", stats.TxCount)
	fmt.Printf("Average Transaction Value: $%.2f\n", stats.AverageTxValue)

	// Get revenue by type
	fmt.Printf("\n=== Revenue by Type ===\n")
	for revenueType, amount := range stats.ByType {
		fmt.Printf("  %s: $%.2f\n", revenueType, amount)
	}

	// Get white label revenue
	fmt.Printf("\n=== Revenue by White Label ===\n")
	wlRevenue := rs.GetAllWhiteLabelRevenue()
	for wlID, amount := range wlRevenue {
		fmt.Printf("  %s: $%.2f\n", wlID, amount)
	}

	// Get top tokens by volume
	fmt.Printf("\n=== Top Tokens by Volume ===\n")
	topTokens := rs.GetTopTokensByVolume(5)
	for _, token := range topTokens {
		fmt.Printf("  %s: %.2f (count: %d)\n", token["symbol"], token["volume"], token["count"])
	}

	// Get daily revenue
	today := time.Now().Format("2006-01-02")
	daily, ok := rs.GetDailyRevenue(today)
	if ok {
		fmt.Printf("\n=== Daily Revenue (%s) ===\n", today)
		fmt.Printf("  Total: $%.2f\n", daily.TotalRevenue)
		fmt.Printf("  Transactions: %d\n", daily.TxCount)
	}

	// Demonstrate fee calculation
	fmt.Printf("\n=== Fee Calculations ===\n")
	swapFee, wlShare, platformShare := CalculateSwapFee(1000.0, 0.2, 0.3)
	fmt.Printf("Swap Fee (1000 USDT, 0.2%% WL, 0.3%% DEX):\n")
	fmt.Printf("  Total: $%.2f\n", swapFee)
	fmt.Printf("  White Label Share: $%.2f\n", wlShare)
	fmt.Printf("  Platform Share: $%.2f\n", platformShare)

	transferFee, wlShare2, networkCost := CalculateTransferFee(500.0, 0.1, 0.5)
	fmt.Printf("\nTransfer Fee (500 USDT, 0.1%% WL, $0.5 network):\n")
	fmt.Printf("  Total: $%.2f\n", transferFee)
	fmt.Printf("  White Label Share: $%.2f\n", wlShare2)
	fmt.Printf("  Network Cost: $%.2f\n", networkCost)

	fmt.Println("\nRevenue Tracking System Initialized Successfully!")
}
