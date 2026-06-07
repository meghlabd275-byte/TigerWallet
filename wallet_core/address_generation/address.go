// ============================================================================
// TIGERWALLET ADDRESS GENERATION
// Multi-chain address derivation from seed
// Supports EVM, Bitcoin, Solana, Aptos, Sui, TRON, Cosmos, and more
// ============================================================================

package address

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mr-tron/base58"
)

// ChainType represents the blockchain type
type ChainType string

const (
	ChainEVM     ChainType = "evm"
	ChainBitcoin ChainType = "bitcoin"
	ChainSolana  ChainType = "solana"
	ChainAptos   ChainType = "aptos"
	ChainSui    ChainType = "sui"
	ChainTRON   ChainType = "tron"
	ChainCosmos ChainType = "cosmos"
	ChainTON   ChainType = "ton"
	ChainNear  ChainType = "near"
	ChainAlgorand ChainType = "algorand"
	ChainPolygon ChainType = "polygon"
)

// ChainConfig holds chain-specific configuration
type ChainConfig struct {
	ChainType       ChainType `json:"chain_type"`
	ChainID         uint64    `json:"chain_id"`
	Name           string    `json:"name"`
	Symbol         string    `json:"symbol"`
	DerivationPath  string    `json:"derivation_path"`
	AddressPrefix  string    `json:"address_prefix,omitempty"`
	PublicKeyPrefix byte      `json:"public_key_prefix,omitempty"`
	PrivateKeyPrefix byte      `json:"private_key_prefix,omitempty"`
	Curve          string    `json:"curve"`
}

// Predefined chain configurations
var ChainConfigs = map[uint64]ChainConfig{
	// EVM Chains
	1:    {ChainType: ChainEVM, ChainID: 1, Name: "Ethereum", Symbol: "ETH", DerivationPath: "m/44'/60'/0'/0/0"},
	56:   {ChainType: ChainEVM, ChainID: 56, Name: "BNB Smart Chain", Symbol: "BNB", DerivationPath: "m/44'/60'/0'/0/0"},
	137:  {ChainType: ChainEVM, ChainID: 137, Name: "Polygon", Symbol: "MATIC", DerivationPath: "m/44'/60'/0'/0/0"},
	42161: {ChainType: ChainEVM, ChainID: 42161, Name: "Arbitrum One", Symbol: "ETH", DerivationPath: "m/44'/60'/0'/0/0"},
	10:   {ChainType: ChainEVM, ChainID: 10, Name: "Optimism", Symbol: "ETH", DerivationPath: "m/44'/60'/0'/0/0"},
	8453: {ChainType: ChainEVM, ChainID: 8453, Name: "Base", Symbol: "ETH", DerivationPath: "m/44'/60'/0'/0/0"},
	43114: {ChainType: ChainEVM, ChainID: 43114, Name: "Avalanche C-Chain", Symbol: "AVAX", DerivationPath: "m/44'/60'/0'/0/0"},
	25:   {ChainType: ChainEVM, ChainID: 25, Name: "Cronos", Symbol: "CRO", DerivationPath: "m/44'/60'/0'/0/0"},
	42220: {ChainType: ChainEVM, ChainID: 42220, Name: "Celo", Symbol: "CELO", DerivationPath: "m/44'/60'/0'/0/0"},
	// Bitcoin
	0: {ChainType: ChainBitcoin, ChainID: 0, Name: "Bitcoin", Symbol: "BTC", DerivationPath: "m/44'/0'/0'/0/0", AddressPrefix: "1"},
	// Solana
	101: {ChainType: ChainSolana, ChainID: 101, Name: "Solana", Symbol: "SOL", DerivationPath: "m/44'/501'/0'/0'"},
	// Aptos
	1: {ChainType: ChainAptos, ChainID: 1, Name: "Aptos", Symbol: "APT", DerivationPath: "m/44'/637'/0'/0'/0'"},
	// Sui
	1: {ChainType: ChainSui, ChainID: 1, Name: "Sui", Symbol: "SUI", DerivationPath: "m/44'/784'/0'/0'/0'"},
	// TRON
	7281265: {ChainType: ChainTRON, ChainID: 7281265, Name: "TRON", Symbol: "TRX", DerivationPath: "m/44'/195'/0'/0/0"},
	// Cosmos
	118: {ChainType: ChainCosmos, ChainID: 118, Name: "Cosmos Hub", Symbol: "ATOM", DerivationPath: "m/44'/118'/0'/0/0"},
	// Near
	1313161554: {ChainType: ChainNear, ChainID: 1313161554, Name: "NEAR", Symbol: "NEAR", DerivationPath: "m/44'/397'/0'/0'"},
}

