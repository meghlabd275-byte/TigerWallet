'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { useTheme } from '../components/ThemeProvider';

const API_BASE_URL = typeof window !== 'undefined' ? '' : (process.env.BACKEND_URL || 'http://localhost:8443');

const fetchAPI = async <T,>(endpoint: string, options?: RequestInit): Promise<T> => {
  const token = typeof window !== 'undefined' ? localStorage.getItem('tigerwallet-token') : null;
  const response = await fetch(`${API_BASE_URL}/api/v1${endpoint}`, {
    ...options,
    headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}), ...options?.headers },
  });
  if (!response.ok) throw new Error(`API Error: ${response.statusText}`);
  const data = await response.json();
  return data.data;
};

interface TokenSale {
  id: string;
  name: string;
  symbol: string;
  description: string;
  salePrice: number;
  listingPrice: number;
  totalSupply: number;
  forSale: number;
  sold: number;
  startTime: number;
  endTime: number;
  status: 'upcoming' | 'sale' | 'ended';
  phases: {
    name: string;
    price: number;
    discount: number;
    allocation: number;
    startTime: number;
    endTime: number;
  }[];
  chain: string;
  logo: string;
  progress: number;
}

interface BackendTokenSale {
  id: string;
  token_name: string;
  token_symbol: string;
  contract_address: string;
  chain_id: number;
  price_per_token: string;
  total_supply: string;
  sold_amount: string;
  min_allocation: string;
  max_allocation: string;
  start_time: number;
  end_time: number;
  status: string;
  description?: string;
  website?: string;
}

const CHAIN_NAMES: Record<number, string> = { 1: 'Ethereum', 56: 'BNB Chain', 137: 'Polygon', 8453: 'Base', 42161: 'Arbitrum' };

function mapSale(s: BackendTokenSale): TokenSale {
  const now = Date.now();
  const startMs = s.start_time * 1000;
  const endMs = s.end_time * 1000;
  let status: TokenSale['status'] = 'upcoming';
  if (now >= startMs && now < endMs) status = 'sale';
  else if (now >= endMs) status = 'ended';
  const salePrice = parseFloat(s.price_per_token) || 0;
  const forSale = parseFloat(s.total_supply) || 0;
  const sold = parseFloat(s.sold_amount) || 0;
  const progress = forSale > 0 ? Math.min(100, (sold / forSale) * 100) : 0;
  return {
    id: s.id,
    name: s.token_name,
    symbol: s.token_symbol,
    description: s.description || '',
    salePrice,
    listingPrice: salePrice * 1.5,
    totalSupply: forSale,
    forSale,
    sold,
    startTime: startMs,
    endTime: endMs,
    status,
    phases: [
      { name: 'Public', price: salePrice, discount: 0, allocation: forSale, startTime: startMs, endTime: endMs },
    ],
    chain: CHAIN_NAMES[s.chain_id] || `Chain ${s.chain_id}`,
    logo: '🪙',
    progress,
  };
}

