"use client";

import { useState, useCallback, useEffect } from "react";
import { 
  Wallet, 
  Search, 
  Filter,
  Grid,
  List,
  ChevronDown,
  CheckCircle,
  ExternalLink,
  Zap,
  Globe,
  Bitcoin,
  Layers,
  Activity,
  RefreshCw,
  Plus,
  ArrowRight
} from "lucide-react";

// Comprehensive Blockchain Registry
interface Blockchain {
  id: number;
  name: string;
  symbol: string;
  type: "evm" | "non-evm";
  rpcUrl: string;
  explorer: string;
  chainId: number;
  logo: string;
  color: string;
  isTestnet: boolean;
  features: string[];
  nativeToken: {
    name: string;
    symbol: string;
    decimals: number;
    address: string;
  };
}

// Top 50 EVM Chains + Top 50 Non-EVM Chains
const BLOCKCHAINS: Blockchain[] = [
  // === TOP EVM CHAINS ===
  { id: 1, name: "Ethereum", symbol: "ETH", type: "evm", rpcUrl: "https://eth.llamarpc.com", explorer: "https://etherscan.io", chainId: 1, logo: "⬡", color: "#627EEA", isTestnet: false, features: ["smart-contracts", "defi", "nft", "staking"], nativeToken: { name: "Ether", symbol: "ETH", decimals: 18, address: "0x0000000000000000000000000000000000000000" }},
  { id: 56, name: "BNB Smart Chain", symbol: "BNB", type: "evm", rpcUrl: "https://bsc-dataseed.binance.org", explorer: "https://bscscan.com", chainId: 56, logo: "📘", color: "#F3BA2F", isTestnet: false, features: ["defi", "nft", "staking", "gaming"], nativeToken: { name: "BNB", symbol: "BNB", decimals: 18, address: "0x0000000000000000000000000000000000000000" }},
  { id: 137, name: "Polygon", symbol: "MATIC", type: "evm", rpcUrl: "https://polygon-rpc.com", explorer: "https://polygonscan.com", chainId: 137, logo: "🔷", color: "#8247E5", isTestnet: false, features: ["defi", "nft", "gaming", "enterprise"], nativeToken: { name: "MATIC", symbol: "MATIC", decimals: 18, address: "0x0000000000000000000000000000000000000000" }},
  { id: 10, name: "Optimism", symbol: "OP", type: "evm", rpcUrl: "https://mainnet.optimism.io", explorer: "https://optimistic.etherscan.io", chainId: 10, logo: "🔴", color: "#FF0420", isTestnet: false, features: ["scaling", "defi", "nft"], nativeToken: { name: "Ether", symbol: "ETH", decimals: 18, address: "0x0000000000000000000000000000000000000000" }},
  { id: 42161, name: "Arbitrum One", symbol: "ETH", type: "evm", rpcUrl: "https://arb1.arbitrum.io/rpc", explorer: "https://arbiscan.io", chainId: 42161, logo: "🔵", color: "#28A0F0", isTestnet: false, features: ["scaling", "defi", "nft"], nativeToken: { name: "Ether", symbol: "ETH", decimals: 18, address: "0x0000000000000000000000000000000000000000" }},
  { id: 8453, name: "Base", symbol: "ETH", type: "evm", rpcUrl: "https://mainnet.base.org", explorer: "https://basescan.org", chainId: 8453, logo: "🔵", color: "#0052FF", isTestnet: false, features: ["defi", "nft", "social"], nativeToken: { name: "Ether", symbol: "ETH", decimals: 18, address: "0x0000000000000000000000000000000000000000" }},
  { id: 43114, name: "Avalanche C-Chain", symbol: "AVAX", type: "evm", rpcUrl: "https://api.avax.network/ext/bc/C/rpc", explorer: "https://snowtrace.io", chainId: 43114, logo: "🔺", color: "#E84142", isTestnet: false, features: ["defi", "nft", "gaming"], nativeToken: { name: "Avalanche", symbol: "AVAX", decimals: 18, address: "0x0000000000000000000000000000000000000000" }},
  { id: 250, name: "Fantom", symbol: "FTM", type: "evm", rpcUrl: "https://rpc.fantom.network", explorer: "https://ftmscan.com", chainId: 250, logo: "👻", color: "#1969FF", isTestnet: false, features: ["defi", "nft", "gaming"], nativeToken: { name: "Fantom", symbol: "FTM", decimals: 18, address: "0x0000000000000000000000000000000000000000" }},
  { id: 100, name: "Gnosis Chain", symbol: "GNO", type: "evm", rpcUrl: "https://rpc.gnosischain.com", explorer: "https://gnosisscan.io", chainId: 100, logo: "🟠", color: "#47725B", isTestnet: false, features: ["defi", "dao"], nativeToken: { name: "Gnosis", symbol: "GNO", decimals: 18, address: "0x0000000000000000000000000000000000000000" }},
  { id: 1284, name: "Moonbeam", symbol: "GLMR", type: "evm", rpcUrl: "https://rpc.api.moonbeam.network", explorer: "https://moonscan.io", chainId: 1284, logo: "🌙", color: "#53CBC9", isTestnet: false, features: ["defi", "nft", "interoperability"], nativeToken: { name: "Moonbeam", symbol: "GLMR", decimals: 18, address: "0x0000000000000000000000000000000000000000" }},
  { id: 1285, name: "Moonriver", symbol: "MOVR", type: "evm", rpcUrl: "https://rpc.moonriver.moonbeam.network", explorer: "https://moonriver.moonscan.io", chainId: 1285, logo: "🌊", color: "#5AAD93", isTestnet: false, features: ["defi", "nft", "interoperability"], nativeToken: { name: "Moonriver", symbol: "MOVR", decimals: 18, address: "0x0000000000000000000000000000000000000000" }},
  { id: 1666600000, name: "Harmony", symbol: "ONE", type: "evm", rpcUrl: "https://api.harmony.one", explorer: "https://explorer.harmony.one", chainId: 1666600000, logo: "✨", color: "#00AEE9", isTestnet: false, features: ["defi", "nft", "staking"], nativeToken: { name: "Harmony", symbol: "ONE", decimals: 18, address: "0x0000000000000000000000000000000000000000" }},
  { id: 8217, name: "Klaytn", symbol: "KLAY", type: "evm", rpcUrl: "https://public-en-cypress.klaytn.net", explorer: "https://scope.klaytn.com", chainId: 8217, logo: "🟢", color: "#39D353", isTestnet: false, features: ["defi", "nft", "gaming"], nativeToken: { name: "Klaytn", symbol: "KLAY", decimals: 18, address: "0x0000000000000000000000000000000000000000" }},
  { id: 42220, name: "Celo", symbol: "CELO", type: "evm", rpcUrl: "https://forno.celo.org", explorer: "https://celoscan.com", chainId: 42220, logo: "🌐", color: "#35D07F", isTestnet: false, features: ["defi", "payments", "social-impact"], nativeToken: { name: "Celo", symbol: "CELO", decimals: 18, address: "0x0000000000000000000000000000000000000000" }},
  { id: 11155111, name: "Ethereum Sepolia", symbol: "ETH", type: "evm", rpcUrl: "https://rpc.sepolia.org", explorer: "https://sepolia.etherscan.io", chainId: 11155111, logo: "⬡", color: "#627EEA", isTestnet: true, features: ["testnet"], nativeToken: { name: "Sepolia Ether", symbol: "ETH", decimals: 18, address: "0x0000000000000000000000000000000000000000" }},
  { id: 80001, name: "Polygon Mumbai", symbol: "MATIC", type: "evm", rpcUrl: "https://rpc-mumbai.maticvigil.com", explorer: "https://mumbai.polygonscan.com", chainId: 80001, logo: "🔷", color: "#8247E5", isTestnet: true, features: ["testnet"], nativeToken: { name: "Mumbai MATIC", symbol: "MATIC", decimals: 18, address: "0x0000000000000000000000000000000000000000" }},
  { id: 420, name: "Optimism Goerli", symbol: "ETH", type: "evm", rpcUrl: "https://goerli.optimism.io", explorer: "https://goerli-optimism.etherscan.io", chainId: 420, logo: "🔴", color: "#FF0420", isTestnet: true, features: ["testnet"], nativeToken: { name: "Goerli Ether", symbol: "ETH", decimals: 18, address: "0x0000000000000000000000000000000000000000" }},
  { id: 421613, name: "Arbitrum Goerli", symbol: "ETH", type: "evm", rpcUrl: "https://goerli-rollup.arbitrum.io/rpc", explorer: "https://goerli.arbiscan.io", chainId: 421613, logo: "🔵", color: "#28A0F0", isTestnet: true, features: ["testnet"], nativeToken: { name: "Goerli Ether", symbol: "ETH", decimals: 18, address: "0x0000000000000000000000000000000000000000" }},
  { id: 84531, name: "Base Goerli", symbol: "ETH", type: "evm", rpcUrl: "https://goerli.base.org", explorer: "https://goerli.basescan.org", chainId: 84531, logo: "🔵", color: "#0052FF", isTestnet: true, features: ["testnet"], nativeToken: { name: "Goerli Ether", symbol: "ETH", decimals: 18, address: "0x0000000000000000000000000000000000000000" }},
  { id: 97, name: "BNB Testnet", symbol: "BNB", type: "evm", rpcUrl: "https://data-seed-prebsc-1-s1.bnbchain.org", explorer: "https://testnet.bscscan.com", chainId: 97, logo: "📘", color: "#F3BA2F", isTestnet: true, features: ["testnet"], nativeToken: { name: "BNB", symbol: "BNB", decimals: 18, address: "0x0000000000000000000000000000000000000000" }},
  { id: 11155420, name: "Optimism Sepolia", symbol: "OP", type: "evm", rpcUrl: "https://sepolia.optimism.io", explorer: "https://sepolia-optimism.etherscan.io", chainId: 11155420, logo: "🔴", color: "#FF0420", isTestnet: true, features: ["testnet"], nativeToken: { name: "Sepolia Ether", symbol: "ETH", decimals: 18, address: "0x0000000000000000000000000000000000000000" }},
  
  // === Additional Popular EVM Chains ===
  { id: 324, name: "zkSync Era", symbol: "ETH", type: "evm", rpcUrl: "https://mainnet.era.zksync.io", explorer: "https://explorer.zksync.io", chainId: 324, logo: "⚡", color: "#8B5CF6", isTestnet: false, features: ["zk-rollup", "defi", "nft"], nativeToken: { name: "Ether", symbol: "ETH", decimals: 18, address: "0x0000000000000000000000000000000000000000" }},
  { id: 59144, name: "Linea", symbol: "ETH", type: "evm", rpcUrl: "https://rpc.linea.build", explorer: "https://lineascan.build", chainId: 59144, logo: "🔷", color: "#121212", isTestnet: false, features: ["zk-rollup", "defi", "nft"], nativeToken: { name: "Ether", symbol: "ETH", decimals: 18, address: "0x0000000000000000000000000000000000000000" }},
  { id: 534352, name: "Scroll", symbol: "ETH", type: "evm", rpcUrl: "https://rpc.scroll.io", explorer: "https://scrollscan.com", chainId: 534352, logo: "📜", color: "#CDA4DE", isTestnet: false, features: ["zk-rollup", "defi", "nft"], nativeToken: { name: "Ether", symbol: "ETH", decimals: 18, address: "0x0000000000000000000000000000000000000000" }},
  { id: 1101, name: "Polygon zkEVM", symbol: "ETH", type: "evm", rpcUrl: "https://zkevm-rpc.polygon.technology", explorer: "https://zkevm.polygonscan.com", chainId: 1101, logo: "🔷", color: "#8247E5", isTestnet: false, features: ["zk-rollup", "defi", "nft"], nativeToken: { name: "Ether", symbol: "ETH", decimals: 18, address: "0x0000000000000000000000000000000000000000" }},
  { id: 5000, name: "Mantle", symbol: "MNT", type: "evm", rpcUrl: "https://rpc.mantle.xyz", explorer: "https://mantlescan.info", chainId: 5000, logo: "🟣", color: "#1A1A2E", isTestnet: false, features: ["defi", "restaking", "nft"], nativeToken: { name: "Mantle", symbol: "MNT", decimals: 18, address: "0x0000000000000000000000000000000000000000" }},
  { id: 5001, name: "Mantle Sepolia", symbol: "MNT", type: "evm", rpcUrl: "https://rpc.sepolia.mantle.xyz", explorer: "https://sepolia.mantlescan.info", chainId: 5001, logo: "🟣", color: "#1A1A2E", isTestnet: true, features: ["testnet"], nativeToken: { name: "Mantle", symbol: "MNT", decimals: 18, address: "0x0000000000000000000000000000000000000000" }},
  { id: 2000, name: "Dogecoin", symbol: "DOGE", type: "evm", rpcUrl: "https://rpc.dogecoin.com", explorer: "https://dogescan.io", chainId: 2000, logo: "🐕", color: "#C2A633", isTestnet: false, features: ["meme", "payments"], nativeToken: { name: "Dogecoin", symbol: "DOGE", decimals: 18, address: "0x0000000000000000000000000000000000000000" }},
  { id: 5700, name: "Rollux", symbol: "SYS", type: "evm", rpcUrl: "https://rpc.rollux.com", explorer: "https://explorer.rollux.com", chainId: 5700, logo: "⚫", color: "#000000", isTestnet: false, features: ["defi", "nft"], nativeToken: { name: "Rollux", symbol: "SYS", decimals: 18, address: "0x0000000000000000000000000000000000000000" }},
  { id: 2810, name: "Morph", symbol: "ETH", type: "evm", rpcUrl: "https://rpc.morphl2.io", explorer: "https://explorer.morphl2.io", chainId: 2810, logo: "🟦", color: "#6B5CE7", isTestnet: false, features: ["defi", "nft"], nativeToken: { name: "Ether", symbol: "ETH", decimals: 18, address: "0x0000000000000000000000000000000000000000" }},
  { id: 204, name: "opBNB", symbol: "BNB", type: "evm", rpcUrl: "https://opbnb-mainnet-rpc.bnbchain.org", explorer: "https://opbnb.bscscan.com", chainId: 204, logo: "📘", color: "#F3BA2F", isTestnet: false, features: ["scaling", "defi", "gaming"], nativeToken: { name: "BNB", symbol: "BNB", decimals: 18, address: "0x0000000000000000000000000000000000000000" }},
  
  // === NON-EVM CHAINS ===
  { id: 1399811149, name: "Solana", symbol: "SOL", type: "non-evm", rpcUrl: "https://api.mainnet-beta.solana.com", explorer: "https://explorer.solana.com", chainId: 1399811149, logo: "☀️", color: "#9945FF", isTestnet: false, features: ["defi", "nft", "payments", "gaming"], nativeToken: { name: "Solana", symbol: "SOL", decimals: 9, address: "" }},
  { id: 101, name: "Solana Devnet", symbol: "SOL", type: "non-evm", rpcUrl: "https://api.devnet.solana.com", explorer: "https://explorer.devnet.solana.com", chainId: 101, logo: "☀️", color: "#9945FF", isTestnet: true, features: ["testnet"], nativeToken: { name: "Solana", symbol: "SOL", decimals: 9, address: "" }},
  { id: 102, name: "Solana Testnet", symbol: "SOL", type: "non-evm", rpcUrl: "https://api.testnet.solana.com", explorer: "https://explorer.testnet.solana.com", chainId: 102, logo: "☀️", color: "#9945FF", isTestnet: true, features: ["testnet"], nativeToken: { name: "Solana", symbol: "SOL", decimals: 9, address: "" }},
  { id: 0, name: "Bitcoin", symbol: "BTC", type: "non-evm", rpcUrl: "https://btc.llamarpc.com", explorer: "https://mempool.space", chainId: 0, logo: "₿", color: "#F7931A", isTestnet: false, features: ["payments", "store-of-value", "ordinals"], nativeToken: { name: "Bitcoin", symbol: "BTC", decimals: 8, address: "" }},
  { id: 0, name: "Bitcoin Signet", symbol: "BTC", type: "non-evm", rpcUrl: "https://bitcoin.signet.io", explorer: "https://mempool.space/signet", chainId: 0, logo: "₿", color: "#F7931A", isTestnet: true, features: ["testnet"], nativeToken: { name: "Bitcoin", symbol: "BTC", decimals: 8, address: "" }},
  { id: 0, name: "Bitcoin Testnet", symbol: "BTC", type: "non-evm", rpcUrl: "https://bitcoin-testnet.llamarpc.com", explorer: "https://mempool.space/testnet", chainId: 0, logo: "₿", color: "#F7931A", isTestnet: true, features: ["testnet"], nativeToken: { name: "Bitcoin", symbol: "BTC", decimals: 8, address: "" }},
  { id: 1, name: "Cosmos Hub", symbol: "ATOM", type: "non-evm", rpcUrl: "https://rpc.cosmoshub.io", explorer: "https://mintscan.io/cosmos-hub", chainId: 1, logo: "🌌", color: "#2E3148", isTestnet: false, features: ["interoperability", "staking", "dao"], nativeToken: { name: "Cosmos Hub", symbol: "ATOM", decimals: 6, address: "" }},
  { id: 2, name: "Osmosis", symbol: "OSMO", type: "non-evm", rpcUrl: "https://rpc.osmosis.zone", explorer: "https://mintscan.io/osmosis", chainId: 2, logo: "💧", color: "#A532FF", isTestnet: false, features: ["defi", "amm", "nft"], nativeToken: { name: "Osmosis", symbol: "OSMO", decimals: 6, address: "" }},
  { id: 3, name: "Terra Classic", symbol: "LUNC", type: "non-evm", rpcUrl: "https://rpc.terra.dev", explorer: "https://finder.terra.money", chainId: 3, logo: "🌍", color: "#2F4F4F", isTestnet: false, features: ["stablecoins", "defi"], nativeToken: { name: "Terra Classic", symbol: "LUNC", decimals: 6, address: "" }},
  { id: 4, name: "Terra", symbol: "UST", type: "non-evm", rpcUrl: "https://terra-rpc.lavenderfive.com", explorer: "https://finder.terra.money", chainId: 4, logo: "🌍", color: "#2F4F4F", isTestnet: false, features: ["stablecoins", "defi"], nativeToken: { name: "Terra", symbol: "UST", decimals: 6, address: "" }},
  { id: 5, name: "Secret Network", symbol: "SCRT", type: "non-evm", rpcUrl: "https://rpc.secretnodes.com", explorer: "https://mintscan.io/secret", chainId: 5, logo: "🔐", color: "#4B4F5D", isTestnet: false, features: ["privacy", "defi"], nativeToken: { name: "Secret", symbol: "SCRT", decimals: 6, address: "" }},
  { id: 6, name: "Kava", symbol: "KAVA", type: "non-evm", rpcUrl: "https://rpc.kava.io", explorer: "https://kavascan.com", chainId: 6, logo: "🦁", color: "#FF4136", isTestnet: false, features: ["defi", "lending", "staking"], nativeToken: { name: "Kava", symbol: "KAVA", decimals: 6, address: "" }},
  { id: 7, name: "Injective", symbol: "INJ", type: "non-evm", rpcUrl: "https://injective-rpc.lavenderfive.com", explorer: "https://explorer.injective.network", chainId: 7, logo: "💉", color: "#00F2FE", isTestnet: false, features: ["defi", "derivatives", "spot-trading"], nativeToken: { name: "Injective", symbol: "INJ", decimals: 18, address: "" }},
  { id: 8, name: "Stargaze", symbol: "STARS", type: "non-evm", rpcUrl: "https://rpc.stargaze.zone", explorer: "https://mintscan.io/stargaze", chainId: 8, logo: "⭐", color: "#5944F6", isTestnet: false, features: ["nft", "dao"], nativeToken: { name: "Stargaze", symbol: "STARS", decimals: 6, address: "" }},
  { id: 9, name: "Juno", symbol: "JUNO", type: "non-evm", rpcUrl: "https://rpc.juno.zone", explorer: "https://mintscan.io/juno", chainId: 9, logo: "🪐", color: "#5700D7", isTestnet: false, features: ["defi", "nft", "dao"], nativeToken: { name: "Juno", symbol: "JUNO", decimals: 6, address: "" }},
  { id: 10, name: "Cronos", symbol: "CRO", type: "non-evm", rpcUrl: "https://rpc.cronos.org", explorer: "https://cronoscan.org", chainId: 10, logo: "⏰", color: "#002D74", isTestnet: false, features: ["defi", "nft", "payments"], nativeToken: { name: "Cronos", symbol: "CRO", decimals: 8, address: "" }},
  { id: 11, name: "Aptos", symbol: "APT", type: "non-evm", rpcUrl: "https://aptos-mainnet.nodereal.io", explorer: "https://explorer.aptoslabs.com", chainId: 11, logo: "▲", color: "#14F195", isTestnet: false, features: ["defi", "nft", "gaming"], nativeToken: { name: "Aptos", symbol: "APT", decimals: 8, address: "" }},
  { id: 12, name: "Aptos Testnet", symbol: "APT", type: "non-evm", rpcUrl: "https://aptos-testnet.nodereal.io", explorer: "https://explorer.aptoslabs.com", chainId: 12, logo: "▲", color: "#14F195", isTestnet: true, features: ["testnet"], nativeToken: { name: "Aptos", symbol: "APT", decimals: 8, address: "" }},
  { id: 13, name: "Sui", symbol: "SUI", type: "non-evm", rpcUrl: "https://fullnode.mainnet.sui.io", explorer: "https://suiscan.xyz", chainId: 13, logo: "🔵", color: "#6ADBD3", isTestnet: false, features: ["defi", "nft", "gaming"], nativeToken: { name: "Sui", symbol: "SUI", decimals: 9, address: "" }},
  { id: 14, name: "Sui Testnet", symbol: "SUI", type: "non-evm", rpcUrl: "https://fullnode.testnet.sui.io", explorer: "https://suiscan.xyz/testnet", chainId: 14, logo: "🔵", color: "#6ADBD3", isTestnet: true, features: ["testnet"], nativeToken: { name: "Sui", symbol: "SUI", decimals: 9, address: "" }},
  { id: 15, name: "Toncoin", symbol: "TON", type: "non-evm", rpcUrl: "https://toncenter.com/api/v2", explorer: "https://tonscan.org", chainId: 15, logo: "📱", color: "#0098EA", isTestnet: false, features: ["payments", "defi", "mini-apps"], nativeToken: { name: "Toncoin", symbol: "TON", decimals: 9, address: "" }},
  { id: 16, name: "Telegram Testnet", symbol: "TON", type: "non-evm", rpcUrl: "https://testnet.toncenter.com/api/v2", explorer: "https://testnet.tonscan.org", chainId: 16, logo: "📱", color: "#0098EA", isTestnet: true, features: ["testnet"], nativeToken: { name: "Toncoin", symbol: "TON", decimals: 9, address: "" }},
  { id: 17, name: "Polkadot", symbol: "DOT", type: "non-evm", rpcUrl: "https://rpc.polkadot.io", explorer: "https://polkadot.subscan.io", chainId: 17, logo: "🔴", color: "#E6007A", isTestnet: false, features: ["interoperability", "staking", "nft"], nativeToken: { name: "Polkadot", symbol: "DOT", decimals: 10, address: "" }},
  { id: 18, name: "Kusama", symbol: "KSM", type: "non-evm", rpcUrl: "https://kusama-rpc.polkadot.io", explorer: "https://kusama.subscan.io", chainId: 18, logo: "⚫", color: "#000000", isTestnet: false, features: ["interoperability", "staking", "canary-network"], nativeToken: { name: "Kusama", symbol: "KSM", decimals: 12, address: "" }},
  { id: 19, name: "Aleph Zero", symbol: "AZERO", type: "non-evm", rpcUrl: "https://rpc.azero.dev", explorer: "https://azero.dev", chainId: 19, logo: "✨", color: "#0099FF", isTestnet: false, features: ["defi", "privacy", "staking"], nativeToken: { name: "Aleph Zero", symbol: "AZERO", decimals: 12, address: "" }},
  { id: 20, name: "Near", symbol: "NEAR", type: "non-evm", rpcUrl: "https://rpc.mainnet.near.org", explorer: "https://explorer.near.org", chainId: 20, logo: "🟢", color: "#000000", isTestnet: false, features: ["defi", "nft", "gaming", "storage"], nativeToken: { name: "NEAR", symbol: "NEAR", decimals: 24, address: "" }},
  { id: 21, name: "Near Testnet", symbol: "NEAR", type: "non-evm", rpcUrl: "https://rpc.testnet.near.org", explorer: "https://explorer.testnet.near.org", chainId: 21, logo: "🟢", color: "#000000", isTestnet: true, features: ["testnet"], nativeToken: { name: "NEAR", symbol: "NEAR", decimals: 24, address: "" }},
  { id: 22, name: "Algorand", symbol: "ALGO", type: "non-evm", rpcUrl: "https://mainnet-api.algonode.cloud", explorer: "https://algoexplorer.io", chainId: 22, logo: "🔷", color: "#000000", isTestnet: false, features: ["defi", "payments", "asa"], nativeToken: { name: "Algorand", symbol: "ALGO", decimals: 6, address: "" }},
  { id: 23, name: "Algorand Testnet", symbol: "ALGO", type: "non-evm", rpcUrl: "https://testnet-api.algonode.cloud", explorer: "https://testnet.algoexplorer.io", chainId: 23, logo: "🔷", color: "#000000", isTestnet: true, features: ["testnet"], nativeToken: { name: "Algorand", symbol: "ALGO", decimals: 6, address: "" }},
  { id: 24, name: "Cardano", symbol: "ADA", type: "non-evm", rpcUrl: "https://cardano-mainnet.blockfrost.io", explorer: "https://cardanoscan.io", chainId: 24, logo: "🃏", color: "#0033AD", isTestnet: false, features: ["defi", "nft", "smart-contracts"], nativeToken: { name: "Cardano", symbol: "ADA", decimals: 6, address: "" }},
  { id: 25, name: "Cardano Preprod", symbol: "ADA", type: "non-evm", rpcUrl: "https://cardano-preprod.blockfrost.io", explorer: "https://preprod.cardanoscan.io", chainId: 25, logo: "🃏", color: "#0033AD", isTestnet: true, features: ["testnet"], nativeToken: { name: "Cardano", symbol: "ADA", decimals: 6, address: "" }},
  { id: 26, name: "Radix", symbol: "XRD", type: "non-evm", rpcUrl: "https://mainnet.radixdlt.com", explorer: "https://dashboard.radixdlt.com", chainId: 26, logo: "🔴", color: "#0B4E68", isTestnet: false, features: ["defi", "smart-contracts", "tokenization"], nativeToken: { name: "Radix", symbol: "XRD", decimals: 18, address: "" }},
  { id: 27, name: "VeChain", symbol: "VET", type: "non-evm", rpcUrl: "https://mainnet.eternals.io", explorer: "https://vechainstats.com", chainId: 27, logo: "💚", color: "#15BDFF", isTestnet: false, features: ["enterprise", "supply-chain", "nft"], nativeToken: { name: "VeChain", symbol: "VET", decimals: 18, address: "" }},
  { id: 28, name: "Hedera", symbol: "HBAR", type: "non-evm", rpcUrl: "https://mainnet.hashio.io", explorer: "https://hashscan.io", chainId: 28, logo: "🌿", color: "#00EECC", isTestnet: false, features: ["enterprise", "defi", "nft"], nativeToken: { name: "Hedera", symbol: "HBAR", decimals: 8, address: "" }},
  { id: 29, name: "Hedera Testnet", symbol: "HBAR", type: "non-evm", rpcUrl: "https://testnet.hashio.io", explorer: "https://testnet.hashscan.io", chainId: 29, logo: "🌿", color: "#00EECC", isTestnet: true, features: ["testnet"], nativeToken: { name: "Hedera", symbol: "HBAR", decimals: 8, address: "" }},
  { id: 30, name: "Stacks", symbol: "STX", type: "non-evm", rpcUrl: "https://stacks-node-api.mainnet.stacks.co", explorer: "https://stacks.co", chainId: 30, logo: "📦", color: "#5546FF", isTestnet: false, features: ["defi", "nft", "bitcoin-l2"], nativeToken: { name: "Stacks", symbol: "STX", decimals: 6, address: "" }},
  { id: 31, name: "Stacks Testnet", symbol: "STX", type: "non-evm", rpcUrl: "https://stacks-node-api.testnet.stacks.co", explorer: "https://testnet.stacks.co", chainId: 31, logo: "📦", color: "#5546FF", isTestnet: true, features: ["testnet"], nativeToken: { name: "Stacks", symbol: "STX", decimals: 6, address: "" }},
  { id: 32, name: "ICP", symbol: "ICP", type: "non-evm", rpcUrl: "https://icp-api.io", explorer: "https://dashboard.internetcomputer.org", chainId: 32, logo: "🌐", color: "#29ABE2", isTestnet: false, features: ["defi", "nft", "daaas"], nativeToken: { name: "Internet Computer", symbol: "ICP", decimals: 8, address: "" }},
  { id: 33, name: "ICP Testnet", symbol: "ICP", type: "non-evm", rpcUrl: "https://icp-api.io", explorer: "https://dashboard.internetcomputer.org", chainId: 33, logo: "🌐", color: "#29ABE2", isTestnet: true, features: ["testnet"], nativeToken: { name: "Internet Computer", symbol: "ICP", decimals: 8, address: "" }},
  { id: 34, name: "Flow", symbol: "FLOW", type: "non-evm", rpcUrl: "https://flow-mainnet.g.alchemy.com/v2/demo", explorer: "https://flowscan.io", chainId: 34, logo: "🌊", color: "#00EF8B", isTestnet: false, features: ["nft", "gaming", "defi"], nativeToken: { name: "Flow", symbol: "FLOW", decimals: 8, address: "" }},
  { id: 35, name: "Flow Testnet", symbol: "FLOW", type: "non-evm", rpcUrl: "https://flow-testnet.g.alchemy.com/v2/demo", explorer: "https://testnet.flowscan.io", chainId: 35, logo: "🌊", color: "#00EF8B", isTestnet: true, features: ["testnet"], nativeToken: { name: "Flow", symbol: "FLOW", decimals: 8, address: "" }},
  { id: 36, name: "Tezos", symbol: "XTZ", type: "non-evm", rpcUrl: "https://mainnet.api.tez.ie", explorer: "https://tzkt.io", chainId: 36, logo: "🔵", color: "#2C7DF7", isTestnet: false, features: ["defi", "nft", "dao"], nativeToken: { name: "Tezos", symbol: "XTZ", decimals: 6, address: "" }},
  { id: 37, name: "Tezos Ghostnet", symbol: "XTZ", type: "non-evm", rpcUrl: "https://ghostnet.api.tez.ie", explorer: "https://ghostnet.tzkt.io", chainId: 37, logo: "🔵", color: "#2C7DF7", isTestnet: true, features: ["testnet"], nativeToken: { name: "Tezos", symbol: "XTZ", decimals: 6, address: "" }},
  { id: 38, name: "Polygon ID", symbol: "ID", type: "non-evm", rpcUrl: "https://rpc.polygonid.xyz", explorer: "https://polygonscan.com", chainId: 38, logo: "🆔", color: "#8247E5", isTestnet: false, features: ["identity", "zero-knowledge"], nativeToken: { name: "Polygon ID", symbol: "ID", decimals: 18, address: "" }},
  { id: 39, name: "Nervos Network", symbol: "CKB", type: "non-evm", rpcUrl: "https://mainnet-api.nervos.org", explorer: "https://explorer.nervos.org", chainId: 39, logo: "🧠", color: "#3CC98B", isTestnet: false, features: ["defi", "smart-contracts", "interoperability"], nativeToken: { name: "Nervos", symbol: "CKB", decimals: 8, address: "" }},
  { id: 40, name: "EOS", symbol: "EOS", type: "non-evm", rpcUrl: "https://api.eosn.io", explorer: "https://bloks.io", chainId: 40, logo: "🔴", color: "#000000", isTestnet: false, features: ["defi", "nft", "gaming"], nativeToken: { name: "EOS", symbol: "EOS", decimals: 4, address: "" }},
  { id: 41, name: "Telos", symbol: "TLOS", type: "non-evm", rpcUrl: "https://mainnet.telos.net", explorer: "https://www.teloscan.io", chainId: 41, logo: "📡", color: "#0066FF", isTestnet: false, features: ["defi", "nft", "gaming"], nativeToken: { name: "Telos", symbol: "TLOS", decimals: 4, address: "" }},
  { id: 42, name: "WAX", symbol: "WAX", type: "non-evm", rpcUrl: "https://waxapi.net", explorer: "https://wax.bloks.io", chainId: 42, logo: "🐝", color: "#F8C93D", isTestnet: false, features: ["nft", "gaming", "defi"], nativeToken: { name: "WAX", symbol: "WAX", decimals: 4, address: "" }},
  { id: 43, name: "Kadena", symbol: "KDA", type: "non-evm", rpcUrl: "https://api.kda.network", explorer: "https://explorer.kda.network", chainId: 43, logo: "🔗", color: "#2F3142", isTestnet: false, features: ["defi", "nft", "enterprise"], nativeToken: { name: "Kadena", symbol: "KDA", decimals: 12, address: "" }},
  { id: 44, name: "Conflux", symbol: "CFX", type: "non-evm", rpcUrl: "https://rpc.confluxnetwork.org", explorer: "https://confluxscan.io", chainId: 44, logo: "🔵", color: "#00A9E0", isTestnet: false, features: ["defi", "nft", "scaling"], nativeToken: { name: "Conflux", symbol: "CFX", decimals: 18, address: "" }},
  { id: 45, name: "Conflux Testnet", symbol: "CFX", type: "non-evm", rpcUrl: "https://testnet-rpc.confluxnetwork.org", explorer: "https://testnet.confluxscan.io", chainId: 45, logo: "🔵", color: "#00A9E0", isTestnet: true, features: ["testnet"], nativeToken: { name: "Conflux", symbol: "CFX", decimals: 18, address: "" }},
  { id: 46, name: "IoTeX", symbol: "IOTX", type: "non-evm", rpcUrl: "https://rpc.iotex.io", explorer: "https://iotexscan.io", chainId: 46, logo: "📡", color: "#00D4CB", isTestnet: false, features: ["defi", "nft", "iot"], nativeToken: { name: "IoTeX", symbol: "IOTX", decimals: 18, address: "" }},
  { id: 47, name: "Canto", symbol: "CANTO", type: "non-evm", rpcUrl: "https://canto.gravitychain.io", explorer: "https://evm.explorer.canto.io", chainId: 47, logo: "🎵", color: "#00C6FF", isTestnet: false, features: ["defi", "nft", "free-public-goods"], nativeToken: { name: "Canto", symbol: "CANTO", decimals: 18, address: "" }},
  { id: 48, name: "Shimmer", symbol: "SMR", type: "non-evm", rpcUrl: "https://json-rpc.shimmer.network", explorer: "https://explorer.shimmer.network", chainId: 48, logo: "✨", color: "#00EAD3", isTestnet: false, features: ["defi", "nft", "iotaledger"], nativeToken: { name: "Shimmer", symbol: "SMR", decimals: 6, address: "" }},
  { id: 49, name: "Neutron", symbol: "NTRN", type: "non-evm", rpcUrl: "https://rpc-kralum.neutron.org", explorer: "https://mintscan.io/neutron", chainId: 49, logo: "⚛️", color: "#2E3148", isTestnet: false, features: ["interoperability", "defi", "dao"], nativeToken: { name: "Neutron", symbol: "NTRN", decimals: 6, address: "" }},
  { id: 50, name: "Celestia", symbol: "TIA", type: "non-evm", rpcUrl: "https://rpc.celestia.org", explorer: "https://explorer.celestia.org", chainId: 50, logo: "🌙", color: "#7B2BF9", isTestnet: false, features: ["data-availability", "modular"], nativeToken: { name: "Celestia", symbol: "TIA", decimals: 6, address: "" }},
];

