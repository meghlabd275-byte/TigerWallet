package master_wallet

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// ============================================================================
// Blockchain Types
// ============================================================================

type BlockchainType string

const (
	TypeEVM       BlockchainType = "evm"
	TypeSolana    BlockchainType = "solana"
	TypeCosmos    BlockchainType = "cosmos"
	TypePolkadot  BlockchainType = "polkadot"
	TypeCardano   BlockchainType = "cardano"
	TypeAlgorand  BlockchainType = "algorand"
	TypeNear      BlockchainType = "near"
	TypeAptos     BlockchainType = "aptos"
	TypeSui       BlockchainType = "sui"
	TypeStarknet  BlockchainType = "starknet"
	TypeTon       BlockchainType = "ton"
	TypeBitcoin   BlockchainType = "bitcoin"
	TypeRipple    BlockchainType = "ripple"
	TypeTron      BlockchainType = "tron"
	TypeTerra     BlockchainType = "terra"
	TypeAcala     BlockchainType = "acala"
	TypeAzero     BlockchainType = "azero"
	TypeBase      BlockchainType = "base"
	TypeBlast     BlockchainType = "blast"
	TypeCelo      BlockchainType = "celo"
	TypeCronos    BlockchainType = "cronos"
	TypeEos       BlockchainType = "eos"
	TypeFantom    BlockchainType = "fantom"
	TypeFilecoin  BlockchainType = "filecoin"
	TypeFlare     BlockchainType = "flare"
	TypeGnosis    BlockchainType = "gnosis"
	TypeHarmony   BlockchainType = "harmony"
	TypeHedera    BlockchainType = "hedera"
	TypeIoTeX     BlockchainType = "iotex"
	TypeKava      BlockchainType = "kava"
	TypeKusama    BlockchainType = "kusama"
	TypeLinea     BlockchainType = "linea"
	TypeMantle    BlockchainType = "mantle"
	TypeMetis     BlockchainType = "metis"
	TypeMoonbeam  BlockchainType = "moonbeam"
	TypeOptimism  BlockchainType = "optimism"
	TypePolygon   BlockchainType = "polygon"
	TypeScroll    BlockchainType = "scroll"
	TypeSei       BlockchainType = "sei"
	TypeSui       BlockchainType = "sui"
	TypeTaiko     BlockchainType = "taiko"
	TypeVelodrome BlockchainType = "velodrome"
	TypeZetachain BlockchainType = "zetachain"
	TypeZksync    BlockchainType = "zksync"
	TypeCore      BlockchainType = "core"
	TypeInjective BlockchainType = "injective"
)

// Network represents a blockchain network
type Network struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	Symbol          string         `json:"symbol"`
	Type            BlockchainType `json:"type"`
	ChainID         int64          `json:"chain_id"`
	Decimals        int            `json:"decimals"`
	Explorer        string         `json:"explorer"`
	RPCURL          string         `json:"rpc_url"`
	APIURL          string         `json:"api_url"`
	WSSURL          string         `json:"wss_url"`
	IsTestnet       bool           `json:"is_testnet"`
	Confirmations   int            `json:"confirmations"`
	MinTransfer     float64        `json:"min_transfer"`
	MaxTransfer     float64        `json:"max_transfer"`
	SupportsEIP1559 bool           `json:"supports_eip1559"`
	GasStation      string         `json:"gas_station"`
	StableCoins     []string       `json:"stable_coins"`
	NativeToken     string         `json:"native_token"`
	LogoURL         string         `json:"logo_url"`
	Color           string         `json:"color"`
	BlockTime       int            `json:"block_time"`
	AverageGasPrice string         `json:"average_gas_price"`
	MaxGasPrice     string         `json:"max_gas_price"`
	AddedAt         int64          `json:"added_at"`
	UpdatedAt       int64          `json:"updated_at"`
}

// Token represents a cryptocurrency token
type Token struct {
	ID            string  `json:"id"`
	Address       string  `json:"address"`
	Name          string  `json:"name"`
	Symbol        string  `json:"symbol"`
	Decimals      int     `json:"decimals"`
	ChainID       int64   `json:"chain_id"`
	Type          string  `json:"type"`
	TotalSupply   string  `json:"total_supply"`
	IsStableCoin bool    `json:"is_stable_coin"`
	IsWrapped    bool    `json:"is_wrapped"`
	IsVerified   bool    `json:"is_verified"`
	LogoURL       string  `json:"logo_url"`
	Price         float64 `json:"price"`
	MarketCap     float64 `json:"market_cap"`
	Volume24h     float64 `json:"volume_24h"`
	Website       string  `json:"website"`
	AddedAt       int64   `json:"added_at"`
}

// BlockchainRegistry manages all supported blockchains
type BlockchainRegistry struct {
	mu        sync.RWMutex
	networks  map[string]*Network
	tokens    map[int64][]*Token
	chainIDs  map[int64]string
	symbolMap map[string]*Token
}

var (
	registry     *BlockchainRegistry
	registryOnce sync.Once
)

// GetRegistry returns the singleton blockchain registry
func GetRegistry() *BlockchainRegistry {
	registryOnce.Do(func() {
		registry = &BlockchainRegistry{
			networks:  make(map[string]*Network),
			tokens:    make(map[int64][]*Token),
			chainIDs:  make(map[int64]string),
			symbolMap: make(map[string]*Token),
		}
		registry.initNetworks()
		registry.initTokens()
	})
	return registry
}

