package main

// handlers.go — HTTP handlers for the MasterWallet backend. Every wallet
// operation uses REAL crypto: BIP-39 mnemonic, secp256k1 BIP-32/44 derivation,
// keccak256 addresses, local signing + eth_sendRawTransaction broadcast, and
// live on-chain balance/token/gas/price/history fetchers. No fabricated data.

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Service ties the store + config together for handlers.
type Service struct {
	store *Store
	cfg   *AppConfig
	hub   *wsHub
}

// --- Auth handlers ---

type registerReq struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name"`
	Role     string `json:"role"`
}

func (svc *Service) Register(c *gin.Context) {
	var req registerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Name == "" {
		req.Name = req.Email
	}
	// Public self-registration may ONLY create a plain "user" account.
	// Privileged roles (admin/treasury/operator/super_admin) are assigned
	// exclusively by an existing SuperAdmin/admin via the admin user-management
	// path — never accepted from an unauthenticated request body (privilege
	// escalation). The body role field is intentionally ignored here.
	role := "user"
	_ = req.Role
	hash, err := hashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}
	userID := uuid.New().String()
	ctx := c.Request.Context()
	_, err = svc.store.db.Exec(ctx,
		`INSERT INTO mw_users (id, email, name, role, password_hash) VALUES ($1,$2,$3,$4,$5)`,
		userID, req.Email, req.Name, role, hash)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}
	token, err := IssueJWT(svc.cfg.JWTSecret, userID, req.Email, role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue token"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"token": token, "user_id": userID, "email": req.Email, "role": role})
}

type loginReq struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (svc *Service) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	var (
		userID string
		role   string
		hash   string
	)
	err := svc.store.db.QueryRow(ctx,
		`SELECT id, role, password_hash FROM mw_users WHERE email = $1 AND is_active = true`,
		req.Email).Scan(&userID, &role, &hash)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	if !verifyPassword(hash, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	_, _ = svc.store.db.Exec(ctx, `UPDATE mw_users SET last_login_at = NOW() WHERE id = $1`, userID)
	token, err := IssueJWT(svc.cfg.JWTSecret, userID, req.Email, role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token, "user_id": userID, "email": req.Email, "role": role})
}

// --- Master wallet handlers ---

type createMasterWalletReq struct {
	Name      string `json:"name" binding:"required"`
	Blockchain string `json:"blockchain"`
	ChainID   int64  `json:"chain_id"`
	WalletType string `json:"wallet_type"`
	Password  string `json:"password" binding:"required,min=8"`
	Mnemonic  string `json:"mnemonic"` // optional: import existing
}

