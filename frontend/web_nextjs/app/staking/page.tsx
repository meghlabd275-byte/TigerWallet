'use client';

import React, { useState, useCallback, useEffect } from 'react';
import { useTheme } from '../components/ThemeProvider';

// A staking asset as reported by GET /api/v1/staking/quote. The backend leaves
// apy at 0 until a live staking contract or oracle is configured, so the UI
// shows that rather than inventing a yield.
interface StakingAsset {
  symbol: string;
  chain_id: number;
  apy: number;
  min_stake: number;
  lock_period: number;
  verified: boolean;
}

interface WalletOption {
  id: string;
  label: string;
  chain_id: number;
}

const CHAIN_NAMES: Record<number, string> = {
  1: 'Ethereum',
  56: 'BNB Chain',
  137: 'Polygon',
  42161: 'Arbitrum',
  10: 'Optimism',
  8453: 'Base',
  43114: 'Avalanche',
  250: 'Fantom',
};

const chainName = (id: number) => CHAIN_NAMES[id] || `Chain ${id}`;

const API_BASE = process.env.NEXT_PUBLIC_API_URL || '';

const getAuthHeaders = (): HeadersInit => {
  const headers: HeadersInit = { 'Content-Type': 'application/json' };
  if (typeof window !== 'undefined') {
    const token = localStorage.getItem('tigerwallet_token');
    if (token) headers['Authorization'] = `Bearer ${token}`;
  }
  return headers;
};

