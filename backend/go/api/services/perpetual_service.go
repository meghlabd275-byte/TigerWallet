package services

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"tigerwallet/backend/go/api/models"
)

type PerpetualService struct {
	mu        sync.RWMutex
	markets   map[string]*models.PerpetualMarket
	positions map[string]*models.PerpetualPosition
	orders    map[string]*models.PerpetualOrder
}

var (
	perpInstance *PerpetualService
	perpOnce    sync.Once
)

func NewPerpetualService() *PerpetualService {
	perpOnce.Do(func() {
		perpInstance = &PerpetualService{
			markets:   make(map[string]*models.PerpetualMarket),
			positions: make(map[string]*models.PerpetualPosition),
			orders:    make(map[string]*models.PerpetualOrder),
		}
		perpInstance.initializeDefaultMarkets()
	})
	return perpInstance
}

func (s *PerpetualService) initializeDefaultMarkets() {
	defaultMarkets := []*models.PerpetualMarket{
		{Symbol: "BTC-PERP", DisplayName: "BTC/USDT Perpetual", IndexPrice: "67500", MarkPrice: "67550", LastPrice: "67550", Change24h: 2.5, ChangePercent24h: 0.0037, High24h: "68000", Low24h: "66000", Volume24h: "1500000000", OpenInterest: "500000000", FundingRate: "0.0001", NextFundingTime: time.Now().Add(8 * time.Hour).Format(time.RFC3339), MaxLeverage: 100, MinMargin: "50", LiquidationFee: "0.005"},
		{Symbol: "ETH-PERP", DisplayName: "ETH/USDT Perpetual", IndexPrice: "3450", MarkPrice: "3455", LastPrice: "3455", Change24h: 50, ChangePercent24h: 0.0145, High24h: "3500", Low24h: "3400", Volume24h: "800000000", OpenInterest: "200000000", FundingRate: "0.0001", NextFundingTime: time.Now().Add(8 * time.Hour).Format(time.RFC3339), MaxLeverage: 100, MinMargin: "10", LiquidationFee: "0.005"},
		{Symbol: "SOL-PERP", DisplayName: "SOL/USDT Perpetual", IndexPrice: "145", MarkPrice: "145.5", LastPrice: "145.5", Change24h: 5, ChangePercent24h: 0.0345, High24h: "150", Low24h: "140", Volume24h: "300000000", OpenInterest: "100000000", FundingRate: "0.0001", NextFundingTime: time.Now().Add(8 * time.Hour).Format(time.RFC3339), MaxLeverage: 50, MinMargin: "1", LiquidationFee: "0.005"},
		{Symbol: "BNB-PERP", DisplayName: "BNB/USDT Perpetual", IndexPrice: "580", MarkPrice: "581", LastPrice: "581", Change24h: 10, ChangePercent24h: 0.0172, High24h: "590", Low24h: "570", Volume24h: "150000000", OpenInterest: "50000000", FundingRate: "0.0001", NextFundingTime: time.Now().Add(8 * time.Hour).Format(time.RFC3339), MaxLeverage: 50, MinMargin: "5", LiquidationFee: "0.005"},
		{Symbol: "XRP-PERP", DisplayName: "XRP/USDT Perpetual", IndexPrice: "0.62", MarkPrice: "0.621", LastPrice: "0.621", Change24h: 0.02, ChangePercent24h: 0.0323, High24h: "0.63", Low24h: "0.60", Volume24h: "200000000", OpenInterest: "80000000", FundingRate: "0.0001", NextFundingTime: time.Now().Add(8 * time.Hour).Format(time.RFC3339), MaxLeverage: 50, MinMargin: "1", LiquidationFee: "0.005"},
		{Symbol: "ADA-PERP", DisplayName: "ADA/USDT Perpetual", IndexPrice: "0.45", MarkPrice: "0.451", LastPrice: "0.451", Change24h: 0.01, ChangePercent24h: 0.0222, High24h: "0.46", Low24h: "0.44", Volume24h: "100000000", OpenInterest: "40000000", FundingRate: "0.0001", NextFundingTime: time.Now().Add(8 * time.Hour).Format(time.RFC3339), MaxLeverage: 50, MinMargin: "1", LiquidationFee: "0.005"},
		{Symbol: "DOGE-PERP", DisplayName: "DOGE/USDT Perpetual", IndexPrice: "0.12", MarkPrice: "0.1205", LastPrice: "0.1205", Change24h: 0.005, ChangePercent24h: 0.0417, High24h: "0.125", Low24h: "0.115", Volume24h: "150000000", OpenInterest: "60000000", FundingRate: "0.0001", NextFundingTime: time.Now().Add(8 * time.Hour).Format(time.RFC3339), MaxLeverage: 50, MinMargin: "1", LiquidationFee: "0.005"},
		{Symbol: "DOT-PERP", DisplayName: "DOT/USDT Perpetual", IndexPrice: "7.5", MarkPrice: "7.52", LastPrice: "7.52", Change24h: 0.2, ChangePercent24h: 0.0267, High24h: "7.7", Low24h: "7.3", Volume24h: "80000000", OpenInterest: "30000000", FundingRate: "0.0001", NextFundingTime: time.Now().Add(8 * time.Hour).Format(time.RFC3339), MaxLeverage: 50, MinMargin: "5", LiquidationFee: "0.005"},
		{Symbol: "LINK-PERP", DisplayName: "LINK/USDT Perpetual", IndexPrice: "14.5", MarkPrice: "14.55", LastPrice: "14.55", Change24h: 0.5, ChangePercent24h: 0.0345, High24h: "15.0", Low24h: "14.0", Volume24h: "100000000", OpenInterest: "40000000", FundingRate: "0.0001", NextFundingTime: time.Now().Add(8 * time.Hour).Format(time.RFC3339), MaxLeverage: 50, MinMargin: "5", LiquidationFee: "0.005"},
		{Symbol: "AVAX-PERP", DisplayName: "AVAX/USDT Perpetual", IndexPrice: "35", MarkPrice: "35.1", LastPrice: "35.1", Change24h: 1, ChangePercent24h: 0.0286, High24h: "36", Low24h: "34", Volume24h: "120000000", OpenInterest: "50000000", FundingRate: "0.0001", NextFundingTime: time.Now().Add(8 * time.Hour).Format(time.RFC3339), MaxLeverage: 50, MinMargin: "5", LiquidationFee: "0.005"},
		{Symbol: "MATIC-PERP", DisplayName: "MATIC/USDT Perpetual", IndexPrice: "0.58", MarkPrice: "0.581", LastPrice: "0.581", Change24h: 0.02, ChangePercent24h: 0.0345, High24h: "0.60", Low24h: "0.56", Volume24h: "80000000", OpenInterest: "30000000", FundingRate: "0.0001", NextFundingTime: time.Now().Add(8 * time.Hour).Format(time.RFC3339), MaxLeverage: 50, MinMargin: "1", LiquidationFee: "0.005"},
		{Symbol: "UNI-PERP", DisplayName: "UNI/USDT Perpetual", IndexPrice: "9.8", MarkPrice: "9.82", LastPrice: "9.82", Change24h: 0.3, ChangePercent24h: 0.0306, High24h: "10.0", Low24h: "9.5", Volume24h: "60000000", OpenInterest: "25000000", FundingRate: "0.0001", NextFundingTime: time.Now().Add(8 * time.Hour).Format(time.RFC3339), MaxLeverage: 50, MinMargin: "5", LiquidationFee: "0.005"},
		{Symbol: "ATOM-PERP", DisplayName: "ATOM/USDT Perpetual", IndexPrice: "8.2", MarkPrice: "8.22", LastPrice: "8.22", Change24h: 0.3, ChangePercent24h: 0.0366, High24h: "8.5", Low24h: "7.9", Volume24h: "70000000", OpenInterest: "28000000", FundingRate: "0.0001", NextFundingTime: time.Now().Add(8 * time.Hour).Format(time.RFC3339), MaxLeverage: 50, MinMargin: "5", LiquidationFee: "0.005"},
		{Symbol: "NEAR-PERP", DisplayName: "NEAR/USDT Perpetual", IndexPrice: "5.2", MarkPrice: "5.22", LastPrice: "5.22", Change24h: 0.2, ChangePercent24h: 0.0385, High24h: "5.4", Low24h: "5.0", Volume24h: "90000000", OpenInterest: "35000000", FundingRate: "0.0001", NextFundingTime: time.Now().Add(8 * time.Hour).Format(time.RFC3339), MaxLeverage: 50, MinMargin: "1", LiquidationFee: "0.005"},
		{Symbol: "APT-PERP", DisplayName: "APT/USDT Perpetual", IndexPrice: "9.5", MarkPrice: "9.53", LastPrice: "9.53", Change24h: 0.3, ChangePercent24h: 0.0316, High24h: "9.8", Low24h: "9.2", Volume24h: "80000000", OpenInterest: "32000000", FundingRate: "0.0001", NextFundingTime: time.Now().Add(8 * time.Hour).Format(time.RFC3339), MaxLeverage: 50, MinMargin: "5", LiquidationFee: "0.005"},
		{Symbol: "ARB-PERP", DisplayName: "ARB/USDT Perpetual", IndexPrice: "1.1", MarkPrice: "1.102", LastPrice: "1.102", Change24h: 0.04, ChangePercent24h: 0.0364, High24h: "1.14", Low24h: "1.06", Volume24h: "100000000", OpenInterest: "40000000", FundingRate: "0.0001", NextFundingTime: time.Now().Add(8 * time.Hour).Format(time.RFC3339), MaxLeverage: 50, MinMargin: "1", LiquidationFee: "0.005"},
		{Symbol: "OP-PERP", DisplayName: "OP/USDT Perpetual", IndexPrice: "2.8", MarkPrice: "2.81", LastPrice: "2.81", Change24h: 0.1, ChangePercent24h: 0.0357, High24h: "2.9", Low24h: "2.7", Volume24h: "70000000", OpenInterest: "28000000", FundingRate: "0.0001", NextFundingTime: time.Now().Add(8 * time.Hour).Format(time.RFC3339), MaxLeverage: 50, MinMargin: "1", LiquidationFee: "0.005"},
		{Symbol: "INJ-PERP", DisplayName: "INJ/USDT Perpetual", IndexPrice: "24", MarkPrice: "24.1", LastPrice: "24.1", Change24h: 1, ChangePercent24h: 0.0417, High24h: "25", Low24h: "23", Volume24h: "80000000", OpenInterest: "35000000", FundingRate: "0.0001", NextFundingTime: time.Now().Add(8 * time.Hour).Format(time.RFC3339), MaxLeverage: 50, MinMargin: "10", LiquidationFee: "0.005"},
		{Symbol: "LDO-PERP", DisplayName: "LDO/USDT Perpetual", IndexPrice: "2.2", MarkPrice: "2.21", LastPrice: "2.21", Change24h: 0.1, ChangePercent24h: 0.0455, High24h: "2.3", Low24h: "2.1", Volume24h: "60000000", OpenInterest: "25000000", FundingRate: "0.0001", NextFundingTime: time.Now().Add(8 * time.Hour).Format(time.RFC3339), MaxLeverage: 50, MinMargin: "1", LiquidationFee: "0.005"},
		{Symbol: "PEPE-PERP", DisplayName: "PEPE/USDT Perpetual", IndexPrice: "0.000007", MarkPrice: "0.0000071", LastPrice: "0.0000071", Change24h: 0.000001, ChangePercent24h: 0.1429, High24h: "0.000008", Low24h: "0.000006", Volume24h: "200000000", OpenInterest: "80000000", FundingRate: "0.0001", NextFundingTime: time.Now().Add(8 * time.Hour).Format(time.RFC3339), MaxLeverage: 50, MinMargin: "10", LiquidationFee: "0.005"},
	}

	for _, market := range defaultMarkets {
		s.markets[market.Symbol] = market
	}
}

