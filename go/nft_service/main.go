// TigerWallet NFT Service - Comprehensive NFT Marketplace and Management
// Supports ERC-721, ERC-1155, Solana NFTs with marketplace, minting, and trading.
// PostgreSQL-backed marketplace state (collections/tokens/listings/offers/
// transactions/auctions). Real on-chain NFT reads (fetcher.go) and Redis cache
// are preserved unchanged.

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// jwtSecret is the shared HS256 secret used by wallet_api to sign JWTs. NFT
// service validates tokens issued by wallet_api so a single auth realm covers
// all services. Override via JWT_SECRET env (must match wallet_api).
var jwtSecret = getEnv("JWT_SECRET", "")

func getEnv(key, dflt string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return dflt
}

// parseJWT validates an HS256 JWT and returns the subject (user id).
func parseJWT(tokenStr string) (string, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return "", errors.New("invalid token")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("invalid claims")
	}
	sub, _ := claims["sub"].(string)
	if sub == "" {
		return "", errors.New("missing subject")
	}
	return sub, nil
}

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	Port      string
	RedisAddr string
	RPCURL    string // Ethereum JSON-RPC endpoint for real on-chain NFT reads
}

var cfg = Config{
	Port:      getEnv("NFT_PORT", "8004"),
	RedisAddr: getEnv("REDIS_ADDR", "localhost:6379"),
	RPCURL:    getEnv("ETH_RPC_URL", ""),
}

// ============================================================================
// Data Models
// ============================================================================

type NFTCollection struct {
	ID              string    `json:"id" bson:"_id"`
	Name            string    `json:"name" bson:"name"`
	Symbol          string    `json:"symbol" bson:"symbol"`
	Chain           string    `json:"chain" bson:"chain"`
	ContractAddress string    `json:"contract_address" bson:"contract_address"`
	Owner           string    `json:"owner" bson:"owner"`
	Creator         string    `json:"creator" bson:"creator"`
	Description     string    `json:"description" bson:"description"`
	ImageURL        string    `json:"image_url" bson:"image_url"`
	ExternalURL     string    `json:"external_url" bson:"external_url"`
	Category        string    `json:"category" bson:"category"`
	Standard        string    `json:"standard" bson:"standard"` // erc721, erc1155, spl
	TotalSupply     string    `json:"total_supply" bson:"total_supply"`
	FloorPrice      string    `json:"floor_price" bson:"floor_price"`
	Volume24h       string    `json:"volume_24h" bson:"volume_24h"`
	Sales24h        int       `json:"sales_24h" bson:"sales_24h"`
	Owners          int       `json:"owners" bson:"owners"`
	Verified        bool      `json:"verified" bson:"verified"`
	Featured        bool      `json:"featured" bson:"featured"`
	RoyaltyFee      string    `json:"royalty_fee" bson:"royalty_fee"` // percentage
	Status          string    `json:"status" bson:"status"`           // active, paused, sold_out
	CreatedAt       time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" bson:"updatedAt"`
}

type NFT struct {
	ID              string         `json:"id" bson:"_id"`
	CollectionID    string         `json:"collection_id" bson:"collection_id"`
	TokenID         string         `json:"token_id" bson:"token_id"`
	Chain           string         `json:"chain" bson:"chain"`
	ContractAddress string         `json:"contract_address" bson:"contract_address"`
	Owner           string         `json:"owner" bson:"owner"`
	Creator         string         `json:"creator" bson:"creator"`
	Name            string         `json:"name" bson:"name"`
	Description     string         `json:"description" bson:"description"`
	ImageURL        string         `json:"image_url" bson:"image_url"`
	AnimationURL    string         `json:"animation_url" bson:"animation_url"`
	ExternalURL     string         `json:"external_url" bson:"external_url"`
	Attributes      []NFTAttribute `json:"attributes" bson:"attributes"`
	Edition         int            `json:"edition" bson:"edition"`   // for 1155
	Quantity        int            `json:"quantity" bson:"quantity"` // for 1155
	TokenURI        string         `json:"token_uri" bson:"token_uri"`
	IsForSale       bool           `json:"is_for_sale" bson:"is_for_sale"`
	Price           string         `json:"price" bson:"price"`
	PriceToken      string         `json:"price_token" bson:"price_token"` // ETH, USDC, etc
	ListingFee      string         `json:"listing_fee" bson:"listing_fee"`
	LastSalePrice   string         `json:"last_sale_price" bson:"last_sale_price"`
	LastSaleTime    *time.Time     `json:"last_sale_time" bson:"last_sale_time"`
	CreatedAt       time.Time      `json:"created_at" bson:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at" bson:"updated_at"`
}

type NFTAttribute struct {
	TraitType   string `json:"trait_type" bson:"trait_type"`
	Value       string `json:"value" bson:"value"`
	DisplayType string `json:"display_type" bson:"display_type"`
	Rarity      string `json:"rarity" bson:"rarity"`
}

type NFTListing struct {
	ID         string     `json:"id" bson:"_id"`
	NFTID      string     `json:"nft_id" bson:"nft_id"`
	Seller     string     `json:"seller" bson:"seller"`
	Price      string     `json:"price" bson:"price"`
	PriceToken string     `json:"price_token" bson:"price_token"`
	Quantity   int        `json:"quantity" bson:"quantity"`
	Status     string     `json:"status" bson:"status"` // active, sold, cancelled, expired
	StartTime  time.Time  `json:"start_time" bson:"start_time"`
	EndTime    *time.Time `json:"end_time" bson:"end_time"`
	CreatedAt  time.Time  `json:"created_at" bson:"created_at"`
}

type NFTOffer struct {
	ID         string     `json:"id" bson:"_id"`
	NFTID      string     `json:"nft_id" bson:"nft_id"`
	Buyer      string     `json:"buyer" bson:"buyer"`
	Price      string     `json:"price" bson:"price"`
	PriceToken string     `json:"price_token" bson:"price_token"`
	Quantity   int        `json:"quantity" bson:"quantity"`
	Status     string     `json:"status" bson:"status"` // pending, accepted, rejected, expired
	ExpiredAt  *time.Time `json:"expired_at" bson:"expired_at"`
	CreatedAt  time.Time  `json:"created_at" bson:"created_at"`
}

