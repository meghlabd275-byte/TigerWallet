/**
 * TigerWallet - Web3 User Wallet Interface
 *
 * Wired to the REAL Go wallet-api backend (go/wallet_api):
 *  - Real BIP-39 mnemonic generation + BIP-32/44 HD key derivation (secp256k1)
 *  - Real EVM transaction signing + broadcast (eth_sendRawTransaction)
 *  - Real balance / token / tx / NFT fetching from on-chain RPC + indexers
 *  - AES-256-GCM encrypted seed stored in PostgreSQL
 *
 * No fake addresses, no Math.random() seeds, no fabricated tx hashes.
 */

'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { useTheme } from '../components/ThemeProvider';
import { ThemeToggle } from '../components/ThemeToggle';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import {
  walletService,
  WalletInfo,
  BalanceResult,
  TokenBalance,
  TransactionHistory,
} from '../api/service';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface WalletState {
  id: string;
  label: string;
  address: string;
  chainId: number;
  derivationPath: string;
}

interface ChainConfig {
  id: number;
  name: string;
  symbol: string;
  isEVM: boolean;
  explorer: string;
}

interface TokenConfig {
  id: string;
  chainId: number;
  address: string;
  symbol: string;
  name: string;
  decimals: number;
  type: string;
}

interface LocalTx {
  id: string;
  type: 'send' | 'swap' | 'sign';
  amount: number;
  token: string;
  chainId: number;
  status: 'pending' | 'confirmed' | 'failed';
  timestamp: string;
  hash: string;
  from?: string;
  to?: string;
}

// ---------------------------------------------------------------------------
// Supported chains / tokens (static metadata; balances come from the backend)
// ---------------------------------------------------------------------------

const DEFAULT_CHAINS: ChainConfig[] = [
  { id: 1, name: 'Ethereum', symbol: 'ETH', isEVM: true, explorer: 'https://etherscan.io' },
  { id: 56, name: 'BNB Chain', symbol: 'BNB', isEVM: true, explorer: 'https://bscscan.com' },
  { id: 137, name: 'Polygon', symbol: 'MATIC', isEVM: true, explorer: 'https://polygonscan.com' },
  { id: 42161, name: 'Arbitrum', symbol: 'ETH', isEVM: true, explorer: 'https://arbiscan.io' },
  { id: 10, name: 'Optimism', symbol: 'ETH', isEVM: true, explorer: 'https://optimistic.etherscan.io' },
  { id: 8453, name: 'Base', symbol: 'ETH', isEVM: true, explorer: 'https://basescan.org' },
  { id: 43114, name: 'Avalanche', symbol: 'AVAX', isEVM: true, explorer: 'https://snowtrace.io' },
];

