package main

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// nftTransferReq is the request body for transferring an ERC-721 token. The
// transfer is performed as a real on-chain safeTransferFrom(from,to,tokenId)
// contract call, signed + broadcast by the wallet backend (no fabricated tx).
type nftTransferReq struct {
	WalletID    string `json:"wallet_id" binding:"required"`
	Password    string `json:"password"`     // optional when unlock_token supplied
	UnlockToken string `json:"unlock_token"` // optional; passwordless NFT transfer
	Contract    string `json:"contract" binding:"required"`
	ToAddress   string `json:"to" binding:"required"`
	TokenID     string `json:"token_id" binding:"required"`
	ChainID     int64  `json:"chain_id"`
}

// safeTransferFrom(address from, address to, uint256 tokenId) selector = 0x42842e0e
const safeTransferFromSelector = "42842e0e"

// handleNFTTransfer builds an ERC-721 safeTransferFrom calldata and delegates
// to the shared executeSend signing/broadcast path (value=0, to=contract).
func handleNFTTransfer(c *gin.Context) {
	if !enforceFeature(c, FeatureNFTTransfer) {
		return
	}
	var req nftTransferReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.ChainID == 0 {
		req.ChainID = 1
	}

	contractAddr, err := hexAddress(req.Contract)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid contract address: " + err.Error()})
		return
	}
	toAddr, err := hexAddress(req.ToAddress)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid recipient address: " + err.Error()})
		return
	}
	tokenIDHex, err := tokenIDToHex32(req.TokenID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	wid, werr := uuid.Parse(req.WalletID)
	if werr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid wallet_id"})
		return
	}
	wallet, werr := store.GetWalletByID(c.Request.Context(), wid)
	if werr != nil || wallet == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}
	fromAddr := common.HexToAddress(wallet.Address)

	// selector + from(address) + to(address) + tokenId(uint256), each 32 bytes
	calldata := "0x" + safeTransferFromSelector +
		pad32(fromAddr.Bytes()) +
		pad32(toAddr.Bytes()) +
		tokenIDHex

	executeSend(c, sendTxReq{
		WalletID:    req.WalletID,
		Password:    req.Password,
		UnlockToken: req.UnlockToken,
		ToAddress:   contractAddr.Hex(),
		Value:       "0",
		GasLimit:    120000, // ERC-721 safeTransferFrom is heavier than a plain transfer
		Data:        calldata,
		ChainID:     req.ChainID,
	})
}

// pad32 left-pads a byte slice to 32 bytes and returns hex (no 0x prefix).
func pad32(b []byte) string {
	out := make([]byte, 64)
	if len(b) > 32 {
		b = b[len(b)-32:]
	}
	copy(out[32-len(b):], b)
	return hex.EncodeToString(out)
}

// tokenIDToHex32 converts a decimal token id string to a 32-byte hex string.
func tokenIDToHex32(dec string) (string, error) {
	dec = strings.TrimSpace(dec)
	bi, ok := new(big.Int).SetString(dec, 10)
	if !ok {
		return "", fmt.Errorf("invalid token_id: %s", dec)
	}
	if bi.Sign() < 0 {
		return "", fmt.Errorf("token_id must be non-negative")
	}
	return pad32(bi.Bytes()), nil
}
