'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { useTheme } from '../components/ThemeProvider'

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

// Types
interface MarginPair {
  id: string;
  base: string;
  quote: string;
  symbol: string;
  price: number;
  change24h: number;
  volume24h: number;
  borrowable: number;
  interestRate: number;
}

interface MarginPosition {
  id: string;
  pair_id: string;
  pair_symbol: string;
  side: 'LONG' | 'SHORT';
  borrowed: string;
  collateral: string;
  leverage: string;
  entry_price: string;
  liq_price: string;
  interest_rate: string;
  unrealized_pnl: string;
  status: string;
  chain_id: number;
  opened_at: number;
  closed_at?: number;
}

interface MarginOrder {
  id: string;
  symbol: string;
  side: 'LONG' | 'SHORT';
  type: 'MARKET' | 'LIMIT' | 'STOP';
  size: number;
  price: number;
  filled: number;
  status: 'PENDING' | 'FILLED' | 'CANCELLED';
  leverage: number;
  marginMode: 'CROSS' | 'ISOLATED';
}

interface MarginAccount {
  totalAssets: number;
  totalLiabilities: number;
  netAssets: number;
  availableBalance: number;
  totalBorrowed: number;
  marginRatio: number;
  riskLevel: 'SAFE' | 'WARNING' | 'LIQUIDATION';
}

const TRADING_PAIRS: MarginPair[] = [
  { id: '1', base: 'BTC', quote: 'USDT', symbol: 'BTC/USDT', price: 43250, change24h: 2.5, volume24h: 125000000, borrowable: 50000000, interestRate: 0.0001 },
  { id: '2', base: 'ETH', quote: 'USDT', symbol: 'ETH/USDT', price: 2280, change24h: 1.8, volume24h: 85000000, borrowable: 25000000, interestRate: 0.0001 },
  { id: '3', base: 'BNB', quote: 'USDT', symbol: 'BNB/USDT', price: 312.5, change24h: -0.5, volume24h: 15000000, borrowable: 5000000, interestRate: 0.0001 },
  { id: '4', base: 'SOL', quote: 'USDT', symbol: 'SOL/USDT', price: 98.75, change24h: 3.2, volume24h: 8000000, borrowable: 3000000, interestRate: 0.0001 },
  { id: '5', base: 'XRP', quote: 'USDT', symbol: 'XRP/USDT', price: 0.62, change24h: -1.2, volume24h: 5000000, borrowable: 2000000, interestRate: 0.0001 },
  { id: '6', base: 'DOGE', quote: 'USDT', symbol: 'DOGE/USDT', price: 0.082, change24h: 5.5, volume24h: 3000000, borrowable: 1000000, interestRate: 0.0001 },
  { id: '7', base: 'ADA', quote: 'USDT', symbol: 'ADA/USDT', price: 0.58, change24h: 0.8, volume24h: 2500000, borrowable: 800000, interestRate: 0.0001 },
  { id: '8', base: 'AVAX', quote: 'USDT', symbol: 'AVAX/USDT', price: 38.2, change24h: -2.1, volume24h: 2000000, borrowable: 600000, interestRate: 0.0001 },
];

