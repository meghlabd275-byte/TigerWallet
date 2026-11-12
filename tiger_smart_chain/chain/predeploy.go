package chain

// predeploy.go — genesis predeploys for TigerSmartChain.
//
// TigerSmartChain ships the platform's own tokens inside the genesis block
// instead of asking an operator to run a deployment script after the chain is
// up. That matters for a wallet platform: every TigerWallet client (web,
// desktop, extension, Android, iOS, Flutter, Rust SDK) already knows the chain
// id and can therefore know the RUSD and WTGR addresses from block 0, with no
// deployment manifest, no per-network address table and no risk of a client
// pointing at an address that was never deployed on the network it is talking
// to.
//
// Two contracts are predeployed:
//
//   RUSD  Royal Tiger USD — the USD-pegged stablecoin, minted only against a
//         reserve held by the treasury.
//   WTGR  Wrapped TGR — the ERC-20 representation of the native TGR coin, so
//         TigerSwap x*y=k pools can quote TGR against RUSD.
//
// The addresses live in the reserved system range 0x0000…0000 – 0x0000…ffff.
// That range is deliberately chosen: it sits outside the EVM precompile
// addresses (0x01–0x11), and no private key is known to produce an address with
// twelve leading zero bytes, so nobody can ever sign a transaction *as* a
// predeploy. The contracts are ordinary EVM contracts that any client can call,
// verify and audit with standard tooling.

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
)

// Predeploy contract addresses. These are fixed by consensus: changing either
// constant changes every genesis hash, so they must never be altered for an
// already-running network.
var (
	// RUSDAddress is the genesis address of the RUSD stablecoin.
	RUSDAddress = common.HexToAddress("0x0000000000000000000000000000000000007710")

	// WTGRAddress is the genesis address of the WTGR wrapped-native token.
	WTGRAddress = common.HexToAddress("0x0000000000000000000000000000000000007711")
)

// Predeploy addresses are only valid while they stay inside the reserved system
// range. The check runs at package load so a careless edit to a constant above
// fails the build instead of silently moving the tokens to an address somebody
// could hold a key for.
func init() {
	for name, addr := range map[string]common.Address{
		"RUSD": RUSDAddress,
		"WTGR": WTGRAddress,
	} {
		for i := 0; i < 12; i++ {
			if addr[i] != 0 {
				panic(fmt.Sprintf("tsc: predeploy %s (%s) is outside the reserved system address range", name, addr.Hex()))
			}
		}
	}
	if RUSDAddress == WTGRAddress {
		panic("tsc: RUSD and WTGR predeploys share an address")
	}
}

// Token metadata written into the predeploy storage.
const (
	// RUSDName is the ERC-20 name of the stablecoin.
	RUSDName = "Royal Tiger USD"

	// RUSDSymbol is the ERC-20 symbol of the stablecoin.
	RUSDSymbol = "RUSD"

	// WTGRName is the ERC-20 name of the wrapped native token.
	WTGRName = "Wrapped TGR"

	// WtgrSymbol is the ERC-20 symbol of the wrapped native token.
	WtgrSymbol = "WTGR"
)

// Predeploy describes one genesis contract in an inspectable form. It is
// exposed so the node RPC and the operator CLI can report what was committed
// without re-deriving it from the bytecode.
type Predeploy struct {
	// Name is the contract name, e.g. "RUSD".
	Name string

	// Address is the deployed address.
	Address common.Address

	// Symbol is the ERC-20 symbol.
	Symbol string

	// Decimals is the ERC-20 decimals.
	Decimals uint8

	// Owner is the address that controls minting and administration.
	Owner common.Address

	// RuntimeBytecodeLen is the size of the committed runtime image.
	RuntimeBytecodeLen int
}

// Predeploys returns the predeploy set for a network definition. The owner of
// both contracts is the network treasury, so minting RUSD requires the same
// key that the protocol already trusts to hold fees.
func Predeploys(cfg *Config) ([]Predeploy, error) {
	if cfg == nil {
		return nil, fmt.Errorf("tsc: nil config")
	}
	treasury, err := NormalizeAddress(cfg.Treasury)
	if err != nil {
		return nil, fmt.Errorf("tsc: predeploys need a treasury owner: %w", err)
	}
	owner := common.HexToAddress(treasury)
	return []Predeploy{
		{
			Name:               "RUSD",
			Address:            RUSDAddress,
			Symbol:             RUSDSymbol,
			Decimals:           Decimals,
			Owner:              owner,
			RuntimeBytecodeLen: len(mustHex(RUSDName, RUSDRuntime)) / 2,
		},
		{
			Name:               "WTGR",
			Address:            WTGRAddress,
			Symbol:             WtgrSymbol,
			Decimals:           Decimals,
			Owner:              owner,
			RuntimeBytecodeLen: len(mustHex(WTGRName, WTGRRuntime)) / 2,
		},
	}, nil
}

