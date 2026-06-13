package nft

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sort"
	"sync"
	"time"
)

// ============================================================================
// NFT Marketplace Integration
// ============================================================================

// MarketplaceService provides NFT marketplace functionality
type MarketplaceService struct {
	mu           sync.RWMutex
	collections  map[string]*Collection
	listings    map[string]*Listing
	offers      map[string]*Offer
	transfers   map[string][]Transfer
	config      *Config
	httpClient *http.Client
}

// Config for NFT service
type Config struct {
	OpenSeaAPIKey   string
	BlurAPIKey    string
	MagicEdenAPIKey string
	Timeout       time.Duration
}

// Collection represents NFT collection
type Collection struct {
	Address     string
	Name        string
	Symbol      string
	ChainID     uint64
	Decimals    uint8
	TotalSupply uint64
	FloorPrice  *big.Rat
	Volume24h  *big.Rat
	Owners     uint64
}

// NFT represents an NFT
type NFT struct {
	TokenID     string
	Collection string
	Owner      string
	URI        string
	Metadata   *Metadata
	Price      *big.Rat
	LastSale   *big.Rat
}

// Metadata represents NFT metadata
type Metadata struct {
	Name        string
	Description string
	Image      string
	Attributes []Attribute
}

// Attribute represents trait
type Attribute struct {
	TraitType   string `json:"trait_type"`
	Value      string `json:"value"`
	DisplayType string `json:"display_type,omitempty"`
	Rarity     float64
}

// Listing represents a listing
type Listing struct {
	ListingID   string
	NFT        *NFT
	Seller     string
	Price      *big.Rat
	Expiration time.Time
	Status     ListingStatus
}

// ListingStatus enum
type ListingStatus string

const (
	ListingStatusActive  ListingStatus = "active"
	ListingStatusFilled ListingStatus = "filled"
	ListingStatusCancelled ListingStatus = "cancelled"
	ListingStatusExpired ListingStatus = "expired"
)

// Offer represents an offer
type Offer struct {
	OfferID   string
	NFT       *NFT
	Buyer     string
	Price     *big.Rat
	Status    OfferStatus
	CreatedAt time.Time
	ExpiresAt time.Time
}

// OfferStatus enum
type OfferStatus string

const (
	OfferStatusPending  OfferStatus = "pending"
	OfferStatusAccepted OfferStatus = "accepted"
	OfferStatusRejected OfferStatus = "rejected"
	OfferStatusExpired OfferStatus = "expired"
)

// Transfer represents a transfer
type Transfer struct {
	TxHash     string
	NFT       *NFT
	From      string
	To        string
	Price     *big.Rat
	Timestamp time.Time
}

// ============================================================================
// Marketplace Implementations
// ============================================================================

// OpenSeaMarketplace represents OpenSea
type OpenSeaMarketplace struct {
	apiKey   string
	baseURL  string
	client  *http.Client
}