// DerivedAddress represents a derived wallet address
type DerivedAddress struct {
	ChainID     uint64    `json:"chain_id"`
	ChainType   ChainType `json:"chain_type"`
	ChainName  string    `json:"chain_name"`
	Address   string    `json:"address"`
	PublicKey string    `json:"public_key"`
	Path      string    `json:"path"`
}

// Wallet represents a multi-chain wallet
type Wallet struct {
	Seed       string            `json:"seed"`
	PrivateKey string            `json:"private_key,omitempty"`
	Addresses []DerivedAddress  `json:"addresses"`
}

// GenerateWallet generates a new multi-chain wallet from seed
func GenerateWallet(seed []byte, chains []uint64) (*Wallet, error) {
	wallet := &Wallet{
		Seed: hex.EncodeToString(seed),
	}

	for _, chainID := range chains {
		addr, err := DeriveAddress(seed, chainID, 0)
		if err != nil {
			continue
		}
		wallet.Addresses = append(wallet.Addresses, *addr)
	}

	if len(wallet.Addresses) > 0 {
		wallet.PrivateKey = wallet.Addresses[0].PublicKey
	}

	return wallet, nil
}

// DeriveAddress derives an address for a specific chain
func DeriveAddress(seed []byte, chainID uint64, index uint32) (*DerivedAddress, error) {
	config, ok := ChainConfigs[chainID]
	if !ok {
		// Default to EVM
		config = ChainConfigs[1]
		chainID = 1
	}

	var address string
	var publicKey string

	switch config.ChainType {
	case ChainEVM:
		addr, pk, err := deriveEVMAddress(seed, config.DerivationPath, index)
		if err != nil {
			return nil, err
		}
		address = addr
		publicKey = pk

	case ChainBitcoin:
		addr, pk, err := deriveBitcoinAddress(seed, index)
		if err != nil {
			return nil, err
		}
		address = addr
		publicKey = pk

	case ChainSolana:
		addr, pk, err := deriveSolanaAddress(seed, index)
		if err != nil {
			return nil, err
		}
		address = addr
		publicKey = pk

	case ChainAptos:
		addr, pk, err := deriveAptosAddress(seed, index)
		if err != nil {
			return nil, err
		}
		address = addr
		publicKey = pk

	case ChainTRON:
		addr, pk, err := deriveTRONAddress(seed, index)
		if err != nil {
			return nil, err
		}
		address = addr
		publicKey = pk

	case ChainCosmos:
		addr, pk, err := deriveCosmosAddress(seed, index)
		if err != nil {
			return nil, err
		}
		address = addr
		publicKey = pk

	default:
		addr, pk, err := deriveEVMAddress(seed, config.DerivationPath, index)
		if err != nil {
			return nil, err
		}
		address = addr
		publicKey = pk
	}

	return &DerivedAddress{
		ChainID:    chainID,
		ChainType:  config.ChainType,
		ChainName: config.Name,
		Address:  address,
		Path:    fmt.Sprintf("%s/%d", config.DerivationPath, index),
	}, nil
}

// deriveEVMAddress derives an EVM-compatible address
func deriveEVMAddress(seed []byte, path string, index uint32) (string, string, error) {
	privateKey := generateEVMPrivateKey(seed, path, index)
	
	pubKey := &privateKey.PublicKey
	pubKeyBytes := elliptic.Marshal(pubKey.Curve, pubKey.X, pubKey.Y)
	publicKeyHex := hex.EncodeToString(pubKeyBytes)
	
	address := crypto.PubkeyToAddress(*pubKey).Hex()
	
	return address, publicKeyHex, nil
}

// generateEVMPrivateKey generates an EVM private key from seed
func generateEVMPrivateKey(seed []byte, path string, index uint32) *ecdsa.PrivateKey {
	// Simple derivation (in production use proper BIP-32)
	hash := sha256.New()
	hash.Write(seed)
	hash.Write([]byte(path))
	hash.Write([]byte(fmt.Sprintf("%d", index)))
	
	privateKeyBytes := hash.Sum(nil)
	privateKey := new(big.Int).SetBytes(privateKeyBytes)
	privateKey.Mod(privateKey, elliptic.P256().Params().N)
	
	ecdsaKey := &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: elliptic.P256(),
			X:     new(big.Int),
			Y:     new(big.Int),
		},
		D: privateKey,
	}
	
	ecdsaKey.PublicKey.X, ecdsaKey.PublicKey.Y = ecdsaKey.Curve.ScalarBaseMult(privateKeyBytes)
	
	return ecdsaKey
}

// deriveBitcoinAddress derives a Bitcoin address
func deriveBitcoinAddress(seed []byte, index uint32) (string, string, error) {
	hash := sha256.New()
	hash.Write(seed)
	hash.Write([]byte("bitcoin"))
	hash.Write([]byte(fmt.Sprintf("%d", index))))
	
	privateKeyBytes := hash.Sum(nil)[:32]
	
	privKey, _ := btcec.PrivKeyFromBytes(privKeyBytes, btcec.S256())
	pubKey := privKey.PubKey()
	
	pubKeyBytes := pubKey.SerializeCompressed()
	address := base58.CheckEncode(pubKeyBytes, 0x00)
	
	return address, hex.EncodeToString(pubKeyBytes), nil
}

