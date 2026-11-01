'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { useTheme } from '../components/ThemeProvider';

interface GasTier {
  priority_fee_wei?: string;
  max_fee_wei?: string;
  base_fee_wei?: string;
}

interface GasPrice {
  chainId: number;
  gasPriceWei: string;
  maxFeeWei: string | null;
  priorityFeeWei: string | null;
  tiers: { slow: GasTier; standard: GasTier; fast: GasTier } | null;
}

// Native price in USD, or null when the price feed is unavailable. A null is
// rendered as "unavailable" rather than as a zero or a stale value.
type PriceMap = Record<number, { gas: GasPrice | null; usd: number | null }>;

// Each entry carries the CoinGecko id the price endpoint expects, so the USD
// column is a real market price rather than an assumed one.
const CHAIN_CONFIG = [
  { id: 1, name: 'Ethereum', symbol: 'ETH', coin: 'ethereum', color: '#627EEA' },
  { id: 137, name: 'Polygon', symbol: 'POL', coin: 'matic-network', color: '#8247E5' },
  { id: 42161, name: 'Arbitrum', symbol: 'ETH', coin: 'ethereum', color: '#28A0F0' },
  { id: 10, name: 'Optimism', symbol: 'ETH', coin: 'ethereum', color: '#FF0420' },
  { id: 43114, name: 'Avalanche', symbol: 'AVAX', coin: 'avalanche-2', color: '#E84142' },
  { id: 56, name: 'BNB Chain', symbol: 'BNB', coin: 'binancecoin', color: '#F3BA2F' },
  { id: 8453, name: 'Base', symbol: 'ETH', coin: 'ethereum', color: '#0052FF' },
];

// Same-origin: the Next.js app proxies /api/v1/* to the wallet backend.
const API_BASE = '';

async function fetchGasPrice(chainId: number): Promise<GasPrice | null> {
  try {
    const res = await fetch(`${API_BASE}/api/v1/gas?chain_id=${chainId}`, { cache: 'no-store' });
    if (!res.ok) return null;
    const data = await res.json();
    if (!data.gas_price) return null;
    return {
      chainId,
      gasPriceWei: String(data.gas_price),
      maxFeeWei: data.max_fee_per_gas ? String(data.max_fee_per_gas) : null,
      priorityFeeWei: data.max_priority_fee ? String(data.max_priority_fee) : null,
      tiers: data.tiers || null,
    };
  } catch {
    return null;
  }
}

async function fetchNativePriceUsd(coin: string): Promise<number | null> {
  try {
    const res = await fetch(`${API_BASE}/api/v1/price?coin=${encodeURIComponent(coin)}`, { cache: 'no-store' });
    if (!res.ok) return null;
    const data = await res.json();
    return typeof data.usd === 'number' ? data.usd : null;
  } catch {
    return null;
  }
}

