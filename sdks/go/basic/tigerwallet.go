package tigerwallet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// ============== Client ==============

type Client struct {
	BaseURL     string
	APIKey      string
	HTTPClient *http.Client
	TenantID    string
}

// NewClient creates a new TigerWallet API client
func NewClient(apiKey, baseURL, tenantID string) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		TenantID: tenantID,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) doRequest(method, endpoint string, body interface{}) (*http.Response, error) {
	var reqBody []byte
	if body != nil {
		reqBody, _ = json.Marshal(body)
	}

	req, err := http.NewRequest(method, c.BaseURL+endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("X-Tenant-ID", c.TenantID)

	return c.HTTPClient.Do(req)
}

// ============== Authentication ==============

type AuthService struct {
	client *Client
}

func NewAuthService(client *Client) *AuthService {
	return &AuthService{client: client}
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
	User      User   `json:"user"`
}

type User struct {
	ID        string   `json:"id"`
	Email     string   `json:"email"`
	Name      string   `json:"name"`
	Role      string   `json:"role"`
	TenantID  string   `json:"tenant_id"`
	Permissions []string `json:"permissions"`
}

func (s *AuthService) Login(email, password string) (*LoginResponse, error) {
	resp, err := s.client.doRequest("POST", "/api/v1/auth/login", LoginRequest{
		Email:    email,
		Password: password,
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("login failed with status: %d", resp.StatusCode)
	}

	var result LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

func (s *AuthService) Register(email, password, name string) (*LoginResponse, error) {
	resp, err := s.client.doRequest("POST", "/api/v1/auth/register", RegisterRequest{
		Email:    email,
		Password: password,
		Name:     name,
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result LoginResponse
	json.NewDecoder(resp.Body).Decode(&result)

	return &result, nil
}

func (s *AuthService) Logout() error {
	resp, err := s.client.doRequest("POST", "/api/v1/auth/logout", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (s *AuthService) RefreshToken(token string) (string, error) {
	resp, err := s.client.doRequest("POST", "/api/v1/auth/refresh", map[string]string{
		"token": token,
	})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Token string `json:"token"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	return result.Token, nil
}

// ============== Wallet Service ==============

type WalletService struct {
	client *Client
}

func NewWalletService(client *Client) *WalletService {
	return &WalletService{client: client}
}

type Wallet struct {
	ID            string `json:"id"`
	Address       string `json:"address"`
	Chain         string `json:"chain"`
	Type          string `json:"type"`
	Balance       string `json:"balance"`
	TokenBalances []TokenBalance `json:"token_balances"`
	CreatedAt     time.Time `json:"created_at"`
}

type TokenBalance struct {
	Token   string `json:"token"`
	Symbol  string `json:"symbol"`
	Balance string `json:"balance"`
}

type CreateWalletRequest struct {
	Chain string `json:"chain"`
	Type  string `json:"type"` // eoa, contract, ledger, trezor
	Name  string `json:"name"`
}

func (s *WalletService) CreateWallet(req CreateWalletRequest) (*Wallet, error) {
	resp, err := s.client.doRequest("POST", "/api/v1/wallets", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result Wallet
	json.NewDecoder(resp.Body).Decode(&result)

	return &result, nil
}

func (s *WalletService) GetWallet(walletID string) (*Wallet, error) {
	resp, err := s.client.doRequest("GET", "/api/v1/wallets/"+walletID, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result Wallet
	json.NewDecoder(resp.Body).Decode(&result)

	return &result, nil
}

func (s *WalletService) ListWallets() ([]Wallet, error) {
	resp, err := s.client.doRequest("GET", "/api/v1/wallets", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Wallets []Wallet `json:"wallets"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	return result.Wallets, nil
}

type TransactionRequest struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Amount   string `json:"amount"`
	Token    string `json:"token"`
	Chain    string `json:"chain"`
	GasLimit uint64 `json:"gas_limit"`
}

type Transaction struct {
	ID        string    `json:"id"`
	Hash     string    `json:"hash"`
	From     string    `json:"from"`
	To       string    `json:"to"`
	Amount   string    `json:"amount"`
	Status   string    `json:"status"`
	Chain    string    `json:"chain"`
	GasUsed  uint64    `json:"gas_used"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *WalletService) SendTransaction(req TransactionRequest) (*Transaction, error) {
	resp, err := s.client.doRequest("POST", "/api/v1/transactions", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result Transaction
	json.NewDecoder(resp.Body).Decode(&result)

	return &result, nil
}

func (s *WalletService) GetTransaction(txHash string) (*Transaction, error) {
	resp, err := s.client.doRequest("GET", "/api/v1/transactions/"+txHash, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result Transaction
	json.NewDecoder(resp.Body).Decode(&result)

	return &result, nil
}

// ============== Billing Service ==============

type BillingService struct {
	client *Client
}

func NewBillingService(client *Client) *BillingService {
	return &BillingService{client: client}
}

type Plan struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Tier            string   `json:"tier"`
	PriceMonthly    int64    `json:"price_monthly"`
	PriceYearly     int64    `json:"price_yearly"`
	APIQuotaMonthly int64    `json:"api_quota_monthly"`
	MaxUsers        int      `json:"max_users"`
	MaxWallets     int      `json:"max_wallets"`
	MaxBots         int      `json:"max_bots"`
}

type Subscription struct {
	ID                 string    `json:"id"`
	PlanID            string    `json:"plan_id"`
	Status            string    `json:"status"`
	CurrentPeriodEnd  time.Time `json:"current_period_end"`
	Plan             *Plan     `json:"plan"`
}

func (s *BillingService) GetPlans() ([]Plan, error) {
	resp, err := s.client.doRequest("GET", "/api/v1/public/plans", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Plans []Plan `json:"plans"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	return result.Plans, nil
}

func (s *BillingService) GetSubscription() (*Subscription, error) {
	resp, err := s.client.doRequest("GET", "/api/v1/subscriptions/"+s.client.TenantID, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result Subscription
	json.NewDecoder(resp.Body).Decode(&result)

	return &result, nil
}

type UpgradeRequest struct {
	PlanID       string `json:"plan_id"`
	BillingCycle string `json:"billing_cycle"` // monthly, yearly
}

func (s *BillingService) UpgradeSubscription(req UpgradeRequest) (*Subscription, error) {
	resp, err := s.client.doRequest("POST", fmt.Sprintf("/api/v1/subscriptions/%s/upgrade", s.client.TenantID), req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Subscription Subscription `json:"subscription"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	return &result.Subscription, nil
}

type Usage struct {
	TotalAPIcalls   int64   `json:"total_api_calls"`
	APILimit        int64   `json:"api_limit"`
	StorageUsedGB   float64 `json:"storage_used_gb"`
	StorageLimitGB  float64 `json:"storage_limit_gb"`
	ActiveUsers     int     `json:"active_users"`
	UsersLimit      int     `json:"users_limit"`
	ActiveWallets   int     `json:"active_wallets"`
	WalletsLimit    int     `json:"wallets_limit"`
}

func (s *BillingService) GetUsage() (*Usage, error) {
	resp, err := s.client.doRequest("GET", "/api/v1/usage/"+s.client.TenantID, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result Usage
	json.NewDecoder(resp.Body).Decode(&result)

	return &result, nil
}

type Invoice struct {
	ID            string    `json:"id"`
	InvoiceNumber string    `json:"invoice_number"`
	Amount       int64     `json:"amount"`
	Status       string    `json:"status"`
	DueDate      time.Time `json:"due_date"`
	PaidAt       *time.Time `json:"paid_at"`
}

func (s *BillingService) GetInvoices() ([]Invoice, error) {
	resp, err := s.client.doRequest("GET", "/api/v1/invoices/"+s.client.TenantID, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Invoices []Invoice `json:"invoices"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	return result.Invoices, nil
}

// ============== Fetcher Service ==============

type FetcherService struct {
	client *Client
}

func NewFetcherService(client *Client) *FetcherService {
	return &FetcherService{client: client}
}

type PriceData struct {
	Symbol        string  `json:"symbol"`
	Price         float64 `json:"price"`
	Change24h     float64 `json:"change_24h"`
	Volume24h     float64 `json:"volume_24h"`
	MarketCap     float64 `json:"market_cap"`
	Timestamp    int64   `json:"timestamp"`
}

func (s *FetcherService) GetPrice(symbols []string) ([]PriceData, error) {
	resp, err := s.client.doRequest("GET", "/api/v1/fetcher/prices?symbols="+joinStrings(symbols), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Prices []PriceData `json:"prices"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	return result.Prices, nil
}

type BlockchainData struct {
	Chain         string `json:"chain"`
	BlockNumber   uint64 `json:"block_number"`
	BlockHash     string `json:"block_hash"`
	Timestamp     int64  `json:"timestamp"`
	GasPrice      string `json:"gas_price"`
}

func (s *FetcherService) GetBlockchainData(chain string) (*BlockchainData, error) {
	resp, err := s.client.doRequest("GET", "/api/v1/fetcher/blockchain/"+chain, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result BlockchainData
	json.NewDecoder(resp.Body).Decode(&result)

	return &result, nil
}

type WalletBalance struct {
	Address   string             `json:"address"`
	Native    string             `json:"native"`
	Tokens    []TokenBalance     `json:"tokens"`
	NFTs     []NFTBalance       `json:"nfts"`
}

type NFTBalance struct {
	Contract string `json:"contract"`
	TokenID  string `json:"token_id"`
	Amount   string `json:"amount"`
}

func (s *FetcherService) GetWalletBalance(chain, address string) (*WalletBalance, error) {
	resp, err := s.client.doRequest("GET", fmt.Sprintf("/api/v1/fetcher/balance/%s/%s", chain, address), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result WalletBalance
	json.NewDecoder(resp.Body).Decode(&result)

	return &result, nil
}

type TransactionData struct {
	Hash         string        `json:"hash"`
	From         string        `json:"from"`
	To           string        `json:"to"`
	Value        string        `json:"value"`
	Status       string        `json:"status"`
	BlockNumber uint64        `json:"block_number"`
	Timestamp   int64         `json:"timestamp"`
	Transfers   []TokenTransfer `json:"transfers"`
}

type TokenTransfer struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Token   string `json:"token"`
	Amount  string `json:"amount"`
}

func (s *FetcherService) GetTransactions(chain, address string, limit int) ([]TransactionData, error) {
	resp, err := s.client.doRequest("GET", fmt.Sprintf("/api/v1/fetcher/transactions/%s/%s?limit=%d", chain, address, limit), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Transactions []TransactionData `json:"transactions"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	return result.Transactions, nil
}

// ============== Notification Service ==============

type NotificationService struct {
	client *Client
}

func NewNotificationService(client *Client) *NotificationService {
	return &NotificationService{client: client}
}

type Notification struct {
	ID        string                 `json:"id"`
	Type     string                 `json:"type"`
	Title    string                 `json:"title"`
	Message  string                 `json:"message"`
	IsRead  bool                   `json:"is_read"`
	Data    map[string]interface{} `json:"data"`
	CreatedAt time.Time            `json:"created_at"`
}

func (s *NotificationService) GetNotifications() ([]Notification, error) {
	resp, err := s.client.doRequest("GET", "/api/v1/notifications", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Notifications []Notification `json:"notifications"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	return result.Notifications, nil
}

func (s *NotificationService) MarkAsRead(notificationID string) error {
	resp, err := s.client.doRequest("PUT", "/api/v1/notifications/"+notificationID+"/read", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (s *NotificationService) MarkAllAsRead() error {
	resp, err := s.client.doRequest("PUT", "/api/v1/notifications/read-all", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

type SendNotificationRequest struct {
	Type     string                 `json:"type"`
	Title    string                 `json:"title"`
	Message  string                 `json:"message"`
	Channel  string                 `json:"channel"` // email, sms, push
	Data    map[string]interface{} `json:"data"`
}

func (s *NotificationService) SendNotification(req SendNotificationRequest) error {
	resp, err := s.client.doRequest("POST", "/api/v1/notifications", req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

type NotificationPreference struct {
	EmailEnabled     bool `json:"email_enabled"`
	SMSEnabled      bool `json:"sms_enabled"`
	PushEnabled     bool `json:"push_enabled"`
	TransactionAlerts bool `json:"transaction_alerts"`
	MarketingAlerts  bool `json:"marketing_alerts"`
	SecurityAlerts  bool `json:"security_alerts"`
}

func (s *NotificationService) GetPreferences() (*NotificationPreference, error) {
	resp, err := s.client.doRequest("GET", "/api/v1/preferences/"+s.client.TenantID, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result NotificationPreference
	json.NewDecoder(resp.Body).Decode(&result)

	return &result, nil
}

func (s *NotificationService) UpdatePreferences(prefs NotificationPreference) error {
	resp, err := s.client.doRequest("PUT", "/api/v1/preferences/"+s.client.TenantID, prefs)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// ============== KYC Service ==============

type KYCService struct {
	client *Client
}

func NewKYCService(client *Client) *KYCService {
	return &KYCService{client: client}
}

type KYCStatus struct {
	Status       string    `json:"status"`
	Level        string    `json:"level"`
	VerifiedAt   *time.Time `json:"verified_at"`
	ExpiryDate   *time.Time `json:"expiry_date"`
	Documents    []string   `json:"documents"`
}

func (s *KYCService) GetStatus() (*KYCStatus, error) {
	resp, err := s.client.doRequest("GET", "/api/v1/kyc/status/"+s.client.TenantID, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result KYCStatus
	json.NewDecoder(resp.Body).Decode(&result)

	return &result, nil
}

type KYCSubmission struct {
	FirstName          string `json:"first_name"`
	LastName           string `json:"last_name"`
	DateOfBirth        string `json:"date_of_birth"`
	Nationality        string `json:"nationality"`
	CountryOfResidence string `json:"country_of_residence"`
	Address            string `json:"address"`
	City               string `json:"city"`
	PostalCode        string `json:"postal_code"`
	PhoneNumber       string `json:"phone_number"`
}

func (s *KYCService) Submit(submission KYCSubmission) (string, error) {
	resp, err := s.client.doRequest("POST", "/api/v1/kyc/submit", submission)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		ApplicationID string `json:"application_id"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	return result.ApplicationID, nil
}

type DocumentUpload struct {
	Type      string `json:"type"` // id_card, passport, driver_license
	Side      string `json:"side"` // front, back
	FileData  string `json:"file_data"` // base64
}

func (s *KYCService) UploadDocument(applicationID string, doc DocumentUpload) error {
	resp, err := s.client.doRequest("POST", "/api/v1/kyc/documents/upload", map[string]interface{}{
		"application_id": applicationID,
		"type":          doc.Type,
		"side":         doc.Side,
		"file_data":     doc.FileData,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// ============== Helper Functions ==============

func joinStrings(s []string) string {
	result := ""
	for i, v := range s {
		if i > 0 {
			result += ","
		}
		result += v
	}
	return result
}