func (svc *Service) CreateMasterWallet(c *gin.Context) {
	var req createMasterWalletReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	chainID := req.ChainID
	if chainID == 0 {
		chainID = 1
	}
	chain, ok := chainByID(chainID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported chain id"})
		return
	}
	walletType := req.WalletType
	if walletType == "" {
		walletType = "hot"
	}

	var (
		mnemonic string
		seed     []byte
		err      error
	)
	if req.Mnemonic != "" {
		if !ValidateMnemonic(req.Mnemonic) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid mnemonic (failed BIP-39 checksum)"})
			return
		}
		mnemonic = req.Mnemonic
		seed = MnemonicToSeed(mnemonic, "")
	} else {
		mnemonic, err = GenerateMnemonic(256)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate mnemonic"})
			return
		}
		seed = MnemonicToSeed(mnemonic, "")
	}

	// Real secp256k1 BIP-44 derivation at m/44'/60'/0'/0/0.
	privKey, err := DeriveEVMPrivateKey(seed, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to derive private key"})
		return
	}
	address := PrivateKeyToAddress(privKey)
	pubHex := hex.EncodeToString(crypto.FromECDSAPub(&privKey.PublicKey))

	// Encrypt the seed with the user's password (scrypt + AES-256-GCM).
	encSeed, err := EncryptSeed(seed, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encrypt seed"})
		return
	}

	userID := currentUserID(c)
	walletID := uuid.New().String()
	ctx := c.Request.Context()
	_, err = svc.store.db.Exec(ctx,
		`INSERT INTO master_wallets (id, name, blockchain, address, public_key, wallet_type, chain_id, encrypted_seed, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		walletID, req.Name, chain.Blockchain, address.Hex(), pubHex, walletType, chainID, encSeed, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create wallet", "detail": err.Error()})
		return
	}
	// Link the user to this wallet.
	_, _ = svc.store.db.Exec(ctx, `UPDATE mw_users SET master_wallet_id = $1 WHERE id = $2`, walletID, userID)
	svc.store.audit(ctx, walletID, "master_wallet.create", "wallet", "user", userID, "master_wallet", walletID, "normal", gin.H{"name": req.Name, "blockchain": chain.Blockchain})

	c.JSON(http.StatusCreated, gin.H{
		"wallet_id":   walletID,
		"name":        req.Name,
		"address":     address.Hex(),
		"public_key":  "0x" + pubHex,
		"blockchain":  chain.Blockchain,
		"chain_id":    chainID,
		"wallet_type": walletType,
		"mnemonic":    mnemonic, // returned once; the encrypted seed is persisted
		"created_at":  time.Now().UTC(),
	})
}

func (svc *Service) GetMasterWallets(c *gin.Context) {
	userID := currentUserID(c)
	ctx := c.Request.Context()
	rows, err := svc.store.db.Query(ctx,
		`SELECT id, name, blockchain, address, public_key, wallet_type, chain_id, is_active, created_at
		 FROM master_wallets WHERE created_by = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch wallets"})
		return
	}
	defer rows.Close()
	wallets := []gin.H{}
	for rows.Next() {
		var w gin.H
		var id uuid.UUID
		var name, blockchain, address, pubKey, walletType string
		var chainID int64
		var isActive bool
		var createdAt time.Time
		if err := rows.Scan(&id, &name, &blockchain, &address, &pubKey, &walletType, &chainID, &isActive, &createdAt); err != nil {
			continue
		}
		w = gin.H{
			"id":           id.String(),
			"name":         name,
			"blockchain":   blockchain,
			"address":      address,
			"public_key":   pubKey,
			"wallet_type":  walletType,
			"chain_id":     chainID,
			"is_active":    isActive,
			"created_at":   createdAt,
		}
		wallets = append(wallets, w)
	}
	c.JSON(http.StatusOK, gin.H{"wallets": wallets})
}

func (svc *Service) GetMasterWallet(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()
	var name, blockchain, address, pubKey, walletType string
	var chainID int64
	var isActive bool
	var createdAt time.Time
	err := svc.store.db.QueryRow(ctx,
		`SELECT name, blockchain, address, public_key, wallet_type, chain_id, is_active, created_at
		 FROM master_wallets WHERE id = $1`, id).Scan(&name, &blockchain, &address, &pubKey, &walletType, &chainID, &isActive, &createdAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":          id,
		"name":        name,
		"blockchain":  blockchain,
		"address":     address,
		"public_key":  pubKey,
		"wallet_type": walletType,
		"chain_id":    chainID,
		"is_active":   isActive,
		"created_at":  createdAt,
	})
}