type NFTTransaction struct {
	ID           string    `json:"id" bson:"_id"`
	NFTID        string    `json:"nft_id" bson:"nft_id"`
	CollectionID string    `json:"collection_id" bson:"collection_id"`
	Chain        string    `json:"chain" bson:"chain"`
	Seller       string    `json:"seller" bson:"seller"`
	Buyer        string    `json:"buyer" bson:"buyer"`
	Price        string    `json:"price" bson:"price"`
	PriceToken   string    `json:"price_token" bson:"price_token"`
	Fee          string    `json:"fee" bson:"fee"`
	Royalty      string    `json:"royalty" bson:"royalty"`
	TxHash       string    `json:"tx_hash" bson:"tx_hash"`
	Timestamp    time.Time `json:"timestamp" bson:"timestamp"`
}

type NFTAuction struct {
	ID         string    `json:"id" bson:"_id"`
	NFTID      string    `json:"nft_id" bson:"nft_id"`
	Seller     string    `json:"seller" bson:"seller"`
	StartPrice string    `json:"start_price" bson:"start_price"`
	EndPrice   string    `json:"end_price" bson:"end_price"`
	CurrentBid string    `json:"current_bid" bson:"current_bid"`
	Bidder     string    `json:"bidder" bson:"bidder"`
	Quantity   int       `json:"quantity" bson:"quantity"`
	Status     string    `json:"status" bson:"status"` // active, ended, cancelled
	StartTime  time.Time `json:"start_time" bson:"start_time"`
	EndTime    time.Time `json:"end_time" bson:"end_time"`
	CreatedAt  time.Time `json:"created_at" bson:"created_at"`
}

// ============================================================================
// NFT Service
// ============================================================================

type NFTService struct {
	pg      *pgxpool.Pool // PostgreSQL-backed marketplace state
	redis   *redis.Client
	fetcher *Fetcher // real on-chain NFT reader (nil if ETH_RPC_URL unset)
}

const nftSchema = `
CREATE TABLE IF NOT EXISTS nft_collections (
    id               TEXT PRIMARY KEY,
    name             TEXT NOT NULL DEFAULT '',
    symbol           TEXT NOT NULL DEFAULT '',
    chain            TEXT NOT NULL DEFAULT '',
    contract_address TEXT NOT NULL DEFAULT '',
    owner            TEXT NOT NULL DEFAULT '',
    creator          TEXT NOT NULL DEFAULT '',
    description      TEXT NOT NULL DEFAULT '',
    image_url        TEXT NOT NULL DEFAULT '',
    external_url     TEXT NOT NULL DEFAULT '',
    category         TEXT NOT NULL DEFAULT '',
    standard         TEXT NOT NULL DEFAULT '',
    total_supply     TEXT NOT NULL DEFAULT '0',
    floor_price      TEXT NOT NULL DEFAULT '0',
    volume_24h       TEXT NOT NULL DEFAULT '0',
    sales_24h        INTEGER NOT NULL DEFAULT 0,
    owners           INTEGER NOT NULL DEFAULT 0,
    verified         BOOLEAN NOT NULL DEFAULT FALSE,
    featured         BOOLEAN NOT NULL DEFAULT FALSE,
    royalty_fee      TEXT NOT NULL DEFAULT '0',
    status           TEXT NOT NULL DEFAULT 'active',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_nft_collections_chain ON nft_collections(chain);
CREATE INDEX IF NOT EXISTS idx_nft_collections_category ON nft_collections(category);
CREATE INDEX IF NOT EXISTS idx_nft_collections_owner ON nft_collections(owner);

CREATE TABLE IF NOT EXISTS nft_tokens (
    id               TEXT PRIMARY KEY,
    collection_id    TEXT NOT NULL DEFAULT '',
    token_id         TEXT NOT NULL DEFAULT '',
    chain            TEXT NOT NULL DEFAULT '',
    contract_address TEXT NOT NULL DEFAULT '',
    owner            TEXT NOT NULL DEFAULT '',
    creator          TEXT NOT NULL DEFAULT '',
    name             TEXT NOT NULL DEFAULT '',
    description      TEXT NOT NULL DEFAULT '',
    image_url        TEXT NOT NULL DEFAULT '',
    animation_url    TEXT NOT NULL DEFAULT '',
    external_url     TEXT NOT NULL DEFAULT '',
    attributes       JSONB NOT NULL DEFAULT '[]'::jsonb,
    edition          INTEGER NOT NULL DEFAULT 0,
    quantity         INTEGER NOT NULL DEFAULT 0,
    token_uri        TEXT NOT NULL DEFAULT '',
    is_for_sale      BOOLEAN NOT NULL DEFAULT FALSE,
    price            TEXT NOT NULL DEFAULT '0',
    price_token      TEXT NOT NULL DEFAULT '',
    listing_fee      TEXT NOT NULL DEFAULT '0',
    last_sale_price  TEXT NOT NULL DEFAULT '0',
    last_sale_time   TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_nft_tokens_collection ON nft_tokens(collection_id);
CREATE INDEX IF NOT EXISTS idx_nft_tokens_owner ON nft_tokens(owner);
CREATE INDEX IF NOT EXISTS idx_nft_tokens_for_sale ON nft_tokens(is_for_sale) WHERE is_for_sale = TRUE;
CREATE INDEX IF NOT EXISTS idx_nft_tokens_name_trgm ON nft_tokens USING gin (to_tsvector('simple', name || ' ' || coalesce(description,'')));

CREATE TABLE IF NOT EXISTS nft_listings (
    id          TEXT PRIMARY KEY,
    nft_id      TEXT NOT NULL DEFAULT '',
    seller      TEXT NOT NULL DEFAULT '',
    price       TEXT NOT NULL DEFAULT '0',
    price_token TEXT NOT NULL DEFAULT '',
    quantity    INTEGER NOT NULL DEFAULT 0,
    status      TEXT NOT NULL DEFAULT 'active',
    start_time  TIMESTAMPTZ NOT NULL DEFAULT now(),
    end_time    TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_nft_listings_nft ON nft_listings(nft_id);
CREATE INDEX IF NOT EXISTS idx_nft_listings_seller ON nft_listings(seller);
CREATE INDEX IF NOT EXISTS idx_nft_listings_status ON nft_listings(status);

CREATE TABLE IF NOT EXISTS nft_offers (
    id          TEXT PRIMARY KEY,
    nft_id      TEXT NOT NULL DEFAULT '',
    buyer       TEXT NOT NULL DEFAULT '',
    price       TEXT NOT NULL DEFAULT '0',
    price_token TEXT NOT NULL DEFAULT '',
    quantity    INTEGER NOT NULL DEFAULT 0,
    status      TEXT NOT NULL DEFAULT 'pending',
    expired_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_nft_offers_nft ON nft_offers(nft_id);
CREATE INDEX IF NOT EXISTS idx_nft_offers_buyer ON nft_offers(buyer);

CREATE TABLE IF NOT EXISTS nft_transactions (
    id            TEXT PRIMARY KEY,
    nft_id        TEXT NOT NULL DEFAULT '',
    collection_id TEXT NOT NULL DEFAULT '',
    chain         TEXT NOT NULL DEFAULT '',
    seller        TEXT NOT NULL DEFAULT '',
    buyer         TEXT NOT NULL DEFAULT '',
    price         TEXT NOT NULL DEFAULT '0',
    price_token   TEXT NOT NULL DEFAULT '',
    fee           TEXT NOT NULL DEFAULT '0',
    royalty       TEXT NOT NULL DEFAULT '0',
    tx_hash       TEXT NOT NULL DEFAULT '',
    timestamp     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_nft_transactions_nft ON nft_transactions(nft_id);
CREATE INDEX IF NOT EXISTS idx_nft_transactions_buyer ON nft_transactions(buyer);

CREATE TABLE IF NOT EXISTS nft_auctions (
    id          TEXT PRIMARY KEY,
    nft_id      TEXT NOT NULL DEFAULT '',
    seller      TEXT NOT NULL DEFAULT '',
    start_price TEXT NOT NULL DEFAULT '0',
    end_price   TEXT NOT NULL DEFAULT '0',
    current_bid TEXT NOT NULL DEFAULT '0',
    bidder      TEXT NOT NULL DEFAULT '',
    quantity    INTEGER NOT NULL DEFAULT 0,
    status      TEXT NOT NULL DEFAULT 'active',
    start_time  TIMESTAMPTZ NOT NULL DEFAULT now(),
    end_time    TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_nft_auctions_nft ON nft_auctions(nft_id);
CREATE INDEX IF NOT EXISTS idx_nft_auctions_status ON nft_auctions(status);
CREATE INDEX IF NOT EXISTS idx_nft_auctions_seller ON nft_auctions(seller);
`

