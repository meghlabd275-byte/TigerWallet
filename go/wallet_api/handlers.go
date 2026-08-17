package main

// handlers.go — HTTP API handlers. Real wallet operations: create/import
// wallet, derive address, fetch balances/tokens/tx/nfts, send signed tx.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ---- Health ----

func handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "tigerwallet-wallet-api",
		"version": "1.0.0",
		"time":    time.Now().UTC(),
	})
}

// ---- Auth ----

type registerReq struct {
	Email    string `json:"email" binding:"required,email"`
	Username string `json:"username"`
	Password string `json:"password" binding:"required,min=8"`
}

type loginReq struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func handleRegister(c *gin.Context) {
	var req registerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Username is optional: derive a stable handle from the email local-part so
	// clients that send only {email, password} (web/desktop/android/ios/react)
	// register successfully, matching the extension which sends a username.
	username := req.Username
	if username == "" {
		username = emailLocalPart(req.Email)
		if username == "" {
			username = "user"
		}
	}
	existing, err := store.GetUserByEmail(c.Request.Context(), req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if existing != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}
	hash, err := HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}
	uid, err := store.CreateUser(c.Request.Context(), req.Email, username, hash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}
	token, _ := IssueJWT(appConfig.JWTSecret, uid.String(), "user")
	c.JSON(http.StatusCreated, gin.H{"user_id": uid, "token": token})
}

// handleGuestAuth provisions an anonymous guest account for the UserWallet app so
// the user can Create/Import a wallet WITHOUT registering. The client supplies a
// stable device id; the backend idempotently creates (or re-uses) a guest user and
// returns a JWT. Guest accounts cannot be logged into via /auth/login (random
// sentinel password hash), so this is not a privilege-escalation vector.
func handleGuestAuth(c *gin.Context) {
	var req struct {
		DeviceID string `json:"device_id"`
	}
	_ = c.ShouldBindJSON(&req) // device_id optional; empty -> random
	uid, err := store.CreateGuestUser(c.Request.Context(), req.DeviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to provision guest account"})
		return
	}
	token, _ := IssueJWT(appConfig.JWTSecret, uid.String(), "user")
	c.JSON(http.StatusCreated, gin.H{"user_id": uid, "token": token, "guest": true})
}

func handleLogin(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user, err := store.GetUserByEmail(c.Request.Context(), req.Email)
	if err != nil || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	if !VerifyPassword(user.PasswordHash, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	token, _ := IssueJWT(appConfig.JWTSecret, user.ID.String(), user.Role)
	c.JSON(http.StatusOK, gin.H{"token": token, "user": user})
}

// ---- Wallet operations ----

type createWalletReq struct {
	Mnemonic     string `json:"mnemonic"`
	Password     string `json:"password" binding:"required,min=8"`
	Label        string `json:"label"`
	ChainID      int64  `json:"chain_id"`
	AccountIndex int    `json:"account_index"`
	EntropyBits  int    `json:"entropy_bits"`
}

type walletResp struct {
	ID             string `json:"id"`
	Label          string `json:"label"`
	ChainID        int64  `json:"chain_id"`
	Address        string `json:"address"`
	DerivationPath string `json:"derivation_path"`
	Mnemonic       string `json:"mnemonic,omitempty"` // only returned on creation
}

func handleCreateWallet(c *gin.Context) {
	var req createWalletReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.ChainID == 0 {
		req.ChainID = 1
	}
	chain := evmChainByChainID(req.ChainID)
	if chain == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported chain"})
		return
	}

	// Generate a real mnemonic if not provided
	mnemonic := strings.TrimSpace(req.Mnemonic)
	if mnemonic == "" {
		bits := req.EntropyBits
		if bits == 0 {
			bits = 256
		}
		m, err := GenerateMnemonic(bits)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate mnemonic"})
			return
		}
		mnemonic = m
	} else {
		if !ValidateMnemonic(mnemonic) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid mnemonic (BIP-39 checksum failed)"})
			return
		}
	}

	if req.AccountIndex < 0 {
		req.AccountIndex = 0
	}

	privKey, err := DeriveEVMPrivateKey(mnemonic, *chain, uint32(req.AccountIndex))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "key derivation failed: " + err.Error()})
		return
	}
	address := PrivateKeyToAddress(privKey).Hex()

	// Encrypt the mnemonic seed with the user password and persist
	seed := MnemonicToSeed(mnemonic, "")
	encSeed, err := EncryptSeed(seed, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encrypt seed"})
		return
	}

	uid, _ := uuid.Parse(getUserID(c))
	label := req.Label
	if label == "" {
		label = chain.Name + " Wallet"
	}

	w := &WalletRecord{
		UserID:         uid,
		Label:          label,
		ChainID:        req.ChainID,
		Address:        address,
		EncryptedSeed:  encSeed,
		DerivationPath: chain.DerivationPath,
		AccountIndex:   req.AccountIndex,
		IsPrimary:      false,
	}
	if err := store.SaveWallet(c.Request.Context(), w); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save wallet"})
		return
	}

	c.JSON(http.StatusCreated, walletResp{
		ID:             w.ID.String(),
		Label:          w.Label,
		ChainID:        w.ChainID,
		Address:        address,
		DerivationPath: w.DerivationPath,
		Mnemonic:       mnemonic,
	})
}

