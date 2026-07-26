/**
 * TigerWallet CLI Tools
 * Command-line interface for developers
 * 
 * Features:
 * - Wallet creation and management
 * - Transaction building and signing
 * - Smart contract deployment
 * - Key management
 * - Network interaction
 * - Debug tools
 */

package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/urfave/cli/v2"
)

// ============================================================================
// Configuration
// ============================================================================

var (
	version   = "1.0.0"
	buildTime = time.Now().Format("2006-01-02 15:04:05")
)

// ============================================================================
// Commands
// ============================================================================

// Wallet commands
var walletCommands = []*cli.Command{
	{
		Name:  "create",
		Usage: "Create a new wallet",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "password",
				Usage: "Wallet password",
			},
			&cli.StringFlag{
				Name:  "format",
				Usage: "Output format (json, raw)",
				Value: "json",
			},
		},
		Action: createWallet,
	},
	{
		Name:  "import",
		Usage: "Import an existing wallet",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "mnemonic",
				Usage:    "24-word seed phrase",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "password",
				Usage: "Wallet password",
			},
		},
		Action: importWallet,
	},
	{
		Name:  "export",
		Usage: "Export wallet details",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "address",
				Usage:    "Wallet address",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "private-key",
				Usage: "Include private key (requires --password)",
			},
		},
		Action: exportWallet,
	},
	{
		Name:  "balance",
		Usage: "Check wallet balance",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "address",
				Usage:    "Wallet address",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "rpc",
				Usage: "RPC URL",
				Value: "https://eth.llamarpc.com",
			},
		},
		Action: checkBalance,
	},
	{
		Name:  "list",
		Usage: "List all wallets",
		Action: listWallets,
	},
}

// Transaction commands
var txCommands = []*cli.Command{
	{
		Name:  "send",
		Usage: "Send a transaction",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "from",
				Usage:    "Sender address",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "to",
				Usage:    "Recipient address",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "amount",
				Usage:    "Amount in ETH",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "rpc",
				Usage: "RPC URL",
				Value: "https://eth.llamarpc.com",
			},
			&cli.StringFlag{
				Name:  "private-key",
				Usage: "Sender private key",
			},
		},
		Action: sendTransaction,
	},
	{
		Name:  "sign",
		Usage: "Sign a transaction",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "to",
				Usage:    "Recipient address",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "amount",
				Usage:    "Amount in ETH",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "private-key",
				Usage:    "Private key",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "nonce",
				Usage: "Nonce",
			},
			&cli.StringFlag{
				Name:  "gas-price",
				Usage: "Gas price in Gwei",
				Value: "20",
			},
			&cli.StringFlag{
				Name:  "gas-limit",
				Usage: "Gas limit",
				Value: "21000",
			},
		},
		Action: signTransaction,
	},
	{
		Name:  "broadcast",
		Usage: "Broadcast a signed transaction",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "signed-tx",
				Usage:    "Signed transaction hex",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "rpc",
				Usage: "RPC URL",
				Value: "https://eth.llamarpc.com",
			},
		},
		Action: broadcastTransaction,
	},
}

// Contract commands
var contractCommands = []*cli.Command{
	{
		Name:  "deploy",
		Usage: "Deploy a smart contract",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "abi",
				Usage:    "Contract ABI JSON file",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "bytecode",
				Usage:    "Contract bytecode hex",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "constructor",
				Usage: "Constructor arguments",
			},
			&cli.StringFlag{
				Name:  "private-key",
				Usage: "Deployer private key",
			},
			&cli.StringFlag{
				Name:  "rpc",
				Usage: "RPC URL",
				Value: "https://eth.llamarpc.com",
			},
		},
		Action: deployContract,
	},
	{
		Name:  "call",
		Usage: "Call a smart contract",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "address",
				Usage:    "Contract address",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "method",
				Usage:    "Contract method name",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "args",
				Usage: "Method arguments (comma-separated)",
			},
			&cli.StringFlag{
				Name:  "rpc",
				Usage: "RPC URL",
				Value: "https://eth.llamarpc.com",
			},
		},
		Action: callContract,
	},
}

// Network commands
var networkCommands = []*cli.Command{
	{
		Name:  "block",
		Usage: "Get block information",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "number",
				Usage:    "Block number (or 'latest')",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "rpc",
				Usage: "RPC URL",
				Value: "https://eth.llamarpc.com",
			},
		},
		Action: getBlock,
	},
	{
		Name:  "gas",
		Usage: "Get current gas prices",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "rpc",
				Usage: "RPC URL",
				Value: "https://eth.llamarpc.com",
			},
		},
		Action: getGasPrice,
	},
	{
		Name:  "chain-id",
		Usage: "Get chain ID",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "rpc",
				Usage: "RPC URL",
				Value: "https://eth.llamarpc.com",
			},
		},
		Action: getChainID,
	},
}

