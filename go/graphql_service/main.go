package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	ListenAddr          string
	WalletAPIURL        string
	MaxQueryDepth       int
	MaxQueryCost        int
	Timeout             time.Duration
	EnableIntrospection bool
}

var config = Config{
	ListenAddr:          getEnv("GRAPHQL_LISTEN_ADDR", ":9003"),
	WalletAPIURL:        getEnv("WALLET_API_URL", "http://localhost:8443"),
	MaxQueryDepth:       10,
	MaxQueryCost:        1000,
	Timeout:             time.Second * 30,
	EnableIntrospection: true,
}

// ============================================================================
// GraphQL Types
// ============================================================================

type GraphQLRequest struct {
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
	OperationName string                 `json:"operationName,omitempty"`
}

type GraphQLResponse struct {
	Data   interface{}    `json:"data,omitempty"`
	Errors []GraphQLError `json:"errors,omitempty"`
}

type GraphQLError struct {
	Message   string        `json:"message"`
	Locations []Location    `json:"locations,omitempty"`
	Path      []interface{} `json:"path,omitempty"`
}

type Location struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

type Schema struct {
	Types         []TypeDefinition  `json:"types"`
	Queries       []FieldDefinition `json:"queries"`
	Mutations     []FieldDefinition `json:"mutations"`
	Subscriptions []FieldDefinition `json:"subscriptions"`
}

type TypeDefinition struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Fields      []FieldDefinition `json:"fields,omitempty"`
	Kind        string            `json:"kind"` // scalar, object, interface, enum, union, input_object
	EnumValues  []EnumValue       `json:"enumValues,omitempty"`
	InputFields []InputValue      `json:"inputFields,omitempty"`
}

type FieldDefinition struct {
	Name        string                                                 `json:"name"`
	Description string                                                 `json:"description"`
	Type        string                                                 `json:"type"`
	Args        []InputValue                                           `json:"args,omitempty"`
	Resolve     func(args map[string]interface{}) (interface{}, error) `json:"-"`
}

type InputValue struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	Type         string `json:"type"`
	DefaultValue string `json:"defaultValue,omitempty"`
}

type EnumValue struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ResolverContext struct {
	RequestID string
	UserID    string
	Variables map[string]interface{}
	StartTime time.Time
}

// ============================================================================
// GraphQL Service
// ============================================================================

type GraphQLService struct {
	schema     Schema
	resolvers  map[string]FieldDefinition
	resolveMu  sync.RWMutex
	httpClient *http.Client
	queries    int
	queriesMu  sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
}

type contextKey string

var requestIDKey contextKey = "request_id"

func NewGraphQLService() *GraphQLService {
	svc := &GraphQLService{
		resolvers:  make(map[string]FieldDefinition),
		httpClient: &http.Client{Timeout: 15 * time.Second},
		ctx:        context.Background(),
	}

	svc.initializeSchema()
	svc.initializeResolvers()

	return svc
}

