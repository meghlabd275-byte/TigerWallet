'use client';

import React, { useState } from 'react';

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
  const [fromChain, setFromChain] = useState('ethereum');
  const [toChain, setToChain] = useState('arbitrum');
  const [fromToken, setFromToken] = useState('ETH');
  const [toToken, setToToken] = useState('ETH');
  const [amount, setAmount] = useState('');
  const [selectedRoute, setSelectedRoute] = useState<string | null>(null);

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-900 to-slate-800 p-8">
      <div className="max-w-4xl mx-auto">
        <h1 className="text-4xl font-bold text-white mb-2">Intent-Based Routing</h1>
        <p className="text-slate-400 mb-8">Cross-chain swaps with best prices</p>

        <div className="grid grid-cols-2 gap-4 mb-8">
          <select value={fromChain} onChange={(e) => setFromChain(e.target.value)} className="bg-slate-700 border border-slate-600 rounded-lg px-4 py-3 text-white">
            <option value="ethereum">Ethereum</option>
            <option value="arbitrum">Arbitrum</option>
            <option value="optimism">Optimism</option>
            <option value="polygon">Polygon</option>
          </select>
          <select value={toChain} onChange={(e) => setToChain(e.target.value)} className="bg-slate-700 border border-slate-600 rounded-lg px-4 py-3 text-white">
            <option value="arbitrum">Arbitrum</option>
            <option value="optimism">Optimism</option>
            <option value="polygon">Polygon</option>
            <option value="avalanche">Avalanche</option>
          </select>
        </div>

        <div className="grid grid-cols-3 gap-4 mb-8">
          <div>
            <label className="block text-slate-400 text-sm mb-2">From Token</label>
            <input type="number" value={amount} onChange={(e) => setAmount(e.target.value)} placeholder="0.0" className="w-full bg-slate-700 border border-slate-600 rounded-lg px-4 py-3 text-white" />
          </div>
          <div className="flex items-center justify-center">
            <span className="text-2xl">→</span>
          </div>
          <div>
            <label className="block text-slate-400 text-sm mb-2">To Token</label>
            <input type="text" value={toToken} readOnly className="w-full bg-slate-700 border border-slate-600 rounded-lg px-4 py-3 text-white" />
          </div>
        </div>

        <h2 className="text-xl font-semibold text-white mb-4">Available Routes</h2>
        <div className="space-y-3">
          {ROUTES.map(route => (
            <div 
              key={route.id}
              onClick={() => setSelectedRoute(route.id)}
              className={`p-4 rounded-xl border cursor-pointer transition-all ${
                selectedRoute === route.id ? 'border-blue-500 bg-blue-500/10' : 'border-slate-600 bg-slate-700/30 hover:border-slate-500'
              }`}
            >
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="text-white font-medium">{route.name}</h3>
                  <p className="text-slate-400 text-sm">{route.protocols.join(' + ')}</p>
                </div>
                <div className="text-right">
                  <p className="text-green-400 font-medium">Save {route.savings}%</p>
                  <p className="text-slate-400 text-sm">{route.estimatedTime}</p>
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
