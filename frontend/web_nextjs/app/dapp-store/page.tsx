'use client';

import React, { useState } from 'react';

// ================================================================================
// Types
// ================================================================================

interface DApp {
  id: string;
  name: string;
  description: string;
  logo: string;
  category: string;
  chains: string[];
  url: string;
  verified: boolean;
  trending: boolean;
  rating: number;
  users: number;
}

interface Category {
  id: string;
  name: string;
  icon: string;
  count: number;
}

// Sample DApps
const SAMPLE_DAPPS: DApp[] = [
  { id: '1', name: 'Uniswap', description: 'Decentralized trading protocol', logo: '🦄', category: 'defi', chains: ['ethereum', 'arbitrum', 'optimism'], url: 'https://uniswap.org', verified: true, trending: true, rating: 4.8, users: 2500000 },
  { id: '2', name: 'Aave', description: 'Non-custodial liquidity protocol', logo: '👻', category: 'defi', chains: ['ethereum', 'polygon', 'avalanche'], url: 'https://aave.com', verified: true, trending: true, rating: 4.7, users: 1800000 },
  { id: '3', name: 'OpenSea', description: 'NFT marketplace', logo: '🌊', category: 'nft', chains: ['ethereum', 'polygon'], url: 'https://opensea.io', verified: true, trending: true, rating: 4.5, users: 5000000 },
  { id: '4', name: 'Compound', description: 'Algorithmic money market', logo: '📊', category: 'defi', chains: ['ethereum'], url: 'https://compound.finance', verified: true, trending: false, rating: 4.6, users: 900000 },
  { id: '5', name: 'Magic Eden', description: 'NFT marketplace', logo: '🧙', category: 'nft', chains: ['solana', 'ethereum'], url: 'https://magiceden.io', verified: true, trending: true, rating: 4.4, users: 1500000 },
  { id: '6', name: 'Raydium', description: 'AMM and liquidity provider', logo: '🌊', category: 'defi', chains: ['solana'], url: 'https://raydium.io', verified: true, trending: false, rating: 4.3, users: 800000 },
  { id: '7', name: 'Stargate', description: 'Cross-chain bridge', logo: '🌉', category: 'bridge', chains: ['ethereum', 'avalanche', 'polygon'], url: 'https://stargate.finance', verified: true, trending: true, rating: 4.5, users: 1200000 },
  { id: '8', name: 'Jupiter', description: 'Solana DEX aggregator', logo: '🪐', category: 'defi', chains: ['solana'], url: 'https://jup.ag', verified: true, trending: true, rating: 4.7, users: 2000000 },
  { id: '9', name: 'Metamask', description: 'Crypto wallet', logo: '🦊', category: 'wallet', chains: ['ethereum', 'polygon'], url: 'https://metamask.io', verified: true, trending: false, rating: 4.6, users: 30000000 },
  { id: '10', name: 'Lens Protocol', description: 'Social graph', logo: '📸', category: 'social', chains: ['polygon'], url: 'https://lens.xyz', verified: true, trending: true, rating: 4.2, users: 300000 },
];

const CATEGORIES: Category[] = [
  { id: 'all', name: 'All', icon: '🏁', count: SAMPLE_DAPPS.length },
  { id: 'defi', name: 'DeFi', icon: '💰', count: SAMPLE_DAPPS.filter(d => d.category === 'defi').length },
  { id: 'nft', name: 'NFT', icon: '🖼️', count: SAMPLE_DAPPS.filter(d => d.category === 'nft').length },
  { id: 'bridge', name: 'Bridges', icon: '🌉', count: SAMPLE_DAPPS.filter(d => d.category === 'bridge').length },
  { id: 'wallet', name: 'Wallets', icon: '👛', count: SAMPLE_DAPPS.filter(d => d.category === 'wallet').length },
  { id: 'social', name: 'Social', icon: '👥', count: SAMPLE_DAPPS.filter(d => d.category === 'social').length },
  { id: 'games', name: 'Games', icon: '🎮', count: 45 },
  { id: 'tools', name: 'Tools', icon: '🔧', count: 78 },
];

// ================================================================================
// Main Component
// ================================================================================

