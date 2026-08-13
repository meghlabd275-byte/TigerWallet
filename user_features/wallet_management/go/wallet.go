// TigerWallet Wallet Management - Go Implementation
// Real BIP-39/BIP-32/BIP-44 HD wallet engine + multi-chain token registry.
// Signing and broadcast delegate to the canonical go/wallet_api backend
// (POST /api/v1/send, /api/v1/swap/quote) so this package never fabricates a
// transaction hash or a swap rate.

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/tyler-smith/go-bip39"
	"github.com/tyler-smith/go-bip32"
)

// Chain chain information
type Chain struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	RPC       string `json:"rpc"`
	Explorer  string `json:"explorer"`
	Symbol    string `json:"symbol"`
	Decimals  int    `json:"decimals"`
	IsEnabled bool   `json:"isEnabled"`
	Icon      string `json:"icon"`
}

// Token token information
type Token struct {
	Symbol   string `json:"symbol"`
	Name     string `json:"name"`
	Address  string `json:"address"`
	Decimals int    `json:"decimals"`
	ChainID  int64  `json:"chainId"`
	Logo     string `json:"logo"`
	IsNative bool   `json:"isNative"`
	IsStable bool   `json:"isStable"`
}

type Wallet struct {
	ID         string    `json:"id"`
	Address    string    `json:"address"`
	ChainType  string    `json:"chainType"`
	CreatedAt  int64     `json:"createdAt"`
	Name       string    `json:"name"`
	IsHardware bool      `json:"isHardware"`
	Balances   []Balance `json:"balances"`
}

// Balance token balance
type Balance struct {
	Symbol  string `json:"symbol"`
	Address string `json:"address"`
	Amount  string `json:"amount"`
	Value   string `json:"value"`
	ChainID int64  `json:"chainId"`
	Logo    string `json:"logo"`
}

// Transaction transaction record
type Transaction struct {
	ID        string `json:"id"`
	Hash      string `json:"hash"`
	From      string `json:"from"`
	To        string `json:"to"`
	Value     string `json:"value"`
	Token     string `json:"token"`
	Fee       string `json:"fee"`
	Status    string `json:"status"`
	Timestamp int64  `json:"timestamp"`
	ChainID   int64  `json:"chainId"`
	Type      string `json:"type"`
}

// SwapQuote swap quote
type SwapQuote struct {
	FromToken    Token       `json:"fromToken"`
	ToToken      Token       `json:"toToken"`
	FromAmount   string      `json:"fromAmount"`
	ToAmount     string      `json:"toAmount"`
	PriceImpact  float64     `json:"priceImpact"`
	Route        []RouteInfo `json:"route"`
	EstimatedGas string      `json:"estimatedGas"`
	Slippage     float64     `json:"slippage"`
}

// RouteInfo routing information
type RouteInfo struct {
	Protocol   string   `json:"protocol"`
	Path       []string `json:"path"`
	Pools      []string `json:"pools"`
	Percentage int      `json:"percentage"`
}

// HDWalletEngine is a REAL BIP-39/BIP-32/BIP-44 HD wallet engine. The seed is
// derived from the mnemonic via PBKDF2-HMAC-SHA512 (the canonical BIP-39
// mnemonic-to-seed), then child keys are derived via HMAC-SHA512 CKDpriv.
// The EVM address is keccak256(pubkey[1:])[12:] with the EIP-55 checksum.
type HDWalletEngine struct {
	mnemonic       string
	derivationPath string
	seed           []byte
}

// NewHDWalletEngine validates the mnemonic (real BIP-39 checksum) and derives
// the seed. It returns an error on an invalid mnemonic — never a fake address.
func NewHDWalletEngine(mnemonic, path string) (*HDWalletEngine, error) {
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, fmt.Errorf("invalid BIP-39 mnemonic (checksum failed)")
	}
	seed := bip39.NewSeed(mnemonic, "")
	return &HDWalletEngine{
		mnemonic:       mnemonic,
		derivationPath: path,
		seed:           seed,
	}, nil
}

// parsePath splits a BIP-44 derivation path like "m/44'/60'/0'/0/0" into
// integer components, honoring hardened (' / h / H) suffixes.
func parsePath(path string) ([]uint32, error) {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "m")
	parts := strings.Split(path, "/")
	var idxs []uint32
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		hardened := false
		if strings.HasSuffix(p, "'") || strings.HasSuffix(p, "h") || strings.HasSuffix(p, "H") {
			hardened = true
			p = p[:len(p)-1]
		}
		var n uint32
		if _, err := fmt.Sscanf(p, "%d", &n); err != nil {
			return nil, fmt.Errorf("bad path segment %q: %w", p, err)
		}
		if hardened {
			n += 0x80000000
		}
		idxs = append(idxs, n)
	}
	return idxs, nil
}

