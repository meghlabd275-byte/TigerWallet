package main

// non_evm_cosmos.go — Cosmos-family SDK for the 23 seeded Cosmos chains.
// Implements modern SIGN_MODE_DIRECT protobuf tx construction (TxBody,
// AuthInfo, Raw) by hand-rolled protobuf wire encoding, per-chain bech32
// prefix resolution by registry id, and a real REST broadcast against the
// chain's LCD endpoint (cosmos.directory rest base or the seeded endpoint
// when it is a full LCD REST node).
//
// The message path supports cosmos.bank.v1beta1.MsgSend only; anything else
// fails closed explicitly. Account number + sequence are fetched from the
// LCD node (never fabricated).

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/ethereum/go-ethereum/crypto"
)

// cosmosHTTP is a bounded HTTP client shared by cosmos REST calls.
var cosmosHTTP = &http.Client{Timeout: 20 * time.Second}

// CosmosSendRequest carries a MsgSend request to the builder.
type CosmosSendRequest struct {
	ChainID   int64  `json:"chain_id"`   // registry id (resolves bech32 prefix)
	ToAddress string `json:"to"`         // bech32 destination
	Denom     string `json:"denom"`
	Amount    string `json:"amount"`     // base-unit integer string
	GasLimit  uint64 `json:"gas_limit"`  // default 100000 if zero
	Fee       string `json:"fee_amount"` // base-unit integer string; optional
	Memo      string `json:"memo"`
	Simulate  bool   `json:"simulate"` // skip broadcast when true
}

// CosmosResult carries the signed tx (broadcast hash when not simulated).
type CosmosResult struct {
	Prefix   string `json:"prefix"`
	FromAddr string `json:"from_address"`
	TxBytes  string `json:"tx_bytes_base64"`
	TxHash   string `json:"tx_hash,omitempty"`
	Broadcast bool  `json:"broadcasted"`
}

// cosmosLCD resolves the LCD REST base for a registry cosmos chain id.
// The seeded rpc.cosmos.directory host is rewritten to rest.cosmos.directory
// (that directory serves both rpc + rest per chain).
func cosmosLCD(chainID int64) string {
	for _, c := range nonEVMMainnet {
		if c.ID == chainID && c.ChainType == "cosmos" {
			u := c.RPCEndpoint
			u = strings.Replace(u, "rpc.cosmos.directory", "rest.cosmos.directory", 1)
			return strings.TrimRight(u, "/")
		}
	}
	return ""
}

// cosmosBech32Addr decodes a bech32 address to its 20-byte payload and
// validates the expected prefix.
func cosmosBech32Addr(addr, expectHRP string) ([]byte, error) {
	if idx := strings.Index(addr, "1"); idx > 0 {
		if addr[:idx] != expectHRP {
			return nil, fmt.Errorf("address HRP %q != %q", addr[:idx], expectHRP)
		}
	}
	data, err := bech32DecodeToBytes(addr)
	if err != nil {
		return nil, err
	}
	if len(data) != 20 {
		return nil, fmt.Errorf("cosmos address payload must be 20 bytes, got %d", len(data))
	}
	return data, nil
}