// handleExportEncryptedSeed returns the wallet's AES-256-GCM encrypted seed blob
// (salt+ciphertext hex) for the user to back up to Google Drive / iCloud / a
// hardware drive. The blob is password-encrypted server-side; the raw seed /
// mnemonic is NEVER exposed. The caller MUST supply the wallet password, which
// is verified by actually decrypting the blob (proves the password is correct
// before handing the blob to the user). Used for Google Drive backup.
func handleExportEncryptedSeed(c *gin.Context) {
	walletID := c.Param("id")
	var req struct {
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	wid, err := uuid.Parse(walletID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid wallet_id"})
		return
	}
	wallet, err := store.GetWalletByID(c.Request.Context(), wid)
	if err != nil || wallet == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}
	uid, _ := uuid.Parse(getUserID(c))
	if wallet.UserID != uid {
		c.JSON(http.StatusForbidden, gin.H{"error": "wallet does not belong to user"})
		return
	}
	// Verify the password by actually decrypting (fail-closed). We do NOT
	// return the blob unless the password is correct.
	if _, err := DecryptSeed(wallet.EncryptedSeed, req.Password); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "incorrect password"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"wallet_id":       wallet.ID,
		"address":         wallet.Address,
		"chain_id":        wallet.ChainID,
		"label":           wallet.Label,
		"derivation_path": wallet.DerivationPath,
		"account_index":   wallet.AccountIndex,
		"encrypted_seed":  wallet.EncryptedSeed,
		"v":               1,
	})
}

