'use client';

import React, { useState } from 'react';
import { useTheme } from '../components/ThemeProvider';

interface IntentRoute {
  id: string;
  name: string;
  protocols: string[];
  estimatedTime: string;
  savings: number;
}

const ROUTES: IntentRoute[] = [
  { id: '1', name: 'UniswapX', protocols: ['UniswapX', '1inch'], estimatedTime: '< 5s', savings: 15 },
  { id: '2', name: 'CoW Swap', protocols: ['CoW Swap', 'Gnosis'], estimatedTime: '< 30s', savings: 25 },
  { id: '3', name: 'Across Protocol', protocols: ['Across', 'Hop'], estimatedTime: '< 10s', savings: 12 },
  { id: '4', name: 'Li.Fi', protocols: ['Li.Fi', 'Socket'], estimatedTime: '< 15s', savings: 10 },
];

export default function IntentRoutingPage() {
  const { theme } = useTheme();
  const isDark = theme === 'dark';
  const [fromChain, setFromChain] = useState('ethereum');
  const [toChain, setToChain] = useState('arbitrum');
  const [fromToken, setFromToken] = useState('ETH');
  const [toToken, setToToken] = useState('ETH');
  const [amount, setAmount] = useState('');
  const [selectedRoute, setSelectedRoute] = useState<string | null>(null);

  return (
    <div className={`min-h-screen p-8 ${isDark ? 'bg-gradient-to-br from-slate-900 to-slate-800' : 'bg-gradient-to-br from-slate-50 to-slate-100'}`}>
      <div className="max-w-4xl mx-auto">
        <h1 className={`text-4xl font-bold mb-2 ${isDark ? 'text-white' : 'text-slate-900'}`}>Intent-Based Routing</h1>
        <p className={`mb-8 ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>Cross-chain swaps with best prices</p>

        <div className="grid grid-cols-2 gap-4 mb-8">
          <select value={fromChain} onChange={(e) => setFromChain(e.target.value)} className={`border rounded-lg px-4 py-3 ${isDark ? 'bg-slate-700 border-slate-600 text-white' : 'bg-white border-slate-300 text-slate-900'}`}>
            <option value="ethereum">Ethereum</option>
            <option value="arbitrum">Arbitrum</option>
            <option value="optimism">Optimism</option>
            <option value="polygon">Polygon</option>
          </select>
          <select value={toChain} onChange={(e) => setToChain(e.target.value)} className={`border rounded-lg px-4 py-3 ${isDark ? 'bg-slate-700 border-slate-600 text-white' : 'bg-white border-slate-300 text-slate-900'}`}>
            <option value="arbitrum">Arbitrum</option>
            <option value="optimism">Optimism</option>
            <option value="polygon">Polygon</option>
            <option value="avalanche">Avalanche</option>
          </select>
        </div>

        <div className="grid grid-cols-3 gap-4 mb-8">
          <div>
            <label className={`block text-sm mb-2 ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>From Token</label>
            <input type="number" value={amount} onChange={(e) => setAmount(e.target.value)} placeholder="0.0" className={`w-full border rounded-lg px-4 py-3 ${isDark ? 'bg-slate-700 border-slate-600 text-white' : 'bg-white border-slate-300 text-slate-900'}`} />
          </div>
          <div className="flex items-center justify-center">
            <span className="text-2xl">→</span>
          </div>
          <div>
            <label className={`block text-sm mb-2 ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>To Token</label>
            <input type="text" value={toToken} readOnly className={`w-full border rounded-lg px-4 py-3 ${isDark ? 'bg-slate-700 border-slate-600 text-white' : 'bg-white border-slate-300 text-slate-900'}`} />
          </div>
        </div>

        <h2 className={`text-xl font-semibold mb-4 ${isDark ? 'text-white' : 'text-slate-900'}`}>Available Routes</h2>
        <div className="space-y-3">
          {ROUTES.map(route => (
            <div
              key={route.id}
              onClick={() => setSelectedRoute(route.id)}
              className={`p-4 rounded-xl border cursor-pointer transition-all ${
                selectedRoute === route.id
                  ? 'border-blue-500 bg-blue-500/10'
                  : isDark ? 'border-slate-600 bg-slate-700/30 hover:border-slate-500' : 'border-slate-200 bg-white hover:border-slate-300'
              }`}
            >
              <div className="flex items-center justify-between">
                <div>
                  <h3 className={`font-medium ${isDark ? 'text-white' : 'text-slate-900'}`}>{route.name}</h3>
                  <p className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>{route.protocols.join(' + ')}</p>
                </div>
                <div className="text-right">
                  <p className="text-green-400 font-medium">Save {route.savings}%</p>
                  <p className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>{route.estimatedTime}</p>
                </div>
              </div>
            </div>
          ))}
        </div>

        {selectedRoute && (
          <button className="w-full mt-6 bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 text-white py-4 rounded-xl font-semibold">
            Execute Cross-Chain Swap
          </button>
        )}
      </div>
    </div>
  );
}