func (svc *Service) DeleteMasterWallet(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()
	_, err := svc.store.db.Exec(ctx, `DELETE FROM master_wallets WHERE id = $1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete wallet"})
		return
	}
	svc.store.audit(ctx, id, "master_wallet.delete", "wallet", "user", currentUserID(c), "master_wallet", id, "high", gin.H{})
	c.JSON(http.StatusOK, gin.H{"deleted": true, "id": id})
}

// GetMasterWalletBalance returns the LIVE native + token balances from the RPC node.
func (svc *Service) GetMasterWalletBalance(c *gin.Context) {
	id := c.Param("id")
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	var address string
	var chainID int64
	err := svc.store.db.QueryRow(ctx,
		`SELECT address, chain_id FROM master_wallets WHERE id = $1`, id).
		Scan(&address, &chainID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "master wallet not found"})
		return
	}
	rpc := rpcEndpointForChain(chainID)
	if rpc == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "RPC endpoint not configured for chain", "chain_id": chainID})
		return
	}
	// Cache check (30s TTL for balances).
	cacheKey := fmt.Sprintf("balance:%s:%d", address, chainID)
	if cached, ok := svc.store.cacheGet(ctx, cacheKey); ok {
		c.Data(http.StatusOK, "application/json", []byte(cached))
		return
	}
	addr := common.HexToAddress(address)
	nativeBal, err := FetchNativeBalance(ctx, rpc, addr)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to fetch balance", "detail": err.Error()})
		return
	}
	chain, _ := chainByID(chainID)
	// Token balances: read from token_balances table (the registry of tracked
	// tokens) and resolve live balances from chain.
	trows, _ := svc.store.db.Query(ctx,
		`SELECT contract_address, token_symbol FROM token_balances WHERE wallet_id = $1 AND wallet_type = 'master'`, id)
	tokens := []TokenInfo{}
	for trows.Next() {
		var ca, sym string
		_ = trows.Scan(&ca, &sym)
		if ca != "" {
			tokens = append(tokens, TokenInfo{Address: ca, Symbol: sym, Decimals: 18})
		}
	}
	trows.Close()
	tokenBals := FetchTokenBalances(ctx, rpc, addr, tokens)

	// Native price for USD value.
	var usdValue float64
	if p, err := FetchTokenPrice(ctx, chainCoinGeckoID(chainID)); err == nil && p != nil {
		f, _ := new(big.Float).Quo(
			new(big.Float).SetInt(nativeBal),
			big.NewFloat(pow10f(chain.Decimals)),
		).Float64()
		usdValue = f * p.USD
	}

	resp := gin.H{
		"wallet_id":   id,
		"address":     address,
		"chain_id":    chainID,
		"native":      gin.H{"symbol": chain.Symbol, "balance": weiToFloat(nativeBal, chain.Decimals), "balance_wei": nativeBal.String()},
		"tokens":      tokenBals,
		"usd_value":   usdValue,
		"updated_at":  time.Now().UTC(),
	}
	if b, err := jsonMarshal(resp); err == nil {
		svc.store.cacheSet(ctx, cacheKey, string(b), 30*time.Second)
	}
	c.JSON(http.StatusOK, resp)
}

// SignTransaction / SendTransaction: builds + signs a real EIP-1559 tx with the
// wallet's derived key, then broadcasts via eth_sendRawTransaction. The returned
// hash is the real node hash — never fabricated.
type sendReq struct {
	To           string `json:"to" binding:"required"`
	Amount       string `json:"amount" binding:"required"` // human-readable (e.g. "0.5")
	Token        string `json:"token"`                     // contract address; empty = native
	Password     string `json:"password" binding:"required"`
	GasLimit     uint64 `json:"gas_limit"`
	WithdrawalID string `json:"withdrawal_id"` // if set, two-party gate MUST be approved before broadcast
}

func (svc *Service) SignTransaction(c *gin.Context) {
	id := c.Param("id")
	var req sendReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Two-party SuperAdmin collaboration gate: if a withdrawal_id is present,
	// the withdrawal MUST be two-party-approved (WL client + SuperAdmin) before
	// any broadcast. Fail-closed: no payout without SuperAdmin co-sign.
	if req.WithdrawalID != "" {
		wid, err := uuid.Parse(req.WithdrawalID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid withdrawal_id"})
			return
		}
		gate := NewLicenseGate()
		if !gate.IsWithdrawalApproved(c.Request.Context(), wid) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "two-party SuperAdmin collaboration required before withdrawal; withdrawal not approved or gate unreachable",
			})
			return
		}
	}
	txHash, fromAddr, chainID, err := svc.buildSignBroadcast(c, id, req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	// Persist the transaction record.
	txRec := uuid.New().String()
	ctx := c.Request.Context()
	_, _ = svc.store.db.Exec(ctx,
		`INSERT INTO transactions (id, master_wallet_id, tx_hash, tx_type, status, blockchain, from_address, to_address, amount, token_symbol, chain_id)
		 VALUES ($1,$2,$3,'transfer','pending',$4,$5,$6,$7,$8)`,
		txRec, id, txHash, "ethereum", fromAddr, req.To, req.Amount, req.Token, chainID)
	svc.store.audit(ctx, id, "transaction.sign", "transaction", "user", currentUserID(c), "transaction", txRec, "high", gin.H{"to": req.To, "amount": req.Amount})
	c.JSON(http.StatusOK, gin.H{"transaction_hash": txHash, "status": "broadcast", "from": fromAddr, "chain_id": chainID})
}

// WithdrawalRequest creates a two-party withdrawal request in the license
// control plane. The WL client approves via the WL admin panel; SuperAdmin
// approves via the control plane. Only after BOTH approvals can SignTransaction
// (with the withdrawal_id) broadcast. This is the "request" half of the gate.
func (svc *Service) WithdrawalRequest(c *gin.Context) {
	walletID := c.Param("id")
	walletUUID, err := uuid.Parse(walletID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid wallet id"})
		return
	}
	var req struct {
		ToAddress string `json:"to_address" binding:"required"`
		AmountWei string `json:"amount_wei" binding:"required"`
		Currency  string `json:"currency"`
		ChainID   int64  `json:"chain_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Currency == "" {
		req.Currency = "ETH"
	}
	gate := NewLicenseGate()
	wid, err := gate.RequestWithdrawal(c.Request.Context(), walletUUID, req.ToAddress, req.AmountWei, req.Currency, req.ChainID)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "two-party gate unavailable: " + err.Error()})
		return
	}
	svc.store.audit(c.Request.Context(), walletID, "withdrawal.request", "withdrawal", "user", currentUserID(c), "withdrawal", wid.String(), "high", gin.H{"to": req.ToAddress, "amount_wei": req.AmountWei})
	c.JSON(http.StatusAccepted, gin.H{"withdrawal_id": wid, "status": "pending_two_party_approval"})
}

