/**
 * TigerWallet Blockchain Explorer Service
 * 
 * Comprehensive blockchain explorer with RPC management,
 * chain registry, token integration, and node infrastructure.
 * Built with Go for high-load distributed operations.
 */

package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"time"
)

// ============================================================================
// Types
// ============================================================================

// Chain represents a blockchain
type Chain struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Symbol        string            `json:"symbol"`
	ChainID       uint64            `json:"chain_id"`
	ChainType     string            `json:"chain_type"` // evm, solana, bitcoin, cosmos, etc.
	RPCURLs       []string          `json:"rpc_urls"`
	ExplorerURL   string            `json:"explorer_url"`
	SymbolNative  string            `json:"symbol_native"`
	Decimals      int               `json:"decimals"`
	BlockTime    int               `json:"block_time"` // seconds
	GasToken      string            `json:"gas_token"`
	Status       string            `json:"status"` // active, maintenance, deprecated
	Features     []string          `json:"features"` // smart_contracts, nft, staking, etc.
	IconURL      string            `json:"icon_url"`
	CreatedAt    int64             `json:"created_at"`
}

// Token represents a blockchain token
type Token struct {
	ID            string  `json:"id"`
	Address       string  `json:"address"`
	Name          string  `json:"name"`
	Symbol        string  `json:"symbol"`
	Decimals      int     `json:"decimals"`
	TotalSupply   string  `json:"total_supply"`
	ChainID       uint64  `json:"chain_id"`
	Type          string  `json:"type"` // erc20, erc721, erc1155, native
	Price         float64 `json:"price"`
	MarketCap     float64 `json:"market_cap"`
	Volume24h     float64 `json:"volume_24h"`
	HoldersCount int     `json:"holders_count"`
	Transfers24h  int    `json:"transfers_24h"`
	IconURL       string  `json:"icon_url"`
	ContractVerified bool  `json:"contract_verified"`
	Status        string  `json:"status"`
}

// Block represents a block
type Block struct {
	Number           uint64    `json:"number"`
	Hash             string    `json:"hash"`
	ParentHash       string    `json:"parent_hash"`
	Timestamp        int64     `json:"timestamp"`
	Transactions     []string  `json:"transactions"`
	TransactionsCount int     `json:"transactions_count"`
	GasUsed          uint64    `json:"gas_used"`
	GasLimit         uint64    `json:"gas_limit"`
	Miner            string    `json:"miner"`
	Reward           string    `json:"reward"`
	Size             uint64    `json:"size"`
	Nonce            string    `json:"nonce"`
	ChainID          uint64    `json:"chain_id"`
}

// Transaction represents a transaction
type Transaction struct {
	Hash           string   `json:"hash"`
	BlockNumber    uint64   `json:"block_number"`
	BlockHash      string   `json:"block_hash"`
	From           string   `json:"from"`
	To             string   `json:"to"`
	Value          string   `json:"value"`
	GasPrice       string   `json:"gas_price"`
	GasUsed        uint64   `json:"gas_used"`
	GasLimit       uint64   `json:"gas_limit"`
	Nonce          uint64   `json:"nonce"`
	TransactionIndex uint64 `json:"transaction_index"`
	Input          string   `json:"input"`
	Status         string   `json:"status"` // success, failed, pending
	Timestamp      int64    `json:"timestamp"`
	ChainID        uint64   `json:"chain_id"`
	Type           string   `json:"type"` // legacy, eip1559
}

// TokenTransfer represents a token transfer
type TokenTransfer struct {
	ID            string  `json:"id"`
	TransactionHash string `json:"transaction_hash"`
	BlockNumber  uint64  `json:"block_number"`
	TokenAddress string  `json:"token_address"`
	From         string  `json:"from"`
	To           string  `json:"to"`
	Value        string  `json:"value"`
	TokenID      string  `json:"token_id,omitempty"`
	Type         string  `json:"type"` // transfer, mint, burn
	Timestamp    int64   `json:"timestamp"`
	ChainID      uint64  `json:"chain_id"`
}

