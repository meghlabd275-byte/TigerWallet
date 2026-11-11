// Package chain defines the TigerSmartChain network parameters, the
// deterministic genesis state and the consensus configuration used by the
// TigerWallet validator set.
//
// TigerSmartChain is a native proof-of-authority chain operated by the
// TigerWallet validator set. It anchors the TGR native coin, the TigerSwap
// liquidity pools and the platform governance contracts, so no external
// blockchain or third-party chain provider is required to operate the wallet
// platform.
package chain

import (
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"
)

// TGR is the smallest indivisible unit of the native TigerSmartChain coin.
// One TGR is 1e18 TGR wei, matching the EVM 18-decimal convention so that all
// existing TigerWallet EVM tooling works without a conversion layer.
const (
	// Decimals is the number of decimals of the native coin.
	Decimals = 18

	// DefaultChainID is the TigerSmartChain mainnet chain id.
	DefaultChainID int64 = 7710

	// TestnetChainID is the public testnet chain id.
	TestnetChainID int64 = 7711

	// NativeSymbol is the native coin ticker.
	NativeSymbol = "TGR"

	// NativeName is the native coin display name.
	NativeName = "Tiger"
)

var (
	// ErrInvalidChainID is returned when the configured chain id is not usable.
	ErrInvalidChainID = errors.New("tsc: invalid chain id")

	// ErrNoValidators is returned when a genesis is requested without any
	// initial validator, which would produce a chain that can never progress.
	ErrNoValidators = errors.New("tsc: genesis requires at least one validator")

	// ErrDuplicateValidator is returned when the same signer is configured twice.
	ErrDuplicateValidator = errors.New("tsc: duplicate validator signer")

	// ErrInvalidAddress is returned for a malformed account address.
	ErrInvalidAddress = errors.New("tsc: invalid address")
)

// Quorum returns the number of authorities that must sign a block before it is
// accepted. The underlying proof-of-authority rule requires a strict majority
// of the authorised signer set, so a network configured with N validators only
// progresses once at least N/2+1 of them are online and voting.
//
// This is a property of the consensus, not a limitation: it is what stops a
// minority of authorities from rewriting history.
func Quorum(validatorCount int) int {
	if validatorCount <= 0 {
		return 0
	}
	return validatorCount/2 + 1
}

// Wei converts a whole-TGR amount to its base unit representation.
func Wei(tgr int64) *big.Int {
	return new(big.Int).Mul(big.NewInt(tgr), new(big.Int).Exp(big.NewInt(10), big.NewInt(Decimals), nil))
}

// WeiFromString parses a decimal TGR amount ("12.5") into wei without floating
// point arithmetic, so no precision is ever lost.
func WeiFromString(amount string) (*big.Int, error) {
	s := strings.TrimSpace(amount)
	if s == "" {
		return nil, fmt.Errorf("tsc: empty amount")
	}
	neg := false
	if strings.HasPrefix(s, "-") {
		neg = true
		s = s[1:]
	}
	intPart, fracPart, _ := strings.Cut(s, ".")
	if intPart == "" && fracPart == "" {
		return nil, fmt.Errorf("tsc: malformed amount %q", amount)
	}
	for _, r := range intPart + fracPart {
		if r < '0' || r > '9' {
			return nil, fmt.Errorf("tsc: malformed amount %q", amount)
		}
	}
	if len(fracPart) > Decimals {
		return nil, fmt.Errorf("tsc: amount %q has more than %d decimals", amount, Decimals)
	}
	if intPart == "" {
		intPart = "0"
	}
	base, ok := new(big.Int).SetString(intPart+fracPart+strings.Repeat("0", Decimals-len(fracPart)), 10)
	if !ok {
		return nil, fmt.Errorf("tsc: malformed amount %q", amount)
	}
	if neg {
		base.Neg(base)
	}
	return base, nil
}

// GenesisAccount is a funded account in the TigerSmartChain genesis state.
type GenesisAccount struct {
	// Address is the account address as a 0x-prefixed 20-byte hex string.
	Address string `json:"address"`

	// Balance is the initial native balance in wei.
	Balance *big.Int `json:"balance"`

	// Label documents the purpose of the allocation. It is metadata only and
	// never affects state, but it is committed to the genesis hash so an
	// operator cannot silently repurpose an allocation.
	Label string `json:"label,omitempty"`

	// Vesting describes an optional linear vesting schedule enforced by the
	// TigerVesting contract at deploy time.
	Vesting *VestingSchedule `json:"vesting,omitempty"`
}