// ============================================================================
// Wallet Functions
// ============================================================================

func createWallet(c *cli.Context) error {
	// Generate private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate key: %w", err)
	}

	// Get address
	publicKey := &privateKey.PublicKey
	address := crypto.PubkeyToAddress(*publicKey)

	// Get private key hex
	privateKeyHex := hex.EncodeToString(crypto.FromECDSA(privateKey))

	// Create wallet info
	wallet := map[string]interface{}{
		"address":    address.Hex(),
		"privateKey": "0x" + privateKeyHex,
		"publicKey":  hex.EncodeToString(crypto.FromECDSAPub(publicKey)),
	}

	// Output
	format := c.String("format")
	if format == "json" {
		jsonData, _ := json.MarshalIndent(wallet, "", "  ")
		fmt.Println(string(jsonData))
	} else {
		fmt.Printf("Address:    %s\n", wallet["address"])
		fmt.Printf("Private:    %s\n", wallet["privateKey"])
	}

	return nil
}

func importWallet(c *cli.Context) error {
	mnemonic := c.String("mnemonic")
	
	// Validate mnemonic (in production, use proper BIP39)
	words := strings.Fields(mnemonic)
	if len(words) != 12 && len(words) != 24 {
		return fmt.Errorf("invalid mnemonic: must be 12 or 24 words")
	}

	// In production, derive key from mnemonic using BIP39/BIP44
	// For now, create a deterministic key from mnemonic hash
	hasher := sha256.New()
	hasher.Write([]byte(mnemonic))
	keyBytes := hasher.Sum(nil)
	
	privateKey, err := crypto.ToECDSA(keyBytes)
	if err != nil {
		return fmt.Errorf("failed to derive key: %w", err)
	}

	address := crypto.PubkeyToAddress(privateKey.PublicKey)
	privateKeyHex := hex.EncodeToString(crypto.FromECDSA(privateKey))

	fmt.Printf("Address:    %s\n", address.Hex())
	fmt.Printf("Private:    0x%s\n", privateKeyHex)

	return nil
}

func exportWallet(c *cli.Context) error {
	address := c.String("address")
	if !common.IsHexAddress(address) {
		return fmt.Errorf("invalid address")
	}

	// In production, read from encrypted wallet file
	wallet := map[string]interface{}{
		"address": address,
		"type":    "imported",
	}

	jsonData, _ := json.MarshalIndent(wallet, "", "  ")
	fmt.Println(string(jsonData))

	return nil
}

func checkBalance(c *cli.Context) error {
	address := c.String("address")
	rpcURL := c.String("rpc")

	if !common.IsHexAddress(address) {
		return fmt.Errorf("invalid address")
	}

	// Connect to network
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return fmt.Errorf("failed to connect to network: %w", err)
	}

	// Get balance
	addr := common.HexToAddress(address)
	balance, err := client.BalanceAt(context.Background(), addr, nil)
	if err != nil {
		return fmt.Errorf("failed to get balance: %w", err)
	}

	// Convert to ETH
	balanceEth := float64(balance.Int64()) / float64(params.Ether)

	fmt.Printf("Address: %s\n", address)
	fmt.Printf("Balance: %.6f ETH\n", balanceEth)

	return nil
}

func listWallets(c *cli.Context) error {
	// In production, list from wallet storage
	wallets := []map[string]string{
		{"address": "0x0000000000000000000000000000000000000000", "type": "demo"},
	}

	jsonData, _ := json.MarshalIndent(wallets, "", "  ")
	fmt.Println(string(jsonData))

	return nil
}

// ============================================================================
// Transaction Functions
// ============================================================================

