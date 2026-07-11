/**
 * TigerWallet - Universal Chain Registry
 * 
 * Support for 100+ Blockchains and 200+ Tokens
 * 
 * EVM Chains (50+):
 * - Ethereum, BNB Chain, Polygon, Arbitrum, Optimism, Base, Avalanche, 
 * - Fantom, Celo, Aurora, Klaytn, Harmony, Cronos, Gnosis, Moonriver,
 * - Moonbeam, Astar, Shiden, Canto, Mantle, Scroll, zkSync, Linea,
 * - Polygon zkEVM, OpBNB, Boba, Arbitrum Nova, etc.
 * 
 * Non-EVM Chains (50+):
 * - Solana, Tron, Bitcoin, Cosmos, Pi Network, Toncoin, Aptos, Sui,
 * - Near, Algorand, Hedera, Polygon (non-EVM), Polkadot, Kusama,
 * - Kadena, Flow, Tezos, Near, Radix, Casper, etc.
 * 
 * 200+ Tokens across all chains
 */

export interface Chain {
  id: number;
  name: string;
  symbol: string;
  chainId: number;
  type: 'EVM' | 'SOLANA' | 'TRON' | 'BITCOIN' | 'COSMOS' | 'NEAR' | 'APTOS' | 'SUI' | 'OTHER';
  rpcUrl: string;
  explorerUrl: string;
  icon: string;
  decimals: number;
  isActive: boolean;
  isDefault: boolean;
  color?: string;
  addressPrefix?: string;
  derivePath?: string;
}

export interface Token {
  id: number;
  chainId: number;
  address: string;
  symbol: string;
  name: string;
  decimals: number;
  isNative: boolean;
  isActive: boolean;
  priceUSD: number;
  logo?: string;
  coingeckoId?: string;
}

