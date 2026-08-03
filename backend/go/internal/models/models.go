package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID            uuid.UUID `json:"id"`
	Email         string    `json:"email"`
	Username      string    `json:"username"`
	PasswordHash  string    `json:"-"`
	WalletAddress string    `json:"walletAddress,omitempty"`
	Phone         string    `json:"phone,omitempty"`
	KYCLevel      string    `json:"kycLevel"`
	KYCStatus     string    `json:"kycStatus"`
	Status        string    `json:"status"`
	RiskScore     int       `json:"riskScore"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	LastLoginAt   time.Time `json:"lastLoginAt,omitempty"`
}

type Session struct {
	ID              uuid.UUID `json:"id"`
	UserID          uuid.UUID `json:"userId"`
	TokenHash       string    `json:"-"`
	ExpiresAt       time.Time `json:"expiresAt"`
	IsRevoked       bool      `json:"isRevoked"`
	CreatedAt       time.Time `json:"createdAt"`
	LastActivityAt time.Time `json:"lastActivityAt"`
}

type P2PMerchant struct {
	ID               uuid.UUID `json:"id"`
	UserID          uuid.UUID `json:"userId"`
	Status          string    `json:"status"`
	CollateralToken string    `json:"collateralToken"`
	CollateralAmount float64   `json:"collateralAmount"`
	CollateralTxHash string    `json:"collateralTxHash,omitempty"`
	CollateralLockedAt time.Time `json:"collateralLockedAt,omitempty"`
	TraderLevel     string    `json:"traderLevel"`
	TotalTrades     int       `json:"totalTrades"`
	TotalVolume     float64   `json:"totalVolume"`
	CompletedTrades int       `json:"completedTrades"`
	CancelledTrades int       `json:"cancelledTrades"`
	DisputeCount    int       `json:"disputeCount"`
	Rating          float64   `json:"rating"`
	TotalReviews    int       `json:"totalReviews"`
	AvgResponseTime float64   `json:"avgResponseTime"`
	AvgReleaseTime  float64   `json:"avgReleaseTime"`
	SecurityScore   int       `json:"securityScore"`
	IsVerified      bool      `json:"isVerified"`
	KYCLevel        string    `json:"kycLevel"`
	JoinedAt        time.Time `json:"joinedAt"`
	LastActiveAt    time.Time `json:"lastActiveAt,omitempty"`
}

type P2PAdvert struct {
	ID              uuid.UUID `json:"id"`
	MerchantID      uuid.UUID `json:"merchantId"`
	Side            string    `json:"side"`
	TokenID         uuid.UUID `json:"tokenId"`
	Token           string    `json:"token"`
	FiatCurrency    string    `json:"fiatCurrency"`
	PaymentMethod   string    `json:"paymentMethod"`
	Price           float64   `json:"price"`
	MinAmount       float64   `json:"minAmount"`
	MaxAmount       float64   `json:"maxAmount"`
	AvailableAmount float64   `json:"availableAmount"`
	IsActive        bool      `json:"isActive"`
	AutoReply       string    `json:"autoReply,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
	ExpiresAt       time.Time `json:"expiresAt,omitempty"`

	// Join fields
	Username        string  `json:"username"`
	Avatar          string  `json:"avatar"`
	MerchantLevel   string  `json:"merchantLevel,omitempty"`
	CollateralLocked float64 `json:"collateralLocked,omitempty"`
	IsMerchant     bool    `json:"isMerchant"`
	IsVerified     bool    `json:"isVerified"`
	SecurityScore  int     `json:"securityScore"`
	OrdersCompleted int     `json:"ordersCompleted"`
	CompletionRate  float64 `json:"completionRate"`
	AvgReleaseTime float64 `json:"avgReleaseTime"`
	IsOnline       bool    `json:"isOnline"`
}

