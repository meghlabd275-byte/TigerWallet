package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
)

// ============================================================================
// BLOCKCHAIN SERVICE - Production Ready
// ============================================================================

type BlockchainService struct {
	config          *BlockchainConfig
	clients         map[uint64]*ChainClient
	wallet         *Wallet
	mu             sync.RWMutex
	tokenAbi       abi.ABI
}

type BlockchainConfig struct {
	SupportedChains []ChainConfig
	GasStrategy    string
	MaxGasPrice    *big.Int
	ConfirmationBlocks int
}

type ChainConfig struct {
	ChainID       uint64
	Name          string
	Symbol        string
	RPCURL        string
	ExplorerURL   string
	ChainType     string
	NativeToken   string
	Decimals      int
	IsTestnet     bool
}

type ChainClient struct {
	Client     *ethclient.Client
	Config     ChainConfig
	Contracts  map[string]*Contract
	mu         sync.RWMutex
}

type Wallet struct {
	Address     common.Address
	PrivateKey string
	PublicKey  string
}

type Contract struct {
	Address  common.Address
	Abi      abi.ABI
}

type Transaction struct {
	Hash       string
	From       string
	To         string
	Value      string
	Data       string
	GasLimit   uint64
	GasPrice   *big.Int
	GasUsed    uint64
	Nonce      uint64
	ChainID    uint64
	Status     string
	BlockNumber uint64
	Timestamp   time.Time
}

type TokenBalance struct {
	Address   string
	Symbol    string
	Name     string
	Decimals int
	Balance  *big.Int
	ValueUSD *big.Int
}

type TokenInfo struct {
	Address   common.Address
	Symbol    string
	Name      string
	Decimals  int
	TotalSupply *big.Int
}

type NFT struct {
	TokenID     *big.Int
	Contract    common.Address
	Owner       common.Address
	URI         string
	Metadata    map[string]interface{}
}

type ChainStatus struct {
	ChainID         uint64
	Name           string
	BlockNumber    uint64
	GasPrice       *big.Int
	Connected      bool
	LastSync       time.Time
	Latency        time.Duration
}

// ============================================================================
// CONSTRUCTOR
// ============================================================================

func NewBlockchainService(config *BlockchainConfig) (*BlockchainService, error) {
	service := &BlockchainService{
		config:    config,
		clients:   make(map[uint64]*ChainClient),
		mu:        sync.RWMutex{},
	}

	// Load ERC-20 ABI
	erc20Abi, err := abi.JSON(strings.NewReader(ERC20ABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ERC20 ABI: %v", err)
	}
	service.tokenAbi = erc20Abi

	// Initialize chain clients
	for _, chainConfig := range config.SupportedChains {
		client, err := ethclient.Dial(chainConfig.RPCURL)
		if err != nil {
			fmt.Printf("Warning: Failed to connect to %s: %v\n", chainConfig.Name, err)
			continue
		}

		service.clients[chainConfig.ChainID] = &ChainClient{
			Client:    client,
			Config:    chainConfig,
			Contracts: make(map[string]*Contract),
		}
	}

	return service, nil
}

// ============================================================================
// WALLET OPERATIONS
// ============================================================================

func (s *BlockchainService) SetWallet(privateKey string) error {
	key, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		return fmt.Errorf("invalid private key: %v", err)
	}

	address := crypto.PubkeyToAddress(key.PublicKey)
	publicKey := hex.EncodeToString(crypto.CompressPubkey(&key.PublicKey))

	s.mu.Lock()
	defer s.mu.Unlock()

	s.wallet = &Wallet{
		Address:     address,
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}

	return nil
}

func (s *BlockchainService) GetAddress() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.wallet == nil {
		return ""
	}
	return s.wallet.Address.Hex()
}

func (s *BlockchainService) GetBalance(ctx context.Context, chainID uint64) (*big.Int, error) {
	client, err := s.getClient(chainID)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	address := s.wallet.Address
	s.mu.RUnlock()

	return client.Client.BalanceAt(ctx, address, nil)
}

