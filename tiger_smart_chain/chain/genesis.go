package chain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

// Errors returned while building a genesis block.
var (
	ErrSignerMissing   = errors.New("tsc: validator signer address is not a valid address")
	ErrAllocOverflow   = errors.New("tsc: total genesis allocation overflows uint256")
	ErrUnknownAllocKey = errors.New("tsc: unknown allocation label")
)

// GenesisExtraVanity is the number of vanity bytes required at the start of a
// checkpoint block extra-data field.
const GenesisExtraVanity = 32

// GenesisExtraSeal is the number of signature bytes appended to every block.
const GenesisExtraSeal = 65

// BuildGenesis converts a validated Config into the concrete genesis block that
// the node writes to its database. The result is fully deterministic: the same
// Config always yields the same genesis hash, which is what makes the network
// verifiable by every participant.
func BuildGenesis(cfg *Config) (*core.Genesis, error) {
	if cfg == nil {
		return nil, fmt.Errorf("tsc: nil config")
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	alloc := core.GenesisAlloc{}
	for _, a := range cfg.Allocations {
		addr, err := NormalizeAddress(a.Address)
		if err != nil {
			return nil, err
		}
		commonAddr := common.HexToAddress(addr)
		if existing, ok := alloc[commonAddr]; ok {
			sum := new(big.Int).Add(existing.Balance, a.Balance)
			if sum.BitLen() > 256 {
				return nil, fmt.Errorf("%w for %s", ErrAllocOverflow, addr)
			}
			existing.Balance = sum
			alloc[commonAddr] = existing
			continue
		}
		alloc[commonAddr] = core.GenesisAccount{Balance: new(big.Int).Set(a.Balance)}
	}

	// A validator that is expected to produce blocks must own its self stake.
	for _, v := range cfg.Validators {
		if v.SelfStake == nil || v.SelfStake.Sign() == 0 {
			continue
		}
		addr := common.HexToAddress(v.Signer)
		acct := alloc[addr]
		if acct.Balance == nil {
			acct.Balance = new(big.Int)
		}
		acct.Balance = new(big.Int).Add(acct.Balance, v.SelfStake)
		alloc[addr] = acct
	}

	// Platform tokens (RUSD stablecoin, WTGR wrapped native) are committed
	// into the genesis state so every client knows their addresses from
	// block 0 without a deployment manifest.
	if err := ApplyPredeploys(alloc, cfg); err != nil {
		return nil, err
	}

	extra, err := validatorExtraData(cfg.Validators)
	if err != nil {
		return nil, err
	}

	baseFee := cfg.BaseFeeWei
	if baseFee == nil {
		baseFee = big.NewInt(1_000_000_000)
	}
	timestamp := uint64(cfg.GenesisTime)
	if timestamp == 0 {
		timestamp = uint64(GenesisTimestamp)
	}
	if override := genesisTimestampFromEnv(); override > 0 {
		timestamp = uint64(override)
	}

	return &core.Genesis{
		Config:     ChainConfig(cfg),
		Nonce:      0,
		Timestamp:  timestamp,
		ExtraData:  extra,
		GasLimit:   cfg.GasLimit,
		Difficulty: big.NewInt(1),
		Mixhash:    common.Hash{},
		Coinbase:   common.Address{},
		Alloc:      alloc,
		BaseFee:    baseFee,
	}, nil
}

// genesisTimestampFromEnv lets an operator pin the genesis time so that a
// private network can start at a known instant.
func genesisTimestampFromEnv() int64 {
	raw := strings.TrimSpace(os.Getenv("TSC_GENESIS_TIMESTAMP"))
	if raw == "" {
		return 0
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

// GenesisTimestamp is a fixed genesis timestamp so that the genesis hash does
// not change between builds. Operators that want a different start time set
// TSC_GENESIS_TIMESTAMP.
var GenesisTimestamp int64 = 1767225600 // 2026-01-01T00:00:00Z

// validatorExtraData encodes the initial authority set into the checkpoint
// extra-data field: 32 vanity bytes, then 20 bytes per signer, then the 65 byte
// seal placeholder.
func validatorExtraData(validators []Validator) ([]byte, error) {
	extra := make([]byte, GenesisExtraVanity, GenesisExtraVanity+len(validators)*common.AddressLength+GenesisExtraSeal)
	copy(extra, []byte("TigerSmartChain genesis block...."))
	for _, v := range validators {
		addr, err := NormalizeAddress(v.Signer)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrSignerMissing, v.Signer)
		}
		extra = append(extra, common.HexToAddress(addr).Bytes()...)
	}
	extra = append(extra, make([]byte, GenesisExtraSeal)...)
	return extra, nil
}

// ChainConfig returns the EVM rule set activated on TigerSmartChain.
//
// Every block-based fork up to London is active from block zero, which is what
// gives the chain EIP-155 replay protection, EIP-1559 fee markets and the
// Berlin/London opcode set from the first block.
//
// The chain is deliberately kept pre-merge. TigerSmartChain is a
// proof-of-authority network operated by the TigerWallet validator set, and it
// must never attempt to follow the Ethereum proof-of-stake beacon chain.
//
// That choice fixes the EVM target of the genesis predeploys: in the EVM
// specification Shanghai and Cancun are *merge* forks, so a pre-merge chain
// executes neither, and an opcode introduced by them (PUSH0, from EIP-3855) is
// invalid. contracts/RUSD.sol and contracts/WTGR.sol are therefore compiled with
// `--evm-version london` and contain no post-London opcode. See the SOLC_FLAGS
// line in the Makefile: changing the EVM target there without changing this
// function would produce predeploy bytecode the chain cannot execute.
func ChainConfig(cfg *Config) *params.ChainConfig {
	zero := big.NewInt(0)
	return &params.ChainConfig{
		ChainID:                       big.NewInt(cfg.ChainID),
		HomesteadBlock:                zero,
		DAOForkBlock:                  zero,
		DAOForkSupport:                true,
		EIP150Block:                   zero,
		EIP155Block:                   zero,
		EIP158Block:                   zero,
		ByzantiumBlock:                zero,
		ConstantinopleBlock:           zero,
		PetersburgBlock:               zero,
		IstanbulBlock:                 zero,
		MuirGlacierBlock:              zero,
		BerlinBlock:                   zero,
		LondonBlock:                   zero,
		ArrowGlacierBlock:             nil,
		GrayGlacierBlock:              nil,
		MergeNetsplitBlock:            nil,
		Clique:                        &params.CliqueConfig{Period: cfg.BlockPeriodSeconds, Epoch: cfg.EpochLength},
		TerminalTotalDifficulty:       new(big.Int).Exp(big.NewInt(10), big.NewInt(30), nil),
		TerminalTotalDifficultyPassed: false,
	}
}

// FileConfig is the on-disk JSON representation of a network definition.
type FileConfig struct {
	Config *Config `json:"config"`
}

// LoadConfigFile reads and validates a network definition from disk.
func LoadConfigFile(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("tsc: read config %s: %w", path, err)
	}
	var fc FileConfig
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&fc); err != nil {
		return nil, fmt.Errorf("tsc: parse config %s: %w", path, err)
	}
	if err := dec.Decode(new(struct{})); err != io.EOF {
		return nil, fmt.Errorf("tsc: config %s has trailing content", path)
	}
	if fc.Config == nil {
		return nil, fmt.Errorf("tsc: config %s is missing the config object", path)
	}
	if err := fc.Config.Validate(); err != nil {
		return nil, err
	}
	return fc.Config, nil
}