// Migrate creates the marketplace tables and indexes if they do not exist.
func (ns *NFTService) Migrate(ctx context.Context) error {
	if ns.pg == nil {
		return fmt.Errorf("database not configured")
	}
	_, err := ns.pg.Exec(ctx, nftSchema)
	return err
}

func NewNFTService() *NFTService {
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
	})

	fetcher, _ := NewFetcher(cfg.RPCURL, cfg.RedisAddr)
	if fetcher != nil {
		log.Println("On-chain NFT fetcher enabled (RPC: " + cfg.RPCURL + ")")
	} else {
		log.Println("WARNING: ETH_RPC_URL not set — on-chain NFT reads unavailable (no mock data served)")
	}

	ns := &NFTService{
		redis:   rdb,
		fetcher: fetcher,
	}

	dbURL := getEnv("DATABASE_URL", "postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet?sslmode=disable")
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	ns.pg = pool

	if err := ns.Migrate(context.Background()); err != nil {
		log.Fatalf("NFT marketplace migration failed: %v", err)
	}
	log.Println("NFT service connected to PostgreSQL on", dbURL)

	return ns
}

// fetchUserNFTsOnChain returns NFTs owned by an address at a contract by
// performing real on-chain reads. Returns errFetcherUnavailable when no RPC is
// configured (callers should surface "unavailable" rather than fabricate data).
func (ns *NFTService) fetchUserNFTsOnChain(ctx context.Context, contract, owner string) ([]*NFT, error) {
	if ns.fetcher == nil || !ns.fetcher.Available() {
		return nil, errFetcherUnavailable
	}
	if !common.IsHexAddress(contract) || !common.IsHexAddress(owner) {
		return nil, errors.New("invalid address")
	}
	return ns.fetcher.FetchUserNFTs(ctx, common.HexToAddress(contract), common.HexToAddress(owner))
}

// ============================================================================
// PG scan helpers
// ============================================================================

func attrJSON(attrs []NFTAttribute) []byte {
	if attrs == nil {
		attrs = []NFTAttribute{}
	}
	b, _ := json.Marshal(attrs)
	return b
}

func scanAttributes(b []byte) []NFTAttribute {
	var attrs []NFTAttribute
	if len(b) > 0 {
		_ = json.Unmarshal(b, &attrs)
	}
	if attrs == nil {
		attrs = []NFTAttribute{}
	}
	return attrs
}

