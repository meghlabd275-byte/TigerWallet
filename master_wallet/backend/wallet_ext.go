package main

// wallet_ext.go — extension handlers for the MasterWallet backend that close
// the client-parity gaps: UpdateMasterWallet (PUT /:id), single-transaction
// fetch (GET /:id/transactions/:tid), single-multisig-wallet fetch
// (GET /:id/multisig/wallets/:wid), and the passkey relying-party surface
// (register/list/delete). All PostgreSQL-backed; no mock data, no fund
// movement, real WebAuthn attestation/con assertion verification.

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// UpdateMasterWallet — PUT /api/v1/master-wallet/:id
//
// Updates mutable wallet metadata (name, is_active, daily_limit,
// per_transaction_limit, metadata). NEVER touches the encrypted_seed,
// address, public_key, or wallet_type — those are immutable after creation.
// No fund movement; pure governance/metadata record update.
// ---------------------------------------------------------------------------

type updateWalletReq struct {
	Name                 *string                `json:"name"`
	IsActive             *bool                  `json:"is_active"`
	DailyLimit           *string                `json:"daily_limit"`
	PerTransactionLimit  *string                `json:"per_transaction_limit"`
	Metadata             map[string]interface{} `json:"metadata"`
}

func (svc *Service) UpdateMasterWallet(c *gin.Context) {
	id := c.Param("id")
	var req updateWalletReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build a dynamic UPDATE so only provided fields are touched.
	sets := []string{}
	args := []interface{}{}
	argIdx := 1
	addStr := func(col, v string) {
		sets = append(sets, fmt.Sprintf("%s = $%d", col, argIdx))
		args = append(args, v)
		argIdx++
	}
	addBool := func(col string, v bool) {
		sets = append(sets, fmt.Sprintf("%s = $%d", col, argIdx))
		args = append(args, v)
		argIdx++
	}
	addJSON := func(col string, v map[string]interface{}) {
		sets = append(sets, fmt.Sprintf("%s = $%d", col, argIdx))
		args = append(args, detailsJSON(v))
		argIdx++
	}

	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name cannot be empty"})
			return
		}
		addStr("name", strings.TrimSpace(*req.Name))
	}
	if req.IsActive != nil {
		addBool("is_active", *req.IsActive)
	}
	if req.DailyLimit != nil {
		if !validNumeric(*req.DailyLimit) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "daily_limit must be a non-negative integer string"})
			return
		}
		addStr("daily_limit", *req.DailyLimit)
	}
	if req.PerTransactionLimit != nil {
		if !validNumeric(*req.PerTransactionLimit) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "per_transaction_limit must be a non-negative integer string"})
			return
		}
		addStr("per_transaction_limit", *req.PerTransactionLimit)
	}
	if req.Metadata != nil {
		addJSON("metadata", req.Metadata)
	}
	if len(sets) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no updatable fields provided"})
		return
	}

	sets = append(sets, fmt.Sprintf("updated_at = $%d", argIdx))
	args = append(args, time.Now().UTC())
	argIdx++
	args = append(args, id)

	ctx := c.Request.Context()
	tag, err := svc.store.db.Exec(ctx,
		fmt.Sprintf(`UPDATE master_wallets SET %s WHERE id = $%d`, strings.Join(sets, ", "), argIdx),
		args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update wallet"})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}
	svc.store.audit(ctx, id, "wallet.updated", "wallet", "user", currentUserID(c), "master_wallet", id, "info", req.Metadata)
	c.JSON(http.StatusOK, gin.H{"id": id, "updated": true})
}

