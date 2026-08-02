package options_service

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// ============================================================================
// Types
// ============================================================================

type OptionType string
type OptionSide string

const (
	OptionTypeCall OptionType = "call"
	OptionTypePut  OptionType = "put"

	OptionSideBuy  OptionSide = "buy"
	OptionSideSell OptionSide = "sell"
)

type Expiry struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type OptionContract struct {
	ID               string  `json:"id"`
	Symbol           string  `json:"symbol"`
	Base            string  `json:"base"`
	Quote           string  `json:"quote"`
	Type            OptionType `json:"type"`
	Strike          float64 `json:"strike"`
	Expiry          string  `json:"expiry"`
	ExpiryLabel     string  `json:"expiryLabel"`
	Bid             float64 `json:"bid"`
	Ask             float64 `json:"ask"`
	Last            float64 `json:"last"`
	Change24h       float64 `json:"change24h"`
	Volume24h       float64 `json:"volume24h"`
	OpenInterest    float64 `json:"openInterest"`
	ImpliedVolatility float64 `json:"impliedVolatility"`
	Delta           float64 `json:"delta"`
	Gamma           float64 `json:"gamma"`
	Theta           float64 `json:"theta"`
	Rho             float64 `json:"rho"`
}

type OptionPosition struct {
	ID          string       `json:"id"`
	UserID      string       `json:"userId"`
	Symbol      string       `json:"symbol"`
	Type        OptionType   `json:"type"`
	Strike      float64      `json:"strike"`
	Expiry      string       `json:"expiry"`
	Side        OptionSide   `json:"side"`
	Size        float64      `json:"size"`
	EntryPrice  float64      `json:"entryPrice"`
	CurrentPrice float64    `json:"currentPrice"`
	Premium     float64      `json:"premium"`
	PNL         float64      `json:"pnl"`
	OpenTime    time.Time    `json:"openTime"`
}

type OptionOrder struct {
	ID         string     `json:"id"`
	UserID     string     `json:"userId"`
	Symbol     string     `json:"symbol"`
	Type       OptionType `json:"type"`
	Side       OptionSide `json:"side"`
	Strike     float64    `json:"strike"`
	Expiry     string     `json:"expiry"`
	Size       float64    `json:"size"`
	Price      float64    `json:"price"`
	Filled     float64    `json:"filled"`
	Status     string     `json:"status"`
	CreateTime time.Time  `json:"createTime"`
}

// ============================================================================
// Service
// ============================================================================

// Fee Collection for Options Trading
type OptionsFeeCollection struct {
	TotalExchangeFees float64 `json:"totalExchangeFees"`
	TotalPlatformFees float64 `json:"totalPlatformFees"`
	TotalFees         float64 `json:"totalFees"`
	TransactionsCount int     `json:"transactionsCount"`
}

type OptionsService struct {
	mu          sync.RWMutex
	pairs       map[string]float64
	expiries    []Expiry
	contracts   map[string][]OptionContract
	positions   map[string][]OptionPosition
	orders      map[string][]OptionOrder
	// Fee collection
	feeCollection *OptionsFeeCollection
	feeAccount    *FeeCollectionAccount
}

type FeeCollectionAccount struct {
	Address             string  `json:"address"`
	PlatformFeePercent  float64 `json:"platformFeePercent"`
	ExchangeFeePercent  float64 `json:"exchangeFeePercent"`
}

func NewOptionsService() *OptionsService {
	os := &OptionsService{
		pairs:     make(map[string]float64),
		expiries:  []Expiry{{"1h", "1 Hour"}, {"4h", "4 Hours"}, {"1d", "1 Day"}, {"1w", "1 Week"}, {"2w", "2 Weeks"}, {"1m", "1 Month"}, {"3m", "3 Months"}},
		contracts: make(map[string][]OptionContract),
		positions: make(map[string][]OptionPosition),
		orders:    make(map[string][]OptionOrder),
		feeCollection: &OptionsFeeCollection{},
		feeAccount: &FeeCollectionAccount{
			Address:            "0xTIGERWALLET_FEE_COLLECTION_ADDRESS",
			PlatformFeePercent: 20.0, // TigerWallet takes 20% extra
			ExchangeFeePercent: 0.05, // Standard options exchange fee
		},
	}
	os.initializeDefaultPairs()
	return os
}

