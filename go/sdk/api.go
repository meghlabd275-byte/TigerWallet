// Package sdk is the TigerWallet developer SDK.
//
// Requests are authenticated with an HMAC-SHA256 signature over the method,
// path, timestamp and body, using the same scheme the platform services
// verify. The secret never travels on the wire.
package sdk

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ============================================================================
// TigerWallet Developer SDK
// ============================================================================

// maxResponseBytes bounds how much of a response body the SDK will read, so a
// hostile or misconfigured endpoint cannot exhaust memory.
const maxResponseBytes = 8 << 20

// maxRequestBodyBytes bounds request bodies accepted by the handler endpoints.
const maxRequestBodyBytes = 1 << 20

// Client provides TigerWallet SDK functionality.
type Client struct {
	mu         sync.RWMutex
	config     *Config
	httpClient *http.Client
	auth       *Auth
	wallets    map[string]*WalletClient
}

// Config for SDK client.
type Config struct {
	APIKey     string
	APISecret  string
	BaseURL    string
	Timeout    time.Duration
	RetryCount int
	RetryDelay time.Duration
}

// NewClient creates a new SDK client.
//
// It fails closed on a missing credential or a non-HTTPS base URL: sending a
// signature over plaintext would expose the caller to interception, and
// signing with an empty secret would produce a signature any attacker could
// forge.
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		return nil, errors.New("sdk: nil config")
	}
	if config.APIKey == "" || config.APISecret == "" {
		return nil, errors.New("sdk: APIKey and APISecret are required")
	}
	base, err := url.Parse(config.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("sdk: invalid BaseURL: %w", err)
	}
	if base.Host == "" {
		return nil, errors.New("sdk: BaseURL must include a host")
	}
	if base.User != nil {
		return nil, errors.New("sdk: BaseURL must not embed credentials")
	}
	if base.Scheme != "https" && base.Hostname() != "localhost" && base.Hostname() != "127.0.0.1" {
		return nil, fmt.Errorf("sdk: BaseURL must use https, got %q", base.Scheme)
	}
	config.BaseURL = strings.TrimRight(config.BaseURL, "/")

	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.RetryCount == 0 {
		config.RetryCount = 2
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 250 * time.Millisecond
	}

	httpClient := &http.Client{
		Timeout: config.Timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
			Proxy:           http.ProxyFromEnvironment,
		},
		// A redirect could silently move a signed request to another host.
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return &Client{
		config:     config,
		httpClient: httpClient,
		auth:       NewAuth(config.APIKey, config.APISecret),
		wallets:    make(map[string]*WalletClient),
	}, nil
}

// ============================================================================
// Authentication
// ============================================================================

// Auth signs requests with HMAC-SHA256.
//
// There is no token endpoint: the signature is derived from the secret for
// every request, so nothing bearer-like is cached or replayed.
type Auth struct {
	APIKey    string
	APISecret string
	mu        sync.RWMutex
}

// NewAuth creates an authenticator.
func NewAuth(apiKey, apiSecret string) *Auth {
	return &Auth{APIKey: apiKey, APISecret: apiSecret}
}

// Sign returns the hex-encoded signature for a request.
//
// The signed message is "METHOD\nPATH\nTIMESTAMP\nBODY". The timestamp is
// included so a captured signature cannot be replayed outside its validity
// window.
func (a *Auth) Sign(method, path, timestamp string, body []byte) string {
	a.mu.RLock()
	secret := a.APISecret
	a.mu.RUnlock()

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(method))
	mac.Write([]byte("\n"))
	mac.Write([]byte(path))
	mac.Write([]byte("\n"))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("\n"))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// ============================================================================
// Wallet Client
// ============================================================================

// WalletClient provides wallet functionality.
type WalletClient struct {
	mu       sync.RWMutex
	client   *Client
	address  string
	chainID  uint64
	balances map[string]*Balance
}