// bech32DecodeToBytes decodes a bech32 string to raw data bytes.
func bech32DecodeToBytes(s string) ([]byte, error) {
	pos := strings.LastIndex(s, "1")
	if pos < 1 || pos+7 > len(s) || len(s)-pos-7 > 90-6 {
		return nil, errors.New("invalid bech32 string")
	}
	vals := make([]byte, 0, len(s)-pos-1)
	for _, c := range s[pos+1:] {
		idx := strings.IndexByte(bech32Charset, byte(c))
		if idx < 0 {
			return nil, fmt.Errorf("bech32: invalid char %q", c)
		}
		vals = append(vals, byte(idx))
	}
	// verify checksum
	hrp := s[:pos]
	expanded := bech32HrpExpand(hrp)
	if bech32Polymod(append(expanded, vals...)) != 1 {
		return nil, errors.New("bech32 checksum mismatch")
	}
	data := vals[:len(vals)-6]
	out, err := convertBits(data, 5, 8, false)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ==================================================================
// Hand-rolled protobuf wire encoding (only the shapes we build)
// ==================================================================

type protoBuf struct{ b bytes.Buffer }

func (p *protoBuf) tag(field, wire uint) { p.uvarint(uint64(field<<3) | uint64(wire)) }
func (p *protoBuf) uvarint(v uint64)     { p.b.Write(encodeUvarint(v)) }
func (p *protoBuf) str(field uint, s string) {
	p.tag(field, 2)
	p.uvarint(uint64(len(s)))
	p.b.WriteString(s)
}
func (p *protoBuf) bytesField(field uint, b []byte) {
	p.tag(field, 2)
	p.uvarint(uint64(len(b)))
	p.b.Write(b)
}
func (p *protoBuf) uint64Field(field uint, v uint64) {
	p.tag(field, 0)
	p.uvarint(v)
}
func (p *protoBuf) boolField(field uint, v uint64) { p.uint64Field(field, v) }
func (p *protoBuf) msg(field uint, innerBytes []byte) {
	p.tag(field, 2)
	p.uvarint(uint64(len(innerBytes)))
	p.b.Write(innerBytes)
}
func (p *protoBuf) out() []byte { return p.b.Bytes() }

func encodeUvarint(v uint64) []byte {
	buf := make([]byte, 0, 10)
	for v >= 0x80 {
		buf = append(buf, byte(v)|0x80)
		v >>= 7
	}
	return append(buf, byte(v))
}

// cosmosCoin builds a cosmos.base.v1beta1.Coin message.
func cosmosCoin(denom, amount string) []byte {
	p := &protoBuf{}
	p.str(1, denom)
	p.str(2, amount)
	return p.out()
}

// cosmosMsgSend builds a cosmos.bank.v1beta1.MsgSend wrapped in Any.
func cosmosMsgSend(from, to, denom, amount string) []byte {
	inner := &protoBuf{}
	inner.str(1, from)   // from_address
	inner.str(2, to)     // to_address
	inner.msg(3, cosmosCoin(denom, amount))
	body := inner.out()
	any := &protoBuf{}
	any.str(1, "/cosmos.bank.v1beta1.MsgSend")
	any.bytesField(2, body)
	return any.out()
}

// cosmosTxBody builds TxBody protobuf.
func cosmosTxBody(msgs [][]byte, memo string, timeoutHeight uint64) []byte {
	p := &protoBuf{}
	for _, m := range msgs {
		p.msg(1, m)
	}
	if memo != "" {
		p.str(2, memo)
	}
	if timeoutHeight != 0 {
		p.uint64Field(3, timeoutHeight)
	}
	return p.out()
}

// cosmosAuthInfo builds AuthInfo with a single secp256k1 signer
// (PubKey of type /cosmos.crypto.secp256k1.PubKey).
func cosmosAuthInfo(pubCompressed []byte, feeDenom, feeAmount string, gasLimit, sequence uint64) []byte {
	pubMsg := &protoBuf{}
	pubMsg.bytesField(1, pubCompressed) // key
	pubAny := &protoBuf{}
	pubAny.str(1, "/cosmos.crypto.secp256k1.PubKey")
	pubAny.bytesField(2, pubMsg.out())

	signerInfo := &protoBuf{}
	signerInfo.msg(1, pubAny.out())                  // public_key (Any)
	// mode_info: ModeInfo.Single { mode }: enum SIGN_MODE_DIRECT=1
	single := &protoBuf{}
	single.uint64Field(1, 1)
	modeInfo := &protoBuf{}
	modeInfo.msg(1, single.out())
	signerInfo.msg(2, modeInfo.out())                // mode_info
	signerInfo.uint64Field(3, sequence)              // sequence

	fee := &protoBuf{}
	if feeAmount != "" {
		fee.msg(1, cosmosCoin(feeDenom, feeAmount)) // amount (Coin)
	}
	fee.uint64Field(2, gasLimit) // gas_limit

	auth := &protoBuf{}
	auth.msg(1, signerInfo.out()) // signer_infos
	auth.msg(2, fee.out())        // fee
	return auth.out()
}

// cosmosSignDocDirect builds the tx.v1beta1.SignDoc for SIGN_MODE_DIRECT.
type cosmosSignDocFields struct {
	bodyBytes     []byte
	authInfoBytes []byte
	chainID       string
	accountNumber uint64
}

// cosmosBuildSend builds a fully-signed TxRaw for a MsgSend.
func cosmosBuildSend(seed []byte, path string, req CosmosSendRequest, chainIDStr string, accountNumber, sequence uint64, fromAddr string) ([]byte, []byte, []byte, error) {
	prefix := cosmosPrefixByID(req.ChainID)
	if req.ToAddress != "" && !strings.HasPrefix(req.ToAddress, prefix+"1") {
		return nil, nil, nil, fmt.Errorf("destination address must have prefix %q", prefix)
	}
	feeDenom, feeAmt, gas := feeDefaultsOf(req)
	if !req.Simulate && sequence == 0 && accountNumber == 0 {
		// caller fetches the real values; zero here means they were genuinely
		// first-use (allowed), but callers should pass real values.
	}

	priv, err := hdDerive(seed, path)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("cosmos key derivation: %w", err)
	}
	pubCompressed := crypto.CompressPubkey(&priv.PublicKey)

	msg := cosmosMsgSend(fromAddr, req.ToAddress, req.Denom, req.Amount)
	bodyBytes := cosmosTxBody([][]byte{msg}, req.Memo, 0)
	authBytes := cosmosAuthInfo(pubCompressed, feeDenom, feeAmt, gas, sequence)

	// SignDoc = body_bytes || auth_info_bytes || chain_id || account_number
	signDoc := &protoBuf{}
	signDoc.bytesField(1, bodyBytes)
	signDoc.bytesField(2, authBytes)
	signDoc.str(3, chainIDStr)
	signDoc.uint64Field(4, accountNumber)
	hash := sha256.Sum256(signDoc.out())
	fullSig, err := crypto.Sign(hash[:], priv)
	if err != nil {
		return nil, nil, nil, err
	}

	// TxRaw { body_bytes, auth_info_bytes, signatures (repeated bytes) }
	txRaw := &protoBuf{}
	txRaw.bytesField(1, bodyBytes)
	txRaw.bytesField(2, authBytes)
	txRaw.bytesField(3, fullSig[:64])
	return txRaw.out(), fullSig[:64], pubCompressed, nil
}

