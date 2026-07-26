'use client';

import React, { useState, useEffect, useCallback } from 'react';

interface GasPrice {
  chainId: number;
  chainName: string;
  slow: string;
  standard: string;
  fast: string;
  slowUsd: number;
  standardUsd: number;
  fastUsd: number;
  lastUpdated: number;
}

const CHAIN_CONFIG = [
  { id: 1, name: 'Ethereum', symbol: 'ETH', color: '#627EEA' },
  { id: 137, name: 'Polygon', symbol: 'MATIC', color: '#8247E5' },
  { id: 42161, name: 'Arbitrum', symbol: 'ETH', color: '#28A0F0' },
  { id: 10, name: 'Optimism', symbol: 'ETH', color: '#FF0420' },
  { id: 43114, name: 'Avalanche', symbol: 'AVAX', color: '#E84142' },
  { id: 56, name: 'BNB Chain', symbol: 'BNB', color: '#F3BA2F' },
  { id: 8453, name: 'Base', symbol: 'ETH', color: '#0052FF' },
];

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8445';

async function fetchGasPrices(): Promise<Record<string, GasPrice>> {
  try {
    const res = await fetch(`${API_BASE}/api/v1/gas`);
    const data = await res.json();
    return data.prices || {};
  } catch {
    return {};
  }
}

async function fetchGasPrice(chainId: number): Promise<GasPrice | null> {
  try {
    const res = await fetch(`${API_BASE}/api/v1/gas/${chainId}`);
    if (!res.ok) return null;
    return await res.json();
  } catch {
    return null;
  }
}

