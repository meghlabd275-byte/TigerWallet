package main

// custody_relay.go — service-to-service transaction relay for MPC signers.
//
// An MPC custody signer (institutional_custody) holds distributed key shares,
// not private keys. It can ask a TSS backend for a secp256k1 signature over a
// 32-byte digest, but it cannot sign a whole transaction itself. This file
// gives it the two calls it needs, without moving any transaction-construction
// logic out of the wallet backend:
//
//	POST /api/v1/service/tx/prepare   -> fetch nonce/gas, build the unsigned
//	                                     transaction, return its signing hash
//	POST /api/v1/service/tx/submit    -> rebuild the identical transaction,
//	                                     inject the signature, verify the
//	                                     recovered sender, broadcast
//
// Both require the same bearer token. Neither signs anything: the only
// signature that is ever accepted comes from the caller.
//
// Why prepare/submit instead of letting the caller build the transaction:
// the signing hash commits to the nonce, gas price and gas limit, so the
// caller must sign a hash the backend computed. Rebuilding on submit (rather
// than trusting a caller-supplied blob) means a tampered value or recipient
// changes the hash and the injected signature then recovers a different
// address, which the sender check rejects. The design is self-verifying.
//
// There is also a lower-level POST /api/v1/service/broadcast that relays
// already-signed raw bytes, for any signer that can produce them.

import (
	"context"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/gin-gonic/gin"
)

// ---------------------------------------------------------------------------
// Service authentication
// ---------------------------------------------------------------------------

// serviceBroadcastToken returns the configured relay token. An empty token
// disables the relay entirely rather than allowing unauthenticated relay.
func serviceBroadcastToken() string {
	return strings.TrimSpace(os.Getenv("SERVICE_BROADCAST_TOKEN"))
}

// requireServiceToken enforces the shared relay token. It writes the error
// response itself and reports whether the caller may continue.
func requireServiceToken(c *gin.Context) bool {
	token := serviceBroadcastToken()
	if token == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "custody relay is not enabled (set SERVICE_BROADCAST_TOKEN)",
		})
		return false
	}
	presented := strings.TrimSpace(strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "))
	if presented == "" || presented != token {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid service token"})
		return false
	}
	return true
}

// ---------------------------------------------------------------------------
// Transaction parameter model
// ---------------------------------------------------------------------------

type txParams struct {
	ChainID int64  `json:"chain_id" binding:"required"`
	From    string `json:"from" binding:"required"`
	To      string `json:"to" binding:"required"`
	// ValueWei is the exact transfer amount in wei as a base-10 string.
	// Wei is used rather than ether so no decimal parsing can round a
	// custody transfer.
	ValueWei string `json:"value_wei"`
	Data     string `json:"data"`
	GasLimit uint64 `json:"gas_limit"`
	// TxType selects the fee model: "legacy" (default) or "eip1559".
	TxType         string `json:"tx_type"`
	GasPriceWei    string `json:"gas_price_wei"`
	MaxFeeWei      string `json:"max_fee_wei"`
	MaxPriorityWei string `json:"max_priority_fee_wei"`
}

// txBundle is the resolved, fully-specified transaction handed back by prepare
// and echoed to submit.
type txBundle struct {
	ChainID        int64  `json:"chain_id"`
	From           string `json:"from"`
	To             string `json:"to"`
	ValueWei       string `json:"value_wei"`
	Data           string `json:"data"`
	GasLimit       uint64 `json:"gas_limit"`
	TxType         string `json:"tx_type"`
	Nonce          uint64 `json:"nonce"`
	GasPriceWei    string `json:"gas_price_wei,omitempty"`
	MaxFeeWei      string `json:"max_fee_wei,omitempty"`
	MaxPriorityWei string `json:"max_priority_fee_wei,omitempty"`
	SigningHash    string `json:"signing_hash"`
}

// parseDecimalBig parses a base-10 integer string, treating "" as zero.
func parseDecimalBig(s string) (*big.Int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return big.NewInt(0), nil
	}
	n, ok := new(big.Int).SetString(s, 10)
	if !ok || n.Sign() < 0 {
		return nil, errInvalidAmount
	}
	return n, nil
}