export default function Staking() {
  const [assets, setAssets] = useState<StakingAsset[]>([]);
  const [wallets, setWallets] = useState<WalletOption[]>([]);
  const [selectedAsset, setSelectedAsset] = useState<StakingAsset | null>(null);
  const [selectedWallet, setSelectedWallet] = useState('');
  const [stakeAmount, setStakeAmount] = useState('');
  const [stakingContract, setStakingContract] = useState('');
  const [callData, setCallData] = useState('');
  const [password, setPassword] = useState('');
  const [activeTab, setActiveTab] = useState<'assets' | 'stake'>('assets');
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);
  const { isDark } = useTheme();

  const card = isDark ? 'bg-gray-800' : 'bg-white border border-gray-200';
  const muted = isDark ? 'text-gray-400' : 'text-gray-500';
  const field = `${isDark ? 'bg-gray-700' : 'bg-gray-100'} border-0 rounded-lg px-4 py-3 w-full`;

  const loadAssets = useCallback(async () => {
    try {
      const res = await fetch(`${API_BASE}/api/v1/staking/quote`, { headers: getAuthHeaders() });
      const data = await res.json();
      if (res.ok) {
        setAssets(Array.isArray(data.assets) ? data.assets : []);
      } else {
        setMessage({ type: 'error', text: data.error || 'Could not load staking assets' });
      }
    } catch {
      setMessage({ type: 'error', text: 'Staking backend unreachable' });
    }
  }, []);

  const loadWallets = useCallback(async () => {
    try {
      const res = await fetch(`${API_BASE}/api/v1/wallets`, { headers: getAuthHeaders() });
      if (!res.ok) return;
      const data = await res.json();
      const list = Array.isArray(data.wallets) ? data.wallets : Array.isArray(data) ? data : [];
      setWallets(list.map((w: any) => ({
        id: w.id,
        label: w.label || w.address,
        chain_id: w.chain_id,
      })));
    } catch {
      // Optional: the stake form still validates its own input.
    }
  }, []);

  useEffect(() => {
    loadAssets();
    loadWallets();
  }, [loadAssets, loadWallets]);

  const submitStake = useCallback(async () => {
    if (!selectedAsset || !selectedWallet) {
      setMessage({ type: 'error', text: 'Select an asset and a wallet' });
      return;
    }
    const amt = parseFloat(stakeAmount);
    if (!Number.isFinite(amt) || amt <= 0) {
      setMessage({ type: 'error', text: 'Enter a valid positive amount' });
      return;
    }
    if (selectedAsset.min_stake > 0 && amt < selectedAsset.min_stake) {
      setMessage({ type: 'error', text: `Minimum stake is ${selectedAsset.min_stake}` });
      return;
    }
    if (!stakingContract || !callData) {
      setMessage({
        type: 'error',
        text: 'Enter the staking contract address and calldata for this protocol. Staking is an on-chain transaction, so the exact contract call is required.',
      });
      return;
    }

    setLoading(true);
    setMessage(null);
    try {
      // The backend never fabricates a transaction. It returns the exact call
      // to broadcast, which is then submitted through /api/v1/send.
      const res = await fetch(`${API_BASE}/api/v1/staking/stake`, {
        method: 'POST',
        headers: getAuthHeaders(),
        body: JSON.stringify({
          wallet_id: selectedWallet,
          password,
          token: selectedAsset.symbol,
          amount: stakeAmount,
          chain_id: selectedAsset.chain_id,
          staking_contract: stakingContract,
          call_data: callData,
        }),
      });
      const data = await res.json();
      if (!res.ok || !data.success) {
        setMessage({ type: 'error', text: data.error || 'Stake request rejected' });
        return;
      }

      const send = await fetch(`${API_BASE}/api/v1/send`, {
        method: 'POST',
        headers: getAuthHeaders(),
        body: JSON.stringify({
          wallet_id: selectedWallet,
          password,
          to: data.tx.to,
          value: data.tx.value || '0',
          data: data.tx.data,
          chain_id: data.tx.chain_id,
        }),
      });
      const sent = await send.json();
      if (send.ok && sent.tx_hash) {
        setMessage({ type: 'success', text: `Stake submitted to the blockchain network: ${sent.tx_hash}` });
        setStakeAmount('');
        setCallData('');
      } else {
        setMessage({ type: 'error', text: sent.error || 'Broadcast failed' });
      }
    } catch (err) {
      setMessage({ type: 'error', text: err instanceof Error ? err.message : 'Stake failed' });
    } finally {
      setLoading(false);
    }
  }, [selectedAsset, selectedWallet, stakeAmount, stakingContract, callData, password]);

  return (
    <div className={`min-h-screen ${isDark ? 'bg-gray-900 text-white' : 'bg-gray-50 text-gray-900'}`}>
      <header className={isDark ? 'bg-gray-800' : 'bg-white border-b border-gray-200'}>
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center gap-4">
              <a href="/" className="text-2xl">🐯</a>
              <h1 className="text-xl font-bold">Staking</h1>
            </div>
            <nav className="flex gap-4">
              <a href="/wallet" className={`${muted} hover:text-orange-500`}>Wallet</a>
            </nav>
          </div>
        </div>
      </header>

      {message && (
        <div className="fixed top-20 right-4 z-50">
          <div className={`px-6 py-3 rounded-lg shadow-lg text-white ${message.type === 'success' ? 'bg-green-500' : 'bg-red-500'}`}>
            {message.text}
          </div>
        </div>
      )}

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className={`${card} rounded-lg p-4 mb-6`}>
          <p className={`text-sm ${muted}`}>
            Staking submits a real on-chain transaction. Choose an asset and a wallet, then supply the
            staking contract address and calldata for that protocol. The transaction is broadcast
            through the wallet backend and the returned hash is shown.
          </p>
        </div>

        <div className={`flex border-b ${isDark ? 'border-gray-700' : 'border-gray-200'} mb-6`}>
          <button onClick={() => setActiveTab('assets')} className={`px-6 py-3 ${activeTab === 'assets' ? 'border-b-2 border-orange-500 text-orange-500' : muted}`}>
            Staking Assets
          </button>
          <button onClick={() => setActiveTab('stake')} className={`px-6 py-3 ${activeTab === 'stake' ? 'border-b-2 border-orange-500 text-orange-500' : muted}`}>
            Stake
          </button>
        </div>

        {activeTab === 'assets' && (
          assets.length === 0 ? (
            <div className={`${card} rounded-lg p-12 text-center`}>
              <h3 className="text-xl font-semibold mb-2">No staking assets available</h3>
              <p className={muted}>The backend reported no supported staking assets.</p>
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {assets.map((a) => (
                <div key={`${a.chain_id}-${a.symbol}`} className={`${card} rounded-lg p-4 shadow-sm`}>
                  <div className="flex items-center justify-between mb-3">
                    <div>
                      <div className="font-semibold">{a.symbol}</div>
                      <div className={`text-xs ${muted}`}>{chainName(a.chain_id)}</div>
                    </div>
                    <span className="bg-orange-100 text-orange-600 px-2 py-1 rounded text-sm font-semibold">
                      {a.apy > 0 ? `${a.apy}% APY` : 'APY n/a'}
                    </span>
                  </div>
                  <div className={`text-xs ${muted} mb-3`}>
                    {a.verified ? 'Verified' : 'Unverified'} · Min stake {a.min_stake} · Lock {a.lock_period} days
                  </div>
                  <button
                    onClick={() => { setSelectedAsset(a); setActiveTab('stake'); }}
                    className="w-full bg-orange-500 hover:bg-orange-600 text-white py-2 rounded-lg"
                  >
                    Stake {a.symbol}
                  </button>
                </div>
              ))}
            </div>
          )
        )}

        {activeTab === 'stake' && (
          <div className={`max-w-lg mx-auto ${card} rounded-lg p-6`}>
            <h2 className="text-xl font-semibold mb-4">Stake {selectedAsset ? selectedAsset.symbol : ''}</h2>

            <div className="mb-4">
              <label className={`block text-sm ${muted} mb-2`}>Asset</label>
              <select
                value={selectedAsset ? `${selectedAsset.chain_id}-${selectedAsset.symbol}` : ''}
                onChange={(e) => {
                  const found = assets.find((a) => `${a.chain_id}-${a.symbol}` === e.target.value);
                  setSelectedAsset(found || null);
                }}
                className={field}
              >
                <option value="">Select an asset</option>
                {assets.map((a) => (
                  <option key={`${a.chain_id}-${a.symbol}`} value={`${a.chain_id}-${a.symbol}`}>
                    {a.symbol} · {chainName(a.chain_id)}
                  </option>
                ))}
              </select>
            </div>

            <div className="mb-4">
              <label className={`block text-sm ${muted} mb-2`}>Wallet</label>
              <select value={selectedWallet} onChange={(e) => setSelectedWallet(e.target.value)} className={field}>
                <option value="">Select a wallet</option>
                {wallets.map((w) => (
                  <option key={w.id} value={w.id}>{w.label} · {chainName(w.chain_id)}</option>
                ))}
              </select>
            </div>

            <div className="mb-4">
              <label className={`block text-sm ${muted} mb-2`}>Amount</label>
              <input
                type="number"
                value={stakeAmount}
                onChange={(e) => setStakeAmount(e.target.value)}
                placeholder={selectedAsset && selectedAsset.min_stake > 0 ? `Min: ${selectedAsset.min_stake}` : 'Amount'}
                className={field}
              />
            </div>

            <div className="mb-4">
              <label className={`block text-sm ${muted} mb-2`}>Staking contract address</label>
              <input value={stakingContract} onChange={(e) => setStakingContract(e.target.value)} placeholder="0x…" className={field} />
            </div>

            <div className="mb-4">
              <label className={`block text-sm ${muted} mb-2`}>Calldata</label>
              <textarea value={callData} onChange={(e) => setCallData(e.target.value)} placeholder="0x…" rows={3} className={`${field} font-mono text-sm`} />
            </div>

            <div className="mb-6">
              <label className={`block text-sm ${muted} mb-2`}>Wallet password</label>
              <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} className={field} />
            </div>

            <div className="flex gap-4">
              <button onClick={() => setActiveTab('assets')} className={`flex-1 ${isDark ? 'bg-gray-700' : 'bg-gray-200'} py-3 rounded-lg`}>
                Cancel
              </button>
              <button onClick={submitStake} disabled={loading} className="flex-1 bg-orange-500 hover:bg-orange-600 disabled:bg-slate-400 text-white py-3 rounded-lg">
                {loading ? 'Submitting…' : 'Stake Now'}
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
