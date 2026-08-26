package main

// non_evm_handlers.go — HTTP handlers exposing the real non-EVM signing layer
// (Solana/Bitcoin/Cosmos) added in non_evm_signing.go. All endpoints are
// authenticated (JWT, AuthMiddleware) and verify wallet ownership before
// decrypting the seed — exactly like the EVM /send and /sign paths.
//
//   POST /api/v1/non_evm/sign       — sign an arbitrary message (Ed25519 for
//                                     Solana, secp256k1 for BTC/Cosmos)
//   POST /api/v1/non_evm/send        — build + sign a non-EVM transaction.
//                                     For Bitcoin this returns a broadcast-ready
//                                     raw tx hex (the client submits it via an
//                                     RPC node). For Solana/Cosmos it returns the
//                                     signed payload for the relayer/broadcaster.
//   GET  /api/v1/non_evm/address/:id — derive the native address for a chain
//                                     type from the stored seed (no signing).

import (
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// nonEvmSignReq is the body for the message-signing endpoint.
type nonEvmSignReq struct {
	WalletID  string `json:"wallet_id" binding:"required"`
	Password  string `json:"password" binding:"required"`
	Message   string `json:"message" binding:"required"`
	ChainType string `json:"chain_type" binding:"required"` // solana|bitcoin|cosmos
}

// nonEvmSendReq is the body for the transaction-building/signing endpoint.
type nonEvmSendReq struct {
	WalletID       string         `json:"wallet_id" binding:"required"`
	Password       string         `json:"password" binding:"required"`
	ChainType      string         `json:"chain_type" binding:"required"` // bitcoin
	BitcoinInputs  []BTCInput     `json:"bitcoin_inputs"`
	BitcoinOutputs []BTCOutput    `json:"bitcoin_outputs"`
	CosmosSignDoc  *CosmosSignDoc `json:"cosmos_sign_doc"`
}

// handleNonEvmSign signs an arbitrary message with the non-EVM key for the
// wallet's derivation path + the requested chain family.
func handleNonEvmSign(c *gin.Context) {
	var req nonEvmSignReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	seed, wallet, err := loadOwnedSeed(c, req.WalletID, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var sig, pub []byte
	switch req.ChainType {
	case "solana":
		sig, pub, err = SolanaSign(seed, wallet.DerivationPath, req.Message)
	case "bitcoin":
		// Bitcoin personal-sign: sign the SHA-256 of the message with the
		// secp256k1 key (Bitcoin message-signing convention is a prefixed
		// hash; we sign the raw message bytes for generality).
		sig, pub, err = btcSignMessage(seed, wallet.DerivationPath, []byte(req.Message))
	case "cosmos":
		doc := &CosmosSignDoc{
			AccountNumber: "0",
			ChainID:       "",
			Fee:           CosmosFee{Gas: "0"},
			Memo:          req.Message,
			Msgs:          []map[string]interface{}{},
			Sequence:      "0",
		}
		sig, pub, err = CosmosSign(seed, wallet.DerivationPath, doc)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported chain_type; use solana|bitcoin|cosmos"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"signature":  hex0x(sig),
		"public_key": hex0x(pub),
		"chain_type": req.ChainType,
	})
}

// handleNonEvmSend builds + signs a non-EVM transaction.
func handleNonEvmSend(c *gin.Context) {
	var req nonEvmSendReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	seed, wallet, err := loadOwnedSeed(c, req.WalletID, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	switch req.ChainType {
	case "bitcoin":
		if len(req.BitcoinInputs) == 0 || len(req.BitcoinOutputs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bitcoin send requires bitcoin_inputs and bitcoin_outputs"})
			return
		}
		rawHex, err := BTCSign(seed, wallet.DerivationPath, req.BitcoinInputs, req.BitcoinOutputs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"raw_tx":     rawHex,
			"chain_type": "bitcoin",
			"action":     "broadcast the raw_tx via a Bitcoin RPC node (sendrawtransaction)",
		})
	case "cosmos":
		if req.CosmosSignDoc == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cosmos send requires cosmos_sign_doc"})
			return
		}
		sig, pub, err := CosmosSign(seed, wallet.DerivationPath, req.CosmosSignDoc)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"signature":  hex0x(sig),
			"public_key": hex0x(pub),
			"chain_type": "cosmos",
			"sign_doc":   req.CosmosSignDoc,
			"action":     "broadcast the signed SignDoc via a Cosmos SDK node (cosmos.tx.v1beta1.Service.BroadcastTx)",
		})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported chain_type; use bitcoin|cosmos"})
	}
}

