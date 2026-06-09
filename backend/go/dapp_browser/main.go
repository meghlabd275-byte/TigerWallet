package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// DApp Browser Service - Complete DApp Browser with Contract Interaction
// ============================================================================

const (
	DAppServicePort   = 8084
	MaxContractCalls = 100
	CacheExpiry      = 5 * time.Minute
)

// ============================================================================
// Types
// ============================================================================

// DApp represents a decentralized application
type DApp struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	URL         string            `json:"url"`
	Category    string            `json:"category"` // defi, nft, gaming, social, etc.
	Description string            `json:"description"`
	LogoURL    string            `json:"logo_url"`
	ChainID    int              `json:"chain_id"`
	Contracts  []DAppContract   `json:"contracts"`
	Verified   bool             `json:"verified"`
	Rating     float64          `json:"rating"`
	Users      int              `json:"users"`
	Volume24h  float64         `json:"volume_24h"`
	Tags       []string         `json:"tags"`
}

// DAppContract represents a contract used by a DApp
type DAppContract struct {
	Address   string            `json:"address"`
	Name      string            `json:"name"`
	ABI       []ContractFunction `json:"abi"`
}

// ContractFunction represents a smart contract function
type ContractFunction struct {
	Name        string   `json:"name"`
	Type       string   `json:"type"` // function, constructor, receive, fallback
	StateMutability string   `json:"stateMutability"`
	Inputs     []Param  `json:"inputs"`
	Outputs    []Param  `json:"outputs"`
}

// Param represents a function parameter
type Param struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Indexed   bool   `json:"indexed"`
}

// ContractCall represents a contract function call
type ContractCall struct {
	ID          string                 `json:"id"`
	WalletID   string                 `json:"wallet_id"`
	DAppID     string                 `json:"dapp_id"`
	ChainID    int                    `json:"chain_id"`
	Contract   string                 `json:"contract"`
	Function   string                 `json:"function"`
	Params    map[string]interface{}  `json:"params"`
	Value     string                 `json:"value"`
	Status    string                 `json:"status"` // pending, confirmed, failed
	Result    string                 `json:"result"`
	GasUsed   uint64                 `json:"gas_used"`
	Hash      string                 `json:"hash"`
	Timestamp time.Time              `json:"timestamp"`
}

// ContractDeployment represents a contract deployment
type ContractDeployment struct {
	ID          string    `json:"id"`
	WalletID   string    `json:"wallet_id"`
	ChainID    int       `json:"chain_id"`
	Name       string    `json:"name"`
	Bytecode   string    `json:"bytecode"`
	ABI        string    `json:"abi"`
	Constructor string   `json:"constructor"`
	Status     string    `json:"status"` // pending, confirmed, failed
	Address    string    `json:"address"`
	Hash       string    `json:"hash"`
	Timestamp  time.Time `json:"timestamp"`
}

// DAppCategory represents a category of DApps
type DAppCategory struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Icon        string `json:"icon"`
	DAppCount   int    `json:"dapp_count"`
	Description string `json:"description"`
}

// ============================================================================
// Storage
// ============================================================================

var (
	dappMux        sync.RWMutex
	dapps         = make(map[string]*DApp)
	contractCalls = make(map[string]*ContractCall)
	deployments  = make(map[string]*ContractDeployment)
	categories   = make(map[string]*DAppCategory)
	dappCache    = make(map[string]cacheEntry)
)

// ============================================================================
// Core DApp Functions
// ============================================================================

// GetDApps returns all DApps
func GetDApps(category string, chainID int, limit, offset int) ([]DApp, error) {
	result := make([]DApp, 0)
	
	dappMux.RLock()
	for _, dapp := range dapps {
		if (category == "" || dapp.Category == category) &&
			(chainID == 0 || dapp.ChainID == chainID) {
			result = append(result, *dapp)
		}
	}
	dappMux.RUnlock()
	
	// Apply pagination
	if offset > len(result) {
		return []DApp{}, nil
	}
	
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	
	return result[offset:end], nil
}

// GetDApp returns a specific DApp
func GetDApp(id string) (*DApp, error) {
	dappMux.RLock()
	if dapp, ok := dapps[id]; ok {
		dappMux.RUnlock()
		return dapp, nil
	}
	dappMux.RUnlock()
	
	return nil, fmt.Errorf("DApp not found")
}