// buildUnsignedTx constructs the transaction described by b. It is used by both
// prepare and submit so the two phases cannot drift apart.
func buildUnsignedTx(b txBundle, to common.Address, value *big.Int, data []byte,
	gasPrice, maxFee, maxPrio *big.Int) (*types.Transaction, types.Signer, error) {

	chainID := big.NewInt(b.ChainID)
	signer := types.LatestSignerForChainID(chainID)

	if b.TxType == "eip1559" {
		if maxFee == nil || maxPrio == nil {
			return nil, nil, errMissingFee
		}
		return types.NewTx(&types.DynamicFeeTx{
			ChainID:   chainID,
			Nonce:     b.Nonce,
			To:        &to,
			Value:     value,
			Gas:       b.GasLimit,
			GasFeeCap: maxFee,
			GasTipCap: maxPrio,
			Data:      data,
		}), signer, nil
	}

	if gasPrice == nil {
		return nil, nil, errMissingFee
	}
	return types.NewTx(&types.LegacyTx{
		Nonce:    b.Nonce,
		To:       &to,
		Value:    value,
		Gas:      b.GasLimit,
		GasPrice: gasPrice,
		Data:     data,
	}), signer, nil
}

// normalizeSignature converts an [R||S||V] signature whose V is in the
// ethereum 27/28 convention into the 0/1 recovery id that go-ethereum's
// SignatureValues expects. Signatures already using 0/1 are returned as-is.
func normalizeSignature(sig []byte) []byte {
	if len(sig) == 65 && sig[64] >= 27 {
		out := make([]byte, 65)
		copy(out, sig)
		out[64] -= 27
		return out
	}
	return sig
}

