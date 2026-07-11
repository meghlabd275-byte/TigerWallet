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

// Chain types for dropdown
const CHAIN_TYPES = [
  { value: 'ethereum', label: 'Ethereum', chainId: 1 },
  { value: 'polygon', label: 'Polygon', chainId: 137 },
  { value: 'arbitrum', label: 'Arbitrum', chainId: 42161 },
  { value: 'optimism', label: 'Optimism', chainId: 10 },
  { value: 'base', label: 'Base', chainId: 8453 },
  { value: 'avalanche', label: 'Avalanche', chainId: 43114 },
  { value: 'bsc', label: 'BNB Chain', chainId: 56 },
  { value: 'solana', label: 'Solana', chainId: 101 },
  { value: 'tron', label: 'Tron', chainId: 728126428 },
  { value: 'bitcoin', label: 'Bitcoin', chainId: 0 },
  { value: 'cosmos', label: 'Cosmos', chainId: 118 },
  { value: 'pi', label: 'Pi Network', chainId: 314159 },
  { value: 'ton', label: 'Toncoin', chainId: -239 },
  { value: 'aptos', label: 'Aptos', chainId: 637 },
  { value: 'pulsechain', label: 'PulseChain', chainId: 369 },
  { value: 'dogecoin', label: 'Dogecoin', chainId: 3 },
  { value: 'litecoin', label: 'Litecoin', chainId: 2 },
  { value: 'ripple', label: 'Ripple', chainId: 144 },
  { value: 'cardano', label: 'Cardano', chainId: 3009 },
  { value: 'near', label: 'NEAR', chainId: 1313161554 },
];

// Popular tokens list
const POPULAR_TOKENS = [
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
  { symbol: 'MATIC', name: 'Polygon', decimals: 18 },
  { symbol: 'ARB', name: 'Arbitrum', decimals: 18 },
  { symbol: 'OP', name: 'Optimism', decimals: 18 },
  { symbol: 'AVAX', name: 'Avalanche', decimals: 18 },
  { symbol: 'LINK', name: 'Chainlink', decimals: 18 },
  { symbol: 'DOT', name: 'Polkadot', decimals: 18 },
  { symbol: 'UNI', name: 'Uniswap', decimals: 18 },
  { symbol: 'AAVE', name: 'Aave', decimals: 18 },
  { symbol: 'PAXG', name: 'Paxos Gold', decimals: 18 },
  { symbol: 'WETH', name: 'Wrapped Ethereum', decimals: 18 },
  { symbol: 'WBTC', name: 'Wrapped Bitcoin', decimals: 8 },
  { symbol: 'DAI', name: 'Dai Stablecoin', decimals: 18 },
  { symbol: 'ATOM', name: 'Cosmos', decimals: 6 },
  { symbol: 'NEAR', name: 'NEAR Protocol', decimals: 24 },
  { symbol: 'APT', name: 'Aptos', decimals: 8 },
  { symbol: 'LTC', name: 'Litecoin', decimals: 8 },
  { symbol: 'BCH', name: 'Bitcoin Cash', decimals: 8 },
  { symbol: 'FIL', name: 'Filecoin', decimals: 18 },
  { symbol: 'HBAR', name: 'Hedera', decimals: 18 },
  { symbol: 'VET', name: 'VeChain', decimals: 18 },
  { symbol: 'ALGO', name: 'Algorand', decimals: 6 },
  { symbol: 'XLM', name: 'Stellar', decimals: 7 },
  { symbol: 'XMR', name: 'Monero', decimals: 12 },
  { symbol: 'ZEC', name: 'Zcash', decimals: 8 },
  { symbol: 'EOS', name: 'EOS', decimals: 18 },
  { symbol: 'THETA', name: 'Theta', decimals: 18 },
  { symbol: 'XTZ', name: 'Tezos', decimals: 6 },
  { symbol: 'CAKE', name: 'PancakeSwap', decimals: 18 },
  { symbol: 'LDO', name: 'Lido DAO', decimals: 18 },
  { symbol: 'MKR', name: 'Maker', decimals: 18 },
  { symbol: 'SNX', name: 'Synthetix', decimals: 18 },
  { symbol: 'CRV', name: 'Curve DAO', decimals: 18 },
  { symbol: 'COMP', name: 'Compound', decimals: 18 },
  { symbol: 'SUSHI', name: 'SushiSwap', decimals: 18 },
  { symbol: 'BAT', name: 'Basic Attention Token', decimals: 18 },
  { symbol: 'ENJ', name: 'Enjin Coin', decimals: 18 },
  { symbol: 'MANA', name: 'Decentraland', decimals: 18 },
  { symbol: 'SAND', name: 'The Sandbox', decimals: 18 },
  { symbol: 'AXS', name: 'Axie Infinity', decimals: 18 },
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
