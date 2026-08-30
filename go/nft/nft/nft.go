package nft

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ============================================================================
// NFT Marketplace Integration
// ============================================================================

// MarketplaceService provides NFT marketplace functionality
type MarketplaceService struct {
	mu          sync.RWMutex
	collections map[string]*Collection
	listings    map[string]*Listing
	offers      map[string]*Offer
	transfers   map[string][]Transfer
	config      *Config
	httpClient  *http.Client
}

// Config for NFT service
type Config struct {
	OpenSeaAPIKey   string
	BlurAPIKey      string
	MagicEdenAPIKey string
	Timeout         time.Duration
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
	Volume24h   *big.Rat
	Owners      uint64
}

// NFT represents an NFT
type NFT struct {
	TokenID    string
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
	Image       string
	Attributes  []Attribute
}

// Attribute represents trait
type Attribute struct {
	TraitType   string `json:"trait_type"`
	Value       string `json:"value"`
	DisplayType string `json:"display_type,omitempty"`
	Rarity      float64
}

// Listing represents a listing
type Listing struct {
	ListingID  string
	NFT        *NFT
	Seller     string
	Price      *big.Rat
	Expiration time.Time
	Status     ListingStatus
	// Chain is the marketplace chain identifier (default per marketplace).
	Chain string
	// SignedOrder carries the client-signed order payload (Seaport order JSON
	// for OpenSea, execute/list payload for Reservoir). Required on create:
	// the service relays signatures, it never fabricates them.
	SignedOrder json.RawMessage
}

// ListingStatus enum
type ListingStatus string

const (
	ListingStatusActive    ListingStatus = "active"
	ListingStatusFilled    ListingStatus = "filled"
	ListingStatusCancelled ListingStatus = "cancelled"
	ListingStatusExpired   ListingStatus = "expired"
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
	OfferStatusExpired  OfferStatus = "expired"
)

// Transfer represents a transfer
type Transfer struct {
	TxHash    string
	NFT       *NFT
	From      string
	To        string
	Price     *big.Rat
	Timestamp time.Time
}

// ============================================================================
// Marketplace Implementations
// ===========================================================================
// Marketplace Implementations
// ===========================================================================

// RawTransaction is an unsigned on-chain transaction envelope returned by
// fulfillment endpoints; the buyer's wallet signs + broadcasts it.
type RawTransaction struct {
	Chain string `json:"chain"`
	To    string `json:"to"`
	Value string `json:"value"`
	Data  string `json:"data"`
}

// seaportProtocolAddress is the deployed Seaport 1.6 conduit controller
// address used by OpenSea fulfillment on all supported EVM chains.
const seaportProtocolAddress = "0x0000000000000068F116a894984e2DB1123eB395"