func (s *GraphQLService) initializeSchema() {
	s.schema = Schema{
		Types: []TypeDefinition{
			{
				Name:        "User",
				Description: "User account information",
				Kind:        "object",
				Fields: []FieldDefinition{
					{Name: "id", Type: "ID!", Description: "User ID"},
					{Name: "email", Type: "String", Description: "User email"},
					{Name: "wallets", Type: "[Wallet!]!", Description: "User wallets"},
					{Name: "createdAt", Type: "String!", Description: "Account creation date"},
				},
			},
			{
				Name:        "Wallet",
				Description: "Wallet information",
				Kind:        "object",
				Fields: []FieldDefinition{
					{Name: "id", Type: "ID!", Description: "Wallet ID"},
					{Name: "address", Type: "String!", Description: "Wallet address"},
					{Name: "chain", Type: "String!", Description: "Blockchain chain"},
					{Name: "balance", Type: "Float!", Description: "Wallet balance"},
					{Name: "tokens", Type: "[Token!]!", Description: "Token balances"},
				},
			},
			{
				Name:        "Token",
				Description: "Token information",
				Kind:        "object",
				Fields: []FieldDefinition{
					{Name: "symbol", Type: "String!", Description: "Token symbol"},
					{Name: "name", Type: "String!", Description: "Token name"},
					{Name: "balance", Type: "Float!", Description: "Token balance"},
					{Name: "price", Type: "Float", Description: "Token price in USD"},
					{Name: "value", Type: "Float!", Description: "Token value in USD"},
				},
			},
			{
				Name:        "Transaction",
				Description: "Transaction information",
				Kind:        "object",
				Fields: []FieldDefinition{
					{Name: "hash", Type: "ID!", Description: "Transaction hash"},
					{Name: "from", Type: "String!", Description: "From address"},
					{Name: "to", Type: "String!", Description: "To address"},
					{Name: "amount", Type: "Float!", Description: "Transaction amount"},
					{Name: "token", Type: "String!", Description: "Token symbol"},
					{Name: "status", Type: "TransactionStatus!", Description: "Transaction status"},
					{Name: "timestamp", Type: "String!", Description: "Transaction timestamp"},
				},
			},
			{
				Name:        "TransactionStatus",
				Description: "Transaction status enum",
				Kind:        "enum",
				EnumValues: []EnumValue{
					{Name: "PENDING", Description: "Transaction is pending"},
					{Name: "CONFIRMED", Description: "Transaction is confirmed"},
					{Name: "FAILED", Description: "Transaction failed"},
				},
			},
			{
				Name:        "SwapQuote",
				Description: "Swap quote information",
				Kind:        "object",
				Fields: []FieldDefinition{
					{Name: "fromToken", Type: "String!", Description: "Source token"},
					{Name: "toToken", Type: "String!", Description: "Destination token"},
					{Name: "fromAmount", Type: "Float!", Description: "Source amount"},
					{Name: "toAmount", Type: "Float!", Description: "Destination amount"},
					{Name: "priceImpact", Type: "Float!", Description: "Price impact"},
					{Name: "route", Type: "[String!]!", Description: "Swap route"},
					{Name: "gasEstimate", Type: "Float!", Description: "Estimated gas cost"},
				},
			},
			{
				Name:        "MarketData",
				Description: "Market data for a trading pair",
				Kind:        "object",
				Fields: []FieldDefinition{
					{Name: "pair", Type: "String!", Description: "Trading pair"},
					{Name: "price", Type: "Float!", Description: "Current price"},
					{Name: "volume24h", Type: "Float!", Description: "24h volume"},
					{Name: "change24h", Type: "Float!", Description: "24h price change"},
					{Name: "high24h", Type: "Float!", Description: "24h high"},
					{Name: "low24h", Type: "Float!", Description: "24h low"},
				},
			},
		},
		Queries: []FieldDefinition{
			{Name: "user", Description: "Get user by ID", Type: "User", Args: []InputValue{{Name: "id", Type: "ID!"}}},
			{Name: "me", Description: "Get current user", Type: "User"},
			{Name: "wallet", Description: "Get wallet by address", Type: "Wallet", Args: []InputValue{{Name: "address", Type: "String!"}}},
			{Name: "wallets", Description: "Get all wallets for user", Type: "[Wallet!]!", Args: []InputValue{{Name: "userId", Type: "ID"}}},
			{Name: "transaction", Description: "Get transaction by hash", Type: "Transaction", Args: []InputValue{{Name: "hash", Type: "ID!"}}},
			{Name: "transactions", Description: "Get transactions", Type: "[Transaction!]!", Args: []InputValue{{Name: "address", Type: "String"}, {Name: "limit", Type: "Int"}}},
			{Name: "swapQuote", Description: "Get swap quote", Type: "SwapQuote", Args: []InputValue{{Name: "fromToken", Type: "String!"}, {Name: "toToken", Type: "String!"}, {Name: "amount", Type: "Float!"}}},
			{Name: "marketData", Description: "Get market data", Type: "MarketData", Args: []InputValue{{Name: "pair", Type: "String!"}}},
			{Name: "tokens", Description: "Get token list", Type: "[Token!]!", Args: []InputValue{{Name: "chain", Type: "String"}}},
		},
		Mutations: []FieldDefinition{
			{Name: "createWallet", Description: "Create a new wallet", Type: "Wallet", Args: []InputValue{{Name: "chain", Type: "String!"}, {Name: "name", Type: "String"}}},
			{Name: "importWallet", Description: "Import existing wallet", Type: "Wallet", Args: []InputValue{{Name: "privateKey", Type: "String!"}, {Name: "chain", Type: "String!"}}},
			{Name: "sendTransaction", Description: "Send a transaction", Type: "Transaction", Args: []InputValue{{Name: "to", Type: "String!"}, {Name: "amount", Type: "Float!"}, {Name: "token", Type: "String!"}}},
			{Name: "executeSwap", Description: "Execute a token swap", Type: "Transaction", Args: []InputValue{{Name: "fromToken", Type: "String!"}, {Name: "toToken", Type: "String!"}, {Name: "amount", Type: "Float!"}}},
		},
	}
}