async function fetchCostEstimate(
  chainId: number,
  from: string,
  to: string,
  data: string
): Promise<{ gas_limit: number; fee?: Record<string, string>; estimated_cost_wei?: string } | null> {
  try {
    const res = await fetch(`${API_BASE}/api/v1/gas/estimate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ chain_id: chainId, from, to, data }),
    });
    if (!res.ok) return null;
    return await res.json();
  } catch {
    return null;
  }
}

// Exact wei -> gwei. BigInt avoids the precision loss of parseFloat on 18-digit
// values, which can move the displayed fee by a meaningful amount.
function formatGwei(wei: string | null | undefined): string {
  if (!wei) return '-';
  try {
    const value = BigInt(wei);
    const whole = value / 1000000000n;
    const frac = (value % 1000000000n).toString().padStart(9, '0').replace(/0+$/, '');
    return (frac ? `${whole}.${frac}` : `${whole}`) + ' gwei';
  } catch {
    return '-';
  }
}

function formatUSD(usd: number): string {
  return '$' + usd.toFixed(2);
}

function FeeLine({ label, value, isDark }: { label: string; value?: string; isDark: boolean }) {
  return (
    <div className="flex justify-between text-sm">
      <span className={isDark ? 'text-slate-400' : 'text-slate-500'}>{label}</span>
      <span className={`font-medium ${isDark ? 'text-white' : 'text-slate-900'}`}>{formatGwei(value)}</span>
    </div>
  );
}

function ChainCard({
  chain,
  gasPrice,
  usd,
  onSelect,
  isSelected,
}: {
  chain: (typeof CHAIN_CONFIG)[number];
  gasPrice: GasPrice | null;
  usd: number | null;
  onSelect: () => void;
  isSelected: boolean;
}) {
  const { isDark } = useTheme();
  return (
    <div
      onClick={onSelect}
      className={`p-4 rounded-xl border-2 cursor-pointer transition-all ${
        isSelected
          ? (isDark ? 'border-orange-500 bg-orange-900/20' : 'border-orange-500 bg-orange-50')
          : (isDark ? 'border-slate-700 hover:border-orange-300' : 'border-slate-200 hover:border-orange-300')
      }`}
    >
      <div className="flex items-center gap-3 mb-3">
        <div className="w-10 h-10 rounded-full flex items-center justify-center text-white font-bold" style={{ backgroundColor: chain.color }}>
          {chain.symbol[0]}
        </div>
        <div>
          <div className={`font-semibold ${isDark ? 'text-white' : 'text-slate-900'}`}>{chain.name}</div>
          <div className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>{chain.symbol}</div>
        </div>
      </div>
      {gasPrice ? (
        <div className="space-y-2">
          {gasPrice.tiers ? (
            <>
              <FeeLine label="Fast" value={gasPrice.tiers.fast.max_fee_wei} isDark={isDark} />
              <FeeLine label="Standard" value={gasPrice.tiers.standard.max_fee_wei} isDark={isDark} />
              <FeeLine label="Slow" value={gasPrice.tiers.slow.max_fee_wei} isDark={isDark} />
            </>
          ) : (
            <FeeLine label="Base fee" value={gasPrice.gasPriceWei} isDark={isDark} />
          )}
          <div className={`pt-2 border-t ${isDark ? 'border-slate-700' : 'border-slate-200'}`}>
            <div className="flex justify-between text-sm">
              <span className={isDark ? 'text-slate-400' : 'text-slate-500'}>Native price</span>
              <span className={`font-semibold ${isDark ? 'text-orange-400' : 'text-orange-600'}`}>
                {usd === null ? 'unavailable' : formatUSD(usd)}
              </span>
            </div>
          </div>
        </div>
      ) : (
        <div className="text-center text-slate-400">Unavailable</div>
      )}
    </div>
  );
}

function CostEstimator({ selectedChain }: { selectedChain: number | null }) {
  const { isDark } = useTheme();
  const [from, setFrom] = useState('');
  const [to, setTo] = useState('');
  const [estimate, setEstimate] = useState<{ gas_limit: number; fee?: Record<string, string>; estimated_cost_wei?: string } | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!selectedChain) return;
    // The backend requires a `from` address and derives the gas limit from the
    // actual call, so a full address pair is needed before an estimate exists.
    if (!/^0x[0-9a-fA-F]{40}$/.test(from) || !/^0x[0-9a-fA-F]{40}$/.test(to)) {
      setEstimate(null);
      setError(null);
      return;
    }
    const controller = new AbortController();
    const run = async () => {
      setLoading(true);
      setError(null);
      const result = await fetchCostEstimate(selectedChain, from, to, '');
      if (!controller.signal.aborted) {
        setEstimate(result);
        if (!result) setError('Estimate unavailable for this chain or call.');
        setLoading(false);
      }
    };
    run();
    return () => controller.abort();
  }, [selectedChain, from, to]);

  if (!selectedChain) return null;

  return (
    <div className={`rounded-xl p-6 border ${isDark ? 'bg-slate-800 border-slate-700' : 'bg-white border-slate-200'}`}>
      <h3 className={`text-lg font-semibold mb-4 ${isDark ? 'text-white' : 'text-slate-900'}`}>Cost Estimator</h3>
      <div className="mb-4 space-y-3">
        <div>
          <label className={`block text-sm font-medium mb-2 ${isDark ? 'text-slate-300' : 'text-slate-700'}`}>From address</label>
          <input
            value={from}
            onChange={(e) => setFrom(e.target.value.trim())}
            placeholder="0x..."
            className={`w-full px-4 py-2 rounded-lg border font-mono text-sm ${isDark ? 'border-slate-600 bg-slate-700 text-white' : 'border-slate-300 bg-white text-slate-900'}`}
          />
        </div>
        <div>
          <label className={`block text-sm font-medium mb-2 ${isDark ? 'text-slate-300' : 'text-slate-700'}`}>To address</label>
          <input
            value={to}
            onChange={(e) => setTo(e.target.value.trim())}
            placeholder="0x..."
            className={`w-full px-4 py-2 rounded-lg border font-mono text-sm ${isDark ? 'border-slate-600 bg-slate-700 text-white' : 'border-slate-300 bg-white text-slate-900'}`}
          />
        </div>
      </div>
      {loading ? (
        <div className="text-center py-4"><div className="animate-spin rounded-full h-8 w-8 border-b-2 border-orange-500 mx-auto"></div></div>
      ) : estimate ? (
        <div className="space-y-3">
          <div className="flex justify-between">
            <span className={isDark ? 'text-slate-400' : 'text-slate-500'}>Gas Limit</span>
            <span className={`font-medium ${isDark ? 'text-white' : 'text-slate-900'}`}>{estimate.gas_limit.toLocaleString()}</span>
          </div>
          <div className="flex justify-between">
            <span className={isDark ? 'text-slate-400' : 'text-slate-500'}>Max fee per gas</span>
            <span className={`font-medium ${isDark ? 'text-white' : 'text-slate-900'}`}>{formatGwei(estimate.fee?.max_fee_per_gas)}</span>
          </div>
          <div className={`pt-3 border-t ${isDark ? 'border-slate-700' : 'border-slate-200'}`}>
            <div className="flex justify-between">
              <span className={`font-medium ${isDark ? 'text-slate-300' : 'text-slate-700'}`}>Total cost (wei)</span>
              <span className={`text-sm font-bold ${isDark ? 'text-orange-400' : 'text-orange-600'}`}>{estimate.estimated_cost_wei ?? '-'}</span>
            </div>
          </div>
        </div>
      ) : (
        <div className="text-center text-slate-400 text-sm">
          {error ?? 'Enter both addresses to estimate'}
        </div>
      )}
    </div>
  );
}

export default function GasTrackerPage() {
  const { isDark } = useTheme();
  const [prices, setPrices] = useState<PriceMap>({});
  const [selectedChain, setSelectedChain] = useState<number | null>(null);
  const [loading, setLoading] = useState(true);
  const [lastUpdate, setLastUpdate] = useState<Date | null>(null);

  const loadPrices = useCallback(async () => {
    setLoading(true);
    const entries = await Promise.all(
      CHAIN_CONFIG.map(async (chain) => {
        const [gas, usd] = await Promise.all([
          fetchGasPrice(chain.id),
          fetchNativePriceUsd(chain.coin),
        ]);
        return [chain.id, { gas, usd }] as const;
      })
    );
    setPrices(Object.fromEntries(entries) as PriceMap);
    setLastUpdate(new Date());
    setLoading(false);
  }, []);

  useEffect(() => {
    loadPrices();
    const interval = setInterval(loadPrices, 30000);
    return () => clearInterval(interval);
  }, [loadPrices]);

  return (
    <div className={`min-h-screen bg-gradient-to-br ${isDark ? 'from-slate-900 to-slate-800' : 'from-slate-50 to-slate-100'}`}>
      <header className={`border-b sticky top-0 z-10 ${isDark ? 'bg-slate-800 border-slate-700' : 'bg-white border-slate-200'}`}>
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center gap-3">
              <a href="/wallet" className="text-2xl">🐯</a>
              <div>
                <h1 className={`text-xl font-bold ${isDark ? 'text-white' : 'text-slate-900'}`}>Gas Tracker</h1>
                <p className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>Real-time gas prices across all chains</p>
              </div>
            </div>
            <div className="flex items-center gap-4">
              {lastUpdate && <span className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>Updated: {lastUpdate.toLocaleTimeString()}</span>}
              <button onClick={loadPrices} disabled={loading} className="px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white rounded-lg font-medium disabled:opacity-50">
                {loading ? 'Refreshing...' : 'Refresh'}
              </button>
            </div>
          </div>
        </div>
      </header>
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-8">
          <div className={`rounded-xl p-6 border ${isDark ? 'bg-slate-800 border-slate-700' : 'bg-white border-slate-200'}`}>
            <div className={`text-3xl font-bold ${isDark ? 'text-white' : 'text-slate-900'}`}>{CHAIN_CONFIG.length}</div>
            <div className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>Supported Chains</div>
          </div>
          <div className={`rounded-xl p-6 border ${isDark ? 'bg-slate-800 border-slate-700' : 'bg-white border-slate-200'}`}>
            <div className="text-3xl font-bold text-green-500">{Object.values(prices).filter((p) => p.gas).length}</div>
            <div className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>Active Networks</div>
          </div>
          <div className={`rounded-xl p-6 border ${isDark ? 'bg-slate-800 border-slate-700' : 'bg-white border-slate-200'}`}>
            <div className="text-3xl font-bold text-orange-500">{lastUpdate ? '30s' : '--'}</div>
            <div className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>Update Interval</div>
          </div>
        </div>
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
          <div className="lg:col-span-2">
            <h2 className={`text-lg font-semibold mb-4 ${isDark ? 'text-white' : 'text-slate-900'}`}>Gas Prices by Chain</h2>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {CHAIN_CONFIG.map((chain) => (
                <ChainCard key={chain.id} chain={chain} gasPrice={prices[chain.id]?.gas ?? null} usd={prices[chain.id]?.usd ?? null} onSelect={() => setSelectedChain(chain.id)} isSelected={selectedChain === chain.id} />
              ))}
            </div>
          </div>
          <div>
            <CostEstimator selectedChain={selectedChain} />
            <div className={`mt-6 rounded-xl p-6 border ${isDark ? 'bg-slate-800 border-slate-700' : 'bg-white border-slate-200'}`}>
              <h3 className={`text-lg font-semibold mb-4 ${isDark ? 'text-white' : 'text-slate-900'}`}>Gas Saving Tips</h3>
              <ul className={`space-y-3 text-sm ${isDark ? 'text-slate-300' : 'text-slate-600'}`}>
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