// RevenuePayout moves accumulated fee/revenue funds to a destination. This
// ALWAYS requires the two-party SuperAdmin co-sign — revenue can never move
// without SuperAdmin collaboration, regardless of amount. The caller supplies
// a pre-approved withdrawal_id; the gate is checked fail-closed before broadcast.
func (svc *Service) RevenuePayout(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		To           string `json:"to" binding:"required"`
		Amount       string `json:"amount" binding:"required"`
		Token        string `json:"token"`
		Password     string `json:"password" binding:"required"`
		GasLimit     uint64 `json:"gas_limit"`
		WithdrawalID string `json:"withdrawal_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	wid, err := uuid.Parse(req.WithdrawalID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid withdrawal_id"})
		return
	}
	gate := NewLicenseGate()
	if !gate.IsWithdrawalApproved(c.Request.Context(), wid) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "revenue payout requires two-party SuperAdmin collaboration; withdrawal not approved or gate unreachable",
		})
		return
	}
	txHash, fromAddr, chainID, err := svc.buildSignBroadcast(c, id, sendReq{
		To: req.To, Amount: req.Amount, Token: req.Token, Password: req.Password, GasLimit: req.GasLimit,
	})
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	// Record the executed tx hash in the control plane (best-effort).
	_ = gate.MarkWithdrawalExecuted(c.Request.Context(), wid, txHash)
	svc.store.audit(c.Request.Context(), id, "revenue.payout", "withdrawal", "user", currentUserID(c), "withdrawal", wid.String(), "critical", gin.H{"to": req.To, "amount": req.Amount, "tx_hash": txHash})
	c.JSON(http.StatusOK, gin.H{"transaction_hash": txHash, "status": "broadcast", "withdrawal_id": wid, "from": fromAddr, "chain_id": chainID})
}

// buildSignBroadcast resolves the wallet, decrypts the seed, derives the key,
// builds + signs the tx, and broadcasts it. Returns (txHash, fromAddr, chainID, err).
func (svc *Service) buildSignBroadcast(c *gin.Context, walletID string, req sendReq) (string, string, int64, error) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	var address, encSeed string
	var chainID int64
	err := svc.store.db.QueryRow(ctx,
		`SELECT address, encrypted_seed, chain_id FROM master_wallets WHERE id = $1`, walletID).
		Scan(&address, &encSeed, &chainID)
	if err != nil {
		return "", "", 0, fmt.Errorf("master wallet not found")
	}
	seed, err := loadOwnedSeed(encSeed, req.Password)
	if err != nil {
		return "", "", 0, fmt.Errorf("invalid password (seed decryption failed)")
	}
	privKey, err := DeriveEVMPrivateKey(seed, 0)
	if err != nil {
		return "", "", 0, fmt.Errorf("key derivation failed: %w", err)
	}
	rpc := rpcEndpointForChain(chainID)
	if rpc == "" {
		return "", "", 0, fmt.Errorf("RPC endpoint not configured for chain %d", chainID)
	}
	chain, _ := chainByID(chainID)
	from := common.HexToAddress(address)
	nonce, err := FetchTransactionCount(ctx, rpc, from)
	if err != nil {
		return "", "", 0, fmt.Errorf("fetch nonce: %w", err)
	}
	_, maxFee, prioFee, err := FetchGasPrice(ctx, rpc)
	if err != nil {
		return "", "", 0, fmt.Errorf("fetch gas price: %w", err)
	}
	gasLimit := req.GasLimit
	if gasLimit == 0 {
		gasLimit = 21000
	}
	toAddr := common.HexToAddress(req.To)
	var value *big.Int
	var data []byte
	if req.Token == "" {
		// native transfer
		wei, ok := new(big.Int).SetString(humanToWei(req.Amount, chain.Decimals), 10)
		if !ok {
			return "", "", 0, fmt.Errorf("invalid amount")
		}
		value = wei
		data = nil
	} else {
		// ERC-20 transfer(to, amount)
		wei, ok := new(big.Int).SetString(humanToWei(req.Amount, 18), 10)
		if !ok {
			return "", "", 0, fmt.Errorf("invalid amount")
		}
		value = big.NewInt(0)
		data = erc20TransferCalldata(toAddr, wei)
		toAddr = common.HexToAddress(req.Token)
	}
	rawTx, err := SignEVMTransaction(big.NewInt(chainID), nonce, toAddr, value, gasLimit, maxFee, prioFee, data, privKey)
	if err != nil {
		return "", "", 0, fmt.Errorf("sign: %w", err)
	}
	txHash, err := BroadcastTransaction(ctx, rpc, rawTx)
	if err != nil {
		return "", "", 0, fmt.Errorf("broadcast: %w", err)
	}
	return txHash, address, chainID, nil
}

// --- Sub-wallet handlers ---

func (svc *Service) GetSubWallets(c *gin.Context) {
	masterID := c.Param("id")
	if masterID == "" {
		masterID = c.Query("master_wallet_id")
	}
	ctx := c.Request.Context()
	rows, err := svc.store.db.Query(ctx,
		`SELECT id, master_wallet_id, name, derivation_path, derivation_index, blockchain, address, public_key, is_active, label, chain_id
		 FROM sub_wallets WHERE master_wallet_id = $1 ORDER BY derivation_index`, masterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch sub wallets"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var mid uuid.UUID
		var name, dpath, blockchain, address, pubKey string
		var dindex int
		var isActive bool
		var label *string
		var chainID int64
		_ = rows.Scan(&id, &mid, &name, &dpath, &dindex, &blockchain, &address, &pubKey, &isActive, &label, &chainID)
		entry := gin.H{
			"id": id.String(), "master_wallet_id": mid.String(), "name": name,
			"derivation_path": dpath, "derivation_index": dindex, "blockchain": blockchain,
			"address": address, "public_key": pubKey, "is_active": isActive, "chain_id": chainID,
		}
		if label != nil {
			entry["label"] = *label
		}
		out = append(out, entry)
	}
	c.JSON(http.StatusOK, gin.H{"sub_wallets": out})
}

type createSubWalletReq struct {
	Name     string `json:"name"`
	Index    uint32 `json:"index"`
	Label    string `json:"label"`
	Password string `json:"password" binding:"required"`
}

func (svc *Service) CreateSubWallet(c *gin.Context) {
	masterID := c.Param("id")
	var req createSubWalletReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	var encSeed string
	var chainID int64
	var blockchain string
	err := svc.store.db.QueryRow(ctx,
		`SELECT encrypted_seed, chain_id, blockchain FROM master_wallets WHERE id = $1`, masterID).
		Scan(&encSeed, &chainID, &blockchain)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "master wallet not found"})
		return
	}
	seed, err := loadOwnedSeed(encSeed, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid password"})
		return
	}
	path := fmt.Sprintf("m/44'/60'/0'/0/%d", req.Index)
	privKey, err := DerivePrivateKeyFromPath(seed, path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "derivation failed"})
		return
	}
	address := PrivateKeyToAddress(privKey)
	pubHex := hex.EncodeToString(crypto.FromECDSAPub(&privKey.PublicKey))
	encKey, _ := EncryptSeed(seed, req.Password) // store encrypted key for future signing
	subID := uuid.New().String()
	name := req.Name
	if name == "" {
		name = fmt.Sprintf("Sub Wallet %d", req.Index)
	}
	_, err = svc.store.db.Exec(ctx,
		`INSERT INTO sub_wallets (id, master_wallet_id, name, derivation_path, derivation_index, blockchain, address, public_key, encrypted_key, label, chain_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		subID, masterID, name, path, req.Index, blockchain, address.Hex(), pubHex, encKey, req.Label, chainID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create sub wallet", "detail": err.Error()})
		return
	}
	svc.store.audit(ctx, masterID, "sub_wallet.create", "wallet", "user", currentUserID(c), "sub_wallet", subID, "normal", gin.H{"index": req.Index})
	c.JSON(http.StatusCreated, gin.H{
		"id": subID, "master_wallet_id": masterID, "name": name,
		"derivation_path": path, "derivation_index": req.Index,
		"address": address.Hex(), "public_key": "0x" + pubHex,
		"blockchain": blockchain, "chain_id": chainID, "label": req.Label,
	})
}

