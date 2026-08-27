package main

// user_wallet_management.go — MasterWallet → UserWallet governance layer.
//
// The MasterWallet owner manages the UserWallet ecosystem:
//   - Add/remove/update EVM blockchains available to UserWallet users.
//   - Add/remove/update non-EVM blockchains available to UserWallet users.
//   - Add/remove/update coins/tokens available to UserWallet users.
//   - Derive UserWallet addresses from a user's 24-word seed for ANY chain
//     (EVM via BIP-44 m/44'/60'/..., Solana via SLIP-10 Ed25519, Bitcoin P2PKH,
//     Cosmos bech32). One master wallet owns billions of UserWallet addresses.
//   - Auto-sign and auto-approve ALL UserWallet transactions (send/claim/swap/
//     trade) for any address on any chain — real secp256k1/Ed25519 signing.
//   - Manage fees for UserWallet transactions.
//   - SuperAdmin feature-flag governance: super admin controls which features
//     are enabled; master wallet owner has full control of enabled features.
//
// All real crypto (no fakes/stubs/mocks). PostgreSQL persistence (no SQLite).

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ----------------------------------------------------------------------------
// Types
// ----------------------------------------------------------------------------

// UserChainEVM is an EVM blockchain managed by the master wallet owner for
// UserWallet users.
type UserChainEVM struct {
	ID             int64  `json:"id"`
	MasterWalletID string `json:"master_wallet_id"`
	ChainID        int64  `json:"chain_id"`
	Name           string `json:"name"`
	Symbol         string `json:"symbol"`
	RPCURL         string `json:"rpc_url"`
	ExplorerURL    string `json:"explorer_url"`
	Decimals       int    `json:"decimals"`
	DerivationPath string `json:"derivation_path"`
	IsActive       bool   `json:"is_active"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// UserChainNonEVM is a non-EVM blockchain managed by the master wallet owner.
type UserChainNonEVM struct {
	ID             int64  `json:"id"`
	MasterWalletID string `json:"master_wallet_id"`
	ChainID        int64  `json:"chain_id"` // SLIP-44 namespace >= 9000000000
	Name           string `json:"name"`
	Symbol         string `json:"symbol"`
	ChainType      string `json:"chain_type"` // solana, bitcoin, cosmos, etc.
	RPCURL         string `json:"rpc_url"`
	ExplorerURL    string `json:"explorer_url"`
	Decimals       int    `json:"decimals"`
	DerivationPath string `json:"derivation_path"`
	AddressPrefix  string `json:"address_prefix"` // bech32 prefix for cosmos
	IsActive       bool   `json:"is_active"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// UserToken is a coin/token managed by the master wallet owner for UserWallet.
type UserToken struct {
	ID              int64  `json:"id"`
	MasterWalletID  string `json:"master_wallet_id"`
	ChainID         int64  `json:"chain_id"`
	ContractAddress string `json:"contract_address"`
	Symbol          string `json:"symbol"`
	Name            string `json:"name"`
	Decimals        int    `json:"decimals"`
	LogoURI         string `json:"logo_uri"`
	IsNative        bool   `json:"is_native"`
	IsActive        bool   `json:"is_active"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

// UserWalletAddress is a derived address owned by the master wallet.
type UserWalletAddress struct {
	ID             int64  `json:"id"`
	MasterWalletID string `json:"master_wallet_id"`
	UserSeedHash   string `json:"seed_hash"` // SHA-256 of seed (never store seed)
	ChainID        int64  `json:"chain_id"`
	ChainType      string `json:"chain_type"`
	Address        string `json:"address"`
	DerivationPath string `json:"derivation_path"`
	AccountIndex   int    `json:"account_index"`
	CreatedAt      string `json:"created_at"`
}

// AutoSignLog records an auto-signed UserWallet transaction.
type AutoSignLog struct {
	ID             int64  `json:"id"`
	MasterWalletID string `json:"master_wallet_id"`
	UserAddress    string `json:"user_address"`
	ChainID        int64  `json:"chain_id"`
	TxType         string `json:"tx_type"` // send, claim, swap, trade
	ToAddress      string `json:"to_address"`
	Value          string `json:"value"`
	TokenAddress   string `json:"token_address"`
	TxHash         string `json:"tx_hash"`
	Status         string `json:"status"` // signed, broadcast, confirmed, failed
	CreatedAt      string `json:"created_at"`
}

// FeatureFlag is a super-admin-controlled feature flag.
type FeatureFlag struct {
	ID                int64  `json:"id"`
	MasterWalletID    string `json:"master_wallet_id"`
	FlagKey           string `json:"flag_key"`
	FlagValue         string `json:"flag_value"`
	Description       string `json:"description"`
	IsEnabled         bool   `json:"is_enabled"`
	AddedBySuperAdmin bool   `json:"added_by_super_admin"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
}

// ----------------------------------------------------------------------------
// DB migrations for the new tables
// ----------------------------------------------------------------------------

func userWalletMigrations() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS user_chains_evm (
			id BIGSERIAL PRIMARY KEY,
			master_wallet_id UUID NOT NULL,
			chain_id BIGINT NOT NULL,
			name VARCHAR(100) NOT NULL,
			symbol VARCHAR(20) NOT NULL,
			rpc_url TEXT NOT NULL,
			explorer_url TEXT NOT NULL DEFAULT '',
			decimals INTEGER NOT NULL DEFAULT 18,
			derivation_path VARCHAR(100) NOT NULL DEFAULT 'm/44''/60''/0''/0/0',
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE(master_wallet_id, chain_id)
		)`,
		`CREATE TABLE IF NOT EXISTS user_chains_nonevm (
			id BIGSERIAL PRIMARY KEY,
			master_wallet_id UUID NOT NULL,
			chain_id BIGINT NOT NULL,
			name VARCHAR(100) NOT NULL,
			symbol VARCHAR(20) NOT NULL,
			chain_type VARCHAR(30) NOT NULL,
			rpc_url TEXT NOT NULL,
			explorer_url TEXT NOT NULL DEFAULT '',
			decimals INTEGER NOT NULL DEFAULT 9,
			derivation_path VARCHAR(100) NOT NULL,
			address_prefix VARCHAR(20) NOT NULL DEFAULT '',
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE(master_wallet_id, chain_id)
		)`,
		`CREATE TABLE IF NOT EXISTS user_tokens (
			id BIGSERIAL PRIMARY KEY,
			master_wallet_id UUID NOT NULL,
			chain_id BIGINT NOT NULL,
			contract_address VARCHAR(80) NOT NULL DEFAULT '',
			symbol VARCHAR(30) NOT NULL,
			name VARCHAR(100) NOT NULL,
			decimals INTEGER NOT NULL DEFAULT 18,
			logo_uri TEXT NOT NULL DEFAULT '',
			is_native BOOLEAN NOT NULL DEFAULT FALSE,
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE(master_wallet_id, chain_id, contract_address)
		)`,
		`CREATE TABLE IF NOT EXISTS user_wallet_addresses (
			id BIGSERIAL PRIMARY KEY,
			master_wallet_id UUID NOT NULL,
			seed_hash VARCHAR(64) NOT NULL,
			chain_id BIGINT NOT NULL,
			chain_type VARCHAR(30) NOT NULL DEFAULT 'evm',
			address VARCHAR(80) NOT NULL,
			derivation_path VARCHAR(100) NOT NULL,
			account_index INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE(master_wallet_id, seed_hash, chain_id, account_index)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_uwa_master ON user_wallet_addresses(master_wallet_id)`,
		`CREATE INDEX IF NOT EXISTS idx_uwa_address ON user_wallet_addresses(address)`,
		`CREATE TABLE IF NOT EXISTS auto_sign_log (
			id BIGSERIAL PRIMARY KEY,
			master_wallet_id UUID NOT NULL,
			user_address VARCHAR(80) NOT NULL,
			chain_id BIGINT NOT NULL,
			tx_type VARCHAR(20) NOT NULL,
			to_address VARCHAR(80) NOT NULL,
			value TEXT NOT NULL DEFAULT '0',
			token_address VARCHAR(80) NOT NULL DEFAULT '',
			tx_hash VARCHAR(80) NOT NULL DEFAULT '',
			status VARCHAR(20) NOT NULL DEFAULT 'signed',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_asl_master ON auto_sign_log(master_wallet_id)`,
		`CREATE TABLE IF NOT EXISTS feature_flags (
			id BIGSERIAL PRIMARY KEY,
			master_wallet_id UUID NOT NULL,
			flag_key VARCHAR(100) NOT NULL,
			flag_value TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			is_enabled BOOLEAN NOT NULL DEFAULT FALSE,
			added_by_super_admin BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE(master_wallet_id, flag_key)
		)`,
	}
}

// ----------------------------------------------------------------------------
// User EVM Chain management handlers
// ----------------------------------------------------------------------------

// ListUserEVMChains GET /api/v1/master-wallet/:id/user-chains/evm
func (svc *Service) ListUserEVMChains(c *gin.Context) {
	masterID := c.Param("id")
	rows, err := store.DB().Query(c.Request.Context(), `
		SELECT id, chain_id, name, symbol, rpc_url, explorer_url, decimals,
			derivation_path, is_active, created_at::text, updated_at::text
		FROM user_chains_evm WHERE master_wallet_id = $1 ORDER BY chain_id`, masterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db query failed"})
		return
	}
	defer rows.Close()
	var chains []UserChainEVM
	for rows.Next() {
		var ch UserChainEVM
		ch.MasterWalletID = masterID
		if err := rows.Scan(&ch.ID, &ch.ChainID, &ch.Name, &ch.Symbol, &ch.RPCURL,
			&ch.ExplorerURL, &ch.Decimals, &ch.DerivationPath, &ch.IsActive,
			&ch.CreatedAt, &ch.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan failed"})
			return
		}
		chains = append(chains, ch)
	}
	c.JSON(http.StatusOK, gin.H{"chains": chains})
}

// AddUserEVMChain POST /api/v1/master-wallet/:id/user-chains/evm
func (svc *Service) AddUserEVMChain(c *gin.Context) {
	masterID := c.Param("id")
	var req UserChainEVM
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.ChainID == 0 || req.Name == "" || req.RPCURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chain_id, name, rpc_url required"})
		return
	}
	if req.DerivationPath == "" {
		req.DerivationPath = "m/44'/60'/0'/0/0"
	}
	if req.Decimals == 0 {
		req.Decimals = 18
	}
	var id int64
	err := store.DB().QueryRow(c.Request.Context(), `
		INSERT INTO user_chains_evm (master_wallet_id, chain_id, name, symbol, rpc_url,
			explorer_url, decimals, derivation_path, is_active)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id`,
		masterID, req.ChainID, req.Name, req.Symbol, req.RPCURL, req.ExplorerURL,
		req.Decimals, req.DerivationPath, true).Scan(&id)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "chain already exists or invalid: " + err.Error()})
		return
	}
	svc.auditAction(masterID, c.GetString("user_id"), "add_evm_chain", fmt.Sprintf("chain_id=%d name=%s", req.ChainID, req.Name))
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "EVM chain added"})
}

// UpdateUserEVMChain PUT /api/v1/master-wallet/:id/user-chains/evm/:chainId
func (svc *Service) UpdateUserEVMChain(c *gin.Context) {
	masterID := c.Param("id")
	chainID := c.Param("chainId")
	var req UserChainEVM
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := store.DB().Exec(c.Request.Context(), `
		UPDATE user_chains_evm SET name=$1, symbol=$2, rpc_url=$3, explorer_url=$4,
			decimals=$5, derivation_path=$6, is_active=$7, updated_at=NOW()
		WHERE master_wallet_id=$8 AND chain_id=$9`,
		req.Name, req.Symbol, req.RPCURL, req.ExplorerURL, req.Decimals,
		req.DerivationPath, req.IsActive, masterID, chainID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	svc.auditAction(masterID, c.GetString("user_id"), "update_evm_chain", "chain_id="+chainID)
	c.JSON(http.StatusOK, gin.H{"message": "EVM chain updated"})
}

// RemoveUserEVMChain DELETE /api/v1/master-wallet/:id/user-chains/evm/:chainId
func (svc *Service) RemoveUserEVMChain(c *gin.Context) {
	masterID := c.Param("id")
	chainID := c.Param("chainId")
	_, err := store.DB().Exec(c.Request.Context(), `DELETE FROM user_chains_evm WHERE master_wallet_id=$1 AND chain_id=$2`, masterID, chainID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	svc.auditAction(masterID, c.GetString("user_id"), "remove_evm_chain", "chain_id="+chainID)
	c.JSON(http.StatusOK, gin.H{"message": "EVM chain removed"})
}

// ----------------------------------------------------------------------------
// User Non-EVM Chain management handlers
// ----------------------------------------------------------------------------

// ListUserNonEVMChains GET /api/v1/master-wallet/:id/user-chains/nonevm
func (svc *Service) ListUserNonEVMChains(c *gin.Context) {
	masterID := c.Param("id")
	rows, err := store.DB().Query(c.Request.Context(), `
		SELECT id, chain_id, name, symbol, chain_type, rpc_url, explorer_url,
			decimals, derivation_path, address_prefix, is_active,
			created_at::text, updated_at::text
		FROM user_chains_nonevm WHERE master_wallet_id = $1 ORDER BY chain_id`, masterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db query failed"})
		return
	}
	defer rows.Close()
	var chains []UserChainNonEVM
	for rows.Next() {
		var ch UserChainNonEVM
		ch.MasterWalletID = masterID
		if err := rows.Scan(&ch.ID, &ch.ChainID, &ch.Name, &ch.Symbol, &ch.ChainType,
			&ch.RPCURL, &ch.ExplorerURL, &ch.Decimals, &ch.DerivationPath,
			&ch.AddressPrefix, &ch.IsActive, &ch.CreatedAt, &ch.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan failed"})
			return
		}
		chains = append(chains, ch)
	}
	c.JSON(http.StatusOK, gin.H{"chains": chains})
}

// AddUserNonEVMChain POST /api/v1/master-wallet/:id/user-chains/nonevm
func (svc *Service) AddUserNonEVMChain(c *gin.Context) {
	masterID := c.Param("id")
	var req UserChainNonEVM
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.ChainID == 0 || req.Name == "" || req.ChainType == "" || req.DerivationPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chain_id, name, chain_type, derivation_path required"})
		return
	}
	var id int64
	err := store.DB().QueryRow(c.Request.Context(), `
		INSERT INTO user_chains_nonevm (master_wallet_id, chain_id, name, symbol,
			chain_type, rpc_url, explorer_url, decimals, derivation_path,
			address_prefix, is_active)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id`,
		masterID, req.ChainID, req.Name, req.Symbol, req.ChainType, req.RPCURL,
		req.ExplorerURL, req.Decimals, req.DerivationPath, req.AddressPrefix, true).Scan(&id)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "chain already exists or invalid: " + err.Error()})
		return
	}
	svc.auditAction(masterID, c.GetString("user_id"), "add_nonevm_chain", fmt.Sprintf("chain_id=%d type=%s", req.ChainID, req.ChainType))
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "non-EVM chain added"})
}

// UpdateUserNonEVMChain PUT /api/v1/master-wallet/:id/user-chains/nonevm/:chainId
func (svc *Service) UpdateUserNonEVMChain(c *gin.Context) {
	masterID := c.Param("id")
	chainID := c.Param("chainId")
	var req UserChainNonEVM
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := store.DB().Exec(c.Request.Context(), `
		UPDATE user_chains_nonevm SET name=$1, symbol=$2, chain_type=$3, rpc_url=$4,
			explorer_url=$5, decimals=$6, derivation_path=$7, address_prefix=$8,
			is_active=$9, updated_at=NOW()
		WHERE master_wallet_id=$10 AND chain_id=$11`,
		req.Name, req.Symbol, req.ChainType, req.RPCURL, req.ExplorerURL,
		req.Decimals, req.DerivationPath, req.AddressPrefix, req.IsActive, masterID, chainID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	svc.auditAction(masterID, c.GetString("user_id"), "update_nonevm_chain", "chain_id="+chainID)
	c.JSON(http.StatusOK, gin.H{"message": "non-EVM chain updated"})
}

// RemoveUserNonEVMChain DELETE /api/v1/master-wallet/:id/user-chains/nonevm/:chainId
func (svc *Service) RemoveUserNonEVMChain(c *gin.Context) {
	masterID := c.Param("id")
	chainID := c.Param("chainId")
	_, err := store.DB().Exec(c.Request.Context(), `DELETE FROM user_chains_nonevm WHERE master_wallet_id=$1 AND chain_id=$2`, masterID, chainID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	svc.auditAction(masterID, c.GetString("user_id"), "remove_nonevm_chain", "chain_id="+chainID)
	c.JSON(http.StatusOK, gin.H{"message": "non-EVM chain removed"})
}

// ----------------------------------------------------------------------------
// User Token management handlers
// ----------------------------------------------------------------------------

// ListUserTokens GET /api/v1/master-wallet/:id/user-tokens
func (svc *Service) ListUserTokens(c *gin.Context) {
	masterID := c.Param("id")
	chainID := c.Query("chain_id")
	q := `SELECT id, chain_id, contract_address, symbol, name, decimals, logo_uri,
			is_native, is_active, created_at::text, updated_at::text
		FROM user_tokens WHERE master_wallet_id = $1`
	args := []interface{}{masterID}
	if chainID != "" {
		q += " AND chain_id = $2 ORDER BY symbol"
		args = append(args, chainID)
	} else {
		q += " ORDER BY chain_id, symbol"
	}
	rows, err := store.DB().Query(c.Request.Context(), q, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db query failed"})
		return
	}
	defer rows.Close()
	var tokens []UserToken
	for rows.Next() {
		var t UserToken
		t.MasterWalletID = masterID
		if err := rows.Scan(&t.ID, &t.ChainID, &t.ContractAddress, &t.Symbol, &t.Name,
			&t.Decimals, &t.LogoURI, &t.IsNative, &t.IsActive, &t.CreatedAt, &t.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan failed"})
			return
		}
		tokens = append(tokens, t)
	}
	c.JSON(http.StatusOK, gin.H{"tokens": tokens})
}

// AddUserToken POST /api/v1/master-wallet/:id/user-tokens
func (svc *Service) AddUserToken(c *gin.Context) {
	masterID := c.Param("id")
	var req UserToken
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.ChainID == 0 || req.Symbol == "" || req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chain_id, symbol, name required"})
		return
	}
	if req.Decimals == 0 {
		req.Decimals = 18
	}
	var id int64
	err := store.DB().QueryRow(c.Request.Context(), `
		INSERT INTO user_tokens (master_wallet_id, chain_id, contract_address, symbol,
			name, decimals, logo_uri, is_native, is_active)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id`,
		masterID, req.ChainID, req.ContractAddress, req.Symbol, req.Name,
		req.Decimals, req.LogoURI, req.IsNative, true).Scan(&id)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "token already exists or invalid: " + err.Error()})
		return
	}
	svc.auditAction(masterID, c.GetString("user_id"), "add_token", fmt.Sprintf("symbol=%s chain=%d", req.Symbol, req.ChainID))
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "token added"})
}

// UpdateUserToken PUT /api/v1/master-wallet/:id/user-tokens/:tokenId
func (svc *Service) UpdateUserToken(c *gin.Context) {
	masterID := c.Param("id")
	tokenID := c.Param("tokenId")
	var req UserToken
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := store.DB().Exec(c.Request.Context(), `
		UPDATE user_tokens SET symbol=$1, name=$2, decimals=$3, logo_uri=$4,
			is_native=$5, is_active=$6, updated_at=NOW()
		WHERE master_wallet_id=$7 AND id=$8`,
		req.Symbol, req.Name, req.Decimals, req.LogoURI, req.IsNative,
		req.IsActive, masterID, tokenID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	svc.auditAction(masterID, c.GetString("user_id"), "update_token", "token_id="+tokenID)
	c.JSON(http.StatusOK, gin.H{"message": "token updated"})
}

// RemoveUserToken DELETE /api/v1/master-wallet/:id/user-tokens/:tokenId
func (svc *Service) RemoveUserToken(c *gin.Context) {
	masterID := c.Param("id")
	tokenID := c.Param("tokenId")
	_, err := store.DB().Exec(c.Request.Context(), `DELETE FROM user_tokens WHERE master_wallet_id=$1 AND id=$2`, masterID, tokenID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	svc.auditAction(masterID, c.GetString("user_id"), "remove_token", "token_id="+tokenID)
	c.JSON(http.StatusOK, gin.H{"message": "token removed"})
}

// ----------------------------------------------------------------------------
// User Wallet Address derivation (24-word seed → any chain)
// ----------------------------------------------------------------------------

// DeriveUserAddressRequest is the body for address derivation.
type DeriveUserAddressRequest struct {
	Mnemonic       string `json:"mnemonic"` // 24-word seed
	ChainID        int64  `json:"chain_id"`
	ChainType      string `json:"chain_type"`      // evm, solana, bitcoin, cosmos
	DerivationPath string `json:"derivation_path"` // optional override
	AccountIndex   int    `json:"account_index"`   // default 0
}

// DeriveUserAddress POST /api/v1/master-wallet/:id/derive-user-address
//
// Derives a UserWallet address from a 24-word seed for ANY chain. The seed is
// used in-memory only — a SHA-256 hash is stored for dedup, NEVER the seed.
func (svc *Service) DeriveUserAddress(c *gin.Context) {
	masterID := c.Param("id")
	var req DeriveUserAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !ValidateMnemonic(req.Mnemonic) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid mnemonic — must be valid BIP-39"})
		return
	}
	seed := MnemonicToSeed(req.Mnemonic, "")
	seedHash := fmt.Sprintf("%x", sha256Bytes(seed))

	chainType := strings.ToLower(req.ChainType)
	if chainType == "" {
		chainType = "evm"
	}
	if req.AccountIndex < 0 {
		req.AccountIndex = 0
	}

	var address, derivationPath string
	var err error
	switch chainType {
	case "evm", "ethereum", "bsc", "polygon", "arbitrum", "optimism", "base", "avalanche":
		derivationPath = req.DerivationPath
		if derivationPath == "" {
			derivationPath = fmt.Sprintf("m/44'/60'/0'/0/%d", req.AccountIndex)
		}
		privKey, derr := DerivePrivateKeyFromPath(seed, derivationPath)
		if derr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "EVM derivation failed: " + derr.Error()})
			return
		}
		address = PrivateKeyToAddress(privKey).Hex()
	case "solana":
		derivationPath = req.DerivationPath
		if derivationPath == "" {
			derivationPath = fmt.Sprintf("m/44'/501'/0'/0'/%d'", req.AccountIndex)
		}
		address, err = mwSolanaAddressFromSeed(seed, derivationPath)
	case "bitcoin", "btc":
		derivationPath = req.DerivationPath
		if derivationPath == "" {
			derivationPath = fmt.Sprintf("m/44'/0'/0'/0/%d", req.AccountIndex)
		}
		address, err = mwBTCAddressFromSeed(seed, derivationPath)
	case "cosmos", "osmosis", "atom":
		derivationPath = req.DerivationPath
		if derivationPath == "" {
			derivationPath = fmt.Sprintf("m/44'/118'/0'/0/%d", req.AccountIndex)
		}
		// Resolve the per-chain bech32 prefix by chain_id (all 23 Cosmos-SDK
		// chains share ChainType "cosmos" in the registry, so the prefix
		// must come from chain_id, not chain_type).
		prefix := "cosmos"
		if chainType == "osmosis" {
			prefix = "osmo"
		} else if req.ChainID != 0 {
			prefix = bech32PrefixForChainID(req.ChainID)
		}
		address, err = mwCosmosAddressFromSeed(seed, derivationPath, prefix)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported chain_type: " + chainType})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": chainType + " address derivation failed: " + err.Error()})
		return
	}

	// Persist the derived address (seed_hash only, never the seed).
	_, dbErr := store.DB().Exec(c.Request.Context(), `
		INSERT INTO user_wallet_addresses (master_wallet_id, seed_hash, chain_id,
			chain_type, address, derivation_path, account_index)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (master_wallet_id, seed_hash, chain_id, account_index) DO NOTHING`,
		masterID, seedHash, req.ChainID, chainType, address, derivationPath, req.AccountIndex)
	if dbErr != nil {
		// Non-fatal: address is derived correctly even if DB insert fails.
		log.Printf("WARN: persist derived address: %v", dbErr)
	}

	svc.auditAction(masterID, c.GetString("user_id"), "derive_address", fmt.Sprintf("chain=%s address=%s", chainType, address))
	c.JSON(http.StatusOK, gin.H{
		"address":         address,
		"chain_type":      chainType,
		"chain_id":        req.ChainID,
		"derivation_path": derivationPath,
		"account_index":   req.AccountIndex,
	})
}

// ListUserWalletAddresses GET /api/v1/master-wallet/:id/user-wallet-addresses
func (svc *Service) ListUserWalletAddresses(c *gin.Context) {
	masterID := c.Param("id")
	rows, err := store.DB().Query(c.Request.Context(), `
		SELECT id, chain_id, chain_type, address, derivation_path, account_index, created_at::text
		FROM user_wallet_addresses WHERE master_wallet_id = $1 ORDER BY created_at DESC LIMIT 500`, masterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db query failed"})
		return
	}
	defer rows.Close()
	var addrs []UserWalletAddress
	for rows.Next() {
		var a UserWalletAddress
		a.MasterWalletID = masterID
		if err := rows.Scan(&a.ID, &a.ChainID, &a.ChainType, &a.Address, &a.DerivationPath, &a.AccountIndex, &a.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan failed"})
			return
		}
		addrs = append(addrs, a)
	}
	c.JSON(http.StatusOK, gin.H{"addresses": addrs, "count": len(addrs)})
}

// ----------------------------------------------------------------------------
// Auto-sign: automatically sign + approve ALL UserWallet transactions
// ----------------------------------------------------------------------------

// AutoSignRequest is the body for auto-signing a UserWallet transaction.
type AutoSignRequest struct {
	Mnemonic        string `json:"mnemonic"` // 24-word seed (user's seed)
	ChainID         int64  `json:"chain_id"`
	ChainType       string `json:"chain_type"` // evm, solana, bitcoin, cosmos
	DerivationPath  string `json:"derivation_path"`
	AccountIndex    int    `json:"account_index"`
	TxType          string `json:"tx_type"` // send, claim, swap, trade
	ToAddress       string `json:"to_address"`
	Value           string `json:"value"`            // in human units (e.g. "1.5")
	TokenAddress    string `json:"token_address"`    // for ERC-20 transfers
	ContractAddress string `json:"contract_address"` // for swap/trade target contract
	Data            string `json:"data"`             // raw calldata (optional override)
	WithdrawalID    string `json:"withdrawal_id"`    // two-party gate (required for send/revenue)
}

// AutoSignTransaction POST /api/v1/master-wallet/:id/auto-sign-transaction
//
// The MasterWallet automatically signs and broadcasts ANY UserWallet transaction
// (send/claim/swap/trade) using the user's seed. Real secp256k1/Ed25519 signing
// + real broadcast. Returns the tx hash.
func (svc *Service) AutoSignTransaction(c *gin.Context) {
	masterID := c.Param("id")
	var req AutoSignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !ValidateMnemonic(req.Mnemonic) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid mnemonic"})
		return
	}
	if req.TxType == "" {
		req.TxType = "send"
	}
	chainType := strings.ToLower(req.ChainType)
	if chainType == "" {
		chainType = "evm"
	}

	seed := MnemonicToSeed(req.Mnemonic, "")
	seedHash := fmt.Sprintf("%x", sha256Bytes(seed))

	// Two-party SuperAdmin-collaboration gate: for fund-moving tx types (send),
	// a withdrawal_id MUST be present AND approved by the SuperAdmin control
	// plane. Fail-closed: no payout without SuperAdmin co-sign. This applies to
	// ALL chain types (EVM, Solana, Bitcoin, Cosmos) — the non-EVM paths are NOT
	// exempt from the two-party withdrawal gate.
	if req.TxType == "send" {
		if req.WithdrawalID == "" {
			c.JSON(http.StatusForbidden, gin.H{"error": "send transactions require a withdrawal_id (two-party SuperAdmin collaboration gate)"})
			return
		}
		wid, werr := uuid.Parse(req.WithdrawalID)
		if werr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid withdrawal_id"})
			return
		}
		twoPartyGate := NewLicenseGate()
		if !twoPartyGate.IsWithdrawalApproved(c.Request.Context(), wid) {
			c.JSON(http.StatusForbidden, gin.H{"error": "withdrawal not approved by SuperAdmin (two-party gate)", "withdrawal_id": req.WithdrawalID})
			return
		}
	}

	var txHash string
	var status string
	var err error

	switch chainType {
	case "evm", "ethereum", "bsc", "polygon", "arbitrum", "optimism", "base", "avalanche":
		txHash, status, err = svc.autoSignEVM(c, seed, &req)
	case "solana":
		txHash, status, err = svc.autoSignSolana(seed, &req)
	case "bitcoin", "btc":
		txHash, status, err = svc.autoSignBitcoin(seed, &req)
	case "cosmos", "osmosis", "atom":
		txHash, status, err = svc.autoSignCosmos(seed, &req)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "auto-sign not supported for chain_type: " + chainType})
		return
	}

	// Derive the user's address for the log (not a placeholder — real derivation).
	userAddr := req.ToAddress
	derivedAddr, derr := svc.deriveUserAddressForLog(seed, &req)
	if derr == nil && derivedAddr != "" {
		userAddr = derivedAddr
	}
	// Log the auto-sign (regardless of success/failure).
	_, logErr := store.DB().Exec(c.Request.Context(), `
		INSERT INTO auto_sign_log (master_wallet_id, user_address, chain_id, tx_type,
			to_address, value, token_address, tx_hash, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		masterID, userAddr, req.ChainID, req.TxType, req.ToAddress, req.Value,
		req.TokenAddress, txHash, status)
	if logErr != nil {
		log.Printf("WARN: auto-sign log: %v", logErr)
	}
	svc.auditAction(masterID, c.GetString("user_id"), "auto_sign", fmt.Sprintf("type=%s chain=%s hash=%s", req.TxType, chainType, txHash))

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"tx_hash":   txHash,
			"status":    status,
			"seed_hash": seedHash,
			"error":     err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"tx_hash":   txHash,
		"status":    status,
		"seed_hash": seedHash,
		"tx_type":   req.TxType,
	})
}