export default function DAppStorePage() {
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedCategory, setSelectedCategory] = useState('all');
  const [selectedChain, setSelectedChain] = useState('all');
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid');
  const [bookmarkedDapps, setBookmarkedDapps] = useState<string[]>([]);

  const filteredDapps = SAMPLE_DAPPS.filter(dapp => {
    const matchesSearch = dapp.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      dapp.description.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesCategory = selectedCategory === 'all' || dapp.category === selectedCategory;
    const matchesChain = selectedChain === 'all' || dapp.chains.includes(selectedChain);
    return matchesSearch && matchesCategory && matchesChain;
  });

  const toggleBookmark = (id: string) => {
    setBookmarkedDapps(prev => 
      prev.includes(id) ? prev.filter(i => i !== id) : [...prev, id]
    );
  };

  const formatUsers = (num: number) => {
    if (num >= 1000000) return `${(num / 1000000).toFixed(1)}M`;
    if (num >= 1000) return `${(num / 1000).toFixed(0)}K`;
    return num.toString();
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-900 to-slate-800 p-8">
      <div className="max-w-6xl mx-auto">
        {/* Header */}
        <div className="mb-8">
          <h1 className="text-4xl font-bold text-white mb-2">DApp Store</h1>
          <p className="text-slate-400">Discover and use decentralized applications</p>
        </div>

        {/* Trending */}
        <div className="mb-8">
          <h2 className="text-xl font-semibold text-white mb-4">🔥 Trending</h2>
          <div className="flex gap-4 overflow-x-auto pb-4">
            {SAMPLE_DAPPS.filter(d => d.trending).map(dapp => (
              <div key={dapp.id} className="flex-shrink-0 w-48 p-4 bg-gradient-to-br from-blue-500/20 to-purple-500/20 rounded-xl border border-blue-500/30">
                <div className="text-3xl mb-2">{dapp.logo}</div>
                <h3 className="text-white font-medium">{dapp.name}</h3>
                <p className="text-slate-400 text-sm">{dapp.category}</p>
              </div>
            ))}
          </div>
        </div>

        {/* Search & Filters */}
        <div className="flex gap-4 mb-6">
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search DApps..."
            className="flex-1 bg-slate-700 border border-slate-600 rounded-lg px-4 py-3 text-white"
          />
          <select
            value={selectedChain}
            onChange={(e) => setSelectedChain(e.target.value)}
            className="bg-slate-700 border border-slate-600 rounded-lg px-4 py-3 text-white"
          >
            <option value="all">All Chains</option>
            <option value="ethereum">Ethereum</option>
            <option value="solana">Solana</option>
            <option value="polygon">Polygon</option>
            <option value="arbitrum">Arbitrum</option>
            <option value="avalanche">Avalanche</option>
          </select>
          <div className="flex gap-2">
            <button
              onClick={() => setViewMode('grid')}
              className={`p-3 rounded-lg ${viewMode === 'grid' ? 'bg-blue-600' : 'bg-slate-700'}`}
            >
              <span className="text-white">▦</span>
            </button>
            <button
              onClick={() => setViewMode('list')}
              className={`p-3 rounded-lg ${viewMode === 'list' ? 'bg-blue-600' : 'bg-slate-700'}`}
            >
              <span className="text-white">☰</span>
            </button>
          </div>
        </div>

        {/* Categories */}
        <div className="flex gap-2 mb-6 overflow-x-auto pb-2">
          {CATEGORIES.map(category => (
            <button
              key={category.id}
              onClick={() => setSelectedCategory(category.id)}
              className={`flex items-center gap-2 px-4 py-2 rounded-full whitespace-nowrap transition-colors ${
                selectedCategory === category.id
                  ? 'bg-blue-600 text-white'
                  : 'bg-slate-700 text-slate-300 hover:bg-slate-600'
              }`}
            >
              <span>{category.icon}</span>
              <span>{category.name}</span>
              <span className="text-xs opacity-70">({category.count})</span>
            </button>
          ))}
        </div>

        {/* DApps Grid */}
        <div className={viewMode === 'grid' ? 'grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4' : 'space-y-4'}>
          {filteredDapps.map(dapp => (
            <div key={dapp.id} className="bg-slate-800 rounded-xl p-4 border border-slate-700 hover:border-slate-600 transition-colors">
              <div className="flex items-start gap-4">
                <span className="text-4xl">{dapp.logo}</span>
                <div className="flex-1">
                  <div className="flex items-center gap-2">
                    <h3 className="text-white font-semibold">{dapp.name}</h3>
                    {dapp.verified && <span className="text-blue-400">✓</span>}
                  </div>
                  <p className="text-slate-400 text-sm mt-1">{dapp.description}</p>
                  
                  <div className="flex items-center gap-4 mt-3 text-sm">
                    <span className="text-yellow-400">★ {dapp.rating}</span>
                    <span className="text-slate-400">{formatUsers(dapp.users)} users</span>
                  </div>
                  
                  <div className="flex gap-2 mt-3">
                    {dapp.chains.map(chain => (
                      <span key={chain} className="text-xs px-2 py-1 bg-slate-700 rounded text-slate-300 capitalize">
                        {chain}
                      </span>
                    ))}
                  </div>
                </div>
                
                <button
                  onClick={() => toggleBookmark(dapp.id)}
                  className="text-slate-400 hover:text-yellow-400"
                >
                  {bookmarkedDapps.includes(dapp.id) ? '★' : '☆'}
                </button>
              </div>
              
              <a
                href={dapp.url}
                target="_blank"
                rel="noopener noreferrer"
                className="block mt-4 text-center bg-blue-600 hover:bg-blue-700 text-white py-2 rounded-lg font-medium"
              >
                Open DApp
              </a>
            </div>
          ))}
        </div>

        {filteredDapps.length === 0 && (
          <div className="text-center py-12">
            <span className="text-4xl mb-4 block">🔍</span>
            <p className="text-slate-400">No DApps found matching your criteria</p>
          </div>
        )}
      </div>
    </div>
  );
}
