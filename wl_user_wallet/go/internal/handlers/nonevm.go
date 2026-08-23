package handlers

import (
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tigerwallet/wl-user-wallet/internal/crypto"
	"github.com/tigerwallet/wl-user-wallet/internal/middleware"
	"github.com/tigerwallet/wl-user-wallet/internal/nonevm"
)

// POST /non_evm/sign — REAL non-EVM signing. Supported chains:
//   - solana: SLIP-0010 Ed25519 from BIP-44 m/44'/501'/0'/0'  → ed25519 sign
//   - bitcoin: BIP-32 secp256k1 from BIP-44 m/44'/0'/0'/0/0    → secp256k1 sign
//   - cosmos:  BIP-32 secp256k1 + Amino SignDoc                 → secp256k1 sign
//
// The seed is decrypted from the user's wallet with their password (fail-closed).
func (s *Svc) NonEvmSign(c *gin.Context) {
	var req struct {
		WalletID       uuid.UUID             `json:"wallet_id" binding:"required"`
		Password       string                `json:"password" binding:"required"`
		Chain          string                `json:"chain" binding:"required"` // solana|bitcoin|cosmos
		DerivationPath string                `json:"derivation_path"`
		Message        string                `json:"message"`
		SignDoc        *nonevm.CosmosSignDoc `json:"sign_doc"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// personal_sign is non-value => Auto mode (license-alive = approved).
	if _, ok := s.requireApproval(c, "personal_sign", "", "", "", ""); !ok {
		return
	}
	seed, err := s.decryptSeed(c, req.WalletID, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	chain := strings.ToLower(req.Chain)
	path := req.DerivationPath
	if path == "" {
		path = defaultPathForChain(chain)
	}
	switch chain {
	case "solana":
		sig, pub, err := nonevm.SolanaSign(seed, path, req.Message)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"signature": hex.EncodeToString(sig), "public_key": base58Pub(pub)})
	case "bitcoin":
		sig, pub, err := nonevm.BTCSignMessage(seed, path, []byte(req.Message))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"signature": hex.EncodeToString(sig), "public_key": hex.EncodeToString(pub)})
	case "cosmos":
		if req.SignDoc == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cosmos requires sign_doc"})
			return
		}
		sig, pub, err := nonevm.CosmosSign(seed, path, req.SignDoc)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"signature": hex.EncodeToString(sig), "public_key": hex.EncodeToString(pub)})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported chain " + req.Chain})
	}
}

// POST /non_evm/send — REAL non-EVM tx build/sign. Bitcoin builds a signed
// legacy P2PKH tx; Cosmos signs the Amino SignDoc. Fail-closed on bad input.
func (s *Svc) NonEvmSend(c *gin.Context) {
	var req struct {
		WalletID       uuid.UUID             `json:"wallet_id" binding:"required"`
		Password       string                `json:"password" binding:"required"`
		Chain          string                `json:"chain" binding:"required"`
		DerivationPath string                `json:"derivation_path"`
		Inputs         []nonevm.BTCInput     `json:"inputs"`
		Outputs        []nonevm.BTCOutput    `json:"outputs"`
		SignDoc        *nonevm.CosmosSignDoc `json:"sign_doc"`
		WithdrawalID   string                `json:"withdrawal_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Classify the non-EVM transfer. For BTC use the first output address; for
	// Cosmos use the sign_doc's first message's to_address if present. These
	// are the treasury-address checks. User transfers are Auto-approved.
	toAddr := ""
	amtStr := ""
	if len(req.Outputs) > 0 {
		toAddr = req.Outputs[0].Address
		amtStr = strconv.FormatInt(req.Outputs[0].AmountSat, 10)
	}
	wid, ok := s.requireApproval(c, "transfer", toAddr, "", amtStr, req.WithdrawalID)
	if !ok {
		return
	}
	seed, err := s.decryptSeed(c, req.WalletID, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	chain := strings.ToLower(req.Chain)
	path := req.DerivationPath
	if path == "" {
		path = defaultPathForChain(chain)
	}
	switch chain {
	case "bitcoin":
		if len(req.Inputs) == 0 || len(req.Outputs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bitcoin send requires inputs and outputs"})
			return
		}
		rawHex, err := nonevm.BTCSign(seed, path, req.Inputs, req.Outputs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		// Mark the two-party withdrawal as executed once the backend has
		// produced the signed raw tx (the fund movement is now authorized).
		if wid != uuid.Nil {
			if g := middleware.GetTwoPartyGate(); g != nil {
				_ = g.MarkWithdrawalExecuted(c.Request.Context(), wid, rawHex)
			}
		}
		c.JSON(http.StatusOK, gin.H{"raw_tx": rawHex, "chain": "bitcoin", "action": "broadcast_btc"})
	case "cosmos":
		if req.SignDoc == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cosmos send requires sign_doc"})
			return
		}
		sig, pub, err := nonevm.CosmosSign(seed, path, req.SignDoc)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		sigHex := hex.EncodeToString(sig)
		if wid != uuid.Nil {
			if g := middleware.GetTwoPartyGate(); g != nil {
				_ = g.MarkWithdrawalExecuted(c.Request.Context(), wid, sigHex)
			}
		}
		c.JSON(http.StatusOK, gin.H{
			"signature":  sigHex,
			"public_key": hex.EncodeToString(pub),
			"chain":      "cosmos",
			"action":     "broadcast_cosmos",
		})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported chain " + req.Chain})
	}
}

// POST /non_evm/address — derive a native non-EVM address from the seed.
// Solana base58, Bitcoin base58check P2PKH, Cosmos bech32.
func (s *Svc) NonEvmAddress(c *gin.Context) {
	var req struct {
		WalletID       uuid.UUID `json:"wallet_id" binding:"required"`
		Password       string    `json:"password" binding:"required"`
		Chain          string    `json:"chain" binding:"required"`
		DerivationPath string    `json:"derivation_path"`
		Prefix         string    `json:"prefix"` // cosmos bech32 hrp
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	seed, err := s.decryptSeed(c, req.WalletID, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	chain := strings.ToLower(req.Chain)
	path := req.DerivationPath
	if path == "" {
		path = defaultPathForChain(chain)
	}
	switch chain {
	case "solana":
		addr, err := nonevm.SolanaAddressFromSeed(seed, path)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"address": addr, "chain": "solana"})
	case "bitcoin":
		addr, err := nonevm.BTCAddressFromSeed(seed, path)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"address": addr, "chain": "bitcoin"})
	case "cosmos":
		prefix := req.Prefix
		if prefix == "" {
			prefix = "cosmos"
		}
		addr, err := nonevm.CosmosAddressFromSeed(seed, path, prefix)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"address": addr, "chain": "cosmos", "prefix": prefix})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported chain " + req.Chain})
	}
}

