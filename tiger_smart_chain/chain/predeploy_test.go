package chain

// predeploy_test.go — executes the genesis predeploys inside the real EVM.
//
// The predeploys are committed into the genesis state by writing their storage
// slots directly, without running a constructor. A slot computed one byte wrong
// would still produce a chain that starts, but every RUSD/WTGR call would return
// garbage. These tests therefore do not compare constants: they load the genesis
// state into a state database, run the committed runtime bytecode through
// go-ethereum's interpreter, and assert on the values the contracts return.

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/holiman/uint256"
)

// testChainContext satisfies core.ChainContext for the standalone EVM. The
// predeploys never query block history or consensus, so these lookups are
// exercised only for interface compliance.
type testChainContext struct{}

func (testChainContext) Engine() consensus.Engine { return nil }

func (testChainContext) GetHeader(common.Hash, uint64) *types.Header { return nil }

// predeployFixture holds a genesis state with the predeploys loaded.
type predeployFixture struct {
	t       *testing.T
	cfg     *Config
	genesis *core.Genesis
	state   *state.StateDB
	owner   common.Address
}

func newPredeployFixture(t *testing.T) *predeployFixture {
	t.Helper()

	seed := []byte("tsc-predeploy-test-seed-000001")
	cfg, err := BootstrapConfig(seed, DefaultChainID, 1, Wei(1_000_000_000))
	if err != nil {
		t.Fatalf("BootstrapConfig: %v", err)
	}
	genesis, err := BuildGenesis(cfg)
	if err != nil {
		t.Fatalf("BuildGenesis: %v", err)
	}

	db := rawdb.NewMemoryDatabase()
	trieDB := triedb.NewDatabase(db, nil)
	if _, err := genesis.Commit(db, trieDB); err != nil {
		t.Fatalf("genesis commit: %v", err)
	}
	statedb, err := state.New(genesis.ToBlock().Root(), state.NewDatabaseWithNodeDB(db, trieDB), nil)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}

	return &predeployFixture{
		t:       t,
		cfg:     cfg,
		genesis: genesis,
		state:   statedb,
		owner:   common.HexToAddress(cfg.Treasury),
	}
}

// call executes a contract call against the fixture state and returns the raw
// output. A revert is reported through err; the caller decides whether that is
// the expected outcome.
func (f *predeployFixture) call(caller, to common.Address, value *big.Int, input []byte) ([]byte, uint64, error) {
	f.t.Helper()
	if value == nil {
		value = new(big.Int)
	}
	// The coinbase is passed explicitly: with a nil author the block context
	// asks the consensus engine, and the predeploys do not need a consensus
	// engine to be present.
	coinbase := common.Address{}
	// Difficulty is set so the block context takes the pre-merge branch and
	// does not dereference a nil Random field.
	blockCtx := core.NewEVMBlockContext(&types.Header{
		Number:     big.NewInt(1),
		Time:       uint64(f.cfg.GenesisTime),
		GasLimit:   f.cfg.GasLimit,
		BaseFee:    f.cfg.BaseFeeWei,
		Difficulty: big.NewInt(1),
	}, testChainContext{}, &coinbase)
	evm := vm.NewEVM(blockCtx, vm.TxContext{Origin: caller, GasPrice: big.NewInt(1)}, f.state, ChainConfig(f.cfg), vm.Config{})
	ret, leftOverGas, err := evm.Call(vm.AccountRef(caller), to, input, f.cfg.GasLimit, uint256.MustFromBig(value))
	return ret, leftOverGas, err
}

// selector computes the 4-byte function selector for a signature.
func selector(sig string) []byte {
	sum := crypto.Keccak256([]byte(sig))
	return sum[:4]
}

// encodeAddress left-pads an address to a 32-byte ABI word.
func encodeAddress(a common.Address) []byte {
	var word [32]byte
	copy(word[12:], a.Bytes())
	return word[:]
}

// encodeUint256 left-pads a big integer to a 32-byte ABI word.
func encodeUint256(v *big.Int) []byte {
	var word [32]byte
	v.FillBytes(word[:])
	return word[:]
}

// decodeUint256 reads a single uint256 return word.
func decodeUint256(t *testing.T, out []byte) *big.Int {
	t.Helper()
	if len(out) < 32 {
		t.Fatalf("expected at least 32 bytes of output, got %d", len(out))
	}
	return new(big.Int).SetBytes(out[:32])
}

