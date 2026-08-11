'use client';

import React, { useState, useEffect } from 'react';
import { useTheme } from '../components/ThemeProvider'

// Types mirror the backend DAppEntry (go/wallet_api/dapp_directory.go).
// Only verifiable fields; no fabricated metrics (no invented ratings/users).

interface DApp {
  id: string;
  name: string;
  description: string;
  logo: string;
  category: string;
  chains: string[];
  url: string;
  verified: boolean;
}

interface Category {
  id: string;
  name: string;
  count: number;
}

const CHAIN_OPTIONS = [
  { value: 'all', label: 'All Chains' },
  { value: 'ethereum', label: 'Ethereum' },
  { value: 'solana', label: 'Solana' },
  { value: 'polygon', label: 'Polygon' },
  { value: 'arbitrum', label: 'Arbitrum' },
  { value: 'optimism', label: 'Optimism' },
  { value: 'avalanche', label: 'Avalanche' },
  { value: 'bsc', label: 'BNB Chain' },
  { value: 'base', label: 'Base' },
];

const CATEGORY_ICONS: Record<string, string> = {
  all: '🏁', defi: '💰', nft: '🖼️', bridge: '🌉', wallet: '👛',
  social: '👥', domain: '🔷', staking: '🔒', game: '🎮',
};