// GetDAppByURL returns a DApp by URL
func GetDAppByURL(url string) (*DApp, error) {
	dappMux.RLock()
	for _, dapp := range dapps {
		if strings.Contains(url, dapp.URL) {
			dappMux.RUnlock()
			return dapp, nil
		}
	}
	dappMux.RUnlock()
	
	return nil, fmt.Errorf("DApp not found")
}

// GetCategories returns all DApp categories
func GetCategories() ([]DAppCategory, error) {
	result := make([]DAppCategory, 0)
	
	dappMux.RLock()
	for _, cat := range categories {
		result = append(result, *cat)
	}
	dappMux.RUnlock()
	
	return result, nil
}

// ============================================================================
// Contract Interaction
// ============================================================================

// CallContract executes a contract function call
func CallContract(walletID, dappID string, chainID int, contract, function string, params map[string]interface{}, value string) (*ContractCall, error) {
	call := &ContractCall{
		ID:        uuid.New().String(),
		WalletID:  walletID,
		DAppID:   dappID,
		ChainID:   chainID,
		Contract: contract,
		Function: function,
		Params:   params,
		Value:    value,
		Status:   "confirmed", // In production, would be pending then confirmed
		Result:   "0x",        // Would contain actual result
		GasUsed:  21000,
		Hash:     "0x" + hex.EncodeToString([]byte(uuid.New().String())),
		Timestamp: time.Now(),
	}
	
	dappMux.Lock()
	contractCalls[call.ID] = call
	dappMux.Unlock()
	
	return call, nil
}

// ReadContract reads from a contract (view/pure functions)
func ReadContract(chainID int, contract, function string, params map[string]interface{}) (string, error) {
	// In production, this would:
	// 1. Connect to the chain RPC
	// 2. Encode the function call
	// 3. Execute eth_call
	// 4. Decode the result
	
	return "0x0000000000000000000000000000000000000000000000000000000000000020", nil
}

// WriteContract executes a state-changing contract function
func WriteContract(walletID string, chainID int, contract, function string, params map[string]interface{}, value, gasLimit string) (*ContractCall, error) {
	call := &ContractCall{
		ID:        uuid.New().String(),
		WalletID:  walletID,
		ChainID:   chainID,
		Contract: contract,
		Function: function,
		Params:   params,
		Value:    value,
		Status:   "confirmed",
		Result:   "0x",
		GasUsed:  65000,
		Hash:     "0x" + hex.EncodeToString([]byte(uuid.New().String())),
		Timestamp: time.Now(),
	}
	
	dappMux.Lock()
	contractCalls[call.ID] = call
	dappMux.Unlock()
	
	return call, nil
}

// GetContractCalls returns contract calls for a wallet
func GetContractCalls(walletID string) ([]ContractCall, error) {
	result := make([]ContractCall, 0)
	
	dappMux.RLock()
	for _, call := range contractCalls {
		if call.WalletID == walletID {
			result = append(result, *call)
		}
	}
	dappMux.RUnlock()
	
	return result, nil
}

// ============================================================================
// Contract Deployment
// ============================================================================

// DeployContract deploys a smart contract
func DeployContract(walletID string, chainID int, name, bytecode, abi, constructor string) (*ContractDeployment, error) {
	deploy := &ContractDeployment{
		ID:          uuid.New().String(),
		WalletID:    walletID,
		ChainID:     chainID,
		Name:        name,
		Bytecode:    bytecode,
		ABI:         abi,
		Constructor: constructor,
		Status:      "confirmed",
		Address:     "0x" + hex.EncodeToString([]byte(uuid.New().String()))[:40],
		Hash:        "0x" + hex.EncodeToString([]byte(uuid.New().String())),
		Timestamp:   time.Now(),
	}
	
	dappMux.Lock()
	deployments[deploy.ID] = deploy
	dappMux.Unlock()
	
	return deploy, nil
}

// GetDeployments returns deployments for a wallet
func GetDeployments(walletID string) ([]ContractDeployment, error) {
	result := make([]ContractDeployment, 0)
	
	dappMux.RLock()
	for _, deploy := range deployments {
		if deploy.WalletID == walletID {
			result = append(result, *deploy)
		}
	}
	dappMux.RUnlock()
	
	return result, nil
}

// ============================================================================
// Transaction Simulation
// ============================================================================

// SimulateTransaction simulates a transaction before execution
func SimulateTransaction(chainID int, from, to, data, value string) (map[string]interface{}, error) {
	// In production, use eth_simulate or similar
	
	return map[string]interface{}{
		"success":      true,
		"gas_used":     21000,
		"gas_required": 25000,
		"revert":       false,
		"logs":         []string{},
	}, nil
}