export default function MarginTradingPage() {
  const { isDark } = useTheme()
  const [activeTab, setActiveTab] = useState<'trade' | 'positions' | 'orders' | 'borrow'>('trade');
  const [selectedPair, setSelectedPair] = useState<MarginPair>(TRADING_PAIRS[0]);
  const [side, setSide] = useState<'LONG' | 'SHORT'>('LONG');
  const [leverage, setLeverage] = useState(10);
  const [marginMode, setMarginMode] = useState<'CROSS' | 'ISOLATED'>('CROSS');
  const [orderType, setOrderType] = useState<'MARKET' | 'LIMIT'>('MARKET');
  const [amount, setAmount] = useState('');
  const [price, setPrice] = useState('');
  const [positions, setPositions] = useState<MarginPosition[]>([]);
  const [orders, setOrders] = useState<MarginOrder[]>([]);
  const [loadingPositions, setLoadingPositions] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);
  const [account, setAccount] = useState<MarginAccount>({
    totalAssets: 50000,
    totalLiabilities: 5000,
    netAssets: 45000,
    availableBalance: 40000,
    totalBorrowed: 5000,
    marginRatio: 9.0,
    riskLevel: 'SAFE',
  });

  const loadPositions = useCallback(async () => {
    setLoadingPositions(true);
    setError(null);
    try {
      const data = await fetchAPI<MarginPosition[]>('/margin/positions?status=open');
      setPositions(data || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load positions');
      setPositions([]);
    } finally {
      setLoadingPositions(false);
    }
  }, []);

  useEffect(() => {
    loadPositions();
  }, [loadPositions]);

  const calculateLiquidationPrice = () => {
    const entryPrice = orderType === 'LIMIT' ? parseFloat(price) || selectedPair.price : selectedPair.price;
    const liquidationPercent = 1 / leverage;
    
    if (side === 'LONG') {
      return entryPrice * (1 - liquidationPercent);
    } else {
      return entryPrice * (1 + liquidationPercent);
    }
  };

  const liquidationPrice = calculateLiquidationPrice();

  const handleOpenPosition = async () => {
    const size = parseFloat(amount) || 0;
    if (size <= 0) {
      setMessage({ type: 'error', text: 'Please enter a valid amount' });
      return;
    }
    setSubmitting(true);
    setError(null);
    const entryPrice = orderType === 'LIMIT' ? parseFloat(price) || selectedPair.price : selectedPair.price;
    const collateral = (size * selectedPair.price / leverage).toString();
    const borrowed = Math.max(0, size * selectedPair.price - parseFloat(collateral)).toString();
    try {
      await fetchAPI('/margin/positions', {
        method: 'POST',
        body: JSON.stringify({
          pair_id: selectedPair.id,
          pair_symbol: selectedPair.symbol,
          side,
          borrowed,
          collateral,
          leverage: leverage.toString(),
          entry_price: entryPrice.toString(),
          liq_price: liquidationPrice.toString(),
          chain_id: 1,
        }),
      });
      setMessage({ type: 'success', text: `${side} position opened successfully!` });
      setAmount('');
      setPrice('');
      await loadPositions();
      setActiveTab('positions');
    } catch (err) {
      setMessage({ type: 'error', text: err instanceof Error ? err.message : 'Failed to open position' });
    } finally {
      setSubmitting(false);
    }
  };

  const handleClosePosition = async (positionId: string) => {
    setSubmitting(true);
    setError(null);
    try {
      await fetchAPI(`/margin/positions/${positionId}/close`, { method: 'POST' });
      setMessage({ type: 'success', text: 'Position closed successfully!' });
      await loadPositions();
    } catch (err) {
      setMessage({ type: 'error', text: err instanceof Error ? err.message : 'Failed to close position' });
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className={`'min-h-screen' ${isDark ? 'bg-gray-900' : 'bg-gray-50'} ${isDark ? 'text-white' : 'text-gray-900'} 'p-6'`}>
      <div className="max-w-7xl mx-auto">
        <div className="flex items-center justify-between mb-8">
          <div>
            <h1 className="text-3xl font-bold">Margin Trading</h1>
            <p className={`${isDark ? 'text-gray-400' : 'text-gray-500'} 'mt-1'`}>Trade with leverage up to 125x</p>
          </div>
          <div className="flex items-center space-x-4">
            <div className={`${isDark ? 'bg-gray-800' : 'bg-white'} 'px-4 py-2 rounded-lg'`}>
              <div className={`'text-sm' ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Net Assets</div>
              <div className="text-xl font-bold text-green-400">${account.netAssets.toLocaleString()}</div>
            </div>
            <div className={`${isDark ? 'bg-gray-800' : 'bg-white'} 'px-4 py-2 rounded-lg'`}>
              <div className={`'text-sm' ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Available</div>
              <div className="text-xl font-bold">${account.availableBalance.toLocaleString()}</div>
            </div>
            <div className={`${isDark ? 'bg-gray-800' : 'bg-white'} 'px-4 py-2 rounded-lg'`}>
              <div className={`'text-sm' ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Margin Ratio</div>
              <div className={`text-xl font-bold ${account.marginRatio < 1.5 ? 'text-red-400' : 'text-green-400'}`}>
                {account.marginRatio.toFixed(2)}x
              </div>
            </div>
          </div>
        </div>

        {message && (
          <div className={`mb-4 p-3 rounded-lg ${message.type === 'success' ? 'bg-green-600' : 'bg-red-600'} text-white`}>
            {message.text}
          </div>
        )}
        {error && !message && (
          <div className="mb-4 p-3 rounded-lg bg-red-600 text-white">Error: {error}</div>
        )}

        <div className={`'flex space-x-4 mb-6 border-b' ${isDark ? 'border-gray-700' : 'border-gray-200'}`}>
          {(['trade', 'positions', 'orders', 'borrow'] as const).map((tab) => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={`pb-3 px-4 font-medium capitalize ${activeTab === tab ? 'text-blue-400 border-b-2 border-blue-400' : 'text-gray-400 hover:text-white'}`}
            >
              {tab}
            </button>
          ))}
        </div>

        {activeTab === 'trade' && (
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            <div className={`'lg:col-span-2' ${isDark ? 'bg-gray-800' : 'bg-white'} 'rounded-xl p-6'`}>
              <div className="mb-6">
                <label className={`'block text-sm' ${isDark ? 'text-gray-400' : 'text-gray-500'} 'mb-2'`}>Trading Pair</label>
                <select
                  value={selectedPair.id}
                  onChange={(e) => setSelectedPair(TRADING_PAIRS.find(p => p.id === e.target.value) || TRADING_PAIRS[0])}
                  className={`'w-full' ${isDark ? 'bg-gray-700' : 'bg-gray-100'} 'border border-gray-600 rounded-lg px-4 py-3' ${isDark ? 'text-white' : 'text-gray-900'}`}
                >
                  {TRADING_PAIRS.map((pair) => (
                    <option key={pair.id} value={pair.id}>
                      {pair.symbol} - ${pair.price.toLocaleString()}
                    </option>
                  ))}
                </select>
              </div>

              <div className="grid grid-cols-3 gap-4 mb-6">
                <div className={`${isDark ? 'bg-gray-700' : 'bg-gray-100'} 'rounded-lg p-4'`}>
                  <div className={`'text-sm' ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Price</div>
                  <div className="text-xl font-bold">${selectedPair.price.toLocaleString()}</div>
                </div>
                <div className={`${isDark ? 'bg-gray-700' : 'bg-gray-100'} 'rounded-lg p-4'`}>
                  <div className={`'text-sm' ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>24h Change</div>
                  <div className={`text-xl font-bold ${selectedPair.change24h >= 0 ? 'text-green-400' : 'text-red-400'}`}>
                    {selectedPair.change24h >= 0 ? '+' : ''}{selectedPair.change24h}%
                  </div>
                </div>
                <div className={`${isDark ? 'bg-gray-700' : 'bg-gray-100'} 'rounded-lg p-4'`}>
                  <div className={`'text-sm' ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>24h Volume</div>
                  <div className="text-xl font-bold">${(selectedPair.volume24h / 1000000).toFixed(1)}M</div>
                </div>
              </div>

              <div className="flex space-x-4 mb-6">
                <button
                  onClick={() => setSide('LONG')}
                  className={`flex-1 py-3 rounded-lg font-bold ${side === 'LONG' ? 'bg-green-600 hover:bg-green-700' : 'bg-gray-700 hover:bg-gray-600'}`}
                >
                  LONG
                </button>
                <button
                  onClick={() => setSide('SHORT')}
                  className={`flex-1 py-3 rounded-lg font-bold ${side === 'SHORT' ? 'bg-red-600 hover:bg-red-700' : 'bg-gray-700 hover:bg-gray-600'}`}
                >
                  SHORT
                </button>
              </div>

              <div className="flex space-x-4 mb-6">
                <button
                  onClick={() => setMarginMode('CROSS')}
                  className={`flex-1 py-2 rounded-lg text-sm ${marginMode === 'CROSS' ? 'bg-blue-600' : 'bg-gray-700'}`}
                >
                  Cross Margin
                </button>
                <button
                  onClick={() => setMarginMode('ISOLATED')}
                  className={`flex-1 py-2 rounded-lg text-sm ${marginMode === 'ISOLATED' ? 'bg-blue-600' : 'bg-gray-700'}`}
                >
                  Isolated Margin
                </button>
              </div>

              <div className="mb-6">
                <div className="flex justify-between mb-2">
                  <label className={`'text-sm' ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Leverage</label>
                  <span className="font-bold">{leverage}x</span>
                </div>
                <input
                  type="range"
                  min="1"
                  max="125"
                  value={leverage}
                  onChange={(e) => setLeverage(parseInt(e.target.value))}
                  className="w-full"
                />
                <div className="flex justify-between text-xs text-gray-500 mt-1">
                  <span>1x</span><span>25x</span><span>50x</span><span>75x</span><span>100x</span><span>125x</span>
                </div>
              </div>

              <div className="flex space-x-4 mb-6">
                <button
                  onClick={() => setOrderType('MARKET')}
                  className={`flex-1 py-2 rounded-lg text-sm ${orderType === 'MARKET' ? 'bg-blue-600' : 'bg-gray-700'}`}
                >
                  Market
                </button>
                <button
                  onClick={() => setOrderType('LIMIT')}
                  className={`flex-1 py-2 rounded-lg text-sm ${orderType === 'LIMIT' ? 'bg-blue-600' : 'bg-gray-700'}`}
                >
                  Limit
                </button>
              </div>

              <div className="mb-6">
                <label className={`'block text-sm' ${isDark ? 'text-gray-400' : 'text-gray-500'} 'mb-2'`}>Amount ({selectedPair.base})</label>
                <input
                  type="number"
                  value={amount}
                  onChange={(e) => setAmount(e.target.value)}
                  placeholder="0.00"
                  className={`'w-full' ${isDark ? 'bg-gray-700' : 'bg-gray-100'} 'border border-gray-600 rounded-lg px-4 py-3' ${isDark ? 'text-white' : 'text-gray-900'}`}
                />
              </div>

              {orderType === 'LIMIT' && (
                <div className="mb-6">
                  <label className={`'block text-sm' ${isDark ? 'text-gray-400' : 'text-gray-500'} 'mb-2'`}>Limit Price (USDT)</label>
                  <input
                    type="number"
                    value={price}
                    onChange={(e) => setPrice(e.target.value)}
                    placeholder={selectedPair.price.toString()}
                    className={`'w-full' ${isDark ? 'bg-gray-700' : 'bg-gray-100'} 'border border-gray-600 rounded-lg px-4 py-3' ${isDark ? 'text-white' : 'text-gray-900'}`}
                  />
                </div>
              )}

              <div className={`${isDark ? 'bg-gray-700' : 'bg-gray-100'} 'rounded-lg p-4 mb-6'`}>
                <div className="flex justify-between mb-2">
                  <span className={`${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Position Value</span>
                  <span>${((parseFloat(amount) || 0) * selectedPair.price).toLocaleString()}</span>
                </div>
                <div className="flex justify-between mb-2">
                  <span className={`${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Required Margin</span>
                  <span>${((parseFloat(amount) || 0) * selectedPair.price / leverage).toLocaleString()}</span>
                </div>
                <div className="flex justify-between">
                  <span className={`${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Est. Liquidation Price</span>
                  <span className="text-red-400">${liquidationPrice.toLocaleString()}</span>
                </div>
              </div>

              <button
                onClick={handleOpenPosition}
                disabled={submitting}
                className={`w-full py-4 rounded-lg font-bold text-lg ${side === 'LONG' ? 'bg-green-600 hover:bg-green-700' : 'bg-red-600 hover:bg-red-700'} ${submitting ? 'opacity-50 cursor-not-allowed' : ''}`}
              >
                {submitting ? 'Submitting…' : side === 'LONG' ? 'Open Long' : 'Open Short'}
              </button>
            </div>

            <div className={`${isDark ? 'bg-gray-800' : 'bg-white'} 'rounded-xl p-6'`}>
              <h3 className="text-xl font-bold mb-4">Market Info</h3>
              <div className="space-y-4">
                {TRADING_PAIRS.map((pair) => (
                  <div
                    key={pair.id}
                    onClick={() => setSelectedPair(pair)}
                    className={`p-4 rounded-lg cursor-pointer ${selectedPair.id === pair.id ? 'bg-gray-700 border border-blue-500' : 'bg-gray-700 hover:bg-gray-600'}`}
                  >
                    <div className="flex justify-between items-center">
                      <div>
                        <div className="font-bold">{pair.symbol}</div>
                        <div className={`'text-sm' ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Borrowable: ${(pair.borrowable / 1000000).toFixed(1)}M</div>
                      </div>
                      <div className="text-right">
                        <div className="font-bold">${pair.price.toLocaleString()}</div>
                        <div className={`text-sm ${pair.change24h >= 0 ? 'text-green-400' : 'text-red-400'}`}>
                          {pair.change24h >= 0 ? '+' : ''}{pair.change24h}%
                        </div>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        )}

        {activeTab === 'positions' && (
          <div className={`${isDark ? 'bg-gray-800' : 'bg-white'} 'rounded-xl p-6'`}>
            <h3 className="text-xl font-bold mb-4">Open Positions</h3>
            {loadingPositions ? (
              <div className={`'text-center py-12 animate-pulse' ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Loading positions…</div>
            ) : positions.length === 0 ? (
              <div className={`'text-center py-12' ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>No open positions</div>
            ) : (
              <div className="space-y-4">
                {positions.map((pos) => (
                  <div key={pos.id} className={`${isDark ? 'bg-gray-700' : 'bg-gray-100'} 'rounded-lg p-4 flex justify-between items-center'`}>
                    <div>
                      <div className="font-bold">{pos.pair_symbol}</div>
                      <div className={`text-sm ${pos.side === 'LONG' ? 'text-green-400' : 'text-red-400'}`}>
                        {pos.side} • {pos.leverage}x • Collateral {pos.collateral}
                      </div>
                    </div>
                    <div className="text-right">
                      <div className="font-bold">{pos.borrowed} {pos.pair_symbol.split('/')[0]}</div>
                      <div className={`text-sm ${parseFloat(pos.unrealized_pnl || '0') >= 0 ? 'text-green-400' : 'text-red-400'}`}>
                        {parseFloat(pos.unrealized_pnl || '0') >= 0 ? '+' : ''}{pos.unrealized_pnl}
                      </div>
                    </div>
                    <button
                      onClick={() => handleClosePosition(pos.id)}
                      disabled={submitting}
                      className={`bg-red-600 px-4 py-2 rounded-lg hover:bg-red-700 ${submitting ? 'opacity-50 cursor-not-allowed' : ''}`}
                    >
                      Close
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {activeTab === 'orders' && (
          <div className={`${isDark ? 'bg-gray-800' : 'bg-white'} 'rounded-xl p-6'`}>
            <h3 className="text-xl font-bold mb-4">Order History</h3>
            <div className={`'text-center py-12' ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>No orders</div>
          </div>
        )}

        {activeTab === 'borrow' && (
          <div className={`${isDark ? 'bg-gray-800' : 'bg-white'} 'rounded-xl p-6'`}>
            <h3 className="text-xl font-bold mb-4">Borrow Assets</h3>
            <div className="space-y-4">
              {TRADING_PAIRS.map((pair) => (
                <div key={pair.id} className={`${isDark ? 'bg-gray-700' : 'bg-gray-100'} 'rounded-lg p-4 flex justify-between items-center'`}>
                  <div>
                    <div className="font-bold">{pair.quote}</div>
                    <div className={`'text-sm' ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Borrowable: ${pair.borrowable.toLocaleString()}</div>
                    <div className={`'text-sm' ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Interest: {pair.interestRate * 100}%/day</div>
                  </div>
                  <button className="bg-blue-600 px-4 py-2 rounded-lg hover:bg-blue-700">Borrow</button>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