// NewOpenSeaMarketplace creates OpenSea marketplace
func NewOpenSeaMarketplace(apiKey string) *OpenSeaMarketplace {
	return &OpenSeaMarketplace{
		apiKey:   apiKey,
		baseURL:  "https://api.opensea.io",
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (o *OpenSeaMarketplace) Name() string { return "OpenSea" }

func (o *OpenSeaMarketplace) GetCollection(ctx context.Context, addr string) (*Collection, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", 
		fmt.Sprintf("%s/v2/collection/%s", o.baseURL, addr), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", o.apiKey))
	
	resp, err := o.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	var result struct {
		Collection struct {
			Name        string `json:"name"`
			Symbol      string `json:"symbol"`
			TotalSupply string `json:"total_supply"`
			FloorPrice string `json:"floor_price"`
		} `json:"collection"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	
	supply, _ := strconv.ParseUint(result.Collection.TotalSupply, 10, 64)
	floor, _ := new(big.Rat).SetString(result.Collection.FloorPrice)
	
	return &Collection{
		Address:     addr,
		Name:        result.Collection.Name,
		Symbol:     result.Collection.Symbol,
		TotalSupply: supply,
		FloorPrice: floor,
	}, nil
}

func (o *OpenSeaMarketplace) GetListings(ctx context.Context, collection string) ([]Listing, error) {
	return []Listing{}, nil
}

func (o *OpenSeaMarketplace) CreateListing(ctx context.Context, listing *Listing) error {
	return nil
}

func (o *OpenSeaMarketplace) FillListing(ctx context.Context, listingID, buyer string) error {
	return nil
}

// MagicEdenMarketplace represents Magic Eden
type MagicEdenMarketplace struct {
	apiKey  string
	baseURL string
	client *http.Client
}

// NewMagicEdenMarketplace creates Magic Eden marketplace
func NewMagicEdenMarketplace(apiKey string) *MagicEdenMarketplace {
	return &MagicEdenMarketplace{
		apiKey:  apiKey,
		baseURL: "https://api-mainnet.magiceden.io",
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (m *MagicEdenMarketplace) Name() string { return "MagicEden" }

func (m *MagicEdenMarketplace) GetCollection(ctx context.Context, addr string) (*Collection, error) {
	return &Collection{
		Address: addr,
		Name:   "Collection",
	}, nil
}

func (m *MagicEdenMarketplace) GetListings(ctx context.Context, collection string) ([]Listing, error) {
	return []Listing{}, nil
}

func (m *MagicEdenMarketplace) CreateListing(ctx context.Context, listing *Listing) error {
	return nil
}

func (m *MagicEdenMarketplace) FillListing(ctx context.Context, listingID, buyer string) error {
	return nil
}

// ============================================================================
// NFT Service
// ============================================================================

// Service provides unified NFT functionality
type Service struct {
	mu          sync.RWMutex
	marketplaces map[string]Marketplace
	nfts        map[string]map[string]*NFT
	transfers   map[string][]Transfer
}

// Marketplace interface
type Marketplace interface {
	Name() string
	GetCollection(ctx context.Context, addr string) (*Collection, error)
	GetListings(ctx context.Context, collection string) ([]Listing, error)
	CreateListing(ctx context.Context, listing *Listing) error
	FillListing(ctx context.Context, listingID, buyer string) error
}

// NewService creates new NFT service
func NewService() *Service {
	return &Service{
		marketplaces: make(map[string]Marketplace),
		nfts:        make(map[string]map[string]*NFT),
		transfers:   make(map[string][]Transfer),
	}
}

// RegisterMarketplace registers a marketplace
func (s *Service) RegisterMarketplace(name string, mp Marketplace) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.marketplaces[name] = mp
}

// GetNFT gets NFT
func (s *Service) GetNFT(collection, tokenID string) *NFT {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if collectionNFTs := s.nfts[collection]; collectionNFTs != nil {
		return collectionNFTs[tokenID]
	}
	return nil
}

// GetCollection gets collection from any marketplace
func (s *Service) GetCollection(ctx context.Context, addr string) (*Collection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	for _, mp := range s.marketplaces {
		collection, err := mp.GetCollection(ctx, addr)
		if err == nil {
			return collection, nil
		}
	}
	return nil, fmt.Errorf("collection not found")
}

// GetListings gets listings from all marketplaces
func (s *Service) GetListings(ctx context.Context, collection string) ([]Listing, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var allListings []Listing
	for _, mp := range s.marketplaces {
		listings, err := mp.GetListings(ctx, collection)
		if err == nil {
			allListings = append(allListings, listings...)
		}
	}
	
	sort.Slice(allListings, func(i, j int) bool {
		return allListings[i].Price.Cmp(allListings[j].Price) < 0
	})
	
	return allListings, nil
}

// CreateListing creates listing
func (s *Service) CreateListing(ctx context.Context, marketplace string, listing *Listing) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	mp, ok := s.marketplaces[marketplace]
	if !ok {
		return fmt.Errorf("marketplace not found: %s", marketplace)
	}
	
	return mp.CreateListing(ctx, listing)
}

// FillListing fills a listing
func (s *Service) FillListing(ctx context.Context, marketplace, listingID, buyer string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	mp, ok := s.marketplaces[marketplace]
	if !ok {
		return fmt.Errorf("marketplace not found: %s", marketplace)
	}
	
	return mp.FillListing(ctx, listingID, buyer)
}

// ============================================================================
// Ordinals (Bitcoin NFT) Support
// ============================================================================

// OrdinalsService provides ordinals support
type OrdinalsService struct {
	indexerURL string
	client    *http.Client
}

// NewOrdinalsService creates ordinals service
func NewOrdinalsService(indexerURL string) *OrdinalsService {
	return &OrdinalsService{
		indexerURL: indexerURL,
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

// GetOrdinal gets ordinal by ID
func (o *OrdinalsService) GetOrdinal(ctx context.Context, id string) (*Ordinal, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", 
		fmt.Sprintf("%s/ordinal/%s", o.indexerURL, id), nil)
	
	resp, err := o.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	var ordinal Ordinal
	if err := json.NewDecoder(resp.Body).Decode(&ordinal); err != nil {
		return nil, err
	}
	
	return &ordinal, nil
}

// GetOrdinals gets ordinals for address
func (o *OrdinalsService) GetOrdinals(ctx context.Context, addr string) ([]Ordinal, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", 
		fmt.Sprintf("%s/ordinals/%s", o.indexerURL, addr), nil)
	
	resp, err := o.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	var ordinals []Ordinal
	if err := json.NewDecoder(resp.Body).Decode(&ordinals); err != nil {
		return nil, err
	}
	
	return ordinals, nil
}

// Ordinal represents a Bitcoin ordinal
type Ordinal struct {
	ID          string    `json:"id"`
	Number      uint64    `json:"number"`
	Address    string    `json:"address"`
	ContentType string   `json:"content_type"`
	Content    string    `json:"content"`
	Metadata   string    `json:"metadata"`
	Timestamp  time.Time `json:"timestamp"`
}

// ============================================================================
// HTTP Handlers
// ============================================================================

// HandleGetCollection handles get collection
func (s *Service) HandleGetCollection(w http.ResponseWriter, r *http.Request) {
	addr := r.URL.Query().Get("address")
	if addr == "" {
		http.Error(w, "Missing address", http.StatusBadRequest)
		return
	}
	
	collection, err := s.GetCollection(r.Context(), addr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(collection)
}

// HandleGetListings handles get listings
func (s *Service) HandleGetListings(w http.ResponseWriter, r *http.Request) {
	collection := r.URL.Query().Get("collection")
	if collection == "" {
		http.Error(w, "Missing collection", http.StatusBadRequest)
		return
	}
	
	listings, err := s.GetListings(r.Context(), collection)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(listings)
}

// HandleCreateListing handles create listing
func (s *Service) HandleCreateListing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		Marketplace string   `json:"marketplace"`
		Listing     *Listing `json:"listing"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	if err := s.CreateListing(r.Context(), req.Marketplace, req.Listing); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// HandleFillListing handles fill listing
func (s *Service) HandleFillListing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		Marketplace string `json:"marketplace"`
		ListingID   string `json:"listingId"`
		Buyer       string `json:"buyer"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	if err := s.FillListing(r.Context(), req.Marketplace, req.ListingID, req.Buyer); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// Serve starts NFT service
func (s *Service) Serve(addr string) error {
	http.HandleFunc("/v1/collection", s.HandleGetCollection)
	http.HandleFunc("/v1/listings", s.HandleGetListings)
	http.HandleFunc("/v1/listings/create", s.HandleCreateListing)
	http.HandleFunc("/v1/listings/fill", s.HandleFillListing)
	
	return http.ListenAndServe(addr, nil)
}