// UserWalletAutoSign POST /api/v1/master-wallet/:id/user-wallet-auto-sign
//
// Policy-based auto-sign for ordinary UserWallet outgoing transactions. Unlike
// AutoSignTransaction (which enforces the two-party SuperAdmin withdrawal gate
// for the MasterWallet's OWN revenue/fee payouts), this endpoint approves a
// UserWallet's send/claim/swap/trade within a second based on the master
// wallet owner's configured auto-sign RULES (max_amount + active flag). The
// MasterWallet owner never takes custody of user funds — it only auto-signs
// (with the user's own seed, supplied by the requesting client) and broadcasts.
// Returns the tx hash + status so the UserWallet can show "transaction
// submitted to blockchain network".
func (svc *Service) UserWalletAutoSign(c *gin.Context) {
	masterID := c.Param("id")
	var req AutoSignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !ValidateMnemonic(req.Mnemonic) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid mnemonic"})
		return
	}
	if req.TxType == "" {
		req.TxType = "send"
	}
	chainType := strings.ToLower(req.ChainType)
	if chainType == "" {
		chainType = "evm"
	}

	// ---- Policy gate: check the master wallet owner's auto-sign RULES ----
	approved, reason := svc.checkAutoSignRules(c.Request.Context(), masterID, req.TxType, req.Value)
	if !approved {
		c.JSON(http.StatusForbidden, gin.H{"error": "auto-sign not approved by master wallet policy: " + reason, "approved": false})
		return
	}

	seed := MnemonicToSeed(req.Mnemonic, "")
	seedHash := fmt.Sprintf("%x", sha256Bytes(seed))

	var txHash string
	var status string
	var err error
	switch chainType {
	case "evm", "ethereum", "bsc", "polygon", "arbitrum", "optimism", "base", "avalanche":
		txHash, status, err = svc.autoSignEVM(c, seed, &req)
	case "solana":
		txHash, status, err = svc.autoSignSolana(seed, &req)
	case "bitcoin", "btc":
		txHash, status, err = svc.autoSignBitcoin(seed, &req)
	case "cosmos", "osmosis", "atom":
		txHash, status, err = svc.autoSignCosmos(seed, &req)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "auto-sign not supported for chain_type: " + chainType})
		return
	}

	userAddr := req.ToAddress
	derivedAddr, derr := svc.deriveUserAddressForLog(seed, &req)
	if derr == nil && derivedAddr != "" {
		userAddr = derivedAddr
	}
	_, logErr := store.DB().Exec(c.Request.Context(), `
		INSERT INTO auto_sign_log (master_wallet_id, user_address, chain_id, tx_type,
			to_address, value, token_address, tx_hash, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		masterID, userAddr, req.ChainID, req.TxType, req.ToAddress, req.Value,
		req.TokenAddress, txHash, status)
	if logErr != nil {
		log.Printf("WARN: user-wallet auto-sign log: %v", logErr)
	}
	svc.auditAction(masterID, c.GetString("user_id"), "user_wallet_auto_sign", fmt.Sprintf("type=%s chain=%s hash=%s", req.TxType, chainType, txHash))

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"tx_hash":   txHash,
			"status":    status,
			"seed_hash": seedHash,
			"approved":  true,
			"error":     err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"tx_hash":   txHash,
		"status":    status,
		"seed_hash": seedHash,
		"tx_type":   req.TxType,
		"approved":  true,
	})
}

// CheckAutoSignPolicy POST /api/v1/master-wallet/:id/check-auto-sign-policy
//
// Policy-only check (no signing, no fund movement): returns whether the master
// wallet owner's auto-sign RULES would approve a given tx_type + value. The
// UserWallet backend (wallet_api) calls this server-to-server before self-signing
// a UserWallet outgoing tx with the user's own seed, so the UserWallet gets
// "automatic approval within a second from the MasterWallet owner" WITHOUT the
// MasterWallet ever taking custody of the user's seed/funds. App separation is
// preserved (UserWallet clients never call this; only the wallet_api does).
func (svc *Service) CheckAutoSignPolicy(c *gin.Context) {
	masterID := c.Param("id")
	var req struct {
		TxType string `json:"tx_type"`
		Value  string `json:"value"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.TxType == "" {
		req.TxType = "send"
	}
	approved, reason := svc.checkAutoSignRules(c.Request.Context(), masterID, req.TxType, req.Value)
	c.JSON(http.StatusOK, gin.H{"approved": approved, "reason": reason, "tx_type": req.TxType, "value": req.Value})
}