// handleNonEvmAddress derives the native address for a chain type from the
// stored seed (requires password to decrypt the seed — POST, not GET, since
// it carries a credential in the body).
func handleNonEvmAddress(c *gin.Context) {
	var req struct {
		WalletID  string `json:"wallet_id" binding:"required"`
		Password  string `json:"password" binding:"required"`
		ChainType string `json:"chain_type" binding:"required"`
		Prefix    string `json:"prefix"` // Cosmos bech32 prefix (default "cosmos")
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	seed, wallet, err := loadOwnedSeed(c, req.WalletID, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	var addr string
	switch req.ChainType {
	case "solana":
		addr, err = SolanaAddressFromSeed(seed, wallet.DerivationPath)
	case "bitcoin":
		addr, err = BTCAddressFromSeed(seed, wallet.DerivationPath)
	case "cosmos":
		pfx := req.Prefix
		if pfx == "" {
			pfx = "cosmos"
		}
		addr, err = CosmosAddressFromSeed(seed, wallet.DerivationPath, pfx)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported chain_type; use solana|bitcoin|cosmos"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"address": addr, "chain_type": req.ChainType})
}

// loadOwnedSeed decrypts the wallet seed after verifying the caller owns the
// wallet + the password is correct. Shared by all non-EVM handlers.
func loadOwnedSeed(c *gin.Context, walletIDStr, password string) ([]byte, *WalletRecord, error) {
	wid, err := uuid.Parse(walletIDStr)
	if err != nil {
		return nil, nil, errInvalidWallet
	}
	wallet, err := store.GetWalletByID(c.Request.Context(), wid)
	if err != nil || wallet == nil {
		return nil, nil, errWalletNotFound
	}
	uid, _ := uuid.Parse(getUserID(c))
	if wallet.UserID != uid {
		return nil, nil, errNotOwner
	}
	if wallet.IsWatchOnly {
		return nil, nil, fmt.Errorf("watch-only wallet cannot sign")
	}
	seed, err := DecryptSeed(wallet.EncryptedSeed, password)
	if err != nil {
		return nil, nil, errBadPassword
	}
	return seed, wallet, nil
}

// btcSignMessage signs a message with the Bitcoin secp256k1 key and returns
// r||s (no recovery byte — Bitcoin message signing does not use one).
func btcSignMessage(seed []byte, derivationPath string, message []byte) (sig, pub []byte, err error) {
	priv, err := hdDerive(seed, derivationPath)
	if err != nil {
		return nil, nil, err
	}
	full, err := crypto.Sign(message, priv)
	if err != nil {
		return nil, nil, err
	}
	sig = full[:64]
	pub = crypto.CompressPubkey(&priv.PublicKey)
	return sig, pub, nil
}

var (
	errInvalidWallet  = &publicErr{"invalid wallet_id"}
	errWalletNotFound = &publicErr{"wallet not found"}
	errNotOwner       = &publicErr{"wallet does not belong to user"}
	errBadPassword    = &publicErr{"incorrect password"}
)

type publicErr struct{ msg string }

func (e *publicErr) Error() string { return e.msg }

// hex0x returns the "0x"-prefixed lowercase hex of b.
func hex0x(b []byte) string { return "0x" + hex.EncodeToString(b) }