// handleImportEncryptedSeed restores a wallet from an AES-256-GCM encrypted
// seed blob (e.g. downloaded from Google Drive). The user supplies the blob +
// the password; the backend decrypts (verifies the password), re-derives the
// address to confirm, and re-stores the encrypted seed under the current user.
// The raw seed is never persisted in plaintext — only the re-stored encrypted
// blob. Used for Google Drive restore.
func handleImportEncryptedSeed(c *gin.Context) {
	var req struct {
		EncryptedSeed string `json:"encrypted_seed" binding:"required"`
		Password      string `json:"password" binding:"required"`
		Label         string `json:"label"`
		ChainID       int64  `json:"chain_id"`
		DerivationPath string `json:"derivation_path"`
		AccountIndex   int    `json:"account_index"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Decrypt to verify the password (fail-closed: wrong password -> reject).
	seed, err := DecryptSeed(req.EncryptedSeed, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "incorrect password or corrupted backup"})
		return
	}
	if req.ChainID == 0 {
		req.ChainID = 1
	}
	chain := evmChainByChainID(req.ChainID)
	if chain == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported chain_id"})
		return
	}
	if req.DerivationPath == "" {
		req.DerivationPath = chain.DerivationPath
	}
	if req.AccountIndex == 0 {
		req.AccountIndex = 0
	}
	// Re-derive the address from the recovered seed to confirm the backup is valid.
	privKey, err := DerivePrivateKeyFromPath(seed, req.DerivationPath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to derive key from backup: " + err.Error()})
		return
	}
	addr := crypto.PubkeyToAddress(privKey.PublicKey).Hex()
	if req.Label == "" {
		req.Label = "Restored Wallet"
	}
	uid, _ := uuid.Parse(getUserID(c))
	w := &WalletRecord{
		UserID:         uid,
		Label:          req.Label,
		ChainID:        req.ChainID,
		Address:        addr,
		EncryptedSeed:  req.EncryptedSeed, // re-store the original encrypted blob (not re-encrypted)
		DerivationPath: req.DerivationPath,
		AccountIndex:   req.AccountIndex,
		IsPrimary:      false,
	}
	if err := store.SaveWallet(c.Request.Context(), w); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to restore wallet: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id":              w.ID,
		"label":           w.Label,
		"chain_id":        w.ChainID,
		"address":         w.Address,
		"derivation_path": w.DerivationPath,
		"account_index":  w.AccountIndex,
		"restored":        true,
	})
}

func handleListWallets(c *gin.Context) {
	uid, _ := uuid.Parse(getUserID(c))
	wallets, err := store.GetWalletsByUser(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list wallets"})
		return
	}
	out := make([]walletResp, 0, len(wallets))
	for _, w := range wallets {
		out = append(out, walletResp{
			ID:             w.ID.String(),
			Label:          w.Label,
			ChainID:        w.ChainID,
			Address:        w.Address,
			DerivationPath: w.DerivationPath,
		})
	}
	c.JSON(http.StatusOK, gin.H{"wallets": out})
}

// ---- Balance / token / tx / nft fetchers ----

func handleBalance(c *gin.Context) {
	address := c.Query("address")
	chainID := parseChainID(c.Query("chain_id"))
	chain := evmChainByChainID(chainID)
	if chain == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chain_id"})
		return
	}
	addr, err := hexAddress(address)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// cache check
	cacheK := store.cacheKey("balance", fmt.Sprintf("%d", chainID), address)
	var cached BalanceResult
	if store.GetCache(ctx, cacheK, &cached) == nil {
		c.JSON(http.StatusOK, cached)
		return
	}

	bal, err := FetchNativeBalance(ctx, chain.RPCEndpoint, addr)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "rpc error: " + err.Error()})
		return
	}
	ethPrice := FetchETHPrice(ctx)
	human := weiToFloat(bal, 18)
	result := BalanceResult{
		ChainID:  chainID,
		Symbol:   chain.Symbol,
		Address:  address,
		Balance:  bal.String(),
		BalanceF: human,
		USDValue: human * ethPrice,
	}
	_ = store.SetCache(ctx, cacheK, result, 30*time.Second)
	c.JSON(http.StatusOK, result)
}

func handleTokenBalances(c *gin.Context) {
	address := c.Query("address")
	chainID := parseChainID(c.Query("chain_id"))
	chain := evmChainByChainID(chainID)
	if chain == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chain_id"})
		return
	}
	addr, err := hexAddress(address)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	tokens := tokensForChain(chainID)
	balances := FetchTokenBalances(ctx, chain.RPCEndpoint, addr, tokens)
	c.JSON(http.StatusOK, gin.H{"tokens": balances})
}

func handleTransactions(c *gin.Context) {
	address := c.Query("address")
	chainID := parseChainID(c.Query("chain_id"))
	chain := evmChainByChainID(chainID)
	if chain == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chain_id"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	txs, err := FetchTransactionHistory(ctx, chain.ExplorerAPI, appConfig.EtherscanAPIKey, address, chainID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"transactions": txs})
}

func handleNFTs(c *gin.Context) {
	address := c.Query("address")
	chainID := parseChainID(c.Query("chain_id"))
	chain := evmChainByChainID(chainID)
	if chain == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chain_id"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	nfts, err := FetchNFTAssets(ctx, chain.ExplorerAPI, appConfig.EtherscanAPIKey, address)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"nfts": nfts})
}

// ---- Send transaction ----

type sendTxReq struct {
	WalletID  string `json:"wallet_id" binding:"required"`
	Password  string `json:"password" binding:"required"`
	ToAddress string `json:"to" binding:"required"`
	Value     string `json:"value" binding:"required"` // in ether
	GasLimit  uint64 `json:"gas_limit"`
	Data      string `json:"data"`
	ChainID   int64  `json:"chain_id"`
}

func handleSendTransaction(c *gin.Context) {
	if !enforceFeature(c, FeatureSendTransactions) {
		return
	}
	var req sendTxReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	executeSend(c, req)
}

// handleAutoSend is the UserWallet outgoing-tx flow with MasterWallet-owner
// policy auto-approval. It performs a server-to-server policy check against the
// MasterWallet backend's /check-auto-sign-policy (the master wallet owner's
// configured auto-sign RULES: max_amount + active flag). If the policy approves
// within a second, the tx is self-signed with the user's own decrypted seed and
// broadcast — the UserWallet shows "transaction submitted to blockchain
// network". If the master wallet is unreachable or the policy denies, the tx
// falls back to a normal self-sign + broadcast (the user always retains
// self-custody; the policy gate is a gas-sponsorship/convenience layer, NOT a
// custody gate). The UserWallet client never talks to the MasterWallet backend
// directly — only this wallet_api does (server-to-server), preserving app
// separation.
func handleAutoSend(c *gin.Context) {
	if !enforceFeature(c, FeatureSendTransactions) {
		return
	}
	var req sendTxReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Optional MasterWallet master-id to consult for policy (env-configured
	// default when omitted). When the master wallet backend is unset or the
	// policy denies/errored, we still self-sign (user retains self-custody).
	masterID := c.Query("master_wallet_id")
	autoApproved, reason := checkMasterWalletPolicy(req.Value, masterID)
	// Self-sign + broadcast regardless (self-custody). The autoApproved flag is
	// surfaced to the client so it can show "auto-approved by master wallet".
	executeSendWithAutoFlag(c, req, autoApproved, reason)
}

// checkMasterWalletPolicy asks the MasterWallet backend whether the master
// wallet owner's auto-sign rules approve a tx of the given value. Returns
// (approved bool, reason string). On any error (backend unreachable, no
// master id, network failure) it returns (false, "policy unavailable —
// self-sign fallback") — never blocks the user's self-custodial send.
func checkMasterWalletPolicy(value, masterID string) (bool, string) {
	base := strings.TrimRight(appConfig.MasterWalletBackendURL, "/")
	if base == "" || masterID == "" {
		return false, "master wallet policy not configured — self-sign fallback"
	}
	body, _ := json.Marshal(map[string]string{"tx_type": "send", "value": value})
	client := &http.Client{Timeout: 3 * time.Second}
	url := base + "/api/v1/master-wallet/" + masterID + "/check-auto-sign-policy"
	resp, err := client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return false, "master wallet policy unreachable — self-sign fallback"
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, "master wallet policy error — self-sign fallback"
	}
	var out struct {
		Approved bool   `json:"approved"`
		Reason   string `json:"reason"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return false, "master wallet policy decode error — self-sign fallback"
	}
	return out.Approved, out.Reason
}

