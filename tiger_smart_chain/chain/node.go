package chain

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rpc"
)

// Errors returned by the node lifecycle.
var (
	ErrSignerNotValidator = errors.New("tsc: configured signer is not in the genesis validator set")
	ErrNoSigner           = errors.New("tsc: no validator signing key configured")
	ErrAlreadyRunning     = errors.New("tsc: node already running")
)

// NodeConfig holds everything required to run a TigerSmartChain node.
type NodeConfig struct {
	// Genesis is the network definition this node follows.
	Genesis *Config

	// DataDir is the chain data directory.
	DataDir string

	// HTTPHost and HTTPPort expose the JSON-RPC endpoint.
	HTTPHost string
	HTTPPort int

	// WSHost and WSPort expose the WebSocket endpoint used by wallet clients
	// for live subscription to blocks and logs.
	WSHost string
	WSPort int

	// AuthPort exposes the authenticated engine API (node-to-node control).
	AuthPort int

	// P2PBind is the listen address for peer connections.
	P2PBind string

	// P2PPort is the peer-to-peer listening port.
	P2PPort int

	// Bootnodes seeds the peer set.
	Bootnodes []string

	// StaticNodes are peers this node always dials.
	StaticNodes []string

	// TrustedNodes are peers that bypass transaction propagation throttling.
	TrustedNodes []string

	// MaxPeers caps the peer count.
	MaxPeers int

	// ValidatorKey is the hex-encoded secp256k1 private key that signs blocks.
	// When empty the node runs as a read-only replica that verifies but does
	// not produce blocks.
	ValidatorKey string

	// KeyStorePassword protects the on-disk key material.
	KeyStorePassword string

	// EnableMining turns this node into a block producer. It requires
	// ValidatorKey to be set.
	EnableMining bool

	// TxPoolPriceLimit is the minimum gas tip accepted into the pool.
	TxPoolPriceLimit *big.Int

	// Archive keeps the full historical state instead of pruning it.
	Archive bool
}

// Node is a running TigerSmartChain node.
type Node struct {
	mu      sync.RWMutex
	cfg     NodeConfig
	genesis *core.Genesis
	stack   *node.Node
	backend *eth.Ethereum
	ks      *keystore.KeyStore
	signer  common.Address
	mining  bool
	started bool
}

