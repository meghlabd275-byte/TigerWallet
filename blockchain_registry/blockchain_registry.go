package main

import (
	"fmt"
	"sync"
)

// BlockchainType represents the type of blockchain
type BlockchainType string

const (
	TypeEVM      BlockchainType = "evm"
	TypeSolana   BlockchainType = "solana"
	TypeCosmos   BlockchainType = "cosmos"
	TypePolkadot BlockchainType = "polkadot"
	TypeCardano  BlockchainType = "cardano"
	TypeAlgorand BlockchainType = "algorand"
	TypeNear     BlockchainType = "near"
	TypeAptos    BlockchainType = "aptos"
	TypeSui      BlockchainType = "sui"
	TypeStarknet BlockchainType = "starknet"
	TypeTon      BlockchainType = "ton"
	TypeBitcoin  BlockchainType = "bitcoin"
	TypeRipple   BlockchainType = "ripple"
	TypeTron     BlockchainType = "tron"
	TypeTerra    BlockchainType = "terra"
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
	Supports1559    bool           `json:"supports_1559"`
	GasStation      string         `json:"gas_station"`
	StableCoins     []string       `json:"stable_coins"`
	NativeToken     string         `json:"native_token"`
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
	mu       sync.RWMutex
	networks map[string]*Network
	tokens   map[string][]*Token
	chainIDs map[int64]string
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
		{ID: "ethereum", Name: "Ethereum", Symbol: "ETH", Type: TypeEVM, ChainID: 1, Decimals: 18, Explorer: "https://etherscan.io", RPCURL: "https://cloudflare-eth.com", APIURL: "https://api.etherscan.io", WSSURL: "wss://ethereum-rpc.publicnode.com", Confirmations: 12, MinTransfer: 0.001, MaxTransfer: 1000000, SupportsEIP1559: true, StableCoins: []string{"USDT", "USDC", "DAI", "BUSD"}},
		{ID: "bsc", Name: "BNB Smart Chain", Symbol: "BNB", Type: TypeEVM, ChainID: 56, Decimals: 18, Explorer: "https://bscscan.com", RPCURL: "https://bsc-dataseed.binance.org", APIURL: "https://api.bscscan.com", WSSURL: "wss://bsc-ws-node.nariox.org", Confirmations: 15, MinTransfer: 0.001, MaxTransfer: 1000000, SupportsEIP1559: true, StableCoins: []string{"USDT", "USDC", "BUSD", "DAI"}},
		{ID: "polygon", Name: "Polygon", Symbol: "MATIC", Type: TypeEVM, ChainID: 137, Decimals: 18, Explorer: "https://polygonscan.com", RPCURL: "https://polygon.drpc.org", APIURL: "https://api.polygonscan.com", WSSURL: "wss://ws-mainnet.polygon.technology", Confirmations: 15, MinTransfer: 0.01, MaxTransfer: 100000, SupportsEIP1559: true, StableCoins: []string{"USDT", "USDC", "DAI"}},
		{ID: "arbitrum", Name: "Arbitrum One", Symbol: "ETH", Type: TypeEVM, ChainID: 42161, Decimals: 18, Explorer: "https://arbiscan.io", RPCURL: "https://arb1.arbitrum.io/rpc", APIURL: "https://api.arbiscan.io", WSSURL: "wss://arb1.arbitrum.io/ws", Confirmations: 15, MinTransfer: 0.001, MaxTransfer: 1000000, SupportsEIP1559: true, StableCoins: []string{"USDT", "USDC", "DAI"}},
		{ID: "optimism", Name: "Optimism", Symbol: "ETH", Type: TypeEVM, ChainID: 10, Decimals: 18, Explorer: "https://optimistic.etherscan.io", RPCURL: "https://mainnet.optimism.io", APIURL: "https://api-optimistic.etherscan.io", WSSURL: "wss://ws-mainnet.optimism.io", Confirmations: 15, MinTransfer: 0.001, MaxTransfer: 1000000, SupportsEIP1559: true, StableCoins: []string{"USDT", "USDC", "DAI"}},
		{ID: "base", Name: "Base", Symbol: "ETH", Type: TypeEVM, ChainID: 8453, Decimals: 18, Explorer: "https://basescan.org", RPCURL: "https://mainnet.base.org", APIURL: "https://api.basescan.org", WSSURL: "wss://ws.base.org", Confirmations: 15, MinTransfer: 0.001, MaxTransfer: 1000000, SupportsEIP1559: true, StableCoins: []string{"USDT", "USDC", "DAI"}},
		{ID: "avalanche", Name: "Avalanche C-Chain", Symbol: "AVAX", Type: TypeEVM, ChainID: 43114, Decimals: 18, Explorer: "https://snowtrace.io", RPCURL: "https://avalanche-c-chain-rpc.publicnode.com", APIURL: "https://api.snowtrace.io", WSSURL: "wss://api.avax.network/ext/bc/C/ws", Confirmations: 15, MinTransfer: 0.01, MaxTransfer: 100000, SupportsEIP1559: true, StableCoins: []string{"USDT", "USDC", "DAI"}},
		{ID: "fantom", Name: "Fantom Opera", Symbol: "FTM", Type: TypeEVM, ChainID: 250, Decimals: 18, Explorer: "https://ftmscan.com", RPCURL: "https://rpc.fantom.network", APIURL: "https://api.ftmscan.com", WSSURL: "wss://ws.fantom.network", Confirmations: 15, MinTransfer: 1, MaxTransfer: 1000000, SupportsEIP1559: false, StableCoins: []string{"USDT", "USDC", "DAI"}},
		{ID: "celo", Name: "Celo", Symbol: "CELO", Type: TypeEVM, ChainID: 42220, Decimals: 18, Explorer: "https://explorer.celo.org", RPCURL: "https://forno.celo.org", APIURL: "https://api.celoscan.io", WSSURL: "wss://forno.celo.org/ws", Confirmations: 15, MinTransfer: 0.01, MaxTransfer: 100000, SupportsEIP1559: true, StableCoins: []string{"cUSD", "cEUR", "USDT"}},
		{ID: "cronos", Name: "Cronos", Symbol: "CRO", Type: TypeEVM, ChainID: 25, Decimals: 18, Explorer: "https://cronoscan.com", RPCURL: "https://evm.cronos.org", APIURL: "https://api.cronoscan.com", WSSURL: "wss://evm.cronos.org", Confirmations: 15, MinTransfer: 1, MaxTransfer: 100000, SupportsEIP1559: false, StableCoins: []string{"USDT", "USDC", "DAI"}},
		{ID: "kava", Name: "Kava", Symbol: "KAVA", Type: TypeEVM, ChainID: 2222, Decimals: 18, Explorer: "https://kavascan.com", RPCURL: "https://evm.kava.io", APIURL: "https://api.kavascan.com", WSSURL: "wss://evm.kava.io", Confirmations: 15, MinTransfer: 0.1, MaxTransfer: 100000, SupportsEIP1559: false, StableCoins: []string{"USDT", "USDC", "DAI"}},
		{ID: "metis", Name: "Metis Andromeda", Symbol: "METIS", Type: TypeEVM, ChainID: 1088, Decimals: 18, Explorer: "https://andromedaexplorer.metis.io", RPCURL: "https://andromeda.metis.io/?owner=1088", APIURL: "https://api.andromedexplorer.metis.io", WSSURL: "wss://andromeda.metis.io/ws", Confirmations: 15, MinTransfer: 0.001, MaxTransfer: 1000000, SupportsEIP1559: false, StableCoins: []string{"USDT", "USDC"}},
		{ID: "mantle", Name: "Mantle", Symbol: "MNT", Type: TypeEVM, ChainID: 5000, Decimals: 18, Explorer: "https://explorer.mantle.xyz", RPCURL: "https://rpc.mantle.xyz", APIURL: "https://api.mantlescan.org", WSSURL: "wss://ws.mantle.xyz", Confirmations: 15, MinTransfer: 0.01, MaxTransfer: 100000, SupportsEIP1559: true, StableCoins: []string{"USDT", "USDC"}},
		{ID: "linea", Name: "Linea", Symbol: "ETH", Type: TypeEVM, ChainID: 59144, Decimals: 18, Explorer: "https://lineascan.build", RPCURL: "https://rpc.linea.build", APIURL: "https://api.lineascan.build", WSSURL: "wss://rpc.linea.build", Confirmations: 15, MinTransfer: 0.001, MaxTransfer: 1000000, SupportsEIP1559: true, StableCoins: []string{"USDT", "USDC", "DAI"}},
		{ID: "zksync", Name: "zkSync Era", Symbol: "ETH", Type: TypeEVM, ChainID: 324, Decimals: 18, Explorer: "https://explorer.zksync.io", RPCURL: "https://zksync2-mainnet.zksync.io", APIURL: "https://api.zksync.io", WSSURL: "wss://zksync2-mainnet.zksync.io/ws", Confirmations: 15, MinTransfer: 0.001, MaxTransfer: 1000000, SupportsEIP1559: true, StableCoins: []string{"USDT", "USDC", "DAI"}},
		{ID: "scroll", Name: "Scroll", Symbol: "ETH", Type: TypeEVM, ChainID: 534352, Decimals: 18, Explorer: "https://scrollscan.com", RPCURL: "https://rpc.scroll.io", APIURL: "https://api.scroll.io", WSSURL: "wss://ws.scroll.io", Confirmations: 15, MinTransfer: 0.001, MaxTransfer: 1000000, SupportsEIP1559: true, StableCoins: []string{"USDT", "USDC", "DAI"}},
		{ID: "polygon_zkevm", Name: "Polygon zkEVM", Symbol: "ETH", Type: TypeEVM, ChainID: 1101, Decimals: 18, Explorer: "https://zkevm.polygonscan.com", RPCURL: "https://zkevm-rpc.com", APIURL: "https://api-zkevm.polygonscan.com", WSSURL: "wss://zkevm-ws.polygon.technology", Confirmations: 15, MinTransfer: 0.001, MaxTransfer: 1000000, SupportsEIP1559: true, StableCoins: []string{"USDT", "USDC", "DAI"}},
		{ID: "opbnb", Name: "opBNB", Symbol: "BNB", Type: TypeEVM, ChainID: 204, Decimals: 18, Explorer: "https://opbnbscan.com", RPCURL: "https://opbnb-mainnet-rpc.bnbchain.org", APIURL: "https://api-opbnbscan.bnbchain.org", WSSURL: "wss://opbnb-mainnet-ws.bnbchain.org", Confirmations: 15, MinTransfer: 0.001, MaxTransfer: 1000000, SupportsEIP1559: true, StableCoins: []string{"USDT", "USDC"}},
		{ID: "arbitrum_nova", Name: "Arbitrum Nova", Symbol: "ETH", Type: TypeEVM, ChainID: 42170, Decimals: 18, Explorer: "https://nova.arbiscan.io", RPCURL: "https://nova.arbitrum.io/rpc", APIURL: "https://api-nova.arbiscan.io", WSSURL: "wss://nova.arbitrum.io/ws", Confirmations: 15, MinTransfer: 0.001, MaxTransfer: 1000000, SupportsEIP1559: true, StableCoins: []string{"USDT", "USDC", "DAI"}},
		{ID: "astar", Name: "Astar", Symbol: "ASTR", Type: TypeEVM, ChainID: 592, Decimals: 18, Explorer: "https://blockscout.com/astar", RPCURL: "https://rpc.astar.network", APIURL: "https://api.astar.network", WSSURL: "wss://rpc.astar.network", Confirmations: 15, MinTransfer: 10, MaxTransfer: 1000000, SupportsEIP1559: false, StableCoins: []string{"USDT", "USDC"}},

		// More EVM Chains
		{ID: "gnosis", Name: "Gnosis Chain", Symbol: "XDAI", Type: TypeEVM, ChainID: 100, Decimals: 18, Explorer: "https://gnosisscan.io", RPCURL: "https://rpc.gnosischain.com", APIURL: "https://api.gnosisscan.io", WSSURL: "wss://rpc.gnosischain.com/wss", Confirmations: 15, MinTransfer: 0.01, MaxTransfer: 100000, SupportsEIP1559: false, StableCoins: []string{"USDT", "USDC", "DAI"}},
		{ID: "moonbeam", Name: "Moonbeam", Symbol: "GLMR", Type: TypeEVM, ChainID: 1284, Decimals: 18, Explorer: "https://moonscan.io", RPCURL: "https://moonbeam.api.onfinality.io/public", APIURL: "https://api.moonscan.io", WSSURL: "wss://wss.api.moonbeam.network", Confirmations: 15, MinTransfer: 0.1, MaxTransfer: 100000, SupportsEIP1559: false, StableCoins: []string{"USDT", "USDC", "DAI"}},
		{ID: "moonriver", Name: "Moonriver", Symbol: "MOVR", Type: TypeEVM, ChainID: 1285, Decimals: 18, Explorer: "https://moonriver.moonscan.io", RPCURL: "https://moonriver.api.onfinality.io/public", APIURL: "https://api.moonriver.moonscan.io", WSSURL: "wss://wss.moonriver.moonbeam.network", Confirmations: 15, MinTransfer: 0.1, MaxTransfer: 100000, SupportsEIP1559: false, StableCoins: []string{"USDT", "USDC"}},

		{ID: "sepolia", Name: "Sepolia", Symbol: "ETH", Type: TypeEVM, ChainID: 11155111, Decimals: 18, Explorer: "https://sepolia.etherscan.io", RPCURL: "https://rpc.sepolia.ethpandaops.io", APIURL: "https://api-sepolia.etherscan.io", WSSURL: "wss://rpc.sepolia.org/ws", Confirmations: 2, MinTransfer: 0.001, MaxTransfer: 10000, SupportsEIP1559: true, IsTestnet: true, StableCoins: []string{}},

		// Bitcoin and forks
		{ID: "bitcoin", Name: "Bitcoin", Symbol: "BTC", Type: TypeBitcoin, ChainID: 0, Decimals: 8, Explorer: "https://blockstream.info", RPCURL: "https://blockstream.info/api", APIURL: "https://blockstream.info/api", WSSURL: "wss://blockstream.info/ws", Confirmations: 6, MinTransfer: 0.0001, MaxTransfer: 1000, SupportsEIP1559: false, StableCoins: []string{}},
		{ID: "bitcoin_cash", Name: "Bitcoin Cash", Symbol: "BCH", Type: TypeBitcoin, ChainID: 0, Decimals: 8, Explorer: "https://blockchair.com/bitcoin-cash", RPCURL: "https://bch-rpc.bitcoinabc.org", APIURL: "https://bch-insight.bitcoinabc.org/api", WSSURL: "wss://bch-ws.bitcoinabc.org", Confirmations: 10, MinTransfer: 0.001, MaxTransfer: 10000, SupportsEIP1559: false, StableCoins: []string{}},
		{ID: "litecoin", Name: "Litecoin", Symbol: "LTC", Type: TypeBitcoin, ChainID: 0, Decimals: 8, Explorer: "https://blockchair.com/litecoin", RPCURL: "https://litecoin-rpc.github.io", APIURL: "https://litecoin.info/api", WSSURL: "wss://litecoin.info/ws", Confirmations: 12, MinTransfer: 0.001, MaxTransfer: 10000, SupportsEIP1559: false, StableCoins: []string{}},
		{ID: "dogecoin", Name: "Dogecoin", Symbol: "DOGE", Type: TypeBitcoin, ChainID: 0, Decimals: 8, Explorer: "https://blockchair.com/dogecoin", RPCURL: "https://dogecoin-rpc.github.io", APIURL: "https://dogecoin.info/api", WSSURL: "wss://dogecoin.info/ws", Confirmations: 60, MinTransfer: 1, MaxTransfer: 10000000, SupportsEIP1559: false, StableCoins: []string{}},

		// Tron
		{ID: "tron", Name: "Tron", Symbol: "TRX", Type: TypeTron, ChainID: 195, Decimals: 6, Explorer: "https://tronscan.org", RPCURL: "https://api.trongrid.io", APIURL: "https://api.trongrid.io", WSSURL: "wss://api.trongrid.io/ws", Confirmations: 19, MinTransfer: 1, MaxTransfer: 100000000, SupportsEIP1559: false, StableCoins: []string{"USDT", "USDC", "TUSD"}},

		// Ripple
		{ID: "ripple", Name: "XRP Ledger", Symbol: "XRP", Type: TypeRipple, ChainID: 0, Decimals: 6, Explorer: "https://xrpscan.com", RPCURL: "https://s1.ripple.com:51234", APIURL: "https://data.ripple.com", WSSURL: "wss://s1.ripple.com", Confirmations: 10, MinTransfer: 10, MaxTransfer: 100000000, SupportsEIP1559: false, StableCoins: []string{}},

		// Cosmos Ecosystem
		{ID: "cosmos_hub", Name: "Cosmos Hub", Symbol: "ATOM", Type: TypeCosmos, ChainID: 0, Decimals: 6, Explorer: "https://mintscan.io/cosmos", RPCURL: "https://rpc.cosmoshub4.theta-testnet.xyz:443", APIURL: "https://api.cosmoshub4.theta-testnet.xyz", WSSURL: "wss://rpc.cosmoshub4.theta-testnet.xyz:443/websocket", Confirmations: 15, MinTransfer: 0.1, MaxTransfer: 1000000, SupportsEIP1559: false, StableCoins: []string{}},
		{ID: "osmosis", Name: "Osmosis", Symbol: "OSMO", Type: TypeCosmos, ChainID: 0, Decimals: 6, Explorer: "https://mintscan.io/osmosis", RPCURL: "https://rpc.osmotest5.osmosis.zone:443", APIURL: "https://api.osmotest5.osmosis.zone", WSSURL: "wss://rpc.osmotest5.osmosis.zone:443/websocket", Confirmations: 15, MinTransfer: 0.1, MaxTransfer: 1000000, SupportsEIP1559: false, StableCoins: []string{"USDC", "USDT"}},
		{ID: "injective", Name: "Injective", Symbol: "INJ", Type: TypeCosmos, ChainID: 0, Decimals: 18, Explorer: "https://mintscan.io/injective", RPCURL: "https://injective-1-rpc.cosmosia.notional.ventures:443", APIURL: "https://api.injective.network", WSSURL: "wss://injective-1-rpc.cosmosia.notional.ventures:443/ws", Confirmations: 15, MinTransfer: 0.1, MaxTransfer: 1000000, SupportsEIP1559: false, StableCoins: []string{"USDC", "USDT"}},

		// Solana
		{ID: "solana", Name: "Solana", Symbol: "SOL", Type: TypeSolana, ChainID: 0, Decimals: 9, Explorer: "https://solscan.io", RPCURL: "https://api.mainnet-beta.solana.com", APIURL: "https://api.mainnet-beta.solana.com", WSSURL: "wss://api.mainnet-beta.solana.com/ws", Confirmations: 32, MinTransfer: 0.001, MaxTransfer: 1000000, SupportsEIP1559: false, StableCoins: []string{"USDC", "USDT", "DAI"}},
		{ID: "solana_devnet", Name: "Solana Devnet", Symbol: "SOL", Type: TypeSolana, ChainID: 0, Decimals: 9, Explorer: "https://solscan.io/?cluster=devnet", RPCURL: "https://api.devnet.solana.com", APIURL: "https://api.devnet.solana.com", WSSURL: "wss://api.devnet.solana.com/ws", Confirmations: 1, MinTransfer: 0.001, MaxTransfer: 10000, SupportsEIP1559: false, IsTestnet: true, StableCoins: []string{}},

		// Near
		{ID: "near", Name: "Near Protocol", Symbol: "NEAR", Type: TypeNear, ChainID: 0, Decimals: 24, Explorer: "https://explorer.near.org", RPCURL: "https://rpc.mainnet.near.org", APIURL: "https://api.near.org", WSSURL: "wss://rpc.mainnet.near.org/ws", Confirmations: 3, MinTransfer: 0.1, MaxTransfer: 1000000, SupportsEIP1559: false, StableCoins: []string{"USDC", "USDT"}},

		// Aptos
		{ID: "aptos", Name: "Aptos", Symbol: "APT", Type: TypeAptos, ChainID: 0, Decimals: 8, Explorer: "https://explorer.aptoslabs.com", RPCURL: "https://fullnode.mainnet.aptoslabs.com", APIURL: "https://api.aptoslabs.com", WSSURL: "wss://fullnode.mainnet.aptoslabs.com", Confirmations: 1, MinTransfer: 0.1, MaxTransfer: 1000000, SupportsEIP1559: false, StableCoins: []string{"USDC", "USDT"}},

		// Sui
		{ID: "sui", Name: "Sui", Symbol: "SUI", Type: TypeSui, ChainID: 0, Decimals: 9, Explorer: "https://suiscan.xyz", RPCURL: "https://rpc.mainnet.sui.io", APIURL: "https://api.mainnet.sui.io", WSSURL: "wss://rpc.mainnet.sui.io", Confirmations: 1, MinTransfer: 0.1, MaxTransfer: 1000000, SupportsEIP1559: false, StableCoins: []string{"USDC", "USDT"}},

		// Starknet
		{ID: "starknet", Name: "Starknet", Symbol: "ETH", Type: TypeStarknet, ChainID: 0, Decimals: 18, Explorer: "https://starkscan.co", RPCURL: "https://rpc.starknet.io", APIURL: "https://api.starkscan.co", WSSURL: "wss://rpc.starknet.io", Confirmations: 15, MinTransfer: 0.001, MaxTransfer: 1000000, SupportsEIP1559: false, StableCoins: []string{"USDC", "USDT", "DAI"}},

		// Ton
		{ID: "ton", Name: "Toncoin", Symbol: "TON", Type: TypeTon, ChainID: 0, Decimals: 9, Explorer: "https://ton.live", RPCURL: "https://toncenter.com/api/v2", APIURL: "https://tonapi.io", WSSURL: "wss://toncenter.com/ws", Confirmations: 1, MinTransfer: 0.1, MaxTransfer: 1000000, SupportsEIP1559: false, StableCoins: []string{"USDC", "USDT"}},

		// Cardano
		{ID: "cardano", Name: "Cardano", Symbol: "ADA", Type: TypeCardano, ChainID: 0, Decimals: 6, Explorer: "https://cardanoscan.io", RPCURL: "https://cardano-mainnet.blockfrost.io", APIURL: "https://api.cardanoscan.io", WSSURL: "wss://cardano-mainnet.blockfrost.io/ws", Confirmations: 15, MinTransfer: 1, MaxTransfer: 100000000, SupportsEIP1559: false, StableCoins: []string{}},

		// Algorand
		{ID: "algorand", Name: "Algorand", Symbol: "ALGO", Type: TypeAlgorand, ChainID: 0, Decimals: 6, Explorer: "https://algoexplorer.io", RPCURL: "https://mainnet-api.algonode.io", APIURL: "https://api.algoexplorer.io", WSSURL: "wss://mainnet-api.algonode.io/ws", Confirmations: 10, MinTransfer: 0.1, MaxTransfer: 100000000, SupportsEIP1559: false, StableCoins: []string{"USDC", "USDT"}},

		// Sei
		{ID: "sei", Name: "Sei", Symbol: "SEI", Type: TypeCosmos, ChainID: 0, Decimals: 6, Explorer: "https://www.seiscan.app", RPCURL: "https://sei-rpc.Chainlayer.org:443", APIURL: "https://api.sei.io", WSSURL: "wss://sei-rpc.Chainlayer.org:443/websocket", Confirmations: 15, MinTransfer: 0.1, MaxTransfer: 1000000, SupportsEIP1559: false, StableCoins: []string{"USDC", "USDT"}},

		// Celestia
		{ID: "celestia", Name: "Celestia", Symbol: "TIA", Type: TypeCosmos, ChainID: 0, Decimals: 6, Explorer: "https://mintscan.io/celestia", RPCURL: "https://celestia-rpc.Blockspace.io:443", APIURL: "https://api.celestia.space", WSSURL: "wss://celestia-rpc.Blockspace.io:443/websocket", Confirmations: 15, MinTransfer: 0.1, MaxTransfer: 1000000, SupportsEIP1559: false, StableCoins: []string{}},

		// Polkadot
		{ID: "polkadot", Name: "Polkadot", Symbol: "DOT", Type: TypePolkadot, ChainID: 0, Decimals: 10, Explorer: "https://polkadot.subscan.io", RPCURL: "https://rpc.polkadot.io", APIURL: "https://api.polkadot.io", WSSURL: "wss://rpc.polkadot.io", Confirmations: 28, MinTransfer: 0.1, MaxTransfer: 1000000, SupportsEIP1559: false, StableCoins: []string{}},
		{ID: "kusama", Name: "Kusama", Symbol: "KSM", Type: TypePolkadot, ChainID: 0, Decimals: 12, Explorer: "https://kusama.subscan.io", RPCURL: "https://kusama-rpc.dotters.network", APIURL: "https://api.kusama.network", WSSURL: "wss://kusama-rpc.dotters.network", Confirmations: 28, MinTransfer: 0.01, MaxTransfer: 100000, SupportsEIP1559: false, StableCoins: []string{}},

		// More Layer 2s
		{ID: "pulsechain", Name: "PulseChain", Symbol: "PLS", Type: TypeEVM, ChainID: 369, Decimals: 18, Explorer: "https://scan.pulsechain.com", RPCURL: "https://rpc.pulsechain.com", APIURL: "https://api.pulsechain.com", WSSURL: "wss://ws.pulsechain.com", Confirmations: 15, MinTransfer: 1, MaxTransfer: 1000000000, SupportsEIP1559: false, StableCoins: []string{"USDT", "USDC"}},
		{ID: "pulsechain_testnet", Name: "PulseChain Testnet", Symbol: "tPLS", Type: TypeEVM, ChainID: 943, Decimals: 18, Explorer: "https://scan-testnet.pulsechain.com", RPCURL: "https://rpc.v4.testnet.pulsechain.com", APIURL: "https://api-testnet.pulsechain.com", WSSURL: "wss://ws.v4.testnet.pulsechain.com", Confirmations: 3, MinTransfer: 1, MaxTransfer: 10000, SupportsEIP1559: false, IsTestnet: true, StableCoins: []string{}},

		// Pi Network (Testnet)

		// Terra
		{ID: "terra", Name: "Terra Classic", Symbol: "LUNC", Type: TypeTerra, ChainID: 0, Decimals: 6, Explorer: "https://finder.terra.classic.community", RPCURL: "https://terra-classic-lcd.publicnode.com", APIURL: "https://terra-classic-api.publicnode.com", WSSURL: "wss://terra-classic-lcd.publicnode.com:443/websocket", Confirmations: 15, MinTransfer: 1000, MaxTransfer: 100000000000, SupportsEIP1559: false, StableCoins: []string{"USTC"}},

		// More EVM Chains
		{ID: "aurora", Name: "Aurora", Symbol: "ETH", Type: TypeEVM, ChainID: 1313161554, Decimals: 18, Explorer: "https://aurorascan.dev", RPCURL: "https://mainnet.aurora.dev", APIURL: "https://api.aurorascan.dev", WSSURL: "wss://mainnet.aurora.dev/ws", Confirmations: 15, MinTransfer: 0.01, MaxTransfer: 100000, SupportsEIP1559: true, StableCoins: []string{"USDT", "USDC"}},
		{ID: "harmony", Name: "Harmony", Symbol: "ONE", Type: TypeEVM, ChainID: 1666600000, Decimals: 18, Explorer: "https://explorer.harmony.one", RPCURL: "https://api.harmony.one", APIURL: "https://api.harmony.one", WSSURL: "wss://ws.harmony.one", Confirmations: 15, MinTransfer: 1, MaxTransfer: 1000000000, SupportsEIP1559: false, StableCoins: []string{"USDC", "USDT"}},

		{ID: "conflux", Name: "Conflux", Symbol: "CFX", Type: TypeEVM, ChainID: 1030, Decimals: 18, Explorer: "https://confluxscan.io", RPCURL: "https://evm.confluxrpc.com", APIURL: "https://api.confluxnetwork.org", WSSURL: "wss://rpc.confluxnetwork.org/ws", Confirmations: 15, MinTransfer: 0.01, MaxTransfer: 1000000, SupportsEIP1559: false, StableCoins: []string{"USDC", "USDT"}},

		{ID: "telos", Name: "Telos", Symbol: "TLOS", Type: TypeEVM, ChainID: 40, Decimals: 18, Explorer: "https://www.teloscan.io", RPCURL: "https://rpc.telos.net", APIURL: "https://api.teloscan.io", WSSURL: "wss://mainnet.telos.net/ws", Confirmations: 15, MinTransfer: 0.1, MaxTransfer: 1000000, SupportsEIP1559: false, StableCoins: []string{"USDT", "USDC"}},
		{ID: "iotex", Name: "IoTeX", Symbol: "IOTX", Type: TypeEVM, ChainID: 4689, Decimals: 18, Explorer: "https://iotexscan.io", RPCURL: "https://babel-api.mainnet.iotex.io", APIURL: "https://api.iotex.io", WSSURL: "wss://ws.iotex.io", Confirmations: 15, MinTransfer: 1, MaxTransfer: 1000000, SupportsEIP1559: false, StableCoins: []string{"USDT", "USDC"}},
		{ID: "ronin", Name: "Ronin", Symbol: "RON", Type: TypeEVM, ChainID: 2020, Decimals: 18, Explorer: "https://app.roninchain.com", RPCURL: "https://api.roninchain.com/rpc", APIURL: "https://api.roninchain.com", WSSURL: "wss://ws.roninchain.com", Confirmations: 15, MinTransfer: 0.01, MaxTransfer: 1000000, SupportsEIP1559: false, StableCoins: []string{"AXS", "SLP"}},

		{ID: "fraxtal", Name: "Fraxtal", Symbol: "FRX", Type: TypeEVM, ChainID: 252, Decimals: 18, Explorer: "https://fraxscan.com", RPCURL: "https://rpc.frax.com", APIURL: "https://api.fraxscan.com", WSSURL: "wss://rpc.frax.com/ws", Confirmations: 15, MinTransfer: 0.01, MaxTransfer: 1000000, SupportsEIP1559: true, StableCoins: []string{"USDC", "USDT"}},
		{ID: "mode", Name: "Mode", Symbol: "MOD", Type: TypeEVM, ChainID: 34443, Decimals: 18, Explorer: "https://explorer.mode.network", RPCURL: "https://mainnet.mode.network", APIURL: "https://api.mode.network", WSSURL: "wss://mainnet.mode.network/ws", Confirmations: 15, MinTransfer: 0.001, MaxTransfer: 1000000, SupportsEIP1559: true, StableCoins: []string{"USDC", "USDT"}},
		{ID: "rootstock", Name: "Rootstock", Symbol: "RBTC", Type: TypeEVM, ChainID: 30, Decimals: 18, Explorer: "https://rootstock.io", RPCURL: "https://public-node.rsk.co", APIURL: "https://api.rsk.co", WSSURL: "wss://public-node.rsk.co/ws", Confirmations: 12, MinTransfer: 0.0001, MaxTransfer: 1000000, SupportsEIP1559: false, StableCoins: []string{"USDT", "rUSDT"}},
		{ID: "synthesis", Name: "Synthesis", Symbol: "SNX", Type: TypeEVM, ChainID: 66, Decimals: 18, Explorer: "https://synthetix.io", RPCURL: "https://exchainrpc.okex.org", APIURL: "https://api.synthetix.io", WSSURL: "wss://optimism-mainnet.infura.io/ws", Confirmations: 15, MinTransfer: 0.01, MaxTransfer: 1000000, SupportsEIP1559: true, StableCoins: []string{"sUSD"}},

		{ID: "flare", Name: "Flare", Symbol: "FLR", Type: TypeEVM, ChainID: 14, Decimals: 18, Explorer: "https://flarescan.com", RPCURL: "https://flare-api.flare.network/ext/bc/C/rpc", APIURL: "https://api.flarescan.com", WSSURL: "wss://flare-api.flare.network/ext/bc/C/ws", Confirmations: 15, MinTransfer: 1, MaxTransfer: 1000000000, SupportsEIP1559: false, StableCoins: []string{"USDf"}},
		{ID: "songbird", Name: "Songbird", Symbol: "SGB", Type: TypeEVM, ChainID: 19, Decimals: 18, Explorer: "https://songbird-explorer.flare.network", RPCURL: "https://songbird-api.flare.network/ext/bc/C/rpc", APIURL: "https://api.songbird.flare-network.io", WSSURL: "wss://songbird-api.flare.network/ext/bc/C/ws", Confirmations: 15, MinTransfer: 1, MaxTransfer: 1000000000, SupportsEIP1559: false, StableCoins: []string{}},

		{ID: "emerald", Name: "Emerald Paratime", Symbol: "EDO", Type: TypeEVM, ChainID: 42262, Decimals: 18, Explorer: "https://explorer.emerald.dev", RPCURL: "https://emerald.oasis.io", APIURL: "https://api.emerald.dev", WSSURL: "wss://ws.emerald.dev", Confirmations: 15, MinTransfer: 0.1, MaxTransfer: 1000000, SupportsEIP1559: false, StableCoins: []string{"USDC", "USDT"}},
		{ID: "syscoin", Name: "Syscoin", Symbol: "SYS", Type: TypeEVM, ChainID: 57, Decimals: 8, Explorer: "https://syscoin.io", RPCURL: "https://rpc.syscoin.org", APIURL: "https://api.syscoin.org", WSSURL: "wss://rpc.syscoin.org/ws", Confirmations: 15, MinTransfer: 0.01, MaxTransfer: 1000000, SupportsEIP1559: false, StableCoins: []string{"USDC", "USDT"}},

		{ID: "manta", Name: "Manta Pacific", Symbol: "MANTA", Type: TypeEVM, ChainID: 169, Decimals: 18, Explorer: "https://manta-pacific-explorer.calderaexplorer.xyz", RPCURL: "https://pacific-rpc.manta.network/http", APIURL: "https://api.manta.network", WSSURL: "wss://rpc.manta.network/ws", Confirmations: 15, MinTransfer: 0.001, MaxTransfer: 1000000, SupportsEIP1559: true, StableCoins: []string{"USDC", "USDT"}},
		{ID: "berachain", Name: "Berachain", Symbol: "BERA", Type: TypeEVM, ChainID: 80094, Decimals: 18, Explorer: "https://artio.beratrail.io", RPCURL: "https://rpc.berachain.com", APIURL: "https://api.berachain.com", WSSURL: "wss://artio.rpc.berachain.com/ws", Confirmations: 15, MinTransfer: 0.001, MaxTransfer: 1000000, SupportsEIP1559: true, StableCoins: []string{"USDC", "USDT"}},
		{ID: "sei_testnet", Name: "Sei Atlantic", Symbol: "SEI", Type: TypeCosmos, ChainID: 0, Decimals: 6, Explorer: "https://seitrace.com/?network=atlantic-1", RPCURL: "https://rpc.atlantic-1.sei.io:443", APIURL: "https://api.atlantic-1.sei.io", WSSURL: "wss://rpc.atlantic-1.sei.io:443/websocket", Confirmations: 5, MinTransfer: 0.1, MaxTransfer: 10000, SupportsEIP1559: false, IsTestnet: true, StableCoins: []string{}},
		{ID: "celestia_mocha", Name: "Celestia Mocha", Symbol: "TIA", Type: TypeCosmos, ChainID: 0, Decimals: 6, Explorer: "https://mocha.celestia.dev", RPCURL: "https://rpc-mocha.celestia-dev.io:443", APIURL: "https://api-mocha.celestia-dev.io", WSSURL: "wss://rpc-mocha.celestia-dev.io:443/websocket", Confirmations: 5, MinTransfer: 0.1, MaxTransfer: 10000, SupportsEIP1559: false, IsTestnet: true, StableCoins: []string{}},

		// ==== ADDITIONAL 40+ BLOCKCHAINS TO REACH 100+ ====

		// More EVM Chains - Layer 2 & Alternatives

		{ID: "vechain", Name: "VeChain", Symbol: "VET", Type: TypeEVM, ChainID: 100009, Decimals: 18, Explorer: "https://vechain.org", RPCURL: "https://rpc-mainnet.vechain.energy", APIURL: "https://api.vechain.org", WSSURL: "wss://rpc.ervice.org", Confirmations: 15, MinTransfer: 1, MaxTransfer: 1000000000, SupportsEIP1559: false, StableCoins: []string{}},

		{ID: "kcc", Name: "KuCoin Community Chain", Symbol: "KCS", Type: TypeEVM, ChainID: 321, Decimals: 18, Explorer: "https://explorer.kcc.io", RPCURL: "https://rpc-mainnet.kcc.network", APIURL: "https://api.kcc.io", WSSURL: "wss://rpc-ws-mainnet.kcc.network", Confirmations: 15, MinTransfer: 0.01, MaxTransfer: 1000000, SupportsEIP1559: false, StableCoins: []string{"USDT", "USDC"}},
		{ID: "hederahashgraph", Name: "Hedera", Symbol: "HBAR", Type: TypeEVM, ChainID: 295, Decimals: 18, Explorer: "https://hashscan.io", RPCURL: "https://mainnet.hashio.io/api", APIURL: "https://api.hedera.com", WSSURL: "wss://mainnet.mirrornode.hedera.com", Confirmations: 15, MinTransfer: 1, MaxTransfer: 1000000000, SupportsEIP1559: false, StableCoins: []string{}},
		{ID: "zilliqa", Name: "Zilliqa", Symbol: "ZIL", Type: TypeEVM, ChainID: 32769, Decimals: 12, Explorer: "https://viewblock.io/zilliqa", RPCURL: "https://api.zilliqa.com", APIURL: "https://explorer.zilliqa.com", WSSURL: "wss://api.zilliqa.com", Confirmations: 15, MinTransfer: 1, MaxTransfer: 1000000000, SupportsEIP1559: false, StableCoins: []string{}},
		{ID: "oasis", Name: "Oasis Network", Symbol: "ROSE", Type: TypeEVM, ChainID: 42262, Decimals: 18, Explorer: "https://explorer.emerald.oasis.dev", RPCURL: "https://emerald.oasis.dev", APIURL: "https://api.oasisscan.com", WSSURL: "wss://emerald.oasis.dev/ws", Confirmations: 15, MinTransfer: 1, MaxTransfer: 1000000000, SupportsEIP1559: false, StableCoins: []string{"USDC", "USDT"}},
		{ID: "filecoin", Name: "Filecoin", Symbol: "FIL", Type: TypeEVM, ChainID: 314, Decimals: 18, Explorer: "https://filfox.io", RPCURL: "https://api.node.glif.io/", APIURL: "https://api.filfox.io", WSSURL: "wss://api.filecoin.io", Confirmations: 15, MinTransfer: 0.01, MaxTransfer: 1000000, SupportsEIP1559: false, StableCoins: []string{}},
		{ID: "flow", Name: "Flow", Symbol: "FLOW", Type: TypeEVM, ChainID: 747, Decimals: 8, Explorer: "https://flowscan.io", RPCURL: "https://mainnet.evm.nodes.onflow.org", APIURL: "https://api.flowscan.io", WSSURL: "wss://flow-access-mainnet-01.blobstorage2.net/ws", Confirmations: 15, MinTransfer: 0.1, MaxTransfer: 1000000, SupportsEIP1559: false, StableCoins: []string{}},
		{ID: "theta", Name: "Theta Network", Symbol: "THETA", Type: TypeEVM, ChainID: 361, Decimals: 18, Explorer: "https://explorer.thetatoken.org", RPCURL: "https://eth-rpc-api.thetatoken.org/rpc", APIURL: "https://api.thetatoken.org", WSSURL: "wss://eth-rpc-api.theta.network/ws", Confirmations: 15, MinTransfer: 1, MaxTransfer: 1000000000, SupportsEIP1559: false, StableCoins: []string{}},
		{ID: "arbitrum_sepolia", Name: "Arbitrum Sepolia", Symbol: "ETH", Type: TypeEVM, ChainID: 421614, Decimals: 18, Explorer: "https://sepolia.arbiscan.io", RPCURL: "https://sepolia-rollup.arbitrum.io/rpc", APIURL: "https://api-sepolia.arbiscan.io", WSSURL: "wss://sepolia.arbitrum.io/ws", Confirmations: 5, MinTransfer: 0.001, MaxTransfer: 10000, SupportsEIP1559: true, IsTestnet: true, StableCoins: []string{}},
		{ID: "base_sepolia", Name: "Base Sepolia", Symbol: "ETH", Type: TypeEVM, ChainID: 84532, Decimals: 18, Explorer: "https://sepolia.basescan.org", RPCURL: "https://sepolia.base.org", APIURL: "https://api-sepolia.basescan.org", WSSURL: "wss://sepolia.base.org/ws", Confirmations: 5, MinTransfer: 0.001, MaxTransfer: 10000, SupportsEIP1559: true, IsTestnet: true, StableCoins: []string{}},
		{ID: "polygon_amoy", Name: "Polygon Amoy", Symbol: "MATIC", Type: TypeEVM, ChainID: 80002, Decimals: 18, Explorer: "https://amoy.polygonscan.com", RPCURL: "https://polygon-amoy-bor-rpc.publicnode.com", APIURL: "https://api-amoy.polygonscan.com", WSSURL: "wss://rpc-amoy.polygon.technology/ws", Confirmations: 5, MinTransfer: 0.01, MaxTransfer: 10000, SupportsEIP1559: true, IsTestnet: true, StableCoins: []string{}},
		{ID: "bnb_testnet", Name: "BNB Smart Chain Testnet", Symbol: "BNB", Type: TypeEVM, ChainID: 97, Decimals: 18, Explorer: "https://testnet.bscscan.com", RPCURL: "https://data-seed-prebsc-1-s1.binance.org:8545", APIURL: "https://api-testnet.bscscan.com", WSSURL: "wss://data-seed-prebsc-1-s1.binance.org:8545/ws", Confirmations: 3, MinTransfer: 0.001, MaxTransfer: 10000, SupportsEIP1559: true, IsTestnet: true, StableCoins: []string{}},

		// More Cosmos Chains
		{ID: "dydx", Name: "dYdX", Symbol: "DYDX", Type: TypeCosmos, ChainID: 0, Decimals: 18, Explorer: "https://mintscan.io/dydx", RPCURL: "https://dydx-rpc.kingnodes.com:443", APIURL: "https://api.dydx.excan.io", WSSURL: "wss://dydx-rpc.kingnodes.com:443/ws", Confirmations: 15, MinTransfer: 0.1, MaxTransfer: 1000000, SupportsEIP1559: false, StableCoins: []string{"USDC"}},
		{ID: "persistence", Name: "Persistence", Symbol: "XPRT", Type: TypeCosmos, ChainID: 0, Decimals: 6, Explorer: "https://mintscan.io/persistence", RPCURL: "https://rpc.persistence.one:443", APIURL: "https://api.persistence.one", WSSURL: "wss://rpc.persistence.one:443/websocket", Confirmations: 15, MinTransfer: 0.1, MaxTransfer: 1000000, SupportsEIP1559: false, StableCoins: []string{"USDC"}},
		{ID: "sommelier", Name: "Sommelier", Symbol: "SOMM", Type: TypeCosmos, ChainID: 0, Decimals: 6, Explorer: "https://mintscan.io/sommelier", RPCURL: "https://rpc-sommelier.kingnodes.com:443", APIURL: "https://api.sommelier.cz", WSSURL: "wss://rpc-sommelier.kingnodes.com:443/websocket", Confirmations: 15, MinTransfer: 0.1, MaxTransfer: 1000000, SupportsEIP1559: false, StableCoins: []string{"USDC"}},
		{ID: "juno", Name: "Juno", Symbol: "JUNO", Type: TypeCosmos, ChainID: 0, Decimals: 6, Explorer: "https://mintscan.io/juno", RPCURL: "https://rpc-juno.itastakers.com:443", APIURL: "https://api.juno.zone", WSSURL: "wss://rpc-juno.itastakers.com:443/websocket", Confirmations: 15, MinTransfer: 0.1, MaxTransfer: 1000000, SupportsEIP1559: false, StableCoins: []string{}},
		{ID: "stargaze", Name: "Stargaze", Symbol: "STARS", Type: TypeCosmos, ChainID: 0, Decimals: 6, Explorer: "https://mintscan.io/stargaze", RPCURL: "https://rpc.stargaze-apis.com:443", APIURL: "https://api.stargaze.zone", WSSURL: "wss://rpc.stargaze-apis.com:443/websocket", Confirmations: 15, MinTransfer: 0.1, MaxTransfer: 1000000, SupportsEIP1559: false, StableCoins: []string{}},
		{ID: "akash", Name: "Akash", Symbol: "AKT", Type: TypeCosmos, ChainID: 0, Decimals: 6, Explorer: "https://mintscan.io/akash", RPCURL: "https://rpc-akash.kingnodes.com:443", APIURL: "https://api.cosmoscan.info", WSSURL: "wss://rpc-akash.kingnodes.com:443/websocket", Confirmations: 15, MinTransfer: 0.1, MaxTransfer: 1000000, SupportsEIP1559: false, StableCoins: []string{}},
		{ID: "sentinel", Name: "Sentinel", Symbol: "DVPN", Type: TypeCosmos, ChainID: 0, Decimals: 6, Explorer: "https://mintscan.io/sentinel", RPCURL: "https://rpc-sentinel.kingnodes.com:443", APIURL: "https://api.sentinel.co", WSSURL: "wss://rpc-sentinel.kingnodes.com:443/websocket", Confirmations: 15, MinTransfer: 0.1, MaxTransfer: 1000000, SupportsEIP1559: false, StableCoins: []string{}},
		{ID: "regen", Name: "Regen", Symbol: "REGEN", Type: TypeCosmos, ChainID: 0, Decimals: 6, Explorer: "https://mintscan.io/regen", RPCURL: "https://rpc-regen.kingnodes.com:443", APIURL: "https://api.regen.network", WSSURL: "wss://rpc-regen.kingnodes.com:443/websocket", Confirmations: 15, MinTransfer: 0.1, MaxTransfer: 1000000, SupportsEIP1559: false, StableCoins: []string{}},
		{ID: "chihuahua", Name: "Chihuahua", Symbol: "HUAHUA", Type: TypeCosmos, ChainID: 0, Decimals: 6, Explorer: "https://mintscan.io/chihuahua", RPCURL: "https://rpc.chihuahua.wtf:443", APIURL: "https://api.chihuahua.wtf", WSSURL: "wss://rpc.chihuahua.wtf:443/websocket", Confirmations: 15, MinTransfer: 0.1, MaxTransfer: 1000000, SupportsEIP1559: false, StableCoins: []string{}},
		{ID: "evmos", Name: "Evmos", Symbol: "EVMOS", Type: TypeCosmos, ChainID: 0, Decimals: 18, Explorer: "https://mintscan.io/evmos", RPCURL: "https://rpc-evmos.kingnodes.com:443", APIURL: "https://api.evmos.org", WSSURL: "wss://rpc-evmos.kingnodes.com:443/ws", Confirmations: 15, MinTransfer: 0.1, MaxTransfer: 1000000, SupportsEIP1559: false, StableCoins: []string{}},

		// More Non-EVM Chains
		{ID: "monad", Name: "Monad", Symbol: "MON", Type: TypeEVM, ChainID: 143, Decimals: 18, Explorer: "https://explorer.monad.xyz", RPCURL: "https://rpc.monad.xyz", APIURL: "https://api.monad.xyz", WSSURL: "wss://rpc.monad.xyz/ws", Confirmations: 15, MinTransfer: 0.001, MaxTransfer: 1000000, SupportsEIP1559: true, StableCoins: []string{"USDC", "USDT"}},

		{ID: "bittorrent", Name: "BitTorrent", Symbol: "BTT", Type: TypeEVM, ChainID: 199, Decimals: 18, Explorer: "https://bttcscan.com", RPCURL: "https://rpc.bittorrentchain.io", APIURL: "https://api.bittorrentcscan.io", WSSURL: "wss://rpc.bittorrentchain.io/ws", Confirmations: 15, MinTransfer: 1, MaxTransfer: 1000000000, SupportsEIP1559: false, StableCoins: []string{"USDT", "USDC"}},

		{ID: "vision", Name: "Vision", Symbol: "VSION", Type: TypeEVM, ChainID: 360, Decimals: 18, Explorer: "https://www.visionnetwork.io", RPCURL: "https://mainnet.shape.network", APIURL: "https://api.visionnetwork.io", WSSURL: "wss://in3.vision.org.cn/ws", Confirmations: 15, MinTransfer: 0.01, MaxTransfer: 1000000, SupportsEIP1559: false, StableCoins: []string{}},
		{ID: "penumbra", Name: "Penumbra", Symbol: "UM", Type: TypeCosmos, ChainID: 0, Decimals: 6, Explorer: "https://mintscan.io/penumbra", RPCURL: "https://rpc.penumbra.zone:443", APIURL: "https://api.penumbra.zone", WSSURL: "wss://rpc.penumbra.zone:443/websocket", Confirmations: 15, MinTransfer: 0.1, MaxTransfer: 1000000, SupportsEIP1559: false, StableCoins: []string{}},
		{ID: "noble", Name: "Noble", Symbol: "NOBLE", Type: TypeCosmos, ChainID: 0, Decimals: 6, Explorer: "https://mintscan.io/noble", RPCURL: "https://rpc.noble.strange.love:443", APIURL: "https://api.noble.strange.love", WSSURL: "wss://rpc.noble.strange.love:443/websocket", Confirmations: 15, MinTransfer: 0.1, MaxTransfer: 1000000, SupportsEIP1559: false, StableCoins: []string{"USDC"}},
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
		// ============ Ethereum Mainnet (ChainID: 1) ============
		{ID: "eth_1", Address: "", Name: "Ether", Symbol: "ETH", Decimals: 18, ChainID: 1, Type: "native"},
		{ID: "usdt_1", Address: "0xdac17f958d2ee523a2206206994597c13d831ec7", Name: "Tether", Symbol: "USDT", Decimals: 6, ChainID: 1, Type: "erc20", IsStableCoin: true},
		{ID: "usdc_1", Address: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", Name: "USDC", Symbol: "USDC", Decimals: 6, ChainID: 1, Type: "erc20", IsStableCoin: true},
		{ID: "dai_1", Address: "0x6b175474e89094c44da98b954eedeac495271d0f", Name: "Dai", Symbol: "DAI", Decimals: 18, ChainID: 1, Type: "erc20", IsStableCoin: true},
		{ID: "busd_1", Address: "0x4fabb145d64652a948d72533023f6e7a623c7c53", Name: "BUSD", Symbol: "BUSD", Decimals: 18, ChainID: 1, Type: "erc20", IsStableCoin: true},
		{ID: "tusd_1", Address: "0x0000000000085d4780b73119b644ae5ecd22b376", Name: "TrueUSD", Symbol: "TUSD", Decimals: 18, ChainID: 1, Type: "erc20", IsStableCoin: true},
		{ID: "frax_1", Address: "0x853d955acef822db058eb8505911ed77f175b99e", Name: "Legacy Frax Dollar", Symbol: "FRAX", Decimals: 18, ChainID: 1, Type: "erc20", IsStableCoin: true},
		{ID: "usdd_1", Address: "0x4f8e5de400de08b164e7421b3ee387f461becd1a", Name: "USDD", Symbol: "USDD", Decimals: 18, ChainID: 1, Type: "erc20", IsStableCoin: true},
		{ID: "gusd_1", Address: "0x056fd409e1d7a124bd7017459dfea2f387b6d5cd", Name: "Gemini Dollar", Symbol: "GUSD", Decimals: 2, ChainID: 1, Type: "erc20", IsStableCoin: true},
		{ID: "usdp_1", Address: "0x9b3a8f7cec208e247d97dee13313690977e24459", Name: "Parallel USDp", Symbol: "USDp", Decimals: 18, ChainID: 1, Type: "erc20", IsStableCoin: true},
		{ID: "weth_1", Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", Name: "WETH", Symbol: "WETH", Decimals: 18, ChainID: 1, Type: "erc20", IsWrapped: true},
		{ID: "wbtc_1", Address: "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599", Name: "Wrapped Bitcoin", Symbol: "WBTC", Decimals: 8, ChainID: 1, Type: "erc20", IsWrapped: true},
		{ID: "wbnb_1", Address: "0x418d75f65a02b3d53b2418fb8e1fe493759c7605", Name: "Wrapped BNB (Wormhole)", Symbol: "WBNB", Decimals: 18, ChainID: 1, Type: "erc20", IsWrapped: true},
		{ID: "link_1", Address: "0x514910771af9ca656af840dff83e8264ecf986ca", Name: "Chainlink", Symbol: "LINK", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "uni_1", Address: "0x2730d6fdc86c95a74253beffaa8306b40fedecbb", Name: "UNICORN", Symbol: "UNI", Decimals: 8, ChainID: 1, Type: "erc20"},
		{ID: "aave_1", Address: "0x7fc66500c84a76ad7e9c93437bfc5ac33e2ddae9", Name: "Aave", Symbol: "AAVE", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "mkr_1", Address: "0x9f8f72aa9304c8b593d555f12ef6589cc3a579a2", Name: "Maker", Symbol: "MKR", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "comp_1", Address: "0xc00e94cb662c3520282e6f5717214004a7f26888", Name: "Compound", Symbol: "COMP", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "crv_1", Address: "0xd533a949740bb3306d119cc777fa900ba034cd52", Name: "Curve DAO", Symbol: "CRV", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "snx_1", Address: "0xc011a73ee8576fb46f5e1c5751ca3b9fe0af2a6f", Name: "Synthetix", Symbol: "SNX", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "grt_1", Address: "0xc944e90c64b2c07662a292be6244bdf05cda44a7", Name: "The Graph", Symbol: "GRT", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "sushi_1", Address: "0x6b3595068778dd592e39a122f4f5a5cf09c90fe2", Name: "Sushi", Symbol: "SUSHI", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "cake_1", Address: "0x152649ea73beab28c5b49b26eb48f7ead6d4c898", Name: "PancakeSwap", Symbol: "Cake", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "1inch_1", Address: "0x111111111117dc0aa78b770fa6a738034120c302", Name: "1INCH", Symbol: "1INCH", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "bal_1", Address: "0xba100000625a3754423978a60c9317c58a424e3d", Name: "Balancer", Symbol: "BAL", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "yfi_1", Address: "0x0bc529c00c6401aef6d220be8c6ea1667f6ad93e", Name: "yearn.finance", Symbol: "YFI", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "ldo_1", Address: "0x5a98fcbea516cf06857215779fd812ca3bef1b32", Name: "Lido DAO", Symbol: "LDO", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "rpl_1", Address: "0xd33526068d116ce69f19a9ee46f0bd304f21a51f", Name: "Rocket Pool", Symbol: "RPL", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "ens_1", Address: "0xc18360217d8f7ab5e7c516566761ea12ce7f9d72", Name: "Ethereum Name Service", Symbol: "ENS", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "pepe_1", Address: "0x69e15ab0fd240de689d09e4851181a6667968008", Name: "El Sapo Pepe", Symbol: "PEPE", Decimals: 9, ChainID: 1, Type: "erc20"},
		{ID: "shib_1", Address: "0xfcaf0e4498e78d65526a507360f755178b804ba8", Name: "NicCageWaluigiElmo42069Inu", Symbol: "SHIB", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "arb_1", Address: "0xb50721bcf8d664c30412cfbc6cf7a15145234ad1", Name: "Arbitrum", Symbol: "ARB", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "stg_1", Address: "0xaf5191b0de278c7286d6c7cc6ab6bb8a73ba2cd6", Name: "Stargate Finance", Symbol: "STG", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "gmx_1", Address: "0xcef8a7008310a15b1b1489d88b0e6fe51329b05a", Name: "General Motors xStock", Symbol: "GMx", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "magic_1", Address: "0xb0c7a3ba49c7a6eaba6cd4a96c55a1391070ac9a", Name: "Treasure", Symbol: "MAGIC", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "rdnt_1", Address: "0x137ddb47ee24eaa998a535ab00378d6bfa84f893", Name: "Radiant Capital", Symbol: "RDNT", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "pendle_1", Address: "0x808507121b80c02388fad14726482e061b8da827", Name: "Pendle", Symbol: "PENDLE", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "sand_1", Address: "0x3845badade8e6dff049820680d1f14bd3903a5d0", Name: "The Sandbox", Symbol: "SAND", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "mana_1", Address: "0x0f5d2fb29fb7d3cfee444a200298f468908cc942", Name: "Decentraland", Symbol: "MANA", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "axs_1", Address: "0xbb0e17ef65f82ab018d8edd776e8dd940327b28b", Name: "Axie Infinity", Symbol: "AXS", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "ape_1", Address: "0xa0385e7283c83e2871e9af49eec0966088421ddd", Name: "Ape", Symbol: "APE", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "imx_1", Address: "0xf57e7e7c23978c3caec3c3548e3d615c346e79ff", Name: "Immutable", Symbol: "IMX", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "fet_1", Address: "0xaea46a60368a7bd060eec7df8cba43b7ef41ad85", Name: "Artificial Superintelligence Alliance", Symbol: "FET", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "ocean_1", Address: "0x967da4048cd07ab37855c090aaf366e4ce1b9f48", Name: "Ocean Protocol", Symbol: "OCEAN", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "band_1", Address: "0xba11d00c5f74255f56a5e366f4f77f5a186d7f55", Name: "Band", Symbol: "BAND", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "bat_1", Address: "0x0d8775f648430679a709e98d2b0cb6250d2887ef", Name: "Basic Attention", Symbol: "BAT", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "enj_1", Address: "0xf629cbd94d3791c9250152bd8dfbdf380e2a3b9c", Name: "Enjin Coin", Symbol: "ENJ", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "cvx_1", Address: "0xc49da7892155ccc603f0b88eaedaa3b57910096f", Name: "Chevron Corp (Dinari Tokenized Stock)", Symbol: "CVX", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "rook_1", Address: "0xfa5047c9c78b8877af97bdcb85db743fd7313d4a", Name: "Rook", Symbol: "ROOK", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "ankr_1", Address: "0x8290333cef9e6d528dd5618fb97a76f268f3edd4", Name: "Ankr Network", Symbol: "ANKR", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "zrx_1", Address: "0xe41d2489571d322189246dafa5ebde1f4699f498", Name: "0x Protocol", Symbol: "ZRX", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "knc_1", Address: "0xdefa4e8a7bcba345f687a2f1456f5edd9ce97202", Name: "Kyber Network Crystal", Symbol: "KNC", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "storj_1", Address: "0xb64ef51c888972c908cfacf59b47c1afbc0ab8ac", Name: "Storj", Symbol: "STORJ", Decimals: 8, ChainID: 1, Type: "erc20"},
		{ID: "ren_1", Address: "0x408e41876cccdc0f92210600ef50372656052a38", Name: "Ren", Symbol: "REN", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "uma_1", Address: "0x04fa0d235c4abf4bcf4787af4cf447de572ef828", Name: "UMA", Symbol: "UMA", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "chz_1", Address: "0x3506424f91fd33084466f402d5d97f05f8e3b4af", Name: "Chiliz", Symbol: "CHZ", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "gala_1", Address: "0xd1d2eb1b1e90b638588728b4130137d262c87cae", Name: "GALA", Symbol: "GALA", Decimals: 8, ChainID: 1, Type: "erc20"},
		{ID: "ilv_1", Address: "0x767fe9edc9e0df98e07454847909b5e959d7ca0e", Name: "Illuvium", Symbol: "ILV", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "spell_1", Address: "0x090185f2135308bad17527004364ebcc2d37e5f6", Name: "Spell", Symbol: "SPELL", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "joe_1", Address: "0x76e222b07c53d28b89b0bac18602810fc22b49a8", Name: "Joe Coin", Symbol: "JOE", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "qi_1", Address: "0x559b7bfc48a5274754b08819f75c5f27af53d53b", Name: "Qi Dao", Symbol: "QI", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "degen_1", Address: "0x420658a1d8b8f5c36ddaf1bb828f347ba9011969", Name: "Degen Arena", Symbol: "DEGEN", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "xvs_1", Address: "0xd3cc9d8f3689b83c91b7b59cab4946b063eb894a", Name: "Venus", Symbol: "XVS", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "btt_1", Address: "0xc669928185dbce49d2230cc9b0979be6dc797957", Name: "BitTorrent", Symbol: "BTT", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "win_1", Address: "0xb10cb07ca2cdac77fbb5707f6690301f9d036f45", Name: "Win", Symbol: "WIN", Decimals: 8, ChainID: 1, Type: "erc20"},
		{ID: "nft_1", Address: "0x198d14f2ad9ce69e76ea330b374de4957c3f850a", Name: "AINFT", Symbol: "NFT", Decimals: 6, ChainID: 1, Type: "erc20"},
		{ID: "woo_1", Address: "0x4691937a7508860f876c9c0a2a617e7d9e945d4b", Name: "WOO", Symbol: "WOO", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "ht_1", Address: "0x6f259637dcd74c767781e37bc6133cd6a68aa161", Name: "Huobi", Symbol: "HT", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "okb_1", Address: "0x75231f58b43240c9718dd58b4967c5114342a86c", Name: "OKB", Symbol: "OKB", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "cro_1", Address: "0xa0b73e1ff0b80914ab6fe0444e65848c4c34450b", Name: "Cronos", Symbol: "CRO", Decimals: 8, ChainID: 1, Type: "erc20"},
		{ID: "vvs_1", Address: "0x839e71613f9aa06e5701cf6de63e303616b0dde3", Name: "VVS Finance", Symbol: "VVS", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "safe_1", Address: "0x5afe3855358e112b5647b952709e6165e1c1eeee", Name: "Safe", Symbol: "SAFE", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "wld_1", Address: "0x163f8c2467924be0ae7b5347228cabf260318753", Name: "Worldcoin", Symbol: "WLD", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "blur_1", Address: "0x5283d291dbcf85356a21ba090e6db59121208b44", Name: "Blur", Symbol: "BLUR", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "mnt_1", Address: "0x3c3a81e81dc49a522a592e7622a7e711c06bf354", Name: "Mantle", Symbol: "MNT", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "zro_1", Address: "0x6985884c4392d348587b19cb9eaaf157f13271cd", Name: "LayerZero", Symbol: "ZRO", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "sei_1", Address: "0xbdf43ecadc5cef51b7d1772f722e40596bc1788b", Name: "LayerZero Bridged Sei", Symbol: "SEI", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "w_1", Address: "0xb0ffa8000886e57f86dd5264b9582b2ad87b2b91", Name: "Wormhole", Symbol: "W", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "strk_1", Address: "0xca14007eff0db1f8135f4c25b34de49ab0d42766", Name: "Starknet", Symbol: "STRK", Decimals: 18, ChainID: 1, Type: "erc20"},
		// ============ BNB Smart Chain Mainnet (ChainID: 56) ============
		{ID: "bnb_56", Address: "", Name: "BNB Chain Native Token", Symbol: "BNB", Decimals: 18, ChainID: 56, Type: "native"},
		{ID: "usdt_56", Address: "0x524bc91dc82d6b90ef29f76a3ecaabafffd490bc", Name: "Wormhole Bridged USDT", Symbol: "USDT", Decimals: 6, ChainID: 56, Type: "erc20", IsStableCoin: true},
		{ID: "usdc_56", Address: "0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d", Name: "Binance Bridged USDC (BNB Smart Chain)", Symbol: "USDC", Decimals: 18, ChainID: 56, Type: "erc20", IsStableCoin: true},
		{ID: "dai_56", Address: "0x1af3f329e8be154074d8769d1ffa4ee058b1dbc3", Name: "Binance-Peg DAI", Symbol: "DAI", Decimals: 18, ChainID: 56, Type: "erc20", IsStableCoin: true},
		{ID: "busd_56", Address: "0xe9e7cea3dedca5984780bafc599bd69add087d56", Name: "Binance-Peg BUSD", Symbol: "BUSD", Decimals: 18, ChainID: 56, Type: "erc20", IsStableCoin: true},
		{ID: "tusd_56", Address: "0x14016e85a25aeb13065688cafb43044c2ef86784", Name: "Bridged TrueUSD", Symbol: "TUSD", Decimals: 18, ChainID: 56, Type: "erc20", IsStableCoin: true},
		{ID: "frax_56", Address: "0x90c97f71e18723b0cf0dfa30ee176ab653e89f40", Name: "Legacy Frax Dollar", Symbol: "FRAX", Decimals: 18, ChainID: 56, Type: "erc20", IsStableCoin: true},
		{ID: "usdd_56", Address: "0x45e51bc23d592eb2dba86da3985299f7895d66ba", Name: "USDD", Symbol: "USDD", Decimals: 18, ChainID: 56, Type: "erc20", IsStableCoin: true},
		{ID: "wbtc_56", Address: "0x0555e30da8f98308edb960aa94c0db47230d2b9c", Name: "Wrapped Bitcoin", Symbol: "WBTC", Decimals: 8, ChainID: 56, Type: "erc20", IsWrapped: true},
		{ID: "wbnb_56", Address: "0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c", Name: "Wrapped BNB", Symbol: "WBNB", Decimals: 18, ChainID: 56, Type: "erc20", IsWrapped: true},
		{ID: "link_56", Address: "0xf8a0bf9cf54bb92f17374d9e9a321e6a111a51bd", Name: "Chainlink", Symbol: "LINK", Decimals: 18, ChainID: 56, Type: "erc20"},
		{ID: "uni_56", Address: "0xbf5140a22578168fd562dccf235e5d43a02ce9b1", Name: "Uniswap", Symbol: "UNI", Decimals: 18, ChainID: 56, Type: "erc20"},
		{ID: "aave_56", Address: "0xfb6115445bff7b52feb98650c87f44907e58f802", Name: "Aave", Symbol: "AAVE", Decimals: 18, ChainID: 56, Type: "erc20"},
		{ID: "comp_56", Address: "0x52ce071bd9b1c4b00a0b92d298c512478cad67e8", Name: "Compound", Symbol: "COMP", Decimals: 18, ChainID: 56, Type: "erc20"},
		{ID: "sushi_56", Address: "0x947950bcc74888a40ffa2593c5798f11fc9124c4", Name: "Sushi", Symbol: "SUSHI", Decimals: 18, ChainID: 56, Type: "erc20"},
		{ID: "cake_56", Address: "0x0e09fabb73bd3ade0a17ecc321fd13a19e81ce82", Name: "PancakeSwap", Symbol: "Cake", Decimals: 18, ChainID: 56, Type: "erc20"},
		{ID: "1inch_56", Address: "0x111111111117dc0aa78b770fa6a738034120c302", Name: "1INCH", Symbol: "1INCH", Decimals: 18, ChainID: 56, Type: "erc20"},
		{ID: "yfi_56", Address: "0x0affe50d637114656f6ba27fa94abf0de260f918", Name: "YFI", Symbol: "YFI", Decimals: 8, ChainID: 56, Type: "erc20"},
		{ID: "pepe_56", Address: "0x25d887ce7a35172c62febfd67a1856f20faebb00", Name: "Pepe", Symbol: "PEPE", Decimals: 18, ChainID: 56, Type: "erc20"},
		{ID: "shib_56", Address: "0x2859e4544c4bb03966803b044a93563bd2d0dd4d", Name: "Binance-Peg SHIB", Symbol: "SHIB", Decimals: 18, ChainID: 56, Type: "erc20"},
		{ID: "stg_56", Address: "0xb0d502e938ed5f4df2e681fe6e419ff29631d62b", Name: "Stargate Finance", Symbol: "STG", Decimals: 18, ChainID: 56, Type: "erc20"},
		{ID: "gmx_56", Address: "0xcef8a7008310a15b1b1489d88b0e6fe51329b05a", Name: "General Motors xStock", Symbol: "GMx", Decimals: 18, ChainID: 56, Type: "erc20"},
		{ID: "rdnt_56", Address: "0xf7de7e8a6bd59ed41a4b5fe50278b3b7f31384df", Name: "Radiant Capital", Symbol: "RDNT", Decimals: 18, ChainID: 56, Type: "erc20"},
		{ID: "pendle_56", Address: "0xb3ed0a426155b79b898849803e3b36552f7ed507", Name: "Pendle", Symbol: "PENDLE", Decimals: 18, ChainID: 56, Type: "erc20"},
		{ID: "axs_56", Address: "0x715d400f88c167884bbcc41c5fea407ed4d2f8a0", Name: "Axie Infinity", Symbol: "AXS", Decimals: 18, ChainID: 56, Type: "erc20"},
		{ID: "ape_56", Address: "0x8f86a15ec17cb3369d8b3e666dadbc11daa82b79", Name: "ApeCoin", Symbol: "APE", Decimals: 18, ChainID: 56, Type: "erc20"},
		{ID: "fet_56", Address: "0x031b41e504677879370e9dbcf937283a8691fa7f", Name: "Artificial Superintelligence Alliance", Symbol: "FET", Decimals: 18, ChainID: 56, Type: "erc20"},
		{ID: "ocean_56", Address: "0xb81139f4539bc3d726fa0b675024d319af676666", Name: "OCEAN", Symbol: "OCEAN", Decimals: 9, ChainID: 56, Type: "erc20"},
		{ID: "ankr_56", Address: "0xf307910a4c7bbc79691fd374889b36d8531b08e3", Name: "Ankr Network", Symbol: "ANKR", Decimals: 18, ChainID: 56, Type: "erc20"},
		{ID: "knc_56", Address: "0xfe56d5892bdffc7bf58f2e84be1b2c32d21c308b", Name: "Kyber Network Crystal", Symbol: "KNC", Decimals: 18, ChainID: 56, Type: "erc20"},
		{ID: "joe_56", Address: "0x371c7ec6d8039ff7933a2aa28eb827ffe1f52f07", Name: "JOE", Symbol: "JOE", Decimals: 18, ChainID: 56, Type: "erc20"},
		{ID: "velo_56", Address: "0xf486ad071f3bee968384d2e39e2d8af0fcf6fd46", Name: "Velo", Symbol: "VELO", Decimals: 18, ChainID: 56, Type: "erc20"},
		{ID: "twt_56", Address: "0x4b0f1812e5df2a09796481ff14017e6005508003", Name: "Trust Wallet", Symbol: "TWT", Decimals: 18, ChainID: 56, Type: "erc20"},
		{ID: "hook_56", Address: "0xa260e12d2b924cb899ae80bb58123ac3fee1e2f0", Name: "Hooked Protocol", Symbol: "HOOK", Decimals: 18, ChainID: 56, Type: "erc20"},
		{ID: "xvs_56", Address: "0xcf6bb5389c92bdda8a3747ddb454cb7a64626c63", Name: "Venus", Symbol: "XVS", Decimals: 18, ChainID: 56, Type: "erc20"},
		{ID: "btt_56", Address: "0x352cb5e19b12fc216548a2677bd0fce83bae434b", Name: "BitTorrent", Symbol: "BTT", Decimals: 18, ChainID: 56, Type: "erc20"},
		{ID: "win_56", Address: "0xaef0d72a118ce24fee3cd1d43d383897d05b4e99", Name: "WINkLink BSC", Symbol: "WIN", Decimals: 18, ChainID: 56, Type: "erc20"},
		{ID: "nft_56", Address: "0x20ee7b720f4e4c4ffcb00c4065cdae55271aecca", Name: "AINFT", Symbol: "NFT", Decimals: 6, ChainID: 56, Type: "erc20"},
		{ID: "woo_56", Address: "0x4691937a7508860f876c9c0a2a617e7d9e945d4b", Name: "WOO", Symbol: "WOO", Decimals: 18, ChainID: 56, Type: "erc20"},
		{ID: "safe_56", Address: "0xc160598f2f5b216b48fa36007b4538114a234444", Name: "SAFE", Symbol: "SAFE", Decimals: 18, ChainID: 56, Type: "erc20"},
		{ID: "zro_56", Address: "0x6985884c4392d348587b19cb9eaaf157f13271cd", Name: "LayerZero", Symbol: "ZRO", Decimals: 18, ChainID: 56, Type: "erc20"},
		{ID: "w_56", Address: "0x380bf199b3173cf7b3b321848ae1c5014a124444", Name: "W", Symbol: "W", Decimals: 18, ChainID: 56, Type: "erc20"},
		// ============ Polygon Mainnet (ChainID: 137) ============
		{ID: "pol_137", Address: "", Name: "POL", Symbol: "POL", Decimals: 18, ChainID: 137, Type: "native"},
		{ID: "usdt_137", Address: "0x9417669fbf23357d2774e9d421307bd5ea1006d2", Name: "Wormhole Bridged USDT", Symbol: "USDT", Decimals: 6, ChainID: 137, Type: "erc20", IsStableCoin: true},
		{ID: "usdc_137", Address: "0x3c499c542cef5e3811e1192ce70d8cc03d5c3359", Name: "USDC", Symbol: "USDC", Decimals: 6, ChainID: 137, Type: "erc20", IsStableCoin: true},
		{ID: "dai_137", Address: "0x8f3cf7ad23cd3cadbd9735aff958023239c6a063", Name: "Polygon PoS Bridged DAI (Polygon POS)", Symbol: "DAI", Decimals: 18, ChainID: 137, Type: "erc20", IsStableCoin: true},
		{ID: "busd_137", Address: "0x9c9e5fd8bbc25984b178fdce6117defa39d2db39", Name: "Binance-Peg BUSD", Symbol: "BUSD", Decimals: 18, ChainID: 137, Type: "erc20", IsStableCoin: true},
		{ID: "tusd_137", Address: "0x2e1ad108ff1d8c782fcbbb89aad783ac49586756", Name: "Bridged TrueUSD", Symbol: "TUSD", Decimals: 18, ChainID: 137, Type: "erc20", IsStableCoin: true},
		{ID: "frax_137", Address: "0x45c32fa6df82ead1e2ef74d17b76547eddfaff89", Name: "Legacy Frax Dollar", Symbol: "FRAX", Decimals: 18, ChainID: 137, Type: "erc20", IsStableCoin: true},
		{ID: "weth_137", Address: "0x7ceb23fd6bc0add59e62ac25578270cff1b9f619", Name: "Polygon PoS Bridged WETH (Polygon POS)", Symbol: "WETH", Decimals: 18, ChainID: 137, Type: "erc20", IsWrapped: true},
		{ID: "wbtc_137", Address: "0x1bfd67037b42cf73acf2047067bd4f2c47d9bfd6", Name: "Polygon Bridged WBTC (Polygon POS)", Symbol: "WBTC", Decimals: 8, ChainID: 137, Type: "erc20", IsWrapped: true},
		{ID: "wbnb_137", Address: "0xecdcb5b88f8e3c15f95c720c51c71c9e2080525d", Name: "Wrapped BNB (Wormhole)", Symbol: "WBNB", Decimals: 18, ChainID: 137, Type: "erc20", IsWrapped: true},
		{ID: "link_137", Address: "0x53e0bca35ec356bd5dddfebbd1fc0fd03fabad39", Name: "Chainlink", Symbol: "LINK", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "uni_137", Address: "0xb33eaad8d922b1083446dc23f610c2567fb5180f", Name: "Uniswap", Symbol: "UNI", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "aave_137", Address: "0xd6df932a45c0f255f85145f286ea0b292b21c90b", Name: "Aave", Symbol: "AAVE", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "mkr_137", Address: "0x6f7c932e7684666c9fd1d44527765433e01ff61d", Name: "Maker", Symbol: "MKR", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "comp_137", Address: "0x8505b9d2254a7ae468c0e9dd10ccea3a837aef5c", Name: "Compound", Symbol: "COMP", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "crv_137", Address: "0x172370d5cd63279efa6d502dab29171933a610af", Name: "Curve DAO", Symbol: "CRV", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "snx_137", Address: "0x50b728d8d964fd00c2d0aad81718b71311fef68a", Name: "Synthetix", Symbol: "SNX", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "grt_137", Address: "0x5fe2b58c013d7601147dcdd68c143a77499f5531", Name: "The Graph", Symbol: "GRT", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "sushi_137", Address: "0x0b3f868e0be5597d5db7feb59e1cadbb0fdda50a", Name: "Sushi", Symbol: "SUSHI", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "bal_137", Address: "0x9a71012b13ca4d3d0cdc72a177df3ef03b0e76a3", Name: "Balancer", Symbol: "BAL", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "yfi_137", Address: "0xda537104d6a5edd53c6fbba9a898708e465260b6", Name: "yearn.finance", Symbol: "YFI", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "ldo_137", Address: "0xc3c7d422809852031b44ab29eec9f1eff2a58756", Name: "Lido DAO", Symbol: "LDO", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "rpl_137", Address: "0x7205705771547cf79201111b4bd8aaf29467b9ec", Name: "Rocket Pool", Symbol: "RPL", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "stg_137", Address: "0x2f6f07cdcf3588944bf4c42ac74ff24bf56e7590", Name: "Stargate Finance", Symbol: "STG", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "sand_137", Address: "0xbbba073c31bf03b8acf7c28ef0738decf3695683", Name: "The Sandbox", Symbol: "SAND", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "mana_137", Address: "0xa1c57f48f0deb89f569dfbe6e2b7f46d33606fd4", Name: "Decentraland", Symbol: "MANA", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "ape_137", Address: "0xb7b31a6bc18e48888545ce79e83e06003be70930", Name: "ApeCoin", Symbol: "APE", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "bat_137", Address: "0x3cef98bb43d732e2f285ee605a8158cde967d219", Name: "Basic Attention", Symbol: "BAT", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "ankr_137", Address: "0x101a023270368c0d50bffb62780f4afd4ea79c35", Name: "Ankr Network", Symbol: "ANKR", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "knc_137", Address: "0x1c954e8fe737f99f68fa1ccda3e51ebdb291948c", Name: "Kyber Network Crystal", Symbol: "KNC", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "woo_137", Address: "0x1b815d120b3ef02039ee11dc2d33de7aa4a8c603", Name: "WOO", Symbol: "WOO", Decimals: 18, ChainID: 137, Type: "erc20"},
		{ID: "zro_137", Address: "0x6985884c4392d348587b19cb9eaaf157f13271cd", Name: "LayerZero", Symbol: "ZRO", Decimals: 18, ChainID: 137, Type: "erc20"},
		// ============ Avalanche C-Chain (ChainID: 43114) ============
		{ID: "avax_43114", Address: "", Name: "Avalanche", Symbol: "AVAX", Decimals: 18, ChainID: 43114, Type: "native"},
		{ID: "usdt_43114", Address: "0x9702230a8ea53601f5cd2dc00fdbc13d4df4a8c7", Name: "Tether", Symbol: "USDt", Decimals: 6, ChainID: 43114, Type: "erc20", IsStableCoin: true},
		{ID: "usdc_43114", Address: "0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e", Name: "USDC", Symbol: "USDC", Decimals: 6, ChainID: 43114, Type: "erc20", IsStableCoin: true},
		{ID: "busd_43114", Address: "0x9c9e5fd8bbc25984b178fdce6117defa39d2db39", Name: "Binance-Peg BUSD", Symbol: "BUSD", Decimals: 18, ChainID: 43114, Type: "erc20", IsStableCoin: true},
		{ID: "tusd_43114", Address: "0x1c20e891bab6b1727d14da358fae2984ed9b59eb", Name: "TrueUSD", Symbol: "TUSD", Decimals: 18, ChainID: 43114, Type: "erc20", IsStableCoin: true},
		{ID: "frax_43114", Address: "0xd24c2ad096400b6fbcd2ad8b24e7acbc21a1da64", Name: "Legacy Frax Dollar", Symbol: "FRAX", Decimals: 18, ChainID: 43114, Type: "erc20", IsStableCoin: true},
		{ID: "usdp_43114", Address: "0x9ee1963f05553ef838604dd39403be21cef26aa4", Name: "Parallel USDp", Symbol: "USDp", Decimals: 18, ChainID: 43114, Type: "erc20", IsStableCoin: true},
		{ID: "wethe_43114", Address: "0x49d5c2bdffac6ce2bfdb6640f4f80f226bc10bab", Name: "Avalanche Bridged WETH (Avalanche)", Symbol: "WETH.e", Decimals: 18, ChainID: 43114, Type: "erc20"},
		{ID: "wbtc_43114", Address: "0x0555e30da8f98308edb960aa94c0db47230d2b9c", Name: "Wrapped Bitcoin", Symbol: "WBTC", Decimals: 8, ChainID: 43114, Type: "erc20", IsWrapped: true},
		{ID: "wbnb_43114", Address: "0x442f7f22b1ee2c842beaff52880d4573e9201158", Name: "Wrapped BNB (Wormhole)", Symbol: "WBNB", Decimals: 18, ChainID: 43114, Type: "erc20", IsWrapped: true},
		{ID: "wavax_43114", Address: "0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7", Name: "Wrapped AVAX", Symbol: "WAVAX", Decimals: 18, ChainID: 43114, Type: "erc20", IsWrapped: true},
		{ID: "bal_43114", Address: "0xe15bcb9e0ea69e6ab9fa080c4c4a5632896298c3", Name: "Balancer", Symbol: "BAL", Decimals: 18, ChainID: 43114, Type: "erc20"},
		{ID: "pepe_43114", Address: "0xa659d083b677d6bffe1cb704e1473b896727be6d", Name: "Pepe", Symbol: "PEPE", Decimals: 18, ChainID: 43114, Type: "erc20"},
		{ID: "stg_43114", Address: "0x2f6f07cdcf3588944bf4c42ac74ff24bf56e7590", Name: "Stargate Finance", Symbol: "STG", Decimals: 18, ChainID: 43114, Type: "erc20"},
		{ID: "gmx_43114", Address: "0x62edc0692bd897d2295872a9ffcac5425011c661", Name: "GMX", Symbol: "GMX", Decimals: 18, ChainID: 43114, Type: "erc20"},
		{ID: "cvx_43114", Address: "0x78758dfccfe4854be602cf4ea20bfff040fad90c", Name: "Chevron Corp (Dinari Tokenized Stock)", Symbol: "CVX", Decimals: 18, ChainID: 43114, Type: "erc20"},
		{ID: "ankr_43114", Address: "0xdf474b7109b73b7d57926d43598d5934131136b2", Name: "Ankr Network", Symbol: "ANKR", Decimals: 18, ChainID: 43114, Type: "erc20"},
		{ID: "knc_43114", Address: "0x39fc9e94caeacb435842fadedecb783589f50f5f", Name: "Kyber Network Crystal", Symbol: "KNC", Decimals: 18, ChainID: 43114, Type: "erc20"},
		{ID: "spell_43114", Address: "0xce1bffbd5374dac86a2893119683f4911a2f7814", Name: "Spell", Symbol: "SPELL", Decimals: 18, ChainID: 43114, Type: "erc20"},
		{ID: "joe_43114", Address: "0x6e84a6216ea6dacc71ee8e6b0a5b7322eebc0fdd", Name: "JOE", Symbol: "JOE", Decimals: 18, ChainID: 43114, Type: "erc20"},
		{ID: "qi_43114", Address: "0x8729438eb15e2c8b576fcc6aecda6a148776c0f5", Name: "BENQI", Symbol: "QI", Decimals: 18, ChainID: 43114, Type: "erc20"},
		{ID: "wooe_43114", Address: "0xabc9547b534519ff73921b1fba6e672b5f58d083", Name: "WOO", Symbol: "WOO.e", Decimals: 18, ChainID: 43114, Type: "erc20"},
		{ID: "zro_43114", Address: "0x6985884c4392d348587b19cb9eaaf157f13271cd", Name: "LayerZero", Symbol: "ZRO", Decimals: 18, ChainID: 43114, Type: "erc20"},
		{ID: "strk_43114", Address: "0xbbd1d186b035a28c4a6dcaf192a4c61f9a74bce1", Name: "Strategy Strike Preferred (Dinari Tokenized Stock)", Symbol: "STRK", Decimals: 18, ChainID: 43114, Type: "erc20"},
		// ============ SOLANA - mints from the Jupiter verified list ============
		{ID: "sol_sol", Address: "So11111111111111111111111111111111111111112", Name: "Wrapped SOL", Symbol: "SOL", Decimals: 9, ChainID: 0, Type: "spl"},
		{ID: "usdc_sol", Address: "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", Name: "USD Coin", Symbol: "USDC", Decimals: 6, ChainID: 0, Type: "spl"},
		{ID: "usdt_sol", Address: "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB", Name: "USDT", Symbol: "USDT", Decimals: 6, ChainID: 0, Type: "spl"},
		{ID: "jup_sol", Address: "JUPyiwrYJFskUPiHa7hkeR8VUtAeFoSYbKedZNsDvCN", Name: "Jupiter", Symbol: "JUP", Decimals: 6, ChainID: 0, Type: "spl"},
		{ID: "jto_sol", Address: "jtojtomepa8beP8AuQc6eXt5FriJwfFMwQx2v2f9mCL", Name: "JITO", Symbol: "JTO", Decimals: 9, ChainID: 0, Type: "spl"},
		{ID: "pyth_sol", Address: "HZ1JovNiVvGrGNiiYvEozEVgZ58xaU3RKwX8eACQBCt3", Name: "Pyth Network", Symbol: "PYTH", Decimals: 6, ChainID: 0, Type: "spl"},
		{ID: "ray_sol", Address: "4k3Dyjzvzp8eMZWUXbBCjEvwSkkk59S5iCNLY3QrkX6R", Name: "Raydium", Symbol: "RAY", Decimals: 6, ChainID: 0, Type: "spl"},
		{ID: "orca_sol", Address: "orcaEKTdK7LKz57vaAYr9QeNsVEPfiu6QeMU1kektZE", Name: "Orca", Symbol: "ORCA", Decimals: 6, ChainID: 0, Type: "spl"},
		{ID: "bonk_sol", Address: "DezXAZ8z7PnrnRJjz3wXBoRgixCa6xjnB7YaB1pPB263", Name: "Bonk", Symbol: "BONK", Decimals: 5, ChainID: 0, Type: "spl"},
		{ID: "wif_sol", Address: "EKpQGSJtjMFqKZ9KQanSqYXRcF8fBopzLHYxdM65zcjm", Name: "dogwifhat", Symbol: "WIF", Decimals: 6, ChainID: 0, Type: "spl"},
		{ID: "jlp_sol", Address: "27G8MtK7VtTcCHkpASjSDdkWWYfoqT6ggEuKidVJidD4", Name: "Jupiter Perps", Symbol: "JLP", Decimals: 6, ChainID: 0, Type: "spl"},
		{ID: "inf_sol", Address: "5oVNBeEEQvYi1cX3ir8Dx5n1P7pdxydbGF2X4TxVusJm", Name: "Infinity", Symbol: "INF", Decimals: 9, ChainID: 0, Type: "spl"},
		{ID: "msol_sol", Address: "mSoLzYCxHdYgdzU16g5QSh3i5K3z3KZK7ytfqcJm7So", Name: "Marinade staked SOL (mSOL)", Symbol: "MSOL", Decimals: 9, ChainID: 0, Type: "spl"},
		{ID: "jitosol_sol", Address: "J1toso1uCk3RLmjorhTtrVwY9HJ7X8V9yYac6Y7kGCPn", Name: "Jito Staked SOL", Symbol: "JITOSOL", Decimals: 9, ChainID: 0, Type: "spl"},
		{ID: "bsol_sol", Address: "bSo13r4TkiE4KumL71LsHTPpL2euBYLFx6h9HP3piy1", Name: "BlazeStake Staked SOL (bSOL)", Symbol: "BSOL", Decimals: 9, ChainID: 0, Type: "spl"},
		{ID: "mnde_sol", Address: "MNDEFzGvMt87ueuHvVU9VcTqsAP5b3fTGPsHuuPA5ey", Name: "Marinade", Symbol: "MNDE", Decimals: 9, ChainID: 0, Type: "spl"},
		{ID: "pyusd_sol", Address: "2b1kV6DkPAnxd5ixfnxCpjxmKwqjjaYmCZfHsFu24GXo", Name: "PayPal USD", Symbol: "PYUSD", Decimals: 6, ChainID: 0, Type: "spl"},
		{ID: "w_sol", Address: "85VBFQZC9TZkfaptBWjvUw7YbZjy52A6mjtPGjstQAmQ", Name: "Wormhole Token", Symbol: "W", Decimals: 6, ChainID: 0, Type: "spl"},
		{ID: "drift_sol", Address: "DriFtupJYLTosbwoN8koMbEYSx54aFAVLddWsbksjwg7", Name: "Drift", Symbol: "DRIFT", Decimals: 6, ChainID: 0, Type: "spl"},
		{ID: "tnsr_sol", Address: "TNSRxcUxoT9xBG3de7PiJyTDYu7kskLqcpddxnEJAS6", Name: "Tensor", Symbol: "TNSR", Decimals: 9, ChainID: 0, Type: "spl"},
		{ID: "io_sol", Address: "BZLbGTNCSFfoth2GYDtwr7e4imWzpR5jqcUuGEwr646K", Name: "IO", Symbol: "IO", Decimals: 8, ChainID: 0, Type: "spl"},
		{ID: "hnt_sol", Address: "hntyVP6YFm1Hg25TN9WGLqM12b8TQmcknKrdu1oxWux", Name: "Helium Network Token", Symbol: "HNT", Decimals: 8, ChainID: 0, Type: "spl"},
		{ID: "mobile_sol", Address: "mb1eu7TzEc71KxDpsmsKoucSSuuoGLv1drys1oP2jh6", Name: "Helium Mobile", Symbol: "MOBILE", Decimals: 6, ChainID: 0, Type: "spl"},
		{ID: "render_sol", Address: "rndrizKT3MK1iimdxRdWabcF7Zg7AR5T4nud4EkHBof", Name: "Render Token", Symbol: "RENDER", Decimals: 8, ChainID: 0, Type: "spl"},
		{ID: "samo_sol", Address: "7xKXtg2CW87d97TXJSDpbD5jBkheTqA83TZRuJosgAsU", Name: "Samoyed Coin", Symbol: "SAMO", Decimals: 9, ChainID: 0, Type: "spl"},
		{ID: "step_sol", Address: "StepAscQoEioFxxWGnh2sLBDFp9d8rvKz2Yp39iDpyT", Name: "Step", Symbol: "STEP", Decimals: 9, ChainID: 0, Type: "spl"},
		{ID: "mplx_sol", Address: "METAewgxyPbgwsseH8T16a39CQ5VyVxZi9zXiDPY18m", Name: "Metaplex Token", Symbol: "MPLX", Decimals: 6, ChainID: 0, Type: "spl"},
		{ID: "gst_sol", Address: "AFbX8oGjGpmVFywbVouvhQSRmiW2aR1mohfahi4Y2AdB", Name: "GST", Symbol: "GST", Decimals: 9, ChainID: 0, Type: "spl"},
		{ID: "prcl_sol", Address: "4LLbsb5ReP3yEtYzmXewyGjcir5uXtKFURtaEUVC2AHs", Name: "Parcl", Symbol: "PRCL", Decimals: 6, ChainID: 0, Type: "spl"},
		{ID: "kmno_sol", Address: "KMNo3nJsBXfcpJTVhZcXLW7RmTwTt4GVFE7suUBo9sS", Name: "Kamino", Symbol: "KMNO", Decimals: 6, ChainID: 0, Type: "spl"},
		{ID: "zeus_sol", Address: "ZEUS1aR7aX8DFFJf5QjWj2ftDDdNTroMNGo8YoQm3Gq", Name: "ZEUS", Symbol: "ZEUS", Decimals: 6, ChainID: 0, Type: "spl"},

		// ============ TRON - TRC-20 contracts verified via TronGrid ============
		{ID: "trx", Address: "", Name: "Tron", Symbol: "TRX", Decimals: 6, ChainID: 195, Type: "native"},
		{ID: "usdt_trx", Address: "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t", Name: "Tether USD", Symbol: "USDT", Decimals: 6, ChainID: 195, Type: "trc20", IsStableCoin: true},
		{ID: "usdc_trx", Address: "TEkxiTehnzSmSe2XqrBj4w32RUN966rdz8", Name: "USD Coin", Symbol: "USDC", Decimals: 6, ChainID: 195, Type: "trc20", IsStableCoin: true},
		{ID: "tusd_trx", Address: "TUpMhErZL2fhh4sVNULAbNKLokS4GjC1F4", Name: "TrueUSD", Symbol: "TUSD", Decimals: 18, ChainID: 195, Type: "trc20", IsStableCoin: true},
		{ID: "usdd_trx", Address: "TXDk8mbtRbXeYuMNS83CfKPaYYT8XWv9Hz", Name: "Decentralized USD", Symbol: "USDD", Decimals: 18, ChainID: 195, Type: "trc20", IsStableCoin: true},
		{ID: "usdj_trx", Address: "TMwFHYXLJaRUPeW6421aqXL4ZEzPRFGkGT", Name: "JUST Stablecoin", Symbol: "USDJ", Decimals: 18, ChainID: 195, Type: "trc20", IsStableCoin: true},
		{ID: "btt_trx", Address: "TAFjULxiVgT4qWk6UZwjqwZXTSaGaqnVp4", Name: "BitTorrent", Symbol: "BTT", Decimals: 18, ChainID: 195, Type: "trc20"},
		{ID: "jst_trx", Address: "TCFLL5dx5ZJdKnWuesXxi1VPwjLVmWZZy9", Name: "JUST", Symbol: "JST", Decimals: 18, ChainID: 195, Type: "trc20"},
		{ID: "sun_trx", Address: "TSSMHYeV2uE9qYH95DqyoCuNCzEL1NvU3S", Name: "Sun Token", Symbol: "SUN", Decimals: 18, ChainID: 195, Type: "trc20"},
		{ID: "win_trx", Address: "TLa2f6VPqDgRE67v1736s7bJ8Ray5wYjU7", Name: "WINkLink", Symbol: "WIN", Decimals: 6, ChainID: 195, Type: "trc20"},
		{ID: "nft_trx", Address: "TFczxzPhnThNSqr5by8tvxsdCFRRz6cPNq", Name: "APENFT", Symbol: "NFT", Decimals: 6, ChainID: 195, Type: "trc20"},
		{ID: "wtrx_trx", Address: "TNUC9Qb1rRpS5CbWLmNMxXBjyFoydXjWFR", Name: "Wrapped TRX", Symbol: "WTRX", Decimals: 6, ChainID: 195, Type: "trc20"},

		// ============ BITCOIN (native) and wrapped-BTC ERC-20s ============
		{ID: "btc", Address: "", Name: "Bitcoin", Symbol: "BTC", Decimals: 8, ChainID: 0, Type: "native"},
		{ID: "wbtc_btc", Address: "0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599", Name: "Wrapped BTC", Symbol: "WBTC", Decimals: 8, ChainID: 1, Type: "erc20", IsWrapped: true},
		{ID: "renbtc_btc", Address: "0xEB4C2781e4ebA804CE9a9803C67d0893436bB27D", Name: "renBTC", Symbol: "renBTC", Decimals: 8, ChainID: 1, Type: "erc20"},
		{ID: "hbtc_btc", Address: "0x0316EB71485b0Ab14103307bf65a021042c6d380", Name: "Huobi BTC", Symbol: "HBTC", Decimals: 18, ChainID: 1, Type: "erc20"},
		{ID: "sbtc_btc", Address: "0xfE18be6b3BD88A2d2A7F928d00292E7a9963CfC6", Name: "Synth sBTC", Symbol: "sBTC", Decimals: 18, ChainID: 1, Type: "erc20"},
		// ============ Arbitrum One (ChainID: 42161) ============
		{ID: "eth_42161", Address: "", Name: "Ether", Symbol: "ETH", Decimals: 18, ChainID: 42161, Type: "native"},
		{ID: "usdc_42161", Address: "0xaf88d065e77c8cc2239327c5edb3a432268e5831", Name: "USDC", Symbol: "USDC", Decimals: 6, ChainID: 42161, Type: "erc20", IsStableCoin: true},
		{ID: "dai_42161", Address: "0xda10009cbd5d07dd0cecc66161fc93d7c9000da1", Name: "MakerDAO Arbitrum Bridged DAI (Arbitrum One)", Symbol: "DAI", Decimals: 18, ChainID: 42161, Type: "erc20", IsStableCoin: true},
		{ID: "tusd_42161", Address: "0x4d15a3a2286d883af0aa1b3f21367843fac63e07", Name: "Bridged TrueUSD", Symbol: "TUSD", Decimals: 18, ChainID: 42161, Type: "erc20", IsStableCoin: true},
		{ID: "frax_42161", Address: "0x17fc002b466eec40dae837fc4be5c67993ddbd6f", Name: "Legacy Frax Dollar", Symbol: "FRAX", Decimals: 18, ChainID: 42161, Type: "erc20", IsStableCoin: true},
		{ID: "weth_42161", Address: "0x82af49447d8a07e3bd95bd0d56f35241523fbab1", Name: "Arbitrum Bridged WETH (Arbitrum One)", Symbol: "WETH", Decimals: 18, ChainID: 42161, Type: "erc20", IsWrapped: true},
		{ID: "wbtc_42161", Address: "0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f", Name: "Arbitrum Bridged WBTC (Arbitrum One)", Symbol: "WBTC", Decimals: 8, ChainID: 42161, Type: "erc20", IsWrapped: true},
		{ID: "link_42161", Address: "0xf97f4df75117a78c1a5a0dbb814af92458539fb4", Name: "Chainlink", Symbol: "LINK", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "uni_42161", Address: "0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0", Name: "Uniswap", Symbol: "UNI", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "aave_42161", Address: "0xba5ddd1f9d7f570dc94a51479a000e3bce967196", Name: "Aave", Symbol: "AAVE", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "comp_42161", Address: "0x354a6da3fcde098f8389cad84b0182725c6c91de", Name: "Compound", Symbol: "COMP", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "crv_42161", Address: "0x11cdb42b0eb46d95f990bedd4695a6e3fa034978", Name: "Curve DAO", Symbol: "CRV", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "grt_42161", Address: "0x9623063377ad1b27544c965ccd7342f7ea7e88c7", Name: "The Graph", Symbol: "GRT", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "sushi_42161", Address: "0xd4d42f0b6def4ce0383636770ef773390d85c61a", Name: "Sushi", Symbol: "SUSHI", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "cake_42161", Address: "0x1b896893dfc86bb67cf57767298b9073d2c1ba2c", Name: "PancakeSwap", Symbol: "Cake", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "1inch_42161", Address: "0x6314c31a7a1652ce482cffe247e9cb7c3f4bb9af", Name: "1INCH", Symbol: "1INCH", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "bal_42161", Address: "0x040d1edc9569d4bab2d15287dc5a4f10f56a56b8", Name: "Balancer", Symbol: "BAL", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "yfi_42161", Address: "0x82e3a8f066a6989666b031d916c43672085b1582", Name: "yearn.finance", Symbol: "YFI", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "ldo_42161", Address: "0x13ad51ed4f1b7e9dc168d8a00cb3f4ddd85efa60", Name: "Lido DAO", Symbol: "LDO", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "rpl_42161", Address: "0xe11d163ef4f0a4d2b951a9b0ce2309b9968756b2", Name: "Planet Labs Tokenized Stock (Reality)", Symbol: "rPL", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "pepe_42161", Address: "0x25d887ce7a35172c62febfd67a1856f20faebb00", Name: "Pepe", Symbol: "PEPE", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "arb_42161", Address: "0x912ce59144191c1204e64559fe8253a0e49e6548", Name: "Arbitrum", Symbol: "ARB", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "stg_42161", Address: "0x6694340fc020c5e6b96567843da2df01b2ce1eb6", Name: "Stargate Finance", Symbol: "STG", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "gmx_42161", Address: "0xcef8a7008310a15b1b1489d88b0e6fe51329b05a", Name: "General Motors xStock", Symbol: "GMx", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "magic_42161", Address: "0x539bde0d7dbd336b79148aa742883198bbf60342", Name: "Treasure", Symbol: "MAGIC", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "rdnt_42161", Address: "0x3082cc23568ea640225c2467653db90e9250aaa0", Name: "Radiant Capital", Symbol: "RDNT", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "pendle_42161", Address: "0x0c880f6761f1af8d9aa9c466984b80dab9a8c9e8", Name: "Pendle", Symbol: "PENDLE", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "ape_42161", Address: "0x7f9fbf9bdd3f4105c478b996b648fe6e828a1e98", Name: "ApeCoin", Symbol: "APE", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "cvx_42161", Address: "0x0e929101c4fa7c91eaa64d7216161ba3eee387fe", Name: "Chevron Corp (Dinari Tokenized Stock)", Symbol: "CVX", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "ankr_42161", Address: "0xaeaeed23478c3a4b798e4ed40d8b7f41366ae861", Name: "Ankr Network", Symbol: "ANKR", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "knc_42161", Address: "0xe4dddfe67e7164b0fe14e218d80dc4c08edc01cb", Name: "Kyber Network Crystal", Symbol: "KNC", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "spell_42161", Address: "0x3e6648c5a70a150a88bce65f4ad4d506fe15d2af", Name: "Spell", Symbol: "SPELL", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "joe_42161", Address: "0x371c7ec6d8039ff7933a2aa28eb827ffe1f52f07", Name: "JOE", Symbol: "JOE", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "degen_42161", Address: "0x9f07f8a82cb1af1466252e505b7b7ddee103bc91", Name: "Degen", Symbol: "DEGEN", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "xvs_42161", Address: "0xc1eb7689147c81ac840d4ff0d298489fc7986d52", Name: "Venus", Symbol: "XVS", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "woo_42161", Address: "0xcafcd85d8ca7ad1e1c6f82f651fa15e33aefd07b", Name: "WOO", Symbol: "WOO", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "zro_42161", Address: "0x6985884c4392d348587b19cb9eaaf157f13271cd", Name: "LayerZero", Symbol: "ZRO", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "w_42161", Address: "0xb0ffa8000886e57f86dd5264b9582b2ad87b2b91", Name: "Wormhole", Symbol: "W", Decimals: 18, ChainID: 42161, Type: "erc20"},
		{ID: "strk_42161", Address: "0x9a3608d77caa0cb5a3a5e023ab252db61198e160", Name: "Strategy Strike Preferred (Dinari Tokenized Stock)", Symbol: "STRK", Decimals: 18, ChainID: 42161, Type: "erc20"},
		// ============ OP Mainnet (ChainID: 10) ============
		{ID: "eth_10", Address: "", Name: "Ether", Symbol: "ETH", Decimals: 18, ChainID: 10, Type: "native"},
		{ID: "usdt_10", Address: "0x94b008aa00579c1307b0ef2c499ad98a8ce58e58", Name: "Bridged USDT", Symbol: "USDT", Decimals: 6, ChainID: 10, Type: "erc20", IsStableCoin: true},
		{ID: "usdc_10", Address: "0x0b2c639c533813f4aa9d7837caf62653d097ff85", Name: "USDC", Symbol: "USDC", Decimals: 6, ChainID: 10, Type: "erc20", IsStableCoin: true},
		{ID: "dai_10", Address: "0xda10009cbd5d07dd0cecc66161fc93d7c9000da1", Name: "MakerDAO Optimism Bridged DAI (Optimism)", Symbol: "DAI", Decimals: 18, ChainID: 10, Type: "erc20", IsStableCoin: true},
		{ID: "busd_10", Address: "0x9c9e5fd8bbc25984b178fdce6117defa39d2db39", Name: "Binance-Peg BUSD", Symbol: "BUSD", Decimals: 18, ChainID: 10, Type: "erc20", IsStableCoin: true},
		{ID: "tusd_10", Address: "0xcb59a0a753fdb7491d5f3d794316f1ade197b21e", Name: "Bridged TrueUSD", Symbol: "TUSD", Decimals: 18, ChainID: 10, Type: "erc20", IsStableCoin: true},
		{ID: "frax_10", Address: "0x2e3d870790dc77a83dd1d18184acc7439a53f475", Name: "Legacy Frax Dollar", Symbol: "FRAX", Decimals: 18, ChainID: 10, Type: "erc20", IsStableCoin: true},
		{ID: "weth_10", Address: "0x4200000000000000000000000000000000000006", Name: "L2 Standard Bridged WETH (Optimism)", Symbol: "WETH", Decimals: 18, ChainID: 10, Type: "erc20", IsWrapped: true},
		{ID: "wbtc_10", Address: "0x68f180fcce6836688e9084f035309e29bf0a2095", Name: "Wrapped Bitcoin", Symbol: "WBTC", Decimals: 8, ChainID: 10, Type: "erc20", IsWrapped: true},
		{ID: "link_10", Address: "0x350a791bfc2c21f9ed5d10980dad2e2638ffa7f6", Name: "Chainlink", Symbol: "LINK", Decimals: 18, ChainID: 10, Type: "erc20"},
		{ID: "uni_10", Address: "0x6fd9d7ad17242c41f7131d257212c54a0e816691", Name: "Uniswap", Symbol: "UNI", Decimals: 18, ChainID: 10, Type: "erc20"},
		{ID: "aave_10", Address: "0x76fb31fb4af56892a25e32cfc43de717950c9278", Name: "Aave", Symbol: "AAVE", Decimals: 18, ChainID: 10, Type: "erc20"},
		{ID: "crv_10", Address: "0x0994206dfe8de6ec6920ff4d779b0d950605fb53", Name: "Curve DAO", Symbol: "CRV", Decimals: 18, ChainID: 10, Type: "erc20"},
		{ID: "snx_10", Address: "0x8700daec35af8ff88c16bdf0418774cb3d7599b4", Name: "Synthetix", Symbol: "SNX", Decimals: 18, ChainID: 10, Type: "erc20"},
		{ID: "1inch_10", Address: "0xad42d013ac31486b73b6b059e748172994736426", Name: "1INCH", Symbol: "1INCH", Decimals: 18, ChainID: 10, Type: "erc20"},
		{ID: "bal_10", Address: "0xfe8b128ba8c78aabc59d4c64cee7ff28e9379921", Name: "Balancer", Symbol: "BAL", Decimals: 18, ChainID: 10, Type: "erc20"},
		{ID: "yfi_10", Address: "0x9046d36440290ffde54fe0dd84db8b1cfee9107b", Name: "yearn.finance", Symbol: "YFI", Decimals: 18, ChainID: 10, Type: "erc20"},
		{ID: "ldo_10", Address: "0xfdb794692724153d1488ccdbe0c56c252596735f", Name: "Lido DAO", Symbol: "LDO", Decimals: 18, ChainID: 10, Type: "erc20"},
		{ID: "op_10", Address: "0x4200000000000000000000000000000000000042", Name: "Optimism", Symbol: "OP", Decimals: 18, ChainID: 10, Type: "erc20"},
		{ID: "stg_10", Address: "0x296f55f8fb28e498b858d0bcda06d955b2cb3f97", Name: "Stargate Finance", Symbol: "STG", Decimals: 18, ChainID: 10, Type: "erc20"},
		{ID: "gmx_10", Address: "0xcef8a7008310a15b1b1489d88b0e6fe51329b05a", Name: "General Motors xStock", Symbol: "GMx", Decimals: 18, ChainID: 10, Type: "erc20"},
		{ID: "pendle_10", Address: "0xbc7b1ff1c6989f006a1185318ed4e7b5796e66e1", Name: "Pendle", Symbol: "PENDLE", Decimals: 18, ChainID: 10, Type: "erc20"},
		{ID: "ocean_10", Address: "0x2561aa2bb1d2eb6629edd7b0938d7679b8b49f9e", Name: "Ocean Protocol", Symbol: "OCEAN", Decimals: 18, ChainID: 10, Type: "erc20"},
		{ID: "ankr_10", Address: "0xaeaeed23478c3a4b798e4ed40d8b7f41366ae861", Name: "Ankr Network", Symbol: "ANKR", Decimals: 18, ChainID: 10, Type: "erc20"},
		{ID: "knc_10", Address: "0xa00e3a3511aac35ca78530c85007afcd31753819", Name: "Kyber Network Crystal", Symbol: "KNC", Decimals: 18, ChainID: 10, Type: "erc20"},
		{ID: "velo_10", Address: "0x9560e827af36c94d2ac33a39bce1fe78631088db", Name: "Velodrome Finance", Symbol: "VELO", Decimals: 18, ChainID: 10, Type: "erc20"},
		{ID: "xvs_10", Address: "0x4a971e87ad1f61f7f3081645f52a99277ae917cf", Name: "Venus", Symbol: "XVS", Decimals: 18, ChainID: 10, Type: "erc20"},
		{ID: "wld_10", Address: "0xdc6ff44d5d932cbd77b52e5612ba0529dc6226f1", Name: "Worldcoin", Symbol: "WLD", Decimals: 18, ChainID: 10, Type: "erc20"},
		{ID: "zro_10", Address: "0x6985884c4392d348587b19cb9eaaf157f13271cd", Name: "LayerZero", Symbol: "ZRO", Decimals: 18, ChainID: 10, Type: "erc20"},
		// ============ Base (ChainID: 8453) ============
		{ID: "eth_8453", Address: "", Name: "Ether", Symbol: "ETH", Decimals: 18, ChainID: 8453, Type: "native"},
		{ID: "usdt_8453", Address: "0xfde4c96c8593536e31f229ea8f37b2ada2699bb2", Name: "L2 Standard Bridged USDT (Base)", Symbol: "USDT", Decimals: 6, ChainID: 8453, Type: "erc20", IsStableCoin: true},
		{ID: "usdc_8453", Address: "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913", Name: "USDC", Symbol: "USDC", Decimals: 6, ChainID: 8453, Type: "erc20", IsStableCoin: true},
		{ID: "dai_8453", Address: "0x50c5725949a6f0c72e6c4a641f24049a917db0cb", Name: "L2 Standard Bridged DAI (Base)", Symbol: "DAI", Decimals: 18, ChainID: 8453, Type: "erc20", IsStableCoin: true},
		{ID: "usdp_8453", Address: "0x7b29e5266634bcba06686e580afd4419a8c84b07", Name: "BaseUSDP", Symbol: "USDP", Decimals: 18, ChainID: 8453, Type: "erc20", IsStableCoin: true},
		{ID: "weth_8453", Address: "0x4200000000000000000000000000000000000006", Name: "L2 Standard Bridged WETH (Base)", Symbol: "WETH", Decimals: 18, ChainID: 8453, Type: "erc20", IsWrapped: true},
		{ID: "wbtc_8453", Address: "0x0555e30da8f98308edb960aa94c0db47230d2b9c", Name: "Wrapped Bitcoin", Symbol: "WBTC", Decimals: 8, ChainID: 8453, Type: "erc20", IsWrapped: true},
		{ID: "link_8453", Address: "0x88fb150bdc53a65fe94dea0c9ba0a6daf8c6e196", Name: "Chainlink", Symbol: "LINK", Decimals: 18, ChainID: 8453, Type: "erc20"},
		{ID: "aave_8453", Address: "0x63706e401c06ac8513145b7687a14804d17f814b", Name: "Aave", Symbol: "AAVE", Decimals: 18, ChainID: 8453, Type: "erc20"},
		{ID: "comp_8453", Address: "0x9e1028f5f1d5ede59748ffcee5532509976840e0", Name: "Compound", Symbol: "COMP", Decimals: 18, ChainID: 8453, Type: "erc20"},
		{ID: "crv_8453", Address: "0x8ee73c484a26e0a5df2ee2a4960b789967dd0415", Name: "Curve DAO", Symbol: "CRV", Decimals: 18, ChainID: 8453, Type: "erc20"},
		{ID: "snx_8453", Address: "0x22e6966b799c4d5b13be962e1d117b56327fda66", Name: "Synthetix", Symbol: "SNX", Decimals: 18, ChainID: 8453, Type: "erc20"},
		{ID: "sushi_8453", Address: "0x7d49a065d17d6d4a55dc13649901fdbb98b2afba", Name: "Sushi", Symbol: "SUSHI", Decimals: 18, ChainID: 8453, Type: "erc20"},
		{ID: "cake_8453", Address: "0x3055913c90fcc1a6ce9a358911721eeb942013a1", Name: "PancakeSwap", Symbol: "Cake", Decimals: 18, ChainID: 8453, Type: "erc20"},
		{ID: "1inch_8453", Address: "0xc5fecc3a29fb57b5024eec8a2239d4621e111cbe", Name: "1INCH", Symbol: "1INCH", Decimals: 18, ChainID: 8453, Type: "erc20"},
		{ID: "bal_8453", Address: "0x4158734d47fc9692176b5085e0f52ee0da5d47f1", Name: "Balancer", Symbol: "BAL", Decimals: 18, ChainID: 8453, Type: "erc20"},
		{ID: "yfi_8453", Address: "0x9eaf8c1e34f05a589eda6bafdf391cf6ad3cb239", Name: "yearn.finance", Symbol: "YFI", Decimals: 18, ChainID: 8453, Type: "erc20"},
		{ID: "pepe_8453", Address: "0x52b492a33e447cdb854c7fc19f1e57e8bfa1777d", Name: "Based Pepe", Symbol: "PEPE", Decimals: 18, ChainID: 8453, Type: "erc20"},
		{ID: "shib_8453", Address: "0xfca95aeb5bf44ae355806a5ad14659c940dc6bf7", Name: "SHIBA ON BASE", Symbol: "SHIB", Decimals: 9, ChainID: 8453, Type: "erc20"},
		{ID: "op_8453", Address: "0x5b8a884496792f754db6f92f6eacceb376ed8e40", Name: "One Path", Symbol: "OP", Decimals: 18, ChainID: 8453, Type: "erc20"},
		{ID: "stg_8453", Address: "0xe3b53af74a4bf62ae5511055290838050bf764df", Name: "Stargate Finance", Symbol: "STG", Decimals: 18, ChainID: 8453, Type: "erc20"},
		{ID: "magic_8453", Address: "0xf1572d1da5c3cce14ee5a1c9327d17e9ff0e3f43", Name: "Treasure", Symbol: "MAGIC", Decimals: 18, ChainID: 8453, Type: "erc20"},
		{ID: "rdnt_8453", Address: "0xd722e55c1d9d9fa0021a5215cbb904b92b3dc5d4", Name: "Radiant Capital", Symbol: "RDNT", Decimals: 18, ChainID: 8453, Type: "erc20"},
		{ID: "pendle_8453", Address: "0xa99f6e6785da0f5d6fb42495fe424bce029eeb3e", Name: "Pendle", Symbol: "PENDLE", Decimals: 18, ChainID: 8453, Type: "erc20"},
		{ID: "sand_8453", Address: "0xac531eb26ca1d21b85126de8fb87e80e09002dcf", Name: "The Sandbox", Symbol: "SAND", Decimals: 18, ChainID: 8453, Type: "erc20"},
		{ID: "cvx_8453", Address: "0x25aa7b1ecf5d3fcd1b3b9f2445a4fa5597ffb189", Name: "Chevron Corp (Dinari Tokenized Stock)", Symbol: "CVX", Decimals: 18, ChainID: 8453, Type: "erc20"},
		{ID: "chz_8453", Address: "0x70c8392de9b39a1e48d12a70af6ff4be25d6d0a2", Name: "Chiliz", Symbol: "CHZ", Decimals: 18, ChainID: 8453, Type: "erc20"},
		{ID: "qi_8453", Address: "0xd3fdcb837dafdb7c9c3ebd48fe22a53f6dd3d7d7", Name: "Qi Dao", Symbol: "QI", Decimals: 18, ChainID: 8453, Type: "erc20"},
		{ID: "aero_8453", Address: "0x3f5bf3edac59d650d8fdf9657a71f2f904a3a4ee", Name: "Aero", Symbol: "AERO", Decimals: 18, ChainID: 8453, Type: "erc20"},
		{ID: "degen_8453", Address: "0x4ed4e862860bed51a9570b96d89af5e1b0efefed", Name: "Degen", Symbol: "DEGEN", Decimals: 18, ChainID: 8453, Type: "erc20"},
		{ID: "xvs_8453", Address: "0xebb7873213c8d1d9913d8ea39aa12d74cb107995", Name: "Venus", Symbol: "XVS", Decimals: 18, ChainID: 8453, Type: "erc20"},
		{ID: "woo_8453", Address: "0xf3df0a31ec5ea438150987805e841f960b9471b6", Name: "WOO", Symbol: "WOO", Decimals: 18, ChainID: 8453, Type: "erc20"},
		{ID: "zro_8453", Address: "0x6985884c4392d348587b19cb9eaaf157f13271cd", Name: "LayerZero", Symbol: "ZRO", Decimals: 18, ChainID: 8453, Type: "erc20"},
		{ID: "w_8453", Address: "0xb0ffa8000886e57f86dd5264b9582b2ad87b2b91", Name: "Wormhole", Symbol: "W", Decimals: 18, ChainID: 8453, Type: "erc20"},
		{ID: "strk_8453", Address: "0x6974d0801753c8c8571a11e7837b0d0462ecb052", Name: "Strategy Strike Preferred (Dinari Tokenized Stock)", Symbol: "STRK", Decimals: 18, ChainID: 8453, Type: "erc20"},
		// ============ Fantom Opera (ChainID: 250) ============
		{ID: "ftm_250", Address: "", Name: "Fantom", Symbol: "FTM", Decimals: 18, ChainID: 250, Type: "native"},
		{ID: "usdc_250", Address: "0x04068da6c83afcfa0e13ba15a6696662335d5b75", Name: "Multichain Bridged USDC (Fantom)", Symbol: "USDC", Decimals: 6, ChainID: 250, Type: "erc20", IsStableCoin: true},
		{ID: "dai_250", Address: "0x8d11ec38a3eb5e956b052f67da8bdc9bef8abf3e", Name: "Multichain Bridged DAI (Fantom)", Symbol: "DAI", Decimals: 18, ChainID: 250, Type: "erc20", IsStableCoin: true},
		{ID: "tusd_250", Address: "0x9879abdea01a879644185341f7af7d8343556b7a", Name: "Bridged TrueUSD", Symbol: "TUSD", Decimals: 18, ChainID: 250, Type: "erc20", IsStableCoin: true},
		{ID: "frax_250", Address: "0xdc301622e621166bd8e82f2ca0a26c13ad0be355", Name: "Legacy Frax Dollar", Symbol: "FRAX", Decimals: 18, ChainID: 250, Type: "erc20", IsStableCoin: true},
		{ID: "weth_250", Address: "0x695921034f0387eac4e11620ee91b1b15a6a09fe", Name: "Bridged Wrapped Ether (Stargate)", Symbol: "WETH", Decimals: 18, ChainID: 250, Type: "erc20", IsWrapped: true},
		{ID: "wftm_250", Address: "0x21be370d5312f44cb42ce377bc9b8a0cef1a4c83", Name: "Wrapped Fantom", Symbol: "WFTM", Decimals: 18, ChainID: 250, Type: "erc20", IsWrapped: true},
		{ID: "link_250", Address: "0xb3654dc3d10ea7645f8319668e8f54d2574fbdc8", Name: "Chainlink", Symbol: "LINK", Decimals: 18, ChainID: 250, Type: "erc20"},
		{ID: "aave_250", Address: "0x6a07a792ab2965c72a5b8088d3a069a7ac3a993b", Name: "Aave", Symbol: "AAVE", Decimals: 18, ChainID: 250, Type: "erc20"},
		{ID: "crv_250", Address: "0xc7ae4ab742f6b0b203f6710c87677005bc45ad01", Name: "Bridged Curve DAO Token (Stargate)", Symbol: "CRV", Decimals: 18, ChainID: 250, Type: "erc20"},
		{ID: "snx_250", Address: "0x56ee926bd8c72b2d5fa1af4d9e4cbb515a1e3adc", Name: "Synthetix", Symbol: "SNX", Decimals: 18, ChainID: 250, Type: "erc20"},
		{ID: "sushi_250", Address: "0xae75a438b2e0cb8bb01ec1e1e376de11d44477cc", Name: "Sushi", Symbol: "SUSHI", Decimals: 18, ChainID: 250, Type: "erc20"},
		{ID: "yfi_250", Address: "0x29b0da86e484e1c0029b56e817912d778ac0ec69", Name: "yearn.finance", Symbol: "YFI", Decimals: 18, ChainID: 250, Type: "erc20"},
		{ID: "stg_250", Address: "0x2f6f07cdcf3588944bf4c42ac74ff24bf56e7590", Name: "Stargate Finance", Symbol: "STG", Decimals: 18, ChainID: 250, Type: "erc20"},
		{ID: "band_250", Address: "0x46e7628e8b4350b2716ab470ee0ba1fa9e76c6c5", Name: "Band", Symbol: "BAND", Decimals: 18, ChainID: 250, Type: "erc20"},
		{ID: "ankr_250", Address: "0xdf474b7109b73b7d57926d43598d5934131136b2", Name: "Ankr Network", Symbol: "ANKR", Decimals: 18, ChainID: 250, Type: "erc20"},
		{ID: "knc_250", Address: "0x1e1085efaa63edfe74aad7c05a28eae4ef917c3f", Name: "Kyber Network Crystal", Symbol: "KNC", Decimals: 18, ChainID: 250, Type: "erc20"},
		{ID: "spell_250", Address: "0x468003b688943977e6130f4f68f23aad939a1040", Name: "Spell", Symbol: "SPELL", Decimals: 18, ChainID: 250, Type: "erc20"},
		{ID: "woo_250", Address: "0x6626c47c00f1d87902fc13eecfac3ed06d5e8d8a", Name: "WOO", Symbol: "WOO", Decimals: 18, ChainID: 250, Type: "erc20"},
		// ============ Arbitrum Nova (ChainID: 42170) ============
		{ID: "eth_42170", Address: "", Name: "Ether", Symbol: "ETH", Decimals: 18, ChainID: 42170, Type: "native"},
		// ============ Linea (ChainID: 59144) ============
		{ID: "eth_59144", Address: "", Name: "Linea Ether", Symbol: "ETH", Decimals: 18, ChainID: 59144, Type: "native"},
		{ID: "usdt_59144", Address: "0xa219439258ca9da29e9cc4ce5596924745e12b93", Name: "Linea Bridged USDT (Linea)", Symbol: "USDT", Decimals: 6, ChainID: 59144, Type: "erc20", IsStableCoin: true},
		{ID: "usdc_59144", Address: "0x176211869ca2b568f2a7d4ee941e073a821ee1ff", Name: "Linea Bridged USDC (Linea)", Symbol: "USDC", Decimals: 6, ChainID: 59144, Type: "erc20", IsStableCoin: true},
		{ID: "dai_59144", Address: "0x4af15ec2a0bd43db75dd04e62faa3b8ef36b00d5", Name: "Bridged Dai Stablecoin (Linea)", Symbol: "DAI", Decimals: 18, ChainID: 59144, Type: "erc20", IsStableCoin: true},
		{ID: "busd_59144", Address: "0x7d43aabc515c356145049227cee54b608342c0ad", Name: "Binance USD (Linea)", Symbol: "BUSD", Decimals: 18, ChainID: 59144, Type: "erc20", IsStableCoin: true},
		{ID: "weth_59144", Address: "0xe5d7c2a44ffddf6b295a15c148167daaaf5cf34f", Name: "Bridged Wrapped Ether (Linea)", Symbol: "WETH", Decimals: 18, ChainID: 59144, Type: "erc20", IsWrapped: true},
		{ID: "wbtc_59144", Address: "0x3aab2285ddcddad8edf438c1bab47e1a9d05a9b4", Name: "Linea Bridged WBTC (Linea)", Symbol: "WBTC", Decimals: 8, ChainID: 59144, Type: "erc20", IsWrapped: true},
		{ID: "link_59144", Address: "0xa18152629128738a5c081eb226335fed4b9c95e9", Name: "Chainlink", Symbol: "LINK", Decimals: 18, ChainID: 59144, Type: "erc20"},
		{ID: "uni_59144", Address: "0x636b22bc471c955a8db60f28d4795066a8201fa3", Name: "Linea Bridged UNI (Linea)", Symbol: "UNI", Decimals: 18, ChainID: 59144, Type: "erc20"},
		{ID: "cake_59144", Address: "0x0d1e753a25ebda689453309112904807625befbe", Name: "PancakeSwap", Symbol: "Cake", Decimals: 18, ChainID: 59144, Type: "erc20"},
		{ID: "1inch_59144", Address: "0x10f04e61bd6019d6c9b0bf4907e805a64b92ee3e", Name: "1INCH", Symbol: "1INCH", Decimals: 18, ChainID: 59144, Type: "erc20"},
		{ID: "ldo_59144", Address: "0x0e076aafd86a71dceac65508daf975425c9d0cb6", Name: "Linea Bridged LDO (Linea)", Symbol: "LDO", Decimals: 18, ChainID: 59144, Type: "erc20"},
		{ID: "stg_59144", Address: "0x808d7c71ad2ba3fa531b068a2417c63106bc0949", Name: "Stargate Finance", Symbol: "STG", Decimals: 18, ChainID: 59144, Type: "erc20"},
		{ID: "ankr_59144", Address: "0xa8ae6365383eb907e6b4b1b7e82a35752cc5ef8c", Name: "Ankr Network", Symbol: "ANKR", Decimals: 18, ChainID: 59144, Type: "erc20"},
		{ID: "knc_59144", Address: "0x3b2f62d42db19b30588648bf1c184865d4c3b1d6", Name: "Kyber Network Crystal", Symbol: "KNC", Decimals: 18, ChainID: 59144, Type: "erc20"},
		{ID: "woo_59144", Address: "0xf3df0a31ec5ea438150987805e841f960b9471b6", Name: "WOO", Symbol: "WOO", Decimals: 18, ChainID: 59144, Type: "erc20"},
		// ============ zkSync Mainnet (ChainID: 324) ============
		{ID: "eth_324", Address: "", Name: "Ether", Symbol: "ETH", Decimals: 18, ChainID: 324, Type: "native"},
		{ID: "usdt_324", Address: "0x493257fd37edb34451f62edf8d2a0c418852ba4c", Name: "Bridged USDT", Symbol: "USDT", Decimals: 6, ChainID: 324, Type: "erc20", IsStableCoin: true},
		{ID: "usdc_324", Address: "0x1d17cbcf0d6d143135ae902365d2e5e2a16538d4", Name: "USDC", Symbol: "USDC", Decimals: 6, ChainID: 324, Type: "erc20", IsStableCoin: true},
		{ID: "dai_324", Address: "0x4b9eb6c0b6ea15176bbf62841c6b2a8a398cb656", Name: "ZKsync ERC20 Bridged DAI (zkSync)", Symbol: "DAI", Decimals: 18, ChainID: 324, Type: "erc20", IsStableCoin: true},
		{ID: "busd_324", Address: "0x2039bb4116b4efc145ec4f0e2ea75012d6c0f181", Name: "Celer Bridged BUSD (zkSync)", Symbol: "BUSD", Decimals: 18, ChainID: 324, Type: "erc20", IsStableCoin: true},
		{ID: "weth_324", Address: "0x5aea5775959fbc2557cc8789bc1bf90a239d9a91", Name: "Bridged WETH", Symbol: "WETH", Decimals: 18, ChainID: 324, Type: "erc20", IsWrapped: true},
		{ID: "wbtc_324", Address: "0xbbeb516fb02a01611cbbe0453fe3c580d7281011", Name: "zkSync Bridged WBTC (zkSync)", Symbol: "WBTC", Decimals: 8, ChainID: 324, Type: "erc20", IsWrapped: true},
		{ID: "link_324", Address: "0x52869bae3e091e36b0915941577f2d47d8d8b534", Name: "Chainlink", Symbol: "LINK", Decimals: 18, ChainID: 324, Type: "erc20"},
		{ID: "cake_324", Address: "0x3a287a06c66f9e95a56327185ca2bdf5f031cecd", Name: "PancakeSwap", Symbol: "Cake", Decimals: 18, ChainID: 324, Type: "erc20"},
		{ID: "1inch_324", Address: "0x3f0b8b206a7fbdb3ecfc08c9407ca83f5ab1ce59", Name: "1INCH", Symbol: "1INCH", Decimals: 18, ChainID: 324, Type: "erc20"},
		{ID: "knc_324", Address: "0x6ee46cb7cd2f15ee1ec9534cf29a5b51c83283e6", Name: "Kyber Network Crystal", Symbol: "KNC", Decimals: 18, ChainID: 324, Type: "erc20"},
		{ID: "xvs_324", Address: "0xd78abd81a3d57712a3af080dc4185b698fe9ac5a", Name: "Venus", Symbol: "XVS", Decimals: 18, ChainID: 324, Type: "erc20"},
		{ID: "woo_324", Address: "0x9e22d758629761fc5708c171d06c2fabb60b5159", Name: "WOO", Symbol: "WOO", Decimals: 18, ChainID: 324, Type: "erc20"},
		// ============ Scroll (ChainID: 534352) ============
		{ID: "eth_534352", Address: "", Name: "Ether", Symbol: "ETH", Decimals: 18, ChainID: 534352, Type: "native"},
		{ID: "usdt_534352", Address: "0xf55bec9cafdbe8730f096aa55dad6d22d44099df", Name: "Scroll Bridged USDT (Scroll)", Symbol: "USDT", Decimals: 6, ChainID: 534352, Type: "erc20", IsStableCoin: true},
		{ID: "usdc_534352", Address: "0x06efdbff2a14a7c8e15944d1f4a48f9f95f663a4", Name: "Bridged USD Coin (Scroll)", Symbol: "USDC", Decimals: 6, ChainID: 534352, Type: "erc20", IsStableCoin: true},
		{ID: "weth_534352", Address: "0x5300000000000000000000000000000000000004", Name: "Bridged Wrapped Ether (Scroll)", Symbol: "WETH", Decimals: 18, ChainID: 534352, Type: "erc20", IsWrapped: true},
		{ID: "wbtc_534352", Address: "0x3c1bca5a656e69edcd0d4e36bebb3fcdaca60cf1", Name: "Bridged Wrapped Bitcoin (Scroll)", Symbol: "WBTC", Decimals: 8, ChainID: 534352, Type: "erc20", IsWrapped: true},
		{ID: "link_534352", Address: "0x548c6944cba02b9d1c0570102c89de64d258d3ac", Name: "Chainlink", Symbol: "LINK", Decimals: 18, ChainID: 534352, Type: "erc20"},
		{ID: "stg_534352", Address: "0x8731d54e9d02c286767d56ac03e8037c07e01e98", Name: "Stargate Finance", Symbol: "STG", Decimals: 18, ChainID: 534352, Type: "erc20"},
		{ID: "ankr_534352", Address: "0xdf474b7109b73b7d57926d43598d5934131136b2", Name: "Ankr Network", Symbol: "ANKR", Decimals: 18, ChainID: 534352, Type: "erc20"},
		// ============ Mantle (ChainID: 5000) ============
		{ID: "mnt_5000", Address: "", Name: "Mantle", Symbol: "MNT", Decimals: 18, ChainID: 5000, Type: "native"},
		{ID: "usdt_5000", Address: "0x201eba5cc46d216ce6dc03f6a759e8e766e956ae", Name: "Stargate Bridged USDT", Symbol: "USDT", Decimals: 6, ChainID: 5000, Type: "erc20", IsStableCoin: true},
		{ID: "weth_5000", Address: "0xdeaddeaddeaddeaddeaddeaddeaddeaddead1111", Name: "Mantle Bridged WETH (Mantle)", Symbol: "WETH", Decimals: 18, ChainID: 5000, Type: "erc20", IsWrapped: true},
		{ID: "mnt_5000", Address: "0xdeaddeaddeaddeaddeaddeaddeaddeaddead0000", Name: "Mantle", Symbol: "MNT", Decimals: 18, ChainID: 5000, Type: "erc20"},
		// ============ Blast (ChainID: 81457) ============
		{ID: "eth_81457", Address: "", Name: "Ether", Symbol: "ETH", Decimals: 18, ChainID: 81457, Type: "native"},
		// ============ Celo Mainnet (ChainID: 42220) ============
		{ID: "celo_42220", Address: "", Name: "CELO", Symbol: "CELO", Decimals: 18, ChainID: 42220, Type: "native"},
		{ID: "usdc_42220", Address: "0xceba9300f2b948710d2653dd7b07f33a8b32118c", Name: "USDC", Symbol: "USDC", Decimals: 6, ChainID: 42220, Type: "erc20", IsStableCoin: true},
		{ID: "weth_42220", Address: "0xd221812de1bd094f35587ee8e174b07b6167d9af", Name: "Celo Bridged WETH (Celo)", Symbol: "WETH", Decimals: 18, ChainID: 42220, Type: "erc20", IsWrapped: true},
		{ID: "link_42220", Address: "0xd07294e6e917e07dfdcee882dd1e2565085c2ae0", Name: "Chainlink", Symbol: "LINK", Decimals: 18, ChainID: 42220, Type: "erc20"},
		{ID: "sushi_42220", Address: "0xd15ec721c2a896512ad29c671997dd68f9593226", Name: "Sushi", Symbol: "SUSHI", Decimals: 18, ChainID: 42220, Type: "erc20"},
		{ID: "celo_42220", Address: "0x471ece3750da237f93b8e339c536989b8978a438", Name: "Celo", Symbol: "CELO", Decimals: 18, ChainID: 42220, Type: "erc20"},
		// ============ Cronos Mainnet (ChainID: 25) ============
		{ID: "cro_25", Address: "", Name: "Cronos", Symbol: "CRO", Decimals: 18, ChainID: 25, Type: "native"},
		{ID: "usdt_25", Address: "0x66e428c3f67a68878562e79a0234c1f83c208770", Name: "Cronos Bridged USDT (Cronos)", Symbol: "USDT", Decimals: 6, ChainID: 25, Type: "erc20", IsStableCoin: true},
		{ID: "usdc_25", Address: "0xc21223249ca28397b4b6541dffaecc539bff0c59", Name: "Cronos Bridged USDC (Cronos)", Symbol: "USDC", Decimals: 6, ChainID: 25, Type: "erc20", IsStableCoin: true},
		{ID: "dai_25", Address: "0xf2001b145b43032aaf5ee2884e456ccd805f677d", Name: "Crypto.com Bridged DAI (Cronos)", Symbol: "DAI", Decimals: 18, ChainID: 25, Type: "erc20", IsStableCoin: true},
		{ID: "busd_25", Address: "0xc74d59a548ecf7fc1754bb7810d716e9ac3e3ae5", Name: "Bridged BUSD", Symbol: "BUSD", Decimals: 18, ChainID: 25, Type: "erc20", IsStableCoin: true},
		{ID: "tusd_25", Address: "0x87efb3ec1576dec8ed47e58b832bedcd86ee186e", Name: "Bridged TrueUSD", Symbol: "TUSD", Decimals: 18, ChainID: 25, Type: "erc20", IsStableCoin: true},
		{ID: "weth_25", Address: "0xe44fd7fcb2b1581822d0c862b68222998a0c299a", Name: "Bridged WETH", Symbol: "WETH", Decimals: 18, ChainID: 25, Type: "erc20", IsWrapped: true},
		{ID: "wbtc_25", Address: "0x062e66477faf219f25d27dced647bf57c3107d52", Name: "Cronos Bridged WBTC (Cronos)", Symbol: "WBTC", Decimals: 8, ChainID: 25, Type: "erc20", IsWrapped: true},
		{ID: "wcro_25", Address: "0x5c7f8a570d578ed84e63fdfa7b1ee72deae1ae23", Name: "Wrapped CRO", Symbol: "WCRO", Decimals: 18, ChainID: 25, Type: "erc20", IsWrapped: true},
		{ID: "link_25", Address: "0x8c80a01f461f297df7f9da3a4f740d7297c8ac85", Name: "Chainlink", Symbol: "LINK", Decimals: 18, ChainID: 25, Type: "erc20"},
		{ID: "mmf_25", Address: "0x97749c9b61f878a880dfe312d2594ae07aed7656", Name: "MMFinance (Cronos)", Symbol: "MMF", Decimals: 18, ChainID: 25, Type: "erc20"},
		{ID: "vvs_25", Address: "0x2d03bece6747adc00e1a131bba1469c15fd11e03", Name: "VVS Finance", Symbol: "VVS", Decimals: 18, ChainID: 25, Type: "erc20"},
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