func (s *BlockchainService) GetTokenBalance(ctx context.Context, chainID uint64, tokenAddress string) (*big.Int, error) {
	client, err := s.getClient(chainID)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	address := s.wallet.Address
	s.mu.RUnlock()

	tokenAddr := common.HexToAddress(tokenAddress)

	// Call balanceOf
	method := "balanceOf"
	data, err := s.tokenAbi.Pack(method, address)
	if err != nil {
		return nil, fmt.Errorf("failed to pack data: %v", err)
	}

	result, err := client.Client.CallContract(ctx, ethereum.CallMsg{
		To:   &tokenAddr,
		Data: data,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call contract: %v", err)
	}

	balance := new(big.Int)
	balance.SetBytes(result)
	return balance, nil
}

func (s *BlockchainService) GetAllTokenBalances(ctx context.Context, chainID uint64, tokens []string) ([]TokenBalance, error) {
	var balances []TokenBalance

	// Get native balance
	nativeBalance, err := s.GetBalance(ctx, chainID)
	if err == nil {
		client, _ := s.getClient(chainID)
		chainName := "Unknown"
		if client != nil {
			chainName = client.Config.Name
		}

		balances = append(balances, TokenBalance{
			Address:   "0x0000000000000000000000000000000000000000",
			Symbol:    client.Config.Symbol,
			Name:      chainName,
			Decimals:  client.Config.Decimals,
			Balance:   nativeBalance,
		})
	}

	// Get token balances
	for _, token := range tokens {
		balance, err := s.GetTokenBalance(ctx, chainID, token)
		if err != nil {
			continue
		}

		if balance.Sign() > 0 {
			info, _ := s.GetTokenInfo(ctx, chainID, token)
			balances = append(balances, TokenBalance{
				Address:   token,
				Symbol:    info.Symbol,
				Name:      info.Name,
				Decimals:  info.Decimals,
				Balance:   balance,
			})
		}
	}

	return balances, nil
}

// ============================================================================
// TRANSACTION OPERATIONS
// ============================================================================

func (s *BlockchainService) SendTransaction(ctx context.Context, chainID uint64, to string, value *big.Int, data []byte) (string, error) {
	client, err := s.getClient(chainID)
	if err != nil {
		return "", err
	}

	s.mu.RLock()
	key, err := crypto.HexToECDSA(s.wallet.PrivateKey)
	from := s.wallet.Address
	s.mu.RUnlock()

	if err != nil {
		return "", fmt.Errorf("invalid private key: %v", err)
	}

	// Get nonce
	nonce, err := client.Client.NonceAt(ctx, from, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get nonce: %v", err)
	}

	// Get gas price
	gasPrice, err := client.Client.SuggestGasPrice(ctx)
	if err != nil {
		gasPrice = big.NewInt(1e9) // 1 gwei fallback
	}

	// Estimate gas
	msg := ethereum.CallMsg{
		From:  from,
		To:    common.HexToAddress(to),
		Value: value,
		Data:  data,
	}

	gasLimit, err := client.Client.EstimateGas(ctx, msg)
	if err != nil {
		gasLimit = 21000 // Default for simple transfers
	}
	gasLimit = gasLimit * 120 / 100 // Add 20% buffer

	// Create transaction
	tx := types.NewTransaction(nonce, common.HexToAddress(to), value, gasLimit, gasPrice, data)

	// Get chain ID
	chainIDBig, err := client.Client.ChainID(ctx)
	if err != nil {
		chainIDBig = big.NewInt(int64(chainID))
	}

	// Sign transaction
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainIDBig), key)
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %v", err)
	}

	// Send transaction
	err = client.Client.SendTransaction(ctx, signedTx)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %v", err)
	}

	return signedTx.Hash().Hex(), nil
}