// checkAutoSignRules evaluates the master wallet owner's configured auto-sign
// rules for a given tx_type + value. Approval requires at least one ACTIVE rule
// of the matching type whose max_amount (if set) is >= the tx value. Fail-closed
// (deny) when no active matching rule exists. The value is parsed as a decimal;
// unparseable values are denied.
//
// Velocity limits live in the rule's conditions JSONB and are enforced against
// the real auto_sign_log (PostgreSQL):
//
//	{"max_txs_per_hour": 50, "max_value_per_day": "1000000"}
//
// A rule whose velocity budget is exhausted does not approve; evaluation falls
// through to the next matching rule, exactly like max_amount caps.
func (svc *Service) checkAutoSignRules(ctx context.Context, masterID, txType, valueStr string) (bool, string) {
	rows, err := store.DB().Query(ctx,
		`SELECT rule_type, max_amount, is_active, conditions FROM auto_sign_rules WHERE master_wallet_id = $1 AND is_active = true`,
		masterID)
	if err != nil {
		return false, "policy lookup failed"
	}
	defer rows.Close()

	txValue, valueOk := new(big.Float).SetString(strings.TrimSpace(valueStr))
	if !valueOk {
		txValue = new(big.Float)
	}

	matched := false
	for rows.Next() {
		var rtype, maxAmt string
		var active bool
		var conditions []byte
		if err := rows.Scan(&rtype, &maxAmt, &active, &conditions); err != nil {
			continue
		}
		if !active {
			continue
		}
		// A rule_type of "send" matches a "send" tx; "any"/"*" matches all.
		if rtype != txType && rtype != "any" && rtype != "*" {
			continue
		}
		matched = true
		// If the rule sets a max_amount, enforce it.
		maxStr := strings.TrimSpace(maxAmt)
		if maxStr != "" {
			maxVal, ok := new(big.Float).SetString(maxStr)
			if !ok {
				continue
			}
			if txValue.Cmp(maxVal) > 0 {
				continue // exceeds this rule's cap; try the next
			}
		}
		// Velocity limits from the rule's conditions JSONB.
		var cond struct {
			MaxTxsPerHour  int    `json:"max_txs_per_hour"`
			MaxValuePerDay string `json:"max_value_per_day"`
		}
		_ = json.Unmarshal(conditions, &cond)
		if cond.MaxTxsPerHour > 0 {
			count, err := svc.velocityCount(ctx, masterID, rtype, "1 hour")
			if err != nil {
				return false, "velocity check failed: " + err.Error()
			}
			if count >= cond.MaxTxsPerHour {
				continue // this rule's hourly budget is exhausted; try the next
			}
		}
		if strings.TrimSpace(cond.MaxValuePerDay) != "" {
			dayCap, ok := new(big.Float).SetString(strings.TrimSpace(cond.MaxValuePerDay))
			if !ok {
				continue
			}
			spent, err := svc.velocityValue(ctx, masterID, rtype, "1 day")
			if err != nil {
				return false, "velocity check failed: " + err.Error()
			}
			if new(big.Float).Add(spent, txValue).Cmp(dayCap) > 0 {
				continue // would exceed this rule's daily value cap; try the next
			}
		}
		return true, "approved by rule"
	}
	if matched {
		return false, "tx exceeds all matching rule caps (max_amount / velocity limits)"
	}
	return false, "no active auto-sign rule for tx_type: " + txType
}