func sendTransaction(c *cli.Context) error {
	from := c.String("from")
	to := c.String("to")
	amountStr := c.String("amount")
	rpcURL := c.String("rpc")
	privateKey := c.String("private-key")

	if !common.IsHexAddress(from) || !common.IsHexAddress(to) {
		return fmt.Errorf("invalid address")
	}

	// Parse amount
	var amount float64
	fmt.Sscanf(amountStr, "%f", &amount)
	amountWei := int64(amount * float64(params.Ether))

	// Parse private key
	keyBytes, err := hex.DecodeString(strings.TrimPrefix(privateKey, "0x"))
	if err != nil {
		return fmt.Errorf("invalid private key: %w", err)
	}

	key, err := crypto.ToECDSA(keyBytes)
	if err != nil {
		return fmt.Errorf("invalid private key: %w", err)
	}

	// Connect to network
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Get nonce
	nonce, err := client.NonceAt(context.Background(), common.HexToAddress(from), nil)
	if err != nil {
		return fmt.Errorf("failed to get nonce: %w", err)
	}

	// Get gas price
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get gas price: %w", err)
	}

	// Create transaction
	tx := types.NewTransaction(nonce, common.HexToAddress(to), big.NewInt(amountWei), 21000, gasPrice, nil)

	// Sign transaction
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(1)), key)
	if err != nil {
		return fmt.Errorf("failed to sign: %w", err)
	}

	// Broadcast
	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return fmt.Errorf("failed to broadcast: %w", err)
	}

	fmt.Printf("Transaction sent: %s\n", signedTx.Hash().Hex())

	return nil
}

func signTransaction(c *cli.Context) error {
	to := c.String("to")
	amountStr := c.String("amount")
	privateKeyStr := c.String("private-key")
	nonceStr := c.String("nonce")
	gasPriceStr := c.String("gas-price")
	gasLimitStr := c.String("gas-limit")

	if !common.IsHexAddress(to) {
		return fmt.Errorf("invalid address")
	}

	// Parse private key
	keyBytes, err := hex.DecodeString(strings.TrimPrefix(privateKeyStr, "0x"))
	if err != nil {
		return fmt.Errorf("invalid private key: %w", err)
	}

	key, err := crypto.ToECDSA(keyBytes)
	if err != nil {
		return fmt.Errorf("invalid private key: %w", err)
	}

	// Parse amount
	var amount float64
	fmt.Sscanf(amountStr, "%f", &amount)
	amountWei := big.NewInt(int64(amount * float64(params.Ether)))

	// Parse nonce
	nonce := uint64(0)
	if nonceStr != "" {
		fmt.Sscanf(nonceStr, "%d", &nonce)
	}

	// Parse gas price
	var gasPrice int64
	fmt.Sscanf(gasPriceStr, "%d", &gasPrice)
	gasPriceWei := big.NewInt(gasPrice * 1000000000) // Gwei to Wei

	// Parse gas limit
	gasLimit := uint64(21000)
	if gasLimitStr != "" {
		fmt.Sscanf(gasLimitStr, "%d", &gasLimit)
	}

	// Create transaction
	tx := types.NewTransaction(nonce, common.HexToAddress(to), amountWei, gasLimit, gasPriceWei, nil)

	// Sign transaction
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(1)), key)
	if err != nil {
		return fmt.Errorf("failed to sign: %w", err)
	}

	// Encode
	encoded, err := signedTx.MarshalBinary()
	if err != nil {
		return fmt.Errorf("failed to encode: %w", err)
	}

	fmt.Printf("Signed transaction: 0x%s\n", hex.EncodeToString(encoded))

	return nil
}

func broadcastTransaction(c *cli.Context) error {
	signedTxHex := c.String("signed-tx")
	rpcURL := c.String("rpc")

	// Decode transaction
	txData, err := hex.DecodeString(strings.TrimPrefix(signedTxHex, "0x"))
	if err != nil {
		return fmt.Errorf("invalid transaction: %w", err)
	}

	// Parse transaction
	tx := new(types.Transaction)
	if err := tx.UnmarshalBinary(txData); err != nil {
		return fmt.Errorf("failed to parse transaction: %w", err)
	}

	// Connect to network
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Broadcast
	err = client.SendTransaction(context.Background(), tx)
	if err != nil {
		return fmt.Errorf("failed to broadcast: %w", err)
	}

	fmt.Printf("Transaction broadcast: %s\n", tx.Hash().Hex())

	return nil
}

// ============================================================================
// Contract Functions
// ============================================================================

