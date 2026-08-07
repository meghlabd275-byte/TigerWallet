'use client';

import React, { useState, useEffect, useCallback } from 'react';

interface PriceData {
  symbol: string;
  name: string;
  price: number;
  change24h: number;
  high24h: number;
  low24h: number;
  volume24h: number;
  marketCap: number;
  logo?: string;
}

interface ApiResponse<T> {
  success: boolean;
  data: T;
  error?: string;
}

interface MarketStats {
  totalMarketCap: string;
  totalVolume24h: string;
  btcDominance: number;
  activeTokens: number;
}

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'https://api.tigerwallet.io';

const fetchAPI = async <T,>(endpoint: string, options?: RequestInit): Promise<T> => {
  const token = typeof window !== 'undefined' ? localStorage.getItem('tigerwallet-token') : null;
  const response = await fetch(`${API_BASE_URL}${endpoint}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options?.headers,
    },
  });
  if (!response.ok) throw new Error(`API Error: ${response.statusText}`);
  const data: ApiResponse<T> = await response.json();
  return data.data;
};

export default function PriceFeeds() {
  const [prices, setPrices] = useState<PriceData[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [marketStats, setMarketStats] = useState<MarketStats | null>(null);
  const [error, setError] = useState<string | null>(null);

  const loadPrices = useCallback(async () => {
    try {
      const [priceData, stats] = await Promise.all([
        fetchAPI<PriceData[]>('/prices'),
        fetchAPI<MarketStats>('/prices/stats'),
      ]);
      if (!Array.isArray(priceData)) {
        throw new Error('Price feed response did not contain a valid data array')
      }
      setPrices(priceData);
      if (!stats) {
        throw new Error('Market statistics are unavailable')
      }
      setMarketStats(stats);
      setError(null);
    } catch (err) {
      setPrices([]);
      setMarketStats(null);
      setError('Live price data is unavailable because the market data API could not be reached.')
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadPrices();
    const interval = setInterval(() => {
      void loadPrices();
    }, 5000);
    return () => clearInterval(interval);
  }, [loadPrices]);

  const filteredPrices = searchQuery ? prices.filter(p => p.symbol.toLowerCase().includes(searchQuery.toLowerCase())) : prices;
  const formatCurrency = (v: number): string => v >= 1e9 ? '$' + (v/1e9).toFixed(2) + 'B' : v >= 1e6 ? '$' + (v/1e6).toFixed(2) + 'M' : v >= 1e3 ? '$' + (v/1e3).toFixed(2) + 'K' : v < 1 ? '$' + v.toFixed(4) : '$' + v.toFixed(2);
  const formatPrice = (p: number): string => p < 1 ? '$' + p.toFixed(4) : p < 100 ? '$' + p.toFixed(2) : '$' + p.toFixed(2);

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-900 text-slate-900 dark:text-slate-50">
      <header className="bg-white dark:bg-slate-800 border-b border-slate-200 dark:border-slate-700">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center gap-4"><a href="/" className="text-2xl">🐯</a><h1 className="text-xl font-bold">Real-Time Prices</h1></div>
            <div className="flex items-center gap-2"><span className="w-2 h-2 bg-green-500 rounded-full animate-pulse"></span><span className="text-sm text-green-500">Live</span></div>
          </div>
        </div>
      </header>
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <input type="text" placeholder="Search tokens..." value={searchQuery} onChange={(e) => setSearchQuery(e.target.value)} className="w-full bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg px-4 py-3 mb-6" />
        {loading ? <div className="text-center py-12"><div className="animate-spin w-12 h-12 border-4 border-orange-500 border-t-transparent rounded-full mx-auto"></div></div> : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
            {filteredPrices.map((token) => (
              <div key={token.symbol} className="bg-white dark:bg-slate-800 rounded-lg p-4 shadow-sm hover:shadow-md">
                <div className="flex items-center justify-between mb-3">
                  <div className="flex items-center gap-2"><div className="w-8 h-8 bg-gradient-to-br from-orange-400 to-orange-600 rounded-full flex items-center justify-center text-white font-bold text-sm">{token.symbol.slice(0,2)}</div><div><div className="font-semibold">{token.symbol}</div><div className="text-xs text-slate-500">USD</div></div></div>
                  <div className={`px-2 py-1 rounded text-xs font-medium ${token.change24h >= 0 ? 'bg-green-100 text-green-600' : 'bg-red-100 text-red-600'}`}>{token.change24h >= 0 ? '+' : ''}{token.change24h.toFixed(2)}%</div>
                </div>
                <div className="text-2xl font-bold mb-2">{formatPrice(token.price)}</div>
                <div className="grid grid-cols-2 gap-2 text-xs text-slate-500"><div>24h High: <span className="text-slate-700">{formatPrice(token.high24h)}</span></div><div>24h Low: <span className="text-slate-700">{formatPrice(token.low24h)}</span></div><div>Volume: <span className="text-slate-700">{formatCurrency(token.volume24h)}</span></div><div>MCap: <span className="text-slate-700">{formatCurrency(token.marketCap)}</span></div></div>
              </div>
            ))}
          </div>
        )}
        {marketStats && <div className="mt-8 grid grid-cols-2 md:grid-cols-4 gap-4">
          <div className="bg-white dark:bg-slate-800 rounded-lg p-4"><div className="text-slate-500 text-sm">Total Market Cap</div><div className="text-xl font-bold">{marketStats.totalMarketCap}</div></div>
          <div className="bg-white dark:bg-slate-800 rounded-lg p-4"><div className="text-slate-500 text-sm">24h Volume</div><div className="text-xl font-bold">{marketStats.totalVolume24h}</div></div>
          <div className="bg-white dark:bg-slate-800 rounded-lg p-4"><div className="text-slate-500 text-sm">BTC Dominance</div><div className="text-xl font-bold">{marketStats.btcDominance.toFixed(2)}%</div></div>
          <div className="bg-white dark:bg-slate-800 rounded-lg p-4"><div className="text-slate-500 text-sm">Active Tokens</div><div className="text-xl font-bold">{marketStats.activeTokens}</div></div>
        </div>}
      </div>
    </div>
  );
}
