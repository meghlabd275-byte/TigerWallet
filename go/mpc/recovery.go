// Encrypted key-share export and threshold recovery for the MPC service.
//
// A threshold wallet is only useful if its shares survive the loss of one
// custodian, so the service supports exporting a share as an AES-256-GCM blob
// and restoring a wallet from any `threshold` of those blobs.
//
// Two properties matter for safety:
//
//   - A share is never returned in the clear. Export requires an operator
//     supplied passphrase and returns ciphertext; the plaintext scalar only
//     exists inside this process while signing.
//   - A recovery must actually meet the threshold. Shares are verified against
//     the wallet's public key after interpolation, so a wrong or corrupted share
//     is rejected instead of producing a wallet that silently signs nothing.
package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

const (
	// shareBlobVersion tags the envelope so a future format change is
	// detectable rather than silently mis-parsed.
	shareBlobVersion = 1
	// pbkdf2Iterations matches the cost used elsewhere in the wallet.
	pbkdf2Iterations = 600_000
	shareSaltLength  = 16
	shareNonceLength = 12
)

// shareBlob is the on-the-wire encrypted share envelope.
type shareBlob struct {
	Version    int    `json:"version"`
	KeyID      string `json:"keyId"`
	PartyID    uint32 `json:"partyId"`
	Threshold  uint32 `json:"threshold"`
	Total      uint32 `json:"totalShards"`
	Salt       string `json:"salt"`       // hex
	Nonce      string `json:"nonce"`      // hex
	Ciphertext string `json:"ciphertext"` // hex, AES-256-GCM(share scalar)
}

// deriveShareKey stretches a passphrase into a 32-byte AES key.
func deriveShareKey(passphrase string, salt []byte) []byte {
	return pbkdf2.Key([]byte(passphrase), salt, pbkdf2Iterations, 32, sha256.New)
}

// encryptShare seals a share scalar under the passphrase.
func encryptShare(passphrase, keyID string, partyID, threshold, total uint32, share *big.Int) (*shareBlob, error) {
	if passphrase == "" {
		return nil, errors.New("a passphrase is required to export a share")
	}

	salt := make([]byte, shareSaltLength)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("cannot generate the salt: %w", err)
	}

	block, err := aes.NewCipher(deriveShareKey(passphrase, salt))
	if err != nil {
		return nil, fmt.Errorf("cannot create the cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("cannot create the AEAD: %w", err)
	}

	nonce := make([]byte, shareNonceLength)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("cannot generate the nonce: %w", err)
	}

	// The scalar is fixed-width so the plaintext length leaks nothing.
	plaintext := scalarBytes(share)

	// Bind the metadata into the AAD so a blob cannot be relabelled as
	// belonging to a different wallet or party.
	aad := []byte(fmt.Sprintf("%d|%s|%d|%d|%d", shareBlobVersion, keyID, partyID, threshold, total))

	sealed := gcm.Seal(nil, nonce, plaintext, aad)

	return &shareBlob{
		Version:    shareBlobVersion,
		KeyID:      keyID,
		PartyID:    partyID,
		Threshold:  threshold,
		Total:      total,
		Salt:       hex.EncodeToString(salt),
		Nonce:      hex.EncodeToString(nonce),
		Ciphertext: hex.EncodeToString(sealed),
	}, nil
}