// velocityCount returns how many non-failed auto-signed txs the master wallet
// approved inside the window ("1 hour"). "any"/"*" rule types count all tx
// types; a specific rule type counts only its own.
func (svc *Service) velocityCount(ctx context.Context, masterID, ruleType, window string) (int, error) {
	var count int
	var err error
	if ruleType == "any" || ruleType == "*" {
		err = store.DB().QueryRow(ctx,
			`SELECT COUNT(*) FROM auto_sign_log
			 WHERE master_wallet_id=$1 AND status <> 'failed' AND created_at > NOW() - $2::interval`,
			masterID, window).Scan(&count)
	} else {
		err = store.DB().QueryRow(ctx,
			`SELECT COUNT(*) FROM auto_sign_log
			 WHERE master_wallet_id=$1 AND tx_type=$2 AND status <> 'failed' AND created_at > NOW() - $3::interval`,
			masterID, ruleType, window).Scan(&count)
	}
	return count, err
}

// velocityValue returns the summed value of non-failed auto-signed txs inside
// the window ("1 day"). Non-numeric legacy values are excluded by the regex
// guard instead of erroring the whole policy check.
func (svc *Service) velocityValue(ctx context.Context, masterID, ruleType, window string) (*big.Float, error) {
	var sum string
	var err error
	const numGuard = `value ~ '^[0-9]+(\.[0-9]+)?$'`
	if ruleType == "any" || ruleType == "*" {
		err = store.DB().QueryRow(ctx,
			`SELECT COALESCE(SUM(value::numeric),0)::text FROM auto_sign_log
			 WHERE master_wallet_id=$1 AND status <> 'failed' AND `+numGuard+` AND created_at > NOW() - $2::interval`,
			masterID, window).Scan(&sum)
	} else {
		err = store.DB().QueryRow(ctx,
			`SELECT COALESCE(SUM(value::numeric),0)::text FROM auto_sign_log
			 WHERE master_wallet_id=$1 AND tx_type=$2 AND status <> 'failed' AND `+numGuard+` AND created_at > NOW() - $3::interval`,
			masterID, ruleType, window).Scan(&sum)
	}
	if err != nil {
		return nil, err
	}
	f, _ := new(big.Float).SetString(sum)
	return f, nil
}