// initNetworks initializes all 103+ supported networks
func (r *BlockchainRegistry) initNetworks() {
	networks := []*Network{
		// === EVM MAINNETS (Top 50) ===
		{id: "ethereum", name: "Ethereum", symbol: "ETH", type: TypeEVM, chainID: 1, decimals: 18, explorer: "https://etherscan.io", RPCURL: "https://eth.llamarpc.com", APIURL: "https://api.etherscan.io", WSSURL: "wss://eth-mainnet.ws.alchemyapi.io", confirmations: 12, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, gasStation: "https://ethgasstation.info", stableCoins: []string{"USDT", "USDC", "DAI", "BUSD", "TUSD"}, nativeToken: "ETH", logoURL: "https://cryptologos.cc/logos/ethereum-eth-logo.png", color: "#627EEA", blockTime: 12, averageGasPrice: "20", maxGasPrice: "100"},
		{id: "bnb-smart-chain", name: "BNB Smart Chain", symbol: "BNB", type: TypeEVM, chainID: 56, decimals: 18, explorer: "https://bscscan.com", RPCURL: "https://bsc-dataseed.binance.org", APIURL: "https://api.bscscan.com", WSSURL: "wss://bsc-ws-node.nariox.org", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, stableCoins: []string{"USDT", "USDC", "BUSD", "DAI"}, nativeToken: "BNB", logoURL: "https://cryptologos.cc/logos/bnb-bnb-logo.png", color: "#F3BA2F", blockTime: 3, averageGasPrice: "5", maxGasPrice: "20"},
		{id: "polygon", name: "Polygon", symbol: "MATIC", type: TypeEVM, chainID: 137, decimals: 18, explorer: "https://polygonscan.com", RPCURL: "https://polygon-rpc.com", APIURL: "https://api.polygonscan.com", WSSURL: "wss://ws-mainnet.polygon.technology", confirmations: 15, minTransfer: 0.01, maxTransfer: 100000, supportsEIP1559: true, stableCoins: []string{"USDT", "USDC", "DAI"}, nativeToken: "MATIC", logoURL: "https://cryptologos.cc/logos/polygon-matic-logo.png", color: "#8247E5", blockTime: 2, averageGasPrice: "50", maxGasPrice: "200"},
		{id: "arbitrum-one", name: "Arbitrum One", symbol: "ETH", type: TypeEVM, chainID: 42161, decimals: 18, explorer: "https://arbiscan.io", RPCURL: "https://arb1.arbitrum.io/rpc", APIURL: "https://api.arbiscan.io", WSSURL: "wss://arb1.arbitrum.io/ws", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, stableCoins: []string{"USDT", "USDC", "DAI"}, nativeToken: "ETH", logoURL: "https://cryptologos.cc/logos/arbitrum-arb-logo.png", color: "#28A0F0", blockTime: 1, averageGasPrice: "0.1", maxGasPrice: "1"},
		{id: "optimism", name: "Optimism", symbol: "ETH", type: TypeEVM, chainID: 10, decimals: 18, explorer: "https://optimistic.etherscan.io", RPCURL: "https://mainnet.optimism.io", APIURL: "https://api-optimistic.etherscan.io", WSSURL: "wss://ws-mainnet.optimism.io", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, stableCoins: []string{"USDT", "USDC", "DAI"}, nativeToken: "ETH", logoURL: "https://cryptologos.cc/logos/optimism-eth-logo.png", color: "#FF0420", blockTime: 2, averageGasPrice: "0.001", maxGasPrice: "0.1"},
		{id: "base", name: "Base", symbol: "ETH", type: TypeEVM, chainID: 8453, decimals: 18, explorer: "https://basescan.org", RPCURL: "https://mainnet.base.org", APIURL: "https://api.basescan.org", WSSURL: "wss://ws.base.org", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, stableCoins: []string{"USDT", "USDC", "DAI"}, nativeToken: "ETH", logoURL: "https://cryptologos.cc/logos/base-logo.png", color: "#0052FF", blockTime: 2, averageGasPrice: "0.001", maxGasPrice: "0.1"},
		{id: "avalanche-c", name: "Avalanche C-Chain", symbol: "AVAX", type: TypeEVM, chainID: 43114, decimals: 18, explorer: "https://snowtrace.io", RPCURL: "https://api.avax.network/ext/bc/C/rpc", APIURL: "https://api.snowtrace.io", WSSURL: "wss://api.avax.network/ext/bc/C/ws", confirmations: 15, minTransfer: 0.01, maxTransfer: 100000, supportsEIP1559: true, stableCoins: []string{"USDT", "USDC", "DAI"}, nativeToken: "AVAX", logoURL: "https://cryptologos.cc/logos/avalanche-avax-logo.png", color: "#E84142", blockTime: 2, averageGasPrice: "25", maxGasPrice: "100"},
		{id: "fantom", name: "Fantom Opera", symbol: "FTM", type: TypeEVM, chainID: 250, decimals: 18, explorer: "https://ftmscan.com", RPCURL: "https://rpc.fantom.network", APIURL: "https://api.ftmscan.com", WSSURL: "wss://ws.fantom.network", confirmations: 15, minTransfer: 1, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{"USDT", "USDC", "DAI"}, nativeToken: "FTM", logoURL: "https://cryptologos.cc/logos/fantom-ftm-logo.png", color: "#1969FF", blockTime: 1, averageGasPrice: "500", maxGasPrice: "2000"},
		{id: "celo", name: "Celo", symbol: "CELO", type: TypeEVM, chainID: 42220, decimals: 18, explorer: "https://explorer.celo.org", RPCURL: "https://forno.celo.org", APIURL: "https://api.celoscan.io", WSSURL: "wss://forno.celo.org/ws", confirmations: 15, minTransfer: 0.01, maxTransfer: 100000, supportsEIP1559: true, stableCoins: []string{"cUSD", "cEUR", "USDT"}, nativeToken: "CELO", logoURL: "https://cryptologos.cc/logos/celo-celo-logo.png", color: "#FCFF52", blockTime: 5, averageGasPrice: "5", maxGasPrice: "20"},
		{id: "cronos", name: "Cronos", symbol: "CRO", type: TypeEVM, chainID: 25, decimals: 18, explorer: "https://cronoscan.com", RPCURL: "https://evm.cronos.org", APIURL: "https://api.cronoscan.com", WSSURL: "wss://evm.cronos.org", confirmations: 15, minTransfer: 1, maxTransfer: 100000, supportsEIP1559: false, stableCoins: []string{"USDT", "USDC", "DAI"}, nativeToken: "CRO", logoURL: "https://cryptologos.cc/logos/cronos-cro-logo.png", color: "#002D74", blockTime: 5, averageGasPrice: "5000000000", maxGasPrice: "10000000000"},
		{id: "kava", name: "Kava", symbol: "KAVA", type: TypeEVM, chainID: 2222, decimals: 18, explorer: "https://kavascan.com", RPCURL: "https://evm.kava.io", APIURL: "https://api.kavascan.com", WSSURL: "wss://evm.kava.io", confirmations: 15, minTransfer: 0.1, maxTransfer: 100000, supportsEIP1559: false, stableCoins: []string{"USDT", "USDC", "DAI"}, nativeToken: "KAVA", logoURL: "https://cryptologos.cc/logos/kava-kava-logo.png", color: "#FF5733", blockTime: 5, averageGasPrice: "2000000000", maxGasPrice: "5000000000"},
		{id: "metis", name: "Metis Andromeda", symbol: "METIS", type: TypeEVM, chainID: 1088, decimals: 18, explorer: "https://andromedaexplorer.metis.io", RPCURL: "https://andromeda.metis.io/?owner=1088", APIURL: "https://api.andromedexplorer.metis.io", WSSURL: "wss://andromeda.metis.io/ws", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{"USDT", "USDC"}, nativeToken: "METIS", logoURL: "https://cryptologos.cc/logos/metis-metis-logo.png", color: "#6F6CEF", blockTime: 2, averageGasPrice: "100", maxGasPrice: "500"},
		{id: "mantle", name: "Mantle", symbol: "MNT", type: TypeEVM, chainID: 5000, decimals: 18, explorer: "https://explorer.mantle.xyz", RPCURL: "https://rpc.mantle.xyz", APIURL: "https://api.mantlescan.org", WSSURL: "wss://ws.mantle.xyz", confirmations: 15, minTransfer: 0.01, maxTransfer: 100000, supportsEIP1559: true, stableCoins: []string{"USDT", "USDC", "DAI"}, nativeToken: "MNT", logoURL: "https://cryptologos.cc/logos/mantle-mnt-logo.png", color: "#1A1A1A", blockTime: 2, averageGasPrice: "0.001", maxGasPrice: "0.01"},
		{id: "blast", name: "Blast", symbol: "ETH", type: TypeEVM, chainID: 81457, decimals: 18, explorer: "https://blastscan.io", RPCURL: "https://rpc.blast.io", APIURL: "https://api.blastscan.io", WSSURL: "wss://ws.blast.io", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, stableCoins: []string{"USDT", "USDC", "DAI", "USDB"}, nativeToken: "ETH", logoURL: "https://cryptologos.cc/logos/blast-blast-logo.png", color: "#FFCB00", blockTime: 2, averageGasPrice: "0.001", maxGasPrice: "0.01"},
		{id: "linea", name: "Linea", symbol: "ETH", type: TypeEVM, chainID: 59144, decimals: 18, explorer: "https://lineascan.build", RPCURL: "https://rpc.linea.build", APIURL: "https://api.lineascan.build", WSSURL: "wss://rpc.linea.build", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, stableCoins: []string{"USDT", "USDC", "DAI"}, nativeToken: "ETH", logoURL: "https://cryptologos.cc/logos/linea-l2a-logo.png", color: "#121212", blockTime: 2, averageGasPrice: "0.001", maxGasPrice: "0.01"},
		{id: "scroll", name: "Scroll", symbol: "ETH", type: TypeEVM, chainID: 534352, decimals: 18, explorer: "https://scrollscan.com", RPCURL: "https://rpc.scroll.io", APIURL: "https://api.scrollscan.com", WSSURL: "wss://ws-rpc.scroll.io", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, stableCoins: []string{"USDT", "USDC", "DAI"}, nativeToken: "ETH", logoURL: "https://cryptologos.cc/logos/scroll-scroll-logo.png", color: "#CDA8FF", blockTime: 3, averageGasPrice: "0.001", maxGasPrice: "0.01"},
		{id: "zksync-era", name: "zkSync Era", symbol: "ETH", type: TypeEVM, chainID: 324, decimals: 18, explorer: "https://zksync2.zkscan.io", RPCURL: "https://mainnet.era.zksync.io", APIURL: "https://api.zksync.io", WSSURL: "wss://mainnet.era.zksync.io/ws", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{"USDT", "USDC", "DAI"}, nativeToken: "ETH", logoURL: "https://cryptologos.cc/logos/zksync-zksync-logo.png", color: "#8B8CFF", blockTime: 1, averageGasPrice: "0.001", maxGasPrice: "0.01"},
		{id: "polygon-zkevm", name: "Polygon zkEVM", symbol: "ETH", type: TypeEVM, chainID: 1101, decimals: 18, explorer: "https://zkevm.polygonscan.com", RPCURL: "https://zkevm-rpc.com", APIURL: "https://api-zkevm.polygonscan.com", WSSURL: "wss://ws-zkevm-rpc.com", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, stableCoins: []string{"USDT", "USDC", "DAI"}, nativeToken: "ETH", logoURL: "https://cryptologos.cc/logos/polygon-zkevm-logo.png", color: "#8247E5", blockTime: 1, averageGasPrice: "0.001", maxGasPrice: "0.01"},
		{id: "starknet", name: "Starknet", symbol: "ETH", type: TypeStarknet, chainID: 0, decimals: 18, explorer: "https://starkscan.co", RPCURL: "https://rpc.starknet.io", APIURL: "https://api.starkscan.io", WSSURL: "wss://rpc.starknet.io", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{"USDT", "USDC"}, nativeToken: "ETH", logoURL: "https://cryptologos.cc/logos/starknet-strk-logo.png", color: "#00D4FF", blockTime: 5, averageGasPrice: "500", maxGasPrice: "5000"},
		{id: "gnosis", name: "Gnosis Chain", symbol: "XDAI", type: TypeEVM, chainID: 100, decimals: 18, explorer: "https://gnosisscan.io", RPCURL: "https://rpc.gnosischain.com", APIURL: "https://api.gnosisscan.io", WSSURL: "wss://rpc.gnosischain.com/ws", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{"USDT", "USDC", "DAI"}, nativeToken: "XDAI", logoURL: "https://cryptologos.cc/logos/gnosis-gno-logo.png", color: "#04795A", blockTime: 5, averageGasPrice: "1000000000", maxGasPrice: "5000000000"},
		{id: "moonbeam", name: "Moonbeam", symbol: "GLMR", type: TypeEVM, chainID: 1284, decimals: 18, explorer: "https://moonscan.io", RPCURL: "https://rpc.api.moonbeam.network", APIURL: "https://api.moonscan.io", WSSURL: "wss://wss.api.moonbeam.network", confirmations: 15, minTransfer: 0.01, maxTransfer: 100000, supportsEIP1559: true, stableCoins: []string{"USDT", "USDC", "DAI"}, nativeToken: "GLMR", logoURL: "https://cryptologos.cc/logos/moonbeam-moonbeam-logo.png", color: "#53CBC9", blockTime: 12, averageGasPrice: "1000000000", maxGasPrice: "5000000000"},
		{id: "moonriver", name: "Moonriver", symbol: "MOVR", type: TypeEVM, chainID: 1285, decimals: 18, explorer: "https://moonriver.moonscan.io", RPCURL: "https://rpc.api.moonriver.network", APIURL: "https://api.moonriver.moonscan.io", WSSURL: "wss://wss.api.moonriver.network", confirmations: 15, minTransfer: 0.01, maxTransfer: 100000, supportsEIP1559: true, stableCoins: []string{"USDT", "USDC"}, nativeToken: "MOVR", logoURL: "https://cryptologos.cc/logos/moonriver-movr-logo.png", color: "#6A2D9E", blockTime: 12, averageGasPrice: "1000000000", maxGasPrice: "5000000000"},
		{id: "arbitrum-nova", name: "Arbitrum Nova", symbol: "ETH", type: TypeEVM, chainID: 42170, decimals: 18, explorer: "https://nova.arbiscan.io", RPCURL: "https://nova.arbitrum.io/rpc", APIURL: "https://api-nova.arbiscan.io", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, stableCoins: []string{"USDT", "USDC", "DAI"}, nativeToken: "ETH", logoURL: "https://cryptologos.cc/logos/arbitrum-arb-logo.png", color: "#28A0F0", blockTime: 1, averageGasPrice: "0.001", maxGasPrice: "0.01"},
		{id: "astar", name: "Astar", symbol: "ASTR", type: TypeEVM, chainID: 592, decimals: 18, explorer: "https://blockscout.com/astar", RPCURL: "https://rpc.astar.network", APIURL: "https://api.astarscan.io", WSSURL: "wss://ws.astar.network", confirmations: 15, minTransfer: 0.01, maxTransfer: 100000, supportsEIP1559: true, stableCoins: []string{"USDT", "USDC", "DAI"}, nativeToken: "ASTR", logoURL: "https://cryptologos.cc/logos/astar-astr-logo.png", color: "#03C3EB", blockTime: 12, averageGasPrice: "10000000000", maxGasPrice: "50000000000"},
		{id: "shiden", name: "Shiden", symbol: "SDN", type: TypeEVM, chainID: 336, decimals: 18, explorer: "https://blockscout.com/shiden", RPCURL: "https://rpc.shiden.astar.network", confirmations: 15, minTransfer: 0.01, maxTransfer: 100000, supportsEIP1559: true, stableCoins: []string{"USDT", "USDC"}, nativeToken: "SDN", logoURL: "https://cryptologos.cc/logos/shiden-sdn-logo.png", color: "#2D68E3", blockTime: 12, averageGasPrice: "10000000000", maxGasPrice: "50000000000"},
		{id: "harmony", name: "Harmony", symbol: "ONE", type: TypeEVM, chainID: 1666600000, decimals: 18, explorer: "https://explorer.harmony.one", RPCURL: "https://api.harmony.one", APIURL: "https://api.harmony.one", WSSURL: "wss://ws.harmony.one", confirmations: 15, minTransfer: 0.01, maxTransfer: 100000, supportsEIP1559: false, stableCoins: []string{"USDT", "USDC"}, nativeToken: "ONE", logoURL: "https://cryptologos.cc/logos/harmony-one-logo.png", color: "#00AEE9", blockTime: 2, averageGasPrice: "1000000000", maxGasPrice: "5000000000"},
		{id: "evmos", name: "Evmos", symbol: "EVMOS", type: TypeEVM, chainID: 9001, decimals: 18, explorer: "https://evm.evmos.org", RPCURL: "https://evmos-rpc.evmos.org", APIURL: "https://api.evmos.org", confirmations: 15, minTransfer: 0.01, maxTransfer: 100000, supportsEIP1559: false, stableCoins: []string{"USDT", "USDC"}, nativeToken: "EVMOS", logoURL: "https://cryptologos.cc/logos/evmos-evmos-logo.png", color: "#00AAFF", blockTime: 6, averageGasPrice: "20000000000", maxGasPrice: "100000000000"},
		{id: "kcc", name: "KuCoin Community Chain", symbol: "KCS", type: TypeEVM, chainID: 321, decimals: 18, explorer: "https://explorer.kcc.io", RPCURL: "https://rpc-mainnet.kcc.network", APIURL: "https://api.kcc.io", confirmations: 15, minTransfer: 0.01, maxTransfer: 100000, supportsEIP1559: false, stableCoins: []string{"USDT", "USDC"}, nativeToken: "KCS", logoURL: "https://cryptologos.cc/logos/kucoin-token-kcs-logo.png", color: "#1A1A1A", blockTime: 3, averageGasPrice: "50000000000", maxGasPrice: "200000000000"},
		{id: "okex-chain", name: "OKX Chain", symbol: "OKT", type: TypeEVM, chainID: 66, decimals: 18, explorer: "https://www.oklink.com/oktc", RPCURL: "https://exchainrpc.okex.org", APIURL: "https://www.oklink.io", confirmations: 15, minTransfer: 0.01, maxTransfer: 100000, supportsEIP1559: false, stableCoins: []string{"USDT", "USDC"}, nativeToken: "OKT", logoURL: "https://cryptologos.cc/logos/okex-token-okt-logo.png", color: "#24720F", blockTime: 3, averageGasPrice: "50000000000", maxGasPrice: "200000000000"},
		{id: "hECO", name: "Huobi ECO Chain", symbol: "HT", type: TypeEVM, chainID: 128, decimals: 18, explorer: "https://hecoinfo.com", RPCURL: "https://http-mainnet.hecochain.com", APIURL: "https://api.hecoinfo.com", confirmations: 15, minTransfer: 0.01, maxTransfer: 100000, supportsEIP1559: false, stableCoins: []string{"USDT", "HUSD"}, nativeToken: "HT", logoURL: "https://cryptologos.cc/logos/huobi-token-ht-logo.png", color: "#227C5E", blockTime: 3, averageGasPrice: "5000000000", maxGasPrice: "20000000000"},
		{id: "ronin", name: "Ronin", symbol: "RON", type: TypeEVM, chainID: 2020, decimals: 18, explorer: "https://ronin-explorer.axieinfinity.com", RPCURL: "https://ronin-rpc.axieinfinity.com", APIURL: "https://api.ronin.com", confirmations: 15, minTransfer: 0.01, maxTransfer: 100000, supportsEIP1559: true, stableCoins: []string{"USDT", "USDC", "AXS"}, nativeToken: "RON", logoURL: "https://cryptologos.cc/logos/ronin-ron-logo.png", color: "#C4161C", blockTime: 3, averageGasPrice: "3000000000", maxGasPrice: "10000000000"},
		{id: "canto", name: "Canto", symbol: "CANTO", type: TypeEVM, chainID: 7700, decimals: 18, explorer: "https://evm.explorer.canto.io", RPCURL: "https://canto.gravitychain.io", APIURL: "https://api.canto.io", confirmations: 15, minTransfer: 0.01, maxTransfer: 100000, supportsEIP1559: false, stableCoins: []string{"USDT", "USDC"}, nativeToken: "CANTO", logoURL: "https://cryptologos.cc/logos/canto-canto-logo.png", color: "#00C8F0", blockTime: 5, averageGasPrice: "1000000000", maxGasPrice: "5000000000"},
		{id: "core", name: "Core", symbol: "CORE", type: TypeEVM, chainID: 1116, decimals: 18, explorer: "https://scan.coredao.org", RPCURL: "https://rpc.coredao.org", APIURL: "https://api.coredao.org", confirmations: 15, minTransfer: 0.01, maxTransfer: 100000, supportsEIP1559: true, stableCoins: []string{"USDT", "USDC"}, nativeToken: "CORE", logoURL: "https://cryptologos.cc/logos/core-dao-logo.png", color: "#A8E61D", blockTime: 1, averageGasPrice: "5000000000", maxGasPrice: "20000000000"},
		{id: "zhejiang", name: "Zhejiang", symbol: "ETH", type: TypeEVM, chainID: 260000, decimals: 18, explorer: "https://zhejiang.etherscan.io", RPCURL: "https://rpc.zhejiang.ethpandaops.io", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, nativeToken: "ETH", isTestnet: true, blockTime: 12},
		
		// === NON-EVM BLOCKCHAINS ===
		// Bitcoin & Bitcoin-like
		{id: "bitcoin", name: "Bitcoin", symbol: "BTC", type: TypeBitcoin, chainID: 0, decimals: 8, explorer: "https://blockstream.info", RPCURL: "https://blockstream.info/api", WSSURL: "wss://blockstream.info/ws", confirmations: 6, minTransfer: 0.0001, maxTransfer: 10000, stableCoins: []string{}, nativeToken: "BTC", logoURL: "https://cryptologos.cc/logos/bitcoin-btc-logo.png", color: "#F7931A", blockTime: 600},
		{id: "bitcoin-cash", name: "Bitcoin Cash", symbol: "BCH", type: TypeBitcoin, chainID: 0, decimals: 8, explorer: "https://blockchair.com/bitcoin-cash", RPCURL: "https://bch.loping.net", confirmations: 10, minTransfer: 0.0001, maxTransfer: 10000, stableCoins: []string{}, nativeToken: "BCH", logoURL: "https://cryptologos.cc/logos/bitcoin-cash-bch-logo.png", color: "#8DC351", blockTime: 600},
		{id: "litecoin", name: "Litecoin", symbol: "LTC", type: TypeBitcoin, chainID: 0, decimals: 8, explorer: "https://blockchair.com/litecoin", RPCURL: "https://litecoin-rpc.loping.net", confirmations: 12, minTransfer: 0.0001, maxTransfer: 10000, stableCoins: []string{}, nativeToken: "LTC", logoURL: "https://cryptologos.cc/logos/litecoin-ltc-logo.png", color: "#BFBBBB", blockTime: 150},
		{id: "dogecoin", name: "Dogecoin", symbol: "DOGE", type: TypeBitcoin, chainID: 0, decimals: 8, explorer: "https://blockchair.com/dogecoin", RPCURL: "https://dogecoin-rpc.loping.net", confirmations: 60, minTransfer: 1, maxTransfer: 1000000, stableCoins: []string{}, nativeToken: "DOGE", logoURL: "https://cryptologos.cc/logos/dogecoin-doge-logo.png", color: "#C2A633", blockTime: 60},
		{id: "bitcoin-sv", name: "Bitcoin SV", symbol: "BSV", type: TypeBitcoin, chainID: 0, decimals: 8, explorer: "https://blockchair.com/bitcoin-sv", RPCURL: "https://bsv.loping.net", confirmations: 10, minTransfer: 0.0001, maxTransfer: 10000, stableCoins: []string{}, nativeToken: "BSV", logoURL: "https://cryptologos.cc/logos/bitcoin-sv-bsv-logo.png", color: "#EAB300", blockTime: 600},
		
		// Solana & Solana-like
		{id: "solana", name: "Solana", symbol: "SOL", type: TypeSolana, chainID: 101, decimals: 9, explorer: "https://solscan.io", RPCURL: "https://api.mainnet-beta.solana.com", APIURL: "https://api.solscan.io", WSSURL: "wss://api.mainnet-beta.solana.com", confirmations: 32, minTransfer: 0.0001, maxTransfer: 10000, supportsEIP1559: true, stableCoins: []string{"USDT", "USDC", "DAI"}, nativeToken: "SOL", logoURL: "https://cryptologos.cc/logos/solana-sol-logo.png", color: "#9945FF", blockTime: 0.4},
		{id: "solana-devnet", name: "Solana Devnet", symbol: "SOL", type: TypeSolana, chainID: 103, decimals: 9, explorer: "https://solscan.io/?cluster=devnet", RPCURL: "https://api.devnet.solana.com", isTestnet: true, confirmations: 32, minTransfer: 0.0001, maxTransfer: 10000, nativeToken: "SOL", blockTime: 0.4},
		
		// Cosmos & Cosmos-like
		{id: "cosmos-hub", name: "Cosmos Hub", symbol: "ATOM", type: TypeCosmos, chainID: 1, decimals: 6, explorer: "https://mintscan.io/cosmoshub", RPCURL: "https://rpc.cosmos.network", WSSURL: "wss://rpc.cosmos.network", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, stableCoins: []string{"USDC"}, nativeToken: "ATOM", logoURL: "https://cryptologos.cc/logos/cosmos-atom-logo.png", color: "#2E3148", blockTime: 7},
		{id: "osmosis", name: "Osmosis", symbol: "OSMO", type: TypeCosmos, chainID: 1, decimals: 6, explorer: "https://mintscan.io/osmosis", RPCURL: "https://rpc-osmosis.ecostake.com", WSSURL: "wss://rpc-osmosis.ecostake.com", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, stableCoins: []string{"USDC", "USDT"}, nativeToken: "OSMO", logoURL: "https://cryptologos.cc/logos/osmosis-osmo-logo.png", color: "#5C6BC0", blockTime: 6},
		{id: "juno", name: "Juno", symbol: "JUNO", type: TypeCosmos, chainID: 1, decimals: 6, explorer: "https://mintscan.io/juno", RPCURL: "https://rpc-juno.ecostake.com", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, stableCoins: []string{}, nativeToken: "JUNO", logoURL: "https://cryptologos.cc/logos/juno-juno-logo.png", color: "#F62D2E", blockTime: 7},
		{id: "akash", name: "Akash Network", symbol: "AKT", type: TypeCosmos, chainID: 1, decimals: 6, explorer: "https://mintscan.io/akash", RPCURL: "https://rpc-akash.ecostake.com", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, stableCoins: []string{}, nativeToken: "AKT", logoURL: "https://cryptologos.cc/logos/akash-network-akt-logo.png", color: "#F62D2E", blockTime: 7},
		{id: "regen", name: "Regen Network", symbol: "REGEN", type: TypeCosmos, chainID: 1, decimals: 6, explorer: "https://mintscan.io/regen", RPCURL: "https://rpc-regen.ecostake.com", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, stableCoins: []string{}, nativeToken: "REGEN", logoURL: "https://cryptologos.cc/logos/regen-network-regen-logo.png", color: "#45B7E1", blockTime: 7},
		{id: "sentinel", name: "Sentinel", symbol: "DVPN", type: TypeCosmos, chainID: 1, decimals: 6, explorer: "https://mintscan.io/sentinel", RPCURL: "https://rpc-sentinel.ecostake.com", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, stableCoins: []string{}, nativeToken: "DVPN", logoURL: "https://cryptologos.cc/logos/sentinel-dvpn-logo.png", color: "#3D5A80", blockTime: 7},
		{id: "persistence", name: "Persistence", symbol: "XPRT", type: TypeCosmos, chainID: 1, decimals: 6, explorer: "https://mintscan.io/persistence", RPCURL: "https://rpc-persistence.ecostake.com", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, stableCoins: []string{}, nativeToken: "XPRT", logoURL: "https://cryptologos.cc/logos/persistence-xprt-logo.png", color: "#E90D2F", blockTime: 7},
		{id: "crypto-com-chain", name: "Crypto.com Chain", symbol: "CRO", type: TypeCosmos, chainID: 1, decimals: 8, explorer: "https://mintscan.io/crypto-com", RPCURL: "https://rpc-crypto-com.ecostake.com", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, stableCoins: []string{}, nativeToken: "CRO", logoURL: "https://cryptologos.cc/logos/crypto-com-coin-cro-logo.png", color: "#002D74", blockTime: 7},
		
		// Polkadot & Substrate
		{id: "polkadot", name: "Polkadot", symbol: "DOT", type: TypePolkadot, chainID: 0, decimals: 10, explorer: "https://polkadot.js.org/apps", RPCURL: "https://rpc.polkadot.io", WSSURL: "wss://rpc.polkadot.io", confirmations: 15, minTransfer: 0.01, maxTransfer: 100000, stableCoins: []string{}, nativeToken: "DOT", logoURL: "https://cryptologos.cc/logos/polkadot-new-dot-logo.png", color: "#E6007A", blockTime: 6},
		{id: "kusama", name: "Kusama", symbol: "KSM", type: TypeKusama, chainID: 0, decimals: 12, explorer: "https://polkadot.js.org/apps/?rpc=wss://kusama-rpc.polkadot.io", RPCURL: "https://kusama-rpc.polkadot.io", WSSURL: "wss://kusama-rpc.polkadot.io", confirmations: 15, minTransfer: 0.01, maxTransfer: 100000, stableCoins: []string{}, nativeToken: "KSM", logoURL: "https://cryptologos.cc/logos/kusama-ksm-logo.png", color: "#000000", blockTime: 6},
		{id: "astar", name: "Astar", symbol: "ASTR", type: TypePolkadot, chainID: 0, decimals: 18, explorer: "https://astar.subscan.io", RPCURL: "https://rpc.astar.network", confirmations: 15, minTransfer: 0.01, maxTransfer: 100000, stableCoins: []string{}, nativeToken: "ASTR", logoURL: "https://cryptologos.cc/logos/astar-astr-logo.png", color: "#03C3EB", blockTime: 12},
		{id: "shiden", name: "Shiden", symbol: "SDN", type: TypePolkadot, chainID: 0, decimals: 18, explorer: "https://shiden.subscan.io", RPCURL: "https://rpc.shiden.astar.network", confirmations: 15, minTransfer: 0.01, maxTransfer: 100000, stableCoins: []string{}, nativeToken: "SDN", logoURL: "https://cryptologos.cc/logos/shiden-sdn-logo.png", color: "#2D68E3", blockTime: 12},
		
		// NEAR Protocol
		{id: "near", name: "NEAR Protocol", symbol: "NEAR", type: TypeNear, chainID: 0, decimals: 24, explorer: "https://explorer.near.org", RPCURL: "https://rpc.mainnet.near.org", WSSURL: "wss://rpc.mainnet.near.org/ws", confirmations: 3, minTransfer: 0.01, maxTransfer: 100000, stableCoins: []string{"USDT", "USDC"}, nativeToken: "NEAR", logoURL: "https://cryptologos.cc/logos/near-near-logo.png", color: "#00C08B", blockTime: 1},
		
		// Aptos
		{id: "aptos", name: "Aptos", symbol: "APT", type: TypeAptos, chainID: 1, decimals: 8, explorer: "https://explorer.aptoslabs.com", RPCURL: "https://fullnode.mainnet.aptoslabs.com", WSSURL: "wss://fullnode.mainnet.aptoslabs.com", confirmations: 1, minTransfer: 0.01, maxTransfer: 100000, stableCoins: []string{"USDT", "USDC"}, nativeToken: "APT", logoURL: "https://cryptologos.cc/logos/aptos-apt-logo.png", color: "#14F195", blockTime: 1},
		
		// Sui
		{id: "sui", name: "Sui", symbol: "SUI", type: TypeSui, chainID: 1, decimals: 9, explorer: "https://suiscan.xyz", RPCURL: "https://fullnode.mainnet.sui.io", WSSURL: "wss://fullnode.mainnet.sui.io", confirmations: 1, minTransfer: 0.01, maxTransfer: 100000, stableCoins: []string{"USDT", "USDC"}, nativeToken: "SUI", logoURL: "https://cryptologos.cc/logos/sui-sui-logo.png", color: "#6FB0F2", blockTime: 1},
		
		// TON
		{id: "ton", name: "TON", symbol: "TON", type: TypeTon, chainID: 0, decimals: 9, explorer: "https://tonscan.org", RPCURL: "https://toncenter.com/api/v2/jsonRPC", APIURL: "https://tonapi.io", WSSURL: "wss://toncenter.com/ws", confirmations: 1, minTransfer: 0.01, maxTransfer: 100000, stableCoins: []string{"USDT", "USDC"}, nativeToken: "TON", logoURL: "https://cryptologos.cc/logos/toncoin-ton-logo.png", color: "#0098EA", blockTime: 5},
		
		// TRON
		{id: "tron", name: "TRON", symbol: "TRX", type: TypeTron, chainID: 728126428, decimals: 6, explorer: "https://tronscan.org", RPCURL: "https://api.trongrid.io", WSSURL: "wss://api.trongrid.io", confirmations: 19, minTransfer: 1, maxTransfer: 10000000, stableCoins: []string{"USDT", "USDC", "TUSD"}, nativeToken: "TRX", logoURL: "https://cryptologos.cc/logos/tron-trx-logo.png", color="#FF0013", blockTime: 3},
		
		// Algorand
		{id: "algorand", name: "Algorand", symbol: "ALGO", type: TypeAlgorand, chainID: 0, decimals: 6, explorer: "https://algoexplorer.io", RPCURL: "https://algoexplorerapi.io", WSSURL: "wss://ws.algorand.org", confirmations: 4, minTransfer: 0.01, maxTransfer: 1000000, stableCoins: []string{"USDC", "USDT"}, nativeToken: "ALGO", logoURL: "https://cryptologos.cc/logos/algorand-algo-logo.png", color="#000000", blockTime: 4},
		
		// Cardano
		{id: "cardano", name: "Cardano", symbol: "ADA", type: TypeCardano, chainID: 1, decimals: 6, explorer: "https://cardanoscan.io", RPCURL: "https://cardano-mainnet.blockfrost.io/api/v0", confirmations: 15, minTransfer: 1, maxTransfer: 10000000, stableCoins: []string{"ADA"}, nativeToken: "ADA", logoURL: "https://cryptologos.cc/logos/cardano-ada-logo.png", color="#0033AD", blockTime: 20},
		
		// Ripple (XRP Ledger)
		{id: "ripple", name: "XRP Ledger", symbol: "XRP", type: TypeRipple, chainID: 0, decimals: 6, explorer: "https://xrpscan.com", RPCURL: "https://s1.ripple.com:51234", WSSURL: "wss://s1.ripple.com/", confirmations: 4, minTransfer: 0.01, maxTransfer: 10000000, stableCoins: []string{"USD"}, nativeToken: "XRP", logoURL: "https://cryptologos.cc/logos/xrp-xrp-logo.png", color="#23292F", blockTime: 4},
		
		// Hedera
		{id: "hedera", name: "Hedera", symbol: "HBAR", type: TypeHedera, chainID: 0, decimals: 8, explorer: "https://hashscan.io", RPCURL: "https://mainnet-preview-api.hedera.com", confirmations: 10, minTransfer: 0.01, maxTransfer: 1000000, stableCoins: []string{"USDC", "USDT"}, nativeToken: "HBAR", logoURL: "https://cryptologos.cc/logos/hedera-hbar-logo.png", color="#00EEC1", blockTime: 2},
		
		// IoTeX
		{ID: "iotex", name: "IoTeX", symbol: "IOTX", type: TypeIoTeX, chainID: 4689, decimals: 18, explorer: "https://iotexscan.io", RPCURL: "https://rpc.iotex.io", confirmations: 15, minTransfer: 0.01, maxTransfer: 1000000, stableCoins: []string{"USDT", "USDC"}, nativeToken: "IOTX", logoURL: "https://cryptologos.cc/logos/iotex-iotx-logo.png", color="#00D400", blockTime: 5},
		
		// Injective
		{id: "injective", name: "Injective", symbol: "INJ", type: TypeInjective, chainID: 1, decimals: 18, explorer: "https://explorer.injective.network", RPCURL: "https://public.api.injective.network", WSSURL: "wss://public.ws.injective.network", confirmations: 15, minTransfer: 0.01, maxTransfer: 1000000, stableCoins: []string{"USDT", "USDC"}, nativeToken: "INJ", logoURL: "https://cryptologos.cc/logos/injective-inj-logo.png", color="#00F2FE", blockTime: 1},
		
		// Sei
		{id: "sei", name: "Sei", symbol: "SEI", type: TypeSei, chainID: 1, decimals: 6, explorer: "https://www.seiscan.app", RPCURL: "https://rpc.sei.io", WSSURL: "wss://ws.sei.io", confirmations: 15, minTransfer: 0.01, maxTransfer: 1000000, stableCoins: []string{"USDC", "USDT"}, nativeToken: "SEI", logoURL: "https://cryptologos.cc/logos/sei-sei-logo.png", color="#9E8DFF", blockTime: 1},
		
		// Terra
		{id: "terra", name: "Terra", symbol: "LUNA", type: TypeTerra, chainID: 1, decimals: 6, explorer: "https://finder.terra.money", RPCURL: "https://terra-rpc.lavenderfive.com", WSSURL: "wss://terra-rpc.lavenderfive.com:443/ws", confirmations: 15, minTransfer: 0.01, maxTransfer: 1000000, stableCoins: []string{"UST", "USDC"}, nativeToken: "LUNA", logoURL: "https://cryptologos.cc/logos/terra-luna-logo.png", color="#2F2F2F", blockTime: 6},
		
		// === TESTNETS ===
		{id: "ethereum-sepolia", name: "Ethereum Sepolia", symbol: "ETH", type: TypeEVM, chainID: 11155111, decimals: 18, explorer: "https://sepolia.etherscan.io", RPCURL: "https://sepolia.infura.io/v3/", confirmations: 12, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, nativeToken: "ETH", isTestnet: true, blockTime: 12},
		{id: "ethereum-goerli", name: "Ethereum Goerli", symbol: "ETH", type: TypeEVM, chainID: 5, decimals: 18, explorer: "https://goerli.etherscan.io", RPCURL: "https://goerli.infura.io/v3/", confirmations: 12, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, nativeToken: "ETH", isTestnet: true, blockTime: 12},
		{id: "polygon-mumbai", name: "Polygon Mumbai", symbol: "MATIC", type: TypeEVM, chainID: 80001, decimals: 18, explorer: "https://mumbai.polygonscan.com", RPCURL: "https://rpc-mumbai.maticvigil.com", confirmations: 15, minTransfer: 0.01, maxTransfer: 100000, supportsEIP1559: true, nativeToken: "MATIC", isTestnet: true, blockTime: 2},
		{id: "arbitrum-goerli", name: "Arbitrum Goerli", symbol: "ETH", type: TypeEVM, chainID: 421613, decimals: 18, explorer: "https://goerli.arbiscan.io", RPCURL: "https://goerli-rollup.arbitrum.io/rpc", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, nativeToken: "ETH", isTestnet: true, blockTime: 1},
		{id: "optimism-goerli", name: "Optimism Goerli", symbol: "ETH", type: TypeEVM, chainID: 420, decimals: 18, explorer: "https://goerli-optimistic.etherscan.io", RPCURL: "https://goerli.optimism.io", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, nativeToken: "ETH", isTestnet: true, blockTime: 2},
		{id: "base-goerli", name: "Base Goerli", symbol: "ETH", type: TypeEVM, chainID: 84531, decimals: 18, explorer: "https://goerli.basescan.org", RPCURL: "https://goerli.base.org", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, nativeToken: "ETH", isTestnet: true, blockTime: 2},
		{id: "avalanche-fuji", name: "Avalanche Fuji", symbol: "AVAX", type: TypeEVM, chainID: 43113, decimals: 18, explorer: "https://testnet.snowtrace.io", RPCURL: "https://api.avax-test.network/ext/bc/C/rpc", confirmations: 15, minTransfer: 0.01, maxTransfer: 100000, supportsEIP1559: true, nativeToken: "AVAX", isTestnet: true, blockTime: 2},
		{id: "fantom-testnet", name: "Fantom Testnet", symbol: "FTM", type: TypeEVM, chainID: 4002, decimals: 18, explorer: "https://testnet.ftmscan.com", RPCURL: "https://rpc.testnet.fantom.network", confirmations: 15, minTransfer: 0.01, maxTransfer: 100000, nativeToken: "FTM", isTestnet: true, blockTime: 1},
		{id: "bsc-testnet", name: "BNB Smart Chain Testnet", symbol: "BNB", type: TypeEVM, chainID: 97, decimals: 18, explorer: "https://testnet.bscscan.com", RPCURL: "https://data-seed-prebsc-1-s1.binance.org:8545", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, nativeToken: "BNB", isTestnet: true, blockTime: 3},
		{id: "linea-goerli", name: "Linea Goerli", symbol: "ETH", type: TypeEVM, chainID: 59140, decimals: 18, explorer: "https://goerli.lineascan.build", RPCURL: "https://rpc.goerli.linea.build", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, nativeToken: "ETH", isTestnet: true, blockTime: 2},
		{id: "scroll-sepolia", name: "Scroll Sepolia", symbol: "ETH", type: TypeEVM, chainID: 534351, decimals: 18, explorer: "https://sepolia.scrollscan.com", RPCURL: "https://sepolia-rpc.scroll.io", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, nativeToken: "ETH", isTestnet: true, blockTime: 3},
		{id: "zksync-sepolia", name: "zkSync Era Sepolia", symbol: "ETH", type: TypeEVM, chainID: 300, decimals: 18, explorer: "https://sepolia.zkscan.io", RPCURL: "https://sepolia.era.zksync.io", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: false, nativeToken: "ETH", isTestnet: true, blockTime: 1},
		{id: "polygon-zkevm-cardona", name: "Polygon zkEVM Cardona", symbol: "ETH", type: TypeEVM, chainID: 2442, decimals: 18, explorer: "https://cardona.zkevm.polygonscan.com", RPCURL: "https://rpc.cardona.zkevm.polygon.technology", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, nativeToken: "ETH", isTestnet: true, blockTime: 1},
		{id: "starknet-sepolia", name: "Starknet Sepolia", symbol: "ETH", type: TypeStarknet, chainID: 0, decimals: 18, explorer: "https://sepolia.starkscan.co", RPCURL: "https://starknet-sepolia.public.blastapi.io", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, nativeToken: "ETH", isTestnet: true, blockTime: 5},
		{id: "celo-alfajores", name: "Celo Alfajores", symbol: "CELO", type: TypeEVM, chainID: 44787, decimals: 18, explorer: "https://alfajores.celoscan.io", RPCURL: "https://alfajores-forno.celo-testnet.org", confirmations: 15, minTransfer: 0.01, maxTransfer: 100000, supportsEIP1559: true, nativeToken: "CELO", isTestnet: true, blockTime: 5},
		{id: "cronos-testnet", name: "Cronos Testnet", symbol: "CRO", type: TypeEVM, chainID: 338, decimals: 18, explorer: "https://testnet.cronoscan.com", RPCURL: "https://evm-t3.cronos.org", confirmations: 15, minTransfer: 0.01, maxTransfer: 100000, supportsEIP1559: false, nativeToken: "CRO", isTestnet: true, blockTime: 5},
		{id: "kava-testnet", name: "Kava Testnet", symbol: "KAVA", type: TypeEVM, chainID: 2221, decimals: 18, explorer: "https://kavascan.com/testnet", RPCURL: "https://evm.testnet.kava.io", confirmations: 15, minTransfer: 0.01, maxTransfer: 100000, nativeToken: "KAVA", isTestnet: true, blockTime: 5},
		{id: "gnosis-chenilla", name: "Gnosis Chiado", symbol: "XDAI", type: TypeEVM, chainID: 10200, decimals: 18, explorer: "https://blockscout.com/gnosis/chiado", RPCURL: "https://rpc.chiado.gnosis.io", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, nativeToken: "XDAI", isTestnet: true, blockTime: 5},
		{id: "moonbase-alpha", name: "Moonbeam Alpha", symbol: "GLMR", type: TypeEVM, chainID: 1287, decimals: 18, explorer: "https://moonbase.moonscan.io", RPCURL: "https://rpc.api.moonbase.moonbeam.network", confirmations: 15, minTransfer: 0.01, maxTransfer: 100000, supportsEIP1559: true, nativeToken: "GLMR", isTestnet: true, blockTime: 12},
		{id: "shibuya", name: "Shibuya", symbol: "SBY", type: TypeEVM, chainID: 81, decimals: 18, explorer: "https://blockscout.com/shibuya", RPCURL: "https://rpc.shibuya.astar.network", confirmations: 15, minTransfer: 0.01, maxTransfer: 100000, supportsEIP1559: true, nativeToken: "SBY", isTestnet: true, blockTime: 12},
		{id: "harmony-testnet", name: "Harmony Testnet", symbol: "ONE", type: TypeEVM, chainID: 1666700000, decimals: 18, explorer: "https://explorer.testnet.harmony.one", RPCURL: "https://api.testnet.harmony.one", confirmations: 15, minTransfer: 0.01, maxTransfer: 100000, nativeToken: "ONE", isTestnet: true, blockTime: 2},
		
		// Additional Layer 2s and Emerging Chains
		{id: "taiko", name: "Taiko", symbol: "ETH", type: TypeTaiko, chainID: 167000, decimals: 18, explorer: "https://taikoscan.io", RPCURL: "https://rpc.taiko.xyz", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, stableCoins: []string{"USDC", "USDT"}, nativeToken: "ETH", logoURL: "https://cryptologos.cc/logos/taiko-tko-logo.png", color: "#E5017A", blockTime: 2},
		{id: "Mode", name: "Mode", symbol: "ETH", type: TypeEVM, chainID: 34443, decimals: 18, explorer: "https://modescan.io", RPCURL: "https://mainnet.mode.network", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, stableCoins: []string{"USDC", "USDT"}, nativeToken: "ETH", logoURL: "https://cryptologos.cc/logos/mode-mode-logo.png", color: "#00DCF5", blockTime: 2},
		{id: "orderly", name: "Orderly", symbol: "ETH", type: TypeEVM, chainID: 291, decimals: 18, explorer: "https://orderly-explorer.xyz", RPCURL: "https://rpc.orderly.network", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, stableCoins: []string{"USDC", "USDT"}, nativeToken: "ETH", logoURL: "https://cryptologos.cc/logos/orderly-ord-logo.png", color: "#1E1E1E", blockTime: 2},
		{id: "redstone", name: "Redstone", symbol: "ETH", type: TypeEVM, chainID: 196, decimals: 18, explorer: "https://explorer.redstone.xyz", RPCURL: "https://rpc.redstone.xyz", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, stableCoins: []string{"USDC", "USDT"}, nativeToken: "ETH", logoURL: "https://cryptologos.cc/logos/redstone-red-logo.png", color: "#2E30D4", blockTime: 2},
		{id: "Fraxtal", name: "Fraxtal", symbol: "ETH", type: TypeEVM, chainID: 252, decimals: 18, explorer: "https://fraxscan.com", RPCURL: "https://rpc.fraxtal.com", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, stableCoins: []string{"USDC", "USDT"}, nativeToken: "ETH", logoURL: "https://cryptologos.cc/logos/fraxtal-frxt-logo.png", color: "#00D4FF", blockTime: 2},
		{id: "bob", name: "BOB", symbol: "ETH", type: TypeEVM, chainID: 60808, decimals: 18, explorer: "https://bobscan.com", RPCURL: "https://rpc.boba.network", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, stableCoins: []string{"USDC", "USDT"}, nativeToken: "ETH", logoURL: "https://cryptologos.cc/logos/bob-bob-logo.png", color: "#A8E61D", blockTime: 2},
		{id: "zora", name: "Zora", symbol: "ETH", type: TypeEVM, chainID: 7777777, decimals: 18, explorer: "https://zora.superscan.network", RPCURL: "https://rpc.zora.energy", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, stableCoins: []string{"USDC", "USDT"}, nativeToken: "ETH", logoURL: "https://cryptologos.cc/logos/zora-zora-logo.png", color: "#E900FF", blockTime: 2},
		{id: "lyra", name: "Lyra", symbol: "ETH", type: TypeEVM, chainID: 957, decimals: 18, explorer: "https://explorer.lyra.finance", RPCURL: "https://rpc.lyra.finance", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, stableCoins: []string{"USDC", "USDT"}, nativeToken: "ETH", logoURL: "https://cryptologos.cc/logos/lyra-lyra-logo.png", color: "#5E27CD", blockTime: 2},
		{id: "manta", name: "Manta", symbol: "ETH", type: TypeEVM, chainID: 169, decimals: 18, explorer: "https://manta-pacific.subscan.io", RPCURL: "https://rpc.manta.network", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, stableCoins: []string{"USDC", "USDT"}, nativeToken: "ETH", logoURL: "https://cryptologos.cc/logos/manta-manta-logo.png", color: "#00A8E1", blockTime: 2},
		{id: "pgn", name: "Public Goods Network", symbol: "ETH", type: TypeEVM, chainID: 424, decimals: 18, explorer: "https://explorer.publicgoods.network", RPCURL: "https://rpc.publicgoods.network", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, stableCoins: []string{"USDC", "USDT"}, nativeToken: "ETH", logoURL: "https://cryptologos.cc/logos/pgn-pgn-logo.png", color: "#00D4FF", blockTime: 2},
		{id: "swell", name: "Swell", symbol: "ETH", type: TypeEVM, chainID: 1923, decimals: 18, explorer: "https://swellchain.io", RPCURL: "https://swell-mainnet.espravy.xyz", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, stableCoins: []string{"USDC", "USDT"}, nativeToken: "ETH", logoURL: "https://cryptologos.cc/logos/swell-swell-logo.png", color: "#E900FF", blockTime: 2},
		{id: "abstract", name: "Abstract", symbol: "ETH", type: TypeEVM, chainID: 2741, decimals: 18, explorer: "https://explorer.abstract.io", RPCURL: "https://api.mainnet.abs.xyz", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, stableCoins: []string{"USDC", "USDT"}, nativeToken: "ETH", logoURL: "https://cryptologos.cc/logos/abstract-abs-logo.png", color: "#FFFFFF", blockTime: 2},
		{id: "rarible", name: "Rarible", symbol: "RARI", type: TypeEVM, chainID: 1, decimals: 18, explorer: "https://rarible.com", RPCURL: "https://mainnet.etnrpc.io", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, stableCoins: []string{"USDC", "USDT"}, nativeToken: "RARI", logoURL: "https://cryptologos.cc/logos/rarible-rari-logo.png", color: "#F9C643", blockTime: 12},
		
		// More Cosmos Chains
		{id: "dymension", name: "Dymension", symbol: "DYM", type: TypeCosmos, chainID: 1, decimals: 18, explorer: "https://explorer.dymension.xyz", RPCURL: "https://rpc.dymension.xyz", confirmations: 15, minTransfer: 0.01, maxTransfer: 1000000, stableCoins: []string{"USDC"}, nativeToken: "DYM", logoURL: "https://cryptologos.cc/logos/dymension-dym-logo.png", color: "#F53B6E", blockTime: 6},
		{id: "celestia", name: "Celestia", symbol: "TIA", type: TypeCosmos, chainID: 1, decimals: 6, explorer: "https://celestia.explorers.guru", RPCURL: "https://rpc.celestia.org", confirmations: 15, minTransfer: 0.01, maxTransfer: 1000000, stableCoins: []string{}, nativeToken: "TIA", logoURL: "https://cryptologos.cc/logos/celestia-tia-logo.png", color: "#C1275D", blockTime: 6},
		{id: "noble", name: "Noble", symbol: "NOBLE", type: TypeCosmos, chainID: 1, decimals: 6, explorer: "https://noble.explorers.guru", RPCURL: "https://rpc.noble.strange.love", confirmations: 15, minTransfer: 0.01, maxTransfer: 1000000, stableCoins: []string{"USDC"}, nativeToken: "NOBLE", logoURL: "https://cryptologos.cc/logos/noble-noble-logo.png", color: "#E70090", blockTime: 6},
		{id: "quicksilver", name: "Quicksilver", symbol: "QCK", type: TypeCosmos, chainID: 1, decimals: 6, explorer: "https://quicksilver.explorers.guru", RPCURL: "https://rpc.quicksilver.zone", confirmations: 15, minTransfer: 0.01, maxTransfer: 1000000, stableCoins: []string{}, nativeToken: "QCK", logoURL: "https://cryptologos.cc/logos/quicksilver-qck-logo.png", color: "#00D4FF", blockTime: 6},
		{id: "stride", name: "Stride", symbol: "STRD", type: TypeCosmos, chainID: 1, decimals: 6, explorer: "https://stride.explorers.guru", RPCURL: "https://stride-rpc.polkachu.com", confirmations: 15, minTransfer: 0.01, maxTransfer: 1000000, stableCoins: []string{}, nativeToken: "STRD", logoURL: "https://cryptologos.cc/logos/stride-strd-logo.png", color: "#00D4FF", blockTime: 6},
		{id: "darc", name: "Darc", symbol: "DARC", type: TypeCosmos, chainID: 1, decimals: 6, explorer: "https://darc.explorers.guru", RPCURL: "https://mainnet.darc.io", confirmations: 15, minTransfer: 0.01, maxTransfer: 1000000, stableCoins: []string{}, nativeToken: "DARC", logoURL: "https://cryptologos.cc/logos/darc-darc-logo.png", color: "#00D4FF", blockTime: 6},
		
		// More Non-EVM
		{id: "vechain", name: "VeChain", symbol: "VET", type: TypeEVM, chainID: 1, decimals: 18, explorer: "https://vechainstats.com", RPCURL: "https://vechain-mainnet.everstake.one", confirmations: 15, minTransfer: 1, maxTransfer: 1000000000, stableCoins: []string{}, nativeToken: "VET", logoURL: "https://cryptologos.cc/logos/vechain-vet-logo.png", color: "#15BDFF", blockTime: 10},
		{id: "thorchain", name: "THORChain", symbol: "RUNE", type: TypeCosmos, chainID: 1, decimals: 8, explorer: "https://thorchain.net", RPCURL: "https://rpc.thorchain.info", confirmations: 15, minTransfer: 0.01, maxTransfer: 1000000, stableCoins: []string{"BTC", "ETH"}, nativeToken: "RUNE", logoURL: "https://cryptologos.cc/logos/thorchain-rune-logo.png", color: "#00F0FF", blockTime: 6},
		{id: "kava-chain", name: "Kava", symbol: "KAVA", type: TypeCosmos, chainID: 1, decimals: 6, explorer: "https://kavascan.com", RPCURL: "https://rpc-kava.zenchainlabs.io", confirmations: 15, minTransfer: 0.01, maxTransfer: 1000000, stableCoins: []string{"USDX"}, nativeToken: "KAVA", logoURL: "https://cryptologos.cc/logos/kava-kava-logo.png", color: "#FF5733", blockTime: 6},
		
		// Filecoin
		{id: "filecoin", name: "Filecoin", symbol: "FIL", type: TypeFilecoin, chainID: 314, decimals: 18, explorer: "https://filfox.io", RPCURL: "https://api.filecoin.io", confirmations: 15, minTransfer: 0.01, maxTransfer: 1000000, stableCoins: []string{}, nativeToken: "FIL", logoURL: "https://cryptologos.cc/logos/filecoin-fil-logo.png", color: "#0090FF", blockTime: 30},
		
		// Flare
		{id: "flare", name: "Flare", symbol: "FLR", type: TypeEVM, chainID: 14, decimals: 18, explorer: "https://flare-explorer.flare.network", RPCURL: "https://flare-api.flare.network", confirmations: 15, minTransfer: 0.01, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{"USDX"}, nativeToken: "FLR", logoURL: "https://cryptologos.cc/logos/flare-flare-logo.png", color: "#C1275D", blockTime: 1},
		
		// Celo
		{id: "celo", name: "Celo", symbol: "CELO", type: TypeCelo, chainID: 42220, decimals: 18, explorer: "https://explorer.celo.org", RPCURL: "https://forno.celo.org", confirmations: 15, minTransfer: 0.01, maxTransfer: 100000, stableCoins: []string{"cUSD", "cEUR"}, nativeToken: "CELO", logoURL: "https://cryptologos.cc/logos/celo-celo-logo.png", color: "#FCFF52", blockTime: 5},
		
		// EOS
		{id: "eos", name: "EOS", symbol: "EOS", type: TypeEos, chainID: 1, decimals: 4, explorer: "https://bloks.io", RPCURL: "https://api.eosla.com", confirmations: 15, minTransfer: 0.01, maxTransfer: 1000000, stableCoins: []string{}, nativeToken: "EOS", logoURL: "https://cryptologos.cc/logos/eos-eos-logo.png", color: "#000000", blockTime: 0.5},
		
		// Ontology
		{id: "ontology", name: "Ontology", symbol: "ONG", type: TypeEVM, chainID: 1, decimals: 18, explorer: "https://explorer.ont.io", RPCURL: "https://dappnode1.ont.io:10339", confirmations: 15, minTransfer: 0.01, maxTransfer: 1000000, stableCoins: []string{}, nativeToken: "ONG", logoURL: "https://cryptologos.cc/logos/ontology-ong-logo.png", color: "#00A0F0", blockTime: 10},
		
		// Kadena
		{id: "kadena", name: "Kadena", symbol: "KDA", type: TypeEVM, chainID: 0, decimals: 12, explorer: "https://explorer.kadena.io", RPCURL: "https://api.testnet.kadena.io", confirmations: 15, minTransfer: 1, maxTransfer: 1000000, stableCoins: []string{}, nativeToken: "KDA", logoURL: "https://cryptologos.cc/logos/kadena-kda-logo.png", color: "#000000", blockTime: 1},
		
		// Mina
		{id: "mina", name: "Mina", symbol: "MINA", type: TypeEVM, chainID: 1, decimals: 9, explorer: "https://minaexplorer.com", RPCURL: "https://api.minascan.io", confirmations: 15, minTransfer: 1, maxTransfer: 1000000, stableCoins: []string{}, nativeToken: "MINA", logoURL: "https://cryptologos.cc/logos/mina-mina-logo.png", color: "#E6BE8A", blockTime: 20},
		
		// Alephium
		{id: "alephium", name: "Alephium", symbol: "ALPH", type: TypeEVM, chainID: 0, decimals: 18, explorer: "https://explorer.alephium.org", RPCURL: "https://node.alephium.org", confirmations: 15, minTransfer: 0.01, maxTransfer: 1000000, stableCoins: []string{}, nativeToken: "ALPH", logoURL: "https://cryptologos.cc/logos/alephium-alph-logo.png", color: "#000000", blockTime: 1},
		
		// Kaspa
		{id: "kaspa", name: "Kaspa", symbol: "KAS", type: TypeBitcoin, chainID: 0, decimals: 8, explorer: "https://kaspa.org", RPCURL: "https://kaspad.kaspa.org", confirmations: 15, minTransfer: 0.01, maxTransfer: 1000000, stableCoins: []string{}, nativeToken: "KAS", logoURL: "https://cryptologos.cc/logos/kaspa-kas-logo.png", color: "#000000", blockTime: 1},
		
		// Sui Testnet
		{id: "sui-testnet", name: "Sui Testnet", symbol: "SUI", type: TypeSui, chainID: 2, decimals: 9, explorer: "https://suiscan.xyz/testnet", RPCURL: "https://fullnode.testnet.sui.io", isTestnet: true, confirmations: 1, minTransfer: 0.01, maxTransfer: 100000, nativeToken: "SUI", blockTime: 1},
		
		// Aptos Testnet
		{id: "aptos-testnet", name: "Aptos Testnet", symbol: "APT", type: TypeAptos, chainID: 2, decimals: 8, explorer: "https://explorer.aptoslabs.com/?network=testnet", RPCURL: "https://fullnode.testnet.aptoslabs.com", isTestnet: true, confirmations: 1, minTransfer: 0.01, maxTransfer: 100000, nativeToken: "APT", blockTime: 1},
		
		// NEAR Testnet
		{id: "near-testnet", name: "NEAR Testnet", symbol: "NEAR", type: TypeNear, chainID: 0, decimals: 24, explorer: "https://explorer.testnet.near.org", RPCURL: "https://rpc.testnet.near.org", isTestnet: true, confirmations: 3, minTransfer: 0.01, maxTransfer: 100000, nativeToken: "NEAR", blockTime: 1},
		
		// TON Testnet
		{id: "ton-testnet", name: "TON Testnet", symbol: "TON", type: TypeTon, chainID: 0, decimals: 9, explorer: "https://testnet.tonscan.org", RPCURL: "https://toncenter.com/api/v2/jsonRPC?testnet=true", isTestnet: true, confirmations: 1, minTransfer: 0.01, maxTransfer: 100000, nativeToken: "TON", blockTime: 5},
		
		// Kadena Testnet
		{id: "kadena-testnet", name: "Kadena Testnet", symbol: "KDA", type: TypeEVM, chainID: 0, decimals: 12, explorer: "https://explorer.testnet.kadena.io", RPCURL: "https://api.testnet.kadena.io", isTestnet: true, confirmations: 15, minTransfer: 1, maxTransfer: 1000000, nativeToken: "KDA", blockTime: 1},
	}

	for _, network := range networks {
		network.AddedAt = time.Now().Unix()
		network.UpdatedAt = time.Now().Unix()
		r.networks[network.ID] = network
		if network.ChainID > 0 || network.Type == TypeBitcoin || network.Type == TypeSolana || network.Type == TypeCosmos || network.Type == TypePolkadot || network.Type == TypeCardano || network.Type == TypeNear || network.Type == TypeAptos || network.Type == TypeSui || network.Type == TypeTon || network.Type == TypeAlgorand || network.Type == TypeRipple || network.Type == TypeTron || network.Type == TypeHedera || network.Type == TypeIoTeX || network.Type == TypeInjective || network.Type == TypeSei || network.Type == TypeTerra || network.Type == TypeFilecoin || network.Type == TypeEos || network.Type == TypeStarknet || network.Type == TypeTaiko || network.Type == TypeKusama || network.Type == TypeGnosis || network.Type == TypeMoonbeam || network.Type == TypeBase || network.Type == TypeBlast || network.Type == TypeLinea || network.Type == TypeScroll || network.Type == TypeMantle || network.Type == TypeMetis || network.Type == TypeZksync || network.Type == TypePolygon || network.Type == TypeOptimism || network.Type == TypeCelo || network.Type == TypeCronos || network.Type == TypeKava || network.Type == TypeCore {
			r.chainIDs[network.ChainID] = network.ID
		}
	}
}