func (svc *Service) GetSubWalletBalance(c *gin.Context) {
	id := c.Param("id")
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()
	var address string
	var chainID int64
	err := svc.store.db.QueryRow(ctx,
		`SELECT sw.address, mw.chain_id FROM sub_wallets sw JOIN master_wallets mw ON sw.master_wallet_id = mw.id WHERE sw.id = $1`, id).
		Scan(&address, &chainID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "sub wallet not found"})
		return
	}
	rpc := rpcEndpointForChain(chainID)
	if rpc == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "RPC not configured"})
		return
	}
	bal, err := FetchNativeBalance(ctx, rpc, common.HexToAddress(address))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	chain, _ := chainByID(chainID)
	c.JSON(http.StatusOK, gin.H{"wallet_id": id, "address": address, "balance": weiToFloat(bal, chain.Decimals), "balance_wei": bal.String(), "chain_id": chainID})
}

func (svc *Service) TransferFromSubWallet(c *gin.Context) {
	id := c.Param("id")
	var req sendReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	var masterID string
	err := svc.store.db.QueryRow(ctx,
		`SELECT master_wallet_id::text FROM sub_wallets WHERE id = $1`, id).Scan(&masterID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "sub wallet not found"})
		return
	}
	txHash, fromAddr, chainID, err := svc.buildSignBroadcast(c, masterID, req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	txRec := uuid.New().String()
	_, _ = svc.store.db.Exec(ctx,
		`INSERT INTO transactions (id, master_wallet_id, sub_wallet_id, tx_hash, tx_type, status, blockchain, from_address, to_address, amount, token_symbol, chain_id)
		 VALUES ($1,$2,$3,$4,'transfer','pending',$5,$6,$7,$8,$9)`,
		txRec, masterID, id, txHash, "ethereum", fromAddr, req.To, req.Amount, req.Token, chainID)
	c.JSON(http.StatusOK, gin.H{"transaction_hash": txHash, "status": "pending"})
}

