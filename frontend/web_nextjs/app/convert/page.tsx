'use client';

import React, { useState, useEffect } from 'react';
import { useTheme } from '../components/ThemeProvider';

interface ConvertToken {
  symbol: string;
  name: string;
  balance: number;
  icon: string;
}

interface ConvertPair {
  from: string;
  to: string;
  rate: number;
  inverseRate: number;
  fee: number;
}

const TOKENS: ConvertToken[] = [
  { symbol: 'BTC', name: 'Bitcoin', balance: 1.5, icon: '₿' },
  { symbol: 'ETH', name: 'Ethereum', balance: 15.0, icon: 'Ξ' },
  { symbol: 'USDT', name: 'Tether', balance: 50000, icon: '₮' },
  { symbol: 'USDC', name: 'USD Coin', balance: 25000, icon: '$' },
  { symbol: 'BNB', name: 'BNB', balance: 50, icon: 'B' },
  { symbol: 'SOL', name: 'Solana', balance: 150, icon: 'S' },
  { symbol: 'XRP', name: 'Ripple', balance: 10000, icon: 'X' },
  { symbol: 'ADA', name: 'Cardano', balance: 5000, icon: 'A' },
  { symbol: 'DOGE', name: 'Dogecoin', balance: 100000, icon: 'D' },
  { symbol: 'AVAX', name: 'Avalanche', balance: 200, icon: 'A' },
];

const CONVERT_PAIRS: ConvertPair[] = [
  { from: 'BTC', to: 'USDT', rate: 43250, inverseRate: 0.00002312, fee: 0.1 },
  { from: 'ETH', to: 'USDT', rate: 2280, inverseRate: 0.0004386, fee: 0.1 },
  { from: 'BNB', to: 'USDT', rate: 312.5, inverseRate: 0.0032, fee: 0.1 },
  { from: 'SOL', to: 'USDT', rate: 98.75, inverseRate: 0.01013, fee: 0.1 },
  { from: 'XRP', to: 'USDT', rate: 0.62, inverseRate: 1.6129, fee: 0.1 },
  { from: 'BTC', to: 'ETH', rate: 18.97, inverseRate: 0.0527, fee: 0.1 },
  { from: 'ETH', to: 'BTC', rate: 0.0527, inverseRate: 18.97, fee: 0.1 },
];