// executeSendWithAutoFlag wraps executeSend, tagging the response with the
// master-wallet auto-approval status so the client can display it.
func executeSendWithAutoFlag(c *gin.Context, req sendTxReq, autoApproved bool, reason string) {
	// executeSend writes the JSON response directly; capture it via a response
	// writer wrapper to inject the auto-approval fields.
	rw := &autoFlagWriter{ResponseWriter: c.Writer, autoApproved: autoApproved, reason: reason}
	orig := c.Writer
	c.Writer = rw
	executeSend(c, req)
	c.Writer = orig
}

// autoFlagWriter is a gin.ResponseWriter wrapper that injects the master-wallet
// auto-approval fields into the JSON response written by executeSend.
type autoFlagWriter struct {
	gin.ResponseWriter
	autoApproved bool
	reason       string
	wrote        bool
}

func (w *autoFlagWriter) Write(b []byte) (int, error) {
	if w.wrote {
		return w.ResponseWriter.Write(b)
	}
	w.wrote = true
	// Inject the auto-approval fields into the JSON object. executeSend writes a
	// compact JSON object starting with '{', so we insert the new keys after it.
	s := string(b)
	if strings.HasPrefix(strings.TrimSpace(s), "{") {
		trimmed := strings.TrimSpace(s)
		inject := fmt.Sprintf(`"auto_approved":%t,"auto_approval_reason":%q,`, w.autoApproved, w.reason)
		// Insert after the leading '{' (or '{\n').
		i := strings.Index(trimmed, "{")
		modified := trimmed[:i+1] + inject + trimmed[i+1:]
		return w.ResponseWriter.Write([]byte(modified))
	}
	return w.ResponseWriter.Write(b)
}