// Balance represents token balance.
type Balance struct {
	Symbol   string
	Amount   *big.Int
	Decimals uint8
	ValueUSD *big.Rat
}

// NewWallet creates a wallet client bound to an address and chain.
func (c *Client) NewWallet(address string, chainID uint64) *WalletClient {
	w := &WalletClient{
		client:   c,
		address:  address,
		chainID:  chainID,
		balances: make(map[string]*Balance),
	}
	c.mu.Lock()
	c.wallets[address] = w
	c.mu.Unlock()
	return w
}

// GetAddress returns the wallet address.
func (w *WalletClient) GetAddress() string {
	return w.address
}

// GetBalance returns the token balance.
func (w *WalletClient) GetBalance(ctx context.Context, token string) (*Balance, error) {
	raw, err := w.client.Call(ctx, "getBalance", map[string]interface{}{
		"address": w.address,
		"token":   token,
		"chainId": w.chainID,
	})
	if err != nil {
		return nil, err
	}

	var payload struct {
		Symbol   string `json:"symbol"`
		Amount   string `json:"amount"`
		Decimals uint8  `json:"decimals"`
		ValueUSD string `json:"valueUsd"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("sdk: getBalance response: %w", err)
	}

	balance := &Balance{Symbol: payload.Symbol, Decimals: payload.Decimals, Amount: new(big.Int)}
	if payload.Amount != "" {
		amount, ok := new(big.Int).SetString(payload.Amount, 10)
		if !ok {
			return nil, fmt.Errorf("sdk: getBalance returned a non-numeric amount %q", payload.Amount)
		}
		balance.Amount = amount
	}
	if payload.ValueUSD != "" {
		if value, ok := new(big.Rat).SetString(payload.ValueUSD); ok {
			balance.ValueUSD = value
		}
	}

	w.mu.Lock()
	w.balances[token] = balance
	w.mu.Unlock()

	return balance, nil
}

// Transfer sends tokens and returns the transaction hash.
//
// A hash is only returned when the backend reports one. A response without a
// hash is an error, because a caller that treated it as success would report a
// transfer that never happened.
func (w *WalletClient) Transfer(ctx context.Context, to string, amount *big.Int, token string) (string, error) {
	if amount == nil || amount.Sign() <= 0 {
		return "", errors.New("sdk: amount must be positive")
	}
	if to == "" {
		return "", errors.New("sdk: recipient is required")
	}

	raw, err := w.client.Call(ctx, "transfer", map[string]interface{}{
		"from":    w.address,
		"to":      to,
		"amount":  amount.String(),
		"token":   token,
		"chainId": w.chainID,
	})
	if err != nil {
		return "", err
	}
	return txHashFrom(raw, "transfer")
}

// Swap swaps tokens and returns the transaction hash.
func (w *WalletClient) Swap(ctx context.Context, fromToken, toToken string, amount *big.Int) (string, error) {
	if amount == nil || amount.Sign() <= 0 {
		return "", errors.New("sdk: amount must be positive")
	}
	if fromToken == "" || toToken == "" {
		return "", errors.New("sdk: fromToken and toToken are required")
	}

	raw, err := w.client.Call(ctx, "swap", map[string]interface{}{
		"address":   w.address,
		"fromToken": fromToken,
		"toToken":   toToken,
		"amount":    amount.String(),
		"chainId":   w.chainID,
	})
	if err != nil {
		return "", err
	}
	return txHashFrom(raw, "swap")
}

// txHashFrom extracts a transaction hash from a backend result.
func txHashFrom(raw json.RawMessage, op string) (string, error) {
	var direct string
	if err := json.Unmarshal(raw, &direct); err == nil && direct != "" {
		return direct, nil
	}

	var envelope struct {
		TxHash string `json:"tx_hash"`
	}
	if err := json.Unmarshal(raw, &envelope); err == nil && envelope.TxHash != "" {
		return envelope.TxHash, nil
	}

	return "", fmt.Errorf("sdk: %s succeeded but the backend returned no transaction hash", op)
}

// ============================================================================
// API Methods
// ============================================================================

// rpcRequest is the JSON-RPC envelope sent to the backend.
type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      int64           `json:"id"`
}

// rpcResponse is the JSON-RPC envelope returned by the backend.
type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error"`
	ID      int64           `json:"id"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *rpcError) Error() string {
	return fmt.Sprintf("rpc error %d: %s", e.Code, e.Message)
}

// Call performs a signed JSON-RPC request and returns the result.
func (c *Client) Call(ctx context.Context, method string, params map[string]interface{}) (json.RawMessage, error) {
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("sdk: marshal params: %w", err)
	}

	body, err := json.Marshal(rpcRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  paramsJSON,
		ID:      time.Now().UnixNano(),
	})
	if err != nil {
		return nil, fmt.Errorf("sdk: marshal request: %w", err)
	}

	const path = "/v1/rpc"

	var lastErr error
	for attempt := 0; attempt <= c.config.RetryCount; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(c.config.RetryDelay * time.Duration(attempt)):
			}
		}

		result, retryable, err := c.doCall(ctx, path, method, body)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if !retryable {
			return nil, err
		}
	}

	return nil, lastErr
}

// doCall performs one signed attempt. It reports whether a retry is sensible.
func (c *Client) doCall(ctx context.Context, path, method string, body []byte) (json.RawMessage, bool, error) {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	signature := c.auth.Sign(http.MethodPost, path, timestamp, body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, false, fmt.Errorf("sdk: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-API-Key", c.auth.APIKey)
	req.Header.Set("X-Timestamp", timestamp)
	req.Header.Set("X-Signature", signature)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, true, fmt.Errorf("sdk: %s: %w", method, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, true, fmt.Errorf("sdk: %s: read response: %w", method, err)
	}

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("sdk: %s: backend returned %s", method, resp.Status)
	}
	if resp.StatusCode >= 400 {
		var envelope rpcResponse
		if json.Unmarshal(raw, &envelope) == nil && envelope.Error != nil {
			return nil, false, fmt.Errorf("sdk: %s: %w", method, envelope.Error)
		}
		return nil, false, fmt.Errorf("sdk: %s: backend returned %s", method, resp.Status)
	}

	var envelope rpcResponse
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, false, fmt.Errorf("sdk: %s: decode response: %w", method, err)
	}
	if envelope.Error != nil {
		return nil, false, fmt.Errorf("sdk: %s: %w", method, envelope.Error)
	}
	if len(envelope.Result) == 0 {
		return nil, false, fmt.Errorf("sdk: %s: response carried no result", method)
	}

	return envelope.Result, false, nil
}

// Request is kept for callers that want the decoded result as a generic value.
func (c *Client) Request(ctx context.Context, method string, params map[string]interface{}) (interface{}, error) {
	raw, err := c.Call(ctx, method, params)
	if err != nil {
		return nil, err
	}
	var out interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("sdk: decode result: %w", err)
	}
	return out, nil
}

// ============================================================================
// Pre-built SDK Functions
// ============================================================================

// GetPortfolio gets the portfolio for an address.
func (c *Client) GetPortfolio(ctx context.Context, address string) (*Portfolio, error) {
	if address == "" {
		return nil, errors.New("sdk: address is required")
	}

	raw, err := c.Call(ctx, "getPortfolio", map[string]interface{}{"address": address})
	if err != nil {
		return nil, err
	}

	var portfolio Portfolio
	if err := json.Unmarshal(raw, &portfolio); err != nil {
		return nil, fmt.Errorf("sdk: getPortfolio response: %w", err)
	}
	return &portfolio, nil
}

// GetTransactions gets transactions for an address.
func (c *Client) GetTransactions(ctx context.Context, address string, limit int) ([]Transaction, error) {
	if address == "" {
		return nil, errors.New("sdk: address is required")
	}
	if limit <= 0 {
		limit = 50
	}

	raw, err := c.Call(ctx, "getTransactions", map[string]interface{}{
		"address": address,
		"limit":   limit,
	})
	if err != nil {
		return nil, err
	}
	return decodeList[Transaction](raw, "transactions")
}

// GetNFTs gets NFTs for an address.
func (c *Client) GetNFTs(ctx context.Context, address string) ([]NFT, error) {
	if address == "" {
		return nil, errors.New("sdk: address is required")
	}

	raw, err := c.Call(ctx, "getNFTs", map[string]interface{}{"address": address})
	if err != nil {
		return nil, err
	}
	return decodeList[NFT](raw, "nfts")
}

// GetStaking gets staking info for an address.
func (c *Client) GetStaking(ctx context.Context, address string) (*StakingInfo, error) {
	if address == "" {
		return nil, errors.New("sdk: address is required")
	}

	raw, err := c.Call(ctx, "getStaking", map[string]interface{}{"address": address})
	if err != nil {
		return nil, err
	}

	var staking StakingInfo
	if err := json.Unmarshal(raw, &staking); err != nil {
		return nil, fmt.Errorf("sdk: getStaking response: %w", err)
	}
	return &staking, nil
}

// decodeList reads an array result, accepting either a bare array or an object
// that wraps it under key.
func decodeList[T any](raw json.RawMessage, key string) ([]T, error) {
	var direct []T
	if err := json.Unmarshal(raw, &direct); err == nil {
		return direct, nil
	}

	var wrapped map[string]json.RawMessage
	if err := json.Unmarshal(raw, &wrapped); err != nil {
		return nil, fmt.Errorf("sdk: decode %s: %w", key, err)
	}
	inner, ok := wrapped[key]
	if !ok {
		return nil, fmt.Errorf("sdk: response has no %q field", key)
	}
	if err := json.Unmarshal(inner, &direct); err != nil {
		return nil, fmt.Errorf("sdk: decode %s: %w", key, err)
	}
	return direct, nil
}

// ============================================================================
// Data Types
// ============================================================================

// Portfolio represents a portfolio.
type Portfolio struct {
	TotalValueUSD string         `json:"totalValueUSD"`
	Tokens        []TokenBalance `json:"tokens"`
	NFTs          []NFTBalance   `json:"nfts"`
}

// TokenBalance represents a token balance.
type TokenBalance struct {
	Symbol    string `json:"symbol"`
	Balance   string `json:"balance"`
	ValueUSD  string `json:"valueUsd"`
	Change24h string `json:"change24h"`
}

// NFTBalance represents an NFT holding.
type NFTBalance struct {
	Collection string `json:"collection"`
	TokenID    string `json:"tokenId"`
	Name       string `json:"name"`
	Image      string `json:"image"`
}

// Transaction represents a transaction.
type Transaction struct {
	TxHash    string `json:"txHash"`
	Type      string `json:"type"`
	Status    string `json:"status"`
	From      string `json:"from"`
	To        string `json:"to"`
	Amount    string `json:"amount"`
	Token     string `json:"token"`
	Timestamp int64  `json:"timestamp"`
}

// NFT represents an NFT.
type NFT struct {
	TokenID    string            `json:"tokenId"`
	Collection string            `json:"collection"`
	Name       string            `json:"name"`
	Image      string            `json:"image"`
	Metadata   map[string]string `json:"metadata"`
}

// StakingInfo represents staking info.
type StakingInfo struct {
	TotalStaked string `json:"totalStaked"`
	Rewards     string `json:"rewards"`
	APY         string `json:"apy"`
	UnlocksAt   int64  `json:"unlocksAt"`
}

// ============================================================================
// Widget Integration
// ============================================================================

// WidgetConfig for embedding.
type WidgetConfig struct {
	Type     string // "send", "swap", "buy", "stake"
	Theme    string // "light", "dark"
	ChainID  uint64
	Token    string
	Amount   string
	Callback string
	Width    string
	Height   string
}

// RenderWidget renders widget HTML.
//
// Every value is escaped: the fields are caller-supplied, and interpolating
// them raw would let a token symbol or callback URL break out of the attribute
// and inject script into the embedding page.
func RenderWidget(baseURL string, config *WidgetConfig) (string, error) {
	if config == nil {
		return "", errors.New("sdk: nil widget config")
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("sdk: invalid widget base URL: %w", err)
	}
	if base.Host == "" {
		return "", errors.New("sdk: widget base URL must include a host")
	}
	if base.User != nil {
		return "", errors.New("sdk: widget base URL must not embed credentials")
	}
	if base.Scheme != "https" && base.Hostname() != "localhost" && base.Hostname() != "127.0.0.1" {
		return "", fmt.Errorf("sdk: widget base URL must use https, got %q", base.Scheme)
	}

	theme := config.Theme
	if theme != "light" && theme != "dark" {
		theme = "light"
	}
	width := config.Width
	if width == "" {
		width = "420"
	}
	height := config.Height
	if height == "" {
		height = "640"
	}

	// Values go through url.Values so the query string is correctly encoded.
	query := url.Values{}
	query.Set("type", config.Type)
	query.Set("theme", theme)
	query.Set("chainId", strconv.FormatUint(config.ChainID, 10))
	if config.Token != "" {
		query.Set("token", config.Token)
	}
	if config.Amount != "" {
		query.Set("amount", config.Amount)
	}
	if config.Callback != "" {
		callback, err := url.Parse(config.Callback)
		if err != nil {
			return "", fmt.Errorf("sdk: invalid callback URL: %w", err)
		}
		if callback.Scheme != "https" {
			return "", errors.New("sdk: callback URL must use https")
		}
		query.Set("callback", callback.String())
	}

	src := base.String() + "/widget?" + query.Encode()

	return fmt.Sprintf(
		`<iframe src="%s" width="%s" height="%s" frameborder="0" sandbox="allow-scripts allow-forms allow-popups" referrerpolicy="no-referrer"></iframe>`,
		html.EscapeString(src),
		html.EscapeString(width),
		html.EscapeString(height),
	), nil
}

// ============================================================================
// JavaScript SDK
// ============================================================================

// JavaScriptSDK returns the embeddable JavaScript SDK.
//
// Chain parameters are fetched from the backend at runtime rather than
// hardcoded here, so the wallet and the SDK cannot drift apart.
func JavaScriptSDK() string {
	return `// TigerWallet JavaScript SDK
// Version: 1.0.0
(function (window) {
  'use strict';

  var ERC20_TRANSFER_SELECTOR = '0xa9059cbb';

  function pad64(hex) {
    if (hex.length > 64) {
      throw new Error('value does not fit in 32 bytes');
    }
    return hex.padStart(64, '0');
  }

  function toHexQuantity(value) {
    var big;
    try {
      big = BigInt(value);
    } catch (e) {
      throw new Error('amount is not an integer: ' + value);
    }
    if (big <= 0n) {
      throw new Error('amount must be positive');
    }
    return '0x' + big.toString(16);
  }

  // Encode an ERC-20 transfer(address,uint256) call.
  function encodeErc20Transfer(to, amount) {
    if (!/^0x[0-9a-fA-F]{40}$/.test(to)) {
      throw new Error('recipient is not a 20-byte address');
    }
    var amountHex = BigInt(amount).toString(16);
    return ERC20_TRANSFER_SELECTOR + pad64(to.slice(2).toLowerCase()) + pad64(amountHex);
  }

  function TigerWallet(config) {
    this.config = config || {};
    this.baseURL = (this.config.baseURL || '').replace(/\/+$/, '');
    this.chainId = this.config.chainId || 1;
    this.address = null;
    this._chains = null;
  }

  TigerWallet.prototype._post = async function (path, body) {
    var response = await fetch(this.baseURL + path, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body)
    });
    if (!response.ok) {
      throw new Error(path + ' failed with status ' + response.status);
    }
    return response.json();
  };

  TigerWallet.prototype.connect = async function () {
    if (typeof window.ethereum === 'undefined') {
      throw new Error('No EIP-1193 provider detected');
    }
    var accounts = await window.ethereum.request({ method: 'eth_requestAccounts' });
    if (!accounts || !accounts.length) {
      throw new Error('The provider returned no accounts');
    }
    this.address = accounts[0];
    return this.address;
  };

  TigerWallet.prototype.getBalance = function (token) {
    if (!this.address) {
      return Promise.reject(new Error('connect() first'));
    }
    return this._post('/v1/balance', {
      address: this.address,
      token: token,
      chainId: this.chainId
    });
  };

  TigerWallet.prototype.transfer = async function (to, amount, token) {
    if (typeof window.ethereum === 'undefined') {
      throw new Error('No EIP-1193 provider detected');
    }
    if (!this.address) {
      throw new Error('connect() first');
    }

    var tx;
    if (!token || token === 'ETH' || token === 'NATIVE') {
      tx = { from: this.address, to: to, value: toHexQuantity(amount) };
    } else {
      var contract = this.config.tokenContracts && this.config.tokenContracts[token];
      if (!contract) {
        throw new Error('no contract address configured for ' + token);
      }
      tx = {
        from: this.address,
        to: contract,
        value: '0x0',
        data: encodeErc20Transfer(to, amount)
      };
    }

    return window.ethereum.request({ method: 'eth_sendTransaction', params: [tx] });
  };

  TigerWallet.prototype.sign = async function (message) {
    if (typeof window.ethereum === 'undefined') {
      throw new Error('No EIP-1193 provider detected');
    }
    if (!this.address) {
      throw new Error('connect() first');
    }
    return window.ethereum.request({
      method: 'personal_sign',
      params: [message, this.address]
    });
  };

  TigerWallet.prototype.loadChains = async function () {
    if (this._chains) {
      return this._chains;
    }
    var response = await fetch(this.baseURL + '/v1/chains');
    if (!response.ok) {
      throw new Error('could not load chain registry: ' + response.status);
    }
    var payload = await response.json();
    var chains = payload.chains || payload;
    var byId = {};
    for (var i = 0; i < chains.length; i++) {
      byId[Number(chains[i].chainId)] = chains[i];
    }
    this._chains = byId;
    return byId;
  };

  TigerWallet.prototype.switchChain = async function (chainId) {
    if (typeof window.ethereum === 'undefined') {
      throw new Error('No EIP-1193 provider detected');
    }
    var chains = await this.loadChains();
    var chain = chains[chainId];
    if (!chain) {
      throw new Error('chain ' + chainId + ' is not in the registry');
    }
    var params = {
      chainId: '0x' + Number(chainId).toString(16),
      chainName: chain.name,
      nativeCurrency: chain.nativeCurrency,
      rpcUrls: chain.rpcUrls
    };
    if (chain.blockExplorerUrls && chain.blockExplorerUrls.length) {
      params.blockExplorerUrls = chain.blockExplorerUrls;
    }

    try {
      await window.ethereum.request({
        method: 'wallet_switchEthereumChain',
        params: [{ chainId: params.chainId }]
      });
    } catch (e) {
      // 4902 means the chain is unknown to the wallet; offer to add it.
      if (e && e.code === 4902) {
        await window.ethereum.request({
          method: 'wallet_addEthereumChain',
          params: [params]
        });
      } else {
        throw e;
      }
    }
  };

  window.TigerWallet = TigerWallet;
})(window);
`
}

// ============================================================================
// REST API Endpoints
// ============================================================================

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// writeError writes a JSON error.
//
// The message is sent to the caller, so it must describe the fault without
// echoing internal detail.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// decodeBody reads a bounded JSON body into dst.
func decodeBody(w http.ResponseWriter, r *http.Request, dst interface{}) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return errors.New("invalid JSON body")
	}
	return nil
}

// HandleBalance handles balance requests.
func (c *Client) HandleBalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Address string `json:"address"`
		Token   string `json:"token"`
		ChainID uint64 `json:"chainId"`
	}
	if err := decodeBody(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Address == "" {
		writeError(w, http.StatusBadRequest, "address is required")
		return
	}

	raw, err := c.Call(r.Context(), "getBalance", map[string]interface{}{
		"address": req.Address,
		"token":   req.Token,
		"chainId": req.ChainID,
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, "upstream request failed")
		return
	}
	writeJSON(w, http.StatusOK, raw)
}

// HandleTransfer handles transfer requests.
func (c *Client) HandleTransfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		From    string `json:"from"`
		To      string `json:"to"`
		Amount  string `json:"amount"`
		Token   string `json:"token"`
		ChainID uint64 `json:"chainId"`
	}
	if err := decodeBody(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.From == "" || req.To == "" {
		writeError(w, http.StatusBadRequest, "from and to are required")
		return
	}
	amount, ok := new(big.Int).SetString(req.Amount, 10)
	if !ok || amount.Sign() <= 0 {
		writeError(w, http.StatusBadRequest, "amount must be a positive integer")
		return
	}

	raw, err := c.Call(r.Context(), "transfer", map[string]interface{}{
		"from":    req.From,
		"to":      req.To,
		"amount":  amount.String(),
		"token":   req.Token,
		"chainId": req.ChainID,
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, "upstream request failed")
		return
	}
	writeJSON(w, http.StatusOK, raw)
}

// HandlePortfolio handles portfolio requests.
func (c *Client) HandlePortfolio(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	address := r.URL.Query().Get("address")
	if address == "" {
		writeError(w, http.StatusBadRequest, "address is required")
		return
	}

	portfolio, err := c.GetPortfolio(r.Context(), address)
	if err != nil {
		writeError(w, http.StatusBadGateway, "upstream request failed")
		return
	}
	writeJSON(w, http.StatusOK, portfolio)
}

// HandleTransactions handles transaction requests.
func (c *Client) HandleTransactions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	address := r.URL.Query().Get("address")
	if address == "" {
		writeError(w, http.StatusBadRequest, "address is required")
		return
	}

	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 || parsed > 500 {
			writeError(w, http.StatusBadRequest, "limit must be an integer between 1 and 500")
			return
		}
		limit = parsed
	}

	txs, err := c.GetTransactions(r.Context(), address, limit)
	if err != nil {
		writeError(w, http.StatusBadGateway, "upstream request failed")
		return
	}
	writeJSON(w, http.StatusOK, txs)
}

// HandleChains serves the chain registry used by the JavaScript SDK.
func (c *Client) HandleChains(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	raw, err := c.Call(r.Context(), "getChains", map[string]interface{}{})
	if err != nil {
		writeError(w, http.StatusBadGateway, "upstream request failed")
		return
	}
	writeJSON(w, http.StatusOK, raw)
}

// HandleSDK serves the JavaScript SDK.
func (c *Client) HandleSDK(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = io.WriteString(w, JavaScriptSDK())
}

// Serve starts the SDK HTTP server.
func (c *Client) Serve(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/balance", c.HandleBalance)
	mux.HandleFunc("/v1/transfer", c.HandleTransfer)
	mux.HandleFunc("/v1/portfolio", c.HandlePortfolio)
	mux.HandleFunc("/v1/transactions", c.HandleTransactions)
	mux.HandleFunc("/v1/chains", c.HandleChains)
	mux.HandleFunc("/sdk.js", c.HandleSDK)

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	return server.ListenAndServe()
}