func (s *PerpetualService) GetMarkets(ctx context.Context) ([]*models.PerpetualMarket, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*models.PerpetualMarket
	for _, market := range s.markets {
		result = append(result, market)
	}

	return result, nil
}

func (s *PerpetualService) GetMarketBySymbol(ctx context.Context, symbol string) (*models.PerpetualMarket, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	market, ok := s.markets[symbol]
	if !ok {
		return nil, errors.New("market not found")
	}

	return market, nil
}

func (s *PerpetualService) OpenPosition(ctx context.Context, req *models.OpenPositionRequest) (*models.PerpetualPosition, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	positionID := fmt.Sprintf("position_%d", time.Now().UnixNano())

	position := &models.PerpetualPosition{
		ID:              positionID,
		WalletID:        req.WalletID,
		Symbol:          req.Symbol,
		Side:            req.Side,
		Size:            req.Size,
		EntryPrice:      req.EntryPrice,
		MarkPrice:       req.EntryPrice,
		LiquidationPrice: req.LiquidationPrice,
		Margin:          req.Margin,
		MarginRatio:     0.1,
		UnrealizedPnL:   "0",
		RealizedPnL:     "0",
		FundingPayment:  "0",
		Leverage:        req.Leverage,
		Status:          "open",
		OpenedAt:        time.Now(),
		UpdatedAt:       time.Now(),
	}

	s.positions[positionID] = position

	return position, nil
}