// resolveTxBundle validates the caller's parameters and fills in the fields the
// backend is responsible for (nonce, gas price, gas limit).
func resolveTxBundle(ctx context.Context, p txParams) (*txBundle, common.Address, *big.Int, []byte, error) {
	chain := evmChainByChainID(p.ChainID)
	if chain == nil || chain.RPCEndpoint == "" {
		return nil, common.Address{}, nil, nil, errUnsupportedChain
	}
	from, err := hexAddress(p.From)
	if err != nil {
		return nil, common.Address{}, nil, nil, errBadFrom
	}
	to, err := hexAddress(p.To)
	if err != nil {
		return nil, common.Address{}, nil, nil, errBadTo
	}
	value, err := parseDecimalBig(p.ValueWei)
	if err != nil {
		return nil, common.Address{}, nil, nil, err
	}
	var data []byte
	if d := strings.TrimSpace(p.Data); d != "" && d != "0x" {
		decoded, err := hexDecode(d)
		if err != nil {
			return nil, common.Address{}, nil, nil, errBadData
		}
		data = decoded
	}

	b := txBundle{
		ChainID:  p.ChainID,
		From:     from.Hex(),
		To:       to.Hex(),
		ValueWei: value.String(),
		Data:     "0x" + common.Bytes2Hex(data),
		GasLimit: p.GasLimit,
		TxType:   p.TxType,
	}
	if b.TxType == "" {
		b.TxType = "legacy"
	}

	nonce, err := FetchTransactionCount(ctx, chain.RPCEndpoint, from)
	if err != nil {
		return nil, common.Address{}, nil, nil, err
	}
	b.Nonce = nonce

	// Caller-supplied fee values win so an operator can pin a fee;
	// otherwise the chain suggestion is used.
	gasPrice, maxFee, maxPrio, err := FetchGasPrice(ctx, chain.RPCEndpoint)
	if err != nil {
		return nil, common.Address{}, nil, nil, err
	}
	if b.TxType == "eip1559" {
		if p.MaxFeeWei != "" {
			if v, err := parseDecimalBig(p.MaxFeeWei); err == nil {
				maxFee = v
			}
		}
		if p.MaxPriorityWei != "" {
			if v, err := parseDecimalBig(p.MaxPriorityWei); err == nil {
				maxPrio = v
			}
		}
		if maxFee == nil || maxPrio == nil {
			return nil, common.Address{}, nil, nil, errMissingFee
		}
		b.MaxFeeWei = maxFee.String()
		b.MaxPriorityWei = maxPrio.String()
	} else {
		if p.GasPriceWei != "" {
			if v, err := parseDecimalBig(p.GasPriceWei); err == nil && v.Sign() > 0 {
				gasPrice = v
			}
		}
		if gasPrice == nil || gasPrice.Sign() <= 0 {
			return nil, common.Address{}, nil, nil, errMissingFee
		}
		b.GasPriceWei = gasPrice.String()
	}

	if b.GasLimit == 0 {
		b.GasLimit = 21000
		if len(data) > 0 {
			b.GasLimit = 150000
		}
	}
	return &b, to, value, data, nil
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

func handleServiceTxPrepare(c *gin.Context) {
	if !requireServiceToken(c) {
		return
	}
	var p txParams
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	b, to, value, data, err := resolveTxBundle(ctx, p)
	if err != nil {
		writeRelayError(c, err)
		return
	}
	var gasPrice, maxFee, maxPrio *big.Int
	if b.TxType == "eip1559" {
		maxFee, _ = new(big.Int).SetString(b.MaxFeeWei, 10)
		maxPrio, _ = new(big.Int).SetString(b.MaxPriorityWei, 10)
	} else {
		gasPrice, _ = new(big.Int).SetString(b.GasPriceWei, 10)
	}

	tx, signer, err := buildUnsignedTx(*b, to, value, data, gasPrice, maxFee, maxPrio)
	if err != nil {
		writeRelayError(c, err)
		return
	}
	b.SigningHash = signer.Hash(tx).Hex()

	c.JSON(http.StatusOK, b)
}

// assembleSignedTx rebuilds the transaction described by b, checks that the
// caller's signing hash matches that rebuild, injects the signature, and
// recovers the sender. It performs no network I/O, so the whole
// tamper-detection path can be tested directly.
//
// It returns the signed transaction and the recovered sender. Every failure
// mode is a rejection: a hash mismatch means the fields were altered after
// signing, an unparseable signature means the signer is broken, and a
// recovered sender that differs from b.From means the signature belongs to
// another key.
func assembleSignedTx(b txBundle, signatureHex string) (*types.Transaction, common.Address, error) {
	to, err := hexAddress(b.To)
	if err != nil {
		return nil, common.Address{}, errBadTo
	}
	from, err := hexAddress(b.From)
	if err != nil {
		return nil, common.Address{}, errBadFrom
	}
	value, err := parseDecimalBig(b.ValueWei)
	if err != nil {
		return nil, common.Address{}, err
	}
	var data []byte
	if d := strings.TrimSpace(b.Data); d != "" && d != "0x" {
		data, err = hexDecode(d)
		if err != nil {
			return nil, common.Address{}, errBadData
		}
	}
	var gasPrice, maxFee, maxPrio *big.Int
	if b.TxType == "eip1559" {
		if maxFee, err = parseDecimalBig(b.MaxFeeWei); err != nil {
			return nil, common.Address{}, err
		}
		if maxPrio, err = parseDecimalBig(b.MaxPriorityWei); err != nil {
			return nil, common.Address{}, err
		}
	} else {
		if gasPrice, err = parseDecimalBig(b.GasPriceWei); err != nil {
			return nil, common.Address{}, err
		}
	}

	tx, signer, err := buildUnsignedTx(b, to, value, data, gasPrice, maxFee, maxPrio)
	if err != nil {
		return nil, common.Address{}, err
	}

	if !strings.EqualFold(signer.Hash(tx).Hex(), b.SigningHash) {
		return nil, common.Address{}, errHashMismatch
	}

	sigBytes := common.FromHex(strings.TrimSpace(signatureHex))
	if len(sigBytes) != 65 {
		return nil, common.Address{}, errBadSignature
	}
	sigBytes = normalizeSignature(sigBytes)

	signedTx, err := tx.WithSignature(signer, sigBytes)
	if err != nil {
		return nil, common.Address{}, errBadSignature
	}
	sender, err := types.Sender(signer, signedTx)
	if err != nil {
		return nil, common.Address{}, errBadSignature
	}
	if sender != from {
		return nil, common.Address{}, errSenderMismatch
	}
	return signedTx, sender, nil
}

type txSubmitReq struct {
	txBundle
	// Signature is the 65-byte [R||S||V] hex signature the MPC backend
	// produced over SigningHash. V may be 0/1 or 27/28.
	Signature string `json:"signature" binding:"required"`
}

func handleServiceTxSubmit(c *gin.Context) {
	if !requireServiceToken(c) {
		return
	}
	var req txSubmitReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	b := req.txBundle
	if b.SigningHash == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "signing_hash is required (call /tx/prepare first)"})
		return
	}

	chain := evmChainByChainID(b.ChainID)
	if chain == nil || chain.RPCEndpoint == "" {
		writeRelayError(c, errUnsupportedChain)
		return
	}

	signedTx, sender, err := assembleSignedTx(b, req.Signature)
	if err != nil {
		writeRelayError(c, err)
		return
	}

	raw, err := signedTx.MarshalBinary()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "marshal signed tx: " + err.Error()})
		return
	}
	rawHex := "0x" + common.Bytes2Hex(raw)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	txHash, err := BroadcastTransaction(ctx, chain.RPCEndpoint, rawHex, big.NewInt(b.ChainID))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "broadcast failed: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"tx_hash":  txHash,
		"chain_id": b.ChainID,
		"from":     sender.Hex(),
		"nonce":    b.Nonce,
		"raw_tx":   rawHex,
	})
}