// TokenHolder represents a token holder
type TokenHolder struct {
	Address    string  `json:"address"`
	Balance   string  `json:"balance"`
	Percent   float64 `json:"percent"`
	Rank      int     `json:"rank"`
}

// RPCEndpoint represents an RPC endpoint
type RPCEndpoint struct {
	ID          string  `json:"id"`
	ChainID     uint64  `json:"chain_id"`
	URL         string  `json:"url"`
	Name        string  `json:"name"`
	Provider    string  `json:"provider"`
	Status      string  `json:"status"` // active, inactive, error
	Latency     int     `json:"latency"` // ms
	SuccessRate float64 `json:"success_rate"`
	Requests24h  int64   `json:"requests_24h"`
	IsDefault   bool    `json:"is_default"`
}

// Contract represents a smart contract
type Contract struct {
	Address       string            `json:"address"`
	ChainID       uint64            `json:"chain_id"`
	Name          string            `json:"name"`
	ABI           string            `json:"abi"`
	Compiler      string            `json:"compiler"`
	Version       string            `json:"version"`
	Optimized     bool              `json:"optimized"`
	SourceCode   string            `json:"source_code"`
	Constructor   string            `json:"constructor"`
	Functions    []ContractFunction `json:"functions"`
	Events       []ContractEvent   `json:"events"`
	CreatedAt    int64             `json:"created_at"`
}

// ContractFunction represents a contract function
type ContractFunction struct {
	Name        string   `json:"name"`
	Signature   string   `json:"signature"`
	Inputs      []string `json:"inputs"`
	Outputs     []string `json:"outputs"`
	StateMutability string `json:"state_mutability"` // pure, view, nonpayable, payable
}

// ContractEvent represents a contract event
type ContractEvent struct {
	Name      string   `json:"name"`
	Signature string   `json:"signature"`
	Inputs    []EventInput `json:"inputs"`
}

// EventInput represents an event input
type EventInput struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Indexed bool   `json:"indexed"`
}

// ============================================================================
// Blockchain Explorer Service
// ============================================================================

// BlockchainExplorerService provides blockchain exploration functionality
type BlockchainExplorerService struct {
	mu          sync.RWMutex
	chains      map[string]*Chain
	tokens      map[string]*Token
	blocks      map[uint64]map[uint64]*Block // chain_id -> block_number -> block
	transactions map[string]*Transaction
	transfers   map[string][]*TokenTransfer
	rpcEndpoints map[string]*RPCEndpoint
	contracts   map[string]*Contract
	chainTokens  map[uint64]map[string]*Token // chain_id -> token_address -> token
}

// NewBlockchainExplorerService creates a new explorer service
func NewBlockchainExplorerService() *BlockchainExplorerService {
	return &BlockchainExplorerService{
		chains:       make(map[string]*Chain),
		tokens:        make(map[string]*Token),
		blocks:        make(map[uint64]map[uint64]*Block),
		transactions: make(map[string]*Transaction),
		transfers:    make(map[string][]*TokenTransfer),
		rpcEndpoints: make(map[string]*RPCEndpoint),
		contracts:    make(map[string]*Contract),
		chainTokens:  make(map[uint64]map[string]*Token),
	}
}

// ============================================================================
// Chain Management
// ============================================================================

// AddChain adds a new chain
func (s *BlockchainExplorerService) AddChain(ctx context.Context, chain *Chain) (*Chain, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	chain.ID = generateID()
	chain.Status = "active"
	chain.CreatedAt = time.Now().UnixMilli()

	s.chains[chain.ID] = chain

	// Initialize chain tokens map
	s.chainTokens[chain.ChainID] = make(map[string]*Token)

	return chain, nil
}

// GetChain retrieves a chain
func (s *BlockchainExplorerService) GetChain(ctx context.Context, id string) (*Chain, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	chain, exists := s.chains[id]
	if !exists {
		return nil, fmt.Errorf("chain not found")
	}
	return chain, nil
}