func handleSendTransactionUnused(c *gin.Context) {}

// executeSend performs the real EVM transaction signing + broadcast for an
// already-bound sendTxReq. Shared by handleSendTransaction and handleNFTTransfer
// (which builds an ERC-721 safeTransferFrom calldata before delegating here).
func executeSend(c *gin.Context, req sendTxReq) {
	if req.ChainID == 0 {
		req.ChainID = 1
	}
	chain := evmChainByChainID(req.ChainID)
	if chain == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported chain"})
		return
	}

	wid, err := uuid.Parse(req.WalletID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid wallet_id"})
		return
	}
	wallet, err := store.GetWalletByID(c.Request.Context(), wid)
	if err != nil || wallet == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}

	// Verify ownership
	uid, _ := uuid.Parse(getUserID(c))
	if wallet.UserID != uid {
		c.JSON(http.StatusForbidden, gin.H{"error": "wallet does not belong to user"})
		return
	}

	// Decrypt seed
	seed, err := DecryptSeed(wallet.EncryptedSeed, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "incorrect password"})
		return
	}

	// Re-derive the private key from the stored seed + path
	privKey, err := hdDerive(seed, wallet.DerivationPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "key derivation failed"})
		return
	}

	toAddr, err := hexAddress(req.ToAddress)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse value (ether -> wei)
	valueFloat := new(big.Float)
	_, ok := valueFloat.SetString(req.Value)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid value"})
		return
	}
	weiValue := etherToWei(valueFloat)

	// Parse data
	var data []byte
	if req.Data != "" && req.Data != "0x" {
		d, err := hexDecode(req.Data)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid data"})
			return
		}
		data = d
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// Fetch nonce
	nonce, err := FetchTransactionCount(ctx, chain.RPCEndpoint, common.HexToAddress(wallet.Address))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to fetch nonce: " + err.Error()})
		return
	}

	// Fetch gas price
	gasPrice, maxFee, maxPrioFee, err := FetchGasPrice(ctx, chain.RPCEndpoint)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to fetch gas price: " + err.Error()})
		return
	}
	gasLimit := req.GasLimit
	if gasLimit == 0 {
		if len(data) > 0 {
			gasLimit = 150000 // contract call estimate
		} else {
			gasLimit = 21000 // simple transfer
		}
	}

	chainID := big.NewInt(req.ChainID)
	rawTx, err := SignEVMTransaction(chainID, nonce, toAddr, weiValue, gasLimit, gasPrice, maxFee, maxPrioFee, data, privKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "signing failed: " + err.Error()})
		return
	}

	txHash, err := BroadcastTransaction(ctx, chain.RPCEndpoint, rawTx)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "broadcast failed: " + err.Error()})
		return
	}

	// Log the transaction
	_ = store.LogTransaction(ctx, &TxLogRecord{
		UserID:   uid,
		WalletID: wid,
		TxHash:   txHash,
		ChainID:  req.ChainID,
		FromAddr: wallet.Address,
		ToAddr:   req.ToAddress,
		Value:    weiValue.String(),
		Status:   "pending",
	})

	c.JSON(http.StatusOK, gin.H{
		"tx_hash":  txHash,
		"raw_tx":   rawTx,
		"chain_id": req.ChainID,
		"nonce":    nonce,
	})
}