// httpGetJSON performs a GET with an optional API key header and decodes JSON.
// Fail-closed: non-2xx is an error carrying the upstream status.
func httpGetJSON(ctx context.Context, client *http.Client, url, apiKey, keyHeader string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	if apiKey != "" {
		req.Header.Set(keyHeader, apiKey)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("upstream returned HTTP %d for %s", resp.StatusCode, url)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// httpPostJSON performs a POST with a JSON body and decodes the JSON response.
func httpPostJSON(ctx context.Context, client *http.Client, url, apiKey, keyHeader string, body, out any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(raw)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if apiKey != "" {
		req.Header.Set(keyHeader, apiKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("upstream returned HTTP %d for %s", resp.StatusCode, url)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// OpenSeaMarketplace represents OpenSea (real API v2 integration).
type OpenSeaMarketplace struct {
	apiKey  string
	baseURL string
	chain   string
	client  *http.Client
}

// NewOpenSeaMarketplace creates OpenSea marketplace. The API key comes from
// the operator (OPENSEA_API_KEY); when empty, keyed endpoints fail closed
// with a descriptive error instead of returning fabricated data.
func NewOpenSeaMarketplace(apiKey string) *OpenSeaMarketplace {
	return &OpenSeaMarketplace{
		apiKey:  apiKey,
		baseURL: "https://api.opensea.io",
		chain:   "ethereum",
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (o *OpenSeaMarketplace) Name() string { return "OpenSea" }

func (o *OpenSeaMarketplace) requireKey() error {
	if o.apiKey == "" {
		return fmt.Errorf("opensea: API key not configured (set OPENSEA_API_KEY)")
	}
	return nil
}

func (o *OpenSeaMarketplace) GetCollection(ctx context.Context, slug string) (*Collection, error) {
	if err := o.requireKey(); err != nil {
		return nil, err
	}
	var detail struct {
		Collection  string `json:"collection"`
		Name        string `json:"name"`
		TotalSupply uint64 `json:"total_supply"`
		NumOwners   uint64 `json:"num_owners"`
	}
	if err := httpGetJSON(ctx, o.client, fmt.Sprintf("%s/api/v2/collections/%s", o.baseURL, slug), o.apiKey, "x-api-key", &detail); err != nil {
		return nil, err
	}
	col := &Collection{
		Address:     slug,
		Name:        detail.Name,
		Symbol:      detail.Collection,
		TotalSupply: detail.TotalSupply,
		Owners:      detail.NumOwners,
	}
	// Floor price lives on the stats endpoint; best-effort (absent stats is
	// not an error — the collection itself is real).
	var stats struct {
		Total struct {
			FloorPrice json.Number `json:"floor_price"`
			Volume     json.Number `json:"volume"`
		} `json:"total"`
	}
	if err := httpGetJSON(ctx, o.client, fmt.Sprintf("%s/api/v2/collections/%s/stats", o.baseURL, slug), o.apiKey, "x-api-key", &stats); err == nil {
		if f, ok := new(big.Rat).SetString(stats.Total.FloorPrice.String()); ok {
			col.FloorPrice = f
		}
		if v, ok := new(big.Rat).SetString(stats.Total.Volume.String()); ok {
			col.Volume24h = v
		}
	}
	return col, nil
}

// ListCollections enumerates collections on a chain (real OpenSea v2).
func (o *OpenSeaMarketplace) ListCollections(ctx context.Context, chain string, limit int) ([]Collection, string, error) {
	if err := o.requireKey(); err != nil {
		return nil, "", err
	}
	if chain == "" {
		chain = o.chain
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var out struct {
		Collections []struct {
			Slug        string `json:"collection"`
			Name        string `json:"name"`
			TotalSupply uint64 `json:"total_supply"`
		} `json:"collections"`
		Next string `json:"next"`
	}
	url := fmt.Sprintf("%s/api/v2/collections?chain=%s&limit=%d", o.baseURL, chain, limit)
	if err := httpGetJSON(ctx, o.client, url, o.apiKey, "x-api-key", &out); err != nil {
		return nil, "", err
	}
	cols := make([]Collection, 0, len(out.Collections))
	for _, c := range out.Collections {
		cols = append(cols, Collection{Address: c.Slug, Name: c.Name, Symbol: c.Slug, TotalSupply: c.TotalSupply})
	}
	return cols, out.Next, nil
}

// GetNFTsByOwner enumerates NFTs owned by an address on a chain (real OpenSea v2).
func (o *OpenSeaMarketplace) GetNFTsByOwner(ctx context.Context, chain, address string, limit int) ([]NFT, string, error) {
	if err := o.requireKey(); err != nil {
		return nil, "", err
	}
	if chain == "" {
		chain = o.chain
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var out struct {
		NFTs []struct {
			Identifier string `json:"identifier"`
			Collection string `json:"collection"`
			Contract   string `json:"contract"`
			Name       string `json:"name"`
			ImageURL   string `json:"image_url"`
		} `json:"nfts"`
		Next string `json:"next"`
	}
	url := fmt.Sprintf("%s/api/v2/chain/%s/account/%s/nfts?limit=%d", o.baseURL, chain, address, limit)
	if err := httpGetJSON(ctx, o.client, url, o.apiKey, "x-api-key", &out); err != nil {
		return nil, "", err
	}
	nfts := make([]NFT, 0, len(out.NFTs))
	for _, n := range out.NFTs {
		nfts = append(nfts, NFT{
			TokenID:    n.Identifier,
			Collection: n.Contract,
			Owner:      address,
			Metadata:   &Metadata{Name: n.Name, Image: n.ImageURL},
		})
	}
	return nfts, out.Next, nil
}

func (o *OpenSeaMarketplace) GetListings(ctx context.Context, collection string) ([]Listing, error) {
	if err := o.requireKey(); err != nil {
		return nil, err
	}
	if collection == "" {
		return nil, fmt.Errorf("collection slug required")
	}
	var out struct {
		Listings []struct {
			OrderHash string `json:"order_hash"`
			Chain     string `json:"chain"`
			Price     struct {
				Current struct {
					Currency string      `json:"currency"`
					Value    json.Number `json:"value"`
					Decimals int         `json:"decimals"`
				} `json:"current"`
			} `json:"price"`
			ProtocolData struct {
				Parameters struct {
					Offerer string `json:"offerer"`
					Offer   []struct {
						IdentifierOrCriteria string `json:"identifierOrCriteria"`
						Token                string `json:"token"`
					} `json:"offer"`
					EndTime string `json:"endTime"`
				} `json:"parameters"`
			} `json:"protocol_data"`
		} `json:"listings"`
	}
	url := fmt.Sprintf("%s/api/v2/listings/collection/%s/all?limit=50", o.baseURL, collection)
	if err := httpGetJSON(ctx, o.client, url, o.apiKey, "x-api-key", &out); err != nil {
		return nil, err
	}
	listings := make([]Listing, 0, len(out.Listings))
	for _, l := range out.Listings {
		price, _ := new(big.Rat).SetString(l.Price.Current.Value.String())
		if price != nil && l.Price.Current.Decimals > 0 {
			denom := new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(l.Price.Current.Decimals)), nil))
			price = new(big.Rat).Quo(price, denom)
		}
		entry := Listing{
			ListingID: l.OrderHash,
			Seller:    l.ProtocolData.Parameters.Offerer,
			Price:     price,
			Status:    ListingStatusActive,
		}
		if len(l.ProtocolData.Parameters.Offer) > 0 {
			entry.NFT = &NFT{
				TokenID:    l.ProtocolData.Parameters.Offer[0].IdentifierOrCriteria,
				Collection: l.ProtocolData.Parameters.Offer[0].Token,
			}
		}
		if endUnix, err := strconv.ParseInt(l.ProtocolData.Parameters.EndTime, 10, 64); err == nil {
			entry.Expiration = time.Unix(endUnix, 0)
		}
		listings = append(listings, entry)
	}
	return listings, nil
}

// CreateListing relays a client-signed Seaport order to OpenSea. The order
// MUST be signed client-side by the seller's wallet — this service never
// fabricates signatures. Fail-closed without a key or a signed order.
func (o *OpenSeaMarketplace) CreateListing(ctx context.Context, listing *Listing) error {
	if err := o.requireKey(); err != nil {
		return err
	}
	if listing == nil || len(listing.SignedOrder) == 0 {
		return fmt.Errorf("opensea: signed_order (client-signed seaport order JSON) is required")
	}
	chain := listing.Chain
	if chain == "" {
		chain = o.chain
	}
	var raw json.RawMessage
	url := fmt.Sprintf("%s/api/v2/orders/%s/seaport/listings", o.baseURL, chain)
	return httpPostJSON(ctx, o.client, url, o.apiKey, "x-api-key", json.RawMessage(listing.SignedOrder), &raw)
}

// FillListing fetches the real fulfillment transaction for an order hash;
// the returned RawTransaction must be signed + broadcast by the buyer wallet.
func (o *OpenSeaMarketplace) FillListing(ctx context.Context, listingID, buyer string) (*RawTransaction, error) {
	if err := o.requireKey(); err != nil {
		return nil, err
	}
	if listingID == "" || buyer == "" {
		return nil, fmt.Errorf("listingID (order hash) and buyer address are required")
	}
	body := map[string]any{
		"listing": map[string]any{
			"hash":             listingID,
			"chain":            o.chain,
			"protocol_address": seaportProtocolAddress,
		},
		"fulfiller": map[string]any{"address": buyer},
	}
	var out struct {
		FulfillmentData struct {
			Transaction struct {
				Chain string      `json:"chain"`
				To    string      `json:"to"`
				Value json.Number `json:"value"`
				Data  string      `json:"data"`
			} `json:"transaction"`
		} `json:"fulfillment_data"`
	}
	if err := httpPostJSON(ctx, o.client, o.baseURL+"/v2/listings/fulfillment_data", o.apiKey, "x-api-key", body, &out); err != nil {
		return nil, err
	}
	if out.FulfillmentData.Transaction.To == "" {
		return nil, fmt.Errorf("opensea returned no fulfillment transaction for order %s", listingID)
	}
	return &RawTransaction{
		Chain: out.FulfillmentData.Transaction.Chain,
		To:    out.FulfillmentData.Transaction.To,
		Value: out.FulfillmentData.Transaction.Value.String(),
		Data:  out.FulfillmentData.Transaction.Data,
	}, nil
}

// MagicEdenMarketplace represents Magic Eden (real public stats endpoint +
// Reservoir execute relay for EVM listing/fill flows).
type MagicEdenMarketplace struct {
	apiKey       string
	baseURL      string
	reservoirURL string
	reservoirKey string
	client       *http.Client
}

// NewMagicEdenMarketplace creates Magic Eden marketplace. reservoirKey is
// optional (RESERVOIR_API_KEY); the public Reservoir endpoints work keyless
// with a source attribution header.
func NewMagicEdenMarketplace(apiKey string) *MagicEdenMarketplace {
	return &MagicEdenMarketplace{
		apiKey:       apiKey,
		baseURL:      "https://api-mainnet.magiceden.dev",
		reservoirURL: "https://api.reservoir.tools",
		client:       &http.Client{Timeout: 30 * time.Second},
	}
}

// WithReservoir configures the Reservoir relay endpoint/key.
func (m *MagicEdenMarketplace) WithReservoir(url, key string) *MagicEdenMarketplace {
	if url != "" {
		m.reservoirURL = url
	}
	m.reservoirKey = key
	return m
}

func (m *MagicEdenMarketplace) Name() string { return "MagicEden" }

// GetCollection fetches real collection stats from Magic Eden's public API.
func (m *MagicEdenMarketplace) GetCollection(ctx context.Context, symbol string) (*Collection, error) {
	if symbol == "" {
		return nil, fmt.Errorf("collection symbol required")
	}
	var out struct {
		Symbol     string      `json:"symbol"`
		Name       string      `json:"name"`
		FloorPrice json.Number `json:"floorPrice"`
		VolumeAll  json.Number `json:"volumeAll"`
	}
	if err := httpGetJSON(ctx, m.client, fmt.Sprintf("%s/v2/collections/%s/stats", m.baseURL, symbol), m.apiKey, "Authorization", &out); err != nil {
		return nil, err
	}
	col := &Collection{Address: symbol, Symbol: out.Symbol, Name: out.Name}
	if f, ok := new(big.Rat).SetString(out.FloorPrice.String()); ok {
		col.FloorPrice = f
	}
	if v, ok := new(big.Rat).SetString(out.VolumeAll.String()); ok {
		col.Volume24h = v
	}
	return col, nil
}

// GetListings fetches real active listings for an EVM collection via the
// Reservoir tokens endpoint (Magic Eden's EVM liquidity runs on Reservoir).
func (m *MagicEdenMarketplace) GetListings(ctx context.Context, collection string) ([]Listing, error) {
	if collection == "" {
		return nil, fmt.Errorf("collection contract address required")
	}
	var out struct {
		Tokens []struct {
			Token struct {
				Contract string `json:"contract"`
				TokenID  string `json:"tokenId"`
				Name     string `json:"name"`
				Owner    string `json:"owner"`
			} `json:"token"`
			Market struct {
				FloorAsk struct {
					ID    string `json:"id"`
					Price struct {
						Amount struct {
							Native json.Number `json:"native"`
						} `json:"amount"`
					} `json:"price"`
					ValidUntil int64 `json:"validUntil"`
				} `json:"floorAsk"`
			} `json:"market"`
		} `json:"tokens"`
	}
	url := fmt.Sprintf("%s/tokens/v7?collection=%s&limit=50&includeAttributes=false", m.reservoirURL, collection)
	if err := httpGetJSON(ctx, m.client, url, m.reservoirKey, "x-api-key", &out); err != nil {
		return nil, err
	}
	listings := make([]Listing, 0, len(out.Tokens))
	for _, t := range out.Tokens {
		if t.Market.FloorAsk.ID == "" {
			continue
		}
		price, _ := new(big.Rat).SetString(t.Market.FloorAsk.Price.Amount.Native.String())
		listings = append(listings, Listing{
			ListingID:  t.Market.FloorAsk.ID,
			Seller:     t.Token.Owner,
			Price:      price,
			Expiration: time.Unix(t.Market.FloorAsk.ValidUntil, 0),
			Status:     ListingStatusActive,
			NFT:        &NFT{TokenID: t.Token.TokenID, Collection: t.Token.Contract, Owner: t.Token.Owner, Metadata: &Metadata{Name: t.Token.Name}},
		})
	}
	return listings, nil
}

// CreateListing relays a client-signed Reservoir list payload. The signing
// steps are executed by the seller's wallet client-side; this service only
// relays. Fail-closed without a signed payload.
func (m *MagicEdenMarketplace) CreateListing(ctx context.Context, listing *Listing) error {
	if listing == nil || len(listing.SignedOrder) == 0 {
		return fmt.Errorf("magiceden/reservoir: signed_order (client-signed execute/list payload) is required")
	}
	var raw json.RawMessage
	return httpPostJSON(ctx, m.client, m.reservoirURL+"/execute/list/v5", m.reservoirKey, "x-api-key", json.RawMessage(listing.SignedOrder), &raw)
}

// FillListing requests real buy steps from Reservoir and returns the first
// unsigned transaction for the buyer wallet to sign + broadcast.
func (m *MagicEdenMarketplace) FillListing(ctx context.Context, listingID, buyer string) (*RawTransaction, error) {
	if listingID == "" || buyer == "" {
		return nil, fmt.Errorf("listingID (order id) and buyer address are required")
	}
	body := map[string]any{
		"items": []map[string]any{{"orderId": listingID}},
		"taker": buyer,
	}
	var out struct {
		Steps []struct {
			Items []struct {
				Data struct {
					To    string `json:"to"`
					Data  string `json:"data"`
					Value string `json:"value"`
				} `json:"data"`
			} `json:"items"`
		} `json:"steps"`
	}
	if err := httpPostJSON(ctx, m.client, m.reservoirURL+"/execute/buy/v7", m.reservoirKey, "x-api-key", body, &out); err != nil {
		return nil, err
	}
	for _, step := range out.Steps {
		for _, item := range step.Items {
			if item.Data.To != "" && item.Data.Data != "" {
				return &RawTransaction{Chain: "ethereum", To: item.Data.To, Value: item.Data.Value, Data: item.Data.Data}, nil
			}
		}
	}
	return nil, fmt.Errorf("reservoir returned no transaction steps for order %s", listingID)
}

// ============================================================================
// NFT Service
// ============================================================================

// Service provides unified NFT functionality
type Service struct {
	mu           sync.RWMutex
	marketplaces map[string]Marketplace
	nfts         map[string]map[string]*NFT
	transfers    map[string][]Transfer
}

// Marketplace interface
type Marketplace interface {
	Name() string
	GetCollection(ctx context.Context, addr string) (*Collection, error)
	GetListings(ctx context.Context, collection string) ([]Listing, error)
	CreateListing(ctx context.Context, listing *Listing) error
	// FillListing returns the real unsigned fulfillment transaction for the
	// buyer wallet to sign + broadcast (never a fabricated hash).
	FillListing(ctx context.Context, listingID, buyer string) (*RawTransaction, error)
}

// NewService creates new NFT service
func NewService() *Service {
	return &Service{
		marketplaces: make(map[string]Marketplace),
		nfts:         make(map[string]map[string]*NFT),
		transfers:    make(map[string][]Transfer),
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

// FillListing resolves the real fulfillment transaction for a listing. The
// returned transaction is unsigned; the buyer's wallet signs + broadcasts it.
func (s *Service) FillListing(ctx context.Context, marketplace, listingID, buyer string) (*RawTransaction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	mp, ok := s.marketplaces[marketplace]
	if !ok {
		return nil, fmt.Errorf("marketplace not found: %s", marketplace)
	}

	return mp.FillListing(ctx, listingID, buyer)
}

// ListCollections enumerates collections on a chain via the OpenSea
// marketplace (the only registered marketplace with a directory endpoint).
func (s *Service) ListCollections(ctx context.Context, chain string, limit int) ([]Collection, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, mp := range s.marketplaces {
		if os, ok := mp.(*OpenSeaMarketplace); ok {
			return os.ListCollections(ctx, chain, limit)
		}
	}
	return nil, "", fmt.Errorf("no marketplace with a collection directory is registered")
}

// GetNFTsByOwner enumerates NFTs owned by an address via OpenSea.
func (s *Service) GetNFTsByOwner(ctx context.Context, chain, address string, limit int) ([]NFT, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, mp := range s.marketplaces {
		if os, ok := mp.(*OpenSeaMarketplace); ok {
			return os.GetNFTsByOwner(ctx, chain, address, limit)
		}
	}
	return nil, "", fmt.Errorf("no marketplace with an owner-inventory endpoint is registered")
}

// ============================================================================
// Ordinals (Bitcoin NFT) Support
// ============================================================================

// OrdinalsService provides ordinals support
type OrdinalsService struct {
	indexerURL string
	client     *http.Client
}

// NewOrdinalsService creates ordinals service
func NewOrdinalsService(indexerURL string) *OrdinalsService {
	return &OrdinalsService{
		indexerURL: indexerURL,
		client:     &http.Client{Timeout: 30 * time.Second},
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
	Address     string    `json:"address"`
	ContentType string    `json:"content_type"`
	Content     string    `json:"content"`
	Metadata    string    `json:"metadata"`
	Timestamp   time.Time `json:"timestamp"`
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

	tx, err := s.FillListing(r.Context(), req.Marketplace, req.ListingID, req.Buyer)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": "fulfillment_ready", "transaction": tx})
}

// Serve starts NFT service
func (s *Service) Serve(addr string) error {
	http.HandleFunc("/v1/collection", s.HandleGetCollection)
	http.HandleFunc("/v1/listings", s.HandleGetListings)
	http.HandleFunc("/v1/listings/create", s.HandleCreateListing)
	http.HandleFunc("/v1/listings/fill", s.HandleFillListing)

	return http.ListenAndServe(addr, nil)
}