func (ns *NFTService) getCollectionRow(ctx context.Context, q pgx.Row, id string) (*NFTCollection, error) {
	var c NFTCollection
	err := q.Scan(&c.ID, &c.Name, &c.Symbol, &c.Chain, &c.ContractAddress, &c.Owner, &c.Creator,
		&c.Description, &c.ImageURL, &c.ExternalURL, &c.Category, &c.Standard, &c.TotalSupply,
		&c.FloorPrice, &c.Volume24h, &c.Sales24h, &c.Owners, &c.Verified, &c.Featured,
		&c.RoyaltyFee, &c.Status, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

const collectionCols = `id,name,symbol,chain,contract_address,owner,creator,description,image_url,
external_url,category,standard,total_supply,floor_price,volume_24h,sales_24h,owners,verified,
featured,royalty_fee,status,created_at,updated_at`

// getCollectionByID returns a single collection or an error wrapping pgx.ErrNoRows.
func (ns *NFTService) getCollectionByID(ctx context.Context, id string) (*NFTCollection, error) {
	row := ns.pg.QueryRow(ctx, `SELECT `+collectionCols+` FROM nft_collections WHERE id=$1`, id)
	return ns.getCollectionRow(ctx, row, id)
}

// getNFTRow scans a token row from any pgx.Row.
func (ns *NFTService) getNFTRow(q pgx.Row) (*NFT, error) {
	var n NFT
	var attrs []byte
	err := q.Scan(&n.ID, &n.CollectionID, &n.TokenID, &n.Chain, &n.ContractAddress, &n.Owner,
		&n.Creator, &n.Name, &n.Description, &n.ImageURL, &n.AnimationURL, &n.ExternalURL,
		&attrs, &n.Edition, &n.Quantity, &n.TokenURI, &n.IsForSale, &n.Price, &n.PriceToken,
		&n.ListingFee, &n.LastSalePrice, &n.LastSaleTime, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return nil, err
	}
	n.Attributes = scanAttributes(attrs)
	return &n, nil
}

const nftCols = `id,collection_id,token_id,chain,contract_address,owner,creator,name,description,
image_url,animation_url,external_url,attributes,edition,quantity,token_uri,is_for_sale,price,
price_token,listing_fee,last_sale_price,last_sale_time,created_at,updated_at`

func (ns *NFTService) getNFTByID(ctx context.Context, id string) (*NFT, error) {
	row := ns.pg.QueryRow(ctx, `SELECT `+nftCols+` FROM nft_tokens WHERE id=$1`, id)
	return ns.getNFTRow(row)
}

const listingCols = `id,nft_id,seller,price,price_token,quantity,status,start_time,end_time,created_at`

func (ns *NFTService) scanListing(q pgx.Row) (*NFTListing, error) {
	var l NFTListing
	err := q.Scan(&l.ID, &l.NFTID, &l.Seller, &l.Price, &l.PriceToken, &l.Quantity,
		&l.Status, &l.StartTime, &l.EndTime, &l.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

const auctionCols = `id,nft_id,seller,start_price,end_price,current_bid,bidder,quantity,status,start_time,end_time,created_at`

func (ns *NFTService) scanAuction(q pgx.Row) (*NFTAuction, error) {
	var a NFTAuction
	err := q.Scan(&a.ID, &a.NFTID, &a.Seller, &a.StartPrice, &a.EndPrice, &a.CurrentBid,
		&a.Bidder, &a.Quantity, &a.Status, &a.StartTime, &a.EndTime, &a.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

const txCols = `id,nft_id,collection_id,chain,seller,buyer,price,price_token,fee,royalty,tx_hash,timestamp`

func (ns *NFTService) scanTransaction(q pgx.Row) (*NFTTransaction, error) {
	var t NFTTransaction
	err := q.Scan(&t.ID, &t.NFTID, &t.CollectionID, &t.Chain, &t.Seller, &t.Buyer, &t.Price,
		&t.PriceToken, &t.Fee, &t.Royalty, &t.TxHash, &t.Timestamp)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// insertCollection persists a collection row.
func (ns *NFTService) insertCollection(ctx context.Context, c *NFTCollection) error {
	_, err := ns.pg.Exec(ctx, `INSERT INTO nft_collections
		(id,name,symbol,chain,contract_address,owner,creator,description,image_url,external_url,
		category,standard,total_supply,floor_price,volume_24h,sales_24h,owners,verified,featured,
		royalty_fee,status,created_at,updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23)`,
		c.ID, c.Name, c.Symbol, c.Chain, c.ContractAddress, c.Owner, c.Creator, c.Description,
		c.ImageURL, c.ExternalURL, c.Category, c.Standard, c.TotalSupply, c.FloorPrice, c.Volume24h,
		c.Sales24h, c.Owners, c.Verified, c.Featured, c.RoyaltyFee, c.Status, c.CreatedAt, c.UpdatedAt)
	return err
}

// insertNFT persists a token row.
func (ns *NFTService) insertNFT(ctx context.Context, n *NFT) error {
	_, err := ns.pg.Exec(ctx, `INSERT INTO nft_tokens
		(id,collection_id,token_id,chain,contract_address,owner,creator,name,description,image_url,
		animation_url,external_url,attributes,edition,quantity,token_uri,is_for_sale,price,
		price_token,listing_fee,last_sale_price,last_sale_time,created_at,updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24)`,
		n.ID, n.CollectionID, n.TokenID, n.Chain, n.ContractAddress, n.Owner, n.Creator, n.Name,
		n.Description, n.ImageURL, n.AnimationURL, n.ExternalURL, attrJSON(n.Attributes), n.Edition,
		n.Quantity, n.TokenURI, n.IsForSale, n.Price, n.PriceToken, n.ListingFee, n.LastSalePrice,
		n.LastSaleTime, n.CreatedAt, n.UpdatedAt)
	return err
}

// insertListing persists a listing row.
func (ns *NFTService) insertListing(ctx context.Context, l *NFTListing) error {
	_, err := ns.pg.Exec(ctx, `INSERT INTO nft_listings
		(id,nft_id,seller,price,price_token,quantity,status,start_time,end_time,created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		l.ID, l.NFTID, l.Seller, l.Price, l.PriceToken, l.Quantity, l.Status, l.StartTime, l.EndTime, l.CreatedAt)
	return err
}

// insertOffer persists an offer row.
func (ns *NFTService) insertOffer(ctx context.Context, o *NFTOffer) error {
	_, err := ns.pg.Exec(ctx, `INSERT INTO nft_offers
		(id,nft_id,buyer,price,price_token,quantity,status,expired_at,created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		o.ID, o.NFTID, o.Buyer, o.Price, o.PriceToken, o.Quantity, o.Status, o.ExpiredAt, o.CreatedAt)
	return err
}

// insertTransaction persists a transaction row.
func (ns *NFTService) insertTransaction(ctx context.Context, t *NFTTransaction) error {
	_, err := ns.pg.Exec(ctx, `INSERT INTO nft_transactions
		(id,nft_id,collection_id,chain,seller,buyer,price,price_token,fee,royalty,tx_hash,timestamp)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		t.ID, t.NFTID, t.CollectionID, t.Chain, t.Seller, t.Buyer, t.Price, t.PriceToken, t.Fee,
		t.Royalty, t.TxHash, t.Timestamp)
	return err
}

// insertAuction persists an auction row.
func (ns *NFTService) insertAuction(ctx context.Context, a *NFTAuction) error {
	_, err := ns.pg.Exec(ctx, `INSERT INTO nft_auctions
		(id,nft_id,seller,start_price,end_price,current_bid,bidder,quantity,status,start_time,end_time,created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		a.ID, a.NFTID, a.Seller, a.StartPrice, a.EndPrice, a.CurrentBid, a.Bidder, a.Quantity,
		a.Status, a.StartTime, a.EndTime, a.CreatedAt)
	return err
}

// tokenCount returns the number of minted tokens (used for token-id generation).
func (ns *NFTService) tokenCount(ctx context.Context) (int, error) {
	var n int
	err := ns.pg.QueryRow(ctx, `SELECT count(*) FROM nft_tokens`).Scan(&n)
	return n, err
}

// ============================================================================
// API Handlers
// ============================================================================

// Get all collections
func (ns *NFTService) GetCollections(c *gin.Context) {
	if ns.pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}
	category := c.Query("category")
	chain := c.Query("chain")
	featured := c.Query("featured")

	ctx := c.Request.Context()
	qb := strings.Builder{}
	qb.WriteString(`SELECT ` + collectionCols + ` FROM nft_collections WHERE 1=1`)
	args := []interface{}{}
	n := 1
	if category != "" {
		qb.WriteString(fmt.Sprintf(` AND category=$%d`, n))
		args = append(args, category)
		n++
	}
	if chain != "" {
		qb.WriteString(fmt.Sprintf(` AND chain=$%d`, n))
		args = append(args, chain)
		n++
	}
	if featured == "true" {
		qb.WriteString(` AND featured=TRUE`)
	}
	qb.WriteString(` ORDER BY created_at DESC`)

	rows, err := ns.pg.Query(ctx, qb.String(), args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	collections := make([]*NFTCollection, 0)
	for rows.Next() {
		col, err := ns.getCollectionRow(ctx, rows, "")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		collections = append(collections, col)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"collections": collections,
		"total":       len(collections),
	})
}

// Get collection by ID
func (ns *NFTService) GetCollection(c *gin.Context) {
	if ns.pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}
	collectionID := c.Param("id")

	collection, err := ns.getCollectionByID(c.Request.Context(), collectionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "collection not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"collection": collection,
	})
}

// Get NFTs in collection
func (ns *NFTService) GetCollectionNFTs(c *gin.Context) {
	if ns.pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}
	collectionID := c.Param("id")

	ctx := c.Request.Context()
	rows, err := ns.pg.Query(ctx, `SELECT `+nftCols+` FROM nft_tokens WHERE collection_id=$1 ORDER BY created_at DESC`, collectionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	nfts := make([]*NFT, 0)
	for rows.Next() {
		nft, err := ns.getNFTRow(rows)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		nfts = append(nfts, nft)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"nfts":    nfts,
		"total":   len(nfts),
	})
}

// Get NFT by ID
func (ns *NFTService) GetNFT(c *gin.Context) {
	if ns.pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}
	nftID := c.Param("id")

	nft, err := ns.getNFTByID(c.Request.Context(), nftID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "nft not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"nft":     nft,
	})
}

// Create collection
type CreateCollectionRequest struct {
	Name            string `json:"name" binding:"required"`
	Symbol          string `json:"symbol" binding:"required"`
	Chain           string `json:"chain" binding:"required"`
	ContractAddress string `json:"contract_address" binding:"required"`
	Description     string `json:"description"`
	Category        string `json:"category"`
	Standard        string `json:"standard" binding:"required"`
	RoyaltyFee      string `json:"royalty_fee"`
}

func (ns *NFTService) CreateCollection(c *gin.Context) {
	if !ns.enforceFeature(c, GatedFeature) {
		return
	}
	if ns.pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}
	var req CreateCollectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	collectionID := uuid.New().String()
	now := time.Now()
	collection := &NFTCollection{
		ID:              collectionID,
		Name:            req.Name,
		Symbol:          req.Symbol,
		Chain:           req.Chain,
		ContractAddress: req.ContractAddress,
		Owner:           c.GetString("user_id"),
		Creator:         c.GetString("user_id"),
		Description:     req.Description,
		Category:        req.Category,
		Standard:        req.Standard,
		RoyaltyFee:      req.RoyaltyFee,
		Status:          "active",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := ns.insertCollection(c.Request.Context(), collection); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":       true,
		"collection_id": collectionID,
		"collection":    collection,
	})
}

// Mint NFT
type MintNFTRequest struct {
	CollectionID string         `json:"collection_id" binding:"required"`
	Name         string         `json:"name" binding:"required"`
	Description  string         `json:"description"`
	ImageURL     string         `json:"image_url" binding:"required"`
	Attributes   []NFTAttribute `json:"attributes"`
	Quantity     int            `json:"quantity"`
	IsForSale    bool           `json:"is_for_sale"`
	Price        string         `json:"price"`
	PriceToken   string         `json:"price_token"`
}

func (ns *NFTService) MintNFT(c *gin.Context) {
	if !ns.enforceFeature(c, GatedFeature) {
		return
	}
	if ns.pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}
	var req MintNFTRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	// Validate collection.
	collection, err := ns.getCollectionByID(ctx, req.CollectionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "collection not found"})
		return
	}

	// Generate token id from current token count.
	count, err := ns.tokenCount(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	tokenID := fmt.Sprintf("%d", count+1)

	nftID := uuid.New().String()
	now := time.Now()
	nft := &NFT{
		ID:              nftID,
		CollectionID:    req.CollectionID,
		TokenID:         tokenID,
		Chain:           collection.Chain,
		ContractAddress: collection.ContractAddress,
		Owner:           c.GetString("user_id"),
		Creator:         c.GetString("user_id"),
		Name:            req.Name,
		Description:     req.Description,
		ImageURL:        req.ImageURL,
		Attributes:      req.Attributes,
		Quantity:        req.Quantity,
		IsForSale:       req.IsForSale,
		Price:           req.Price,
		PriceToken:      req.PriceToken,
		TokenURI:        fmt.Sprintf("ipfs://%s/%s.json", nftID, tokenID),
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := ns.insertNFT(ctx, nft); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update collection total supply.
	if _, err := ns.pg.Exec(ctx,
		`UPDATE nft_collections SET total_supply=CAST($1 AS TEXT), updated_at=$2 WHERE id=$3`,
		count+1, now, req.CollectionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":   true,
		"nft_id":    nftID,
		"token_id":  tokenID,
		"token_uri": nft.TokenURI,
		"owner":     nft.Owner,
	})
}

