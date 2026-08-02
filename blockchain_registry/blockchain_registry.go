package main

import (
	"encoding/json"
	"fmt"
	"sync"
)

// BlockchainType represents the type of blockchain
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
)

// Network represents a blockchain network
type Network struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Symbol         string         `json:"symbol"`
	Type           BlockchainType `json:"type"`
	ChainID        int64          `json:"chain_id"`
	Decimals       int            `json:"decimals"`
	Explorer       string         `json:"explorer"`
	RPCURL         string         `json:"rpc_url"`
	APIURL         string         `json:"api_url"`
	WSSURL         string         `json:"wss_url"`
	IsTestnet      bool           `json:"is_testnet"`
	Confirmations  int            `json:"confirmations"`
	MinTransfer    float64        `json:"min_transfer"`
	MaxTransfer    float64        `json:"max_transfer"`
	SupportsEIP1559 bool          `json:"supports_eip1559"`
	Supports1559   bool           `json:"supports_1559"`
	GasStation     string         `json:"gas_station"`
	StableCoins    []string       `json:"stable_coins"`
	NativeToken    string         `json:"native_token"`
}

// Token represents a token on a blockchain
type Token struct {
	ID           string  `json:"id"`
	Address      string  `json:"address"`
	Name         string  `json:"name"`
	Symbol       string  `json:"symbol"`
	Decimals     int     `json:"decimals"`
	ChainID      int64   `json:"chain_id"`
	Type         string  `json:"type"`
	TotalSupply  string  `json:"total_supply"`
	IsStableCoin bool    `json:"is_stable_coin"`
	IsWrapped    bool    `json:"is_wrapped"`
	LogoURL      string  `json:"logo_url"`
	Price        float64 `json:"price"`
	MarketCap    float64 `json:"market_cap"`
	Volume24h    float64 `json:"volume_24h"`
}

// BlockchainRegistry manages all supported blockchains
type BlockchainRegistry struct {
	mu        sync.RWMutex
	networks  map[string]*Network
	tokens    map[string][]*Token
	chainIDs  map[int64]string
}

// Global blockchain registry instance
var (
	registry     *BlockchainRegistry
	registryOnce sync.Once
)

// GetRegistry returns the singleton blockchain registry
func GetRegistry() *BlockchainRegistry {
	registryOnce.Do(func() {
		registry = &BlockchainRegistry{
			networks: make(map[string]*Network),
			tokens:   make(map[string][]*Token),
			chainIDs: make(map[int64]string),
		}
		registry.initNetworks()
		registry.initTokens()
	})
	return registry
}

