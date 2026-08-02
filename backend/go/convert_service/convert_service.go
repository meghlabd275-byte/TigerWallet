package convert_service

import (
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"
)

// ============================================================================
// Types
// ============================================================================

type ConvertPair struct {
	From      string  `json:"from"`
	To        string  `json:"to"`
	Rate      float64 `json:"rate"`
	InverseRate float64 `json:"inverseRate"`
	Fee       float64 `json:"fee"`
	Enabled   bool    `json:"enabled"`
}

type ConvertOrder struct {
	ID          string    `json:"id"`
	UserID      string    `json:"userId"`
	FromToken   string    `json:"fromToken"`
	ToToken     string    `json:"toToken"`
	FromAmount  float64   `json:"fromAmount"`
	ToAmount    float64   `json:"toAmount"`
	Rate        float64   `json:"rate"`
	Fee         float64   `json:"fee"`
	Status      string    `json:"status"`
	CreateTime  time.Time `json:"createTime"`
	CompleteTime *time.Time `json:"completeTime,omitempty"`
	TxHash      string    `json:"txHash,omitempty"`
}

type UserBalance struct {
	UserID     string              `json:"userId"`
	Balances   map[string]float64  `json:"balances"`
}

type ConvertSettings struct {
	Enabled       bool    `json:"enabled"`
	MinAmount     float64 `json:"minAmount"`
	MaxAmount     float64 `json:"maxAmount"`
	DefaultFee    float64 `json:"defaultFee"`
}

// ============================================================================
// Service
// ============================================================================

type ConvertService struct {
	mu          sync.RWMutex
	pairs       map[string]*ConvertPair
	orders      map[string][]ConvertOrder
	balances    map[string]*UserBalance
	settings    ConvertSettings
}

func NewConvertService() *ConvertService {
	cs := &ConvertService{
		pairs:    make(map[string]*ConvertPair),
		orders:   make(map[string][]ConvertOrder),
		balances: make(map[string]*UserBalance),
		settings: ConvertSettings{
			Enabled:    true,
			MinAmount:  1,
			MaxAmount:  1000000,
			DefaultFee:  0.1, // 0.1%
		},
	}
	cs.initializeDefaultPairs()
	return cs
}

func (cs *ConvertService) initializeDefaultPairs() {
	// Common trading pairs with USDT as base
	pairs := []struct{ from, to string; rate float64 }{
		{"BTC", "USDT", 43250},
		{"ETH", "USDT", 2280},
		{"BNB", "USDT", 312.5},
		{"SOL", "USDT", 98.75},
		{"XRP", "USDT", 0.62},
		{"DOGE", "USDT", 0.082},
		{"ADA", "USDT", 0.58},
		{"AVAX", "USDT", 38.20},
		{"DOT", "USDT", 7.85},
		{"LINK", "USDT", 14.50},
		{"MATIC", "USDT", 0.92},
		{"LTC", "USDT", 72.30},
		{"UNI", "USDT", 6.25},
		{"ATOM", "USDT", 10.45},
		{"XLM", "USDT", 0.125},
		{"NEAR", "USDT", 3.25},
		{"USDC", "USDT", 1.0001},
		{"USDT", "USDC", 0.9999},
		{"DOGE", "BTC", 0.0000019},
		{"ETH", "BTC", 0.052},
	}

	for _, p := range pairs {
		key := fmt.Sprintf("%s_%s", p.from, p.to)
		cs.pairs[key] = &ConvertPair{
			From:        p.from,
			To:          p.to,
			Rate:        p.rate,
			InverseRate: 1 / p.rate,
			Fee:         0.1,
			Enabled:     true,
		}
	}
}

func (cs *ConvertService) GetAllPairs() []*ConvertPair {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	pairs := make([]*ConvertPair, 0, len(cs.pairs))
	for _, pair := range cs.pairs {
		pairs = append(pairs, pair)
	}
	return pairs
}