// NewNode creates the node and initialises its data directory, genesis block
// and RPC surface. It does not start the peer-to-peer stack; call Start for
// that.
func NewNode(cfg NodeConfig) (*Node, error) {
	if cfg.Genesis == nil {
		return nil, fmt.Errorf("tsc: node requires a genesis configuration")
	}
	if err := cfg.Genesis.Validate(); err != nil {
		return nil, err
	}
	if cfg.DataDir == "" {
		return nil, fmt.Errorf("tsc: node requires a data directory")
	}
	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		return nil, fmt.Errorf("tsc: create data dir: %w", err)
	}

	genesis, err := BuildGenesis(cfg.Genesis)
	if err != nil {
		return nil, err
	}

	if cfg.HTTPHost == "" {
		cfg.HTTPHost = "127.0.0.1"
	}
	if cfg.WSHost == "" {
		cfg.WSHost = "127.0.0.1"
	}
	if cfg.HTTPPort == 0 {
		cfg.HTTPPort = 8545
	}
	if cfg.WSPort == 0 {
		cfg.WSPort = 8546
	}
	if cfg.AuthPort == 0 {
		cfg.AuthPort = 8551
	}
	if cfg.P2PBind == "" {
		cfg.P2PBind = "0.0.0.0"
	}
	if cfg.P2PPort == 0 {
		cfg.P2PPort = 30310
	}
	if cfg.MaxPeers == 0 {
		cfg.MaxPeers = 50
	}
	if cfg.TxPoolPriceLimit == nil {
		cfg.TxPoolPriceLimit = big.NewInt(1)
	}
	if cfg.KeyStorePassword == "" {
		cfg.KeyStorePassword = "tsc-keystore"
	}

	bootnodes, err := parseEnodes(cfg.Bootnodes)
	if err != nil {
		return nil, fmt.Errorf("tsc: bootnodes: %w", err)
	}
	staticNodes, err := parseEnodes(cfg.StaticNodes)
	if err != nil {
		return nil, fmt.Errorf("tsc: static nodes: %w", err)
	}
	trustedNodes, err := parseEnodes(cfg.TrustedNodes)
	if err != nil {
		return nil, fmt.Errorf("tsc: trusted nodes: %w", err)
	}

	stackCfg := &node.Config{
		Name:        "tigersmartchain",
		DataDir:     cfg.DataDir,
		KeyStoreDir: cfg.DataDir + "/keystore",
		HTTPHost:    cfg.HTTPHost,
		HTTPPort:    cfg.HTTPPort,
		WSHost:      cfg.WSHost,
		WSPort:      cfg.WSPort,
		AuthAddr:    fmt.Sprintf("%s:%d", cfg.HTTPHost, cfg.AuthPort),
		P2P: p2p.Config{
			ListenAddr:      fmt.Sprintf("%s:%d", cfg.P2PBind, cfg.P2PPort),
			MaxPeers:        cfg.MaxPeers,
			MaxPendingPeers: 50,
			BootstrapNodes:  bootnodes,
			StaticNodes:     staticNodes,
			TrustedNodes:    trustedNodes,
		},
		HTTPModules: []string{"eth", "net", "web3", "txpool", "debug", "tsc"},
		WSModules:   []string{"eth", "net", "web3", "tsc"},
	}
	stack, err := node.New(stackCfg)
	if err != nil {
		return nil, fmt.Errorf("tsc: create node: %w", err)
	}

	ks := keystore.NewKeyStore(cfg.DataDir+"/keystore", keystore.StandardScryptN, keystore.StandardScryptP)
	stack.AccountManager().AddBackend(ks)

	ethCfg := ethconfig.Defaults
	ethCfg.Genesis = genesis
	ethCfg.NetworkId = uint64(cfg.Genesis.ChainID)
	ethCfg.SyncMode = downloader.FullSync
	ethCfg.NoPruning = cfg.Archive
	ethCfg.Miner.GasPrice = big.NewInt(1)
	ethCfg.Miner.GasCeil = cfg.Genesis.GasLimit
	ethCfg.TxPool.PriceLimit = cfg.TxPoolPriceLimit.Uint64()
	ethCfg.TxPool.NoLocals = true
	ethCfg.TxPool.Journal = cfg.DataDir + "/txpool/journal"

	backend, err := eth.New(stack, &ethCfg)
	if err != nil {
		stack.Close()
		return nil, fmt.Errorf("tsc: create eth backend: %w", err)
	}

	n := &Node{
		cfg:     cfg,
		genesis: genesis,
		stack:   stack,
		backend: backend,
		ks:      ks,
	}

	if cfg.ValidatorKey != "" {
		addr, err := n.importValidatorKey(cfg.ValidatorKey)
		if err != nil {
			stack.Close()
			return nil, err
		}
		n.signer = addr
		if !isGenesisValidator(cfg.Genesis, addr) {
			stack.Close()
			return nil, fmt.Errorf("%w: %s", ErrSignerNotValidator, addr.Hex())
		}
	}
	if cfg.EnableMining && n.signer == (common.Address{}) {
		stack.Close()
		return nil, ErrNoSigner
	}

	if err := n.registerTSCRPC(); err != nil {
		stack.Close()
		return nil, err
	}
	return n, nil
}

// parseEnodes converts operator supplied enode URLs into p2p nodes, failing
// closed on a malformed entry rather than silently dropping a peer.
func parseEnodes(raw []string) ([]*enode.Node, error) {
	out := make([]*enode.Node, 0, len(raw))
	for _, u := range raw {
		s := strings.TrimSpace(u)
		if s == "" {
			continue
		}
		n, err := enode.ParseV4(s)
		if err != nil {
			return nil, fmt.Errorf("%q: %w", s, err)
		}
		out = append(out, n)
	}
	return out, nil
}

// importValidatorKey loads the signing key into the node keystore and unlocks
// it. The plaintext key is never written to disk or logged.
func (n *Node) importValidatorKey(hexKey string) (common.Address, error) {
	key, err := crypto.HexToECDSA(strings.TrimPrefix(strings.TrimSpace(hexKey), "0x"))
	if err != nil {
		return common.Address{}, fmt.Errorf("tsc: parse validator key: %w", err)
	}
	addr := crypto.PubkeyToAddress(key.PublicKey)

	for _, existing := range n.ks.Accounts() {
		if existing.Address == addr {
			if err := n.ks.Unlock(existing, n.cfg.KeyStorePassword); err != nil {
				return common.Address{}, fmt.Errorf("tsc: unlock validator account: %w", err)
			}
			return addr, nil
		}
	}
	acct, err := n.ks.ImportECDSA(key, n.cfg.KeyStorePassword)
	if err != nil {
		return common.Address{}, fmt.Errorf("tsc: import validator key: %w", err)
	}
	if err := n.ks.Unlock(acct, n.cfg.KeyStorePassword); err != nil {
		return common.Address{}, fmt.Errorf("tsc: unlock validator account: %w", err)
	}
	return addr, nil
}