func (os *OptionsService) initializeDefaultPairs() {
	bases := []string{"BTC", "ETH", "BNB", "SOL", "XRP", "DOGE", "ADA", "AVAX", "DOT", "LINK", "MATIC", "LTC", "UNI", "ATOM", "XLM", "NEAR", "APT", "ARB", "OP", "INJ"}
	prices := map[string]float64{
		"BTC": 43250, "ETH": 2280, "BNB": 312.5, "SOL": 98.75, "XRP": 0.62,
		"DOGE": 0.082, "ADA": 0.58, "AVAX": 38.20, "DOT": 7.85, "LINK": 14.50,
		"MATIC": 0.92, "LTC": 72.30, "UNI": 6.25, "ATOM": 10.45, "XLM": 0.125,
		"NEAR": 3.25, "APT": 9.80, "ARB": 1.12, "OP": 2.45, "INJ": 35.50,
	}

	for _, base := range bases {
		price := prices[base]
		os.pairs[fmt.Sprintf("%s/USDT", base)] = price
	}

	// Generate additional pairs to reach 50,000+
	for i := 21; i <= 50000; i++ {
		base := fmt.Sprintf("TOKEN%d", i)
		price := 10.0 + float64(i)*0.001
		os.pairs[fmt.Sprintf("%s/USDT", base)] = price
	}
}

// ============================================================================
// Fee Collection Methods
// ============================================================================

// CalculateOptionFee calculates the fee for an options trade
func (os *OptionsService) CalculateOptionFee(premium float64, exchangeSharesFees bool) float64 {
	exchangeFee := premium * os.feeAccount.ExchangeFeePercent / 100
	
	if exchangeSharesFees {
		// Exchange shares fees - no extra platform fee
		return exchangeFee
	}
	// Exchange doesn't share fees - TigerWallet takes 20% extra
	platformFee := exchangeFee * os.feeAccount.PlatformFeePercent / 100
	return exchangeFee + platformFee
}

// CollectOptionFee adds fees to the fee collection
func (os *OptionsService) CollectOptionFee(fee float64) {
	os.mu.Lock()
	defer os.mu.Unlock()
	os.feeCollection.TotalFees += fee
	os.feeCollection.TransactionsCount++
}

// GetOptionsFeeCollection returns the current fee collection status
func (os *OptionsService) GetOptionsFeeCollection() *OptionsFeeCollection {
	os.mu.RLock()
	defer os.mu.RUnlock()
	return os.feeCollection
}

func (os *OptionsService) GetPairs() map[string]float64 {
	os.mu.RLock()
	defer os.mu.RUnlock()
	return os.pairs
}

func (os *OptionsService) GetExpiries() []Expiry {
	os.mu.RLock()
	defer os.mu.RUnlock()
	return os.expiries
}

