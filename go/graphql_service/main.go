package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
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
	ListenAddr      string
	MaxQueryDepth   int
	MaxQueryCost    int
	Timeout         time.Duration
	EnableIntrospection bool
}

var config = Config{
	ListenAddr:        getEnv("GRAPHQL_LISTEN_ADDR", ":9003"),
	MaxQueryDepth:     10,
	MaxQueryCost:      1000,
	Timeout:           time.Second * 30,
	EnableIntrospection: true,
}

// ============================================================================
// GraphQL Types
// ============================================================================

type GraphQLRequest struct {
	Query         string                 `json:"query"`
	Variables    map[string]interface{} `json:"variables,omitempty"`
	OperationName string                `json:"operationName,omitempty"`
}

type GraphQLResponse struct {
	Data   interface{} `json:"data,omitempty"`
	Errors []GraphQLError `json:"errors,omitempty"`
}

type GraphQLError struct {
	Message   string   `json:"message"`
	Locations []Location `json:"locations,omitempty"`
	Path      []interface{} `json:"path,omitempty"`
}

type Location struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

type Schema struct {
	Types       []TypeDefinition    `json:"types"`
	Queries     []FieldDefinition   `json:"queries"`
	Mutations   []FieldDefinition   `json:"mutations"`
	Subscriptions []FieldDefinition `json:"subscriptions"`
}

type TypeDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Fields      []FieldDefinition `json:"fields,omitempty"`
	Kind        string          `json:"kind"` // scalar, object, interface, enum, union, input_object
	EnumValues  []EnumValue     `json:"enumValues,omitempty"`
	InputFields []InputValue    `json:"inputFields,omitempty"`
}

type FieldDefinition struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Type        string     `json:"type"`
	Args        []InputValue `json:"args,omitempty"`
	Resolve     func(args map[string]interface{}) (interface{}, error) `json:"-"`
}

type InputValue struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	DefaultValue string `json:"defaultValue,omitempty"`
}

type EnumValue struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ResolverContext struct {
	RequestID  string
	UserID     string
	Variables  map[string]interface{}
	StartTime  time.Time
}

// ============================================================================
// GraphQL Service
// ============================================================================

type GraphQLService struct {
	schema      Schema
	resolvers   map[string]FieldDefinition
	resolveMu   sync.RWMutex
	queries     int
	queriesMu   sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
}

type contextKey string

var requestIDKey contextKey = "request_id"

func NewGraphQLService() *GraphQLService {
	svc := &GraphQLService{
		resolvers: make(map[string]FieldDefinition),
		ctx:       context.Background(),
	}
	
	svc.initializeSchema()
	svc.initializeResolvers()
	
	return svc
}

