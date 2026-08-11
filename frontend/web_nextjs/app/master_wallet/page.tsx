'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { generateMnemonic, validateMnemonic } from '@scure/bip39';
import { wordlist } from '@scure/bip39/wordlists/english.js';
import { ethers } from 'ethers';
import { useTheme } from '../components/ThemeProvider';

// ============================================================================
// Types
// ============================================================================

// Theme Context
const MasterThemeContext = React.createContext({
  isDarkMode: true,
  toggleTheme: () => {}
});

export function MasterWalletThemeProvider({ children }: { children: React.ReactNode }) {
  const [isDarkMode, setIsDarkMode] = useState(true);

  useEffect(() => {
    const stored = localStorage.getItem('master_wallet_theme');
    if (stored) setIsDarkMode(stored === 'dark');
  }, []);

  const toggleTheme = () => {
    const newMode = !isDarkMode;
    setIsDarkMode(newMode);
    localStorage.setItem('master_wallet_theme', newMode ? 'dark' : 'light');
  };

  return (
    <MasterThemeContext.Provider value={{ isDarkMode, toggleTheme }}>
      {children}
    </MasterThemeContext.Provider>
  );
}

export { MasterThemeContext };

interface MasterWallet {
  address: string;
  seedPhrase: string;
  backupCode: string;
  balance: number;
  totalRevenue: number;
}

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
  isEVM: boolean;
  isNonEVM: boolean;
}

interface Token {
  id: number;
  chainId: number;
  chainName: string;
  address: string;
  symbol: string;
  name: string;
  decimals: number;
  isActive: boolean;
  isPopular: boolean;
  isStablecoin: boolean;
  priceUsd: number;
}

interface FeeConfig {
  swapFee: number;
  withdrawalFee: number;
  transactionFee: number;
  depositFee: number;
  bridgeFee: number;
  airdropFee: number;
  campaignFee: number;
}

interface UserWallet {
  id: string;
  address: string;
  seedPhrase: string;
  createdAt: number;
  lastActive: number;
  totalVolume: number;
  status: 'active' | 'suspended';
}

interface Transaction {
  id: string;
  type: 'swap' | 'withdraw' | 'transfer' | 'bridge' | 'airdrop' | 'campaign' | 'fee';
  amount: number;
  token: string;
  chain: string;
  fromUser: string;
  toAddress: string;
  fee: number;
  masterWalletRevenue: number;
  status: 'pending' | 'completed' | 'failed';
  timestamp: number;
  txHash: string;
}

interface SystemStats {
  totalUsers: number;
  activeUsers: number;
  totalTransactions: number;
  totalVolume: number;
  dailyRevenue: number;
  monthlyRevenue: number;
}

// ============================================================================
// Constants
// ============================================================================

const BLOCKCHAINS: Blockchain[] = [
  { id: 1, name: 'Ethereum', symbol: 'ETH', chainId: 1, rpcUrl: 'https://eth.llamarpc.com', explorerUrl: 'https://etherscan.io', decimals: 18, isActive: true, gasLimit: 21000, confirmations: 12, isEVM: true, isNonEVM: false },
  { id: 2, name: 'Polygon', symbol: 'MATIC', chainId: 137, rpcUrl: 'https://polygon.llamarpc.com', explorerUrl: 'https://polygonscan.com', decimals: 18, isActive: true, gasLimit: 21000, confirmations: 64, isEVM: true, isNonEVM: false },
  { id: 3, name: 'Arbitrum', symbol: 'ARB', chainId: 42161, rpcUrl: 'https://arb1.arbitrum.io/rpc', explorerUrl: 'https://arbiscan.io', decimals: 18, isActive: true, gasLimit: 21000, confirmations: 12, isEVM: true, isNonEVM: false },
  { id: 4, name: 'Optimism', symbol: 'OP', chainId: 10, rpcUrl: 'https://mainnet.optimism.io', explorerUrl: 'https://optimistic.etherscan.io', decimals: 18, isActive: true, gasLimit: 21000, confirmations: 12, isEVM: true, isNonEVM: false },
  { id: 5, name: 'Base', symbol: 'ETH', chainId: 8453, rpcUrl: 'https://mainnet.base.org', explorerUrl: 'https://basescan.org', decimals: 18, isActive: true, gasLimit: 21000, confirmations: 12, isEVM: true, isNonEVM: false },
  { id: 6, name: 'Avalanche', symbol: 'AVAX', chainId: 43114, rpcUrl: 'https://api.avax.network/ext/bc/C/rpc', explorerUrl: 'https://snowtrace.io', decimals: 18, isActive: true, gasLimit: 21000, confirmations: 12, isEVM: true, isNonEVM: false },
  { id: 7, name: 'BNB Chain', symbol: 'BNB', chainId: 56, rpcUrl: 'https://bsc-dataseed.binance.org', explorerUrl: 'https://bscscan.com', decimals: 18, isActive: true, gasLimit: 21000, confirmations: 19, isEVM: true, isNonEVM: false },
  { id: 8, name: 'Solana', symbol: 'SOL', chainId: 101, rpcUrl: 'https://api.mainnet-beta.solana.com', explorerUrl: 'https://solscan.io', decimals: 9, isActive: true, gasLimit: 0, confirmations: 32, isEVM: false, isNonEVM: true },
  { id: 9, name: 'Tron', symbol: 'TRX', chainId: 728126428, rpcUrl: 'https://api.trongrid.io', explorerUrl: 'https://tronscan.org', decimals: 6, isActive: true, gasLimit: 0, confirmations: 19, isEVM: false, isNonEVM: true },
  { id: 10, name: 'Bitcoin', symbol: 'BTC', chainId: 0, rpcUrl: 'https://blockstream.info/api', explorerUrl: 'https://blockstream.info', decimals: 8, isActive: true, gasLimit: 0, confirmations: 6, isEVM: false, isNonEVM: true },
  { id: 11, name: 'Pi Network', symbol: 'PI', chainId: 314159, rpcUrl: 'https://minepi.com/rpc', explorerUrl: 'https://blockexplorer.pi', decimals: 18, isActive: true, gasLimit: 21000, confirmations: 10, isEVM: false, isNonEVM: true },
  { id: 12, name: 'Toncoin', symbol: 'TON', chainId: -239, rpcUrl: 'https://toncenter.com/api/v2', explorerUrl: 'https://tonscan.org', decimals: 9, isActive: true, gasLimit: 0, confirmations: 1, isEVM: false, isNonEVM: true },
  { id: 13, name: 'Aptos', symbol: 'APT', chainId: 637, rpcUrl: 'https://aptos-mainnet.nodereal.io/v1', explorerUrl: 'https://aptoscan.com', decimals: 8, isActive: true, gasLimit: 0, confirmations: 1, isEVM: false, isNonEVM: true },
  { id: 14, name: 'Cosmos', symbol: 'ATOM', chainId: 118, rpcUrl: 'https://cosmos-rpc.polkachu.com', explorerUrl: 'https://mintscan.io/cosmos', decimals: 6, isActive: true, gasLimit: 0, confirmations: 16, isEVM: false, isNonEVM: true },
  { id: 15, name: 'Cardano', symbol: 'ADA', chainId: 3009, rpcUrl: 'https://cardano-mainnet.blockfrost.io', explorerUrl: 'https://cardanoscan.io', decimals: 6, isActive: true, gasLimit: 0, confirmations: 30, isEVM: false, isNonEVM: true },
];

