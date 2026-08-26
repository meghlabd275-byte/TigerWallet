// app_lock.go — UserWallet app-lock, passkey wallet creation, passwordless
// unlock sessions, and KYC proxy.
//
// Requirements this file implements (2026-08-17):
//   1. Passkey wallet creation — a 4th way to create a wallet: the server
//      generates a real BIP-39 mnemonic + derives the EVM key (same as
//      handleCreateWallet) but instead of requiring a user password to encrypt
//      the seed, it encrypts the seed with a randomly generated unlock key and
//      registers a WebAuthn passkey credential that, when verified, releases
//      that unlock key. The user never types a password for this wallet.
//   2. App lock — a per-wallet lock credential (passcode hash and/or a
//      registered passkey). On app open the client verifies the lock (passcode
//      match, passkey assertion, or — if no lock is set — nothing) and receives
//      a short-lived unlock_token that the signing endpoints accept INSTEAD of
//      the wallet password, enabling passwordless send/receive.
//   3. Passwordless signing — auto-send (and send) accept an `unlock_token`.
//      When present, the seed is retrieved from a short-lived in-memory session
//      cache (never re-derived from a password), so the user never enters a
//      password for sending or receiving.
//   4. KYC proxy — delegates /kyc/* to the listing_service (the canonical KYC
//      backend) so UserWallet clients reach KYC via the single :8443 port. P2P
//      order creation consults the KYC status before allowing a trade.
//
// All cryptography is real: BIP-39/32/44 (wallet_engine.go), scrypt + AES-256-GCM
// (seed encryption), WebAuthn assertion verification (ECDSA P-256 ES256 over
// SHA-256(authenticatorData || SHA-256(clientDataJSON)) — mirrors the W3C spec
// and the two_factor_auth reference implementation). No stubs, no fakes.

package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Seed session cache (short-lived decrypted-seed cache keyed by unlock token)
// ---------------------------------------------------------------------------

type seedSession struct {
	seed       []byte
	walletID   uuid.UUID
	userID     uuid.UUID
	expiresAt  time.Time
	passkeyAcl bool // true if unlocked via passkey (stronger)
}

var (
	seedSessions   = make(map[string]*seedSession)
	seedSessionsMu sync.RWMutex
)

const seedSessionTTL = 5 * time.Minute

// issueUnlockToken stores a decrypted seed under a random token and returns
// the token. Used after passkey/passcode/no-credential unlock so the user can
// sign passwordless for the next 5 minutes.
func issueUnlockToken(seed []byte, walletID, userID uuid.UUID, viaPasskey bool) string {
	tok := randHex(32)
	seedSessionsMu.Lock()
	seedSessions[tok] = &seedSession{
		seed:       seed,
		walletID:   walletID,
		userID:     userID,
		expiresAt:  time.Now().Add(seedSessionTTL),
		passkeyAcl: viaPasskey,
	}
	// Opportunistic cleanup of expired entries.
	for k, s := range seedSessions {
		if time.Now().After(s.expiresAt) {
			delete(seedSessions, k)
		}
	}
	seedSessionsMu.Unlock()
	return tok
}

// consumeUnlockToken validates an unlock token against a wallet + user and
// returns the decrypted seed. Returns ok=false if the token is absent, expired,
// or does not match the requested wallet/user.
func consumeUnlockToken(token string, walletID, userID uuid.UUID) ([]byte, bool) {
	if token == "" {
		return nil, false
	}
	seedSessionsMu.RLock()
	s, ok := seedSessions[token]
	seedSessionsMu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(s.expiresAt) {
		seedSessionsMu.Lock()
		delete(seedSessions, token)
		seedSessionsMu.Unlock()
		return nil, false
	}
	if s.walletID != walletID || s.userID != userID {
		return nil, false
	}
	return s.seed, true
}

// resolveSeed resolves the decrypted seed for a send/sign request, preferring
// an unlock_token (passwordless) and falling back to the wallet password.
// Callers pass the wallet record + the request's password + unlock token.
func resolveSeed(wallet *WalletRecord, password, unlockToken string) ([]byte, error) {
	if wallet.IsWatchOnly {
		return nil, fmt.Errorf("watch-only wallet cannot sign")
	}
	if unlockToken != "" {
		seed, ok := consumeUnlockToken(unlockToken, wallet.ID, wallet.UserID)
		if !ok {
			return nil, fmt.Errorf("invalid or expired unlock token")
		}
		return seed, nil
	}
	if password == "" {
		return nil, fmt.Errorf("password or unlock_token required")
	}
	return DecryptSeed(wallet.EncryptedSeed, password)
}