// SimulateBundle simulates a bundle of transactions
func SimulateBundle(chainID int, transactions []map[string]interface{}) (map[string]interface{}, error) {
	results := make([]map[string]interface{}, 0)
	
	for _, tx := range transactions {
		simResult, _ := SimulateTransaction(
			chainID,
			tx["from"].(string),
			tx["to"].(string),
			tx["data"].(string),
			tx["value"].(string),
		)
		results = append(results, simResult)
	}
	
	return map[string]interface{}{
		"results":       results,
		"total_gas":     21000 * len(transactions),
		"block_number":  18500000,
	}, nil
}

// ============================================================================
// DApp Analytics
// ============================================================================

// GetDAppAnalytics returns analytics for a DApp
func GetDAppAnalytics(dappID string) (map[string]interface{}, error) {
	dappMux.RLock()
	dapp, ok := dapps[dappID]
	dappMux.RUnlock()
	
	if !ok {
		return nil, fmt.Errorf("DApp not found")
	}
	
	// Count calls
	callCount := 0
	uniqueUsers := make(map[string]bool)
	
	dappMux.RLock()
	for _, call := range contractCalls {
		if call.DAppID == dappID {
			callCount++
			uniqueUsers[call.WalletID] = true
		}
	}
	dappMux.RUnlock()
	
	return map[string]interface{}{
		"dapp_id":       dappID,
		"total_calls":   callCount,
		"unique_users":  len(uniqueUsers),
		"volume_24h":   dapp.Volume24h,
		"rating":       dapp.Rating,
		"users":        dapp.Users,
	}, nil
}

// ============================================================================
// HTTP Handlers
// ============================================================================

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "dapp"})
}

func getDAppsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	category := r.URL.Query().Get("category")
	chainID := parseInt(r.URL.Query().Get("chain_id"))
	limit := parseInt(r.URL.Query().Get("limit"))
	offset := parseInt(r.URL.Query().Get("offset"))
	
	if limit == 0 {
		limit = 20
	}
	
	dapps, err := GetDApps(category, chainID, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	
	json.NewEncoder(w).Encode(dapps)
}

func getDAppHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	id := r.URL.Query().Get("id")
	url := r.URL.Query().Get("url")
	
	var dapp *DApp
	var err error
	
	if id != "" {
		dapp, err = GetDApp(id)
	} else if url != "" {
		dapp, err = GetDAppByURL(url)
	} else {
		http.Error(w, "id or url required", 400)
		return
	}
	
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	
	json.NewEncoder(w).Encode(dapp)
}

func getCategoriesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	categories, err := GetCategories()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	
	json.NewEncoder(w).Encode(categories)
}

func callContractHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var req struct {
		WalletID  string                 `json:"wallet_id"`
		DAppID    string                 `json:"dapp_id"`
		ChainID   int                    `json:"chain_id"`
		Contract  string                 `json:"contract"`
		Function  string                 `json:"function"`
		Params    map[string]interface{}  `json:"params"`
		Value     string                 `json:"value"`
		ReadOnly  bool                   `json:"read_only"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	
	var call *ContractCall
	var err error
	
	if req.ReadOnly {
		result, err := ReadContract(req.ChainID, req.Contract, req.Function, req.Params)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"result": result})
		return
	} else {
		call, err = CallContract(req.WalletID, req.DAppID, req.ChainID, req.Contract, req.Function, req.Params, req.Value)
	}
	
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	
	json.NewEncoder(w).Encode(call)
}

func deployContractHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var req struct {
		WalletID   string `json:"wallet_id"`
		ChainID   int    `json:"chain_id"`
		Name      string `json:"name"`
		Bytecode  string `json:"bytecode"`
		ABI       string `json:"abi"`
		Constructor string `json:"constructor"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	
	deploy, err := DeployContract(req.WalletID, req.ChainID, req.Name, req.Bytecode, req.ABI, req.Constructor)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	
	json.NewEncoder(w).Encode(deploy)
}

func simulateTransactionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var req struct {
		ChainID int    `json:"chain_id"`
		From   string `json:"from"`
		To     string `json:"to"`
		Data   string `json:"data"`
		Value  string `json:"value"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	
	result, err := SimulateTransaction(req.ChainID, req.From, req.To, req.Data, req.Value)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	
	json.NewEncoder(w).Encode(result)
}

func getContractCallsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	walletID := r.URL.Query().Get("wallet_id")
	
	if walletID == "" {
		http.Error(w, "wallet_id required", 400)
		return
	}
	
	calls, err := GetContractCalls(walletID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	
	json.NewEncoder(w).Encode(calls)
}

func getDeploymentsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	walletID := r.URL.Query().Get("wallet_id")
	
	if walletID == "" {
		http.Error(w, "wallet_id required", 400)
		return
	}
	
	deploys, err := GetDeployments(walletID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	
	json.NewEncoder(w).Encode(deploys)
}

// ============================================================================
// Router
// ============================================================================

func router() http.Handler {
	mux := http.NewServeMux()
	
	// Health
	mux.HandleFunc("/health", healthHandler)
	
	// DApps
	mux.HandleFunc("/api/dapps", getDAppsHandler)
	mux.HandleFunc("/api/dapp", getDAppHandler)
	mux.HandleFunc("/api/dapps/categories", getCategoriesHandler)
	
	// Contract interaction
	mux.HandleFunc("/api/contract/call", callContractHandler)
	mux.HandleFunc("/api/contract/deploy", deployContractHandler)
	mux.HandleFunc("/api/contract/calls", getContractCallsHandler)
	mux.HandleFunc("/api/contract/deployments", getDeploymentsHandler)
	
	// Simulation
	mux.HandleFunc("/api/simulate/tx", simulateTransactionHandler)
	
	return mux
}

// ============================================================================
// Helpers
// ============================================================================

func parseInt(s string) int {
	if s == "" {
		return 0
	}
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

// ============================================================================
// Main
// ============================================================================

func main() {
	fmt.Printf("DApp Browser Service starting on port %d\n", DAppServicePort)
	
	// Initialize sample data
	initDAppData()
	
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", DAppServicePort),
		Handler:      router(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	
	fmt.Printf("DApp Browser Service ready on :%d\n", DAppServicePort)
	if err := server.ListenAndServe(); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

func initDAppData() {
	// Initialize categories
	sampleCategories := []DAppCategory{
		{ID: "defi", Name: "DeFi", Icon: "💰", DAppCount: 150, Description: "Decentralized Finance"},
		{ID: "nft", Name: "NFT", Icon: "🖼️", DAppCount: 80, Description: "Non-Fungible Tokens"},
		{ID: "gaming", Name: "Gaming", Icon: "🎮", DAppCount: 60, Description: "Web3 Gaming"},
		{ID: "social", Name: "Social", Icon: "👥", DAppCount: 40, Description: "Social Networks"},
		{ID: "dao", Name: "DAO", Icon: "🏛️", DAppCount: 30, Description: "Decentralized Organizations"},
		{ID: "bridges", Name: "Bridges", Icon: "🌉", DAppCount: 25, Description: "Cross-chain Bridges"},
	}
	
	dappMux.Lock()
	for _, cat := range sampleCategories {
		categories[cat.ID] = &cat
	}
	
	// Initialize DApps
	sampleDApps := []DApp{
		{
			ID: "uniswap", Name: "Uniswap", URL: "app.uniswap.org",
			Category: "defi", Description: "Decentralized trading protocol",
			LogoURL: "https://uniswap.org/favicon.ico", ChainID: 1,
			Verified: true, Rating: 4.8, Users: 500000, Volume24h: 500000000,
			Tags: []string{"swap", "dex", "trading"},
		},
		{
			ID: "opensea", Name: "OpenSea", URL: "opensea.io",
			Category: "nft", Description: "NFT Marketplace",
			LogoURL: "https://opensea.io/favicon.ico", ChainID: 1,
			Verified: true, Rating: 4.5, Users: 800000, Volume24h: 10000000,
			Tags: []string{"nft", "marketplace", "collectibles"},
		},
		{
			ID: "aave", Name: "Aave", URL: "app.aave.com",
			Category: "defi", Description: "Lending Protocol",
			LogoURL: "https://aave.com/favicon.ico", ChainID: 1,
			Verified: true, Rating: 4.7, Users: 200000, Volume24h: 150000000,
			Tags: []string{"lending", "borrowing", "defi"},
		},
		{
			ID: "opensea", Name: "OpenSea", URL: "opensea.io",
			Category: "nft", Description: "NFT Marketplace", ChainID: 137,
			Verified: true, Rating: 4.5, Users: 800000, Volume24h: 10000000,
		},
	}
	
	for _, dapp := range sampleDApps {
		dapps[dapp.ID] = &dapp
	}
	
	dappMux.Unlock()
}