const TOKENS: Token[] = [
  { id: 1, chainId: 1, chainName: 'Ethereum', address: '0x0000000000000000000000000000000000000000', symbol: 'ETH', name: 'Ethereum', decimals: 18, isActive: true, isPopular: true, isStablecoin: false, priceUsd: 3500 },
  { id: 2, chainId: 1, chainName: 'Ethereum', address: '0xdAC17F958D2ee523a2206206994597C13D831ec7', symbol: 'USDT', name: 'Tether USD', decimals: 6, isActive: true, isPopular: true, isStablecoin: true, priceUsd: 1 },
  { id: 3, chainId: 1, chainName: 'Ethereum', address: '0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48', symbol: 'USDC', name: 'USD Coin', decimals: 6, isActive: true, isPopular: true, isStablecoin: true, priceUsd: 1 },
  { id: 4, chainId: 1, chainName: 'Ethereum', address: '0x2260FAC5E5542a773Aa44fbCfEDf7C193bc2C599', symbol: 'WBTC', name: 'Wrapped Bitcoin', decimals: 8, isActive: true, isPopular: true, isStablecoin: false, priceUsd: 65000 },
  { id: 5, chainId: 1, chainName: 'Ethereum', address: '0x6B175474E89094c44Da98b954EedeAC495271d0F', symbol: 'DAI', name: 'Dai Stablecoin', decimals: 18, isActive: true, isPopular: true, isStablecoin: true, priceUsd: 1 },
  { id: 6, chainId: 7, chainName: 'BNB Chain', address: '0x0000000000000000000000000000000000000000', symbol: 'BNB', name: 'BNB', decimals: 18, isActive: true, isPopular: true, isStablecoin: false, priceUsd: 600 },
  { id: 7, chainId: 2, chainName: 'Polygon', address: '0x0000000000000000000000000000000000000000', symbol: 'MATIC', name: 'Polygon', decimals: 18, isActive: true, isPopular: true, isStablecoin: false, priceUsd: 0.8 },
  { id: 8, chainId: 8, chainName: 'Solana', address: '0x0000000000000000000000000000000000000000', symbol: 'SOL', name: 'Solana', decimals: 9, isActive: true, isPopular: true, isStablecoin: false, priceUsd: 150 },
  { id: 9, chainId: 9, chainName: 'Tron', address: '0x0000000000000000000000000000000000000000', symbol: 'TRX', name: 'Tron', decimals: 6, isActive: true, isPopular: true, isStablecoin: false, priceUsd: 0.12 },
  { id: 10, chainId: 10, chainName: 'Bitcoin', address: '0x0000000000000000000000000000000000000000', symbol: 'BTC', name: 'Bitcoin', decimals: 8, isActive: true, isPopular: true, isStablecoin: false, priceUsd: 65000 },
  { id: 11, chainId: 1, chainName: 'Ethereum', address: '0x514910771AF9ca656af840dff83e8264ecF986CA', symbol: 'LINK', name: 'Chainlink', decimals: 18, isActive: true, isPopular: true, isStablecoin: false, priceUsd: 18 },
  { id: 12, chainId: 1, chainName: 'Ethereum', address: '0x7Fc66500c84A76ad7e9c93437bFc5Ac33E2DDaE9', symbol: 'AAVE', name: 'Aave', decimals: 18, isActive: true, isPopular: true, isStablecoin: false, priceUsd: 280 },
];

// Derive a real EVM address from a BIP-39 mnemonic using BIP-32/BIP-44 HD
// derivation (m/44'/60'/0'/0/0) via ethers — the same path the canonical
// go/wallet_api uses (verified against the BIP-44 test vector). All EVM
// chains share one address because they use the same derivation path. Non-EVM
// chains (Solana/Bitcoin/Tron/...) need chain-specific derivation libraries
// and are left empty here; they are derived on demand by the backend.
const generateAddressFromSeed = (mnemonic: string, chain: Blockchain): string => {
  if (!chain.isEVM) {
    return '';
  }
  try {
    const node = ethers.utils.HDNode.fromMnemonic(mnemonic).derivePath("m/44'/60'/0'/0/0");
    return node.address;
  } catch {
    return '';
  }
};

// Generate backup code
const generateBackupCode = (): string => {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789';
  // Use the Web Crypto API for an unpredictable backup code (Math.random()
  // is not cryptographically secure).
  const rand = new Uint8Array(16);
  crypto.getRandomValues(rand);
  let code = '';
  for (let i = 0; i < 16; i++) {
    if (i > 0 && i % 4 === 0) code += '-';
    code += chars.charAt(rand[i] % chars.length);
  }
  return code;
};

// ============================================================================
// Secure storage helpers
// ----------------------------------------------------------------------------
// The master wallet mnemonic must never be persisted in plaintext. We encrypt
// it with AES-GCM using a key derived from a user password via PBKDF2
// (600k iterations, SHA-256), then store only the ciphertext + salt + iv in
// localStorage. The mnemonic is held in memory (React state) for the active
// session and re-decrypted only when the user supplies the password.
//
// NOTE: Production should NOT rely on this. Store the mnemonic in a hardware
// wallet / secure enclave / server-side HSM-backed KMS. This local encryption
// only reduces (does not eliminate) the risk of leaking the seed via a
// compromised browser extension or XSS.
// ============================================================================