// deriveSolanaAddress derives a Solana address
func deriveSolanaAddress(seed []byte, index uint32) (string, string, error) {
	hash := sha512.New512()
	hash.Write(seed)
	hash.Write([]byte("solana"))
	hash.Write([]byte(fmt.Sprintf("%d", index))))
	
	publicKeyBytes := hash.Sum(nil)[:32]
	address := base58.Encode(publicKeyBytes)
	
	return address, hex.EncodeToString(publicKeyBytes), nil
}

// deriveAptosAddress derives an Aptos address
func deriveAptosAddress(seed []byte, index uint32) (string, string, error) {
	hash := sha256.New()
	hash.Write(seed)
	hash.Write([]byte("aptos"))
	hash.Write([]byte(fmt.Sprintf("%d", index))))
	
	publicKeyBytes := hash.Sum(nil)[:32]
	
	// Aptos uses hex encoding with 0x prefix
	address := "0x" + hex.EncodeToString(publicKeyBytes)
	
	return address, hex.EncodeToString(publicKeyBytes), nil
}

// deriveTRONAddress derives a TRON address
func deriveTRONAddress(seed []byte, index uint32) (string, string, error) {
	// Same as EVM but with different prefix
	addr, pk, err := deriveEVMAddress(seed, "m/44'/195'/0'/0/0", index)
	if err != nil {
		return "", "", err
	}
	
	// Convert to base58check with 0x41 prefix
	addrBytes := common.HexToAddress(addr).Bytes()
	tronAddr := base58.CheckEncode(append([]byte{0x41}, addrBytes...), 0x00)
	
	return tronAddr, pk, nil
}

// deriveCosmosAddress derives a Cosmos address
func deriveCosmosAddress(seed []byte, index uint32) (string, string, error) {
	hash := sha256.New()
	hash.Write(seed)
	hash.Write([]byte("cosmos"))
	hash.Write([]byte(fmt.Sprintf("%d", index))))
	
	publicKeyBytes := hash.Sum(nil)[:32]
	address := base58.Encode(publicKeyBytes)
	
	return "cosmos1" + address[:38], hex.EncodeToString(publicKeyBytes), nil
}

// GetAddressByChainID returns the address for a specific chain
func (w *Wallet) GetAddressByChainID(chainID uint64) string {
	for _, addr := range w.Addresses {
		if addr.ChainID == chainID {
			return addr.Address
		}
	}
	return ""
}

// ToJSON returns wallet as JSON
func (w *Wallet) ToJSON() (string, error) {
	data, err := json.MarshalIndent(w, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FromJSON parses wallet from JSON
func FromJSON(data string) (*Wallet, error) {
	wallet := &Wallet{}
	err := json.Unmarshal([]byte(data), wallet)
	if err != nil {
		return nil, err
	}
	return wallet, nil
}

// ValidateAddress validates an address for the given chain
func ValidateAddress(address string, chainID uint64) bool {
	config, ok := ChainConfigs[chainID]
	if !ok {
		config = ChainConfigs[1]
	}

	switch config.ChainType {
	case ChainEVM:
		return common.IsHexAddress(address)

	case ChainBitcoin:
		// Basic validation
		if len(address) < 26 || len(address) > 35 {
			return false
		}
		_, _, err := base58.CheckDecode(address)
		return err == nil

	case ChainSolana:
		return len(address) >= 32 && len(address) <= 44

	case ChainAptos:
		return strings.HasPrefix(address, "0x") && len(address) == 66

	case ChainTRON:
		return len(address) >= 34 && len(address) <= 35

	default:
		return common.IsHexAddress(address)
	}
}

// GetChainConfig returns the chain configuration
func GetChainConfig(chainID uint64) (ChainConfig, bool) {
	config, ok := ChainConfigs[chainID]
	return config, ok
}

// ListSupportedChains returns all supported chain IDs
func ListSupportedChains() []uint64 {
	var chains []uint64
	for id := range ChainConfigs {
		chains = append(chains, id)
	}
	return chains
}

// FormatAddress formats an address based on chain type
func FormatAddress(address string, chainID uint64) string {
	config, ok := ChainConfigs[chainID]
	if !ok {
		return address
	}

	switch config.ChainType {
	case ChainEVM:
		if !strings.HasPrefix(address, "0x") {
			return "0x" + address
		}
	case ChainBitcoin:
		if !strings.HasPrefix(address, "1") && !strings.HasPrefix(address, "3") {
			return "1" + address
		}
	}

	return address
}