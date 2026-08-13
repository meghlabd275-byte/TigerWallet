'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { useTheme } from '../components/ThemeProvider';

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

// API Base URL - same-origin in browser (proxied via Next.js API routes)
const API_BASE_URL = typeof window !== 'undefined' ? '' : (process.env.BACKEND_URL || 'http://localhost:8443');

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

// Fallback data if API is unavailable — real contract addresses, honest TVL/APY
const FALLBACK_PROTOCOLS: DeFiProtocol[] = [
  { id: 'aave', name: 'Aave', category: 'lending', tvl: '—', apy: '—', chains: [1, 137, 56], logo: '👻', contractAddress: '0x7Fc66500c84A76Ad7e9c93437bFc5Ac33E2DDaE9', protocolUrl: 'https://app.aave.com' },
  { id: 'compound', name: 'Compound', category: 'lending', tvl: '—', apy: '—', chains: [1], logo: '📊', contractAddress: '0xc00e94Cb662C3520282E6f5717214004A7f26888', protocolUrl: 'https://app.compound.finance' },
  { id: 'yearn', name: 'Yearn Finance', category: 'yield', tvl: '—', apy: '—', chains: [1], logo: '📊', contractAddress: '0x0bc529c00C6401aEF6D220BE8C6Ea1667F6Ad93e', protocolUrl: 'https://yearn.finance' },
  { id: 'uniswap', name: 'Uniswap', category: 'dex', tvl: '—', apy: '—', chains: [1, 56, 137, 42161], logo: '🦄', contractAddress: '0x1f9840a85d5aF5bf1D1762F925BDADdC4201F984', protocolUrl: 'https://app.uniswap.org' },
  { id: 'curve', name: 'Curve', category: 'dex', tvl: '—', apy: '—', chains: [1, 56], logo: '🔵', contractAddress: '0xD533a949740bb3306d119CC777fa900bA034cd52', protocolUrl: 'https://curve.fi' },
  { id: 'lido', name: 'Lido', category: 'yield', tvl: '—', apy: '—', chains: [1], logo: '💧', contractAddress: '0xae7ab96520DE3A18E5e111B5EaAb095312D7fE84', protocolUrl: 'https://stake.lido.fi' },
  { id: 'pancake', name: 'PancakeSwap', category: 'dex', tvl: '—', apy: '—', chains: [56], logo: '🥞', contractAddress: '0x18BF1C73aC38B4e2c60c2b1a3a3cE33c38D78f3E', protocolUrl: 'https://pancakeswap.finance' },
  { id: 'sushi', name: 'SushiSwap', category: 'dex', tvl: '—', apy: '—', chains: [1, 56, 137], logo: '🍣', contractAddress: '0x6B3595068778DD592e39A122f4f5a5cF09C90fE2', protocolUrl: 'https://www.sushi.com' },
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
  const { isDark } = useTheme();

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
    <div className={`min-h-screen ${isDark ? 'bg-gray-900 text-white' : 'bg-gray-50 text-gray-900'}`}>
      <header className={`${isDark ? 'bg-gray-800' : 'bg-white border border-gray-200'} border-b p-4`}>
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
          <div className={`${isDark ? 'bg-red-900/20 text-red-400' : 'bg-red-50 text-red-600'} p-4 rounded-lg mb-6`}>
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
                  : isDark ? 'bg-gray-700 hover:bg-gray-600' : 'bg-gray-200 hover:bg-gray-300'
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
              <div key={protocol.id} className={`${isDark ? 'bg-gray-800' : 'bg-white'} rounded-lg p-4 shadow-sm border ${isDark ? 'border-gray-700' : 'border-gray-200'}`}>
                <div className="flex items-center gap-3 mb-3">
                  <div className="text-3xl">{protocol.logo}</div>
                  <div>
                    <div className="font-semibold">{protocol.name}</div>
                    <div className={`text-xs ${isDark ? 'text-gray-400' : 'text-gray-500'} capitalize`}>{protocol.category}</div>
                  </div>
                </div>
                <div className="grid grid-cols-2 gap-2 text-sm mb-3">
                  <div>
                    <span className={isDark ? 'text-gray-400' : 'text-gray-500'}>TVL:</span>{' '}
                    <span className="font-medium">{protocol.tvl}</span>
                  </div>
                  <div>
                    <span className={isDark ? 'text-gray-400' : 'text-gray-500'}>APY:</span>{' '}
                    <span className="font-medium text-green-500">{protocol.apy}</span>
                  </div>
                </div>
                <div className={`text-xs ${isDark ? 'text-gray-400' : 'text-gray-500'} mb-3`}>
                  Chains: {protocol.chains.map(c => CHAIN_NAMES[c] || `Chain ${c}`).join(', ')}
                </div>
                
                {userPositions[protocol.id] && (
                  <div className={`${isDark ? 'bg-gray-700' : 'bg-gray-100'} rounded p-2 mb-3 text-sm`}>
                    <div className={isDark ? 'text-gray-400' : 'text-gray-500'}>Your Position</div>
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