export default function DAppStorePage() {
  const { isDark } = useTheme()
  const [dapps, setDapps] = useState<DApp[]>([])
  const [categories, setCategories] = useState<Category[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedCategory, setSelectedCategory] = useState('all');
  const [selectedChain, setSelectedChain] = useState('all');
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid');
  const [bookmarkedDapps, setBookmarkedDapps] = useState<string[]>([]);

  useEffect(() => {
    let cancelled = false
    async function load() {
      setLoading(true)
      setError(null)
      try {
        const [dappsRes, catsRes] = await Promise.all([
          fetch('/api/v1/dapps', { cache: 'no-store' }),
          fetch('/api/v1/dapps/categories', { cache: 'no-store' }),
        ])
        if (!dappsRes.ok) throw new Error(`dapps ${dappsRes.status}`)
        const dappsData = await dappsRes.json()
        const catsData = await catsRes.json()
        if (!cancelled) {
          setDapps(dappsData.dapps || [])
          setCategories(catsData.categories || [])
        }
      } catch {
        if (!cancelled) setError('Unable to load dApp directory. Is the wallet backend running?')
      } finally {
        if (!cancelled) setLoading(false)
      }
    }
    load()
    return () => { cancelled = true }
  }, [])

  const filteredDapps = dapps.filter(dapp => {
    const matchesSearch = dapp.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      dapp.description.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesCategory = selectedCategory === 'all' || dapp.category === selectedCategory;
    const matchesChain = selectedChain === 'all' || dapp.chains.includes(selectedChain);
    return matchesSearch && matchesCategory && matchesChain;
  });

  const featuredDapps = dapps.filter(d => d.verified).slice(0, 8);

  const toggleBookmark = (id: string) => {
    setBookmarkedDapps(prev =>
      prev.includes(id) ? prev.filter(i => i !== id) : [...prev, id]
    );
  };

  return (
    <div className={`min-h-screen bg-gradient-to-br ${isDark ? 'from-slate-900 to-slate-800' : 'from-slate-50 to-slate-100'} p-8`}>
      <div className="max-w-6xl mx-auto">
        <div className="mb-8">
          <h1 className={`text-4xl font-bold mb-2 ${isDark ? 'text-white' : 'text-slate-900'}`}>DApp Store</h1>
          <p className={`${isDark ? 'text-slate-400' : 'text-slate-500'}`}>Discover and use decentralized applications</p>
        </div>

        {loading && (
          <div className="text-center py-12">
            <p className={`${isDark ? 'text-slate-400' : 'text-slate-500'}`}>Loading dApp directory…</p>
          </div>
        )}

        {error && !loading && (
          <div className={`text-center py-12 rounded-xl border ${isDark ? 'bg-red-900/20 border-red-800' : 'bg-red-50 border-red-200'}`}>
            <span className="text-4xl mb-4 block">⚠️</span>
            <p className={`${isDark ? 'text-red-300' : 'text-red-700'}`}>{error}</p>
          </div>
        )}

        {!loading && !error && (
          <>
            {/* Featured */}
            {featuredDapps.length > 0 && (
              <div className="mb-8">
                <h2 className={`text-xl font-semibold mb-4 ${isDark ? 'text-white' : 'text-slate-900'}`}>⭐ Featured</h2>
                <div className="flex gap-4 overflow-x-auto pb-4">
                  {featuredDapps.map(dapp => (
                    <a key={dapp.id} href={dapp.url} target="_blank" rel="noopener noreferrer"
                       className="flex-shrink-0 w-48 p-4 bg-gradient-to-br from-blue-500/20 to-purple-500/20 rounded-xl border border-blue-500/30 hover:border-blue-500/60 transition-colors">
                      <div className="text-3xl mb-2">{dapp.logo}</div>
                      <h3 className={`font-medium ${isDark ? 'text-white' : 'text-slate-900'}`}>{dapp.name}</h3>
                      <p className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>{dapp.category}</p>
                    </a>
                  ))}
                </div>
              </div>
            )}

            {/* Search & Filters */}
            <div className="flex gap-4 mb-6">
              <input
                type="text"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder="Search DApps..."
                className={`flex-1 border rounded-lg px-4 py-3 ${isDark ? 'bg-slate-700 border-slate-600 text-white' : 'bg-slate-100 border-slate-300 text-slate-900'}`}
              />
              <select
                value={selectedChain}
                onChange={(e) => setSelectedChain(e.target.value)}
                className={`border rounded-lg px-4 py-3 ${isDark ? 'bg-slate-700 border-slate-600 text-white' : 'bg-slate-100 border-slate-300 text-slate-900'}`}
              >
                {CHAIN_OPTIONS.map(c => <option key={c.value} value={c.value}>{c.label}</option>)}
              </select>
              <div className="flex gap-2">
                <button onClick={() => setViewMode('grid')}
                  className={`p-3 rounded-lg ${viewMode === 'grid' ? 'bg-blue-600' : isDark ? 'bg-slate-700' : 'bg-slate-200'}`}>
                  <span className={viewMode === 'grid' ? 'text-white' : isDark ? 'text-white' : 'text-slate-900'}>▦</span>
                </button>
                <button onClick={() => setViewMode('list')}
                  className={`p-3 rounded-lg ${viewMode === 'list' ? 'bg-blue-600' : isDark ? 'bg-slate-700' : 'bg-slate-200'}`}>
                  <span className={viewMode === 'list' ? 'text-white' : isDark ? 'text-white' : 'text-slate-900'}>☰</span>
                </button>
              </div>
            </div>

            {/* Categories */}
            <div className="flex gap-2 mb-6 overflow-x-auto pb-2">
              {categories.map(category => (
                <button
                  key={category.id}
                  onClick={() => setSelectedCategory(category.id)}
                  className={`flex items-center gap-2 px-4 py-2 rounded-full whitespace-nowrap transition-colors ${
                    selectedCategory === category.id
                      ? 'bg-blue-600 text-white'
                      : isDark ? 'bg-slate-700 text-slate-300 hover:bg-slate-600' : 'bg-slate-200 text-slate-700 hover:bg-slate-300'
                  }`}
                >
                  <span>{CATEGORY_ICONS[category.id] || '📁'}</span>
                  <span className="capitalize">{category.name}</span>
                  <span className="text-xs opacity-70">({category.count})</span>
                </button>
              ))}
            </div>

            {/* DApps Grid */}
            <div className={viewMode === 'grid' ? 'grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4' : 'space-y-4'}>
              {filteredDapps.map(dapp => (
                <div key={dapp.id} className={`rounded-xl p-4 border transition-colors ${isDark ? 'bg-slate-800 border-slate-700 hover:border-slate-600' : 'bg-white border-slate-200 hover:border-slate-300'}`}>
                  <div className="flex items-start gap-4">
                    <span className="text-4xl">{dapp.logo}</span>
                    <div className="flex-1">
                      <div className="flex items-center gap-2">
                        <h3 className={`font-semibold ${isDark ? 'text-white' : 'text-slate-900'}`}>{dapp.name}</h3>
                        {dapp.verified && <span className="text-blue-400">✓</span>}
                      </div>
                      <p className={`text-sm mt-1 ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>{dapp.description}</p>
                      <div className="flex gap-2 mt-3">
                        {dapp.chains.map(chain => (
                          <span key={chain} className={`text-xs px-2 py-1 rounded capitalize ${isDark ? 'bg-slate-700 text-slate-300' : 'bg-slate-100 text-slate-700'}`}>
                            {chain}
                          </span>
                        ))}
                      </div>
                    </div>
                    <button onClick={() => toggleBookmark(dapp.id)}
                      className={`${isDark ? 'text-slate-400' : 'text-slate-500'} hover:text-yellow-400`}>
                      {bookmarkedDapps.includes(dapp.id) ? '★' : '☆'}
                    </button>
                  </div>
                  <a href={dapp.url} target="_blank" rel="noopener noreferrer"
                     className="block mt-4 text-center bg-blue-600 hover:bg-blue-700 text-white py-2 rounded-lg font-medium">
                    Open DApp
                  </a>
                </div>
              ))}
            </div>

            {filteredDapps.length === 0 && (
              <div className="text-center py-12">
                <span className="text-4xl mb-4 block">🔍</span>
                <p className={`${isDark ? 'text-slate-400' : 'text-slate-500'}`}>No DApps found matching your criteria</p>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}