// ---- Gas & prices ----

func handleGasPrice(c *gin.Context) {
	chainID := parseChainID(c.Query("chain_id"))
	chain := evmChainByChainID(chainID)
	if chain == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chain_id"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	gasPrice, maxFee, maxPrioFee, err := FetchGasPrice(ctx, chain.RPCEndpoint)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"chain_id":         chainID,
		"gas_price":        gasPrice.String(),
		"max_fee_per_gas":  maxFee.String(),
		"max_priority_fee": maxPrioFee.String(),
		"gas_price_gwei":   weiToGweiFloat(gasPrice),
	})
}

func handlePrice(c *gin.Context) {
	// Accept any of the param names used by the clients: web/desktop send
	// ?symbol=, android/ios send ?token=, the canonical name is ?coin=.
	coinID := c.DefaultQuery("coin", c.DefaultQuery("symbol", c.DefaultQuery("token", "ethereum")))
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	cacheK := store.cacheKey("price", coinID)
	var cached CoinGeckoPrice
	if store.GetCache(ctx, cacheK, &cached) == nil {
		c.JSON(http.StatusOK, cached)
		return
	}
	p, err := FetchTokenPrice(ctx, coinID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	_ = store.SetCache(ctx, cacheK, p, 60*time.Second)
	c.JSON(http.StatusOK, p)
}

func handleSupportedChains(c *gin.Context) {
	ct := c.Query("type")
	var chains []ChainConfig
	if ct != "" {
		chains = listChainsByType(ct)
	} else {
		chains = listSupportedChains()
	}
	c.JSON(http.StatusOK, gin.H{
		"chains":        chains,
		"count":         len(chains),
		"evm_count":     evmChainCount(),
		"non_evm_count": nonEvmChainCount(),
		"mainnet_only":  true,
	})
}

func handleSignMessage(c *gin.Context) {
	var req struct {
		WalletID string `json:"wallet_id" binding:"required"`
		Password string `json:"password" binding:"required"`
		Message  string `json:"message" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	wid, _ := uuid.Parse(req.WalletID)
	wallet, err := store.GetWalletByID(c.Request.Context(), wid)
	if err != nil || wallet == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}
	uid, _ := uuid.Parse(getUserID(c))
	if wallet.UserID != uid {
		c.JSON(http.StatusForbidden, gin.H{"error": "wallet does not belong to user"})
		return
	}
	seed, err := DecryptSeed(wallet.EncryptedSeed, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "incorrect password"})
		return
	}
	privKey, err := hdDerive(seed, wallet.DerivationPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "key derivation failed"})
		return
	}
	sig, err := SignPersonalMessage(privKey, []byte(req.Message))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "signing failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"signature": "0x" + common.Bytes2Hex(sig)})
}

// ---- dApp directory (public read) ----

func handleListDApps(c *gin.Context) {
	category := c.Query("category")
	chain := c.Query("chain")
	c.JSON(http.StatusOK, gin.H{
		"dapps": listDApps(category, chain),
		"count": len(listDApps(category, chain)),
	})
}

func handleGetDApp(c *gin.Context) {
	id := c.Param("id")
	d := getDApp(id)
	if d == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "dapp not found"})
		return
	}
	c.JSON(http.StatusOK, d)
}

func handleDAppCategories(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"categories": dAppCategories()})
}

// ---- Token asset registry (public read) ----

func handleTokenRegistry(c *gin.Context) {
	chainIDStr := c.Query("chain_id")
	if chainIDStr == "" {
		// Return the full registry grouped by chain.
		out := make(map[string]interface{}, len(defaultTokenRegistry))
		for cid, toks := range defaultTokenRegistry {
			out[fmt.Sprintf("%d", cid)] = toks
		}
		c.JSON(http.StatusOK, gin.H{"tokens": out, "count": len(defaultTokenRegistry)})
		return
	}
	chainID, err := strconv.ParseInt(chainIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chain_id"})
		return
	}
	toks := tokensForChain(chainID)
	if toks == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no tokens for chain_id"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"chain_id": chainID, "tokens": toks, "count": len(toks)})
}

// handleExportKeystore exports a wallet's private key as a standard Web3 Secret
// Storage V3 (scrypt variant) keystore JSON, interoperable with geth/MetaMask.
// The wallet seed is decrypted with the user's password, the private key is
// re-derived, then re-encrypted with the export password into the V3 format.
func handleExportKeystore(c *gin.Context) {
	var req struct {
		WalletID       string `json:"wallet_id" binding:"required"`
		Password       string `json:"password" binding:"required"`
		ExportPassword string `json:"export_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	wid, _ := uuid.Parse(req.WalletID)
	wallet, err := store.GetWalletByID(c.Request.Context(), wid)
	if err != nil || wallet == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}
	uid, _ := uuid.Parse(getUserID(c))
	if wallet.UserID != uid {
		c.JSON(http.StatusForbidden, gin.H{"error": "wallet does not belong to user"})
		return
	}
	seed, err := DecryptSeed(wallet.EncryptedSeed, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "incorrect password"})
		return
	}
	privKey, err := hdDerive(seed, wallet.DerivationPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "key derivation failed"})
		return
	}
	keystoreJSON, err := ExportKeystoreV3(privKey, req.ExportPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "keystore export failed"})
		return
	}
	c.Data(http.StatusOK, "application/json", keystoreJSON)
}