// ---------------------------------------------------------------------------
// Low-level raw relay
// ---------------------------------------------------------------------------

type rawBroadcastReq struct {
	RawTx        string `json:"raw_tx" binding:"required"`
	ChainID      int64  `json:"chain_id" binding:"required"`
	ExpectedFrom string `json:"expected_from"`
}

func handleServiceBroadcast(c *gin.Context) {
	if !requireServiceToken(c) {
		return
	}
	var req rawBroadcastReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	raw := strings.TrimSpace(req.RawTx)
	if !strings.HasPrefix(raw, "0x") || len(raw) < 4 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "raw_tx must be 0x-prefixed hex"})
		return
	}

	chain := evmChainByChainID(req.ChainID)
	if chain == nil || chain.RPCEndpoint == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported chain"})
		return
	}

	// Decode so we can prove the payload is a real signed transaction and
	// recover its sender. Bytes that do not decode are never relayed.
	var tx types.Transaction
	if err := tx.UnmarshalBinary(common.FromHex(raw)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "raw_tx is not a decodable transaction: " + err.Error()})
		return
	}
	chainIDBig := big.NewInt(chain.ID)
	signer := types.LatestSignerForChainID(chainIDBig)
	sender, err := types.Sender(signer, &tx)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "raw_tx signature is invalid: " + err.Error()})
		return
	}
	if req.ExpectedFrom != "" {
		want := common.HexToAddress(req.ExpectedFrom)
		if sender != want {
			c.JSON(http.StatusForbidden, gin.H{
				"error":  "raw_tx sender does not match expected_from",
				"sender": sender.Hex(),
				"want":   want.Hex(),
			})
			return
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	txHash, err := BroadcastTransaction(ctx, chain.RPCEndpoint, raw, chainIDBig)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "broadcast failed: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"tx_hash":  txHash,
		"chain_id": req.ChainID,
		"from":     sender.Hex(),
	})
}

// ---------------------------------------------------------------------------
// Errors
// ---------------------------------------------------------------------------

var (
	errUnsupportedChain = relayError{"unsupported chain"}
	errBadFrom          = relayError{"from must be a valid 0x address"}
	errBadTo            = relayError{"to must be a valid 0x address"}
	errBadData          = relayError{"data must be hex"}
	errInvalidAmount    = relayError{"value_wei must be a base-10 integer string"}
	errMissingFee       = relayError{"could not determine a gas price for this chain"}
	errHashMismatch     = relayError{"signing_hash does not match the supplied transaction fields"}
	errBadSignature     = relayError{"signature is not a valid 65-byte secp256k1 signature"}
	errSenderMismatch   = relayError{"signature does not belong to the declared sender"}
)

type relayError struct{ msg string }

func (e relayError) Error() string { return e.msg }

// writeRelayError maps an internal error onto an HTTP status. Chain and
// parameter problems are the caller's fault (400); everything else is
// upstream (502).
func writeRelayError(c *gin.Context, err error) {
	switch err {
	case errUnsupportedChain, errBadFrom, errBadTo, errBadData, errInvalidAmount, errBadSignature:
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errHashMismatch:
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errSenderMismatch:
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
	}
}