// VestingSchedule is a linear unlock schedule for a genesis allocation.
type VestingSchedule struct {
	// Beneficiary is the address that receives unlocked funds.
	Beneficiary string `json:"beneficiary"`

	// CliffSeconds is the delay from genesis before any funds unlock.
	CliffSeconds int64 `json:"cliff_seconds"`

	// DurationSeconds is the total vesting duration after the cliff.
	DurationSeconds int64 `json:"duration_seconds"`
}

// Validator is an initial TigerSmartChain authority.
type Validator struct {
	// Signer is the address that signs TigerSmartChain blocks.
	Signer string `json:"signer"`

	// Name is the operator label shown in the explorer and admin console.
	Name string `json:"name"`

	// Payout is the address that receives the validator commission.
	Payout string `json:"payout"`

	// CommissionBps is the validator commission in basis points (0-10000).
	CommissionBps uint32 `json:"commission_bps"`

	// SelfStake is the amount of TGR the validator bonds at genesis, in wei.
	SelfStake *big.Int `json:"self_stake"`
}

// Config is the full TigerSmartChain node configuration.
type Config struct {
	// ChainID is the EVM chain id. Mainnet defaults to DefaultChainID.
	ChainID int64 `json:"chain_id"`

	// Name is the human readable network name.
	Name string `json:"name"`

	// Validators is the initial authority set.
	Validators []Validator `json:"validators"`

	// Allocations is the genesis token distribution.
	Allocations []GenesisAccount `json:"allocations"`

	// BlockPeriodSeconds is the minimum interval between blocks.
	BlockPeriodSeconds uint64 `json:"block_period_seconds"`

	// EpochLength is the number of blocks between checkpoint blocks that
	// re-commit the full validator set.
	EpochLength uint64 `json:"epoch_length"`

	// GasLimit is the per block gas limit.
	GasLimit uint64 `json:"gas_limit"`

	// BaseFeeWei is the initial EIP-1559 base fee in wei.
	BaseFeeWei *big.Int `json:"base_fee_wei"`

	// UnbondingSeconds is the validator unbonding period enforced by the
	// TigerStaking contract.
	UnbondingSeconds int64 `json:"unbonding_seconds"`

	// MinValidatorStake is the minimum bonded stake in wei.
	MinValidatorStake *big.Int `json:"min_validator_stake"`

	// MaxValidators caps the active authority set.
	MaxValidators int `json:"max_validators"`

	// Treasury is the address that receives protocol fees.
	Treasury string `json:"treasury"`

	// GenesisTime is the Unix timestamp of the genesis block. Pinning it in
	// the network definition is what makes the genesis hash reproducible from
	// the config file alone.
	GenesisTime int64 `json:"genesis_time"`
}

// DefaultConfig returns the TigerSmartChain mainnet configuration.
//
// The token distribution is fixed and auditable:
//
//	40%  ecosystem + community rewards
//	20%  treasury (protocol owned liquidity and operations)
//	15%  TigerSwap liquidity bootstrap
//	15%  founding team, vested linearly over 4 years after a 1 year cliff
//	10%  strategic partners, vested linearly over 2 years after a 6 month cliff
func DefaultConfig() *Config {
	return &Config{
		ChainID:            DefaultChainID,
		Name:               "TigerSmartChain",
		BlockPeriodSeconds: 1,
		EpochLength:        30000,
		GasLimit:           30_000_000,
		BaseFeeWei:         big.NewInt(1_000_000_000),
		UnbondingSeconds:   21 * 24 * 3600,
		MinValidatorStake:  Wei(100_000),
		MaxValidators:      64,
	}
}