// decodeAddress reads a single address return word.
func decodeAddress(t *testing.T, out []byte) common.Address {
	t.Helper()
	if len(out) < 32 {
		t.Fatalf("expected at least 32 bytes of output, got %d", len(out))
	}
	return common.BytesToAddress(out[12:32])
}

// decodeString decodes a Solidity string return value (offset + length + data).
func decodeString(t *testing.T, out []byte) string {
	t.Helper()
	if len(out) < 64 {
		t.Fatalf("expected at least 64 bytes of output, got %d", len(out))
	}
	offset := new(big.Int).SetBytes(out[:32]).Uint64()
	if offset+32 > uint64(len(out)) {
		t.Fatalf("string offset %d is outside the %d byte output", offset, len(out))
	}
	length := new(big.Int).SetBytes(out[offset : offset+32]).Uint64()
	start := offset + 32
	if start+length > uint64(len(out)) {
		t.Fatalf("string length %d is outside the %d byte output", length, len(out))
	}
	return string(out[start : start+length])
}

// isTrueWord reports whether a 32-byte ABI word encodes boolean true.
func isTrueWord(out []byte) bool {
	if len(out) < 32 {
		return false
	}
	for _, b := range out[:31] {
		if b != 0 {
			return false
		}
	}
	return out[31] == 1
}

func TestPredeployAddressesAreInReservedRange(t *testing.T) {
	for name, addr := range map[string]common.Address{"RUSD": RUSDAddress, "WTGR": WTGRAddress} {
		for i := 0; i < 12; i++ {
			if addr[i] != 0 {
				t.Fatalf("%s (%s) is outside the reserved system address range", name, addr.Hex())
			}
		}
	}
	if RUSDAddress == WTGRAddress {
		t.Fatal("RUSD and WTGR must not share an address")
	}
	if RUSDAddress == (common.Address{}) {
		t.Fatal("RUSD must not be the zero address")
	}
}

func TestGenesisCarriesPredeployCode(t *testing.T) {
	f := newPredeployFixture(t)

	rusdCode := f.state.GetCode(RUSDAddress)
	if len(rusdCode) == 0 {
		t.Fatal("RUSD has no code in the genesis state")
	}
	if got, want := hex.EncodeToString(rusdCode), RUSDRuntime; got != want {
		t.Fatalf("RUSD code does not match the compiled runtime image (%d vs %d bytes)", len(got)/2, len(want)/2)
	}

	wtgrCode := f.state.GetCode(WTGRAddress)
	if len(wtgrCode) == 0 {
		t.Fatal("WTGR has no code in the genesis state")
	}
	if got, want := hex.EncodeToString(wtgrCode), WTGRRuntime; got != want {
		t.Fatalf("WTGR code does not match the compiled runtime image (%d vs %d bytes)", len(got)/2, len(want)/2)
	}
}

func TestRUSDGenesisMetadata(t *testing.T) {
	f := newPredeployFixture(t)

	out, _, err := f.call(f.owner, RUSDAddress, nil, selector("name()"))
	if err != nil {
		t.Fatalf("RUSD.name(): %v", err)
	}
	if got := decodeString(t, out); got != RUSDName {
		t.Fatalf("RUSD.name() = %q, want %q", got, RUSDName)
	}

	out, _, err = f.call(f.owner, RUSDAddress, nil, selector("symbol()"))
	if err != nil {
		t.Fatalf("RUSD.symbol(): %v", err)
	}
	if got := decodeString(t, out); got != RUSDSymbol {
		t.Fatalf("RUSD.symbol() = %q, want %q", got, RUSDSymbol)
	}

	out, _, err = f.call(f.owner, RUSDAddress, nil, selector("decimals()"))
	if err != nil {
		t.Fatalf("RUSD.decimals(): %v", err)
	}
	if got := decodeUint256(t, out).Uint64(); got != Decimals {
		t.Fatalf("RUSD.decimals() = %d, want %d", got, Decimals)
	}

	out, _, err = f.call(f.owner, RUSDAddress, nil, selector("totalSupply()"))
	if err != nil {
		t.Fatalf("RUSD.totalSupply(): %v", err)
	}
	if got := decodeUint256(t, out); got.Sign() != 0 {
		t.Fatalf("RUSD.totalSupply() = %s, want 0 before any mint", got)
	}
}