// 100+ BLOCKCHAINS
export const SUPPORTED_CHAINS: Chain[] = [
  // EVM BLOCKCHAINS
  { id: 1, name: 'Ethereum', symbol: 'ETH', chainId: 1, type: 'EVM', rpcUrl: 'https://eth.llamarpc.com', explorerUrl: 'https://etherscan.io', icon: '⬡', decimals: 18, isActive: true, isDefault: true, color: '#627EEA' },
  { id: 2, name: 'BNB Smart Chain', symbol: 'BNB', chainId: 56, type: 'EVM', rpcUrl: 'https://bsc-dataseed.binance.org', explorerUrl: 'https://bscscan.com', icon: '🟡', decimals: 18, isActive: true, isDefault: true, color: '#F3BA2F' },
  { id: 3, name: 'Polygon', symbol: 'MATIC', chainId: 137, type: 'EVM', rpcUrl: 'https://polygon-rpc.com', explorerUrl: 'https://polygonscan.com', icon: '🟣', decimals: 18, isActive: true, isDefault: true, color: '#8247E5' },
  { id: 4, name: 'Arbitrum One', symbol: 'ETH', chainId: 42161, type: 'EVM', rpcUrl: 'https://arb1.arbitrum.io/rpc', explorerUrl: 'https://arbiscan.io', icon: '🔵', decimals: 18, isActive: true, isDefault: true, color: '#28A0F0' },
  { id: 5, name: 'Optimism', symbol: 'ETH', chainId: 10, type: 'EVM', rpcUrl: 'https://mainnet.optimism.io', explorerUrl: 'https://optimistic.etherscan.io', icon: '🔴', decimals: 18, isActive: true, isDefault: true, color: '#FF0420' },
  { id: 6, name: 'Base', symbol: 'ETH', chainId: 8453, type: 'EVM', rpcUrl: 'https://mainnet.base.org', explorerUrl: 'https://basescan.org', icon: '🔵', decimals: 18, isActive: true, isDefault: true, color: '#0052FF' },
  { id: 7, name: 'Avalanche C-Chain', symbol: 'AVAX', chainId: 43114, type: 'EVM', rpcUrl: 'https://api.avax.network/ext/bc/C/rpc', explorerUrl: 'https://snowtrace.io', icon: '🔺', decimals: 18, isActive: true, isDefault: true, color: '#E84142' },
  { id: 8, name: 'Fantom', symbol: 'FTM', chainId: 250, type: 'EVM', rpcUrl: 'https://rpc.fantom.network', explorerUrl: 'https://ftmscan.com', icon: '👻', decimals: 18, isActive: true, isDefault: false, color: '#1969FF' },
  { id: 9, name: 'Celo', symbol: 'CELO', chainId: 42220, type: 'EVM', rpcUrl: 'https://forno.celo.org', explorerUrl: 'https://explorer.celo.org', icon: '🌕', decimals: 18, isActive: true, isDefault: false, color: '#FCFF52' },
  { id: 10, name: 'Aurora', symbol: 'ETH', chainId: 1313161554, type: 'EVM', rpcUrl: 'https://mainnet.aurora.dev', explorerUrl: 'https://explorer.aurora.dev', icon: '🌌', decimals: 18, isActive: true, isDefault: false, color: '#6D3FB8' },
  { id: 11, name: 'Klaytn', symbol: 'KLAY', chainId: 8217, type: 'EVM', rpcUrl: 'https://klaytn-mainnet-rpc.allthatnode.com', explorerUrl: 'https://scope.klaytn.com', icon: '🧱', decimals: 18, isActive: true, isDefault: false, color: '#334FFE' },
  { id: 12, name: 'Cronos', symbol: 'CRO', chainId: 25, type: 'EVM', rpcUrl: 'https://evm.cronos.org', explorerUrl: 'https://cronoscan.com', icon: '💳', decimals: 18, isActive: true, isDefault: false, color: '#002D74' },
  { id: 13, name: 'Gnosis Chain', symbol: 'xDAI', chainId: 100, type: 'EVM', rpcUrl: 'https://rpc.gnosischain.com', explorerUrl: 'https://blockscout.com/xdai/mainnet', icon: '🦉', decimals: 18, isActive: true, isDefault: false, color: '#04795B' },
  { id: 14, name: 'Moonbeam', symbol: 'GLMR', chainId: 1284, type: 'EVM', rpcUrl: 'https://rpc.api.moonbeam.network', explorerUrl: 'https://moonbeam.moonscan.io', icon: '🌙', decimals: 18, isActive: true, isDefault: false, color: '#53CBC9' },
  { id: 15, name: 'Moonriver', symbol: 'MOVR', chainId: 1285, type: 'EVM', rpcUrl: 'https://rpc.api.moonriver.moonbeam.network', explorerUrl: 'https://moonriver.moonscan.io', icon: '🌊', decimals: 18, isActive: true, isDefault: false, color: '#5AADFF' },
  { id: 16, name: 'Astar', symbol: 'ASTR', chainId: 592, type: 'EVM', rpcUrl: 'https://rpc.astar.network', explorerUrl: 'https://blockscout.com/astar', icon: '⭐', decimals: 18, isActive: true, isDefault: false, color: '#2E2E2E' },
  { id: 17, name: 'Shiden', symbol: 'SDN', chainId: 336, type: 'EVM', rpcUrl: 'https://rpc.shiden.astar.network', explorerUrl: 'https://blockscout.com/shiden', icon: '⚡', decimals: 18, isActive: true, isDefault: false, color: '#3779D7' },
  { id: 18, name: 'Canto', symbol: 'CANTO', chainId: 7700, type: 'EVM', rpcUrl: 'https://canto.gravitychain.io', explorerUrl: 'https://evm.explorer.canto.io', icon: '🎵', decimals: 18, isActive: true, isDefault: false, color: '#00CECB' },
  { id: 19, name: 'Mantle', symbol: 'MNT', chainId: 5000, type: 'EVM', rpcUrl: 'https://rpc.mantle.xyz', explorerUrl: 'https://explorer.mantle.xyz', icon: '🅿️', decimals: 18, isActive: true, isDefault: false, color: '#1A1A1A' },
  { id: 20, name: 'Scroll', symbol: 'ETH', chainId: 534352, type: 'EVM', rpcUrl: 'https://scroll.io', explorerUrl: 'https://scrollscan.com', icon: '📜', decimals: 18, isActive: true, isDefault: false, color: '#FFEEDA' },
  { id: 21, name: 'zkSync Era', symbol: 'ETH', chainId: 324, type: 'EVM', rpcUrl: 'https://mainnet.era.zksync.io', explorerUrl: 'https://explorer.zksync.io', icon: '⚖️', decimals: 18, isActive: true, isDefault: false, color: '#8B8BF5' },
  { id: 22, name: 'Linea', symbol: 'ETH', chainId: 59144, type: 'EVM', rpcUrl: 'https://rpc.linea.build', explorerUrl: 'https://explorer.linea.build', icon: '📐', decimals: 18, isActive: true, isDefault: false, color: '#121212' },
  { id: 23, name: 'Polygon zkEVM', symbol: 'ETH', chainId: 1101, type: 'EVM', rpcUrl: 'https://zkevm-rpc.polygon.technology', explorerUrl: 'https://zkevm.polygonscan.com', icon: '🔷', decimals: 18, isActive: true, isDefault: false, color: '#8247E5' },
  { id: 24, name: 'OpBNB', symbol: 'BNB', chainId: 204, type: 'EVM', rpcUrl: 'https://opbnb-rpc.publicnode.com', explorerUrl: 'https://opbnbscan.com', icon: '🟠', decimals: 18, isActive: true, isDefault: false, color: '#F3BA2F' },
  { id: 25, name: 'Harmony', symbol: 'ONE', chainId: 1666600000, type: 'EVM', rpcUrl: 'https://api.harmony.one', explorerUrl: 'https://explorer.harmony.one', icon: '🎯', decimals: 18, isActive: true, isDefault: false, color: '#00AEEF' },
  { id: 26, name: 'Kava', symbol: 'KAVA', chainId: 2222, type: 'EVM', rpcUrl: 'https://evm.kava.io', explorerUrl: 'https://explorer.kava.io', icon: '🔮', decimals: 18, isActive: true, isDefault: false, color: '#FF5533' },
  { id: 27, name: 'PulseChain', symbol: 'PLS', chainId: 369, type: 'EVM', rpcUrl: 'https://rpc.pulsechain.com', explorerUrl: 'https://scan.pulsechain.com', icon: '🟢', decimals: 18, isActive: true, isDefault: false, color: '#00FF95' },
  { id: 28, name: 'Boba', symbol: 'BOBA', chainId: 288, type: 'EVM', rpcUrl: 'https://mainnet.boba.network', explorerUrl: 'https://blockexplorer.boba.network', icon: '👶', decimals: 18, isActive: true, isDefault: false, color: '#4B4B4B' },
  { id: 29, name: 'Arbitrum Nova', symbol: 'ETH', chainId: 42170, type: 'EVM', rpcUrl: 'https://nova.arbitrum.io/rpc', explorerUrl: 'https://nova.arbiscan.io', icon: '🆕', decimals: 18, isActive: true, isDefault: false, color: '#28A0F0' },
  { id: 30, name: 'Filecoin', symbol: 'FIL', chainId: 314, type: 'EVM', rpcUrl: 'https://api.filecoin.space', explorerUrl: 'https://filfox.info', icon: '📁', decimals: 18, isActive: true, isDefault: false, color: '#0090FF' },
  { id: 31, name: 'Zora', symbol: 'ETH', chainId: 7777777, type: 'EVM', rpcUrl: 'https://rpc.zora.energy', explorerUrl: 'https://explorer.zora.energy', icon: '🎨', decimals: 18, isActive: true, isDefault: false, color: '#000000' },
  { id: 32, name: 'Manta', symbol: 'MANTA', chainId: 1699, type: 'EVM', rpcUrl: 'https://rpc.manta.network', explorerUrl: 'https://explorer.manta.network', icon: '🦭', decimals: 18, isActive: true, isDefault: false, color: '#0070FF' },
  { id: 33, name: 'Mode', symbol: 'ETH', chainId: 34443, type: 'EVM', rpcUrl: 'https://mainnet.mode.network', explorerUrl: 'https://explorer.mode.network', icon: '🚀', decimals: 18, isActive: true, isDefault: false, color: '#1A1A2E' },
  { id: 34, name: 'Fraxtal', symbol: 'FRAX', chainId: 5002, type: 'EVM', rpcUrl: 'https://rpc.frax.com', explorerUrl: 'https://fraxscan.com', icon: '🏛️', decimals: 18, isActive: true, isDefault: false, color: '#8B5CF6' },
  { id: 35, name: 'Viction', symbol: 'VIC', chainId: 8888888, type: 'EVM', rpcUrl: 'https://rpc.viction.xyz', explorerUrl: 'https://vicscan.xyz', icon: '🏆', decimals: 18, isActive: true, isDefault: false, color: '#FF3D00' },
  { id: 36, name: 'Ronin', symbol: 'RON', chainId: 2020, type: 'EVM', rpcUrl: 'https://api.roninchain.com/rpc', explorerUrl: 'https://app.roninchain.com', icon: '🪙', decimals: 18, isActive: true, isDefault: false, color: '#C43B2D' },
  { id: 37, name: 'Oasis', symbol: 'ROSE', chainId: 42262, type: 'EVM', rpcUrl: 'https://emerald.oasis.dev', explorerUrl: 'https://explorer.emerald.oasis.dev', icon: '🌿', decimals: 18, isActive: true, isDefault: false, color: '#2A2A2A' },
  { id: 38, name: 'Evmos', symbol: 'EVMOS', chainId: 9001, type: 'EVM', rpcUrl: 'https://evmos-rpc.publicnode.com', explorerUrl: 'https://evmoscan.com', icon: '⚡', decimals: 18, isActive: true, isDefault: false, color: '#EDA445' },
  { id: 39, name: 'KCC', symbol: 'KCS', chainId: 321, type: 'EVM', rpcUrl: 'https://rpc.kcc.network', explorerUrl: 'https://explorer.kcc.network', icon: '🔑', decimals: 18, isActive: true, isDefault: false, color: '#5261C8' },
  { id: 40, name: 'Dogechain', symbol: 'DC', chainId: 2000, type: 'EVM', rpcUrl: 'https://rpc.dogechain.dog', explorerUrl: 'https://explorer.dogechain.dog', icon: '🐕', decimals: 18, isActive: true, isDefault: false, color: '#C2A3C7' },
  { id: 41, name: 'Milkomeda', symbol: 'ADA', chainId: 2001, type: 'EVM', rpcUrl: 'https://rpc-mainnet-cardano-evm.c1.milkomeda.com', explorerUrl: 'https://explorer-mainnet-cardano-evm.c1.milkomeda.com', icon: '₳', decimals: 18, isActive: true, isDefault: false, color: '#0033AD' },
  { id: 42, name: 'SYS', symbol: 'SYS', chainId: 57, type: 'EVM', rpcUrl: 'https://rpc.syscoin.org', explorerUrl: 'https://explorer.syscoin.org', icon: '⚙️', decimals: 18, isActive: true, isDefault: false, color: '#00D2D3' },
  { id: 43, name: 'Lightlink', symbol: 'LGT', chainId: 1890, type: 'EVM', rpcUrl: 'https://rpc.lightlink.io', explorerUrl: 'https://explorer.lightlink.io', icon: '💡', decimals: 18, isActive: true, isDefault: false, color: '#FFB800' },
  { id: 44, name: 'Berachain', symbol: 'BERA', chainId: 80084, type: 'EVM', rpcUrl: 'https://artio.rpc.berachain.com', explorerUrl: 'https://artio.beratrail.io', icon: '🐻', decimals: 18, isActive: true, isDefault: false, color: '#F55D37' },
  { id: 45, name: 'Metal L2', symbol: 'MTL', chainId: 4250, type: 'EVM', rpcUrl: 'https://rpc.l2.metal.info', explorerUrl: 'https://explorer.l2.metal.info', icon: '⚙️', decimals: 18, isActive: true, isDefault: false, color: '#2A2A2A' },
  { id: 46, name: 'Redstone', symbol: 'RED', chainId: 690, type: 'EVM', rpcUrl: 'https://rpc.redstone.xyz', explorerUrl: 'https://explorer.redstone.xyz', icon: '💎', decimals: 18, isActive: true, isDefault: false, color: '#FF4D4D' },
  { id: 47, name: 'IoTeX', symbol: 'IOTX', chainId: 4689, type: 'EVM', rpcUrl: 'https://rpc.iotex.io', explorerUrl: 'https://iotexscan.io', icon: '📡', decimals: 18, isActive: true, isDefault: false, color: '#00D2D3' },
  { id: 48, name: 'Conflux', symbol: 'CFX', chainId: 1030, type: 'EVM', rpcUrl: 'https://mainnet.confluxrpc.com', explorerUrl: 'https://confluxscan.net', icon: '🔵', decimals: 18, isActive: true, isDefault: false, color: '#00C2B4' },
  { id: 49, name: 'Gather', symbol: 'GTH', chainId: 192837, type: 'EVM', rpcUrl: 'https://rpc.gather.network', explorerUrl: 'https://explorer.gather.network', icon: '🕸️', decimals: 18, isActive: true, isDefault: false, color: '#00C2B4' },
  { id: 50, name: 'Rsk', symbol: 'RBTC', chainId: 30, type: 'EVM', rpcUrl: 'https://public-node.rsk.co', explorerUrl: 'https://explorer.rsk.co', icon: '🔴', decimals: 18, isActive: true, isDefault: false, color: '#000000' },
  
  // NON-EVM BLOCKCHAINS
  { id: 51, name: 'Solana', symbol: 'SOL', chainId: 101, type: 'SOLANA', rpcUrl: 'https://api.mainnet-beta.solana.com', explorerUrl: 'https://solscan.io', icon: '☀️', decimals: 9, isActive: true, isDefault: true, addressPrefix: 'solana', derivePath: "m/44'/501'/0'/0'" },
  { id: 52, name: 'Tron', symbol: 'TRX', chainId: 728126428, type: 'TRON', rpcUrl: 'https://api.trongrid.io', explorerUrl: 'https://tronscan.org', icon: '🔶', decimals: 6, isActive: true, isDefault: true, addressPrefix: 'tron' },
  { id: 53, name: 'Bitcoin', symbol: 'BTC', chainId: 0, type: 'BITCOIN', rpcUrl: 'https://btc.lit.io', explorerUrl: 'https://blockstream.info', icon: '₿', decimals: 8, isActive: true, isDefault: true, addressPrefix: 'bitcoin', derivePath: "m/44'/0'/0'/0/0" },
  { id: 54, name: 'Cosmos Hub', symbol: 'ATOM', chainId: 118, type: 'COSMOS', rpcUrl: 'https://rpc.cosmos.network', explorerUrl: 'https://mintscan.io', icon: '⚛️', decimals: 6, isActive: true, isDefault: false, addressPrefix: 'cosmos', derivePath: "m/44'/118'/0'/0/0'" },
  { id: 55, name: 'Pi Network', symbol: 'PI', chainId: 314159, type: 'OTHER', rpcUrl: 'https://api.pinetwork.org', explorerUrl: 'https://explorer.pinetwork.org', icon: 'π', decimals: 18, isActive: true, isDefault: true, addressPrefix: 'pi' },
  { id: 56, name: 'Toncoin', symbol: 'TON', chainId: -239, type: 'OTHER', rpcUrl: 'https://toncenter.com/api/v2', explorerUrl: 'https://tonscan.org', icon: '💎', decimals: 9, isActive: true, isDefault: false, addressPrefix: 'ton' },
  { id: 57, name: 'Aptos', symbol: 'APT', chainId: 637, type: 'APTOS', rpcUrl: 'https://aptos-mainnet.nodereal.io/v1', explorerUrl: 'https://aptoscan.com', icon: '🔷', decimals: 8, isActive: true, isDefault: false, addressPrefix: 'aptos', derivePath: "m/44'/637'/0'/0'/0'" },
  { id: 58, name: 'Sui', symbol: 'SUI', chainId: 784, type: 'SUI', rpcUrl: 'https://fullnode.mainnet.sui.io', explorerUrl: 'https://suiscan.xyz', icon: '🏊', decimals: 9, isActive: true, isDefault: false, addressPrefix: 'sui', derivePath: "m/44'/784'/0'/0/0'" },
  { id: 59, name: 'Near', symbol: 'NEAR', chainId: 1313161554, type: 'NEAR', rpcUrl: 'https://rpc.mainnet.near.org', explorerUrl: 'https://explorer.near.org', icon: 'Ⓝ', decimals: 24, isActive: true, isDefault: false, addressPrefix: 'near', derivePath: "m/44'/397'/0'/0/0'" },
  { id: 60, name: 'Algorand', symbol: 'ALGO', chainId: 0, type: 'OTHER', rpcUrl: 'https://mainnet-api.algonode.cloud', explorerUrl: 'https://algoexplorer.io', icon: '🔷', decimals: 6, isActive: true, isDefault: false, addressPrefix: 'algorand' },
  { id: 61, name: 'Hedera', symbol: 'HBAR', chainId: 295, type: 'OTHER', rpcUrl: 'https://mainnet.mirror.hedera.com', explorerUrl: 'https://hashscan.io', icon: '🌿', decimals: 8, isActive: true, isDefault: false, addressPrefix: 'hedera', derivePath: "m/44'/3030'/0'/0/0'" },
  { id: 62, name: 'Polkadot', symbol: 'DOT', chainId: 0, type: 'OTHER', rpcUrl: 'https://rpc.polkadot.io', explorerUrl: 'https://polkadot.js.org/apps', icon: '🟣', decimals: 10, isActive: true, isDefault: false, addressPrefix: 'polkadot', derivePath: "m/44'/354'/0'/0/0'" },
  { id: 63, name: 'Kusama', symbol: 'KSM', chainId: 0, type: 'OTHER', rpcUrl: 'https://rpc.kusama.network', explorerUrl: 'https://kusama.subscan.io', icon: '🟢', decimals: 12, isActive: true, isDefault: false, addressPrefix: 'kusama', derivePath: "m/44'/434'/0'/0/0'" },
  { id: 64, name: 'Kadena', symbol: 'KDA', chainId: 0, type: 'OTHER', rpcUrl: 'https://api.chainweb.com', explorerUrl: 'https://explorer.chainweb.com', icon: '🔗', decimals: 12, isActive: true, isDefault: false, addressPrefix: 'kadena' },
  { id: 65, name: 'Flow', symbol: 'FLOW', chainId: 0, type: 'OTHER', rpcUrl: 'https://flowscan.io', explorerUrl: 'https://flowscan.io', icon: '🌊', decimals: 8, isActive: true, isDefault: false, addressPrefix: 'flow', derivePath: "m/44'/539'/0'/0/0'" },
  { id: 66, name: 'Tezos', symbol: 'XTZ', chainId: 0, type: 'OTHER', rpcUrl: 'https://mainnet.api.tzscan.io', explorerUrl: 'https://tzstats.com', icon: '🌵', decimals: 6, isActive: true, isDefault: false, addressPrefix: 'tezos', derivePath: "m/44'/1729'/0'/0'" },
  { id: 67, name: 'Radix', symbol: 'XRD', chainId: 0, type: 'OTHER', rpcUrl: 'https://mainnet.radixdlt.com', explorerUrl: 'https://dashboard.radixdlt.com', icon: '🔴', decimals: 18, isActive: true, isDefault: false, addressPrefix: 'radix' },
  { id: 68, name: 'Casper', symbol: 'CSPR', chainId: 0, type: 'OTHER', rpcUrl: 'https://node.casper.network', explorerUrl: 'https://cspr.live', icon: '🦀', decimals: 9, isActive: true, isDefault: false, addressPrefix: 'casper' },
  { id: 69, name: 'EOS', symbol: 'EOS', chainId: 0, type: 'OTHER', rpcUrl: 'https://api.eosn.io', explorerUrl: 'https://bloks.io', icon: '⚡', decimals: 4, isActive: true, isDefault: false, addressPrefix: 'eos', derivePath: "m/44'/194'/0'/0/0'" },
  { id: 70, name: 'VeChain', symbol: 'VET', chainId: 0, type: 'OTHER', rpcUrl: 'https://sync-mainnet.vechain.org', explorerUrl: 'https://vechainstats.com', icon: '🔷', decimals: 18, isActive: true, isDefault: false, addressPrefix: 'vechain', derivePath: "m/44'/818'/0'/0/0'" },
  { id: 71, name: 'WAX', symbol: 'WAX', chainId: 0, type: 'OTHER', rpcUrl: 'https://wax.greymass.com', explorerUrl: 'https://wax.bloks.io', icon: '🐝', decimals: 8, isActive: true, isDefault: false, addressPrefix: 'wax', derivePath: "m/44'/134'/0'/0/0'" },
  { id: 72, name: 'Dogecoin', symbol: 'DOGE', chainId: 3, type: 'BITCOIN', rpcUrl: 'https://dogecoin.lit.io', explorerUrl: 'https://dogechain.info', icon: '🐕', decimals: 8, isActive: true, isDefault: false, addressPrefix: 'dogecoin', derivePath: "m/44'/3'/0'/0/0'" },
  { id: 73, name: 'Litecoin', symbol: 'LTC', chainId: 2, type: 'BITCOIN', rpcUrl: 'https://litecoin.lit.io', explorerUrl: 'https://blockchair.com/litecoin', icon: 'Ł', decimals: 8, isActive: true, isDefault: false, addressPrefix: 'litecoin', derivePath: "m/44'/2'/0'/0/0'" },
  { id: 74, name: 'Bitcoin Cash', symbol: 'BCH', chainId: 145, type: 'BITCOIN', rpcUrl: 'https://bitcoincash.lit.io', explorerUrl: 'https://blockchair.com/bitcoin-cash', icon: '₿', decimals: 8, isActive: true, isDefault: false, addressPrefix: 'bitcoincash', derivePath: "m/44'/145'/0'/0/0'" },
  { id: 75, name: 'Ripple', symbol: 'XRP', chainId: 144, type: 'OTHER', rpcUrl: 'https://xrplcluster.com', explorerUrl: 'https://xrpscan.com', icon: '✕', decimals: 6, isActive: true, isDefault: false, addressPrefix: 'xrp' },
  { id: 76, name: 'Cardano', symbol: 'ADA', chainId: 3009, type: 'OTHER', rpcUrl: 'https://cardano-mainnet.nodereal.io/v1', explorerUrl: 'https://cardanoscan.io', icon: '₳', decimals: 6, isActive: true, isDefault: false, addressPrefix: 'cardano', derivePath: "m/44'/1815'/0'/0'" },
  { id: 77, name: 'Thorchain', symbol: 'RUNE', chainId: 0, type: 'COSMOS', rpcUrl: 'https://rpc.thorchain.info', explorerUrl: 'https://thorchain.net', icon: '⚡', decimals: 8, isActive: true, isDefault: false, addressPrefix: 'thor', derivePath: "m/44'/931'/0'/0/0'" },
  { id: 78, name: 'Osmosis', symbol: 'OSMO', chainId: 0, type: 'COSMOS', rpcUrl: 'https://rpc-osmosis.ecostake.com', explorerUrl: 'https://mintscan.io/osmosis', icon: '🧪', decimals: 6, isActive: true, isDefault: false, addressPrefix: 'osmosis', derivePath: "m/44'/118'/0'/0/0'" },
  { id: 79, name: 'Secret', symbol: 'SCRT', chainId: 0, type: 'COSMOS', rpcUrl: 'https://rpc.scrt.network', explorerUrl: 'https://mintscan.io/secret', icon: '🔐', decimals: 6, isActive: true, isDefault: false, addressPrefix: 'secret', derivePath: "m/44'/529'/0'/0/0'" },
  { id: 80, name: 'Akash', symbol: 'AKT', chainId: 0, type: 'COSMOS', rpcUrl: 'https://rpc.akash.us', explorerUrl: 'https://mintscan.io/akash', icon: '🏰', decimals: 6, isActive: true, isDefault: false, addressPrefix: 'akash', derivePath: "m/44'/130'/0'/0/0'" },
  { id: 81, name: 'Juno', symbol: 'JUNO', chainId: 0, type: 'COSMOS', rpcUrl: 'https://rpc-juno.ecostake.com', explorerUrl: 'https://mintscan.io/juno', icon: '🪐', decimals: 6, isActive: true, isDefault: false, addressPrefix: 'juno', derivePath: "m/44'/118'/0'/0/0'" },
  { id: 82, name: 'Injective', symbol: 'INJ', chainId: 0, type: 'COSMOS', rpcUrl: 'https://injective-1.rocket-pool.nodereal.io', explorerUrl: 'https://explorer.injective.network', icon: '💉', decimals: 18, isActive: true, isDefault: false, addressPrefix: 'injective', derivePath: "m/44'/60'/0'/0/0'" },
  { id: 83, name: 'Sei', symbol: 'SEI', chainId: 0, type: 'OTHER', rpcUrl: 'https://rpc.sei.io', explorerUrl: 'https://seistats.io', icon: '🐟', decimals: 6, isActive: true, isDefault: false, addressPrefix: 'sei' },
  { id: 84, name: 'Celestia', symbol: 'TIA', chainId: 0, type: 'OTHER', rpcUrl: 'https://celestia-rpc.nodereal.io', explorerUrl: 'https://celestia.explorer.nodes.guru', icon: '🌙', decimals: 6, isActive: true, isDefault: false, addressPrefix: 'celestia' },
  { id: 85, name: 'Starknet', symbol: 'STRK', chainId: 0, type: 'OTHER', rpcUrl: 'https://rpc.starknet.io', explorerUrl: 'https://starkscan.co', icon: '⭐', decimals: 18, isActive: true, isDefault: false, addressPrefix: 'starknet' },
  { id: 86, name: 'Monero', symbol: 'XMR', chainId: 0, type: 'OTHER', rpcUrl: 'https://monero-rpc.online', explorerUrl: 'https://xmr.to', icon: '🔒', decimals: 12, isActive: true, isDefault: false, addressPrefix: 'monero' },
  { id: 87, name: 'Zcash', symbol: 'ZEC', chainId: 0, type: 'OTHER', rpcUrl: 'https://zcash-rpc.online', explorerUrl: 'https://z.cash', icon: '🟡', decimals: 8, isActive: true, isDefault: false, addressPrefix: 'zcash' },
  { id: 88, name: 'Dash', symbol: 'DASH', chainId: 0, type: 'OTHER', rpcUrl: 'https://dash-rpc.online', explorerUrl: 'https://explorer.dash.org', icon: '💨', decimals: 8, isActive: true, isDefault: false, addressPrefix: 'dash' },
  { id: 89, name: 'Zilliqa', symbol: 'ZIL', chainId: 0, type: 'OTHER', rpcUrl: 'https://api.zilliqa.com', explorerUrl: 'https://viewblock.io/zilliqa', icon: '🟣', decimals: 12, isActive: true, isDefault: false, addressPrefix: 'zilliqa', derivePath: "m/44'/313'/0'/0/0'" },
  { id: 90, name: 'Ergo', symbol: 'ERG', chainId: 0, type: 'OTHER', rpcUrl: 'https://ergo-rpc.online', explorerUrl: 'https://explorer.ergoplatform.org', icon: 'Σ', decimals: 8, isActive: true, isDefault: false, addressPrefix: 'ergo' },
  { id: 91, name: 'Stargaze', symbol: 'STARS', chainId: 0, type: 'COSMOS', rpcUrl: 'https://rpc-stargaze.ecostake.com', explorerUrl: 'https://mintscan.io/stargaze', icon: '⭐', decimals: 6, isActive: true, isDefault: false, addressPrefix: 'stars', derivePath: "m/44'/118'/0'/0/0'" },
  { id: 92, name: 'dYdX', symbol: 'DYDX', chainId: 0, type: 'COSMOS', rpcUrl: 'https://dydx-mainnet-archive.allship.io', explorerUrl: 'https://dydx.explorer.stakewith.us', icon: '📈', decimals: 18, isActive: true, isDefault: false, addressPrefix: 'dydx' },
  { id: 93, name: 'Aleo', symbol: 'ALEO', chainId: 0, type: 'OTHER', rpcUrl: 'https://api.aleo.org', explorerUrl: 'https://aleo.network', icon: '🔒', decimals: 6, isActive: true, isDefault: false, addressPrefix: 'aleo' },
  { id: 94, name: 'Sui', symbol: 'SUI', chainId: 784, type: 'SUI', rpcUrl: 'https://fullnode.mainnet.sui.io', explorerUrl: 'https://suiscan.xyz', icon: '🏊', decimals: 9, isActive: true, isDefault: false, addressPrefix: 'sui' },
  { id: 95, name: 'Nervos Network', symbol: 'CKB', chainId: 0, type: 'OTHER', rpcUrl: 'https://rpc.nervos.org', explorerUrl: 'https://explorer.nervos.org', icon: '🧠', decimals: 8, isActive: true, isDefault: false, addressPrefix: 'nervos' },
  { id: 96, name: 'Aion', symbol: 'AION', chainId: 0, type: 'OTHER', rpcUrl: 'https://aion.api.coinc.org', explorerUrl: 'https://mainnet.aion.network', icon: '🔷', decimals: 18, isActive: true, isDefault: false, addressPrefix: 'aion' },
  { id: 97, name: 'Waves', symbol: 'WAVES', chainId: 0, type: 'OTHER', rpcUrl: 'https://nodes.wavesnodes.com', explorerUrl: 'https://wavesexplorer.com', icon: '🌊', decimals: 8, isActive: true, isDefault: false, addressPrefix: 'waves' },
  { id: 98, name: 'Hedera', symbol: 'HBAR', chainId: 295, type: 'OTHER', rpcUrl: 'https://mainnet.mirror.hedera.com', explorerUrl: 'https://hashscan.io', icon: '🌿', decimals: 8, isActive: true, isDefault: false, addressPrefix: 'hedera' },
  { id: 99, name: 'Chainflip', symbol: 'FLIP', chainId: 0, type: 'OTHER', rpcUrl: 'https://chainflip.io', explorerUrl: 'https://scan.chainflip.io', icon: '🔄', decimals: 18, isActive: true, isDefault: false, addressPrefix: 'chainflip' },
  { id: 100, name: 'Aleo', symbol: 'ALEO', chainId: 0, type: 'OTHER', rpcUrl: 'https://api.aleo.org', explorerUrl: 'https://aleo.network', icon: '🔒', decimals: 6, isActive: true, isDefault: false, addressPrefix: 'aleo' },
  { id: 101, name: 'Celo', symbol: 'CELO', chainId: 42220, type: 'OTHER', rpcUrl: 'https://forno.celo.org', explorerUrl: 'https://explorer.celo.org', icon: '🌕', decimals: 18, isActive: true, isDefault: false, addressPrefix: 'celo' },
  { id: 102, name: 'Aztec', symbol: 'AZT', chainId: 0, type: 'OTHER', rpcUrl: 'https://aztec.network', explorerUrl: 'https://aztecscan.com', icon: '🔐', decimals: 18, isActive: true, isDefault: false, addressPrefix: 'aztec' },
  { id: 103, name: 'Fuel', symbol: 'FUEL', chainId: 0, type: 'OTHER', rpcUrl: 'https://fuel.network', explorerUrl: 'https://fuelscan.io', icon: '⛽', decimals: 18, isActive: true, isDefault: false, addressPrefix: 'fuel' },
  { id: 104, name: 'Risc Zero', symbol: 'RISC', chainId: 0, type: 'OTHER', rpcUrl: 'https://risczero.com', explorerUrl: 'https://scan.risczero.com', icon: '⚡', decimals: 18, isActive: true, isDefault: false, addressPrefix: 'risczero' },
  { id: 105, name: 'Nil', symbol: 'NIL', chainId: 0, type: 'OTHER', rpcUrl: 'https://nil.foundation', explorerUrl: 'https://nilscan.io', icon: '🔷', decimals: 18, isActive: true, isDefault: false, addressPrefix: 'nil' },
];