// GetNetwork returns a network by ID
func (r *BlockchainRegistry) GetNetwork(id string) (*Network, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	network, ok := r.networks[id]
	return network, ok
}

// GetNetworkByChainID returns a network by chain ID
func (r *BlockchainRegistry) GetNetworkByChainID(chainID int64) (*Network, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.chainIDs[chainID]
	if !ok {
		return nil, false
	}
	network, ok := r.networks[id]
	return network, ok
}

// GetAllNetworks returns all networks
func (r *BlockchainRegistry) GetAllNetworks() []*Network {
	r.mu.RLock()
	defer r.mu.RUnlock()
	networks := make([]*Network, 0, len(r.networks))
	for _, network := range r.networks {
		networks = append(networks, network)
	}
	return networks
}

// GetNetworksByType returns networks by type
func (r *BlockchainRegistry) GetNetworksByType(blockchainType BlockchainType) []*Network {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var networks []*Network
	for _, network := range r.networks {
		if network.Type == blockchainType {
			networks = append(networks, network)
		}
	}
	return networks
}

// GetActiveNetworks returns only active (non-testnet) networks
func (r *BlockchainRegistry) GetActiveNetworks() []*Network {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var networks []*Network
	for _, network := range r.networks {
		if !network.IsTestnet {
			networks = append(networks, network)
		}
	}
	return networks
}

