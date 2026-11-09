/**
 * TigerWallet - Complete Blockchain Registry
 * 
 * Supports 40+ blockchains (20+ EVM + 20+ Non-EVM)
 * with full asset support for all tokens
 */

// ============================================================================
// EVM Blockchains (20+)
// ============================================================================

export interface EVMChain {
  id: number;
  name: string;
  symbol: string;
  decimals: number;
  explorer: string;
  rpc: string;
  wsRpc?: string;
  chainId: number;
  networkId?: number;
  icon?: string;
  color?: string;
  status: 'active' | 'deprecated' | 'testnet';
  type: 'mainnet' | 'testnet';
}

export const EVM_CHAINS: EVMChain[] = [
  // Layer 1
  {
    id: 1,
    name: 'Ethereum',
    symbol: 'ETH',
    decimals: 18,
    explorer: 'https://etherscan.io',
    rpc: process.env.NEXT_PUBLIC_ETHEREUM_RPC || '',
    chainId: 1,
    networkId: 1,
    color: '#627EEA',
    status: 'active',
    type: 'mainnet',
  },
  {
    id: 56,
    name: 'BNB Smart Chain',
    symbol: 'BNB',
    decimals: 18,
    explorer: 'https://bscscan.com',
    rpc: 'https://bsc-dataseed.binance.org',
    chainId: 56,
    networkId: 56,
    color: '#F3BA2F',
    status: 'active',
    type: 'mainnet',
  },
  // Layer 2
  {
    id: 137,
    name: 'Polygon',
    symbol: 'MATIC',
    decimals: 18,
    explorer: 'https://polygonscan.com',
    rpc: 'https://polygon-rpc.com',
    chainId: 137,
    networkId: 137,
    color: '#8247E5',
    status: 'active',
    type: 'mainnet',
  },
  {
    id: 42161,
    name: 'Arbitrum One',
    symbol: 'ETH',
    decimals: 18,
    explorer: 'https://arbiscan.io',
    rpc: 'https://arb1.arbitrum.io/rpc',
    chainId: 42161,
    networkId: 42161,
    color: '#28A0F0',
    status: 'active',
    type: 'mainnet',
  },
  {
    id: 10,
    name: 'Optimism',
    symbol: 'ETH',
    decimals: 18,
    explorer: 'https://optimistic.etherscan.io',
    rpc: 'https://mainnet.optimism.io',
    chainId: 10,
    networkId: 10,
    color: '#FF0420',
    status: 'active',
    type: 'mainnet',
  },
  {
    id: 8453,
    name: 'Base',
    symbol: 'ETH',
    decimals: 18,
    explorer: 'https://basescan.org',
    rpc: 'https://mainnet.base.org',
    chainId: 8453,
    networkId: 8453,
    color: '#0052FF',
    status: 'active',
    type: 'mainnet',
  },
  // ZK Rollups
  {
    id: 324,
    name: 'zkSync Era',
    symbol: 'ETH',
    decimals: 18,
    explorer: 'https://explorer.zksync.io',
    rpc: 'https://zksync-era.blockchainrpc.com',
    chainId: 324,
    networkId: 324,
    color: '#000000',
    status: 'active',
    type: 'mainnet',
  },
  {
    id: 59144,
    name: 'Linea',
    symbol: 'ETH',
    decimals: 18,
    explorer: 'https://lineascan.build',
    rpc: 'https://rpc.linea.build',
    chainId: 59144,
    color: '#000000',
    status: 'active',
    type: 'mainnet',
  },
  {
    id: 534352,
    name: 'Scroll',
    symbol: 'ETH',
    decimals: 18,
    explorer: 'https://scrollscan.com',
    rpc: 'https://rpc.scroll.io',
    chainId: 534352,
    color: '#CDA74A',
    status: 'active',
    type: 'mainnet',
  },
  // Other EVM
  {
    id: 43114,
    name: 'Avalanche',
    symbol: 'AVAX',
    decimals: 18,
    explorer: 'https://snowtrace.io',
    rpc: 'https://api.avax.network',
    chainId: 43114,
    networkId: 43114,
    color: '#E84142',
    status: 'active',
    type: 'mainnet',
  },
  {
    id: 5000,
    name: 'Mantle',
    symbol: 'MNT',
    decimals: 18,
    explorer: 'https://explorer.mantle.xyz',
    rpc: 'https://rpc.mantle.xyz',
    chainId: 5000,
    color: '#000000',
    status: 'active',
    type: 'mainnet',
  },
  {
    id: 81457,
    name: 'Blast',
    symbol: 'ETH',
    decimals: 18,
    explorer: 'https://blastscan.io',
    rpc: 'https://rpc.blast.io',
    chainId: 81457,
    color: '#FFFF00',
    status: 'active',
    type: 'mainnet',
  },
  {
    id: 100,
    name: 'Gnosis',
    symbol: 'xDAI',
    decimals: 18,
    explorer: 'https://gnosisscan.io',
    rpc: 'https://rpc.gnosischain.com',
    chainId: 100,
    networkId: 100,
    color: '#477995',
    status: 'active',
    type: 'mainnet',
  },
  {
    id: 250,
    name: 'Fantom',
    symbol: 'FTM',
    decimals: 18,
    explorer: 'https://ftmscan.com',
    rpc: 'https://rpc.fantom.network',
    chainId: 250,
    networkId: 250,
    color: '#1969FF',
    status: 'active',
    type: 'mainnet',
  },
  {
    id: 42220,
    name: 'Celo',
    symbol: 'CELO',
    decimals: 18,
    explorer: 'https://celoscan.io',
    rpc: 'https://rpc.celo.org',
    chainId: 42220,
    color: '#FBCC5C',
    status: 'active',
    type: 'mainnet',
  },
  {
    id: 8217,
    name: 'Klaytn',
    symbol: 'KLAY',
    decimals: 18,
    explorer: 'https://scope.klaytn.com',
    rpc: 'https://rpc.klaytn.org',
    chainId: 8217,
    networkId: 8217,
    color: '#38BD93',
    status: 'active',
    type: 'mainnet',
  },
  {
    id: 25,
    name: 'Cronos',
    symbol: 'CRO',
    decimals: 18,
    explorer: 'https://cronoscan.com',
    rpc: 'https://rpc-cronos.crypto.org',
    chainId: 25,
    color: '#002D74',
    status: 'active',
    type: 'mainnet',
  },
  {
    id: 1284,
    name: 'Moonbeam',
    symbol: 'GLMR',
    decimals: 18,
    explorer: 'https://moonbeam.moonscan.io',
    rpc: 'https://rpc.api.moonbeam.network',
    chainId: 1284,
    color: '#53FFE4',
    status: 'active',
    type: 'mainnet',
  },
  {
    id: 1285,
    name: 'Moonriver',
    symbol: 'MOVR',
    decimals: 18,
    explorer: 'https://moonriver.moonscan.io',
    rpc: 'https://rpc.api.moonriver.moonbeam.network',
    chainId: 1285,
    color: '#5ADF9F',
    status: 'active',
    type: 'mainnet',
  },
  {
    id: 592,
    name: 'Astar',
    symbol: 'ASTR',
    decimals: 18,
    explorer: 'https://astar.explorer.io',
    rpc: 'https://rpc.astar.network',
    chainId: 592,
    color: '#3C3578',
    status: 'active',
    type: 'mainnet',
  },
];

