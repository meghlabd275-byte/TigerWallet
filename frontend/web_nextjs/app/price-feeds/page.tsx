'use client';

import React, { useState, useEffect } from 'react';

interface PriceData {
  symbol: string;
  price: number;
  change24h: number;
  high24h: number;
  low24h: number;
  volume24h: number;
  marketCap: number;
}

export default function PriceFeeds() {
  const [prices, setPrices] = useState<PriceData[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');

  useEffect(() => {
    const mockPrices: PriceData[] = [
      { symbol: 'ETH', price: 3524.50, change24h: 2.35, high24h: 3580, low24h: 3420, volume24h: 12500000000, marketCap: 423000000000 },
      { symbol: 'BTC', price: 65234.20, change24h: 1.82, high24h: 66200, low24h: 64100, volume24h: 28000000000, marketCap: 1278000000000 },
      { symbol: 'BNB', price: 612.45, change24h: -0.54, high24h: 625, low24h: 605, volume24h: 1800000000, marketCap: 92000000000 },
      { symbol: 'SOL', price: 148.75, change24h: 5.23, high24h: 155, low24h: 142, volume24h: 3500000000, marketCap: 65000000000 },
      { symbol: 'TRX', price: 0.1245, change24h: 0.85, high24h: 0.128, low24h: 0.122, volume24h: 850000000, marketCap: 10800000000 },
      { symbol: 'MATIC', price: 0.825, change24h: -1.25, high24h: 0.85, low24h: 0.81, volume24h: 420000000, marketCap: 7800000000 },
      { symbol: 'AVAX', price: 38.45, change24h: 3.12, high24h: 39.50, low24h: 37.20, volume24h: 520000000, marketCap: 14200000000 },
      { symbol: 'DOT', price: 7.25, change24h: 1.45, high24h: 7.50, low24h: 7.05, volume24h: 280000000, marketCap: 9800000000 },
      { symbol: 'LINK', price: 15.82, change24h: 2.85, high24h: 16.20, low24h: 15.30, volume24h: 520000000, marketCap: 9200000000 },
      { symbol: 'UNI', price: 10.25, change24h: 1.92, high24h: 10.55, low24h: 9.95, volume24h: 180000000, marketCap: 6100000000 },
      { symbol: 'AAVE', price: 215.40, change24h: 4.25, high24h: 220, low24h: 205, volume24h: 150000000, marketCap: 3100000000 },
      { symbol: 'USDT', price: 1.00, change24h: 0.01, high24h: 1.001, low24h: 0.999, volume24h: 45000000000, marketCap: 83000000000 },
      { symbol: 'USDC', price: 1.00, change24h: 0.00, high24h: 1.001, low24h: 0.999, volume24h: 32000000000, marketCap: 42000000000 },
      { symbol: 'PI', price: 52.50, change24h: 1.25, high24h: 54, low24h: 51, volume24h: 120000000, marketCap: 0 },
      { symbol: 'TON', price: 5.85, change24h: -2.15, high24h: 6.10, low24h: 5.70, volume24h: 180000000, marketCap: 21000000000 },
      { symbol: 'DOGE', price: 0.158, change24h: 3.25, high24h: 0.165, low24h: 0.152, volume24h: 2800000000, marketCap: 23000000000 },
      { symbol: 'XRP', price: 0.625, change24h: 0.85, high24h: 0.64, low24h: 0.61, volume24h: 1500000000, marketCap: 34000000000 },
      { symbol: 'LTC', price: 88.50, change24h: 2.15, high24h: 91, low24h: 86, volume24h: 520000000, marketCap: 6600000000 },
      { symbol: 'BCH', price: 485.20, change24h: 1.75, high24h: 498, low24h: 475, volume24h: 380000000, marketCap: 9500000000 },
    ];
    setPrices(mockPrices);
    setLoading(false);
    const interval = setInterval(() => {
      setPrices(prev => prev.map(p => ({ ...p, price: p.price * (1 + (Math.random() - 0.5) * 0.001) })));
    }, 5000);
    return () => clearInterval(interval);
  }, []);

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
        <div className="mt-8 grid grid-cols-2 md:grid-cols-4 gap-4">
          <div className="bg-white dark:bg-slate-800 rounded-lg p-4"><div className="text-slate-500 text-sm">Total Market Cap</div><div className="text-xl font-bold">$2.45T</div></div>
          <div className="bg-white dark:bg-slate-800 rounded-lg p-4"><div className="text-slate-500 text-sm">24h Volume</div><div className="text-xl font-bold">$98.2B</div></div>
          <div className="bg-white dark:bg-slate-800 rounded-lg p-4"><div className="text-slate-500 text-sm">BTC Dominance</div><div className="text-xl font-bold">52.1%</div></div>
          <div className="bg-white dark:bg-slate-800 rounded-lg p-4"><div className="text-slate-500 text-sm">Active Tokens</div><div className="text-xl font-bold">{prices.length}+</div></div>
        </div>
      </div>
    </div>
  );
}
