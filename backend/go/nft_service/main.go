package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/uuid"
)

// ============================================================================
// NFT Service - Complete NFT Support for ERC-721, ERC-1155, and SPL
// ============================================================================

// Configuration
const (
	NFTServicePort    = 8081
	MaxBatchSize    = 50
	CacheExpiry    = 5 * time.Minute
	IPFSGateway    = "https://ipfs.io/ipfs/"
	ArweaveGateway = "https://arweave.net/"
)

// ERC-721 ABI (minimal required for token interaction)
var erc721ABI, _ = abi.JSON(strings.NewReader(`[
	{"inputs":[{"name":"owner","type":"address"},{"name":"approved","type":"address"}],"name":"approve","outputs":[],"stateMutability":"nonpayable","type":"function"},
	{"inputs":[{"name":"operator","type":"address"},{"name":"approved","type":"bool"}],"name":"setApprovalForAll","outputs":[],"stateMutability":"nonpayable","type":"function"},
	{"inputs":[{"name":"to","type":"address"},{"name":"tokenId","type":"uint256"}],"name":"mint","outputs":[],"stateMutability":"nonpayable","type":"function"},
	{"inputs":[{"name":"tokenId","type":"uint256"}],"name":"ownerOf","outputs":[{"name":"","type":"address"}],"stateMutability":"view","type":"function"},
	{"inputs":[{"name":"tokenId","type":"uint256"}],"name":"tokenURI","outputs":[{"name":"","type":"string"}],"stateMutability":"view","type":"function"},
	{"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
	{"inputs":[{"name":"owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
	{"inputs":[{"name":"index","type":"uint256"}],"name":"tokenByIndex","outputs":[{"name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
	{"inputs":[{"name":"owner","type":"address"},{"name":"start","type":"uint256"},{"name":"stop","type":"uint256"}],"name":"tokensOfOwner","outputs":[{"name":"","type":"uint256[]"}],"stateMutability":"view","type":"function"}
]`))

// ERC-1155 ABI
var erc1155ABI, _ = abi.JSON(strings.NewReader(`[
	{"inputs":[{"name":"account","type":"address"},{"name":"id","type":"uint256"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
	{"inputs":[{"name":"accounts","type":"address[]"},{"name":"ids","type":"uint256[]"}],"name":"balanceOfBatch","outputs":[{"name":"","type":"uint256[]"}],"stateMutability":"view","type":"function"},
	{"inputs":[{"name":"operator","type":"address"},{"name":"approved","type":"bool"}],"name":"setApprovalForAll","outputs":[],"stateMutability":"nonpayable","type":"function"},
	{"inputs":[{"name":"to","type":"address"},{"name":"id","type":"uint256"},{"name":"amount","type":"uint256"},{"name":"data","type":"bytes"}],"name":"mint","outputs":[],"stateMutability":"nonpayable","type":"function"},
	{"inputs":[{"name":"to","type":"address"},{"name":"ids","type":"uint256[]"},{"name":"amounts","type":"uint256[]"},{"name":"data","type":"bytes"}],"name":"mintBatch","outputs":[],"stateMutability":"nonpayable","type":"function"},
	{"inputs":[{"name":"from","type":"address"},{"name":"to","type":"address"},{"name":"id","type":"uint256"},{"name":"amount","type":"uint256"},{"name":"data","type":"bytes"}],"name":"safeTransferFrom","outputs":[],"stateMutability":"nonpayable","type":"function"}
]`))

// ============================================================================
// Types
// ============================================================================

// NFT represents a non-fungible token
type NFT struct {
	ID              string            `json:"id"`
	TokenID         string            `json:"token_id"`
	ContractAddress string            `json:"contract_address"`
	ChainID        int              `json:"chain_id"`
	Owner          string            `json:"owner"`
	TokenURI       string            `json:"token_uri"`
	Metadata       *NFTMetadata      `json:"metadata,omitempty"`
	Standard      string            `json:"standard"` // erc721, erc1155, spl
	Quantity      int              `json:"quantity"` // For ERC-1155
	BlockNumber   int64           `json:"block_number"`
	Timestamp    time.Time        `json:"timestamp"`
}

type NFTMetadata struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Image      string      `json:"image"`
	ImageURL   string      `json:"image_url"`
	ExternalURL string      `json:"external_url"`
	Attributes []NFTAttribute `json:"attributes"`
	Animation  string      `json:"animation_url"`
	BackgroundColor string   `json:"background_color"`
}