func (cs *ConvertService) GetPair(fromToken, toToken string) (*ConvertPair, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	key := fmt.Sprintf("%s_%s", fromToken, toToken)
	pair, ok := cs.pairs[key]
	if !ok {
		return nil, fmt.Errorf("convert pair not found: %s/%s", fromToken, toToken)
	}

	if !pair.Enabled {
		return nil, fmt.Errorf("convert pair is disabled: %s/%s", fromToken, toToken)
	}

	return pair, nil
}

func (cs *ConvertService) GetRate(fromToken, toToken string) (float64, float64, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	// First try direct pair
	key := fmt.Sprintf("%s_%s", fromToken, toToken)
	if pair, ok := cs.pairs[key]; ok && pair.Enabled {
		return pair.Rate, pair.Fee, nil
	}

	// Try reverse pair
	reverseKey := fmt.Sprintf("%s_%s", toToken, fromToken)
	if pair, ok := cs.pairs[reverseKey]; ok && pair.Enabled {
		return pair.InverseRate, pair.Fee, nil
	}

	// Try through USDT as intermediate
	fromUSDTKey := fmt.Sprintf("%s_USDT", fromToken)
	toUSDTKey := fmt.Sprintf("USDT_%s", toToken)

	var fromRate, toRate float64
	var fromFee, toFee float64

	if fromPair, ok := cs.pairs[fromUSDTKey]; ok && fromPair.Enabled {
		fromRate = fromPair.Rate
		fromFee = fromPair.Fee
	} else if fromReverse, ok := cs.pairs["USDT_"+fromToken]; ok && fromReverse.Enabled {
		fromRate = fromReverse.InverseRate
		fromFee = fromReverse.Fee
	} else {
		return 0, 0, fmt.Errorf("no conversion path found")
	}

	if toPair, ok := cs.pairs[toUSDTKey]; ok && toPair.Enabled {
		toRate = toPair.Rate
		toFee = toPair.Fee
	} else if toReverse, ok := cs.pairs[toToken+"_USDT"]; ok && toReverse.Enabled {
		toRate = toReverse.InverseRate
		toFee = toReverse.Fee
	} else {
		return 0, 0, fmt.Errorf("no conversion path found")
	}

	combinedRate := fromRate * toRate
	combinedFee := (fromFee + toFee) / 2

	return combinedRate, combinedFee, nil
}

func (cs *ConvertService) Convert(userID, fromToken, toToken string, fromAmount float64) (*ConvertOrder, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if !cs.settings.Enabled {
		return nil, fmt.Errorf("convert is currently disabled")
	}

	if fromAmount < cs.settings.MinAmount {
		return nil, fmt.Errorf("amount below minimum: %f", cs.settings.MinAmount)
	}

	if fromAmount > cs.settings.MaxAmount {
		return nil, fmt.Errorf("amount above maximum: %f", cs.settings.MaxAmount)
	}

	// Check user balance
	balance, ok := cs.balances[userID]
	if !ok {
		return nil, fmt.Errorf("user balance not found")
	}

	if balance.Balances[fromToken] < fromAmount {
		return nil, fmt.Errorf("insufficient balance: %f %s", balance.Balances[fromToken], fromToken)
	}

	// Get rate
	rate, fee, err := cs.getRateUnsafe(fromToken, toToken)
	if err != nil {
		return nil, err
	}

	// Calculate amounts
	feeAmount := fromAmount * fee / 100
	netAmount := fromAmount - feeAmount
	toAmount := netAmount * rate

	order := &ConvertOrder{
		ID:         fmt.Sprintf("convert-%d", time.Now().Unix()),
		UserID:     userID,
		FromToken:  fromToken,
		ToToken:    toToken,
		FromAmount: fromAmount,
		ToAmount:   toAmount,
		Rate:       rate,
		Fee:        feeAmount,
		Status:     "completed",
		CreateTime: time.Now(),
	}

	// Update balances
	balance.Balances[fromToken] -= fromAmount
	balance.Balances[toToken] += toAmount

	cs.orders[userID] = append(cs.orders[userID], *order)
	return order, nil
}

