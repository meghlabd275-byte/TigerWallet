package services

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

// Config holds all service configurations
type Config struct {
	DatabaseURL      string
	RedisURL         string
	WalletServiceURL string
	JWTSecret        string
}

// Chain types supported by TigerWallet
type ChainType string

const (
	ChainEthereum    ChainType = "ethereum"
	ChainPolygon     ChainType = "polygon"
	ChainArbitrum    ChainType = "arbitrum"
	ChainOptimism    ChainType = "optimism"
	ChainBase        ChainType = "base"
	ChainAvalanche   ChainType = "avalanche"
	ChainBSC         ChainType = "bsc"
	ChainSolana      ChainType = "solana"
	ChainTron        ChainType = "tron"
	ChainBitcoin     ChainType = "bitcoin"
	ChainCosmos      ChainType = "cosmos"
	ChainPi          ChainType = "pi"
	ChainTon         ChainType = "ton"
	ChainAptos       ChainType = "aptos"
	ChainPulseChain  ChainType = "pulsechain"
)

// ChainConfig represents blockchain configuration
type ChainConfig struct {
	ID            uint64    `db:"id" json:"id"`
	Name          string    `db:"name" json:"name"`
	Symbol        string    `db:"symbol" json:"symbol"`
	ChainType     ChainType `db:"chain_type" json:"chainType"`
	ChainID       uint64    `db:"chain_id" json:"chainId"`
	RPCURL        string    `db:"rpc_url" json:"rpcUrl"`
	ExplorerURL   string    `db:"explorer_url" json:"explorerUrl"`
	IconURL       string    `db:"icon_url" json:"iconUrl"`
	Decimals      uint8     `db:"decimals" json:"decimals"`
	IsActive      bool      `db:"is_active" json:"isActive"`
	GasLimit      uint64    `db:"gas_limit" json:"gasLimit"`
	Confirmations uint64    `db:"confirmations" json:"confirmations"`
	CreatedAt     time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt     time.Time `db:"updated_at" json:"updatedAt"`
}

// TokenConfig represents token configuration
type TokenConfig struct {
	ID            uint64    `db:"id" json:"id"`
	ChainID       uint64    `db:"chain_id" json:"chainId"`
	Address       string    `db:"address" json:"address"`
	Symbol        string    `db:"symbol" json:"symbol"`
	Name          string    `db:"name" json:"name"`
	Decimals      uint8     `db:"decimals" json:"decimals"`
	TotalSupply   string    `db:"total_supply" json:"totalSupply"`
	IconURL       string    `db:"icon_url" json:"iconUrl"`
	IsActive      bool      `db:"is_active" json:"isActive"`
	IsPopular     bool      `db:"is_popular" json:"isPopular"`
	IsStablecoin  bool      `db:"is_stablecoin" json:"isStablecoin"`
	PriceUSD      float64   `db:"price_usd" json:"priceUsd"`
	MarketCap     float64   `db:"market_cap" json:"marketCap"`
	Volume24h     float64   `db:"volume_24h" json:"volume24h"`
	CreatedAt     time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt     time.Time `db:"updated_at" json:"updatedAt"`
}

// Wallet represents a user wallet
type Wallet struct {
	ID           uint64    `db:"id" json:"id"`
	UserID       uint64    `db:"user_id" json:"userId"`
	Address      string    `db:"address" json:"address"`
	ChainType    ChainType `db:"chain_type" json:"chainType"`
	ChainID      uint64    `db:"chain_id" json:"chainId"`
	WalletType   string    `db:"wallet_type" json:"walletType"` // user, master
	IsPrimary    bool      `db:"is_primary" json:"isPrimary"`
	DerivationPath string  `db:"derivation_path" json:"derivationPath"`
	CreatedAt    time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt    time.Time `db:"updated_at" json:"updatedAt"`
}

// Balance represents token balance
type Balance struct {
	Address    string   `json:"address"`
	ChainID    uint64   `json:"chainId"`
	Native     *big.Int `json:"native"`
	NativeUSD  float64  `json:"nativeUsd"`
	Tokens     []TokenBalance `json:"tokens"`
	TotalUSD   float64  `json:"totalUsd"`
}

// TokenBalance represents a token balance
type TokenBalance struct {
	Address       string  `json:"address"`
	Symbol        string  `json:"symbol"`
	Name          string  `json:"name"`
	Decimals      uint8   `json:"decimals"`
	Balance       *big.Int `json:"balance"`
	BalanceFormatted string `json:"balanceFormatted"`
	PriceUSD      float64 `json:"priceUsd"`
	ValueUSD      float64 `json:"valueUsd"`
	LogoURL       string  `json:"logoUrl"`
}