func (s *PerpetualService) ClosePosition(ctx context.Context, positionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	position, ok := s.positions[positionID]
	if !ok {
		return errors.New("position not found")
	}

	position.Status = "closed"
	position.UpdatedAt = time.Now()

	return nil
}

func (s *PerpetualService) GetPositions(ctx context.Context, walletID string) ([]*models.PerpetualPosition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*models.PerpetualPosition
	for _, position := range s.positions {
		if position.WalletID == walletID && position.Status == "open" {
			result = append(result, position)
		}
	}

	return result, nil
}

func (s *PerpetualService) CreateOrder(ctx context.Context, req *models.CreatePerpetualOrderRequest) (*models.PerpetualOrder, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	orderID := fmt.Sprintf("order_%d", time.Now().UnixNano())

	order := &models.PerpetualOrder{
		ID:          orderID,
		WalletID:    req.WalletID,
		Symbol:      req.Symbol,
		Side:        req.Side,
		OrderType:   req.OrderType,
		Size:        req.Size,
		Price:       req.Price,
		TriggerPrice: req.TriggerPrice,
		Margin:      req.Margin,
		Leverage:    req.Leverage,
		Status:      "pending",
		FilledSize:  "0",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	s.orders[orderID] = order

	return order, nil
}

func (s *PerpetualService) CancelOrder(ctx context.Context, orderID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	order, ok := s.orders[orderID]
	if !ok {
		return errors.New("order not found")
	}

	order.Status = "cancelled"
	order.UpdatedAt = time.Now()

	return nil
}

func (s *PerpetualService) GetOrders(ctx context.Context, walletID string) ([]*models.PerpetualOrder, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*models.PerpetualOrder
	for _, order := range s.orders {
		if order.WalletID == walletID {
			result = append(result, order)
		}
	}

	return result, nil
}

// Request/Response types
type OpenPositionRequest struct {
	WalletID          string
	Symbol            string
	Side              string
	Size              string
	EntryPrice        string
	LiquidationPrice string
	Margin            string
	Leverage          int
}

type CreatePerpetualOrderRequest struct {
	WalletID      string
	Symbol        string
	Side          string
	OrderType     string
	Size          string
	Price         string
	TriggerPrice  string
	Margin        string
	Leverage      int
}