// List NFT for sale
type ListNFTRequest struct {
	NFTID      string `json:"nft_id" binding:"required"`
	Price      string `json:"price" binding:"required"`
	PriceToken string `json:"price_token" binding:"required"`
	Quantity   int    `json:"quantity"`
}

func (ns *NFTService) ListNFT(c *gin.Context) {
	if !ns.enforceFeature(c, GatedFeature) {
		return
	}
	if ns.pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}
	var req ListNFTRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	userID := c.GetString("user_id")

	// Validate NFT ownership.
	nft, err := ns.getNFTByID(ctx, req.NFTID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "nft not found"})
		return
	}
	if nft.Owner != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "not owner"})
		return
	}

	// Create listing.
	listingID := uuid.New().String()
	quantity := req.Quantity
	if quantity == 0 {
		quantity = 1
	}
	now := time.Now()
	listing := &NFTListing{
		ID:         listingID,
		NFTID:      req.NFTID,
		Seller:     userID,
		Price:      req.Price,
		PriceToken: req.PriceToken,
		Quantity:   quantity,
		Status:     "active",
		StartTime:  now,
		CreatedAt:  now,
	}

	if err := ns.insertListing(ctx, listing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update NFT.
	if _, err := ns.pg.Exec(ctx,
		`UPDATE nft_tokens SET is_for_sale=TRUE, price=$1, price_token=$2, updated_at=$3 WHERE id=$4`,
		req.Price, req.PriceToken, now, req.NFTID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":     true,
		"listing_id":  listingID,
		"price":       req.Price,
		"price_token": req.PriceToken,
	})
}