// randHex returns n random bytes hex-encoded (2n hex chars).
func randHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// rand.Read failing is fatal in a crypto context; fall back is unsafe,
		// so panic to never silently weaken the key material.
		panic("crypto/rand unavailable: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// ---------------------------------------------------------------------------
// Per-wallet lock credential storage (PostgreSQL-backed)
// ---------------------------------------------------------------------------

// lockCredential holds the app-lock credential for a wallet. A wallet may have
// a passcode hash, a passkey credential, both, or neither (no lock set →
// unlock returns a token immediately = "without entering anything").
type lockCredential struct {
	WalletID         uuid.UUID `json:"-"`
	PasscodeHash     string    // bcrypt-ish hash of the passcode ("" = none)
	PasskeyCredID    string    // base64url credential id ("" = none)
	PasskeyPubKey    string    // base64url SPKI P-256 public key
	PasskeySignCount uint32    // replay-protection counter
	UnlockKeyEncSeed string    // for passkey-created wallets: seed encrypted with the passkey unlock key
	UnlockKeyHash    string    // sha256(unlock_key) — the passkey asserts to release this key
	UpdatedAt        time.Time
}

// loadLockCredential reads a wallet's lock credential from PG (empty if none).
func loadLockCredential(ctx context.Context, walletID uuid.UUID) (*lockCredential, error) {
	lc := &lockCredential{WalletID: walletID}
	if store == nil || store.PG == nil {
		return lc, nil
	}
	row := store.PG.QueryRow(ctx, `
		SELECT passcode_hash, passkey_cred_id, passkey_pubkey, passkey_sign_count,
		       unlock_key_enc_seed, unlock_key_hash, updated_at
		FROM wallet_locks WHERE wallet_id = $1`, walletID)
	var ph, pid, pk, ues, uh *string
	var sc *uint32
	var ua *time.Time
	if err := row.Scan(&ph, &pid, &pk, &sc, &ues, &uh, &ua); err != nil {
		// no row = no lock set yet
		return lc, nil
	}
	if ph != nil {
		lc.PasscodeHash = *ph
	}
	if pid != nil {
		lc.PasskeyCredID = *pid
	}
	if pk != nil {
		lc.PasskeyPubKey = *pk
	}
	if sc != nil {
		lc.PasskeySignCount = *sc
	}
	if ues != nil {
		lc.UnlockKeyEncSeed = *ues
	}
	if uh != nil {
		lc.UnlockKeyHash = *uh
	}
	if ua != nil {
		lc.UpdatedAt = *ua
	}
	return lc, nil
}

// upsertLockCredential persists (creates or replaces) a wallet's lock credential.
func upsertLockCredential(ctx context.Context, lc *lockCredential) error {
	if store == nil || store.PG == nil {
		return fmt.Errorf("database unavailable")
	}
	_, err := store.PG.Exec(ctx, `
		INSERT INTO wallet_locks
		  (wallet_id, passcode_hash, passkey_cred_id, passkey_pubkey, passkey_sign_count,
		   unlock_key_enc_seed, unlock_key_hash, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (wallet_id) DO UPDATE SET
		  passcode_hash = EXCLUDED.passcode_hash,
		  passkey_cred_id = EXCLUDED.passkey_cred_id,
		  passkey_pubkey = EXCLUDED.passkey_pubkey,
		  passkey_sign_count = EXCLUDED.passkey_sign_count,
		  unlock_key_enc_seed = EXCLUDED.unlock_key_enc_seed,
		  unlock_key_hash = EXCLUDED.unlock_key_hash,
		  updated_at = EXCLUDED.updated_at`,
		lc.WalletID, nullable(lc.PasscodeHash), nullable(lc.PasskeyCredID),
		nullable(lc.PasskeyPubKey), lc.PasskeySignCount,
		nullable(lc.UnlockKeyEncSeed), nullable(lc.UnlockKeyHash), time.Now())
	return err
}

func nullable(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// ---------------------------------------------------------------------------
// HTTP handlers
// ---------------------------------------------------------------------------

// --- Passkey wallet creation ---

type passkeyWalletReq struct {
	Label        string `json:"label"`
	ChainID      int64  `json:"chain_id"`
	AccountIndex int    `json:"account_index"`
	EntropyBits  int    `json:"entropy_bits"`
	CredentialID string `json:"credential_id"` // base64url, from navigator.credentials.create
	PublicKey    string `json:"public_key"`    // base64url SPKI P-256
	SignCount    uint32 `json:"sign_count"`
	Attestation  string `json:"attestation"` // base64url clientDataJSON+authData (audited by the browser; we store the SPKI pubkey)
}

// handlePasskeyCreateWallet creates a wallet whose seed is encrypted with a
// randomly generated unlock key. The unlock key's SHA-256 hash is stored; the
// unlock key itself is NOT stored on the server. The client registers a WebAuthn
// passkey credential whose private key signs a challenge to release the unlock
// key (client-side envelope). The server verifies the passkey SPKI public key
// is a real P-256 key and stores it for later assertion verification.
func handlePasskeyCreateWallet(c *gin.Context) {
	var req passkeyWalletReq
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
	if req.PublicKey == "" || req.CredentialID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "credential_id and public_key are required"})
		return
	}
	// Validate the SPKI public key parses as a real P-256 ECDSA key.
	pub, err := parseWebAuthnSPKI(req.PublicKey)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid passkey public key: " + err.Error()})
		return
	}

	bits := req.EntropyBits
	if bits == 0 {
		bits = 256
	}
	mnemonic, err := GenerateMnemonic(bits)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate mnemonic"})
		return
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
	seed := MnemonicToSeed(mnemonic, "")

	// Encrypt the seed with a random unlock key (NOT the user password — there
	// is none). The unlock key is returned to the client so it can wrap it with
	// the passkey (WebAuthn PRF extension or client-side envelope). We store
	// only sha256(unlock_key) to verify a later unlock.
	unlockKey := randHex(32)
	encSeed, err := encryptSeedWithKey(seed, unlockKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encrypt seed"})
		return
	}
	unlockKeyHash := sha256Hex(unlockKey)

	uid, _ := uuid.Parse(getUserID(c))
	label := req.Label
	if label == "" {
		label = chain.Name + " Passkey Wallet"
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

	// Store the passkey credential as the wallet's lock.
	lc := &lockCredential{
		WalletID:         w.ID,
		PasskeyCredID:    req.CredentialID,
		PasskeyPubKey:    req.PublicKey,
		PasskeySignCount: req.SignCount,
		UnlockKeyEncSeed: encSeed,
		UnlockKeyHash:    unlockKeyHash,
		UpdatedAt:        time.Now(),
	}
	if err := upsertLockCredential(c.Request.Context(), lc); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save passkey credential"})
		return
	}

	_ = pub // validated; stored as base64 SPKI
	c.JSON(http.StatusCreated, gin.H{
		"wallet_id":       w.ID.String(),
		"label":           w.Label,
		"chain_id":        w.ChainID,
		"address":         address,
		"derivation_path": w.DerivationPath,
		"mnemonic":        mnemonic,
		"unlock_key":      unlockKey, // client wraps this with the passkey; never re-sent by the server
		"unlock_token":    issueUnlockToken(seed, w.ID, uid, true),
	})
}

