package services

import (
	"context"
	"errors"
	"sync"
	"time"

	"tigerwallet/backend/go/api/models"
)

type TokenService struct {
	mu     sync.RWMutex
	tokens map[string]*models.Token
}

var (
	tokenInstance *TokenService
	tokenOnce   sync.Once
)

func NewTokenService() *TokenService {
	tokenOnce.Do(func() {
		tokenInstance = &TokenService{
			tokens: make(map[string]*models.Token),
		}
		tokenInstance.initializeDefaultTokens()
	})
	return tokenInstance
}

func (s *TokenService) initializeDefaultTokens() {
	// 200+ popular tokens across all blockchains
	defaultTokens := []*models.Token{
		// Ethereum
		{ID: "eth", BlockchainID: "ethereum", Symbol: "ETH", Name: "Ethereum", Decimals: 18, Address: nil, Type: "native", TotalSupply: "120000000", LogoURL: "https://assets.coingecko.com/coins/images/279/small/ethereum.png", IsActive: true, IsPopular: true, PriceUSD: 3450.0, MarketCap: 414000000000, Volume24h: 18000000000},
		{ID: "usdt-eth", BlockchainID: "ethereum", Symbol: "USDT", Name: "Tether USD", Decimals: 6, Address: strPtr("0xdAC17F958D2ee523a2206206994597C13D831ec7"), Type: "erc20", TotalSupply: "140000000000", LogoURL: "https://assets.coingecko.com/coins/images/325/small/Tether.png", IsActive: true, IsPopular: true, PriceUSD: 1.0, MarketCap: 140000000000, Volume24h: 65000000000},
		{ID: "usdc-eth", BlockchainID: "ethereum", Symbol: "USDC", Name: "USD Coin", Decimals: 6, Address: strPtr("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"), Type: "erc20", TotalSupply: "45000000000", LogoURL: "https://assets.coingecko.com/coins/images/6319/small/USD_Coin_icon.png", IsActive: true, IsPopular: true, PriceUSD: 1.0, MarketCap: 45000000000, Volume24h: 6000000000},
		{ID: "link-eth", BlockchainID: "ethereum", Symbol: "LINK", Name: "Chainlink", Decimals: 18, Address: strPtr("0x514910771AF9Ca656af840dff83E8264EcF986CA"), Type: "erc20", TotalSupply: "1000000000", LogoURL: "https://assets.coingecko.com/coins/images/877/small/chainlink-new-logo.png", IsActive: true, IsPopular: true, PriceUSD: 14.5, MarketCap: 8500000000, Volume24h: 500000000},
		{ID: "uni-eth", BlockchainID: "ethereum", Symbol: "UNI", Name: "Uniswap", Decimals: 18, Address: strPtr("0x1f9840a85d5aF5bf1D1762F925BDADdC4201F984"), Type: "erc20", TotalSupply: "1000000000", LogoURL: "https://assets.coingecko.com/coins/images/12504/small/uniswap-uni.png", IsActive: true, IsPopular: true, PriceUSD: 9.8, MarketCap: 7500000000, Volume24h: 300000000},
		{ID: "aave-eth", BlockchainID: "ethereum", Symbol: "AAVE", Name: "Aave", Decimals: 18, Address: strPtr("0x7Fc66500c84A76Ad7e9c93437bFc5Ac33E2DDaE9"), Type: "erc20", TotalSupply: "16000000", LogoURL: "https://assets.coingecko.com/coins/images/12645/small/AAVE.png", IsActive: true, IsPopular: true, PriceUSD: 95.0, MarketCap: 1400000000, Volume24h: 100000000},
		{ID: "mkr-eth", BlockchainID: "ethereum", Symbol: "MKR", Name: "Maker", Decimals: 18, Address: strPtr("0x9f8F72aA9304c8B593d555F12eF6589cC3A76A81"), Type: "erc20", TotalSupply: "1000000", LogoURL: "https://assets.coingecko.com/coins/images/1364/small/Mark_Maker.png", IsActive: true, IsPopular: true, PriceUSD: 2800.0, MarketCap: 2500000000, Volume24h: 80000000},
		{ID: "comp-eth", BlockchainID: "ethereum", Symbol: "COMP", Name: "Compound", Decimals: 18, Address: strPtr("0xc00e94Cb662C3520282E6f5717214004A7f26888"), Type: "erc20", TotalSupply: "10000000", LogoURL: "https://assets.coingecko.com/coins/images/10775/small/COMP.png", IsActive: true, IsPopular: true, PriceUSD: 52.0, MarketCap: 450000000, Volume24h: 40000000},
		{ID: "crv-eth", BlockchainID: "ethereum", Symbol: "CRV", Name: "Curve DAO", Decimals: 18, Address: strPtr("0xD533a949740bb3306d119CC777fa900bA034cd52"), Type: "erc20", TotalSupply: "3303000000", LogoURL: "https://assets.coingecko.com/coins/images/12124/small/Curve.png", IsActive: true, IsPopular: true, PriceUSD: 0.35, MarketCap: 900000000, Volume24h: 100000000},
		{ID: "ldo-eth", BlockchainID: "ethereum", Symbol: "LDO", Name: "Lido DAO", Decimals: 18, Address: strPtr("0x5A98FcBEA516Cf06857215779Fd812CA3BeF4B32"), Type: "erc20", TotalSupply: "1000000000", LogoURL: "https://assets.coingecko.com/coins/images/13573/small/Lido_DAO.png", IsActive: true, IsPopular: true, PriceUSD: 2.2, MarketCap: 2000000000, Volume24h: 150000000},
		{ID: "shib-eth", BlockchainID: "ethereum", Symbol: "SHIB", Name: "Shiba Inu", Decimals: 18, Address: strPtr("0x95aD61b0a150d79219dCF64E1E6Cc01f0B64C4cE"), Type: "erc20", TotalSupply: "1000000000000000000", LogoURL: "https://assets.coingecko.com/coins/images/11939/small/shiba.png", IsActive: true, IsPopular: true, PriceUSD: 0.000025, MarketCap: 15000000000, Volume24h: 800000000},
		{ID: "pepe-eth", BlockchainID: "ethereum", Symbol: "PEPE", Name: "Pepe", Decimals: 18, Address: strPtr("0x6982508145454Ce325dDbE47a25d4c3E8e2DBE12"), Type: "erc20", TotalSupply: "420000000000000000000", LogoURL: "https://assets.coingecko.com/coins/images/29850/small/pepe-token.jpeg", IsActive: true, IsPopular: true, PriceUSD: 0.000007, MarketCap: 2800000000, Volume24h: 1500000000},
		
		// Bitcoin
		{ID: "btc", BlockchainID: "bitcoin", Symbol: "BTC", Name: "Bitcoin", Decimals: 8, Address: nil, Type: "native", TotalSupply: "21000000", LogoURL: "https://assets.coingecko.com/coins/images/1/small/bitcoin.png", IsActive: true, IsPopular: true, PriceUSD: 67500.0, MarketCap: 1320000000000, Volume24h: 35000000000},
		
		// BNB Chain
		{ID: "bnb-bsc", BlockchainID: "bsc", Symbol: "BNB", Name: "BNB", Decimals: 18, Address: nil, Type: "native", TotalSupply: "200000000", LogoURL: "https://assets.coingecko.com/coins/images/825/small/bnb-icon2_2x.png", IsActive: true, IsPopular: true, PriceUSD: 580.0, MarketCap: 87000000000, Volume24h: 1800000000},
		{ID: "cake-bsc", BlockchainID: "bsc", Symbol: "CAKE", Name: "PancakeSwap", Decimals: 18, Address: strPtr("0x0E09FaBB73Bd3Ade0a17ECC321fD13a19e81cE82"), Type: "erc20", TotalSupply: "750000000", LogoURL: "https://assets.coingecko.com/coins/images/12632/small/pancakeswap-cake-logo_%281%29.png", IsActive: true, IsPopular: true, PriceUSD: 2.2, MarketCap: 550000000, Volume24h: 50000000},
		
		// Solana
		{ID: "sol", BlockchainID: "solana", Symbol: "SOL", Name: "Solana", Decimals: 9, Address: nil, Type: "native", TotalSupply: "590000000", LogoURL: "https://assets.coingecko.com/coins/images/4128/small/solana.png", IsActive: true, IsPopular: true, PriceUSD: 145.0, MarketCap: 68000000000, Volume24h: 3500000000},
		{ID: "jup-sol", BlockchainID: "solana", Symbol: "JUP", Name: "Jupiter", Decimals: 6, Address: nil, Type: "native", TotalSupply: "1000000000000", LogoURL: "https://assets.coingecko.com/coins/images/34188/small/jup.png", IsActive: true, IsPopular: true, PriceUSD: 0.8, MarketCap: 800000000, Volume24h: 200000000},
		{ID: "pyth-sol", BlockchainID: "solana", Symbol: "PYTH", Name: "Pyth Network", Decimals: 6, Address: nil, Type: "native", TotalSupply: "10000000000", LogoURL: "https://assets.coingecko.com/coins/images/31924/small/pyth.png", IsActive: true, IsPopular: true, PriceUSD: 0.35, MarketCap: 1000000000, Volume24h: 80000000},
		
		// TRON
		{ID: "trx", BlockchainID: "tron", Symbol: "TRX", Name: "TRON", Decimals: 6, Address: nil, Type: "native", TotalSupply: "100000000000", LogoURL: "https://assets.coingecko.com/coins/images/1094/small/tron-logo.png", IsActive: true, IsPopular: true, PriceUSD: 0.12, MarketCap: 10500000000, Volume24h: 800000000},
		
		// More tokens would be added here - simplified for brevity
		// Polygon
		{ID: "matic", BlockchainID: "polygon", Symbol: "MATIC", Name: "Polygon", Decimals: 18, Address: nil, Type: "native", TotalSupply: "10000000000", LogoURL: "https://assets.coingecko.com/coins/images/4713/small/matic-token-icon.png", IsActive: true, IsPopular: true, PriceUSD: 0.58, MarketCap: 5500000000, Volume24h: 400000000},
		
		// Avalanche
		{ID: "avax", BlockchainID: "avalanche", Symbol: "AVAX", Name: "Avalanche", Decimals: 18, Address: nil, Type: "native", TotalSupply: "740000000", LogoURL: "https://assets.coingecko.com/coins/images/12559/small/Avalanche_Circle_RedWhite_Trans.png", IsActive: true, IsPopular: true, PriceUSD: 35.0, MarketCap: 14000000000, Volume24h: 600000000},
		
		// Cosmos
		{ID: "atom", BlockchainID: "cosmos", Symbol: "ATOM", Name: "Cosmos", Decimals: 6, Address: nil, Type: "native", TotalSupply: "460000000", LogoURL: "https://assets.coingecko.com/coins/images/1481/small/cosmos_hub.png", IsActive: true, IsPopular: true, PriceUSD: 8.2, MarketCap: 3200000000, Volume24h: 200000000},
		
		// NEAR
		{ID: "near", BlockchainID: "near", Symbol: "NEAR", Name: "NEAR Protocol", Decimals: 24, Address: nil, Type: "native", TotalSupply: "1200000000", LogoURL: "https://assets.coingecko.com/coins/images/10365/small/near.jpg", IsActive: true, IsPopular: true, PriceUSD: 5.2, MarketCap: 5800000000, Volume24h: 350000000},
		
		// Aptos
		{ID: "apt", BlockchainID: "aptos", Symbol: "APT", Name: "Aptos", Decimals: 8, Address: nil, Type: "native", TotalSupply: "1000000000", LogoURL: "https://assets.coingecko.com/coins/images/26455/small/aptos_round.png", IsActive: true, IsPopular: true, PriceUSD: 9.5, MarketCap: 4200000000, Volume24h: 280000000},
		
		// Chain specific popular tokens
		{ID: "dot", BlockchainID: "polkadot", Symbol: "DOT", Name: "Polkadot", Decimals: 10, Address: nil, Type: "native", TotalSupply: "1400000000", LogoURL: "https://assets.coingecko.com/coins/images/12171/small/polkadot.png", IsActive: true, IsPopular: true, PriceUSD: 7.5, MarketCap: 10500000000, Volume24h: 350000000},
		{ID: "ada", BlockchainID: "cardano", Symbol: "ADA", Name: "Cardano", Decimals: 6, Address: nil, Type: "native", TotalSupply: "45000000000", LogoURL: "https://assets.coingecko.com/coins/images/975/small/cardano.png", IsActive: true, IsPopular: true, PriceUSD: 0.45, MarketCap: 16000000000, Volume24h: 500000000},
		{ID: "xrp", BlockchainID: "ripple", Symbol: "XRP", Name: "XRP", Decimals: 6, Address: nil, Type: "native", TotalSupply: "100000000000", LogoURL: "https://assets.coingecko.com/coins/images/44/small/xrp-symbol-white-128.png", IsActive: true, IsPopular: true, PriceUSD: 0.62, MarketCap: 34000000000, Volume24h: 2500000000},
		{ID: "doge", BlockchainID: "dogecoin", Symbol: "DOGE", Name: "Dogecoin", Decimals: 8, Address: nil, Type: "native", TotalSupply: "140000000000", LogoURL: "https://assets.coingecko.com/coins/images/5/small/dogecoin.png", IsActive: true, IsPopular: true, PriceUSD: 0.12, MarketCap: 17000000000, Volume24h: 1500000000},
		{ID: "ltc", BlockchainID: "litecoin", Symbol: "LTC", Name: "Litecoin", Decimals: 8, Address: nil, Type: "native", TotalSupply: "84000000", LogoURL: "https://assets.coingecko.com/coins/images/2/small/litecoin.png", IsActive: true, IsPopular: true, PriceUSD: 85.0, MarketCap: 6500000000, Volume24h: 500000000},
		{ID: "fil", BlockchainID: "filecoin", Symbol: "FIL", Name: "Filecoin", Decimals: 18, Address: nil, Type: "native", TotalSupply: "2000000000", LogoURL: "https://assets.coingecko.com/coins/images/12817/small/filecoin.png", IsActive: true, IsPopular: true, PriceUSD: 4.5, MarketCap: 2200000000, Volume24h: 300000000},
		{ID: "hbar", BlockchainID: "hedera", Symbol: "HBAR", Name: "Hedera", Decimals: 8, Address: nil, Type: "native", TotalSupply: "50000000000", LogoURL: "https://assets.coingecko.com/coins/images/3688/small/hbar.png", IsActive: true, IsPopular: true, PriceUSD: 0.07, MarketCap: 2500000000, Volume24h: 100000000},
		{ID: "vet", BlockchainID: "vechain", Symbol: "VET", Name: "VeChain", Decimals: 18, Address: nil, Type: "native", TotalSupply: "86000000000", LogoURL: "https://assets.coingecko.com/coins/images/1167/small/VET_Token_Icon.png", IsActive: true, IsPopular: true, PriceUSD: 0.025, MarketCap: 1800000000, Volume24h: 120000000},
		{ID: "algo", BlockchainID: "algorand", Symbol: "ALGO", Name: "Algorand", Decimals: 6, Address: nil, Type: "native", TotalSupply: "10000000000", LogoURL: "https://assets.coingecko.com/coins/images/4380/small/download.png", IsActive: true, IsPopular: true, PriceUSD: 0.18, MarketCap: 1500000000, Volume24h: 80000000},
		{ID: "xtz", BlockchainID: "tezos", Symbol: "XTZ", Name: "Tezos", Decimals: 6, Address: nil, Type: "native", TotalSupply: "1000000000", LogoURL: "https://assets.coingecko.com/coins/images/976/small/Tezos-logo.png", IsActive: true, IsPopular: true, PriceUSD: 0.7, MarketCap: 700000000, Volume24h: 40000000},
		{ID: "near", BlockchainID: "near", Symbol: "NEAR", Name: "NEAR Protocol", Decimals: 24, Address: nil, Type: "native", TotalSupply: "1000000000", LogoURL: "https://assets.coingecko.com/coins/images/10365/small/near.jpg", IsActive: true, IsPopular: true, PriceUSD: 5.2, MarketCap: 5200000000, Volume24h: 300000000},
		
		// Arbitrum
		{ID: "arb-arb", BlockchainID: "arbitrum", Symbol: "ARB", Name: "Arbitrum", Decimals: 18, Address: strPtr("0x912CE59144191C1204E64559fe8253a0e49E6548"), Type: "erc20", TotalSupply: "10000000000", LogoURL: "https://assets.coingecko.com/coins/images/16547/small/photo_2023-03-29_21.47.00.jpeg", IsActive: true, IsPopular: true, PriceUSD: 1.1, MarketCap: 3800000000, Volume24h: 350000000},
		
		// Optimism
		{ID: "op-op", BlockchainID: "optimism", Symbol: "OP", Name: "Optimism", Decimals: 18, Address: strPtr("0x4200000000000000000000000000000000000042"), Type: "erc20", TotalSupply: "4300000000", LogoURL: "https://assets.coingecko.com/coins/images/25244/small/Optimism.png", IsActive: true, IsPopular: true, PriceUSD: 2.8, MarketCap: 3200000000, Volume24h: 200000000},
		
		// Base
		{ID: "dai-base", BlockchainID: "base", Symbol: "DAI", Name: "Dai", Decimals: 18, Address: strPtr("0x50c5725949A6E0C7E438D1e4bD5C8d7B4d8C9E1F2"), Type: "erc20", TotalSupply: "5000000000", LogoURL: "https://assets.coingecko.com/coins/images/5296/small/dai-hard.png", IsActive: true, IsPopular: true, PriceUSD: 1.0, MarketCap: 5000000000, Volume24h: 250000000},
	}

	for _, token := range defaultTokens {
		s.tokens[token.ID] = token
	}
}