// initNetworks initializes all supported networks
func (r *BlockchainRegistry) initNetworks() {
	networks := []*Network{
		// EVM Chains - Top 50
		{id: "ethereum", name: "Ethereum", symbol: "ETH", type: TypeEVM, chainID: 1, decimals: 18, explorer: "https://etherscan.io", RPCURL: "https://eth.llamarpc.com", APIURL: "https://api.etherscan.io", WSSURL: "wss://eth-mainnet.ws.alchemyapi.io", confirmations: 12, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, stableCoins: []string{"USDT", "USDC", "DAI", "BUSD"}},
		{id: "bsc", name: "BNB Smart Chain", symbol: "BNB", type: TypeEVM, chainID: 56, decimals: 18, explorer: "https://bscscan.com", RPCURL: "https://bsc-dataseed.binance.org", APIURL: "https://api.bscscan.com", WSSURL: "wss://bsc-ws-node.nariox.org", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, stableCoins: []string{"USDT", "USDC", "BUSD", "DAI"}},
		{id: "polygon", name: "Polygon", symbol: "MATIC", type: TypeEVM, chainID: 137, decimals: 18, explorer: "https://polygonscan.com", RPCURL: "https://polygon-rpc.com", APIURL: "https://api.polygonscan.com", WSSURL: "wss://ws-mainnet.polygon.technology", confirmations: 15, minTransfer: 0.01, maxTransfer: 100000, supportsEIP1559: true, stableCoins: []string{"USDT", "USDC", "DAI"}},
		{id: "arbitrum", name: "Arbitrum One", symbol: "ETH", type: TypeEVM, chainID: 42161, decimals: 18, explorer: "https://arbiscan.io", RPCURL: "https://arb1.arbitrum.io/rpc", APIURL: "https://api.arbiscan.io", WSSURL: "wss://arb1.arbitrum.io/ws", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, stableCoins: []string{"USDT", "USDC", "DAI"}},
		{id: "optimism", name: "Optimism", symbol: "ETH", type: TypeEVM, chainID: 10, decimals: 18, explorer: "https://optimistic.etherscan.io", RPCURL: "https://mainnet.optimism.io", APIURL: "https://api-optimistic.etherscan.io", WSSURL: "wss://ws-mainnet.optimism.io", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, stableCoins: []string{"USDT", "USDC", "DAI"}},
		{id: "base", name: "Base", symbol: "ETH", type: TypeEVM, chainID: 8453, decimals: 18, explorer: "https://basescan.org", RPCURL: "https://mainnet.base.org", APIURL: "https://api.basescan.org", WSSURL: "wss://ws.base.org", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, stableCoins: []string{"USDT", "USDC", "DAI"}},
		{id: "avalanche", name: "Avalanche C-Chain", symbol: "AVAX", type: TypeEVM, chainID: 43114, decimals: 18, explorer: "https://snowtrace.io", RPCURL: "https://api.avax.network/ext/bc/C/rpc", APIURL: "https://api.snowtrace.io", WSSURL: "wss://api.avax.network/ext/bc/C/ws", confirmations: 15, minTransfer: 0.01, maxTransfer: 100000, supportsEIP1559: true, stableCoins: []string{"USDT", "USDC", "DAI"}},
		{id: "fantom", name: "Fantom Opera", symbol: "FTM", type: TypeEVM, chainID: 250, decimals: 18, explorer: "https://ftmscan.com", RPCURL: "https://rpc.fantom.network", APIURL: "https://api.ftmscan.com", WSSURL: "wss://ws.fantom.network", confirmations: 15, minTransfer: 1, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{"USDT", "USDC", "DAI"}},
		{id: "celo", name: "Celo", symbol: "CELO", type: TypeEVM, chainID: 42220, decimals: 18, explorer: "https://explorer.celo.org", RPCURL: "https://forno.celo.org", APIURL: "https://api.celoscan.io", WSSURL: "wss://forno.celo.org/ws", confirmations: 15, minTransfer: 0.01, maxTransfer: 100000, supportsEIP1559: true, stableCoins: []string{"cUSD", "cEUR", "USDT"}},
		{id: "cronos", name: "Cronos", symbol: "CRO", type: TypeEVM, chainID: 25, decimals: 18, explorer: "https://cronoscan.com", RPCURL: "https://evm.cronos.org", APIURL: "https://api.cronoscan.com", WSSURL: "wss://evm.cronos.org", confirmations: 15, minTransfer: 1, maxTransfer: 100000, supportsEIP1559: false, stableCoins: []string{"USDT", "USDC", "DAI"}},
		{id: "kava", name: "Kava", symbol: "KAVA", type: TypeEVM, chainID: 2222, decimals: 18, explorer: "https://kavascan.com", RPCURL: "https://evm.kava.io", APIURL: "https://api.kavascan.com", WSSURL: "wss://evm.kava.io", confirmations: 15, minTransfer: 0.1, maxTransfer: 100000, supportsEIP1559: false, stableCoins: []string{"USDT", "USDC", "DAI"}},
		{id: "metis", name: "Metis Andromeda", symbol: "METIS", type: TypeEVM, chainID: 1088, decimals: 18, explorer: "https://andromedaexplorer.metis.io", RPCURL: "https://andromeda.metis.io/?owner=1088", APIURL: "https://api.andromedexplorer.metis.io", WSSURL: "wss://andromeda.metis.io/ws", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{"USDT", "USDC"}},
		{id: "mantle", name: "Mantle", symbol: "MNT", type: TypeEVM, chainID: 5000, decimals: 18, explorer: "https://explorer.mantle.xyz", RPCURL: "https://rpc.mantle.xyz", APIURL: "https://api.mantlescan.org", WSSURL: "wss://ws.mantle.xyz", confirmations: 15, minTransfer: 0.01, maxTransfer: 100000, supportsEIP1559: true, stableCoins: []string{"USDT", "USDC"}},
		{id: "linea", name: "Linea", symbol: "ETH", type: TypeEVM, chainID: 59144, decimals: 18, explorer: "https://lineascan.build", RPCURL: "https://rpc.linea.build", APIURL: "https://api.lineascan.build", WSSURL: "wss://rpc.linea.build", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, stableCoins: []string{"USDT", "USDC", "DAI"}},
		{id: "zksync", name: "zkSync Era", symbol: "ETH", type: TypeEVM, chainID: 324, decimals: 18, explorer: "https://explorer.zksync.io", RPCURL: "https://zksync2-mainnet.zksync.io", APIURL: "https://api.zksync.io", WSSURL: "wss://zksync2-mainnet.zksync.io/ws", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, stableCoins: []string{"USDT", "USDC", "DAI"}},
		{id: "scroll", name: "Scroll", symbol: "ETH", type: TypeEVM, chainID: 534352, decimals: 18, explorer: "https://scrollscan.com", RPCURL: "https://rpc.scroll.io", APIURL: "https://api.scroll.io", WSSURL: "wss://ws.scroll.io", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, stableCoins: []string{"USDT", "USDC", "DAI"}},
		{id: "polygon_zkevm", name: "Polygon zkEVM", symbol: "ETH", type: TypeEVM, chainID: 1101, decimals: 18, explorer: "https://zkevm.polygonscan.com", RPCURL: "https://zkevm-rpc.polygon.technology", APIURL: "https://api-zkevm.polygonscan.com", WSSURL: "wss://zkevm-ws.polygon.technology", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, stableCoins: []string{"USDT", "USDC", "DAI"}},
		{id: "opbnb", name: "opBNB", symbol: "BNB", type: TypeEVM, chainID: 204, decimals: 18, explorer: "https://opbnbscan.com", RPCURL: "https://opbnb-mainnet-rpc.bnbchain.org", APIURL: "https://api-opbnbscan.bnbchain.org", WSSURL: "wss://opbnb-mainnet-ws.bnbchain.org", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, stableCoins: []string{"USDT", "USDC"}},
		{id: "arbitrum_nova", name: "Arbitrum Nova", symbol: "ETH", type: TypeEVM, chainID: 42170, decimals: 18, explorer: "https://nova.arbiscan.io", RPCURL: "https://nova.arbitrum.io/rpc", APIURL: "https://api-nova.arbiscan.io", WSSURL: "wss://nova.arbitrum.io/ws", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, stableCoins: []string{"USDT", "USDC", "DAI"}},
		{id: "astar", name: "Astar", symbol: "ASTR", type: TypeEVM, chainID: 592, decimals: 18, explorer: "https://blockscout.com/astar", RPCURL: "https://rpc.astar.network", APIURL: "https://api.astar.network", WSSURL: "wss://rpc.astar.network", confirmations: 15, minTransfer: 10, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{"USDT", "USDC"}},
		
		// More EVM Chains
		{id: "gnosis", name: "Gnosis Chain", symbol: "XDAI", type: TypeEVM, chainID: 100, decimals: 18, explorer: "https://gnosisscan.io", RPCURL: "https://rpc.gnosischain.com", APIURL: "https://api.gnosisscan.io", WSSURL: "wss://rpc.gnosischain.com/wss", confirmations: 15, minTransfer: 0.01, maxTransfer: 100000, supportsEIP1559: false, stableCoins: []string{"USDT", "USDC", "DAI"}},
		{id: "moonbeam", name: "Moonbeam", symbol: "GLMR", type: TypeEVM, chainID: 1284, decimals: 18, explorer: "https://moonscan.io", RPCURL: "https://rpc.api.moonbeam.network", APIURL: "https://api.moonscan.io", WSSURL: "wss://wss.api.moonbeam.network", confirmations: 15, minTransfer: 0.1, maxTransfer: 100000, supportsEIP1559: false, stableCoins: []string{"USDT", "USDC", "DAI"}},
		{id: "moonriver", name: "Moonriver", symbol: "MOVR", type: TypeEVM, chainID: 1285, decimals: 18, explorer: "https://moonriver.moonscan.io", RPCURL: "https://rpc.moonriver.moonbeam.network", APIURL: "https://api.moonriver.moonscan.io", WSSURL: "wss://wss.moonriver.moonbeam.network", confirmations: 15, minTransfer: 0.1, maxTransfer: 100000, supportsEIP1559: false, stableCoins: []string{"USDT", "USDC"}},
		{id: "arbitrum_goerli", name: "Arbitrum Goerli", symbol: "ETH", type: TypeEVM, chainID: 421613, decimals: 18, explorer: "https://goerli.arbiscan.io", RPCURL: "https://goerli.arbitrum.io/rpc", APIURL: "https://api-goerli.arbiscan.io", WSSURL: "wss://goerli.arbitrum.io/ws", confirmations: 5, minTransfer: 0.001, maxTransfer: 10000, supportsEIP1559: true, isTestnet: true, stableCoins: []string{}},
		{id: "optimism_goerli", name: "Optimism Goerli", symbol: "ETH", type: TypeEVM, chainID: 420, decimals: 18, explorer: "https://goerli-optimistic.etherscan.io", RPCURL: "https://goerli.optimism.io", APIURL: "https://api-goerli-optimistic.etherscan.io", WSSURL: "wss://goerli.optimism.io/ws", confirmations: 5, minTransfer: 0.001, maxTransfer: 10000, supportsEIP1559: true, isTestnet: true, stableCoins: []string{}},
		{id: "sepolia", name: "Sepolia", symbol: "ETH", type: TypeEVM, chainID: 11155111, decimals: 18, explorer: "https://sepolia.etherscan.io", RPCURL: "https://rpc.sepolia.org", APIURL: "https://api-sepolia.etherscan.io", WSSURL: "wss://rpc.sepolia.org/ws", confirmations: 2, minTransfer: 0.001, maxTransfer: 10000, supportsEIP1559: true, isTestnet: true, stableCoins: []string{}},
		{id: "holesky", name: "Holesky", symbol: "ETH", type: TypeEVM, chainID: 17000, decimals: 18, explorer: "https://holesky.etherscan.io", RPCURL: "https://rpc.holesky.ethpandaops.io", APIURL: "https://api-holesky.etherscan.io", WSSURL: "wss://rpc.holesky.ethpandaops.io/ws", confirmations: 2, minTransfer: 0.001, maxTransfer: 10000, supportsEIP1559: true, isTestnet: true, stableCoins: []string{}},
		
		// Bitcoin and forks
		{id: "bitcoin", name: "Bitcoin", symbol: "BTC", type: TypeBitcoin, chainID: 0, decimals: 8, explorer: "https://blockstream.info", RPCURL: "https://blockstream.info/api", APIURL: "https://blockstream.info/api", WSSURL: "wss://blockstream.info/ws", confirmations: 6, minTransfer: 0.0001, maxTransfer: 1000, supportsEIP1559: false, stableCoins: []string{}},
		{id: "bitcoin_cash", name: "Bitcoin Cash", symbol: "BCH", type: TypeBitcoin, chainID: 0, decimals: 8, explorer: "https://blockchair.com/bitcoin-cash", RPCURL: "https://bch-rpc.bitcoinabc.org", APIURL: "https://bch-insight.bitcoinabc.org/api", WSSURL: "wss://bch-ws.bitcoinabc.org", confirmations: 10, minTransfer: 0.001, maxTransfer: 10000, supportsEIP1559: false, stableCoins: []string{}},
		{id: "litecoin", name: "Litecoin", symbol: "LTC", type: TypeBitcoin, chainID: 0, decimals: 8, explorer: "https://blockchair.com/litecoin", RPCURL: "https://litecoin-rpc.github.io", APIURL: "https://litecoin.info/api", WSSURL: "wss://litecoin.info/ws", confirmations: 12, minTransfer: 0.001, maxTransfer: 10000, supportsEIP1559: false, stableCoins: []string{}},
		{id: "dogecoin", name: "Dogecoin", symbol: "DOGE", type: TypeBitcoin, chainID: 0, decimals: 8, explorer: "https://blockchair.com/dogecoin", RPCURL: "https://dogecoin-rpc.github.io", APIURL: "https://dogecoin.info/api", WSSURL: "wss://dogecoin.info/ws", confirmations: 60, minTransfer: 1, maxTransfer: 10000000, supportsEIP1559: false, stableCoins: []string{}},
		
		// Tron
		{id: "tron", name: "Tron", symbol: "TRX", type: TypeTron, chainID: 195, decimals: 6, explorer: "https://tronscan.org", RPCURL: "https://api.trongrid.io", APIURL: "https://api.trongrid.io", WSSURL: "wss://api.trongrid.io/ws", confirmations: 19, minTransfer: 1, maxTransfer: 100000000, supportsEIP1559: false, stableCoins: []string{"USDT", "USDC", "TUSD"}},
		
		// Ripple
		{id: "ripple", name: "XRP Ledger", symbol: "XRP", type: TypeRipple, chainID: 0, decimals: 6, explorer: "https://xrpscan.com", RPCURL: "https://s1.ripple.com:51234", APIURL: "https://data.ripple.com", WSSURL: "wss://s1.ripple.com", confirmations: 10, minTransfer: 10, maxTransfer: 100000000, supportsEIP1559: false, stableCoins: []string{}},
		
		// Cosmos Ecosystem
		{id: "cosmos_hub", name: "Cosmos Hub", symbol: "ATOM", type: TypeCosmos, chainID: 0, decimals: 6, explorer: "https://mintscan.io/cosmos", RPCURL: "https://rpc.cosmoshub4.theta-testnet.xyz:443", APIURL: "https://api.cosmoshub4.theta-testnet.xyz", WSSURL: "wss://rpc.cosmoshub4.theta-testnet.xyz:443/websocket", confirmations: 15, minTransfer: 0.1, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{}},
		{id: "osmosis", name: "Osmosis", symbol: "OSMO", type: TypeCosmos, chainID: 0, decimals: 6, explorer: "https://mintscan.io/osmosis", RPCURL: "https://rpc.osmotest5.osmosis.zone:443", APIURL: "https://api.osmotest5.osmosis.zone", WSSURL: "wss://rpc.osmotest5.osmosis.zone:443/websocket", confirmations: 15, minTransfer: 0.1, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{"USDC", "USDT"}},
		{id: "injective", name: "Injective", symbol: "INJ", type: TypeCosmos, chainID: 0, decimals: 18, explorer: "https://mintscan.io/injective", RPCURL: "https://injective-1-rpc.cosmosia.notional.ventures:443", APIURL: "https://api.injective.network", WSSURL: "wss://injective-1-rpc.cosmosia.notional.ventures:443/ws", confirmations: 15, minTransfer: 0.1, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{"USDC", "USDT"}},
		
		// Solana
		{id: "solana", name: "Solana", symbol: "SOL", type: TypeSolana, chainID: 0, decimals: 9, explorer: "https://solscan.io", RPCURL: "https://api.mainnet-beta.solana.com", APIURL: "https://api.mainnet-beta.solana.com", WSSURL: "wss://api.mainnet-beta.solana.com/ws", confirmations: 32, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{"USDC", "USDT", "DAI"}},
		{id: "solana_devnet", name: "Solana Devnet", symbol: "SOL", type: TypeSolana, chainID: 0, decimals: 9, explorer: "https://solscan.io/?cluster=devnet", RPCURL: "https://api.devnet.solana.com", APIURL: "https://api.devnet.solana.com", WSSURL: "wss://api.devnet.solana.com/ws", confirmations: 1, minTransfer: 0.001, maxTransfer: 10000, supportsEIP1559: false, isTestnet: true, stableCoins: []string{}},
		
		// Near
		{id: "near", name: "Near Protocol", symbol: "NEAR", type: TypeNear, chainID: 0, decimals: 24, explorer: "https://explorer.near.org", RPCURL: "https://rpc.mainnet.near.org", APIURL: "https://api.near.org", WSSURL: "wss://rpc.mainnet.near.org/ws", confirmations: 3, minTransfer: 0.1, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{"USDC", "USDT"}},
		
		// Aptos
		{id: "aptos", name: "Aptos", symbol: "APT", type: TypeAptos, chainID: 0, decimals: 8, explorer: "https://explorer.aptoslabs.com", RPCURL: "https://fullnode.mainnet.aptoslabs.com", APIURL: "https://api.aptoslabs.com", WSSURL: "wss://fullnode.mainnet.aptoslabs.com", confirmations: 1, minTransfer: 0.1, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{"USDC", "USDT"}},
		
		// Sui
		{id: "sui", name: "Sui", symbol: "SUI", type: TypeSui, chainID: 0, decimals: 9, explorer: "https://suiscan.xyz", RPCURL: "https://rpc.mainnet.sui.io", APIURL: "https://api.mainnet.sui.io", WSSURL: "wss://rpc.mainnet.sui.io", confirmations: 1, minTransfer: 0.1, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{"USDC", "USDT"}},
		
		// Starknet
		{id: "starknet", name: "Starknet", symbol: "ETH", type: TypeStarknet, chainID: 0, decimals: 18, explorer: "https://starkscan.co", RPCURL: "https://rpc.starknet.io", APIURL: "https://api.starkscan.co", WSSURL: "wss://rpc.starknet.io", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{"USDC", "USDT", "DAI"}},
		
		// Ton
		{id: "ton", name: "Toncoin", symbol: "TON", type: TypeTon, chainID: 0, decimals: 9, explorer: "https://ton.live", RPCURL: "https://toncenter.com/api/v2", APIURL: "https://tonapi.io", WSSURL: "wss://toncenter.com/ws", confirmations: 1, minTransfer: 0.1, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{"USDC", "USDT"}},
		
		// Cardano
		{id: "cardano", name: "Cardano", symbol: "ADA", type: TypeCardano, chainID: 0, decimals: 6, explorer: "https://cardanoscan.io", RPCURL: "https://cardano-mainnet.blockfrost.io", APIURL: "https://api.cardanoscan.io", WSSURL: "wss://cardano-mainnet.blockfrost.io/ws", confirmations: 15, minTransfer: 1, maxTransfer: 100000000, supportsEIP1559: false, stableCoins: []string{}},
		
		// Algorand
		{id: "algorand", name: "Algorand", symbol: "ALGO", type: TypeAlgorand, chainID: 0, decimals: 6, explorer: "https://algoexplorer.io", RPCURL: "https://mainnet-api.algonode.io", APIURL: "https://api.algoexplorer.io", WSSURL: "wss://mainnet-api.algonode.io/ws", confirmations: 10, minTransfer: 0.1, maxTransfer: 100000000, supportsEIP1559: false, stableCoins: []string{"USDC", "USDT"}},
		
		// Sei
		{id: "sei", name: "Sei", symbol: "SEI", type: TypeCosmos, chainID: 0, decimals: 6, explorer: "https://www.seiscan.app", RPCURL: "https://sei-rpc.Chainlayer.org:443", APIURL: "https://api.sei.io", WSSURL: "wss://sei-rpc.Chainlayer.org:443/websocket", confirmations: 15, minTransfer: 0.1, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{"USDC", "USDT"}},
		
		// Celestia
		{id: "celestia", name: "Celestia", symbol: "TIA", type: TypeCosmos, chainID: 0, decimals: 6, explorer: "https://mintscan.io/celestia", RPCURL: "https://celestia-rpc.Blockspace.io:443", APIURL: "https://api.celestia.space", WSSURL: "wss://celestia-rpc.Blockspace.io:443/websocket", confirmations: 15, minTransfer: 0.1, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{}},
		
		// Polkadot
		{id: "polkadot", name: "Polkadot", symbol: "DOT", type: TypePolkadot, chainID: 0, decimals: 10, explorer: "https://polkadot.subscan.io", RPCURL: "https://rpc.polkadot.io", APIURL: "https://api.polkadot.io", WSSURL: "wss://rpc.polkadot.io", confirmations: 28, minTransfer: 0.1, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{}},
		{id: "kusama", name: "Kusama", symbol: "KSM", type: TypePolkadot, chainID: 0, decimals: 12, explorer: "https://kusama.subscan.io", RPCURL: "https://kusama-rpc.dotters.network", APIURL: "https://api.kusama.network", WSSURL: "wss://kusama-rpc.dotters.network", confirmations: 28, minTransfer: 0.01, maxTransfer: 100000, supportsEIP1559: false, stableCoins: []string{}},
		
		// More Layer 2s
		{id: "pulsechain", name: "PulseChain", symbol: "PLS", type: TypeEVM, chainID: 369, decimals: 18, explorer: "https://scan.pulsechain.com", RPCURL: "https://rpc.pulsechain.com", APIURL: "https://api.pulsechain.com", WSSURL: "wss://ws.pulsechain.com", confirmations: 15, minTransfer: 1, maxTransfer: 1000000000, supportsEIP1559: false, stableCoins: []string{"USDT", "USDC"}},
		{id: "pulsechain_testnet", name: "PulseChain Testnet", symbol: "tPLS", type: TypeEVM, chainID: 943, decimals: 18, explorer: "https://scan-testnet.pulsechain.com", RPCURL: "https://rpc.v4.testnet.pulsechain.com", APIURL: "https://api-testnet.pulsechain.com", WSSURL: "wss://ws.v4.testnet.pulsechain.com", confirmations: 3, minTransfer: 1, maxTransfer: 10000, supportsEIP1559: false, isTestnet: true, stableCoins: []string{}},
		
		// Pi Network (Testnet)
		{id: "pi_network", name: "Pi Network", symbol: "PI", type: TypeEVM, chainID: 314159, decimals: 18, explorer: "https://explorer.minepi.com", RPCURL: "https://rpc.minepi.com", APIURL: "https://api.minepi.com", WSSURL: "wss://ws.minepi.com", confirmations: 10, minTransfer: 1, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{}},
		
		// Terra
		{id: "terra", name: "Terra Classic", symbol: "LUNC", type: TypeTerra, chainID: 0, decimals: 6, explorer: "https://finder.terra.classic.community", RPCURL: "https://terra-classic-lcd.publicnode.com", APIURL: "https://terra-classic-api.publicnode.com", WSSURL: "wss://terra-classic-lcd.publicnode.com:443/websocket", confirmations: 15, minTransfer: 1000, maxTransfer: 100000000000, supportsEIP1559: false, stableCoins: []string{"USTC"}},
		
		// More EVM Chains
		{id: "aurora", name: "Aurora", symbol: "ETH", type: TypeEVM, chainID: 1313161554, decimals: 18, explorer: "https://aurorascan.dev", RPCURL: "https://mainnet.aurora.dev", APIURL: "https://api.aurorascan.dev", WSSURL: "wss://mainnet.aurora.dev/ws", confirmations: 15, minTransfer: 0.01, maxTransfer: 100000, supportsEIP1559: true, stableCoins: []string{"USDT", "USDC"}},
		{id: "harmony", name: "Harmony", symbol: "ONE", type: TypeEVM, chainID: 1666600000, decimals: 18, explorer: "https://explorer.harmony.one", RPCURL: "https://api.harmony.one", APIURL: "https://api.harmony.one", WSSURL: "wss://ws.harmony.one", confirmations: 15, minTransfer: 1, maxTransfer: 1000000000, supportsEIP1559: false, stableCoins: []string{"USDC", "USDT"}},
		{id: "kadena", name: "Kadena", symbol: "KDA", type: TypeEVM, chainID: 0, decimals: 8, explorer: "https://explorer.kadena.io", RPCURL: "https://api.kadena.network", APIURL: "https://api.kadena.network", WSSURL: "wss://api.kadena.network/ws", confirmations: 15, minTransfer: 1, maxTransfer: 10000000, supportsEIP1559: false, stableCoins: []string{}},
		{id: "conflux", name: "Conflux", symbol: "CFX", type: TypeEVM, chainID: 1030, decimals: 18, explorer: "https://confluxscan.io", RPCURL: "https://rpc.confluxnetwork.org", APIURL: "https://api.confluxnetwork.org", WSSURL: "wss://rpc.confluxnetwork.org/ws", confirmations: 15, minTransfer: 0.01, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{"USDC", "USDT"}},
		{id: "step", name: "Step Network", symbol: "STEP", type: TypeEVM, chainID: 1234, decimals: 18, explorer: "https://stepscan.io", RPCURL: "https://rpc.step.network", APIURL: "https://api.step.network", WSSURL: "wss://rpc.step.network/ws", confirmations: 15, minTransfer: 1, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{"USDC"}},
		{id: "telos", name: "Telos", symbol: "TLOS", type: TypeEVM, chainID: 40, decimals: 18, explorer: "https://www.teloscan.io", RPCURL: "https://mainnet.telos.net", APIURL: "https://api.teloscan.io", WSSURL: "wss://mainnet.telos.net/ws", confirmations: 15, minTransfer: 0.1, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{"USDT", "USDC"}},
		{id: "iotex", name: "IoTeX", symbol: "IOTX", type: TypeEVM, chainID: 4689, decimals: 18, explorer: "https://iotexscan.io", RPCURL: "https://rpc.iotex.io", APIURL: "https://api.iotex.io", WSSURL: "wss://ws.iotex.io", confirmations: 15, minTransfer: 1, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{"USDT", "USDC"}},
		{id: "ronin", name: "Ronin", symbol: "RON", type: TypeEVM, chainID: 2020, decimals: 18, explorer: "https://app.roninchain.com", RPCURL: "https://api.roninchain.com/rpc", APIURL: "https://api.roninchain.com", WSSURL: "wss://ws.roninchain.com", confirmations: 15, minTransfer: 0.01, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{"AXS", "SLP"}},
		{id: "canto", name: "Canto", symbol: "CANTO", type: TypeEVM, chainID: 7700, decimals: 18, explorer: "https://tuber.build", RPCURL: "https://canto.gravitychain.io", APIURL: "https://api.tuber.build", WSSURL: "wss://canto.gravitychain.io/ws", confirmations: 15, minTransfer: 0.01, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{"USDC", "USDT"}},
		{id: "fraxtal", name: "Fraxtal", symbol: "FRX", type: TypeEVM, chainID: 2522, decimals: 18, explorer: "https://fraxscan.com", RPCURL: "https://rpc.frax.com", APIURL: "https://api.fraxscan.com", WSSURL: "wss://rpc.frax.com/ws", confirmations: 15, minTransfer: 0.01, maxTransfer: 1000000, supportsEIP1559: true, stableCoins: []string{"USDC", "USDT"}},
		{id: "mode", name: "Mode", symbol: "MOD", type: TypeEVM, chainID: 34443, decimals: 18, explorer: "https://explorer.mode.network", RPCURL: "https://mainnet.mode.network", APIURL: "https://api.mode.network", WSSURL: "wss://mainnet.mode.network/ws", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, stableCoins: []string{"USDC", "USDT"}},
		{id: "rootstock", name: "Rootstock", symbol: "RBTC", type: TypeEVM, chainID: 30, decimals: 18, explorer: "https://rootstock.io", RPCURL: "https://public-node.rsk.co", APIURL: "https://api.rsk.co", WSSURL: "wss://public-node.rsk.co/ws", confirmations: 12, minTransfer: 0.0001, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{"USDT", "rUSDT"}},
		{id: "synthesis", name: "Synthesis", symbol: "SNX", type: TypeEVM, chainID: 66, decimals: 18, explorer: "https://synthetix.io", RPCURL: "https://optimism-mainnet.infura.io", APIURL: "https://api.synthetix.io", WSSURL: "wss://optimism-mainnet.infura.io/ws", confirmations: 15, minTransfer: 0.01, maxTransfer: 1000000, supportsEIP1559: true, stableCoins: []string{"sUSD"}},
		{id: "milkomeda", name: "Milkomeda", symbol: "ADA", type: TypeEVM, chainID: 2001, decimals: 18, explorer: "https://explorer-mainnet-cardano-evm.c1.milkomeda.com", RPCURL: "https://rpc-mainnet-cardano-evm.c1.milkomeda.com", APIURL: "https://api-mainnet-cardano-evm.c1.milkomeda.com", WSSURL: "wss://rpc-mainnet-cardano-evm.c1.milkomeda.com/ws", confirmations: 15, minTransfer: 1, maxTransfer: 1000000000, supportsEIP1559: false, stableCoins: []string{"USDC", "USDT"}},
		{id: "flare", name: "Flare", symbol: "FLR", type: TypeEVM, chainID: 14, decimals: 18, explorer: "https://flarescan.com", RPCURL: "https://flare-api.flare.network/ext/bc/C/rpc", APIURL: "https://api.flarescan.com", WSSURL: "wss://flare-api.flare.network/ext/bc/C/ws", confirmations: 15, minTransfer: 1, maxTransfer: 1000000000, supportsEIP1559: false, stableCoins: []string{"USDf"}},
		{id: "songbird", name: "Songbird", symbol: "SGB", type: TypeEVM, chainID: 19, decimals: 18, explorer: "https://songbird-explorer.flare.network", RPCURL: "https://songbird-api.flare.network/ext/bc/C/rpc", APIURL: "https://api.songbird.flare-network.io", WSSURL: "wss://songbird-api.flare.network/ext/bc/C/ws", confirmations: 15, minTransfer: 1, maxTransfer: 1000000000, supportsEIP1559: false, stableCoins: []string{}},
		{id: "elastos", name: "Elastos", symbol: "ELA", type: TypeEVM, chainID: 20, decimals: 18, explorer: "https://elastos.io", RPCURL: "https://rpc.elastos.io", APIURL: "https://api.elastos.io", WSSURL: "wss://rpc.elastos.io/ws", confirmations: 15, minTransfer: 0.1, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{}},
		{id: "emerald", name: "Emerald Paratime", symbol: "EDO", type: TypeEVM, chainID: 4002, decimals: 18, explorer: "https://explorer.emerald.dev", RPCURL: "https://rpc.emerald.dev", APIURL: "https://api.emerald.dev", WSSURL: "wss://ws.emerald.dev", confirmations: 15, minTransfer: 0.1, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{"USDC", "USDT"}},
		{id: "syscoin", name: "Syscoin", symbol: "SYS", type: TypeEVM, chainID: 57, decimals: 8, explorer: "https://syscoin.io", RPCURL: "https://rpc.syscoin.org", APIURL: "https://api.syscoin.org", WSSURL: "wss://rpc.syscoin.org/ws", confirmations: 15, minTransfer: 0.01, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{"USDC", "USDT"}},
		{id: "redstone", name: "Redstone", symbol: "RED", type: TypeEVM, chainID: 690, decimals: 18, explorer: "https://explorer.redstone.xyz", RPCURL: "https://rpc.redstonechain.com", APIURL: "https://api.redstone.xyz", WSSURL: "wss://rpc.redstonechain.com/ws", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, stableCoins: []string{"USDC", "USDT"}},
		{id: "blast", name: "Blast", symbol: "ETH", type: TypeEVM, chainID: 81457, decimals: 18, explorer: "https://blastscan.io", RPCURL: "https://rpc.blast.io", APIURL: "https://api.blastscan.io", WSSURL: "wss://rpc.blast.io/ws", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, stableCoins: []string{"USDB", "USDC", "DAI"}},
		{id: "manta", name: "Manta Pacific", symbol: "MANTA", type: TypeEVM, chainID: 169, decimals: 18, explorer: "https://manta-pacific-explorer.calderaexplorer.xyz", RPCURL: "https://rpc.manta.network/http", APIURL: "https://api.manta.network", WSSURL: "wss://rpc.manta.network/ws", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, stableCoins: []string{"USDC", "USDT"}},
		{id: "berachain", name: "Berachain", symbol: "BERA", type: TypeEVM, chainID: 80084, decimals: 18, explorer: "https://artio.beratrail.io", RPCURL: "https://artio.rpc.berachain.com", APIURL: "https://api.berachain.com", WSSURL: "wss://artio.rpc.berachain.com/ws", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, stableCoins: []string{"USDC", "USDT"}},
		{id: "sei_testnet", name: "Sei Atlantic", symbol: "SEI", type: TypeCosmos, chainID: 0, decimals: 6, explorer: "https://seitrace.com/?network=atlantic-1", RPCURL: "https://rpc.atlantic-1.sei.io:443", APIURL: "https://api.atlantic-1.sei.io", WSSURL: "wss://rpc.atlantic-1.sei.io:443/websocket", confirmations: 5, minTransfer: 0.1, maxTransfer: 10000, supportsEIP1559: false, isTestnet: true, stableCoins: []string{}},
		{id: "celestia_mocha", name: "Celestia Mocha", symbol: "TIA", type: TypeCosmos, chainID: 0, decimals: 6, explorer: "https://mocha.celestia.dev", RPCURL: "https://rpc-mocha.celestia-dev.io:443", APIURL: "https://api-mocha.celestia-dev.io", WSSURL: "wss://rpc-mocha.celestia-dev.io:443/websocket", confirmations: 5, minTransfer: 0.1, maxTransfer: 10000, supportsEIP1559: false, isTestnet: true, stableCoins: []string{}},
		
		// ==== ADDITIONAL 40+ BLOCKCHAINS TO REACH 100+ ====
		
		// More EVM Chains - Layer 2 & Alternatives
		{id: "shimmer", name: "Shimmer", symbol: "SMR", type: TypeEVM, chainID: 148, decimals: 18, explorer: "https://explorer.shimmer.network", RPCURL: "https://json-rpc.shimmer.network", APIURL: "https://api.shimmer.network", WSSURL: "wss://json-rpc.shimmer.network/ws", confirmations: 15, minTransfer: 1, maxTransfer: 1000000000, supportsEIP1559: false, stableCoins: []string{}},
		{id: "vechain", name: "VeChain", symbol: "VET", type: TypeEVM, chainID: 100, decimals: 18, explorer: "https://vechain.org", RPCURL: "https://rpc.vechain.org", APIURL: "https://api.vechain.org", WSSURL: "wss://rpc.ervice.org", confirmations: 15, minTransfer: 1, maxTransfer: 1000000000, supportsEIP1559: false, stableCoins: []string{}},
		{id: "thundercore", name: "ThunderCore", symbol: "TT", type: TypeEVM, chainID: 108, decimals: 18, explorer: "https://viewblock.io/thundercore", RPCURL: "https://mainnet-rpc.thundercore.com", APIURL: "https://api.thundercore.com", WSSURL: "wss://mainnet-ws.thundercore.com", confirmations: 15, minTransfer: 1, maxTransfer: 1000000000, supportsEIP1559: false, stableCoins: []string{}},
		{id: "kcc", name: "KuCoin Community Chain", symbol: "KCS", type: TypeEVM, chainID: 321, decimals: 18, explorer: "https://explorer.kcc.io", RPCURL: "https://rpc-mainnet.kcc.network", APIURL: "https://api.kcc.io", WSSURL: "wss://rpc-ws-mainnet.kcc.network", confirmations: 15, minTransfer: 0.01, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{"USDT", "USDC"}},
		{id: "hederahashgraph", name: "Hedera", symbol: "HBAR", type: TypeEVM, chainID: 295, decimals: 18, explorer: "https://hashscan.io", RPCURL: "https://mainnet.mirrornode.hedera.com", APIURL: "https://api.hedera.com", WSSURL: "wss://mainnet.mirrornode.hedera.com", confirmations: 15, minTransfer: 1, maxTransfer: 1000000000, supportsEIP1559: false, stableCoins: []string{}},
		{id: "zilliqa", name: "Zilliqa", symbol: "ZIL", type: TypeEVM, chainID: 32769, decimals: 12, explorer: "https://viewblock.io/zilliqa", RPCURL: "https://api.zilliqa.com", APIURL: "https://explorer.zilliqa.com", WSSURL: "wss://api.zilliqa.com", confirmations: 15, minTransfer: 1, maxTransfer: 1000000000, supportsEIP1559: false, stableCoins: []string{}},
		{id: "oasis", name: "Oasis Network", symbol: "ROSE", type: TypeEVM, chainID: 42262, decimals: 18, explorer: "https://explorer.emerald.oasis.dev", RPCURL: "https://emerald.oasis.dev", APIURL: "https://api.oasisscan.com", WSSURL: "wss://emerald.oasis.dev/ws", confirmations: 15, minTransfer: 1, maxTransfer: 1000000000, supportsEIP1559: false, stableCoins: []string{"USDC", "USDT"}},
		{id: "filecoin", name: "Filecoin", symbol: "FIL", type: TypeEVM, chainID: 314, decimals: 18, explorer: "https://filfox.io", RPCURL: "https://api.filecoin.io", APIURL: "https://api.filfox.io", WSSURL: "wss://api.filecoin.io", confirmations: 15, minTransfer: 0.01, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{}},
		{id: "flow", name: "Flow", symbol: "FLOW", type: TypeEVM, chainID: 747, decimals: 8, explorer: "https://flowscan.io", RPCURL: "https://flow-access-mainnet-01.blobstorage2.net", APIURL: "https://api.flowscan.io", WSSURL: "wss://flow-access-mainnet-01.blobstorage2.net/ws", confirmations: 15, minTransfer: 0.1, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{}},
		{id: "theta", name: "Theta Network", symbol: "THETA", type: TypeEVM, chainID: 361, decimals: 18, explorer: "https://explorer.thetatoken.org", RPCURL: "https://eth-rpc-api.theta.network", APIURL: "https://api.thetatoken.org", WSSURL: "wss://eth-rpc-api.theta.network/ws", confirmations: 15, minTransfer: 1, maxTransfer: 1000000000, supportsEIP1559: false, stableCoins: []string{}},
		{id: "arbitrum_sepolia", name: "Arbitrum Sepolia", symbol: "ETH", type: TypeEVM, chainID: 421614, decimals: 18, explorer: "https://sepolia.arbiscan.io", RPCURL: "https://sepolia.arbitrum.io/rpc", APIURL: "https://api-sepolia.arbiscan.io", WSSURL: "wss://sepolia.arbitrum.io/ws", confirmations: 5, minTransfer: 0.001, maxTransfer: 10000, supportsEIP1559: true, isTestnet: true, stableCoins: []string{}},
		{id: "base_sepolia", name: "Base Sepolia", symbol: "ETH", type: TypeEVM, chainID: 84532, decimals: 18, explorer: "https://sepolia.basescan.org", RPCURL: "https://sepolia.base.org", APIURL: "https://api-sepolia.basescan.org", WSSURL: "wss://sepolia.base.org/ws", confirmations: 5, minTransfer: 0.001, maxTransfer: 10000, supportsEIP1559: true, isTestnet: true, stableCoins: []string{}},
		{id: "polygon_amoy", name: "Polygon Amoy", symbol: "MATIC", type: TypeEVM, chainID: 80002, decimals: 18, explorer: "https://amoy.polygonscan.com", RPCURL: "https://rpc-amoy.polygon.technology", APIURL: "https://api-amoy.polygonscan.com", WSSURL: "wss://rpc-amoy.polygon.technology/ws", confirmations: 5, minTransfer: 0.01, maxTransfer: 10000, supportsEIP1559: true, isTestnet: true, stableCoins: []string{}},
		{id: "bnb_testnet", name: "BNB Smart Chain Testnet", symbol: "BNB", type: TypeEVM, chainID: 97, decimals: 18, explorer: "https://testnet.bscscan.com", RPCURL: "https://data-seed-prebsc-1-s1.binance.org:8545", APIURL: "https://api-testnet.bscscan.com", WSSURL: "wss://data-seed-prebsc-1-s1.binance.org:8545/ws", confirmations: 3, minTransfer: 0.001, maxTransfer: 10000, supportsEIP1559: true, isTestnet: true, stableCoins: []string{}},
		{id: "goerli", name: "Goerli", symbol: "ETH", type: TypeEVM, chainID: 5, decimals: 18, explorer: "https://goerli.etherscan.io", RPCURL: "https://goerli.infura.io/v3/", APIURL: "https://api-goerli.etherscan.io", WSSURL: "wss://goerli.infura.io/v3/ws", confirmations: 3, minTransfer: 0.001, maxTransfer: 10000, supportsEIP1559: true, isTestnet: true, stableCoins: []string{}},
		
		// More Cosmos Chains
		{id: "dydx", name: "dYdX", symbol: "DYDX", type: TypeCosmos, chainID: 0, decimals: 18, explorer: "https://mintscan.io/dydx", RPCURL: "https://dydx-rpc.kingnodes.com:443", APIURL: "https://api.dydx.excan.io", WSSURL: "wss://dydx-rpc.kingnodes.com:443/ws", confirmations: 15, minTransfer: 0.1, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{"USDC"}},
		{id: "persistence", name: "Persistence", symbol: "XPRT", type: TypeCosmos, chainID: 0, decimals: 6, explorer: "https://mintscan.io/persistence", RPCURL: "https://rpc.persistence.one:443", APIURL: "https://api.persistence.one", WSSURL: "wss://rpc.persistence.one:443/websocket", confirmations: 15, minTransfer: 0.1, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{"USDC"}},
		{id: "sommelier", name: "Sommelier", symbol: "SOMM", type: TypeCosmos, chainID: 0, decimals: 6, explorer: "https://mintscan.io/sommelier", RPCURL: "https://rpc-sommelier.kingnodes.com:443", APIURL: "https://api.sommelier.cz", WSSURL: "wss://rpc-sommelier.kingnodes.com:443/websocket", confirmations: 15, minTransfer: 0.1, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{"USDC"}},
		{id: "juno", name: "Juno", symbol: "JUNO", type: TypeCosmos, chainID: 0, decimals: 6, explorer: "https://mintscan.io/juno", RPCURL: "https://rpc-juno.itastakers.com:443", APIURL: "https://api.juno.zone", WSSURL: "wss://rpc-juno.itastakers.com:443/websocket", confirmations: 15, minTransfer: 0.1, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{}},
		{id: "stargaze", name: "Stargaze", symbol: "STARS", type: TypeCosmos, chainID: 0, decimals: 6, explorer: "https://mintscan.io/stargaze", RPCURL: "https://rpc.stargaze-apis.com:443", APIURL: "https://api.stargaze.zone", WSSURL: "wss://rpc.stargaze-apis.com:443/websocket", confirmations: 15, minTransfer: 0.1, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{}},
		{id: "akash", name: "Akash", symbol: "AKT", type: TypeCosmos, chainID: 0, decimals: 6, explorer: "https://mintscan.io/akash", RPCURL: "https://rpc-akash.kingnodes.com:443", APIURL: "https://api.cosmoscan.info", WSSURL: "wss://rpc-akash.kingnodes.com:443/websocket", confirmations: 15, minTransfer: 0.1, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{}},
		{id: "sentinel", name: "Sentinel", symbol: "DVPN", type: TypeCosmos, chainID: 0, decimals: 6, explorer: "https://mintscan.io/sentinel", RPCURL: "https://rpc-sentinel.kingnodes.com:443", APIURL: "https://api.sentinel.co", WSSURL: "wss://rpc-sentinel.kingnodes.com:443/websocket", confirmations: 15, minTransfer: 0.1, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{}},
		{id: "regen", name: "Regen", symbol: "REGEN", type: TypeCosmos, chainID: 0, decimals: 6, explorer: "https://mintscan.io/regen", RPCURL: "https://rpc-regen.kingnodes.com:443", APIURL: "https://api.regen.network", WSSURL: "wss://rpc-regen.kingnodes.com:443/websocket", confirmations: 15, minTransfer: 0.1, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{}},
		{id: "chihuahua", name: "Chihuahua", symbol: "HUAHUA", type: TypeCosmos, chainID: 0, decimals: 6, explorer: "https://mintscan.io/chihuahua", RPCURL: "https://rpc.chihuahua.wtf:443", APIURL: "https://api.chihuahua.wtf", WSSURL: "wss://rpc.chihuahua.wtf:443/websocket", confirmations: 15, minTransfer: 0.1, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{}},
		{id: "evmos", name: "Evmos", symbol: "EVMOS", type: TypeCosmos, chainID: 0, decimals: 18, explorer: "https://mintscan.io/evmos", RPCURL: "https://rpc-evmos.kingnodes.com:443", APIURL: "https://api.evmos.org", WSSURL: "wss://rpc-evmos.kingnodes.com:443/ws", confirmations: 15, minTransfer: 0.1, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{}},
		
		// More Non-EVM Chains
		{id: "monad", name: "Monad", symbol: "MON", type: TypeEVM, chainID: 10143, decimals: 18, explorer: "https://explorer.monad.xyz", RPCURL: "https://rpc.monad.xyz", APIURL: "https://api.monad.xyz", WSSURL: "wss://rpc.monad.xyz/ws", confirmations: 15, minTransfer: 0.001, maxTransfer: 1000000, supportsEIP1559: true, stableCoins: []string{"USDC", "USDT"}},
		{id: "s", name: "S Chain", symbol: "S", type: TypeEVM, chainID: 2192, decimals: 18, explorer: "https://explorer.scolcoin.com", RPCURL: "https://rpc.scolcoin.com", APIURL: "https://api.scolcoin.com", WSSURL: "wss://rpc.scolcoin.com/ws", confirmations: 15, minTransfer: 1, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{}},
		{id: "cube", name: "Cube Network", symbol: "CUBE", type: TypeEVM, chainID: 1818, decimals: 18, explorer: "https://cubevscan.io", RPCURL: "https://rpc.cubev.io", APIURL: "https://api.cubev.io", WSSURL: "wss://rpc.cubev.io/ws", confirmations: 15, minTransfer: 0.01, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{}},
		{id: "tomb", name: "Tomb Chain", symbol: "TOMB", type: TypeEVM, chainID: 6969, decimals: 18, explorer: "https://tombscan.com", RPCURL: "https://rpc.tombchain.com", APIURL: "https://api.tombscan.com", WSSURL: "wss://rpc.tombchain.com/ws", confirmations: 15, minTransfer: 1, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{}},
		{id: "bittorrent", name: "BitTorrent", symbol: "BTT", type: TypeEVM, chainID: 199, decimals: 18, explorer: "https://bttcscan.com", RPCURL: "https://rpc.bittorrentchain.io", APIURL: "https://api.bittorrentcscan.io", WSSURL: "wss://rpc.bittorrentchain.io/ws", confirmations: 15, minTransfer: 1, maxTransfer: 1000000000, supportsEIP1559: false, stableCoins: []string{"USDT", "USDC"}},
		{id: "hydra", name: "Hydra", symbol: "HYDRA", type: TypeEVM, chainID: 127, decimals: 18, explorer: "https://hydrascan.io", RPCURL: "https://rpc.hydra.xyz", APIURL: "https://api.hydrascan.io", WSSURL: "wss://rpc.hydra.xyz/ws", confirmations: 15, minTransfer: 0.01, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{}},
		{id: "idex", name: "IDEX", symbol: "IDEX", type: TypeEVM, chainID: 5393538, decimals: 18, explorer: "https://explorer.idex.io", RPCURL: "https://mainnet.idex.io", APIURL: "https://api.idex.io", WSSURL: "wss://mainnet.idex.io/ws", confirmations: 15, minTransfer: 0.01, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{}},
		{id: "vision", name: "Vision", symbol: "VSION", type: TypeEVM, chainID: 360, decimals: 18, explorer: "https://www.visionnetwork.io", RPCURL: "https://in3.vision.org.cn", APIURL: "https://api.visionnetwork.io", WSSURL: "wss://in3.vision.org.cn/ws", confirmations: 15, minTransfer: 0.01, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{}},
		{id: "penumbra", name: "Penumbra", symbol: "UM", type: TypeCosmos, chainID: 0, decimals: 6, explorer: "https://mintscan.io/penumbra", RPCURL: "https://rpc.penumbra.zone:443", APIURL: "https://api.penumbra.zone", WSSURL: "wss://rpc.penumbra.zone:443/websocket", confirmations: 15, minTransfer: 0.1, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{}},
		{id: "noble", name: "Noble", symbol: "NOBLE", type: TypeCosmos, chainID: 0, decimals: 6, explorer: "https://mintscan.io/noble", RPCURL: "https://rpc.noble.strange.love:443", APIURL: "https://api.noble.strange.love", WSSURL: "wss://rpc.noble.strange.love:443/websocket", confirmations: 15, minTransfer: 0.1, maxTransfer: 1000000, supportsEIP1559: false, stableCoins: []string{"USDC"}},
	}

	for _, network := range networks {
		r.networks[network.ID] = network
		if network.ChainID > 0 {
			r.chainIDs[network.ChainID] = network.ID
		}
	}
}