func (s *BlockchainService) SendToken(ctx context.Context, chainID uint64, tokenAddress string, to string, amount *big.Int) (string, error) {
	client, err := s.getClient(chainID)
	if err != nil {
		return "", err
	}

	s.mu.RLock()
	key := s.wallet.PrivateKey
	from := s.wallet.Address
	s.mu.RUnlock()

	// Pack transfer data
	data, err := s.tokenAbi.Pack("transfer", common.HexToAddress(to), amount)
	if err != nil {
		return "", fmt.Errorf("failed to pack data: %v", err)
	}

	// Get token decimals for value
	info, err := s.GetTokenInfo(ctx, chainID, tokenAddress)
	if err != nil {
		return "", err
	}
	value := big.NewInt(0)

	// Get nonce
	nonce, err := client.Client.NonceAt(ctx, from, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get nonce: %v", err)
	}

	// Get gas price
	gasPrice, err := client.Client.SuggestGasPrice(ctx)
	if err != nil {
		gasPrice = big.NewInt(1e9)
	}

	// Estimate gas
	msg := ethereum.CallMsg{
		From: from,
		To:   common.HexToAddress(tokenAddress),
		Value: value,
		Data:  data,
	}

	gasLimit, err := client.Client.EstimateGas(ctx, msg)
	if err != nil {
		gasLimit = 65000 // ERC-20 transfer estimate
	}
	gasLimit = gasLimit * 120 / 100

	// Create transaction
	tx := types.NewTransaction(nonce, common.HexToAddress(tokenAddress), value, gasLimit, gasPrice, data)

	// Sign and send
	chainIDBig, _ := client.Client.ChainID(ctx)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainIDBig), crypto.HexToECDSA(key))
	if err != nil {
		return "", err
	}

	err = client.Client.SendTransaction(ctx, signedTx)
	if err != nil {
		return "", err
	}

	return signedTx.Hash().Hex(), nil
}

func (s *BlockchainService) GetTransactionReceipt(ctx context.Context, chainID uint64, txHash string) (*Transaction, error) {
	client, err := s.getClient(chainID)
	if err != nil {
		return nil, err
	}

	receipt, err := client.Client.TransactionReceipt(ctx, common.HexToHash(txHash))
	if err != nil {
		return nil, fmt.Errorf("failed to get receipt: %v", err)
	}

	tx, _, err := client.Client.TransactionByHash(ctx, common.HexToHash(txHash))
	if err != nil {
		return nil, err
	}

	block, err := client.Client.BlockByNumber(ctx, big.NewInt(int64(receipt.BlockNumber)))
	if err != nil {
		return nil, err
	}

	status := "failed"
	if receipt.Status == 1 {
		status = "success"
	}

	return &Transaction{
		Hash:       txHash,
		From:       receipt.From.Hex(),
		To:         receipt.To.Hex(),
		Value:      tx.Value().String(),
		GasLimit:   tx.Gas(),
		GasUsed:    receipt.GasUsed,
		GasPrice:   tx.GasPrice(),
		Nonce:      tx.Nonce(),
		ChainID:    chainID,
		Status:     status,
		BlockNumber: receipt.BlockNumber,
		Timestamp:   block.Time() * 1000,
	}, nil
}

// ============================================================================
// TOKEN OPERATIONS
// ============================================================================

func (s *BlockchainService) GetTokenInfo(ctx context.Context, chainID uint64, tokenAddress string) (*TokenInfo, error) {
	client, err := s.getClient(chainID)
	if err != nil {
		return nil, err
	}

	addr := common.HexToAddress(tokenAddress)

	// Get symbol
	symbolData, _ := s.tokenAbi.Pack("symbol")
	symbolResult, _ := client.Client.CallContract(ctx, ethereum.CallMsg{
		To:   &addr,
		Data: symbolData,
	}, nil)
	symbol, _ := s.tokenAbi.Unpack("symbol", symbolResult)

	// Get name
	nameData, _ := s.tokenAbi.Pack("name")
	nameResult, _ := client.Client.CallContract(ctx, ethereum.CallMsg{
		To:   &addr,
		Data: nameData,
	}, nil)
	name, _ := s.tokenAbi.Unpack("name", nameResult)

	// Get decimals
	decimalsData, _ := s.tokenAbi.Pack("decimals")
	decimalsResult, _ := client.Client.CallContract(ctx, ethereum.CallMsg{
		To:   &addr,
		Data: decimalsData,
	}, nil)
	decimals, _ := s.tokenAbi.Unpack("decimals", decimalsResult)

	// Get total supply
	supplyData, _ := s.tokenAbi.Pack("totalSupply")
	supplyResult, _ := client.Client.CallContract(ctx, ethereum.CallMsg{
		To:   &addr,
		Data: supplyData,
	}, nil)
	supply, _ := s.tokenAbi.Unpack("totalSupply", supplyResult)

	return &TokenInfo{
		Address:      addr,
		Symbol:      symbol.(string),
		Name:        name.(string),
		Decimals:    int(decimals.(uint8)),
		TotalSupply: supply.(*big.Int),
	}, nil
}