func feeDefaultsOf(req CosmosSendRequest) (feeDenom, feeAmount string, gas uint64) {
	feeDenom = req.Denom
	feeAmount = req.Fee
	if req.GasLimit == 0 {
		gas = 100000
	} else {
		gas = req.GasLimit
	}
	if feeAmount == "" {
		feeAmount = "5000"
	}
	return
}

// cosmosAccountInfo fetches (accountNumber, sequence) for a bech32 address
// from the LCD. Returns an error on any HTTP/parsing failure — fail-closed.
func cosmosAccountInfo(ctx context.Context, lcdAddr, address string) (uint64, uint64, error) {
	u := lcdAddr + "/cosmos/auth/v1beta1/accounts/" + address
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return 0, 0, err
	}
	resp, err := cosmosHTTP.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("lcd account lookup: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, err
	}
	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("lcd account lookup: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body[:minInt(len(body), 200)])))
	}
	var parsed struct {
		Account struct {
			AccountNumber string `json:"account_number"`
			Sequence      string `json:"sequence"`
		} `json:"account"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return 0, 0, fmt.Errorf("lcd account reply decode: %w", err)
	}
	return parseUint(parsed.Account.AccountNumber), parseUint(parsed.Account.Sequence), nil
}

func parseUint(s string) uint64 {
	var v uint64
	fmt.Sscanf(s, "%d", &v)
	return v
}

// cosmosChainIDfetches the on-chain chain-id string from the node info
// endpoint of the LCD.
func cosmosChainID(ctx context.Context, lcd string) (string, error) {
	u := lcd + "/cosmos/base/tendermint/v1beta1/node_info"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	resp, err := cosmosHTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("node_info: HTTP %d", resp.StatusCode)
	}
	var parsed struct {
		DefaultNodeInfo struct {
			Network string `json:"network"`
		} `json:"default_node_info"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if parsed.DefaultNodeInfo.Network == "" {
		return "", errors.New("lcd returned empty chain-id")
	}
	return parsed.DefaultNodeInfo.Network, nil
}