type NFTAttribute struct {
	TraitType   string `json:"trait_type"`
	Value     any    `json:"value"`
	DisplayType string `json:"display_type"`
}

type NFTCollection struct {
	Address         string            `json:"address"`
	ChainID        int               `json:"chain_id"`
	Name          string            `json:"name"`
	Symbol        string            `json:"symbol"`
	Standard      string            `json:"standard"`
	TotalSupply   string           `json:"total_supply"`
	Owner        string           `json:"owner"`
	ContractType string           `json:"contract_type"` // erc721, erc1155
	BaseURI      string           `json:"base_uri"`
	 royalties   string           `json:"royalty"`
	FloorPrice  float64          `json:"floor_price"`
	MarketCap   float64         `json:"market_cap"`
	Volume24h   float64         `json:"volume_24h"`
}

type NFTTransaction struct {
	ID           string    `json:"id"`
	NFTID       string    `json:"nft_id"`
	ChainID     int       `json:"chain_id"`
	Type        string    `json:"type"` // mint, transfer, sale, burn
	From        string    `json:"from"`
	To          string    `json:"to"`
	Price       string    `json:"price"`
	Token       string    `json:"token"`
	Hash        string    `json:"hash"`
	BlockNumber int64     `json:"block_number"`
	Timestamp   time.Time `json:"timestamp"`
}

type NFTListing struct {
	ID           string    `json:"id"`
	NFTID       string    `json:"nft_id"`
	Seller      string    `json:"seller"`
	Price       string    `json:"price"`
	PaymentToken string    `json:"payment_token"`
	Quantity   int       `json:"quantity"`
	Status     string    `json:"status"` // active, sold, cancelled
	ExpiresAt  time.Time `json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`
}

type NFTOwner struct {
	Address         string            `json:"address"`
	ChainID        int               `json:"chain_id"`
	NFTs          []NFT              `json:"nfts"`
	TotalValue    float64           `json:"total_value"`
	LastUpdated   time.Time         `json:"last_updated"`
}

// ============================================================================
// Storage
// ============================================================================

var (
	nftMux       sync.RWMutex
	nfts         = make(map[string]*NFT)         // tokenID -> NFT
	collections = make(map[string]*NFTCollection) // contractAddress -> Collection
	transactions = make(map[string]*NFTTransaction)
	listings    = make(map[string]*NFTListing)
	owners     = make(map[string]*NFTOwner) // address -> owner info
	nftCache   = make(map[string]cacheEntry)
)

type cacheEntry struct {
	data      interface{}
	expiresAt time.Time
}

// ============================================================================
// Core NFT Functions
// ============================================================================

// GetNFT retrieves an NFT by ID
func GetNFT(chainID int, contractAddress, tokenID string) (*NFT, error) {
	key := fmt.Sprintf("%d:%s:%s", chainID, strings.ToLower(contractAddress), tokenID)
	
	nftMux.RLock()
	if entry, ok := nftCache[key]; ok && time.Now().Before(entry.expiresAt) {
		nftMux.RUnlock()
		if nft, ok := entry.data.(*NFT); ok {
			return nft, nil
		}
	}
	nftMux.RUnlock()
	
	// Check in-memory storage
	if nft, ok := nfts[key]; ok {
		return nft, nil
	}
	
	// Fetch from blockchain (in production, this would query RPC)
	return fetchNFTFromChain(chainID, contractAddress, tokenID)
}

func fetchNFTFromChain(chainID int, contractAddress, tokenID string) (*NFT, error) {
	// In production, this would:
	// 1. Connect to the appropriate chain RPC
	// 2. Call tokenURI() on the contract
	// 3. Fetch metadata from IPFS/HTTP
	// 4. Parse and return the NFT
	
	// For now, return a placeholder
	return &NFT{
		ID:              uuid.New().String(),
		TokenID:         tokenID,
		ContractAddress: contractAddress,
		ChainID:        chainID,
		Owner:          "",
		TokenURI:       "",
		Standard:      "erc721",
		Quantity:      1,
		Timestamp:    time.Now(),
	}, nil
}

// GetNFTsOfOwner retrieves all NFTs owned by an address
func GetNFTsOfOwner(chainID int, ownerAddress string, limit, offset int) ([]NFT, error) {
	result := make([]NFT, 0)
	
	nftMux.RLock()
	for _, nft := range nfts {
		if strings.EqualFold(nft.Owner, ownerAddress) && nft.ChainID == chainID {
			result = append(result, *nft)
		}
	}
	nftMux.RUnlock()
	
	// Apply pagination
	if offset > len(result) {
		return []NFT{}, nil
	}
	
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	
	return result[offset:end], nil
}