// GetChainByChainID retrieves a chain by chain ID
func (s *BlockchainExplorerService) GetChainByChainID(ctx context.Context, chainID uint64) (*Chain, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, chain := range s.chains {
		if chain.ChainID == chainID {
			return chain, nil
		}
	}
	return nil, fmt.Errorf("chain not found")
}

// ListChains lists all chains
func (s *BlockchainExplorerService) ListChains(ctx context.Context, status string) ([]*Chain, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Chain, 0)
	for _, chain := range s.chains {
		if status != "" && chain.Status != status {
			continue
		}
		result = append(result, chain)
	}
	return result, nil
}

// UpdateChain updates a chain
func (s *BlockchainExplorerService) UpdateChain(ctx context.Context, id string, updates map[string]interface{}) (*Chain, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	chain, exists := s.chains[id]
	if !exists {
		return nil, fmt.Errorf("chain not found")
	}

	if name, ok := updates["name"].(string); ok {
		chain.Name = name
	}
	if status, ok := updates["status"].(string); ok {
		chain.Status = status
	}
	if rpcURLs, ok := updates["rpc_urls"].([]string); ok {
		chain.RPCURLs = rpcURLs
	}

	return chain, nil
}

// DeleteChain deletes a chain
func (s *BlockchainExplorerService) DeleteChain(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.chains[id]; !exists {
		return fmt.Errorf("chain not found")
	}

	delete(s.chains, id)
	return nil
}

// ============================================================================
// Token Management
// ============================================================================

// AddToken adds a new token
func (s *BlockchainExplorerService) AddToken(ctx context.Context, token *Token) (*Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	token.ID = generateID()
	token.Status = "active"

	// Add to global tokens
	s.tokens[token.ID] = token

	// Add to chain-specific tokens
	if _, ok := s.chainTokens[token.ChainID]; !ok {
		s.chainTokens[token.ChainID] = make(map[string]*Token)
	}
	s.chainTokens[token.ChainID][token.Address] = token

	return token, nil
}

// GetToken retrieves a token
func (s *BlockchainExplorerService) GetToken(ctx context.Context, id string) (*Token, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	token, exists := s.tokens[id]
	if !exists {
		return nil, fmt.Errorf("token not found")
	}
	return token, nil
}

// GetTokenByAddress retrieves a token by address
func (s *BlockchainExplorerService) GetTokenByAddress(ctx context.Context, chainID uint64, address string) (*Token, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if tokens, ok := s.chainTokens[chainID]; ok {
		if token, exists := tokens[address]; exists {
			return token, nil
		}
	}
	return nil, fmt.Errorf("token not found")
}

// ListTokens lists tokens for a chain
func (s *BlockchainExplorerService) ListTokens(ctx context.Context, chainID uint64, tokenType string) ([]*Token, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Token, 0)
	if tokens, ok := s.chainTokens[chainID]; ok {
		for _, token := range tokens {
			if tokenType != "" && token.Type != tokenType {
				continue
			}
			result = append(result, token)
		}
	}
	return result, nil
}

// UpdateToken updates a token
func (s *BlockchainExplorerService) UpdateToken(ctx context.Context, id string, updates map[string]interface{}) (*Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	token, exists := s.tokens[id]
	if !exists {
		return nil, fmt.Errorf("token not found")
	}

	if price, ok := updates["price"].(float64); ok {
		token.Price = price
	}
	if marketCap, ok := updates["market_cap"].(float64); ok {
		token.MarketCap = marketCap
	}
	if volume24h, ok := updates["volume_24h"].(float64); ok {
		token.Volume24h = volume24h
	}

	return token, nil
}

// GetTokenHolders retrieves token holders
func (s *BlockchainExplorerService) GetTokenHolders(ctx context.Context, tokenID string, limit int) ([]*TokenHolder, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// In production, this would query a database
	// Simplified: return empty list
	return make([]*TokenHolder, 0), nil
}

