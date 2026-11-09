package handlers

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"github.com/tigerwallet/wl-user-wallet/internal/onchain"
)

// stakingAsset describes a staking protocol and the contract that receives
// the deposit.
//
// Every address below is a real, deployed mainnet contract, and every selector
// is the first four bytes of the keccak-256 hash of the signature shown beside
// it. The rate is never a constant: it is read from the protocol at request
// time (see staking_apr.go) and reported as null when unavailable.
type stakingAsset struct {
	Symbol  string `json:"symbol"`
	Name    string `json:"name"`
	ChainID int64  `json:"chain_id"`
	// DepositContract receives the staked funds.
	DepositContract string `json:"deposit_contract"`
	// DepositMethod is the function selector for the deposit call.
	DepositMethod string `json:"deposit_method"`
	// DepositSignature is the human-readable form, for wallet display.
	DepositSignature string `json:"deposit_signature"`
	// DepositNative is true when the deposit sends native ETH as msg.value.
	DepositNative bool `json:"deposit_native"`
	// WithdrawalContract releases the stake. For Lido this is the withdrawal
	// queue, not the stETH token.
	WithdrawalContract  string `json:"withdrawal_contract,omitempty"`
	WithdrawalMethod    string `json:"withdrawal_method,omitempty"`
	WithdrawalSignature string `json:"withdrawal_signature,omitempty"`
	// RateProvider names the protocol whose published rate describes the
	// return.
	RateProvider string `json:"rate_provider"`
}

// stakingAssets lists the supported protocols.
//
// Lido deposits through Lido.submit(address) at
// 0xae7ab96520DE3A18E5e111B5EaAb095312D7fE84, passing a referral address.
// Withdrawals go through the WithdrawalQueueERC721 at
// 0x889edC2eDab5f40e902b864aD4d7AdE8E412F9B1 using
// requestWithdrawals(uint256[],address).
//
// Rocket Pool deposits through RocketDepositPool.deposit() at
// 0xDD3f50F8A6CafbE9b31a427582963f465E745AF8. Exiting rETH is a burn this
// service does not build, so no withdrawal selector is advertised.
var stakingAssets = []stakingAsset{
	{
		Symbol:              "stETH",
		Name:                "Lido — staked ETH",
		ChainID:             1,
		DepositContract:     "0xae7ab96520DE3A18E5e111B5EaAb095312D7fE84",
		DepositMethod:       "0xa1903eab",
		DepositSignature:    "submit(address)",
		DepositNative:       true,
		WithdrawalContract:  "0x889edC2eDab5f40e902b864aD4d7AdE8E412F9B1",
		WithdrawalMethod:    "0xd6681042",
		WithdrawalSignature: "requestWithdrawals(uint256[],address)",
		RateProvider:        "lido",
	},
	{
		Symbol:           "rETH",
		Name:             "Rocket Pool — staked ETH",
		ChainID:          1,
		DepositContract:  "0xDD3f50F8A6CafbE9b31a427582963f465E745AF8",
		DepositMethod:    "0xd0e30db0",
		DepositSignature: "deposit()",
		DepositNative:    true,
		RateProvider:     "rocketpool",
	},
}

// GET /staking/quote — returns the supported protocols with their live rates.
// The rate is null when the protocol's API cannot be reached.
func (s *Svc) StakingQuote(c *gin.Context) {
	out := make([]gin.H, 0, len(stakingAssets))
	for _, a := range stakingAssets {
		apr, ok := resolveStakingApr(c.Request.Context(), a.Symbol)
		entry := gin.H{
			"symbol":            a.Symbol,
			"name":              a.Name,
			"chain_id":          a.ChainID,
			"deposit_contract":  a.DepositContract,
			"deposit_signature": a.DepositSignature,
			"deposit_native":    a.DepositNative,
			"apy":               aprOrNil(apr, ok),
		}
		if a.WithdrawalContract != "" {
			entry["withdrawal_contract"] = a.WithdrawalContract
			entry["withdrawal_signature"] = a.WithdrawalSignature
		}
		out = append(out, entry)
	}
	c.JSON(http.StatusOK, gin.H{"assets": out})
}