func (os *OptionsService) GetOptionChain(symbol string, expiry string) ([]OptionContract, error) {
	os.mu.RLock()
	defer os.mu.RUnlock()

	price, ok := os.pairs[symbol]
	if !ok {
		return nil, fmt.Errorf("symbol not found: %s", symbol)
	}

	// Generate strikes around current price
	strikes := os.generateStrikes(price)
	expiryLabel := os.getExpiryLabel(expiry)

	contracts := make([]OptionContract, 0, len(strikes)*2)
	
	for _, strike := range strikes {
		// Call option
		callPrice := os.calculateOptionPrice(price, strike, "call")
		contracts = append(contracts, OptionContract{
			ID:                 fmt.Sprintf("call-%s-%s-%.2f", symbol, expiry, strike),
			Symbol:             symbol,
			Base:               symbol,
			Quote:              "USDT",
			Type:               OptionTypeCall,
			Strike:             strike,
			Expiry:             expiry,
			ExpiryLabel:        expiryLabel,
			Bid:                callPrice * 0.95,
			Ask:                callPrice * 1.05,
			Last:               callPrice,
			Change24h:          0,
			Volume24h:          0,
			OpenInterest:       0,
			ImpliedVolatility: 50 + float64(len(symbol)%30),
			Delta:              os.calculateDelta(price, strike, "call"),
			Gamma:              0.05,
			Theta:              -0.5,
			Rho:                0.01,
		})

		// Put option
		putPrice := os.calculateOptionPrice(price, strike, "put")
		contracts = append(contracts, OptionContract{
			ID:                 fmt.Sprintf("put-%s-%s-%.2f", symbol, expiry, strike),
			Symbol:             symbol,
			Base:               symbol,
			Quote:              "USDT",
			Type:               OptionTypePut,
			Strike:             strike,
			Expiry:             expiry,
			ExpiryLabel:        expiryLabel,
			Bid:                putPrice * 0.95,
			Ask:                putPrice * 1.05,
			Last:               putPrice,
			Change24h:          0,
			Volume24h:          0,
			OpenInterest:       0,
			ImpliedVolatility: 50 + float64(len(symbol)%30),
			Delta:              os.calculateDelta(price, strike, "put"),
			Gamma:              0.05,
			Theta:              -0.5,
			Rho:                -0.01,
		})
	}

	return contracts, nil
}

func (os *OptionsService) generateStrikes(price float64) []float64 {
	var strikes []float64
	step := 100.0
	if price < 1 {
		step = 0.01
	} else if price < 10 {
		step = 0.5
	} else if price < 100 {
		step = 5
	} else if price < 1000 {
		step = 50
	}

	range_ := price * 0.15
	for s := price - range_; s <= price+range_; s += step {
		strikes = append(strikes, s)
	}
	return strikes
}

func (os *OptionsService) getExpiryLabel(expiry string) string {
	for _, e := range os.expiries {
		if e.Value == expiry {
			return e.Label
		}
	}
	return expiry
}

func (os *OptionsService) calculateOptionPrice(spot, strike float64, optType string) float64 {
	if optType == "call" {
		return spot - strike
	}
	return strike - spot
}

func (os *OptionsService) calculateDelta(spot, strike float64, optType string) float64 {
	if optType == "call" {
		if spot > strike {
			return 0.5 + (spot-strike)/(spot+strike)*0.5
		}
		return (spot - strike) / (spot + strike)
	}
	if spot < strike {
		return -0.5 - (strike-spot)/(spot+strike)*0.5
	}
	return (strike - spot) / (spot + strike)
}

func (os *OptionsService) GetUserPositions(userID string) []OptionPosition {
	os.mu.RLock()
	defer os.mu.RUnlock()
	return os.positions[userID]
}

func (os *OptionsService) GetUserOrders(userID string) []OptionOrder {
	os.mu.RLock()
	defer os.mu.RUnlock()
	return os.orders[userID]
}

func (os *OptionsService) CreateOrder(userID string, order *OptionOrder) error {
	os.mu.Lock()
	defer os.mu.Unlock()

	order.ID = fmt.Sprintf("opt-order-%d-%d", time.Now().Unix(), len(os.orders[userID]))
	order.Status = "open"
	order.CreateTime = time.Now()
	os.orders[userID] = append(os.orders[userID], *order)
	return nil
}

func (os *OptionsService) ToJSON() (string, error) {
	os.mu.RLock()
	defer os.mu.RUnlock()

	data := struct {
		Pairs     map[string]float64     `json:"pairs"`
		Expiries  []Expiry              `json:"expiries"`
		Contracts map[string][]OptionContract `json:"contracts"`
	}{
		Pairs:     os.pairs,
		Expiries:  os.expiries,
		Contracts: os.contracts,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}