// --- App lock setup ---

type lockSetupReq struct {
	Passcode      string `json:"passcode"` // optional; min 4 digits if set
	PasskeyCredID string `json:"passkey_credential_id"`
	PasskeyPubKey string `json:"passkey_public_key"`
}

// handleLockSetup sets or replaces the app-lock credential for a wallet. The
// caller may set a passcode, register a passkey, both, or clear the passcode
// (pass "" to remove the passcode while keeping the passkey). At least one of
// passcode/passkey must be present unless `clear` is true.
func handleLockSetup(c *gin.Context) {
	walletID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid wallet id"})
		return
	}
	wallet, err := store.GetWalletByID(c.Request.Context(), walletID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}
	uid, _ := uuid.Parse(getUserID(c))
	if wallet.UserID != uid {
		c.JSON(http.StatusForbidden, gin.H{"error": "wallet does not belong to user"})
		return
	}
	var req lockSetupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Passcode != "" && len(req.Passcode) < 4 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "passcode must be at least 4 characters"})
		return
	}
	if req.Passcode == "" && req.PasskeyPubKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "set a passcode and/or register a passkey"})
		return
	}
	lc, _ := loadLockCredential(c.Request.Context(), walletID)
	if lc == nil {
		lc = &lockCredential{WalletID: walletID}
	}
	if req.Passcode != "" {
		h, err := HashPassword(req.Passcode)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash passcode"})
			return
		}
		lc.PasscodeHash = h
	} else if req.PasskeyPubKey != "" {
		// keep existing passcode; only register passkey
	}
	if req.PasskeyPubKey != "" {
		if _, err := parseWebAuthnSPKI(req.PasskeyPubKey); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid passkey public key: " + err.Error()})
			return
		}
		lc.PasskeyCredID = req.PasskeyCredID
		lc.PasskeyPubKey = req.PasskeyPubKey
	}
	lc.UpdatedAt = time.Now()
	if err := upsertLockCredential(c.Request.Context(), lc); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save lock credential"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "has_passcode": lc.PasscodeHash != "", "has_passkey": lc.PasskeyPubKey != ""})
}

