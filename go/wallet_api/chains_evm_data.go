package main

// Code generated from the canonical ethereum-lists/chains registry
// (chainid.network/chains.json). DO NOT EDIT BY HAND. Each entry is a verified
// mainnet EVM chain with a public https RPC and a block explorer. Entries that
// were already present are preserved verbatim, including hand-checked RPC
// alternates and derivation paths. Admins can add or override any entry at
// runtime through the admin_chain_config PostgreSQL table.
//
// A chain is only listed here when the registry publishes at least one https
// RPC for it; an entry with no reachable endpoint would misrepresent the chain
// as usable.

var evmMainnet = []ChainConfig{
	{
		ID: 1, Name: "Ethereum Mainnet", Symbol: "ETH", RPCEndpoint: "https://api.mycryptoapi.com/eth", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://etherscan.io", ExplorerURL: "https://etherscan.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://ethereum-rpc.publicnode.com", "https://eth.drpc.org", "https://cloudflare-eth.com"},
	},
	{
		ID: 2, Name: "Expanse Network", Symbol: "EXP", RPCEndpoint: "https://node.expanse.tech", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "", ExplorerURL: "", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 8, Name: "Ubiq", Symbol: "UBQ", RPCEndpoint: "https://rpc.octano.dev", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://ubiqscan.io", ExplorerURL: "https://ubiqscan.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 10, Name: "Optimism", Symbol: "OP", RPCEndpoint: "https://mainnet.optimism.io", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://optimistic.etherscan.io", ExplorerURL: "https://optimistic.etherscan.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://optimism-rpc.publicnode.com", "https://optimism.drpc.org"},
	},
	{
		ID: 11, Name: "Metadium Mainnet", Symbol: "META", RPCEndpoint: "https://api.metadium.com/prod", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.metadium.com", ExplorerURL: "https://explorer.metadium.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 14, Name: "Flare Mainnet", Symbol: "FLR", RPCEndpoint: "https://flare-api.flare.network/ext/C/rpc", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://flare-explorer.flare.network", ExplorerURL: "https://flare-explorer.flare.network", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://flare.rpc.thirdweb.com", "https://flare-bundler.etherspot.io", "https://rpc.ankr.com/flare"},
	},
	{
		ID: 19, Name: "Songbird Canary-Network", Symbol: "SGB", RPCEndpoint: "https://songbird-api.flare.network/ext/C/rpc", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://songbird-explorer.flare.network", ExplorerURL: "https://songbird-explorer.flare.network", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://01-gravelines-006-01.rpc.tatum.io/ext/bc/C/rpc", "https://01-vinthill-006-02.rpc.tatum.io/ext/bc/C/rpc", "https://02-tokyo-006-03.rpc.tatum.io/ext/bc/C/rpc"},
	},
	{
		ID: 20, Name: "Elastos Smart Chain", Symbol: "ELA", RPCEndpoint: "https://api.elastos.io/eth", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://esc.elastos.io", ExplorerURL: "https://esc.elastos.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 25, Name: "Cronos Mainnet", Symbol: "CRO", RPCEndpoint: "https://evm.cronos.org", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://api.cronoscan.com", ExplorerURL: "https://cronoscan.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://cronos-evm-rpc.publicnode.com", "https://cronos.drpc.org"},
	},
	{
		ID: 27, Name: "ShibaChain", Symbol: "SHIB", RPCEndpoint: "https://rpc.shibchain.org", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://exp.shibchain.org", ExplorerURL: "https://exp.shibchain.org", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 30, Name: "Rootstock Mainnet", Symbol: "RBTC", RPCEndpoint: "https://public-node.rsk.co", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.rsk.co", ExplorerURL: "https://explorer.rsk.co", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 40, Name: "Telos EVM Mainnet", Symbol: "TLOS", RPCEndpoint: "https://rpc.telos.net", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://teloscan.io", ExplorerURL: "https://teloscan.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://telos.drpc.org"},
	},
	{
		ID: 50, Name: "XDC Network", Symbol: "XDC", RPCEndpoint: "https://erpc.xinfin.network", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://xdcscan.com", ExplorerURL: "https://xdcscan.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://rpc.xinfin.network", "https://rpc1.xinfin.network", "https://rpc.xdcrpc.com"},
	},
	{
		ID: 56, Name: "BNB Smart Chain Mainnet", Symbol: "BNB", RPCEndpoint: "https://bsc-dataseed1.bnbchain.org", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://bscscan.com", ExplorerURL: "https://bscscan.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://bsc-rpc.publicnode.com", "https://1rpc.io/bnb"},
	},
	{
		ID: 57, Name: "Syscoin Mainnet", Symbol: "SYS", RPCEndpoint: "https://rpc.syscoin.org", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.syscoin.org", ExplorerURL: "https://explorer.syscoin.org", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://syscoin.public-rpc.com", "https://57.rpc.thirdweb.com", "https://syscoin-evm.publicnode.com"},
	},
	{
		ID: 60, Name: "GoChain", Symbol: "GO", RPCEndpoint: "https://rpc.gochain.io", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.gochain.io", ExplorerURL: "https://explorer.gochain.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 61, Name: "Ethereum Classic", Symbol: "ETC", RPCEndpoint: "https://etc.rivet.link", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://etc.blockscout.com", ExplorerURL: "https://etc.blockscout.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 70, Name: "Hoo Smart Chain", Symbol: "HOO", RPCEndpoint: "https://http-mainnet.hoosmartchain.com", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://www.hooscan.com", ExplorerURL: "https://www.hooscan.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 82, Name: "Meter Mainnet", Symbol: "MTR", RPCEndpoint: "https://rpc.meter.io", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://scan.meter.io", ExplorerURL: "https://scan.meter.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 86, Name: "GateChain Mainnet", Symbol: "GT", RPCEndpoint: "https://evm.gatenode.cc", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://www.gatescan.org", ExplorerURL: "https://www.gatescan.org", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 88, Name: "Viction", Symbol: "VIC", RPCEndpoint: "https://rpc.viction.xyz", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "", ExplorerURL: "", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 99, Name: "POA Network Core", Symbol: "POA", RPCEndpoint: "https://core.poa.network", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://blockscout.com/poa/core", ExplorerURL: "https://blockscout.com/poa/core", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 100, Name: "Gnosis", Symbol: "XDAI", RPCEndpoint: "https://rpc.gnosischain.com", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://gnosisscan.io", ExplorerURL: "https://gnosisscan.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://gnosis-rpc.publicnode.com", "https://gnosis.drpc.org"},
	},
	{
		ID: 106, Name: "Velas EVM Mainnet", Symbol: "VLX", RPCEndpoint: "https://evmexplorer.velas.com/rpc", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://evmexplorer.velas.com", ExplorerURL: "https://evmexplorer.velas.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://explorer.velas.com/rpc"},
	},
	{
		ID: 108, Name: "ThunderCore Mainnet", Symbol: "TT", RPCEndpoint: "https://mainnet-rpc.thundercore.com", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://viewblock.io/thundercore", ExplorerURL: "https://viewblock.io/thundercore", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 113, Name: "Dehvo", Symbol: "Deh", RPCEndpoint: "https://connect.dehvo.com", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.dehvo.com", ExplorerURL: "https://explorer.dehvo.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 121, Name: "Realchain Mainnet", Symbol: "REAL", RPCEndpoint: "https://rcl-dataseed1.rclsidechain.com", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://rclscan.com", ExplorerURL: "https://rclscan.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 122, Name: "Fuse Mainnet", Symbol: "FUSE", RPCEndpoint: "https://rpc.fuse.io", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.fuse.io", ExplorerURL: "https://explorer.fuse.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://fuse.drpc.org"},
	},
	{
		ID: 126, Name: "OYchain Mainnet", Symbol: "OY", RPCEndpoint: "https://rpc.mainnet.oychain.io", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.oychain.io", ExplorerURL: "https://explorer.oychain.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 128, Name: "Huobi ECO Chain Mainnet", Symbol: "HT", RPCEndpoint: "https://http-mainnet.hecochain.com", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://hecoinfo.com", ExplorerURL: "https://hecoinfo.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 137, Name: "Polygon Mainnet", Symbol: "POL", RPCEndpoint: "https://polygon.drpc.org", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://polygonscan.com", ExplorerURL: "https://polygonscan.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://polygon-bor-rpc.publicnode.com", "https://polygon.drpc.org"},
	},
	{
		ID: 138, Name: "Defi Oracle Meta Mainnet", Symbol: "ETH", RPCEndpoint: "https://rpc-http-pub.d-bis.org", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.d-bis.org", ExplorerURL: "https://explorer.d-bis.org", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 143, Name: "Monad", Symbol: "MON", RPCEndpoint: "https://rpc.monad.xyz", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://monadvision.com", ExplorerURL: "https://monadvision.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 146, Name: "Sonic Mainnet", Symbol: "S", RPCEndpoint: "https://rpc.soniclabs.com", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://sonicscan.org", ExplorerURL: "https://sonicscan.org", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://sonic-rpc.publicnode.com", "https://sonic.drpc.org", "https://sonic-rpc.blockreq.com/v1/rpc/public"},
	},
	{
		ID: 148, Name: "ShimmerEVM", Symbol: "SMR", RPCEndpoint: "https://json-rpc.evm.shimmer.network", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.evm.shimmer.network", ExplorerURL: "https://explorer.evm.shimmer.network", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 151, Name: "Redbelly Network Mainnet", Symbol: "RBNT", RPCEndpoint: "https://governors.mainnet.redbelly.network", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://redbelly.routescan.io", ExplorerURL: "https://redbelly.routescan.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 166, Name: "Nomina", Symbol: "NOM", RPCEndpoint: "https://mainnet.nomina.io", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://nomscan.io", ExplorerURL: "https://nomscan.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 168, Name: "AIOZ Network", Symbol: "AIOZ", RPCEndpoint: "https://eth-dataseed.aioz.network", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.aioz.network", ExplorerURL: "https://explorer.aioz.network", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 169, Name: "Manta Pacific", Symbol: "ETH", RPCEndpoint: "https://pacific-rpc.manta.network/http", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://pacific-explorer.manta.network", ExplorerURL: "https://pacific-explorer.manta.network", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 182, Name: "IOST Mainnet", Symbol: "BNB", RPCEndpoint: "https://l2-mainnet.iost.io", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://l2-scan.iost.io", ExplorerURL: "https://l2-scan.iost.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 196, Name: "X Layer Mainnet", Symbol: "OKB", RPCEndpoint: "https://rpc.xlayer.tech", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://www.oklink.com/xlayer", ExplorerURL: "https://www.oklink.com/xlayer", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://xlayerrpc.okx.com"},
	},
	{
		ID: 204, Name: "opBNB Mainnet", Symbol: "BNB", RPCEndpoint: "https://opbnb-mainnet-rpc.bnbchain.org", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://mainnet.opbnbscan.com", ExplorerURL: "https://mainnet.opbnbscan.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 238, Name: "Blast Mainnet", Symbol: "OWCT", RPCEndpoint: "https://zkevmrpc.blastchain.org", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://blastchain.org", ExplorerURL: "https://blastchain.org", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 239, Name: "TAC Mainnet", Symbol: "TAC", RPCEndpoint: "https://rpc.tac.build", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.tac.build", ExplorerURL: "https://explorer.tac.build", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 246, Name: "Energy Web Chain", Symbol: "EWT", RPCEndpoint: "https://rpc.energyweb.org", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.energyweb.org", ExplorerURL: "https://explorer.energyweb.org", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 248, Name: "Oasys Mainnet", Symbol: "OAS", RPCEndpoint: "https://rpc.mainnet.oasys.games", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.oasys.games", ExplorerURL: "https://explorer.oasys.games", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 250, Name: "Fantom Opera", Symbol: "FTM", RPCEndpoint: "https://rpcapi.fantom.network", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://api.ftmscan.com", ExplorerURL: "https://ftmscan.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://fantom.drpc.org", "https://1rpc.io/ftm"},
	},
	{
		ID: 252, Name: "Fraxtal", Symbol: "FRAX", RPCEndpoint: "https://rpc.frax.com", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://api.fraxscan.com", ExplorerURL: "https://fraxscan.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 269, Name: "High Performance Blockchain", Symbol: "HPB", RPCEndpoint: "https://hpbnode.com", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://hscan.org", ExplorerURL: "https://hscan.org", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 273, Name: "XR One", Symbol: "XR1", RPCEndpoint: "https://xr1.calderachain.xyz/http", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://xr1.calderaexplorer.xyz", ExplorerURL: "https://xr1.calderaexplorer.xyz", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 288, Name: "Boba Network", Symbol: "ETH", RPCEndpoint: "https://mainnet.boba.network", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://bobascan.com", ExplorerURL: "https://bobascan.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://boba-ethereum.gateway.tenderly.co", "https://gateway.tenderly.co/public/boba-ethereum", "https://boba-eth.drpc.org"},
	},
	{
		ID: 295, Name: "Hedera Mainnet", Symbol: "HBAR", RPCEndpoint: "https://mainnet.hashio.io/api", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.arkhia.io", ExplorerURL: "https://explorer.arkhia.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 314, Name: "Filecoin - Mainnet", Symbol: "FIL", RPCEndpoint: "https://api.node.glif.io/", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://filfox.info/en", ExplorerURL: "https://filfox.info/en", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 321, Name: "KCC Mainnet", Symbol: "KCS", RPCEndpoint: "https://rpc-mainnet.kcc.network", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.kcc.io/en", ExplorerURL: "https://explorer.kcc.io/en", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 324, Name: "zkSync Era", Symbol: "ETH", RPCEndpoint: "https://mainnet.era.zksync.io", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.zksync.io", ExplorerURL: "https://explorer.zksync.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://zksync.drpc.org", "https://1rpc.io/zksync2-era"},
	},
	{
		ID: 336, Name: "Shiden", Symbol: "SDN", RPCEndpoint: "https://evm.shiden.astar.network", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://shiden.subscan.io", ExplorerURL: "https://shiden.subscan.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://shiden.api.onfinality.io/public", "https://shiden.public.blastapi.io"},
	},
	{
		ID: 361, Name: "Theta Mainnet", Symbol: "TFUEL", RPCEndpoint: "https://eth-rpc-api.thetatoken.org/rpc", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.thetatoken.org", ExplorerURL: "https://explorer.thetatoken.org", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 369, Name: "PulseChain", Symbol: "PLS", RPCEndpoint: "https://rpc.pulsechain.com", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://ipfs.scan.pulsechain.com", ExplorerURL: "https://ipfs.scan.pulsechain.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 416, Name: "SX Network Mainnet", Symbol: "SX", RPCEndpoint: "https://rpc.sx.technology", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.sx.technology", ExplorerURL: "https://explorer.sx.technology", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 480, Name: "World Chain", Symbol: "ETH", RPCEndpoint: "https://worldchain-mainnet.g.alchemy.com/public", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://worldscan.org", ExplorerURL: "https://worldscan.org", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 484, Name: "Camp Network Mainnet", Symbol: "CAMP", RPCEndpoint: "https://rpc.camp.raas.gelato.cloud", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://camp.cloud.blockscout.com", ExplorerURL: "https://camp.cloud.blockscout.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 499, Name: "Rupaya", Symbol: "RUPX", RPCEndpoint: "https://rpc.rupaya.io", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://scan.rupaya.io", ExplorerURL: "https://scan.rupaya.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 512, Name: "Double-A Chain Mainnet", Symbol: "AAC", RPCEndpoint: "https://rpc.acuteangle.com", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://scan.acuteangle.com", ExplorerURL: "https://scan.acuteangle.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 534, Name: "Candle", Symbol: "CNDL", RPCEndpoint: "https://candle-rpc.com/", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://candleexplorer.com", ExplorerURL: "https://candleexplorer.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 571, Name: "MetaChain Mainnet", Symbol: "MTC", RPCEndpoint: "https://rpc.metatime.com", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.metatime.com", ExplorerURL: "https://explorer.metatime.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 592, Name: "Astar", Symbol: "ASTR", RPCEndpoint: "https://evm.astar.network", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://astar.subscan.io", ExplorerURL: "https://astar.subscan.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://rpc.astar.network:8545"},
	},
	{
		ID: 686, Name: "Karura Network", Symbol: "KAR", RPCEndpoint: "https://eth-rpc-karura.aca-api.network", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://blockscout.karura.network", ExplorerURL: "https://blockscout.karura.network", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 698, Name: "Matchain", Symbol: "BNB", RPCEndpoint: "https://rpc.matchain.io", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://matchscan.io", ExplorerURL: "https://matchscan.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 711, Name: "Tucana", Symbol: "TUC", RPCEndpoint: "https://evm-rpc.tucana.zone", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.tucana.zone", ExplorerURL: "https://explorer.tucana.zone", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 712, Name: "Birdee-2", Symbol: "TUC", RPCEndpoint: "https://evm-rpc.birdee-2.tucana.zone", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.birdee-2.tucana.zone", ExplorerURL: "https://explorer.birdee-2.tucana.zone", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 787, Name: "Acala Network", Symbol: "ACA", RPCEndpoint: "https://eth-rpc-acala.aca-api.network", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://blockscout.acala.network", ExplorerURL: "https://blockscout.acala.network", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 813, Name: "Qitmeer Network Mainnet", Symbol: "MEER", RPCEndpoint: "https://evm-dataseed1.meerscan.io", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://qng.qitmeer.io", ExplorerURL: "https://qng.qitmeer.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 818, Name: "BeOne Chain Mainnet", Symbol: "BOC", RPCEndpoint: "https://dataseed1.beonechain.com", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://beonescan.com", ExplorerURL: "https://beonescan.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 820, Name: "Callisto Mainnet", Symbol: "CLO", RPCEndpoint: "https://rpc.callistodao.org", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.callistodao.org", ExplorerURL: "https://explorer.callistodao.org", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 841, Name: "Taraxa Mainnet", Symbol: "TARA", RPCEndpoint: "https://rpc.mainnet.taraxa.io/", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://tara.to", ExplorerURL: "https://tara.to", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 888, Name: "Wanchain", Symbol: "WAN", RPCEndpoint: "https://gwan-ssl.wandevs.org:56891/", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://wanscan.org", ExplorerURL: "https://wanscan.org", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 917, Name: "Rinia", Symbol: "FIRE", RPCEndpoint: "https://rinia-rpc1.thefirechain.com", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://rinia.firescan.io", ExplorerURL: "https://rinia.firescan.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 999, Name: "HyperEVM", Symbol: "HYPE", RPCEndpoint: "https://rpc.hyperliquid.xyz/evm", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://hyperevmscan.io", ExplorerURL: "https://hyperevmscan.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 1009, Name: "Jumbochain Mainnet", Symbol: "JNFTC", RPCEndpoint: "https://rpcpriv.jumbochain.org", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://jumboscan.jumbochain.org", ExplorerURL: "https://jumboscan.jumbochain.org", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 1029, Name: "BitTorrent Chain Donau", Symbol: "BTT", RPCEndpoint: "https://pre-rpc.bt.io", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://testnet.bttcscan.com", ExplorerURL: "https://testnet.bttcscan.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 1088, Name: "Metis Andromeda Mainnet", Symbol: "METIS", RPCEndpoint: "https://andromeda.metis.io/?owner=1088", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://andromeda-explorer.metis.io", ExplorerURL: "https://andromeda-explorer.metis.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://metis.drpc.org", "https://metis-rpc.publicnode.com"},
	},
	{
		ID: 1101, Name: "Polygon zkEVM", Symbol: "ETH", RPCEndpoint: "https://zkevm-rpc.com", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://api-zkevm.polygonscan.com", ExplorerURL: "https://zkevm.polygonscan.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://polygon-zkevm.drpc.org", "https://1rpc.io/polygon/zkevm"},
	},
	{
		ID: 1116, Name: "Core Blockchain Mainnet", Symbol: "CORE", RPCEndpoint: "https://rpc.coredao.org/", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://scan.coredao.org", ExplorerURL: "https://scan.coredao.org", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://rpc-core.icecreamswap.com", "https://core.drpc.org"},
	},
	{
		ID: 1134, Name: "StateMesh", Symbol: "MESH", RPCEndpoint: "https://rpc.statemesh.net", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://scan.statemesh.net", ExplorerURL: "https://scan.statemesh.net", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 1135, Name: "Lisk", Symbol: "ETH", RPCEndpoint: "https://rpc.api.lisk.com", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://blockscout.lisk.com", ExplorerURL: "https://blockscout.lisk.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 1234, Name: "Step Network", Symbol: "FITFI", RPCEndpoint: "https://rpc.step.network", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://stepscan.io", ExplorerURL: "https://stepscan.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 1284, Name: "Moonbeam", Symbol: "GLMR", RPCEndpoint: "https://moonbeam.api.onfinality.io/public", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://moonbeam.subscan.io", ExplorerURL: "https://moonbeam.subscan.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://rpc.api.moonbeam.network", "https://moonbeam.public.blastapi.io", "https://moonbeam.unitedbloc.com"},
	},
	{
		ID: 1285, Name: "Moonriver", Symbol: "MOVR", RPCEndpoint: "https://moonriver.api.onfinality.io/public", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://moonriver.subscan.io", ExplorerURL: "https://moonriver.subscan.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://rpc.api.moonriver.moonbeam.network", "https://moonriver.public.blastapi.io", "https://moonriver.unitedbloc.com"},
	},
	{
		ID: 1329, Name: "Sei Network", Symbol: "SEI", RPCEndpoint: "https://evm-rpc.sei-apis.com", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://seiscan.io", ExplorerURL: "https://seiscan.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 1453, Name: "MetaChain Istanbul", Symbol: "MTC", RPCEndpoint: "https://istanbul-rpc.metachain.dev", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://istanbul-explorer.metachain.dev", ExplorerURL: "https://istanbul-explorer.metachain.dev", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 1456, Name: "ZKBase Mainnet", Symbol: "ETH", RPCEndpoint: "https://mainnet-rpc.zkbase.app", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.zkbase.app", ExplorerURL: "https://explorer.zkbase.app", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 1559, Name: "Tenet", Symbol: "TENET", RPCEndpoint: "https://rpc.tenet.org", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://tenetscan.io", ExplorerURL: "https://tenetscan.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://tenet-evm.publicnode.com"},
	},
	{
		ID: 1625, Name: "Gravity Alpha Mainnet", Symbol: "G", RPCEndpoint: "https://rpc.gravity.xyz", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.gravity.xyz", ExplorerURL: "https://explorer.gravity.xyz", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://rpc.ankr.com/gravity"},
	},
	{
		ID: 1689, Name: "NERO Mainnet", Symbol: "NERO", RPCEndpoint: "https://rpc.nerochain.io", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://www.neroscan.io", ExplorerURL: "https://www.neroscan.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 1776, Name: "Injective", Symbol: "INJ", RPCEndpoint: "https://sentry.evm-rpc.injective.network", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://blockscout.injective.network", ExplorerURL: "https://blockscout.injective.network", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 1804, Name: "Kerleano", Symbol: "CRC", RPCEndpoint: "https://rpc1.kerleano.ca-dag.work", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://ethereum-pocr.github.io/explorer/kerleano", ExplorerURL: "https://ethereum-pocr.github.io/explorer/kerleano", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 1818, Name: "Cube Chain Mainnet", Symbol: "CUBE", RPCEndpoint: "https://http-mainnet.cube.network", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://cubescan.network", ExplorerURL: "https://cubescan.network", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 1821, Name: "Ruby Smart Chain MAINNET", Symbol: "RUBY", RPCEndpoint: "https://mainnet-data.rubychain.io/", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://rubyscan.net", ExplorerURL: "https://rubyscan.net", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 1829, Name: "PlayBlock", Symbol: "PBG", RPCEndpoint: "https://rpc.playblock.io", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.playblock.io", ExplorerURL: "https://explorer.playblock.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 1848, Name: "Swisstronik Mainnet", Symbol: "SWTR", RPCEndpoint: "https://json-rpc.mainnet.swisstronik.com/unencrypted/", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer-evm.mainnet.swisstronik.com", ExplorerURL: "https://explorer-evm.mainnet.swisstronik.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 1890, Name: "Lightlink Phoenix Mainnet", Symbol: "ETH", RPCEndpoint: "https://replicator.phoenix.lightlink.io/rpc/v1", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://phoenix.lightlink.io", ExplorerURL: "https://phoenix.lightlink.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 1899, Name: "ReDeFi Layer 2", Symbol: "RED", RPCEndpoint: "https://layer2.redefi.world", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://scanlayer2.redefi.world", ExplorerURL: "https://scanlayer2.redefi.world", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 1967, Name: "Eleanor", Symbol: "MTC", RPCEndpoint: "https://rpc.metatime.com/eleanor", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.metatime.com/eleanor", ExplorerURL: "https://explorer.metatime.com/eleanor", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 1997, Name: "Kyoto", Symbol: "KYOTO", RPCEndpoint: "https://rpc.kyotochain.io", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://kyotoscan.io", ExplorerURL: "https://kyotoscan.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 2000, Name: "Dogechain Mainnet", Symbol: "DOGE", RPCEndpoint: "https://rpc.dogechain.dog", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.dogechain.dog", ExplorerURL: "https://explorer.dogechain.dog", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://rpc01-sg.dogechain.dog", "https://rpc.ankr.com/dogechain"},
	},
	{
		ID: 2001, Name: "Milkomeda C1 Mainnet", Symbol: "mADA", RPCEndpoint: "https://rpc-mainnet-cardano-evm.c1.milkomeda.com", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer-mainnet-cardano-evm.c1.milkomeda.com", ExplorerURL: "https://explorer-mainnet-cardano-evm.c1.milkomeda.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 2002, Name: "Milkomeda A1 Mainnet", Symbol: "mALGO", RPCEndpoint: "https://rpc-mainnet-algorand-rollup.a1.milkomeda.com", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer-mainnet-algorand-rollup.a1.milkomeda.com", ExplorerURL: "https://explorer-mainnet-algorand-rollup.a1.milkomeda.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 2017, Name: "Adiri", Symbol: "TEL", RPCEndpoint: "https://rpc.telcoin.network", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://telscan.io", ExplorerURL: "https://telscan.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 2020, Name: "Ronin Mainnet", Symbol: "RON", RPCEndpoint: "https://api.roninchain.com/rpc", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://app.roninchain.com", ExplorerURL: "https://app.roninchain.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 2021, Name: "Edgeware EdgeEVM Mainnet", Symbol: "EDG", RPCEndpoint: "https://edgeware-evm.jelliedowl.net", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://edgscan.live", ExplorerURL: "https://edgscan.live", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 2025, Name: "Rangers Protocol Mainnet", Symbol: "RPG", RPCEndpoint: "https://mainnet.rangersprotocol.com/api/jsonrpc", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://scan.rangersprotocol.com", ExplorerURL: "https://scan.rangersprotocol.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 2049, Name: "Movo Smart Chain Mainnet", Symbol: "MOVO", RPCEndpoint: "https://msc-rpc.movoscan.com", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://movoscan.com", ExplorerURL: "https://movoscan.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 2109, Name: "Exosama Network", Symbol: "SAMA", RPCEndpoint: "https://rpc.exosama.com", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.exosama.com", ExplorerURL: "https://explorer.exosama.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 2199, Name: "Moonsama Network", Symbol: "SAMA", RPCEndpoint: "https://rpc.moonsama.com", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.moonsama.com", ExplorerURL: "https://explorer.moonsama.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 2222, Name: "Kava EVM", Symbol: "KAVA", RPCEndpoint: "https://evm.kava.io", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://api.kavascan.com", ExplorerURL: "https://kavascan.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 2366, Name: "KiteAI", Symbol: "KITE", RPCEndpoint: "https://rpc.gokite.ai", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://kitescan.ai", ExplorerURL: "https://kitescan.ai", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 2390, Name: "TAC Turin", Symbol: "TAC", RPCEndpoint: "https://turin.rpc.tac.build", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://turin.explorer.tac.build", ExplorerURL: "https://turin.explorer.tac.build", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 2391, Name: "TAC Saint Petersburg", Symbol: "TAC", RPCEndpoint: "https://spb.rpc.tac.build", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://spb.explorer.tac.build", ExplorerURL: "https://spb.explorer.tac.build", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 2425, Name: "King Of Legends Mainnet", Symbol: "KCC", RPCEndpoint: "https://rpc-mainnet.kinggamer.org/", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://kingscan.org", ExplorerURL: "https://kingscan.org", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 2440, Name: "Atleta Network", Symbol: "ATLA", RPCEndpoint: "https://rpc.mainnet.atleta.network", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://blockscout.atleta.network", ExplorerURL: "https://blockscout.atleta.network", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 2748, Name: "Nanon", Symbol: "ETH", RPCEndpoint: "https://rpc.nanon.network", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.nanon.network", ExplorerURL: "https://explorer.nanon.network", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 2818, Name: "Morph", Symbol: "ETH", RPCEndpoint: "https://rpc.morphl2.io", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.morphl2.io", ExplorerURL: "https://explorer.morphl2.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 3111, Name: "Alpha Chain Mainnet", Symbol: "ETH", RPCEndpoint: "https://rpc.goalpha.org", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://scan.goalpha.org", ExplorerURL: "https://scan.goalpha.org", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 3490, Name: "GTCSCAN", Symbol: "GTC", RPCEndpoint: "https://gtc-dataseed.gtcscan.io/", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://gtcscan.io", ExplorerURL: "https://gtcscan.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 3776, Name: "Astar zkEVM", Symbol: "ETH", RPCEndpoint: "https://rpc.startale.com/astar-zkevm", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://astar-zkevm.explorer.startale.com", ExplorerURL: "https://astar-zkevm.explorer.startale.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 4003, Name: "X1 Fastnet", Symbol: "XN", RPCEndpoint: "https://x1-fastnet.xen.network", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.x1-fastnet.xen.network", ExplorerURL: "https://explorer.x1-fastnet.xen.network", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 4114, Name: "Citrea Mainnet", Symbol: "cBTC", RPCEndpoint: "https://rpc.mainnet.citrea.xyz", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.mainnet.citrea.xyz", ExplorerURL: "https://explorer.mainnet.citrea.xyz", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 4158, Name: "CrossFi Mainnet", Symbol: "XFI", RPCEndpoint: "https://rpc.mainnet.ms/", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://xfiscan.com", ExplorerURL: "https://xfiscan.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 4242, Name: "Nexi Mainnet", Symbol: "NEXI", RPCEndpoint: "https://rpc.chain.nexi.technology/", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://www.nexiscan.com", ExplorerURL: "https://www.nexiscan.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 4243, Name: "Nexi V2 Mainnet", Symbol: "NEXI", RPCEndpoint: "https://chain.nexiv2.nexilix.com", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://www.nexiscan.com", ExplorerURL: "https://www.nexiscan.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 4646, Name: "MST Mainnet", Symbol: "MSTC", RPCEndpoint: "https://mariorpc.mstblockchain.com", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://mstscan.com", ExplorerURL: "https://mstscan.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 4663, Name: "Robinhood Chain", Symbol: "ETH", RPCEndpoint: "https://rpc.mainnet.chain.robinhood.com", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://robinhoodchain.blockscout.com", ExplorerURL: "https://robinhoodchain.blockscout.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://robinhood-rpc.publicnode.com", "https://rpc.ordofi.network", "https://robinhood.drpc.org"},
	},
	{
		ID: 5000, Name: "Mantle", Symbol: "MNT", RPCEndpoint: "https://rpc.mantle.xyz", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://api.mantlescan.xyz", ExplorerURL: "https://mantlescan.xyz", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://mantle-rpc.publicnode.com", "https://mantle.drpc.org"},
	},
	{
		ID: 5042, Name: "Arc", Symbol: "USDC", RPCEndpoint: "https://rpc.mainnet.arc.io", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.arc.io", ExplorerURL: "https://explorer.arc.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://rpc.blockdaemon.mainnet.arc.io", "https://rpc.drpc.mainnet.arc.io", "https://rpc.quicknode.mainnet.arc.io"},
	},
	{
		ID: 5232, Name: "LiterMark Chain", Symbol: "LMK", RPCEndpoint: "https://litermark.org/rpc", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://litermark.org", ExplorerURL: "https://litermark.org", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 5330, Name: "Superseed", Symbol: "ETH", RPCEndpoint: "https://mainnet.superseed.xyz", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.superseed.xyz", ExplorerURL: "https://explorer.superseed.xyz", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 5424, Name: "edeXa Mainnet", Symbol: "EDX", RPCEndpoint: "https://rpc.edexa.network", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.edexa.network", ExplorerURL: "https://explorer.edexa.network", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 5888, Name: "MANTRA Chain", Symbol: "MANTRA", RPCEndpoint: "https://evm.mantrachain.io", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://blockscout.mantrascan.io", ExplorerURL: "https://blockscout.mantrascan.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 6278, Name: "Rails", Symbol: "STEAMX", RPCEndpoint: "https://mainnet.steamexchange.io", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explore.steamexchange.io", ExplorerURL: "https://explore.steamexchange.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 6322, Name: "Aura Mainnet", Symbol: "AURA", RPCEndpoint: "https://jsonrpc.aura.network", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://aurascan.io", ExplorerURL: "https://aurascan.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 6779, Name: "Compverse Mainnet", Symbol: "CPV", RPCEndpoint: "https://rpc.compverse.io/", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://scan.compverse.io", ExplorerURL: "https://scan.compverse.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 6805, Name: "RACE Mainnet", Symbol: "ETH", RPCEndpoint: "https://racemainnet.io/", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://racescan.io", ExplorerURL: "https://racescan.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 6868, Name: "Pools Mainnet", Symbol: "POOLS", RPCEndpoint: "https://rpc.poolsmobility.com", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://scan.poolsmobility.com", ExplorerURL: "https://scan.poolsmobility.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 6941, Name: "Monolythium", Symbol: "LYTH", RPCEndpoint: "https://evm.mainnet.mononodes.xyz", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://monoscan.xyz", ExplorerURL: "https://monoscan.xyz", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 7000, Name: "ZetaChain Mainnet", Symbol: "ZETA", RPCEndpoint: "https://zetachain-evm.blockpi.network/v1/rpc/public", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://zetascan.com", ExplorerURL: "https://zetascan.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://zetachain-mainnet.g.allthatnode.com/archive/evm", "https://zeta-chain.drpc.org", "https://zetachain-mainnet.public.blastapi.io"},
	},
	{
		ID: 7341, Name: "Shyft Mainnet", Symbol: "SHYFT", RPCEndpoint: "https://rpc.shyft.network/", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://bx.shyft.network", ExplorerURL: "https://bx.shyft.network", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 7700, Name: "Canto", Symbol: "CANTO", RPCEndpoint: "https://canto.slingshot.finance", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://www.oklink.com/canto", ExplorerURL: "https://www.oklink.com/canto", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://canto-rpc.ansybl.io", "https://mainnode.plexnode.org:8545", "https://canto.gravitychain.io/"},
	},
	{
		ID: 7778, Name: "Orenium Mainnet Protocol", Symbol: "ORE", RPCEndpoint: "https://validator-mainnet.orenium.org", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://oreniumscan.org", ExplorerURL: "https://oreniumscan.org", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 7897, Name: "arena-z", Symbol: "ETH", RPCEndpoint: "https://rpc.arena-z.gg", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.arena-z.gg", ExplorerURL: "https://explorer.arena-z.gg", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 8017, Name: "iSunCoin Mainnet", Symbol: "ISC", RPCEndpoint: "https://mainnet.isuncoin.com", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://baifa.io/app/chains/8017", ExplorerURL: "https://baifa.io/app/chains/8017", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 8047, Name: "BOAT Mainnet", Symbol: "BOAT", RPCEndpoint: "https://rpc0.come.boat/", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://scan.come.boats", ExplorerURL: "https://scan.come.boats", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 8217, Name: "Kaia Mainnet", Symbol: "KAIA", RPCEndpoint: "https://public-en.node.kaia.io", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://kaiascope.com", ExplorerURL: "https://kaiascope.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 8453, Name: "Base", Symbol: "ETH", RPCEndpoint: "https://mainnet.base.org", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://api.basescan.org", ExplorerURL: "https://basescan.org", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://base-rpc.publicnode.com", "https://base.drpc.org"},
	},
	{
		ID: 8723, Name: "TOOL Global Mainnet", Symbol: "OLO", RPCEndpoint: "https://mainnet-web3.wolot.io", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://www.olo.network", ExplorerURL: "https://www.olo.network", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 8732, Name: "Bullions Smart Chain", Symbol: "BLN", RPCEndpoint: "https://rpc.bullionsx.org", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://bullionscan.org", ExplorerURL: "https://bullionscan.org", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 8822, Name: "IOTA EVM", Symbol: "IOTA", RPCEndpoint: "https://json-rpc.evm.iotaledger.net", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.evm.iota.org", ExplorerURL: "https://explorer.evm.iota.org", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 9001, Name: "Evmos", Symbol: "EVMOS", RPCEndpoint: "https://evmos.lava.build", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://escan.live", ExplorerURL: "https://escan.live", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://evmos-evm-rpc.publicnode.com"},
	},
	{
		ID: 9745, Name: "Plasma Mainnet", Symbol: "XPL", RPCEndpoint: "https://rpc.plasma.to", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://plasmascan.to", ExplorerURL: "https://plasmascan.to", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 9898, Name: "Larissa Chain", Symbol: "LRS", RPCEndpoint: "https://rpc.larissa.network", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://scan.larissa.network", ExplorerURL: "https://scan.larissa.network", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 9922, Name: "JingleX L2", Symbol: "JNX", RPCEndpoint: "https://rpc.jinglex.net", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://jinglex.net/explorer", ExplorerURL: "https://jinglex.net/explorer", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 9982, Name: "MFEV CHAIN MAINNET", Symbol: "MFEV", RPCEndpoint: "https://rpc.mfevscan.com", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://mfevscan.com", ExplorerURL: "https://mfevscan.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 18888, Name: "Titan (TKX)", Symbol: "TKX", RPCEndpoint: "https://titan-json-rpc.titanlab.io", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://tkxscan.io/Titan", ExplorerURL: "https://tkxscan.io/Titan", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 23294, Name: "Oasis Sapphire", Symbol: "ROSE", RPCEndpoint: "https://sapphire.oasis.io", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.oasis.io/mainnet/sapphire", ExplorerURL: "https://explorer.oasis.io/mainnet/sapphire", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 30088, Name: "MiYou Mainnet", Symbol: "MY", RPCEndpoint: "https://blockchain.miyou.io", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://myscan.miyou.io", ExplorerURL: "https://myscan.miyou.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 33139, Name: "ApeChain", Symbol: "APE", RPCEndpoint: "https://rpc.apechain.com", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://apescan.io", ExplorerURL: "https://apescan.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 34443, Name: "Mode", Symbol: "ETH", RPCEndpoint: "https://mainnet.mode.network", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.mode.network", ExplorerURL: "https://explorer.mode.network", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 36969, Name: "AMA Mainnet", Symbol: "AMA", RPCEndpoint: "https://mainnet-rpc.ama.one", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://ama-explorer.ddns.net", ExplorerURL: "https://ama-explorer.ddns.net", ChainType: "evm", Decimals: 9, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 42161, Name: "Arbitrum One", Symbol: "ETH", RPCEndpoint: "https://arb1.arbitrum.io/rpc", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://api.arbiscan.io", ExplorerURL: "https://arbiscan.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://arbitrum-one-rpc.publicnode.com", "https://arbitrum.drpc.org"},
	},
	{
		ID: 42170, Name: "Arbitrum Nova", Symbol: "ETH", RPCEndpoint: "https://nova.arbitrum.io/rpc", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://nova-explorer.arbitrum.io", ExplorerURL: "https://nova-explorer.arbitrum.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://arbitrum-nova-rpc.publicnode.com"},
	},
	{
		ID: 42220, Name: "Celo Mainnet", Symbol: "CELO", RPCEndpoint: "https://forno.celo.org", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://api.celoscan.io", ExplorerURL: "https://celoscan.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://celo-rpc.publicnode.com", "https://celo.drpc.org"},
	},
	{
		ID: 42766, Name: "ZKFair Mainnet", Symbol: "USDC", RPCEndpoint: "https://rpc.zkfair.io", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://scan.zkfair.io", ExplorerURL: "https://scan.zkfair.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 43114, Name: "Avalanche C-Chain", Symbol: "AVAX", RPCEndpoint: "https://api.avax.network/ext/bc/C/rpc", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://snowscan.xyz", ExplorerURL: "https://snowscan.xyz", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://avalanche-c-chain-rpc.publicnode.com", "https://avalanche.drpc.org"},
	},
	{
		ID: 43522, Name: "Insectarium", Symbol: "M", RPCEndpoint: "https://rpc.insectarium.memecore.net", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://insectarium.memecorescan.io", ExplorerURL: "https://insectarium.memecorescan.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 47805, Name: "REI Network", Symbol: "REI", RPCEndpoint: "https://rpc.rei.network", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://scan.rei.network", ExplorerURL: "https://scan.rei.network", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 57073, Name: "Ink", Symbol: "ETH", RPCEndpoint: "https://rpc-gel.inkonchain.com", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.inkonchain.com", ExplorerURL: "https://explorer.inkonchain.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://rpc-qnd.inkonchain.com"},
	},
	{
		ID: 59144, Name: "Linea", Symbol: "ETH", RPCEndpoint: "https://rpc.linea.build", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://api.lineascan.build", ExplorerURL: "https://lineascan.build", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://linea-rpc.publicnode.com", "https://linea.drpc.org"},
	},
	{
		ID: 71402, Name: "Godwoken Mainnet", Symbol: "pCKB", RPCEndpoint: "https://v1.mainnet.godwoken.io/rpc", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://v1.gwscan.com", ExplorerURL: "https://v1.gwscan.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 77001, Name: "BORAchain mainnet", Symbol: "BORA", RPCEndpoint: "https://public-node.api.boraportal.com/bora/mainnet", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://scope.boraportal.com", ExplorerURL: "https://scope.boraportal.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 80094, Name: "Berachain", Symbol: "BERA", RPCEndpoint: "https://rpc.berachain.com", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://berascan.com", ExplorerURL: "https://berascan.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://berachain-rpc.publicnode.com", "https://rpc.berachain-apis.com", "https://berachain-rpc.blockreq.com/v1/rpc/public"},
	},
	{
		ID: 81457, Name: "Blast", Symbol: "ETH", RPCEndpoint: "https://rpc.blast.io", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://api.blastscan.io", ExplorerURL: "https://blastscan.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://blast.drpc.org"},
	},
	{
		ID: 84841, Name: "O Chain", Symbol: "O", RPCEndpoint: "https://rpc.o.xyz", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.o.xyz", ExplorerURL: "https://explorer.o.xyz", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 88888, Name: "Chiliz Chain Mainnet", Symbol: "CHZ", RPCEndpoint: "https://rpc.chiliz.com", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://chiliscan.com", ExplorerURL: "https://chiliscan.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://rpc.ankr.com/chiliz", "https://chiliz.publicnode.com"},
	},
	{
		ID: 96370, Name: "Lumoz Chain Mainnet", Symbol: "MOZ", RPCEndpoint: "https://rpc.lumoz.org", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://scan.lumoz.info", ExplorerURL: "https://scan.lumoz.info", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 98866, Name: "Plume Mainnet", Symbol: "PLUME", RPCEndpoint: "https://rpc.plume.org", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.plume.org", ExplorerURL: "https://explorer.plume.org", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 111188, Name: "re.al", Symbol: "reETH", RPCEndpoint: "https://rpc.realforreal.gelato.digital", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.re.al", ExplorerURL: "https://explorer.re.al", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 167000, Name: "Taiko", Symbol: "ETH", RPCEndpoint: "https://rpc.mainnet.taiko.xyz", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://taikoscan.io", ExplorerURL: "https://taikoscan.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://taiko-rpc.publicnode.com"},
	},
	{
		ID: 200901, Name: "Bitlayer Mainnet", Symbol: "BTC", RPCEndpoint: "https://rpc.bitlayer.org", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://www.btrscan.com", ExplorerURL: "https://www.btrscan.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 534352, Name: "Scroll", Symbol: "ETH", RPCEndpoint: "https://rpc.scroll.io", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://api.scrollscan.com", ExplorerURL: "https://scrollscan.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://scroll-rpc.publicnode.com", "https://scroll.drpc.org"},
	},
	{
		ID: 747474, Name: "katana", Symbol: "ETH", RPCEndpoint: "https://rpc.katana.network", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://katanascan.com", ExplorerURL: "https://katanascan.com", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://katana.gateway.tenderly.co/", "https://rpc.katanarpc.com/"},
	},
	{
		ID: 955305, Name: "Eluvio Content Fabric", Symbol: "ELV", RPCEndpoint: "https://host-76-74-28-226.contentfabric.io/eth/", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.eluv.io", ExplorerURL: "https://explorer.eluv.io", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 7777777, Name: "Zora", Symbol: "ETH", RPCEndpoint: "https://rpc.zora.energy", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.zora.energy", ExplorerURL: "https://explorer.zora.energy", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 65000000, Name: "Autonity Mainnet", Symbol: "ATN", RPCEndpoint: "https://rpc.autonity-apis.com", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://autonityscan.org", ExplorerURL: "https://autonityscan.org", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 728126428, Name: "Tron Mainnet", Symbol: "TRX", RPCEndpoint: "https://rpc.ankr.com/tron_jsonrpc", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://tronscan.org", ExplorerURL: "https://tronscan.org", ChainType: "evm", Decimals: 6, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 1313161554, Name: "Aurora Mainnet", Symbol: "ETH", RPCEndpoint: "https://mainnet.aurora.dev", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.aurora.dev", ExplorerURL: "https://explorer.aurora.dev", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false, RPCAlternates: []string{"https://aurora.drpc.org"},
	},
	{
		ID: 1666600000, Name: "Harmony Mainnet Shard 0", Symbol: "ONE", RPCEndpoint: "https://api.harmony.one", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.harmony.one", ExplorerURL: "https://explorer.harmony.one", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
	{
		ID: 1666600001, Name: "Harmony Mainnet Shard 1", Symbol: "ONE", RPCEndpoint: "https://api.s1.t.hmny.io", DerivationPath: "m/44'/60'/0'/0/0", ExplorerAPI: "https://explorer.harmony.one/blocks/shard/1", ExplorerURL: "https://explorer.harmony.one/blocks/shard/1", ChainType: "evm", Decimals: 18, CoinType: 60, IsTestnet: false,
	},
}

const evmMainnetCount = 196