// GetNFTCollection retrieves collection metadata
func GetNFTCollection(chainID int, contractAddress string) (*NFTCollection, error) {
	key := fmt.Sprintf("%d:%s", chainID, strings.ToLower(contractAddress))
	
	nftMux.RLock()
	if col, ok := collections[key]; ok {
		nftMux.RUnlock()
		return col, nil
	}
	nftMux.RUnlock()
	
	return fetchCollectionFromChain(chainID, contractAddress)
}

func fetchCollectionFromChain(chainID int, contractAddress string) (*NFTCollection, error) {
	// In production, query contract for name, symbol, totalSupply
	return &NFTCollection{
		Address:    contractAddress,
		ChainID:   chainID,
		Name:      "Unknown Collection",
		Symbol:    "UNKNOWN",
		Standard: "erc721",
	}, nil
}

// GetCollectionNFTs retrieves all NFTs in a collection
func GetCollectionNFTs(chainID int, contractAddress string, limit, offset int) ([]NFT, error) {
	result := make([]NFT, 0)
	
	nftMux.RLock()
	for _, nft := range nfts {
		if strings.EqualFold(nft.ContractAddress, contractAddress) && nft.ChainID == chainID {
			result = append(result, *nft)
		}
	}
	nftMux.RUnlock()
	
	if offset > len(result) {
		return []NFT{}, nil
	}
	
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	
	return result[offset:end], nil
}

// ============================================================================
// NFT Transactions
// ============================================================================

// TransferNFT transfers an NFT to a new owner
func TransferNFT(chainID int, contractAddress, tokenID, from, to string, quantity int) (*NFTTransaction, error) {
	// Validate addresses
	if !common.IsHexAddress(from) || !common.IsHexAddress(to) {
		return nil, fmt.Errorf("invalid address format")
	}
	
	tx := &NFTTransaction{
		ID:        uuid.New().String(),
		NFTID:     fmt.Sprintf("%s:%s", contractAddress, tokenID),
		ChainID:   chainID,
		Type:      "transfer",
		From:     from,
		To:       to,
		Timestamp: time.Now(),
	}
	
	nftMux.Lock()
	transactions[tx.ID] = tx
	
	// Update NFT owner
	key := fmt.Sprintf("%d:%s:%s", chainID, strings.ToLower(contractAddress), tokenID)
	if nft, ok := nfts[key]; ok {
		nft.Owner = to
		nft.Timestamp = time.Now()
	}
	nftMux.Unlock()
	
	return tx, nil
}

// BatchTransferNFT transfers multiple NFTs
func BatchTransferNFT(transfers []struct {
	ChainID        int
	ContractAddress string
	TokenID       string
	From         string
	To           string
	Quantity     int
}) ([]NFTTransaction, error) {
	result := make([]NFTTransaction, 0)
	
	for _, t := range transfers {
		tx, err := TransferNFT(t.ChainID, t.ContractAddress, t.TokenID, t.From, t.To, t.Quantity)
		if err != nil {
			return result, err
		}
		result = append(result, *tx)
	}
	
	return result, nil
}

// MintNFT mints a new NFT
func MintNFT(chainID int, contractAddress, to, tokenURI string, quantity int) (*NFTTransaction, error) {
	if !common.IsHexAddress(to) {
		return nil, fmt.Errorf("invalid address")
	}
	
	tokenID := uuid.New().String()
	
	nft := &NFT{
		ID:              uuid.New().String(),
		TokenID:         tokenID,
		ContractAddress: contractAddress,
		ChainID:        chainID,
		Owner:          to,
		TokenURI:       tokenURI,
		Standard:      "erc721",
		Quantity:      quantity,
		Timestamp:    time.Now(),
	}
	
	tx := &NFTTransaction{
		ID:        uuid.New().String(),
		NFTID:     nft.ID,
		ChainID:   chainID,
		Type:      "mint",
		To:       to,
		Timestamp: time.Now(),
	}
	
	nftMux.Lock()
	nfts[fmt.Sprintf("%d:%s:%s", chainID, strings.ToLower(contractAddress), tokenID)] = nft
	transactions[tx.ID] = tx
	nftMux.Unlock()
	
	return tx, nil
}