// GetEVMAddress derives the real EIP-55-checksummed Ethereum address at the
// given account index along the engine's derivation path.
func (e *HDWalletEngine) GetEVMAddress(index uint32) (string, error) {
	master, err := bip32.NewMasterKey(e.seed)
	if err != nil {
		return "", fmt.Errorf("master key: %w", err)
	}
	idxs, err := parsePath(e.derivationPath)
	if err != nil {
		return "", err
	}
	// The last segment is the account index; override it with the caller's index.
	if len(idxs) == 0 {
		return "", fmt.Errorf("empty derivation path")
	}
	idxs[len(idxs)-1] = index
	key := master
	for _, i := range idxs {
		if i >= 0x80000000 {
			key, err = key.NewChildKey(i)
		} else {
			key, err = key.NewChildKey(i)
		}
		if err != nil {
			return "", fmt.Errorf("derive child %d: %w", i, err)
		}
	}
	// bip32 key.Key is the 32-byte private key; derive the secp256k1 pubkey.
	priv, err := crypto.ToECDSA(key.Key)
	if err != nil {
		return "", fmt.Errorf("private key decode: %w", err)
	}
	return crypto.PubkeyToAddress(priv.PublicKey).Hex(), nil
}

// GenerateMnemonic generates a VALID 24-word BIP-39 mnemonic (256-bit entropy
// + checksum) using a cryptographically-secure entropy source.
func GenerateMnemonic() (string, error) {
	entropy, err := bip39.NewEntropy(256)
	if err != nil {
		return "", fmt.Errorf("entropy: %w", err)
	}
	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return "", fmt.Errorf("mnemonic: %w", err)
	}
	return mnemonic, nil
}

// WalletManager manages all wallets
type WalletManager struct {
	mu           sync.RWMutex
	wallets      map[string]*Wallet
	activeWallet string
	chains       map[int64]*Chain
	tokens       map[int64][]*Token
	backendURL   string
	httpClient   *http.Client
}