// ============================================================================
// Block Functions
// ============================================================================

// GetBlock retrieves a block
func (s *BlockchainExplorerService) GetBlock(ctx context.Context, chainID uint64, blockNumber uint64) (*Block, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if blocks, ok := s.blocks[chainID]; ok {
		if block, exists := blocks[blockNumber]; exists {
			return block, nil
		}
	}
	return nil, fmt.Errorf("block not found")
}

// GetBlockByHash retrieves a block by hash
func (s *BlockchainExplorerService) GetBlockByHash(ctx context.Context, chainID uint64, hash string) (*Block, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if blocks, ok := s.blocks[chainID]; ok {
		for _, block := range blocks {
			if block.Hash == hash {
				return block, nil
			}
		}
	}
	return nil, fmt.Errorf("block not found")
}

// GetLatestBlock retrieves the latest block
func (s *BlockchainExplorerService) GetLatestBlock(ctx context.Context, chainID uint64) (*Block, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if blocks, ok := s.blocks[chainID]; ok {
		var latest *Block
		for _, block := range blocks {
			if latest == nil || block.Number > latest.Number {
				latest = block
			}
		}
		if latest != nil {
			return latest, nil
		}
	}
	return nil, fmt.Errorf("no blocks found")
}

// GetBlockTransactions retrieves transactions for a block
func (s *BlockchainExplorerService) GetBlockTransactions(ctx context.Context, chainID uint64, blockNumber uint64) ([]*Transaction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Transaction, 0)
	for _, tx := range s.transactions {
		if tx.ChainID == chainID && tx.BlockNumber == blockNumber {
			result = append(result, tx)
		}
	}
	return result, nil
}

// ============================================================================
// Transaction Functions
// ============================================================================

// GetTransaction retrieves a transaction
func (s *BlockchainExplorerService) GetTransaction(ctx context.Context, hash string) (*Transaction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tx, exists := s.transactions[hash]
	if !exists {
		return nil, fmt.Errorf("transaction not found")
	}
	return tx, nil
}

// GetTransactionsByAddress retrieves transactions for an address
func (s *BlockchainExplorerService) GetTransactionsByAddress(ctx context.Context, address string, limit, offset int) ([]*Transaction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Transaction, 0)
	count := 0
	skipped := 0
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

// GetTokenTransfers retrieves token transfers
func (s *BlockchainExplorerService) GetTokenTransfers(ctx context.Context, address string, limit int) ([]*TokenTransfer, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*TokenTransfer, 0)
	for _, transfers := range s.transfers {
		for _, transfer := range transfers {
			if transfer.From == address || transfer.To == address {
				result = append(result, transfer)
				if limit > 0 && len(result) >= limit {
					return result, nil
				}
			}
		}
	}
	return result, nil
}

// ============================================================================
// RPC Management
// ============================================================================

// AddRPCEndpoint adds an RPC endpoint
func (s *BlockchainExplorerService) AddRPCEndpoint(ctx context.Context, rpc *RPCEndpoint) (*RPCEndpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rpc.ID = generateID()
	rpc.Status = "active"

	s.rpcEndpoints[rpc.ID] = rpc
	return rpc, nil
}

// GetRPCEndpoint retrieves an RPC endpoint
func (s *BlockchainExplorerService) GetRPCEndpoint(ctx context.Context, chainID uint64) (*RPCEndpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, rpc := range s.rpcEndpoints {
		if rpc.ChainID == chainID && rpc.Status == "active" {
			return rpc, nil
		}
	}
	return nil, fmt.Errorf("no active RPC endpoint found for chain")
}

// ListRPCEndpoints lists RPC endpoints
func (s *BlockchainExplorerService) ListRPCEndpoints(ctx context.Context, chainID uint64) ([]*RPCEndpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*RPCEndpoint, 0)
	for _, rpc := range s.rpcEndpoints {
		if rpc.ChainID == chainID {
			result = append(result, rpc)
		}
	}
	return result, nil
}