// Transaction represents a blockchain transaction
type Transaction struct {
	ID          uint64    `db:"id" json:"id"`
	Hash        string    `db:"hash" json:"hash"`
	From        string    `db:"from_address" json:"from"`
	To          string    `db:"to_address" json:"to"`
	Value       string    `db:"value" json:"value"`
	GasPrice    string    `db:"gas_price" json:"gasPrice"`
	GasLimit    uint64    `db:"gas_limit" json:"gasLimit"`
	GasUsed     uint64    `db:"gas_used" json:"gasUsed"`
	Nonce       uint64    `db:"nonce" json:"nonce"`
	ChainID     uint64    `db:"chain_id" json:"chainId"`
	Token       string    `db:"token" json:"token"`
	Status      string    `db:"status" json:"status"` // pending, confirmed, failed
	Type        string    `db:"type" json:"type"` // send, receive, swap, contract
	BlockNumber uint64    `db:"block_number" json:"blockNumber"`
	Timestamp   time.Time `db:"timestamp" json:"timestamp"`
	CreatedAt   time.Time `db:"created_at" json:"createdAt"`
}

// WalletService handles wallet operations
type WalletService struct {
	db           *sqlx.DB
	redis        *redis.Client
	config       *Config
}

// NewWalletService creates a new wallet service
func NewWalletService(config *Config) *WalletService {
	return &WalletService{
		config: config,
	}
}

// CreateWallet creates a new wallet for a user
func (s *WalletService) CreateWallet(ctx context.Context, userID uint64, chainType ChainType) (*Wallet, error) {
	wallet := &Wallet{
		UserID:     userID,
		ChainType:  chainType,
		WalletType: "user",
		IsPrimary:  true,
	}

	// Generate address based on chain type
	address, err := s.generateAddress(chainType)
	if err != nil {
		return nil, fmt.Errorf("failed to generate address: %w", err)
	}
	wallet.Address = address

	return wallet, nil
}

// ImportFromMnemonic imports a wallet from mnemonic phrase
func (s *WalletService) ImportFromMnemonic(ctx context.Context, userID uint64, mnemonic string, password string, chainType ChainType) (*Wallet, error) {
	// Validate mnemonic
	mnemonicWords := strings.Fields(mnemonic)
	if len(mnemonicWords) != 12 && len(mnemonicWords) != 24 {
		return nil, fmt.Errorf("invalid mnemonic length: expected 12 or 24 words")
	}

	// Derive address from mnemonic
	address, derivationPath, err := s.deriveFromMnemonic(mnemonic, chainType)
	if err != nil {
		return nil, fmt.Errorf("failed to derive address: %w", err)
	}

	wallet := &Wallet{
		UserID:         userID,
		Address:        address,
		ChainType:      chainType,
		WalletType:     "user",
		DerivationPath: derivationPath,
		IsPrimary:      true,
	}

	return wallet, nil
}

// ImportFromPrivateKey imports a wallet from private key
func (s *WalletService) ImportFromPrivateKey(ctx context.Context, userID uint64, privateKeyHex string, chainType ChainType) (*Wallet, error) {
	// Remove 0x prefix if present
	privateKeyHex = strings.TrimPrefix(privateKeyHex, "0x")

	// Parse private key
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key format: %w", err)
	}

	if len(privateKeyBytes) != 32 {
		return nil, fmt.Errorf("invalid private key length: expected 32 bytes")
	}

	// Derive address from private key
	address, err := s.deriveAddressFromPrivateKey(privateKeyBytes, chainType)
	if err != nil {
		return nil, fmt.Errorf("failed to derive address: %w", err)
	}

	wallet := &Wallet{
		UserID:     userID,
		Address:    address,
		ChainType:  chainType,
		WalletType: "user",
		IsPrimary:  true,
	}

	return wallet, nil
}

// GetAddresses returns all addresses for a user
func (s *WalletService) GetAddresses(ctx context.Context, userID uint64) ([]Wallet, error) {
	// In production, query from database
	wallets := []Wallet{
		{
			ID:        1,
			UserID:    userID,
			Address:   "0x742d35Cc6634C0532925a3b844Bc9e7595f0eB1E",
			ChainType: ChainEthereum,
			ChainID:   1,
			IsPrimary: true,
		},
	}

	return wallets, nil
}

// GetBalance returns the balance for an address on a specific chain
func (s *WalletService) GetBalance(ctx context.Context, address string, chainID uint64) (*Balance, error) {
	balance := &Balance{
		Address: address,
		ChainID: chainID,
		Native:  big.NewInt(0),
		NativeUSD: 0,
		Tokens: []TokenBalance{},
		TotalUSD: 0,
	}

	// In production, query from blockchain RPC
	// For demo, return zero balance

	return balance, nil
}

// GetAllBalances returns all balances for a user across all chains
func (s *WalletService) GetAllBalances(ctx context.Context, userID uint64) ([]Balance, error) {
	// Get all user wallets
	wallets, err := s.GetAddresses(ctx, userID)
	if err != nil {
		return nil, err
	}

	balances := []Balance{}
	for _, wallet := range wallets {
		balance, err := s.GetBalance(ctx, wallet.Address, wallet.ChainID)
		if err != nil {
			continue
		}
		balances = append(balances, *balance)
	}

	return balances, nil
}

// BroadcastTransaction broadcasts a signed transaction to the network
func (s *WalletService) BroadcastTransaction(ctx context.Context, signedTx string) (string, error) {
	// Transaction broadcast is not implemented; a real tx hash can only be
	// obtained from the blockchain after broadcasting. Return empty to signal
	// "pending_broadcast" rather than fabricating a hash.
	return "", nil
}