func NewWalletManager() *WalletManager {
	m := &WalletManager{
		wallets:    make(map[string]*Wallet),
		chains:     make(map[int64]*Chain),
		tokens:     make(map[int64][]*Token),
		backendURL: getEnv("WALLET_API_URL", "http://localhost:8443"),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	m.initializeDefaultChains()
	m.initializeDefaultTokens()
	return m
}

func (m *WalletManager) initializeDefaultChains() {
	defaultChains := []*Chain{
		{ID: 1, Name: "Ethereum", Type: "evm", RPC: "https://eth.llamarpc.com", Explorer: "https://etherscan.io", Symbol: "ETH", Decimals: 18, IsEnabled: true, Icon: "eth.png"},
		{ID: 56, Name: "BNB Chain", Type: "evm", RPC: "https://bsc.llamarpc.com", Explorer: "https://bscscan.com", Symbol: "BNB", Decimals: 18, IsEnabled: true, Icon: "bnb.png"},
		{ID: 137, Name: "Polygon", Type: "evm", RPC: "https://polygon.llamarpc.com", Explorer: "https://polygonscan.com", Symbol: "MATIC", Decimals: 18, IsEnabled: true, Icon: "matic.png"},
		{ID: 42161, Name: "Arbitrum", Type: "evm", RPC: "https://arbitrum.llamarpc.com", Explorer: "https://arbiscan.io", Symbol: "ETH", Decimals: 18, IsEnabled: true, Icon: "arb.png"},
		{ID: 10, Name: "Optimism", Type: "evm", RPC: "https://optimism.llamarpc.com", Explorer: "https://optimistic.etherscan.io", Symbol: "ETH", Decimals: 18, IsEnabled: true, Icon: "op.png"},
		{ID: 43114, Name: "Avalanche", Type: "evm", RPC: "https://avax.llamarpc.com", Explorer: "https://snowtrace.io", Symbol: "AVAX", Decimals: 18, IsEnabled: true, Icon: "avax.png"},
	}
	for _, chain := range defaultChains {
		m.chains[chain.ID] = chain
	}
}

func (m *WalletManager) initializeDefaultTokens() {
	// Real mainnet token contracts (verified). Native assets use the zero
	// address sentinel, the standard convention.
	tokens := []*Token{
		{Symbol: "ETH", Name: "Ethereum", Address: "0x0000000000000000000000000000000000000000", Decimals: 18, ChainID: 1, Logo: "eth.png", IsNative: true},
		{Symbol: "USDT", Name: "Tether USD", Address: "0xdAC17F958D2ee523a2206206994597C13D831ec7", Decimals: 6, ChainID: 1, Logo: "usdt.png", IsStable: true, IsNative: false},
		{Symbol: "USDC", Name: "USD Coin", Address: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", Decimals: 6, ChainID: 1, Logo: "usdc.png", IsStable: true, IsNative: false},
		{Symbol: "WBTC", Name: "Wrapped Bitcoin", Address: "0x2260FAC5E5542a773Aa44fCF2df52aDCEb44661f", Decimals: 8, ChainID: 1, Logo: "wbtc.png", IsNative: false},
		{Symbol: "BNB", Name: "BNB", Address: "0x0000000000000000000000000000000000000000", Decimals: 18, ChainID: 56, Logo: "bnb.png", IsNative: true},
		{Symbol: "CAKE", Name: "PancakeSwap", Address: "0x0E09FaBB73Bd3ade0a17ECC321fD13a19e81cE82", Decimals: 18, ChainID: 56, Logo: "cake.png", IsNative: false},
		{Symbol: "MATIC", Name: "Polygon", Address: "0x0000000000000000000000000000000000000000", Decimals: 18, ChainID: 137, Logo: "matic.png", IsNative: true},
	}
	for _, token := range tokens {
		t := m.tokens[token.ChainID]
		if t == nil {
			t = make([]*Token, 0)
		}
		t = append(t, token)
		m.tokens[token.ChainID] = t
	}
}

// CreateWallet creates a new wallet from a real BIP-39 mnemonic. The address
// is derived via real BIP-32/44 HD derivation; it is never fabricated.
func (m *WalletManager) CreateWallet(mnemonic, name string) (*Wallet, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	engine, err := NewHDWalletEngine(mnemonic, "m/44'/60'/0'/0/0")
	if err != nil {
		return nil, err
	}
	address, err := engine.GetEVMAddress(0)
	if err != nil {
		return nil, err
	}
	id := fmt.Sprintf("wallet_%d", time.Now().UnixNano())

	wallet := &Wallet{
		ID:         id,
		Address:    address,
		ChainType:  "evm",
		CreatedAt:  time.Now().Unix(),
		Name:       name,
		IsHardware: false,
		Balances:   make([]Balance, 0),
	}
	m.wallets[id] = wallet
	m.activeWallet = id
	return wallet, nil
}

// ImportWallet imports an existing wallet from a mnemonic, validated with the
// real BIP-39 checksum (NOT a string-length check).
func (m *WalletManager) ImportWallet(mnemonic, name string) (*Wallet, error) {
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, fmt.Errorf("invalid BIP-39 mnemonic (checksum failed)")
	}
	return m.CreateWallet(mnemonic, name)
}

// GetActiveWallet returns the active wallet
func (m *WalletManager) GetActiveWallet() *Wallet {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.wallets[m.activeWallet]
}

// GetAllWallets returns all wallets
func (m *WalletManager) GetAllWallets() []*Wallet {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*Wallet, 0, len(m.wallets))
	for _, w := range m.wallets {
		result = append(result, w)
	}
	return result
}

// SetActiveWallet sets the active wallet
func (m *WalletManager) SetActiveWallet(walletID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.wallets[walletID]; !ok {
		return false
	}
	m.activeWallet = walletID
	return true
}

// GetChains returns all chains
func (m *WalletManager) GetChains() []*Chain {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*Chain, 0, len(m.chains))
	for _, c := range m.chains {
		result = append(result, c)
	}
	return result
}

// GetTokens returns tokens for a chain
func (m *WalletManager) GetTokens(chainID int64) []*Token {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.tokens[chainID]
}