func (s *GraphQLService) initializeSchema() {
	s.schema = Schema{
		Types: []TypeDefinition{
			{
				Name: "User",
				Description: "User account information",
				Kind: "object",
				Fields: []FieldDefinition{
					{Name: "id", Type: "ID!", Description: "User ID"},
					{Name: "email", Type: "String", Description: "User email"},
					{Name: "wallets", Type: "[Wallet!]!", Description: "User wallets"},
					{Name: "createdAt", Type: "String!", Description: "Account creation date"},
				},
			},
			{
				Name: "Wallet",
				Description: "Wallet information",
				Kind: "object",
				Fields: []FieldDefinition{
					{Name: "id", Type: "ID!", Description: "Wallet ID"},
					{Name: "address", Type: "String!", Description: "Wallet address"},
					{Name: "chain", Type: "String!", Description: "Blockchain chain"},
					{Name: "balance", Type: "Float!", Description: "Wallet balance"},
					{Name: "tokens", Type: "[Token!]!", Description: "Token balances"},
				},
			},
			{
				Name: "Token",
				Description: "Token information",
				Kind: "object",
				Fields: []FieldDefinition{
					{Name: "symbol", Type: "String!", Description: "Token symbol"},
					{Name: "name", Type: "String!", Description: "Token name"},
					{Name: "balance", Type: "Float!", Description: "Token balance"},
					{Name: "price", Type: "Float", Description: "Token price in USD"},
					{Name: "value", Type: "Float!", Description: "Token value in USD"},
				},
			},
			{
				Name: "Transaction",
				Description: "Transaction information",
				Kind: "object",
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
				Name: "TransactionStatus",
				Description: "Transaction status enum",
				Kind: "enum",
				EnumValues: []EnumValue{
					{Name: "PENDING", Description: "Transaction is pending"},
					{Name: "CONFIRMED", Description: "Transaction is confirmed"},
					{Name: "FAILED", Description: "Transaction failed"},
				},
			},
			{
				Name: "SwapQuote",
				Description: "Swap quote information",
				Kind: "object",
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
				Name: "MarketData",
				Description: "Market data for a trading pair",
				Kind: "object",
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
			return map[string]interface{}{
				"id":        args["id"],
				"email":     "user@example.com",
				"wallets":   []interface{}{},
				"createdAt": time.Now().Format(time.RFC3339),
			}, nil
		},
	}
	
	s.resolvers["me"] = FieldDefinition{
		Name: "me", Type: "User",
		Resolve: func(args map[string]interface{}) (interface{}, error) {
			return map[string]interface{}{
				"id":        "user_123",
				"email":     "me@example.com",
				"wallets":   []interface{}{},
				"createdAt": time.Now().Format(time.RFC3339),
			}, nil
		},
	}
	
	s.resolvers["wallet"] = FieldDefinition{
		Name: "wallet", Type: "Wallet",
		Resolve: func(args map[string]interface{}) (interface{}, error) {
			address := args["address"].(string)
			return map[string]interface{}{
				"id":      "wallet_" + address[:8],
				"address": address,
				"chain":   "ethereum",
				"balance": 1.5,
				"tokens":  []interface{}{},
			}, nil
		},
	}
	
	s.resolvers["swapQuote"] = FieldDefinition{
		Name: "swapQuote", Type: "SwapQuote",
		Resolve: func(args map[string]interface{}) (interface{}, error) {
			fromToken := args["fromToken"].(string)
			toToken := args["toToken"].(string)
			amount := args["amount"].(float64)
			
			// Simulate quote calculation
			rate := getMockRate(fromToken, toToken)
			toAmount := amount * rate
			
			return map[string]interface{}{
				"fromToken":    fromToken,
				"toToken":      toToken,
				"fromAmount":   amount,
				"toAmount":     toAmount,
				"priceImpact":  0.5,
				"route":        []string{fromToken, toToken},
				"gasEstimate":  0.01,
			}, nil
		},
	}
	
	s.resolvers["marketData"] = FieldDefinition{
		Name: "marketData", Type: "MarketData",
		Resolve: func(args map[string]interface{}) (interface{}, error) {
			pair := args["pair"].(string)
			parts := strings.Split(pair, "/")
			
			price := getMockPrice(parts[0])
			
			return map[string]interface{}{
				"pair":       pair,
				"price":      price,
				"volume24h":  rand.Float64() * 1000000,
				"change24h":  (rand.Float64() - 0.5) * 10,
				"high24h":    price * 1.05,
				"low24h":     price * 0.95,
			}, nil
		},
	}
	
	// Mutation resolvers
	s.resolvers["createWallet"] = FieldDefinition{
		Name: "createWallet", Type: "Wallet",
		Resolve: func(args map[string]interface{}) (interface{}, error) {
			chain := args["chain"].(string)
			walletID := fmt.Sprintf("wallet_%d", rand.Intn(1000000))
			
			return map[string]interface{}{
				"id":      walletID,
				"address": generateAddress(),
				"chain":   chain,
				"balance": 0.0,
				"tokens":  []interface{}{},
			}, nil
		},
	}
	
	s.resolvers["sendTransaction"] = FieldDefinition{
		Name: "sendTransaction", Type: "Transaction",
		Resolve: func(args map[string]interface{}) (interface{}, error) {
			to := args["to"].(string)
			amount := args["amount"].(float64)
			token := args["token"].(string)
			
			return map[string]interface{}{
				"hash":     "0x" + generateHash(),
				"from":     "0x742d35Cc6634C0532925a3b844Bc9e7595f1234",
				"to":       to,
				"amount":   amount,
				"token":    token,
				"status":   "PENDING",
				"timestamp": time.Now().Format(time.RFC3339),
			}, nil
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
			result["tokens"] = []interface{}{
				map[string]interface{}{"symbol": "ETH", "name": "Ethereum", "balance": 1.5, "price": 3500.0, "value": 5250.0},
				map[string]interface{}{"symbol": "USDT", "name": "Tether", "balance": 1000.0, "price": 1.0, "value": 1000.0},
			}
		}
	}
	
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
			"queryType":        map[string]string{"name": "Query"},
			"mutationType":     map[string]string{"name": "Mutation"},
			"types":            s.schema.Types,
			"directives":      []interface{}{},
		},
	}
	
	c.JSON(200, introspection)
}

// ============================================================================
// Helper Functions
// ============================================================================

func getMockRate(fromToken, toToken string) float64 {
	rates := map[string]map[string]float64{
		"ETH": {"USDT": 3500, "BTC": 0.053, "BNB": 5.8},
		"BTC": {"USDT": 65000, "ETH": 18.8, "BNB": 108},
		"USDT": {"ETH": 0.00028, "BTC": 0.000015, "BNB": 0.0017},
	}
	
	if rates[fromToken] != nil {
		if rate, ok := rates[fromToken][toToken]; ok {
			return rate
		}
	}
	
	return 1.0
}

func getMockPrice(token string) float64 {
	prices := map[string]float64{
		"ETH": 3500.0,
		"BTC": 65000.0,
		"BNB": 600.0,
		"SOL": 100.0,
		"USDT": 1.0,
		"USDC": 1.0,
	}
	
	if price, ok := prices[token]; ok {
		return price
	}
	
	return 100.0
}

func generateAddress() string {
	chars := "0123456789abcdef"
	addr := "0x"
	for i := 0; i < 40; i++ {
		addr += string(chars[rand.Intn(len(chars))])
	}
	return addr
}

func generateHash() string {
	chars := "0123456789abcdef"
	hash := ""
	for i := 0; i < 64; i++ {
		hash += string(chars[rand.Intn(len(chars))])
	}
	return hash
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
	rand.Seed(time.Now().UnixNano())
	
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
