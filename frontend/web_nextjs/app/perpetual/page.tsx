'use client';

import React, { useState, useEffect } from 'react';
import { useTheme } from '../components/ThemeProvider';

// Types
interface Position {
  id: string;
  pair: string;
  side: 'LONG' | 'SHORT';
  size: string;
  collateral: string;
  leverage: number;
  entryPrice: string;
  markPrice: string;
  pnl: number;
  pnlPercent: number;
  liquidationPrice: string;
  status: 'open' | 'closed' | 'liquidated';
  openedAt: number;
  closedAt?: number;
}

interface FundingRate {
  pair: string;
  rate: number;
  nextFunding: number;
}

const MOCK_POSITIONS: Position[] = [
  {
    id: 'pos_1',
    pair: 'ETH/USDT',
    side: 'LONG',
    size: '2.5',
    collateral: '0.5',
    leverage: 5,
    entryPrice: '3200.00',
    markPrice: '3500.00',
    pnl: 750,
    pnlPercent: 150,
    liquidationPrice: '2720.00',
    status: 'open',
    openedAt: Date.now() - 86400000 * 3,
  },
  {
    id: 'pos_2',
    pair: 'BTC/USDT',
    side: 'SHORT',
    size: '0.1',
    collateral: '1.0',
    leverage: 10,
    entryPrice: '68000.00',
    markPrice: '65000.00',
    pnl: 300,
    pnlPercent: 30,
    liquidationPrice: '74800.00',
    status: 'open',
    openedAt: Date.now() - 86400000,
  },
];

const FUNDING_RATES: FundingRate[] = [
  { pair: 'ETH/USDT', rate: 0.01, nextFunding: Date.now() + 28800000 },
  { pair: 'BTC/USDT', rate: 0.01, nextFunding: Date.now() + 28800000 },
  { pair: 'SOL/USDT', rate: 0.02, nextFunding: Date.now() + 28800000 },
  { pair: 'ARB/USDT', rate: 0.015, nextFunding: Date.now() + 28800000 },
];

const TRADING_PAIRS = [
  { symbol: 'ETH/USDT', price: 3500.00, change24h: 2.5, high24h: 3600, low24h: 3400, volume24h: 1250000000 },
  { symbol: 'BTC/USDT', price: 65000.00, change24h: 1.8, high24h: 66000, low24h: 64000, volume24h: 2500000000 },
  { symbol: 'SOL/USDT', price: 150.00, change24h: 5.2, high24h: 155, low24h: 142, volume24h: 850000000 },
  { symbol: 'ARB/USDT', price: 1.85, change24h: -1.2, high24h: 1.90, low24h: 1.80, volume24h: 180000000 },
];

