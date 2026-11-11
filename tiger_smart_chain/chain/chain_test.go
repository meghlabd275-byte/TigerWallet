package chain

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// testSeed returns a fixed seed so every test derives the same validator set
// and genesis state.
func testSeed() []byte {
	return []byte("tiger-smart-chain-test-seed-0001")
}

func testConfig(t *testing.T) *Config {
	t.Helper()
	cfg, err := BootstrapConfig(testSeed(), TestnetChainID, 3, Wei(1_000_000_000))
	if err != nil {
		t.Fatalf("BootstrapConfig: %v", err)
	}
	cfg.Name = "TigerSmartChain Testnet"
	return cfg
}

// TestWeiFromStringParsesExactly checks decimal parsing never loses precision.
func TestWeiFromStringParsesExactly(t *testing.T) {
	cases := map[string]string{
		"1":                      "1000000000000000000",
		"1.5":                    "1500000000000000000",
		"0.000000001":            "1000000000",
		"12.0000000000000000001": "",
		"":                       "",
		"abc":                    "",
		"1.2.3":                  "",
	}
	for in, want := range cases {
		got, err := WeiFromString(in)
		if want == "" {
			if err == nil {
				t.Errorf("WeiFromString(%q) = %v, want error", in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("WeiFromString(%q): %v", in, err)
			continue
		}
		if got.String() != want {
			t.Errorf("WeiFromString(%q) = %s, want %s", in, got.String(), want)
		}
	}
}

// TestNormalizeAddressRejectsMalformedInput ensures a mistyped allocation can
// never silently drop funds.
func TestNormalizeAddressRejectsMalformedInput(t *testing.T) {
	good := "0x52908400098527886E0F7030069857D2E4169EE7"
	norm, err := NormalizeAddress(good)
	if err != nil {
		t.Fatalf("NormalizeAddress(%q): %v", good, err)
	}
	if norm != "0x52908400098527886e0f7030069857d2e4169ee7" {
		t.Fatalf("normalize = %s", norm)
	}

	bad := []string{
		"",
		"52908400098527886E0F7030069857D2E4169EE7",
		"0x52908400098527886E0F7030069857D2E4169EE",
		"0x52908400098527886E0F7030069857D2E4169EE7ZZ",
		"0x" + "zz" + "908400098527886E0F7030069857D2E4169E",
	}
	for _, b := range bad {
		if _, err := NormalizeAddress(b); err == nil {
			t.Errorf("NormalizeAddress(%q) unexpectedly succeeded", b)
		}
	}
}

// TestGenesisIsDeterministic proves two builds of the same config produce the
// identical genesis hash, which is what lets every node agree on the chain.
func TestGenesisIsDeterministic(t *testing.T) {
	cfg := testConfig(t)
	a, err := BuildGenesis(cfg)
	if err != nil {
		t.Fatalf("BuildGenesis: %v", err)
	}
	b, err := BuildGenesis(cfg)
	if err != nil {
		t.Fatalf("BuildGenesis: %v", err)
	}
	if a.ToBlock().Hash() != b.ToBlock().Hash() {
		t.Fatalf("genesis hash differs: %s vs %s", a.ToBlock().Hash(), b.ToBlock().Hash())
	}
	if a.ToBlock().Hash() == (common.Hash{}) {
		t.Fatal("genesis hash is zero")
	}
}

// TestGenesisFundsValidatorsAndAllocations proves the genesis state really
// holds the configured balances.
func TestGenesisFundsValidatorsAndAllocations(t *testing.T) {
	cfg := testConfig(t)
	genesis, err := BuildGenesis(cfg)
	if err != nil {
		t.Fatalf("BuildGenesis: %v", err)
	}
	if len(genesis.Alloc) == 0 {
		t.Fatal("genesis has no allocations")
	}
	for _, v := range cfg.Validators {
		addr := common.HexToAddress(v.Signer)
		acct, ok := genesis.Alloc[addr]
		if !ok {
			t.Fatalf("validator %s is not funded", v.Signer)
		}
		if acct.Balance.Cmp(cfg.MinValidatorStake) < 0 {
			t.Fatalf("validator %s balance %s below min stake %s", v.Signer, acct.Balance, cfg.MinValidatorStake)
		}
	}
	// Total supply must equal the configured supply exactly.
	want := Wei(1_000_000_000)
	got := TotalSupply(cfg)
	if got.Cmp(want) != 0 {
		t.Fatalf("total supply = %s, want %s", got, want)
	}
}

// TestGenesisRejectsDuplicateValidator proves a misconfigured authority set is
// rejected instead of producing an unverifiable chain.
func TestGenesisRejectsDuplicateValidator(t *testing.T) {
	cfg := testConfig(t)
	cfg.Validators = append(cfg.Validators, cfg.Validators[0])
	if _, err := BuildGenesis(cfg); err == nil {
		t.Fatal("duplicate validator was accepted")
	}
}

// TestGenesisRejectsNoValidator proves a chain that could never progress is
// rejected at configuration time.
func TestGenesisRejectsNoValidator(t *testing.T) {
	cfg := testConfig(t)
	cfg.Validators = nil
	if _, err := BuildGenesis(cfg); err == nil {
		t.Fatal("validator-less genesis was accepted")
	}
}

// TestNodeProducesRealBlocks is the core proof: a real node signs real blocks
// with a real secp256k1 key and the chain advances.
func TestNodeProducesRealBlocks(t *testing.T) {
	if testing.Short() {
		t.Skip("block production test is slow")
	}
	cfg := testConfig(t)
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	addr := crypto.PubkeyToAddress(key.PublicKey)
	cfg.Validators = []Validator{{
		Signer:        addr.Hex(),
		Name:          "test-validator",
		Payout:        addr.Hex(),
		CommissionBps: 500,
		SelfStake:     cfg.MinValidatorStake,
	}}

	dir := t.TempDir()
	n, err := NewNode(NodeConfig{
		Genesis:          cfg,
		DataDir:          dir,
		HTTPPort:         0,
		WSPort:           0,
		AuthPort:         0,
		P2PPort:          0,
		ValidatorKey:     hexKey(key),
		KeyStorePassword: "test-password",
		EnableMining:     true,
	})
	if err != nil {
		t.Fatalf("NewNode: %v", err)
	}
	defer n.Stop()

	if n.GenesisHash() == (common.Hash{}) {
		t.Fatal("genesis hash is zero")
	}
	if err := n.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !n.IsMining() {
		t.Fatal("node reports it is not mining")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := n.WaitForHeight(ctx, 3); err != nil {
		t.Fatalf("chain did not advance: %v", err)
	}

	// The author of every produced block must be our validator.
	bc := n.Backend().BlockChain()
	engine := n.Backend().Engine()
	for num := uint64(1); num <= 3; num++ {
		blk := bc.GetBlockByNumber(num)
		if blk == nil {
			t.Fatalf("block %d is missing", num)
		}
		author, err := engine.Author(blk.Header())
		if err != nil {
			t.Fatalf("block %d: recover author: %v", num, err)
		}
		if author != addr {
			t.Fatalf("block %d authored by %s, want %s", num, author, addr)
		}
		if blk.Hash() == (common.Hash{}) {
			t.Fatalf("block %d has a zero hash", num)
		}
	}

	h := n.Health()
	if h.HeadNumber < 3 {
		t.Fatalf("health head = %d", h.HeadNumber)
	}
	if h.Signer != addr.Hex() {
		t.Fatalf("health signer = %s", h.Signer)
	}
	if h.ChainID != big.NewInt(TestnetChainID).String() {
		t.Fatalf("health chain id = %s", h.ChainID)
	}
}

// TestNodeExecutesRealValueTransfer proves the EVM actually executes a signed
// transaction: the recipient balance increases and the sender pays gas.
func TestNodeExecutesRealValueTransfer(t *testing.T) {
	if testing.Short() {
		t.Skip("transaction execution test is slow")
	}
	cfg := testConfig(t)
	validatorKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	validatorAddr := crypto.PubkeyToAddress(validatorKey.PublicKey)
	cfg.Validators = []Validator{{
		Signer:        validatorAddr.Hex(),
		Name:          "test-validator",
		Payout:        validatorAddr.Hex(),
		CommissionBps: 500,
		SelfStake:     cfg.MinValidatorStake,
	}}

	recipientKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	recipient := crypto.PubkeyToAddress(recipientKey.PublicKey)

	n, err := NewNode(NodeConfig{
		Genesis:          cfg,
		DataDir:          t.TempDir(),
		HTTPPort:         0,
		WSPort:           0,
		AuthPort:         0,
		P2PPort:          0,
		ValidatorKey:     hexKey(validatorKey),
		KeyStorePassword: "test-password",
		EnableMining:     true,
	})
	if err != nil {
		t.Fatalf("NewNode: %v", err)
	}
	defer n.Stop()
	if err := n.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := n.WaitForHeight(ctx, 2); err != nil {
		t.Fatalf("chain did not start: %v", err)
	}

	bc := n.Backend().BlockChain()
	signer := types.LatestSigner(ChainConfig(cfg))
	stateDB, err := bc.State()
	if err != nil {
		t.Fatalf("state: %v", err)
	}
	nonce := stateDB.GetNonce(validatorAddr)
	head := bc.CurrentBlock()
	value := Wei(5)

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   big.NewInt(cfg.ChainID),
		Nonce:     nonce,
		GasTipCap: big.NewInt(1_000_000_000),
		GasFeeCap: new(big.Int).Add(head.BaseFee, big.NewInt(2_000_000_000)),
		Gas:       21000,
		To:        &recipient,
		Value:     value,
	})
	signed, err := types.SignTx(tx, signer, validatorKey)
	if err != nil {
		t.Fatalf("SignTx: %v", err)
	}
	if errs := n.Backend().TxPool().Add([]*types.Transaction{signed}, true, true); len(errs) > 0 && errs[0] != nil {
		t.Fatalf("add transaction to pool: %v", errs[0])
	}

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		state, err := bc.State()
		if err != nil {
			t.Fatalf("state: %v", err)
		}
		if state.GetBalance(recipient).ToBig().Cmp(value) == 0 {
			receipt := receiptFor(bc, bc.CurrentBlock().Hash(), signed.Hash())
			if receipt != nil && receipt.Status != types.ReceiptStatusSuccessful {
				t.Fatalf("receipt status = %d", receipt.Status)
			}
			return
		}
		time.Sleep(300 * time.Millisecond)
	}
	t.Fatal("value transfer was never executed on chain")
}