func (cs *ConvertService) getRateUnsafe(fromToken, toToken string) (float64, float64, error) {
	// Direct pair
	key := fmt.Sprintf("%s_%s", fromToken, toToken)
	if pair, ok := cs.pairs[key]; ok && pair.Enabled {
		return pair.Rate, pair.Fee, nil
	}

	// Reverse pair
	reverseKey := fmt.Sprintf("%s_%s", toToken, fromToken)
	if pair, ok := cs.pairs[reverseKey]; ok && pair.Enabled {
		return pair.InverseRate, pair.Fee, nil
	}

	// Through USDT
	fromUSDTKey := fmt.Sprintf("%s_USDT", fromToken)
	toUSDTKey := fmt.Sprintf("USDT_%s", toToken)

	var fromRate, toRate float64
	var fromFee, toFee float64

	if fromPair, ok := cs.pairs[fromUSDTKey]; ok && fromPair.Enabled {
		fromRate = fromPair.Rate
		fromFee = fromPair.Fee
	} else if fromReverse, ok := cs.pairs["USDT_"+fromToken]; ok && fromReverse.Enabled {
		fromRate = fromReverse.InverseRate
		fromFee = fromReverse.Fee
	} else {
		return 0, 0, fmt.Errorf("no conversion path")
	}

	if toPair, ok := cs.pairs[toUSDTKey]; ok && toPair.Enabled {
		toRate = toPair.Rate
		toFee = toPair.Fee
	} else if toReverse, ok := cs.pairs[toToken+"_USDT"]; ok && toReverse.Enabled {
		toRate = toReverse.InverseRate
		toFee = toReverse.Fee
	} else {
		return 0, 0, fmt.Errorf("no conversion path")
	}

	return fromRate * toRate, (fromFee + toFee) / 2, nil
}

func (cs *ConvertService) GetUserOrders(userID string) []ConvertOrder {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.orders[userID]
}

func (cs *ConvertService) GetUserBalance(userID string) *UserBalance {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	if balance, ok := cs.balances[userID]; ok {
		return balance
	}

	// Create default balance
	defaultBalance := &UserBalance{
		UserID: userID,
		Balances: map[string]float64{
			"USDT": 50000,
			"USDC": 25000,
			"BTC":   0.5,
			"ETH":   5,
			"BNB":   50,
			"SOL":   100,
		},
	}
	cs.balances[userID] = defaultBalance
	return defaultBalance
}

func (cs *ConvertService) EnablePair(fromToken, toToken string, enabled bool) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	key := fmt.Sprintf("%s_%s", fromToken, toToken)
	pair, ok := cs.pairs[key]
	if !ok {
		return fmt.Errorf("pair not found: %s/%s", fromToken, toToken)
	}

	pair.Enabled = enabled
	return nil
}

func (cs *ConvertService) UpdatePairFee(fromToken, toToken string, fee float64) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	key := fmt.Sprintf("%s_%s", fromToken, toToken)
	pair, ok := cs.pairs[key]
	if !ok {
		return fmt.Errorf("pair not found: %s/%s", fromToken, toToken)
	}

	pair.Fee = fee
	return nil
}

func (cs *ConvertService) GetSettings() ConvertSettings {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.settings
}

func (cs *ConvertService) UpdateSettings(settings ConvertSettings) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.settings = settings
}

func (cs *ConvertService) ToJSON() (string, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	data := struct {
		Pairs   map[string]*ConvertPair      `json:"pairs"`
		Orders  map[string][]ConvertOrder   `json:"orders"`
		Balances map[string]*UserBalance    `json:"balances"`
		Settings ConvertSettings            `json:"settings"`
	}{
		Pairs:    cs.pairs,
		Orders:   cs.orders,
		Balances: cs.balances,
		Settings: cs.settings,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}

// Helper function for math operations
func round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

func toFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(round(num*output)) / output
}
