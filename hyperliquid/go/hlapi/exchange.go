package hlapi

// exchange.go — real Hyperliquid exchange actions: EIP-712 (phantom Agent)
// signing of msgpack-encoded actions, submitted to the live /exchange
// endpoint. Fail-closed: no order is recorded as submitted unless the venue
// accepted it; no signatures are fabricated.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/vmihailenco/msgpack/v5"
	"golang.org/x/crypto/sha3"
)

// ExchangeURL is the Hyperliquid exchange endpoint (override HL_EXCHANGE_URL).
var ExchangeURL = "https://api.hyperliquid.xyz/exchange"

// eip712Domain constants for Hyperliquid mainnet action signing.
var (
	eip712DomainName    = "Exchange"
	eip712DomainVersion = "1"
	eip712ChainID       = int64(1337)
	eip712Source        = "https://api.hyperliquid.xyz"
)

// Wire structs: msgpack encodes struct fields in declaration order, which
// matches Hyperliquid's canonical action encoding.

type orderTypeWire struct {
	Limit *limitWire `json:"limit,omitempty" msgpack:"limit,omitempty"`
}

type limitWire struct {
	Tif string `json:"tif" msgpack:"tif"` // Gtc, Ioc, Alo
}

type orderWire struct {
	A int           `json:"a" msgpack:"a"` // asset index
	B bool          `json:"b" msgpack:"b"` // isBuy
	P string        `json:"p" msgpack:"p"` // price
	S string        `json:"s" msgpack:"s"` // size
	R bool          `json:"r" msgpack:"r"` // reduceOnly
	T orderTypeWire `json:"t" msgpack:"t"`
}

type orderAction struct {
	Type     string      `json:"type" msgpack:"type"`
	Orders   []orderWire `json:"orders" msgpack:"orders"`
	Grouping string      `json:"grouping" msgpack:"grouping"`
}

type cancelWire struct {
	A int   `json:"a" msgpack:"a"`
	O int64 `json:"o" msgpack:"o"`
}

type cancelAction struct {
	Type    string       `json:"type" msgpack:"type"`
	Cancels []cancelWire `json:"cancels" msgpack:"cancels"`
}

// keccak256 helper.
func keccak(b []byte) []byte {
	h := sha3.NewLegacyKeccak256()
	h.Write(b)
	return h.Sum(nil)
}

// actionHash computes keccak256(msgpack(action) || nonce_be8 || 0x00) — the
// Hyperliquid action hash (no vault address variant).
func actionHash(action any, nonce uint64) ([]byte, error) {
	packed, err := msgpack.Marshal(action)
	if err != nil {
		return nil, fmt.Errorf("msgpack action: %w", err)
	}
	buf := append([]byte{}, packed...)
	nonceBE := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		nonceBE[i] = byte(nonce)
		nonce >>= 8
	}
	buf = append(buf, nonceBE...)
	buf = append(buf, 0x00) // no vault address
	return keccak(buf), nil
}

// signActionHash produces the EIP-712 signature over the phantom Agent type.
func signActionHash(privKeyHex string, hash []byte) (r, s string, v int, err error) {
	key, err := crypto.HexToECDSA(strings.TrimPrefix(privKeyHex, "0x"))
	if err != nil {
		return "", "", 0, fmt.Errorf("invalid private key: %w", err)
	}

	// EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)
	domainTypeHash := keccak([]byte("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"))
	domainSep := keccak(bytes.Join([][]byte{
		domainTypeHash,
		keccak([]byte(eip712DomainName)),
		keccak([]byte(eip712DomainVersion)),
		common.LeftPadBytes(big.NewInt(eip712ChainID).Bytes(), 32),
		common.LeftPadBytes(common.Address{}.Bytes(), 32),
	}, nil))

	// Agent(string source,bytes32 connectionId)
	agentTypeHash := keccak([]byte("Agent(string source,bytes32 connectionId)"))
	structHash := keccak(bytes.Join([][]byte{
		agentTypeHash,
		keccak([]byte(eip712Source)),
		common.LeftPadBytes(hash, 32),
	}, nil))

	digest := keccak(append([]byte{0x19, 0x01}, append(domainSep, structHash...)...))
	sig, err := crypto.Sign(digest, key)
	if err != nil {
		return "", "", 0, err
	}
	r = hex64(sig[:32])
	s = hex64(sig[32:64])
	v = int(sig[64]) + 27
	return r, s, v, nil
}

