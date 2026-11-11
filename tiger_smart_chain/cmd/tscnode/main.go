// Command tscnode runs a TigerSmartChain node.
//
// Subcommands:
//
//	tscnode init       write a network definition and print its genesis hash
//	tscnode bootstrap  derive a validator set from a seed and write the config
//	tscnode run        start the node and (optionally) produce blocks
//	tscnode account    derive a deterministic account from the genesis seed
//	tscnode info       print the network parameters of a config file
//	tscnode genesis    print the genesis JSON for a config file
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/tigerwallet/tiger-smart-chain/chain"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd := os.Args[1]
	args := os.Args[2:]

	var err error
	switch cmd {
	case "init":
		err = runInit(args)
	case "bootstrap":
		err = runBootstrap(args)
	case "run":
		err = runNode(args)
	case "account":
		err = runAccount(args)
	case "info":
		err = runInfo(args)
	case "genesis":
		err = runGenesis(args)
	case "-h", "--help", "help":
		usage()
		return
	default:
		err = fmt.Errorf("unknown subcommand %q", cmd)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "tscnode: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `TigerSmartChain node

  tscnode init      -config <path> -chain-id <id> -validators <n> -supply <tgr> [-seed <hex|text>]
  tscnode bootstrap -config <path> -seed <hex|text> -validators <n> -chain-id <id> -supply <tgr>
  tscnode account   -seed <hex|text> -index <n>
  tscnode info      -config <path>
  tscnode genesis   -config <path> [-out <path>]
  tscnode run       -config <path> -datadir <dir> [-mine] [-validator-key <hex>]
                    [-http-port <n>] [-ws-port <n>] [-p2p-port <n>]
                    [-bootnodes <a,b>] [-static-nodes <a,b>] [-archive]

Environment:
  TSC_CHAIN_ID           override the chain id
  TSC_GENESIS_SEED       seed used by bootstrap/account when -seed is omitted
  TSC_VALIDATOR_KEY      validator private key when -validator-key is omitted
  TSC_KEYSTORE_PASSWORD  keystore password (defaults to tsc-keystore)
  TSC_BOOTNODES          comma separated enode URLs
  TSC_STATIC_NODES       comma separated enode URLs
`)
}

func runInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	configPath := fs.String("config", "network.json", "path to write the network definition")
	chainID := fs.Int64("chain-id", chain.DefaultChainID, "EVM chain id")
	validatorCount := fs.Int("validators", 4, "number of validators to bootstrap")
	supply := fs.String("supply", "1000000000", "total TGR supply")
	seed := fs.String("seed", "", "genesis seed (hex or text)")
	force := fs.Bool("force", false, "overwrite an existing config file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if _, err := os.Stat(*configPath); err == nil && !*force {
		return fmt.Errorf("%s already exists, pass -force to overwrite", *configPath)
	}

	seedBytes, err := resolveSeed(*seed)
	if err != nil {
		return err
	}
	total, err := chain.WeiFromString(*supply)
	if err != nil {
		return fmt.Errorf("invalid supply: %w", err)
	}
	cfg, err := chain.BootstrapConfig(seedBytes, *chainID, *validatorCount, total)
	if err != nil {
		return err
	}
	cfg.Name = networkName(*chainID)
	if err := chain.SaveConfigFile(*configPath, cfg); err != nil {
		return err
	}
	genesis, err := chain.BuildGenesis(cfg)
	if err != nil {
		return err
	}
	fmt.Printf("wrote %s\n", *configPath)
	fmt.Printf("network        %s\n", cfg.Name)
	fmt.Printf("chain id       %d\n", cfg.ChainID)
	fmt.Printf("validators     %d\n", len(cfg.Validators))
	fmt.Printf("total supply   %s TGR\n", *supply)
	fmt.Printf("genesis hash   %s\n", genesis.ToBlock().Hash().Hex())
	fmt.Println("validator signers:")
	for _, v := range cfg.Validators {
		fmt.Printf("  %-14s %s\n", v.Name, v.Signer)
	}
	return nil
}

func runBootstrap(args []string) error {
	fs := flag.NewFlagSet("bootstrap", flag.ContinueOnError)
	configPath := fs.String("config", "network.json", "path to write the network definition")
	seed := fs.String("seed", "", "genesis seed (hex or text)")
	validatorCount := fs.Int("validators", 4, "number of validators")
	chainID := fs.Int64("chain-id", 0, "EVM chain id")
	supply := fs.String("supply", "1000000000", "total TGR supply")
	if err := fs.Parse(args); err != nil {
		return err
	}
	seedBytes, err := resolveSeed(*seed)
	if err != nil {
		return err
	}
	total, err := chain.WeiFromString(*supply)
	if err != nil {
		return err
	}
	effectiveChainID := *chainID
	if effectiveChainID == 0 {
		effectiveChainID = chain.ChainIDFromEnv()
	}
	cfg, err := chain.BootstrapConfig(seedBytes, effectiveChainID, *validatorCount, total)
	if err != nil {
		return err
	}
	cfg.Name = networkName(effectiveChainID)
	if err := chain.SaveConfigFile(*configPath, cfg); err != nil {
		return err
	}
	fmt.Printf("bootstrapped %s with %d validators (chain id %d)\n", *configPath, len(cfg.Validators), cfg.ChainID)
	return nil
}

func runAccount(args []string) error {
	fs := flag.NewFlagSet("account", flag.ContinueOnError)
	seed := fs.String("seed", "", "genesis seed (hex or text)")
	index := fs.Uint("index", 0, "derivation index")
	if err := fs.Parse(args); err != nil {
		return err
	}
	seedBytes, err := resolveSeed(*seed)
	if err != nil {
		return err
	}
	addr, key, err := chain.DeriveAccount(seedBytes, uint32(*index))
	if err != nil {
		return err
	}
	fmt.Printf("index        %d\n", *index)
	fmt.Printf("address      %s\n", addr.Hex())
	fmt.Printf("private key  %064x\n", key)
	return nil
}

func runInfo(args []string) error {
	fs := flag.NewFlagSet("info", flag.ContinueOnError)
	configPath := fs.String("config", "network.json", "path to the network definition")
	asJSON := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := chain.LoadConfigFile(*configPath)
	if err != nil {
		return err
	}
	genesis, err := chain.BuildGenesis(cfg)
	if err != nil {
		return err
	}
	info := map[string]interface{}{
		"name":                cfg.Name,
		"chain_id":            cfg.ChainID,
		"native_symbol":       chain.NativeSymbol,
		"genesis_hash":        genesis.ToBlock().Hash().Hex(),
		"block_period":        cfg.BlockPeriodSeconds,
		"epoch_length":        cfg.EpochLength,
		"gas_limit":           cfg.GasLimit,
		"validator_count":     len(cfg.Validators),
		"allocation_count":    len(cfg.Allocations),
		"total_supply_wei":    chain.TotalSupply(cfg).String(),
		"min_validator_stake": cfg.MinValidatorStake.String(),
		"unbonding_seconds":   cfg.UnbondingSeconds,
		"treasury":            cfg.Treasury,
	}
	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(info)
	}
	fmt.Printf("name               %s\n", cfg.Name)
	fmt.Printf("chain id           %d\n", cfg.ChainID)
	fmt.Printf("native symbol      %s\n", chain.NativeSymbol)
	fmt.Printf("genesis hash       %s\n", genesis.ToBlock().Hash().Hex())
	fmt.Printf("block period       %ds\n", cfg.BlockPeriodSeconds)
	fmt.Printf("epoch length       %d\n", cfg.EpochLength)
	fmt.Printf("gas limit          %d\n", cfg.GasLimit)
	fmt.Printf("validators         %d\n", len(cfg.Validators))
	fmt.Printf("allocations        %d\n", len(cfg.Allocations))
	fmt.Printf("total supply wei   %s\n", chain.TotalSupply(cfg).String())
	fmt.Printf("min stake wei      %s\n", cfg.MinValidatorStake.String())
	fmt.Printf("unbonding seconds  %d\n", cfg.UnbondingSeconds)
	fmt.Printf("treasury           %s\n", cfg.Treasury)
	return nil
}

func runGenesis(args []string) error {
	fs := flag.NewFlagSet("genesis", flag.ContinueOnError)
	configPath := fs.String("config", "network.json", "path to the network definition")
	out := fs.String("out", "", "write the genesis JSON here instead of stdout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := chain.LoadConfigFile(*configPath)
	if err != nil {
		return err
	}
	genesis, err := chain.BuildGenesis(cfg)
	if err != nil {
		return err
	}
	blob, err := json.MarshalIndent(genesis, "", "  ")
	if err != nil {
		return err
	}
	if *out != "" {
		if err := os.WriteFile(*out, append(blob, '\n'), 0o600); err != nil {
			return err
		}
		fmt.Printf("wrote genesis to %s\n", *out)
		return nil
	}
	fmt.Println(string(blob))
	return nil
}

func runNode(args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	configPath := fs.String("config", "network.json", "path to the network definition")
	dataDir := fs.String("datadir", "", "chain data directory")
	mine := fs.Bool("mine", false, "produce blocks (requires a validator key)")
	validatorKey := fs.String("validator-key", "", "validator private key (hex)")
	httpPort := fs.Int("http-port", 8545, "JSON-RPC port")
	wsPort := fs.Int("ws-port", 8546, "WebSocket port")
	authPort := fs.Int("auth-port", 8551, "authenticated engine API port")
	p2pPort := fs.Int("p2p-port", 30310, "peer-to-peer port")
	httpHost := fs.String("http-host", "127.0.0.1", "JSON-RPC bind address")
	p2pBind := fs.String("p2p-bind", "0.0.0.0", "peer-to-peer bind address")
	bootnodes := fs.String("bootnodes", "", "comma separated enode URLs")
	staticNodes := fs.String("static-nodes", "", "comma separated enode URLs")
	maxPeers := fs.Int("max-peers", 50, "maximum peer count")
	archive := fs.Bool("archive", false, "keep the full historical state")
	keystorePassword := fs.String("keystore-password", "", "keystore password")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := chain.LoadConfigFile(*configPath)
	if err != nil {
		return err
	}
	if envID := chain.ChainIDFromEnv(); os.Getenv("TSC_CHAIN_ID") != "" {
		cfg.ChainID = envID
	}
	if *dataDir == "" {
		*dataDir = fmt.Sprintf("./tsc-data-%d", cfg.ChainID)
	}
	if *validatorKey == "" {
		*validatorKey = strings.TrimSpace(os.Getenv("TSC_VALIDATOR_KEY"))
	}
	if *keystorePassword == "" {
		*keystorePassword = os.Getenv("TSC_KEYSTORE_PASSWORD")
	}
	if *bootnodes == "" {
		*bootnodes = os.Getenv("TSC_BOOTNODES")
	}
	if *staticNodes == "" {
		*staticNodes = os.Getenv("TSC_STATIC_NODES")
	}
	if *mine && *validatorKey == "" {
		return errors.New("mining requires -validator-key or TSC_VALIDATOR_KEY")
	}

	n, err := chain.NewNode(chain.NodeConfig{
		Genesis:          cfg,
		DataDir:          *dataDir,
		HTTPHost:         *httpHost,
		HTTPPort:         *httpPort,
		WSHost:           *httpHost,
		WSPort:           *wsPort,
		AuthPort:         *authPort,
		P2PBind:          *p2pBind,
		P2PPort:          *p2pPort,
		Bootnodes:        splitList(*bootnodes),
		StaticNodes:      splitList(*staticNodes),
		MaxPeers:         *maxPeers,
		ValidatorKey:     *validatorKey,
		KeyStorePassword: *keystorePassword,
		EnableMining:     *mine,
		Archive:          *archive,
	})
	if err != nil {
		return err
	}
	if err := n.Start(); err != nil {
		return err
	}

	h := n.Health()
	fmt.Printf("TigerSmartChain node started\n")
	fmt.Printf("  network      %s\n", h.NetworkName)
	fmt.Printf("  chain id     %s\n", h.ChainID)
	fmt.Printf("  genesis      %s\n", h.GenesisHash)
	fmt.Printf("  data dir     %s\n", *dataDir)
	fmt.Printf("  json-rpc     http://%s:%d\n", *httpHost, *httpPort)
	fmt.Printf("  websocket    ws://%s:%d\n", *httpHost, *wsPort)
	fmt.Printf("  p2p          %s:%d\n", *p2pBind, *p2pPort)
	if h.IsValidator {
		fmt.Printf("  validator    %s\n", h.Signer)
		fmt.Printf("  mining       %v\n", h.Mining)
	} else {
		fmt.Printf("  validator    none (read-only replica)\n")
	}

	// A proof-of-authority chain only advances once a majority of the
	// authorised signers are online. Warn the operator when this node alone
	// cannot reach quorum, instead of letting them watch a chain that never
	// moves.
	required := chain.Quorum(len(cfg.Validators))
	if *mine && required > 1 {
		fmt.Printf("  quorum       %d of %d validators must be online for blocks to be produced\n",
			required, len(cfg.Validators))
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-sig:
			fmt.Println("shutting down")
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			_ = ctx
			cancel()
			return n.Stop()
		case <-ticker.C:
			s := n.Health()
			fmt.Printf("[%s] height=%d hash=%s peers=%d pending=%d\n",
				time.Now().UTC().Format(time.RFC3339), s.HeadNumber, short(s.HeadHash), s.PeerCount, s.TxPoolSize)
		}
	}
}

// resolveSeed returns the operator supplied seed, falling back to the
// environment. It never invents a default, because a shared default seed would
// make every deployment use the same validator keys.
func resolveSeed(flagValue string) ([]byte, error) {
	if strings.TrimSpace(flagValue) != "" {
		return chain.SeedFromString(flagValue)
	}
	return chain.SeedFromEnv()
}

func splitList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func networkName(chainID int64) string {
	switch chainID {
	case chain.DefaultChainID:
		return "TigerSmartChain"
	case chain.TestnetChainID:
		return "TigerSmartChain Testnet"
	default:
		return fmt.Sprintf("TigerSmartChain %d", chainID)
	}
}

func short(hash string) string {
	if len(hash) <= 14 {
		return hash
	}
	return hash[:14]
}
