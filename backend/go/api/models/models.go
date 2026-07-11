package models

import "time"

type Blockchain struct {
	ID              string    `json:"id" db:"id"`
	Name            string    `json:"name" db:"name"`
	Symbol          string    `json:"symbol" db:"symbol"`
	ChainID         int64     `json:"chain_id" db:"chain_id"`
	Type            string    `json:"type" db:"type"`
	RPCURL          string    `json:"rpc_url" db:"rpc_url"`
	ExplorerURL     string    `json:"explorer_url" db:"explorer_url"`
	LogoURL         string    `json:"logo_url" db:"logo_url"`
	IsActive        bool      `json:"is_active" db:"is_active"`
	IsTestnet       bool      `json:"is_testnet" db:"is_testnet"`
	Decimals        int       `json:"decimals" db:"decimals"`
	GasToken        string    `json:"gas_token" db:"gas_token"`
	AvgBlockTime    int       `json:"avg_block_time" db:"avg_block_time"`
	MaxGasPrice     int64     `json:"max_gas_price" db:"max_gas_price"`
	SupportsEIP1559 bool      `json:"supports_eip1559" db:"supports_eip1559"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

type Token struct {
	ID            string    `json:"id" db:"id"`
	BlockchainID  string    `json:"blockchain_id" db:"blockchain_id"`
	Symbol        string    `json:"symbol" db:"symbol"`
	Name          string    `json:"name" db:"name"`
	Decimals      int       `json:"decimals" db:"decimals"`
	Address       *string   `json:"address" db:"address"`
	Type          string    `json:"type" db:"type"`
	TotalSupply   string    `json:"total_supply" db:"total_supply"`
	LogoURL       string    `json:"logo_url" db:"logo_url"`
	IsActive      bool      `json:"is_active" db:"is_active"`
	IsPopular     bool      `json:"is_popular" db:"is_popular"`
	PriceUSD      float64   `json:"price_usd" db:"price_usd"`
	MarketCap     int64     `json:"market_cap" db:"market_cap"`
	Volume24h     int64     `json:"volume_24h" db:"volume_24h"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

type Wallet struct {
	ID                 string    `json:"id" db:"id"`
	UserID             string    `json:"user_id" db:"user_id"`
	Type               string    `json:"type" db:"type"`
	Address            string    `json:"address" db:"address"`
	BlockchainID       string    `json:"blockchain_id" db:"blockchain_id"`
	PublicKey          string    `json:"public_key" db:"public_key"`
	EncryptedPrivateKey string   `json:"encrypted_private_key,omitempty" db:"encrypted_private_key"`
	DerivationPath     string    `json:"derivation_path" db:"derivation_path"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
	IsActive           bool      `json:"is_active" db:"is_active"`
	Label             *string   `json:"label,omitempty" db:"label"`
}

type Transaction struct {
	ID            string    `json:"id" db:"id"`
	WalletID      string    `json:"wallet_id" db:"wallet_id"`
	BlockchainID  string    `json:"blockchain_id" db:"blockchain_id"`
	Type          string    `json:"type" db:"type"`
	Status        string    `json:"status" db:"status"`
	From          string    `json:"from" db:"from"`
	To            string    `json:"to" db:"to"`
	TokenSymbol   string    `json:"token_symbol" db:"token_symbol"`
	TokenAddress  *string   `json:"token_address,omitempty" db:"token_address"`
	Amount        string    `json:"amount" db:"amount"`
	AmountUSD     float64   `json:"amount_usd" db:"amount_usd"`
	Fee           string    `json:"fee" db:"fee"`
	FeeUSD        float64   `json:"fee_usd" db:"fee_usd"`
	GasPrice      *string   `json:"gas_price,omitempty" db:"gas_price"`
	GasUsed       *string   `json:"gas_used,omitempty" db:"gas_used"`
	Nonce         *uint64   `json:"nonce,omitempty" db:"nonce"`
	Hash          string    `json:"hash" db:"hash"`
	BlockNumber   *uint64   `json:"block_number,omitempty" db:"block_number"`
	Timestamp     time.Time `json:"timestamp" db:"timestamp"`
	Error         *string   `json:"error,omitempty" db:"error"`
}

type User struct {
	ID                string    `json:"id" db:"id"`
	Email             string    `json:"email" db:"email"`
	Username          string    `json:"username" db:"username"`
	PasswordHash      string    `json:"-" db:"password_hash"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
	IsActive          bool      `json:"is_active" db:"is_active"`
	IsEmailVerified   bool      `json:"is_email_verified" db:"is_email_verified"`
	KYCStatus         string    `json:"kyc_status" db:"kyc_status"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
	Meta    *APIMeta    `json:"meta,omitempty"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type APIMeta struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Total int `json:"total"`
}
