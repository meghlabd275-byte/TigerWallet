package sdk

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ============================================================================
// TigerWallet Developer SDK
// ============================================================================

// Client provides TigerWallet SDK functionality
type Client struct {
	mu          sync.RWMutex
	config      *Config
	httpClient *http.Client
	auth      *Auth
	wallets   map[string]*WalletClient
}

// Config for SDK client
type Config struct {
	APIKey        string
	APISecret     string
	BaseURL      string
	Timeout      time.Duration
	RetryCount   int
	RetryDelay   time.Duration
}

// NewClient creates new SDK client
func NewClient(config *Config) *Client {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	
	httpClient := &http.Client{
		Timeout: config.Timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
		},
	}
	
	return &Client{
		config:     config,
		httpClient: httpClient,
		auth:      NewAuth(config.APIKey, config.APISecret),
		wallets:   make(map[string]*WalletClient),
	}
}

// ============================================================================
// Authentication
// ============================================================================

// Auth handles authentication
type Auth struct {
	APIKey    string
	APISecret string
	token    string
	expires  time.Time
	mu       sync.RWMutex
}

func NewAuth(apiKey, apiSecret string) *Auth {
	return &Auth{
		APIKey:    apiKey,
		APISecret: apiSecret,
	}
}

func (a *Auth) Token() (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	if a.token != "" && time.Now().Before(a.expires) {
		return a.token, nil
	}
	
	// In production, would call auth API
	a.token = fmt.Sprintf("token_%d", time.Now().UnixNano())
	a.expires = time.Now().Add(1 * time.Hour)
	
	return a.token, nil
}

// ============================================================================
// Wallet Client
// ============================================================================

// WalletClient provides wallet functionality
type WalletClient struct {
	mu        sync.RWMutex
	client    *Client
	address  string
	chainID  uint64
	balances map[string]*Balance
}

// Balance represents token balance
type Balance struct {
	Symbol    string
	Amount   *big.Int
	Decimals uint8
	ValueUSD *big.Rat
}

// NewWallet creates wallet client
func (c *Client) NewWallet(address string, chainID uint64) *WalletClient {
	return &WalletClient{
		client:   c,
		address:  address,
		chainID:  chainID,
		balances: make(map[string]*Balance),
	}
}

// GetAddress returns wallet address
func (w *WalletClient) GetAddress() string {
	return w.address
}

// GetBalance returns token balance
func (w *WalletClient) GetBalance(ctx context.Context, token string) (*Balance, error) {
	result, err := w.client.Request(ctx, "getBalance", map[string]interface{}{
		"address": w.address,
		"token":   token,
		"chainId": w.chainID,
	})
	
	if err != nil {
		return nil, err
	}
	
	var balance Balance
	if data, ok := result.(map[string]interface{}); ok {
		if amount, ok := data["amount"].(string); ok {
			balance.Amount = new(big.Int)
			balance.Amount.SetString(amount, 10)
		}
	}
	
	w.mu.Lock()
	w.balances[token] = &balance
	w.mu.Unlock()
	
	return &balance, nil
}

// Transfer sends tokens
func (w *WalletClient) Transfer(ctx context.Context, to string, amount *big.Int, token string) (string, error) {
	result, err := w.client.Request(ctx, "transfer", map[string]interface{}{
		"from":   w.address,
		"to":     to,
		"amount": amount.String(),
		"token":  token,
		"chainId": w.chainID,
	})
	
	if err != nil {
		return "", err
	}
	
	if txHash, ok := result.(string); ok {
		return txHash, nil
	}
	
	return "", fmt.Errorf("transfer failed")
}

// Swap swaps tokens
func (w *WalletClient) Swap(ctx context.Context, fromToken, toToken string, amount *big.Int) (string, error) {
	result, err := w.client.Request(ctx, "swap", map[string]interface{}{
		"address":   w.address,
		"fromToken": fromToken,
		"toToken":  toToken,
		"amount":   amount.String(),
		"chainId":  w.chainID,
	})
	
	if err != nil {
		return "", err
	}
	
	if txHash, ok := result.(string); ok {
		return txHash, nil
	}
	
	return "", fmt.Errorf("swap failed")
}

// ============================================================================
// API Methods
// ============================================================================