export default function ConvertPage() {
  const { isDark } = useTheme();
  const [fromToken, setFromToken] = useState<ConvertToken>(TOKENS[0]);
  const [toToken, setToToken] = useState<ConvertToken>(TOKENS[2]);
  const [fromAmount, setFromAmount] = useState('');
  const [toAmount, setToAmount] = useState('');
  const [activeTab, setActiveTab] = useState<'convert' | 'history'>('convert');
  const [conversion, setConversion] = useState<{from: string, to: string, amount: number, result: number, time: Date}[]>([]);

  const getRate = (from: string, to: string): number => {
    if (from === to) return 1;
    const pair = CONVERT_PAIRS.find(p => p.from === from && p.to === to);
    if (pair) return pair.rate;
    const reversePair = CONVERT_PAIRS.find(p => p.from === to && p.to === from);
    if (reversePair) return 1 / reversePair.rate;
    
    const fromToUsdt = CONVERT_PAIRS.find(p => p.from === from && p.to === 'USDT');
    const toFromUsdt = CONVERT_PAIRS.find(p => p.from === to && p.to === 'USDT');
    if (fromToUsdt && toFromUsdt) return fromToUsdt.rate / toFromUsdt.rate;
    return 1;
  };

  const rate = getRate(fromToken.symbol, toToken.symbol);
  const fee = (parseFloat(fromAmount) || 0) * 0.001;

  useEffect(() => {
    if (fromAmount && parseFloat(fromAmount) > 0) {
      const result = parseFloat(fromAmount) * rate;
      setToAmount(result.toFixed(fromToken.symbol === 'BTC' || fromToken.symbol === 'ETH' ? 6 : 2));
    } else {
      setToAmount('');
    }
  }, [fromAmount, rate, fromToken.symbol]);

  const handleSwap = () => {
    const temp = fromToken;
    setFromToken(toToken);
    setToToken(temp);
    setFromAmount('');
    setToAmount('');
  };

  const handleConvert = () => {
    if (!fromAmount || !toAmount) return;
    
    const newConversion = {
      from: fromToken.symbol,
      to: toToken.symbol,
      amount: parseFloat(fromAmount),
      result: parseFloat(toAmount),
      time: new Date(),
    };
    
    setConversion([newConversion, ...conversion]);
    setFromAmount('');
    setToAmount('');
  };

  const selectToken = (token: ConvertToken, isFrom: boolean) => {
    if (isFrom) {
      setFromToken(token);
    } else {
      setToToken(token);
    }
  };

  return (
    <div className={`min-h-screen p-6 ${isDark ? 'bg-gray-900 text-white' : 'bg-gray-50 text-gray-900'}`}>
      <div className="max-w-4xl mx-auto">
        <div className="text-center mb-8">
          <h1 className="text-3xl font-bold">Convert</h1>
          <p className={`mt-1 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>One-click conversion between crypto assets</p>
        </div>

        <div className="flex justify-center mb-6">
          <div className={`rounded-lg p-1 flex ${isDark ? 'bg-gray-800' : 'bg-white border border-gray-200'}`}>
            <button
              onClick={() => setActiveTab('convert')}
              className={`px-6 py-2 rounded-lg ${activeTab === 'convert' ? 'bg-blue-600' : 'bg-transparent'}`}
            >
              Convert
            </button>
            <button
              onClick={() => setActiveTab('history')}
              className={`px-6 py-2 rounded-lg ${activeTab === 'history' ? 'bg-blue-600' : 'bg-transparent'}`}
            >
              History
            </button>
          </div>
        </div>

        {activeTab === 'convert' && (
          <div className={`rounded-xl p-6 ${isDark ? 'bg-gray-800' : 'bg-white border border-gray-200 shadow-sm'}`}>
            <div className="space-y-4">
              {/* From */}
              <div className={`rounded-lg p-4 ${isDark ? 'bg-gray-700' : 'bg-gray-100'}`}>
                <div className="flex justify-between items-center mb-2">
                  <span className={isDark ? 'text-gray-400' : 'text-gray-500'}>From</span>
                  <span className={isDark ? 'text-gray-400' : 'text-gray-500'}>Balance: {fromToken.balance.toLocaleString()} {fromToken.symbol}</span>
                </div>
                <div className="flex items-center space-x-4">
                  <div className="flex-1">
                    <input
                      type="number"
                      value={fromAmount}
                      onChange={(e) => setFromAmount(e.target.value)}
                      placeholder="0.00"
                      className="w-full bg-transparent text-2xl font-bold outline-none"
                    />
                  </div>
                  <div className="relative">
                    <button className={`flex items-center space-x-2 px-4 py-2 rounded-lg ${isDark ? 'bg-gray-600 hover:bg-gray-500' : 'bg-gray-200 hover:bg-gray-300'}`}>
                      <span>{fromToken.icon}</span>
                      <span>{fromToken.symbol}</span>
                      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                      </svg>
                    </button>
                    <div className={`absolute top-full right-0 mt-2 rounded-lg shadow-xl z-10 w-48 max-h-60 overflow-y-auto ${isDark ? 'bg-gray-700' : 'bg-white border border-gray-200'}`}>
                      {TOKENS.map((token) => (
                        <button
                          key={token.symbol}
                          onClick={() => selectToken(token, true)}
                          className={`w-full px-4 py-2 flex items-center space-x-2 ${isDark ? 'hover:bg-gray-600' : 'hover:bg-gray-100'}`}
                        >
                          <span>{token.icon}</span>
                          <span>{token.symbol}</span>
                        </button>
                      ))}
                    </div>
                  </div>
                </div>
              </div>

              {/* Swap Button */}
              <div className="flex justify-center -my-2 relative z-10">
                <button
                  onClick={handleSwap}
                  className="bg-blue-600 p-2 rounded-full hover:bg-blue-700"
                >
                  <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 16V4m0 0L3 8m4-4l4 4m6 0v12m0 0l4-4m-4 4l-4-4" />
                  </svg>
                </button>
              </div>

              {/* To */}
              <div className={`rounded-lg p-4 ${isDark ? 'bg-gray-700' : 'bg-gray-100'}`}>
                <div className="flex justify-between items-center mb-2">
                  <span className={isDark ? 'text-gray-400' : 'text-gray-500'}>To</span>
                  <span className={isDark ? 'text-gray-400' : 'text-gray-500'}>Balance: {toToken.balance.toLocaleString()} {toToken.symbol}</span>
                </div>
                <div className="flex items-center space-x-4">
                  <div className="flex-1">
                    <input
                      type="number"
                      value={toAmount}
                      readOnly
                      placeholder="0.00"
                      className={`w-full bg-transparent text-2xl font-bold outline-none ${isDark ? 'text-gray-300' : 'text-gray-600'}`}
                    />
                  </div>
                  <div className="relative">
                    <button className={`flex items-center space-x-2 px-4 py-2 rounded-lg ${isDark ? 'bg-gray-600 hover:bg-gray-500' : 'bg-gray-200 hover:bg-gray-300'}`}>
                      <span>{toToken.icon}</span>
                      <span>{toToken.symbol}</span>
                      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                      </svg>
                    </button>
                    <div className={`absolute top-full right-0 mt-2 rounded-lg shadow-xl z-10 w-48 max-h-60 overflow-y-auto ${isDark ? 'bg-gray-700' : 'bg-white border border-gray-200'}`}>
                      {TOKENS.map((token) => (
                        <button
                          key={token.symbol}
                          onClick={() => selectToken(token, false)}
                          className={`w-full px-4 py-2 flex items-center space-x-2 ${isDark ? 'hover:bg-gray-600' : 'hover:bg-gray-100'}`}
                        >
                          <span>{token.icon}</span>
                          <span>{token.symbol}</span>
                        </button>
                      ))}
                    </div>
                  </div>
                </div>
              </div>

              {/* Rate Info */}
              <div className={`rounded-lg p-4 ${isDark ? 'bg-gray-700' : 'bg-gray-100'}`}>
                <div className="flex justify-between mb-2">
                  <span className={isDark ? 'text-gray-400' : 'text-gray-500'}>Exchange Rate</span>
                  <span>1 {fromToken.symbol} = {rate.toFixed(6)} {toToken.symbol}</span>
                </div>
                <div className="flex justify-between">
                  <span className={isDark ? 'text-gray-400' : 'text-gray-500'}>Fee (0.1%)</span>
                  <span>≈ {fee.toFixed(6)} {fromToken.symbol}</span>
                </div>
              </div>

              {/* Convert Button */}
              <button
                onClick={handleConvert}
                disabled={!fromAmount || parseFloat(fromAmount) <= 0 || parseFloat(fromAmount) > fromToken.balance}
                className="w-full bg-blue-600 py-4 rounded-lg font-bold text-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Convert Now
              </button>

              {parseFloat(fromAmount) > fromToken.balance && (
                <p className="text-red-400 text-center">Insufficient balance</p>
              )}
            </div>
          </div>
        )}

        {activeTab === 'history' && (
          <div className={`rounded-xl p-6 ${isDark ? 'bg-gray-800' : 'bg-white border border-gray-200 shadow-sm'}`}>
            <h3 className="text-xl font-bold mb-4">Conversion History</h3>
            {conversion.length === 0 ? (
              <div className={`text-center py-12 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>No conversions yet</div>
            ) : (
              <div className="space-y-4">
                {conversion.map((c, i) => (
                  <div key={i} className={`rounded-lg p-4 flex justify-between items-center ${isDark ? 'bg-gray-700' : 'bg-gray-100'}`}>
                    <div>
                      <div className="font-bold">{c.from} → {c.to}</div>
                      <div className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>{c.time.toLocaleString()}</div>
                    </div>
                    <div className="text-right">
                      <div className="font-bold">{c.amount} {c.from}</div>
                      <div className="text-green-400">+{c.result.toFixed(6)} {c.to}</div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {/* Popular Pairs */}
        <div className="mt-8">
          <h3 className="text-xl font-bold mb-4">Popular Pairs</h3>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            {CONVERT_PAIRS.slice(0, 8).map((pair, i) => (
              <div key={i} className={`rounded-lg p-4 cursor-pointer ${isDark ? 'bg-gray-800 hover:bg-gray-700' : 'bg-white border border-gray-200 hover:bg-gray-50 shadow-sm'}`}>
                <div className="font-bold">{pair.from}/{pair.to}</div>
                <div className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>{pair.rate.toFixed(4)}</div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