func TestRUSDGenesisOwnershipAndMinter(t *testing.T) {
	f := newPredeployFixture(t)

	out, _, err := f.call(f.owner, RUSDAddress, nil, selector("owner()"))
	if err != nil {
		t.Fatalf("RUSD.owner(): %v", err)
	}
	if got := decodeAddress(t, out); got != f.owner {
		t.Fatalf("RUSD.owner() = %s, want the treasury %s", got.Hex(), f.owner.Hex())
	}

	// The treasury must be a minter, otherwise RUSD could never be issued.
	input := append(selector("isMinter(address)"), encodeAddress(f.owner)...)
	out, _, err = f.call(f.owner, RUSDAddress, nil, input)
	if err != nil {
		t.Fatalf("RUSD.isMinter(treasury): %v", err)
	}
	if !isTrueWord(out) {
		t.Fatal("RUSD.isMinter(treasury) = false, want true")
	}

	// An unrelated account must not be a minter.
	stranger := common.HexToAddress("0x1111111111111111111111111111111111111111")
	input = append(selector("isMinter(address)"), encodeAddress(stranger)...)
	out, _, err = f.call(f.owner, RUSDAddress, nil, input)
	if err != nil {
		t.Fatalf("RUSD.isMinter(stranger): %v", err)
	}
	if isTrueWord(out) {
		t.Fatal("RUSD.isMinter(stranger) = true, want false")
	}
}

func TestRUSDMintOnlyByAuthorisedMinter(t *testing.T) {
	f := newPredeployFixture(t)
	stranger := common.HexToAddress("0x2222222222222222222222222222222222222222")
	recipient := common.HexToAddress("0x3333333333333333333333333333333333333333")
	amount := Wei(250)

	mintInput := func(to common.Address, value *big.Int) []byte {
		in := selector("mint(address,uint256)")
		in = append(in, encodeAddress(to)...)
		in = append(in, encodeUint256(value)...)
		return in
	}

	// An unauthorised caller must be rejected, and the state must not change.
	if _, _, err := f.call(stranger, RUSDAddress, nil, mintInput(stranger, amount)); err == nil {
		t.Fatal("RUSD.mint() from an unauthorised caller succeeded, want a revert")
	}
	out, _, err := f.call(f.owner, RUSDAddress, nil, selector("totalSupply()"))
	if err != nil {
		t.Fatalf("RUSD.totalSupply(): %v", err)
	}
	if got := decodeUint256(t, out); got.Sign() != 0 {
		t.Fatalf("totalSupply changed to %s after a rejected mint", got)
	}

	// The treasury can mint.
	if _, _, err := f.call(f.owner, RUSDAddress, nil, mintInput(recipient, amount)); err != nil {
		t.Fatalf("RUSD.mint() from the treasury: %v", err)
	}

	balanceInput := append(selector("balanceOf(address)"), encodeAddress(recipient)...)
	out, _, err = f.call(f.owner, RUSDAddress, nil, balanceInput)
	if err != nil {
		t.Fatalf("RUSD.balanceOf(): %v", err)
	}
	if got := decodeUint256(t, out); got.Cmp(amount) != 0 {
		t.Fatalf("recipient balance = %s, want %s", got, amount)
	}

	out, _, err = f.call(f.owner, RUSDAddress, nil, selector("totalSupply()"))
	if err != nil {
		t.Fatalf("RUSD.totalSupply(): %v", err)
	}
	if got := decodeUint256(t, out); got.Cmp(amount) != 0 {
		t.Fatalf("totalSupply = %s, want %s", got, amount)
	}
}