// Request makes API request
func (c *Client) Request(ctx context.Context, method string, params map[string]interface{}) (interface{}, error) {
	// Get auth token
	token, err := c.auth.Token()
	if err != nil {
		return nil, err
	}
	
	// Build request
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params": params,
		"id":     time.Now().UnixNano(),
	}
	
	reqBodyJSON, _ := json.Marshal(reqBody)
	
	// Make request
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", c.config.BaseURL+"/v1/rpc", nil)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	httpReq.Body = nil // Would set body
	
	// Simplified response
	return map[string]interface{}{
		"result": "0x0",
	}, nil
}

// ============================================================================
// Pre-built SDK Functions
// ============================================================================

// GetPortfolio gets portfolio
func (c *Client) GetPortfolio(ctx context.Context, address string) (*Portfolio, error) {
	result, err := c.Request(ctx, "getPortfolio", map[string]interface{}{
		"address": address,
	})
	
	if err != nil {
		return nil, err
	}
	
	var portfolio Portfolio
	if data, ok := result.(map[string]interface{}); ok {
		if value, ok := data["totalValueUSD"].(string); ok {
			portfolio.TotalValueUSD = value
		}
	}
	
	return &portfolio, nil
}

// GetTransactions gets transactions
func (c *Client) GetTransactions(ctx context.Context, address string, limit int) ([]Transaction, error) {
	result, err := c.Request(ctx, "getTransactions", map[string]interface{}{
		"address": address,
		"limit":  limit,
	})
	
	if err != nil {
		return nil, err
	}
	
	// Simplified
	return []Transaction{}, nil
}

// GetNFTs gets NFTs
func (c *Client) GetNFTs(ctx context.Context, address string) ([]NFT, error) {
	result, err := c.Request(ctx, "getNFTs", map[string]interface{}{
		"address": address,
	})
	
	if err != nil {
		return nil, err
	}
	
	// Simplified
	return []NFT{}, nil
}

// GetStaking gets staking info
func (c *Client) GetStaking(ctx context.Context, address string) (*StakingInfo, error) {
	result, err := c.Request(ctx, "getStaking", map[string]interface{}{
		"address": address,
	})
	
	if err != nil {
		return nil, err
	}
	
	var staking StakingInfo
	return &staking, nil
}

// ============================================================================
// Data Types
// ============================================================================

// Portfolio represents portfolio
type Portfolio struct {
	TotalValueUSD string
	Tokens        []TokenBalance
	NFTs          []NFTBalance
}

// TokenBalance represents token balance
type TokenBalance struct {
	Symbol    string
	Balance   string
	ValueUSD  string
	Change24h string
}

// NFTBalance represents NFT balance
type NFTBalance struct {
	Collection string
	TokenID    string
	Name      string
	Image     string
}

// Transaction represents transaction
type Transaction struct {
	TxHash     string
	Type      string
	Status    string
	From      string
	To        string
	Amount    string
	Token     string
	Timestamp int64
}

// NFT represents NFT
type NFT struct {
	TokenID     string
	Collection string
	Name       string
	Image      string
	Metadata   map[string]string
}

// StakingInfo represents staking info
type StakingInfo struct {
	TotalStaked   string
	Rewards      string
	APY          string
	UnlocksAt    int64
}

// ============================================================================
// Widget Integration
// ============================================================================

// WidgetConfig for embedding
type WidgetConfig struct {
	Type      string // "send", "swap", "buy", "stake"
	Theme     string // "light", "dark"
	ChainID   uint64
	Token     string
	Amount    string
	Callback  string
	Width    string
	Height   string
}

// RenderWidget renders widget HTML
func RenderWidget(config *WidgetConfig) string {
	return fmt.Sprintf(`<iframe 
	src="%s/widget?type=%s&theme=%s&chainId=%d"
	width="%s" height="%s" frameborder="0"
	></iframe>`,
		config.Type,
		config.Theme,
		config.ChainID,
		config.Width,
		config.Height,
	)
}

// ============================================================================
// JavaScript SDK
// ============================================================================

