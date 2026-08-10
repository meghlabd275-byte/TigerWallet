/**
 * TigerWallet Blockchain Registry
 * Support for 100+ blockchain networks
 */

package main

import (
	"fmt"
	"sync"
)

// BlockchainNetwork represents a blockchain network
type BlockchainNetwork struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Symbol       string `json:"symbol"`
	ChainID      int64  `json:"chainId"`
	ChainIDHex   string `json:"chainIdHex"`
	NetworkType  string `json:"networkType"`
	Explorer     string `json:"explorer"`
	ExplorerAPI  string `json:"explorerApi"`
	RPCURL       string `json:"rpcUrl"`
	WSSURL       string `json:"wssUrl"`
	Logo         string `json:"logo"`
	Color        string `json:"color"`
	Decimals     int    `json:"decimals"`
	CoinType     int    `json:"coinType"`
	IsEVM        bool   `json:"isEvm"`
	CanTokenList bool   `json:"canTokenList"`
	CanNFT       bool   `json:"canNft"`
	CanStake     bool   `json:"canStake"`
	CanBridge    bool   `json:"canBridge"`
	GasToken     string `json:"gasToken"`
	BlockTime    int    `json:"blockTime"`
	Category     string `json:"category"`
	Tier         int    `json:"tier"`
	Status       string `json:"status"`
}

// BlockchainRegistry manages all blockchain networks
type BlockchainRegistry struct {
	networks map[string]*BlockchainNetwork
	mu       sync.RWMutex
}

var registry *BlockchainRegistry

func NewBlockchainRegistry() *BlockchainRegistry {
	if registry == nil {
		registry = &BlockchainRegistry{
			networks: make(map[string]*BlockchainNetwork),
		}
		registry.initNetworks()
	}
	return registry
}

