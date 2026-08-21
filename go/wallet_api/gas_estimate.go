package main

// Real EVM gas estimation via eth_estimateGas against the target chain's RPC
// endpoint. Fail-closed: if the chain has no RPC configured or the node
// returns an error, the endpoint returns 5xx — it NEVER fabricates a gas
// limit (a fabricated limit can strand or revert real transactions).

import (
	"fmt"
	"math/big"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
)

type estimateGasReq struct {
	From    string `json:"from" binding:"required"`
	To      string `json:"to"`
	Value   string `json:"value"`
	Data    string `json:"data"`
	ChainID int64  `json:"chain_id"`
}

// parseWeiAmount accepts decimal or 0x-hex wei amounts.
func parseWeiAmount(s string) (*big.Int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return big.NewInt(0), nil
	}
	n := new(big.Int)
	var ok bool
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		_, ok = n.SetString(s[2:], 16)
	} else {
		_, ok = n.SetString(s, 10)
	}
	if !ok {
		return nil, fmt.Errorf("not a valid wei amount")
	}
	return n, nil
}

func handleEstimateGas(c *gin.Context) {
	var req estimateGasReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: from address required"})
		return
	}
	if !common.IsHexAddress(req.From) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid from address"})
		return
	}
	if req.To != "" && !common.IsHexAddress(req.To) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid to address"})
		return
	}
	chainID := req.ChainID
	if chainID == 0 {
		chainID = 1
	}
	chain := chainByID(chainID)
	if chain == nil || !chain.IsEVM() {
		c.JSON(http.StatusNotFound, gin.H{"error": "unsupported or non-EVM chain"})
		return
	}
	if strings.TrimSpace(chain.RPCEndpoint) == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no RPC endpoint configured for chain"})
		return
	}

	value, err := parseWeiAmount(req.Value)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid value (must be decimal or 0x-hex wei)"})
		return
	}
	var data []byte
	if req.Data != "" {
		data, err = hexutil.Decode(req.Data)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid data (must be 0x-hex)"})
			return
		}
	}

	client, err := ethclient.DialContext(c.Request.Context(), chain.RPCEndpoint)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "RPC dial failed", "detail": err.Error()})
		return
	}
	defer client.Close()

	msg := ethereum.CallMsg{
		From:  common.HexToAddress(req.From),
		Value: value,
		Data:  data,
	}
	if req.To != "" {
		to := common.HexToAddress(req.To)
		msg.To = &to
	}
	gas, err := client.EstimateGas(c.Request.Context(), msg)
	if err != nil {
		// Real node rejection (e.g. revert, insufficient funds) — surface it.
		c.JSON(http.StatusBadGateway, gin.H{"error": "eth_estimateGas failed", "detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"gas_limit": gas,
		"chain_id":  chainID,
	})
}