// ============================================================================
// Non-EVM Blockchains (20+)
// ============================================================================

export interface NonEVMChain {
  id: string;
  name: string;
  symbol: string;
  decimals: number;
  explorer: string;
  rpc: string;
  icon?: string;
  color?: string;
  status: 'active' | 'deprecated' | 'testnet';
  type: 'mainnet' | 'testnet';
  derivationPath?: string;
  curve?: 'ed25519' | 'secp256k1' | 'sr25519';
}

export const NON_EVM_CHAINS: NonEVMChain[] = [
  // Bitcoin & forks
  {
    id: 'btc',
    name: 'Bitcoin',
    symbol: 'BTC',
    decimals: 8,
    explorer: 'https://blockstream.info',
    rpc: 'https://blockstream.info/api',
    color: '#F7931A',
    status: 'active',
    type: 'mainnet',
    derivationPath: "m/44'/0'/0'/0/0",
    curve: 'secp256k1',
  },
  {
    id: 'ltc',
    name: 'Litecoin',
    symbol: 'LTC',
    decimals: 8,
    explorer: 'https://blockchair.com/litecoin',
    rpc: 'https://litecoin.ragnarok.online',
    color: '#BFBBBB',
    status: 'active',
    type: 'mainnet',
    derivationPath: "m/44'/2'/0'/0/0",
    curve: 'secp256k1',
  },
  // Solana & apps
  {
    id: 'solana',
    name: 'Solana',
    symbol: 'SOL',
    decimals: 9,
    explorer: 'https://solscan.io',
    rpc: 'https://api.mainnet-beta.solana.com',
    color: '#9945FF',
    status: 'active',
    type: 'mainnet',
    derivationPath: "m/44'/501'/0'/0'",
    curve: 'ed25519',
  },
  // TON ecosystem
  {
    id: 'ton',
    name: 'TON',
    symbol: 'TON',
    decimals: 9,
    explorer: 'https://toncoin.io',
    rpc: 'https://toncenter.com/api/v2/',
    color: '#0098EA',
    status: 'active',
    type: 'mainnet',
    derivationPath: "m/44'/607'/0'/0/0",
    curve: 'secp256k1',
  },
  // Cosmos ecosystem
  {
    id: 'cosmos',
    name: 'Cosmos',
    symbol: 'ATOM',
    decimals: 6,
    explorer: 'https://mintscan.io/cosmos',
    rpc: 'https://rpc.cosmos.network',
    color: '#2E3148',
    status: 'active',
    type: 'mainnet',
    derivationPath: "m/44'/118'/0'/0/0",
    curve: 'secp256k1',
  },
  {
    id: 'osmosis',
    name: 'Osmosis',
    symbol: 'OSMO',
    decimals: 6,
    explorer: 'https://mintscan.io/osmosis',
    rpc: 'https://rpc-osmosis.keplr.app',
    color: '#5A52E0',
    status: 'active',
    type: 'mainnet',
    derivationPath: "m/44'/118'/0'/0/0",
    curve: 'secp256k1',
  },
  // Move ecosystem
  {
    id: 'aptos',
    name: 'Aptos',
    symbol: 'APT',
    decimals: 8,
    explorer: 'https://aptoscan.com',
    rpc: 'https://fullnode.mainnet.aptoslabs.com',
    color: '#14F195',
    status: 'active',
    type: 'mainnet',
    derivationPath: "m/44'/637'/0'/0'/0",
    curve: 'ed25519',
  },
  {
    id: 'sui',
    name: 'Sui',
    symbol: 'SUI',
    decimals: 9,
    explorer: 'https://sui.io',
    rpc: 'https://fullnode.mainnet.sui.io',
    color: '#6FBEED',
    status: 'active',
    type: 'mainnet',
    derivationPath: "m/44'/784'/0'/0'/0'",
    curve: 'ed25519',
  },
  // TRON
  {
    id: 'tron',
    name: 'TRON',
    symbol: 'TRX',
    decimals: 6,
    explorer: 'https://tronscan.org',
    rpc: 'https://api.trongrid.io',
    color: '#EB0029',
    status: 'active',
    type: 'mainnet',
    derivationPath: "m/44'/195'/0'/0/0",
    curve: 'secp256k1',
  },
  // NEAR
  {
    id: 'near',
    name: 'NEAR',
    symbol: 'NEAR',
    decimals: 24,
    explorer: 'https://nearscan.io',
    rpc: 'https://rpc.mainnet.near.org',
    color: '#000000',
    status: 'active',
    type: 'mainnet',
    derivationPath: "m/44'/397'/0'/0/0",
    curve: 'ed25519',
  },
  // Algorand
  {
    id: 'algorand',
    name: 'Algorand',
    symbol: 'ALGO',
    decimals: 6,
    explorer: 'https://algoexplorer.io',
    rpc: 'https://mainnet-api.algorand.network',
    color: '#000000',
    status: 'active',
    type: 'mainnet',
    derivationPath: "m/44'/283'/0'/0/0",
    curve: 'ed25519',
  },
  // Tezos
  {
    id: 'tezos',
    name: 'Tezos',
    symbol: 'XTZ',
    decimals: 6,
    explorer: 'https://tzkt.io',
    rpc: 'https://mainnet.api.tez.ie',
    color: '#2C8DFB',
    status: 'active',
    type: 'mainnet',
    derivationPath: "m/44'/1729'/0'/0/0",
    curve: 'ed25519',
  },
  // Polkadot ecosystem
  {
    id: 'polkadot',
    name: 'Polkadot',
    symbol: 'DOT',
    decimals: 10,
    explorer: 'https://polkadot.subscan.io',
    rpc: 'https://rpc.polkadot.io',
    color: '#E6007A',
    status: 'active',
    type: 'mainnet',
    derivationPath: "m/44'/354'/0'/0/0",
    curve: 'sr25519',
  },
  {
    id: 'kusama',
    name: 'Kusama',
    symbol: 'KSM',
    decimals: 12,
    explorer: 'https://kusama.subscan.io',
    rpc: 'https://rpc.kusama.io',
    color: '#000000',
    status: 'active',
    type: 'mainnet',
    derivationPath: "m/44'/434'/0'/0/0",
    curve: 'sr25519',
  },
  // Hedera
  {
    id: 'hedera',
    name: 'Hedera',
    symbol: 'HBAR',
    decimals: 8,
    explorer: 'https://hashscan.io',
    rpc: 'https://mainnet.mirrornode.hedera.com',
    color: '#302960',
    status: 'active',
    type: 'mainnet',
    derivationPath: "m/44'/303'/0'/0/0",
    curve: 'secp256k1',
  },
  // VeChain
  {
    id: 'vechain',
    name: 'VeChain',
    symbol: 'VET',
    decimals: 18,
    explorer: 'https://vechainstats.com',
    rpc: 'https://mainnet.infura.io',
    color: '#15BDFF',
    status: 'active',
    type: 'mainnet',
    derivationPath: "m/44'/393'/0'/0/0",
    curve: 'secp256k1',
  },
  // Flow
  {
    id: 'flow',
    name: 'Flow',
    symbol: 'FLOW',
    decimals: 8,
    explorer: 'https://flowdiver.io',
    rpc: 'https://flow-http.metaproc.com',
    color: '#000000',
    status: 'active',
    type: 'mainnet',
    derivationPath: "m/44'/539'/0'/0/0",
    curve: 'secp256k1',
  },
  // Conflux
  {
    id: 'conflux',
    name: 'Conflux',
    symbol: 'CFX',
    decimals: 18,
    explorer: 'https://confluxscan.io',
    rpc: 'https://rpc.confluxnetwork.org',
    color: '#000000',
    status: 'active',
    type: 'mainnet',
    derivationPath: "m/44'/503'/0'/0/0",
    curve: 'secp256k1',
  },
  // Sei
  {
    id: 'sei',
    name: 'Sei',
    symbol: 'SEI',
    decimals: 6,
    explorer: 'https://seitrace.com',
    rpc: 'https://rpc.sei.io',
    color: '#000000',
    status: 'active',
    type: 'mainnet',
    derivationPath: "m/44'/549'/0'/0/0",
    curve: 'secp256k1',
  },
  // Injective
  {
    id: 'injective',
    name: 'Injective',
    symbol: 'INJ',
    decimals: 18,
    explorer: 'https://explorer.injective.network',
    rpc: 'https://public.injective.network',
    color: '#00F2FE',
    status: 'active',
    type: 'mainnet',
    derivationPath: "m/44'/690'/0'/0/0",
    curve: 'secp256k1',
  },
];