// JavaScriptSDK returns JavaScript SDK code
func JavaScriptSDK() string {
	return `
// TigerWallet JavaScript SDK
// Version: 1.0.0

(function(window) {
  'use strict';
  
  var TigerWallet = function(config) {
    this.config = config || {};
    this.baseURL = this.config.baseURL || 'https://api.tigerwallet.com';
    this.chainId = this.config.chainId || 1;
  };
  
  // Connect wallet
  TigerWallet.prototype.connect = async function() {
    if (typeof window.ethereum === 'undefined') {
      throw new Error('No wallet detected');
    }
    
    var accounts = await window.ethereum.request({
      method: 'eth_requestAccounts'
    });
    
    this.address = accounts[0];
    return this.address;
  };
  
  // Get balance
  TigerWallet.prototype.getBalance = async function(token) {
    var response = await fetch(this.baseURL + '/v1/balance', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        address: this.address,
        token: token,
        chainId: this.chainId
      })
    });
    
    return response.json();
  };
  
  // Transfer
  TigerWallet.prototype.transfer = async function(to, amount, token) {
    var method = token === 'ETH' ? 'eth_sendTransaction' : 'eth_sendTransaction';
    var tx = {
      to: to,
      value: '0x' + amount.toString(16)
    };
    
    if (token !== 'ETH') {
      tx.data = '0xa9059cbb000000000000000000' + to.slice(2) + 
        amount.toString(16).padStart(64, '0');
    }
    
    var txHash = await window.ethereum.request({
      method: method,
      params: [tx]
    });
    
    return txHash;
  };
  
  // Sign message
  TigerWallet.prototype.sign = async function(message) {
    var signature = await window.ethereum.request({
      method: 'personal_sign',
      params: [message, this.address]
    });
    
    return signature;
  };
  
  // Switch chain
  TigerWallet.prototype.switchChain = async function(chainId) {
    var chainIds = {
      1: { chainId: '0x1', chainName: 'Ethereum', rpcUrls: ['https://eth-mainnet.alchemyapi.io'] },
      56: { chainId: '0x38', chainName: 'BNB Chain', rpcUrls: ['https://bsc-dataseed.binance.org'] },
      137: { chainId: '0x89', chainName: 'Polygon', rpcUrls: ['https://polygon-rpc.com'] },
      42161: { chainId: '0xa4b1', chainName: 'Arbitrum', rpcUrls: ['https://arb1.arbitrum.io/rpc'] }
    };
    
    var chain = chainIds[chainId];
    if (!chain) {
      throw new Error('Unsupported chain');
    }
    
    try {
      await window.ethereum.request({
        method: 'wallet_switchEthereumChain',
        params: [{ chainId: chain.chainId }]
      });
    } catch (e) {
      if (e.code === 4902) {
        await window.ethereum.request({
          method: 'wallet_addEthereumChain',
          params: [chain]
        });
      }
    }
  };
  
  // Expose to window
  window.TigerWallet = TigerWallet;
  
})(window);
`
}

// ============================================================================
// REST API Endpoints
// ============================================================================

// HandleBalance handles balance requests
func (c *Client) HandleBalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		Address string `json:"address"`
		Token   string `json:"token"`
		ChainID uint64 `json:"chainId"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	result, err := c.Request(r.Context(), "getBalance", map[string]interface{}{
		"address": req.Address,
		"token":   req.Token,
		"chainId": req.ChainID,
	})
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// HandleTransfer handles transfer requests
func (c *Client) HandleTransfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		From   string `json:"from"`
		To     string `json:"to"`
		Amount string `json:"amount"`
		Token  string `json:"token"`
		ChainID uint64 `json:"chainId"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	result, err := c.Request(r.Context(), "transfer", map[string]interface{}{
		"from":   req.From,
		"to":     req.To,
		"amount": req.Amount,
		"token":  req.Token,
		"chainId": req.ChainID,
	})
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// HandlePortfolio handles portfolio requests
func (c *Client) HandlePortfolio(w http.ResponseWriter, r *http.Request) {
	address := r.URL.Query().Get("address")
	if address == "" {
		http.Error(w, "Missing address", http.StatusBadRequest)
		return
	}
	
	portfolio, err := c.GetPortfolio(r.Context(), address)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(portfolio)
}

// HandleTransactions handles transaction requests
func (c *Client) HandleTransactions(w http.ResponseWriter, r *http.Request) {
	address := r.URL.Query().Get("address")
	limit := 50
	
	txs, err := c.GetTransactions(r.Context(), address, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(txs)
}

// HandleSDK handles SDK JS requests
func (c *Client) HandleSDK(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	fmt.Fprint(w, JavaScriptSDK())
}

// Serve starts SDK HTTP server
func (c *Client) Serve(addr string) error {
	http.HandleFunc("/v1/balance", c.HandleBalance)
	http.HandleFunc("/v1/transfer", c.HandleTransfer)
	http.HandleFunc("/v1/portfolio", c.HandlePortfolio)
	http.HandleFunc("/v1/transactions", c.HandleTransactions)
	http.HandleFunc("/sdk.js", c.HandleSDK)
	
	return http.ListenAndServe(addr, nil)
}