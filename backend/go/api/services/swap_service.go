package services

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"tigerwallet/backend/go/api/models"
)

type SwapService struct {
	mu    sync.RWMutex
	quotes map[string]*models.SwapQuote
	swaps  map[string]*models.Swap
}

var (
	swapInstance *SwapService
	swapOnce    sync.Once
)

func NewSwapService() *SwapService {
	swapOnce.Do(func() {
		swapInstance = &SwapService{
			quotes: make(map[string]*models.SwapQuote),
			swaps:  make(map[string]*models.Swap),
		}
	})
	return swapInstance
}

func (s *SwapService) GetQuote(ctx context.Context, req *models.SwapRequest) (*models.SwapQuote, error) {
	// Calculate quote based on current prices
	fromPrice := s.getTokenPrice(req.FromToken)
	toPrice := s.getTokenPrice(req.ToToken)

	if fromPrice == 0 || toPrice == 0 {
		return nil, errors.New("token price not available")
	}

	amountFloat := new(big.Float).SetString(req.FromAmount)
	if amountFloat == nil {
		return nil, errors.New("invalid amount")
	}

	// Calculate output amount (simplified - real implementation would query DEXes)
	rate := toPrice / fromPrice
	amountFloat.Mul(amountFloat, big.NewFloat(rate))
	outputAmount, _ := amountFloat.Float64()

	slippage := req.SlippageTolerance / 100.0
	minOutput := outputAmount * (1 - slippage)

	quoteID := fmt.Sprintf("quote_%d", time.Now().UnixNano())

	quote := &models.SwapQuote{
		ID:                quoteID,
		FromToken:         req.FromToken,
		ToToken:           req.ToToken,
		FromAmount:        req.FromAmount,
		ToAmount:          fmt.Sprintf("%f", outputAmount),
		ToAmountUSD:       outputAmount * toPrice,
		PriceImpact:       0.1,
		GuaranteedPrice:   fmt.Sprintf("%f", minOutput),
		Route:             []string{"TigerSwap", "Uniswap", "Curve"},
		AllowanceTarget:   "0x0000000000000000000000000000000000000000",
		TxData:           "0x",
		ValidityPeriod:    300,
		GasEstimate:       "150000",
		CreatedAt:          time.Now(),
	}

	s.mu.Lock()
	s.quotes[quoteID] = quote
	s.mu.Unlock()

	return quote, nil
}

func (s *SwapService) getTokenPrice(symbol string) float64 {
	// Simplified price lookup - in production, use price oracle
	prices := map[string]float64{
		"ETH":  3450.0,
		"BTC":  67500.0,
		"USDT": 1.0,
		"USDC": 1.0,
		"BNB":  580.0,
		"SOL":  145.0,
		"TRX":  0.12,
		"DOGE": 0.12,
		"ADA":  0.45,
		"XRP":  0.62,
		"DOT":  7.5,
		"AVAX": 35.0,
		"LINK": 14.5,
		"UNI":  9.8,
		"MATIC": 0.58,
		"ARB":  1.1,
		"OP":   2.8,
		"ATOM": 8.2,
		"NEAR": 5.2,
		"APT":  9.5,
	}

	return prices[symbol]
}

func (s *SwapService) ExecuteSwap(ctx context.Context, walletID, quoteID string) (*models.Swap, error) {
	s.mu.RLock()
	quote, ok := s.quotes[quoteID]
	s.mu.RUnlock()

	if !ok {
		return nil, errors.New("quote not found")
	}

	swapID := fmt.Sprintf("swap_%d", time.Now().UnixNano())

	swap := &models.Swap{
		ID:            swapID,
		WalletID:      walletID,
		QuoteID:       quoteID,
		FromToken:     quote.FromToken,
		ToToken:       quote.ToToken,
		FromAmount:    quote.FromAmount,
		ToAmount:      quote.ToAmount,
		Status:        "pending",
		TransactionID: "",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	s.mu.Lock()
	s.swaps[swapID] = swap
	s.mu.Unlock()

	return swap, nil
}

func (s *SwapService) GetSwapByID(ctx context.Context, swapID string) (*models.Swap, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	swap, ok := s.swaps[swapID]
	if !ok {
		return nil, errors.New("swap not found")
	}

	return swap, nil
}

func (s *SwapService) GetSwapsByWalletID(ctx context.Context, walletID string) ([]*models.Swap, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*models.Swap
	for _, swap := range s.swaps {
		if swap.WalletID == walletID {
			result = append(result, swap)
		}
	}

	return result, nil
}

func (s *SwapService) UpdateSwapStatus(ctx context.Context, swapID, status, txID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	swap, ok := s.swaps[swapID]
	if !ok {
		return errors.New("swap not found")
	}

	swap.Status = status
	swap.TransactionID = txID
	swap.UpdatedAt = time.Now()

	return nil
}