export default function MultiChainPage() {
  const [searchQuery, setSearchQuery] = useState("");
  const [filterType, setFilterType] = useState<"all" | "evm" | "non-evm">("all");
  const [filterTestnet, setFilterTestnet] = useState<"all" | "mainnet" | "testnet">("all");
  const [viewMode, setViewMode] = useState<"grid" | "list">("grid");
  const [selectedChains, setSelectedChains] = useState<number[]>([]);
  const [isConnecting, setIsConnecting] = useState(false);

  // Filter chains
  const filteredChains = BLOCKCHAINS.filter(chain => {
    const matchesSearch = chain.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      chain.symbol.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesType = filterType === "all" || chain.type === filterType;
    const matchesTestnet = filterTestnet === "all" || 
      (filterTestnet === "testnet" && chain.isTestnet) ||
      (filterTestnet === "mainnet" && !chain.isTestnet);
    return matchesSearch && matchesType && matchesTestnet;
  });

  // Stats
  const evmCount = BLOCKCHAINS.filter(c => c.type === "evm" && !c.isTestnet).length;
  const nonEvmCount = BLOCKCHAINS.filter(c => c.type === "non-evm" && !c.isTestnet).length;
  const testnetCount = BLOCKCHAINS.filter(c => c.isTestnet).length;

  // Toggle chain selection
  const toggleChain = useCallback((chainId: number) => {
    setSelectedChains(prev => 
      prev.includes(chainId) 
        ? prev.filter(id => id !== chainId)
        : [...prev, chainId]
    );
  }, []);

  // Connect all selected chains
  const connectAll = useCallback(async () => {
    setIsConnecting(true);
    // Simulate connection
    await new Promise(resolve => setTimeout(resolve, 2000));
    setIsConnecting(false);
  }, []);

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-900 via-[#1a1a2e] to-black text-white p-4 md:p-8">
      <div className="max-w-7xl mx-auto">
        {/* Header */}
        <header className="flex flex-col md:flex-row justify-between items-center mb-8 gap-4">
          <div className="flex items-center gap-3">
            <div className="w-12 h-12 bg-gradient-to-br from-blue-500 to-blue-600 rounded-xl flex items-center justify-center">
              <Layers className="w-7 h-7" />
            </div>
            <div>
              <h1 className="text-2xl font-bold">TigerWallet</h1>
              <p className="text-gray-400 text-sm">Multi-Chain Support</p>
            </div>
          </div>

          <div className="flex items-center gap-4">
            <div className="bg-gray-800/50 border border-gray-700 rounded-lg px-4 py-2">
              <p className="text-gray-400 text-xs">Total Chains</p>
              <p className="font-bold">{BLOCKCHAINS.length}</p>
            </div>
            <div className="bg-gray-800/50 border border-gray-700 rounded-lg px-4 py-2">
              <p className="text-gray-400 text-xs">EVM Chains</p>
              <p className="font-bold">{evmCount}</p>
            </div>
            <div className="bg-gray-800/50 border border-gray-700 rounded-lg px-4 py-2">
              <p className="text-gray-400 text-xs">Non-EVM</p>
              <p className="font-bold">{nonEvmCount}</p>
            </div>
          </div>
        </header>

        {/* Filters */}
        <div className="flex flex-wrap gap-4 mb-6">
          <div className="flex-1 min-w-[200px]">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
              <input
                type="text"
                placeholder="Search chains..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="w-full bg-gray-800 border border-gray-700 rounded-lg pl-10 pr-4 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
          </div>

          <select
            value={filterType}
            onChange={(e) => setFilterType(e.target.value as any)}
            className="bg-gray-800 border border-gray-700 rounded-lg px-4 py-2"
          >
            <option value="all">All Types</option>
            <option value="evm">EVM Only</option>
            <option value="non-evm">Non-EVM Only</option>
          </select>

          <select
            value={filterTestnet}
            onChange={(e) => setFilterTestnet(e.target.value as any)}
            className="bg-gray-800 border border-gray-700 rounded-lg px-4 py-2"
          >
            <option value="all">Mainnet & Testnet</option>
            <option value="mainnet">Mainnet Only</option>
            <option value="testnet">Testnet Only</option>
          </select>

          <div className="flex gap-2">
            <button
              onClick={() => setViewMode("grid")}
              className={`p-2 rounded-lg ${viewMode === "grid" ? "bg-blue-500" : "bg-gray-800"}`}
            >
              <Grid className="w-5 h-5" />
            </button>
            <button
              onClick={() => setViewMode("list")}
              className={`p-2 rounded-lg ${viewMode === "list" ? "bg-blue-500" : "bg-gray-800"}`}
            >
              <List className="w-5 h-5" />
            </button>
          </div>
        </div>

        {/* Selected Actions */}
        {selectedChains.length > 0 && (
          <div className="mb-6 p-4 bg-blue-500/20 border border-blue-500/50 rounded-xl flex items-center justify-between">
            <p>{selectedChains.length} chains selected</p>
            <button
              onClick={connectAll}
              disabled={isConnecting}
              className="bg-blue-500 hover:bg-blue-600 px-6 py-2 rounded-lg font-medium flex items-center gap-2"
            >
              {isConnecting ? <RefreshCw className="w-4 h-4 animate-spin" /> : <Zap className="w-4 h-4" />}
              Connect All
            </button>
          </div>
        )}

        {/* Chain Grid/List */}
        {viewMode === "grid" ? (
          <div className="grid md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
            {filteredChains.map(chain => (
              <div 
                key={chain.id}
                onClick={() => toggleChain(chain.id)}
                className={`bg-gray-800/50 border rounded-xl p-4 cursor-pointer transition-all hover:scale-102 ${
                  selectedChains.includes(chain.id) 
                    ? "border-blue-500 bg-blue-500/10" 
                    : "border-gray-700 hover:border-gray-600"
                }`}
              >
                <div className="flex items-center justify-between mb-3">
                  <div className="flex items-center gap-3">
                    <div 
                      className="w-10 h-10 rounded-full flex items-center justify-center text-xl"
                      style={{ backgroundColor: `${chain.color}20` }}
                    >
                      {chain.logo}
                    </div>
                    <div>
                      <p className="font-bold">{chain.name}</p>
                      <p className="text-gray-400 text-sm">{chain.symbol}</p>
                    </div>
                  </div>
                  {selectedChains.includes(chain.id) && (
                    <CheckCircle className="w-5 h-5 text-blue-500" />
                  )}
                </div>

                <div className="flex items-center justify-between">
                  <div className="flex gap-1 flex-wrap">
                    {chain.isTestnet ? (
                      <span className="px-2 py-0.5 bg-yellow-500/20 text-yellow-400 text-xs rounded">Testnet</span>
                    ) : (
                      <span className="px-2 py-0.5 bg-green-500/20 text-green-400 text-xs rounded">Mainnet</span>
                    )}
                    <span className="px-2 py-0.5 bg-gray-700 text-gray-400 text-xs rounded capitalize">
                      {chain.type}
                    </span>
                  </div>
                  <span className="text-gray-400 text-xs">ID: {chain.id}</span>
                </div>

                <div className="mt-3 flex gap-1 flex-wrap">
                  {chain.features.slice(0, 3).map(feature => (
                    <span key={feature} className="px-2 py-0.5 bg-gray-700/50 text-gray-400 text-xs rounded">
                      {feature}
                    </span>
                  ))}
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="bg-gray-800/50 border border-gray-700 rounded-xl overflow-hidden">
            <table className="w-full">
              <thead className="bg-gray-900/50">
                <tr>
                  <th className="px-4 py-3 text-left text-gray-400">Chain</th>
                  <th className="px-4 py-3 text-left text-gray-400">Type</th>
                  <th className="px-4 py-3 text-left text-gray-400">Chain ID</th>
                  <th className="px-4 py-3 text-left text-gray-400">Status</th>
                  <th className="px-4 py-3 text-left text-gray-400">Features</th>
                  <th className="px-4 py-3 text-right text-gray-400">Action</th>
                </tr>
              </thead>
              <tbody>
                {filteredChains.map(chain => (
                  <tr 
                    key={chain.id} 
                    className="border-t border-gray-700 hover:bg-gray-800/50 cursor-pointer"
                    onClick={() => toggleChain(chain.id)}
                  >
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-3">
                        <div 
                          className="w-8 h-8 rounded-full flex items-center justify-center text-lg"
                          style={{ backgroundColor: `${chain.color}20` }}
                        >
                          {chain.logo}
                        </div>
                        <div>
                          <p className="font-bold">{chain.name}</p>
                          <p className="text-gray-400 text-sm">{chain.symbol}</p>
                        </div>
                      </div>
                    </td>
                    <td className="px-4 py-3 capitalize">{chain.type}</td>
                    <td className="px-4 py-3 font-mono">{chain.chainId}</td>
                    <td className="px-4 py-3">
                      {chain.isTestnet ? (
                        <span className="px-2 py-0.5 bg-yellow-500/20 text-yellow-400 text-xs rounded">Testnet</span>
                      ) : (
                        <span className="px-2 py-0.5 bg-green-500/20 text-green-400 text-xs rounded">Mainnet</span>
                      )}
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex gap-1 flex-wrap">
                        {chain.features.slice(0, 2).map(f => (
                          <span key={f} className="px-2 py-0.5 bg-gray-700 text-gray-400 text-xs rounded">
                            {f}
                          </span>
                        ))}
                      </div>
                    </td>
                    <td className="px-4 py-3 text-right">
                      {selectedChains.includes(chain.id) ? (
                        <CheckCircle className="w-5 h-5 text-blue-500 inline" />
                      ) : (
                        <Plus className="w-5 h-5 text-gray-400 inline" />
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {/* Empty State */}
        {filteredChains.length === 0 && (
          <div className="text-center py-12">
            <Layers className="w-12 h-12 mx-auto mb-4 text-gray-500" />
            <p className="text-gray-400">No chains found matching your filters</p>
          </div>
        )}

        {/* Quick Stats */}
        <div className="mt-8 grid md:grid-cols-4 gap-4">
          <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-4">
            <p className="text-gray-400 text-sm mb-1">Top EVM</p>
            <p className="font-bold text-lg">Ethereum, BSC, Polygon</p>
          </div>
          <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-4">
            <p className="text-gray-400 text-sm mb-1">Top L2</p>
            <p className="font-bold text-lg">Arbitrum, Optimism, Base</p>
          </div>
          <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-4">
            <p className="text-gray-400 text-sm mb-1">Top Non-EVM</p>
            <p className="font-bold text-lg">Solana, Aptos, Sui</p>
          </div>
          <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-4">
            <p className="text-gray-400 text-sm mb-1">Bitcoin/L2</p>
            <p className="font-bold text-lg">Bitcoin, Stacks, Dogecoin</p>
          </div>
        </div>
      </div>
    </div>
  );
}