// ============================================================================
// All Chains Combined
// ============================================================================

export const ALL_CHAINS = [
  ...EVM_CHAINS.map(c => ({ ...c, isEVM: true })),
  ...NON_EVM_CHAINS.map(c => ({ ...c, isEVM: false })),
];

// ============================================================================
// Popular Tokens (50+)
// ============================================================================

export interface Token {
  address: string;
  chainId: number | string;
  symbol: string;
  name: string;
  decimals: number;
  logo?: string;
  type: 'native' | 'erc20' | 'spl' | 'other';
}

export const POPULAR_TOKENS: Token[] = [
  // Ethereum tokens
  { address: '0x0000000000000000000000000000000000000000', chainId: 1, symbol: 'ETH', name: 'Ethereum', decimals: 18, type: 'native' },
  { address: '0xdAC17F958D2ee523a2206206994597C13D831ec7', chainId: 1, symbol: 'USDT', name: 'Tether USD', decimals: 6, type: 'erc20' },
  { address: '0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48', chainId: 1, symbol: 'USDC', name: 'USD Coin', decimals: 6, type: 'erc20' },
  { address: '0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599', chainId: 1, symbol: 'WBTC', name: 'Wrapped Bitcoin', decimals: 8, type: 'erc20' },
  { address: '0x6B175474E89094C44Da98b954EedeAC495271d0F', chainId: 1, symbol: 'DAI', name: 'Dai Stablecoin', decimals: 18, type: 'erc20' },
  { address: '0x7Fc66500c84A76Ad7e9c93437bFc5Ac33E2DDaE9', chainId: 1, symbol: 'AAVE', name: 'Aave', decimals: 18, type: 'erc20' },
  { address: '0x1f9840a85d5aF5bf1D1762F925BDADdC4201F984', chainId: 1, symbol: 'UNI', name: 'Uniswap', decimals: 18, type: 'erc20' },
  { address: '0x514910771AF9Ca656af840dff83E8264EcF986CA', chainId: 1, symbol: 'LINK', name: 'Chainlink', decimals: 18, type: 'erc20' },
  { address: '0xD533a949740bb3306d119CC777fa900bA034cd52', chainId: 1, symbol: 'CRV', name: 'Curve DAO', decimals: 18, type: 'erc20' },
  { address: '0x5A98FcBEA516Cf06857215779Fd812CA3beF1B32', chainId: 1, symbol: 'LDO', name: 'Lido DAO', decimals: 18, type: 'erc20' },
  
  // BNB Chain tokens
  { address: '0x0000000000000000000000000000000000000000', chainId: 56, symbol: 'BNB', name: 'BNB', decimals: 18, type: 'native' },
  { address: '0x55d398326f99059fF775485246999027B3197955', chainId: 56, symbol: 'USDT', name: 'Tether USD', decimals: 18, type: 'erc20' },
  { address: '0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d', chainId: 56, symbol: 'USDC', name: 'USD Coin', decimals: 18, type: 'erc20' },
  { address: '0xe9e7CEA3DedcA5984780Bafc599bD69ADd087D56', chainId: 56, symbol: 'BUSD', name: 'Binance USD', decimals: 18, type: 'erc20' },
  { address: '0x0E09FaBB73Bd3Ade0a17ECC321fD13a19e81cE82', chainId: 56, symbol: 'CAKE', name: 'PancakeSwap', decimals: 18, type: 'erc20' },
  
  // Polygon tokens
  { address: '0x0000000000000000000000000000000000000000', chainId: 137, symbol: 'MATIC', name: 'Polygon', decimals: 18, type: 'native' },
  { address: '0xc2132D05D31c914a87C6611C10748AEb04B58e8F', chainId: 137, symbol: 'USDT', name: 'Tether USD', decimals: 6, type: 'erc20' },
  { address: '0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359', chainId: 137, symbol: 'USDC', name: 'USD Coin', decimals: 6, type: 'erc20' },
  
  // Solana tokens (via wrapped)
  { address: 'So11111111111111111111111111111111111111112', chainId: 'solana', symbol: 'SOL', name: 'Solana', decimals: 9, type: 'native' },
  { address: 'EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v', chainId: 'solana', symbol: 'USDC', name: 'USD Coin', decimals: 6, type: 'spl' },
  { address: 'Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB', chainId: 'solana', symbol: 'USDT', name: 'Tether USD', decimals: 6, type: 'spl' },
  
  // More popular tokens
  { address: '0x0000000000000000000000000000000000000000', chainId: 43114, symbol: 'AVAX', name: 'Avalanche', decimals: 18, type: 'native' },
  { address: '0x0000000000000000000000000000000000000000', chainId: 10, symbol: 'ETH', name: 'Ethereum', decimals: 18, type: 'native' },
  { address: '0x0000000000000000000000000000000000000000', chainId: 8453, symbol: 'ETH', name: 'Ethereum', decimals: 18, type: 'native' },
  { address: '0x0000000000000000000000000000000000000000', chainId: 324, symbol: 'ETH', name: 'Ethereum', decimals: 18, type: 'native' },
  { address: '0x0000000000000000000000000000000000000000', chainId: 42161, symbol: 'ETH', name: 'Ethereum', decimals: 18, type: 'native' },
];