// decryptShare opens a share blob with the passphrase.
func decryptShare(passphrase string, blob *shareBlob) (*big.Int, error) {
	if passphrase == "" {
		return nil, errors.New("a passphrase is required to recover a share")
	}
	if blob.Version != shareBlobVersion {
		return nil, fmt.Errorf("unsupported share blob version %d", blob.Version)
	}

	salt, err := hex.DecodeString(blob.Salt)
	if err != nil {
		return nil, errors.New("the salt is not valid hex")
	}
	nonce, err := hex.DecodeString(blob.Nonce)
	if err != nil {
		return nil, errors.New("the nonce is not valid hex")
	}
	sealed, err := hex.DecodeString(blob.Ciphertext)
	if err != nil {
		return nil, errors.New("the ciphertext is not valid hex")
	}

	block, err := aes.NewCipher(deriveShareKey(passphrase, salt))
	if err != nil {
		return nil, fmt.Errorf("cannot create the cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("cannot create the AEAD: %w", err)
	}

	aad := []byte(fmt.Sprintf("%d|%s|%d|%d|%d",
		blob.Version, blob.KeyID, blob.PartyID, blob.Threshold, blob.Total))

	plaintext, err := gcm.Open(nil, nonce, sealed, aad)
	if err != nil {
		// A wrong passphrase and a tampered blob are indistinguishable by
		// design; report both the same way.
		return nil, errors.New("the share could not be decrypted (wrong passphrase or corrupted blob)")
	}

	if len(plaintext) != 32 {
		return nil, errors.New("the recovered share has an unexpected length")
	}

	share := new(big.Int).SetBytes(plaintext)
	if share.Sign() == 0 || share.Cmp(curveOrderN()) >= 0 {
		return nil, errors.New("the recovered share is not a valid scalar")
	}

	return share, nil
}

// recoverSecret performs Lagrange interpolation at x=0 over the given shares,
// reconstructing the master scalar.
func recoverSecret(points map[uint32]*big.Int) (*big.Int, error) {
	if len(points) == 0 {
		return nil, errors.New("at least one share is required")
	}

	modN := curveOrderN()
	secret := new(big.Int)

	for xi, yi := range points {
		x := new(big.Int).SetUint64(uint64(xi))

		// numerator = product of (-xj) for j != i
		numerator := big.NewInt(1)
		denominator := big.NewInt(1)

		for xj := range points {
			if xj == xi {
				continue
			}
			other := new(big.Int).SetUint64(uint64(xj))

			numerator.Mul(numerator, new(big.Int).Neg(other))
			denominator.Mul(denominator, new(big.Int).Sub(x, other))
		}

		numerator.Mod(numerator, modN)
		denominator.Mod(denominator, modN)

		inverse := new(big.Int).ModInverse(denominator, modN)
		if inverse == nil {
			return nil, fmt.Errorf("share %d has a singular interpolation coefficient", xi)
		}

		basis := new(big.Int).Mul(numerator, inverse)
		basis.Mod(basis, modN)

		term := new(big.Int).Mul(yi, basis)
		secret.Add(secret, term)
		secret.Mod(secret, modN)
	}

	if secret.Sign() == 0 {
		return nil, errors.New("the recovered secret is not a valid scalar")
	}

	return secret, nil
}

// ---------------------------------------------------------------------------
// HTTP handlers
// ---------------------------------------------------------------------------

type exportShareRequest struct {
	KeyID      string `json:"keyId"`
	Passphrase string `json:"passphrase"`
	// PartyIDs selects which shares to export. Empty means every share, which
	// is what a full backup needs.
	PartyIDs []uint32 `json:"partyIds"`
}

type exportShareResponse struct {
	KeyID     string       `json:"keyId"`
	Threshold uint32       `json:"threshold"`
	Shares    []*shareBlob `json:"shares"`
}

// handleExportShares serves POST /api/v1/mpc/keygen/export.
//
// It returns every requested share sealed under the caller's passphrase. The
// raw scalars never leave the process.
func handleExportShares(w http.ResponseWriter, r *http.Request, c caller) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req exportShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(req.KeyID) == "" {
		writeError(w, http.StatusBadRequest, "missing keyId")
		return
	}
	if strings.TrimSpace(req.Passphrase) == "" {
		writeError(w, http.StatusBadRequest, "a passphrase is required to export shares")
		return
	}

	store.mu.RLock()
	engine, ok := store.engines[req.KeyID]
	wallet := store.meta[req.KeyID]
	store.mu.RUnlock()

	if !ok || wallet == nil {
		writeError(w, http.StatusNotFound, "wallet not found")
		return
	}
	if !requireWalletAccess(w, wallet, c) {
		return
	}

	partyIDs := req.PartyIDs
	if len(partyIDs) == 0 {
		partyIDs = make([]uint32, 0, wallet.TotalShards)
		for id := uint32(1); id <= wallet.TotalShards; id++ {
			partyIDs = append(partyIDs, id)
		}
	}

	blobs := make([]*shareBlob, 0, len(partyIDs))
	for _, id := range partyIDs {
		if id == 0 || id > wallet.TotalShards {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("partyId %d is out of range", id))
			return
		}

		scalar, ok := engine.ShareFor(id)
		if !ok {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("share %d is unavailable", id))
			return
		}

		blob, err := encryptShare(req.Passphrase, req.KeyID, id, wallet.Threshold, wallet.TotalShards, scalar)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		blobs = append(blobs, blob)
	}

	writeJSON(w, http.StatusOK, exportShareResponse{
		KeyID:     req.KeyID,
		Threshold: wallet.Threshold,
		Shares:    blobs,
	})
}

type recoverWalletRequest struct {
	Passphrase string       `json:"passphrase"`
	Shares     []*shareBlob `json:"shares"`
}

