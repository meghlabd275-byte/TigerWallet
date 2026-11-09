package aggregator

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"
)

// Bridge fee configuration.
//
// A bridge's fee is a property of that bridge's live liquidity and messaging
// cost; it cannot be derived locally. Rather than invent a plausible-looking
// percentage, each adapter reads an operator-supplied fee schedule and refuses
// to quote until one exists. This keeps every number the aggregator returns
// traceable to either real configuration or a real on-chain/API read.
//
// Configure with, per bridge:
//
//	BRIDGE_<NAME>_FEE_BPS      fee in basis points of the transfer amount
//	BRIDGE_<NAME>_FEE_FIXED    flat fee in the source token's smallest unit
//	BRIDGE_<NAME>_MIN_AMOUNT   minimum transferable amount, smallest unit
//	BRIDGE_<NAME>_DURATION_SEC expected settlement time in seconds
//
// Example: BRIDGE_STARGATE_FEE_BPS=5 BRIDGE_STARGATE_FEE_FIXED=0
//
// A bridge with no BRIDGE_<NAME>_FEE_BPS setting is treated as unconfigured:
// it reports no routes and returns ErrBridgeNotConfigured from GetQuote, so it
// can never contribute a made-up price to the aggregate.

// ErrBridgeNotConfigured is returned when a bridge has no fee schedule. It is
// distinct from a transient failure so callers can tell "this bridge is off"
// from "this bridge is broken".
var ErrBridgeNotConfigured = errors.New("bridge fee schedule not configured")

// FeeSchedule describes how a bridge prices a transfer.
type FeeSchedule struct {
	// FeeBPS is the proportional fee in basis points (1 bps = 0.01%).
	FeeBPS int64
	// FeeFixed is a flat fee in the source token's smallest unit.
	FeeFixed *big.Int
	// MinAmount is the smallest transferable amount in the source token's
	// smallest unit. Zero means no minimum.
	MinAmount *big.Int
	// Duration is the expected settlement time.
	Duration time.Duration
}

// FeeFor returns the fee and net output for an amount. The result is clamped at
// zero so a misconfigured fixed fee can never produce a negative payout.
func (f FeeSchedule) FeeFor(amount *big.Int) (*big.Int, *big.Int) {
	if amount == nil || amount.Sign() <= 0 {
		return big.NewInt(0), big.NewInt(0)
	}
	fee := new(big.Int)
	if f.FeeBPS > 0 {
		fee.Mul(amount, big.NewInt(f.FeeBPS))
		fee.Div(fee, big.NewInt(10000))
	}
	if f.FeeFixed != nil && f.FeeFixed.Sign() > 0 {
		fee.Add(fee, f.FeeFixed)
	}
	if fee.Cmp(amount) >= 0 {
		return fee, big.NewInt(0)
	}
	return fee, new(big.Int).Sub(amount, fee)
}

// loadFeeSchedule reads a bridge's fee schedule from the environment. ok is
// false when the bridge has not been configured.
func loadFeeSchedule(name string) (FeeSchedule, bool) {
	prefix := "BRIDGE_" + strings.ToUpper(name) + "_"
	rawBPS := strings.TrimSpace(os.Getenv(prefix + "FEE_BPS"))
	if rawBPS == "" {
		return FeeSchedule{}, false
	}
	bps, err := strconv.ParseInt(rawBPS, 10, 64)
	if err != nil || bps < 0 || bps > 10000 {
		return FeeSchedule{}, false
	}
	schedule := FeeSchedule{
		FeeBPS:   bps,
		FeeFixed: envBigInt(prefix + "FEE_FIXED"),
		Duration: time.Duration(envInt64(prefix+"DURATION_SEC", 300)) * time.Second,
	}
	if min := envBigInt(prefix + "MIN_AMOUNT"); min != nil && min.Sign() > 0 {
		schedule.MinAmount = min
	}
	if schedule.Duration <= 0 {
		schedule.Duration = 5 * time.Minute
	}
	return schedule, true
}

func envInt64(key string, def int64) int64 {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return def
	}
	return v
}

func envBigInt(key string) *big.Int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return big.NewInt(0)
	}
	v, ok := new(big.Int).SetString(raw, 10)
	if !ok || v.Sign() < 0 {
		return big.NewInt(0)
	}
	return v
}