// SaveConfigFile writes a network definition to disk with restrictive
// permissions, because a network definition lists validator signer addresses.
func SaveConfigFile(path string, cfg *Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	blob, err := json.MarshalIndent(FileConfig{Config: cfg}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(blob, '\n'), 0o600)
}

// DeriveAccount deterministically derives a TigerSmartChain account from a
// master seed and an index. The derivation mirrors the TigerWallet HD scheme:
// HMAC-free SHA-256 expansion over the seed and the index, hashed into an
// secp256k1 key. It exists so an operator can bootstrap a full validator set
// from a single secret without ever writing a private key to a config file.
func DeriveAccount(seed []byte, index uint32) (common.Address, *big.Int, error) {
	if len(seed) < 16 {
		return common.Address{}, nil, fmt.Errorf("tsc: seed must be at least 16 bytes")
	}
	for attempt := uint32(0); attempt < 1024; attempt++ {
		h := sha256.New()
		h.Write([]byte("TigerSmartChain/account/v1"))
		h.Write(seed)
		var idx [8]byte
		big.NewInt(int64(index)).FillBytes(idx[:])
		h.Write(idx[:])
		var att [4]byte
		big.NewInt(int64(attempt)).FillBytes(att[:])
		h.Write(att[:])
		sum := h.Sum(nil)
		key, err := crypto.ToECDSA(sum)
		if err != nil {
			continue
		}
		addr := crypto.PubkeyToAddress(key.PublicKey)
		return addr, new(big.Int).SetBytes(sum), nil
	}
	return common.Address{}, nil, fmt.Errorf("tsc: unable to derive account %d", index)
}