// ApplyPredeploys writes the predeploy code and initial storage into a genesis
// allocation set. It is called by BuildGenesis, so a chain built from any
// Config always carries RUSD and WTGR.
func ApplyPredeploys(alloc core.GenesisAlloc, cfg *Config) error {
	list, err := Predeploys(cfg)
	if err != nil {
		return err
	}
	owner := list[0].Owner

	// --- RUSD -------------------------------------------------------------
	rusdCode, err := hexToBytes(RUSDName, RUSDRuntime)
	if err != nil {
		return err
	}
	rusdStorage := map[common.Hash]common.Hash{}
	// slot 2 - totalSupply. Starts at zero: the treasury mints against reserve.
	rusdStorage[slotKey(2)] = common.BigToHash(big.NewInt(0))
	// slot 3 - name.
	rusdStorage[slotKey(3)] = shortStringSlot(RUSDName)
	// slot 4 - symbol.
	rusdStorage[slotKey(4)] = shortStringSlot(RUSDSymbol)
	// slot 5 - owner.
	rusdStorage[slotKey(5)] = common.BytesToHash(owner.Bytes())
	// slot 6 - minters[owner] = true, so the treasury can mint and burn.
	rusdStorage[mappingSlot(owner, 6)] = common.BigToHash(big.NewInt(1))

	alloc[RUSDAddress] = core.GenesisAccount{
		Code:    rusdCode,
		Storage: rusdStorage,
		Balance: new(big.Int),
	}

	// --- WTGR -------------------------------------------------------------
	wtgrCode, err := hexToBytes(WTGRName, WTGRRuntime)
	if err != nil {
		return err
	}
	wtgrStorage := map[common.Hash]common.Hash{}
	// slot 0 - name.
	wtgrStorage[slotKey(0)] = shortStringSlot(WTGRName)
	// slot 1 - symbol.
	wtgrStorage[slotKey(1)] = shortStringSlot(WtgrSymbol)
	// slot 4 - totalSupply. Zero: WTGR only exists once TGR is deposited.
	wtgrStorage[slotKey(4)] = common.BigToHash(big.NewInt(0))
	// slot 5 - owner.
	wtgrStorage[slotKey(5)] = common.BytesToHash(owner.Bytes())

	alloc[WTGRAddress] = core.GenesisAccount{
		Code:    wtgrCode,
		Storage: wtgrStorage,
		Balance: new(big.Int),
	}

	return nil
}

// slotKey returns the storage key of a whole-slot state variable.
func slotKey(slot uint64) common.Hash {
	var buf [32]byte
	binary.BigEndian.PutUint64(buf[24:], slot)
	return common.BytesToHash(buf[:])
}

// mappingSlot returns the storage key of `mapping(address => T)` entry `key` in
// the mapping declared at `slot`. The layout is keccak256(abi.encode(key, slot))
// exactly as the Solidity ABI specification defines it, so the value written
// here is the value the compiled contract reads.
func mappingSlot(key common.Address, slot uint64) common.Hash {
	var buf [64]byte
	copy(buf[12:32], key.Bytes())
	binary.BigEndian.PutUint64(buf[56:], slot)
	return crypto.Keccak256Hash(buf[:])
}

// shortStringSlot encodes a Solidity short string (<= 31 bytes) into a storage
// word: the bytes are left aligned and the final byte holds length*2, which is
// how the compiler marks a short string as opposed to a long-string pointer.
func shortStringSlot(s string) common.Hash {
	var buf [32]byte
	if len(s) > 31 {
		// A long string would need a separate data slot; the predeploy metadata
		// is fixed and short, so this can only fire on a source edit.
		panic("tsc: predeploy metadata string exceeds 31 bytes: " + s)
	}
	copy(buf[:], s)
	buf[31] = byte(len(s) * 2)
	return common.BytesToHash(buf[:])
}

// mustHex decodes a compile-time constant hex string. The bytecode constants are
// generated by cmd/genpredeploy and validated there, so a failure here is a
// build-time defect rather than a runtime condition.
func mustHex(name, hexStr string) []byte {
	b, err := hexToBytes(name, hexStr)
	if err != nil {
		panic(err)
	}
	return b
}

func hexToBytes(name, hexStr string) ([]byte, error) {
	trimmed := strings.TrimPrefix(strings.TrimSpace(hexStr), "0x")
	if trimmed == "" {
		return nil, fmt.Errorf("tsc: %s runtime bytecode is empty", name)
	}
	if len(trimmed)%2 != 0 {
		return nil, fmt.Errorf("tsc: %s runtime bytecode has an odd number of hex digits", name)
	}
	out := make([]byte, len(trimmed)/2)
	for i := 0; i < len(out); i++ {
		hi, ok := hexNibble(trimmed[i*2])
		if !ok {
			return nil, fmt.Errorf("tsc: %s runtime bytecode has a non-hex character", name)
		}
		lo, ok := hexNibble(trimmed[i*2+1])
		if !ok {
			return nil, fmt.Errorf("tsc: %s runtime bytecode has a non-hex character", name)
		}
		out[i] = hi<<4 | lo
	}
	return out, nil
}

func hexNibble(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}