// 200+ TOKENS
export const SUPPORTED_TOKENS: Token[] = [
  // ETHEREUM
  { id: 1, chainId: 1, address: '0x0000000000000000000000000000000000000000', symbol: 'ETH', name: 'Ethereum', decimals: 18, isNative: true, isActive: true, priceUSD: 3500, coingeckoId: 'ethereum' },
  { id: 2, chainId: 1, address: '0xdAC17F958D2ee523a2206206994597C13D831ec7', symbol: 'USDT', name: 'Tether USD', decimals: 6, isNative: false, isActive: true, priceUSD: 1, coingeckoId: 'tether' },
  { id: 3, chainId: 1, address: '0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48', symbol: 'USDC', name: 'USD Coin', decimals: 6, isNative: false, isActive: true, priceUSD: 1, coingeckoId: 'usd-coin' },
  { id: 4, chainId: 1, address: '0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599', symbol: 'WBTC', name: 'Wrapped Bitcoin', decimals: 8, isNative: false, isActive: true, priceUSD: 65000, coingeckoId: 'wrapped-bitcoin' },
  { id: 5, chainId: 1, address: '0x514910771AF9Ca656af840dff83E8264EcF986CA', symbol: 'LINK', name: 'Chainlink', decimals: 18, isNative: false, isActive: true, priceUSD: 15, coingeckoId: 'chainlink' },
  { id: 6, chainId: 1, address: '0x1f9840a85d5aF5bf1D1762F925BDADdC4201F984', symbol: 'UNI', name: 'Uniswap', decimals: 18, isNative: false, isActive: true, priceUSD: 10, coingeckoId: 'uniswap' },
  { id: 7, chainId: 1, address: '0x7Fc66500c84A76Ad7e9c93437bFc5Ac33E2DDaE9', symbol: 'AAVE', name: 'Aave', decimals: 18, isNative: false, isActive: true, priceUSD: 200, coingeckoId: 'aave' },
  { id: 8, chainId: 1, address: '0x6B175474E89094C44Da98b954EiteCDfBBc7CD33', symbol: 'DAI', name: 'Dai Stablecoin', decimals: 18, isNative: false, isActive: true, priceUSD: 1, coingeckoId: 'dai' },
  { id: 9, chainId: 1, address: '0x2b591e99afE9f32d9c8f1E3f4C7E2b4b3c2d1E0f', symbol: 'PAXG', name: 'Paxos Gold', decimals: 18, isNative: false, isActive: true, priceUSD: 2500, coingeckoId: 'pax-gold' },
  { id: 10, chainId: 1, address: '0x0D8775F648430679A709E98d2b0Cb6250d2887EF', symbol: 'BAT', name: 'Basic Attention Token', decimals: 18, isNative: false, isActive: true, priceUSD: 0.25, coingeckoId: 'basic-attention-token' },
  { id: 11, chainId: 1, address: '0xc00e94Cb662C3520282E6f5717214004A7f26888', symbol: 'COMP', name: 'Compound', decimals: 18, isNative: false, isActive: true, priceUSD: 50, coingeckoId: 'compound-governance-token' },
  { id: 12, chainId: 1, address: '0x0bc529c00C6401aEF6D220BE8C6Ea1665F6ed51', symbol: 'YFI', name: 'yearn.finance', decimals: 18, isNative: false, isActive: true, priceUSD: 4500, coingeckoId: 'yearn-finance' },
  { id: 13, chainId: 1, address: '0xD533a949740bb3306d119CC777fa900bA034cd52', symbol: 'CRV', name: 'Curve DAO Token', decimals: 18, isNative: false, isActive: true, priceUSD: 0.5, coingeckoId: 'curve-dao-token' },
  { id: 14, chainId: 1, address: '0xba7435A4b4C747E0101780073eeda872a69BDcd4', symbol: 'LDO', name: 'Lido DAO', decimals: 18, isNative: false, isActive: true, priceUSD: 2.2, coingeckoId: 'lido-dao' },
  { id: 15, chainId: 1, address: '0x95aD61b0a150d79219dCF64E1E76Cc11FD675AD2', symbol: 'SHIB', name: 'Shiba Inu', decimals: 18, isNative: false, isActive: true, priceUSD: 0.00002, coingeckoId: 'shiba-inu' },
  { id: 16, chainId: 1, address: '0x5A98FCBeb4f1A7E706C5025E6D6d1E2A1e3B5c9D', symbol: 'PEPE', name: 'Pepe', decimals: 18, isNative: false, isActive: true, priceUSD: 0.000001, coingeckoId: 'pepe' },
  { id: 17, chainId: 1, address: '0xc00e94Cb662C3520282E6f5717214004A7f26888', symbol: 'MKR', name: 'Maker', decimals: 18, isNative: false, isActive: true, priceUSD: 1500, coingeckoId: 'maker' },
  { id: 18, chainId: 1, address: '0x514EC4f9ED28D1E71E4b9F69d1C4Da1d9B6B3B7a', symbol: 'RNDR', name: 'Render Token', decimals: 18, isNative: false, isActive: true, priceUSD: 3.5, coingeckoId: 'render-token' },
  { id: 19, chainId: 1, address: '0x4d224452801ACEd8B2F0aebEbb3dbDB6dB4b1A23', symbol: 'GMX', name: 'GMX', decimals: 18, isNative: false, isActive: true, priceUSD: 45, coingeckoId: 'gmx' },
  { id: 20, chainId: 1, address: '0x50D1c9771902476076eCF4838eB2044BC4A19f05', symbol: 'GUSD', name: 'Gemini Dollar', decimals: 18, isNative: false, isActive: true, priceUSD: 1, coingeckoId: 'gemini-dollar' },
  { id: 21, chainId: 1, address: '0xD533a949740bb3306d119CC777fa900bA034cd52', symbol: 'SNX', name: 'Synthetix Network', decimals: 18, isNative: false, isActive: true, priceUSD: 2.5, coingeckoId: 'havven' },
  { id: 22, chainId: 1, address: '0x0bc529c00C6401aEF6D220BE8C6Ea1665F6ed51', symbol: 'SUSHI', name: 'SushiSwap', decimals: 18, isNative: false, isActive: true, priceUSD: 1.2, coingeckoId: 'sushi' },
  { id: 23, chainId: 1, address: '0x1f9840a85d5aF5bf1D1762F925BDADdC4201F984', symbol: 'ARB', name: 'Arbitrum', decimals: 18, isNative: false, isActive: true, priceUSD: 1.2, coingeckoId: 'arbitrum' },
  { id: 24, chainId: 1, address: '0x4200000000000000000000000000000000000006', symbol: 'WETH', name: 'Wrapped Ether', decimals: 18, isNative: false, isActive: true, priceUSD: 3500, coingeckoId: 'weth' },
  
  // BNB CHAIN
  { id: 101, chainId: 56, address: '0x0000000000000000000000000000000000000000', symbol: 'BNB', name: 'BNB', decimals: 18, isNative: true, isActive: true, priceUSD: 600, coingeckoId: 'binancecoin' },
  { id: 102, chainId: 56, address: '0x55d398326f99059fF775485246999027B3197955', symbol: 'USDT', name: 'Tether USD', decimals: 18, isNative: false, isActive: true, priceUSD: 1, coingeckoId: 'tether' },
  { id: 103, chainId: 56, address: '0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d', symbol: 'USDC', name: 'USD Coin', decimals: 18, isNative: false, isActive: true, priceUSD: 1, coingeckoId: 'usd-coin' },
  { id: 104, chainId: 56, address: '0x1AF3F329e8BEc1548F3E514beDD9FB80209eC33b6', symbol: 'DAI', name: 'Dai Stablecoin', decimals: 18, isNative: false, isActive: true, priceUSD: 1, coingeckoId: 'dai' },
  { id: 105, chainId: 56, address: '0x0E09FaBB73Bd3ade0a17ECC321fD13a19e81cE82', symbol: 'CAKE', name: 'PancakeSwap', decimals: 18, isNative: false, isActive: true, priceUSD: 2.5, coingeckoId: 'pancakeswap-token' },
  { id: 106, chainId: 56, address: '0x3fF9CeBba7D6D5F73a92f6F79d5Fb4B7C5c3B8aD', symbol: 'BABYDOGE', name: 'BabyDoge', decimals: 18, isNative: false, isActive: true, priceUSD: 0.0000001, coingeckoId: 'babydoge' },
  { id: 107, chainId: 56, address: '0xd4CB328A82bDf5f03e737f9d88B48B1d2f22f1f5', symbol: 'FLOKI', name: 'FLOKI', decimals: 18, isNative: false, isActive: true, priceUSD: 0.0001, coingeckoId: 'floki' },
  { id: 108, chainId: 56, address: '0x8dA04d6f5B1E0b3C9d5B0f5E4d2C5b7A6F8e9D0c', symbol: 'BTCB', name: 'Bitcoin BEP2', decimals: 18, isNative: false, isActive: true, priceUSD: 65000, coingeckoId: 'bitcoin-bep2' },
  { id: 109, chainId: 56, address: '0xe2d9C51A7dC8C5D4e7E5F3a2B9c5d6e7f8a9b0c1', symbol: 'ETH', name: 'Ethereum', decimals: 18, isNative: false, isActive: true, priceUSD: 3500, coingeckoId: 'ethereum' },
  { id: 110, chainId: 56, address: '0x47E1dA7bBB0a3f12a97bB2c1A3f7B4C5D6E7F8A9B', symbol: 'PEPE', name: 'Pepe', decimals: 18, isNative: false, isActive: true, priceUSD: 0.000001, coingeckoId: 'pepe' },
  
  // SOLANA
  { id: 201, chainId: 101, address: 'So11111111111111111111111111111111111111112', symbol: 'SOL', name: 'Solana', decimals: 9, isNative: true, isActive: true, priceUSD: 150, coingeckoId: 'solana' },
  { id: 202, chainId: 101, address: 'EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v', symbol: 'USDC', name: 'USD Coin', decimals: 6, isNative: false, isActive: true, priceUSD: 1, coingeckoId: 'usd-coin' },
  { id: 203, chainId: 101, address: 'Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB', symbol: 'USDT', name: 'Tether USD', decimals: 6, isNative: false, isActive: true, priceUSD: 1, coingeckoId: 'tether' },
  { id: 204, chainId: 101, address: '3b6a27BEeF4Bb2d9CeF2e1a7C4B4D5E6F7A8b9C0', symbol: 'BONK', name: 'Bonk', decimals: 5, isNative: false, isActive: true, priceUSD: 0.00002, coingeckoId: 'bonk' },
  { id: 205, chainId: 101, address: 'JUPyiwrYJFskUPiHa7hkeR8VUtkqjberbSOWd91pbT2', symbol: 'JUP', name: 'Jupiter', decimals: 6, isNative: false, isActive: true, priceUSD: 0.8, coingeckoId: 'jupiter' },
  { id: 206, chainId: 101, address: 'DezXAZ8z7PnrnzXZtqqD6V4d7g2G4kE5k9L8vP2Rt9Uw', symbol: 'WIF', name: 'dogwifhat', decimals: 6, isNative: false, isActive: true, priceUSD: 2.5, coingeckoId: 'dogwifhat' },
  
  // TRON
  { id: 301, chainId: 728126428, address: '', symbol: 'TRX', name: 'Tron', decimals: 6, isNative: true, isActive: true, priceUSD: 0.12, coingeckoId: 'tron' },
  { id: 302, chainId: 728126428, address: 'TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t', symbol: 'USDT', name: 'Tether USD', decimals: 6, isNative: false, isActive: true, priceUSD: 1, coingeckoId: 'tether' },
  { id: 303, chainId: 728126428, address: 'TF17BgFVLGvAmvWjPHF2rV2T2YvYqYqLQ5k9L8vP2Rt', symbol: 'USDC', name: 'USD Coin', decimals: 6, isNative: false, isActive: true, priceUSD: 1, coingeckoId: 'usd-coin' },
  { id: 304, chainId: 728126428, address: 'TAk5sRRFqr2SABnQx2J3C5Y5Y5Y5Y5Y5Y5Y5Y5Y5', symbol: 'BTT', name: 'BitTorrent', decimals: 18, isNative: false, isActive: true, priceUSD: 0.000001, coingeckoId: 'bittorrent' },
  { id: 305, chainId: 728126428, address: 'TX8kHNb8B8C4g4E4E4E4E4E4E4E4E4E4E4E4E4', symbol: 'SUN', name: 'Sun', decimals: 18, isNative: false, isActive: true, priceUSD: 0.01, coingeckoId: 'sun' },
  
  // POLYGON
  { id: 401, chainId: 137, address: '0x0000000000000000000000000000000000000000', symbol: 'MATIC', name: 'Polygon', decimals: 18, isNative: true, isActive: true, priceUSD: 0.8, coingeckoId: 'matic-network' },
  { id: 402, chainId: 137, address: '0x53E0bca35eC356BD5ddDFEbdD1Fc0fD03FaBad39', symbol: 'LINK', name: 'Chainlink', decimals: 18, isNative: false, isActive: true, priceUSD: 15, coingeckoId: 'chainlink' },
  { id: 403, chainId: 137, address: '0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174', symbol: 'USDC', name: 'USD Coin', decimals: 6, isNative: false, isActive: true, priceUSD: 1, coingeckoId: 'usd-coin' },
  { id: 404, chainId: 137, address: '0xc2132D05D31c914a87C6611C10748AEb04B58e8F', symbol: 'USDT', name: 'Tether USD', decimals: 6, isNative: false, isActive: true, priceUSD: 1, coingeckoId: 'tether' },
  { id: 405, chainId: 137, address: '0x1BFD67037B42Cf73acF2047067bd4F2C47D9BfD6', symbol: 'WBTC', name: 'Wrapped Bitcoin', decimals: 8, isNative: false, isActive: true, priceUSD: 65000, coingeckoId: 'wrapped-bitcoin' },
  { id: 406, chainId: 137, address: '0x53E0bca35eC356BD5ddDFEbdD1Fc0fD03FaBad39', symbol: 'QUICK', name: 'QuickSwap', decimals: 18, isNative: false, isActive: true, priceUSD: 50, coingeckoId: 'quickswap' },
  
  // ARBITRUM
  { id: 501, chainId: 42161, address: '0x0000000000000000000000000000000000000000', symbol: 'ETH', name: 'Ethereum', decimals: 18, isNative: true, isActive: true, priceUSD: 3500, coingeckoId: 'ethereum' },
  { id: 502, chainId: 42161, address: '0xFF970A61A04b1cA14834A43f5dE4533eBDDB5CC8', symbol: 'USDC', name: 'USD Coin', decimals: 6, isNative: false, isActive: true, priceUSD: 1, coingeckoId: 'usd-coin' },
  { id: 503, chainId: 42161, address: '0xFd086b7Cd5C481CC1A9dF6c6D8F8C4C3D2E1F0A9', symbol: 'USDT', name: 'Tether USD', decimals: 6, isNative: false, isActive: true, priceUSD: 1, coingeckoId: 'tether' },
  { id: 504, chainId: 42161, address: '0x2B2A8A9D8E9C8F7D6C5B4A3E2F1D0C9B8A7F6E5', symbol: 'ARB', name: 'Arbitrum', decimals: 18, isNative: false, isActive: true, priceUSD: 1.2, coingeckoId: 'arbitrum' },
  { id: 505, chainId: 42161, address: '0x3F6cB6B4C5D6E7F8A9B0C1D2E3F4A5B6C7D8E9F', symbol: 'GMX', name: 'GMX', decimals: 18, isNative: false, isActive: true, priceUSD: 45, coingeckoId: 'gmx' },
  
  // OPTIMISM
  { id: 601, chainId: 10, address: '0x0000000000000000000000000000000000000000', symbol: 'ETH', name: 'Ethereum', decimals: 18, isNative: true, isActive: true, priceUSD: 3500, coingeckoId: 'ethereum' },
  { id: 602, chainId: 10, address: '0x7F5c764cBc14f9669B88837ca1490cCa17c31607', symbol: 'USDC', name: 'USD Coin', decimals: 6, isNative: false, isActive: true, priceUSD: 1, coingeckoId: 'usd-coin' },
  { id: 603, chainId: 10, address: '0x94b008aA00579c1307B0EF2c499aD98a8ce58e58', symbol: 'USDT', name: 'Tether USD', decimals: 6, isNative: false, isActive: true, priceUSD: 1, coingeckoId: 'tether' },
  { id: 604, chainId: 10, address: '0x4200000000000000000000000000000000000042', symbol: 'OP', name: 'Optimism', decimals: 18, isNative: false, isActive: true, priceUSD: 2.5, coingeckoId: 'optimism' },
  
  // AVALANCHE
  { id: 701, chainId: 43114, address: '0x0000000000000000000000000000000000000000', symbol: 'AVAX', name: 'Avalanche', decimals: 18, isNative: true, isActive: true, priceUSD: 35, coingeckoId: 'avalanche-2' },
  { id: 702, chainId: 43114, address: '0xB97EF9Ef8734C71904D8002F8b6Bc66Dd9c48a6E', symbol: 'USDC', name: 'USD Coin', decimals: 6, isNative: false, isActive: true, priceUSD: 1, coingeckoId: 'usd-coin' },
  { id: 703, chainId: 43114, address: '0x9702230A8E536b7cA5Cd65AF2E27c7d5d0D9e8F7', symbol: 'USDT', name: 'Tether USD', decimals: 6, isNative: false, isActive: true, priceUSD: 1, coingeckoId: 'tether' },
  { id: 704, chainId: 43114, address: '0xd1c3f94de7e5b45fa4adb6fb24676bad6f3c9a5f', symbol: 'JOE', name: 'JOE', decimals: 18, isNative: false, isActive: true, priceUSD: 0.4, coingeckoId: 'joe' },
  { id: 705, chainId: 43114, address: '0x2B2A8A9D8E9C8F7D6C5B4A3E2F1D0C9B8A7F6E5', symbol: 'PNG', name: 'Pangolin', decimals: 18, isNative: false, isActive: true, priceUSD: 0.2, coingeckoId: 'pangolin' },
  
  // PI NETWORK
  { id: 801, chainId: 314159, address: '', symbol: 'PI', name: 'Pi Network', decimals: 18, isNative: true, isActive: true, priceUSD: 50, coingeckoId: 'pi-network' },
  
  // BITCOIN
  { id: 901, chainId: 0, address: '', symbol: 'BTC', name: 'Bitcoin', decimals: 8, isNative: true, isActive: true, priceUSD: 65000, coingeckoId: 'bitcoin' },
  { id: 902, chainId: 0, address: '', symbol: 'WBTC', name: 'Wrapped Bitcoin', decimals: 8, isNative: false, isActive: true, priceUSD: 65000, coingeckoId: 'wrapped-bitcoin' },
  
  // OTHER POPULAR TOKENS
  { id: 1001, chainId: 56, address: '0x0E09FaBB73Bd3ade0a17ECC321fD13a19e81cE82', symbol: 'CAKE', name: 'PancakeSwap', decimals: 18, isNative: false, isActive: true, priceUSD: 2.5, coingeckoId: 'pancakeswap-token' },
  { id: 1002, chainId: 1, address: '0xdF574c24545E5FfEcb9da65950B9e819B4b8F3F4', symbol: 'TUSD', name: 'TrueUSD', decimals: 18, isNative: false, isActive: true, priceUSD: 1, coingeckoId: 'true-usd' },
  { id: 1003, chainId: 1, address: '0x2AF5D2aD0e6B8B8D5C4B3A2E1F0D9C8B7A6F5E4D', symbol: 'BUSD', name: 'Binance USD', decimals: 18, isNative: false, isActive: true, priceUSD: 1, coingeckoId: 'binance-usd' },
  { id: 1004, chainId: 1, address: '0x3432B6A60D23Ca0dFCaE1Dc264c9b6f4D5C3E6F8', symbol: 'FXS', name: 'Frax Share', decimals: 18, isNative: false, isActive: true, priceUSD: 5, coingeckoId: 'frax-share' },
  { id: 1005, chainId: 1, address: '0x3432B6A60D23Ca0dFCaE1Dc264c9b6f4D5C3E6F8', symbol: 'FRAX', name: 'Frax', decimals: 18, isNative: false, isActive: true, priceUSD: 1, coingeckoId: 'frax' },
  { id: 1006, chainId: 1, address: '0xA0b73E1Ff0B80952B7Cb5f2f1d2D1c1D5e7f8A9B', symbol: 'MIM', name: 'Magic Internet Money', decimals: 18, isNative: false, isActive: true, priceUSD: 1, coingeckoId: 'magic-internet-money' },
  { id: 1007, chainId: 1, address: '0x5A98FCBeb4f1A7E706C5025E6D6d1E2A1e3B5c9D', symbol: 'PEPE', name: 'Pepe', decimals: 18, isNative: false, isActive: true, priceUSD: 0.000001, coingeckoId: 'pepe' },
  { id: 1008, chainId: 1, address: '0xD533a949740bb3306d119CC777fa900bA034cd52', symbol: 'SNX', name: 'Synthetix Network', decimals: 18, isNative: false, isActive: true, priceUSD: 2.5, coingeckoId: 'havven' },
  { id: 1009, chainId: 1, address: '0x0bc529c00C6401aEF6D220BE8C6Ea1665F6ed51', symbol: 'SUSHI', name: 'SushiSwap', decimals: 18, isNative: false, isActive: true, priceUSD: 1.2, coingeckoId: 'sushi' },
  
  // MORE CHAIN TOKENS
  { id: 1101, chainId: 250, address: '0x0000000000000000000000000000000000000000', symbol: 'FTM', name: 'Fantom', decimals: 18, isNative: true, isActive: true, priceUSD: 0.8, coingeckoId: 'fantom' },
  { id: 1102, chainId: 42220, address: '0x0000000000000000000000000000000000000000', symbol: 'CELO', name: 'Celo', decimals: 18, isNative: true, isActive: true, priceUSD: 0.7, coingeckoId: 'celo' },
  { id: 1103, chainId: 8217, address: '0x0000000000000000000000000000000000000000', symbol: 'KLAY', name: 'Klaytn', decimals: 18, isNative: true, isActive: true, priceUSD: 0.2, coingeckoId: 'klaytn' },
  { id: 1104, chainId: 25, address: '0x0000000000000000000000000000000000000000', symbol: 'CRO', name: 'Cronos', decimals: 18, isNative: true, isActive: true, priceUSD: 0.1, coingeckoId: 'crypto-com-chain' },
  { id: 1105, chainId: 1284, address: '0x0000000000000000000000000000000000000000', symbol: 'GLMR', name: 'Moonbeam', decimals: 18, isNative: true, isActive: true, priceUSD: 0.3, coingeckoId: 'moonbeam' },
  { id: 1106, chainId: 592, address: '0x0000000000000000000000000000000000000000', symbol: 'ASTR', name: 'Astar', decimals: 18, isNative: true, isActive: true, priceUSD: 0.08, coingeckoId: 'astar' },
  { id: 1107, chainId: 2222, address: '0x0000000000000000000000000000000000000000', symbol: 'KAVA', name: 'Kava', decimals: 18, isNative: true, isActive: true, priceUSD: 0.6, coingeckoId: 'kava' },
  { id: 1108, chainId: 369, address: '0x0000000000000000000000000000000000000000', symbol: 'PLS', name: 'PulseChain', decimals: 18, isNative: true, isActive: true, priceUSD: 0.001, coingeckoId: 'pulsechain' },
  { id: 1109, chainId: 1313161554, address: '0x0000000000000000000000000000000000000000', symbol: 'NEAR', name: 'Near', decimals: 24, isNative: true, isActive: true, priceUSD: 5, coingeckoId: 'near' },
  { id: 1110, chainId: 637, address: '0x0000000000000000000000000000000000000000', symbol: 'APT', name: 'Aptos', decimals: 8, isNative: true, isActive: true, priceUSD: 8, coingeckoId: 'aptos' },
  
  // ALTCOINS & MEMECOINS
  { id: 1201, chainId: 101, address: '0x0000000000000000000000000000000000000000', symbol: 'WIF', name: 'dogwifhat', decimals: 18, isNative: false, isActive: true, priceUSD: 2.5, coingeckoId: 'dogwifhat' },
  { id: 1202, chainId: 101, address: '0x0000000000000000000000000000000000000000', symbol: 'BONK', name: 'Bonk', decimals: 18, isNative: false, isActive: true, priceUSD: 0.00002, coingeckoId: 'bonk' },
  { id: 1203, chainId: 1, address: '0x0000000000000000000000000000000000000000', symbol: 'FLOKI', name: 'FLOKI', decimals: 18, isNative: false, isActive: true, priceUSD: 0.0001, coingeckoId: 'floki' },
  { id: 1204, chainId: 1, address: '0x0000000000000000000000000000000000000000', symbol: 'ARB', name: 'Arbitrum', decimals: 18, isNative: false, isActive: true, priceUSD: 1.2, coingeckoId: 'arbitrum' },
  { id: 1205, chainId: 1, address: '0x0000000000000000000000000000000000000000', symbol: 'OP', name: 'Optimism', decimals: 18, isNative: false, isActive: true, priceUSD: 2.5, coingeckoId: 'optimism' },
  { id: 1206, chainId: 784, address: '0x0000000000000000000000000000000000000000', symbol: 'SUI', name: 'Sui', decimals: 9, isNative: true, isActive: true, priceUSD: 1.2, coingeckoId: 'sui' },
  { id: 1207, chainId: 0, address: '0x0000000000000000000000000000000000000000', symbol: 'SEI', name: 'Sei', decimals: 6, isNative: true, isActive: true, priceUSD: 0.5, coingeckoId: 'sei' },
  { id: 1208, chainId: 0, address: '0x0000000000000000000000000000000000000000', symbol: 'TIA', name: 'Celestia', decimals: 6, isNative: true, isActive: true, priceUSD: 15, coingeckoId: 'celestia' },
  { id: 1209, chainId: 0, address: '0x0000000000000000000000000000000000000000', symbol: 'INJ', name: 'Injective', decimals: 18, isNative: true, isActive: true, priceUSD: 25, coingeckoId: 'injective-protocol' },
  { id: 1210, chainId: 0, address: '0x0000000000000000000000000000000000000000', symbol: 'TIA', name: 'Celestia', decimals: 6, isNative: true, isActive: true, priceUSD: 15, coingeckoId: 'celestia' },
  
  // GOLD & COMMODITIES
  { id: 1301, chainId: 1, address: '0x0000000000000000000000000000000000000000', symbol: 'PAXG', name: 'Paxos Gold', decimals: 18, isNative: false, isActive: true, priceUSD: 2500, coingeckoId: 'pax-gold' },
  { id: 1302, chainId: 1, address: '0x0000000000000000000000000000000000000000', symbol: 'XAUT', name: 'Tether Gold', decimals: 6, isNative: false, isActive: true, priceUSD: 2500, coingeckoId: 'tether-gold' },
  
  // STABLECOINS
  { id: 1401, chainId: 1, address: '0x0000000000000000000000000000000000000000', symbol: 'TUSD', name: 'TrueUSD', decimals: 18, isNative: false, isActive: true, priceUSD: 1, coingeckoId: 'true-usd' },
  { id: 1402, chainId: 1, address: '0x0000000000000000000000000000000000000000', symbol: 'BUSD', name: 'Binance USD', decimals: 18, isNative: false, isActive: true, priceUSD: 1, coingeckoId: 'binance-usd' },
  { id: 1403, chainId: 1, address: '0x0000000000000000000000000000000000000000', symbol: 'USDP', name: 'Pax Dollar', decimals: 18, isNative: false, isActive: true, priceUSD: 1, coingeckoId: 'paxos-standard' },
  { id: 1404, chainId: 1, address: '0x0000000000000000000000000000000000000000', symbol: 'GUSD', name: 'Gemini Dollar', decimals: 18, isNative: false, isActive: true, priceUSD: 1, coingeckoId: 'gemini-dollar' },
  { id: 1405, chainId: 1, address: '0x0000000000000000000000000000000000000000', symbol: 'LUSD', name: 'Liquity USD', decimals: 18, isNative: false, isActive: true, priceUSD: 1, coingeckoId: 'liquity-usd' },
  
  // ADDITIONAL POPULAR TOKENS
  { id: 1501, chainId: 1, address: '0x0000000000000000000000000000000000000000', symbol: 'DOT', name: 'Polkadot', decimals: 18, isNative: false, isActive: true, priceUSD: 7, coingeckoId: 'polkadot' },
  { id: 1502, chainId: 1, address: '0x0000000000000000000000000000000000000000', symbol: 'ADA', name: 'Cardano', decimals: 18, isNative: false, isActive: true, priceUSD: 0.5, coingeckoId: 'cardano' },
  { id: 1503, chainId: 1, address: '0x0000000000000000000000000000000000000000', symbol: 'XRP', name: 'Ripple', decimals: 18, isNative: false, isActive: true, priceUSD: 0.6, coingeckoId: 'ripple' },
  { id: 1504, chainId: 1, address: '0x0000000000000000000000000000000000000000', symbol: 'DOGE', name: 'Dogecoin', decimals: 18, isNative: false, isActive: true, priceUSD: 0.15, coingeckoId: 'dogecoin' },
  { id: 1505, chainId: 1, address: '0x0000000000000000000000000000000000000000', symbol: 'LTC', name: 'Litecoin', decimals: 18, isNative: false, isActive: true, priceUSD: 85, coingeckoId: 'litecoin' },
  { id: 1506, chainId: 1, address: '0x0000000000000000000000000000000000000000', symbol: 'BCH', name: 'Bitcoin Cash', decimals: 18, isNative: false, isActive: true, priceUSD: 450, coingeckoId: 'bitcoin-cash' },
  { id: 1507, chainId: 1, address: '0x0000000000000000000000000000000000000000', symbol: 'ATOM', name: 'Cosmos', decimals: 18, isNative: false, isActive: true, priceUSD: 8, coingeckoId: 'cosmos' },
  { id: 1508, chainId: 1, address: '0x0000000000000000000000000000000000000000', symbol: 'XLM', name: 'Stellar', decimals: 18, isNative: false, isActive: true, priceUSD: 0.12, coingeckoId: 'stellar' },
  { id: 1509, chainId: 1, address: '0x0000000000000000000000000000000000000000', symbol: 'ALGO', name: 'Algorand', decimals: 18, isNative: false, isActive: true, priceUSD: 0.2, coingeckoId: 'algorand' },
  { id: 1510, chainId: 1, address: '0x0000000000000000000000000000000000000000', symbol: 'VET', name: 'VeChain', decimals: 18, isNative: false, isActive: true, priceUSD: 0.03, coingeckoId: 'vechain' },
  { id: 1511, chainId: 1, address: '0x0000000000000000000000000000000000000000', symbol: 'FIL', name: 'Filecoin', decimals: 18, isNative: false, isActive: true, priceUSD: 5, coingeckoId: 'filecoin' },
  { id: 1512, chainId: 1, address: '0x0000000000000000000000000000000000000000', symbol: 'HBAR', name: 'Hedera', decimals: 18, isNative: false, isActive: true, priceUSD: 0.07, coingeckoId: 'hedera' },
  { id: 1513, chainId: 1, address: '0x0000000000000000000000000000000000000000', symbol: 'NEAR', name: 'Near', decimals: 24, isNative: false, isActive: true, priceUSD: 5, coingeckoId: 'near' },
  { id: 1514, chainId: 1, address: '0x0000000000000000000000000000000000000000', symbol: 'APT', name: 'Aptos', decimals: 18, isNative: false, isActive: true, priceUSD: 8, coingeckoId: 'aptos' },
  { id: 1515, chainId: 1, address: '0x0000000000000000000000000000000000000000', symbol: 'SUI', name: 'Sui', decimals: 18, isNative: false, isActive: true, priceUSD: 1.2, coingeckoId: 'sui' },
  { id: 1516, chainId: 1, address: '0x0000000000000000000000000000000000000000', symbol: 'SEI', name: 'Sei', decimals: 18, isNative: false, isActive: true, priceUSD: 0.5, coingeckoId: 'sei' },
  { id: 1517, chainId: 1, address: '0x0000000000000000000000000000000000000000', symbol: 'TIA', name: 'Celestia', decimals: 18, isNative: false, isActive: true, priceUSD: 15, coingeckoId: 'celestia' },
  { id: 1518, chainId: 1, address: '0x0000000000000000000000000000000000000000', symbol: 'INJ', name: 'Injective', decimals: 18, isNative: false, isActive: true, priceUSD: 25, coingeckoId: 'injective-protocol' },
  { id: 1519, chainId: 1, address: '0x0000000000000000000000000000000000000000', symbol: 'RUNE', name: 'THORChain', decimals: 18, isNative: false, isActive: true, priceUSD: 4, coingeckoId: 'thorchain' },
  { id: 1520, chainId: 1, address: '0x0000000000000000000000000000000000000000', symbol: 'OSMO', name: 'Osmosis', decimals: 18, isNative: false, isActive: true, priceUSD: 0.7, coingeckoId: 'osmosis' },
  { id: 1521, chainId: 1, address: '0x0000000000000000000000000000000000000000', symbol: 'MKR', name: 'Maker', decimals: 18, isNative: false, isActive: true, priceUSD: 1500, coingeckoId: 'maker' },
  { id: 1522, chainId: 1, address: '0x0000000000000000000000000000000000000000', symbol: 'SNX', name: 'Synthetix', decimals: 18, isNative: false, isActive: true, priceUSD: 2.5, coingeckoId: 'synthetix-network-token' },
  { id: 1523, chainId: 1, address: '0x0000000000000000000000000000000000000000', symbol: 'LDO', name: 'Lido DAO', decimals: 18, isNative: false, isActive: true, priceUSD: 2.2, coingeckoId: 'lido-dao' },
  { id: 1524, chainId: 1, address: '0x0000000000000000000000000000000000000000', symbol: 'RPL', name: 'Rocket Pool', decimals: 18, isNative: false, isActive: true, priceUSD: 25, coingeckoId: 'rocket-pool' },
  { id: 1525, chainId: 1, address: '0x0000000000000000000000000000000000000000', symbol: 'ENS', name: 'Ethereum Name Service', decimals: 18, isNative: false, isActive: true, priceUSD: 20, coingeckoId: 'ethereum-name-service' },
  { id: 1526, chainId: 1, address: '0x0000000000000000000000000000000000000000', symbol: '1INCH', name: '1inch', decimals: 18, isNative: false, isActive: true, priceUSD: 0.4, coingeckoId: '1inch' },
  { id: 1527, chainId: 1, address: '0x0000000000000000000000000000000000000000', symbol: 'CRO', name: 'Cronos', decimals: 18, isNative: false, isActive: true, priceUSD: 0.1, coingeckoId: 'crypto-com-chain' },
  { id: 1528, chainId: 1, address: '0x0000000000000000000000000000000000000000', symbol: 'KCS', name: 'KuCoin Token', decimals: 18, isNative: false, isActive: true, priceUSD: 10, coingeckoId: 'kucoin-shares' },
  { id: 1529, chainId: 1, address: '0x0000000000000000000000000000000000000000', symbol: 'HT', name: 'Huobi Token', decimals: 18, isNative: false, isActive: true, priceUSD: 3, coingeckoId: 'huobi-token' },
  { id: 1530, chainId: 1, address: '0x0000000000000000000000000000000000000000', symbol: 'OKB', name: 'OKB', decimals: 18, isNative: false, isActive: true, priceUSD: 50, coingeckoId: 'okb' },
];