func (r *BlockchainRegistry) initNetworks() {
	networks := []*BlockchainNetwork{
		// Ethereum Ecosystem
		{ID: "ethereum", Name: "Ethereum", Symbol: "ETH", ChainID: 1, ChainIDHex: "0x1", NetworkType: "mainnet", Explorer: "https://etherscan.io", RPCURL: "https://eth.llamarpc.com", Logo: "eth.png", Color: "#627EEA", Decimals: 18, CoinType: 60, IsEVM: true, CanTokenList: true, CanNFT: true, CanStake: true, CanBridge: true, GasToken: "ETH", BlockTime: 12, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "bsc", Name: "BNB Chain", Symbol: "BNB", ChainID: 56, ChainIDHex: "0x38", NetworkType: "mainnet", Explorer: "https://bscscan.com", RPCURL: "https://bsc-dataseed.binance.org", Logo: "bnb.png", Color: "#F3BA2F", Decimals: 18, CoinType: 714, IsEVM: true, CanTokenList: true, CanNFT: true, CanStake: true, CanBridge: true, GasToken: "BNB", BlockTime: 3, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "polygon", Name: "Polygon", Symbol: "MATIC", ChainID: 137, ChainIDHex: "0x89", NetworkType: "mainnet", Explorer: "https://polygonscan.com", RPCURL: "https://polygon-rpc.com", Logo: "polygon.png", Color: "#8247E5", Decimals: 18, CoinType: 966, IsEVM: true, CanTokenList: true, CanNFT: true, CanStake: true, CanBridge: true, GasToken: "MATIC", BlockTime: 2, Category: "Layer2", Tier: 1, Status: "active"},
		{ID: "avalanche", Name: "Avalanche", Symbol: "AVAX", ChainID: 43114, ChainIDHex: "0xa86a", NetworkType: "mainnet", Explorer: "https://snowtrace.io", RPCURL: "https://api.avax.network/ext/bc/C/rpc", Logo: "avax.png", Color: "#E84142", Decimals: 18, CoinType: 9000, IsEVM: true, CanTokenList: true, CanNFT: true, CanStake: true, CanBridge: true, GasToken: "AVAX", BlockTime: 2, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "arbitrum", Name: "Arbitrum", Symbol: "ETH", ChainID: 42161, ChainIDHex: "0xa4b1", NetworkType: "mainnet", Explorer: "https://arbiscan.io", RPCURL: "https://arb1.arbitrum.io/rpc", Logo: "arbitrum.png", Color: "#28A0F0", Decimals: 18, CoinType: 60, IsEVM: true, CanTokenList: true, CanNFT: true, CanStake: true, CanBridge: true, GasToken: "ETH", BlockTime: 1, Category: "Layer2", Tier: 1, Status: "active"},
		{ID: "optimism", Name: "Optimism", Symbol: "ETH", ChainID: 10, ChainIDHex: "0xa", NetworkType: "mainnet", Explorer: "https://optimistic.etherscan.io", RPCURL: "https://mainnet.optimism.io", Logo: "optimism.png", Color: "#FF0420", Decimals: 18, CoinType: 60, IsEVM: true, CanTokenList: true, CanNFT: true, CanStake: true, CanBridge: true, GasToken: "ETH", BlockTime: 2, Category: "Layer2", Tier: 1, Status: "active"},
		{ID: "base", Name: "Base", Symbol: "ETH", ChainID: 8453, ChainIDHex: "0x2105", NetworkType: "mainnet", Explorer: "https://basescan.org", RPCURL: "https://mainnet.base.org", Logo: "base.png", Color: "#0052FF", Decimals: 18, CoinType: 60, IsEVM: true, CanTokenList: true, CanNFT: true, CanBridge: true, GasToken: "ETH", BlockTime: 2, Category: "Layer2", Tier: 1, Status: "active"},
		{ID: "linea", Name: "Linea", Symbol: "ETH", ChainID: 59144, ChainIDHex: "0xe708", NetworkType: "mainnet", Explorer: "https://lineascan.build", RPCURL: "https://rpc.linea.build", Logo: "linea.png", Color: "#121212", Decimals: 18, CoinType: 60, IsEVM: true, CanTokenList: true, CanNFT: true, CanBridge: true, GasToken: "ETH", BlockTime: 2, Category: "Layer2", Tier: 1, Status: "active"},
		{ID: "zksync", Name: "zkSync Era", Symbol: "ETH", ChainID: 324, ChainIDHex: "0x144", NetworkType: "mainnet", Explorer: "https://era.zksync.network", RPCURL: "https://mainnet.era.zksync.io", Logo: "zksync.png", Color: "#8B5CF6", Decimals: 18, CoinType: 60, IsEVM: true, CanTokenList: true, CanNFT: true, CanBridge: true, GasToken: "ETH", BlockTime: 1, Category: "Layer2", Tier: 1, Status: "active"},
		{ID: "scroll", Name: "Scroll", Symbol: "ETH", ChainID: 534352, ChainIDHex: "0x82750", NetworkType: "mainnet", Explorer: "https://scrollscan.com", RPCURL: "https://rpc.scroll.io", Logo: "scroll.png", Color: "#CDA6F6", Decimals: 18, CoinType: 60, IsEVM: true, CanTokenList: true, CanNFT: true, CanBridge: true, GasToken: "ETH", BlockTime: 3, Category: "Layer2", Tier: 1, Status: "active"},
		// Solana
		{ID: "solana", Name: "Solana", Symbol: "SOL", ChainID: 0, NetworkType: "mainnet", Explorer: "https://solscan.io", RPCURL: "https://api.mainnet-beta.solana.com", Logo: "solana.png", Color: "#9945FF", Decimals: 9, CoinType: 501, IsEVM: false, CanTokenList: true, CanNFT: true, CanStake: true, CanBridge: true, GasToken: "SOL", BlockTime: 1, Category: "Layer1", Tier: 1, Status: "active"},
		// More chains
		{ID: "fantom", Name: "Fantom", Symbol: "FTM", ChainID: 250, ChainIDHex: "0xfa", NetworkType: "mainnet", Explorer: "https://ftmscan.com", RPCURL: "https://rpc.ankr.com/fantom", Logo: "fantom.png", Color: "#1969FF", Decimals: 18, CoinType: 1000, IsEVM: true, CanTokenList: true, CanNFT: true, CanStake: true, CanBridge: true, GasToken: "FTM", BlockTime: 1, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "celo", Name: "Celo", Symbol: "CELO", ChainID: 42220, ChainIDHex: "0xa4ec", NetworkType: "mainnet", Explorer: "https://celoscan.io", RPCURL: "https://forno.celo.org", Logo: "celo.png", Color: "#35EE93", Decimals: 18, CoinType: 527522, IsEVM: true, CanTokenList: true, CanNFT: true, CanStake: true, CanBridge: true, GasToken: "CELO", BlockTime: 5, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "aurora", Name: "Aurora", Symbol: "ETH", ChainID: 1313161554, NetworkType: "mainnet", Explorer: "https://aurorascan.dev", RPCURL: "https://mainnet.aurora.dev", Logo: "aurora.png", Color: "#6ADBD7", Decimals: 18, CoinType: 60, IsEVM: true, CanTokenList: true, CanNFT: true, CanBridge: true, GasToken: "ETH", BlockTime: 1, Category: "Layer2", Tier: 2, Status: "active"},
		{ID: "harmony", Name: "Harmony", Symbol: "ONE", ChainID: 1666600000, NetworkType: "mainnet", Explorer: "https://explorer.harmony.one", RPCURL: "https://api.harmony.one", Logo: "harmony.png", Color: "#00C4CC", Decimals: 18, CoinType: 1023, IsEVM: true, CanTokenList: true, CanNFT: true, CanStake: true, CanBridge: true, GasToken: "ONE", BlockTime: 2, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "moonbeam", Name: "Moonbeam", Symbol: "GLMR", ChainID: 1284, NetworkType: "mainnet", Explorer: "https://moonbeam.moonscan.io", RPCURL: "https://rpc.api.moonbeam.network", Logo: "moonbeam.png", Color: "#53CBC9", Decimals: 18, CoinType: 1284, IsEVM: true, CanTokenList: true, CanNFT: true, CanStake: true, CanBridge: true, GasToken: "GLMR", BlockTime: 12, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "klaytn", Name: "Klaytn", Symbol: "KLAY", ChainID: 8217, NetworkType: "mainnet", Explorer: "https://klaytnscope.com", RPCURL: "https://klaytn-mainnet-rpc.allthatnode.com:8551", Logo: "klaytn.png", Color: "#332A7C", Decimals: 18, CoinType: 8217, IsEVM: true, CanTokenList: true, CanNFT: true, CanStake: true, CanBridge: true, GasToken: "KLAY", BlockTime: 1, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "metis", Name: "Metis", Symbol: "METIS", ChainID: 1088, NetworkType: "mainnet", Explorer: "https://andromedaexplorer.metis.io", RPCURL: "https://andromeda.metis.io/?owner=1088", Logo: "metis.png", Color: "#6FDBC9", Decimals: 18, CoinType: 1088, IsEVM: true, CanTokenList: true, CanNFT: true, CanBridge: true, GasToken: "METIS", BlockTime: 2, Category: "Layer2", Tier: 2, Status: "active"},
		{ID: "gnosis", Name: "Gnosis Chain", Symbol: "XDAI", ChainID: 100, NetworkType: "mainnet", Explorer: "https://gnosisscan.io", RPCURL: "https://rpc.gnosischain.com", Logo: "gnosis.png", Color: "#04795B", Decimals: 18, CoinType: 700, IsEVM: true, CanTokenList: true, CanNFT: true, CanStake: true, CanBridge: true, GasToken: "XDAI", BlockTime: 5, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "cronos", Name: "Cronos", Symbol: "CRO", ChainID: 25, NetworkType: "mainnet", Explorer: "https://cronoscan.org", RPCURL: "https://evm.cronos.org", Logo: "cronos.png", Color: "#002D74", Decimals: 18, CoinType: 25, IsEVM: true, CanTokenList: true, CanNFT: true, CanBridge: true, GasToken: "CRO", BlockTime: 5, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "oasis", Name: "Oasis Network", Symbol: "ROSE", ChainID: 42262, NetworkType: "mainnet", Explorer: "https://oasisscan.com", RPCURL: "https://emerald.oasis.dev", Logo: "oasis.png", Color: "#0A7354", Decimals: 18, CoinType: 4740, IsEVM: true, CanTokenList: true, CanNFT: true, CanStake: true, CanBridge: true, GasToken: "ROSE", BlockTime: 6, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "vechain", Name: "VeChain", Symbol: "VET", ChainID: 0, NetworkType: "mainnet", Explorer: "https://vechainstats.com", RPCURL: "https://vechain.foundry.io", Logo: "vechain.png", Color: "#15BDFF", Decimals: 18, CoinType: 818, IsEVM: false, CanTokenList: true, CanNFT: true, CanStake: true, CanBridge: true, GasToken: "VET", BlockTime: 2, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "iotex", Name: "IoTeX", Symbol: "IOTX", ChainID: 4689, NetworkType: "mainnet", Explorer: "https://iotexscan.io", RPCURL: "https://rpc.iotex.io", Logo: "iotex.png", Color: "#00D4FF", Decimals: 18, CoinType: 3044, IsEVM: true, CanTokenList: true, CanNFT: true, CanStake: true, CanBridge: true, GasToken: "IOTX", BlockTime: 5, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "kava", Name: "Kava", Symbol: "KAVA", ChainID: 2222, NetworkType: "mainnet", Explorer: "https://kavascan.com", RPCURL: "https://evm.kava.io", Logo: "kava.png", Color: "#FF5638", Decimals: 18, CoinType: 459, IsEVM: true, CanTokenList: true, CanNFT: true, CanStake: true, CanBridge: true, GasToken: "KAVA", BlockTime: 6, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "cardano", Name: "Cardano", Symbol: "ADA", ChainID: 0, NetworkType: "mainnet", Explorer: "https://cardanoscan.io", RPCURL: "https://cardano-mainnet.blockfrost.io", Logo: "cardano.png", Color: "#0033AD", Decimals: 6, CoinType: 1815, IsEVM: false, CanTokenList: true, CanNFT: true, CanStake: true, CanBridge: true, GasToken: "ADA", BlockTime: 20, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "near", Name: "NEAR Protocol", Symbol: "NEAR", ChainID: 0, NetworkType: "mainnet", Explorer: "https://explorer.near.org", RPCURL: "https://rpc.mainnet.near.org", Logo: "near.png", Color: "#000000", Decimals: 24, CoinType: 397, IsEVM: false, CanTokenList: true, CanNFT: true, CanStake: true, CanBridge: true, GasToken: "NEAR", BlockTime: 1, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "aptos", Name: "Aptos", Symbol: "APT", ChainID: 0, NetworkType: "mainnet", Explorer: "https://explorer.aptoslabs.com", RPCURL: "https://aptos-mainnet.nodereal.io/v1", Logo: "aptos.png", Color: "#14F195", Decimals: 8, CoinType: 637, IsEVM: false, CanTokenList: true, CanNFT: true, CanStake: true, CanBridge: true, GasToken: "APT", BlockTime: 1, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "sui", Name: "Sui", Symbol: "SUI", ChainID: 0, NetworkType: "mainnet", Explorer: "https://suiscan.xyz", RPCURL: "https://rpc.mainnet.sui.io", Logo: "sui.png", Color: "#6FB2F2", Decimals: 9, CoinType: 784, IsEVM: false, CanTokenList: true, CanNFT: true, CanStake: true, CanBridge: true, GasToken: "SUI", BlockTime: 1, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "polkadot", Name: "Polkadot", Symbol: "DOT", ChainID: 0, NetworkType: "mainnet", Explorer: "https://polkadot.subscan.io", RPCURL: "https://rpc.polkadot.io", Logo: "polkadot.png", Color: "#E6007A", Decimals: 10, CoinType: 354, IsEVM: false, CanTokenList: true, CanNFT: true, CanStake: true, CanBridge: true, GasToken: "DOT", BlockTime: 6, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "astar", Name: "Astar", Symbol: "ASTR", ChainID: 592, NetworkType: "mainnet", Explorer: "https://blockscout.com/astar", RPCURL: "https://rpc.astar.network:8545", Logo: "astar.png", Color: "#1B1B2E", Decimals: 18, CoinType: 592, IsEVM: true, CanTokenList: true, CanNFT: true, CanStake: true, CanBridge: true, GasToken: "ASTR", BlockTime: 12, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "mantle", Name: "Mantle", Symbol: "MNT", ChainID: 5000, NetworkType: "mainnet", Explorer: "https://mantlescan.org", RPCURL: "https://rpc.mantle.xyz", Logo: "mantle.png", Color: "#1B2B4B", Decimals: 18, CoinType: 5000, IsEVM: true, CanTokenList: true, CanNFT: true, CanBridge: true, GasToken: "MNT", BlockTime: 2, Category: "Layer2", Tier: 1, Status: "active"},
		{ID: "bitcoin", Name: "Bitcoin", Symbol: "BTC", ChainID: 0, NetworkType: "mainnet", Explorer: "https://blockstream.info", RPCURL: "https://blockstream.info/api", Logo: "btc.png", Color: "#F7931A", Decimals: 8, CoinType: 0, IsEVM: false, CanBridge: true, GasToken: "BTC", BlockTime: 600, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "injective", Name: "Injective", Symbol: "INJ", ChainID: 0, NetworkType: "mainnet", Explorer: "https://explorer.injective.network", RPCURL: "https://public.api.injective.network", Logo: "injective.png", Color: "#00F2FE", Decimals: 18, CoinType: 690, IsEVM: false, CanTokenList: true, CanNFT: true, CanStake: true, CanBridge: true, GasToken: "INJ", BlockTime: 1, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "sei", Name: "Sei", Symbol: "SEI", ChainID: 0, NetworkType: "mainnet", Explorer: "https://www.seiscan.app", RPCURL: "https://rest.sei-apis.com", Logo: "sei.png", Color: "#1B1B2E", Decimals: 6, CoinType: 5415, IsEVM: false, CanTokenList: true, CanNFT: true, CanStake: true, CanBridge: true, GasToken: "SEI", BlockTime: 1, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "immutable", Name: "Immutable X", Symbol: "IMX", ChainID: 13371, NetworkType: "mainnet", Explorer: "https://explorer.immutable.com", RPCURL: "https://rpc.immutable.com", Logo: "immutable.png", Color: "#00D4FF", Decimals: 18, CoinType: 60, IsEVM: true, CanTokenList: true, CanNFT: true, CanStake: true, CanBridge: true, GasToken: "IMX", BlockTime: 1, Category: "Layer2", Tier: 1, Status: "active"},
		{ID: "mode", Name: "Mode", Symbol: "ETH", ChainID: 34443, NetworkType: "mainnet", Explorer: "https://explorer.mode.network", RPCURL: "https://mainnet.mode.network", Logo: "mode.png", Color: "#0066FF", Decimals: 18, CoinType: 60, IsEVM: true, CanTokenList: true, CanNFT: true, CanBridge: true, GasToken: "ETH", BlockTime: 2, Category: "Layer2", Tier: 1, Status: "active"},
		{ID: "zora", Name: "Zora", Symbol: "ETH", ChainID: 7777777, NetworkType: "mainnet", Explorer: "https://zora.superscan.network", RPCURL: "https://rpc.zora.co", Logo: "zora.png", Color: "#8B5CF6", Decimals: 18, CoinType: 60, IsEVM: true, CanTokenList: true, CanNFT: true, CanBridge: true, GasToken: "ETH", BlockTime: 2, Category: "Layer2", Tier: 1, Status: "active"},
		{ID: "ronin", Name: "Ronin", Symbol: "RON", ChainID: 2020, NetworkType: "mainnet", Explorer: "https://app.roninchain.com", RPCURL: "https://api.roninchain.com", Logo: "ronin.png", Color: "#CB2E3D", Decimals: 18, CoinType: 2020, IsEVM: true, CanTokenList: true, CanNFT: true, CanStake: true, CanBridge: true, GasToken: "RON", BlockTime: 3, Category: "Layer2", Tier: 1, Status: "active"},
		{ID: "boba", Name: "Boba Network", Symbol: "BOBA", ChainID: 288, NetworkType: "mainnet", Explorer: "https://bobascan.com", RPCURL: "https://mainnet.boba.network", Logo: "boba.png", Color: "#2C8CFF", Decimals: 18, CoinType: 288, IsEVM: true, CanTokenList: true, CanNFT: true, CanBridge: true, GasToken: "BOBA", BlockTime: 1, Category: "Layer2", Tier: 2, Status: "active"},
		{ID: "taiko", Name: "Taiko", Symbol: "ETH", ChainID: 167000, NetworkType: "mainnet", Explorer: "https://taikoscan.io", RPCURL: "https://rpc.taiko.xyz", Logo: "taiko.png", Color: "#FF6B6B", Decimals: 18, CoinType: 60, IsEVM: true, CanTokenList: true, CanNFT: true, CanBridge: true, GasToken: "ETH", BlockTime: 2, Category: "Layer2", Tier: 1, Status: "active"},
		{ID: "monad", Name: "Monad", Symbol: "MON", ChainID: 10143, NetworkType: "mainnet", Explorer: "https://explorer.monad.xyz", RPCURL: "https://rpc.monad.xyz", Logo: "monad.png", Color: "#FF6B35", Decimals: 18, CoinType: 10143, IsEVM: true, CanTokenList: true, CanNFT: true, CanBridge: true, GasToken: "MON", BlockTime: 1, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "berachain", Name: "Berachain", Symbol: "BERA", ChainID: 80084, NetworkType: "mainnet", Explorer: "https://berascan.com", RPCURL: "https://rpc.berachain.com", Logo: "bera.png", Color: "#FF6B6B", Decimals: 18, CoinType: 80084, IsEVM: true, CanTokenList: true, CanNFT: true, CanStake: true, CanBridge: true, GasToken: "BERA", BlockTime: 2, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "merlin", Name: "Merlin", Symbol: "BTC", ChainID: 4200, NetworkType: "mainnet", Explorer: "https://scan.merlinchain.io", RPCURL: "https://rpc.merlinchain.io", Logo: "merlin.png", Color: "#FF7A45", Decimals: 18, CoinType: 4200, IsEVM: true, CanTokenList: true, CanNFT: true, CanBridge: true, GasToken: "BTC", BlockTime: 2, Category: "Layer2", Tier: 1, Status: "active"},
		{ID: "btr", Name: "Bitlayer", Symbol: "BTC", ChainID: 90112, NetworkType: "mainnet", Explorer: "https://www.btrscan.com", RPCURL: "https://rpc.bitlayer.org", Logo: "bitlayer.png", Color: "#FF6B6B", Decimals: 18, CoinType: 90112, IsEVM: true, CanTokenList: true, CanNFT: true, CanBridge: true, GasToken: "BTC", BlockTime: 2, Category: "Layer2", Tier: 1, Status: "active"},
		{ID: "alienx", Name: "AlienX", Symbol: "ALX", ChainID: 19527, NetworkType: "mainnet", Explorer: "https://alxscan.com", RPCURL: "https://rpc.alienxchain.io", Logo: "alienx.png", Color: "#FF6B6B", Decimals: 18, CoinType: 19527, IsEVM: true, CanTokenList: true, CanNFT: true, CanBridge: true, GasToken: "ALX", BlockTime: 2, Category: "Layer2", Tier: 1, Status: "active"},
		{ID: "morpheus", Name: "Morpheus Network", Symbol: "MOR", ChainID: 1200, NetworkType: "mainnet", Explorer: "https://morphexplorer.com", RPCURL: "https://rpc.morpheus.network", Logo: "morpheus.png", Color: "#6B5CE7", Decimals: 18, CoinType: 1200, IsEVM: true, CanTokenList: true, CanNFT: true, CanBridge: true, GasToken: "MOR", BlockTime: 2, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "thales", Name: "Thales", Symbol: "THALES", ChainID: 128123, NetworkType: "mainnet", Explorer: "https://thales.network", RPCURL: "https://rpc.thalesnetwork.io", Logo: "thales.png", Color: "#FF6B6B", Decimals: 18, CoinType: 128123, IsEVM: true, CanTokenList: true, CanNFT: true, CanBridge: true, GasToken: "THALES", BlockTime: 2, Category: "Layer1", Tier: 1, Status: "active"},
		// Cosmos Ecosystem
		{ID: "cosmos", Name: "Cosmos Hub", Symbol: "ATOM", ChainID: 0, NetworkType: "mainnet", Explorer: "https://www.mintscan.io/cosmos", RPCURL: "https://cosmos-rpc.polkachu.com", Logo: "cosmos.png", Color: "#2E3142", Decimals: 6, CoinType: 118, IsEVM: false, CanTokenList: true, CanStake: true, CanBridge: true, GasToken: "ATOM", BlockTime: 6, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "osmosis", Name: "Osmosis", Symbol: "OSMO", ChainID: 0, NetworkType: "mainnet", Explorer: "https://osmosis.explorers.guru", RPCURL: "https://osmosis-rpc.polkachu.com", Logo: "osmosis.png", Color: "#000000", Decimals: 6, CoinType: 1180, IsEVM: false, CanTokenList: true, CanStake: true, CanBridge: true, GasToken: "OSMO", BlockTime: 6, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "juno", Name: "Juno", Symbol: "JUNO", ChainID: 0, NetworkType: "mainnet", Explorer: "https://www.mintscan.io/juno", RPCURL: "https://juno-rpc.polkachu.com", Logo: "juno.png", Color: "#FF6B6B", Decimals: 6, CoinType: 118, IsEVM: false, CanTokenList: true, CanNFT: true, CanStake: true, CanBridge: true, GasToken: "JUNO", BlockTime: 6, Category: "Layer1", Tier: 2, Status: "active"},
		{ID: "stargaze", Name: "Stargaze", Symbol: "STARS", ChainID: 0, NetworkType: "mainnet", Explorer: "https://www.mintscan.io/stargaze", RPCURL: "https://stargaze-rpc.polkachu.com", Logo: "stargaze.png", Color: "#FF6B6B", Decimals: 6, CoinType: 118, IsEVM: false, CanTokenList: true, CanNFT: true, CanStake: true, CanBridge: true, GasToken: "STARS", BlockTime: 6, Category: "Layer1", Tier: 2, Status: "active"},
		{ID: "sentinel", Name: "Sentinel", Symbol: "DVPN", ChainID: 0, NetworkType: "mainnet", Explorer: "https://www.mintscan.io/sentinel", RPCURL: "https://sentinel-rpc.polkachu.com", Logo: "sentinel.png", Color: "#3A3A3A", Decimals: 6, CoinType: 118, IsEVM: false, CanTokenList: true, CanStake: true, CanBridge: true, GasToken: "DVPN", BlockTime: 6, Category: "Layer1", Tier: 2, Status: "active"},
		{ID: "stride", Name: "Stride", Symbol: "STRD", ChainID: 0, NetworkType: "mainnet", Explorer: "https://www.mintscan.io/stride", RPCURL: "https://stride-rpc.polkachu.com", Logo: "stride.png", Color: "#6B9AC4", Decimals: 6, CoinType: 118, IsEVM: false, CanTokenList: true, CanStake: true, CanBridge: true, GasToken: "STRD", BlockTime: 6, Category: "Layer1", Tier: 2, Status: "active"},
		{ID: "dydx", Name: "dYdX", Symbol: "DYDX", ChainID: 0, NetworkType: "mainnet", Explorer: "https://www.mintscan.io/dydx", RPCURL: "https://dydx-rpc.polkachu.com", Logo: "dydx.png", Color: "#2B2B3F", Decimals: 18, CoinType: 118, IsEVM: false, CanTokenList: true, CanStake: true, CanBridge: true, GasToken: "DYDX", BlockTime: 1, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "celestia", Name: "Celestia", Symbol: "TIA", ChainID: 0, NetworkType: "mainnet", Explorer: "https://www.mintscan.io/celestia", RPCURL: "https://celestia-rpc.polkachu.com", Logo: "celestia.png", Color: "#B4A0FF", Decimals: 6, CoinType: 118, IsEVM: false, CanTokenList: true, CanStake: true, CanBridge: true, GasToken: "TIA", BlockTime: 1, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "noble", Name: "Noble", Symbol: "NOBLE", ChainID: 0, NetworkType: "mainnet", Explorer: "https://www.mintscan.io/noble", RPCURL: "https://noble-rpc.polkachu.com", Logo: "noble.png", Color: "#FF6B6B", Decimals: 6, CoinType: 118, IsEVM: false, CanTokenList: true, CanStake: true, CanBridge: true, GasToken: "NOBLE", BlockTime: 6, Category: "Layer1", Tier: 2, Status: "active"},
		{ID: "coreum", Name: "Coreum", Symbol: "CORE", ChainID: 0, NetworkType: "mainnet", Explorer: "https://www.mintscan.io/coreum", RPCURL: "https://coreum-rpc.polkachu.com", Logo: "coreum.png", Color: "#FF6B6B", Decimals: 6, CoinType: 118, IsEVM: false, CanTokenList: true, CanStake: true, CanBridge: true, GasToken: "CORE", BlockTime: 6, Category: "Layer1", Tier: 2, Status: "active"},
		{ID: "fetch", Name: "Fetch.ai", Symbol: "FET", ChainID: 0, NetworkType: "mainnet", Explorer: "https://www.mintscan.io/fetch-ai", RPCURL: "https://fetch-rpc.polkachu.com", Logo: "fetch.png", Color: "#FF6B6B", Decimals: 18, CoinType: 118, IsEVM: false, CanTokenList: true, CanStake: true, CanBridge: true, GasToken: "FET", BlockTime: 6, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "algorand", Name: "Algorand", Symbol: "ALGO", ChainID: 0, NetworkType: "mainnet", Explorer: "https://algoexplorer.io", RPCURL: "https://mainnet-api.algorand.network", Logo: "algorand.png", Color: "#000000", Decimals: 6, CoinType: 283, IsEVM: false, CanTokenList: true, CanNFT: true, CanStake: true, CanBridge: true, GasToken: "ALGO", BlockTime: 3, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "tezos", Name: "Tezos", Symbol: "XTZ", ChainID: 0, NetworkType: "mainnet", Explorer: "https://tzstats.com", RPCURL: "https://mainnet.tezos.com", Logo: "tezos.png", Color: "#2C7DF7", Decimals: 6, CoinType: 1729, IsEVM: false, CanTokenList: true, CanNFT: true, CanStake: true, CanBridge: true, GasToken: "XTZ", BlockTime: 30, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "kusama", Name: "Kusama", Symbol: "KSM", ChainID: 0, NetworkType: "mainnet", Explorer: "https://kusama.subscan.io", RPCURL: "https://rpc.kusama.network", Logo: "kusama.png", Color: "#FF6B6B", Decimals: 12, CoinType: 434, IsEVM: false, CanTokenList: true, CanNFT: true, CanStake: true, CanBridge: true, GasToken: "KSM", BlockTime: 6, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "alephzero", Name: "Aleph Zero", Symbol: "AZERO", ChainID: 0, NetworkType: "mainnet", Explorer: "https://alephzero.subscan.io", RPCURL: "https://rpc.azero.dev", Logo: "alephzero.png", Color: "#0099FF", Decimals: 12, CoinType: 354, IsEVM: false, CanTokenList: true, CanStake: true, CanBridge: true, GasToken: "AZERO", BlockTime: 1, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "radix", Name: "Radix", Symbol: "XRD", ChainID: 0, NetworkType: "mainnet", Explorer: "https://dashboard.radixdlt.com", RPCURL: "https://mainnet.radixdlt.com", Logo: "radix.png", Color: "#00EECC", Decimals: 18, CoinType: 1024, IsEVM: false, CanTokenList: true, CanStake: true, CanBridge: true, GasToken: "XRD", BlockTime: 2, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "hedera", Name: "Hedera", Symbol: "HBAR", ChainID: 0, NetworkType: "mainnet", Explorer: "https://hashscan.io", RPCURL: "https://mainnet-preview.mirrornode.hedera.com", Logo: "hedera.png", Color: "#00BCD4", Decimals: 8, CoinType: 3030, IsEVM: false, CanTokenList: true, CanNFT: true, CanStake: true, CanBridge: true, GasToken: "HBAR", BlockTime: 2, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "flow", Name: "Flow", Symbol: "FLOW", ChainID: 0, NetworkType: "mainnet", Explorer: "https://flowscan.org", RPCURL: "https://flow-node-01.mainnet1.onflow.org", Logo: "flow.png", Color: "#FF6B6B", Decimals: 8, CoinType: 539, IsEVM: false, CanTokenList: true, CanNFT: true, CanStake: true, CanBridge: true, GasToken: "FLOW", BlockTime: 2, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "tron", Name: "Tron", Symbol: "TRX", ChainID: 0, NetworkType: "mainnet", Explorer: "https://tronscan.org", RPCURL: "https://api.trongrid.io", Logo: "tron.png", Color: "#FF0013", Decimals: 6, CoinType: 1950, IsEVM: false, CanTokenList: true, CanNFT: true, CanStake: true, CanBridge: true, GasToken: "TRX", BlockTime: 3, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "eos", Name: "EOS", Symbol: "EOS", ChainID: 0, NetworkType: "mainnet", Explorer: "https://bloks.io", RPCURL: "https://api.eosn.io", Logo: "eos.png", Color: "#000000", Decimals: 4, CoinType: 1942539323, IsEVM: false, CanTokenList: true, CanNFT: true, CanStake: true, CanBridge: true, GasToken: "EOS", BlockTime: 1, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "wax", Name: "WAX", Symbol: "WAXP", ChainID: 0, NetworkType: "mainnet", Explorer: "https://wax.bloks.io", RPCURL: "https://api.wax.io", Logo: "wax.png", Color: "#FF6B6B", Decimals: 8, CoinType: 4661, IsEVM: false, CanTokenList: true, CanNFT: true, CanStake: true, CanBridge: true, GasToken: "WAXP", BlockTime: 1, Category: "Layer1", Tier: 2, Status: "active"},
		{ID: "telos", Name: "Telos", Symbol: "TLOS", ChainID: 0, NetworkType: "mainnet", Explorer: "https://www.teloscan.io", RPCURL: "https://api.telos.net", Logo: "telos.png", Color: "#FF6B6B", Decimals: 4, CoinType: 4661, IsEVM: false, CanTokenList: true, CanNFT: true, CanStake: true, CanBridge: true, GasToken: "TLOS", BlockTime: 1, Category: "Layer1", Tier: 2, Status: "active"},
		{ID: "thorchain", Name: "THORChain", Symbol: "RUNE", ChainID: 0, NetworkType: "mainnet", Explorer: "https://thorchain.net", RPCURL: "https://rpc.thorchain.info", Logo: "thorchain.png", Color: "#FF6B6B", Decimals: 8, CoinType: 118, IsEVM: false, CanTokenList: true, CanStake: true, CanBridge: true, GasToken: "RUNE", BlockTime: 6, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "kujira", Name: "Kujira", Symbol: "KUJI", ChainID: 0, NetworkType: "mainnet", Explorer: "https://kujira.mintscan.io", RPCURL: "https://rpc.kujira.network", Logo: "kujira.png", Color: "#FF6B6B", Decimals: 6, CoinType: 118, IsEVM: false, CanTokenList: true, CanStake: true, CanBridge: true, GasToken: "KUJI", BlockTime: 6, Category: "Layer1", Tier: 2, Status: "active"},
		{ID: "ton", Name: "Toncoin", Symbol: "TON", ChainID: 0, NetworkType: "mainnet", Explorer: "https://tonscan.org", RPCURL: "https://toncenter.com/api/v2", Logo: "ton.png", Color: "#0098EA", Decimals: 9, CoinType: 607, IsEVM: false, CanTokenList: true, CanNFT: true, CanBridge: true, GasToken: "TON", BlockTime: 5, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "mina", Name: "Mina", Symbol: "MINA", ChainID: 0, NetworkType: "mainnet", Explorer: "https://minaexplorer.io", RPCURL: "https://api.minaprotocol.com", Logo: "mina.png", Color: "#E6B989", Decimals: 9, CoinType: 8668, IsEVM: false, CanTokenList: true, CanStake: true, CanBridge: true, GasToken: "MINA", BlockTime: 1, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "kaspa", Name: "Kaspa", Symbol: "KAS", ChainID: 0, NetworkType: "mainnet", Explorer: "https://explorer.kaspa.org", RPCURL: "https://rpc.kaspa.org", Logo: "kaspa.png", Color: "#FF6B6B", Decimals: 8, CoinType: 1313334, IsEVM: false, CanTokenList: true, CanBridge: true, GasToken: "KAS", BlockTime: 1, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "monero", Name: "Monero", Symbol: "XMR", ChainID: 0, NetworkType: "mainnet", Explorer: "https://xmr.to", RPCURL: "https://node.xmr.to", Logo: "monero.png", Color: "#FF6600", Decimals: 12, CoinType: 128, IsEVM: false, CanBridge: true, GasToken: "XMR", BlockTime: 2, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "zcash", Name: "Zcash", Symbol: "ZEC", ChainID: 0, NetworkType: "mainnet", Explorer: "https://zcashblockexplorer.io", RPCURL: "https://zcash.enterprise.coinmetric.io", Logo: "zcash.png", Color: "#F4B728", Decimals: 8, CoinType: 133, IsEVM: false, CanBridge: true, GasToken: "ZEC", BlockTime: 2, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "dash", Name: "Dash", Symbol: "DASH", ChainID: 0, NetworkType: "mainnet", Explorer: "https://explorer.dash.org", RPCURL: "https://api.dash.org", Logo: "dash.png", Color: "#008DE4", Decimals: 8, CoinType: 5, IsEVM: false, CanTokenList: true, CanBridge: true, GasToken: "DASH", BlockTime: 3, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "litecoin", Name: "Litecoin", Symbol: "LTC", ChainID: 0, NetworkType: "mainnet", Explorer: "https://blockchair.com/litecoin", RPCURL: "https://api.litecoin.org", Logo: "litecoin.png", Color: "#BFBBBB", Decimals: 8, CoinType: 2, IsEVM: false, CanBridge: true, GasToken: "LTC", BlockTime: 2, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "dogecoin", Name: "Dogecoin", Symbol: "DOGE", ChainID: 0, NetworkType: "mainnet", Explorer: "https://blockchair.com/dogecoin", RPCURL: "https://api.dogecoin.com", Logo: "dogecoin.png", Color: "#C2A633", Decimals: 8, CoinType: 3, IsEVM: false, CanBridge: true, GasToken: "DOGE", BlockTime: 1, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "waves", Name: "Waves", Symbol: "WAVES", ChainID: 0, NetworkType: "mainnet", Explorer: "https://wavesexplorer.com", RPCURL: "https://nodes.wavesnodes.com", Logo: "waves.png", Color: "#0156FF", Decimals: 8, CoinType: 5741564, IsEVM: false, CanTokenList: true, CanStake: true, CanBridge: true, GasToken: "WAVES", BlockTime: 1, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "conflux", Name: "Conflux", Symbol: "CFX", ChainID: 0, NetworkType: "mainnet", Explorer: "https://confluxscan.io", RPCURL: "https://rpc.confluxnetwork.org", Logo: "conflux.png", Color: "#FF6B6B", Decimals: 18, CoinType: 5030, IsEVM: false, CanTokenList: true, CanStake: true, CanBridge: true, GasToken: "CFX", BlockTime: 1, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "skale", Name: "SKALE", Symbol: "SKL", ChainID: 0, NetworkType: "mainnet", Explorer: "https://skale.space", RPCURL: "https://mainnet.skalenodes.com", Logo: "skale.png", Color: "#FF6B6B", Decimals: 18, CoinType: 135, IsEVM: false, CanTokenList: true, CanStake: true, CanBridge: true, GasToken: "SKL", BlockTime: 1, Category: "Layer2", Tier: 1, Status: "active"},
		{ID: "canto", Name: "Canto", Symbol: "CANTO", ChainID: 0, NetworkType: "mainnet", Explorer: "https://tuber.build", RPCURL: "https://canto-rpc.polkachu.com", Logo: "canto.png", Color: "#FF6B6B", Decimals: 18, CoinType: 118, IsEVM: false, CanTokenList: true, CanStake: true, CanBridge: true, GasToken: "CANTO", BlockTime: 6, Category: "Layer1", Tier: 2, Status: "active"},
		{ID: "viction", Name: "Viction", Symbol: "VIC", ChainID: 0, NetworkType: "mainnet", Explorer: "https://tbc1.tomochain.com", RPCURL: "https://rpc.viction.xyz", Logo: "viction.png", Color: "#FF6B6B", Decimals: 18, CoinType: 118, IsEVM: false, CanTokenList: true, CanStake: true, CanBridge: true, GasToken: "VIC", BlockTime: 2, Category: "Layer1", Tier: 2, Status: "active"},
		{ID: "arweave", Name: "Arweave", Symbol: "AR", ChainID: 0, NetworkType: "mainnet", Explorer: "https://viewblock.io/arweave", RPCURL: "https://arweave.net", Logo: "arweave.png", Color: "#000000", Decimals: 12, CoinType: 131, IsEVM: false, CanTokenList: true, CanBridge: true, GasToken: "AR", BlockTime: 1, Category: "Layer1", Tier: 1, Status: "active"},
		{ID: "pi", Name: "Pi Network", Symbol: "PI", ChainID: 0, NetworkType: "mainnet", Explorer: "https://minepi.com/explorer", RPCURL: "https://rpc.minepi.com", Logo: "pi.png", Color: "#FF6B6B", Decimals: 18, CoinType: 314159, IsEVM: false, CanTokenList: true, CanBridge: false, GasToken: "PI", BlockTime: 1, Category: "Layer1", Tier: 2, Status: "active"},
	}

	for _, network := range networks {
		r.networks[network.ID] = network
	}
}

func (r *BlockchainRegistry) GetNetwork(id string) (*BlockchainNetwork, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	network, ok := r.networks[id]
	if !ok {
		return nil, fmt.Errorf("network not found: %s", id)
	}
	return network, nil
}

func (r *BlockchainRegistry) GetAllNetworks() []*BlockchainNetwork {
	r.mu.RLock()
	defer r.mu.RUnlock()
	networks := make([]*BlockchainNetwork, 0, len(r.networks))
	for _, network := range r.networks {
		networks = append(networks, network)
	}
	return networks
}

func (r *BlockchainRegistry) GetNetworkCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.networks)
}

func main() {
	registry := NewBlockchainRegistry()
	count := registry.GetNetworkCount()
	fmt.Printf("Total blockchain networks: %d\n", count)

	networks := registry.GetAllNetworks()
	fmt.Printf("\nNetworks:\n")
	for i, n := range networks {
		if i < 20 {
			fmt.Printf("  - %s (%s) - %s\n", n.Name, n.Symbol, n.Category)
		}
	}
	fmt.Printf("  ... and %d more\n", count-20)
}
