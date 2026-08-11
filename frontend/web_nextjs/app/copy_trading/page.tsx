'use client';

import React, { useState } from 'react';
import { useTheme } from '../components/ThemeProvider';

// Types
interface Trader {
  address: string;
  totalTrades: number;
  successRate: number;
  totalPnL: number;
  monthlyPnL: number;
  followers: number;
  isFollowing: boolean;
  avatar?: string;
}

interface Signal {
  id: string;
  trader: string;
  tokenA: string;
  tokenB: string;
  action: 'BUY' | 'SELL';
  amount: string;
  price: string;
  timestamp: number;
  status: 'active' | 'closed';
  pnl?: number;
}

interface Position {
  id: string;
  signal: Signal;
  amount: string;
  entryPrice: string;
  currentPrice: string;
  pnl: number;
  pnlPercent: number;
  status: 'open' | 'closed';
  openedAt: number;
  closedAt?: number;
}

export default function CopyTrading() {
  const { isDark } = useTheme();
  const [activeTab, setActiveTab] = useState<'traders' | 'signals' | 'portfolio'>('traders');
  const [traders] = useState<Trader[]>([]);
  const [signals] = useState<Signal[]>([]);
  const [positions, setPositions] = useState<Position[]>([]);
  const [selectedTrader, setSelectedTrader] = useState<Trader | null>(null);
  const [copyAmount, setCopyAmount] = useState('');
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  const handleFollow = async (trader: Trader) => {
    setMessage({ type: 'error', text: 'Following traders is unavailable until an authenticated copy-trading API is configured.' });
  };

  const handleCopyTrade = async (signal: Signal) => {
    if (!copyAmount || parseFloat(copyAmount) <= 0) {
      setMessage({ type: 'error', text: 'Please enter a valid amount' });
      return;
    }
    
    setMessage({ type: 'error', text: 'Copy execution is unavailable until an authenticated execution provider is configured.' });
  };

  const formatAddress = (addr: string) => {
    return addr.slice(0, 6) + '...' + addr.slice(-4);
  };

  const formatTime = (timestamp: number) => {
    const diff = Date.now() - timestamp;
    const hours = Math.floor(diff / 3600000);
    const minutes = Math.floor(diff / 60000);
    
    if (hours > 24) {
      return `${Math.floor(hours / 24)}d ago`;
    }
    if (hours > 0) {
      return `${hours}h ago`;
    }
    return `${minutes}m ago`;
  };

  const formatCurrency = (value: number) => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    }).format(value);
  };

  const totalPortfolioValue = positions.reduce((acc, p) => acc + parseFloat(p.amount) * parseFloat(p.currentPrice), 0);
  const totalPnL = positions.reduce((acc, p) => acc + p.pnl, 0);

  return (
    <div className={`min-h-screen ${isDark ? 'bg-slate-900 text-slate-50' : 'bg-slate-50 text-slate-900'}`}>
      {/* Header */}
      <header className={`border-b ${isDark ? 'bg-slate-800 border-gray-700' : 'bg-white border-gray-200'}`}>
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center gap-4">
              <a href="/" className="text-2xl">🐯</a>
              <h1 className="text-xl font-bold">Copy Trading</h1>
            </div>
            <nav className="flex gap-4">
              <a href="/wallet" className={`${isDark ? 'text-gray-400' : 'text-gray-600'} hover:text-orange-500`}>Wallet</a>
              <a href="/swap" className={`${isDark ? 'text-gray-400' : 'text-gray-600'} hover:text-orange-500`}>Swap</a>
              <a href="/portfolio" className={`${isDark ? 'text-gray-400' : 'text-gray-600'} hover:text-orange-500`}>Portfolio</a>
            </nav>
          </div>
        </div>
      </header>

      {/* Message */}
      {message && (
        <div className="max-w-7xl mx-auto px-4 pt-4">
          <div className={`p-3 rounded-lg ${message.type === 'success' ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'}`}>
            {message.text}
          </div>
        </div>
      )}

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Stats */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-8">
          <div className={`rounded-lg p-6 shadow-sm ${isDark ? 'bg-slate-800' : 'bg-white'}`}>
            <div className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Total Portfolio</div>
            <div className="text-2xl font-bold text-orange-500">{formatCurrency(totalPortfolioValue || 0)}</div>
          </div>
          <div className={`rounded-lg p-6 shadow-sm ${isDark ? 'bg-slate-800' : 'bg-white'}`}>
            <div className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Total P&L</div>
            <div className={`text-2xl font-bold ${totalPnL >= 0 ? 'text-green-500' : 'text-red-500'}`}>
              {formatCurrency(totalPnL)}
            </div>
          </div>
          <div className={`rounded-lg p-6 shadow-sm ${isDark ? 'bg-slate-800' : 'bg-white'}`}>
            <div className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Open Positions</div>
            <div className="text-2xl font-bold">{positions.filter(p => p.status === 'open').length}</div>
          </div>
          <div className={`rounded-lg p-6 shadow-sm ${isDark ? 'bg-slate-800' : 'bg-white'}`}>
            <div className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Following</div>
            <div className="text-2xl font-bold">{traders.filter(t => t.isFollowing).length}</div>
          </div>
        </div>

        {/* Tabs */}
        <div className={`flex border-b mb-6 ${isDark ? 'border-gray-700' : 'border-gray-200'}`}>
          <button
            onClick={() => setActiveTab('traders')}
            className={`px-4 py-2 ${activeTab === 'traders' ? 'border-b-2 border-orange-500 text-orange-500' : isDark ? 'text-gray-400' : 'text-gray-500'}`}
          >
            Top Traders
          </button>
          <button
            onClick={() => setActiveTab('signals')}
            className={`px-4 py-2 ${activeTab === 'signals' ? 'border-b-2 border-orange-500 text-orange-500' : isDark ? 'text-gray-400' : 'text-gray-500'}`}
          >
            Live Signals
          </button>
          <button
            onClick={() => setActiveTab('portfolio')}
            className={`px-4 py-2 ${activeTab === 'portfolio' ? 'border-b-2 border-orange-500 text-orange-500' : isDark ? 'text-gray-400' : 'text-gray-500'}`}
          >
            My Portfolio
          </button>
        </div>

        {/* Traders Tab */}
        {activeTab === 'traders' && (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {traders.map((trader) => (
              <div key={trader.address} className={`rounded-lg p-6 shadow-sm ${isDark ? 'bg-slate-800' : 'bg-white'}`}>
                <div className="flex items-start justify-between mb-4">
                  <div>
                    <div className={`font-mono text-sm px-2 py-1 rounded ${isDark ? 'bg-slate-700' : 'bg-slate-100'}`}>
                      {formatAddress(trader.address)}
                    </div>
                    <div className={`text-sm mt-1 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
                      {trader.totalTrades} trades
                    </div>
                  </div>
                  <span className={`px-2 py-1 rounded text-xs ${trader.successRate >= 0.7 ? 'bg-green-100 text-green-800' : 'bg-yellow-100 text-yellow-800'}`}>
                    {(trader.successRate * 100).toFixed(0)}% SR
                  </span>
                </div>
                
                <div className="grid grid-cols-2 gap-4 mb-4">
                  <div>
                    <div className={`text-xs ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Total P&L</div>
                    <div className={`font-bold ${trader.totalPnL >= 0 ? 'text-green-500' : 'text-red-500'}`}>
                      {formatCurrency(trader.totalPnL)} ETH
                    </div>
                  </div>
                  <div>
                    <div className={`text-xs ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Monthly P&L</div>
                    <div className={`font-bold ${trader.monthlyPnL >= 0 ? 'text-green-500' : 'text-red-500'}`}>
                      {formatCurrency(trader.monthlyPnL)} ETH
                    </div>
                  </div>
                </div>
                
                <div className={`text-sm mb-4 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
                  👥 {trader.followers.toLocaleString()} followers
                </div>
                
                <button
                  onClick={() => handleFollow(trader)}
                  disabled={loading}
                  className={`w-full py-2 px-4 rounded-lg transition-colors ${
                    trader.isFollowing
                      ? isDark ? 'bg-slate-700 text-slate-300 hover:bg-slate-600' : 'bg-slate-200 text-slate-700 hover:bg-slate-300'
                      : 'bg-orange-500 text-white hover:bg-orange-600'
                  }`}
                >
                  {trader.isFollowing ? 'Following' : 'Follow'}
                </button>
              </div>
            ))}
          </div>
        )}

        {/* Signals Tab */}
        {activeTab === 'signals' && (
          <div className="space-y-4">
            {signals.map((signal) => (
              <div key={signal.id} className={`rounded-lg p-6 shadow-sm ${isDark ? 'bg-slate-800' : 'bg-white'}`}>
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-4">
                    <div className={`w-10 h-10 rounded-full flex items-center justify-center ${
                      signal.action === 'BUY' ? 'bg-green-100 text-green-600' : 'bg-red-100 text-red-600'
                    }`}>
                      {signal.action === 'BUY' ? '↑' : '↓'}
                    </div>
                    <div>
                      <div className="font-semibold">
                        {signal.tokenA}/{signal.tokenB}
                      </div>
                      <div className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
                        {signal.action} • {signal.amount} {signal.tokenA} @ {formatCurrency(parseFloat(signal.price))}
                      </div>
                    </div>
                  </div>
                  
                  <div className="text-right">
                    <div className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
                      {formatTime(signal.timestamp)}
                    </div>
                    {signal.status === 'closed' && signal.pnl !== undefined && (
                      <div className={`font-bold ${signal.pnl >= 0 ? 'text-green-500' : 'text-red-500'}`}>
                        {signal.pnl >= 0 ? '+' : ''}{signal.pnl.toFixed(2)}%
                      </div>
                    )}
                  </div>
                </div>
                
                {signal.status === 'active' && (
                  <div className={`mt-4 pt-4 border-t ${isDark ? 'border-gray-700' : 'border-gray-200'}`}>
                    <div className="flex gap-2">
                      <input
                        type="number"
                        placeholder="Amount to copy"
                        value={copyAmount}
                        onChange={(e) => setCopyAmount(e.target.value)}
                        className={`flex-1 border-0 rounded-lg px-4 py-2 ${isDark ? 'bg-slate-700' : 'bg-slate-100'}`}
                      />
                      <button
                        onClick={() => handleCopyTrade(signal)}
                        disabled={loading}
                        className="bg-orange-500 hover:bg-orange-600 disabled:bg-slate-400 text-white px-6 py-2 rounded-lg transition-colors"
                      >
                        Copy Trade
                      </button>
                    </div>
                  </div>
                )}
              </div>
            ))}
          </div>
        )}

        {/* Portfolio Tab */}
        {activeTab === 'portfolio' && (
          <div>
            {positions.length === 0 ? (
              <div className="text-center py-12">
                <div className="text-6xl mb-4">📊</div>
                <h3 className="text-xl font-semibold mb-2">No Positions Yet</h3>
                <p className={`${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Follow traders and copy their signals to get started</p>
                <button
                  onClick={() => setActiveTab('traders')}
                  className="mt-4 bg-orange-500 hover:bg-orange-600 text-white px-6 py-2 rounded-lg transition-colors"
                >
                  Find Traders
                </button>
              </div>
            ) : (
              <div className="space-y-4">
                {positions.map((position) => (
                  <div key={position.id} className={`rounded-lg p-6 shadow-sm ${isDark ? 'bg-slate-800' : 'bg-white'}`}>
                    <div className="flex items-start justify-between">
                      <div>
                        <div className="font-semibold text-lg">
                          {position.signal.tokenA}/{position.signal.tokenB}
                        </div>
                        <div className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
                          {position.signal.action} • {position.amount} {position.signal.tokenA}
                        </div>
                      </div>
                      <div className="text-right">
                        <div className={`text-xl font-bold ${position.pnl >= 0 ? 'text-green-500' : 'text-red-500'}`}>
                          {position.pnl >= 0 ? '+' : ''}{position.pnlPercent.toFixed(2)}%
                        </div>
                        <div className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
                          {formatCurrency(position.pnl)}
                        </div>
                      </div>
                    </div>
                    
                    <div className={`grid grid-cols-3 gap-4 mt-4 pt-4 border-t ${isDark ? 'border-gray-700' : 'border-gray-200'}`}>
                      <div>
                        <div className={`text-xs ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Entry Price</div>
                        <div className="font-semibold">{formatCurrency(parseFloat(position.entryPrice))}</div>
                      </div>
                      <div>
                        <div className={`text-xs ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Current Price</div>
                        <div className="font-semibold">{formatCurrency(parseFloat(position.currentPrice))}</div>
                      </div>
                      <div>
                        <div className={`text-xs ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Status</div>
                        <span className={`px-2 py-1 rounded text-xs ${position.status === 'open' ? 'bg-green-100 text-green-800' : isDark ? 'bg-slate-700 text-slate-200' : 'bg-slate-100 text-slate-800'}`}>
                          {position.status.toUpperCase()}
                        </span>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
