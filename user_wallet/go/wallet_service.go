/**
 * TigerWallet Complete Wallet Service
 * 
 * Comprehensive wallet service with user wallet, master wallet,
 * multi-chain support, and all wallet operations.
 * Built with Go for high-load distributed operations.
 */

package wallet

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// ============================================================================
// Types
// ============================================================================

// Wallet represents a crypto wallet
type Wallet struct {
	ID            string            `json:"id"`
	Address       string            `json:"address"`
	ChainType     string            `json:"chain_type"` // evm, solana, bitcoin, etc.
	ChainID       uint64            `json:"chain_id"`
	PublicKey    string            `json:"public_key"`
	Type         string            `json:"type"` // user, master, white_label
	WhiteLabelID string            `json:"white_label_id,omitempty"`
	Name         string            `json:"name"`
	CreatedAt    int64             `json:"created_at"`
	UpdatedAt    int64             `json:"updated_at"`
	IsImported   bool              `json:"is_imported"`
}

// WalletAccount represents a complete wallet account
type WalletAccount struct {
	Wallet          *Wallet              `json:"wallet"`
	Balances        map[string]Balance   `json:"balances"`
	Transactions    []Transaction        `json:"transactions"`
	Allowances     []TokenAllowance     `json:"allowances"`
	Nonce          uint64               `json:"nonce"`
	ChainState     map[string]interface{} `json:"chain_state"`
}

// Balance represents a token balance
type Balance struct {
	Available  string `json:"available"`
	Locked    string `json:"locked"`
	Staked    string `json:"staked"`
	Total     string `json:"total"`
	USDValue  float64 `json:"usd_value"`
}

// TokenAllowance represents an ERC20 allowance
type TokenAllowance struct {
	TokenAddress string `json:"token_address"`
	Spender     string `json:"spender"`
	Allowance   string `json:"allowance"`
}

// Transaction represents a wallet transaction
type Transaction struct {
	ID           string   `json:"id"`
	Hash         string   `json:"hash"`
	From         string   `json:"from"`
	To           string   `json:"to"`
	Value        string   `json:"value"`
	Token        string   `json:"token"`
	TokenAmount  string   `json:"token_amount,omitempty"`
	Fee          string   `json:"fee"`
	Status       string   `json:"status"` // pending, confirmed, failed
	BlockNumber  uint64   `json:"block_number"`
	Timestamp    int64    `json:"timestamp"`
	ChainID      uint64   `json:"chain_id"`
	Type         string   `json:"type"` // send, receive, swap, stake, unstake, approve, contract_call
	GasPrice     string   `json:"gas_price"`
	GasUsed      uint64   `json:"gas_used"`
	Nonce        uint64   `json:"nonce"`
	InputData   string   `json:"input_data,omitempty"`
}

// WalletConfig represents wallet configuration
type WalletConfig struct {
	WalletID         string    `json:"wallet_id"`
	AutoSignEnabled  bool      `json:"auto_sign_enabled"`
	AutoSignTimeout  int       `json:"auto_sign_timeout"` // seconds
	MaxTransactionFee float64  `json:"max_transaction_fee"`
	AllowedTokens   []string  `json:"allowed_tokens"`
	BlacklistedTokens []string `json:"blacklisted_tokens"`
	RequiredConfirmations map[uint64]int `json:"required_confirmations"`
}

// MasterWalletConfig represents master wallet configuration
type MasterWalletConfig struct {
	WalletID          string             `json:"wallet_id"`
	WithdrawalFee     float64            `json:"withdrawal_fee"`
	SwapFee           float64            `json:"swap_fee"`
	TransactionFee   float64            `json:"transaction_fee"`
	SupportedChains   []uint64           `json:"supported_chains"`
	SupportedTokens   []string           `json:"supported_tokens"`
	AutoApproveLimit float64            `json:"auto_approve_limit"`
	FeeCollectionAddress string          `json:"fee_collection_address"`
	RevenueAddress    string            `json:"revenue_address"`
}