func isGenesisValidator(cfg *Config, addr common.Address) bool {
	for _, v := range cfg.Validators {
		norm, err := NormalizeAddress(v.Signer)
		if err != nil {
			continue
		}
		if common.HexToAddress(norm) == addr {
			return true
		}
	}
	return false
}

// Start brings the node online and, when configured, begins producing blocks.
func (n *Node) Start() error {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.started {
		return ErrAlreadyRunning
	}
	if err := n.stack.Start(); err != nil {
		return fmt.Errorf("tsc: start stack: %w", err)
	}
	n.started = true

	if n.cfg.EnableMining {
		n.backend.SetEtherbase(n.signer)
		if err := n.backend.StartMining(); err != nil {
			_ = n.stack.Close()
			n.started = false
			return fmt.Errorf("tsc: start mining: %w", err)
		}
		n.mining = true
	}
	return nil
}

// Stop shuts the node down and flushes its databases.
func (n *Node) Stop() error {
	n.mu.Lock()
	defer n.mu.Unlock()
	if !n.started {
		return nil
	}
	if n.mining {
		n.backend.StopMining()
		n.mining = false
	}
	n.started = false
	return n.stack.Close()
}

// Backend exposes the underlying execution engine for the indexer and API.
func (n *Node) Backend() *eth.Ethereum { return n.backend }

// Stack exposes the hosting node.
func (n *Node) Stack() *node.Node { return n.stack }

// Signer returns the block signing address, or the zero address for a
// non-producing replica.
func (n *Node) Signer() common.Address { return n.signer }

// IsMining reports whether this node is currently producing blocks.
func (n *Node) IsMining() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.mining
}

// ChainID returns the configured chain id.
func (n *Node) ChainID() *big.Int { return n.genesis.Config.ChainID }

// GenesisHash returns the hash of the genesis block this node follows.
func (n *Node) GenesisHash() common.Hash { return n.backend.BlockChain().Genesis().Hash() }

// Client returns a JSON-RPC client bound to this node's HTTP endpoint.
func (n *Node) Client() (*ethclient.Client, error) {
	return ethclient.Dial(fmt.Sprintf("http://%s:%d", n.cfg.HTTPHost, n.cfg.HTTPPort))
}

// WaitForHeight blocks until the chain reaches the requested height or the
// context expires. It is used by startup probes and integration tests.
func (n *Node) WaitForHeight(ctx context.Context, height uint64) error {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		if n.backend.BlockChain().CurrentBlock().Number.Uint64() >= height {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("tsc: timed out waiting for height %d (at %d): %w",
				height, n.backend.BlockChain().CurrentBlock().Number.Uint64(), ctx.Err())
		case <-ticker.C:
		}
	}
}

// Health reports the node's operational status for readiness probes.
type Health struct {
	ChainID      string `json:"chain_id"`
	NetworkName  string `json:"network_name"`
	GenesisHash  string `json:"genesis_hash"`
	HeadNumber   uint64 `json:"head_number"`
	HeadHash     string `json:"head_hash"`
	Signer       string `json:"signer,omitempty"`
	Mining       bool   `json:"mining"`
	PeerCount    int    `json:"peer_count"`
	TxPoolSize   int    `json:"txpool_pending"`
	GasLimit     uint64 `json:"gas_limit"`
	IsValidator  bool   `json:"is_validator"`
	KeystorePath string `json:"-"`
}

// Health samples the current node state.
func (n *Node) Health() Health {
	head := n.backend.BlockChain().CurrentBlock()
	h := Health{
		ChainID:     n.genesis.Config.ChainID.String(),
		NetworkName: n.cfg.Genesis.Name,
		GenesisHash: n.backend.BlockChain().Genesis().Hash().Hex(),
		HeadNumber:  head.Number.Uint64(),
		HeadHash:    head.Hash().Hex(),
		Mining:      n.IsMining(),
		GasLimit:    head.GasLimit,
	}
	if n.signer != (common.Address{}) {
		h.Signer = n.signer.Hex()
		h.IsValidator = true
	}
	if n.started {
		h.PeerCount = n.stack.Server().PeerCount()
		pending, _ := n.backend.TxPool().Stats()
		h.TxPoolSize = pending
	}
	return h
}

// RPCClient exposes a low level rpc client for internal tooling.
func (n *Node) RPCClient() (*rpc.Client, error) {
	return rpc.DialHTTP(fmt.Sprintf("http://%s:%d", n.cfg.HTTPHost, n.cfg.HTTPPort))
}
