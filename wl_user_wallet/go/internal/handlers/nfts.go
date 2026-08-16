package handlers

import (
	"context"
	"math/big"
	"net/http"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"github.com/tigerwallet/wl-user-wallet/internal/onchain"
)

// GET /nfts?address=&chain_id= — real ERC-721 holdings via Etherscan-compatible
// token-nft-tx API + HTTP metadata fetch with ipfs:// resolution.
// Fail-closed 503 if no explorer configured.
func (s *Svc) GetNFTs(c *gin.Context) {
	address := c.Query("address")
	chainID, _ := strconv.ParseInt(c.Query("chain_id"), 10, 64)
	if address == "" || chainID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "address and chain_id required"})
		return
	}
	explorer := explorerForChain(chainID)
	if explorer == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no explorer API configured for chain"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 25*time.Second)
	defer cancel()
	assets, err := onchain.FetchNFTAssets(ctx, explorer, s.cfg.EtherscanAPIKey, address)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "nft fetch failed: " + err.Error()})
		return
	}
	// Enrich with metadata fetched over HTTP where a tokenURI is discoverable.
	// We resolve tokenURI via eth_call tokenURI(uint256) for each asset whose
	// contract supports ERC-721.
	rpc := rpcForChain(chainID)
	for i, a := range assets {
		if rpc == "" {
			break
		}
		tokenID, ok := new(big.Int).SetString(a.TokenID, 10)
		if !ok {
			continue
		}
		uri, err := fetchTokenURI(ctx, rpc, common.HexToAddress(a.Contract), tokenID)
		if err != nil || uri == "" {
			continue
		}
		name, desc, img := onchain.FetchNFTMetadata(ctx, uri)
		if name != "" {
			assets[i].Name = name
		}
		assets[i].Description = desc
		assets[i].ImageURL = img
	}
	c.JSON(http.StatusOK, gin.H{"chain_id": chainID, "address": address, "nfts": assets})
}

// fetchTokenURI calls ERC-721 tokenURI(uint256) via eth_call.
func fetchTokenURI(ctx context.Context, endpoint string, contract common.Address, tokenID *big.Int) (string, error) {
	data := make([]byte, 4+32)
	copy(data[:4], []byte{0xc8, 0x7b, 0x2d, 0xd0}) // tokenURI(uint256)
	copy(data[4:], common.LeftPadBytes(tokenID.Bytes(), 32))
	res, err := onchain.EthCall(ctx, endpoint, contract, data)
	if err != nil {
		return "", err
	}
	return decodeStringResult(res), nil
}

func decodeStringResult(res []byte) string {
	if len(res) < 64 {
		return ""
	}
	// Dynamic string: offset, length, data
	offset := new(big.Int).SetBytes(res[:32]).Int64()
	if offset == 0 || offset > 31 {
		// Possibly bytes-encoded short string in first slot
		return stringTrimRight(res[:32])
	}
	if len(res) >= 64 {
		length := new(big.Int).SetBytes(res[32:64]).Int64()
		if int(length) > 0 && int(length) <= len(res)-64 {
			return stringTrimRight(res[64 : 64+int(length)])
		}
	}
	return ""
}

func stringTrimRight(b []byte) string {
	out := make([]byte, len(b))
	copy(out, b)
	for len(out) > 0 && out[len(out)-1] == 0 {
		out = out[:len(out)-1]
	}
	return string(out)
}