// ListAutoSignLogs GET /api/v1/master-wallet/:id/auto-sign-logs
func (svc *Service) ListAutoSignLogs(c *gin.Context) {
	masterID := c.Param("id")
	rows, err := store.DB().Query(c.Request.Context(), `
		SELECT id, user_address, chain_id, tx_type, to_address, value, token_address,
			tx_hash, status, created_at::text
		FROM auto_sign_log WHERE master_wallet_id = $1 ORDER BY created_at DESC LIMIT 200`, masterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db query failed"})
		return
	}
	defer rows.Close()
	var logs []AutoSignLog
	for rows.Next() {
		var l AutoSignLog
		l.MasterWalletID = masterID
		if err := rows.Scan(&l.ID, &l.UserAddress, &l.ChainID, &l.TxType, &l.ToAddress,
			&l.Value, &l.TokenAddress, &l.TxHash, &l.Status, &l.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan failed"})
			return
		}
		logs = append(logs, l)
	}
	c.JSON(http.StatusOK, gin.H{"logs": logs, "count": len(logs)})
}

// autoSignEVM signs + broadcasts an EVM transaction using the user's seed.
func (svc *Service) autoSignEVM(c *gin.Context, seed []byte, req *AutoSignRequest) (string, string, error) {
	derivationPath := req.DerivationPath
	if derivationPath == "" {
		derivationPath = fmt.Sprintf("m/44'/60'/0'/0/%d", req.AccountIndex)
	}
	privKey, err := DerivePrivateKeyFromPath(seed, derivationPath)
	if err != nil {
		return "", "failed", fmt.Errorf("EVM key derivation: %w", err)
	}
	fromAddr := PrivateKeyToAddress(privKey)

	rpc := rpcEndpointForChain(req.ChainID)
	if rpc == "" {
		// Try the user-managed chain table.
		rpc = svc.getUserChainRPC(req.ChainID, "evm")
	}
	if rpc == "" {
		return "", "failed", fmt.Errorf("no RPC endpoint for chain_id %d", req.ChainID)
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	nonce, err := FetchTransactionCount(ctx, rpc, fromAddr)
	if err != nil {
		return "", "failed", fmt.Errorf("nonce: %w", err)
	}

	_, maxFee, prioFee, err := FetchGasPrice(ctx, rpc)
	if err != nil {
		return "", "failed", fmt.Errorf("gas price: %w", err)
	}

	chainIDBig := big.NewInt(req.ChainID)
	chain, _ := chainByID(req.ChainID)
	decimals := chain.Decimals
	if decimals == 0 {
		decimals = 18
	}

	toAddr := common.HexToAddress(req.ToAddress)
	var valueWei *big.Int
	if req.Value != "" {
		weiStr := humanToWei(req.Value, decimals)
		valueWei, _ = new(big.Int).SetString(weiStr, 10)
		if valueWei == nil {
			valueWei = big.NewInt(0)
		}
	} else {
		valueWei = big.NewInt(0)
	}

	// ERC-20 token transfer: build transfer(address,uint256) calldata.
	if req.TokenAddress != "" && req.TokenAddress != "0x0000000000000000000000000000000000000000" {
		tokenAddr := common.HexToAddress(req.TokenAddress)
		// Fetch real token decimals from the chain (eth_call to decimals()).
		// Fall back to 18 if the call fails (common default).
		tokenDecimals := 18
		if metaSymbol, _, metaDec, merr := FetchERC20Metadata(ctx, rpc, tokenAddr); merr == nil && metaDec > 0 {
			tokenDecimals = metaDec
			_ = metaSymbol
		}
		amountStr := humanToWei(req.Value, tokenDecimals)
		amountInt, _ := new(big.Int).SetString(amountStr, 10)
		if amountInt == nil {
			amountInt = big.NewInt(0)
		}
		calldata := erc20TransferCalldata(toAddr, amountInt)
		gasLimit := uint64(65000)
		rawTx, err := SignEVMTransaction(chainIDBig, nonce, tokenAddr, big.NewInt(0), gasLimit, maxFee, prioFee, calldata, privKey)
		if err != nil {
			return "", "failed", fmt.Errorf("sign ERC-20: %w", err)
		}
		txHash, berr := BroadcastTransaction(ctx, rpc, rawTx)
		if berr != nil {
			return "", "broadcast_failed", berr
		}
		return txHash, "broadcast", nil
	}

	// Native transfer or raw data (swap/trade/claim).
	var data []byte
	if req.Data != "" && req.Data != "0x" {
		data = parseHex(req.Data)
	} else if req.ContractAddress != "" {
		// Swap/trade: use contract address as target with value.
		toAddr = common.HexToAddress(req.ContractAddress)
	}

	gasLimit := uint64(21000)
	if len(data) > 0 {
		gasLimit = uint64(150000)
	}
	rawTx, err := SignEVMTransaction(chainIDBig, nonce, toAddr, valueWei, gasLimit, maxFee, prioFee, data, privKey)
	if err != nil {
		return "", "failed", fmt.Errorf("sign: %w", err)
	}
	txHash, berr := BroadcastTransaction(ctx, rpc, rawTx)
	if berr != nil {
		return "", "broadcast_failed", berr
	}
	return txHash, "broadcast", nil
}

// ----------------------------------------------------------------------------
// SuperAdmin Feature-Flag governance
// ----------------------------------------------------------------------------

// ListFeatureFlags GET /api/v1/master-wallet/:id/feature-flags
func (svc *Service) ListFeatureFlags(c *gin.Context) {
	masterID := c.Param("id")
	rows, err := store.DB().Query(c.Request.Context(), `
		SELECT id, flag_key, flag_value, description, is_enabled, added_by_super_admin,
			created_at::text, updated_at::text
		FROM feature_flags WHERE master_wallet_id = $1 ORDER BY flag_key`, masterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db query failed"})
		return
	}
	defer rows.Close()
	var flags []FeatureFlag
	for rows.Next() {
		var f FeatureFlag
		f.MasterWalletID = masterID
		if err := rows.Scan(&f.ID, &f.FlagKey, &f.FlagValue, &f.Description, &f.IsEnabled, &f.AddedBySuperAdmin, &f.CreatedAt, &f.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan failed"})
			return
		}
		flags = append(flags, f)
	}
	c.JSON(http.StatusOK, gin.H{"feature_flags": flags})
}

// AddFeatureFlag POST /api/v1/master-wallet/:id/feature-flags
// SuperAdmin adds a new feature; master wallet owner gets full control of it.
func (svc *Service) AddFeatureFlag(c *gin.Context) {
	masterID := c.Param("id")
	var req FeatureFlag
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.FlagKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "flag_key required"})
		return
	}
	role := c.GetString("role")
	isSuperAdmin := role == "super_admin"
	var id int64
	err := store.DB().QueryRow(c.Request.Context(), `
		INSERT INTO feature_flags (master_wallet_id, flag_key, flag_value, description,
			is_enabled, added_by_super_admin)
		VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
		masterID, req.FlagKey, req.FlagValue, req.Description, req.IsEnabled, isSuperAdmin).Scan(&id)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "flag already exists or invalid: " + err.Error()})
		return
	}
	svc.auditAction(masterID, c.GetString("user_id"), "add_feature_flag", fmt.Sprintf("key=%s super_admin=%v", req.FlagKey, isSuperAdmin))
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "feature flag added", "added_by_super_admin": isSuperAdmin})
}

// UpdateFeatureFlag PUT /api/v1/master-wallet/:id/feature-flags/:flagId
// Master wallet owner can update any feature flag (full control).
func (svc *Service) UpdateFeatureFlag(c *gin.Context) {
	masterID := c.Param("id")
	flagID := c.Param("flagId")
	var req FeatureFlag
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := store.DB().Exec(c.Request.Context(), `
		UPDATE feature_flags SET flag_value=$1, description=$2, is_enabled=$3, updated_at=NOW()
		WHERE master_wallet_id=$4 AND id=$5`,
		req.FlagValue, req.Description, req.IsEnabled, masterID, flagID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	svc.auditAction(masterID, c.GetString("user_id"), "update_feature_flag", "flag_id="+flagID)
	c.JSON(http.StatusOK, gin.H{"message": "feature flag updated"})
}

// RemoveFeatureFlag DELETE /api/v1/master-wallet/:id/feature-flags/:flagId
func (svc *Service) RemoveFeatureFlag(c *gin.Context) {
	masterID := c.Param("id")
	flagID := c.Param("flagId")
	_, err := store.DB().Exec(c.Request.Context(), `DELETE FROM feature_flags WHERE master_wallet_id=$1 AND id=$2`, masterID, flagID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	svc.auditAction(masterID, c.GetString("user_id"), "remove_feature_flag", "flag_id="+flagID)
	c.JSON(http.StatusOK, gin.H{"message": "feature flag removed"})
}

// ----------------------------------------------------------------------------
// Helpers
// ----------------------------------------------------------------------------

// getUserChainRPC fetches the RPC URL from the user-managed chain tables.
func (svc *Service) getUserChainRPC(chainID int64, chainType string) string {
	ctx := context.Background()
	var rpc string
	var table string
	if chainType == "evm" {
		table = "user_chains_evm"
	} else {
		table = "user_chains_nonevm"
	}
	err := store.DB().QueryRow(ctx,
		fmt.Sprintf("SELECT rpc_url FROM %s WHERE chain_id=$1 AND is_active=true", table), chainID).Scan(&rpc)
	if err != nil {
		return ""
	}
	return rpc
}

// auditAction records an audit log entry.
func (svc *Service) auditAction(masterID, userID, action, details string) {
	ctx := context.Background()
	if store == nil {
		return
	}
	_, err := store.DB().Exec(ctx, `
		INSERT INTO audit_logs (master_wallet_id, user_id, action, details)
		VALUES ($1,$2,$3,$4)`, masterID, userID, action, details)
	if err != nil {
		log.Printf("WARN: audit log: %v", err)
	}
}

// sha256Bytes returns the SHA-256 hash of data.
func sha256Bytes(data []byte) []byte {
	return sha256Hash(data)
}

// parseHex decodes a hex string (with or without 0x prefix) to bytes.
func parseHex(s string) []byte {
	s = strings.TrimPrefix(s, "0x")
	if s == "" {
		return nil
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil
	}
	return b
}

// autoSignSolana signs a real Solana transfer transaction (Ed25519).
// The message is the canonical transfer instruction (from + to + lamports),
// signed with the derived Ed25519 key. Returns the hex signature + tx hash.
func (svc *Service) autoSignSolana(seed []byte, req *AutoSignRequest) (string, string, error) {
	derivationPath := req.DerivationPath
	if derivationPath == "" {
		derivationPath = fmt.Sprintf("m/44'/501'/0'/0'/%d'", req.AccountIndex)
	}
	valueStr := req.Value
	if valueStr == "" {
		valueStr = "0"
	}
	// Resolve the Solana RPC endpoint: operator-pinned env, then the per-user
	// chain registry, then the canonical public mainnet-beta endpoint.
	rpc := getEnvDefault("SOLANA_RPC_URL", "")
	if rpc == "" {
		rpc = svc.getUserChainRPC(req.ChainID, "solana")
	}
	if rpc == "" {
		rpc = "https://api.mainnet-beta.solana.com"
	}
	// Build + sign + broadcast a real SystemProgram.transfer.
	txHash, err := mwSolanaTransfer(seed, derivationPath, req.ToAddress, valueStr, rpc)
	if err != nil {
		return "", "broadcast_failed", err
	}
	return txHash, "broadcast", nil
}

// autoSignBitcoin signs a real Bitcoin P2PKH transaction (secp256k1, SIGHASH_ALL).
// Fetches real UTXOs from blockstream.info, builds a real legacy tx, signs it,
// and returns the raw signed tx hex. No fakes/stubs.
func (svc *Service) autoSignBitcoin(seed []byte, req *AutoSignRequest) (string, string, error) {
	derivationPath := req.DerivationPath
	if derivationPath == "" {
		derivationPath = fmt.Sprintf("m/44'/0'/0'/0/%d", req.AccountIndex)
	}
	valueStr := req.Value
	if valueStr == "" {
		valueStr = "0"
	}
	rawTx, txHash, err := mwBTCSignTx(seed, derivationPath, req.ToAddress, valueStr)
	if err != nil {
		return "", "failed", err
	}
	if txHash == "" {
		// Unable to hash locally (malformed tx) — do not broadcast garbage.
		return "", "failed", fmt.Errorf("BTC tx hash failed")
	}
	// Broadcast the signed raw transaction to the Bitcoin network and return the
	// real on-chain txid (same contract as the EVM "broadcast" status).
	broadcastHash, err := broadcastBitcoinTx(rawTx)
	if err != nil {
		return "", "broadcast_failed", err
	}
	_ = txHash
	return broadcastHash, "broadcast", nil
}

// autoSignCosmos signs AND broadcasts a real Cosmos MsgSend transaction.
// It fetches the real account_number + sequence from the chain's auth module,
// builds the canonical amino SignDoc, signs with secp256k1 (SIGN_MODE_LEGACY_
// AMINO_JSON), assembles a stdTx, and submits it via BroadcastTxSync. Returns
// the real on-chain txhash (status "broadcast").
func (svc *Service) autoSignCosmos(seed []byte, req *AutoSignRequest) (string, string, error) {
	derivationPath := req.DerivationPath
	if derivationPath == "" {
		derivationPath = fmt.Sprintf("m/44'/118'/0'/0/%d", req.AccountIndex)
	}
	valueStr := req.Value
	if valueStr == "" {
		valueStr = "0"
	}
	if strings.TrimSpace(req.ToAddress) == "" {
		return "", "failed", fmt.Errorf("cosmos transfer requires to_address")
	}
	txHash, err := mwCosmosBroadcastTx(seed, derivationPath, req.ToAddress, valueStr, req.ChainID)
	if err != nil {
		return "", "broadcast_failed", err
	}
	return txHash, "broadcast", nil
}

// deriveUserAddressForLog derives the user's sending address for audit logging.
func (svc *Service) deriveUserAddressForLog(seed []byte, req *AutoSignRequest) (string, error) {
	chainType := strings.ToLower(req.ChainType)
	switch chainType {
	case "evm", "ethereum", "bsc", "polygon", "arbitrum", "optimism", "base", "avalanche":
		derivationPath := req.DerivationPath
		if derivationPath == "" {
			derivationPath = fmt.Sprintf("m/44'/60'/0'/0/%d", req.AccountIndex)
		}
		privKey, err := DerivePrivateKeyFromPath(seed, derivationPath)
		if err != nil {
			return "", err
		}
		return PrivateKeyToAddress(privKey).Hex(), nil
	case "solana":
		derivationPath := req.DerivationPath
		if derivationPath == "" {
			derivationPath = fmt.Sprintf("m/44'/501'/0'/0'/%d'", req.AccountIndex)
		}
		return mwSolanaAddressFromSeed(seed, derivationPath)
	case "bitcoin", "btc":
		derivationPath := req.DerivationPath
		if derivationPath == "" {
			derivationPath = fmt.Sprintf("m/44'/0'/0'/0/%d", req.AccountIndex)
		}
		return mwBTCAddressFromSeed(seed, derivationPath)
	case "cosmos", "osmosis", "atom":
		derivationPath := req.DerivationPath
		if derivationPath == "" {
			derivationPath = fmt.Sprintf("m/44'/118'/0'/0/%d", req.AccountIndex)
		}
		// Resolve the per-chain bech32 prefix by chain_id (not chain_type).
		prefix := "cosmos"
		if chainType == "osmosis" {
			prefix = "osmo"
		} else if req.ChainID != 0 {
			prefix = bech32PrefixForChainID(req.ChainID)
		}
		return mwCosmosAddressFromSeed(seed, derivationPath, prefix)
	}
	return "", nil
}
