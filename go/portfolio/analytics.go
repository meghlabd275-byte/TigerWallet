package portfolio

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sort"
	"sync"
	"time"
)

// ============================================================================
// Portfolio Analytics Service
// ============================================================================

// Service provides portfolio analytics
type Service struct {
	mu          sync.RWMutex
	balances    map[string]map[string]*Balance
	transactions map[string][]Transaction
	prices      map[string]map[string]*Price
	config      *Config
}

// Config for portfolio service
type Config struct {
	PriceUpdateInterval time.Duration
	HistoryRetention  time.Duration
}

// Balance represents token balance
type Balance struct {
	Address   string
	Symbol    string
	ChainID   uint64
	Balance   *big.Int
	Decimals  uint8
	ValueUSD  *big.Rat
	CostBasis *big.Rat
}

// Transaction represents a transaction
type Transaction struct {
	TxHash       string
	ChainID      uint64
	Address      string
	Token       string
	Type        TxType
	Amount      *big.Int
	ValueUSD    *big.Rat
	FeeUSD      *big.Rat
	Timestamp   time.Time
	BlockNumber uint64
	Status     TxStatus
}

// TxType enum
type TxType string

const (
	TxTypeTransfer TxType = "transfer"
	TxTypeSwap    TxType = "swap"
	TxTypeStake   TxType = "stake"
	TxTypeMint   TxType = "mint"
	TxTypeBurn   TxType = "burn"
)

// TxStatus enum
type TxStatus string

const (
	TxStatusPending   TxStatus = "pending"
	TxStatusConfirmed TxStatus = "confirmed"
	TxStatusFailed  TxStatus = "failed"
)

// Price represents price data
type Price struct {
	Symbol    string
	PriceUSD *big.Rat
	Change24h float64
	Volume24h *big.Rat
	UpdatedAt time.Time
}

// Portfolio represents user portfolio
type Portfolio struct {
	UserID       string
	TotalValueUSD *big.Rat
	Balances    []Balance
	PnL         *PnL
	History     []HistoryPoint
}

// PnL represents profit and loss
type PnL struct {
	TotalValue   *big.Rat
	TotalCost   *big.Rat
	Unrealized   *big.Rat
	Realized    *big.Rat
	Change24h   *big.Rat
	Change24hPct float64
}

// HistoryPoint represents portfolio value over time
type HistoryPoint struct {
	Timestamp time.Time
	ValueUSD  *big.Rat
}

// NewService creates new portfolio service
func NewService(cfg *Config) *Service {
	return &Service{
		balances:     make(map[string]map[string]*Balance),
		transactions: make(map[string][]Transaction),
		prices:      make(map[string]map[string]*Price),
		config:      cfg,
	}
}

// ============================================================================
// Balance Management
// ============================================================================

// UpdateBalance updates a balance
func (s *Service) UpdateBalance(userID, address, symbol string, chainID uint64, balance *big.Int, decimals uint8) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.balances[userID] == nil {
		s.balances[userID] = make(map[string]*Balance)
	}
	
	s.balances[userID][address] = &Balance{
		Address:  address,
		Symbol:  symbol,
		ChainID: chainID,
		Balance: balance,
		Decimals: decimals,
	}
}

// UpdatePrices updates prices
func (s *Service) UpdatePrices(chainID uint64, prices map[string]*Price) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	chainKey := fmt.Sprintf("%d", chainID)
	s.prices[chainKey] = prices
}

// GetPortfolio gets user portfolio
func (s *Service) GetPortfolio(userID string) (*Portfolio, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	balances, ok := s.balances[userID]
	if !ok {
		return &Portfolio{
			UserID:       userID,
			TotalValueUSD: big.NewRat(0, 1),
			Balances:    []Balance{},
			PnL: &PnL{
				TotalValue: big.NewRat(0, 1),
				TotalCost:  big.NewRat(0, 1),
			},
		}, nil
	}
	
	var totalValue big.Rat
	var totalCost big.Rat
	var portfolioBalances []Balance
	
	for _, balance := range balances {
		chainKey := fmt.Sprintf("%d", balance.ChainID)
		priceData, ok := s.prices[chainKey]
		if !ok {
			continue
		}
		
		price, ok := priceData[balance.Symbol]
		if !ok {
			continue
		}
		
		balanceFloat, _ := new(big.Float).SetInt(balance.Balance).Float64()
		priceFloat, _ := price.PriceUSD.Float64()
		decimalsFloat := float64(balance.Decimals)
		
		valueUSD := big.NewRat(int64(balanceFloat*priceFloat*decimalsFloat*1000000), 1000000)
		
		balance.ValueUSD = valueUSD
		totalValue.Add(&totalValue, valueUSD)
		
		if balance.CostBasis != nil {
			totalCost.Add(&totalCost, balance.CostBasis)
		}
		
		portfolioBalances = append(portfolioBalances, *balance)
	}
	
	var unrealized big.Rat
	unrealized.Sub(&totalValue, &totalCost)
	
	change24h := big.NewRat(0, 1)
	change24hPct := 0.0
	
	sort.Slice(portfolioBalances, func(i, j int) bool {
		return portfolioBalances[i].ValueUSD.Cmp(portfolioBalances[j].ValueUSD) > 0
	})
	
	return &Portfolio{
		UserID:       userID,
		TotalValueUSD: &totalValue,
		Balances:     portfolioBalances,
		PnL: &PnL{
			TotalValue:   &totalValue,
			TotalCost:   &totalCost,
			Unrealized:  &unrealized,
			Change24h:  change24h,
			Change24hPct: change24hPct,
		},
		History: []HistoryPoint{},
	}, nil
}