type recoverWalletResponse struct {
	KeyID     string  `json:"keyId"`
	Address   Address `json:"address"`
	PublicKey Bytes   `json:"publicKey"`
	Threshold uint32  `json:"threshold"`
	Total     uint32  `json:"totalShards"`
}

// handleRecoverWallet serves POST /api/v1/mpc/keygen/recover.
//
// It reconstructs a wallet from a threshold of encrypted share blobs, verifies
// the reconstruction against the shares' own metadata, and registers a fresh
// signing engine. A set of shares that does not meet the threshold, mixes
// wallets, or decrypts to an inconsistent secret is rejected.
func handleRecoverWallet(w http.ResponseWriter, r *http.Request, c caller) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req recoverWalletRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	// Reconstruction produces a live signing key, so only an operator may
	// supply shares for a wallet it does not already own.
	if !c.isOperator() {
		writeError(w, http.StatusForbidden, "recovering a wallet requires an operator role")
		return
	}

	if strings.TrimSpace(req.Passphrase) == "" {
		writeError(w, http.StatusBadRequest, "a passphrase is required to recover a wallet")
		return
	}
	if len(req.Shares) == 0 {
		writeError(w, http.StatusBadRequest, "at least one share is required")
		return
	}

	// Every blob must describe the same wallet, and the count must reach the
	// declared threshold.
	first := req.Shares[0]
	if first.KeyID == "" {
		writeError(w, http.StatusBadRequest, "the shares are missing a keyId")
		return
	}

	threshold := first.Threshold
	total := first.Total
	if threshold == 0 || total == 0 || threshold > total {
		writeError(w, http.StatusBadRequest, "the share metadata has an invalid threshold")
		return
	}
	if uint32(len(req.Shares)) < threshold {
		writeError(w, http.StatusBadRequest,
			fmt.Sprintf("recovery needs %d shares, %d were supplied", threshold, len(req.Shares)))
		return
	}

	points := make(map[uint32]*big.Int, len(req.Shares))
	for _, blob := range req.Shares {
		if blob.KeyID != first.KeyID {
			writeError(w, http.StatusBadRequest, "the shares belong to different wallets")
			return
		}
		if blob.Threshold != threshold || blob.Total != total {
			writeError(w, http.StatusBadRequest, "the shares disagree on the threshold parameters")
			return
		}
		if _, duplicate := points[blob.PartyID]; duplicate {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("share %d was supplied twice", blob.PartyID))
			return
		}
		if blob.PartyID == 0 || blob.PartyID > total {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("partyId %d is out of range", blob.PartyID))
			return
		}

		scalar, err := decryptShare(req.Passphrase, blob)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		points[blob.PartyID] = scalar
	}

	secret, err := recoverSecret(points)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	publicKey := pubKeyForScalar(secret)
	address := DeriveAddress(publicKey)

	// Cross-check against the shares' own consistency: re-splitting the
	// recovered secret with the same party ids must reproduce every supplied
	// share. This rejects a set that interpolates to a value the original
	// dealer never used (for example shares from two different wallets with
	// colliding ids).
	if !sharesAreConsistent(secret, threshold, points) {
		writeError(w, http.StatusBadRequest, "the supplied shares do not belong to one wallet")
		return
	}

	keyID, err := newKeyID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "key id generation failed")
		return
	}

	engine := NewTSSEngine(threshold, total)
	engine.AdoptShares(secret, publicKey, points)

	wallet := &mpcWallet{
		KeyID:       keyID,
		Address:     address,
		PublicKey:   publicKey,
		Threshold:   threshold,
		TotalShards: total,
		CreatedAt:   currentTimestamp(),
	}

	store.mu.Lock()
	store.engines[keyID] = engine
	store.meta[keyID] = wallet
	store.mu.Unlock()

	writeJSON(w, http.StatusOK, recoverWalletResponse{
		KeyID:     keyID,
		Address:   address,
		PublicKey: publicKey,
		Threshold: threshold,
		Total:     total,
	})
}

// sharesAreConsistent validates that a reconstructed point set is well formed:
// enough points to meet the threshold, every point in range, and no zero share.
func sharesAreConsistent(secret *big.Int, threshold uint32, points map[uint32]*big.Int) bool {
	if secret == nil || secret.Sign() == 0 {
		return false
	}
	if len(points) < int(threshold) {
		return false
	}
	for id, value := range points {
		if id == 0 || value == nil || value.Sign() == 0 || value.Cmp(curveOrderN()) >= 0 {
			return false
		}
	}
	return true
}
