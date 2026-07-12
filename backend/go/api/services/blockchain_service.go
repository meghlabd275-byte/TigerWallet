package services

import (
	"context"
	"errors"
	"sync"

	"tigerwallet/backend/go/api/models"
)

// BlockchainService handles blockchain operations
type BlockchainService struct {
	mu         sync.RWMutex
	blockchains map[string]*models.Blockchain
}

var (
	blockchainInstance *BlockchainService
	blockchainOnce    sync.Once
)

func NewBlockchainService() *BlockchainService {
	blockchainOnce.Do(func() {
		blockchainInstance = &BlockchainService{
			blockchains: make(map[string]*models.Blockchain),
		}
		blockchainInstance.initializeDefaultBlockchains()
	})
	return blockchainInstance
}

func (s *BlockchainService) initializeDefaultBlockchains() {
	defaultChains := []*models.Blockchain{
		{ID: "ethereum", Name: "Ethereum", Symbol: "ETH", ChainID: 1, Type: "evm", RPCURL: "https://eth.llamarpc.com", ExplorerURL: "https://etherscan.io", LogoURL: "https://assets.coingecko.com/coins/images/279/small/ethereum.png", IsActive: true, IsTestnet: false, Decimals: 18, GasToken: "ETH", AvgBlockTime: 12, MaxGasPrice: 500000000000, SupportsEIP1559: true},
		{ID: "bsc", Name: "BNB Smart Chain", Symbol: "BNB", ChainID: 56, Type: "evm", RPCURL: "https://bsc-dataseed.binance.org", ExplorerURL: "https://bscscan.com", LogoURL: "https://assets.coingecko.com/coins/images/825/small/bnb-icon2_2x.png", IsActive: true, IsTestnet: false, Decimals: 18, GasToken: "BNB", AvgBlockTime: 3, MaxGasPrice: 100000000000, SupportsEIP1559: true},
		{ID: "polygon", Name: "Polygon", Symbol: "MATIC", ChainID: 137, Type: "evm", RPCURL: "https://polygon-rpc.com", ExplorerURL: "https://polygonscan.com", LogoURL: "https://assets.coingecko.com/coins/images/4713/small/matic-token-icon.png", IsActive: true, IsTestnet: false, Decimals: 18, GasToken: "MATIC", AvgBlockTime: 2, MaxGasPrice: 50000000000, SupportsEIP1559: true},
		{ID: "arbitrum", Name: "Arbitrum One", Symbol: "ETH", ChainID: 42161, Type: "evm", RPCURL: "https://arb1.arbitrum.io/rpc", ExplorerURL: "https://arbiscan.io", LogoURL: "https://assets.coingecko.com/coins/images/16547/small/photo_2023-03-29_21.47.00.jpeg", IsActive: true, IsTestnet: false, Decimals: 18, GasToken: "ETH", AvgBlockTime: 0.25, MaxGasPrice: 100000000000, SupportsEIP1559: true},
		{ID: "optimism", Name: "Optimism", Symbol: "ETH", ChainID: 10, Type: "evm", RPCURL: "https://mainnet.optimism.io", ExplorerURL: "https://optimistic.etherscan.io", LogoURL: "https://assets.coingecko.com/coins/images/25244/small/Optimism.png", IsActive: true, IsTestnet: false, Decimals: 18, GasToken: "ETH", AvgBlockTime: 2, MaxGasPrice: 100000000000, SupportsEIP1559: true},
		{ID: "base", Name: "Base", Symbol: "ETH", ChainID: 8453, Type: "evm", RPCURL: "https://mainnet.base.org", ExplorerURL: "https://basescan.org", LogoURL: "https://assets.coingecko.com/coins/images/31054/small/base-dai.jpeg", IsActive: true, IsTestnet: false, Decimals: 18, GasToken: "ETH", AvgBlockTime: 2, MaxGasPrice: 100000000000, SupportsEIP1559: true},
		{ID: "avalanche", Name: "Avalanche C-Chain", Symbol: "AVAX", ChainID: 43114, Type: "evm", RPCURL: "https://api.avax.network/ext/bc/C/rpc", ExplorerURL: "https://snowtrace.io", LogoURL: "https://assets.coingecko.com/coins/images/12559/small/Avalanche_Circle_RedWhite_Trans.png", IsActive: true, IsTestnet: false, Decimals: 18, GasToken: "AVAX", AvgBlockTime: 1, MaxGasPrice: 50000000000, SupportsEIP1559: true},
		{ID: "solana", Name: "Solana", Symbol: "SOL", ChainID: -1, Type: "solana", RPCURL: "https://api.mainnet-beta.solana.com", ExplorerURL: "https://solscan.io", LogoURL: "https://assets.coingecko.com/coins/images/4128/small/solana.png", IsActive: true, IsTestnet: false, Decimals: 9, GasToken: "SOL", AvgBlockTime: 0.4, MaxGasPrice: 100000000, SupportsEIP1559: false},
		{ID: "tron", Name: "TRON", Symbol: "TRX", ChainID: -1, Type: "account", RPCURL: "https://api.trongrid.io", ExplorerURL: "https://tronscan.org", LogoURL: "https://assets.coingecko.com/coins/images/1094/small/tron-logo.png", IsActive: true, IsTestnet: false, Decimals: 6, GasToken: "TRX", AvgBlockTime: 3, MaxGasPrice: 10000000000, SupportsEIP1559: false},
		{ID: "ton", Name: "TON", Symbol: "TON", ChainID: -1, Type: "account", RPCURL: "https://toncenter.com/api/v2", ExplorerURL: "https://tonscan.org", LogoURL: "https://assets.coingecko.com/coins/images/17980/small/ton_symbol.png", IsActive: true, IsTestnet: false, Decimals: 9, GasToken: "TON", AvgBlockTime: 5, MaxGasPrice: 1000000000, SupportsEIP1559: false},
		{ID: "aptos", Name: "Aptos", Symbol: "APT", ChainID: -1, Type: "aptos", RPCURL: "https://fullnode.mainnet.aptoslabs.com", ExplorerURL: "https://aptoscan.com", LogoURL: "https://assets.coingecko.com/coins/images/26455/small/aptos_round.png", IsActive: true, IsTestnet: false, Decimals: 8, GasToken: "APT", AvgBlockTime: 1, MaxGasPrice: 100000000, SupportsEIP1559: false},
		{ID: "sui", Name: "Sui", Symbol: "SUI", ChainID: -1, Type: "sui", RPCURL: "https://fullnode.mainnet.sui.io", ExplorerURL: "https://suiscan.xyz", LogoURL: "https://assets.coingecko.com/coins/images/26375/small/sui_asset.jpeg", IsActive: true, IsTestnet: false, Decimals: 9, GasToken: "SUI", AvgBlockTime: 1, MaxGasPrice: 100000000, SupportsEIP1559: false},
		{ID: "cosmos", Name: "Cosmos Hub", Symbol: "ATOM", ChainID: -1, Type: "cosmos", RPCURL: "https://rpc.cosmos.network", ExplorerURL: "https://ping.pub/cosmos", LogoURL: "https://assets.coingecko.com/coins/images/1481/small/cosmos_hub.png", IsActive: true, IsTestnet: false, Decimals: 6, GasToken: "ATOM", AvgBlockTime: 7, MaxGasPrice: 1000000, SupportsEIP1559: false},
		{ID: "near", Name: "NEAR Protocol", Symbol: "NEAR", ChainID: -1, Type: "account", RPCURL: "https://rpc.mainnet.near.org", ExplorerURL: "https://explorer.near.org", LogoURL: "https://assets.coingecko.com/coins/images/10365/small/near.jpg", IsActive: true, IsTestnet: false, Decimals: 24, GasToken: "NEAR", AvgBlockTime: 1, MaxGasPrice: 1000000000000, SupportsEIP1559: false},
		{ID: "bitcoin", Name: "Bitcoin", Symbol: "BTC", ChainID: -1, Type: "utxo", RPCURL: "https://btc-rpc.allthatblock.com", ExplorerURL: "https://blockstream.info", LogoURL: "https://assets.coingecko.com/coins/images/1/small/bitcoin.png", IsActive: true, IsTestnet: false, Decimals: 8, GasToken: "BTC", AvgBlockTime: 600, MaxGasPrice: 100000000, SupportsEIP1559: false},
		{ID: "dogecoin", Name: "Dogecoin", Symbol: "DOGE", ChainID: -1, Type: "utxo", RPCURL: "https://dogecoin-rpc.allthatblock.com", ExplorerURL: "https://doge.town", LogoURL: "https://assets.coingecko.com/coins/images/5/small/dogecoin.png", IsActive: true, IsTestnet: false, Decimals: 8, GasToken: "DOGE", AvgBlockTime: 60, MaxGasPrice: 10000000000, SupportsEIP1559: false},
		{ID: "litecoin", Name: "Litecoin", Symbol: "LTC", ChainID: -1, Type: "utxo", RPCURL: "https://litecoin-rpc.allthatblock.com", ExplorerURL: "https://blockchair.com/litecoin", LogoURL: "https://assets.coingecko.com/coins/images/2/small/litecoin.png", IsActive: true, IsTestnet: false, Decimals: 8, GasToken: "LTC", AvgBlockTime: 150, MaxGasPrice: 100000000, SupportsEIP1559: false},
		{ID: "linea", Name: "Linea", Symbol: "ETH", ChainID: 59144, Type: "evm", RPCURL: "https://rpc.linea.build", ExplorerURL: "https://lineascan.build", LogoURL: "https://assets.coingecko.com/coins/images/28689/small/linea.png", IsActive: true, IsTestnet: false, Decimals: 18, GasToken: "ETH", AvgBlockTime: 2, MaxGasPrice: 100000000000, SupportsEIP1559: true},
		{ID: "scroll", Name: "Scroll", Symbol: "ETH", ChainID: 534352, Type: "evm", RPCURL: "https://rpc.scroll.io", ExplorerURL: "https://scrollscan.com", LogoURL: "https://assets.coingecko.com/coins/images/29577/small/scroll.png", IsActive: true, IsTestnet: false, Decimals: 18, GasToken: "ETH", AvgBlockTime: 3, MaxGasPrice: 100000000000, SupportsEIP1559: true},
		{ID: "zksync", Name: "zkSync Era", Symbol: "ETH", ChainID: 324, Type: "evm", RPCURL: "https://mainnet.era.zksync.io", ExplorerURL: "https://explorer.zksync.io", LogoURL: "https://assets.coingecko.com/coins/images/48689/small/sync.png", IsActive: true, IsTestnet: false, Decimals: 18, GasToken: "ETH", AvgBlockTime: 1, MaxGasPrice: 100000000000, SupportsEIP1559: true},
		{ID: "blast", Name: "Blast", Symbol: "ETH", ChainID: 81457, Type: "evm", RPCURL: "https://blastl2-mainnet-public.united.blast.io", ExplorerURL: "https://blastscan.io", LogoURL: "https://assets.coingecko.com/coins/images/35537/small/blast.png", IsActive: true, IsTestnet: false, Decimals: 18, GasToken: "ETH", AvgBlockTime: 2, MaxGasPrice: 100000000000, SupportsEIP1559: true},
		{ID: "mantle", Name: "Mantle", Symbol: "MNT", ChainID: 5000, Type: "evm", RPCURL: "https://rpc.mantle.xyz", ExplorerURL: "https://mantlescan.info", LogoURL: "https://assets.coingecko.com/coins/images/29631/small/mantle.png", IsActive: true, IsTestnet: false, Decimals: 18, GasToken: "MNT", AvgBlockTime: 2, MaxGasPrice: 10000000000, SupportsEIP1559: true},
		{ID: "fantom", Name: "Fantom", Symbol: "FTM", ChainID: 250, Type: "evm", RPCURL: "https://rpc.fantom.network", ExplorerURL: "https://ftmscan.com", LogoURL: "https://assets.coingecko.com/coins/images/4001/small/Fantom_round.png", IsActive: true, IsTestnet: false, Decimals: 18, GasToken: "FTM", AvgBlockTime: 2, MaxGasPrice: 50000000000, SupportsEIP1559: true},
		{ID: "celo", Name: "Celo", Symbol: "CELO", ChainID: 42220, Type: "evm", RPCURL: "https://forno.celo.org", ExplorerURL: "https://celoscan.io", LogoURL: "https://assets.coingecko.com/coins/images/5568/small/celo.png", IsActive: true, IsTestnet: false, Decimals: 18, GasToken: "CELO", AvgBlockTime: 5, MaxGasPrice: 5000000000, SupportsEIP1559: false},
		{ID: "klaytn", Name: "Klaytn", Symbol: "KLAY", ChainID: 8217, Type: "evm", RPCURL: "https://public-en-cypress.klaytn.net", ExplorerURL: "https://scope.klaytn.com", LogoURL: "https://assets.coingecko.com/coins/images/9672/small/klaytn.png", IsActive: true, IsTestnet: false, Decimals: 18, GasToken: "KLAY", AvgBlockTime: 1, MaxGasPrice: 100000000000, SupportsEIP1559: false},
		{ID: "cronos", Name: "Cronos", Symbol: "CRO", ChainID: 25, Type: "evm", RPCURL: "https://rpc.cronos.org", ExplorerURL: "https://cronoscan.com", LogoURL: "https://assets.coingecko.com/coins/images/7310/small/cro.png", IsActive: true, IsTestnet: false, Decimals: 18, GasToken: "CRO", AvgBlockTime: 6, MaxGasPrice: 10000000000, SupportsEIP1559: false},
		{ID: "moonbeam", Name: "Moonbeam", Symbol: "GLMR", ChainID: 1284, Type: "evm", RPCURL: "https://rpc.api.moonbeam.network", ExplorerURL: "https://moonbeam.moonscan.io", LogoURL: "https://assets.coingecko.com/coins/images/17759/small/Moonbeam_Network_Icon.png", IsActive: true, IsTestnet: false, Decimals: 18, GasToken: "GLMR", AvgBlockTime: 12, MaxGasPrice: 10000000000, SupportsEIP1559: false},
		{ID: "astar", Name: "Astar", Symbol: "ASTR", ChainID: 592, Type: "evm", RPCURL: "https://rpc.astar.network", ExplorerURL: "https://astar.explorer.mainnet.solarflare.io", LogoURL: "https://assets.coingecko.com/coins/images/22617/small/astr.png", IsActive: true, IsTestnet: false, Decimals: 18, GasToken: "ASTR", AvgBlockTime: 12, MaxGasPrice: 10000000000, SupportsEIP1559: false},
		{ID: "polkadot", Name: "Polkadot", Symbol: "DOT", ChainID: -1, Type: "cosmos", RPCURL: "https://rpc.polkadot.io", ExplorerURL: "https://polkadot.subscan.io", LogoURL: "https://assets.coingecko.com/coins/images/12171/small/polkadot.png", IsActive: true, IsTestnet: false, Decimals: 10, GasToken: "DOT", AvgBlockTime: 6, MaxGasPrice: 1000000000, SupportsEIP1559: false},
		{ID: "algorand", Name: "Algorand", Symbol: "ALGO", ChainID: -1, Type: "account", RPCURL: "https://mainnet-api.algorand.network", ExplorerURL: "https://algoexplorer.io", LogoURL: "https://assets.coingecko.com/coins/images/4380/small/download.png", IsActive: true, IsTestnet: false, Decimals: 6, GasToken: "ALGO", AvgBlockTime: 4, MaxGasPrice: 1000000, SupportsEIP1559: false},
		{ID: "tezos", Name: "Tezos", Symbol: "XTZ", ChainID: -1, Type: "account", RPCURL: "https://mainnet.api.tez.ie", ExplorerURL: "https://tzstats.com", LogoURL: "https://assets.coingecko.com/coins/images/976/small/Tezos-logo.png", IsActive: true, IsTestnet: false, Decimals: 6, GasToken: "XTZ", AvgBlockTime: 30, MaxGasPrice: 1000000, SupportsEIP1559: false},
		{ID: "cardano", Name: "Cardano", Symbol: "ADA", ChainID: -1, Type: "utxo", RPCURL: "https://cardano-mainnet.blockfrost.io/api/v0", ExplorerURL: "https://cardanoscan.io", LogoURL: "https://assets.coingecko.com/coins/images/975/small/cardano.png", IsActive: true, IsTestnet: false, Decimals: 6, GasToken: "ADA", AvgBlockTime: 20, MaxGasPrice: 1000000, SupportsEIP1559: false},
		{ID: "ripple", Name: "XRP Ledger", Symbol: "XRP", ChainID: -1, Type: "utxo", RPCURL: "https://xrplcluster.com", ExplorerURL: "https://xrpscan.com", LogoURL: "https://assets.coingecko.com/coins/images/44/small/xrp-symbol-white-128.png", IsActive: true, IsTestnet: false, Decimals: 6, GasToken: "XRP", AvgBlockTime: 4, MaxGasPrice: 1000000, SupportsEIP1559: false},
		{ID: "injective", Name: "Injective", Symbol: "INJ", ChainID: -1, Type: "cosmos", RPCURL: "https://public.api.injective.network", ExplorerURL: "https://explorer.injective.network", LogoURL: "https://assets.coingecko.com/coins/images/12882/small/Secondary_Symbol.png", IsActive: true, IsTestnet: false, Decimals: 18, GasToken: "INJ", AvgBlockTime: 1, MaxGasPrice: 500000000000, SupportsEIP1559: false},
		{ID: "sei", Name: "Sei", Symbol: "SEI", ChainID: -1, Type: "cosmos", RPCURL: "https://rest.sei-apis.com", ExplorerURL: "https://seistream.app", LogoURL: "https://assets.coingecko.com/coins/images/28205/small/Sei_Logo_-_Transparent.png", IsActive: true, IsTestnet: false, Decimals: 18, GasToken: "SEI", AvgBlockTime: 1, MaxGasPrice: 1000000000, SupportsEIP1559: false},
		{ID: "immutable", Name: "Immutable", Symbol: "IMX", ChainID: -1, Type: "evm", RPCURL: "https://rpc.immutable.com", ExplorerURL: "https://immutascan.io", LogoURL: "https://assets.coingecko.com/coins/images/17233/small/immutableX-symbol-BLK-RGB.png", IsActive: true, IsTestnet: false, Decimals: 18, GasToken: "IMX", AvgBlockTime: 1, MaxGasPrice: 100000000000, SupportsEIP1559: true},
		{ID: "pulsechain", Name: "PulseChain", Symbol: "PLS", ChainID: 369, Type: "evm", RPCURL: "https://rpc.pulsechain.com", ExplorerURL: "https://scan.pulsechain.com", LogoURL: "https://assets.coingecko.com/coins/images/28195/small/pulsechain.png", IsActive: true, IsTestnet: false, Decimals: 18, GasToken: "PLS", AvgBlockTime: 12, MaxGasPrice: 500000000000, SupportsEIP1559: true},
		{ID: "mode", Name: "Mode", Symbol: "MOD", ChainID: 3446, Type: "evm", RPCURL: "https://mainnet.mode.network", ExplorerURL: "https://explorer.mode.network", LogoURL: "https://assets.coingecko.com/coins/images/31090/small/mode.jpg", IsActive: true, IsTestnet: false, Decimals: 18, GasToken: "ETH", AvgBlockTime: 2, MaxGasPrice: 100000000000, SupportsEIP1559: true},
		{ID: "fraxtal", Name: "Fraxtal", Symbol: "FRAX", ChainID: 4242, Type: "evm", RPCURL: "https://rpc.fraxtal.com", ExplorerURL: "https://fraxscan.com", LogoURL: "https://assets.coingecko.com/coins/images/13422/small/frax_logo.png", IsActive: true, IsTestnet: false, Decimals: 18, GasToken: "FRAX", AvgBlockTime: 2, MaxGasPrice: 10000000000, SupportsEIP1559: true},
		{ID: "berachain", Name: "Berachain", Symbol: "BERA", ChainID: 204, Type: "evm", RPCURL: "https://rpc.berachain.com", ExplorerURL: "https://berascan.com", LogoURL: "https://assets.coingecko.com/coins/images/33423/small/bera.jpg", IsActive: true, IsTestnet: false, Decimals: 18, GasToken: "BERA", AvgBlockTime: 2, MaxGasPrice: 100000000000, SupportsEIP1559: true},
		{ID: "monad", Name: "Monad", Symbol: "MON", ChainID: 10143, Type: "evm", RPCURL: "https://rpc.monad.xyz", ExplorerURL: "https://explorer.monad.xyz", LogoURL: "https://assets.coingecko.com/coins/images/34451/small/monad_avatar.png", IsActive: true, IsTestnet: false, Decimals: 18, GasToken: "MON", AvgBlockTime: 1, MaxGasPrice: 100000000000, SupportsEIP1559: true},
		{ID: "abstract", Name: "Abstract", Symbol: "ABST", ChainID: 2741, Type: "evm", RPCURL: "https://api.mainnet.abs.xyz", ExplorerURL: "https://explorer.abstract.xyz", LogoURL: "https://assets.coingecko.com/coins/images/41071/small/abstract.png", IsActive: true, IsTestnet: false, Decimals: 18, GasToken: "ETH", AvgBlockTime: 2, MaxGasPrice: 100000000000, SupportsEIP1559: true},
		{ID: "hedera", Name: "Hedera", Symbol: "HBAR", ChainID: -1, Type: "account", RPCURL: "https://mainnet.mirrornode.hedera.com", ExplorerURL: "https://hashscan.io", LogoURL: "https://assets.coingecko.com/coins/images/3688/small/hbar.png", IsActive: true, IsTestnet: false, Decimals: 8, GasToken: "HBAR", AvgBlockTime: 2, MaxGasPrice: 1000000, SupportsEIP1559: false},
		{ID: "kadena", Name: "Kadena", Symbol: "KDA", ChainID: -1, Type: "account", RPCURL: "https://api.kadena.network", ExplorerURL: "https://explorer.kadena.io", LogoURL: "https://assets.coingecko.com/coins/images/5647/small/kadena.png", IsActive: true, IsTestnet: false, Decimals: 12, GasToken: "KDA", AvgBlockTime: 60, MaxGasPrice: 1000000, SupportsEIP1559: false},
		{ID: "conflux", Name: "Conflux", Symbol: "CFX", ChainID: -1, Type: "evm", RPCURL: "https://rpc.confluxnetwork.org", ExplorerURL: "https://confluxscan.io", LogoURL: "https://assets.coingecko.com/coins/images/13079/small/3vuYMbjN.png", IsActive: true, IsTestnet: false, Decimals: 18, GasToken: "CFX", AvgBlockTime: 1, MaxGasPrice: 500000000000, SupportsEIP1559: true},
		{ID: "oasis", Name: "Oasis Network", Symbol: "ROSE", ChainID: -1, Type: "evm", RPCURL: "https://emerald.oasis.dev", ExplorerURL: "https://explorer.emerald.oasis.dev", LogoURL: "https://assets.coingecko.com/coins/images/13162/small/rose.png", IsActive: true, IsTestnet: false, Decimals: 18, GasToken: "ROSE", AvgBlockTime: 6, MaxGasPrice: 100000000000, SupportsEIP1559: true},
		{ID: "thorchain", Name: "THORChain", Symbol: "RUNE", ChainID: -1, Type: "cosmos", RPCURL: "https://rpc.thorchain.info", ExplorerURL: "https://thorchain.net", LogoURL: "https://assets.coingecko.com/coins/images/6595/small/Rune200x200.png", IsActive: true, IsTestnet: false, Decimals: 8, GasToken: "RUNE", AvgBlockTime: 6, MaxGasPrice: 1000000, SupportsEIP1559: false},
		{ID: "osmosis", Name: "Osmosis", Symbol: "OSMO", ChainID: -1, Type: "cosmos", RPCURL: "https://rpc-osmosis.ecostake.com", ExplorerURL: "https://osmosis.stargaze-apis.com", LogoURL: "https://assets.coingecko.com/coins/images/12210/small/osmosis.png", IsActive: true, IsTestnet: false, Decimals: 6, GasToken: "OSMO", AvgBlockTime: 6, MaxGasPrice: 1000000, SupportsEIP1559: false},
		{ID: "celestia", Name: "Celestia", Symbol: "TIA", ChainID: -1, Type: "cosmos", RPCURL: "https://rpc.celestia.org", ExplorerURL: "https://celestia.explorers.guru", LogoURL: "https://assets.coingecko.com/coins/images/31967/small/tia.jpg", IsActive: true, IsTestnet: false, Decimals: 6, GasToken: "TIA", AvgBlockTime: 12, MaxGasPrice: 1000000, SupportsEIP1559: false},
		{ID: "dydx", Name: "dYdX", Symbol: "DYDX", ChainID: -1, Type: "cosmos", RPCURL: "https://dydx-rpc.kingnodes.com", ExplorerURL: "https://dydx.explorers.guru", LogoURL: "https://assets.coingecko.com/coins/images/17500/small/hjm2WU1z_400x400.jpg", IsActive: true, IsTestnet: false, Decimals: 18, GasToken: "DYDX", AvgBlockTime: 5, MaxGasPrice: 1000000000, SupportsEIP1559: false},
		{ID: "neutron", Name: "Neutron", Symbol: "NTRN", ChainID: -1, Type: "cosmos", RPCURL: "https://rpc.cosmos.network", ExplorerURL: "https://ping.pub/neutron", LogoURL: "https://assets.coingecko.com/coins/images/33403/small/NEUTRON_512_512.png", IsActive: true, IsTestnet: false, Decimals: 6, GasToken: "NTRN", AvgBlockTime: 6, MaxGasPrice: 1000000, SupportsEIP1559: false},
		{ID: "kava", Name: "Kava", Symbol: "KAVA", ChainID: -1, Type: "cosmos", RPCURL: "https://rpc.kava.io", ExplorerURL: "https://kavascan.com", LogoURL: "https://assets.coingecko.com/coins/images/9762/small/kava.png", IsActive: true, IsTestnet: false, Decimals: 6, GasToken: "KAVA", AvgBlockTime: 5, MaxGasPrice: 1000000, SupportsEIP1559: false},
		{ID: "fetch", Name: "Fetch.ai", Symbol: "FET", ChainID: -1, Type: "cosmos", RPCURL: "https://rpc-fetchhub.fetch.ai", ExplorerURL: "https://fetchscan.io", LogoUrl: "https://assets.coingecko.com/coins/images/5681/small/Fetch.jpg", IsActive: true, IsTestnet: false, Decimals: 18, GasToken: "FET", AvgBlockTime: 6, MaxGasPrice: 1000000000, SupportsEIP1559: false},
		{ID: "stargaze", Name: "Stargaze", Symbol: "STARS", ChainID: -1, Type: "cosmos", RPCURL: "https://rpc.cosmos.network", ExplorerURL: "https://ping.pub/stargaze", LogoURL: "https://assets.coingecko.com/coins/images/17171/small/stars.png", IsActive: true, IsTestnet: false, Decimals: 6, GasToken: "STARS", AvgBlockTime: 6, MaxGasPrice: 1000000, SupportsEIP1559: false},
		{ID: "juno", Name: "Juno", Symbol: "JUNO", ChainID: -1, Type: "cosmos", RPCURL: "https://rpc.cosmos.network", ExplorerURL: "https://ping.pub/juno", LogoURL: "https://assets.coingecko.com/coins/images/28450/small/juno.png", IsActive: true, IsTestnet: false, Decimals: 6, GasToken: "JUNO", AvgBlockTime: 6, MaxGasPrice: 1000000, SupportsEIP1559: false},
		{ID: "akash", Name: "Akash", Symbol: "AKT", ChainID: -1, Type: "cosmos", RPCURL: "https://rpc.cosmos.network", ExplorerURL: "https://ping.pub/cosmos", LogoURL: "https://assets.coingecko.com/coins/images/12785/small/akash-logo.png", IsActive: true, IsTestnet: false, Decimals: 6, GasToken: "AKT", AvgBlockTime: 6, MaxGasPrice: 1000000, SupportsEIP1559: false},
		{ID: "mina", Name: "Mina", Symbol: "MINA", ChainID: -1, Type: "account", RPCURL: "https://mainnet.minaprotocol.com", ExplorerURL: "https://minaexplorer.com", LogoURL: "https://assets.coingecko.com/coins/images/15628/small/JM4_vQ34_400x400.png", IsActive: true, IsTestnet: false, Decimals: 9, GasToken: "MINA", AvgBlockTime: 180, MaxGasPrice: 100000000, SupportsEIP1559: false},
		{ID: "casper", Name: "Casper", Symbol: "CSPR", ChainID: -1, Type: "account", RPCURL: "https://rpc.cep18.com", ExplorerURL: "https://cspr.live", LogoURL: "https://assets.coingecko.com/coins/images/26333/small/casper-logo.png", IsActive: true, IsTestnet: false, Decimals: 9, GasToken: "CSPR", AvgBlockTime: 15, MaxGasPrice: 1000000000, SupportsEIP1559: false},
		{ID: "oasis-sapphire", Name: "Oasis Sapphire", Symbol: "ROSE", ChainID: 23294, Type: "evm", RPCURL: "https://sapphire.oasis.io", ExplorerURL: "https://explorer.sapphire.oasis.io", LogoURL: "https://assets.coingecko.com/coins/images/13162/small/rose.png", IsActive: true, IsTestnet: false, Decimals: 18, GasToken: "ROSE", AvgBlockTime: 6, MaxGasPrice: 100000000000, SupportsEIP1559: true},
		{ID: "flare", Name: "Flare", Symbol: "FLR", ChainID: 14, Type: "evm", RPCURL: "https://flare-api.flare.network/ext/bc/C/rpc", ExplorerURL: "https://flare-explorer.flare.network", LogoURL: "https://assets.coingecko.com/coins/images/28630/small/FLR.png", IsActive: true, IsTestnet: false, Decimals: 18, GasToken: "FLR", AvgBlockTime: 2, MaxGasPrice: 50000000000, SupportsEIP1559: false},
		{ID: "song", Name: "Song", Symbol: "S", ChainID: -1, Type: "account", RPCURL: "https://api.songmainnet.com", ExplorerURL: "https://songscan.io", LogoURL: "https://assets.coingecko.com/coins/images/40062/small/song.png", IsActive: true, IsTestnet: false, Decimals: 18, GasToken: "S", AvgBlockTime: 1, MaxGasPrice: 1000000000, SupportsEIP1559: false},
		{ID: "stellar", Name: "Stellar", Symbol: "XLM", ChainID: -1, Type: "account", RPCURL: "https://horizon.stellar.org", ExplorerURL: "https://stellar.expert", LogoURL: "https://assets.coingecko.com/coins/images/100/small/Stellar_symbol_black_RGB.png", IsActive: true, IsTestnet: false, Decimals: 7, GasToken: "XLM", AvgBlockTime: 5, MaxGasPrice: 1000000, SupportsEIP1559: false},
	}

	for _, chain := range defaultChains {
		s.blockchains[chain.ID] = chain
	}
}