// Validate reports whether the configuration can produce a usable chain.
func (c *Config) Validate() error {
	if c.ChainID <= 0 {
		return fmt.Errorf("%w: %d", ErrInvalidChainID, c.ChainID)
	}
	if c.Name == "" {
		return fmt.Errorf("tsc: network name is required")
	}
	if len(c.Validators) == 0 {
		return ErrNoValidators
	}
	if c.MaxValidators > 0 && len(c.Validators) > c.MaxValidators {
		return fmt.Errorf("tsc: %d validators exceeds max %d", len(c.Validators), c.MaxValidators)
	}
	seen := make(map[string]struct{}, len(c.Validators))
	for _, v := range c.Validators {
		addr, err := NormalizeAddress(v.Signer)
		if err != nil {
			return fmt.Errorf("validator %s: %w", v.Name, err)
		}
		if _, dup := seen[addr]; dup {
			return fmt.Errorf("%w: %s", ErrDuplicateValidator, addr)
		}
		seen[addr] = struct{}{}
		if v.Payout != "" {
			if _, err := NormalizeAddress(v.Payout); err != nil {
				return fmt.Errorf("validator %s payout: %w", v.Name, err)
			}
		}
		if v.CommissionBps > 10_000 {
			return fmt.Errorf("validator %s: commission %d bps exceeds 10000", v.Name, v.CommissionBps)
		}
		if v.SelfStake != nil && v.SelfStake.Sign() < 0 {
			return fmt.Errorf("validator %s: negative self stake", v.Name)
		}
	}
	if c.Treasury != "" {
		if _, err := NormalizeAddress(c.Treasury); err != nil {
			return fmt.Errorf("treasury: %w", err)
		}
	}
	for _, a := range c.Allocations {
		if _, err := NormalizeAddress(a.Address); err != nil {
			return fmt.Errorf("allocation %s: %w", a.Label, err)
		}
		if a.Balance == nil || a.Balance.Sign() < 0 {
			return fmt.Errorf("allocation %s: balance must be non-negative", a.Label)
		}
		if a.Vesting != nil {
			if _, err := NormalizeAddress(a.Vesting.Beneficiary); err != nil {
				return fmt.Errorf("allocation %s vesting beneficiary: %w", a.Label, err)
			}
			if a.Vesting.CliffSeconds < 0 || a.Vesting.DurationSeconds <= 0 {
				return fmt.Errorf("allocation %s: invalid vesting schedule", a.Label)
			}
		}
	}
	if c.BlockPeriodSeconds == 0 {
		return fmt.Errorf("tsc: block period must be greater than zero")
	}
	if c.EpochLength == 0 {
		return fmt.Errorf("tsc: epoch length must be greater than zero")
	}
	if c.GasLimit < 21_000 {
		return fmt.Errorf("tsc: gas limit %d is below the 21000 minimum", c.GasLimit)
	}
	return nil
}

// NormalizeAddress validates a 0x-prefixed 20-byte hex address and returns it
// in canonical lower-case form. Accepts only the strict hex form so that a
// mistyped allocation fails at startup rather than silently dropping funds.
func NormalizeAddress(addr string) (string, error) {
	s := strings.TrimSpace(addr)
	if s == "" {
		return "", fmt.Errorf("%w: empty", ErrInvalidAddress)
	}
	if !strings.HasPrefix(s, "0x") && !strings.HasPrefix(s, "0X") {
		return "", fmt.Errorf("%w: %q must be 0x-prefixed", ErrInvalidAddress, addr)
	}
	body := s[2:]
	if len(body) != 40 {
		return "", fmt.Errorf("%w: %q must have 40 hex digits, got %d", ErrInvalidAddress, addr, len(body))
	}
	for _, r := range body {
		switch {
		case r >= '0' && r <= '9', r >= 'a' && r <= 'f', r >= 'A' && r <= 'F':
		default:
			return "", fmt.Errorf("%w: %q contains a non-hex character", ErrInvalidAddress, addr)
		}
	}
	return "0x" + strings.ToLower(body), nil
}

// ChainIDFromEnv resolves the effective chain id, allowing an operator to run a
// testnet or a private chain from the same binary.
func ChainIDFromEnv() int64 {
	raw := strings.TrimSpace(os.Getenv("TSC_CHAIN_ID"))
	if raw == "" {
		return DefaultChainID
	}
	n, ok := new(big.Int).SetString(raw, 10)
	if !ok || !n.IsInt64() || n.Int64() <= 0 {
		return DefaultChainID
	}
	return n.Int64()
}