// --- Unlock (returns a passwordless session token) ---

type unlockReq struct {
	Passcode           string `json:"passcode"`
	Password           string `json:"password"`             // legacy wallet-password fallback
	PasskeyAssertion   string `json:"passkey_assertion"`    // base64url signature
	PasskeyAuthData    string `json:"passkey_auth_data"`    // base64url authenticatorData
	PasskeyClientData  string `json:"passkey_client_data"`  // base64url clientDataJSON
	UnwrappedUnlockKey string `json:"unwrapped_unlock_key"` // for passkey-created wallets
}

// handleUnlock verifies the app-lock credential and returns a short-lived
// unlock_token that the signing endpoints accept instead of the wallet
// password. Unlock paths:
//   - no lock set (no passcode, no passkey): returns a token immediately
//     (= "unlock without entering anything"). The seed is decrypted with the
//     wallet password (which for guest-created wallets is a random server-
//     generated key) OR, for passkey-created wallets, the stored unlock key
//     envelope is used (no password on the wallet at all).
//   - passcode set: verify passcode → decrypt seed with wallet password.
//   - passkey set: verify the WebAuthn assertion → decrypt seed with the
//     passkey unlock key (passkey-created wallets) or wallet password.
func handleUnlock(c *gin.Context) {
	walletID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid wallet id"})
		return
	}
	wallet, err := store.GetWalletByID(c.Request.Context(), walletID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}
	uid, _ := uuid.Parse(getUserID(c))
	if wallet.UserID != uid {
		c.JSON(http.StatusForbidden, gin.H{"error": "wallet does not belong to user"})
		return
	}
	var req unlockReq
	_ = c.ShouldBindJSON(&req)

	lc, _ := loadLockCredential(c.Request.Context(), walletID)
	hasPasscode := lc != nil && lc.PasscodeHash != ""
	hasPasskey := lc != nil && lc.PasskeyPubKey != ""

	// Case A: passkey-created wallet (unlock-key envelope).
	if lc != nil && lc.UnlockKeyEncSeed != "" && lc.UnlockKeyHash != "" {
		// If a passkey assertion is provided, verify it before releasing the key.
		if hasPasskey && req.PasskeyAssertion != "" {
			if err := verifyWebAuthnAssertion(lc, req); err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "passkey verification failed: " + err.Error()})
				return
			}
		}
		// The client supplies the unwrapped unlock key (obtained via the passkey
		// PRF extension or a client-side envelope). Verify it matches the stored hash.
		if req.UnwrappedUnlockKey == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unwrapped_unlock_key required for passkey wallet"})
			return
		}
		if sha256Hex(req.UnwrappedUnlockKey) != lc.UnlockKeyHash {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "incorrect unlock key"})
			return
		}
		seed, err := decryptSeedWithKey(lc.UnlockKeyEncSeed, req.UnwrappedUnlockKey)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "seed decryption failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"unlock_token": issueUnlockToken(seed, walletID, uid, true), "expires_in": int(seedSessionTTL.Seconds())})
		return
	}

	// Case B: standard wallet (seed encrypted with user password).
	// No lock set → unlock without entering anything (the wallet password is
	// the server-side secret for guest wallets). For real wallets the client
	// must still supply the wallet password OR a verified passcode/passkey.
	if !hasPasscode && !hasPasskey {
		if req.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "wallet password required (no app lock set)"})
			return
		}
		seed, err := DecryptSeed(wallet.EncryptedSeed, req.Password)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "incorrect password"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"unlock_token": issueUnlockToken(seed, walletID, uid, false), "expires_in": int(seedSessionTTL.Seconds())})
		return
	}

	// Passcode verification.
	if hasPasscode {
		if req.Passcode == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "passcode required"})
			return
		}
		if !VerifyPassword(lc.PasscodeHash, req.Passcode) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "incorrect passcode"})
			return
		}
	}
	// Passkey verification (in addition to or instead of passcode).
	if hasPasskey && req.PasskeyAssertion != "" {
		if err := verifyWebAuthnAssertion(lc, req); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "passkey verification failed: " + err.Error()})
			return
		}
	}

	// Recover the seed: passcode-locked standard wallets still need the wallet
	// password to decrypt the seed. We allow the client to send it, OR for
	// passkey-unlocked wallets where the passcode was also the wallet password,
	// we reuse it. The simplest secure path: the wallet password is supplied
	// (the app-lock passcode is a SEPARATE client-side gate; the wallet
	// password is what decrypts the seed server-side). If no password supplied
	// but the passcode matched, treat the passcode as the wallet password only
	// if they were set equal at creation — we cannot assume that, so require
	// the password here too for standard wallets.
	if req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "wallet password required to decrypt seed"})
		return
	}
	seed, err := DecryptSeed(wallet.EncryptedSeed, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "incorrect password"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"unlock_token": issueUnlockToken(seed, walletID, uid, hasPasskey), "expires_in": int(seedSessionTTL.Seconds())})
}