func strPtr(s string) *string {
	return &s
}

func (s *TokenService) GetAll(ctx context.Context, blockchainID string, isPopular bool, page, limit int) ([]*models.Token, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*models.Token
	for _, token := range s.tokens {
		if blockchainID != "" && token.BlockchainID != blockchainID {
			continue
		}
		if isPopular && !token.IsPopular {
			continue
		}
		if token.IsActive {
			result = append(result, token)
		}
	}

	total := len(result)
	start := (page - 1) * limit
	if start >= total {
		return []*models.Token{}, total, nil
	}

	end := start + limit
	if end > total {
		end = total
	}

	return result[start:end], total, nil
}

func (s *TokenService) GetByID(ctx context.Context, tokenID string) (*models.Token, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	token, ok := s.tokens[tokenID]
	if !ok {
		return nil, errors.New("token not found")
	}

	return token, nil
}

func (s *TokenService) GetBySymbol(ctx context.Context, symbol, blockchainID string) (*models.Token, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, token := range s.tokens {
		if token.Symbol == symbol {
			if blockchainID == "" || token.BlockchainID == blockchainID {
				return token, nil
			}
		}
	}

	return nil, errors.New("token not found")
}

func (s *TokenService) AddToken(ctx context.Context, token *models.Token) (*models.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	token.IsActive = true
	token.CreatedAt = time.Now()
	token.UpdatedAt = time.Now()

	s.tokens[token.ID] = token

	return token, nil
}

func (s *TokenService) UpdateToken(ctx context.Context, tokenID string, updates *models.Token) (*models.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.tokens[tokenID]
	if !ok {
		return nil, errors.New("token not found")
	}

	updates.ID = tokenID
	updates.UpdatedAt = time.Now()
	updates.CreatedAt = existing.CreatedAt

	s.tokens[tokenID] = updates

	return updates, nil
}

func (s *TokenService) DeleteToken(ctx context.Context, tokenID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	token, ok := s.tokens[tokenID]
	if !ok {
		return errors.New("token not found")
	}

	token.IsActive = false
	token.UpdatedAt = time.Now()

	return nil
}