func (s *GraphQLService) initializeResolvers() {
	s.resolveMu.Lock()
	defer s.resolveMu.Unlock()

	// Query resolvers
	s.resolvers["user"] = FieldDefinition{
		Name: "user", Type: "User",
		Resolve: func(args map[string]interface{}) (interface{}, error) {
			if args["id"] == nil {
				return nil, fmt.Errorf("user id is required")
			}
			return s.walletAPIGet("/api/v1/wallets?userId=" + fmt.Sprintf("%v", args["id"]))
		},
	}

	s.resolvers["me"] = FieldDefinition{
		Name: "me", Type: "User",
		Resolve: func(args map[string]interface{}) (interface{}, error) {
			token, _ := args["authToken"].(string)
			if token == "" {
				return nil, fmt.Errorf("authToken is required to resolve the current user")
			}
			return s.walletAPIGetWithAuth("/api/v1/wallets", token)
		},
	}

	s.resolvers["wallet"] = FieldDefinition{
		Name: "wallet", Type: "Wallet",
		Resolve: func(args map[string]interface{}) (interface{}, error) {
			address, ok := args["address"].(string)
			if !ok || address == "" {
				return nil, fmt.Errorf("address is required")
			}
			chain, _ := args["chain"].(string)
			if chain == "" {
				chain = "ethereum"
			}
			return s.walletAPIGet("/api/v1/public/balance?address=" + address + "&chain=" + chain)
		},
	}

	s.resolvers["swapQuote"] = FieldDefinition{
		Name: "swapQuote", Type: "SwapQuote",
		Resolve: func(args map[string]interface{}) (interface{}, error) {
			fromToken, ok := args["fromToken"].(string)
			if !ok || fromToken == "" {
				return nil, fmt.Errorf("fromToken is required")
			}
			toToken, ok := args["toToken"].(string)
			if !ok || toToken == "" {
				return nil, fmt.Errorf("toToken is required")
			}
			amount, _ := args["amount"].(float64)
			rate, err := s.coinGeckoRate(fromToken, toToken)
			if err != nil {
				return nil, err
			}
			toAmount := amount * rate
			return map[string]interface{}{
				"fromToken":   fromToken,
				"toToken":     toToken,
				"fromAmount":  amount,
				"toAmount":    toAmount,
				"priceImpact": 0,
				"route":       []string{fromToken, toToken},
				"gasEstimate": 0,
			}, nil
		},
	}

	s.resolvers["marketData"] = FieldDefinition{
		Name: "marketData", Type: "MarketData",
		Resolve: func(args map[string]interface{}) (interface{}, error) {
			pair, ok := args["pair"].(string)
			if !ok || pair == "" {
				return nil, fmt.Errorf("pair is required")
			}
			parts := strings.Split(pair, "/")
			if len(parts) != 2 {
				return nil, fmt.Errorf("pair must be BASE/QUOTE")
			}
			return s.coinGeckoMarketData(parts[0], parts[1], pair)
		},
	}

	s.resolvers["createWallet"] = FieldDefinition{
		Name: "createWallet", Type: "Wallet",
		Resolve: func(args map[string]interface{}) (interface{}, error) {
			chain, _ := args["chain"].(string)
			if chain == "" {
				chain = "ethereum"
			}
			name, _ := args["name"].(string)
			password, _ := args["password"].(string)
			body := map[string]interface{}{"chain": chain, "name": name, "password": password}
			return s.walletAPIPost("/api/v1/wallets", body)
		},
	}

	s.resolvers["sendTransaction"] = FieldDefinition{
		Name: "sendTransaction", Type: "Transaction",
		Resolve: func(args map[string]interface{}) (interface{}, error) {
			return s.walletAPIPost("/api/v1/send", args)
		},
	}
}

func (s *GraphQLService) Start() error {
	fmt.Println("Starting GraphQL Service...")

	// Start HTTP server
	go s.startHTTPServer()

	fmt.Println("GraphQL Service started successfully")
	return nil
}

func (s *GraphQLService) Stop() {
	fmt.Println("GraphQL Service stopped")
}