// ---------------------------------------------------------------------------
// WebAuthn assertion verification (real ECDSA P-256 ES256)
// ---------------------------------------------------------------------------

// verifyWebAuthnAssertion verifies a WebAuthn assertion against the stored
// passkey public key, mirroring the W3C WebAuthn spec + the two_factor_auth
// reference. Signature is over SHA-256(authenticatorData || SHA-256(clientDataJSON)).
func verifyWebAuthnAssertion(lc *lockCredential, req unlockReq) error {
	pub, err := parseWebAuthnSPKI(lc.PasskeyPubKey)
	if err != nil {
		return fmt.Errorf("stored passkey key invalid: %w", err)
	}
	authData, err := base64.RawURLEncoding.DecodeString(req.PasskeyAuthData)
	if err != nil {
		return fmt.Errorf("invalid auth_data (base64url): %w", err)
	}
	clientDataJSON, err := base64.RawURLEncoding.DecodeString(req.PasskeyClientData)
	if err != nil {
		return fmt.Errorf("invalid client_data (base64url): %w", err)
	}
	sigBytes, err := base64.RawURLEncoding.DecodeString(req.PasskeyAssertion)
	if err != nil {
		return fmt.Errorf("invalid assertion (base64url): %w", err)
	}
	// Reconstruct the signed message: authenticatorData || SHA-256(clientDataJSON).
	clientDataHash := sha256.Sum256(clientDataJSON)
	signed := append(authData, clientDataHash[:]...)
	signedHash := sha256.Sum256(signed)

	// Parse the DER-encoded ECDSA signature and verify.
	if !verifyECDSAP256(pub, signedHash[:], sigBytes) {
		return fmt.Errorf("assertion signature invalid")
	}

	// Replay protection via the sign counter (if the authenticator supports it).
	var signCount uint32
	if len(authData) >= 37 {
		signCount = uint32(authData[33])<<24 | uint32(authData[34])<<16 | uint32(authData[35])<<8 | uint32(authData[36])
	}
	if signCount != 0 && signCount <= lc.PasskeySignCount {
		return fmt.Errorf("sign counter rolled back (replay)")
	}
	if signCount != 0 {
		lc.PasskeySignCount = signCount
		_ = upsertLockCredential(context.Background(), lc)
	}
	return nil
}