async function fetchCostEstimate(chainId: number, operation: string, gasLimit?: number): Promise<any> {
  try {
    const res = await fetch(`${API_BASE}/api/v1/gas/estimate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ chainId, operation, gasLimit }),
    });
    if (!res.ok) return null;
    return await res.json();
  } catch {
    return null;
  }
}

function formatGwei(wei: string): string {
  const num = parseFloat(wei);
  if (num >= 1e9) return (num / 1e9).toFixed(2) + ' gwei';
  if (num >= 1e6) return (num / 1e6).toFixed(2) + ' Mwei';
  return wei + ' wei';
}

function formatUSD(usd: number): string {
  return '$' + usd.toFixed(2);
}

function ChainCard({ chain, gasPrice, onSelect, isSelected }: any) {
  return (
    <div
      onClick={onSelect}
      className={`p-4 rounded-xl border-2 cursor-pointer transition-all ${
        isSelected ? 'border-orange-500 bg-orange-50 dark:bg-orange-900/20' : 'border-slate-200 dark:border-slate-700 hover:border-orange-300'
      }`}
    >
      <div className="flex items-center gap-3 mb-3">
        <div className="w-10 h-10 rounded-full flex items-center justify-center text-white font-bold" style={{ backgroundColor: chain.color }}>
          {chain.symbol[0]}
        </div>
        <div>
          <div className="font-semibold text-slate-900 dark:text-white">{chain.name}</div>
          <div className="text-sm text-slate-500 dark:text-slate-400">{chain.symbol}</div>
        </div>
      </div>
      {gasPrice ? (
        <div className="space-y-2">
          <div className="flex justify-between text-sm">
            <span className="text-slate-500 dark:text-slate-400">Fast</span>
            <span className="font-medium text-slate-900 dark:text-white">{formatGwei(gasPrice.fast)}</span>
          </div>
          <div className="flex justify-between text-sm">
            <span className="text-slate-500 dark:text-slate-400">Standard</span>
            <span className="font-medium text-slate-900 dark:text-white">{formatGwei(gasPrice.standard)}</span>
          </div>
          <div className="flex justify-between text-sm">
            <span className="text-slate-500 dark:text-slate-400">Slow</span>
            <span className="font-medium text-slate-900 dark:text-white">{formatGwei(gasPrice.slow)}</span>
          </div>
          <div className="pt-2 border-t border-slate-200 dark:border-slate-700">
            <div className="flex justify-between text-sm">
              <span className="text-slate-500 dark:text-slate-400">Est. Cost</span>
              <span className="font-semibold text-orange-600 dark:text-orange-400">{formatUSD(gasPrice.standardUsd)}</span>
            </div>
          </div>
        </div>
      ) : (
        <div className="text-center text-slate-400">Loading...</div>
      )}
    </div>
  );
}

function CostEstimator({ selectedChain }: { selectedChain: number | null }) {
  const [operation, setOperation] = useState('transfer');
  const [estimate, setEstimate] = useState<any>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!selectedChain) return;
    const fetchEstimate = async () => {
      setLoading(true);
      const result = await fetchCostEstimate(selectedChain, operation);
      setEstimate(result);
      setLoading(false);
    };
    fetchEstimate();
  }, [selectedChain, operation]);

  if (!selectedChain) return null;

  return (
    <div className="bg-white dark:bg-slate-800 rounded-xl p-6 border border-slate-200 dark:border-slate-700">
      <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">Cost Estimator</h3>
      <div className="mb-4">
        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">Operation Type</label>
        <select
          value={operation}
          onChange={(e) => setOperation(e.target.value)}
          className="w-full px-4 py-2 rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-700 text-slate-900 dark:text-white"
        >
          <option value="transfer">Token Transfer</option>
          <option value="swap">DEX Swap</option>
          <option value="approve">Token Approval</option>
          <option value="nft_transfer">NFT Transfer</option>
          <option value="stake">Staking</option>
          <option value="bridge">Cross-Chain Bridge</option>
        </select>
      </div>
      {loading ? (
        <div className="text-center py-4"><div className="animate-spin rounded-full h-8 w-8 border-b-2 border-orange-500 mx-auto"></div></div>
      ) : estimate ? (
        <div className="space-y-3">
          <div className="flex justify-between">
            <span className="text-slate-500 dark:text-slate-400">Gas Limit</span>
            <span className="font-medium text-slate-900 dark:text-white">{estimate.gasLimit?.toLocaleString()}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-slate-500 dark:text-slate-400">Gas Price</span>
            <span className="font-medium text-slate-900 dark:text-white">{formatGwei(estimate.gasPrice)}</span>
          </div>
          <div className="pt-3 border-t border-slate-200 dark:border-slate-700">
            <div className="flex justify-between">
              <span className="text-slate-700 dark:text-slate-300 font-medium">Total Cost</span>
              <span className="text-xl font-bold text-orange-600 dark:text-orange-400">{formatUSD(estimate.totalCostUsd)}</span>
            </div>
          </div>
        </div>
      ) : (
        <div className="text-center text-slate-400">Select a chain to estimate</div>
      )}
    </div>
  );
}

export default function GasTrackerPage() {
  const [prices, setPrices] = useState<Record<string, GasPrice>>({});
  const [selectedChain, setSelectedChain] = useState<number | null>(null);
  const [loading, setLoading] = useState(true);
  const [lastUpdate, setLastUpdate] = useState<Date | null>(null);

  const loadPrices = useCallback(async () => {
    setLoading(true);
    const data = await fetchGasPrices();
    setPrices(data);
    setLastUpdate(new Date());
    setLoading(false);
  }, []);

  useEffect(() => {
    loadPrices();
    const interval = setInterval(loadPrices, 30000);
    return () => clearInterval(interval);
  }, [loadPrices]);

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 to-slate-100 dark:from-slate-900 dark:to-slate-800">
      <header className="bg-white dark:bg-slate-800 border-b border-slate-200 dark:border-slate-700 sticky top-0 z-10">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center gap-3">
              <a href="/wallet" className="text-2xl">🐯</a>
              <div>
                <h1 className="text-xl font-bold text-slate-900 dark:text-white">Gas Tracker</h1>
                <p className="text-sm text-slate-500 dark:text-slate-400">Real-time gas prices across all chains</p>
              </div>
            </div>
            <div className="flex items-center gap-4">
              {lastUpdate && <span className="text-sm text-slate-500 dark:text-slate-400">Updated: {lastUpdate.toLocaleTimeString()}</span>}
              <button onClick={loadPrices} disabled={loading} className="px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white rounded-lg font-medium disabled:opacity-50">
                {loading ? 'Refreshing...' : 'Refresh'}
              </button>
            </div>
          </div>
        </div>
      </header>
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-8">
          <div className="bg-white dark:bg-slate-800 rounded-xl p-6 border border-slate-200 dark:border-slate-700">
            <div className="text-3xl font-bold text-slate-900 dark:text-white">{CHAIN_CONFIG.length}</div>
            <div className="text-sm text-slate-500 dark:text-slate-400">Supported Chains</div>
          </div>
          <div className="bg-white dark:bg-slate-800 rounded-xl p-6 border border-slate-200 dark:border-slate-700">
            <div className="text-3xl font-bold text-green-500">{Object.keys(prices).length}</div>
            <div className="text-sm text-slate-500 dark:text-slate-400">Active Networks</div>
          </div>
          <div className="bg-white dark:bg-slate-800 rounded-xl p-6 border border-slate-200 dark:border-slate-700">
            <div className="text-3xl font-bold text-orange-500">{lastUpdate ? '30s' : '--'}</div>
            <div className="text-sm text-slate-500 dark:text-slate-400">Update Interval</div>
          </div>
        </div>
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
          <div className="lg:col-span-2">
            <h2 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">Gas Prices by Chain</h2>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {CHAIN_CONFIG.map((chain) => (
                <ChainCard key={chain.id} chain={chain} gasPrice={prices[chain.name] || null} onSelect={() => setSelectedChain(chain.id)} isSelected={selectedChain === chain.id} />
              ))}
            </div>
          </div>
          <div>
            <CostEstimator selectedChain={selectedChain} />
            <div className="mt-6 bg-white dark:bg-slate-800 rounded-xl p-6 border border-slate-200 dark:border-slate-700">
              <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">Gas Saving Tips</h3>
              <ul className="space-y-3 text-sm text-slate-600 dark:text-slate-300">
                <li className="flex items-start gap-2"><span className="text-green-500">✓</span><span>Set appropriate slippage to avoid failed transactions</span></li>
                <li className="flex items-start gap-2"><span className="text-green-500">✓</span><span>Avoid peak hours (9am-5pm EST) for lower fees</span></li>
                <li className="flex items-start gap-2"><span className="text-green-500">✓</span><span>Use Layer 2 networks for cheaper transactions</span></li>
                <li className="flex items-start gap-2"><span className="text-green-500">✓</span><span>Batch multiple operations when possible</span></li>
              </ul>
            </div>
          </div>
        </div>
      </main>
    </div>
  );
}