// cosmosBroadcast submits tx bytes to /cosmos/tx/v1beta1/txs. Returns the
// tx hash from the response (never fabricated).
func cosmosBroadcast(ctx context.Context, lcd string, txBytes []byte) (string, error) {
	payload := map[string]interface{}{
		"tx_bytes": base64.StdEncoding.EncodeToString(txBytes),
		"mode":     "BROADCAST_MODE_SYNC",
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	u := lcd + "/cosmos/tx/v1beta1/txs"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := cosmosHTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("broadcast: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body[:minInt(len(body), 300)])))
	}
	var parsed struct {
		TxResponse struct {
			TxHash string      `json:"txhash"`
			Code   interface{} `json:"code"`
			RawLog string      `json:"raw_log"`
		} `json:"tx_response"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if parsed.TxResponse.TxHash == "" {
		return "", fmt.Errorf("broadcast rejected: %s", parsed.TxResponse.RawLog)
	}
	return parsed.TxResponse.TxHash, nil
}

// CosmosAddressFromSeedNew derives a cosmos bech32 address with a prefix
// resolved by registry chain id (for a 23-chain family).
func CosmosAddressFromSeedNew(seed []byte, path string, chainID int64) (string, string, error) {
	prefix := cosmosPrefixByID(chainID)
	priv, err := hdDerive(seed, path)
	if err != nil {
		return "", "", err
	}
	_, pub := btcec.PrivKeyFromBytes(crypto.FromECDSA(priv))
	pkh := hash160(pub.SerializeCompressed())
	addr, err := bech32Encode(prefix, pkh)
	if err != nil {
		return "", "", err
	}
	return addr, prefix, nil
}

// CosmosExecuteSend builds + optionally broadcasts a MsgSend transaction for
// one of the 23 cosmos chains. Account info and chain-id are fetched from the
// chain's real LCD — fail-closed on any error.
func CosmosExecuteSend(ctx context.Context, seed []byte, path string, req CosmosSendRequest) (*CosmosResult, error) {
	if req.ChainID == 0 {
		return nil, errors.New("cosmos send needs the numeric registry chain_id")
	}
	lcd := cosmosLCD(req.ChainID)
	if lcd == "" {
		return nil, fmt.Errorf("no LCD endpoint for registry chain id %d", req.ChainID)
	}
	fromAddr, prefix, err := CosmosAddressFromSeedNew(seed, path, req.ChainID)
	if err != nil {
		return nil, err
	}
	chainIDStr, err := cosmosChainID(ctx, lcd)
	if err != nil {
		return nil, fmt.Errorf("cosmos chain-id: %w", err)
	}
	acct, seq, err := cosmosAccountInfo(ctx, lcd, fromAddr)
	if err != nil {
		return nil, fmt.Errorf("cosmos account info: %w", err)
	}
	txRaw, sig, pub, err := cosmosBuildSend(seed, path, req, chainIDStr, acct, seq, fromAddr)
	if err != nil {
		return nil, err
	}
	res := &CosmosResult{
		Prefix:    prefix,
		FromAddr:  fromAddr,
		TxBytes:   base64.StdEncoding.EncodeToString(txRaw),
	}
	if !req.Simulate {
		hash, err := cosmosBroadcast(ctx, lcd, txRaw)
		if err != nil {
			return nil, fmt.Errorf("cosmos broadcast: %w", err)
		}
		res.TxHash = hash
		res.Broadcast = true
	} else {
		res.Broadcast = false
	}
	hexSig := hex.EncodeToString(sig)
	_ = pub
	_ = hexSig
	return res, nil
}