const WALLET_STORAGE_KEY = 'masterWallet';
const PBKDF2_ITERATIONS = 600_000;
const SALT_BYTES = 16;
const IV_BYTES = 12;

interface EncryptedBlob {
  v: 1;
  salt: string; // base64
  iv: string;   // base64
  ciphertext: string; // base64
}

const toBase64 = (bytes: Uint8Array): string =>
  btoa(String.fromCharCode(...bytes));

const fromBase64 = (b64: string): Uint8Array =>
  Uint8Array.from(atob(b64), (c) => c.charCodeAt(0));

async function deriveKey(password: string, salt: Uint8Array): Promise<CryptoKey> {
  const baseKey = await crypto.subtle.importKey(
    'raw',
    new TextEncoder().encode(password),
    { name: 'PBKDF2' },
    false,
    ['deriveKey']
  );
  return crypto.subtle.deriveKey(
    { name: 'PBKDF2', salt: salt as BufferSource, iterations: PBKDF2_ITERATIONS, hash: 'SHA-256' },
    baseKey,
    { name: 'AES-GCM', length: 256 },
    false,
    ['encrypt', 'decrypt']
  );
}

// Encrypt the wallet payload and persist it. The plaintext mnemonic is NOT
// stored; only the encrypted blob is written to localStorage.
async function encryptAndStoreWallet(wallet: MasterWallet, password: string): Promise<void> {
  const salt = crypto.getRandomValues(new Uint8Array(SALT_BYTES));
  const iv = crypto.getRandomValues(new Uint8Array(IV_BYTES));
  const key = await deriveKey(password, salt);

  const plaintext = new TextEncoder().encode(JSON.stringify(wallet));
  const ciphertext = new Uint8Array(
    await crypto.subtle.encrypt({ name: 'AES-GCM', iv }, key, plaintext)
  );

  const blob: EncryptedBlob = {
    v: 1,
    salt: toBase64(salt),
    iv: toBase64(iv),
    ciphertext: toBase64(ciphertext),
  };
  localStorage.setItem(WALLET_STORAGE_KEY, JSON.stringify(blob));
}

async function decryptStoredWallet(password: string): Promise<MasterWallet | null> {
  const raw = localStorage.getItem(WALLET_STORAGE_KEY);
  if (!raw) return null;

  let blob: EncryptedBlob;
  try {
    blob = JSON.parse(raw);
  } catch {
    return null;
  }
  if (!blob || blob.v !== 1 || !blob.salt || !blob.iv || !blob.ciphertext) return null;

  const salt = fromBase64(blob.salt);
  const iv = fromBase64(blob.iv);
  const ciphertext = fromBase64(blob.ciphertext);
  const key = await deriveKey(password, salt);

  try {
    const plaintext = await crypto.subtle.decrypt({ name: 'AES-GCM', iv: iv as BufferSource }, key, ciphertext as BufferSource);
    return JSON.parse(new TextDecoder().decode(plaintext)) as MasterWallet;
  } catch {
    // Wrong password or tampered ciphertext.
    return null;
  }
}

function hasStoredWallet(): boolean {
  const raw = localStorage.getItem(WALLET_STORAGE_KEY);
  if (!raw) return false;
  try {
    return JSON.parse(raw)?.v === 1;
  } catch {
    return false;
  }
}

// Generate a valid 24-word BIP-39 mnemonic (256 bits of entropy + 8-bit
// SHA-256 checksum). This replaces the previous invalid approach of picking
// 24 words from only the first 24 BIP-39 words.
const generateSeedPhrase = (): string => generateMnemonic(wordlist, 256);

// ============================================================================
// Component
// ============================================================================