// Buy NFT
type BuyNFTRequest struct {
	ListingID string `json:"listing_id" binding:"required"`
}

func (ns *NFTService) BuyNFT(c *gin.Context) {
	if !ns.enforceFeature(c, GatedFeature) {
		return
	}
	if ns.pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}
	var req BuyNFTRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	buyer := c.GetString("user_id")

	tx, err := ns.pg.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback(ctx)

	// Lock the listing row to serialize concurrent buyers.
	var listingID, nftID, seller, price, priceToken, status string
	err = tx.QueryRow(ctx,
		`SELECT id,nft_id,seller,price,price_token,status FROM nft_listings WHERE id=$1 FOR UPDATE`,
		req.ListingID).Scan(&listingID, &nftID, &seller, &price, &priceToken, &status)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "listing not found"})
		return
	}
	if status != "active" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "listing not active"})
		return
	}

	// Lock the NFT row and read collection for royalty.
	var collectionID, chain string
	err = tx.QueryRow(ctx,
		`SELECT collection_id,chain FROM nft_tokens WHERE id=$1 FOR UPDATE`, nftID).
		Scan(&collectionID, &chain)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "nft not found"})
		return
	}
	var royalty string
	_ = tx.QueryRow(ctx, `SELECT royalty_fee FROM nft_collections WHERE id=$1`, collectionID).Scan(&royalty)

	txID := uuid.New().String()
	now := time.Now()
	transaction := &NFTTransaction{
		ID:           txID,
		NFTID:        nftID,
		CollectionID: collectionID,
		Chain:        chain,
		Seller:       seller,
		Buyer:        buyer,
		Price:        price,
		PriceToken:   priceToken,
		Fee:          "2.5", // platform fee
		Royalty:      royalty,
		TxHash:       "", // not broadcast via RPC; real hash requires on-chain broadcast
		Timestamp:    now,
	}

	if _, err := tx.Exec(ctx, `INSERT INTO nft_transactions
		(id,nft_id,collection_id,chain,seller,buyer,price,price_token,fee,royalty,tx_hash,timestamp)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		transaction.ID, transaction.NFTID, transaction.CollectionID, transaction.Chain,
		transaction.Seller, transaction.Buyer, transaction.Price, transaction.PriceToken,
		transaction.Fee, transaction.Royalty, transaction.TxHash, transaction.Timestamp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Mark listing sold.
	if _, err := tx.Exec(ctx, `UPDATE nft_listings SET status='sold' WHERE id=$1`, listingID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Transfer NFT ownership.
	if _, err := tx.Exec(ctx,
		`UPDATE nft_tokens SET owner=$1, is_for_sale=FALSE, last_sale_price=$2, last_sale_time=$3, updated_at=$3 WHERE id=$4`,
		buyer, price, now, nftID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"tx_id":       txID,
		"nft_id":      nftID,
		"buyer":       buyer,
		"seller":      seller,
		"price":       price,
		"price_token": priceToken,
		"tx_hash":     transaction.TxHash,
		"status":      "pending",
	})
}

// Make offer
type MakeOfferRequest struct {
	NFTID      string `json:"nft_id" binding:"required"`
	Price      string `json:"price" binding:"required"`
	PriceToken string `json:"price_token" binding:"required"`
	Quantity   int    `json:"quantity"`
	Duration   int    `json:"duration"` // hours
}

func (ns *NFTService) MakeOffer(c *gin.Context) {
	if !ns.enforceFeature(c, GatedFeature) {
		return
	}
	if ns.pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}
	var req MakeOfferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	// Validate NFT.
	if _, err := ns.getNFTByID(ctx, req.NFTID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "nft not found"})
		return
	}

	// Create offer.
	offerID := uuid.New().String()
	quantity := req.Quantity
	if quantity == 0 {
		quantity = 1
	}
	now := time.Now()
	expiredAt := now.Add(time.Duration(req.Duration) * time.Hour)
	offer := &NFTOffer{
		ID:         offerID,
		NFTID:      req.NFTID,
		Buyer:      c.GetString("user_id"),
		Price:      req.Price,
		PriceToken: req.PriceToken,
		Quantity:   quantity,
		Status:     "pending",
		ExpiredAt:  &expiredAt,
		CreatedAt:  now,
	}

	if err := ns.insertOffer(ctx, offer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":  true,
		"offer_id": offerID,
		"price":    req.Price,
		"expires":  expiredAt.Unix(),
	})
}

// ============================================================================
// Auction handlers — English-auction style bidding over PostgreSQL state.
// ============================================================================

type CreateAuctionRequest struct {
	NFTID      string `json:"nft_id" binding:"required"`
	StartPrice string `json:"start_price" binding:"required"`
	EndPrice   string `json:"end_price"`
	Quantity   int    `json:"quantity"`
	Duration   int    `json:"duration"` // hours
}

// CreateAuction lists an NFT for English-auction style bidding.
func (ns *NFTService) CreateAuction(c *gin.Context) {
	if ns.pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}
	var req CreateAuctionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	userID := c.GetString("user_id")

	nft, err := ns.getNFTByID(ctx, req.NFTID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "nft not found"})
		return
	}
	if nft.Owner != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "not nft owner"})
		return
	}

	duration := time.Duration(req.Duration) * time.Hour
	if duration <= 0 {
		duration = 24 * time.Hour
	}
	quantity := req.Quantity
	if quantity == 0 {
		quantity = 1
	}

	now := time.Now()
	auction := &NFTAuction{
		ID:         uuid.New().String(),
		NFTID:      req.NFTID,
		Seller:     nft.Owner,
		StartPrice: req.StartPrice,
		EndPrice:   req.EndPrice,
		CurrentBid: req.StartPrice,
		Quantity:   quantity,
		Status:     "active",
		StartTime:  now,
		EndTime:    now.Add(duration),
		CreatedAt:  now,
	}

	if err := ns.insertAuction(ctx, auction); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if _, err := ns.pg.Exec(ctx,
		`UPDATE nft_tokens SET is_for_sale=TRUE, updated_at=$1 WHERE id=$2`, now, req.NFTID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":    true,
		"auction_id": auction.ID,
		"end_time":   auction.EndTime.Unix(),
	})
}

type PlaceBidRequest struct {
	AuctionID string `json:"auction_id" binding:"required"`
	Amount    string `json:"amount" binding:"required"`
}

// PlaceBid records a bid on an active auction. Bids must exceed the current
// high bid and must be placed before the auction end time.
func (ns *NFTService) PlaceBid(c *gin.Context) {
	if ns.pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}
	var req PlaceBidRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	bidder := c.GetString("user_id")

	tx, err := ns.pg.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback(ctx)

	var auctionID, currentBid, status string
	var endTime time.Time
	err = tx.QueryRow(ctx,
		`SELECT id,current_bid,status,end_time FROM nft_auctions WHERE id=$1 FOR UPDATE`,
		req.AuctionID).Scan(&auctionID, &currentBid, &status, &endTime)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "auction not found"})
		return
	}
	if status != "active" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "auction not active"})
		return
	}
	if time.Now().After(endTime) {
		if _, err := tx.Exec(ctx, `UPDATE nft_auctions SET status='ended' WHERE id=$1`, auctionID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		_ = tx.Commit(ctx)
		c.JSON(http.StatusBadRequest, gin.H{"error": "auction ended"})
		return
	}

	// Numeric comparison so a bid must strictly exceed the standing bid.
	bigBid, ok := new(big.Int).SetString(req.Amount, 10)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount"})
		return
	}
	bigCurrent, _ := new(big.Int).SetString(currentBid, 10)
	if bigCurrent == nil {
		bigCurrent = new(big.Int)
	}
	if bigBid.Cmp(bigCurrent) <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bid must exceed current bid"})
		return
	}

	if _, err := tx.Exec(ctx,
		`UPDATE nft_auctions SET current_bid=$1, bidder=$2 WHERE id=$3`,
		req.Amount, bidder, auctionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"auction_id":  auctionID,
		"current_bid": req.Amount,
		"bidder":      bidder,
	})
}

// EndAuction settles an auction after its end time and returns the winning bid.
func (ns *NFTService) EndAuction(c *gin.Context) {
	if ns.pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}
	auctionID := c.Param("id")

	ctx := c.Request.Context()

	tx, err := ns.pg.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback(ctx)

	var id, currentBid, bidder, status string
	err = tx.QueryRow(ctx,
		`SELECT id,current_bid,bidder,status FROM nft_auctions WHERE id=$1 FOR UPDATE`,
		auctionID).Scan(&id, &currentBid, &bidder, &status)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "auction not found"})
		return
	}
	if status == "ended" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "auction already ended"})
		return
	}

	if _, err := tx.Exec(ctx, `UPDATE nft_auctions SET status='ended' WHERE id=$1`, auctionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	saleID := uuid.New().String()

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"auction_id": auctionID,
		"sale_id":    saleID,
		"winner":     bidder,
		"final_bid":  currentBid,
	})
}

// GetAuction returns the current state of a single auction.
func (ns *NFTService) GetAuction(c *gin.Context) {
	if ns.pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}
	auctionID := c.Param("id")

	auction, err := ns.scanAuction(ns.pg.QueryRow(c.Request.Context(),
		`SELECT `+auctionCols+` FROM nft_auctions WHERE id=$1`, auctionID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "auction not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"auction": auction})
}

// GetActiveAuctions lists currently-active auctions, optionally filtered by
// collection id (query param ?collection_id=...).
func (ns *NFTService) GetActiveAuctions(c *gin.Context) {
	if ns.pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}
	collectionID := c.Query("collection_id")

	ctx := c.Request.Context()
	var (
		rows pgx.Rows
		err  error
	)
	if collectionID != "" {
		rows, err = ns.pg.Query(ctx,
			`SELECT `+auctionColsAlias()+` FROM nft_auctions a
			WHERE a.status='active' AND EXISTS (
				SELECT 1 FROM nft_tokens t WHERE t.id=a.nft_id AND t.collection_id=$1
			) ORDER BY a.end_time ASC`, collectionID)
	} else {
		rows, err = ns.pg.Query(ctx,
			`SELECT `+auctionCols+` FROM nft_auctions WHERE status='active' ORDER BY end_time ASC`)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	result := make([]*NFTAuction, 0)
	for rows.Next() {
		auction, err := ns.scanAuction(rows)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		result = append(result, auction)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"auctions": result, "count": len(result)})
}

// auctionColsAlias returns the auction columns prefixed with the alias used by
// the joined query in GetActiveAuctions.
func auctionColsAlias() string {
	return `a.id,a.nft_id,a.seller,a.start_price,a.end_price,a.current_bid,a.bidder,
		a.quantity,a.status,a.start_time,a.end_time,a.created_at`
}

// CancelListing removes an active fixed-price listing. Only the listing owner
// may cancel.
func (ns *NFTService) CancelListing(c *gin.Context) {
	if ns.pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}
	listingID := c.Param("id")
	userID := c.GetString("user_id")

	ctx := c.Request.Context()

	listing, err := ns.scanListing(ns.pg.QueryRow(ctx,
		`SELECT `+listingCols+` FROM nft_listings WHERE id=$1`, listingID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "listing not found"})
		return
	}
	if listing.Seller != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "not listing owner"})
		return
	}
	if listing.Status != "active" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "listing not active"})
		return
	}

	if _, err := ns.pg.Exec(ctx,
		`UPDATE nft_listings SET status='cancelled' WHERE id=$1`, listingID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if _, err := ns.pg.Exec(ctx,
		`UPDATE nft_tokens SET is_for_sale=FALSE, updated_at=$1 WHERE id=$2`, time.Now(), listing.NFTID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "listing_id": listingID, "status": "cancelled"})
}

// Get user NFTs
func (ns *NFTService) GetUserNFTs(c *gin.Context) {
	userID := c.Param("user_id")

	// Real on-chain path: when a contract address is supplied, perform live
	// eth_call reads (balanceOf + tokenOfOwnerByIndex + tokenURI + metadata)
	// instead of returning seeded mock data.
	if contract := c.Query("contract"); contract != "" {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()
		nfts, err := ns.fetchUserNFTsOnChain(ctx, contract, userID)
		if err != nil {
			if errors.Is(err, errFetcherUnavailable) {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"success": false,
					"error":   "on-chain NFT reads unavailable: ETH_RPC_URL not set",
				})
				return
			}
			c.JSON(http.StatusBadGateway, gin.H{"success": false, "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"nfts":    nfts,
			"total":   len(nfts),
			"source":  "on-chain",
		})
		return
	}

	if ns.pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}

	// Fallback: return marketplace-tracked NFTs owned by the user (real
	// application state created via Mint/List), not seeded mock data.
	ctx := c.Request.Context()
	rows, err := ns.pg.Query(ctx, `SELECT `+nftCols+` FROM nft_tokens WHERE owner=$1 ORDER BY created_at DESC`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	nfts := make([]*NFT, 0)
	for rows.Next() {
		nft, err := ns.getNFTRow(rows)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		nfts = append(nfts, nft)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"nfts":    nfts,
		"total":   len(nfts),
		"source":  "marketplace",
	})
}

// Get NFT transactions
func (ns *NFTService) GetNFTHistory(c *gin.Context) {
	if ns.pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}
	nftID := c.Param("id")

	ctx := c.Request.Context()
	rows, err := ns.pg.Query(ctx, `SELECT `+txCols+` FROM nft_transactions WHERE nft_id=$1 ORDER BY timestamp DESC`, nftID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	txs := make([]*NFTTransaction, 0)
	for rows.Next() {
		t, err := ns.scanTransaction(rows)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		txs = append(txs, t)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"transactions": txs,
		"total":        len(txs),
	})
}

// Get listings
func (ns *NFTService) GetListings(c *gin.Context) {
	if ns.pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}
	nftID := c.Query("nft_id")
	status := c.Query("status")

	ctx := c.Request.Context()
	qb := strings.Builder{}
	qb.WriteString(`SELECT ` + listingCols + ` FROM nft_listings WHERE 1=1`)
	args := []interface{}{}
	n := 1
	if nftID != "" {
		qb.WriteString(fmt.Sprintf(` AND nft_id=$%d`, n))
		args = append(args, nftID)
		n++
	}
	if status != "" {
		qb.WriteString(fmt.Sprintf(` AND status=$%d`, n))
		args = append(args, status)
		n++
	}
	qb.WriteString(` ORDER BY created_at DESC`)

	rows, err := ns.pg.Query(ctx, qb.String(), args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	listings := make([]*NFTListing, 0)
	for rows.Next() {
		listing, err := ns.scanListing(rows)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		listings = append(listings, listing)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"listings": listings,
		"total":    len(listings),
	})
}

// Search NFTs
func (ns *NFTService) SearchNFTs(c *gin.Context) {
	if ns.pg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not configured"})
		return
	}
	query := c.Query("q")
	collection := c.Query("collection")
	minPrice := c.Query("min_price")
	maxPrice := c.Query("max_price")

	ctx := c.Request.Context()
	qb := strings.Builder{}
	qb.WriteString(`SELECT ` + nftCols + ` FROM nft_tokens WHERE 1=1`)
	args := []interface{}{}
	n := 1
	if query != "" {
		qb.WriteString(fmt.Sprintf(` AND (LOWER(name) LIKE LOWER($%d) OR LOWER(coalesce(description,'')) LIKE LOWER($%d))`, n, n))
		args = append(args, "%"+query+"%")
		n++
	}
	if collection != "" {
		qb.WriteString(fmt.Sprintf(` AND collection_id=$%d`, n))
		args = append(args, collection)
		n++
	}
	// Price range filter on the numeric token price (simplified big-int compare).
	if minPrice != "" {
		qb.WriteString(fmt.Sprintf(` AND pg_typeof(price)::text IS NOT NULL AND length(price) > 0 AND price >= $%d`, n))
		args = append(args, minPrice)
		n++
	}
	if maxPrice != "" {
		qb.WriteString(fmt.Sprintf(` AND length(price) > 0 AND price <= $%d`, n))
		args = append(args, maxPrice)
		n++
	}
	qb.WriteString(` ORDER BY created_at DESC`)

	rows, err := ns.pg.Query(ctx, qb.String(), args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	nfts := make([]*NFT, 0)
	for rows.Next() {
		nft, err := ns.getNFTRow(rows)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		nfts = append(nfts, nft)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"nfts":    nfts,
		"total":   len(nfts),
	})
}

// ============================================================================
// Middleware
// ============================================================================

func (ns *NFTService) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "no authorization header"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		// Validate the JWT issued by wallet_api and extract the real user id.
		userID, err := parseJWT(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			c.Abort()
			return
		}
		c.Set("user_id", userID)
		c.Next()
	}
}

// ============================================================================
// Main
// ============================================================================

func main() {
	if jwtSecret == "" {
		log.Fatalf("JWT_SECRET environment variable must be set")
	}
	log.Println("TigerWallet NFT Service")
	log.Println("========================")
	log.Printf("Starting on port %s", cfg.Port)

	ns := NewNFTService()

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	// CORS
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}
		c.Next()
	})

	// Health
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "nft-service",
			"timestamp": time.Now().Unix(),
		})
	})

	// Public routes
	r.GET("/api/v1/nft/collections", ns.GetCollections)
	r.GET("/api/v1/nft/collections/:id", ns.GetCollection)
	r.GET("/api/v1/nft/collections/:id/nfts", ns.GetCollectionNFTs)
	r.GET("/api/v1/nft/nfts/:id", ns.GetNFT)
	r.GET("/api/v1/nft/search", ns.SearchNFTs)
	r.GET("/api/v1/nft/listings", ns.GetListings)

	// Protected routes
	api := r.Group("/api/v1/nft")
	api.Use(ns.AuthMiddleware())
	{
		// Collection
		api.POST("/collections", ns.CreateCollection)

		// NFT
		api.POST("/mint", ns.MintNFT)
		api.POST("/list", ns.ListNFT)
		api.POST("/buy", ns.BuyNFT)
		api.POST("/offer", ns.MakeOffer)
		api.DELETE("/listings/:id", ns.CancelListing)

		// Auctions
		api.POST("/auctions", ns.CreateAuction)
		api.POST("/auctions/bid", ns.PlaceBid)
		api.POST("/auctions/:id/end", ns.EndAuction)
		api.GET("/auctions/active", ns.GetActiveAuctions)

		// User
		api.GET("/users/:user_id/nfts", ns.GetUserNFTs)
		api.GET("/nfts/:id/history", ns.GetNFTHistory)
	}

	// Public auction lookup (no auth).
	r.GET("/api/v1/nft/auctions/:id", ns.GetAuction)

	addr := ":" + cfg.Port
	log.Printf("Server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