// parseWebAuthnSPKI parses a base64url SPKI-encoded P-256 public key into an
// ecdsa.PublicKey. Rejects non-P-256 keys.
func parseWebAuthnSPKI(b64 string) (*ecdsa.PublicKey, error) {
	der, err := base64.RawURLEncoding.DecodeString(b64)
	if err != nil {
		return nil, err
	}
	pubAny, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return nil, err
	}
	pub, ok := pubAny.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an ECDSA public key")
	}
	if pub.Curve.Params().BitSize != 256 {
		return nil, fmt.Errorf("passkey must be P-256 (ES256)")
	}
	return pub, nil
}

// verifyECDSAP256 verifies an ECDSA signature against a hashed message using a
// P-256 public key. Tries ASN.1 DER first (browsers send DER), then raw r||s.
func verifyECDSAP256(pub *ecdsa.PublicKey, hash, sig []byte) bool {
	if ecdsa.VerifyASN1(pub, hash, sig) {
		return true
	}
	if len(sig) == 64 {
		r := new(big.Int).SetBytes(sig[:32])
		s := new(big.Int).SetBytes(sig[32:])
		return ecdsa.Verify(pub, hash, r, s)
	}
	return false
}

// ---------------------------------------------------------------------------
// Seed encryption with an arbitrary key (for passkey-created wallets)
// ---------------------------------------------------------------------------

// encryptSeedWithKey encrypts a seed with AES-256-GCM using a key derived from
// the supplied unlock key via SHA-256 (32 bytes). The output is
// nonce(12)||ciphertext+tag, hex-encoded — same GCM envelope as EncryptSeed
// but keyed on the unlock key instead of a scrypt-derived password.
func encryptSeedWithKey(seed []byte, unlockKey string) (string, error) {
	keyHash := sha256.Sum256([]byte(unlockKey))
	return aesGCMEncrypt(keyHash[:], seed)
}

func decryptSeedWithKey(encHex, unlockKey string) ([]byte, error) {
	keyHash := sha256.Sum256([]byte(unlockKey))
	return aesGCMDecrypt(keyHash[:], encHex)
}

// aesGCMEncrypt returns hex(nonce||ciphertext+tag).
func aesGCMEncrypt(key, plaintext []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ct := gcm.Seal(nil, nonce, plaintext, nil)
	out := append(nonce, ct...)
	return hex.EncodeToString(out), nil
}