func hex64(b []byte) string {
	const hexdigits = "0123456789abcdef"
	out := make([]byte, 64)
	for i, c := range b {
		out[i*2] = hexdigits[c>>4]
		out[i*2+1] = hexdigits[c&0x0f]
	}
	return "0x" + string(out)
}

// floatWire formats price/size per Hyperliquid wire rules (plain decimal,
// no exponent).
func floatWire(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// postExchange submits a signed action to the live exchange endpoint.
func postExchange(ctx context.Context, action any, nonce uint64, r, s string, v int, out any) error {
	body := map[string]any{
		"action": action,
		"nonce":  nonce,
		"signature": map[string]any{
			"r": r,
			"s": s,
			"v": v,
		},
		"vaultAddress": nil,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ExchangeURL, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var parsed struct {
		Status   string          `json:"status"`
		Response json.RawMessage `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return fmt.Errorf("exchange response decode (HTTP %d): %w", resp.StatusCode, err)
	}
	if parsed.Status != "ok" {
		var errMsg string
		json.Unmarshal(parsed.Response, &errMsg)
		return fmt.Errorf("hyperliquid rejected action: %s", errMsg)
	}
	if out != nil && len(parsed.Response) > 0 {
		return json.Unmarshal(parsed.Response, out)
	}
	return nil
}

// OrderResult is the real venue response for one order.
type OrderResult struct {
	// VenueOrderID is the real order id when the order rests or the fill id.
	VenueOrderID int64
	Status       string // "resting" | "filled" | "error"
	AvgPrice     float64
	FilledSize   float64
	Error        string
}

type orderResponse struct {
	Type string `json:"type"`
	Data struct {
		Statuses []struct {
			Resting *struct {
				Oid int64 `json:"oid"`
			} `json:"resting,omitempty"`
			Filled *struct {
				Oid     int64  `json:"oid"`
				AvgPx   string `json:"avgPx"`
				TotalSz string `json:"totalSz"`
			} `json:"filled,omitempty"`
			Error *string `json:"error,omitempty"`
		} `json:"statuses"`
	} `json:"data"`
}

// PlacePerpOrder signs and submits a real perpetual order. tif: "Gtc" (limit)
// or "Ioc" (market-like). Returns the real venue result.
func PlacePerpOrder(ctx context.Context, privKeyHex string, assetID int, isBuy bool, price, size float64, reduceOnly bool, tif string) (*OrderResult, error) {
	if tif == "" {
		tif = "Gtc"
	}
	action := orderAction{
		Type: "order",
		Orders: []orderWire{{
			A: assetID,
			B: isBuy,
			P: floatWire(price),
			S: floatWire(size),
			R: reduceOnly,
			T: orderTypeWire{Limit: &limitWire{Tif: tif}},
		}},
		Grouping: "na",
	}
	nonce := uint64(time.Now().UnixMilli())
	hash, err := actionHash(action, nonce)
	if err != nil {
		return nil, err
	}
	r, sv, v, err := signActionHash(privKeyHex, hash)
	if err != nil {
		return nil, err
	}
	var resp orderResponse
	if err := postExchange(ctx, action, nonce, r, sv, v, &resp); err != nil {
		return nil, err
	}
	if len(resp.Data.Statuses) == 0 {
		return nil, fmt.Errorf("hyperliquid returned no order status")
	}
	st := resp.Data.Statuses[0]
	switch {
	case st.Resting != nil:
		return &OrderResult{VenueOrderID: st.Resting.Oid, Status: "resting"}, nil
	case st.Filled != nil:
		avg, _ := strconv.ParseFloat(st.Filled.AvgPx, 64)
		sz, _ := strconv.ParseFloat(st.Filled.TotalSz, 64)
		return &OrderResult{VenueOrderID: st.Filled.Oid, Status: "filled", AvgPrice: avg, FilledSize: sz}, nil
	case st.Error != nil:
		return nil, fmt.Errorf("order rejected by venue: %s", *st.Error)
	}
	return nil, fmt.Errorf("unrecognized venue order status")
}

// CancelVenueOrder signs and submits a real cancel for a venue order id.
func CancelVenueOrder(ctx context.Context, privKeyHex string, assetID int, venueOrderID int64) error {
	action := cancelAction{
		Type:    "cancel",
		Cancels: []cancelWire{{A: assetID, O: venueOrderID}},
	}
	nonce := uint64(time.Now().UnixMilli())
	hash, err := actionHash(action, nonce)
	if err != nil {
		return err
	}
	r, s, v, err := signActionHash(privKeyHex, hash)
	if err != nil {
		return err
	}
	return postExchange(ctx, action, nonce, r, s, v, nil)
}

// AssetIndex resolves the real venue asset index from live meta. Fail-closed.
func AssetIndex(ctx context.Context, asset string) (int, error) {
	var raw []json.RawMessage
	if err := postInfo(ctx, map[string]any{"type": "metaAndAssetCtxs"}, &raw); err != nil {
		return 0, err
	}
	if len(raw) < 1 {
		return 0, fmt.Errorf("empty meta response")
	}
	var meta struct {
		Universe []struct {
			Name string `json:"name"`
		} `json:"universe"`
	}
	if err := json.Unmarshal(raw[0], &meta); err != nil {
		return 0, err
	}
	for i, u := range meta.Universe {
		if strings.EqualFold(u.Name, asset) {
			return i, nil
		}
	}
	return 0, fmt.Errorf("asset %q not found on hyperliquid", asset)
}

// VenuePosition is a real open position from the clearinghouse state.
type VenuePosition struct {
	Asset            string
	Size             float64 // signed: positive=long, negative=short
	EntryPrice       float64
	Leverage         float64
	LiquidationPrice float64
	UnrealizedPNL    float64
}

// GetVenuePositions fetches real open positions via clearinghouseState.
func GetVenuePositions(ctx context.Context, address string) ([]VenuePosition, error) {
	var parsed struct {
		AssetPositions []struct {
			Position struct {
				Coin     string `json:"coin"`
				Szi      string `json:"szi"`
				EntryPx  string `json:"entryPx"`
				Leverage struct {
					Value float64 `json:"value"`
				} `json:"leverage"`
				LiquidationPx *string `json:"liquidationPx"`
				UnrealizedPnl string  `json:"unrealizedPnl"`
			} `json:"position"`
		} `json:"assetPositions"`
	}
	if err := postInfo(ctx, map[string]any{"type": "clearinghouseState", "user": address}, &parsed); err != nil {
		return nil, err
	}
	var out []VenuePosition
	for _, ap := range parsed.AssetPositions {
		p := ap.Position
		vp := VenuePosition{Asset: p.Coin}
		vp.Size, _ = strconv.ParseFloat(p.Szi, 64)
		vp.EntryPrice, _ = strconv.ParseFloat(p.EntryPx, 64)
		vp.Leverage = p.Leverage.Value
		if p.LiquidationPx != nil {
			vp.LiquidationPrice, _ = strconv.ParseFloat(*p.LiquidationPx, 64)
		}
		vp.UnrealizedPNL, _ = strconv.ParseFloat(p.UnrealizedPnl, 64)
		out = append(out, vp)
	}
	return out, nil
}

// SignerKeyFromEnv returns the configured signer key; empty when unset
// (callers fail closed).
func SignerKeyFromEnv() string {
	return os.Getenv("HL_PRIVATE_KEY")
}