// ============================================================================
// Transaction History
// ============================================================================

// AddTransaction adds a transaction
func (s *Service) AddTransaction(userID string, tx Transaction) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.transactions[userID] = append(s.transactions[userID], tx)
}

// GetTransactions gets user transactions
func (s *Service) GetTransactions(userID string, limit int) []Transaction {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	txs := s.transactions[userID]
	if limit > 0 && len(txs) > limit {
		return txs[len(txs)-limit:]
	}
	
	return txs
}

// ============================================================================
// Tax Report
// ============================================================================

// TaxReport represents tax report
type TaxReport struct {
	UserID          string
	Year           int
	TotalProceeds  *big.Rat
	TotalCostBasis *big.Rat
	TotalGain     *big.Rat
	ShortTermGain *big.Rat
	LongTermGain  *big.Rat
	Transactions  []TaxableTransaction
}

// TaxableTransaction represents taxable transaction
type TaxableTransaction struct {
	Date           time.Time
	Type          TxType
	Proceeds      *big.Rat
	CostBasis     *big.Rat
	Gain          *big.Rat
	HoldingPeriod string
}

// GenerateTaxReport generates tax report
func (s *Service) GenerateTaxReport(userID string, year int) (*TaxReport, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	txs := s.transactions[userID]
	
	var proceeds, costBasis, gain, shortTerm, longTerm big.Rat
	var taxableTxns []TaxableTransaction
	
	oneYearAgo := time.Date(year-1, time.January, 1, 0, 0, 0, 0, time.UTC)
	
	for _, tx := range txs {
		if tx.Timestamp.Year() != year {
			continue
		}
		
		if tx.Type == TxTypeTransfer || tx.Type == TxTypeSwap || tx.Type == TxTypeMint {
			proceeds.Add(&proceeds, tx.ValueUSD)
			
			if tx.ValueUSD != nil {
				gain.Add(&gain, tx.ValueUSD)
				
				holdingPeriod := "short"
				if tx.Timestamp.Before(oneYearAgo) {
					holdingPeriod = "long"
					longTerm.Add(&longTerm, tx.ValueUSD)
				} else {
					shortTerm.Add(&shortTerm, tx.ValueUSD)
				}
				
				taxableTxns = append(taxableTxns, TaxableTransaction{
					Date:          tx.Timestamp,
					Type:          tx.Type,
					Proceeds:     tx.ValueUSD,
					HoldingPeriod: holdingPeriod,
				})
			}
		}
	}
	
	var totalGain big.Rat
	totalGain.Sub(&proceeds, &costBasis)
	
	return &TaxReport{
		UserID:          userID,
		Year:           year,
		TotalProceeds:  &proceeds,
		TotalCostBasis: &costBasis,
		TotalGain:      &totalGain,
		ShortTermGain:  &shortTerm,
		LongTermGain:   &longTerm,
		Transactions:   taxableTxns,
	}, nil
}

// ============================================================================
// HTTP Handlers
// ============================================================================

// HandleGetPortfolio handles get portfolio
func (s *Service) HandleGetPortfolio(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")
	if userID == "" {
		http.Error(w, "Missing userId", http.StatusBadRequest)
		return
	}
	
	portfolio, err := s.GetPortfolio(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(portfolio)
}

// HandleGetTransactions handles get transactions
func (s *Service) HandleGetTransactions(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")
	if userID == "" {
		http.Error(w, "Missing userId", http.StatusBadRequest)
		return
	}
	
	txs := s.GetTransactions(userID, 100)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(txs)
}

// HandleGetTaxReport handles get tax report
func (s *Service) HandleGetTaxReport(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")
	year := r.URL.Query().Get("year")
	
	if userID == "" {
		http.Error(w, "Missing userId", http.StatusBadRequest)
		return
	}
	
	var yearInt int
	fmt.Sscanf(year, "%d", &yearInt)
	
	report, err := s.GenerateTaxReport(userID, yearInt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// HandleUpdateBalance handles update balance
func (s *Service) HandleUpdateBalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		UserID  string `json:"userId"`
		Symbol string `json:"symbol"`
		ChainID uint64 `json:"chainId"`
		Balance string `json:"balance"`
		Decimals uint8 `json:"decimals"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	balance := new(big.Int)
	balance.SetString(req.Balance, 10)
	
	s.UpdateBalance(req.UserID, req.Symbol, req.Symbol, req.ChainID, balance, req.Decimals)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// HandleUpdatePrices handles update prices
func (s *Service) HandleUpdatePrices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		ChainID uint64            `json:"chainId"`
		Prices map[string]*Price `json:"prices"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	s.UpdatePrices(req.ChainID, req.Prices)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// Serve starts the portfolio service HTTP server
func (s *Service) Serve(addr string) error {
	http.HandleFunc("/v1/portfolio", s.HandleGetPortfolio)
	http.HandleFunc("/v1/transactions", s.HandleGetTransactions)
	http.HandleFunc("/v1/tax-report", s.HandleGetTaxReport)
	http.HandleFunc("/v1/balance", s.HandleUpdateBalance)
	http.HandleFunc("/v1/prices", s.HandleUpdatePrices)
	
	return http.ListenAndServe(addr, nil)
}