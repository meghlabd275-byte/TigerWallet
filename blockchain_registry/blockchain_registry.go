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
	}

	for _, network := range networks {
		r.networks[network.ID] = network
		if network.ChainID > 0 {
			r.chainIDs[network.ChainID] = network.ID
		}
	}
}

// initTokens initializes common tokens
func (r *BlockchainRegistry) initTokens() {
	tokens := []*Token{
		// Ethereum tokens
		{ID: "eth", Address: "", Name: "Ethereum", Symbol: "ETH", Decimals: 18, ChainID: 1, Type: "native", TotalSupply: "210000000000000000", IsStableCoin: false, IsWrapped: false},
		{ID: "usdt_eth", Address: "0xdac17f958d2ee523a2206206994597c13d831ec7", Name: "Tether USD", Symbol: "USDT", Decimals: 6, ChainID: 1, Type: "erc20", IsStableCoin: true},
		{ID: "usdc_eth", Address: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", Name: "USD Coin", Symbol: "USDC", Decimals: 6, ChainID: 1, Type: "erc20", IsStableCoin: true},
		{ID: "dai_eth", Address: "0x6b175474e89094c44da98b954eedeac495271d0f", Name: "Dai Stablecoin", Symbol: "DAI", Decimals: 18, ChainID: 1, Type: "erc20", IsStableCoin: true},
		{ID: "wbtc_eth", Address: "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599", Name: "Wrapped Bitcoin", Symbol: "WBTC", Decimals: 8, ChainID: 1, Type: "erc20", IsWrapped: true},
		{ID: "link_eth", Address: "0x514910771af9ca656af840dff83e8264ecf986ca", Name: "Chainlink", Symbol: "LINK", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "uni_eth", Address: "0x1f9840a85d5af5bf1d1762f925bdaddc4201f984", Name: "Uniswap", Symbol: "UNI", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "aave_eth", Address: "0x7fc66500c84a76ad7e9c93437bfc5ac33e2ddae9", Name: "Aave", Symbol: "AAVE", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "matic_eth", Address: "0x7d1afa7b718fb893db30a3abc0cfc608aacfebb0", Name: "Polygon", Symbol: "MATIC", Decimals: 18, ChainID: 1, Type: "erc20"},
		
		// BSC tokens
		{ID: "bnb_bsc", Address: "", Name: "BNB", Symbol: "BNB", Decimals: 18, ChainID: 56, Type: "native", IsStableCoin: false},
		{ID: "usdt_bsc", Address: "0x55d398326f99059ff775485246999027b3197955", Name: "Tether USD", Symbol: "USDT", Decimals: 18, ChainID: 56, Type: "bep20", IsStableCoin: true},
		{ID: "usdc_bsc", Address: "0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d", Name: "USD Coin", Symbol: "USDC", Decimals: 18, ChainID: 56, Type: "bep20", IsStableCoin: true},
		{ID: "busd_bsc", Address: "0xe9e7cea3dedca5984780bafc599bd69add087d56", Name: "Binance USD", Symbol: "BUSD", Decimals: 18, ChainID: 56, Type: "bep20", IsStableCoin: true},
		
		// Polygon tokens
		{ID: "matic_pol", Address: "", Name: "Polygon", Symbol: "MATIC", Decimals: 18, ChainID: 137, Type: "native", IsStableCoin: false},
		{ID: "usdt_pol", Address: "0xc2132d05d31c914a87c6611c10748aeb04b58e8f", Name: "Tether USD", Symbol: "USDT", Decimals: 6, ChainID: 137, Type: "erc20", IsStableCoin: true},
		{ID: "usdc_pol", Address: "0x2791bca1f2de4661ed88a30c99a7a9449aa84174", Name: "USD Coin", Symbol: "USDC", Decimals: 6, ChainID: 137, Type: "erc20", IsStableCoin: true},
		
		// Solana tokens
		{ID: "sol", Address: "", Name: "Solana", Symbol: "SOL", Decimals: 9, ChainID: 0, Type: "native", IsStableCoin: false},
		{ID: "usdc_sol", Address: "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", Name: "USD Coin", Symbol: "USDC", Decimals: 6, ChainID: 0, Type: "spl"},
		{ID: "usdt_sol", Address: "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB", Name: "Tether USD", Symbol: "USDT", Decimals: 6, ChainID: 0, Type: "spl"},
		
		// Tron tokens
		{ID: "trx", Address: "", Name: "Tron", Symbol: "TRX", Decimals: 6, ChainID: 195, Type: "native", IsStableCoin: false},
		{ID: "usdt_trx", Address: "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t", Name: "Tether USD", Symbol: "USDT", Decimals: 6, ChainID: 195, Type: "trc20", IsStableCoin: true},
		
		// Bitcoin
		{ID: "btc", Address: "", Name: "Bitcoin", Symbol: "BTC", Decimals: 8, ChainID: 0, Type: "native", IsStableCoin: false},
		
		// More tokens can be added as needed
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