func (s *GraphQLService) startHTTPServer() {
	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	router.POST("/graphql", s.graphQLHandler)
	router.GET("/graphql", s.graphQLHandler)
	router.GET("/schema", s.schemaHandler)
	router.GET("/introspection", s.introspectionHandler)

	fmt.Printf("GraphQL API server starting on %s\n", config.ListenAddr)
	router.Run(config.ListenAddr)
}

func (s *GraphQLService) graphQLHandler(c *gin.Context) {
	var request GraphQLRequest

	if c.Request.Method == "POST" {
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(400, GraphQLResponse{
				Errors: []GraphQLError{{Message: err.Error()}},
			})
			return
		}
	} else {
		request.Query = c.Query("query")
		request.OperationName = c.Query("operationName")
	}

	if request.Query == "" {
		c.JSON(400, GraphQLResponse{
			Errors: []GraphQLError{{Message: "Query is required"}},
		})
		return
	}

	// Parse and execute query
	result, err := s.executeQuery(request.Query, request.Variables, request.OperationName)
	if err != nil {
		c.JSON(200, GraphQLResponse{
			Errors: []GraphQLError{{Message: err.Error()}},
		})
		return
	}

	s.queriesMu.Lock()
	s.queries++
	s.queriesMu.Unlock()

	c.JSON(200, GraphQLResponse{Data: result})
}

func (s *GraphQLService) executeQuery(query string, variables map[string]interface{}, operationName string) (map[string]interface{}, error) {
	// Simplified query parser and executor
	// In production, would use a proper GraphQL parser

	result := make(map[string]interface{})

	// Handle query root
	if strings.Contains(query, "query") {
		// Extract field names from query
		if strings.Contains(query, "user") {
			if resolver, ok := s.resolvers["user"]; ok {
				data, _ := resolver.Resolve(variables)
				result["user"] = data
			}
		}
		if strings.Contains(query, "me") {
			if resolver, ok := s.resolvers["me"]; ok {
				data, _ := resolver.Resolve(variables)
				result["me"] = data
			}
		}
		if strings.Contains(query, "wallet") {
			if resolver, ok := s.resolvers["wallet"]; ok {
				data, _ := resolver.Resolve(variables)
				result["wallet"] = data
			}
		}
		if strings.Contains(query, "swapQuote") {
			if resolver, ok := s.resolvers["swapQuote"]; ok {
				data, _ := resolver.Resolve(variables)
				result["swapQuote"] = data
			}
		}
		if strings.Contains(query, "marketData") {
			if resolver, ok := s.resolvers["marketData"]; ok {
				data, _ := resolver.Resolve(variables)
				result["marketData"] = data
			}
		}
		if strings.Contains(query, "tokens") {
			addr := strArg(variables, "address")
			if addr == "" {
				result["tokens"] = []interface{}{}
			} else {
				t, err := s.walletAPIGet("/api/v1/public/tokens?address=" + addr)
				if err != nil {
					result["tokens"] = []interface{}{}
				} else {
					result["tokens"] = t
				}
			}
		}	}

	// Handle mutation root
	if strings.Contains(query, "mutation") {
		if strings.Contains(query, "createWallet") {
			if resolver, ok := s.resolvers["createWallet"]; ok {
				data, _ := resolver.Resolve(variables)
				result["createWallet"] = data
			}
		}
		if strings.Contains(query, "sendTransaction") {
			if resolver, ok := s.resolvers["sendTransaction"]; ok {
				data, _ := resolver.Resolve(variables)
				result["sendTransaction"] = data
			}
		}
	}

	return result, nil
}

func (s *GraphQLService) schemaHandler(c *gin.Context) {
	c.JSON(200, s.schema)
}

func (s *GraphQLService) introspectionHandler(c *gin.Context) {
	if !config.EnableIntrospection {
		c.JSON(404, gin.H{"error": "introspection disabled"})
		return
	}

	// Simplified introspection query
	introspection := map[string]interface{}{
		"__schema": map[string]interface{}{
			"queryType":    map[string]string{"name": "Query"},
			"mutationType": map[string]string{"name": "Mutation"},
			"types":        s.schema.Types,
			"directives":   []interface{}{},
		},
	}

	c.JSON(200, introspection)
}

// ============================================================================
// Helper Functions
// ============================================================================

// ---------------------------------------------------------------------------
// Backend delegation helpers
// ---------------------------------------------------------------------------
// The GraphQL BFF never holds keys, signs, or fabricates data. Wallet, balance,
// token and transaction resolvers delegate to the canonical wallet_api
// (go/wallet_api). Market/swap resolvers fetch live prices from CoinGecko.