export default function TokenSalePage() {
  const [activeTab, setActiveTab] = useState<'upcoming' | 'sale' | 'ended'>('sale');
  const [selectedSale, setSelectedSale] = useState<TokenSale | null>(null);
  const [buyAmount, setBuyAmount] = useState('');
  const [selectedPhase, setSelectedPhase] = useState(0);
  const [loading, setLoading] = useState(false);
  const [loadingSales, setLoadingSales] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [sales, setSales] = useState<TokenSale[]>([]);
  const { isDark } = useTheme();

  const loadSales = useCallback(async () => {
    setLoadingSales(true);
    setError(null);
    try {
      const data = await fetchAPI<BackendTokenSale[]>('/token-sales');
      setSales((data || []).map(mapSale));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load token sales');
      setSales([]);
    } finally {
      setLoadingSales(false);
    }
  }, []);

  useEffect(() => {
    loadSales();
  }, [loadSales]);

  const filteredSales = sales.filter(s => s.status === activeTab);

  const handleBuy = async () => {
    if (!selectedSale || !buyAmount) return;
    setLoading(true);
    setError(null);
    try {
      const usd = parseFloat(buyAmount);
      const price = selectedSale.phases[selectedPhase]?.price || selectedSale.salePrice;
      const tokenAmount = (usd / price).toString();
      await fetchAPI(`/token-sales/${selectedSale.id}/participate`, {
        method: 'POST',
        body: JSON.stringify({ amount: tokenAmount, cost: buyAmount }),
      });
      alert(`Successfully purchased ${parseFloat(tokenAmount).toFixed(2)} ${selectedSale.symbol}!`);
      setBuyAmount('');
      setSelectedSale(null);
      await loadSales();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to participate');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className={`min-h-screen ${isDark ? 'bg-gray-900 text-white' : 'bg-gray-50 text-gray-900'}`}>
      <header className="bg-gradient-to-r from-indigo-600 to-purple-600 text-white">
        <div className="max-w-7xl mx-auto px-4 py-6">
          <div className="flex items-center gap-4">
            <a href="/" className="text-3xl">🐯</a>
            <div>
              <h1 className="text-2xl font-bold">Token Sale</h1>
              <p className="text-indigo-200">Exclusive token sales at best prices</p>
            </div>
          </div>
        </div>
      </header>

      <div className="max-w-7xl mx-auto px-4 py-6">
        <div className="flex gap-2 mb-6">
          {(['upcoming', 'sale', 'ended'] as const).map(tab => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={`px-6 py-2 rounded-lg font-medium ${activeTab === tab ? 'bg-indigo-600 text-white' : isDark ? 'bg-gray-800' : 'bg-white border border-gray-200'}`}
            >
              {tab.charAt(0).toUpperCase() + tab.slice(1)}
            </button>
          ))}
        </div>

        <div className="space-y-4">
          {error && (
            <div className="p-3 rounded-lg bg-red-600 text-white">Error: {error}</div>
          )}
          {loadingSales ? (
            <div className={`text-center py-12 animate-pulse ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Loading token sales…</div>
          ) : filteredSales.length === 0 ? (
            <div className={`text-center py-12 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>No {activeTab} sales</div>
          ) : (
            <>
          {filteredSales.map(sale => (
            <div key={sale.id} className={`${isDark ? 'bg-gray-800' : 'bg-white'} rounded-xl p-6 border ${isDark ? 'border-gray-700' : 'border-gray-200'}`}>
              <div className="flex items-start justify-between">
                <div className="flex items-center gap-4">
                  <span className="text-5xl">{sale.logo}</span>
                  <div>
                    <h3 className="text-xl font-bold">{sale.name}</h3>
                    <p className={isDark ? 'text-gray-400' : 'text-gray-500'}>{sale.symbol} • {sale.chain}</p>
                  </div>
                </div>
                <span className={`px-3 py-1 rounded-full text-sm ${
                  sale.status === 'sale' ? 'bg-green-100 text-green-800' :
                  sale.status === 'upcoming' ? 'bg-blue-100 text-blue-800' :
                  'bg-gray-100 text-gray-800'
                }`}>
                  {sale.status.toUpperCase()}
                </span>
              </div>

              <p className="text-slate-500 mt-4">{sale.description}</p>

              <div className="grid grid-cols-4 gap-4 mt-6">
                <div>
                  <p className="text-xs text-slate-500">Current Price</p>
                  <p className="font-bold text-lg">${sale.phases[selectedPhase]?.price || sale.salePrice}</p>
                </div>
                <div>
                  <p className="text-xs text-slate-500">Listing Price</p>
                  <p className="font-bold">${sale.listingPrice}</p>
                </div>
                <div>
                  <p className="text-xs text-slate-500">Sold</p>
                  <p className="font-bold">{sale.sold.toLocaleString()}</p>
                </div>
                <div>
                  <p className="text-xs text-slate-500">Supply</p>
                  <p className="font-bold">{sale.totalSupply.toLocaleString()}</p>
                </div>
              </div>

              {sale.status !== 'ended' && (
                <div className="mt-6">
                  <p className="text-sm font-medium mb-2">Sale Phases</p>
                  <div className="flex gap-2 mb-4">
                    {sale.phases.map((phase, idx) => (
                      <button
                        key={idx}
                        onClick={() => setSelectedPhase(idx)}
                        className={`flex-1 p-2 rounded-lg border ${selectedPhase === idx ? 'bg-indigo-50 border-indigo-500' : 'bg-slate-50'}`}
                      >
                        <p className="font-medium text-sm">{phase.name}</p>
                        <p className="text-xs text-slate-500">${phase.price} {phase.discount > 0 && `(-${phase.discount}%)`}</p>
                      </button>
                    ))}
                  </div>
                </div>
              )}

              {sale.status === 'sale' && (
                <div className="mt-6">
                  <div className="h-3 bg-slate-200 rounded-full overflow-hidden">
                    <div 
                      className="h-full bg-gradient-to-r from-indigo-500 to-purple-500"
                      style={{ width: `${sale.progress}%` }}
                    />
                  </div>
                  <div className="flex justify-between mt-2 text-sm text-slate-500">
                    <span>{sale.sold.toLocaleString()} sold</span>
                    <span>{sale.forSale.toLocaleString()} total</span>
                  </div>
                  <button
                    onClick={() => { setSelectedSale(sale); setSelectedPhase(0); }}
                    className="w-full mt-4 py-3 bg-gradient-to-r from-indigo-600 to-purple-600 text-white rounded-lg font-semibold"
                  >
                    Buy Tokens
                  </button>
                </div>
              )}
            </div>
          ))}
            </>
          )}
        </div>
      </div>

      {selectedSale && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className={`${isDark ? 'bg-gray-800' : 'bg-white border border-gray-200'} rounded-xl p-6 max-w-md`}>
            <h3 className="text-xl font-bold mb-4">Buy {selectedSale.symbol}</h3>
            <div className="space-y-4">
              <div>
                <label className="block text-sm mb-2">Amount (USD)</label>
                <input
                  type="number"
                  value={buyAmount}
                  onChange={(e) => setBuyAmount(e.target.value)}
                  className="w-full p-3 border rounded-lg"
                />
              </div>
              <div className="p-3 bg-slate-50 rounded-lg">
                <p className="text-sm">You will receive: {buyAmount ? (parseFloat(buyAmount) / selectedSale.phases[selectedPhase].price).toFixed(2) : '0'} {selectedSale.symbol}</p>
                <p className="text-xs text-slate-500 mt-1">Listing at: ${selectedSale.listingPrice} (+{((selectedSale.listingPrice - selectedSale.phases[selectedPhase].price) / selectedSale.phases[selectedPhase].price * 100).toFixed(0)}%)</p>
              </div>
              <div className="flex gap-4">
                <button onClick={() => setSelectedSale(null)} className="flex-1 py-3 bg-slate-200 rounded-lg">Cancel</button>
                <button onClick={handleBuy} disabled={loading} className="flex-1 py-3 bg-indigo-600 text-white rounded-lg">
                  {loading ? 'Processing...' : 'Confirm'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