// POST /staking/stake — builds the deposit transaction for the named asset.
// The client signs and broadcasts it through /send.
func (s *Svc) StakingStake(c *gin.Context) {
	var req struct {
		Asset   string `json:"asset" binding:"required"`
		Amount  string `json:"amount" binding:"required"`
		ChainID int64  `json:"chain_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	asset := findStakingAsset(req.Asset, req.ChainID)
	if asset == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported staking asset"})
		return
	}
	amount, ok := new(big.Float).SetString(req.Amount)
	if !ok || amount.Sign() <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount"})
		return
	}

	// The deposit takes the staked amount as native value. The only argument
	// is the referral address, which is the zero address when none applies.
	calldata := selectorWithArgs(asset.DepositMethod, []common.Address{{}})
	valueWei, _ := new(big.Float).Mul(amount, big.NewFloat(1e18)).Int(nil)

	c.JSON(http.StatusOK, gin.H{
		"chain_id": asset.ChainID,
		"to":       asset.DepositContract,
		"data":     onchain.HexEncode(calldata),
		"value":    valueWei.String(),
		"amount":   req.Amount,
		"asset":    asset.Symbol,
		"action":   "send_raw_tx",
	})
}

// POST /staking/unstake — builds the withdrawal request for the named asset.
func (s *Svc) StakingUnstake(c *gin.Context) {
	var req struct {
		Asset   string `json:"asset" binding:"required"`
		Amount  string `json:"amount" binding:"required"`
		Owner   string `json:"owner" binding:"required"`
		ChainID int64  `json:"chain_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	asset := findStakingAsset(req.Asset, req.ChainID)
	if asset == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported staking asset"})
		return
	}
	if asset.WithdrawalMethod == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "this protocol has no withdrawal call this service can build; exit the position in the protocol's own interface",
		})
		return
	}
	amount, ok := new(big.Float).SetString(req.Amount)
	if !ok || amount.Sign() <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount"})
		return
	}
	if !common.IsHexAddress(req.Owner) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "a valid owner address is required to request a withdrawal"})
		return
	}
	owner := common.HexToAddress(req.Owner)
	if owner == (common.Address{}) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "a valid owner address is required to request a withdrawal"})
		return
	}

	// requestWithdrawals(uint256[] amounts, address owner) takes the stETH
	// amounts in wei and the address that owns the shares.
	amountWei, _ := new(big.Float).Mul(amount, big.NewFloat(1e18)).Int(nil)
	calldata := selectorWithUintArray(asset.WithdrawalMethod, []*big.Int{amountWei}, owner)

	c.JSON(http.StatusOK, gin.H{
		"chain_id": asset.ChainID,
		"to":       asset.WithdrawalContract,
		"data":     onchain.HexEncode(calldata),
		"value":    "0",
		"amount":   req.Amount,
		"asset":    asset.Symbol,
		"action":   "send_raw_tx",
	})
}

// POST /staking/claim — rewards in these protocols accrue to the staked
// balance, so there is no separate claim call to build.
func (s *Svc) StakingClaim(c *gin.Context) {
	var req struct {
		Asset   string `json:"asset" binding:"required"`
		ChainID int64  `json:"chain_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	asset := findStakingAsset(req.Asset, req.ChainID)
	if asset == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported staking asset"})
		return
	}
	c.JSON(http.StatusServiceUnavailable, gin.H{
		"error": "rewards accrue to the staked balance for " + asset.Symbol + "; there is no separate claim transaction",
	})
}

func findStakingAsset(name string, chainID int64) *stakingAsset {
	want := strings.ToLower(name)
	for i := range stakingAssets {
		a := &stakingAssets[i]
		if chainID != 0 && a.ChainID != chainID {
			continue
		}
		if strings.ToLower(a.Symbol) == want || strings.ToLower(a.Name) == want {
			return a
		}
	}
	return nil
}

// selectorWithArgs encodes a call whose arguments are all address values.
func selectorWithArgs(selectorHex string, args []common.Address) []byte {
	selector, err := decodeSelector(selectorHex)
	if err != nil {
		return nil
	}
	out := make([]byte, 0, 4+32*len(args))
	out = append(out, selector...)
	for _, a := range args {
		out = append(out, leftPad32(a.Bytes())...)
	}
	return out
}

// selectorWithUintArray encodes f(uint256[] values, address owner): a head
// holding the offset and the owner, then the array length and its elements.
func selectorWithUintArray(selectorHex string, values []*big.Int, owner common.Address) []byte {
	selector, err := decodeSelector(selectorHex)
	if err != nil {
		return nil
	}
	out := make([]byte, 0, 4+32*3+32*len(values))
	out = append(out, selector...)

	// Offset to the array data, measured from the start of the argument block.
	out = append(out, leftPad32(big.NewInt(64).Bytes())...)
	out = append(out, leftPad32(owner.Bytes())...)
	out = append(out, leftPad32(big.NewInt(int64(len(values))).Bytes())...)
	for _, v := range values {
		out = append(out, leftPad32(v.Bytes())...)
	}
	return out
}

func leftPad32(b []byte) []byte {
	if len(b) > 32 {
		b = b[len(b)-32:]
	}
	word := make([]byte, 32)
	copy(word[32-len(b):], b)
	return word
}

func decodeSelector(selectorHex string) ([]byte, error) {
	clean := strings.TrimPrefix(strings.TrimPrefix(selectorHex, "0x"), "0X")
	b, err := hex.DecodeString(clean)
	if err != nil {
		return nil, fmt.Errorf("invalid selector %q: %w", selectorHex, err)
	}
	if len(b) != 4 {
		return nil, fmt.Errorf("selector %q must be 4 bytes", selectorHex)
	}
	return b, nil
}