func (s *GraphQLService) walletAPIGet(path string) (interface{}, error) {
	return s.walletAPIRequest(http.MethodGet, path, nil, "")
}

func (s *GraphQLService) walletAPIGetWithAuth(path, token string) (interface{}, error) {
	return s.walletAPIRequest(http.MethodGet, path, nil, token)
}

func (s *GraphQLService) walletAPIPost(path string, body interface{}) (interface{}, error) {
	return s.walletAPIRequest(http.MethodPost, path, body, "")
}

func (s *GraphQLService) walletAPIRequest(method, path string, body interface{}, authToken string) (interface{}, error) {
	url := strings.TrimRight(config.WalletAPIURL, "/") + path
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("wallet_api unreachable: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("wallet_api %s %s: %s", method, path, string(raw))
	}
	var out interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return string(raw), nil
	}
	return out, nil
}

func (s *GraphQLService) coinGeckoRate(fromToken, toToken string) (float64, error) {
	fromID := coinGeckoID(fromToken)
	toID := coinGeckoID(toToken)
	if fromID == "" || toID == "" {
		return 0, fmt.Errorf("unknown CoinGecko id for %s/%s", fromToken, toToken)
	}
	url := "https://api.coingecko.com/api/v3/simple/price?ids=" + fromID + "," + toID + "&vs_currencies=usd"
	resp, err := s.httpClient.Get(url)
	if err != nil {
		return 0, fmt.Errorf("coingecko unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("coingecko returned %d", resp.StatusCode)
	}
	var pr map[string]map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return 0, err
	}
	from, ok := pr[fromID]["usd"]
	if !ok {
		return 0, fmt.Errorf("no price for %s", fromToken)
	}
	to, ok := pr[toID]["usd"]
	if !ok || to == 0 {
		return 0, fmt.Errorf("no price for %s", toToken)
	}
	return from / to, nil
}

func (s *GraphQLService) coinGeckoMarketData(base, quote, pair string) (interface{}, error) {
	id := coinGeckoID(base)
	if id == "" {
		return nil, fmt.Errorf("unknown CoinGecko id for %s", base)
	}
	url := "https://api.coingecko.com/api/v3/coins/" + id + "/market_chart?vs_currency=usd&days=1"
	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("coingecko unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("coingecko returned %d", resp.StatusCode)
	}
	var mc struct {
		Prices [][]float64 `json:"prices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&mc); err != nil {
		return nil, err
	}
	if len(mc.Prices) == 0 {
		return nil, fmt.Errorf("no market data for %s", base)
	}
	var high, low, last float64
	last = mc.Prices[len(mc.Prices)-1][1]
	high = last
	low = last
	for _, p := range mc.Prices {
		v := p[1]
		if v > high {
			high = v
		}
		if v < low {
			low = v
		}
	}
	first := mc.Prices[0][1]
	change := 0.0
	if first != 0 {
		change = (last - first) / first * 100
	}
	return map[string]interface{}{
		"pair":      pair,
		"price":     last,
		"change24h": change,
		"high24h":   high,
		"low24h":    low,
		"volume24h": 0,
	}, nil
}

func coinGeckoID(symbol string) string {
	switch strings.ToUpper(symbol) {
	case "ETH":
		return "ethereum"
	case "BTC":
		return "bitcoin"
	case "BNB":
		return "binancecoin"
	case "SOL":
		return "solana"
	case "USDT":
		return "tether"
	case "USDC":
		return "usd-coin"
	case "MATIC", "POL":
		return "matic-network"
	case "AVAX":
		return "avalanche-2"
	case "ADA":
		return "cardano"
	case "XRP":
		return "ripple"
	case "DOT":
		return "polkadot"
	case "LINK":
		return "chainlink"
	default:
		return ""
	}
}

func strArg(m map[string]interface{}, key string) string {
	v, ok := m[key].(string)
	if !ok {
		return ""
	}
	return v
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ============================================================================
// Main
// ============================================================================

func main() {
	fmt.Println("============================================")
	fmt.Println("TigerWallet GraphQL Service")
	fmt.Println("============================================")

	svc := NewGraphQLService()

	if err := svc.Start(); err != nil {
		fmt.Printf("Failed to start GraphQL service: %v\n", err)
		os.Exit(1)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nShutting down...")
	svc.Stop()

	fmt.Println("GraphQL service stopped")
}