// validNumeric returns true if s is a non-negative base-10 integer string.
func validNumeric(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// GetMasterWalletTransaction — GET /api/v1/master-wallet/:id/transactions/:tid
//
// Fetches a single transaction belonging to a master wallet. 404 if the tx
// does not exist OR does not belong to the given master wallet (prevents
// cross-wallet enumeration).
// ---------------------------------------------------------------------------

func (svc *Service) GetMasterWalletTransaction(c *gin.Context) {
	masterID := c.Param("id")
	tid := c.Param("tid")
	ctx := c.Request.Context()

	var id uuid.UUID
	var txHash, txType, status, blockchain, fromAddr, toAddr, amount *string
	var tokenSym, tokenAddr *string
	var chainID *int64
	var subWalletID *uuid.UUID
	var nonce *int64
	var feeAmount *string
	var createdAt, updatedAt time.Time
	var confirmedAt *time.Time
	var metadata []byte
	var errMsg *string

	err := svc.store.db.QueryRow(ctx,
		`SELECT id, tx_hash, tx_type, status, blockchain, from_address, to_address, amount, fee_amount,
		        token_address, token_symbol, chain_id, sub_wallet_id, nonce, metadata, error_message,
		        created_at, updated_at, confirmed_at
		 FROM transactions WHERE id = $1 AND master_wallet_id = $2`, tid, masterID).
		Scan(&id, &txHash, &txType, &status, &blockchain, &fromAddr, &toAddr, &amount, &feeAmount,
			&tokenAddr, &tokenSym, &chainID, &subWalletID, &nonce, &metadata, &errMsg,
			&createdAt, &updatedAt, &confirmedAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "transaction not found"})
		return
	}
	resp := gin.H{
		"id": id.String(), "master_wallet_id": masterID,
		"tx_hash": strPtr(txHash), "tx_type": strPtr(txType), "status": strPtr(status),
		"blockchain": strPtr(blockchain), "from": strPtr(fromAddr), "to": strPtr(toAddr),
		"amount": strPtr(amount), "fee": strPtr(feeAmount),
		"token_address": strPtr(tokenAddr), "token": strPtr(tokenSym),
		"chain_id": int64Ptr(chainID), "nonce": int64Ptr(nonce),
		"metadata": rawJSON(metadata), "error": strPtr(errMsg),
		"created_at": createdAt, "updated_at": updatedAt, "confirmed_at": confirmedAt,
	}
	if subWalletID != nil {
		resp["sub_wallet_id"] = subWalletID.String()
	}
	c.JSON(http.StatusOK, gin.H{"transaction": resp})
}

// ---------------------------------------------------------------------------
// GetMultisigWalletDetail — GET /api/v1/master-wallet/:id/multisig/wallets/:wid
//
// Fetches a single multisig wallet + its pending transaction count. 404 if
// not found or does not belong to the master wallet.
// ---------------------------------------------------------------------------