// SwapQuote represents a swap quote
type SwapQuote struct {
	ID                string          `json:"id"`
	FromToken        string          `json:"from_token"`
	ToToken          string          `json:"to_token"`
	FromAmount       string          `json:"from_amount"`
	ToAmount         string          `json:"to_amount"`
	MinReceived      string          `json:"min_received"`
	PriceImpact      float64         `json:"price_impact"`
	Route            []SwapRoute      `json:"route"`
	Fee              string          `json:"fee"`
	GasEstimate      string          `json:"gas_estimate"`
	ExpirationTime   int64           `json:"expiration_time"`
}

// StakingPosition represents a staking position
type StakingPosition struct {
	ID            string  `json:"id"`
	Protocol      string  `json:"protocol"`
	Token         string  `json:"token"`
	Amount        string  `json:"amount"`
	RewardClaimed string  `json:"reward_claimed"`
	StartTime     int64   `json:"start_time"`
	EndTime       int64   `json:"end_time"`
	Status        string  `json:"status"` // active, unstaked, claimed
}

// NFT represents an NFT
type NFT struct {
	ID            string `json:"id"`
	TokenID       string `json:"token_id"`
	ContractAddress string `json:"contract_address"`
	Owner         string `json:"owner"`
	TokenURI      string `json:"token_uri"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	ImageURL      string `json:"image_url"`
	Attributes    map[string]string `json:"attributes"`
	ChainID       uint64 `json:"chain_id"`
}

// ============================================================================
// Wallet Service
// ============================================================================

// WalletService provides complete wallet functionality
type WalletService struct {
	mu              sync.RWMutex
	wallets         map[string]*Wallet
	accounts        map[string]*WalletAccount
	transactions    map[string]*Transaction
	quotes          map[string]*SwapQuote
	staking         map[string][]StakingPosition
	nfts            map[string][]NFT
	configs         map[string]*WalletConfig
	masterConfigs   map[string]*MasterWalletConfig
	userMasterLinks map[string]string // user_wallet_id -> master_wallet_id
	chainBalances   map[string]map[string]Balance // wallet_address -> token -> balance
}

// NewWalletService creates a new wallet service
func NewWalletService() *WalletService {
	return &WalletService{
		wallets:        make(map[string]*Wallet),
		accounts:       make(map[string]*WalletAccount),
		transactions:   make(map[string]*Transaction),
		quotes:         make(map[string]*SwapQuote),
		staking:         make(map[string][]StakingPosition),
		nfts:           make(map[string][]NFT),
		configs:        make(map[string]*WalletConfig),
		masterConfigs:   make(map[string]*MasterWalletConfig),
		userMasterLinks: make(map[string]string),
		chainBalances:  make(map[string]map[string]Balance),
	}
}

// ============================================================================
// Wallet Creation
// ============================================================================

// CreateWallet creates a new wallet
func (s *WalletService) CreateWallet(ctx context.Context, chainType string, chainID uint64, walletType string, whiteLabelID string) (*Wallet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	wallet := &Wallet{
		ID:            generateID(),
		Address:       generateAddress(chainType),
		ChainType:     chainType,
		ChainID:       chainID,
		Type:          walletType,
		WhiteLabelID:  whiteLabelID,
		Name:          fmt.Sprintf("Wallet %s", time.Now().Format("2006-01-02")),
		CreatedAt:     time.Now().UnixMilli(),
		UpdatedAt:     time.Now().UnixMilli(),
		IsImported:    false,
	}

	s.wallets[wallet.ID] = wallet

	// Initialize account
	s.accounts[wallet.ID] = &WalletAccount{
		Wallet:   wallet,
		Balances: make(map[string]Balance),
		Transactions: make([]Transaction, 0),
		Allowances: make([]TokenAllowance, 0),
		ChainState: make(map[string]interface{}),
	}

	// Initialize chain balances
	s.chainBalances[wallet.Address] = make(map[string]Balance)

	return wallet, nil
}

// ImportWallet imports a wallet with existing private key
func (s *WalletService) ImportWallet(ctx context.Context, privateKey, chainType string, chainID uint64, walletType string) (*Wallet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	address := deriveAddressFromPrivateKey(privateKey)

	// Check if wallet already exists
	for _, w := range s.wallets {
		if w.Address == address {
			return w, nil
		}
	}

	wallet := &Wallet{
		ID:        generateID(),
		Address:   address,
		ChainType: chainType,
		ChainID:   chainID,
		Type:      walletType,
		CreatedAt: time.Now().UnixMilli(),
		UpdatedAt: time.Now().UnixMilli(),
		IsImported: true,
	}

	s.wallets[wallet.ID] = wallet
	s.accounts[wallet.ID] = &WalletAccount{
		Wallet:   wallet,
		Balances: make(map[string]Balance),
	}

	s.chainBalances[wallet.Address] = make(map[string]Balance)

	return wallet, nil
}

// GetWallet retrieves a wallet
func (s *WalletService) GetWallet(ctx context.Context, walletID string) (*Wallet, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	wallet, exists := s.wallets[walletID]
	if !exists {
		return nil, fmt.Errorf("wallet not found")
	}
	return wallet, nil
}

// GetWalletByAddress retrieves a wallet by address
func (s *WalletService) GetWalletByAddress(ctx context.Context, address string) (*Wallet, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, wallet := range s.wallets {
		if wallet.Address == address {
			return wallet, nil
		}
	}
	return nil, fmt.Errorf("wallet not found")
}

// ListWallets lists all wallets
func (s *WalletService) ListWallets(ctx context.Context, walletType, whiteLabelID string) ([]*Wallet, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Wallet, 0)
	for _, wallet := range s.wallets {
		if walletType != "" && wallet.Type != walletType {
			continue
		}
		if whiteLabelID != "" && wallet.WhiteLabelID != whiteLabelID {
			continue
		}
		result = append(result, wallet)
	}
	return result, nil
}

// GetAccount retrieves wallet account with balances
func (s *WalletService) GetAccount(ctx context.Context, walletID string) (*WalletAccount, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	account, exists := s.accounts[walletID]
	if !exists {
		return nil, fmt.Errorf("account not found")
	}
	return account, nil
}

// ============================================================================
// Balance Management
// ============================================================================

// GetBalance retrieves balance for a wallet
func (s *WalletService) GetBalance(ctx context.Context, address, token string) (*Balance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if balances, ok := s.chainBalances[address]; ok {
		if balance, exists := balances[token]; exists {
			return &balance, nil
		}
	}
	return &Balance{}, nil
}

// UpdateBalance updates balance for a wallet
func (s *WalletService) UpdateBalance(ctx context.Context, address, token string, amount float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.chainBalances[address]; !ok {
		s.chainBalances[address] = make(map[string]Balance)
	}

	balance := s.chainBalances[address][token]
	balance.Available = fmt.Sprintf("%f", amount)
	balance.Total = fmt.Sprintf("%f", amount)

	s.chainBalances[address][token] = balance

	return nil
}

// GetAllBalances retrieves all balances for a wallet
func (s *WalletService) GetAllBalances(ctx context.Context, address string) (map[string]Balance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if balances, ok := s.chainBalances[address]; ok {
		return balances, nil
	}
	return make(map[string]Balance), nil
}

// ============================================================================
// Transaction Functions
// ============================================================================

// SendTransaction sends a transaction
func (s *WalletService) SendTransaction(ctx context.Context, walletID, to, value, token, data string, gasPrice uint64) (*Transaction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	wallet, exists := s.wallets[walletID]
	if !exists {
		return nil, fmt.Errorf("wallet not found")
	}

	tx := &Transaction{
		ID:           generateTxID(),
		From:         wallet.Address,
		To:           to,
		Value:        value,
		Token:        token,
		Fee:          "0",
		Status:       "pending",
		Timestamp:    time.Now().UnixMilli(),
		ChainID:      wallet.ChainID,
		GasPrice:     fmt.Sprintf("%d", gasPrice),
		InputData:    data,
	}

	s.transactions[tx.ID] = tx
	s.transactions[tx.Hash] = tx

	return tx, nil
}

// SignAndBroadcast signs and broadcasts a transaction
func (s *WalletService) SignAndBroadcast(ctx context.Context, txID, privateKey string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, exists := s.transactions[txID]
	if !exists {
		return "", fmt.Errorf("transaction not found")
	}

	// In production, this would actually sign and broadcast
	// Simplified: just update status
	tx.Hash = generateTxHash()
	tx.Status = "confirmed"
	tx.BlockNumber = 1

	return tx.Hash, nil
}

// GetTransaction retrieves a transaction
func (s *WalletService) GetTransaction(ctx context.Context, txHash string) (*Transaction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tx, exists := s.transactions[txHash]
	if !exists {
		return nil, fmt.Errorf("transaction not found")
	}
	return tx, nil
}

// GetTransactions retrieves transactions for a wallet
func (s *WalletService) GetTransactions(ctx context.Context, address string, limit, offset int) ([]*Transaction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Transaction, 0)
	skipped := 0
	count := 0

	for _, tx := range s.transactions {
		if tx.From == address || tx.To == address {
			if skipped < offset {
				skipped++
				continue
			}
			result = append(result, tx)
			count++
			if limit > 0 && count >= limit {
				break
			}
		}
	}

	return result, nil
}

// ============================================================================
// Swap Functions
// ============================================================================

// GetSwapQuote gets a swap quote
func (s *WalletService) GetSwapQuote(ctx context.Context, fromToken, toToken, amount string) (*SwapQuote, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// In production, this would query DEX aggregators
	// Simplified: generate mock quote
	quote := &SwapQuote{
		ID:              generateID(),
		FromToken:       fromToken,
		ToToken:         toToken,
		FromAmount:      amount,
		ToAmount:        amount, // Simplified: 1:1
		MinReceived:      fmt.Sprintf("%f", parseFloat(amount)*0.99),
		PriceImpact:     0.1,
		Route:           []SwapRoute{},
		Fee:             fmt.Sprintf("%f", parseFloat(amount)*0.003),
		GasEstimate:     "21000",
		ExpirationTime:  time.Now().Add(30*time.Second).UnixMilli(),
	}

	s.quotes[quote.ID] = quote

	return quote, nil
}

// ExecuteSwap executes a swap
func (s *WalletService) ExecuteSwap(ctx context.Context, walletID, quoteID string) (*Transaction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	quote, exists := s.quotes[quoteID]
	if !exists {
		return nil, fmt.Errorf("quote not found")
	}

	wallet, exists := s.wallets[walletID]
	if !exists {
		return nil, fmt.Errorf("wallet not found")
	}

	tx := &Transaction{
		ID:        generateTxID(),
		From:      wallet.Address,
		To:        "swap_router",
		Value:     quote.FromAmount,
		Token:     quote.FromToken,
		Fee:       quote.Fee,
		Status:    "pending",
		Type:      "swap",
		Timestamp: time.Now().UnixMilli(),
		ChainID:   wallet.ChainID,
	}

	s.transactions[tx.ID] = tx

	return tx, nil
}

// ============================================================================
// Staking Functions
// ============================================================================

// Stake stakes tokens
func (s *WalletService) Stake(ctx context.Context, walletID, token, amount, protocol string) (*StakingPosition, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	position := StakingPosition{
		ID:            generateID(),
		Protocol:      protocol,
		Token:         token,
		Amount:        amount,
		RewardClaimed: "0",
		StartTime:     time.Now().UnixMilli(),
		Status:        "active",
	}

	s.staking[walletID] = append(s.staking[walletID], position)

	return &position, nil
}

// Unstake unstakes tokens
func (s *WalletService) Unstake(ctx context.Context, walletID, positionID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	positions := s.staking[walletID]
	for i := range positions {
		if positions[i].ID == positionID {
			positions[i].Status = "unstaked"
			return positions[i].Amount, nil
		}
	}
	return "", fmt.Errorf("position not found")
}

// GetStakingPositions retrieves staking positions
func (s *WalletService) GetStakingPositions(ctx context.Context, walletID string) ([]StakingPosition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.staking[walletID], nil
}

// ============================================================================
// NFT Functions
// ============================================================================

// GetNFTs retrieves NFTs for a wallet
func (s *WalletService) GetNFTs(ctx context.Context, address string) ([]NFT, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.nfts[address], nil
}

// TransferNFT transfers an NFT
func (s *WalletService) TransferNFT(ctx context.Context, from, to, contractAddress, tokenID string) (*Transaction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx := &Transaction{
		ID:        generateTxID(),
		From:      from,
		To:         to,
		Token:     contractAddress,
		TokenAmount: tokenID,
		Status:    "pending",
		Type:      "nft_transfer",
		Timestamp: time.Now().UnixMilli(),
	}

	s.transactions[tx.ID] = tx

	return tx, nil
}

// ============================================================================
// Master Wallet Functions
// ============================================================================

// LinkToMasterWallet links a user wallet to a master wallet
func (s *WalletService) LinkToMasterWallet(ctx context.Context, userWalletID, masterWalletID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.wallets[userWalletID]; !exists {
		return fmt.Errorf("user wallet not found")
	}
	if _, exists := s.wallets[masterWalletID]; !exists {
		return fmt.Errorf("master wallet not found")
	}

	s.userMasterLinks[userWalletID] = masterWalletID

	return nil
}

// GetMasterWallet gets the master wallet for a user wallet
func (s *WalletService) GetMasterWallet(ctx context.Context, userWalletID string) (*Wallet, error) {
	s.mu.RLock()
	masterID := s.userMasterLinks[userWalletID]
	s.mu.RUnlock()

	if masterID == "" {
		return nil, fmt.Errorf("no master wallet linked")
	}

	return s.GetWallet(ctx, masterID)
}

// ConfigureMasterWallet configures master wallet
func (s *WalletService) ConfigureMasterWallet(ctx context.Context, config *MasterWalletConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.masterConfigs[config.WalletID] = config

	return nil
}

// GetMasterWalletConfig gets master wallet configuration
func (s *WalletService) GetMasterWalletConfig(ctx context.Context, walletID string) (*MasterWalletConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	config, exists := s.masterConfigs[walletID]
	if !exists {
		return nil, fmt.Errorf("config not found")
	}

	return config, nil
}

// SetFee sets transaction fee for master wallet
func (s *WalletService) SetFee(ctx context.Context, walletID, feeType string, fee float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	config, exists := s.masterConfigs[walletID]
	if !exists {
		return fmt.Errorf("config not found")
	}

	switch feeType {
	case "withdrawal":
		config.WithdrawalFee = fee
	case "swap":
		config.SwapFee = fee
	case "transaction":
		config.TransactionFee = fee
	default:
		return fmt.Errorf("unknown fee type")
	}

	return nil
}

// ============================================================================
// Wallet Configuration
// ============================================================================

// ConfigureWallet configures a wallet
func (s *WalletService) ConfigureWallet(ctx context.Context, config *WalletConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.configs[config.WalletID] = config

	return nil
}

// GetWalletConfig gets wallet configuration
func (s *WalletService) GetWalletConfig(ctx context.Context, walletID string) (*WalletConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	config, exists := s.configs[walletID]
	if !exists {
		return nil, fmt.Errorf("config not found")
	}

	return config, nil
}

// ============================================================================
// Helper Functions
// ============================================================================

func generateID() string {
	return fmt.Sprintf("id_%d_%s", time.Now().UnixNano(), randomString(12))
}

func generateTxID() string {
	return fmt.Sprintf("tx_%d_%s", time.Now().UnixNano(), randomString(12))
}

func generateTxHash() string {
	data := fmt.Sprintf("txhash_%d_%s", time.Now().UnixNano(), randomString(16))
	hash := sha256.Sum256([]byte(data))
	return "0x" + hex.EncodeToString(hash[:])
}

func generateAddress(chainType string) string {
	data := fmt.Sprintf("addr_%s_%d_%s", chainType, time.Now().UnixNano(), randomString(16))
	hash := sha256.Sum256([]byte(data))
	return "0x" + hex.EncodeToString(hash[:])[0:40]
}

func deriveAddressFromPrivateKey(privateKey string) string {
	// Simplified: in production use proper key derivation
	hash := sha256.Sum256([]byte(privateKey))
	return "0x" + hex.EncodeToString(hash[:])[0:40]
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		time.Sleep(time.Nanosecond)
	}
	return string(result)
}

func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

// Serialize/Deserialize
func (w *Wallet) Serialize() (string, error) {
	data, err := json.Marshal(w)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

func DeserializeWallet(data string) (*Wallet, error) {
	decoded, err := hex.DecodeString(data)
	if err != nil {
		return nil, err
	}
	var wallet Wallet
	if err := json.Unmarshal(decoded, &wallet); err != nil {
		return nil, err
	}
	return &wallet, nil
}