// SeedFromString accepts either a hex encoded seed or a plain passphrase of at
// least 16 characters. It never falls back to a built-in default, because a
// shared default seed would make every deployment derive the same validator
// keys.
func SeedFromString(raw string) ([]byte, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil, fmt.Errorf("tsc: seed must not be empty")
	}
	if decoded, err := hex.DecodeString(strings.TrimPrefix(s, "0x")); err == nil && len(decoded) >= 16 {
		return decoded, nil
	}
	if len(s) < 16 {
		return nil, fmt.Errorf("tsc: seed must be at least 16 characters or 32 hex digits")
	}
	return []byte(s), nil
}

// SeedFromEnv reads the genesis seed used to derive deterministic bootstrap
// accounts. An operator that does not supply one gets an error rather than a
// silently reused public seed.
func SeedFromEnv() ([]byte, error) {
	raw := strings.TrimSpace(os.Getenv("TSC_GENESIS_SEED"))
	if raw == "" {
		return nil, fmt.Errorf("tsc: TSC_GENESIS_SEED is not set")
	}
	return SeedFromString(raw)
}

// TokenDistribution returns the canonical TigerSmartChain allocation split for
// a total supply, expressed in wei.
func TokenDistribution(totalSupply *big.Int, seed []byte) ([]GenesisAccount, error) {
	if totalSupply == nil || totalSupply.Sign() <= 0 {
		return nil, fmt.Errorf("tsc: total supply must be positive")
	}
	pct := func(bps int64) *big.Int {
		out := new(big.Int).Mul(totalSupply, big.NewInt(bps))
		return out.Div(out, big.NewInt(10_000))
	}

	labels := []struct {
		label    string
		bps      int64
		index    uint32
		vesting  *VestingSchedule
		nickname string
	}{
		{"ecosystem", 4000, 0, nil, "Ecosystem and community rewards"},
		{"treasury", 2000, 1, nil, "Protocol treasury"},
		{"liquidity", 1500, 2, nil, "TigerSwap liquidity bootstrap"},
		{"team", 1500, 3, &VestingSchedule{CliffSeconds: 365 * 24 * 3600, DurationSeconds: 4 * 365 * 24 * 3600}, "Founding team, 1 year cliff then 4 year linear vesting"},
		{"partners", 1000, 4, &VestingSchedule{CliffSeconds: 180 * 24 * 3600, DurationSeconds: 2 * 365 * 24 * 3600}, "Strategic partners, 6 month cliff then 2 year linear vesting"},
	}

	out := make([]GenesisAccount, 0, len(labels))
	for _, l := range labels {
		addr, _, err := DeriveAccount(seed, l.index)
		if err != nil {
			return nil, err
		}
		entry := GenesisAccount{
			Address: addr.Hex(),
			Balance: pct(l.bps),
			Label:   l.label,
		}
		if l.vesting != nil {
			v := *l.vesting
			v.Beneficiary = addr.Hex()
			entry.Vesting = &v
		}
		out = append(out, entry)
	}
	return out, nil
}

// BootstrapConfig builds a complete, ready-to-run network definition from a
// seed. Validator signer accounts are derived deterministically, so the
// operator can reproduce the exact validator set from the seed alone.
func BootstrapConfig(seed []byte, chainID int64, validatorCount int, totalSupply *big.Int) (*Config, error) {
	if validatorCount <= 0 {
		return nil, ErrNoValidators
	}
	cfg := DefaultConfig()
	if chainID > 0 {
		cfg.ChainID = chainID
	}

	allocs, err := TokenDistribution(totalSupply, seed)
	if err != nil {
		return nil, err
	}
	cfg.Allocations = allocs
	cfg.Treasury = allocs[1].Address

	validators := make([]Validator, 0, validatorCount)
	for i := 0; i < validatorCount; i++ {
		addr, _, err := DeriveAccount(seed, uint32(1000+i))
		if err != nil {
			return nil, err
		}
		payout, _, err := DeriveAccount(seed, uint32(2000+i))
		if err != nil {
			return nil, err
		}
		validators = append(validators, Validator{
			Signer:        addr.Hex(),
			Name:          fmt.Sprintf("validator-%d", i+1),
			Payout:        payout.Hex(),
			CommissionBps: 500,
			SelfStake:     cfg.MinValidatorStake,
		})
	}
	cfg.Validators = validators
	sort.Slice(cfg.Validators, func(i, j int) bool {
		return strings.ToLower(cfg.Validators[i].Signer) < strings.ToLower(cfg.Validators[j].Signer)
	})
	if cfg.GenesisTime == 0 {
		cfg.GenesisTime = GenesisTimestamp
	}
	return cfg, nil
}

// TotalSupply returns the sum of every genesis allocation.
func TotalSupply(cfg *Config) *big.Int {
	total := new(big.Int)
	for _, a := range cfg.Allocations {
		if a.Balance != nil {
			total.Add(total, a.Balance)
		}
	}
	return total
}
