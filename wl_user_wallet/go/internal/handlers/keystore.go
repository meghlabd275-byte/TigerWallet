package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tigerwallet/wl-user-wallet/internal/crypto"
	"github.com/tigerwallet/wl-user-wallet/internal/middleware"
	"github.com/tigerwallet/wl-user-wallet/internal/nonevm"
)

// POST /keystore/export — export a wallet's EVM private key as a standard
// Web3 Secret Storage V3 (scrypt + AES-128-CTR + keccak256 MAC) keystore JSON,
// re-encrypted with the caller-provided export password. Real scrypt.
func (s *Svc) ExportKeystore(c *gin.Context) {
	var req struct {
		WalletID       uuid.UUID `json:"wallet_id" binding:"required"`
		WalletPassword string    `json:"wallet_password" binding:"required"`
		ExportPassword string    `json:"export_password" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	seed, err := s.decryptSeed(c, req.WalletID, req.WalletPassword)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	priv, err := crypto.DeriveEVMPrivateKey(seed, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "key derivation failed"})
		return
	}
	ks, err := nonevm.ExportKeystoreV3(priv, req.ExportPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Data(http.StatusOK, "application/json", ks)
}

// POST /keystore/import — decrypt a Web3 Secret Storage V3 keystore with the
// provided password (real scrypt + MAC check; wrong password fails closed),
// re-encrypt the underlying seed and store it as a new wallet.
func (s *Svc) ImportKeystore(c *gin.Context) {
	userID := middleware.UserID(c)
	var req struct {
		Keystore       []byte   `json:"keystore" binding:"required"`
		ImportPassword string   `json:"import_password" binding:"required"`
		WalletPassword string   `json:"wallet_password" binding:"required,min=8"`
		Label          string   `json:"label"`
		ChainID        int64    `json:"chain_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.ChainID == 0 {
		req.ChainID = 1
	}
	priv, err := nonevm.ImportKeystoreV3(req.Keystore, req.ImportPassword)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "keystore decrypt failed: " + err.Error()})
		return
	}
	// We do not have the original mnemonic; persist the private key bytes as a
	// 32-byte "seed surrogate" encrypted with the wallet password. Derivation
	// from this surrogate reproduces the same address (DeriveEVMPrivateKey
	// treats the input as a raw 32-byte seed at index 0).
	seed := make([]byte, 32)
	priv.D.FillBytes(seed)
	address := crypto.AddressFromPrivateKey(priv)
	encSeed, err := crypto.EncryptSeedAtRest(seed, req.WalletPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "seed encryption failed"})
		return
	}
	w, err := s.store.CreateWallet(c.Request.Context(), userID, req.Label, address, encSeed, req.ChainID, s.cfg.WLClientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": w.ID, "label": w.Label, "address": address, "chain_id": w.ChainID})
}