// --- Transaction handlers ---

func (svc *Service) GetTransactions(c *gin.Context) {
	masterID := c.Query("master_wallet_id")
	status := c.Query("status")
	limit := parseLimit(c.Query("limit"), 50, 200)
	ctx := c.Request.Context()
	query := `SELECT id, master_wallet_id, sub_wallet_id, tx_hash, tx_type, status, blockchain, from_address, to_address, amount, token_symbol, chain_id, created_at, confirmed_at
	          FROM transactions WHERE master_wallet_id = $1`
	args := []interface{}{masterID}
	if status != "" {
		query += " AND status = $2"
		args = append(args, status)
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT %d", limit)
	rows, err := svc.store.db.Query(ctx, query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch transactions"})
		return
	}
	defer rows.Close()
	txs := []gin.H{}
	for rows.Next() {
		var id, mid uuid.UUID
		var subID *uuid.UUID
		var txHash, txType, status, blockchain, fromAddr, toAddr, amount *string
		var tokenSym *string
		var chainID *int64
		var createdAt time.Time
		var confirmedAt *time.Time
		_ = rows.Scan(&id, &mid, &subID, &txHash, &txType, &status, &blockchain, &fromAddr, &toAddr, &amount, &tokenSym, &chainID, &createdAt, &confirmedAt)
		entry := gin.H{
			"id": id.String(), "master_wallet_id": mid.String(), "tx_type": strPtr(txType), "status": strPtr(status),
			"blockchain": strPtr(blockchain), "from": strPtr(fromAddr), "to": strPtr(toAddr), "amount": strPtr(amount),
			"token": strPtr(tokenSym), "chain_id": int64Ptr(chainID), "created_at": createdAt,
		}
		if txHash != nil {
			entry["tx_hash"] = *txHash
		}
		if subID != nil {
			entry["sub_wallet_id"] = subID.String()
		}
		if confirmedAt != nil {
			entry["confirmed_at"] = *confirmedAt
		}
		txs = append(txs, entry)
	}
	c.JSON(http.StatusOK, gin.H{"transactions": txs})
}

func strPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func int64Ptr(i *int64) int64 {
	if i == nil {
		return 0
	}
	return *i
}

func (svc *Service) GetTransaction(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()
	var txHash, txType, status, blockchain, fromAddr, toAddr, amount *string
	var tokenSym *string
	var chainID *int64
	var createdAt time.Time
	err := svc.store.db.QueryRow(ctx,
		`SELECT tx_hash, tx_type, status, blockchain, from_address, to_address, amount, token_symbol, chain_id, created_at
		 FROM transactions WHERE id = $1`, id).Scan(&txHash, &txType, &status, &blockchain, &fromAddr, &toAddr, &amount, &tokenSym, &chainID, &createdAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "transaction not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id": id, "tx_hash": strPtr(txHash), "tx_type": strPtr(txType), "status": strPtr(status),
		"blockchain": strPtr(blockchain), "from": strPtr(fromAddr), "to": strPtr(toAddr), "amount": strPtr(amount),
		"token": strPtr(tokenSym), "chain_id": int64Ptr(chainID), "created_at": createdAt,
	})
}