// GetSupportedChains returns the count of supported chains
func (r *BlockchainRegistry) GetSupportedChains() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.networks)
}

// GetActiveChainCount returns the count of active (non-testnet) chains
func (r *BlockchainRegistry) GetActiveChainCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	count := 0
	for _, network := range r.networks {
		if !network.IsTestnet {
			count++
		}
	}
	return count
}

// AddNetwork adds a new network
func (r *BlockchainRegistry) AddNetwork(network *Network) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.networks[network.ID]; exists {
		return fmt.Errorf("network %s already exists", network.ID)
	}
	
	network.AddedAt = time.Now().Unix()
	network.UpdatedAt = time.Now().Unix()
	r.networks[network.ID] = network
	if network.ChainID > 0 {
		r.chainIDs[network.ChainID] = network.ID
	}
	
	return nil
}

// UpdateNetwork updates an existing network
func (r *BlockchainRegistry) UpdateNetwork(network *Network) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.networks[network.ID]; !exists {
		return fmt.Errorf("network %s not found", network.ID)
	}
	
	network.UpdatedAt = time.Now().Unix()
	r.networks[network.ID] = network
	return nil
}

// DeleteNetwork removes a network
func (r *BlockchainRegistry) DeleteNetwork(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	network, exists := r.networks[id]
	if !exists {
		return fmt.Errorf("network %s not found", id)
	}
	
	delete(r.networks, id)
	if network.ChainID > 0 {
		delete(r.chainIDs, network.ChainID)
	}
	
	return nil
}

// GetNetworkJSON returns network as JSON string
func (r *BlockchainRegistry) GetNetworkJSON(id string) (string, error) {
	network, ok := r.GetNetwork(id)
	if !ok {
		return "", fmt.Errorf("network not found")
	}
	
	data, err := json.Marshal(network)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetAllNetworksJSON returns all networks as JSON string
func (r *BlockchainRegistry) GetAllNetworksJSON() (string, error) {
	networks := r.GetAllNetworks()
	data, err := json.Marshal(networks)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