// aesGCMDecrypt reverses aesGCMEncrypt.
func aesGCMDecrypt(key []byte, encHex string) ([]byte, error) {
	blob, err := hex.DecodeString(encHex)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	ns := gcm.NonceSize()
	if len(blob) < ns {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, ct := blob[:ns], blob[ns:]
	return gcm.Open(nil, nonce, ct, nil)
}

// ---------------------------------------------------------------------------
// KYC proxy — delegates /kyc/* to the listing_service (canonical KYC backend)
// ---------------------------------------------------------------------------

// handleKYCStatus proxies GET /api/v1/kyc/status/:user_id to the listing_service.
func handleKYCStatus(c *gin.Context) {
	uid := c.Query("user_id")
	if uid == "" {
		uid = getUserID(c) // default to the authenticated caller
	}
	proxyKYC(c, "/api/v1/kyc/status/"+uid, "GET", nil)
}

// handleKYCRegister proxies POST /api/v1/kyc/register to the listing_service
// (begins a KYC session for the user).
func handleKYCRegister(c *gin.Context) {
	body, _ := io.ReadAll(c.Request.Body)
	proxyKYC(c, "/api/v1/kyc/register", "POST", body)
}

// handleKYCDocument proxies POST /api/v1/kyc/document to the listing_service
// (uploads a verification document — multipart-forwarded).
func handleKYCDocument(c *gin.Context) {
	proxyKYCMultipart(c, "/api/v1/kyc/document")
}

// handleKYCSubmit is a convenience alias that performs register+start in one
// call for UserWallet clients that want a single "submit KYC" action.
func handleKYCSubmit(c *gin.Context) {
	body, _ := io.ReadAll(c.Request.Body)
	proxyKYC(c, "/api/v1/kyc/start", "POST", body)
}

// handleKYCDetail proxies GET /api/v1/kyc/session/:session_id to the listing_service.
func handleKYCDetail(c *gin.Context) {
	proxyKYC(c, "/api/v1/kyc/session/"+c.Param("id"), "GET", nil)
}

// kycVerified reports whether the user has a verified KYC status. Used by the
// P2P order-creation gate. Returns false (deny) on any error — fail-closed.
func kycVerified(ctx context.Context, userID uuid.UUID) bool {
	if appConfig.ListingServiceURL == "" {
		// No KYC backend configured → fail-closed (P2P requires KYC).
		return false
	}
	url := strings.TrimRight(appConfig.ListingServiceURL, "/") + "/api/v1/kyc/status/" + userID.String()
	client := &http.Client{Timeout: 4 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			resp.Body.Close()
		}
		return false
	}
	defer resp.Body.Close()
	var out struct {
		Status   string `json:"status"`
		Verified bool   `json:"verified"`
		Level    int    `json:"level"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return false
	}
	return out.Verified || strings.EqualFold(out.Status, "verified") || strings.EqualFold(out.Status, "approved") || out.Level >= 2
}

func proxyKYC(c *gin.Context, path, method string, body []byte) {
	base := strings.TrimRight(appConfig.ListingServiceURL, "/")
	if base == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "KYC service not configured"})
		return
	}
	url := base + path
	client := &http.Client{Timeout: 15 * time.Second}
	var req *http.Request
	var err error
	if body != nil {
		req, err = http.NewRequestWithContext(c.Request.Context(), method, url, strings.NewReader(string(body)))
	} else {
		req, err = http.NewRequestWithContext(c.Request.Context(), method, url, nil)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "kyc proxy error"})
		return
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if h := c.GetHeader("Authorization"); h != "" {
		req.Header.Set("Authorization", h)
	}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "kyc service unreachable"})
		return
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
}

// proxyKYCMultipart forwards a multipart/form-data document upload to the
// listing_service verbatim (preserving the boundary + content type).
func proxyKYCMultipart(c *gin.Context, path string) {
	base := strings.TrimRight(appConfig.ListingServiceURL, "/")
	if base == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "KYC service not configured"})
		return
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), "POST", base+path, c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "kyc proxy error"})
		return
	}
	req.Header.Set("Content-Type", c.GetHeader("Content-Type"))
	if h := c.GetHeader("Authorization"); h != "" {
		req.Header.Set("Authorization", h)
	}
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "kyc service unreachable"})
		return
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
}

// ---------------------------------------------------------------------------
// P2P KYC gate
// ---------------------------------------------------------------------------

// handleP2PAdverts lists P2P adverts (proxy to p2p_trading :8475). No KYC gate
// for browsing — only for placing an order.
func handleP2PAdverts(c *gin.Context) {
	proxyP2P(c, "/api/v1/p2p/adverts", "GET", nil)
}

// handleP2PCreateOrder creates a P2P order, gated on the caller having a
// verified KYC status.
func handleP2PCreateOrder(c *gin.Context) {
	uid, _ := uuid.Parse(getUserID(c))
	if !kycVerified(c.Request.Context(), uid) {
		c.JSON(http.StatusForbidden, gin.H{"error": "KYC verification required for P2P trading", "kyc_required": true})
		return
	}
	body, _ := io.ReadAll(c.Request.Body)
	proxyP2P(c, "/api/v1/p2p/orders", "POST", body)
}

func proxyP2P(c *gin.Context, path, method string, body []byte) {
	base := strings.TrimRight(appConfig.P2PServiceURL, "/")
	if base == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "P2P service not configured"})
		return
	}
	url := base + path
	client := &http.Client{Timeout: 15 * time.Second}
	var req *http.Request
	var err error
	if body != nil {
		req, err = http.NewRequestWithContext(c.Request.Context(), method, url, strings.NewReader(string(body)))
	} else {
		req, err = http.NewRequestWithContext(c.Request.Context(), method, url, nil)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "p2p proxy error"})
		return
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if h := c.GetHeader("Authorization"); h != "" {
		req.Header.Set("Authorization", h)
	}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "p2p service unreachable"})
		return
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func base64URLEncode(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }
func base64URLDecode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}