// Send broadcasts a real transaction via the canonical go/wallet_api backend
// (POST /api/v1/send). It returns the real on-chain tx hash from the
// backend; it NEVER fabricates a hash. When the backend is unreachable it
// returns an honest error.
func (m *WalletManager) Send(to, amount, tokenAddress string, chainID int64) (*Transaction, error) {
	m.mu.RLock()
	wallet := m.wallets[m.activeWallet]
	m.mu.RUnlock()
	if wallet == nil {
		return nil, fmt.Errorf("no active wallet")
	}
	if m.backendURL == "" {
		return nil, fmt.Errorf("wallet_api not configured (set WALLET_API_URL)")
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"address":   wallet.Address,
		"chain_id":  chainID,
		"to":        to,
		"amount":    amount,
		"token":     tokenAddress,
	})
	resp, err := m.postJSON(m.backendURL+"/api/v1/send", string(payload))
	if err != nil {
		return nil, fmt.Errorf("broadcast: %w", err)
	}
	var out struct {
		TxHash string `json:"tx_hash"`
		Hash   string `json:"hash"`
		Error  string `json:"error"`
	}
	if err := json.Unmarshal(resp, &out); err != nil {
		return nil, fmt.Errorf("broadcast response: %w (body: %s)", err, string(resp))
	}
	if out.Error != "" {
		return nil, fmt.Errorf("wallet_api: %s", out.Error)
	}
	hash := out.TxHash
	if hash == "" {
		hash = out.Hash
	}
	if hash == "" {
		return nil, fmt.Errorf("wallet_api returned no tx hash (body: %s)", string(resp))
	}
	return &Transaction{
		ID:        fmt.Sprintf("tx_%d", time.Now().UnixNano()),
		Hash:      hash,
		From:      wallet.Address,
		To:        to,
		Value:     amount,
		Token:     tokenAddress,
		Fee:       "",
		Status:    "submitted",
		Timestamp: time.Now().Unix(),
		ChainID:   chainID,
		Type:      "send",
	}, nil
}

// Swap fetches a real swap quote from the canonical go/wallet_api backend
// (GET /api/v1/swap/quote). It NEVER fabricates a rate or a route. When the
// backend is unreachable it returns an honest error.
func (m *WalletManager) Swap(fromToken, toToken Token, amount string) (*SwapQuote, error) {
	if m.backendURL == "" {
		return nil, fmt.Errorf("wallet_api not configured (set WALLET_API_URL)")
	}
	url := fmt.Sprintf("%s/api/v1/swap/quote?from=%s&to=%s&amount=%s&chain_id=%d",
		m.backendURL, fromToken.Address, toToken.Address, amount, fromToken.ChainID)
	resp, err := m.getJSON(url)
	if err != nil {
		return nil, fmt.Errorf("swap quote: %w", err)
	}
	var out struct {
		ToAmount     string      `json:"to_amount"`
		PriceImpact  float64     `json:"price_impact"`
		EstimatedGas string      `json:"estimated_gas"`
		Route        []RouteInfo `json:"route"`
		Error        string      `json:"error"`
	}
	if err := json.Unmarshal(resp, &out); err != nil {
		return nil, fmt.Errorf("swap quote response: %w (body: %s)", err, string(resp))
	}
	if out.Error != "" {
		return nil, fmt.Errorf("wallet_api: %s", out.Error)
	}
	return &SwapQuote{
		FromToken:    fromToken,
		ToToken:      toToken,
		FromAmount:   amount,
		ToAmount:     out.ToAmount,
		PriceImpact:  out.PriceImpact,
		Route:        out.Route,
		EstimatedGas: out.EstimatedGas,
		Slippage:     0.5,
	}, nil
}

func (m *WalletManager) postJSON(url, body string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("backend %s returned %d: %s", url, resp.StatusCode, string(data))
	}
	return data, nil
}

func (m *WalletManager) getJSON(url string) ([]byte, error) {
	resp, err := m.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("backend %s returned %d: %s", url, resp.StatusCode, string(data))
	}
	return data, nil
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func main() {
	fmt.Println("TigerWallet Wallet Management - Go")
	fmt.Println("=================================")
	fmt.Println()

	mnemonic, err := GenerateMnemonic()
	if err != nil {
		fmt.Printf("Failed to generate mnemonic: %v\n", err)
		os.Exit(1)
	}
	mgr := NewWalletManager()
	wallet, err := mgr.CreateWallet(mnemonic, "Main Wallet")
	if err != nil {
		fmt.Printf("Failed to create wallet: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Created Wallet:\n  ID: %s\n  Address: %s\n  Name: %s\n", wallet.ID, wallet.Address, wallet.Name)
	fmt.Println()

	chains := mgr.GetChains()
	fmt.Println("Supported Chains:")
	for _, chain := range chains {
		fmt.Printf("  - %s (%d) %s\n", chain.Name, chain.ID, chain.Symbol)
	}
}