// handleImportKeystore imports a standard Web3 Secret Storage V3 keystore JSON
// (e.g. produced by geth/MetaMask/TigerWallet) and, together with a mnemonic or
// a new label, persists it as a TigerWallet encrypted-seed wallet. Returns the
// new wallet id + address.
func handleImportKeystore(c *gin.Context) {
	var req struct {
		KeystoreJSON string `json:"keystore_json" binding:"required"`
		Password     string `json:"password" binding:"required"`
		Label        string `json:"label"`
		ChainID      int64  `json:"chain_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.ChainID == 0 {
		req.ChainID = 1 // Ethereum mainnet by default for V3 keystore imports
	}
	chain := evmChainByChainID(req.ChainID)
	if chain == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported chain_id"})
		return
	}
	key, err := ImportKeystoreV3([]byte(req.KeystoreJSON), req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "keystore import failed: " + err.Error()})
		return
	}
	addr := crypto.PubkeyToAddress(key.PublicKey)

	// Re-encrypt the 32-byte private key bytes with the TigerWallet AES-GCM/scrypt
	// seed encryption (wallet_api's canonical at-rest format) and persist.
	privBytes := make([]byte, 32)
	key.D.FillBytes(privBytes)
	encSeed, err := EncryptSeed(privBytes, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "seed encryption failed"})
		return
	}

	uid, _ := uuid.Parse(getUserID(c))
	if req.Label == "" {
		req.Label = "Imported Keystore " + addr.Hex()[:10]
	}
	w := &WalletRecord{
		UserID:         uid,
		Label:          req.Label,
		ChainID:        req.ChainID,
		Address:        addr.Hex(),
		DerivationPath: chain.DerivationPath,
		EncryptedSeed:  encSeed,
		IsPrimary:      false,
	}
	if err := store.SaveWallet(c.Request.Context(), w); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "persist wallet failed"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"wallet_id": w.ID.String(),
		"address":   w.Address,
		"label":     w.Label,
		"chain_id":  w.ChainID,
		"source":    "keystore-v3",
	})
}