export default function MasterWalletPage() {
  const router = useRouter();
  const { isDark } = useTheme();

  // Master Wallet State
  const [masterWallet, setMasterWallet] = useState<MasterWallet | null>(null);
  const [showSeedPhrase, setShowSeedPhrase] = useState(false);
  const [showBackupCode, setShowBackupCode] = useState(false);
  const [isCreating, setIsCreating] = useState(false);
  const [hasExistingWallet, setHasExistingWallet] = useState(false);
  // Password used to encrypt/decrypt the wallet blob in localStorage. Required
  // to create or unlock a wallet; the mnemonic is only held in memory after
  // a successful unlock.
  const [walletPassword, setWalletPassword] = useState('');
  // Seed phrase typed into the import textarea (submitted explicitly rather
  // than auto-importing once 24 words are detected).
  const [importSeedPhrase, setImportSeedPhrase] = useState('');

  // Tab State
  const [activeTab, setActiveTab] = useState<'dashboard' | 'fees' | 'blockchains' | 'tokens' | 'users' | 'transactions' | 'settings'>('dashboard');

  // Fee Configuration
  const [fees, setFees] = useState<FeeConfig>({
    swapFee: 0.3,
    withdrawalFee: 0.1,
    transactionFee: 0.05,
    depositFee: 0,
    bridgeFee: 0.5,
    airdropFee: 0.2,
    campaignFee: 0.5,
  });

  // Blockchain State
  const [blockchains, setBlockchains] = useState<Blockchain[]>(BLOCKCHAINS);
  const [showAddBlockchain, setShowAddBlockchain] = useState(false);
  const [newBlockchain, setNewBlockchain] = useState<Partial<Blockchain>>({});

  // Token State
  const [tokens, setTokens] = useState<Token[]>(TOKENS);
  const [showAddToken, setShowAddToken] = useState(false);
  const [newToken, setNewToken] = useState<Partial<Token>>({});

  // User Wallets State
  const [userWallets, setUserWallets] = useState<UserWallet[]>([]);

  // Transactions State
  const [transactions, setTransactions] = useState<Transaction[]>([]);

  // Stats
  const [stats, setStats] = useState<SystemStats>({
    totalUsers: 0,
    activeUsers: 0,
    totalTransactions: 0,
    totalVolume: 0,
    dailyRevenue: 0,
    monthlyRevenue: 0,
  });

  // Check for existing master wallet. We only detect whether an encrypted
  // wallet blob exists; it is not decrypted here. The user must supply their
  // password to unlock it (see unlockWallet).
  // Load admin stats + user wallets/transactions from the backend. No hardcoded
  // sample data; the dashboard reflects the real wallet_api / monitoring state.
  useEffect(() => {
    if (hasStoredWallet()) {
      setHasExistingWallet(true);
    }
    (async () => {
      const apiBase = ''; // same-origin -> Next.js proxy routes -> wallet_api
      try {
        const r = await fetch(`${apiBase}/api/v1/admin/stats`);
        if (r.ok) {
          const d = await r.json();
          if (d) {
            setStats({
              totalUsers: d.totalUsers ?? 0,
              activeUsers: d.activeUsers ?? 0,
              totalTransactions: d.totalTransactions ?? 0,
              totalVolume: d.totalVolume ?? 0,
              dailyRevenue: d.dailyRevenue ?? 0,
              monthlyRevenue: d.monthlyRevenue ?? 0,
            });
          }
        }
      } catch { /* backend unavailable; stats stay at zero */ }
      try {
        const wr = await fetch(`${apiBase}/api/v1/admin/wallets`);
        if (wr.ok) setUserWallets(await wr.json());
      } catch { /* leave empty */ }
      try {
        const tr = await fetch(`${apiBase}/api/v1/admin/transactions`);
        if (tr.ok) setTransactions(await tr.json());
      } catch { /* leave empty */ }
    })();
  }, []);

  // Create Master Wallet
  const createMasterWallet = useCallback(async () => {
    if (!walletPassword || walletPassword.length < 8) {
      alert('Please set an encryption password (min 8 characters) to protect your wallet.');
      return;
    }
    setIsCreating(true);

    try {
      // Generate a valid 24-word BIP-39 mnemonic (256 bits of entropy + checksum).
      const seedPhrase = generateSeedPhrase();

      // Generate addresses for all blockchains
      const addresses: { [key: string]: string } = {};
      BLOCKCHAINS.forEach((chain) => {
        addresses[chain.name] = generateAddressFromSeed(seedPhrase, chain);
      });

      const backupCode = generateBackupCode();

      const newWallet: MasterWallet = {
        address: addresses['Ethereum'],
        seedPhrase,
        backupCode,
        balance: 0,
        totalRevenue: 457500,
      };

      // Persist only an encrypted blob; the plaintext mnemonic lives in memory.
      await encryptAndStoreWallet(newWallet, walletPassword);
      setMasterWallet(newWallet);
      setHasExistingWallet(true);
    } catch (err) {
      console.error('Failed to create wallet', err);
      alert('Failed to create wallet. Please try again.');
    } finally {
      setIsCreating(false);
    }
  }, [walletPassword]);

  // Unlock an existing encrypted wallet with the user's password.
  const unlockWallet = useCallback(async () => {
    if (!walletPassword) {
      alert('Please enter your wallet password.');
      return;
    }
    const wallet = await decryptStoredWallet(walletPassword);
    if (!wallet) {
      alert('Incorrect password or corrupted wallet data.');
      return;
    }
    setMasterWallet(wallet);
  }, [walletPassword]);

  // Import Master Wallet
  const importMasterWallet = useCallback(async (seedPhrase: string) => {
    if (!walletPassword || walletPassword.length < 8) {
      alert('Please set an encryption password (min 8 characters) to protect your wallet.');
      return;
    }

    const trimmed = seedPhrase.trim().replace(/\s+/g, ' ');
    // Validate against the BIP-39 wordlist + checksum (replaces the previous
    // word-count-only check, which accepted invalid mnemonics).
    if (!validateMnemonic(trimmed, wordlist)) {
      alert('Please enter a valid 24-word BIP-39 seed phrase (checksum must match).');
      return;
    }

    const addresses: { [key: string]: string } = {};
    BLOCKCHAINS.forEach((chain) => {
      addresses[chain.name] = generateAddressFromSeed(trimmed, chain);
    });

    const backupCode = generateBackupCode();

    const wallet: MasterWallet = {
      address: addresses['Ethereum'],
      seedPhrase: trimmed,
      backupCode,
      balance: 0,
      totalRevenue: 0,
    };

    await encryptAndStoreWallet(wallet, walletPassword);
    setMasterWallet(wallet);
    setHasExistingWallet(true);
  }, [walletPassword]);

  // Add Blockchain
  const handleAddBlockchain = useCallback(() => {
    if (!newBlockchain.name || !newBlockchain.symbol) return;

    const chain: Blockchain = {
      id: blockchains.length + 1,
      name: newBlockchain.name,
      symbol: newBlockchain.symbol,
      chainId: newBlockchain.chainId || 0,
      rpcUrl: newBlockchain.rpcUrl || '',
      explorerUrl: newBlockchain.explorerUrl || '',
      decimals: newBlockchain.decimals || 18,
      isActive: true,
      gasLimit: newBlockchain.gasLimit || 21000,
      confirmations: newBlockchain.confirmations || 12,
      isEVM: newBlockchain.isEVM || true,
      isNonEVM: newBlockchain.isNonEVM || false,
    };

    setBlockchains(prev => [...prev, chain]);
    setNewBlockchain({});
    setShowAddBlockchain(false);
  }, [newBlockchain, blockchains]);

  // Add Token
  const handleAddToken = useCallback(() => {
    if (!newToken.name || !newToken.symbol) return;

    const token: Token = {
      id: tokens.length + 1,
      chainId: newToken.chainId || 1,
      chainName: newToken.chainName || 'Ethereum',
      address: newToken.address || '0x0000000000000000000000000000000000000000',
      symbol: newToken.symbol,
      name: newToken.name,
      decimals: newToken.decimals || 18,
      isActive: true,
      isPopular: newToken.isPopular || false,
      isStablecoin: newToken.isStablecoin || false,
      priceUsd: newToken.priceUsd || 0,
    };

    setTokens(prev => [...prev, token]);
    setNewToken({});
    setShowAddToken(false);
  }, [newToken, tokens]);

  // Update Fee
  const updateFee = useCallback((key: keyof FeeConfig, value: number) => {
    setFees(prev => ({ ...prev, [key]: value }));
  }, []);

  // Render Import/Create Screen (no wallet stored yet)
  if (!hasExistingWallet) {
    return (
      <div className={`min-h-screen flex items-center justify-center p-4 ${isDark ? 'bg-gradient-to-br from-purple-900 via-blue-900 to-purple-900 text-white' : 'bg-gray-50 text-gray-900'}`}>
        <div className={`rounded-2xl p-8 max-w-md w-full ${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'}`}>
          <div className="text-center mb-8">
            <div className="text-6xl mb-4">🐯</div>
            <h1 className="text-2xl font-bold">Master Wallet Setup</h1>
            <p className={`mt-2 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Create or import your master wallet</p>
          </div>

          {!masterWallet ? (
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium mb-2">Encryption Password (min 8 chars)</label>
                <input
                  type="password"
                  placeholder="Choose a strong password"
                  className={`w-full p-3 border rounded-lg ${isDark ? 'bg-slate-700 border-gray-700' : 'bg-white border-gray-200'}`}
                  value={walletPassword}
                  onChange={(e) => setWalletPassword(e.target.value)}
                />
                <p className={`text-xs mt-1 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
                  Encrypts your wallet before storing locally. The mnemonic is never saved in plaintext.
                </p>
              </div>

              <button
                onClick={createMasterWallet}
                disabled={isCreating}
                className="w-full py-4 bg-gradient-to-r from-purple-600 to-blue-600 text-white rounded-xl font-semibold hover:opacity-90 transition-opacity"
              >
                {isCreating ? 'Creating Wallet...' : 'Create New Master Wallet'}
              </button>

              <div className="relative">
                <div className="absolute inset-0 flex items-center"><div className={`w-full border-t ${isDark ? 'border-gray-700' : 'border-gray-200'}`}></div></div>
                <div className="relative flex justify-center text-sm"><span className={`px-2 ${isDark ? 'bg-slate-800 text-gray-400' : 'bg-white text-gray-500'}`}>or</span></div>
              </div>

              <div>
                <label className="block text-sm font-medium mb-2">Import with 24-word seed phrase</label>
                <textarea
                  placeholder="Enter your 24-word BIP-39 seed phrase..."
                  className={`w-full p-3 border rounded-lg h-24 resize-none ${isDark ? 'bg-slate-700 border-gray-700' : 'bg-white border-gray-200'}`}
                  value={importSeedPhrase}
                  onChange={(e) => setImportSeedPhrase(e.target.value)}
                />
                <p className={`text-xs mt-1 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Enter 24 words separated by spaces (checksum is validated)</p>
                <button
                  onClick={() => importMasterWallet(importSeedPhrase)}
                  className={`mt-2 w-full py-3 text-white rounded-lg ${isDark ? 'bg-slate-700 hover:bg-slate-600' : 'bg-gray-800 hover:bg-gray-700'}`}
                >
                  Import Wallet
                </button>
              </div>
            </div>
          ) : (
            <div className="text-center">
              <p className="text-green-600 mb-4">Master wallet created successfully!</p>
              <button onClick={() => router.refresh()} className="py-3 px-6 bg-blue-600 text-white rounded-lg">
                Continue to Dashboard
              </button>
            </div>
          )}
        </div>
      </div>
    );
  }

  // Render Unlock Screen (encrypted wallet exists, not yet decrypted)
  if (!masterWallet) {
    return (
      <div className={`min-h-screen flex items-center justify-center p-4 ${isDark ? 'bg-gradient-to-br from-purple-900 via-blue-900 to-purple-900 text-white' : 'bg-gray-50 text-gray-900'}`}>
        <div className={`rounded-2xl p-8 max-w-md w-full ${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'}`}>
          <div className="text-center mb-8">
            <div className="text-6xl mb-4">🐯</div>
            <h1 className="text-2xl font-bold">Unlock Master Wallet</h1>
            <p className={`mt-2 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Enter your password to decrypt your wallet</p>
          </div>
          <div className="space-y-4">
            <input
              type="password"
              placeholder="Wallet password"
              className={`w-full p-3 border rounded-lg ${isDark ? 'bg-slate-700 border-gray-700' : 'bg-white border-gray-200'}`}
              value={walletPassword}
              onChange={(e) => setWalletPassword(e.target.value)}
            />
            <button
              onClick={unlockWallet}
              className="w-full py-4 bg-gradient-to-r from-purple-600 to-blue-600 text-white rounded-xl font-semibold hover:opacity-90 transition-opacity"
            >
              Unlock
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className={`min-h-screen ${isDark ? 'bg-slate-900 text-white' : 'bg-slate-50 text-gray-900'}`}>
      {/* Header */}
      <header className="bg-gradient-to-r from-purple-600 to-blue-600 text-white">
        <div className="max-w-7xl mx-auto px-4 py-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <a href="/" className="text-3xl">🐯</a>
              <div>
                <h1 className="text-2xl font-bold">Master Wallet Admin</h1>
                <p className="text-purple-200 text-sm">Complete control over all user wallets</p>
              </div>
            </div>
            <div className="flex items-center gap-4">
              <div className="text-right">
                <p className="text-sm text-purple-200">Total Revenue</p>
                <p className="text-2xl font-bold">${stats.monthlyRevenue.toLocaleString()}</p>
              </div>
            </div>
          </div>
        </div>
      </header>

      <div className="max-w-7xl mx-auto px-4 py-6">
        {/* Master Wallet Info */}
        <div className={`rounded-xl p-6 mb-6 border ${isDark ? 'bg-slate-800 border-gray-700' : 'bg-white border-gray-200'}`}>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <div className="w-12 h-12 bg-gradient-to-br from-purple-500 to-blue-600 rounded-full flex items-center justify-center text-white text-xl font-bold">
                M
              </div>
              <div>
                <h2 className="font-semibold">Master Wallet</h2>
                <p className={`text-sm font-mono ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>{masterWallet?.address}</p>
              </div>
            </div>
            <div className="flex gap-2">
              <button
                onClick={() => setShowSeedPhrase(!showSeedPhrase)}
                className={`px-4 py-2 rounded-lg text-sm ${isDark ? 'bg-slate-700' : 'bg-gray-100'}`}
              >
                {showSeedPhrase ? 'Hide' : 'Show'} Seed
              </button>
              <button
                onClick={() => setShowBackupCode(!showBackupCode)}
                className={`px-4 py-2 rounded-lg text-sm ${isDark ? 'bg-slate-700' : 'bg-gray-100'}`}
              >
                {showBackupCode ? 'Hide' : 'Show'} Backup Code
              </button>
            </div>
          </div>

          {(showSeedPhrase || showBackupCode) && (
            <div className={`mt-4 p-4 border rounded-lg ${isDark ? 'bg-yellow-900/20 border-yellow-800' : 'bg-yellow-50 border-yellow-200'}`}>
              {showSeedPhrase && (
                <div className="mb-4">
                  <p className={`font-semibold mb-2 ${isDark ? 'text-yellow-200' : 'text-yellow-800'}`}>24-Word Seed Phrase (KEEP SECRET)</p>
                  <p className="font-mono text-sm break-all">{masterWallet?.seedPhrase}</p>
                </div>
              )}
              {showBackupCode && (
                <div>
                  <p className={`font-semibold mb-2 ${isDark ? 'text-yellow-200' : 'text-yellow-800'}`}>Backup Code</p>
                  <p className="font-mono text-lg tracking-wider">{masterWallet?.backupCode}</p>
                </div>
              )}
            </div>
          )}
        </div>

        {/* Stats */}
        <div className="grid grid-cols-6 gap-4 mb-6">
          <div className={`rounded-lg p-4 border ${isDark ? 'bg-slate-800 border-gray-700' : 'bg-white border-gray-200'}`}>
            <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Total Users</p>
            <p className="text-2xl font-bold">{stats.totalUsers.toLocaleString()}</p>
          </div>
          <div className={`rounded-lg p-4 border ${isDark ? 'bg-slate-800 border-gray-700' : 'bg-white border-gray-200'}`}>
            <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Active Users</p>
            <p className="text-2xl font-bold text-green-600">{stats.activeUsers.toLocaleString()}</p>
          </div>
          <div className={`rounded-lg p-4 border ${isDark ? 'bg-slate-800 border-gray-700' : 'bg-white border-gray-200'}`}>
            <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Transactions</p>
            <p className="text-2xl font-bold">{stats.totalTransactions.toLocaleString()}</p>
          </div>
          <div className={`rounded-lg p-4 border ${isDark ? 'bg-slate-800 border-gray-700' : 'bg-white border-gray-200'}`}>
            <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Total Volume</p>
            <p className="text-2xl font-bold">${(stats.totalVolume / 1000000).toFixed(1)}M</p>
          </div>
          <div className={`rounded-lg p-4 border ${isDark ? 'bg-slate-800 border-gray-700' : 'bg-white border-gray-200'}`}>
            <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Daily Revenue</p>
            <p className="text-2xl font-bold text-green-600">${stats.dailyRevenue.toLocaleString()}</p>
          </div>
          <div className={`rounded-lg p-4 border ${isDark ? 'bg-slate-800 border-gray-700' : 'bg-white border-gray-200'}`}>
            <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Blockchains</p>
            <p className="text-2xl font-bold">{blockchains.length}</p>
          </div>
        </div>

        {/* Tabs */}
        <div className={`rounded-xl border ${isDark ? 'bg-slate-800 border-gray-700' : 'bg-white border-gray-200'}`}>
          <div className={`flex border-b overflow-x-auto ${isDark ? 'border-gray-700' : 'border-gray-200'}`}>
            {[
              { key: 'dashboard', label: 'Dashboard' },
              { key: 'fees', label: 'Fee Settings' },
              { key: 'blockchains', label: 'Blockchains' },
              { key: 'tokens', label: 'Tokens' },
              { key: 'users', label: 'User Wallets' },
              { key: 'transactions', label: 'Transactions' },
              { key: 'settings', label: 'Settings' },
            ].map(tab => (
              <button
                key={tab.key}
                onClick={() => setActiveTab(tab.key as any)}
                className={`px-6 py-4 font-medium whitespace-nowrap ${activeTab === tab.key ? `text-blue-600 border-b-2 border-blue-600 ${isDark ? 'bg-blue-900/20' : 'bg-blue-50'}` : isDark ? 'text-gray-400 hover:text-gray-200' : 'text-gray-500 hover:text-gray-700'}`}
              >
                {tab.label}
              </button>
            ))}
          </div>

          <div className="p-6">
            {/* Dashboard Tab */}
            {activeTab === 'dashboard' && (
              <div className="space-y-6">
                <div className="grid grid-cols-2 gap-6">
                  <div className={`p-4 rounded-lg ${isDark ? 'bg-blue-900/20' : 'bg-blue-50'}`}>
                    <h3 className="font-semibold mb-4">Quick Actions</h3>
                    <div className="space-y-2">
                      <button onClick={() => setActiveTab('blockchains')} className={`w-full text-left px-4 py-2 rounded-lg ${isDark ? 'bg-slate-700 hover:bg-slate-600' : 'bg-white hover:bg-slate-50'}`}>
                        + Add Blockchain
                      </button>
                      <button onClick={() => setActiveTab('tokens')} className={`w-full text-left px-4 py-2 rounded-lg ${isDark ? 'bg-slate-700 hover:bg-slate-600' : 'bg-white hover:bg-slate-50'}`}>
                        + Add Token
                      </button>
                      <button onClick={() => setActiveTab('fees')} className={`w-full text-left px-4 py-2 rounded-lg ${isDark ? 'bg-slate-700 hover:bg-slate-600' : 'bg-white hover:bg-slate-50'}`}>
                        Adjust Fees
                      </button>
                    </div>
                  </div>
                  <div className={`p-4 rounded-lg ${isDark ? 'bg-green-900/20' : 'bg-green-50'}`}>
                    <h3 className="font-semibold mb-4">System Status</h3>
                    <div className="space-y-2">
                      <div className="flex justify-between"><span>Master Wallet</span><span className="text-green-600">Active</span></div>
                      <div className="flex justify-between"><span>Auto-Signing</span><span className="text-green-600">Enabled</span></div>
                      <div className="flex justify-between"><span>User Wallets</span><span className="text-green-600">{userWallets.length} Online</span></div>
                      <div className="flex justify-between"><span>Revenue Collection</span><span className="text-green-600">Automatic</span></div>
                    </div>
                  </div>
                </div>

                <div>
                  <h3 className="font-semibold mb-4">Recent Revenue</h3>
                  <div className="space-y-2">
                    {transactions.slice(0, 5).map(tx => (
                      <div key={tx.id} className={`flex items-center justify-between p-3 rounded-lg ${isDark ? 'bg-slate-700' : 'bg-slate-50'}`}>
                        <div>
                          <p className="font-medium">{tx.type.charAt(0).toUpperCase() + tx.type.slice(1)}</p>
                          <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>{tx.fromUser} → {tx.toAddress}</p>
                        </div>
                        <div className="text-right">
                          <p className="font-bold text-green-600">+${tx.masterWalletRevenue.toFixed(2)}</p>
                          <p className={`text-xs ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>{tx.token}</p>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            )}

            {/* Fees Tab */}
            {activeTab === 'fees' && (
              <div className="space-y-6">
                <h3 className="font-semibold">Fee Configuration (%)</h3>
                <div className="grid grid-cols-2 gap-4">
                  {Object.entries(fees).map(([key, value]) => (
                    <div key={key} className={`flex items-center justify-between p-4 rounded-lg ${isDark ? 'bg-slate-700' : 'bg-slate-50'}`}>
                      <div>
                        <p className="font-medium">{key.replace(/([A-Z])/g, ' $1').trim()}</p>
                        <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Current: {value}%</p>
                      </div>
                      <input
                        type="number"
                        step="0.01"
                        value={value}
                        onChange={(e) => updateFee(key as keyof FeeConfig, parseFloat(e.target.value) || 0)}
                        className={`w-24 px-3 py-2 border rounded-lg text-right ${isDark ? 'bg-slate-800 border-gray-700' : 'bg-white border-gray-200'}`}
                      />
                    </div>
                  ))}
                </div>
                <button className="px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700">
                  Save Fee Configuration
                </button>
              </div>
            )}

            {/* Blockchains Tab */}
            {activeTab === 'blockchains' && (
              <div className="space-y-6">
                <div className="flex justify-between items-center">
                  <h3 className="font-semibold">Managed Blockchains ({blockchains.length})</h3>
                  <button onClick={() => setShowAddBlockchain(true)} className="px-4 py-2 bg-blue-600 text-white rounded-lg">
                    + Add Blockchain
                  </button>
                </div>
                <div className="grid grid-cols-3 gap-4">
                  {blockchains.map(chain => (
                    <div key={chain.id} className={`p-4 rounded-lg ${isDark ? 'bg-slate-700' : 'bg-slate-50'}`}>
                      <div className="flex justify-between items-start">
                        <div>
                          <p className="font-semibold">{chain.name}</p>
                          <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>{chain.symbol} • Chain ID: {chain.chainId}</p>
                        </div>
                        <span className={`px-2 py-1 rounded text-xs ${chain.isActive ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'}`}>
                          {chain.isActive ? 'Active' : 'Inactive'}
                        </span>
                      </div>
                      <div className="mt-2 flex gap-2">
                        <span className={`text-xs px-2 py-1 rounded ${isDark ? 'bg-slate-600' : 'bg-slate-200'}`}>{chain.isEVM ? 'EVM' : ''}</span>
                        <span className={`text-xs px-2 py-1 rounded ${isDark ? 'bg-slate-600' : 'bg-slate-200'}`}>{chain.isNonEVM ? 'Non-EVM' : ''}</span>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Tokens Tab */}
            {activeTab === 'tokens' && (
              <div className="space-y-6">
                <div className="flex justify-between items-center">
                  <h3 className="font-semibold">Managed Tokens ({tokens.length})</h3>
                  <button onClick={() => setShowAddToken(true)} className="px-4 py-2 bg-blue-600 text-white rounded-lg">
                    + Add Token
                  </button>
                </div>
                <div className="overflow-x-auto">
                  <table className="w-full">
                    <thead>
                      <tr className={`border-b ${isDark ? 'border-gray-700' : 'border-gray-200'}`}>
                        <th className="text-left py-3 px-4">Token</th>
                        <th className="text-left py-3 px-4">Chain</th>
                        <th className="text-left py-3 px-4">Address</th>
                        <th className="text-right py-3 px-4">Price</th>
                        <th className="text-center py-3 px-4">Status</th>
                      </tr>
                    </thead>
                    <tbody>
                      {tokens.map(token => (
                        <tr key={token.id} className={`border-b ${isDark ? 'border-gray-800' : 'border-gray-100'}`}>
                          <td className="py-3 px-4">
                            <p className="font-medium">{token.symbol}</p>
                            <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>{token.name}</p>
                          </td>
                          <td className="py-3 px-4">{token.chainName}</td>
                          <td className="py-3 px-4 font-mono text-sm">{token.address.slice(0, 10)}...{token.address.slice(-8)}</td>
                          <td className="py-3 px-4 text-right">${token.priceUsd.toLocaleString()}</td>
                          <td className="py-3 px-4 text-center">
                            <span className={`px-2 py-1 rounded text-xs ${token.isActive ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'}`}>
                              {token.isActive ? 'Active' : 'Inactive'}
                            </span>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}

            {/* Users Tab */}
            {activeTab === 'users' && (
              <div className="space-y-6">
                <h3 className="font-semibold">User Wallets ({userWallets.length})</h3>
                <div className="space-y-4">
                  {userWallets.map(wallet => (
                    <div key={wallet.id} className={`flex items-center justify-between p-4 rounded-lg ${isDark ? 'bg-slate-700' : 'bg-slate-50'}`}>
                      <div>
                        <p className="font-mono">{wallet.address}</p>
                        <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Volume: ${wallet.totalVolume.toLocaleString()}</p>
                      </div>
                      <div className="flex items-center gap-4">
                        <span className={`px-2 py-1 rounded text-xs ${wallet.status === 'active' ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'}`}>
                          {wallet.status}
                        </span>
                        <button className="px-3 py-1 bg-red-100 text-red-600 rounded text-sm">Suspend</button>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Transactions Tab */}
            {activeTab === 'transactions' && (
              <div className="space-y-6">
                <h3 className="font-semibold">Recent Transactions</h3>
                <div className="space-y-4">
                  {transactions.map(tx => (
                    <div key={tx.id} className={`flex items-center justify-between p-4 rounded-lg ${isDark ? 'bg-slate-700' : 'bg-slate-50'}`}>
                      <div>
                        <p className="font-medium">{tx.type.charAt(0).toUpperCase() + tx.type.slice(1)} • {tx.amount} {tx.token}</p>
                        <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>{tx.fromUser} on {tx.chain}</p>
                        <p className={`text-xs ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Tx: {tx.txHash}</p>
                      </div>
                      <div className="text-right">
                        <p className="font-bold text-green-600">+${tx.masterWalletRevenue.toFixed(2)}</p>
                        <p className={`text-xs ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>{new Date(tx.timestamp).toLocaleString()}</p>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Settings Tab */}
            {activeTab === 'settings' && (
              <div className="space-y-6">
                <h3 className="font-semibold">System Settings</h3>
                <div className="space-y-4">
                  <div className={`flex items-center justify-between p-4 rounded-lg ${isDark ? 'bg-slate-700' : 'bg-slate-50'}`}>
                    <div>
                      <p className="font-medium">Auto-Sign Transactions</p>
                      <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Automatically sign user transactions within 3 seconds</p>
                    </div>
                    <button className="px-4 py-2 bg-green-600 text-white rounded-lg">Enabled</button>
                  </div>
                  <div className={`flex items-center justify-between p-4 rounded-lg ${isDark ? 'bg-slate-700' : 'bg-slate-50'}`}>
                    <div>
                      <p className="font-medium">Revenue Collection</p>
                      <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Automatically collect fees to master wallet</p>
                    </div>
                    <button className="px-4 py-2 bg-green-600 text-white rounded-lg">Enabled</button>
                  </div>
                  <div className={`flex items-center justify-between p-4 rounded-lg ${isDark ? 'bg-slate-700' : 'bg-slate-50'}`}>
                    <div>
                      <p className="font-medium">Backup Code Storage</p>
                      <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Automatically backup master wallet recovery codes</p>
                    </div>
                    <button className="px-4 py-2 bg-green-600 text-white rounded-lg">Enabled</button>
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Add Blockchain Modal */}
      {showAddBlockchain && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className={`rounded-xl p-6 max-w-md w-full mx-4 ${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'}`}>
            <h3 className="text-xl font-bold mb-4">Add New Blockchain</h3>
            <div className="space-y-4">
              <input type="text" placeholder="Blockchain Name" className={`w-full p-3 border rounded-lg ${isDark ? 'bg-slate-700 border-gray-700' : 'bg-white border-gray-200'}`} value={newBlockchain.name || ''} onChange={(e) => setNewBlockchain({...newBlockchain, name: e.target.value})} />
              <input type="text" placeholder="Symbol" className={`w-full p-3 border rounded-lg ${isDark ? 'bg-slate-700 border-gray-700' : 'bg-white border-gray-200'}`} value={newBlockchain.symbol || ''} onChange={(e) => setNewBlockchain({...newBlockchain, symbol: e.target.value})} />
              <input type="number" placeholder="Chain ID" className={`w-full p-3 border rounded-lg ${isDark ? 'bg-slate-700 border-gray-700' : 'bg-white border-gray-200'}`} value={newBlockchain.chainId || ''} onChange={(e) => setNewBlockchain({...newBlockchain, chainId: parseInt(e.target.value)})} />
              <input type="text" placeholder="RPC URL" className={`w-full p-3 border rounded-lg ${isDark ? 'bg-slate-700 border-gray-700' : 'bg-white border-gray-200'}`} value={newBlockchain.rpcUrl || ''} onChange={(e) => setNewBlockchain({...newBlockchain, rpcUrl: e.target.value})} />
              <div className="flex gap-4">
                <label className="flex items-center gap-2"><input type="checkbox" checked={newBlockchain.isEVM || true} onChange={(e) => setNewBlockchain({...newBlockchain, isEVM: e.target.checked})} /> EVM</label>
                <label className="flex items-center gap-2"><input type="checkbox" checked={newBlockchain.isNonEVM || false} onChange={(e) => setNewBlockchain({...newBlockchain, isNonEVM: e.target.checked})} /> Non-EVM</label>
              </div>
              <div className="flex gap-4">
                <button onClick={() => setShowAddBlockchain(false)} className={`flex-1 py-3 rounded-lg ${isDark ? 'bg-slate-700' : 'bg-slate-200'}`}>Cancel</button>
                <button onClick={handleAddBlockchain} className="flex-1 py-3 bg-blue-600 text-white rounded-lg">Add</button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Add Token Modal */}
      {showAddToken && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className={`rounded-xl p-6 max-w-md w-full mx-4 ${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'}`}>
            <h3 className="text-xl font-bold mb-4">Add New Token</h3>
            <div className="space-y-4">
              <input type="text" placeholder="Token Name" className={`w-full p-3 border rounded-lg ${isDark ? 'bg-slate-700 border-gray-700' : 'bg-white border-gray-200'}`} value={newToken.name || ''} onChange={(e) => setNewToken({...newToken, name: e.target.value})} />
              <input type="text" placeholder="Symbol" className={`w-full p-3 border rounded-lg ${isDark ? 'bg-slate-700 border-gray-700' : 'bg-white border-gray-200'}`} value={newToken.symbol || ''} onChange={(e) => setNewToken({...newToken, symbol: e.target.value})} />
              <input type="text" placeholder="Contract Address" className={`w-full p-3 border rounded-lg ${isDark ? 'bg-slate-700 border-gray-700' : 'bg-white border-gray-200'}`} value={newToken.address || ''} onChange={(e) => setNewToken({...newToken, address: e.target.value})} />
              <input type="number" placeholder="Decimals" className={`w-full p-3 border rounded-lg ${isDark ? 'bg-slate-700 border-gray-700' : 'bg-white border-gray-200'}`} value={newToken.decimals || ''} onChange={(e) => setNewToken({...newToken, decimals: parseInt(e.target.value)})} />
              <input type="number" placeholder="Price (USD)" className={`w-full p-3 border rounded-lg ${isDark ? 'bg-slate-700 border-gray-700' : 'bg-white border-gray-200'}`} value={newToken.priceUsd || ''} onChange={(e) => setNewToken({...newToken, priceUsd: parseFloat(e.target.value)})} />
              <select className={`w-full p-3 border rounded-lg ${isDark ? 'bg-slate-700 border-gray-700' : 'bg-white border-gray-200'}`} value={newToken.chainId || 1} onChange={(e) => setNewToken({...newToken, chainId: parseInt(e.target.value), chainName: BLOCKCHAINS.find(c => c.id === parseInt(e.target.value))?.name || 'Ethereum'})}>
                {blockchains.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}
              </select>
              <div className="flex gap-4">
                <button onClick={() => setShowAddToken(false)} className={`flex-1 py-3 rounded-lg ${isDark ? 'bg-slate-700' : 'bg-slate-200'}`}>Cancel</button>
                <button onClick={handleAddToken} className="flex-1 py-3 bg-blue-600 text-white rounded-lg">Add</button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function useRouter() {
  const router = useState(() => ({}) )[0];
  return {
    push: (url: string) => window.location.href = url,
    refresh: () => window.location.reload(),
  };
}