type P2POrder struct {
	ID              uuid.UUID  `json:"id"`
	AdvertID        uuid.UUID  `json:"advertId"`
	BuyerID         uuid.UUID  `json:"buyerId"`
	SellerID        uuid.UUID  `json:"sellerId"`
	Side            string     `json:"side"`
	TokenID         uuid.UUID  `json:"tokenId"`
	Token           string     `json:"token"`
	FiatCurrency    string     `json:"fiatCurrency"`
	PaymentMethod   string     `json:"paymentMethod"`
	Price           float64    `json:"price"`
	Amount          float64    `json:"amount"`
	FiatAmount      float64    `json:"fiatAmount"`
	BuyerDeposit    float64    `json:"buyerDeposit,omitempty"`
	SellerDeposit   float64    `json:"sellerDeposit,omitempty"`
	Status          string     `json:"status"`
	BuyerConfirmTime *time.Time `json:"buyerConfirmTime,omitempty"`
	SellerConfirmTime *time.Time `json:"sellerConfirmTime,omitempty"`
	ReleaseTime     *time.Time `json:"releaseTime,omitempty"`
	CancelTime      *time.Time `json:"cancelTime,omitempty"`
	CancelReason    string     `json:"cancelReason,omitempty"`
	DisputeOpened   bool       `json:"disputeOpened"`
	DisputeReason   string     `json:"disputeReason,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

type SecurityDeposit struct {
	ID          uuid.UUID `json:"id"`
	OrderID     uuid.UUID `json:"orderId"`
	UserID      uuid.UUID `json:"userId"`
	DepositType string    `json:"depositType"`
	TokenID     uuid.UUID `json:"tokenId"`
	Amount      float64   `json:"amount"`
	USDValue    float64   `json:"usdValue"`
	TxHash      string    `json:"txHash"`
	Status      string    `json:"status"`
	LockedAt    time.Time `json:"lockedAt"`
	ReleasedAt  *time.Time `json:"releasedAt,omitempty"`
}

type MarginAccount struct {
	ID              uuid.UUID `json:"id"`
	UserID          uuid.UUID `json:"userId"`
	TotalAssets     float64   `json:"totalAssets"`
	TotalLiabilities float64  `json:"totalLiabilities"`
	NetAssets       float64   `json:"netAssets"`
	AvailableBalance float64  `json:"availableBalance"`
	MarginRatio     float64   `json:"marginRatio"`
	RiskLevel       string    `json:"riskLevel"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type MarginPosition struct {
	ID                uuid.UUID  `json:"id"`
	AccountID         uuid.UUID  `json:"accountId"`
	TokenID           uuid.UUID  `json:"tokenId"`
	Side              string     `json:"side"`
	Size              float64    `json:"size"`
	EntryPrice        float64    `json:"entryPrice"`
	MarkPrice         float64    `json:"markPrice,omitempty"`
	Leverage          int        `json:"leverage"`
	Margin            float64    `json:"margin"`
	MarginMode        string     `json:"marginMode"`
	PNL               float64    `json:"pnl"`
	LiquidationPrice  float64    `json:"liquidationPrice,omitempty"`
	Status            string     `json:"status"`
	OpenedAt          time.Time   `json:"openedAt"`
	ClosedAt          *time.Time  `json:"closedAt,omitempty"`
}

type Wallet struct {
	ID                uuid.UUID `json:"id"`
	UserID            uuid.UUID `json:"userId"`
	Chain             string    `json:"chain"`
	Address           string    `json:"address"`
	PrivateKeyEncrypted string  `json:"-"`
	Balance           float64   `json:"balance"`
	ReservedBalance   float64   `json:"reservedBalance"`
	CreatedAt         time.Time `json:"createdAt"`
}

type CryptoCard struct {
	ID              uuid.UUID `json:"id"`
	UserID          uuid.UUID `json:"userId"`
	CardNumberMasked string    `json:"cardNumberMasked"`
	CardType        string    `json:"cardType"`
	Network         string    `json:"network"`
	Status          string    `json:"status"`
	DailyLimit      float64   `json:"dailyLimit"`
	MonthlyLimit    float64   `json:"monthlyLimit"`
	DailySpent      float64   `json:"dailySpent"`
	MonthlySpent    float64   `json:"monthlySpent"`
	CreatedAt       time.Time `json:"createdAt"`
	ActivatedAt     *time.Time `json:"activatedAt,omitempty"`
}

// API Request/Response types
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=8"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type CreateOrderRequest struct {
	AdvertID string  `json:"advertId" binding:"required"`
	Amount   float64 `json:"amount" binding:"required,gt=0"`
}

type CreateAdvertRequest struct {
	Side          string  `json:"side" binding:"required,oneof=BUY SELL"`
	Token         string  `json:"token" binding:"required"`
	FiatCurrency  string  `json:"fiatCurrency" binding:"required"`
	PaymentMethod string  `json:"paymentMethod" binding:"required"`
	Price         float64 `json:"price" binding:"required,gt=0"`
	MinAmount     float64 `json:"minAmount" binding:"required,gt=0"`
	MaxAmount     float64 `json:"maxAmount" binding:"gtefield=MinAmount"`
}

type ApplyMerchantRequest struct {
	CollateralToken string  `json:"collateralToken" binding:"required"`
	CollateralAmount float64 `json:"collateralAmount" binding:"required,gt=0"`
}

// API Responses
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
}
