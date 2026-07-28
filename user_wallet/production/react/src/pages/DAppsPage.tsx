/**
 * DApps Page - DApp Browser
 */

import React, { useState } from 'react';
import { useTheme } from '../contexts/ThemeContext';

function DAppsPage() {
  const { theme } = useTheme();
  const [url, setUrl] = useState('');
  const [activeTab, setActiveTab] = useState('dapps');

  const popularDApps = [
    { name: 'Uniswap', category: 'DEX', url: 'https://app.uniswap.org' },
    { name: 'OpenSea', category: 'NFT', url: 'https://opensea.io' },
    { name: 'Aave', category: 'Lending', url: 'https://app.aave.com' },
    { name: 'Compound', category: 'Lending', url: 'https://app.compound.finance' },
    { name: 'Curve', category: 'DEX', url: 'https://curve.fi' },
    { name: '1inch', category: 'Aggregator', url: 'https://app.1inch.io' },
  ];

  const categories = ['All', 'DEX', 'NFT', 'Lending', 'Games', 'Social'];

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">DApps</h1>

      {/* URL Bar */}
      <div className={`card mb-6 ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
        <div className="flex gap-2">
          <input
            type="text"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            placeholder="Enter DApp URL..."
            className="input flex-1"
          />
          <button className="btn btn-primary">Go</button>
        </div>
      </div>

      {/* Categories */}
      <div className="flex gap-2 mb-6 overflow-x-auto">
        {categories.map(cat => (
          <button
            key={cat}
            onClick={() => setActiveTab(cat.toLowerCase())}
            className={`px-4 py-2 rounded-lg text-sm whitespace-nowrap ${
              activeTab === cat.toLowerCase() 
                ? 'bg-amber-500 text-black' 
                : theme === 'dark' ? 'bg-slate-800' : 'bg-gray-200'
            }`}
          >
            {cat}
          </button>
        ))}
      </div>

      {/* DApps Grid */}
      <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
        {popularDApps.map(dapp => (
          <div 
            key={dapp.name} 
            className={`card cursor-pointer hover:border-amber-500 ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}
          >
            <div className="w-12 h-12 bg-amber-500/20 rounded-lg mb-3 flex items-center justify-center text-2xl">
              🔗
            </div>
            <h3 className="font-semibold">{dapp.name}</h3>
            <p className="text-sm opacity-60">{dapp.category}</p>
          </div>
        ))}
      </div>
    </div>
  );
}

export default DAppsPage;