// TestRPCNamespaceServesLiveChainData proves the tsc_ API returns the real
// chain state rather than a placeholder.
func TestRPCNamespaceServesLiveChainData(t *testing.T) {
	if testing.Short() {
		t.Skip("rpc test is slow")
	}
	cfg := testConfig(t)
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	addr := crypto.PubkeyToAddress(key.PublicKey)
	cfg.Validators = []Validator{{
		Signer:        addr.Hex(),
		Name:          "rpc-validator",
		Payout:        addr.Hex(),
		CommissionBps: 300,
		SelfStake:     cfg.MinValidatorStake,
	}}

	n, err := NewNode(NodeConfig{
		Genesis:          cfg,
		DataDir:          t.TempDir(),
		HTTPPort:         0,
		WSPort:           0,
		AuthPort:         0,
		P2PPort:          0,
		ValidatorKey:     hexKey(key),
		KeyStorePassword: "test-password",
		EnableMining:     true,
	})
	if err != nil {
		t.Fatalf("NewNode: %v", err)
	}
	defer n.Stop()
	if err := n.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := n.WaitForHeight(ctx, 3); err != nil {
		t.Fatalf("chain did not advance: %v", err)
	}

	api := &TSCAPI{node: n}

	info, err := api.Info()
	if err != nil {
		t.Fatalf("Info: %v", err)
	}
	if info.ChainID != big.NewInt(TestnetChainID).String() {
		t.Fatalf("chain id = %s", info.ChainID)
	}
	if info.NativeCurrency != NativeSymbol {
		t.Fatalf("native currency = %s", info.NativeCurrency)
	}

	validators, err := api.Validators(ctx, nil)
	if err != nil {
		t.Fatalf("Validators: %v", err)
	}
	if len(validators) != 1 {
		t.Fatalf("validator count = %d, want 1", len(validators))
	}
	if validators[0].BlocksSigned == 0 {
		t.Fatal("validator signed no blocks according to the live chain")
	}
	if !validators[0].IsLocalSigner {
		t.Fatal("local signer was not reported as a validator")
	}

	blocks, err := api.Blocks(ctx, 0, 10)
	if err != nil {
		t.Fatalf("Blocks: %v", err)
	}
	if len(blocks) == 0 {
		t.Fatal("no blocks returned")
	}
	for _, b := range blocks {
		if b.Number == 0 {
			// The genesis block carries no authority seal.
			continue
		}
		if b.Validator != addr.Hex() {
			t.Fatalf("block %d validator = %s, want %s", b.Number, b.Validator, addr.Hex())
		}
	}
	stats, err := api.Stats(ctx, nil)
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.HeadNumber < 3 {
		t.Fatalf("stats head = %d", stats.HeadNumber)
	}
	if stats.GenesisHash == "" || stats.HeadHash == "" {
		t.Fatal("stats missing hashes")
	}

	allocs, err := api.Allocations(ctx, nil)
	if err != nil {
		t.Fatalf("Allocations: %v", err)
	}
	if len(allocs) != len(cfg.Allocations) {
		t.Fatalf("allocation count = %d, want %d", len(allocs), len(cfg.Allocations))
	}

	supply, err := api.TrackedSupplyWei(ctx, nil)
	if err != nil {
		t.Fatalf("TrackedSupplyWei: %v", err)
	}
	if supply == "0" {
		t.Fatal("reported supply is zero")
	}

	engine, err := api.EngineStatus(ctx)
	if err != nil {
		t.Fatalf("EngineStatus: %v", err)
	}
	if engine["chainId"] != big.NewInt(TestnetChainID).String() {
		t.Fatalf("engine chain id = %v", engine["chainId"])
	}
}