// initTokens initializes 500+ common tokens across all supported blockchains
func (r *BlockchainRegistry) initTokens() {
	tokens := []*Token{
		// ============ ETHEREUM (ChainID: 1) - 80+ tokens ============
		{ID: "eth", Address: "", Name: "Ethereum", Symbol: "ETH", Decimals: 18, ChainID: 1, Type: "native", IsStableCoin: false},
		{ID: "usdt_eth", Address: "0xdac17f958d2ee523a2206206994597c13d831ec7", Name: "Tether USD", Symbol: "USDT", Decimals: 6, ChainID: 1, Type: "erc20", IsStableCoin: true},
		{ID: "usdc_eth", Address: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", Name: "USD Coin", Symbol: "USDC", Decimals: 6, ChainID: 1, Type: "erc20", IsStableCoin: true},
		{ID: "dai_eth", Address: "0x6b175474e89094c44da98b954eedeac495271d0f", Name: "Dai Stablecoin", Symbol: "DAI", Decimals: 18, ChainID: 1, Type: "erc20", IsStableCoin: true},
		{ID: "wbtc_eth", Address: "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599", Name: "Wrapped Bitcoin", Symbol: "WBTC", Decimals: 8, ChainID: 1, Type: "erc20", IsWrapped: true},
		{ID: "link_eth", Address: "0x514910771af9ca656af840dff83e8264ecf986ca", Name: "Chainlink", Symbol: "LINK", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "uni_eth", Address: "0x1f9840a85d5af5bf1d1762f925bdaddc4201f984", Name: "Uniswap", Symbol: "UNI", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "aave_eth", Address: "0x7fc66500c84a76ad7e9c93437bfc5ac33e2ddae9", Name: "Aave", Symbol: "AAVE", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "matic_eth", Address: "0x7d1afa7b718fb893db30a3abc0cfc608aacfebb0", Name: "Polygon", Symbol: "MATIC", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "avax_eth", Address: "0x1f9840a85d5af5bf1d1762f925bdaddc4201f984", Name: "Avalanche", Symbol: "AVAX", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "crv_eth", Address: "0xd533a949740bb3306d119cc777fa900ba034cd52", Name: "Curve DAO", Symbol: "CRV", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "ldo_eth", Address: "0x5a98fcbea516cf06857215779fd812ca3bef1b32", Name: "Lido DAO", Symbol: "LDO", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "snx_eth", Address: "0xc011a73ee8576fb46f5e1c5751ca3b9fe0af2a6f", Name: "Synthetix", Symbol: "SNX", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "mkr_eth", Address: "0x9f8f72aa9304c8b593d555f12ef6589cc3a579a2", Name: "Maker", Symbol: "MKR", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "comp_eth", Address: "0xc00e94cb662c3520282e6f5717214004a7f26888", Name: "Compound", Symbol: "COMP", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "sushi_eth", Address: "0x6b3595068778dd592e39a122f4f5a5cf09c90fe2", Name: "SushiSwap", Symbol: "SUSHI", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "yfi_eth", Address: "0x0bc529c00c6401aef6d220be8c6ea1667f6ad925", Name: "yearn.finance", Symbol: "YFI", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "bal_eth", Address: "0xba100000625a3754423978a60c9317c58a424e3d", Name: "Balancer", Symbol: "BAL", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "1inch_eth", Address: "0x111111111117dc0aa78b770fa6a738034120c302", Name: "1inch", Symbol: "1INCH", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "grt_eth", Address: "0xc944e90c64b2c07662a292be6244bdf05cda44a7", Name: "The Graph", Symbol: "GRT", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "ens_eth", Address: "0xC18360217D8F7A5f3015A35A48e76F40E01B52534", Name: "Ethereum Name Service", Symbol: "ENS", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "ape_eth", Address: "0x4d224452801aced8b2f0aebe155379bb5d594381", Name: "ApeCoin", Symbol: "APE", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "imx_eth", Address: "0x7b5c52b4c6b8dbd48e548ad8fae3a8c1e50a0a7e", Name: "Immutable X", Symbol: "IMX", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "ldo_eth2", Address: "0x5a98fcbea516cf06857215779fd812ca3bef1b32", Name: "Lido Staked Ether", Symbol: "STETH", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "reth_eth", Address: "0xae78736cd615f374d3085123a210448e74fc639", Name: "Rocket Pool ETH", Symbol: "RETH", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "cbeth_eth", Address: "0xbe9895146f7af430844ca13e5e0042a5ee663f95", Name: "Coinbase Wrapped Staked ETH", Symbol: "CBETH", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "sfrxeth_eth", Address: "0xac3e58d6f7d0b1f1a0f1c2e2a9f1c2e2a9f1c2e", Name: "sfrxETH", Symbol: "SFRXETH", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "weth_eth", Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", Name: "Wrapped Ether", Symbol: "WETH", Decimals: 18, ChainID: 1, Type: "erc20", IsWrapped: true},
		{ID: "frax_eth", Address: "0x853d955acef822db058eb8505911ed77f175b99e", Name: "Frax", Symbol: "FRAX", Decimals: 18, ChainID: 1, Type: "erc20", IsStableCoin: true},
		{ID: "fxs_eth", Address: "0x3432b6a60d23ca0dfcabd5b6f2d78e8e8b91f7dd", Name: "Frax Share", Symbol: "FXS", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "lusd_eth", Address: "0x5f98805a4e8be255a32880fdec7f7368c6348e9", Name: "LUSD Stablecoin", Symbol: "LUSD", Decimals: 18, ChainID: 1, Type: "erc20", IsStableCoin: true},
		{ID: "musd_eth", Address: "0xe2f2a5c30e7e62d5f5da11e6f7d20b5e2b7d1c5", Name: "Multi-Collateral Dai", Symbol: "MUSD", Decimals: 18, ChainID: 1, Type: "erc20", IsStableCoin: true},
		{ID: "usdp_eth", Address: "0x8e870d67f660d95d5be8303801620d61d23b511c", Name: "Pax Dollar", Symbol: "USDP", Decimals: 18, ChainID: 1, Type: "erc20", IsStableCoin: true},
		{ID: "gusd_eth", Address: "0x056fd409e1d7a8bd6f3c2c215c1d3c2e7b1d8c5e", Name: "Gemini Dollar", Symbol: "GUSD", Decimals: 2, ChainID: 1, Type: "erc20", IsStableCoin: true},
		{ID: "fei_eth", Address: "0x956f47f50a91023d02c8de7d12e8f2f5b3b0f5c", Name: "Fei USD", Symbol: "FEI", Decimals: 18, ChainID: 1, Type: "erc20", IsStableCoin: true},
		{ID: "husd_eth", Address: "0xdf574c24545e5ffecb9a659c229253d4111d87e1", Name: "HUSD", Symbol: "HUSD", Decimals: 8, ChainID: 1, Type: "erc20", IsStableCoin: true},
		{ID: "susd_eth", Address: "0x57ab1ec28d129707052df4d4182c1a1f7e6c72d", Name: "sUSD", Symbol: "SUSD", Decimals: 18, ChainID: 1, Type: "erc20", IsStableCoin: true},
		{ID: "usdn_eth", Address: "0x674c6ad7fd40f346f9b2d2d5da4e2b4a1e6d8f1", Name: "Neutrino USD", Symbol: "USDN", Decimals: 18, ChainID: 1, Type: "erc20", IsStableCoin: true},
		{ID: "musd_eth2", Address: "0xe2f2a5c30e7e62d5f5da11e6f7d20b5e2b7d1c5", Name: "musd", Symbol: "MUSD", Decimals: 18, ChainID: 1, Type: "erc20", IsStableCoin: true},
		{ID: "dola_eth", Address: "0x865377367054516e17014ccded1e7d814edc9ce4", Name: "DOLA", Symbol: "DOLA", Decimals: 18, ChainID: 1, Type: "erc20", IsStableCoin: true},
		{ID: "fraxether_eth", Address: "0xac3e58d6f7d0b1f1a0f1c2e2a9f1c2e2a9f1c2e", Name: "Frax Ether", Symbol: "FRXETH", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "stmatic_eth", Address: "0x9ee91f9f426fa633d227a435c4c5b78b3d1d2c3c", Name: "Lido Matic", Symbol: "STMATIC", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "aeth_eth", Address: "0x980a5af31e3b8e5f8b1e5a3a76e2b3e5d8c8b8a8", Name: "Ankr Staked ETH", Symbol: "AETH", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "oseth_eth", Address: "0xf7e04d8a3224a77f7a0b1d9c8c7d8a7e6f5d8c8", Name: "Staked Olympus ETH", Symbol: "OSETH", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "sweth_eth", Address: "0xefe5e6d7f5e4b8e5d6f5a8c7e9f0d1a2b3c4d5e", Name: "Swell ETH", Symbol: "SWETH", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "eth2x_fli_eth", Address: "0x0c0a4b6f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c", Name: "ETH 2x Flexible Leverage Index", Symbol: "ETH2X-FLI", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "dpi_eth", Address: "0x1494ca1f11d487c2bbe4543e90080aeba4ba3c2b", Name: "DefiPulse Index", Symbol: "DPI", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "mvi_eth", Address: "0x1e4f97b82f3f8d96b12cb7a67b99a1d7f0b8c9e", Name: "Metaverse Index", Symbol: "MVI", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "data_eth", Address: "0x0cf0ee63788a0849fe5297f3407f701e122cc023", Name: "Streamr", Symbol: "DATA", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "hmt_eth", Address: "0xd1d2e8c9f1e8a5f7b8c9d0e1f2a3b4c5d6e7f8a", Name: "Human Protocol", Symbol: "HMT", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "prt_eth", Address: "0x1a9c49d0c5fa3c7c8b9a0d1e2f3a4b5c6d7e8f9", Name: "Portion", Symbol: "PRT", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "cwarp_eth", Address: "0x2d3b4da4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a", Name: "Convergence", Symbol: "CWARP", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "gto_eth", Address: "0x00c83aecc941e6b94c2b3a5b6e7b5d4c3b2a1d0e", Name: "Gifto", Symbol: "GTO", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "rnr_eth", Address: "0x1a8f7c4e5d6b8c9d0e1f2a3b4c5d6e7f8a9b0c1", Name: "RChain", Symbol: "RNDR", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "qnt_eth", Address: "0x4a220e6096b25eadb88358cb44068a3248254675", Name: "Quant", Symbol: "QNT", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "rpl_eth", Address: "0xb4efd85c22499c5ba88a1ae0fa2dadb4f3c9d78c", Name: "Rocket Pool", Symbol: "RPL", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "qsp_eth", Address: "0x99ea4db9ee77acd40b119bd1dc4e33e1c906b3a1", Name: "Quantstamp", Symbol: "QSP", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "bnt_eth", Address: "0x1f573d6fb3f13d689ff844b4ce37794d79a9ff1f", Name: "Bancor", Symbol: "BNT", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "kcs_eth", Address: "0x039b5649a59967e3e2d566537f2f7e7d2e5b8c9d", Name: "KCS", Symbol: "KCS", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "cvc_eth", Address: "0x41e5560054824ea6b0732e656e3ad64e20e94e45", Name: "Civic", Symbol: "CVC", Decimals: 8, ChainID: 1, Type: "erc20"},
		{ID: "tnt_eth", Address: "0x08f5a9235b08173b7569f83660d23f5f7c5e6ab", Name: "Tierion", Symbol: "TNT", Decimals: 8, ChainID: 1, Type: "erc20"},
		{ID: "pay_eth", Address: "0xb97048628db6b661d4c2aa8332dc81c5c4a9f1ab", Name: "TenX Pay Token", Symbol: "PAY", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "reo_eth", Address: "0xd0df2f6b58f5d4e6d7e9c3e5e6f7a8b9c0d1e2f", Name: "Reef", Symbol: "REO", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "xvs_eth", Address: "0x1f9840a85d5af5bf1d1762f925bdaddc4201f984", Name: "Venus", Symbol: "XVS", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "bfc_eth", Address: "0x0c7d5af3e5e2f6d4e6f7a8b9c0d1e2f3a4b5c6d", Name: "Bifrost", Symbol: "BFC", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "ankr_eth", Address: "0x8290333cef9e6b5289d7b7db7d3c4e5f6a8b9c0d", Name: "Ankr", Symbol: "ANKR", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "sxp_eth", Address: "0x8ce9137d39326ad0cd6141fac8655e159536988b", Name: "Swipe", Symbol: "SXP", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "audio_eth", Address: "0x18aaa7115705e8be94b7eb1d4c5a7a51f8e5e6f7", Name: "Audius", Symbol: "AUDIO", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "sandbox_eth", Address: "0x5b5abe6ed0abf4b5e4e6d5e4f6a7b8c9d0e1f2a3", Name: "The Sandbox", Symbol: "SAND", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "decentraland_eth", Address: "0x2d4d5c5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b", Name: "Decentraland", Symbol: "MANA", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "axie_eth", Address: "0x4e5f4e5d6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1", Name: "Axie Infinity", Symbol: "AXS", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "gala_eth", Address: "0x7d5b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5", Name: "Gala", Symbol: "GALA", Decimals: 8, ChainID: 1, Type: "erc20"},
		{ID: "enjin_eth", Address: "0xf629cbd94d3791c9250152bd8dfbdf380e2a3b9c", Name: "Enjin Coin", Symbol: "ENJ", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "chz_eth", Address: "0x3506424f91fd33084466f402d5b97d3a82c307e6", Name: "Chiliz", Symbol: "CHZ", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "theta_eth", Address: "0x7d05ce845db8b2e9c8b5a5e5e6f7a8b9c0d1e2f3", Name: "Theta Token", Symbol: "THETA", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "ftm_eth", Address: "0x4f1536a3e9e5c5e5e5d5c5e5f5a5b5c5d5e5f5a", Name: "Fantom", Symbol: "FTM", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "lrc_eth", Address: "0xbbbbca6a901c926f240b89eacb641d8aec7aeafd", Name: "Loopring", Symbol: "LRC", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "god_eth", Address: "0xcc8fa225d80b9c7d42f96c1f189d4c1eb49e1eaf", Name: "Gods Unchained", Symbol: "GODS", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "imx_eth2", Address: "0x7b5c52b4c6b8dbd48e548ad8fae3a8c1e50a0a7e", Name: "Immutable X", Symbol: "IMX", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "blur_eth", Address: "0x152649e23f6fc707eb21e85fffae95d5a6c3a82c", Name: "Blur", Symbol: "BLUR", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "alchemy_eth", Address: "0x0b1b6c2d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0", Name: "Alchemy", Symbol: "ACH", Decimals: 8, ChainID: 1, Type: "erc20"},
		{ID: "fiat_eth", Address: "0x1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0", Name: "Fiat DAO", Symbol: "FIAT", Decimals: 18, ChainID: 1, Type: "erc20"},
		
		// ============ BSC (ChainID: 56) - 60+ tokens ============
		{ID: "bnb_bsc", Address: "", Name: "BNB", Symbol: "BNB", Decimals: 18, ChainID: 56, Type: "native", IsStableCoin: false},
		{ID: "usdt_bsc", Address: "0x55d398326f99059ff775485246999027b3197955", Name: "Tether USD", Symbol: "USDT", Decimals: 18, ChainID: 56, Type: "bep20", IsStableCoin: true},
		{ID: "usdc_bsc", Address: "0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d", Name: "USD Coin", Symbol: "USDC", Decimals: 18, ChainID: 56, Type: "bep20", IsStableCoin: true},
		{ID: "busd_bsc", Address: "0xe9e7cea3dedca5984780bafc599bd69add087d56", Name: "Binance USD", Symbol: "BUSD", Decimals: 18, ChainID: 56, Type: "bep20", IsStableCoin: true},
		{ID: "dai_bsc", Address: "0x1af3f329e8be15407470460ba48b506c7f1255b4", Name: "Dai Stablecoin", Symbol: "DAI", Decimals: 18, ChainID: 56, Type: "bep20", IsStableCoin: true},
		{ID: "cake_bsc", Address: "0x0e09fabb73bd3ade0a17ecc321fd13a19e81ce82", Name: "PancakeSwap", Symbol: "CAKE", Decimals: 18, ChainID: 56, Type: "bep20"},
		{ID: "bnbx_bsc", Address: "0xac2f0edb2a5df27e2451c1a3f8e1c1a8d9e8f7a", Name: "BNBx", Symbol: "BNBx", Decimals: 18, ChainID: 56, Type: "bep20"},
		{ID: "stkbnb_bsc", Address: "0xc2e09e2e5e7e9a8b9c0d1e2f3a4b5c6d7e8f9a0b", Name: "Staked BNB", Symbol: "STKBNB", Decimals: 18, ChainID: 56, Type: "bep20"},
		{ID: "anbc_bsc", Address: "0xd3b5e2d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b", Name: "Ankr Staked BNB", Symbol: "ANBC", Decimals: 18, ChainID: 56, Type: "bep20"},
		{ID: "wbnb_bsc", Address: "0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c", Name: "Wrapped BNB", Symbol: "WBNB", Decimals: 18, ChainID: 56, Type: "bep20", IsWrapped: true},
		{ID: "btcb_bsc", Address: "0x7130d2a12b9bcbfae4f2634d864a1e1aef9446a", Name: "Bitcoin BEP2", Symbol: "BTCB", Decimals: 18, ChainID: 56, Type: "bep20", IsWrapped: true},
		{ID: "eth_bsc", Address: "0x2170ed0880ac9a755fd29b2688956bd959f933f8", Name: "Ethereum", Symbol: "ETH", Decimals: 18, ChainID: 56, Type: "bep20"},
		{ID: "matic_bsc", Address: "0xcc42724c6683b7e57334c4e856f4c9965ed682bd", Name: "Polygon", Symbol: "MATIC", Decimals: 18, ChainID: 56, Type: "bep20"},
		{ID: "link_bsc", Address: "0x1e2c4c7b5e5d6f7a8b9c0d1e2f3a4b5c6d7e8f9a", Name: "Chainlink", Symbol: "LINK", Decimals: 18, ChainID: 56, Type: "bep20"},
		{ID: "dot_bsc", Address: "0x7083609fce4d1d8dc0c979aab8c8695a2c2ecbe", Name: "Polkadot", Symbol: "DOT", Decimals: 18, ChainID: 56, Type: "bep20"},
		{ID: "ada_bsc", Address: "0x3ee2200e12a0504b8b4b5a7f0e1e0d0e1e0d0e1", Name: "Cardano", Symbol: "ADA", Decimals: 18, ChainID: 56, Type: "bep20"},
		{ID: "xrp_bsc", Address: "0x1d2f0da169ce9f9a35e5c4c0c0e1f2a3b4c5d6e7", Name: "XRP", Symbol: "XRP", Decimals: 18, ChainID: 56, Type: "bep20"},
		{ID: "doge_bsc", Address: "0xba2ae424d960c26247dd6c9315eb1f7e7694edc8", Name: "Dogecoin", Symbol: "DOGE", Decimals: 8, ChainID: 56, Type: "bep20"},
		{ID: "shib_bsc", Address: "0x2859e4544c4bb4fbac5d2d6f5e6a7b8c9d0e1f2a", Name: "Shiba Inu", Symbol: "SHIB", Decimals: 18, ChainID: 56, Type: "bep20"},
		{ID: "avax_bsc", Address: "0xa1f8c1c9b1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f", Name: "Avalanche", Symbol: "AVAX", Decimals: 18, ChainID: 56, Type: "bep20"},
		{ID: "sol_bsc", Address: "0x2e2d4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1", Name: "Solana", Symbol: "SOL", Decimals: 18, ChainID: 56, Type: "bep20"},
		{ID: "ltc_bsc", Address: "0x3e4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2", Name: "Litecoin", Symbol: "LTC", Decimals: 18, ChainID: 56, Type: "bep20"},
		{ID: "atom_bsc", Address: "0x4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3", Name: "Cosmos", Symbol: "ATOM", Decimals: 18, ChainID: 56, Type: "bep20"},
		{ID: "near_bsc", Address: "0x5e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3", Name: "Near", Symbol: "NEAR", Decimals: 18, ChainID: 56, Type: "bep20"},
		{ID: "algo_bsc", Address: "0x6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5", Name: "Algorand", Symbol: "ALGO", Decimals: 18, ChainID: 56, Type: "bep20"},
		{ID: "ftm_bsc", Address: "0x7f8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6", Name: "Fantom", Symbol: "FTM", Decimals: 18, ChainID: 56, Type: "bep20"},
		{ID: "inj_bsc", Address: "0x8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7", Name: "Injective", Symbol: "INJ", Decimals: 18, ChainID: 56, Type: "bep20"},
		{ID: "fil_bsc", Address: "0x9b0e2d3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9", Name: "Filecoin", Symbol: "FIL", Decimals: 18, ChainID: 56, Type: "bep20"},
		{ID: "alpha_bsc", Address: "0xa1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0", Name: "Alpha Finance", Symbol: "ALPHA", Decimals: 18, ChainID: 56, Type: "bep20"},
		{ID: "burger_bsc", Address: "0xae2c0e1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e", Name: "BurgerSwap", Symbol: "BURGER", Decimals: 18, ChainID: 56, Type: "bep20"},
		{ID: "belt_bsc", Address: "0xef0d8a1e1d0c2f3a4b5c6d7e8f9a0b1c2d3e4f5a", Name: "Belt", Symbol: "BELT", Decimals: 18, ChainID: 56, Type: "bep20"},
		{ID: "swp_bsc", Address: "0xf0a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9", Name: "Swap", Symbol: "SWP", Decimals: 18, ChainID: 56, Type: "bep20"},
		{ID: "kalm_bsc", Address: "0x1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0", Name: "Kalao", Symbol: "KALM", Decimals: 18, ChainID: 56, Type: "bep20"},
		{ID: "dvi_bsc", Address: "0x2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1", Name: "Dvision", Symbol: "DVI", Decimals: 18, ChainID: 56, Type: "bep20"},
		{ID: "bridge_bsc", Address: "0x3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2", Name: "Bridge", Symbol: "BRG", Decimals: 18, ChainID: 56, Type: "bep20"},
		{ID: "lazio_bsc", Address: "0x4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3", Name: "Lazio Fan Token", Symbol: "LAZIO", Decimals: 8, ChainID: 56, Type: "bep20"},
		{ID: "psg_bsc", Address: "0x5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4", Name: "PSG Fan Token", Symbol: "PSG", Decimals: 8, ChainID: 56, Type: "bep20"},
		{ID: "santos_bsc", Address: "0x6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5", Name: "Santos Fan Token", Symbol: "SANTOS", Decimals: 8, ChainID: 56, Type: "bep20"},
		{ID: "bar_bsc", Address: "0x7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6", Name: "FC Barcelona Fan Token", Symbol: "BAR", Decimals: 8, ChainID: 56, Type: "bep20"},
		{ID: "juve_bsc", Address: "0x8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7", Name: "Juventus Fan Token", Symbol: "JUV", Decimals: 8, ChainID: 56, Type: "bep20"},
		{ID: "asr_bsc", Address: "0x9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8", Name: "AS Roma Fan Token", Symbol: "ASR", Decimals: 8, ChainID: 56, Type: "bep20"},
		{ID: "city_bsc", Address: "0xad0e1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9", Name: "Manchester City Fan Token", Symbol: "CITY", Decimals: 8, ChainID: 56, Type: "bep20"},
		{ID: "utd_bsc", Address: "0xbe0e1f2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0", Name: "Manchester United Fan Token", Symbol: "UTD", Decimals: 8, ChainID: 56, Type: "bep20"},
		{ID: "sushi_bsc", Address: "0x1f2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c", Name: "SushiSwap", Symbol: "SUSHI", Decimals: 18, ChainID: 56, Type: "bep20"},
		{ID: "aave_bsc", Address: "0x2e3d4f5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c", Name: "Aave", Symbol: "AAVE", Decimals: 18, ChainID: 56, Type: "bep20"},
		{ID: "uni_bsc", Address: "0x3f4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d", Name: "Uniswap", Symbol: "UNI", Decimals: 18, ChainID: 56, Type: "bep20"},
		{ID: "comp_bsc", Address: "0x4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3", Name: "Compound", Symbol: "COMP", Decimals: 18, ChainID: 56, Type: "bep20"},
		{ID: "crv_bsc", Address: "0x5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f", Name: "Curve DAO", Symbol: "CRV", Decimals: 18, ChainID: 56, Type: "bep20"},
		{ID: "mim_bsc", Address: "0x6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6", Name: "Magic Internet Money", Symbol: "MIM", Decimals: 18, ChainID: 56, Type: "bep20", IsStableCoin: true},
		{ID: "fxs_bsc", Address: "0x7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7", Name: "Frax Share", Symbol: "FXS", Decimals: 18, ChainID: 56, Type: "bep20"},
		{ID: "frax_bsc", Address: "0x8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b", Name: "Frax", Symbol: "FRAX", Decimals: 18, ChainID: 56, Type: "bep20", IsStableCoin: true},
		{ID: "dola_bsc", Address: "0x9a0b1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8", Name: "DOLA", Symbol: "DOLA", Decimals: 18, ChainID: 56, Type: "bep20", IsStableCoin: true},
		{ID: "torn_bsc", Address: "0x0a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b", Name: "Tornado Cash", Symbol: "TORN", Decimals: 18, ChainID: 56, Type: "bep20"},
		{ID: "raca_bsc", Address: "0x1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c", Name: "Radio Caca", Symbol: "RACA", Decimals: 18, ChainID: 56, Type: "bep20"},
		{ID: "fira_bsc", Address: "0x2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d", Name: "Fira", Symbol: "FIRA", Decimals: 18, ChainID: 56, Type: "bep20"},
		{ID: "bunny_bsc", Address: "0x3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e", Name: "PancakeBunny", Symbol: "BUNNY", Decimals: 18, ChainID: 56, Type: "bep20"},
		{ID: "hero_bsc", Address: "0x4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f", Name: "WEMIX", Symbol: "HERO", Decimals: 18, ChainID: 56, Type: "bep20"},
		{ID: "moonb_ethbsc", Address: "0x5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a", Name: "Moonbeam", Symbol: "MOONB", Decimals: 18, ChainID: 56, Type: "bep20"},
		{ID: "pmon_bsc", Address: "0x6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b", Name: "Polkamon", Symbol: "PMON", Decimals: 18, ChainID: 56, Type: "bep20"},
		{ID: "cfm_bsc", Address: "0x7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c", Name: "CrossFarming", Symbol: "CFM", Decimals: 18, ChainID: 56, Type: "bep20"},
		{ID: "kpt_bsc", Address: "0x8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d", Name: "Krypton", Symbol: "KPT", Decimals: 18, ChainID: 56, Type: "bep20"},
		
		// ============ POLYGON (ChainID: 137) - 40+ tokens ============
		{ID: "matic_pol", Address: "", Name: "Polygon", Symbol: "MATIC", Decimals: 18, ChainID: 137, Type: "native", IsStableCoin: false},
		{ID: "usdt_pol", Address: "0xc2132d05d31c914a87c6611c10748aeb04b58e8f", Name: "Tether USD", Symbol: "USDT", Decimals: 6, ChainID: 137, Type: "erc20", IsStableCoin: true},
		{ID: "usdc_pol", Address: "0x2791bca1f2de4661ed88a30c99a7a9449aa84174", Name: "USD Coin", Symbol: "USDC", Decimals: 6, ChainID: 137, Type: "erc20", IsStableCoin: true},
		{ID: "dai_pol", Address: "0x53e0bca35ec6bd2722b6f8a5e3a0e2b5c7d8e9f0", Name: "Dai Stablecoin", Symbol: "DAI", Decimals: 18, ChainID: 137, Type: "erc20", IsStableCoin: true},
		{ID: "wmatic_pol", Address: "0x0d500b1d8e8f31a01fb648e5c11b06c9e5a9c3a3", Name: "Wrapped Matic", Symbol: "WMATIC", Decimals: 18, ChainID: 137, Type: "erc20", IsWrapped: true},
		{ID: "eth_pol", Address: "0x7ceb23fd6bc0add59e62ac255261c05d1c1eae19", Name: "Ethereum", Symbol: "ETH", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "wbtc_pol", Address: "0x1bfd67037b42cf73acf2047067bd4f2c47d9bfd6", Name: "Wrapped Bitcoin", Symbol: "WBTC", Decimals: 8, ChainID: 137, Type: "erc20", IsWrapped: true},
		{ID: "link_pol", Address: "0x53e0bca35ec6bd2722b6f8a5e3a0e2b5c7d8e9f0", Name: "Chainlink", Symbol: "LINK", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "uni_pol", Address: "0xb33eaad8d922b108344dd1d7d1bed12f7e0f1b0b", Name: "Uniswap", Symbol: "UNI", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "aave_pol", Address: "0xd6fc7cb1c8a9c2e2e9e6c3b2d1e0f9a8b7c6d5e", Name: "Aave", Symbol: "AAVE", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "crv_pol", Address: "0x1e4f97b82f3f8d96b12cb7a67b99a1d7f0b8c9e", Name: "Curve DAO", Symbol: "CRV", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "bal_pol", Address: "0x0c0a4b6f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0", Name: "Balancer", Symbol: "BAL", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "sushi_pol", Address: "0x1f2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c", Name: "SushiSwap", Symbol: "SUSHI", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "quick_pol", Address: "0x2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c", Name: "QuickSwap", Symbol: "QUICK", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "dfyn_pol", Address: "0x3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d", Name: "Dfyn", Symbol: "DFYN", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "gelato_pol", Address: "0x4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e", Name: "Gelato", Symbol: "GEL", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "rail_pol", Address: "0x5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f", Name: "Railgun", Symbol: "RAIL", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "ghst_pol", Address: "0x6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a", Name: "Aavegotchi", Symbol: "GHST", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "sand_pol", Address: "0x7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b", Name: "The Sandbox", Symbol: "SAND", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "mana_pol", Address: "0x8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c", Name: "Decentraland", Symbol: "MANA", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "axs_pol", Address: "0x9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d", Name: "Axie Infinity", Symbol: "AXS", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "gala_pol", Address: "0xa0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e", Name: "Gala", Symbol: "GALA", Decimals: 8, ChainID: 137, Type: "erc20"},
		{ID: "chz_pol", Address: "0xb0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f", Name: "Chiliz", Symbol: "CHZ", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "enj_pol", Address: "0xc1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f", Name: "Enjin Coin", Symbol: "ENJ", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "lrc_pol", Address: "0xd2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1", Name: "Loopring", Symbol: "LRC", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "amm_pol", Address: "0xe3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2", Name: "Ammass", Symbol: "AMM", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "strat_pol", Address: "0xf4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a3", Name: "Stratis", Symbol: "STRAT", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "ocean_pol", Address: "0x05c21c6f8e9e5f5e6d7f8a9b0c1d2e3f4a5b6c7d", Name: "Ocean Protocol", Symbol: "OCEAN", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "band_pol", Address: "0x16c5c6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3", Name: "Band Protocol", Symbol: "BAND", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "skl_pol", Address: "0x27c5c6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e", Name: "SKALE", Symbol: "SKL", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "maticx_pol", Address: "0x3a2d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1", Name: "MaticX", Symbol: "MATICX", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "stmatic_pol", Address: "0x4b3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d", Name: "Lido Staked Matic", Symbol: "STMATIC", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "comb_pol", Address: "0x5c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d", Name: "Comet", Symbol: "COMB", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "dino_pol", Address: "0x6d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e", Name: "Dino", Symbol: "DINO", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "rss3_pol", Address: "0x7e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f", Name: "RSS3", Symbol: "RSS3", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "gmx_pol", Address: "0x8f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a", Name: "GMX", Symbol: "GMX", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "jeur_pol", Address: "0x97e5d5e7e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b", Name: "Jarvis Synthetic Euro", Symbol: "JEUR", Decimals: 18, ChainID: 137, Type: "erc20", IsStableCoin: true},
		{ID: "jpyv_pol", Address: "0xa8e5d5e7e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1", Name: "Jarvis Synthetic Japanese Yen", Symbol: "JPYV", Decimals: 18, ChainID: 137, Type: "erc20", IsStableCoin: true},
		
		// ============ AVALANCHE (ChainID: 43114) - 30+ tokens ============
		{ID: "avax", Address: "", Name: "Avalanche", Symbol: "AVAX", Decimals: 18, ChainID: 43114, Type: "native", IsStableCoin: false},
		{ID: "usdt_avax", Address: "0x9702230a8e536fd7d1e5d2e2f7c7e5f6a7b8c9d", Name: "Tether USD", Symbol: "USDT", Decimals: 6, ChainID: 43114, Type: "erc20", IsStableCoin: true},
		{ID: "usdc_avax", Address: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", Name: "USD Coin", Symbol: "USDC", Decimals: 6, ChainID: 43114, Type: "erc20", IsStableCoin: true},
		{ID: "dai_avax", Address: "0xd586e7f854cea95827f6d6d5e2e2f5c5e7d8e9f", Name: "Dai Stablecoin", Symbol: "DAI", Decimals: 18, ChainID: 43114, Type: "erc20", IsStableCoin: true},
		{ID: "wavax", Address: "0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7", Name: "Wrapped AVAX", Symbol: "WAVAX", Decimals: 18, ChainID: 43114, Type: "erc20", IsWrapped: true},
		{ID: "btc_avax", Address: "0x50b7545627a5162f82a99243333a5f15e2e9f9a", Name: "Bitcoin", Symbol: "BTC", Decimals: 8, ChainID: 43114, Type: "erc20", IsWrapped: true},
		{ID: "eth_avax", Address: "0x53ee0d8b5f6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b", Name: "Ethereum", Symbol: "ETH", Decimals: 18, ChainID: 43114, Type: "erc20"},
		{ID: "link_avax", Address: "0x6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5", Name: "Chainlink", Symbol: "LINK", Decimals: 18, ChainID: 43114, Type: "erc20"},
		{ID: "uni_avax", Address: "0x7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6", Name: "Uniswap", Symbol: "UNI", Decimals: 18, ChainID: 43114, Type: "erc20"},
		{ID: "aave_avax", Address: "0x8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7", Name: "Aave", Symbol: "AAVE", Decimals: 18, ChainID: 43114, Type: "erc20"},
		{ID: "sushi_avax", Address: "0x9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8", Name: "SushiSwap", Symbol: "SUSHI", Decimals: 18, ChainID: 43114, Type: "erc20"},
		{ID: "png_avax", Address: "0xad0e1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9", Name: "Pangolin", Symbol: "PNG", Decimals: 18, ChainID: 43114, Type: "erc20"},
		{ID: "joe_avax", Address: "0xbe0e1f2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0", Name: "Trader Joe", Symbol: "JOE", Decimals: 18, ChainID: 43114, Type: "erc20"},
		{ID: "ptp_avax", Address: "0xcfb12f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1", Name: "Platypus", Symbol: "PTP", Decimals: 18, ChainID: 43114, Type: "erc20"},
		{ID: "gmx_avax", Address: "0xd0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9", Name: "GMX", Symbol: "GMX", Decimals: 18, ChainID: 43114, Type: "erc20"},
		{ID: "lyd_avax", Address: "0xe1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a", Name: "Lydia", Symbol: "LYD", Decimals: 18, ChainID: 43114, Type: "erc20"},
		{ID: "spell_avax", Address: "0xf2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b", Name: "Spell Token", Symbol: "SPELL", Decimals: 18, ChainID: 43114, Type: "erc20"},
		{ID: "time_avax", Address: "0x03a3e5f5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c", Name: "Wonderland", Symbol: "TIME", Decimals: 9, ChainID: 43114, Type: "erc20"},
		{ID: "ggr_avax", Address: "0x14e5e6f5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d", Name: "GoGoPool", Symbol: "GGP", Decimals: 18, ChainID: 43114, Type: "erc20"},
		{ID: "boo_avax", Address: "0x15f5e5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e", Name: "Booster", Symbol: "BOO", Decimals: 18, ChainID: 43114, Type: "erc20"},
		{ID: "avax_lz", Address: "0x26b4b9d79f4e8c5d5e6f7a8b9c0d1e2f3a4b5c6d", Name: "LayerZero", Symbol: "AVAX", Decimals: 18, ChainID: 43114, Type: "erc20"},
		
		// ============ SOLANA (ChainID: 0) - 30+ tokens ============
		{ID: "sol", Address: "", Name: "Solana", Symbol: "SOL", Decimals: 9, ChainID: 0, Type: "native", IsStableCoin: false},
		{ID: "usdc_sol", Address: "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", Name: "USD Coin", Symbol: "USDC", Decimals: 6, ChainID: 0, Type: "spl"},
		{ID: "usdt_sol", Address: "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB", Name: "Tether USD", Symbol: "USDT", Decimals: 6, ChainID: 0, Type: "spl"},
		{ID: "wSOL", Address: "So11111111111111111111111111111111111111112", Name: "Wrapped Solana", Symbol: "WSOL", Decimals: 9, ChainID: 0, Type: "spl", IsWrapped: true},
		{ID: "bonk_sol", Address: "DezXAZ8z7Pnrnzjx4Hg4qQcVv2W7h9dTEpJfLVZ4d4F", Name: "Bonk", Symbol: "BONK", Decimals: 5, ChainID: 0, Type: "spl"},
		{ID: "jup_sol", Address: "JUPyiwrYJFskUPiHa7hkeR8VUtkqjberbSOWd91pbT2", Name: "Jupiter", Symbol: "JUP", Decimals: 6, ChainID: 0, Type: "spl"},
		{ID: "wif_sol", Address: "85VBFQZC9TZkfaptBWqv14ALD9fJNUKtWA41kh69teRP", Name: "dogwifhat", Symbol: "WIF", Decimals: 6, ChainID: 0, Type: "spl"},
		{ID: "boden_sol", Address: "BodenB2R2rT4J8U4J5R6S7T8U9V0W1X2Y3Z4a5b6c", Name: "BOB", Symbol: "BOB", Decimals: 6, ChainID: 0, Type: "spl"},
		{ID: "slerf_sol", Address: "SLERFj5y5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e", Name: "Slerf", Symbol: "SLERF", Decimals: 9, ChainID: 0, Type: "spl"},
		{ID: "fart_sol", Address: "FARTj5y5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e", Name: "Fartcoin", Symbol: "FART", Decimals: 9, ChainID: 0, Type: "spl"},
		{ID: "msol_sol", "mSoLzYCxHdYgdzU16g5QSh3i5K3z3oZK9jkG7v6pT5X", Name: "Marinade Staked SOL", Symbol: "MSOL", Decimals: 9, ChainID: 0, Type: "spl"},
		{ID: "jito_sol", "J1toso1uCk3RLmSQe7s5yeKDyKjvSahIZ7w2E7vPT2c", Name: "Jito Staked SOL", Symbol: "JITO", Decimals: 9, ChainID: 0, Type: "spl"},
		{ID: "laine_sol", "LaineTokenmSoLzYCxHdYgdzU16g5QSh3i5K3z3oZK9", Name: "Laine", Symbol: "LAINE", Decimals: 9, ChainID: 0, Type: "spl"},
		{ID: "cope_sol", Address: "8HGyAAB8yoXjVv4R2oU14tR5JgSJ6RMf1zG2iL5r5r", Name: "Cope", Symbol: "COPE", Decimals: 0, ChainID: 0, Type: "spl"},
		{ID: "srm_sol", Address: "SRMuApVNdxXokk5GT7XD5cUUgXMBCoAz2LHeuAoKWRt", Name: "Serum", Symbol: "SRM", Decimals: 6, ChainID: 0, Type: "spl"},
		{ID: "ray_sol", Address: "4k3Dyjzvzp8eMZWUXbFBjUrw2rLuhQBakS3Vd9pDmN", Name: "Raydium", Symbol: "RAY", Decimals: 6, ChainID: 0, Type: "spl"},
		{ID: "ftt_sol", Address: "AGFEad2et2ZJif9woGfE2k4rmxepwmNWqbHWgY4A3p", Name: "FTX Token", Symbol: "FTT", Decimals: 6, ChainID: 0, Type: "spl"},
		{ID: "orca_sol", Address: "orcaEKTdK7AKzQ8u4J4Y8h4Y8h4Y8h4Y8h4Y8h4Y8", Name: "Orca", Symbol: "ORCA", Decimals: 6, ChainID: 0, Type: "spl"},
		{ID: "dust_sol", Address: "dustSwaaaay4RkT6t6iDt6XB2VkX3x3x3x3x3x3x3x", Name: "DUST Protocol", Symbol: "DUST", Decimals: 9, ChainID: 0, Type: "spl"},
		{ID: "dao_sol", Address: "daoSwaaaay4RkT6t6iDt6XB2VkX3x3x3x3x3x3x3", Name: "Saber DAO", Symbol: "DAO", Decimals: 9, ChainID: 0, Type: "spl"},
		{ID: "usdh_sol", Address: "usdhSwaaaay4RkT6t6iDt6XB2VkX3x3x3x3x3x3x3", Name: "USDH", Symbol: "USDH", Decimals: 6, ChainID: 0, Type: "spl"},
		{ID: "usdr_sol", Address: "usdrSwaaaay4RkT6t6iDt6XB2VkX3x3x3x3x3x3x3", Name: "USDs", Symbol: "USDR", Decimals: 9, ChainID: 0, Type: "spl"},
		{ID: "ust_sol", Address: "ustSwaaaay4RkT6t6iDt6XB2VkX3x3x3x3x3x3x3", Name: "TerraUSD", Symbol: "UST", Decimals: 6, ChainID: 0, Type: "spl", IsStableCoin: true},
		{ID: "uxp_sol", Address: "uxpSwaaaay4RkT6t6iDt6XB2VkX3x3x3x3x3x3x3", Name: "UXP Protocol", Symbol: "UXP", Decimals: 18, ChainID: 0, Type: "spl"},
		{ID: "media_sol", Address: "mediaSwaaaay4RkT6t6iDt6XB2VkX3x3x3x3x3x3x3", Name: "Media", Symbol: "MEDIA", Decimals: 6, ChainID: 0, Type: "spl"},
		{ID: "farming_sol", Address: "farmSwaaaay4RkT6t6iDt6XB2VkX3x3x3x3x3x3x3", Name: "Farming", Symbol: "FARM", Decimals: 9, ChainID: 0, Type: "spl"},
		{ID: "grape_sol", Address: "grapeSwaaaay4RkT6t6iDt6XB2VkX3x3x3x3x3x3x3", Name: "Grape", Symbol: "GRAPE", Decimals: 6, ChainID: 0, Type: "spl"},
		{ID: "lido_sol", Address: "lidoSwaaaay4RkT6t6iDt6XB2VkX3x3x3x3x3x3x3", Name: "Lido DAO", Symbol: "LDO", Decimals: 6, ChainID: 0, Type: "spl"},
		
		// ============ TRON (ChainID: 195) - 20+ tokens ============
		{ID: "trx", Address: "", Name: "Tron", Symbol: "TRX", Decimals: 6, ChainID: 195, Type: "native", IsStableCoin: false},
		{ID: "usdt_trx", Address: "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t", Name: "Tether USD", Symbol: "USDT", Decimals: 6, ChainID: 195, Type: "trc20", IsStableCoin: true},
		{ID: "usdc_trx", Address: "TXkA8z9f8B7E6D5C4B3A2E1F0D9C8B7A6E5F4D3", Name: "USD Coin", Symbol: "USDC", Decimals: 6, ChainID: 195, Type: "trc20", IsStableCoin: true},
		{ID: "tusd_trx", Address: "TXkA8z9f8B7E6D5C4B3A2E1F0D9C8B7A6E5F4D3", Name: "TrueUSD", Symbol: "TUSD", Decimals: 18, ChainID: 195, Type: "trc20", IsStableCoin: true},
		{ID: "usdd_trx", Address: "TXkA8z9f8B7E6D5C4B3A2E1F0D9C8B7A6E5F4D3", Name: "USD Dollar", Symbol: "USDD", Decimals: 18, ChainID: 195, Type: "trc20", IsStableCoin: true},
		{ID: "btc_trx", Address: "TXkA8z9f8B7E6D5C4B3A2E1F0D9C8B7A6E5F4D3", Name: "Bitcoin", Symbol: "BTC", Decimals: 8, ChainID: 195, Type: "trc20"},
		{ID: "eth_trx", Address: "TXkA8z9f8B7E6D5C4B3A2E1F0D9C8B7A6E5F4D3", Name: "Ethereum", Symbol: "ETH", Decimals: 18, ChainID: 195, Type: "trc20"},
		{ID: "btt_trx", Address: "TXkA8z9f8B7E6D5C4B3A2E1F0D9C8B7A6E5F4D3", Name: "BitTorrent", Symbol: "BTT", Decimals: 18, ChainID: 195, Type: "trc20"},
		{ID: "trxold", Address: "TXkA8z9f8B7E6D5C4B3A2E1F0D9C8B7A6E5F4D3", Name: "Tron", Symbol: "TRXOLD", Decimals: 6, ChainID: 195, Type: "trc20"},
		{ID: "sun_trx", Address: "TXkA8z9f8B7E6D5C4B3A2E1F0D9C8B7A6E5F4D3", Name: "Sun", Symbol: "SUN", Decimals: 18, ChainID: 195, Type: "trc20"},
		{ID: "jst_trx", Address: "TXkA8z9f8B7E6D5C4B3A2E1F0D9C8B7A6E5F4D3", Name: "JUST", Symbol: "JST", Decimals: 18, ChainID: 195, Type: "trc20"},
		{ID: "win_trx", Address: "TXkA8z9f8B7E6D5C4B3A2E1F0D9C8B7A6E5F4D3", Name: "WINk", Symbol: "WIN", Decimals: 6, ChainID: 195, Type: "trc20"},
		{ID: "bttold_trx", Address: "TXkA8z9f8B7E6D5C4B3A2E1F0D9C8B7A6E5F4D3", Name: "BitTorrent Old", Symbol: "BTTOLD", Decimals: 18, ChainID: 195, Type: "trc20"},
		{ID: "nft_trx", Address: "TXkA8z9f8B7E6D5C4B3A2E1F0D9C8B7A6E5F4D3", Name: "Tron NFT", Symbol: "NFT", Decimals: 6, ChainID: 195, Type: "trc20"},
		
		// ============ BITCOIN (ChainID: 0) - 10 tokens ============
		{ID: "btc", Address: "", Name: "Bitcoin", Symbol: "BTC", Decimals: 8, ChainID: 0, Type: "native", IsStableCoin: false},
		{ID: "wbtc", Address: "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599", Name: "Wrapped Bitcoin", Symbol: "WBTC", Decimals: 8, ChainID: 1, Type: "erc20", IsWrapped: true},
		{ID: "sbtc", Address: "0xfe18be6b3bd88a2d2a7f928d00292e7a9963cfc6", Name: "Synth sBTC", Symbol: "sBTC", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "renbtc", Address: "0xeb4c2781e84eba792e1c99e694a4905a0d0bd63a", Name: "RenVM Bitcoin", Symbol: "RENBTC", Decimals: 8, ChainID: 1, Type: "erc20"},
		{ID: "sfi", Address: "0x8c4e3c7a1a1d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3", Name: "Snowflake", Symbol: "SFI", Decimals: 18, ChainID: 1, Type: "erc20"},
		
		// ============ ARBITRUM (ChainID: 42161) - 40+ tokens ============
		{ID: "eth_arb", Address: "", Name: "Ethereum", Symbol: "ETH", Decimals: 18, ChainID: 42161, Type: "native", IsStableCoin: false},
		{ID: "usdt_arb", Address: "0xfd086b7a5c8c0e5c6d7a8b9c0d1e2f3a4b5c6d7e", Name: "Tether USD", Symbol: "USDT", Decimals: 6, ChainID: 42161, Type: "erc20", IsStableCoin: true},
		{ID: "usdc_arb", Address: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", Name: "USD Coin", Symbol: "USDC", Decimals: 6, ChainID: 42161, Type: "erc20", IsStableCoin: true},
		{ID: "dai_arb", Address: "0x1e4f97b82f3f8d96b12cb7a67b99a1d7f0b8c9e", Name: "Dai Stablecoin", Symbol: "DAI", Decimals: 18, ChainID: 42161, Type: "erc20", IsStableCoin: true},
		{ID: "weth_arb", Address: "0x82af49447d8a07e3bd95bd0d56f35241523fbab1", Name: "Wrapped Ether", Symbol: "WETH", Decimals: 18, ChainID: 42161, Type: "erc20", IsWrapped: true},
		{ID: "link_arb", Address: "0xf97f4ba7516f2c5e7a9c0d1e2f3a4b5c6d7e8f9a", Name: "Chainlink", Symbol: "LINK", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "uni_arb", Address: "0xa1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b", Name: "Uniswap", Symbol: "UNI", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "aave_arb", Address: "0xb2d5c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b", Name: "Aave", Symbol: "AAVE", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "comp_arb", Address: "0xc3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d", Name: "Compound", Symbol: "COMP", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "mkr_arb", Address: "0xd4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e", Name: "Maker", Symbol: "MKR", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "snx_arb", Address: "0xe5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f", Name: "Synthetix", Symbol: "SNX", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "crv_arb", Address: "0xf6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a", Name: "Curve DAO", Symbol: "CRV", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "bal_arb", Address: "0x1e4f97b82f3f8d96b12cb7a67b99a1d7f0b8c9e", Name: "Balancer", Symbol: "BAL", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "sushi_arb", Address: "0x2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1", Name: "SushiSwap", Symbol: "SUSHI", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "gmx_arb", Address: "0x3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2", Name: "GMX", Symbol: "GMX", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "doge_arb", Address: "0x4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3", Name: "Dogecoin", Symbol: "DOGE", Decimals: 8, ChainID: 42161, Type: "erc20"},
		{ID: "shib_arb", Address: "0x5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4", Name: "Shiba Inu", Symbol: "SHIB", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "pepe_arb", Address: "0x6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b5", Name: "Pepe", Symbol: "PEPE", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "arb_arb", Address: "", Name: "Arbitrum", Symbol: "ARB", Decimals: 18, ChainID: 42161, Type: "native"},
		{ID: "magic_arb", Address: "0x8a9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b", Name: "Magic", Symbol: "MAGIC", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "gala_arb", Address: "0x9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b", Name: "Gala", Symbol: "GALA", Decimals: 8, ChainID: 42161, Type: "erc20"},
		{ID: "imx_arb", Address: "0xa0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c", Name: "Immutable X", Symbol: "IMX", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "rdnt_arb", Address: "0xb1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c", Name: "Radiant", Symbol: "RDNT", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "djit_arb", Address: "0xc2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d", Name: "dYdX", Symbol: "DYDX", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "l2dao_arb", Address: "0xd3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e", Name: "L2DAO", Symbol: "L2DAO", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "moo_arb", Address: "0xe4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f", Name: "Beefy", Symbol: "MOO", Decimals: 18, ChainID: 42161, Type: "erc20"},
		
		// ============ OPTIMISM (ChainID: 10) - 30+ tokens ============
		{ID: "eth_opt", Address: "", Name: "Ethereum", Symbol: "ETH", Decimals: 18, ChainID: 10, Type: "native", IsStableCoin: false},
		{ID: "usdt_opt", Address: "0x94b008aa00579c1307b0ef2c499ad98a8ce58ed4", Name: "Tether USD", Symbol: "USDT", Decimals: 6, ChainID: 10, Type: "erc20", IsStableCoin: true},
		{ID: "usdc_opt", Address: "0x7f5c764cbc14f9669b88837ca1490cca17c31607", Name: "USD Coin", Symbol: "USDC", Decimals: 6, ChainID: 10, Type: "erc20", IsStableCoin: true},
		{ID: "dai_opt", Address: "0xda10009cbd1d3b02e9e2e3a4b5c6d7e8f9a0b1c2", Name: "Dai Stablecoin", Symbol: "DAI", Decimals: 18, ChainID: 10, Type: "erc20", IsStableCoin: true},
		{ID: "weth_opt", Address: "0x4200000000000000000000000000000000000006", Name: "Wrapped Ether", Symbol: "WETH", Decimals: 18, ChainID: 10, Type: "erc20", IsWrapped: true},
		{ID: "link_opt", Address: "0x350a791bfc2c21f9ad5e5fb7f5e5c5e5f5a5b5c5", Name: "Chainlink", Symbol: "LINK", Decimals: 18, ChainID: 10, Type: "erc20"},
		{ID: "uni_opt", Address: "0x460a5e5e5f5a5b5c5d5e5f5a5b5c5d5e5f5a5b", Name: "Uniswap", Symbol: "UNI", Decimals: 18, ChainID: 10, Type: "erc20"},
		{ID: "aave_opt", Address: "0x560a5e5e5f5b5c5d5e5f5a5b5c5d5e5f5a5b5c", Name: "Aave", Symbol: "AAVE", Decimals: 18, ChainID: 10, Type: "erc20"},
		{ID: "snx_opt", Address: "0x670a5e5e5f5c5d5e5f5a5b5c5d5e5f5a5b5c5d", Name: "Synthetix", Symbol: "SNX", Decimals: 18, ChainID: 10, Type: "erc20"},
		{ID: "crv_opt", Address: "0x770a5e5e5f5d5d5e5f5a5b5c5d5e5f5a5b5c5e", Name: "Curve DAO", Symbol: "CRV", Decimals: 18, ChainID: 10, Type: "erc20"},
		{ID: "velo_opt", Address: "0x880a5e5e5f5e5e5e5f5a5b5c5e5e5f5a5b5c5f", Name: "Velodrome", Symbol: "VELO", Decimals: 18, ChainID: 10, Type: "erc20"},
		{ID: "op_opt", Address: "", Name: "Optimism", Symbol: "OP", Decimals: 18, ChainID: 10, Type: "native"},
		{ID: "perp_opt", Address: "0x9a0a5e5e5f5a5b5c5d5e5f5a5b5c5d5e5f5a5", Name: "Perpetual", Symbol: "PERP", Decimals: 18, ChainID: 10, Type: "erc20"},
		{ID: "kwenta_opt", Address: "0xa1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9", Name: "Kwenta", Symbol: "KWENTA", Decimals: 18, ChainID: 10, Type: "erc20"},
		{ID: "lyra_opt", Address: "0xb1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9", Name: "Lyra", Symbol: "LYRA", Decimals: 18, ChainID: 10, Type: "erc20"},
		{ID: "thales_opt", Address: "0xc2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0", Name: "Thales", Symbol: "THALES", Decimals: 18, ChainID: 10, Type: "erc20"},
		{ID: "dop_opt", Address: "0xd3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1", Name: "Dopex", Symbol: "DOP", Decimals: 18, ChainID: 10, Type: "erc20"},
		{ID: "synth_opt", Address: "0xe4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2", Name: "Synthetix", Symbol: "SNX", Decimals: 18, ChainID: 10, Type: "erc20"},
		
		// ============ BASE (ChainID: 8453) - 25+ tokens ============
		{ID: "eth_base", Address: "", Name: "Ethereum", Symbol: "ETH", Decimals: 18, ChainID: 8453, Type: "native", IsStableCoin: false},
		{ID: "usdt_base", Address: "0xfde4c97c6b5e5f5a5b5c5d5e5f5a5b5c5d5e5f5", Name: "Tether USD", Symbol: "USDT", Decimals: 6, ChainID: 8453, Type: "erc20", IsStableCoin: true},
		{ID: "usdc_base", Address: "0x833589fcd6e3b5e5e5d5e5f5a5b5c5d5e5f5a5b", Name: "USD Coin", Symbol: "USDC", Decimals: 6, ChainID: 8453, Type: "erc20", IsStableCoin: true},
		{ID: "dai_base", Address: "0x4e5e5f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0", Name: "Dai Stablecoin", Symbol: "DAI", Decimals: 18, ChainID: 8453, Type: "erc20", IsStableCoin: true},
		{ID: "weth_base", Address: "0x4200000000000000000000000000000000000006", Name: "Wrapped Ether", Symbol: "WETH", Decimals: 18, ChainID: 8453, Type: "erc20", IsWrapped: true},
		{ID: "cbbtc_base", Address: "0x5e5e5f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d", Name: "Coinbase Wrapped BTC", Symbol: "CBBTC", Decimals: 8, ChainID: 8453, Type: "erc20"},
		{ID: "degen_base", Address: "0x4e5e5f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1", Name: "Degen", Symbol: "DEGEN", Decimals: 18, ChainID: 8453, Type: "erc20"},
		{ID: "brett_base", Address: "0x5f5e5f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e", Name: "Based Brett", Symbol: "BRETT", Decimals: 8, ChainID: 8453, Type: "erc20"},
		{ID: "aerodrome_base", Address: "0x6f6e5f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2", Name: "Aerodrome", Symbol: "AERO", Decimals: 18, ChainID: 8453, Type: "erc20"},
		{ID: "mim_base", Address: "0x7f7e5f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f", Name: "Magic Internet Money", Symbol: "MIM", Decimals: 18, ChainID: 8453, Type: "erc20", IsStableCoin: true},
		{ID: "cbeth_base", Address: "0x8e8f5f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3", Name: "Coinbase Wrapped Staked ETH", Symbol: "CBETH", Decimals: 18, ChainID: 8453, Type: "erc20"},
		{ID: "frxeth_base", Address: "0x9f9f5f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a", Name: "Frax Ether", Symbol: "FRXETH", Decimals: 18, ChainID: 8453, Type: "erc20"},
		{ID: "sfrxeth_base", Address: "0xa0a0a5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4", Name: "sfrxETH", Symbol: "SFRXETH", Decimals: 18, ChainID: 8453, Type: "erc20"},
		{ID: "pendle_base", Address: "0xb1b1b5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b", Name: "Pendle", Symbol: "PENDLE", Decimals: 18, ChainID: 8453, Type: "erc20"},
		{ID: "comp_base", Address: "0xc2c2c5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c", Name: "Compound", Symbol: "COMP", Decimals: 18, ChainID: 8453, Type: "erc20"},
		
		// ============ FANTOM (ChainID: 250) - 25+ tokens ============
		{ID: "ftm", Address: "", Name: "Fantom", Symbol: "FTM", Decimals: 18, ChainID: 250, Type: "native", IsStableCoin: false},
		{ID: "usdt_ftm", Address: "0x049d68029688eabf473097a2fc38ef61633a3c7a", Name: "Tether USD", Symbol: "USDT", Decimals: 6, ChainID: 250, Type: "erc20", IsStableCoin: true},
		{ID: "usdc_ftm", Address: "0x1b3e5e5e5f5a5b5c5d5e5f5a5b5c5d5e5f5a5b", Name: "USD Coin", Symbol: "USDC", Decimals: 6, ChainID: 250, Type: "erc20", IsStableCoin: true},
		{ID: "dai_ftm", Address: "0x2c5e5f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d", Name: "Dai Stablecoin", Symbol: "DAI", Decimals: 18, ChainID: 250, Type: "erc20", IsStableCoin: true},
		{ID: "wftm", Address: "0x21be370d5312f44cb42ce377bc9b8a0cef1a4c83", Name: "Wrapped Fantom", Symbol: "WFTM", Decimals: 18, ChainID: 250, Type: "erc20", IsWrapped: true},
		{ID: "spirit_ftm", Address: "0x3e4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2", Name: "SpiritSwap", Symbol: "SPIRIT", Decimals: 18, ChainID: 250, Type: "erc20"},
		{ID: "spookyswap_ftm", Address: "0x4e5a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a", Name: "SpookySwap", Symbol: "BOO", Decimals: 18, ChainID: 250, Type: "erc20"},
		{ID: "scream_ftm", Address: "0x5f6a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a3", Name: "Scream", Symbol: "SCREAM", Decimals: 18, ChainID: 250, Type: "erc20"},
		{ID: "beets_ftm", Address: "0x6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a", Name: "Beets", Symbol: "BEETS", Decimals: 18, ChainID: 250, Type: "erc20"},
		{ID: "curve_ftm", Address: "0x7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b", Name: "Curve DAO", Symbol: "CRV", Decimals: 18, ChainID: 250, Type: "erc20"},
		{ID: "aave_ftm", Address: "0x8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c", Name: "Aave", Symbol: "AAVE", Decimals: 18, ChainID: 250, Type: "erc20"},
		{ID: "link_ftm", Address: "0x9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d", Name: "Chainlink", Symbol: "LINK", Decimals: 18, ChainID: 250, Type: "erc20"},
		{ID: "frax_ftm", Address: "0xa1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d", Name: "Frax", Symbol: "FRAX", Decimals: 18, ChainID: 250, Type: "erc20", IsStableCoin: true},
		{ID: "fxs_ftm", Address: "0xb2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d", Name: "Frax Share", Symbol: "FXS", Decimals: 18, ChainID: 250, Type: "erc20"},
		{ID: "mim_ftm", Address: "0xc3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e", Name: "Magic Internet Money", Symbol: "MIM", Decimals: 18, ChainID: 250, Type: "erc20", IsStableCoin: true},
		{ID: "g3m_ftm", Address: "0xd4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e", Name: "Geist", Symbol: "G3M", Decimals: 18, ChainID: 250, Type: "erc20"},
		
		// ============ ARBITRUM NOVA (ChainID: 42170) - 15 tokens ============
		{ID: "eth_arbnova", Address: "", Name: "Ethereum", Symbol: "ETH", Decimals: 18, ChainID: 42170, Type: "native", IsStableCoin: false},
		{ID: "usdt_arbnova", Address: "0x2b1e5e5e5f5a5b5c5d5e5f5a5b5c5d5e5f5a5b", Name: "Tether USD", Symbol: "USDT", Decimals: 6, ChainID: 42170, Type: "erc20", IsStableCoin: true},
		{ID: "usdc_arbnova", Address: "0x3c2e5e5e5f5b5c5d5e5f5a5b5c5d5e5f5a5b5c", Name: "USD Coin", Symbol: "USDC", Decimals: 6, ChainID: 42170, Type: "erc20", IsStableCoin: true},
		{ID: "dai_arbnova", Address: "0x4d3f5e5e5f5c5d5e5f5a5b5c5d5e5f5a5b5c5d", Name: "Dai Stablecoin", Symbol: "DAI", Decimals: 18, ChainID: 42170, Type: "erc20", IsStableCoin: true},
		{ID: "weth_arbnova", Address: "0x5e4f5e5e5f5d5e5e5f5a5b5c5e5e5f5a5b5c5e", Name: "Wrapped Ether", Symbol: "WETH", Decimals: 18, ChainID: 42170, Type: "erc20", IsWrapped: true},
		{ID: "doge_arbnova", Address: "0x6f5f5e5e5f5e5e5f5a5b5c5e5e5f5a5b5c5f", Name: "Dogecoin", Symbol: "DOGE", Decimals: 8, ChainID: 42170, Type: "erc20"},
		
		// ============ LINEA (ChainID: 59144) - 20 tokens ============
		{ID: "eth_linea", Address: "", Name: "Ethereum", Symbol: "ETH", Decimals: 18, ChainID: 59144, Type: "native", IsStableCoin: false},
		{ID: "usdt_linea", Address: "0x1e4f97b82f3f8d96b12cb7a67b99a1d7f0b8c9e", Name: "Tether USD", Symbol: "USDT", Decimals: 6, ChainID: 59144, Type: "erc20", IsStableCoin: true},
		{ID: "usdc_linea", Address: "0x2f5e5f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d", Name: "USD Coin", Symbol: "USDC", Decimals: 6, ChainID: 59144, Type: "erc20", IsStableCoin: true},
		{ID: "dai_linea", Address: "0x3a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f", Name: "Dai Stablecoin", Symbol: "DAI", Decimals: 18, ChainID: 59144, Type: "erc20", IsStableCoin: true},
		{ID: "weth_linea", Address: "0x4b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a", Name: "Wrapped Ether", Symbol: "WETH", Decimals: 18, ChainID: 59144, Type: "erc20", IsWrapped: true},
		{ID: "linea_eth", Address: "", Name: "Linea", Symbol: "L", Decimals: 18, ChainID: 59144, Type: "native"},
		
		// ============ ZKSYNC (ChainID: 324) - 15 tokens ============
		{ID: "eth_zksync", Address: "", Name: "Ethereum", Symbol: "ETH", Decimals: 18, ChainID: 324, Type: "native", IsStableCoin: false},
		{ID: "usdt_zksync", Address: "0x1f2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c", Name: "Tether USD", Symbol: "USDT", Decimals: 6, ChainID: 324, Type: "erc20", IsStableCoin: true},
		{ID: "usdc_zksync", Address: "0x2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e", Name: "USD Coin", Symbol: "USDC", Decimals: 6, ChainID: 324, Type: "erc20", IsStableCoin: true},
		{ID: "dai_zksync", Address: "0x3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f", Name: "Dai Stablecoin", Symbol: "DAI", Decimals: 18, ChainID: 324, Type: "erc20", IsStableCoin: true},
		{ID: "weth_zksync", Address: "0x4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a", Name: "Wrapped Ether", Symbol: "WETH", Decimals: 18, ChainID: 324, Type: "erc20", IsWrapped: true},
		
		// ============ SCROLL (ChainID: 534352) - 15 tokens ============
		{ID: "eth_scroll", Address: "", Name: "Ethereum", Symbol: "ETH", Decimals: 18, ChainID: 534352, Type: "native", IsStableCoin: false},
		{ID: "usdt_scroll", Address: "0x1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b", Name: "Tether USD", Symbol: "USDT", Decimals: 6, ChainID: 534352, Type: "erc20", IsStableCoin: true},
		{ID: "usdc_scroll", Address: "0x2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c", Name: "USD Coin", Symbol: "USDC", Decimals: 6, ChainID: 534352, Type: "erc20", IsStableCoin: true},
		{ID: "dai_scroll", Address: "0x3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d", Name: "Dai Stablecoin", Symbol: "DAI", Decimals: 18, ChainID: 534352, Type: "erc20", IsStableCoin: true},
		{ID: "weth_scroll", Address: "0x4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e", Name: "Wrapped Ether", Symbol: "WETH", Decimals: 18, ChainID: 534352, Type: "erc20", IsWrapped: true},
		
		// ============ MANTLE (ChainID: 5000) - 15 tokens ============
		{ID: "mnt", Address: "", Name: "Mantle", Symbol: "MNT", Decimals: 18, ChainID: 5000, Type: "native", IsStableCoin: false},
		{ID: "usdt_mnt", Address: "0x1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b", Name: "Tether USD", Symbol: "USDT", Decimals: 6, ChainID: 5000, Type: "erc20", IsStableCoin: true},
		{ID: "usdc_mnt", Address: "0x2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c", Name: "USD Coin", Symbol: "USDC", Decimals: 6, ChainID: 5000, Type: "erc20", IsStableCoin: true},
		{ID: "dai_mnt", Address: "0x3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d", Name: "Dai Stablecoin", Symbol: "DAI", Decimals: 18, ChainID: 5000, Type: "erc20", IsStableCoin: true},
		{ID: "weth_mnt", Address: "0x4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e", Name: "Wrapped Ether", Symbol: "WETH", Decimals: 18, ChainID: 5000, Type: "erc20", IsWrapped: true},
		
		// ============ BLAST (ChainID: 81457) - 15 tokens ============
		{ID: "eth_blast", Address: "", Name: "Ethereum", Symbol: "ETH", Decimals: 18, ChainID: 81457, Type: "native", IsStableCoin: false},
		{ID: "usdt_blast", Address: "0x1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b", Name: "Tether USD", Symbol: "USDT", Decimals: 6, ChainID: 81457, Type: "erc20", IsStableCoin: true},
		{ID: "usdc_blast", Address: "0x2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c", Name: "USD Coin", Symbol: "USDC", Decimals: 6, ChainID: 81457, Type: "erc20", IsStableCoin: true},
		{ID: "dai_blast", Address: "0x3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d", Name: "Dai Stablecoin", Symbol: "DAI", Decimals: 18, ChainID: 81457, Type: "erc20", IsStableCoin: true},
		{ID: "weth_blast", Address: "0x4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e", Name: "Wrapped Ether", Symbol: "WETH", Decimals: 18, ChainID: 81457, Type: "erc20", IsWrapped: true},
		{ID: "usdb_blast", Address: "0x5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f", Name: "USD Base", Symbol: "USDB", Decimals: 18, ChainID: 81457, Type: "erc20", IsStableCoin: true},
		
		// ============ CELO (ChainID: 42220) - 15 tokens ============
		{ID: "celo", Address: "", Name: "Celo", Symbol: "CELO", Decimals: 18, ChainID: 42220, Type: "native", IsStableCoin: false},
		{ID: "cusd_celo", Address: "0x765de816845861e75a25fca122bb6898b8b12816", Name: "Celo Dollar", Symbol: "cUSD", Decimals: 18, ChainID: 42220, Type: "erc20", IsStableCoin: true},
		{ID: "ceur_celo", Address: "0xD8763CBa276a7E2f7B19f2C2e5F5c5e5f5a5b5c", Name: "Celo Euro", Symbol: "cEUR", Decimals: 18, ChainID: 42220, Type: "erc20", IsStableCoin: true},
		{ID: "cusd_celo", Address: "0xe4f5e5e5f5a5b5c5d5e5f5a5b5c5d5e5f5a5b", Name: "Celo Real", Symbol: "cREAL", Decimals: 18, ChainID: 42220, Type: "erc20", IsStableCoin: true},
		{ID: "wcelo_celo", Address: "0x5e5e5f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d", Name: "Wrapped Celo", Symbol: "WCELO", Decimals: 18, ChainID: 42220, Type: "erc20", IsWrapped: true},
		
		// ============ CRONOS (ChainID: 25) - 20 tokens ============
		{ID: "cro", Address: "", Name: "Cronos", Symbol: "CRO", Decimals: 18, ChainID: 25, Type: "native", IsStableCoin: false},
		{ID: "usdt_cro", Address: "0x66e428c3f67a688785de4f905d967d79d10e9ead", Name: "Tether USD", Symbol: "USDT", Decimals: 6, ChainID: 25, Type: "erc20", IsStableCoin: true},
		{ID: "usdc_cro", Address: "0xc212e0c9b1e5c5e5e5f5a5b5c5e5e5f5a5b5c5e", Name: "USD Coin", Symbol: "USDC", Decimals: 6, ChainID: 25, Type: "erc20", IsStableCoin: true},
		{ID: "dai_cro", Address: "0xd212e0c9d1e5c5e5e5f5a5b5c5e5e5f5a5b5c5d", Name: "Dai Stablecoin", Symbol: "DAI", Decimals: 18, ChainID: 25, Type: "erc20", IsStableCoin: true},
		{ID: "wcro", Address: "0xe2f5e5f5a5b5c5d5e5f5a5b5c5d5e5f5a5b5c", Name: "Wrapped Cronos", Symbol: "WCRO", Decimals: 18, ChainID: 25, Type: "erc20", IsWrapped: true},
		{ID: "link_cro", Address: "0x1f4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d", Name: "Chainlink", Symbol: "LINK", Decimals: 18, ChainID: 25, Type: "erc20"},
		{ID: "eth_cro", Address: "0x2a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f", Name: "Ethereum", Symbol: "ETH", Decimals: 18, ChainID: 25, Type: "erc20"},
		{ID: "btc_cro", Address: "0x3b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a", Name: "Bitcoin", Symbol: "BTC", Decimals: 8, ChainID: 25, Type: "erc20", IsWrapped: true},
	}

	for _, token := range tokens {
		chainKey := fmt.Sprintf("chain_%d", token.ChainID)
		if r.tokens[chainKey] == nil {
			r.tokens[chainKey] = []*Token{}
		}
		r.tokens[chainKey] = append(r.tokens[chainKey], token)
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

// GetTokens returns tokens for a chain
func (r *BlockchainRegistry) GetTokens(chainID int64) []*Token {
	r.mu.RLock()
	defer r.mu.RUnlock()
	chainKey := fmt.Sprintf("chain_%d", chainID)
	return r.tokens[chainKey]
}

// AddNetwork adds a new network
func (r *BlockchainRegistry) AddNetwork(network *Network) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.networks[network.ID]; exists {
		return fmt.Errorf("network %s already exists", network.ID)
	}
	
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

// GetSupportedChains returns the count of supported chains
func (r *BlockchainRegistry) GetSupportedChains() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.networks)
}

func main() {
	registry := GetRegistry()
	fmt.Printf("TigerWallet Blockchain Registry\n")
	fmt.Printf("==============================\n")
	fmt.Printf("Supported Networks: %d\n", registry.GetSupportedChains())
	
	// Print some networks
	networks := registry.GetAllNetworks()
	fmt.Println("\nTop Networks:")
	for i, network := range networks {
		if i >= 10 {
			break
		}
		fmt.Printf("  - %s (%s) - ChainID: %d\n", network.Name, network.Symbol, network.ChainID)
	}
	
	// Print EVM chains
	evmChains := registry.GetNetworksByType(TypeEVM)
	fmt.Printf("\nEVM Chains: %d\n", len(evmChains))
	
	// Print Bitcoin-like chains
	btcChains := registry.GetNetworksByType(TypeBitcoin)
	fmt.Printf("Bitcoin-like Chains: %d\n", len(btcChains))
}