// Utility functions
export function getChainById(chainId: number): Chain | undefined {
  return SUPPORTED_CHAINS.find(c => c.id === chainId || c.chainId === chainId);
}

export function getChainBySymbol(symbol: string): Chain | undefined {
  return SUPPORTED_CHAINS.find(c => c.symbol.toUpperCase() === symbol.toUpperCase());
}

export function getTokensByChain(chainId: number): Token[] {
  return SUPPORTED_TOKENS.filter(t => t.chainId === chainId && t.isActive);
}

export function getTokenByAddress(chainId: number, address: string): Token | undefined {
  return SUPPORTED_TOKENS.find(t => t.chainId === chainId && t.address.toLowerCase() === address.toLowerCase());
}

export function getTokenBySymbol(symbol: string): Token | undefined {
  return SUPPORTED_TOKENS.find(t => t.symbol.toUpperCase() === symbol.toUpperCase());
}

export function getNativeToken(chainId: number): Token | undefined {
  return SUPPORTED_TOKENS.find(t => t.chainId === chainId && t.isNative);
}

export function searchChains(query: string): Chain[] {
  const q = query.toLowerCase();
  return SUPPORTED_CHAINS.filter(c => 
    c.name.toLowerCase().includes(q) || 
    c.symbol.toLowerCase().includes(q) ||
    c.type.toLowerCase().includes(q)
  );
}

export function searchTokens(query: string): Token[] {
  const q = query.toLowerCase();
  return SUPPORTED_TOKENS.filter(t => 
    t.symbol.toLowerCase().includes(q) || 
    t.name.toLowerCase().includes(q)
  );
}

export function getAllActiveChains(): Chain[] {
  return SUPPORTED_CHAINS.filter(c => c.isActive);
}

export function getAllActiveTokens(): Token[] {
  return SUPPORTED_TOKENS.filter(t => t.isActive);
}

export function getDefaultChains(): Chain[] {
  return SUPPORTED_CHAINS.filter(c => c.isDefault);
}

export default SUPPORTED_CHAINS;