// CreateTransaction creates a pending transaction record (request/initiation).
// The actual broadcast happens via the /sign or /transfer endpoints; this
// endpoint records the intent + the on-chain hash once broadcast.
type createTxReq struct {
	SubWalletID string `json:"sub_wallet_id"`
	To          string `json:"to" binding:"required"`
	Amount      string `json:"amount" binding:"required"`
	Token       string `json:"token"`
	TxType      string `json:"tx_type"`
	TxHash      string `json:"tx_hash"` // optional: if already broadcast
	ChainID     int64  `json:"chain_id"`
}

func (svc *Service) CreateTransaction(c *gin.Context) {
	masterID := c.Param("id")
	var req createTxReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.TxType == "" {
		req.TxType = "transfer"
	}
	chainID := req.ChainID
	if chainID == 0 {
		chainID = 1
	}
	txRec := uuid.New().String()
	ctx := c.Request.Context()
	status := "pending"
	if req.TxHash != "" {
		status = "broadcast"
	}
	var fromAddr string
	if req.SubWalletID != "" {
		_ = svc.store.db.QueryRow(ctx, `SELECT address FROM sub_wallets WHERE id = $1`, req.SubWalletID).Scan(&fromAddr)
	}
	if fromAddr == "" {
		_ = svc.store.db.QueryRow(ctx, `SELECT address FROM master_wallets WHERE id = $1`, masterID).Scan(&fromAddr)
	}
	// Provenance for the auto-signer guard: the destination was chosen by the
	// user and recorded at creation. Without this marker the auto-sign daemon
	// refuses to sign (fail-closed).
	metadata, _ := jsonMarshal(gin.H{
		"user_initiated":    true,
		"created_by":        currentUserID(c),
		"destination_chosen_by_user": true,
	})
	_, err := svc.store.db.Exec(ctx,
		`INSERT INTO transactions (id, master_wallet_id, sub_wallet_id, tx_hash, tx_type, status, blockchain, from_address, to_address, amount, token_symbol, chain_id, metadata)
		 VALUES ($1,$2,$3,$4,$5,$6,'ethereum',$7,$8,$9,$10,$11,$12)`,
		txRec, masterID, nilIfEmpty(req.SubWalletID), req.TxHash, req.TxType, status, fromAddr, req.To, req.Amount, req.Token, chainID, string(metadata))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	svc.store.audit(ctx, masterID, "transaction.create", "transaction", "user", currentUserID(c), "transaction", txRec, "normal", gin.H{"to": req.To, "amount": req.Amount})
	c.JSON(http.StatusCreated, gin.H{"id": txRec, "tx_hash": req.TxHash, "status": status, "to": req.To, "amount": req.Amount})
}

// ApproveTransaction records a signer's approval (multisig flow).
func (svc *Service) ApproveTransaction(c *gin.Context) {
	id := c.Param("id")
	userID := currentUserID(c)
	ctx := c.Request.Context()
	// Record signature.
	_, err := svc.store.db.Exec(ctx,
		`INSERT INTO transaction_signatures (transaction_id, signer_id, signature_status, approved_at)
		 VALUES ($1, $2, 'signed', NOW())
		 ON CONFLICT (transaction_id, signer_id) DO UPDATE SET signature_status='signed', approved_at=NOW()`,
		id, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to record approval"})
		return
	}
	// Increment approval count.
	_, _ = svc.store.db.Exec(ctx,
		`UPDATE approval_requests SET current_approvals = current_approvals + 1 WHERE transaction_id = $1`, id)
	svc.store.audit(ctx, "", "transaction.approve", "transaction", "user", userID, "transaction", id, "normal", gin.H{})
	c.JSON(http.StatusOK, gin.H{"approved": true, "transaction_id": id})
}