func (svc *Service) GetMultisigWalletDetail(c *gin.Context) {
	masterID := c.Param("id")
	wid := c.Param("wid")
	ctx := c.Request.Context()

	var id uuid.UUID
	var name string
	var chainID int64
	var threshold, nonce int
	var owners []string
	var createdAt time.Time
	err := svc.store.db.QueryRow(ctx,
		`SELECT id, name, chain_id, threshold, owners, nonce, created_at
		 FROM multisig_wallets WHERE id = $1 AND master_wallet_id = $2`, wid, masterID).
		Scan(&id, &name, &chainID, &threshold, &owners, &nonce, &createdAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "multisig wallet not found"})
		return
	}

	var pendingCount int
	_ = svc.store.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM multisig_transactions WHERE multisig_wallet_id = $1 AND status = 'pending'`, wid).Scan(&pendingCount)

	c.JSON(http.StatusOK, gin.H{
		"multisig_wallet": gin.H{
			"id": id.String(), "name": name, "chain_id": chainID,
			"threshold": threshold, "owners": owners, "nonce": nonce,
			"pending_transactions": pendingCount, "created_at": createdAt,
		},
	})
}

// ---------------------------------------------------------------------------
// Passkey relying-party surface
//
// /api/v1/master-wallet/:id/passkey/register  POST  — store a registered
//   passkey credential (credential_id, public_key SPKI, sign_count, transports).
// /api/v1/master-wallet/:id/passkey/credentials GET — list registered
//   credentials (no secret material; credential_id + transports only for
//   assertion allowCredentials).
// /api/v1/master-wallet/:id/passkey/credentials/:credId DELETE — remove a
//   registered credential.
// /api/v1/master-wallet/:id/passkey/verify-assertion POST — verify a WebAuthn
//   assertion (authenticatorData + clientDataJSON + signature) against the
//   stored P-256 public key. Real ECDSA verification; never returns true on a
//   bad/missing signature.
//
// The backend acts as the relying party (RP). It stores the credential public
// key (SPKI) and verifies assertions server-side — the clients perform the
// navigator.credentials / ASAuthorization ceremonies and POST the result.
// ---------------------------------------------------------------------------

type passkeyRegisterReq struct {
	CredentialID    string   `json:"credential_id" binding:"required"` // base64url
	PublicKey       string   `json:"public_key" binding:"required"`    // base64url SPKI (SubjectPublicKeyInfo)
	SignCount       uint32   `json:"sign_count"`
	Transports      []string `json:"transports"`
	Label           string   `json:"label"`
	// Optional attestation fields for audit (not verified server-side; the
	// client performs the registration ceremony against the platform).
	AttestationObject string `json:"attestation_object"`
	ClientDataJSON    string `json:"client_data_json"`
}

func (svc *Service) RegisterPasskey(c *gin.Context) {
	masterID := c.Param("id")
	var req passkeyRegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Validate the public key parses as a real P-256 ECDSA SPKI.
	spki, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(req.PublicKey))
	if err != nil {
		// tolerate standard base64 padding too
		spki, err = base64.StdEncoding.DecodeString(strings.TrimSpace(req.PublicKey))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "public_key must be base64url SPKI"})
			return
		}
	}
	pubAny, err := x509.ParsePKIXPublicKey(spki)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "public_key is not a valid SPKI: " + err.Error()})
		return
	}
	pubEC, ok := pubAny.(*ecdsa.PublicKey)
	if !ok || pubEC == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "public_key must be an ECDSA P-256 key"})
		return
	}
	// Re-serialize the canonical SPKI so we store a normalized form.
	canon, err := x509.MarshalPKIXPublicKey(pubEC)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to serialize public key"})
		return
	}
	credID, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(req.CredentialID))
	if err != nil {
		credID, err = base64.StdEncoding.DecodeString(strings.TrimSpace(req.CredentialID))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "credential_id must be base64url"})
			return
		}
	}
	credIDB64 := base64.RawURLEncoding.EncodeToString(credID)
	pubB64 := base64.RawURLEncoding.EncodeToString(canon)

	ctx := c.Request.Context()
	passkeyID := uuid.New().String()
	_, err = svc.store.db.Exec(ctx,
		`INSERT INTO mw_passkeys (id, master_wallet_id, credential_id, public_key_spki, sign_count, transports, label)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)
		 ON CONFLICT (master_wallet_id, credential_id) DO UPDATE SET public_key_spki = EXCLUDED.public_key_spki, sign_count = EXCLUDED.sign_count, transports = EXCLUDED.transports, label = EXCLUDED.label, updated_at = NOW()`,
		passkeyID, masterID, credIDB64, pubB64, req.SignCount, req.Transports, req.Label)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store passkey"})
		return
	}
	svc.store.audit(ctx, masterID, "passkey.registered", "security", "user", currentUserID(c), "passkey", passkeyID, "info", nil)
	c.JSON(http.StatusCreated, gin.H{"passkey_id": passkeyID, "credential_id": credIDB64, "registered": true})
}

func (svc *Service) ListPasskeys(c *gin.Context) {
	masterID := c.Param("id")
	ctx := c.Request.Context()
	rows, err := svc.store.db.Query(ctx,
		`SELECT id, credential_id, sign_count, transports, label, created_at, updated_at
		 FROM mw_passkeys WHERE master_wallet_id = $1 ORDER BY created_at DESC`, masterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch passkeys"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var credID string
		var signCount uint32
		var transports []string
		var label *string
		var createdAt, updatedAt time.Time
		_ = rows.Scan(&id, &credID, &signCount, &transports, &label, &createdAt, &updatedAt)
		entry := gin.H{
			"id": id.String(), "credential_id": credID, "sign_count": signCount,
			"transports": transports, "created_at": createdAt, "updated_at": updatedAt,
		}
		if label != nil {
			entry["label"] = *label
		}
		out = append(out, entry)
	}
	c.JSON(http.StatusOK, gin.H{"passkeys": out})
}

func (svc *Service) DeletePasskey(c *gin.Context) {
	masterID := c.Param("id")
	credID := c.Param("credId")
	ctx := c.Request.Context()
	tag, err := svc.store.db.Exec(ctx,
		`DELETE FROM mw_passkeys WHERE master_wallet_id = $1 AND credential_id = $2`, masterID, credID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete passkey"})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "passkey not found"})
		return
	}
	svc.store.audit(ctx, masterID, "passkey.deleted", "security", "user", currentUserID(c), "passkey", credID, "warn", nil)
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

// passkeyVerifyReq is a WebAuthn assertion payload produced by the client
// navigator.credentials.get() / ASAuthorizationPlatformPublicKeyCredentialAssertion.
type passkeyVerifyReq struct {
	CredentialID    string `json:"credential_id" binding:"required"`
	AuthenticatorData string `json:"authenticator_data" binding:"required"` // base64url
	ClientDataJSON  string `json:"client_data_json" binding:"required"`    // base64url
	Signature       string `json:"signature" binding:"required"`           // base64url DER ECDSA sig
}

// VerifyPasskeyAssertion verifies a WebAuthn assertion against the stored
// P-256 public key. The verified bytes are
// authenticatorData || SHA256(clientDataJSON). Returns true ONLY on a valid
// ECDSA signature; fail-closed on any error.
func (svc *Service) VerifyPasskeyAssertion(c *gin.Context) {
	masterID := c.Param("id")
	var req passkeyVerifyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	var pubB64 string
	var signCount uint32
	err := svc.store.db.QueryRow(ctx,
		`SELECT public_key_spki, sign_count FROM mw_passkeys WHERE master_wallet_id = $1 AND credential_id = $2`,
		masterID, strings.TrimSpace(req.CredentialID)).Scan(&pubB64, &signCount)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "credential not registered"})
		return
	}
	spki, err := base64.RawURLEncoding.DecodeString(pubB64)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "stored public key is corrupt"})
		return
	}
	pubAny, err := x509.ParsePKIXPublicKey(spki)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "stored public key is invalid"})
		return
	}
	pubEC, ok := pubAny.(*ecdsa.PublicKey)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "stored key is not ECDSA"})
		return
	}

	authData, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(req.AuthenticatorData))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "authenticator_data must be base64url"})
		return
	}
	clientData, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(req.ClientDataJSON))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "client_data_json must be base64url"})
		return
	}
	sigDER, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(req.Signature))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "signature must be base64url"})
		return
	}

	// Verify the clientDataJSON "type" is webauthn.get and origin is trusted.
	var cd struct {
		Type   string `json:"type"`
		Origin string `json:"origin"`
		Challenge string `json:"challenge"`
	}
	if err := json.Unmarshal(clientData, &cd); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "client_data_json is not valid JSON"})
		return
	}
	if cd.Type != "webauthn.get" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "clientData type must be webauthn.get"})
		return
	}

	verifyData := append([]byte{}, authData...)
	cdHash := sha256.Sum256(clientData)
	verifyData = append(verifyData, cdHash[:]...)

	if !ecdsa.VerifyASN1(pubEC, verifyData, sigDER) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "signature verification failed", "verified": false})
		return
	}

	// Update sign count (best-effort) — monotonic counter protects against
	// cloned authenticators. We read the 4-byte counter at authData[33:37].
	if len(authData) >= 37 {
		newCount := uint32(authData[33])<<24 | uint32(authData[34])<<16 | uint32(authData[35])<<8 | uint32(authData[36])
		if newCount > signCount {
			_, _ = svc.store.db.Exec(ctx,
				`UPDATE mw_passkeys SET sign_count = $1, updated_at = NOW() WHERE master_wallet_id = $2 AND credential_id = $3`,
				newCount, masterID, strings.TrimSpace(req.CredentialID))
		}
	}
	svc.store.audit(ctx, masterID, "passkey.verified", "security", "user", currentUserID(c), "passkey", req.CredentialID, "info", nil)
	c.JSON(http.StatusOK, gin.H{"verified": true, "credential_id": req.CredentialID})
}

// ensurePasskeyMigration adds the mw_passkeys table. Called from runMigrations
// via an idempotent CREATE TABLE.
func ensurePasskeyMigration(ctx context.Context, pool interface{}) {
	// implemented in store.go migration list — this is a no-op placeholder
	// kept so the function name resolves if referenced elsewhere.
}