// ============================================================================
// NFT OPERATIONS
// ============================================================================

func (s *BlockchainService) GetNFT(ctx context.Context, chainID uint64, contractAddress string, tokenID *big.Int) (*NFT, error) {
	client, err := s.getClient(chainID)
	if err != nil {
		return nil, err
	}

	// Load ERC-721 ABI
	erc721Abi, _ := abi.JSON(strings.NewReader(ERC721ABI))

	addr := common.HexToAddress(contractAddress)

	// Get owner
	ownerData, _ := erc721Abi.Pack("ownerOf", tokenID)
	ownerResult, err := client.Client.CallContract(ctx, ethereum.CallMsg{
		To:   &addr,
		Data: ownerData,
	}, nil)
	if err != nil {
		return nil, err
	}
	var owner common.Address
	erc721Abi.UnpackIntoInterface(&owner, "ownerOf", ownerResult)

	// Get token URI
	uriData, _ := erc721Abi.Pack("tokenURI", tokenID)
	uriResult, _ := client.Client.CallContract(ctx, ethereum.CallMsg{
		To:   &addr,
		Data: uriData,
	}, nil)
	var uri string
	erc721Abi.UnpackIntoInterface(&uri, "tokenURI", uriResult)

	return &NFT{
		TokenID:    tokenID,
		Contract:   addr,
		Owner:      owner,
		URI:        uri,
		Metadata:   make(map[string]interface{}),
	}, nil
}

// ============================================================================
// CHAIN STATUS
// ============================================================================

func (s *BlockchainService) GetChainStatus(ctx context.Context, chainID uint64) (*ChainStatus, error) {
	client, err := s.getClient(chainID)
	if err != nil {
		return nil, err
	}

	start := time.Now()

	blockNumber, err := client.Client.BlockNumber(ctx)
	latency := time.Since(start)

	gasPrice, _ := client.Client.SuggestGasPrice(ctx)

	return &ChainStatus{
		ChainID:      chainID,
		Name:         client.Config.Name,
		BlockNumber:  blockNumber,
		GasPrice:     gasPrice,
		Connected:    true,
		LastSync:     time.Now(),
		Latency:      latency,
	}, nil
}

func (s *BlockchainService) GetAllChainStatus(ctx context.Context) []ChainStatus {
	var statuses []ChainStatus

	for chainID := range s.clients {
		status, err := s.GetChainStatus(ctx, chainID)
		if err == nil {
			statuses = append(statuses, *status)
		}
	}

	return statuses
}

// ============================================================================
// CONTRACT DEPLOYMENT
// ============================================================================

func (s *BlockchainService) DeployContract(ctx context.Context, chainID uint64, bytecode string, abiJson string, params ...interface{}) (string, error) {
	client, err := s.getClient(chainID)
	if err != nil {
		return "", err
	}

	s.mu.RLock()
	key, _ := crypto.HexToECDSA(s.wallet.PrivateKey)
	from := s.wallet.Address
	s.mu.RUnlock()

	parsedAbi, err := abi.JSON(strings.NewReader(abiJson))
	if err != nil {
		return "", err
	}

	packedArgs, err := parsedAbi.Pack("", params...)
	if err != nil {
		return "", err
	}

	fullBytecode := bytecode + hex.EncodeToString(packedArgs)
	bytecodeBytes, _ := hex.DecodeString(fullBytecode)

	nonce, _ := client.Client.NonceAt(ctx, from, nil)
	gasPrice, _ := client.Client.SuggestGasPrice(ctx)

	tx := types.NewContractCreation(nonce, bytecodeBytes, 5000000, gasPrice, big.NewInt(0))

	chainIDBig, _ := client.Client.ChainID(ctx)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainIDBig), key)
	if err != nil {
		return "", err
	}

	err = client.Client.SendTransaction(ctx, signedTx)
	if err != nil {
		return "", err
	}

	return signedTx.Hash().Hex(), nil
}