func deployContract(c *cli.Context) error {
	abiPath := c.String("abi")
	bytecodeStr := c.String("bytecode")
	rpcURL := c.String("rpc")
	privateKeyStr := c.String("private-key")

	// Read ABI
	abiData, err := os.ReadFile(abiPath)
	if err != nil {
		return fmt.Errorf("failed to read ABI: %w", err)
	}

	parsedABI, err := abi.JSON(strings.NewReader(string(abiData)))
	if err != nil {
		return fmt.Errorf("failed to parse ABI: %w", err)
	}

	// Parse bytecode
	bytecode, err := hex.DecodeString(strings.TrimPrefix(bytecodeStr, "0x"))
	if err != nil {
		return fmt.Errorf("invalid bytecode: %w", err)
	}

	// Parse private key
	keyBytes, err := hex.DecodeString(strings.TrimPrefix(privateKeyStr, "0x"))
	if err != nil {
		return fmt.Errorf("invalid private key: %w", err)
	}

	key, err := crypto.ToECDSA(keyBytes)
	if err != nil {
		return fmt.Errorf("invalid private key: %w", err)
	}

	// Connect to network
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Get nonce and gas price
	from := crypto.PubkeyToAddress(key.PublicKey)
	nonce, err := client.NonceAt(context.Background(), from, nil)
	if err != nil {
		return fmt.Errorf("failed to get nonce: %w", err)
	}

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get gas price: %w", err)
	}

	// Create deploy transaction
	input := append(bytecode, parsedABI.Methods[""].ID...)
	tx := types.NewContractCreation(nonce, big.NewInt(0), 1500000, gasPrice, input)

	// Sign and broadcast
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(1)), key)
	if err != nil {
		return fmt.Errorf("failed to sign: %w", err)
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return fmt.Errorf("failed to deploy: %w", err)
	}

	fmt.Printf("Contract deployed: %s\n", signedTx.Hash().Hex())

	return nil
}

func callContract(c *cli.Context) error {
	contractAddr := c.String("address")
	methodName := c.String("method")
	argsStr := c.String("args")
	rpcURL := c.String("rpc")

	if !common.IsHexAddress(contractAddr) {
		return fmt.Errorf("invalid contract address")
	}

	// In production, read contract ABI from file or blockchain
	// For now, use a basic example
	
	// Connect to network
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Call contract (simplified - in production use proper ABI encoding)
	addr := common.HexToAddress(contractAddr)
	result, err := client.CallContract(context.Background(), addr, nil)
	if err != nil {
		return fmt.Errorf("call failed: %w", err)
	}

	fmt.Printf("Result: %s\n", hex.EncodeToString(result))

	return nil
}

// ============================================================================
// Network Functions
// ============================================================================

func getBlock(c *cli.Context) error {
	blockNumStr := c.String("number")
	rpcURL := c.String("rpc")

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	var blockNum int64
	if blockNumStr == "latest" {
		header, err := client.HeaderByNumber(context.Background(), nil)
		if err != nil {
			return fmt.Errorf("failed to get latest block: %w", err)
		}
		blockNum = int64(header.Number.Int64())
	} else {
		fmt.Sscanf(blockNumStr, "%d", &blockNum)
	}

	block, err := client.BlockByNumber(context.Background(), big.NewInt(blockNum))
	if err != nil {
		return fmt.Errorf("failed to get block: %w", err)
	}

	fmt.Printf("Block #%d\n", block.Number())
	fmt.Printf("Hash: %s\n", block.Hash().Hex())
	fmt.Printf("Transactions: %d\n", len(block.Transactions()))
	fmt.Printf("Timestamp: %d\n", block.Time())

	return nil
}

func getGasPrice(c *cli.Context) error {
	rpcURL := c.String("rpc")

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get gas price: %w", err)
	}

	gasPriceGwei := float64(gasPrice.Int64()) / 1000000000

	fmt.Printf("Slow:   %.1f Gwei\n", gasPriceGwei*0.8)
	fmt.Printf("Normal: %.1f Gwei\n", gasPriceGwei)
	fmt.Printf("Fast:   %.1f Gwei\n", gasPriceGwei*1.2)

	return nil
}

func getChainID(c *cli.Context) error {
	rpcURL := c.String("rpc")

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get chain ID: %w", err)
	}

	fmt.Printf("Chain ID: %d\n", chainID.Int64())

	return nil
}

// ============================================================================
// Main
// ============================================================================

func main() {
	// Setup logging
	log.SetFlags(0)
	log.SetOutput(os.Stderr)

	// Create app
	app := &cli.App{
		Name:    "tigerwallet",
		Usage:   "TigerWallet CLI - Developer tools for wallet management",
		Version: version,
		Commands: []*cli.Command{
			{
				Name:     "wallet",
				Usage:    "Wallet management commands",
				Commands: walletCommands,
			},
			{
				Name:     "tx",
				Usage:    "Transaction commands",
				Commands: txCommands,
			},
			{
				Name:     "contract",
				Usage:    "Smart contract commands",
				Commands: contractCommands,
			},
			{
				Name:     "network",
				Usage:    "Network commands",
				Commands: networkCommands,
			},
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "verbose",
				Usage: "Enable verbose logging",
			},
		},
		Before: func(c *cli.Context) error {
			if c.Bool("verbose") {
				log.SetFlags(log.LstdFlags | log.Lshortfile)
			}
			return nil
		},
	}

	// Run app
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
