package models

import "time"

// Swap Quote
type SwapQuote struct {
	ID                string    `json:"id"`
	FromToken         string    `json:"from_token"`
	ToToken           string    `json:"to_token"`
	FromAmount        string    `json:"from_amount"`
	ToAmount          string    `json:"to_amount"`
	ToAmountUSD       float64   `json:"to_amount_usd"`
	PriceImpact       float64   `json:"price_impact"`
	GuaranteedPrice   string    `json:"guaranteed_price"`
	Route             []string  `json:"route"`
	AllowanceTarget   string    `json:"allowance_target"`
	TxData            string    `json:"tx_data"`
	ValidityPeriod    int       `json:"validity_period"`
	GasEstimate       string    `json:"gas_estimate"`
	CreatedAt         time.Time `json:"created_at"`
}

type Swap struct {
	ID            string    `json:"id"`
	WalletID      string    `json:"wallet_id"`
	QuoteID       string    `json:"quote_id"`
	FromToken     string    `json:"from_token"`
	ToToken       string    `json:"to_token"`
	FromAmount    string    `json:"from_amount"`
	ToAmount      string    `json:"to_amount"`
	Status        string    `json:"status"`
	TransactionID string    `json:"transaction_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type SwapRequest struct {
	FromToken        string  `json:"from_token"`
	ToToken          string  `json:"to_token"`
	FromAmount       string  `json:"from_amount"`
	SlippageTolerance float64 `json:"slippage_tolerance"`
}

// Staking
type StakingPool struct {
	ID            string    `json:"id"`
	BlockchainID  string    `json:"blockchain_id"`
	TokenSymbol   string    `json:"token_symbol"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	MinStake      string    `json:"min_stake"`
	MaxStake      string    `json:"max_stake"`
	APY           float64   `json:"apy"`
	LockPeriod    int       `json:"lock_period"`
	IsActive      bool      `json:"is_active"`
}

type StakingPosition struct {
	ID              string    `json:"id"`
	WalletID        string    `json:"wallet_id"`
	PoolID          string    `json:"pool_id"`
	BlockchainID    string    `json:"blockchain_id"`
	TokenSymbol     string    `json:"token_symbol"`
	Amount          string    `json:"amount"`
	RewardAmount    string    `json:"reward_amount"`
	RewardClaimed   string    `json:"reward_claimed"`
	APY             float64   `json:"apy"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Perpetual Trading
type PerpetualMarket struct {
	Symbol            string  `json:"symbol"`
	DisplayName       string  `json:"display_name"`
	IndexPrice       string  `json:"index_price"`
	MarkPrice        string  `json:"mark_price"`
	LastPrice        string  `json:"last_price"`
	Change24h        float64 `json:"change_24h"`
	ChangePercent24h float64 `json:"change_percent_24h"`
	High24h          string  `json:"high_24h"`
	Low24h           string  `json:"low_24h"`
	Volume24h        string  `json:"volume_24h"`
	OpenInterest     string  `json:"open_interest"`
	FundingRate      string  `json:"funding_rate"`
	NextFundingTime   string  `json:"next_funding_time"`
	MaxLeverage      int     `json:"max_leverage"`
	MinMargin        string  `json:"min_margin"`
	LiquidationFee   string  `json:"liquidation_fee"`
}

type PerpetualPosition struct {
	ID                string    `json:"id"`
	WalletID          string    `json:"wallet_id"`
	Symbol            string    `json:"symbol"`
	Side              string    `json:"side"`
	Size              string    `json:"size"`
	EntryPrice        string    `json:"entry_price"`
	MarkPrice         string    `json:"mark_price"`
	LiquidationPrice  string    `json:"liquidation_price"`
	Margin            string    `json:"margin"`
	MarginRatio       float64   `json:"margin_ratio"`
	UnrealizedPnL    string    `json:"unrealized_pnl"`
	RealizedPnL       string    `json:"realized_pnl"`
	FundingPayment   string    `json:"funding_payment"`
	Leverage          int       `json:"leverage"`
	Status            string    `json:"status"`
	OpenedAt          time.Time `json:"opened_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type PerpetualOrder struct {
	ID            string    `json:"id"`
	WalletID      string    `json:"wallet_id"`
	Symbol        string    `json:"symbol"`
	Side          string    `json:"side"`
	OrderType     string    `json:"order_type"`
	Size          string    `json:"size"`
	Price         string    `json:"price"`
	TriggerPrice  string    `json:"trigger_price"`
	Margin        string    `json:"margin"`
	Leverage      int       `json:"leverage"`
	Status        string    `json:"status"`
	FilledSize    string    `json:"filled_size"`
	FilledPrice   string    `json:"filled_price"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Copy Trading
type CopyTrader struct {
	ID             string    `json:"id"`
	Address        string    `json:"address"`
	Name           string    `json:"name"`
	Avatar         string    `json:"avatar"`
	TotalTrades    int       `json:"total_trades"`
	WinRate        float64   `json:"win_rate"`
	ProfitFactor   float64   `json:"profit_factor"`
	AUM            string    `json:"aum"`
	FollowersCount int       `json:"followers_count"`
	Performance    struct {
		Daily   float64 `json:"daily"`
		Weekly  float64 `json:"weekly"`
		Monthly float64 `json:"monthly"`
		AllTime float64 `json:"all_time"`
	} `json:"performance"`
	IsVerified     bool      `json:"is_verified"`
}

type CopyTrade struct {
	ID          string    `json:"id"`
	FollowerID  string    `json:"follower_id"`
	TraderID    string    `json:"trader_id"`
	Symbol      string    `json:"symbol"`
	Side        string    `json:"side"`
	Size        string    `json:"size"`
	EntryPrice  string    `json:"entry_price"`
	ExitPrice   string    `json:"exit_price"`
	PnL         string    `json:"pnl"`
	PnLPercent  float64   `json:"pnl_percent"`
	Status      string    `json:"status"`
	OpenedAt    time.Time `json:"opened_at"`
	ClosedAt    time.Time `json:"closed_at"`
}

// NFT
type NFTCollection struct {
	Address        string    `json:"address"`
	BlockchainID   string    `json:"blockchain_id"`
	Name          string    `json:"name"`
	Symbol        string    `json:"symbol"`
	Description   string    `json:"description"`
	ImageURL      string    `json:"image_url"`
	BannerURL     string    `json:"banner_url"`
	FloorPrice    string    `json:"floor_price"`
	FloorPriceUSD float64   `json:"floor_price_usd"`
	TotalSupply   string    `json:"total_supply"`
	HolderCount   string    `json:"holder_count"`
}

type NFT struct {
	ID              string    `json:"id"`
	WalletID        string    `json:"wallet_id"`
	CollectionAddr  string    `json:"collection_address"`
	TokenID        string    `json:"token_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	ImageURL       string    `json:"image_url"`
	AnimationURL   string    `json:"animation_url"`
	Attributes     []string  `json:"attributes"`
	Owner          string    `json:"owner"`
	Standard       string    `json:"standard"`
	MetadataURL    string    `json:"metadata_url"`
	IsListed      bool      `json:"is_listed"`
	ListingPrice  string    `json:"listing_price"`
}