// GetTransactionHistory returns transaction history for an address
func (s *WalletService) GetTransactionHistory(ctx context.Context, address string, chainID uint64, limit, offset int) ([]Transaction, error) {
	// In production, query from database or blockchain explorer
	return []Transaction{}, nil
}

// SignTransaction signs a transaction with the wallet's private key
func (s *WalletService) SignTransaction(ctx context.Context, txData string, chainID uint64) (string, error) {
	// In production, sign using the wallet's private key
	// For security, this should be done client-side or in a secure enclave
	return "", nil
}

// generateAddress generates an address based on chain type
func (s *WalletService) generateAddress(chainType ChainType) (string, error) {
	// Generate a random address for demo
	// In production, derive from mnemonic or generate properly
	return "0x" + strings.Repeat("0", 40), nil
}

// deriveFromMnemonic derives an address from mnemonic
func (s *WalletService) deriveFromMnemonic(mnemonic string, chainType ChainType) (string, string, error) {
	// In production, use proper BIP39/BIP44 derivation
	// Derivation path varies by chain:
	// Ethereum: m/44'/60'/0'/0/0
	// Bitcoin: m/44'/0'/0'/0/0
	// Solana: m/44'/501'/0'/0'
	derivationPath := "m/44'/60'/0'/0/0"
	
	// Generate address from mnemonic
	address := "0x742d35Cc6634C0532925a3b844Bc9e7595f0eB1E"
	
	return address, derivationPath, nil
}

// deriveAddressFromPrivateKey derives an address from private key
func (s *WalletService) deriveAddressFromPrivateKey(privateKey []byte, chainType ChainType) (string, error) {
	switch chainType {
	case ChainEthereum, ChainPolygon, ChainArbitrum, ChainOptimism, ChainBase, ChainBSC:
		// Derive Ethereum-style address
		publicKey, err := crypto.UnmarshalPubkey(crypto.FromECDSAPub(crypto.ToECDSAPub(privateKey)))
		if err != nil {
			return "", err
		}
		address := common.BytesToAddress(crypto.Keccak256(publicKey[1:])[12:])
		return address.Hex(), nil

	default:
		return "", fmt.Errorf("unsupported chain type: %s", chainType)
	}
}

// SupportedChains returns the list of supported chains
func (s *WalletService) SupportedChains() []ChainConfig {
	return []ChainConfig{
		{ID: 1, Name: "Ethereum", Symbol: "ETH", ChainType: ChainEthereum, ChainID: 1, Decimals: 18, IsActive: true},
		{ID: 2, Name: "Polygon", Symbol: "MATIC", ChainType: ChainPolygon, ChainID: 137, Decimals: 18, IsActive: true},
		{ID: 3, Name: "Arbitrum", Symbol: "ARB", ChainType: ChainArbitrum, ChainID: 42161, Decimals: 18, IsActive: true},
		{ID: 4, Name: "Optimism", Symbol: "OP", ChainType: ChainOptimism, ChainID: 10, Decimals: 18, IsActive: true},
		{ID: 5, Name: "Base", Symbol: "BASE", ChainType: ChainBase, ChainID: 8453, Decimals: 18, IsActive: true},
		{ID: 6, Name: "Avalanche", Symbol: "AVAX", ChainType: ChainAvalanche, ChainID: 43114, Decimals: 18, IsActive: true},
		{ID: 7, Name: "BNB Chain", Symbol: "BNB", ChainType: ChainBSC, ChainID: 56, Decimals: 18, IsActive: true},
		{ID: 8, Name: "Solana", Symbol: "SOL", ChainType: ChainSolana, ChainID: 101, Decimals: 9, IsActive: true},
		{ID: 9, Name: "Tron", Symbol: "TRX", ChainType: ChainTron, ChainID: 728126428, Decimals: 6, IsActive: true},
		{ID: 10, Name: "Bitcoin", Symbol: "BTC", ChainType: ChainBitcoin, ChainID: 0, Decimals: 8, IsActive: true},
		{ID: 11, Name: "Cosmos", Symbol: "ATOM", ChainType: ChainCosmos, ChainID: 118, Decimals: 6, IsActive: true},
		{ID: 12, Name: "Pi Network", Symbol: "PI", ChainType: ChainPi, ChainID: 314159, Decimals: 18, IsActive: true},
		{ID: 13, Name: "Toncoin", Symbol: "TON", ChainType: ChainTon, ChainID: -239, Decimals: 9, IsActive: true},
		{ID: 14, Name: "Aptos", Symbol: "APT", ChainType: ChainAptos, ChainID: 637, Decimals: 8, IsActive: true},
		{ID: 15, Name: "PulseChain", Symbol: "PLS", ChainType: ChainPulseChain, ChainID: 369, Decimals: 18, IsActive: true},
	}
}

// MarshalJSON custom JSON marshaling
func (c ChainType) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(c))
}

// UnmarshalJSON custom JSON unmarshaling
func (c *ChainType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*c = ChainType(s)
	return nil
}