// UpdateRPCStatus updates RPC endpoint status
func (s *BlockchainExplorerService) UpdateRPCStatus(ctx context.Context, rpcID string, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rpc, exists := s.rpcEndpoints[rpcID]
	if !exists {
		return fmt.Errorf("RPC endpoint not found")
	}

	rpc.Status = status
	return nil
}

// ============================================================================
// Contract Management
// ============================================================================

// AddContract adds a verified contract
func (s *BlockchainExplorerService) AddContract(ctx context.Context, contract *Contract) (*Contract, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	contract.Address = contract.Address
	contract.CreatedAt = time.Now().UnixMilli()

	key := fmt.Sprintf("%d_%s", contract.ChainID, contract.Address)
	s.contracts[key] = contract

	return contract, nil
}

// GetContract retrieves a contract
func (s *BlockchainExplorerService) GetContract(ctx context.Context, chainID uint64, address string) (*Contract, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := fmt.Sprintf("%d_%s", chainID, address)
	contract, exists := s.contracts[key]
	if !exists {
		return nil, fmt.Errorf("contract not found")
	}
	return contract, nil
}

// VerifyContract verifies a contract
func (s *BlockchainExplorerService) VerifyContract(ctx context.Context, chainID uint64, address, sourceCode, abi string) (*Contract, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("%d_%s", chainID, address)
	contract := &Contract{
		Address:     address,
		ChainID:     chainID,
		SourceCode:  sourceCode,
		ABI:         abi,
		Compiler:    "solc",
		Version:     "0.8.0",
		Optimized:   true,
		CreatedAt:   time.Now().UnixMilli(),
	}

	s.contracts[key] = contract

	// Update token as verified
	if tokens, ok := s.chainTokens[chainID]; ok {
		if token, exists := tokens[address]; exists {
			token.ContractVerified = true
		}
	}

	return contract, nil
}

// ============================================================================
// Statistics
// ============================================================================

// GetChainStats retrieves chain statistics
func (s *BlockchainExplorerService) GetChainStats(ctx context.Context, chainID uint64) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make(map[string]interface{})

	// Count tokens
	if tokens, ok := s.chainTokens[chainID]; ok {
		stats["total_tokens"] = len(tokens)
	}

	// Count transactions
	txCount := 0
	for _, tx := range s.transactions {
		if tx.ChainID == chainID {
			txCount++
		}
	}
	stats["total_transactions"] = txCount

	// Count blocks
	if blocks, ok := s.blocks[chainID]; ok {
		stats["total_blocks"] = len(blocks)
	}

	return stats, nil
}

// ============================================================================
// Helper Functions
// ============================================================================

func generateID() string {
	return fmt.Sprintf("id_%d_%s", time.Now().UnixNano(), randomString(12))
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

// FormatWei converts wei to ether
func FormatWei(wei *big.Int, decimals int) string {
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	result := new(big.Float).Quo(new(big.Float).SetInt(wei), new(big.Float).SetInt(divisor))
	return result.Text('f', 8)
}

// ParseWei parses ether to wei
func ParseWei(ether string, decimals int) (*big.Int, error) {
	f, _, err := big.ParseFloat(ether, 10, 0, 0)
	if err != nil {
		return nil, err
	}
	multiplier := new(big.Float).Exp(big.NewFloat(10), big.NewFloat(float64(decimals)), nil)
	result := new(big.Int)
	f.Mul(f, multiplier)
	return result.Int(nil)
}

// Serialize/Deserialize
func (c *Chain) Serialize() (string, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

func DeserializeChain(data string) (*Chain, error) {
	decoded, err := hex.DecodeString(data)
	if err != nil {
		return nil, err
	}
	var chain Chain
	if err := json.Unmarshal(decoded, &chain); err != nil {
		return nil, err
	}
	return &chain, nil
}