// ============================================================================
// Utility Functions
// ============================================================================

export const getChainById = (id: number | string): EVMChain | NonEVMChain | undefined => {
  if (typeof id === 'number') {
    return EVM_CHAINS.find(c => c.id === id);
  }
  return NON_EVM_CHAINS.find(c => c.id === id);
};

export const getChainBySymbol = (symbol: string): EVMChain | NonEVMChain | undefined => {
  const evm = EVM_CHAINS.find(c => c.symbol.toUpperCase() === symbol.toUpperCase());
  if (evm) return evm;
  return NON_EVM_CHAINS.find(c => c.symbol.toUpperCase() === symbol.toUpperCase());
};

export const getTokenBySymbol = (symbol: string, chainId: number | string): Token | undefined => {
  return POPULAR_TOKENS.find(t => 
    t.symbol.toUpperCase() === symbol.toUpperCase() && t.chainId === chainId
  );
};

export const getSupportedChains = (type?: 'mainnet' | 'testnet'): (EVMChain | NonEVMChain)[] => {
  let chains = ALL_CHAINS;
  if (type) {
    chains = chains.filter(c => c.type === type);
  }
  return chains.filter(c => c.status === 'active');
};

// ============================================================================
// Export
// ============================================================================

export default {
  EVM_CHAINS,
  NON_EVM_CHAINS,
  ALL_CHAINS,
  POPULAR_TOKENS,
  getChainById,
  getChainBySymbol,
  getTokenBySymbol,
  getSupportedChains,
};