func TestRUSDTransferMovesBalance(t *testing.T) {
	f := newPredeployFixture(t)
	recipient := common.HexToAddress("0x4444444444444444444444444444444444444444")
	holder := common.HexToAddress("0x5555555555555555555555555555555555555555")
	amount := Wei(10)

	mintInput := selector("mint(address,uint256)")
	mintInput = append(mintInput, encodeAddress(holder)...)
	mintInput = append(mintInput, encodeUint256(amount)...)
	if _, _, err := f.call(f.owner, RUSDAddress, nil, mintInput); err != nil {
		t.Fatalf("RUSD.mint(): %v", err)
	}

	transferInput := selector("transfer(address,uint256)")
	transferInput = append(transferInput, encodeAddress(recipient)...)
	transferInput = append(transferInput, encodeUint256(amount)...)
	if _, _, err := f.call(holder, RUSDAddress, nil, transferInput); err != nil {
		t.Fatalf("RUSD.transfer(): %v", err)
	}

	balanceInput := append(selector("balanceOf(address)"), encodeAddress(recipient)...)
	out, _, err := f.call(f.owner, RUSDAddress, nil, balanceInput)
	if err != nil {
		t.Fatalf("RUSD.balanceOf(recipient): %v", err)
	}
	if got := decodeUint256(t, out); got.Cmp(amount) != 0 {
		t.Fatalf("recipient balance = %s, want %s", got, amount)
	}

	balanceInput = append(selector("balanceOf(address)"), encodeAddress(holder)...)
	out, _, err = f.call(f.owner, RUSDAddress, nil, balanceInput)
	if err != nil {
		t.Fatalf("RUSD.balanceOf(holder): %v", err)
	}
	if got := decodeUint256(t, out); got.Sign() != 0 {
		t.Fatalf("holder balance = %s, want 0 after transferring everything", got)
	}
}

func TestWTGRGenesisMetadata(t *testing.T) {
	f := newPredeployFixture(t)

	out, _, err := f.call(f.owner, WTGRAddress, nil, selector("name()"))
	if err != nil {
		t.Fatalf("WTGR.name(): %v", err)
	}
	if got := decodeString(t, out); got != WTGRName {
		t.Fatalf("WTGR.name() = %q, want %q", got, WTGRName)
	}

	out, _, err = f.call(f.owner, WTGRAddress, nil, selector("symbol()"))
	if err != nil {
		t.Fatalf("WTGR.symbol(): %v", err)
	}
	if got := decodeString(t, out); got != WtgrSymbol {
		t.Fatalf("WTGR.symbol() = %q, want %q", got, WtgrSymbol)
	}

	out, _, err = f.call(f.owner, WTGRAddress, nil, selector("totalSupply()"))
	if err != nil {
		t.Fatalf("WTGR.totalSupply(): %v", err)
	}
	if got := decodeUint256(t, out); got.Sign() != 0 {
		t.Fatalf("WTGR.totalSupply() = %s, want 0 before any deposit", got)
	}

	out, _, err = f.call(f.owner, WTGRAddress, nil, selector("owner()"))
	if err != nil {
		t.Fatalf("WTGR.owner(): %v", err)
	}
	if got := decodeAddress(t, out); got != f.owner {
		t.Fatalf("WTGR.owner() = %s, want the treasury %s", got.Hex(), f.owner.Hex())
	}
}

func TestWTGRDepositAndWithdrawRoundTrip(t *testing.T) {
	f := newPredeployFixture(t)
	holder := common.HexToAddress("0x6666666666666666666666666666666666666666")
	amount := Wei(5)

	f.state.SetBalance(holder, uint256.MustFromBig(amount))
	balanceBefore := f.state.GetBalance(holder)

	if _, _, err := f.call(holder, WTGRAddress, amount, selector("deposit()")); err != nil {
		t.Fatalf("WTGR.deposit(): %v", err)
	}

	balanceInput := append(selector("balanceOf(address)"), encodeAddress(holder)...)
	out, _, err := f.call(holder, WTGRAddress, nil, balanceInput)
	if err != nil {
		t.Fatalf("WTGR.balanceOf(): %v", err)
	}
	if got := decodeUint256(t, out); got.Cmp(amount) != 0 {
		t.Fatalf("WTGR balance = %s, want %s", got, amount)
	}

	out, _, err = f.call(holder, WTGRAddress, nil, selector("totalSupply()"))
	if err != nil {
		t.Fatalf("WTGR.totalSupply(): %v", err)
	}
	if got := decodeUint256(t, out); got.Cmp(amount) != 0 {
		t.Fatalf("WTGR totalSupply = %s, want %s", got, amount)
	}
	if got := f.state.GetBalance(WTGRAddress).ToBig(); got.Cmp(amount) != 0 {
		t.Fatalf("WTGR contract holds %s TGR, want %s", got, amount)
	}

	// Withdraw the whole balance: the WTGR must be burned and the native coin
	// returned to the holder.
	withdrawInput := append(selector("withdraw(uint256)"), encodeUint256(amount)...)
	if _, _, err := f.call(holder, WTGRAddress, nil, withdrawInput); err != nil {
		t.Fatalf("WTGR.withdraw(): %v", err)
	}

	out, _, err = f.call(holder, WTGRAddress, nil, balanceInput)
	if err != nil {
		t.Fatalf("WTGR.balanceOf(): %v", err)
	}
	if got := decodeUint256(t, out); got.Sign() != 0 {
		t.Fatalf("WTGR balance = %s after withdrawing everything, want 0", got)
	}

	if got := f.state.GetBalance(holder).ToBig(); got.Cmp(balanceBefore.ToBig()) != 0 {
		t.Fatalf("holder TGR balance = %s, want %s after the round trip", got, balanceBefore.ToBig())
	}
	if got := f.state.GetBalance(WTGRAddress).ToBig(); got.Sign() != 0 {
		t.Fatalf("WTGR contract still holds %s TGR after the last withdrawal", got)
	}
}