// BurnNFT burns an NFT
func BurnNFT(chainID int, contractAddress, tokenID string) (*NFTTransaction, error) {
	tx := &NFTTransaction{
		ID:        uuid.New().String(),
		NFTID:     fmt.Sprintf("%s:%s", contractAddress, tokenID),
		ChainID:   chainID,
		Type:      "burn",
		Timestamp: time.Now(),
	}
	
	nftMux.Lock()
	transactions[tx.ID] = tx
	
	// Remove NFT
	key := fmt.Sprintf("%d:%s:%s", chainID, strings.ToLower(contractAddress), tokenID)
	delete(nfts, key)
	nftMux.Unlock()
	
	return tx, nil
}

// ============================================================================
// NFT Marketplace
// ============================================================================

// CreateListing creates an NFT listing
func CreateListing(nftID, seller, price, paymentToken string, quantity int, expiresIn time.Duration) (*NFTListing, error) {
	listing := &NFTListing{
		ID:         uuid.New().String(),
		NFTID:     nftID,
		Seller:    seller,
		Price:    price,
		Quantity: quantity,
		Status:   "active",
		ExpiresAt: time.Now().Add(expiresIn),
		CreatedAt: time.Now(),
	}
	
	nftMux.Lock()
	listings[listing.ID] = listing
	nftMux.Unlock()
	
	return listing, nil
}

// FillListing completes a listing (purchase)
func FillListing(listingID, buyer string) (*NFTTransaction, error) {
	nftMux.Lock()
	listing, ok := listings[listingID]
	if !ok {
		nftMux.Unlock()
		return nil, fmt.Errorf("listing not found")
	}
	
	if listing.Status != "active" {
		nftMux.Unlock()
		return nil, fmt.Errorf("listing not active")
	}
	
	if time.Now().After(listing.ExpiresAt) {
		nftMux.Unlock()
		return nil, fmt.Errorf("listing expired")
	}
	
	listing.Status = "sold"
	
	// Create transfer transaction
	tx := &NFTTransaction{
		ID:       uuid.New().String(),
		NFTID:    listing.NFTID,
		Type:     "sale",
		From:     listing.Seller,
		To:       buyer,
		Price:   listing.Price,
		Token:   listing.PaymentToken,
		Timestamp: time.Now(),
	}
	transactions[tx.ID] = tx
	nftMux.Unlock()
	
	return tx, nil
}

// CancelListing cancels a listing
func CancelListing(listingID string) error {
	nftMux.Lock()
	if listing, ok := listings[listingID]; ok {
		listing.Status = "cancelled"
	}
	nftMux.Unlock()
	
	return nil
}

// GetActiveListings returns all active listings for an NFT
func GetActiveListings(chainID int, contractAddress string) ([]NFTListing, error) {
	result := make([]NFTListing, 0)
	
	nftMux.RLock()
	for _, listing := range listings {
		if listing.Status == "active" && !time.Now().After(listing.ExpiresAt) {
			// Check if it matches the contract
			for _, nft := range nfts {
				if nft.ContractAddress == contractAddress && nft.ChainID == chainID && 
				   strings.Contains(listing.NFTID, nft.TokenID) {
					result = append(result, *listing)
					break
				}
			}
		}
	}
	nftMux.RUnlock()
	
	return result, nil
}

// ============================================================================
// Metadata Fetching
// ============================================================================

// FetchNFTMetadata fetches and parses NFT metadata
func FetchNFTMetadata(tokenURI string) (*NFTMetadata, error) {
	if tokenURI == "" {
		return nil, nil
	}
	
	// Handle IPFS URLs
	if strings.HasPrefix(tokenURI, "ipfs://") {
		tokenURI = IPFSGateway + strings.TrimPrefix(tokenURI, "ipfs://")
	} else if strings.HasPrefix(tokenURI, "ar://") {
		tokenURI = ArweaveGateway + strings.TrimPrefix(tokenURI, "ar://")
	}
	
	// In production, fetch from HTTP/IPFS
	// For now, return basic metadata
	return &NFTMetadata{
		Name:        "NFT",
		Description: "TigerWallet NFT",
		Image:      tokenURI,
		ImageURL:   tokenURI,
	}, nil
}