export default function PerpetualTrading() {
  const { isDark } = useTheme();
  const [activeTab, setActiveTab] = useState<'trade' | 'positions' | 'history'>('trade');
  const [positions, setPositions] = useState<Position[]>(MOCK_POSITIONS);
  const [selectedPair, setSelectedPair] = useState(TRADING_PAIRS[0]);
  const [side, setSide] = useState<'LONG' | 'SHORT'>('LONG');
  const [orderType, setOrderType] = useState<'market' | 'limit'>('market');
  const [leverage, setLeverage] = useState(10);
  const [collateral, setCollateral] = useState('');
  const [limitPrice, setLimitPrice] = useState('');
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  const calculatePositionSize = () => {
    if (!collateral || parseFloat(collateral) <= 0) return '0';
    const size = parseFloat(collateral) * leverage;
    return size.toFixed(4);
  };

  const calculateLiquidationPrice = () => {
    if (!collateral || parseFloat(collateral) <= 0 || !selectedPair.price) return '0';
    
    const entryPrice = orderType === 'limit' && limitPrice ? parseFloat(limitPrice) : selectedPair.price;
    const size = parseFloat(collateral) * leverage;
    const maintenanceMargin = size * 0.005; // 0.5% maintenance margin
    
    if (side === 'LONG') {
      const liqPrice = entryPrice - (maintenanceMargin / parseFloat(collateral));
      return liqPrice.toFixed(2);
    } else {
      const liqPrice = entryPrice + (maintenanceMargin / parseFloat(collateral));
      return liqPrice.toFixed(2);
    }
  };

  const handleOpenPosition = async () => {
    if (!collateral || parseFloat(collateral) <= 0) {
      setMessage({ type: 'error', text: 'Please enter a valid collateral amount' });
      return;
    }

    setLoading(true);
    await new Promise(resolve => setTimeout(resolve, 1500));

    const entryPrice = orderType === 'limit' && limitPrice ? limitPrice : selectedPair.price.toString();
    const size = calculatePositionSize();

    const newPosition: Position = {
      id: `pos_${Date.now()}`,
      pair: selectedPair.symbol,
      side,
      size,
      collateral,
      leverage,
      entryPrice,
      markPrice: selectedPair.price.toString(),
      pnl: 0,
      pnlPercent: 0,
      liquidationPrice: calculateLiquidationPrice(),
      status: 'open',
      openedAt: Date.now(),
    };

    setPositions(prev => [...prev, newPosition]);
    setMessage({ type: 'success', text: `${side} position opened successfully!` });
    setCollateral('');
    setLoading(false);
    setActiveTab('positions');
  };

  const handleClosePosition = async (positionId: string) => {
    setLoading(true);
    await new Promise(resolve => setTimeout(resolve, 1000));

    setPositions(prev => prev.map(p => {
      if (p.id === positionId) {
        return {
          ...p,
          status: 'closed' as const,
          closedAt: Date.now(),
        };
      }
      return p;
    }));

    setMessage({ type: 'success', text: 'Position closed successfully!' });
    setLoading(false);
  };

  const formatCurrency = (value: number) => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    }).format(value);
  };

  const formatTime = (timestamp: number) => {
    return new Date(timestamp).toLocaleString();
  };

  const totalCollateral = positions.filter(p => p.status === 'open').reduce((acc, p) => acc + parseFloat(p.collateral), 0);
  const totalPnl = positions.filter(p => p.status === 'open').reduce((acc, p) => acc + p.pnl, 0);

  return (
    <div className={`min-h-screen ${isDark ? 'bg-slate-900 text-slate-50' : 'bg-slate-50 text-slate-900'}`}>
      {/* Header */}
      <header className={`${isDark ? 'bg-slate-800 border-slate-700' : 'bg-white border-slate-200'} border-b`}>
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center gap-4">
              <a href="/" className="text-2xl">🐯</a>
              <h1 className="text-xl font-bold">Perpetual Trading</h1>
            </div>
            <nav className="flex gap-4">
              <a href="/wallet" className={`${isDark ? 'text-slate-400' : 'text-slate-600'} hover:text-orange-500`}>Wallet</a>
              <a href="/swap" className={`${isDark ? 'text-slate-400' : 'text-slate-600'} hover:text-orange-500`}>Swap</a>
              <a href="/copy_trading" className={`${isDark ? 'text-slate-400' : 'text-slate-600'} hover:text-orange-500`}>Copy Trading</a>
            </nav>
          </div>
        </div>
      </header>

      {/* Message */}
      {message && (
        <div className="max-w-7xl mx-auto px-4 pt-4">
          <div className={`p-3 rounded-lg ${message.type === 'success' ? (isDark ? 'bg-green-900 text-green-200' : 'bg-green-100 text-green-800') : (isDark ? 'bg-red-900 text-red-200' : 'bg-red-100 text-red-800')}`}>
            {message.text}
          </div>
        </div>
      )}

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Stats */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-8">
          <div className={`${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'} rounded-lg p-6 shadow-sm`}>
            <div className={`${isDark ? 'text-slate-400' : 'text-slate-500'} text-sm`}>Total Collateral</div>
            <div className="text-2xl font-bold">{formatCurrency(totalCollateral)}</div>
          </div>
          <div className={`${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'} rounded-lg p-6 shadow-sm`}>
            <div className={`${isDark ? 'text-slate-400' : 'text-slate-500'} text-sm`}>Unrealized P&L</div>
            <div className={`text-2xl font-bold ${totalPnl >= 0 ? 'text-green-500' : 'text-red-500'}`}>
              {formatCurrency(totalPnl)}
            </div>
          </div>
          <div className={`${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'} rounded-lg p-6 shadow-sm`}>
            <div className={`${isDark ? 'text-slate-400' : 'text-slate-500'} text-sm`}>Open Positions</div>
            <div className="text-2xl font-bold">{positions.filter(p => p.status === 'open').length}</div>
          </div>
          <div className={`${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'} rounded-lg p-6 shadow-sm`}>
            <div className={`${isDark ? 'text-slate-400' : 'text-slate-500'} text-sm`}>Funding (24h)</div>
            <div className="text-2xl font-bold text-blue-500">$12,450</div>
          </div>
        </div>

        {/* Tabs */}
        <div className={`flex border-b mb-6 ${isDark ? 'border-slate-700' : 'border-slate-200'}`}>
          <button
            onClick={() => setActiveTab('trade')}
            className={`px-4 py-2 ${activeTab === 'trade' ? 'border-b-2 border-orange-500 text-orange-500' : isDark ? 'text-slate-400' : 'text-slate-500'}`}
          >
            Trade
          </button>
          <button
            onClick={() => setActiveTab('positions')}
            className={`px-4 py-2 ${activeTab === 'positions' ? 'border-b-2 border-orange-500 text-orange-500' : isDark ? 'text-slate-400' : 'text-slate-500'}`}
          >
            Positions
          </button>
          <button
            onClick={() => setActiveTab('history')}
            className={`px-4 py-2 ${activeTab === 'history' ? 'border-b-2 border-orange-500 text-orange-500' : isDark ? 'text-slate-400' : 'text-slate-500'}`}
          >
            History
          </button>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Trading Panel */}
          {activeTab === 'trade' && (
            <div className="lg:col-span-2 space-y-6">
              {/* Market Overview */}
              <div className={`${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'} rounded-lg p-6 shadow-sm`}>
                <h3 className="font-semibold mb-4">Markets</h3>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  {TRADING_PAIRS.map((pair) => (
                    <div
                      key={pair.symbol}
                      onClick={() => setSelectedPair(pair)}
                      className={`p-4 rounded-lg cursor-pointer transition-colors ${
                        selectedPair.symbol === pair.symbol
                          ? (isDark ? 'bg-orange-900 border border-orange-500' : 'bg-orange-100 border border-orange-500')
                          : (isDark ? 'bg-slate-700 hover:bg-slate-600' : 'bg-slate-100 hover:bg-slate-200')
                      }`}
                    >
                      <div className="flex justify-between items-start">
                        <div>
                          <div className="font-semibold">{pair.symbol}</div>
                          <div className="text-2xl font-bold">{formatCurrency(pair.price)}</div>
                        </div>
                        <span className={`px-2 py-1 rounded text-xs ${pair.change24h >= 0 ? (isDark ? 'bg-green-900 text-green-200' : 'bg-green-100 text-green-800') : (isDark ? 'bg-red-900 text-red-200' : 'bg-red-100 text-red-800')}`}>
                          {pair.change24h >= 0 ? '+' : ''}{pair.change24h}%
                        </span>
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              {/* Order Form */}
              <div className={`${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'} rounded-lg p-6 shadow-sm`}>
                <h3 className="font-semibold mb-4">Open Position</h3>
                
                {/* Pair Selection */}
                <div className="mb-4">
                  <label className={`block text-sm mb-2 ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>Trading Pair</label>
                  <select
                    value={selectedPair.symbol}
                    onChange={(e) => setSelectedPair(TRADING_PAIRS.find(p => p.symbol === e.target.value) || TRADING_PAIRS[0])}
                    className={`w-full border-0 rounded-lg px-4 py-2 ${isDark ? 'bg-slate-700' : 'bg-slate-100'}`}
                  >
                    {TRADING_PAIRS.map(pair => (
                      <option key={pair.symbol} value={pair.symbol}>{pair.symbol}</option>
                    ))}
                  </select>
                </div>

                {/* Side Selection */}
                <div className="grid grid-cols-2 gap-2 mb-4">
                  <button
                    onClick={() => setSide('LONG')}
                    className={`py-3 rounded-lg font-semibold transition-colors ${
                      side === 'LONG'
                        ? 'bg-green-500 text-white'
                        : isDark
                          ? 'bg-slate-700 text-slate-400 hover:bg-green-900'
                          : 'bg-slate-100 text-slate-600 hover:bg-green-100'
                    }`}
                  >
                    LONG
                  </button>
                  <button
                    onClick={() => setSide('SHORT')}
                    className={`py-3 rounded-lg font-semibold transition-colors ${
                      side === 'SHORT'
                        ? 'bg-red-500 text-white'
                        : isDark
                          ? 'bg-slate-700 text-slate-400 hover:bg-red-900'
                          : 'bg-slate-100 text-slate-600 hover:bg-red-100'
                    }`}
                  >
                    SHORT
                  </button>
                </div>

                {/* Order Type */}
                <div className="grid grid-cols-2 gap-2 mb-4">
                  <button
                    onClick={() => setOrderType('market')}
                    className={`py-2 rounded-lg text-sm transition-colors ${
                      orderType === 'market'
                        ? 'bg-orange-500 text-white'
                        : isDark ? 'bg-slate-700 text-slate-400' : 'bg-slate-100 text-slate-600'
                    }`}
                  >
                    Market
                  </button>
                  <button
                    onClick={() => setOrderType('limit')}
                    className={`py-2 rounded-lg text-sm transition-colors ${
                      orderType === 'limit'
                        ? 'bg-orange-500 text-white'
                        : isDark ? 'bg-slate-700 text-slate-400' : 'bg-slate-100 text-slate-600'
                    }`}
                  >
                    Limit
                  </button>
                </div>

                {/* Limit Price */}
                {orderType === 'limit' && (
                  <div className="mb-4">
                    <label className={`block text-sm mb-2 ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>Limit Price</label>
                    <input
                      type="number"
                      value={limitPrice}
                      onChange={(e) => setLimitPrice(e.target.value)}
                      placeholder={selectedPair.price.toString()}
                      className={`w-full border-0 rounded-lg px-4 py-2 ${isDark ? 'bg-slate-700' : 'bg-slate-100'}`}
                    />
                  </div>
                )}

                {/* Leverage */}
                <div className="mb-4">
                  <div className="flex justify-between mb-2">
                    <label className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>Leverage</label>
                    <span className="font-semibold">{leverage}x</span>
                  </div>
                  <input
                    type="range"
                    min="1"
                    max="50"
                    value={leverage}
                    onChange={(e) => setLeverage(parseInt(e.target.value))}
                    className="w-full"
                  />
                  <div className={`flex justify-between text-xs mt-1 ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>
                    <span>1x</span>
                    <span>10x</span>
                    <span>25x</span>
                    <span>50x</span>
                  </div>
                </div>

                {/* Collateral */}
                <div className="mb-4">
                  <label className={`block text-sm mb-2 ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>Collateral (USDT)</label>
                  <input
                    type="number"
                    value={collateral}
                    onChange={(e) => setCollateral(e.target.value)}
                    placeholder="0.00"
                    className={`w-full border-0 rounded-lg px-4 py-2 ${isDark ? 'bg-slate-700' : 'bg-slate-100'}`}
                  />
                </div>

                {/* Position Summary */}
                <div className={`rounded-lg p-4 mb-4 ${isDark ? 'bg-slate-700' : 'bg-slate-100'}`}>
                  <div className="grid grid-cols-2 gap-2 text-sm">
                    <div className={isDark ? 'text-slate-400' : 'text-slate-500'}>Position Size</div>
                    <div className="text-right font-semibold">{calculatePositionSize()} {selectedPair.symbol.split('/')[0]}</div>
                    <div className={isDark ? 'text-slate-400' : 'text-slate-500'}>Entry Price</div>
                    <div className="text-right font-semibold">
                      {orderType === 'limit' && limitPrice ? formatCurrency(parseFloat(limitPrice)) : formatCurrency(selectedPair.price)}
                    </div>
                    <div className={isDark ? 'text-slate-400' : 'text-slate-500'}>Liquidation Price</div>
                    <div className="text-right font-semibold text-red-500">{calculateLiquidationPrice()}</div>
                  </div>
                </div>

                <button
                  onClick={handleOpenPosition}
                  disabled={loading}
                  className={`w-full py-3 rounded-lg font-semibold transition-colors ${
                    side === 'LONG'
                      ? 'bg-green-500 hover:bg-green-600'
                      : 'bg-red-500 hover:bg-red-600'
                  } text-white disabled:opacity-50`}
                >
                  {loading ? 'Opening...' : `${side} ${selectedPair.symbol}`}
                </button>
              </div>
            </div>
          )}

          {/* Sidebar */}
          <div className="space-y-6">
            {/* Funding Rates */}
            <div className={`${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'} rounded-lg p-6 shadow-sm`}>
              <h3 className="font-semibold mb-4">Funding Rates</h3>
              <div className="space-y-3">
                {FUNDING_RATES.map((rate) => (
                  <div key={rate.pair} className="flex justify-between text-sm">
                    <span>{rate.pair}</span>
                    <span className={rate.rate > 0 ? 'text-green-500' : 'text-red-500'}>
                      {rate.rate > 0 ? '+' : ''}{rate.rate.toFixed(3)}%
                    </span>
                  </div>
                ))}
              </div>
            </div>

            {/* Quick Stats */}
            <div className={`${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'} rounded-lg p-6 shadow-sm`}>
              <h3 className="font-semibold mb-4">24h Stats</h3>
              <div className="space-y-3 text-sm">
                <div className="flex justify-between">
                  <span className={isDark ? 'text-slate-400' : 'text-slate-500'}>High</span>
                  <span className="font-semibold">{formatCurrency(selectedPair.high24h)}</span>
                </div>
                <div className="flex justify-between">
                  <span className={isDark ? 'text-slate-400' : 'text-slate-500'}>Low</span>
                  <span className="font-semibold">{formatCurrency(selectedPair.low24h)}</span>
                </div>
                <div className="flex justify-between">
                  <span className={isDark ? 'text-slate-400' : 'text-slate-500'}>Volume</span>
                  <span className="font-semibold">{formatCurrency(selectedPair.volume24h)}</span>
                </div>
              </div>
            </div>
          </div>

          {/* Positions Tab */}
          {activeTab === 'positions' && (
            <div className="lg:col-span-3">
              {positions.filter(p => p.status === 'open').length === 0 ? (
                <div className="text-center py-12">
                  <div className="text-6xl mb-4">📈</div>
                  <h3 className="text-xl font-semibold mb-2">No Open Positions</h3>
                  <p className={isDark ? 'text-slate-400' : 'text-slate-500'}>Open a position to start trading</p>
                  <button
                    onClick={() => setActiveTab('trade')}
                    className="mt-4 bg-orange-500 hover:bg-orange-600 text-white px-6 py-2 rounded-lg transition-colors"
                  >
                    Open Position
                  </button>
                </div>
              ) : (
                <div className="space-y-4">
                  {positions.filter(p => p.status === 'open').map((position) => (
                    <div key={position.id} className={`${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'} rounded-lg p-6 shadow-sm`}>
                      <div className="flex items-start justify-between">
                        <div>
                          <div className="flex items-center gap-2">
                            <span className={`px-2 py-1 rounded text-xs font-semibold ${position.side === 'LONG' ? (isDark ? 'bg-green-900 text-green-200' : 'bg-green-100 text-green-800') : (isDark ? 'bg-red-900 text-red-200' : 'bg-red-100 text-red-800')}`}>
                              {position.side}
                            </span>
                            <span className="font-semibold text-lg">{position.pair}</span>
                          </div>
                          <div className={`text-sm mt-1 ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>
                            {position.size} @ {formatCurrency(parseFloat(position.entryPrice))}
                          </div>
                        </div>
                        <div className="text-right">
                          <div className={`text-xl font-bold ${position.pnl >= 0 ? 'text-green-500' : 'text-red-500'}`}>
                            {position.pnl >= 0 ? '+' : ''}{position.pnlPercent.toFixed(2)}%
                          </div>
                          <div className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>
                            {formatCurrency(position.pnl)}
                          </div>
                        </div>
                      </div>
                      
                      <div className={`grid grid-cols-4 gap-4 mt-4 pt-4 border-t ${isDark ? 'border-slate-700' : 'border-slate-200'}`}>
                        <div>
                          <div className={`text-xs ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>Collateral</div>
                          <div className="font-semibold">{position.collateral} USDT</div>
                        </div>
                        <div>
                          <div className={`text-xs ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>Leverage</div>
                          <div className="font-semibold">{position.leverage}x</div>
                        </div>
                        <div>
                          <div className={`text-xs ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>Mark Price</div>
                          <div className="font-semibold">{formatCurrency(parseFloat(position.markPrice))}</div>
                        </div>
                        <div>
                          <div className={`text-xs ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>Liq. Price</div>
                          <div className="font-semibold text-red-500">{formatCurrency(parseFloat(position.liquidationPrice))}</div>
                        </div>
                      </div>
                      
                      <div className="mt-4 flex gap-2">
                        <button
                          onClick={() => handleClosePosition(position.id)}
                          disabled={loading}
                          className={`flex-1 py-2 rounded-lg transition-colors ${isDark ? 'bg-slate-700 hover:bg-slate-600' : 'bg-slate-200 hover:bg-slate-300'}`}
                        >
                          Close Position
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}

          {/* History Tab */}
          {activeTab === 'history' && (
            <div className="lg:col-span-3">
              <div className={`${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'} rounded-lg p-6 shadow-sm`}>
                <h3 className="font-semibold mb-4">Position History</h3>
                {positions.filter(p => p.status !== 'open').length === 0 ? (
                  <div className={`text-center py-8 ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>
                    No closed positions yet
                  </div>
                ) : (
                  <div className="space-y-4">
                    {positions.filter(p => p.status !== 'open').map((position) => (
                      <div key={position.id} className={`flex items-center justify-between py-4 border-b ${isDark ? 'border-slate-700' : 'border-slate-200'}`}>
                        <div>
                          <div className="font-semibold">{position.pair}</div>
                          <div className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>
                            {position.side} • {position.size} @ {formatCurrency(parseFloat(position.entryPrice))}
                          </div>
                        </div>
                        <div className="text-right">
                          <div className={`font-bold ${position.pnl >= 0 ? 'text-green-500' : 'text-red-500'}`}>
                            {position.pnl >= 0 ? '+' : ''}{formatCurrency(position.pnl)}
                          </div>
                          <div className={`text-xs ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>
                            {formatTime(position.closedAt || 0)}
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