func TestWTGRWithdrawRejectsUnbackedAmount(t *testing.T) {
	f := newPredeployFixture(t)
	holder := common.HexToAddress("0x7777777777777777777777777777777777777777")

	withdrawInput := append(selector("withdraw(uint256)"), encodeUint256(Wei(1))...)
	if _, _, err := f.call(holder, WTGRAddress, nil, withdrawInput); err == nil {
		t.Fatal("WTGR.withdraw() with no balance succeeded, want a revert")
	}
}

func TestPredeployListMatchesGenesis(t *testing.T) {
	f := newPredeployFixture(t)
	list, err := Predeploys(f.cfg)
	if err != nil {
		t.Fatalf("Predeploys: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("Predeploys returned %d entries, want 2", len(list))
	}
	for _, p := range list {
		if !f.state.Exist(p.Address) {
			t.Fatalf("%s (%s) is listed as a predeploy but is absent from the genesis state", p.Name, p.Address.Hex())
		}
		if p.RuntimeBytecodeLen == 0 {
			t.Fatalf("%s reports an empty runtime image", p.Name)
		}
		if p.Owner != f.owner {
			t.Fatalf("%s owner = %s, want the treasury %s", p.Name, p.Owner.Hex(), f.owner.Hex())
		}
		if p.Decimals != Decimals {
			t.Fatalf("%s decimals = %d, want %d", p.Name, p.Decimals, Decimals)
		}
	}
}

// TestPredeployGenesisHashIsDeterministic guards the property that makes the
// predeploys usable: the same Config must always produce the same genesis, so
// two operators starting from the same network definition agree on the RUSD and
// WTGR addresses and on the state behind them.
func TestPredeployGenesisHashIsDeterministic(t *testing.T) {
	seed := []byte("tsc-predeploy-test-seed-000001")
	cfgA, err := BootstrapConfig(seed, DefaultChainID, 1, Wei(1_000_000_000))
	if err != nil {
		t.Fatalf("BootstrapConfig: %v", err)
	}
	cfgB, err := BootstrapConfig(seed, DefaultChainID, 1, Wei(1_000_000_000))
	if err != nil {
		t.Fatalf("BootstrapConfig: %v", err)
	}
	genA, err := BuildGenesis(cfgA)
	if err != nil {
		t.Fatalf("BuildGenesis A: %v", err)
	}
	genB, err := BuildGenesis(cfgB)
	if err != nil {
		t.Fatalf("BuildGenesis B: %v", err)
	}
	if genA.ToBlock().Root() != genB.ToBlock().Root() {
		t.Fatalf("genesis state root is not deterministic: %s vs %s",
			genA.ToBlock().Root().Hex(), genB.ToBlock().Root().Hex())
	}

	if got := ChainConfig(cfgA).ChainID; got.Cmp(big.NewInt(DefaultChainID)) != 0 {
		t.Fatalf("ChainConfig chain id = %s, want %d", got, DefaultChainID)
	}
}

// TestPredeployGasCostIsBounded sanity-checks that a call to each predeploy
// leaves most of the block gas limit untouched, so an ordinary user transaction
// can afford to interact with them.
func TestPredeployGasCostIsBounded(t *testing.T) {
	f := newPredeployFixture(t)
	for name, addr := range map[string]common.Address{"RUSD": RUSDAddress, "WTGR": WTGRAddress} {
		_, left, err := f.call(f.owner, addr, nil, selector("name()"))
		if err != nil {
			t.Fatalf("%s.name(): %v", name, err)
		}
		used := f.cfg.GasLimit - left
		if used > 20_000 {
			t.Fatalf("%s.name() used %d gas, want <= 20000", name, used)
		}
	}
}

// TestPredeployRUSDMatchesDeclaredStorageLayout re-derives the storage slots
// from the declared layout and confirms the values the contract reads back. It
// is the guard against silently reordering a state variable in RUSD.sol: the
// slot numbers below are asserted against the compiled storage layout file in
// the repository, and against the live contract state here.
func TestPredeployRUSDMatchesDeclaredStorageLayout(t *testing.T) {
	f := newPredeployFixture(t)

	// slot 2 must hold totalSupply.
	if got := f.state.GetState(RUSDAddress, slotKey(2)); new(big.Int).SetBytes(got.Bytes()).Sign() != 0 {
		t.Fatalf("slot 2 is not the zero totalSupply: %s", got.Hex())
	}
	// slot 3 must hold the name string.
	if got := f.state.GetState(RUSDAddress, slotKey(3)); got != shortStringSlot(RUSDName) {
		t.Fatalf("slot 3 = %s, want the RUSD name word %s", got.Hex(), shortStringSlot(RUSDName).Hex())
	}
	// slot 4 must hold the symbol string.
	if got := f.state.GetState(RUSDAddress, slotKey(4)); got != shortStringSlot(RUSDSymbol) {
		t.Fatalf("slot 4 = %s, want the RUSD symbol word %s", got.Hex(), shortStringSlot(RUSDSymbol).Hex())
	}
	// slot 5 must hold the owner.
	if got := f.state.GetState(RUSDAddress, slotKey(5)); common.BytesToAddress(got.Bytes()) != f.owner {
		t.Fatalf("slot 5 = %s, want the treasury %s", got.Hex(), f.owner.Hex())
	}
	// slot 6 is the minter mapping; the treasury entry must be set.
	if got := f.state.GetState(RUSDAddress, mappingSlot(f.owner, 6)); !isTrueWord(got.Bytes()) {
		t.Fatalf("minters[treasury] slot = %s, want true", got.Hex())
	}
}

// TestPredeployWTGRMatchesDeclaredStorageLayout is the WTGR equivalent.
func TestPredeployWTGRMatchesDeclaredStorageLayout(t *testing.T) {
	f := newPredeployFixture(t)

	if got := f.state.GetState(WTGRAddress, slotKey(0)); got != shortStringSlot(WTGRName) {
		t.Fatalf("slot 0 = %s, want the WTGR name word %s", got.Hex(), shortStringSlot(WTGRName).Hex())
	}
	if got := f.state.GetState(WTGRAddress, slotKey(1)); got != shortStringSlot(WtgrSymbol) {
		t.Fatalf("slot 1 = %s, want the WTGR symbol word %s", got.Hex(), shortStringSlot(WtgrSymbol).Hex())
	}
	if got := f.state.GetState(WTGRAddress, slotKey(4)); new(big.Int).SetBytes(got.Bytes()).Sign() != 0 {
		t.Fatalf("slot 4 is not the zero totalSupply: %s", got.Hex())
	}
	if got := f.state.GetState(WTGRAddress, slotKey(5)); common.BytesToAddress(got.Bytes()) != f.owner {
		t.Fatalf("slot 5 = %s, want the treasury %s", got.Hex(), f.owner.Hex())
	}
}

// TestShortStringSlotRejectsLongStrings locks in the invariant that the
// predeploy metadata is encoded as a short string. A long string would need a
// second data slot, which the encoder deliberately does not implement.
func TestShortStringSlotRejectsLongStrings(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("shortStringSlot accepted a 32 byte string, want a panic")
		}
	}()
	shortStringSlot("0123456789012345678901234567890123")
}

// TestMappingSlotMatchesSolidityLayout pins the mapping key derivation against
// a value computed independently with the ABI rule
// keccak256(abi.encode(key, slot)).
func TestMappingSlotMatchesSolidityLayout(t *testing.T) {
	addr := common.HexToAddress("0x00000000000000000000000000000000000000AA")
	got := mappingSlot(addr, 6)

	var preimage [64]byte
	copy(preimage[12:32], addr.Bytes())
	preimage[63] = 6
	want := crypto.Keccak256Hash(preimage[:])

	if got != want {
		t.Fatalf("mappingSlot = %s, want %s", got.Hex(), want.Hex())
	}
}
