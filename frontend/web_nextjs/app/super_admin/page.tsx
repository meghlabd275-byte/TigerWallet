'use client';

import React, { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';

// Types
interface Blockchain {
  id: number;
  name: string;
  symbol: string;
  chainId: number;
  rpcUrl: string;
  explorerUrl: string;
  decimals: number;
  isActive: boolean;
  gasLimit: number;
  confirmations: number;
}

interface Token {
  id: number;
  chainId: number;
  address: string;
  symbol: string;
  name: string;
  decimals: number;
  isActive: boolean;
  isPopular: boolean;
  isStablecoin: boolean;
  priceUsd: number;
}

interface Stats {
  totalUsers: number;
  activeUsers: number;
  totalTransactions: number;
  totalVolume: number;
  totalWallets: number;
}

// Chain types for dropdown - 100+ Multi-chain Support
const CHAIN_TYPES = [
  // Top 50 EVM Chains
  { value: 'ethereum', label: 'Ethereum', chainId: 1 },
  { value: 'polygon', label: 'Polygon', chainId: 137 },
  { value: 'arbitrum', label: 'Arbitrum One', chainId: 42161 },
  { value: 'optimism', label: 'Optimism', chainId: 10 },
  { value: 'base', label: 'Base', chainId: 8453 },
  { value: 'avalanche', label: 'Avalanche C-Chain', chainId: 43114 },
  { value: 'bsc', label: 'BNB Smart Chain', chainId: 56 },
  { value: 'fantom', label: 'Fantom Opera', chainId: 250 },
  { value: 'cronos', label: 'Cronos', chainId: 25 },
  { value: 'celo', label: 'Celo', chainId: 42220 },
  { value: 'harmony', label: 'Harmony One', chainId: 1666600000 },
  { value: 'moonbeam', label: 'Moonbeam', chainId: 1284 },
  { value: 'moonriver', label: 'Moonriver', chainId: 1285 },
  { value: 'astar', label: 'Astar', chainId: 592 },
  { value: 'shiden', label: 'Shiden', chainId: 336 },
  { value: 'oasis', label: 'Oasis Emerald', chainId: 42262 },
  { value: 'telos', label: 'Telos EVM', chainId: 40 },
  { value: 'kava', label: 'Kava EVM', chainId: 2222 },
  { value: 'evmos', label: 'Evmos', chainId: 9001 },
  { value: 'aurora', label: 'Aurora', chainId: 1313161554 },
  { value: ' canto', label: 'Canto', chainId: 7700 },
  { value: 'zkEVM', label: 'Polygon zkEVM', chainId: 1101 },
  { value: 'linea', label: 'Linea', chainId: 59144 },
  { value: 'scroll', label: 'Scroll', chainId: 534352 },
  { value: 'zkSync', label: 'zkSync Era', chainId: 324 },
  { value: 'opBNB', label: 'opBNB', chainId: 204 },
  { value: 'mantle', label: 'Mantle', chainId: 5000 },
  { value: 'baseSepolia', label: 'Base Sepolia', chainId: 84532 },
  { value: 'arbitrumSepolia', label: 'Arbitrum Sepolia', chainId: 421614 },
  { value: 'optimismSepolia', label: 'Optimism Sepolia', chainId: 11155420 },
  { value: 'ethereumClassic', label: 'Ethereum Classic', chainId: 61 },
  { value: 'poa', label: 'POA Network', chainId: 99 },
  { value: 'xDAI', label: 'Gnosis Chain', chainId: 100 },
  { value: 'theta', label: 'Theta Network', chainId: 361 },
  { value: 'iotex', label: 'IoTeX', chainId: 4689 },
  { value: 'ronin', label: 'Ronin', chainId: 2020 },
  { value: 'klaytn', label: 'Klaytn', chainId: 8217 },
  { value: 'findora', label: 'Findora', chainId: 2152 },
  { value: 'hydra', label: 'Hydra', chainId: 127 },
  { value: 'freq', label: 'Frequency', chainId: 2099156 },
  { value: 'Mode', label: 'Mode', chainId: 34443 },
  { value: 'Manta', label: 'Manta Pacific', chainId: 169 },
  { value: 'Fraxtal', label: 'Fraxtal', chainId: 2522 },
  { value: 'Blast', label: 'Blast', chainId: 81457 },
  { value: 'ink', label: 'Ink', chainId: 570 },
  { value: 'berachain', label: 'Berachain', chainId: 238886 },
  { value: 'sei', label: 'Sei', chainId: 1329 },
  { value: 'soneium', label: 'Soneium', chainId: 1946 },
  { value: 'titan', label: 'Titan', chainId: 5050 },
  { value: 'unichain', label: 'Unichain', chainId: 130 },
  // Non-EVM Chains
  { value: 'solana', label: 'Solana', chainId: 101 },
  { value: 'tron', label: 'Tron', chainId: 728126428 },
  { value: 'bitcoin', label: 'Bitcoin', chainId: 0 },
  { value: 'litecoin', label: 'Litecoin', chainId: 2 },
  { value: 'dogecoin', label: 'Dogecoin', chainId: 3 },
  { value: 'ripple', label: 'XRP Ledger', chainId: 144 },
  { value: 'cardano', label: 'Cardano', chainId: 3009 },
  { value: 'near', label: 'NEAR Protocol', chainId: 1313161554 },
  { value: 'aptos', label: 'Aptos', chainId: 637 },
  { value: 'sui', label: 'Sui', chainId: 784 },
  { value: 'ton', label: 'TON', chainId: -239 },
  { value: 'cosmos', label: 'Cosmos Hub', chainId: 118 },
  { value: 'osmosis', label: 'Osmosis', chainId: 0 },
  { value: 'terra', label: 'Terra Classic', chainId: 0 },
  { value: 'juno', label: 'Juno', chainId: 0 },
  { value: 'injective', label: 'Injective', chainId: 690 },
  { value: 'sei', label: 'Sei', chainId: 0 },
  { value: 'celestia', label: 'Celestia', chainId: 0 },
  { value: 'akt', label: 'Akash Network', chainId: 0 },
  { value: 'dym', label: 'Dymension', chainId: 0 },
  { value: 'pi', label: 'Pi Network', chainId: 314159 },
  { value: 'pulsechain', label: 'PulseChain', chainId: 369 },
  { value: 'ergo', label: 'Ergo', chainId: 0 },
  { value: 'kadena', label: 'Kadena', chainId: 0 },
  { value: 'tezos', label: 'Tezos', chainId: 0 },
  { value: 'algorand', label: 'Algorand', chainId: 0 },
  { value: 'hedera', label: 'Hedera', chainId: 295 },
  { value: 'stacks', label: 'Stacks', chainId: 0 },
  { value: 'flow', label: 'Flow', chainId: 0 },
  { value: 'apt', label: 'Aptos', chainId: 637 },
  { value: 'toncoin', label: 'Toncoin', chainId: -239 },
  { value: 'polkadot', label: 'Polkadot', chainId: 0 },
  { value: 'kusama', label: 'Kusama', chainId: 0 },
  { value: 'parallel', label: 'Parallel', chainId: 0 },
  { value: 'acala', label: 'Acala', chainId: 0 },
  { value: 'astar', label: 'Astar', chainId: 0 },
  { value: 'interlay', label: 'Interlay', chainId: 0 },
  { value: 'composable', label: 'Composable', chainId: 0 },
  { value: 'oasis', label: 'Oasis Network', chainId: 0 },
  { value: 'secret', label: 'Secret Network', chainId: 0 },
  { value: 'stargaze', label: 'Stargaze', chainId: 0 },
  { value: 'umee', label: 'Umee', chainId: 0 },
  { value: 'gravity', label: 'Gravity Bridge', chainId: 0 },
  { value: 'kava', label: 'Kava', chainId: 0 },
  { value: 'terraclassic', label: 'Terra Classic', chainId: 0 },
  { value: ' Chihuahua', label: 'Chihuahua', chainId: 0 },
  { value: 'juno', label: 'Juno', chainId: 0 },
  { value: 'stargaze', label: 'Stargaze', chainId: 0 },
  { value: 'quicksilver', label: 'Quicksilver', chainId: 0 },
  { value: 'sommelier', label: 'Sommelier', chainId: 0 },
  { value: 'teritoric', label: 'Teritori', chainId: 0 },
  { value: 'persistence', label: 'Persistence', chainId: 0 },
  { value: 'sentinel', label: 'Sentinel', chainId: 0 },
  { value: 'aioz', label: 'AIOZ Network', chainId: 0 },
  { value: 'desmos', label: 'Desmos', chainId: 0 },
  { value: 'kichain', label: 'Ki Chain', chainId: 0 },
  { value: 'shentu', label: 'Shentu', chainId: 0 },
  { value: 'eMoney', label: 'e-Money', chainId: 0 },
  { value: 'ixo', label: 'IXO', chainId: 0 },
  { value: 'bitsong', label: 'BitSong', chainId: 0 },
  { value: 'lum', label: 'Lum Network', chainId: 0 },
  { value: 'assetMantle', label: 'Asset Mantle', chainId: 0 },
  { value: 'cryptoOrg', label: 'Crypto.org', chainId: 0 },
  { value: 'chain4Travel', label: 'Chain4Travel', chainId: 0 },
  { value: 'fetch', label: 'Fetch.ai', chainId: 0 },
  { value: 'band', label: 'Band Protocol', chainId: 0 },
  { value: 'regen', label: 'Regen Network', chainId: 0 },
  { value: 'bittorrent', label: 'BitTorrent Chain', chainId: 0 },
  { value: 'conflux', label: 'Conflux', chainId: 0 },
  { value: 'fusion', label: 'Fusion', chainId: 0 },
  { value: 'wanchain', label: 'Wanchain', chainId: 888 },
  { value: 'callisto', label: 'Callisto', chainId: 820 },
  { value: 'thundercore', label: 'ThunderCore', chainId: 108 },
  { value: 'metis', label: 'Metis', chainId: 1088 },
  { value: 'rsk', label: 'RSK', chainId: 30 },
  { value: 'syscoin', label: 'Syscoin', chainId: 57 },
];

// Popular tokens list - 200+ Cryptocurrencies Support
const POPULAR_TOKENS = [
  // Top 20 by Market Cap
  { symbol: 'ETH', name: 'Ethereum', decimals: 18 },
  { symbol: 'BTC', name: 'Bitcoin', decimals: 8 },
  { symbol: 'USDT', name: 'Tether USD', decimals: 6 },
  { symbol: 'USDC', name: 'USD Coin', decimals: 6 },
  { symbol: 'BNB', name: 'BNB', decimals: 18 },
  { symbol: 'XRP', name: 'Ripple', decimals: 6 },
  { symbol: 'DOGE', name: 'Dogecoin', decimals: 8 },
  { symbol: 'PI', name: 'Pi Network', decimals: 18 },
  { symbol: 'TON', name: 'Toncoin', decimals: 9 },
  { symbol: 'TRX', name: 'Tron', decimals: 6 },
  { symbol: 'SOL', name: 'Solana', decimals: 9 },
  { symbol: 'ADA', name: 'Cardano', decimals: 6 },
  { symbol: 'SHIB', name: 'Shiba Inu', decimals: 18 },
  { symbol: 'DOT', name: 'Polkadot', decimals: 18 },
  { symbol: 'MATIC', name: 'Polygon', decimals: 18 },
  { symbol: 'LTC', name: 'Litecoin', decimals: 8 },
  { symbol: 'AVAX', name: 'Avalanche', decimals: 18 },
  { symbol: 'LINK', name: 'Chainlink', decimals: 18 },
  { symbol: 'UNI', name: 'Uniswap', decimals: 18 },
  { symbol: 'ATOM', name: 'Cosmos', decimals: 6 },
  // EVM Chain Native Tokens
  { symbol: 'ARB', name: 'Arbitrum', decimals: 18 },
  { symbol: 'OP', name: 'Optimism', decimals: 18 },
  { symbol: 'BASE', name: 'Base', decimals: 18 },
  { symbol: 'FTM', name: 'Fantom', decimals: 18 },
  { symbol: 'CRO', name: 'Cronos', decimals: 18 },
  { symbol: 'CELO', name: 'Celo', decimals: 18 },
  { symbol: 'ONE', name: 'Harmony', decimals: 18 },
  { symbol: 'GLMR', name: 'Moonbeam', decimals: 18 },
  { symbol: 'MOVR', name: 'Moonriver', decimals: 18 },
  { symbol: 'ASTR', name: 'Astar', decimals: 18 },
  { symbol: 'SDN', name: 'Shiden', decimals: 18 },
  { symbol: 'ROSE', name: 'Oasis Network', decimals: 18 },
  { symbol: 'TLOS', name: 'Telos', decimals: 18 },
  { symbol: 'KAVA', name: 'Kava', decimals: 18 },
  { symbol: 'EVMOS', name: 'Evmos', decimals: 18 },
  { symbol: 'AURORA', name: 'Aurora', decimals: 18 },
  { symbol: 'CANTO', name: 'Canto', decimals: 18 },
  { symbol: 'LINEA', name: 'Linea', decimals: 18 },
  { symbol: 'SCROLL', name: 'Scroll', decimals: 18 },
  { symbol: 'ZKSYNC', name: 'zkSync', decimals: 18 },
  { symbol: 'MNT', name: 'Mantle', decimals: 18 },
  { symbol: 'METIS', name: 'Metis', decimals: 18 },
  { symbol: 'KLAY', name: 'Klaytn', decimals: 18 },
  { symbol: 'RBN', name: 'Ribbon Finance', decimals: 18 },
  { symbol: 'FRA', name: 'Fraxtal', decimals: 18 },
  { symbol: 'MNTA', name: 'Manta', decimals: 18 },
  { symbol: 'BLAST', name: 'Blast', decimals: 18 },
  { symbol: 'SEI', name: 'Sei', decimals: 18 },
  { symbol: 'INJ', name: 'Injective', decimals: 18 },
  { symbol: 'TIA', name: 'Celestia', decimals: 6 },
  { symbol: 'SUI', name: 'Sui', decimals: 9 },
  { symbol: 'APT', name: 'Aptos', decimals: 8 },
  // DeFi Tokens
  { symbol: 'AAVE', name: 'Aave', decimals: 18 },
  { symbol: 'MKR', name: 'Maker', decimals: 18 },
  { symbol: 'SNX', name: 'Synthetix', decimals: 18 },
  { symbol: 'CRV', name: 'Curve DAO', decimals: 18 },
  { symbol: 'COMP', name: 'Compound', decimals: 18 },
  { symbol: 'SUSHI', name: 'SushiSwap', decimals: 18 },
  { symbol: 'CAKE', name: 'PancakeSwap', decimals: 18 },
  { symbol: 'LDO', name: 'Lido DAO', decimals: 18 },
  { symbol: 'RETH', name: 'Rocket Pool', decimals: 18 },
  { symbol: 'STETH', name: 'Lido Staked Ether', decimals: 18 },
  { symbol: 'CBETH', name: 'Coinbase Wrapped Staked ETH', decimals: 18 },
  { symbol: 'WSTETH', name: 'Lido Wrapped Staked ETH', decimals: 18 },
  { symbol: 'RPL', name: 'Rocket Pool', decimals: 18 },
  { symbol: 'OSMO', name: 'Osmosis', decimals: 6 },
  { symbol: 'JUNO', name: 'Juno', decimals: 6 },
  { symbol: 'GRAV', name: 'Gravity Bridge', decimals: 6 },
  { symbol: 'UMEE', name: 'Umee', decimals: 6 },
  { symbol: 'AXL', name: 'Axelar', decimals: 6 },
  { symbol: 'QRD', name: 'Quicksilver', decimals: 6 },
  { symbol: 'SOMM', name: 'Sommelier', decimals: 6 },
  { symbol: 'TIA', name: 'Celestia', decimals: 6 },
  { symbol: 'DYM', name: 'Dymension', decimals: 6 },
  // Stablecoins
  { symbol: 'DAI', name: 'Dai Stablecoin', decimals: 18 },
  { symbol: 'TUSD', name: 'TrueUSD', decimals: 18 },
  { symbol: 'BUSD', name: 'Binance USD', decimals: 18 },
  { symbol: 'USDP', name: 'Pax Dollar', decimals: 18 },
  { symbol: 'FRAX', name: 'Frax', decimals: 18 },
  { symbol: 'USDD', name: 'USDD', decimals: 18 },
  { symbol: 'PAXG', name: 'Paxos Gold', decimals: 18 },
  { symbol: 'XAUT', name: 'Tether Gold', decimals: 6 },
  { symbol: 'EURT', name: 'Euro Tether', decimals: 6 },
  { symbol: 'CNHT', name: 'CNH Tether', decimals: 6 },
  { symbol: 'MIM', name: 'Magic Internet Money', decimals: 18 },
  { symbol: 'USTC', name: 'Terra Classic USD', decimals: 6 },
  // Wrapped Assets
  { symbol: 'WETH', name: 'Wrapped Ethereum', decimals: 18 },
  { symbol: 'WBTC', name: 'Wrapped Bitcoin', decimals: 8 },
  { symbol: 'WBNB', name: 'Wrapped BNB', decimals: 18 },
  { symbol: 'WSOL', name: 'Wrapped Solana', decimals: 9 },
  { symbol: 'WAVAX', name: 'Wrapped Avalanche', decimals: 18 },
  { symbol: 'WMATIC', name: 'Wrapped Polygon', decimals: 18 },
  { symbol: 'WFTM', name: 'Wrapped Fantom', decimals: 18 },
  { symbol: 'WCELO', name: 'Wrapped Celo', decimals: 18 },
  { symbol: 'WONE', name: 'Wrapped Harmony', decimals: 18 },
  // NFT/Gaming Tokens
  { symbol: 'ENJ', name: 'Enjin Coin', decimals: 18 },
  { symbol: 'MANA', name: 'Decentraland', decimals: 18 },
  { symbol: 'SAND', name: 'The Sandbox', decimals: 18 },
  { symbol: 'AXS', name: 'Axie Infinity', decimals: 18 },
  { symbol: 'GALA', name: 'Gala', decimals: 18 },
  { symbol: 'APE', name: 'ApeCoin', decimals: 18 },
  { symbol: 'IMX', name: 'Immutable X', decimals: 18 },
  { symbol: 'BLUR', name: 'Blur', decimals: 18 },
  { symbol: 'FLOW', name: 'Flow', decimals: 8 },
  { symbol: 'ALCH', name: 'Alchemint', decimals: 18 },
  // Utility Tokens
  { symbol: 'BAT', name: 'Basic Attention Token', decimals: 18 },
  { symbol: 'ZRX', name: '0x', decimals: 18 },
  { symbol: 'REP', name: 'Augur', decimals: 18 },
  { symbol: 'ENJ', name: 'Enjin Coin', decimals: 18 },
  { symbol: 'ELF', name: 'Aelf', decimals: 18 },
  { symbol: 'WAXP', name: 'WAX', decimals: 8 },
  { symbol: 'BNT', name: 'Bancor', decimals: 18 },
  { symbol: 'ICX', name: 'Icon', decimals: 18 },
  { symbol: 'ZIL', name: 'Zilliqa', decimals: 12 },
  { symbol: 'ONT', name: 'Ontology', decimals: 18 },
  { symbol: 'NEO', name: 'Neo', decimals: 18 },
  { symbol: 'KSM', name: 'Kusama', decimals: 12 },
  { symbol: 'PHB', name: 'Phoenix', decimals: 18 },
  { symbol: 'NEM', name: 'XEM', decimals: 6 },
  { symbol: 'XLM', name: 'Stellar', decimals: 7 },
  { symbol: 'XMR', name: 'Monero', decimals: 12 },
  { symbol: 'ZEC', name: 'Zcash', decimals: 8 },
  { symbol: 'DASH', name: 'Dash', decimals: 8 },
  { symbol: 'XEM', name: 'NEM', decimals: 6 },
  { symbol: 'EOS', name: 'EOS', decimals: 18 },
  { symbol: 'THETA', name: 'Theta Network', decimals: 18 },
  { symbol: 'XTZ', name: 'Tezos', decimals: 6 },
  { symbol: 'ALGO', name: 'Algorand', decimals: 6 },
  { symbol: 'VET', name: 'VeChain', decimals: 18 },
  { symbol: 'HBAR', name: 'Hedera', decimals: 18 },
  { symbol: 'FIL', name: 'Filecoin', decimals: 18 },
  { symbol: 'ICP', name: 'Internet Computer', decimals: 8 },
  { symbol: 'CHZ', name: 'Chiliz', decimals: 18 },
  { symbol: 'HFT', name: 'Hashflow', decimals: 18 },
  { symbol: 'GMX', name: 'GMX', decimals: 18 },
  { symbol: 'RDNT', name: 'Radiant', decimals: 18 },
  { symbol: 'CGLD', name: 'Celo', decimals: 18 },
  { symbol: 'LSK', name: 'Lisk', decimals: 8 },
  { symbol: 'ANKR', name: 'Ankr', decimals: 18 },
  { symbol: 'RSR', name: 'Reserve Rights', decimals: 18 },
  { symbol: 'OGN', name: 'Origin Protocol', decimals: 18 },
  { symbol: 'BAND', name: 'Band Protocol', decimals: 18 },
  { symbol: 'LRC', name: 'Loopring', decimals: 18 },
  { symbol: '1INCH', name: '1inch', decimals: 18 },
  { symbol: 'MAGIC', name: 'Magic', decimals: 18 },
  { symbol: 'GNS', name: 'Gains Network', decimals: 18 },
  { symbol: 'RDNT', name: 'Radiant', decimals: 18 },
  { symbol: 'DYDX', name: 'dYdX', decimals: 18 },
  { symbol: 'PENDLE', name: 'Pendle', decimals: 18 },
  { symbol: 'STG', name: 'Stargate', decimals: 18 },
  { symbol: 'GMX', name: 'GMX', decimals: 18 },
  { symbol: 'LQTY', name: 'Liquity', decimals: 18 },
  { symbol: 'LUSD', name: 'Liquity USD', decimals: 18 },
  { symbol: 'SPELL', name: 'Spell Token', decimals: 18 },
  { symbol: 'MIM', name: 'Magic Internet Money', decimals: 18 },
  { symbol: 'CRVUSD', name: 'Curve USD', decimals: 18 },
  { symbol: 'CVX', name: 'Convex Finance', decimals: 18 },
  { symbol: 'FXS', name: 'Frax Share', decimals: 18 },
  { symbol: 'LQTY', name: 'Liquity', decimals: 18 },
  { symbol: 'ANGLE', name: 'Angle', decimals: 18 },
  { symbol: 'HAY', name: 'Hay', decimals: 18 },
  { symbol: 'USDL', name: 'USD+', decimals: 6 },
  { symbol: 'DAI+', name: 'Dai+', decimals: 18 },
  // Privacy Coins
  { symbol: 'XMR', name: 'Monero', decimals: 12 },
  { symbol: 'ZEC', name: 'Zcash', decimals: 8 },
  { symbol: 'DASH', name: 'Dash', decimals: 8 },
  { symbol: 'ZEN', name: 'Horizen', decimals: 8 },
  { symbol: 'Firo', name: 'Firo', decimals: 8 },
  // Payment Tokens
  { symbol: 'XRP', name: 'Ripple', decimals: 6 },
  { symbol: 'XLM', name: 'Stellar', decimals: 7 },
  { symbol: 'ALGO', name: 'Algorand', decimals: 6 },
  { symbol: 'HBAR', name: 'Hedera', decimals: 18 },
  // Oracle Tokens
  { symbol: 'LINK', name: 'Chainlink', decimals: 18 },
  { symbol: 'BAND', name: 'Band Protocol', decimals: 18 },
  { symbol: 'TRB', name: 'Tellor', decimals: 18 },
  { symbol: 'API3', name: 'API3', decimals: 18 },
  { symbol: 'DOS', name: 'DOS Network', decimals: 18 },
  { symbol: 'ROOK', name: 'KeeperDAO', decimals: 18 },
  // Exchange Tokens
  { symbol: 'BNB', name: 'Binance Coin', decimals: 18 },
  { symbol: 'OKB', name: 'OKB', decimals: 18 },
  { symbol: 'HT', name: 'Huobi Token', decimals: 18 },
  { symbol: 'KCS', name: 'KuCoin Token', decimals: 18 },
  { symbol: 'GT', name: 'GateToken', decimals: 18 },
  { symbol: 'MX', name: 'MX Token', decimals: 18 },
  { symbol: 'LEO', name: 'UNUS SED LEO', decimals: 18 },
  { symbol: 'BGB', name: 'Bitget Token', decimals: 18 },
  { symbol: 'BN', name: 'BitNewChain', decimals: 8 },
  // Other Popular Tokens
  { symbol: 'NEAR', name: 'NEAR Protocol', decimals: 24 },
  { symbol: 'FTM', name: 'Fantom', decimals: 18 },
  { symbol: 'SAND', name: 'The Sandbox', decimals: 18 },
  { symbol: 'MANA', name: 'Decentraland', decimals: 18 },
  { symbol: 'AXS', name: 'Axie Infinity', decimals: 18 },
  { symbol: 'GALA', name: 'Gala', decimals: 18 },
  { symbol: 'APE', name: 'ApeCoin', decimals: 18 },
  { symbol: 'IMX', name: 'Immutable X', decimals: 18 },
  { symbol: 'RNDR', name: 'Render', decimals: 18 },
  { symbol: 'INJ', name: 'Injective', decimals: 18 },
  { symbol: 'TIA', name: 'Celestia', decimals: 6 },
  { symbol: 'SUI', name: 'Sui', decimals: 9 },
  { symbol: 'SEI', name: 'Sei', decimals: 18 },
  { symbol: 'BLUR', name: 'Blur', decimals: 18 },
  { symbol: 'PEPE', name: 'Pepe', decimals: 18 },
  { symbol: 'SHIB', name: 'Shiba Inu', decimals: 18 },
  { symbol: 'FLOKI', name: 'FLOKI', decimals: 9 },
  { symbol: 'BONK', name: 'Bonk', decimals: 5 },
  { symbol: 'WIF', name: 'dogwifhat', decimals: 6 },
  { symbol: 'ORDI', name: 'ORDI', decimals: 18 },
  { symbol: 'SATS', name: 'Sats', decimals: 18 },
  { symbol: 'STX', name: 'Stacks', decimals: 6 },
  { symbol: 'RUNE', name: 'THORChain', decimals: 8 },
  { symbol: 'KAVA', name: 'Kava', decimals: 18 },
  { symbol: 'ZIL', name: 'Zilliqa', decimals: 12 },
  { symbol: 'ENS', name: 'Ethereum Name Service', decimals: 18 },
  { symbol: 'TWT', name: 'Trust Wallet', decimals: 18 },
  { symbol: 'GTC', name: 'Gitcoin', decimals: 18 },
  { symbol: 'KLAY', name: 'Klaytn', decimals: 18 },
  { symbol: 'MINA', name: 'Mina', decimals: 9 },
  { symbol: 'TOP', name: 'TOP', decimals: 18 },
  { symbol: 'CSPR', name: 'Casper', decimals: 9 },
  { symbol: 'NEO', name: 'Neo', decimals: 18 },
  { symbol: 'EGLD', name: 'MultiversX', decimals: 18 },
  { symbol: 'AR', name: 'Arweave', decimals: 12 },
  { symbol: 'LDO', name: 'Lido DAO', decimals: 18 },
  { symbol: 'RPL', name: 'Rocket Pool', decimals: 18 },
  { symbol: 'SSV', name: 'ssv.network', decimals: 18 },
  { symbol: 'OSMO', name: 'Osmosis', decimals: 6 },
  { symbol: 'JUNO', name: 'Juno', decimals: 6 },
  { symbol: 'STARS', name: 'Stargaze', decimals: 6 },
  { symbol: 'CRE', name: 'Crescent', decimals: 6 },
];

export default function SuperAdmin() {
  const router = useRouter();
  const [activeTab, setActiveTab] = useState('dashboard');
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  // Form states
  const [blockchainForm, setBlockchainForm] = useState<Partial<Blockchain>>({
    name: '',
    symbol: '',
    chainId: 0,
    rpcUrl: '',
    explorerUrl: '',
    decimals: 18,
    isActive: true,
    gasLimit: 21000,
    confirmations: 12,
  });

  const [tokenForm, setTokenForm] = useState<Partial<Token>>({
    chainId: 1,
    address: '',
    symbol: '',
    name: '',
    decimals: 18,
    isActive: true,
    isPopular: false,
    isStablecoin: false,
    priceUsd: 0,
  });

  const [stats, setStats] = useState<Stats>({
    totalUsers: 0,
    activeUsers: 0,
    totalTransactions: 0,
    totalVolume: 0,
    totalWallets: 0,
  });

  const [blockchains, setBlockchains] = useState<Blockchain[]>([]);
  const [tokens, setTokens] = useState<Token[]>([]);

  // Fetch data on mount
  useEffect(() => {
    fetchStats();
    fetchBlockchains();
    fetchTokens();
  }, []);

  const fetchStats = async () => {
    try {
      const response = await fetch('/api/v1/super-admin/stats', {
        headers: { 'Authorization': 'Bearer token' }
      });
      const data = await response.json();
      if (data.success) {
        setStats(data.data);
      }
    } catch (error) {
      console.error('Failed to fetch stats:', error);
    }
  };

  const fetchBlockchains = async () => {
    try {
      const response = await fetch('/api/v1/chains');
      const data = await response.json();
      if (data.success) {
        setBlockchains(data.data.chains);
      }
    } catch (error) {
      console.error('Failed to fetch blockchains:', error);
    }
  };

  const fetchTokens = async () => {
    try {
      const response = await fetch('/api/v1/tokens');
      const data = await response.json();
      if (data.success) {
        setTokens(data.data.tokens);
      }
    } catch (error) {
      console.error('Failed to fetch tokens:', error);
    }
  };

  const handleAddBlockchain = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setMessage(null);

    try {
      const response = await fetch('/api/v1/super-admin/blockchain', {
        method: 'POST',
        headers: { 
          'Content-Type': 'application/json',
          'Authorization': 'Bearer token'
        },
        body: JSON.stringify(blockchainForm),
      });
      const data = await response.json();
      
      if (data.success) {
        setMessage({ type: 'success', text: 'Blockchain added successfully!' });
        setBlockchainForm({
          name: '',
          symbol: '',
          chainId: 0,
          rpcUrl: '',
          explorerUrl: '',
          decimals: 18,
          isActive: true,
          gasLimit: 21000,
          confirmations: 12,
        });
        fetchBlockchains();
      } else {
        setMessage({ type: 'error', text: data.error || 'Failed to add blockchain' });
      }
    } catch (error) {
      setMessage({ type: 'error', text: 'Failed to add blockchain' });
    } finally {
      setLoading(false);
    }
  };

  const handleDeleteBlockchain = async (id: number) => {
    if (!confirm('Are you sure you want to delete this blockchain?')) return;
    
    setLoading(true);
    try {
      const response = await fetch(`/api/v1/super-admin/blockchain/${id}`, {
        method: 'DELETE',
        headers: { 'Authorization': 'Bearer token' }
      });
      const data = await response.json();
      
      if (data.success) {
        setMessage({ type: 'success', text: 'Blockchain deleted successfully!' });
        fetchBlockchains();
      } else {
        setMessage({ type: 'error', text: data.error || 'Failed to delete blockchain' });
      }
    } catch (error) {
      setMessage({ type: 'error', text: 'Failed to delete blockchain' });
    } finally {
      setLoading(false);
    }
  };

  const handleAddToken = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setMessage(null);

    try {
      const response = await fetch('/api/v1/super-admin/token', {
        method: 'POST',
        headers: { 
          'Content-Type': 'application/json',
          'Authorization': 'Bearer token'
        },
        body: JSON.stringify(tokenForm),
      });
      const data = await response.json();
      
      if (data.success) {
        setMessage({ type: 'success', text: 'Token added successfully!' });
        setTokenForm({
          chainId: 1,
          address: '',
          symbol: '',
          name: '',
          decimals: 18,
          isActive: true,
          isPopular: false,
          isStablecoin: false,
          priceUsd: 0,
        });
        fetchTokens();
      } else {
        setMessage({ type: 'error', text: data.error || 'Failed to add token' });
      }
    } catch (error) {
      setMessage({ type: 'error', text: 'Failed to add token' });
    } finally {
      setLoading(false);
    }
  };

  const handleDeleteToken = async (id: number) => {
    if (!confirm('Are you sure you want to delete this token?')) return;
    
    setLoading(true);
    try {
      const response = await fetch(`/api/v1/super-admin/token/${id}`, {
        method: 'DELETE',
        headers: { 'Authorization': 'Bearer token' }
      });
      const data = await response.json();
      
      if (data.success) {
        setMessage({ type: 'success', text: 'Token deleted successfully!' });
        fetchTokens();
      } else {
        setMessage({ type: 'error', text: data.error || 'Failed to delete token' });
      }
    } catch (error) {
      setMessage({ type: 'error', text: 'Failed to delete token' });
    } finally {
      setLoading(false);
    }
  };

  const formatNumber = (num: number) => {
    if (num >= 1e9) return (num / 1e9).toFixed(2) + 'B';
    if (num >= 1e6) return (num / 1e6).toFixed(2) + 'M';
    if (num >= 1e3) return (num / 1e3).toFixed(2) + 'K';
    return num.toString();
  };

  const formatCurrency = (num: number) => {
    return '$' + formatNumber(num);
  };

  return (
    <div className="min-h-screen bg-slate-900 text-slate-50">
      {/* Header */}
      <header className="bg-slate-800 border-b border-slate-700">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center gap-4">
              <span className="text-2xl">🐯</span>
              <h1 className="text-xl font-bold">TigerWallet Super Admin</h1>
            </div>
            <nav className="flex gap-4">
              <button
                onClick={() => router.push('/')}
                className="text-slate-400 hover:text-white transition-colors"
              >
                Back to Wallet
              </button>
            </nav>
          </div>
        </div>
      </header>

      {/* Message */}
      {message && (
        <div className={`max-w-7xl mx-auto px-4 pt-4 ${message.type === 'success' ? 'text-green-400' : 'text-red-400'}`}>
          {message.text}
        </div>
      )}

      {/* Stats */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="grid grid-cols-1 md:grid-cols-5 gap-4 mb-8">
          <div className="bg-slate-800 rounded-lg p-6">
            <div className="text-slate-400 text-sm">Total Users</div>
            <div className="text-2xl font-bold text-orange-500">{formatNumber(stats.totalUsers)}</div>
          </div>
          <div className="bg-slate-800 rounded-lg p-6">
            <div className="text-slate-400 text-sm">Active Users</div>
            <div className="text-2xl font-bold text-green-500">{formatNumber(stats.activeUsers)}</div>
          </div>
          <div className="bg-slate-800 rounded-lg p-6">
            <div className="text-slate-400 text-sm">Total Transactions</div>
            <div className="text-2xl font-bold text-blue-500">{formatNumber(stats.totalTransactions)}</div>
          </div>
          <div className="bg-slate-800 rounded-lg p-6">
            <div className="text-slate-400 text-sm">Total Volume</div>
            <div className="text-2xl font-bold text-purple-500">{formatCurrency(stats.totalVolume)}</div>
          </div>
          <div className="bg-slate-800 rounded-lg p-6">
            <div className="text-slate-400 text-sm">Total Wallets</div>
            <div className="text-2xl font-bold text-yellow-500">{formatNumber(stats.totalWallets)}</div>
          </div>
        </div>

        {/* Tabs */}
        <div className="flex border-b border-slate-700 mb-6">
          <button
            onClick={() => setActiveTab('dashboard')}
            className={`px-4 py-2 ${activeTab === 'dashboard' ? 'border-b-2 border-orange-500 text-orange-500' : 'text-slate-400'}`}
          >
            Dashboard
          </button>
          <button
            onClick={() => setActiveTab('blockchains')}
            className={`px-4 py-2 ${activeTab === 'blockchains' ? 'border-b-2 border-orange-500 text-orange-500' : 'text-slate-400'}`}
          >
            Blockchains
          </button>
          <button
            onClick={() => setActiveTab('tokens')}
            className={`px-4 py-2 ${activeTab === 'tokens' ? 'border-b-2 border-orange-500 text-orange-500' : 'text-slate-400'}`}
          >
            Tokens
          </button>
        </div>

        {/* Dashboard Tab */}
        {activeTab === 'dashboard' && (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="bg-slate-800 rounded-lg p-6">
              <h3 className="text-lg font-semibold mb-4">Quick Actions</h3>
              <div className="space-y-3">
                <button
                  onClick={() => setActiveTab('blockchains')}
                  className="w-full bg-orange-600 hover:bg-orange-700 text-white py-2 px-4 rounded-lg transition-colors"
                >
                  + Add New Blockchain
                </button>
                <button
                  onClick={() => setActiveTab('tokens')}
                  className="w-full bg-blue-600 hover:bg-blue-700 text-white py-2 px-4 rounded-lg transition-colors"
                >
                  + Add New Token
                </button>
              </div>
            </div>
            <div className="bg-slate-800 rounded-lg p-6">
              <h3 className="text-lg font-semibold mb-4">System Status</h3>
              <div className="space-y-3">
                <div className="flex justify-between">
                  <span className="text-slate-400">API Server</span>
                  <span className="text-green-500">● Online</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-slate-400">Database</span>
                  <span className="text-green-500">● Online</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-slate-400">Blockchain Nodes</span>
                  <span className="text-green-500">● 15 Active</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-slate-400">Supported Chains</span>
                  <span className="text-white">{blockchains.length}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-slate-400">Supported Tokens</span>
                  <span className="text-white">{tokens.length}+</span>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Blockchains Tab */}
        {activeTab === 'blockchains' && (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* Add Blockchain Form */}
            <div className="bg-slate-800 rounded-lg p-6">
              <h3 className="text-lg font-semibold mb-4">Add New Blockchain</h3>
              <form onSubmit={handleAddBlockchain} className="space-y-4">
                <div>
                  <label className="block text-sm text-slate-400 mb-1">Chain Name</label>
                  <input
                    type="text"
                    value={blockchainForm.name}
                    onChange={(e) => setBlockchainForm({ ...blockchainForm, name: e.target.value })}
                    className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2 text-white"
                    placeholder="e.g., Ethereum"
                    required
                  />
                </div>
                <div>
                  <label className="block text-sm text-slate-400 mb-1">Symbol</label>
                  <input
                    type="text"
                    value={blockchainForm.symbol}
                    onChange={(e) => setBlockchainForm({ ...blockchainForm, symbol: e.target.value })}
                    className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2 text-white"
                    placeholder="e.g., ETH"
                    required
                  />
                </div>
                <div>
                  <label className="block text-sm text-slate-400 mb-1">Chain ID</label>
                  <input
                    type="number"
                    value={blockchainForm.chainId}
                    onChange={(e) => setBlockchainForm({ ...blockchainForm, chainId: parseInt(e.target.value) })}
                    className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2 text-white"
                    placeholder="e.g., 1"
                    required
                  />
                </div>
                <div>
                  <label className="block text-sm text-slate-400 mb-1">RPC URL</label>
                  <input
                    type="url"
                    value={blockchainForm.rpcUrl}
                    onChange={(e) => setBlockchainForm({ ...blockchainForm, rpcUrl: e.target.value })}
                    className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2 text-white"
                    placeholder="https://..."
                    required
                  />
                </div>
                <div>
                  <label className="block text-sm text-slate-400 mb-1">Explorer URL</label>
                  <input
                    type="url"
                    value={blockchainForm.explorerUrl}
                    onChange={(e) => setBlockchainForm({ ...blockchainForm, explorerUrl: e.target.value })}
                    className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2 text-white"
                    placeholder="https://..."
                    required
                  />
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm text-slate-400 mb-1">Decimals</label>
                    <input
                      type="number"
                      value={blockchainForm.decimals}
                      onChange={(e) => setBlockchainForm({ ...blockchainForm, decimals: parseInt(e.target.value) })}
                      className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2 text-white"
                    />
                  </div>
                  <div>
                    <label className="block text-sm text-slate-400 mb-1">Gas Limit</label>
                    <input
                      type="number"
                      value={blockchainForm.gasLimit}
                      onChange={(e) => setBlockchainForm({ ...blockchainForm, gasLimit: parseInt(e.target.value) })}
                      className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2 text-white"
                    />
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    id="isActive"
                    checked={blockchainForm.isActive}
                    onChange={(e) => setBlockchainForm({ ...blockchainForm, isActive: e.target.checked })}
                    className="w-4 h-4"
                  />
                  <label htmlFor="isActive" className="text-sm text-slate-400">Active</label>
                </div>
                <button
                  type="submit"
                  disabled={loading}
                  className="w-full bg-orange-600 hover:bg-orange-700 disabled:bg-slate-600 text-white py-2 px-4 rounded-lg transition-colors"
                >
                  {loading ? 'Adding...' : 'Add Blockchain'}
                </button>
              </form>
            </div>

            {/* Blockchain List */}
            <div className="bg-slate-800 rounded-lg p-6">
              <h3 className="text-lg font-semibold mb-4">Supported Blockchains ({blockchains.length})</h3>
              <div className="space-y-2 max-h-[500px] overflow-y-auto">
                {blockchains.map((chain) => (
                  <div key={chain.id} className="flex items-center justify-between bg-slate-700 rounded-lg p-3">
                    <div>
                      <div className="font-semibold">{chain.name}</div>
                      <div className="text-sm text-slate-400">{chain.symbol} • Chain ID: {chain.chainId}</div>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className={`px-2 py-1 rounded text-xs ${chain.isActive ? 'bg-green-600' : 'bg-red-600'}`}>
                        {chain.isActive ? 'Active' : 'Inactive'}
                      </span>
                      <button
                        onClick={() => handleDeleteBlockchain(chain.id)}
                        className="text-red-400 hover:text-red-300"
                      >
                        Delete
                      </button>
                    </div>
                  </div>
                ))}
                {blockchains.length === 0 && (
                  <div className="text-center text-slate-400 py-8">
                    No blockchains added yet
                  </div>
                )}
              </div>
            </div>
          </div>
        )}

        {/* Tokens Tab */}
        {activeTab === 'tokens' && (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* Add Token Form */}
            <div className="bg-slate-800 rounded-lg p-6">
              <h3 className="text-lg font-semibold mb-4">Add New Token</h3>
              <form onSubmit={handleAddToken} className="space-y-4">
                <div>
                  <label className="block text-sm text-slate-400 mb-1">Blockchain</label>
                  <select
                    value={tokenForm.chainId}
                    onChange={(e) => setTokenForm({ ...tokenForm, chainId: parseInt(e.target.value) })}
                    className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2 text-white"
                    required
                  >
                    {CHAIN_TYPES.map((chain) => (
                      <option key={chain.value} value={chain.chainId}>
                        {chain.label} ({chain.chainId})
                      </option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="block text-sm text-slate-400 mb-1">Token Address (0x... for EVM)</label>
                  <input
                    type="text"
                    value={tokenForm.address}
                    onChange={(e) => setTokenForm({ ...tokenForm, address: e.target.value })}
                    className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2 text-white"
                    placeholder="0x... (leave empty for native)"
                  />
                </div>
                <div>
                  <label className="block text-sm text-slate-400 mb-1">Symbol</label>
                  <input
                    type="text"
                    value={tokenForm.symbol}
                    onChange={(e) => setTokenForm({ ...tokenForm, symbol: e.target.value })}
                    className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2 text-white"
                    placeholder="e.g., ETH"
                    required
                  />
                </div>
                <div>
                  <label className="block text-sm text-slate-400 mb-1">Name</label>
                  <input
                    type="text"
                    value={tokenForm.name}
                    onChange={(e) => setTokenForm({ ...tokenForm, name: e.target.value })}
                    className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2 text-white"
                    placeholder="e.g., Ethereum"
                    required
                  />
                </div>
                <div>
                  <label className="block text-sm text-slate-400 mb-1">Decimals</label>
                  <input
                    type="number"
                    value={tokenForm.decimals}
                    onChange={(e) => setTokenForm({ ...tokenForm, decimals: parseInt(e.target.value) })}
                    className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2 text-white"
                    required
                  />
                </div>
                <div>
                  <label className="block text-sm text-slate-400 mb-1">Price (USD)</label>
                  <input
                    type="number"
                    step="0.00000001"
                    value={tokenForm.priceUsd}
                    onChange={(e) => setTokenForm({ ...tokenForm, priceUsd: parseFloat(e.target.value) })}
                    className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2 text-white"
                    placeholder="0.00"
                  />
                </div>
                <div className="flex flex-wrap gap-4">
                  <div className="flex items-center gap-2">
                    <input
                      type="checkbox"
                      id="isPopular"
                      checked={tokenForm.isPopular}
                      onChange={(e) => setTokenForm({ ...tokenForm, isPopular: e.target.checked })}
                      className="w-4 h-4"
                    />
                    <label htmlFor="isPopular" className="text-sm text-slate-400">Popular</label>
                  </div>
                  <div className="flex items-center gap-2">
                    <input
                      type="checkbox"
                      id="isStablecoin"
                      checked={tokenForm.isStablecoin}
                      onChange={(e) => setTokenForm({ ...tokenForm, isStablecoin: e.target.checked })}
                      className="w-4 h-4"
                    />
                    <label htmlFor="isStablecoin" className="text-sm text-slate-400">Stablecoin</label>
                  </div>
                  <div className="flex items-center gap-2">
                    <input
                      type="checkbox"
                      id="tokenIsActive"
                      checked={tokenForm.isActive}
                      onChange={(e) => setTokenForm({ ...tokenForm, isActive: e.target.checked })}
                      className="w-4 h-4"
                    />
                    <label htmlFor="tokenIsActive" className="text-sm text-slate-400">Active</label>
                  </div>
                </div>
                <button
                  type="submit"
                  disabled={loading}
                  className="w-full bg-blue-600 hover:bg-blue-700 disabled:bg-slate-600 text-white py-2 px-4 rounded-lg transition-colors"
                >
                  {loading ? 'Adding...' : 'Add Token'}
                </button>
              </form>
            </div>

            {/* Token List */}
            <div className="bg-slate-800 rounded-lg p-6">
              <h3 className="text-lg font-semibold mb-4">Supported Tokens ({tokens.length}+)</h3>
              <div className="space-y-2 max-h-[500px] overflow-y-auto">
                {POPULAR_TOKENS.slice(0, 30).map((token, index) => (
                  <div key={index} className="flex items-center justify-between bg-slate-700 rounded-lg p-3">
                    <div>
                      <div className="font-semibold">{token.symbol}</div>
                      <div className="text-sm text-slate-400">{token.name}</div>
                    </div>
                    <span className="text-xs text-slate-500">
                      {token.decimals} decimals
                    </span>
                  </div>
                ))}
                {tokens.map((token) => (
                  <div key={token.id} className="flex items-center justify-between bg-slate-700 rounded-lg p-3">
                    <div>
                      <div className="font-semibold">{token.symbol}</div>
                      <div className="text-sm text-slate-400">{token.name} • Chain: {token.chainId}</div>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className={`px-2 py-1 rounded text-xs ${token.isActive ? 'bg-green-600' : 'bg-red-600'}`}>
                        {token.isActive ? 'Active' : 'Inactive'}
                      </span>
                      <button
                        onClick={() => handleDeleteToken(token.id)}
                        className="text-red-400 hover:text-red-300"
                      >
                        Delete
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
