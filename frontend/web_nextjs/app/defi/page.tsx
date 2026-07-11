'use client';

import React, { useState } from 'react';

interface DeFiProtocol {
  id: string;
  name: string;
  category: string;
  tvl: string;
  apy: string;
  chains: number[];
  logo: string;
}

const DEFI_PROTOCOLS: DeFiProtocol[] = [
  { id: '1', name: 'Aave', category: 'lending', tvl: '$12.5B', apy: '3.5-8.5%', chains: [1, 137, 56], logo: '👻' },
  { id: '2', name: 'Compound', category: 'lending', tvl: '$2.8B', apy: '2.5-5.2%', chains: [1], logo: '📈' },
  { id: '3', name: 'Yearn Finance', category: 'yield', tvl: '$3.2B', apy: '5-15%', chains: [1], logo: '📊' },
  { id: '4', name: 'Uniswap', category: 'dex', tvl: '$4.1B', apy: '2-8%', chains: [1, 56, 137, 42161], logo: '🦄' },
  { id: '5', name: 'Curve', category: 'dex', tvl: '$2.3B', apy: '3-10%', chains: [1, 56], logo: '📉' },
  { id: '6', name: 'Lido', category: 'yield', tvl: '$15.2B', apy: '4.2%', chains: [1], logo: '💧' },
  { id: '7', name: 'PancakeSwap', category: 'dex', tvl: '$1.5B', apy: '3-8%', chains: [56], logo: '🥞' },
  { id: '8', name: 'SushiSwap', category: 'dex', tvl: '$1.2B', apy: '2-6%', chains: [1, 56, 137], logo: '🍣' },
];

export default function DeFiIntegration() {
  const [category, setCategory] = useState('all');
  const filteredProtocols = category === 'all' ? DEFI_PROTOCOLS : DEFI_PROTOCOLS.filter(p => p.category === category);

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-900 text-slate-900 dark:text-white">
      <header className="bg-white dark:bg-slate-800 border-b p-4"><div className="flex items-center gap-4"><a href="/" className="text-2xl">🐯</a><h1 className="text-xl font-bold">DeFi Integration</h1></div></header>
      <div className="max-w-7xl mx-auto p-8">
        <div className="flex gap-2 mb-6 flex-wrap">{['all', 'lending', 'yield', 'dex'].map(cat => <button key={cat} onClick={() => setCategory(cat)} className={`px-4 py-2 rounded-lg ${category === cat ? 'bg-orange-500 text-white' : 'bg-slate-200 dark:bg-slate-700'}`}>{cat.charAt(0).toUpperCase() + cat.slice(1)}</button>)}</div>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {filteredProtocols.map(protocol => (
            <div key={protocol.id} className="bg-white dark:bg-slate-800 rounded-lg p-4 shadow-sm">
              <div className="flex items-center gap-3 mb-3"><div className="text-3xl">{protocol.logo}</div><div><div className="font-semibold">{protocol.name}</div><div className="text-xs text-slate-500 capitalize">{protocol.category}</div></div></div>
              <div className="grid grid-cols-2 gap-2 text-sm mb-3"><div><span className="text-slate-500">TVL:</span> <span className="font-medium">{protocol.tvl}</span></div><div><span className="text-slate-500">APY:</span> <span className="font-medium text-green-500">{protocol.apy}</span></div></div>
              <button className="w-full bg-orange-500 hover:bg-orange-600 text-white py-2 rounded-lg">Connect</button>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