// ParseMetadataAttribute parses attribute from metadata JSON
func ParseMetadataAttribute(data map[string]interface{}) []NFTAttribute {
	result := make([]NFTAttribute, 0)
	
	if attrs, ok := data["attributes"].([]interface{}); ok {
		for _, a := range attrs {
			if attrMap, ok := a.(map[string]interface{}); ok {
				attr := NFTAttribute{}
				
				if traitType, ok := attrMap["trait_type"].(string); ok {
					attr.TraitType = traitType
				}
				
				if value, ok := attrMap["value"]; ok {
					attr.Value = value
				}
				
				if displayType, ok := attrMap["display_type"].(string); ok {
					attr.DisplayType = displayType
				}
				
				result = append(result, attr)
			}
		}
	}
	
	return result
}

// ============================================================================
// Portfolio Analytics
// ============================================================================

// CalculatePortfolioValue calculates total value of NFTs owned
func CalculatePortfolioValue(chainID int, ownerAddress string) (float64, error) {
	total := 0.0
	
	nftMux.RLock()
	for _, nft := range nfts {
		if strings.EqualFold(nft.Owner, ownerAddress) && nft.ChainID == chainID {
			// In production, fetch floor price and calculate
			total += 0.0 // Placeholder
		}
	}
	nftMux.RUnlock()
	
	return total, nil
}

// GetPortfolioSummary returns portfolio summary
func GetPortfolioSummary(chainID int, ownerAddress string) (map[string]interface{}, error) {
	nfts, err := GetNFTsOfOwner(chainID, ownerAddress, 1000, 0)
	if err != nil {
		return nil, err
	}
	
	collections := make(map[string]int)
	standards := make(map[string]int)
	totalValue := 0.0
	
	for _, nft := range nfts {
		collections[nft.ContractAddress]++
		standards[nft.Standard]++
		totalValue += 0.0 // Placeholder
	}
	
	return map[string]interface{}{
		"total_nfts":      len(nfts),
		"collections":    collections,
		"standards":     standards,
		"total_value":   totalValue,
		"chain_id":     chainID,
		"owner":       ownerAddress,
	}, nil
}

// ============================================================================
// HTTP Handlers
// ============================================================================

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "nft"})
}

func getNFTHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	chainID := r.URL.Query().Get("chain_id")
	contract := r.URL.Query().Get("contract")
	tokenID := r.URL.Query().Get("token_id")
	
	if chainID == "" || contract == "" || tokenID == "" {
		http.Error(w, "missing parameters", 400)
		return
	}
	
	nft, err := GetNFT(parseInt(chainID), contract, tokenID)
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	
	json.NewEncoder(w).Encode(nft)
}

func getOwnerNFTsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	chainID := r.URL.Query().Get("chain_id")
	owner := r.URL.Query().Get("owner")
	limit := parseInt(r.URL.Query().Get("limit"))
	offset := parseInt(r.URL.Query().Get("offset"))
	
	if chainID == "" || owner == "" {
		http.Error(w, "missing parameters", 400)
		return
	}
	
	if limit == 0 {
		limit = 20
	}
	
	nfts, err := GetNFTsOfOwner(parseInt(chainID), owner, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	
	json.NewEncoder(w).Encode(map[string]interface{}{
		"nfts":  nfts,
		"total": len(nfts),
	})
}

func getCollectionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	chainID := r.URL.Query().Get("chain_id")
	contract := r.URL.Query().Get("contract")
	
	if chainID == "" || contract == "" {
		http.Error(w, "missing parameters", 400)
		return
	}
	
	col, err := GetNFTCollection(parseInt(chainID), contract)
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	
	json.NewEncoder(w).Encode(col)
}

func transferNFTHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var req struct {
		ChainID        int    `json:"chain_id"`
		ContractAddress string `json:"contract_address"`
		TokenID       string `json:"token_id"`
		From         string `json:"from"`
		To           string `json:"to"`
		Quantity     int    `json:"quantity"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	
	tx, err := TransferNFT(req.ChainID, req.ContractAddress, req.TokenID, req.From, req.To, req.Quantity)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	
	json.NewEncoder(w).Encode(tx)
}

func batchTransferHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var req struct {
		Transfers []struct {
			ChainID        int    `json:"chain_id"`
			ContractAddress string `json:"contract_address"`
			TokenID       string `json:"token_id"`
			From         string `json:"from"`
			To           string `json:"to"`
			Quantity     int    `json:"quantity"`
		} `json:"transfers"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	
	txs, err := BatchTransferNFT(req.Transfers)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	
	json.NewEncoder(w).Encode(txs)
}

func mintNFTHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var req struct {
		ChainID        int    `json:"chain_id"`
		ContractAddress string `json:"contract_address"`
		To           string `json:"to"`
		TokenURI     string `json:"token_uri"`
		Quantity    int    `json:"quantity"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	
	tx, err := MintNFT(req.ChainID, req.ContractAddress, req.To, req.TokenURI, req.Quantity)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	
	json.NewEncoder(w).Encode(tx)
}

func burnNFTHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var req struct {
		ChainID        int    `json:"chain_id"`
		ContractAddress string `json:"contract_address"`
		TokenID     string `json:"token_id"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	
	tx, err := BurnNFT(req.ChainID, req.ContractAddress, req.TokenID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	
	json.NewEncoder(w).Encode(tx)
}

func createListingHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var req struct {
		NFTID       string `json:"nft_id"`
		Seller     string `json:"seller"`
		Price     string `json:"price"`
		PaymentToken string `json:"payment_token"`
		Quantity   int    `json:"quantity"`
		ExpiresIn int64  `json:"expires_in"` // hours
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	
	listing, err := CreateListing(req.NFTID, req.Seller, req.Price, req.PaymentToken, req.Quantity, time.Duration(req.ExpiresIn)*time.Hour)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	
	json.NewEncoder(w).Encode(listing)
}

func fillListingHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var req struct {
		ListingID string `json:"listing_id"`
		Buyer    string `json:"buyer"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	
	tx, err := FillListing(req.ListingID, req.Buyer)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	
	json.NewEncoder(w).Encode(tx)
}

func getListingsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	chainID := r.URL.Query().Get("chain_id")
	contract := r.URL.Query().Get("contract")
	
	listings, err := GetActiveListings(parseInt(chainID), contract)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	
	json.NewEncoder(w).Encode(listings)
}

func portfolioSummaryHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	chainID := r.URL.Query().Get("chain_id")
	owner := r.URL.Query().Get("owner")
	
	if chainID == "" || owner == "" {
		http.Error(w, "missing parameters", 400)
		return
	}
	
	summary, err := GetPortfolioSummary(parseInt(chainID), owner)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	
	json.NewEncoder(w).Encode(summary)
}

// ============================================================================
// Router
// ============================================================================

func router() http.Handler {
	mux := http.NewServeMux()
	
	// Health
	mux.HandleFunc("/health", healthHandler)
	
	// NFT queries
	mux.HandleFunc("/api/nft", getNFTHandler)
	mux.HandleFunc("/api/nfts/owner", getOwnerNFTsHandler)
	mux.HandleFunc("/api/nft/collection", getCollectionHandler)
	mux.HandleFunc("/api/nft/listings", getListingsHandler)
	
	// NFT transactions
	mux.HandleFunc("/api/nft/transfer", transferNFTHandler)
	mux.HandleFunc("/api/nft/transfer/batch", batchTransferHandler)
	mux.HandleFunc("/api/nft/mint", mintNFTHandler)
	mux.HandleFunc("/api/nft/burn", burnNFTHandler)
	
	// Marketplace
	mux.HandleFunc("/api/nft/listing/create", createListingHandler)
	mux.HandleFunc("/api/nft/listing/fill", fillListingHandler)
	
	// Analytics
	mux.HandleFunc("/api/nft/portfolio", portfolioSummaryHandler)
	
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
	fmt.Printf("NFT Service starting on port %d\n", NFTServicePort)
	
	// Initialize sample data
	initNFTData()
	
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", NFTServicePort),
		Handler:      router(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	
	fmt.Printf("NFT Service ready on :%d\n", NFTServicePort)
	if err := server.ListenAndServe(); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

func initNFTData() {
	// Initialize with sample NFTs for demo
	sampleNFTs := []NFT{
		{
			ID:              "nft_1",
			TokenID:         "1",
			ContractAddress: "0x1234567890123456789012345678901234567890",
			ChainID:        1,
			Owner:          "0xabcd1234567890123456789012345678901234567",
			Standard:      "erc721",
			Quantity:      1,
			Timestamp:    time.Now(),
		},
		{
			ID:              "nft_2",
			TokenID:         "2",
			ContractAddress: "0x1234567890123456789012345678901234567890",
			ChainID:        1,
			Owner:          "0xabcd1234567890123456789012345678901234567",
			Standard:      "erc721",
			Quantity:      1,
			Timestamp:    time.Now(),
		},
	}
	
	nftMux.Lock()
	for _, nft := range sampleNFTs {
		key := fmt.Sprintf("%d:%s:%s", nft.ChainID, strings.ToLower(nft.ContractAddress), nft.TokenID)
		nfts[key] = &nft
	}
	nftMux.Unlock()
}