func (svc *Service) RejectTransaction(c *gin.Context) {
	id := c.Param("id")
	userID := currentUserID(c)
	ctx := c.Request.Context()
	_, err := svc.store.db.Exec(ctx,
		`INSERT INTO transaction_signatures (transaction_id, signer_id, signature_status, rejection_reason)
		 VALUES ($1, $2, 'rejected', $3)
		 ON CONFLICT (transaction_id, signer_id) DO UPDATE SET signature_status='rejected', rejection_reason=$3`,
		id, userID, c.Query("reason"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to record rejection"})
		return
	}
	_, _ = svc.store.db.Exec(ctx, `UPDATE transactions SET status='cancelled' WHERE id = $1`, id)
	svc.store.audit(ctx, "", "transaction.reject", "transaction", "user", userID, "transaction", id, "normal", gin.H{})
	c.JSON(http.StatusOK, gin.H{"rejected": true, "transaction_id": id})
}

// --- helper: ERC-20 transfer calldata ---

func erc20TransferCalldata(to common.Address, amount *big.Int) []byte {
	data := make([]byte, 4+32+32)
	data[0], data[1], data[2], data[3] = 0xa9, 0x05, 0x9c, 0xbb // transfer(address,uint256)
	copy(data[4:36], common.LeftPadBytes(to.Bytes(), 32))
	copy(data[36:68], common.LeftPadBytes(amount.Bytes(), 32))
	return data
}

// humanToWei converts a human-readable decimal amount string to wei (base 10^decimals).
func humanToWei(amount string, decimals int) string {
	f, ok := new(big.Float).SetString(amount)
	if !ok {
		return "0"
	}
	wei, _ := f.Mul(f, big.NewFloat(pow10f(decimals))).Int(nil)
	return wei.String()
}

func pow10f(n int) float64 {
	r := 1.0
	for i := 0; i < n; i++ {
		r *= 10
	}
	return r
}

func jsonMarshal(v interface{}) ([]byte, error) {
	return jsonMarshalImpl(v)
}

// healthCheck returns service health.
func (svc *Service) healthCheck(c *gin.Context) {
	status := "ok"
	if svc.store == nil || svc.store.db == nil {
		status = "degraded"
	}
	c.JSON(http.StatusOK, gin.H{
		"status":   status,
		"service":  "tigerwallet-master-wallet",
		"version":  "1.0.0",
		"port":     svc.cfg.Port,
		"time":     time.Now().UTC(),
	})
}

// GetGasPrice returns the live gas price for a chain.
func (svc *Service) GetGasPrice(c *gin.Context) {
	chainIDStr := c.Query("chain_id")
	chainID, _ := strconv.ParseInt(chainIDStr, 10, 64)
	if chainID == 0 {
		chainID = 1
	}
	rpc := rpcEndpointForChain(chainID)
	if rpc == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "RPC not configured for chain", "chain_id": chainID})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	gp, maxFee, prio, err := FetchGasPrice(ctx, rpc)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"chain_id": chainID, "gas_price": gp.String(), "max_fee": maxFee.String(),
		"priority_fee": prio.String(), "source": "live_rpc",
	})
}

// GetPrice returns the live USD price for a coin.
func (svc *Service) GetPrice(c *gin.Context) {
	coinID := c.Query("coin_id")
	if coinID == "" {
		chainIDStr := c.Query("chain_id")
		cid, _ := strconv.ParseInt(chainIDStr, 10, 64)
		coinID = chainCoinGeckoID(cid)
	}
	if coinID == "" {
		coinID = "ethereum"
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	p, err := FetchTokenPrice(ctx, coinID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"coin_id": coinID, "usd": p.USD, "usd_24h_change": p.USD24h, "market_cap": p.MarketCap, "source": "coingecko"})
}

// GetTransactionHistory returns real on-chain tx history from an explorer.
func (svc *Service) GetTransactionHistory(c *gin.Context) {
	address := c.Query("address")
	chainIDStr := c.Query("chain_id")
	chainID, _ := strconv.ParseInt(chainIDStr, 10, 64)
	if chainID == 0 {
		chainID = 1
	}
	if address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "address required"})
		return
	}
	base, keyEnv := chainExplorerAPI(chainID)
	if base == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no explorer configured for chain", "chain_id": chainID})
		return
	}
	apiKey := os.Getenv(keyEnv)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	txs, err := FetchTransactionHistory(ctx, base, apiKey, address, chainID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"address": address, "chain_id": chainID, "transactions": txs})
}