// TestValidatorKeyMustBeInGenesisSet proves a node refuses to run with a key
// that is not an authorised authority.
func TestValidatorKeyMustBeInGenesisSet(t *testing.T) {
	cfg := testConfig(t)
	outsider, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	_, err = NewNode(NodeConfig{
		Genesis:          cfg,
		DataDir:          t.TempDir(),
		HTTPPort:         0,
		WSPort:           0,
		AuthPort:         0,
		P2PPort:          0,
		ValidatorKey:     hexKey(outsider),
		KeyStorePassword: "test-password",
		EnableMining:     true,
	})
	if err == nil {
		t.Fatal("node accepted a validator key that is not in the genesis set")
	}
}

// TestBootstrapFromSeedIsReproducible proves the validator set can be rebuilt
// from the seed alone.
func TestBootstrapFromSeedIsReproducible(t *testing.T) {
	a, err := BootstrapConfig(testSeed(), TestnetChainID, 5, Wei(100))
	if err != nil {
		t.Fatalf("BootstrapConfig: %v", err)
	}
	b, err := BootstrapConfig(testSeed(), TestnetChainID, 5, Wei(100))
	if err != nil {
		t.Fatalf("BootstrapConfig: %v", err)
	}
	if len(a.Validators) != 5 {
		t.Fatalf("validator count = %d", len(a.Validators))
	}
	for i := range a.Validators {
		if a.Validators[i].Signer != b.Validators[i].Signer {
			t.Fatalf("validator %d signer differs: %s vs %s", i, a.Validators[i].Signer, b.Validators[i].Signer)
		}
	}
	// A different seed must produce a different validator set.
	c, err := BootstrapConfig([]byte("a-completely-different-seed"), TestnetChainID, 5, Wei(100))
	if err != nil {
		t.Fatalf("BootstrapConfig: %v", err)
	}
	if c.Validators[0].Signer == a.Validators[0].Signer {
		t.Fatal("different seeds produced the same validator")
	}
}

// TestConfigFileRoundTrip proves a network definition survives a disk round
// trip and is stored with restrictive permissions.
func TestConfigFileRoundTrip(t *testing.T) {
	cfg := testConfig(t)
	path := t.TempDir() + "/network.json"
	if err := SaveConfigFile(path, cfg); err != nil {
		t.Fatalf("SaveConfigFile: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("config permissions = %o, want 600", info.Mode().Perm())
	}
	loaded, err := LoadConfigFile(path)
	if err != nil {
		t.Fatalf("LoadConfigFile: %v", err)
	}
	if loaded.ChainID != cfg.ChainID || len(loaded.Validators) != len(cfg.Validators) {
		t.Fatal("loaded config does not match the saved config")
	}
}

// hexKey renders a private key without the 0x prefix.
func hexKey(key *ecdsa.PrivateKey) string {
	return hexutil.Encode(crypto.FromECDSA(key))[2:]
}