// quoteFromSchedule builds a QuoteResponse from a configured schedule. It
// validates the amount against the bridge's minimum and never invents a value.
func quoteFromSchedule(name string, schedule FeeSchedule, req QuoteRequest) (QuoteResponse, error) {
	if req.Amount == nil || req.Amount.Sign() <= 0 {
		return QuoteResponse{}, fmt.Errorf("%s: invalid amount", name)
	}
	if schedule.MinAmount != nil && req.Amount.Cmp(schedule.MinAmount) < 0 {
		return QuoteResponse{}, fmt.Errorf("%s: amount below minimum %s", name, schedule.MinAmount)
	}
	fee, amountOut := schedule.FeeFor(req.Amount)
	if amountOut.Sign() <= 0 {
		return QuoteResponse{}, fmt.Errorf("%s: fee consumes the entire amount", name)
	}
	return QuoteResponse{
		Bridge:        name,
		SrcToken:      req.SrcToken,
		DstToken:      req.DstToken,
		AmountIn:      new(big.Int).Set(req.Amount),
		AmountOut:     amountOut,
		BridgeFee:     fee,
		Duration:      schedule.Duration,
		EstimatedTime: int64(schedule.Duration.Seconds()),
		Fee:           fee.String(),
		Route:         []Hop{{Bridge: name, Chain: req.DstChain, TokenIn: req.SrcToken, TokenOut: req.DstToken}},
	}, nil
}

// configurableBridge implements the Bridge interface for a protocol whose fee
// schedule is supplied by the operator. It is used by every adapter that has no
// locally-derivable fee.
type configurableBridge struct {
	name     string
	chains   []uint64
	schedule FeeSchedule
}

func (c *configurableBridge) Name() string              { return c.name }
func (c *configurableBridge) SupportedChains() []uint64 { return c.chains }

func (c *configurableBridge) GetQuote(ctx context.Context, req QuoteRequest) (QuoteResponse, error) {
	return quoteFromSchedule(c.name, c.schedule, req)
}

// Execute never fabricates a transaction hash. Cross-chain settlement requires
// a signed transaction, which only the key holder can produce, so the adapter
// reports that the caller must broadcast. Callers treat this as "action
// required", not as a completed transfer.
func (c *configurableBridge) Execute(ctx context.Context, req ExecuteRequest) (ExecuteResponse, error) {
	return ExecuteResponse{}, ErrExecutionRequiresSignature
}

func (c *configurableBridge) GetStatus(ctx context.Context, txHash string) (BridgeStatus, error) {
	if strings.TrimSpace(txHash) == "" {
		return StatusFailed, errors.New("empty tx hash")
	}
	// Without an indexer bound to this bridge's messaging layer the true status
	// is unknown. Reporting pending is the honest answer: it neither claims
	// success nor failure.
	return StatusPending, nil
}

// ErrExecutionRequiresSignature signals that the bridge adapter cannot settle
// the transfer itself and the client must sign and broadcast the transaction.
var ErrExecutionRequiresSignature = errors.New("bridge execution requires a client-signed transaction")

// stargateChains and acrossChains list the networks each protocol serves. They
// gate which routes the aggregator will even consider, so a quote is never
// offered for a chain the protocol cannot reach.
var (
	stargateChains = []uint64{1, 56, 137, 42161, 10, 8453}
	acrossChains   = []uint64{1, 10, 42161, 137, 8453}
)

// NewStargateBridge returns a Stargate adapter. It is unconfigured until
// BRIDGE_STARGATE_FEE_BPS is set, in which case it reports no quote.
func NewStargateBridge(rpcURL string) Bridge {
	if schedule, ok := loadFeeSchedule("stargate"); ok {
		return &configurableBridge{name: "Stargate", chains: stargateChains, schedule: schedule}
	}
	return &unconfiguredBridge{name: "Stargate", chains: stargateChains}
}

// NewAcrossBridge returns an Across adapter. It is unconfigured until
// BRIDGE_ACROSS_FEE_BPS is set.
func NewAcrossBridge(rpcURL string) Bridge {
	if schedule, ok := loadFeeSchedule("across"); ok {
		return &configurableBridge{name: "Across", chains: acrossChains, schedule: schedule}
	}
	return &unconfiguredBridge{name: "Across", chains: acrossChains}
}

// unconfiguredBridge represents a known protocol with no fee schedule. It
// advertises no chains and refuses to quote, so it silently drops out of route
// aggregation instead of contributing an invented price.
type unconfiguredBridge struct {
	name   string
	chains []uint64
}

func (u *unconfiguredBridge) Name() string              { return u.name }
func (u *unconfiguredBridge) SupportedChains() []uint64 { return nil }

func (u *unconfiguredBridge) GetQuote(ctx context.Context, req QuoteRequest) (QuoteResponse, error) {
	return QuoteResponse{}, fmt.Errorf("%s: %w", u.name, ErrBridgeNotConfigured)
}

func (u *unconfiguredBridge) Execute(ctx context.Context, req ExecuteRequest) (ExecuteResponse, error) {
	return ExecuteResponse{}, fmt.Errorf("%s: %w", u.name, ErrBridgeNotConfigured)
}

func (u *unconfiguredBridge) GetStatus(ctx context.Context, txHash string) (BridgeStatus, error) {
	return StatusFailed, fmt.Errorf("%s: %w", u.name, ErrBridgeNotConfigured)
}