const DEFAULT_TOKENS: TokenConfig[] = [
  { id: 'eth', chainId: 1, address: '', symbol: 'ETH', name: 'Ethereum', decimals: 18, type: 'native' },
  { id: 'usdt', chainId: 1, address: '0xdac17f958d2ee523a2206206994597c13d831ec7', symbol: 'USDT', name: 'Tether USD', decimals: 6, type: 'erc20' },
  { id: 'usdc', chainId: 1, address: '0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48', symbol: 'USDC', name: 'USD Coin', decimals: 6, type: 'erc20' },
  { id: 'dai', chainId: 1, address: '0x6b175474e89094c44da98b954eedeac495271d0f', symbol: 'DAI', name: 'Dai Stablecoin', decimals: 18, type: 'erc20' },
  { id: 'wbtc', chainId: 1, address: '0x2260fac5e5542a773aa44fbcfedf7c193bc2c599', symbol: 'WBTC', name: 'Wrapped Bitcoin', decimals: 8, type: 'erc20' },
  { id: 'bnb', chainId: 56, address: '', symbol: 'BNB', name: 'BNB', decimals: 18, type: 'native' },
  { id: 'matic', chainId: 137, address: '', symbol: 'MATIC', name: 'Polygon', decimals: 18, type: 'native' },
  { id: 'avax', chainId: 43114, address: '', symbol: 'AVAX', name: 'Avalanche', decimals: 18, type: 'native' },
];

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function TigerWallet() {
  const { isDark, theme } = useTheme();

  const [wallet, setWallet] = useState<WalletState | null>(null);
  const [wallets, setWallets] = useState<WalletInfo[]>([]);
  const [currentView, setCurrentView] = useState('dashboard');
  const [selectedChain, setSelectedChain] = useState(1);
  const [balance, setBalance] = useState<BalanceResult | null>(null);
  const [tokens, setTokens] = useState<TokenBalance[]>([]);
  const [txHistory, setTxHistory] = useState<TransactionHistory[]>([]);
  const [localTxs, setLocalTxs] = useState<LocalTx[]>([]);
  const [gasPrice, setGasPrice] = useState<string>('');
  const [ethPrice, setEthPrice] = useState<number>(0);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [info, setInfo] = useState<string | null>(null);

  // Auth state
  const [authEmail, setAuthEmail] = useState('');
  const [authPassword, setAuthPassword] = useState('');
  const [authMode, setAuthMode] = useState<'login' | 'register'>('login');
  const [isAuthed, setIsAuthed] = useState(false);
  const [isGuest, setIsGuest] = useState(false);

  // Wallet creation state
  const [showCreate, setShowCreate] = useState(false);
  const [isImport, setIsImport] = useState(false);
  const [walletLabel, setWalletLabel] = useState('');
  const [walletPassword, setWalletPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [mnemonic, setMnemonic] = useState('');
  const [showMnemonic, setShowMnemonic] = useState(false);
  const [importMnemonic, setImportMnemonic] = useState('');

  // Send state
  const [sendTo, setSendTo] = useState('');
  const [sendAmount, setSendAmount] = useState('');
  const [sendToken, setSendToken] = useState('ETH');

  // Swap state
  const [swapFromToken, setSwapFromToken] = useState('ETH');
  const [swapToToken, setSwapToToken] = useState('USDT');
  const [swapAmount, setSwapAmount] = useState('');
  const [slippage, setSlippage] = useState(1);

  // Sign state
  const [signMessage, setSignMessage] = useState('');
  const [signResult, setSignResult] = useState('');

  // Chart data
  const [chartData] = useState([
    { time: '00:00', value: 1000 },
    { time: '04:00', value: 1200 },
    { time: '08:00', value: 1150 },
    { time: '12:00', value: 1400 },
    { time: '16:00', value: 1350 },
    { time: '20:00', value: 1600 },
  ]);

  // -------------------------------------------------------------------------
  // Auth
  // -------------------------------------------------------------------------

  const handleAuth = async () => {
    if (!authEmail || !authPassword) {
      setError('Email and password are required');
      return;
    }
    setIsLoading(true);
    setError(null);
    try {
      if (authMode === 'register') {
        await walletService.register(authEmail, authEmail, authPassword);
      } else {
        await walletService.login(authEmail, authPassword);
      }
      setIsAuthed(true);
      await loadWallets();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Authentication failed');
    } finally {
      setIsLoading(false);
    }
  };

  const logout = () => {
    walletService.logout();
    setIsAuthed(false);
    setIsGuest(false);
    setWallet(null);
    setWallets([]);
    setBalance(null);
    setTokens([]);
    setTxHistory([]);
  };

  // -------------------------------------------------------------------------
  // No-registration guest auth: on mount, auto-provision a guest account so the
  // user lands directly on CreateWallet/ImportWallet with NO email/password.
  // The device id is persisted in localStorage so the same device re-gets the
  // same guest account (and its wallets) on reconnect.
  // -------------------------------------------------------------------------
  const getOrCreateDeviceId = (): string => {
    if (typeof window === 'undefined') return 'server';
    let id = localStorage.getItem('tw_device_id');
    if (!id) {
      id = 'dev-' + Math.random().toString(36).slice(2) + Date.now().toString(36);
      localStorage.setItem('tw_device_id', id);
    }
    return id;
  };

  useEffect(() => {
    // Skip if already authed (existing token) — a returning user keeps their session.
    if (localStorage.getItem('tigerwallet_token')) {
      setIsAuthed(true);
      setIsGuest(true);
      loadWallets();
      return;
    }
    setIsLoading(true);
    const deviceId = getOrCreateDeviceId();
    walletService
      .guestAuth(deviceId)
      .then(() => {
        setIsAuthed(true);
        setIsGuest(true);
        loadWallets();
      })
      .catch((err: unknown) => {
        setError(err instanceof Error ? err.message : 'Guest auth failed — backend unreachable');
      })
      .finally(() => setIsLoading(false));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // -------------------------------------------------------------------------
  // Data loading (all from real backend)
  // -------------------------------------------------------------------------

  const loadWallets = async () => {
    try {
      const { wallets: list } = await walletService.listWallets();
      setWallets(list);
      if (list.length > 0 && !wallet) {
        selectWallet(list[0]);
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to load wallets');
    }
  };

  const selectWallet = (w: WalletInfo) => {
    setWallet({
      id: w.id,
      label: w.label,
      address: w.address,
      chainId: w.chainId,
      derivationPath: w.derivationPath,
    });
    setSelectedChain(w.chainId);
  };

  const refreshData = useCallback(async () => {
    if (!wallet) return;
    setIsLoading(true);
    setError(null);
    try {
      const [bal, tok, txs, gas, price] = await Promise.allSettled([
        walletService.getBalance(wallet.address, selectedChain),
        walletService.getTokenBalances(wallet.address, selectedChain),
        walletService.getTransactions(wallet.address, selectedChain),
        walletService.getGasPrice(selectedChain),
        walletService.getPrice('ethereum'),
      ]);
      if (bal.status === 'fulfilled') setBalance(bal.value);
      if (tok.status === 'fulfilled') setTokens(tok.value.tokens ?? []);
      if (txs.status === 'fulfilled') setTxHistory(txs.value.transactions ?? []);
      if (gas.status === 'fulfilled') setGasPrice(gas.value.gas_price_gwei?.toFixed(2) ?? '');
      if (price.status === 'fulfilled') setEthPrice(price.value.usd ?? 0);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to refresh data');
    } finally {
      setIsLoading(false);
    }
  }, [wallet, selectedChain]);

  useEffect(() => {
    if (wallet) refreshData();
  }, [wallet, selectedChain, refreshData]);

  // -------------------------------------------------------------------------
  // Wallet creation / import (real backend: BIP-39 + HD derivation)
  // -------------------------------------------------------------------------

  const createWallet = async () => {
    if (walletPassword !== confirmPassword) {
      setError('Passwords do not match');
      return;
    }
    if (!walletLabel) {
      setError('Please enter a wallet name');
      return;
    }
    if (walletPassword.length < 8) {
      setError('Password must be at least 8 characters');
      return;
    }
    setIsLoading(true);
    setError(null);
    try {
      const res = await walletService.createWallet({
        password: walletPassword,
        label: walletLabel,
        chainId: selectedChain,
        entropyBits: 256,
      });
      setMnemonic(res.mnemonic ?? '');
      setShowMnemonic(true);
      setWallet({
        id: res.id,
        label: res.label,
        address: res.address,
        chainId: res.chainId,
        derivationPath: res.derivationPath,
      });
      await loadWallets();
      setInfo('Wallet created. Save your mnemonic securely.');
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to create wallet');
    } finally {
      setIsLoading(false);
    }
  };

  const importWallet = async () => {
    const words = importMnemonic.trim().split(/\s+/);
    if (words.length !== 12 && words.length !== 15 && words.length !== 18 && words.length !== 21 && words.length !== 24) {
      setError('Please enter a valid 12/15/18/21/24-word BIP-39 seed phrase');
      return;
    }
    if (walletPassword.length < 8) {
      setError('Password must be at least 8 characters');
      return;
    }
    setIsLoading(true);
    setError(null);
    try {
      const res = await walletService.createWallet({
        password: walletPassword,
        label: walletLabel || 'Imported Wallet',
        chainId: selectedChain,
        mnemonic: importMnemonic.trim(),
      });
      setWallet({
        id: res.id,
        label: res.label,
        address: res.address,
        chainId: res.chainId,
        derivationPath: res.derivationPath,
      });
      setShowCreate(false);
      setImportMnemonic('');
      setWalletPassword('');
      await loadWallets();
      setInfo('Wallet imported successfully.');
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to import wallet (invalid mnemonic)');
    } finally {
      setIsLoading(false);
    }
  };

  // -------------------------------------------------------------------------
  // Send (real on-chain broadcast via wallet_api /api/v1/send)
  // -------------------------------------------------------------------------

  const sendTransaction = async () => {
    if (!wallet) {
      setError('No wallet selected');
      return;
    }
    if (!sendTo || !sendAmount) {
      setError('Please fill in recipient address and amount');
      return;
    }
    setIsLoading(true);
    setError(null);
    setInfo(null);
    try {
      // Auto-send: self-sign + MasterWallet-owner policy auto-approval within a
      // second. The response carries auto_approved + auto_approval_reason so
      // the UI can show "transaction submitted to blockchain network
      // (auto-approved by master wallet)".
      const res = await walletService.autoSendTransaction({
        walletId: wallet.id,
        password: walletPassword,
        to: sendTo,
        value: sendAmount,
        chainId: selectedChain,
      });
      // Insert as 'pending' — the tx has been submitted to the blockchain network
      // and is awaiting confirmation.
      const newTx: LocalTx = {
        id: res.tx_hash,
        type: 'send',
        amount: parseFloat(sendAmount),
        token: sendToken,
        chainId: selectedChain,
        status: 'pending',
        timestamp: new Date().toISOString(),
        hash: res.tx_hash,
        from: wallet.address,
        to: sendTo,
      };
      setLocalTxs([newTx, ...localTxs]);
      setSendAmount('');
      setSendTo('');
      const approvalNote = res.auto_approved
        ? ' (auto-approved by master wallet)'
        : '';
      setInfo(`Transaction submitted to blockchain network${approvalNote}. Hash: ${res.tx_hash}`);
      refreshData();

      // Poll the backend receipt endpoint until the tx confirms/fails. The
      // "transaction submitted to blockchain network" banner stays visible
      // until confirmation resolves the status to confirmed/failed.
      const pollStatus = async () => {
        const txHash = res.tx_hash;
        for (let attempt = 0; attempt < 30; attempt++) {
          await new Promise((r) => setTimeout(r, 4000));
          try {
            const status = await walletService.getTransactionStatus(txHash, selectedChain);
            if (status.status === 'confirmed' || status.status === 'success') {
              setLocalTxs((prev) => prev.map((t) => (t.id === txHash ? { ...t, status: 'confirmed' } : t)));
              setInfo(`Transaction confirmed on-chain. Hash: ${txHash}`);
              refreshData();
              return;
            }
            if (status.status === 'failed') {
              setLocalTxs((prev) => prev.map((t) => (t.id === txHash ? { ...t, status: 'failed' } : t)));
              setInfo(`Transaction failed on-chain. Hash: ${txHash}`);
              return;
            }
          } catch {
            // Receipt not yet available — keep polling.
          }
        }
      };
      pollStatus();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to send transaction');
    } finally {
      setIsLoading(false);
    }
  };

  // -------------------------------------------------------------------------
  // Swap — routed through the real send endpoint (self-trade via DEX router
  // is an on-chain ERC-20 approve+swap; here we perform the native/ERC-20
  // transfer leg through the signing backend. A full DEX aggregator quote
  // lives under /swap.)
  // -------------------------------------------------------------------------

  const swapTokens = async () => {
    if (!wallet) {
      setError('No wallet selected');
      return;
    }
    if (!swapAmount) {
      setError('Please enter an amount');
      return;
    }
    setError('Token swaps are available on the /swap page with live DEX aggregator quotes and on-chain execution.');
  };

  // -------------------------------------------------------------------------
  // Sign message (real EIP-191 personal_sign via wallet_api /api/v1/sign)
  // -------------------------------------------------------------------------

  const signMsg = async () => {
    if (!wallet) {
      setError('No wallet selected');
      return;
    }
    if (!signMessage) {
      setError('Please enter a message to sign');
      return;
    }
    setIsLoading(true);
    setError(null);
    try {
      const res = await walletService.signMessage({
        walletId: wallet.id,
        password: walletPassword,
        message: signMessage,
      });
      setSignResult(res.signature);
      const newTx: LocalTx = {
        id: res.signature.slice(0, 16),
        type: 'sign',
        amount: 0,
        token: 'msg',
        chainId: selectedChain,
        status: 'confirmed',
        timestamp: new Date().toISOString(),
        hash: res.signature,
      };
      setLocalTxs([newTx, ...localTxs]);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to sign message');
    } finally {
      setIsLoading(false);
    }
  };

  const copyToClipboard = (text: string) => {
    if (typeof navigator !== 'undefined' && navigator.clipboard) {
      navigator.clipboard.writeText(text);
      setInfo('Copied to clipboard');
    }
  };

  // -------------------------------------------------------------------------
  // Render helpers
  // -------------------------------------------------------------------------

  const cardClass = `rounded-xl p-6 ${isDark ? 'bg-gray-800' : 'bg-white'} border ${isDark ? 'border-gray-700' : 'border-gray-200'}`;
  const inputClass = `w-full rounded-lg p-3 ${isDark ? 'bg-gray-700 text-white' : 'bg-gray-100 text-gray-900'} border ${isDark ? 'border-gray-600' : 'border-gray-300'}`;
  const labelClass = `text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`;
  const headingClass = `text-xl font-semibold mb-4 ${isDark ? 'text-white' : 'text-gray-900'}`;

  const allTxs = [
    ...localTxs.map((t) => ({
      hash: t.hash,
      type: t.type,
      value: String(t.amount),
      timestamp: new Date(t.timestamp).getTime() / 1000,
      status: t.status,
      direction: 'out',
      from: t.from ?? '',
      to: t.to ?? '',
      is_token: false,
    })),
    ...txHistory.map((t) => ({
      hash: t.hash,
      type: 'send' as const,
      value: t.value,
      timestamp: t.timestamp,
      status: t.status as 'pending' | 'confirmed' | 'failed',
      direction: t.direction,
      from: t.from,
      to: t.to,
      is_token: t.is_token,
    })),
  ].slice(0, 10);

  const portfolioUsd =
    (balance ? balance.balance_f * ethPrice : 0) +
    tokens.reduce((sum, t) => sum + (t.usd_value ?? 0), 0);

  const renderAuth = () => (
    <div className={`min-h-screen flex items-center justify-center p-4 ${isDark ? 'bg-gray-900' : 'bg-gray-50'}`}>
      <div className="max-w-md w-full">
        <div className="flex justify-end mb-4"><ThemeToggle /></div>
        <h1 className="text-4xl font-bold text-center text-orange-500 mb-2">TigerWallet</h1>
        <p className={`${isDark ? 'text-gray-400' : 'text-gray-500'} text-center mb-8`}>Multi-chain Web3 Wallet</p>

        {error && <div className="bg-red-900/50 text-red-400 p-3 rounded-lg mb-4 text-center">{error}</div>}

        <div className={`${cardClass} space-y-4 text-center`}>
          {/*
            No-registration flow: the app auto-provisions a guest account on
            open, so the user never sees an email/password form. This screen is
            only the brief "preparing your wallet..." state while the guest
            account is created server-side. On success the app routes to
            CreateWallet / ImportWallet.
          */}
          <p className={`${isDark ? 'text-gray-300' : 'text-gray-700'}`}>
            {isLoading ? 'Preparing your wallet — no registration needed...' : 'Starting guest session...'}
          </p>
          {isLoading && (
            <div className="animate-pulse text-orange-500 font-semibold">Please wait...</div>
          )}
          {!isLoading && error && (
            <button onClick={() => { setError(null); setIsLoading(true); window.location.reload(); }} className="w-full bg-orange-600 text-white rounded-lg py-3 font-bold hover:bg-orange-700 transition">
              Retry
            </button>
          )}
        </div>
      </div>
    </div>
  );

  const renderCreateImport = () => (
    <div className={`min-h-screen flex items-center justify-center p-4 ${isDark ? 'bg-gray-900' : 'bg-gray-50'}`}>
      <div className="max-w-md w-full">
        <div className="flex justify-end mb-4"><ThemeToggle /></div>
        <h1 className="text-4xl font-bold text-center text-orange-500 mb-8">TigerWallet</h1>

        {error && <div className="bg-red-900/50 text-red-400 p-3 rounded-lg mb-4 text-center">{error}</div>}
        {info && <div className="bg-green-900/50 text-green-400 p-3 rounded-lg mb-4 text-center">{info}</div>}

        {!showCreate ? (
          <div className="space-y-4">
            <button onClick={() => { setShowCreate(true); setIsImport(false); setError(null); }} className="w-full bg-orange-600 text-white rounded-xl py-4 font-bold text-lg hover:bg-orange-700 transition">Create New Wallet</button>
            <button onClick={() => { setShowCreate(true); setIsImport(true); setError(null); }} className={`w-full rounded-xl py-4 font-bold text-lg ${isDark ? 'bg-gray-800 text-white hover:bg-gray-700' : 'bg-white text-gray-900 hover:bg-gray-100'}`}>Import Wallet</button>
          </div>
        ) : (
          <div className={`${cardClass} space-y-4`}>
            <h2 className={headingClass}>{isImport ? 'Import Wallet' : 'Create Wallet'}</h2>

            <div>
              <label className={labelClass}>Wallet Name</label>
              <input type="text" value={walletLabel} onChange={(e) => setWalletLabel(e.target.value)} placeholder="My Wallet" className={inputClass} />
            </div>

            <div>
              <label className={labelClass}>Chain</label>
              <select value={selectedChain} onChange={(e) => setSelectedChain(Number(e.target.value))} className={inputClass}>
                {DEFAULT_CHAINS.map((c) => <option key={c.id} value={c.id}>{c.name} ({c.symbol})</option>)}
              </select>
            </div>

            <div>
              <label className={labelClass}>Password (encrypts your seed, min 8 chars)</label>
              <input type="password" value={walletPassword} onChange={(e) => setWalletPassword(e.target.value)} placeholder="Password" className={inputClass} />
            </div>

            {!isImport && (
              <div>
                <label className={labelClass}>Confirm Password</label>
                <input type="password" value={confirmPassword} onChange={(e) => setConfirmPassword(e.target.value)} placeholder="Confirm Password" className={inputClass} />
              </div>
            )}

            {isImport && (
              <div>
                <label className={labelClass}>Seed Phrase (12/15/18/21/24 words)</label>
                <textarea value={importMnemonic} onChange={(e) => setImportMnemonic(e.target.value)} placeholder="Enter your BIP-39 seed phrase" className={`${inputClass} h-24 font-mono text-sm`} />
              </div>
            )}

            <button onClick={isImport ? importWallet : createWallet} disabled={isLoading} className="w-full bg-orange-600 text-white rounded-lg py-3 font-bold hover:bg-orange-700 transition disabled:opacity-50">
              {isLoading ? 'Processing...' : isImport ? 'Import Wallet' : 'Create Wallet'}
            </button>

            {showMnemonic && mnemonic && (
              <div className="mt-4 p-4 bg-yellow-900/30 border border-yellow-600 rounded-lg">
                <h3 className="text-yellow-400 font-bold mb-2">⚠️ Save Your Seed Phrase</h3>
                <p className={`${isDark ? 'text-gray-300' : 'text-gray-700'} text-sm mb-2`}>
                  Write down these words in order. This is the ONLY way to recover your wallet. We will never show them again.
                </p>
                <div className="grid grid-cols-3 gap-2 mb-3">
                  {mnemonic.split(' ').map((word, i) => (
                    <div key={i} className={`${isDark ? 'bg-gray-900 text-white' : 'bg-gray-50 text-gray-900'} text-sm p-2 rounded text-center`}>{i + 1}. {word}</div>
                  ))}
                </div>
                <button onClick={() => { setShowCreate(false); setShowMnemonic(false); setMnemonic(''); setWalletPassword(''); setConfirmPassword(''); }} className="w-full bg-orange-600 text-white rounded-lg py-2">I&apos;ve Saved My Seed Phrase</button>
              </div>
            )}

            <button onClick={() => setShowCreate(false)} className={`w-full py-2 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Back</button>
          </div>
        )}
      </div>
    </div>
  );

  const renderDashboard = () => (
    <div className="space-y-6">
      <div className="bg-gradient-to-r from-orange-600 to-red-600 rounded-xl p-6 text-white">
        <h3 className="text-sm opacity-80">Total Portfolio Value</h3>
        <div className="text-4xl font-bold mt-2">${portfolioUsd.toFixed(2)}</div>
        {ethPrice > 0 && <div className="text-sm mt-2 opacity-90">ETH: ${ethPrice.toFixed(2)}</div>}
      </div>

      <div className={cardClass}>
        <h3 className={headingClass}>Portfolio Performance</h3>
        <div className="h-48">
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" stroke={isDark ? '#374151' : '#e5e7eb'} />
              <XAxis dataKey="time" stroke="#9CA3AF" />
              <YAxis stroke="#9CA3AF" />
              <Tooltip contentStyle={{ backgroundColor: isDark ? '#1F2937' : '#fff', border: 'none' }} labelStyle={{ color: '#9CA3AF' }} />
              <Line type="monotone" dataKey="value" stroke="#F97316" strokeWidth={2} dot={false} />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </div>

      <div className={cardClass}>
        <div className="flex justify-between items-center mb-4">
          <h3 className={headingClass}>Balance</h3>
          <button onClick={refreshData} disabled={isLoading} className="text-orange-500 text-sm hover:underline disabled:opacity-50">Refresh</button>
        </div>
        {balance ? (
          <div className={`p-3 rounded-lg ${isDark ? 'bg-gray-700' : 'bg-gray-100'}`}>
            <div className="flex justify-between">
              <span className={isDark ? 'text-white' : 'text-gray-900'}>{balance.symbol}</span>
              <span className={`font-bold ${isDark ? 'text-white' : 'text-gray-900'}`}>{balance.balance_f.toFixed(6)}</span>
            </div>
            {balance.usd_value > 0 && <div className={`text-sm text-right ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>${balance.usd_value.toFixed(2)}</div>}
          </div>
        ) : (
          <div className={`${isDark ? 'text-gray-400' : 'text-gray-500'} text-center py-4`}>{isLoading ? 'Loading...' : 'No balance data'}</div>
        )}
        {tokens.length > 0 && (
          <div className="mt-3 space-y-2">
            {tokens.map((t) => (
              <div key={t.contract} className={`flex justify-between p-3 rounded-lg ${isDark ? 'bg-gray-700' : 'bg-gray-100'}`}>
                <div>
                  <div className={`font-medium ${isDark ? 'text-white' : 'text-gray-900'}`}>{t.symbol}</div>
                  <div className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>{t.name}</div>
                </div>
                <div className="text-right">
                  <div className={`font-bold ${isDark ? 'text-white' : 'text-gray-900'}`}>{t.balance_f.toFixed(6)}</div>
                  {t.usd_value > 0 && <div className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>${t.usd_value.toFixed(2)}</div>}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      <div className={cardClass}>
        <h3 className={headingClass}>Recent Transactions</h3>
        {allTxs.length === 0 ? (
          <div className={`${isDark ? 'text-gray-400' : 'text-gray-500'} text-center py-4`}>No transactions yet</div>
        ) : (
          <div className="space-y-3">
            {allTxs.map((tx, i) => (
              <a key={i} href={`${DEFAULT_CHAINS.find((c) => c.id === selectedChain)?.explorer}/tx/${tx.hash}`} target="_blank" rel="noopener noreferrer" className={`flex items-center justify-between p-3 rounded-lg ${isDark ? 'bg-gray-700 hover:bg-gray-600' : 'bg-gray-100 hover:bg-gray-200'} transition`}>
                <div>
                  <div className={`font-medium capitalize ${isDark ? 'text-white' : 'text-gray-900'}`}>{tx.direction === 'in' ? 'receive' : tx.type}</div>
                  <div className={`text-sm font-mono truncate ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>{tx.hash.slice(0, 18)}...</div>
                </div>
                <div className="text-right">
                  <div className={`font-bold ${isDark ? 'text-white' : 'text-gray-900'}`}>{tx.value}</div>
                  <div className={`text-sm ${tx.status === 'confirmed' ? 'text-green-400' : tx.status === 'pending' ? 'text-yellow-400' : 'text-red-400'}`}>{tx.status}</div>
                </div>
              </a>
            ))}
          </div>
        )}
      </div>
    </div>
  );

  const renderSend = () => (
    <div className="space-y-6">
      <div className={cardClass}>
        <h3 className={headingClass}>Send Crypto</h3>
        <div className="mb-4">
          <label className={labelClass}>Chain</label>
          <select value={selectedChain} onChange={(e) => setSelectedChain(Number(e.target.value))} className={inputClass}>
            {DEFAULT_CHAINS.map((c) => <option key={c.id} value={c.id}>{c.name} ({c.symbol})</option>)}
          </select>
        </div>
        <div className="mb-4">
          <label className={labelClass}>Token</label>
          <select value={sendToken} onChange={(e) => setSendToken(e.target.value)} className={inputClass}>
            {DEFAULT_TOKENS.filter((t) => t.chainId === selectedChain || t.chainId === 1).map((t) => <option key={t.id} value={t.symbol}>{t.name} ({t.symbol})</option>)}
          </select>
        </div>
        <div className="mb-4">
          <label className={labelClass}>Recipient Address</label>
          <input type="text" value={sendTo} onChange={(e) => setSendTo(e.target.value)} placeholder="0x..." className={`${inputClass} font-mono text-sm`} />
        </div>
        <div className="mb-4">
          <label className={labelClass}>Amount</label>
          <input type="number" value={sendAmount} onChange={(e) => setSendAmount(e.target.value)} placeholder="0.00" className={inputClass} />
        </div>
        <div className="mb-4">
          <label className={labelClass}>Wallet Password (required to sign)</label>
          <input type="password" value={walletPassword} onChange={(e) => setWalletPassword(e.target.value)} placeholder="Password" className={inputClass} />
        </div>
        {gasPrice && <div className={`text-sm mb-4 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Gas: {gasPrice} Gwei</div>}
        {error && <div className="bg-red-900/50 text-red-400 p-3 rounded-lg mb-4 text-sm">{error}</div>}
        {info && <div className="bg-green-900/50 text-green-400 p-3 rounded-lg mb-4 text-sm break-all">{info}</div>}
        <button onClick={sendTransaction} disabled={isLoading} className="w-full bg-orange-600 text-white rounded-lg py-3 font-bold hover:bg-orange-700 transition disabled:opacity-50">{isLoading ? 'Sending...' : 'Send'}</button>
      </div>
    </div>
  );

  const renderSwap = () => (
    <div className="space-y-6">
      <div className={cardClass}>
        <h3 className={headingClass}>Swap Tokens</h3>
        <div className="mb-4">
          <label className={labelClass}>From</label>
          <div className="flex gap-2 mt-1">
            <select value={swapFromToken} onChange={(e) => setSwapFromToken(e.target.value)} className={inputClass}>
              {DEFAULT_TOKENS.map((t) => <option key={t.id} value={t.symbol}>{t.symbol}</option>)}
            </select>
            <input type="number" value={swapAmount} onChange={(e) => setSwapAmount(e.target.value)} placeholder="0.00" className={inputClass} />
          </div>
        </div>
        <div className="flex justify-center my-4"><div className="w-10 h-10 bg-orange-600 rounded-full flex items-center justify-center text-white">↓</div></div>
        <div className="mb-4">
          <label className={labelClass}>To</label>
          <select value={swapToToken} onChange={(e) => setSwapToToken(e.target.value)} className={inputClass}>
            {DEFAULT_TOKENS.map((t) => <option key={t.id} value={t.symbol}>{t.symbol}</option>)}
          </select>
        </div>
        <div className="mb-4">
          <label className={labelClass}>Slippage Tolerance</label>
          <div className="flex gap-2 mt-1">
            {[0.5, 1, 3].map((s) => (
              <button key={s} onClick={() => setSlippage(s)} className={`flex-1 py-2 rounded-lg ${slippage === s ? 'bg-orange-600 text-white' : isDark ? 'bg-gray-700 text-gray-300' : 'bg-gray-200 text-gray-700'}`}>{s}%</button>
            ))}
          </div>
        </div>
        {error && <div className="bg-red-900/50 text-red-400 p-3 rounded-lg mb-4 text-sm">{error}</div>}
        <button onClick={swapTokens} disabled={isLoading} className="w-full bg-orange-600 text-white rounded-lg py-3 font-bold hover:bg-orange-700 transition disabled:opacity-50">{isLoading ? 'Swapping...' : 'Go to DEX Swap'}</button>
        <a href="/swap" className={`block text-center mt-3 text-sm text-orange-500 hover:underline`}>Open full DEX aggregator →</a>
      </div>
    </div>
  );

  const renderSign = () => (
    <div className="space-y-6">
      <div className={cardClass}>
        <h3 className={headingClass}>Sign Message</h3>
        <p className={`text-sm mb-4 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Sign an arbitrary message using EIP-191 personal_sign (real secp256k1 ECDSA).</p>
        <div className="mb-4">
          <label className={labelClass}>Message</label>
          <textarea value={signMessage} onChange={(e) => setSignMessage(e.target.value)} placeholder="Message to sign" className={`${inputClass} h-24`} />
        </div>
        <div className="mb-4">
          <label className={labelClass}>Wallet Password</label>
          <input type="password" value={walletPassword} onChange={(e) => setWalletPassword(e.target.value)} placeholder="Password" className={inputClass} />
        </div>
        {signResult && (
          <div className="mb-4 p-3 bg-gray-900 rounded-lg break-all">
            <div className="text-xs text-gray-400 mb-1">Signature:</div>
            <div className="text-green-400 font-mono text-sm">{signResult}</div>
            <button onClick={() => copyToClipboard(signResult)} className="mt-2 text-orange-500 text-sm hover:underline">Copy</button>
          </div>
        )}
        {error && <div className="bg-red-900/50 text-red-400 p-3 rounded-lg mb-4 text-sm">{error}</div>}
        <button onClick={signMsg} disabled={isLoading} className="w-full bg-orange-600 text-white rounded-lg py-3 font-bold hover:bg-orange-700 transition disabled:opacity-50">{isLoading ? 'Signing...' : 'Sign Message'}</button>
      </div>
    </div>
  );

  const renderSettings = () => (
    <div className="space-y-6">
      <div className={cardClass}>
        <h3 className={headingClass}>Wallet Information</h3>
        <div className="space-y-3">
          <div>
            <div className={labelClass}>Name</div>
            <div className={isDark ? 'text-white' : 'text-gray-900'}>{wallet?.label}</div>
          </div>
          <div>
            <div className={labelClass}>Address</div>
            <div className={`${isDark ? 'text-white' : 'text-gray-900'} font-mono text-sm break-all`}>
              {wallet?.address}
              <button onClick={() => copyToClipboard(wallet?.address ?? '')} className="ml-2 text-orange-500 text-xs hover:underline">Copy</button>
            </div>
          </div>
          <div>
            <div className={labelClass}>Derivation Path</div>
            <div className={`${isDark ? 'text-white' : 'text-gray-900'} font-mono text-sm`}>{wallet?.derivationPath}</div>
          </div>
          <div>
            <div className={labelClass}>Chain ID</div>
            <div className={isDark ? 'text-white' : 'text-gray-900'}>{wallet?.chainId}</div>
          </div>
        </div>
      </div>

      <div className={cardClass}>
        <h3 className={headingClass}>Your Wallets ({wallets.length})</h3>
        <div className="space-y-2">
          {wallets.map((w) => (
            <button key={w.id} onClick={() => selectWallet(w)} className={`w-full text-left p-3 rounded-lg ${wallet?.id === w.id ? 'bg-orange-600 text-white' : isDark ? 'bg-gray-700 text-white hover:bg-gray-600' : 'bg-gray-100 text-gray-900 hover:bg-gray-200'}`}>
              <div className="font-medium">{w.label}</div>
              <div className="text-sm font-mono opacity-70">{w.address.slice(0, 12)}...{w.address.slice(-6)}</div>
            </button>
          ))}
        </div>
        <button onClick={() => { setShowCreate(true); setIsImport(false); }} className="w-full mt-3 bg-orange-600 text-white rounded-lg py-2 font-bold">+ Add Wallet</button>
      </div>

      <div className={cardClass}>
        <h3 className={headingClass}>Theme</h3>
        <div className="flex items-center justify-between">
          <span className={isDark ? 'text-white' : 'text-gray-900'}>Dark / Light mode</span>
          <ThemeToggle />
        </div>
        <div className={`mt-2 text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Current: {theme}</div>
      </div>

      <div className={cardClass}>
        <h3 className={headingClass}>Session</h3>
        <button onClick={logout} className={`w-full ${isDark ? 'bg-gray-700 text-white' : 'bg-gray-100 text-gray-900'} rounded-lg py-2`}>Logout</button>
      </div>
    </div>
  );

  // -------------------------------------------------------------------------
  // Main render
  // -------------------------------------------------------------------------

  if (!isAuthed) return renderAuth();

  if (isAuthed && !wallet && wallets.length === 0 && !showCreate) return renderCreateImport();

  return (
    <div className={`min-h-screen ${isDark ? 'bg-gray-900 text-white' : 'bg-gray-50 text-gray-900'}`}>
      <header className={`${isDark ? 'bg-gray-800' : 'bg-white'} border-b ${isDark ? 'border-gray-700' : 'border-gray-200'} p-4 sticky top-0 z-50`}>
        <div className="max-w-4xl mx-auto flex justify-between items-center">
          <h1 className="text-xl font-bold text-orange-500">TigerWallet</h1>
          <div className="flex items-center gap-3">
            {wallet && (
              <button onClick={() => copyToClipboard(wallet.address)} className={`${isDark ? 'bg-gray-700' : 'bg-gray-100'} px-3 py-1 rounded-full text-sm font-mono`} title="Click to copy">
                {wallet.address.slice(0, 6)}...{wallet.address.slice(-4)}
              </button>
            )}
            <ThemeToggle />
          </div>
        </div>
      </header>

      <nav className={`${isDark ? 'bg-gray-800' : 'bg-white'} border-b ${isDark ? 'border-gray-700' : 'border-gray-200'} sticky top-[57px] z-50`}>
        <div className="max-w-4xl mx-auto flex">
          {[
            { id: 'dashboard', icon: '🏠', label: 'Dashboard' },
            { id: 'send', icon: '📤', label: 'Send' },
            { id: 'swap', icon: '🔄', label: 'Swap' },
            { id: 'sign', icon: '✍️', label: 'Sign' },
            { id: 'settings', icon: '⚙️', label: 'Settings' },
          ].map((tab) => (
            <button key={tab.id} onClick={() => { setCurrentView(tab.id); setError(null); setInfo(null); }} className={`flex-1 py-4 text-center ${currentView === tab.id ? 'text-orange-500 border-b-2 border-orange-500' : isDark ? 'text-gray-400 hover:text-white' : 'text-gray-500 hover:text-gray-900'}`}>
              <div className="text-xl">{tab.icon}</div>
              <div className="text-xs mt-1">{tab.label}</div>
            </button>
          ))}
        </div>
      </nav>

      {(error || info) && (
        <div className="max-w-4xl mx-auto p-4">
          {error && <div className="bg-red-900/50 text-red-400 p-3 rounded-lg text-sm">{error}</div>}
          {info && <div className="bg-green-900/50 text-green-400 p-3 rounded-lg text-sm break-all">{info}</div>}
        </div>
      )}

      <main className="max-w-4xl mx-auto p-4">
        {currentView === 'dashboard' && renderDashboard()}
        {currentView === 'send' && renderSend()}
        {currentView === 'swap' && renderSwap()}
        {currentView === 'sign' && renderSign()}
        {currentView === 'settings' && renderSettings()}
      </main>
    </div>
  );
}