// decryptSeed loads the wallet, checks ownership, decrypts the seed with the
// password (fail-closed on bad password).
func (s *Svc) decryptSeed(c *gin.Context, walletID uuid.UUID, password string) ([]byte, error) {
	w, err := s.store.GetWallet(c.Request.Context(), walletID)
	if err != nil {
		return nil, err
	}
	if w.UserID != middleware.UserID(c) {
		return nil, errNotYourWallet
	}
	return crypto.DecryptSeedAtRest(w.EncryptedSeed, password)
}

func defaultPathForChain(chain string) string {
	switch chain {
	case "solana":
		return "m/44'/501'/0'/0'"
	case "bitcoin":
		return "m/44'/0'/0'/0/0"
	case "cosmos":
		return "m/44'/118'/0'/0/0"
	}
	return "m/44'/0'/0'/0/0"
}

// base58Pub is not used (Solana returns base58 pubkey via address helper) but
// kept as a stable hook to render a pubkey for the /sign response.
func base58Pub(pub []byte) string {
	if len(pub) == 0 {
		return ""
	}
	out := make([]byte, 0, len(pub)*2)
	for _, b := range pub {
		out = append(out, hexByte(b>>4), hexByte(b&0x0f))
	}
	return string(out)
}

func hexByte(n byte) byte {
	if n < 10 {
		return '0' + n
	}
	return 'a' + n - 10
}