// ============================================================================
// HELPER METHODS
// ============================================================================

func (s *BlockchainService) getClient(chainID uint64) (*ChainClient, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	client, ok := s.clients[chainID]
	if !ok {
		return nil, fmt.Errorf("chain %d not supported", chainID)
	}

	return client, nil
}

// ============================================================================
// ERC-20 ABI
// ============================================================================

const ERC20ABI = `[
  {"constant":true,"inputs":[],"name":"name","outputs":[{"name":"","type":"string"}],"type":"function"},
  {"constant":true,"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"type":"function"},
  {"constant":true,"inputs":[],"name":"decimals","outputs":[{"name":"","type":"uint8"}],"type":"function"},
  {"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"type":"function"},
  {"constant":true,"inputs":[{"name":"_owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"balance","type":"uint256"}],"type":"function"},
  {"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transfer","outputs":[{"name":"","type":"bool"}],"type":"function"},
  {"constant":false,"inputs":[{"name":"_spender","type":"address"},{"name":"_value","type":"uint256"}],"name":"approve","outputs":[{"name":"","type":"bool"}],"type":"function"},
  {"constant":true,"inputs":[{"name":"_owner","type":"address"},{"name":"_spender","type":"address"}],"name":"allowance","outputs":[{"name":"","type":"uint256"}],"type":"function"}
]`

const ERC721ABI = `[
  {"constant":true,"inputs":[{"name":"tokenId","type":"uint256"}],"name":"ownerOf","outputs":[{"name":"","type":"address"}],"type":"function"},
  {"constant":true,"inputs":[{"name":"tokenId","type":"uint256"}],"name":"tokenURI","outputs":[{"name":"","type":"string"}],"type":"function"},
  {"constant":true,"inputs":[],"name":"name","outputs":[{"name":"","type":"string"}],"type":"function"},
  {"constant":true,"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"type":"function"}
]`

// ============================================================================
// MAIN
// ============================================================================

func main() {
	config := &BlockchainConfig{
		SupportedChains: []ChainConfig{
			{ChainID: 1, Name: "Ethereum", Symbol: "ETH", RPCURL: "https://eth.llamarpc.com", ExplorerURL: "https://etherscan.io", ChainType: "evm", Decimals: 18},
			{ChainID: 56, Name: "BSC", Symbol: "BNB", RPCURL: "https://bsc-dataseed.binance.org", ExplorerURL: "https://bscscan.com", ChainType: "evm", Decimals: 18},
			{ChainID: 137, Name: "Polygon", Symbol: "MATIC", RPCURL: "https://polygon-rpc.com", ExplorerURL: "https://polygonscan.com", ChainType: "evm", Decimals: 18},
			{ChainID: 42161, Name: "Arbitrum", Symbol: "ETH", RPCURL: "https://arb1.arbitrum.io/rpc", ExplorerURL: "https://arbiscan.io", ChainType: "evm", Decimals: 18},
			{ChainID: 10, Name: "Optimism", Symbol: "ETH", RPCURL: "https://mainnet.optimism.io", ExplorerURL: "https://optimistic.etherscan.io", ChainType: "evm", Decimals: 18},
		},
		GasStrategy:    "adaptive",
		MaxGasPrice:    big.NewInt(100e9),
		ConfirmationBlocks: 12,
	}

	service, err := NewBlockchainService(config)
	if err != nil {
		fmt.Printf("Failed to initialize blockchain service: %v\n", err)
		return
	}

	fmt.Println("Blockchain service started successfully")

	// Test chain status
	ctx := context.Background()
	statuses := service.GetAllChainStatus(ctx)
	for _, status := range statuses {
		fmt.Printf("Chain: %s - Block: %d - Gas: %s - Latency: %v\n", 
			status.Name, status.BlockNumber, status.GasPrice.String(), status.Latency)
	}

	select {}
}