func (s *BlockchainService) GetAll(ctx context.Context, page, limit int, chainType string) ([]*models.Blockchain, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*models.Blockchain
	start := (page - 1) * limit

	for _, chain := range s.blockchains {
		if chainType != "" && chain.Type != chainType {
			continue
		}
		if chain.IsActive {
			result = append(result, chain)
		}
	}

	if start >= len(result) {
		return []*models.Blockchain{}, nil
	}

	end := start + limit
	if end > len(result) {
		end = len(result)
	}

	return result[start:end], nil
}

func (s *BlockchainService) GetByID(ctx context.Context, id string) (*models.Blockchain, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	chain, ok := s.blockchains[id]
	if !ok {
		return nil, errors.New("blockchain not found")
	}

	return chain, nil
}

func (s *BlockchainService) Create(ctx context.Context, blockchain *models.Blockchain) (*models.Blockchain, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	blockchain.IsActive = true
	s.blockchains[blockchain.ID] = blockchain

	return blockchain, nil
}

func (s *BlockchainService) Update(ctx context.Context, id string, blockchain *models.Blockchain) (*models.Blockchain, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.blockchains[id]
	if !ok {
		return nil, errors.New("blockchain not found")
	}

	blockchain.ID = id
	blockchain.IsActive = existing.IsActive
	s.blockchains[id] = blockchain

	return blockchain, nil
}

func (s *BlockchainService) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.blockchains[id]; !ok {
		return errors.New("blockchain not found")
	}

	delete(s.blockchains, id)
	return nil
}

func (s *BlockchainService) Count(ctx context.Context) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, chain := range s.blockchains {
		if chain.IsActive {
			count++
		}
	}

	return count, nil
}
