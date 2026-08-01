'use client';

import React, { useState, useEffect, useCallback } from 'react';

interface DeFiProtocol {
  id: string;
  name: string;
  category: string;
  tvl: string;
  apy: string;
  chains: number[];
  logo: string;
  contractAddress?: string;
  protocolUrl?: string;
}

interface ApiResponse<T> {
  success: boolean;
  data: T;
  error?: string;
}

// API Base URL - configured per environment
const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'https://api.tigerwallet.io';

const fetchAPI = async <T,>(endpoint: string, options?: RequestInit): Promise<T> => {
  const token = typeof window !== 'undefined' ? localStorage.getItem('tigerwallet-token') : null;
  const response = await fetch(`${API_BASE_URL}${endpoint}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options?.headers,
    },
  });
  if (!response.ok) {
    throw new Error(`API Error: ${response.statusText}`);
  }
  const data: ApiResponse<T> = await response.json();
  return data.data;
};

// Fallback data if API is unavailable
const FALLBACK_PROTOCOLS: DeFiProtocol[] = [
  { id: '1', name: 'Aave', category: 'lending', tvl: '$12.5B', apy: '3.5-8.5%', chains: [1, 137, 56], logo: '👻', contractAddress: '0x7Fc66500c84A76Ad7e9c93437bFc5Ac33E2DDaE9', protocolUrl: 'https://app.aave.com' },
  { id: '2', name: 'Compound', category: 'lending', tvl: '$2.8B', apy: '2.5-5.2%', chains: [1], logo: '📈', contractAddress: '0x3d9819210A31b4961b30EF54bE2aeD79B9c9Cd3B', protocolUrl: 'https://app.compound.finance' },
  { id: '3', name: 'Yearn Finance', category: 'yield', tvl: '$3.2B', apy: '5-15%', chains: [1], logo: '📊', contractAddress: '0x0d53E096a3Bc3170bb4A42A0097b13aF55BC4C2e', protocolUrl: 'https://yearn.finance' },
  { id: '4', name: 'Uniswap', category: 'dex', tvl: '$4.1B', apy: '2-8%', chains: [1, 56, 137, 42161], logo: '🦄', contractAddress: '0x1f98431c8aD98523631AE4a59f267346ea31F984', protocolUrl: 'https://app.uniswap.org' },
  { id: '5', name: 'Curve', category: 'dex', tvl: '$2.3B', apy: '3-10%', chains: [1, 56], logo: '📉', contractAddress: '0xD533a949740bb3306d119CC777fa900bA034cd52', protocolUrl: 'https://curve.fi' },
  { id: '6', name: 'Lido', category: 'yield', tvl: '$15.2B', apy: '4.2%', chains: [1], logo: '💧', contractAddress: '0xae7ab96520DE3A18E5e111B5EaAb095312D7fE84', protocolUrl: 'https://stake.lido.fi' },
  { id: '7', name: 'PancakeSwap', category: 'dex', tvl: '$1.5B', apy: '3-8%', chains: [56], logo: '🥞', contractAddress: '0x10ED43C718714eb63d5aA57B78B54704E256024E', protocolUrl: 'https://pancakeswap.finance' },
  { id: '8', name: 'SushiSwap', category: 'dex', tvl: '$1.2B', apy: '2-6%', chains: [1, 56, 137], logo: '🍣', contractAddress: '0x1f98431c8aD98523631AE4a59f267346ea31F984', protocolUrl: 'https://app.sushi.com' },
];

const CHAIN_NAMES: Record<number, string> = {
  1: 'Ethereum',
  56: 'BSC',
  137: 'Polygon',
  42161: 'Arbitrum',
  10: 'Optimism',
  43114: 'Avalanche',
};

export default function DeFiIntegration() {
  const [category, setCategory] = useState('all');
  const [protocols, setProtocols] = useState<DeFiProtocol[]>(FALLBACK_PROTOCOLS);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [userPositions, setUserPositions] = useState<Record<string, { balance: string; valueUSD: string }>>({});

  const loadProtocols = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await fetchAPI<DeFiProtocol[]>('/defi/protocols');
      if (data && data.length > 0) {
        setProtocols(data);
      }
    } catch (err) {
      console.log('Using fallback protocol data');
      setProtocols(FALLBACK_PROTOCOLS);
    } finally {
      setLoading(false);
    }
  }, []);

  const loadUserPositions = useCallback(async () => {
    try {
      const positions = await fetchAPI<Record<string, { balance: string; valueUSD: string }>>('/defi/positions');
      if (positions) {
        setUserPositions(positions);
      }
    } catch (err) {
      console.log('No user positions found');
    }
  }, []);

  useEffect(() => {
    loadProtocols();
    loadUserPositions();
  }, [loadProtocols, loadUserPositions]);

  const handleConnect = async (protocol: DeFiProtocol) => {
    try {
      // Initiate protocol connection via API
      const result = await fetchAPI<{ redirectUrl: string }>('/defi/connect', {
        method: 'POST',
        body: JSON.stringify({ protocolId: protocol.id }),
      });
      
      if (result.redirectUrl) {
        window.open(result.redirectUrl, '_blank');
      }
    } catch (err) {
      // Fallback: open protocol URL directly
      if (protocol.protocolUrl) {
        window.open(protocol.protocolUrl, '_blank');
      }
    }
  };

  const filteredProtocols = category === 'all' 
    ? protocols 
    : protocols.filter(p => p.category === category);

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-900 text-slate-900 dark:text-white">
      <header className="bg-white dark:bg-slate-800 border-b p-4">
        <div className="flex items-center gap-4 max-w-7xl mx-auto">
          <a href="/" className="text-2xl">🐯</a>
          <h1 className="text-xl font-bold">DeFi Integration</h1>
          <div className="ml-auto flex items-center gap-2">
            <button onClick={loadProtocols} className="text-sm text-orange-500 hover:text-orange-600">
              Refresh
            </button>
          </div>
        </div>
      </header>
      
      <div className="max-w-7xl mx-auto p-8">
        {error && (
          <div className="bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400 p-4 rounded-lg mb-6">
            {error}
          </div>
        )}
        
        <div className="flex gap-2 mb-6 flex-wrap">
          {['all', 'lending', 'yield', 'dex'].map(cat => (
            <button
              key={cat}
              onClick={() => setCategory(cat)}
              className={`px-4 py-2 rounded-lg transition-colors ${
                category === cat 
                  ? 'bg-orange-500 text-white' 
                  : 'bg-slate-200 dark:bg-slate-700 hover:bg-slate-300 dark:hover:bg-slate-600'
              }`}
            >
              {cat.charAt(0).toUpperCase() + cat.slice(1)}
            </button>
          ))}
        </div>

        {loading ? (
          <div className="flex items-center justify-center py-20">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-orange-500"></div>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {filteredProtocols.map(protocol => (
              <div key={protocol.id} className="bg-white dark:bg-slate-800 rounded-lg p-4 shadow-sm border border-slate-200 dark:border-slate-700">
                <div className="flex items-center gap-3 mb-3">
                  <div className="text-3xl">{protocol.logo}</div>
                  <div>
                    <div className="font-semibold">{protocol.name}</div>
                    <div className="text-xs text-slate-500 capitalize">{protocol.category}</div>
                  </div>
                </div>
                <div className="grid grid-cols-2 gap-2 text-sm mb-3">
                  <div>
                    <span className="text-slate-500">TVL:</span>{' '}
                    <span className="font-medium">{protocol.tvl}</span>
                  </div>
                  <div>
                    <span className="text-slate-500">APY:</span>{' '}
                    <span className="font-medium text-green-500">{protocol.apy}</span>
                  </div>
                </div>
                <div className="text-xs text-slate-500 mb-3">
                  Chains: {protocol.chains.map(c => CHAIN_NAMES[c] || `Chain ${c}`).join(', ')}
                </div>
                
                {userPositions[protocol.id] && (
                  <div className="bg-slate-100 dark:bg-slate-700 rounded p-2 mb-3 text-sm">
                    <div className="text-slate-500">Your Position</div>
                    <div className="font-medium">${userPositions[protocol.id].valueUSD}</div>
                  </div>
                )}
                
                <button 
                  onClick={() => handleConnect(protocol)}
                  className="w-full bg-orange-500 hover:bg-orange-600 text-white py-2 rounded-lg transition-colors"
                >
                  {userPositions[protocol.id] ? 'Manage' : 'Connect'}
                </button>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
