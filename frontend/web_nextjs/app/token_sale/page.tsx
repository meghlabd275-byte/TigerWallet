'use client';

import React, { useState } from 'react';
import { useTheme } from '../components/ThemeProvider';

interface TokenSale {
  id: string;
  name: string;
  symbol: string;
  description: string;
  salePrice: number;
  listingPrice: number;
  totalSupply: number;
  forSale: number;
  sold: number;
  startTime: number;
  endTime: number;
  status: 'upcoming' | 'sale' | 'ended';
  phases: {
    name: string;
    price: number;
    discount: number;
    allocation: number;
    startTime: number;
    endTime: number;
  }[];
  chain: string;
  logo: string;
  progress: number;
}

const MOCK_SALES: TokenSale[] = [
  {
    id: '1',
    name: 'TigerChain',
    symbol: 'TIGC',
    description: 'Layer 1 blockchain with lightning fast transactions and near-zero fees',
    salePrice: 0.15,
    listingPrice: 0.25,
    totalSupply: 100000000,
    forSale: 30000000,
    sold: 18500000,
    startTime: Date.now() - 86400000 * 3,
    endTime: Date.now() + 86400000 * 11,
    status: 'sale',
    phases: [
      { name: 'Phase 1', price: 0.10, discount: 33, allocation: 10000000, startTime: Date.now() - 86400000 * 3, endTime: Date.now() },
      { name: 'Phase 2', price: 0.15, discount: 0, allocation: 20000000, startTime: Date.now(), endTime: Date.now() + 86400000 * 7 },
    ],
    chain: 'Ethereum',
    logo: '🐯',
    progress: 61.7,
  },
  {
    id: '2',
    name: 'DeFi Pro',
    symbol: 'DFPRO',
    description: 'Professional DeFi tools for institutional investors',
    salePrice: 0.08,
    listingPrice: 0.15,
    totalSupply: 50000000,
    forSale: 15000000,
    sold: 0,
    startTime: Date.now() + 86400000 * 5,
    endTime: Date.now() + 86400000 * 19,
    status: 'upcoming',
    phases: [
      { name: 'Whitelist', price: 0.06, discount: 25, allocation: 5000000, startTime: Date.now() + 86400000 * 5, endTime: Date.now() + 86400000 * 8 },
      { name: 'Public', price: 0.08, discount: 0, allocation: 10000000, startTime: Date.now() + 86400000 * 8, endTime: Date.now() + 86400000 * 19 },
    ],
    chain: 'BNB Chain',
    logo: '💹',
    progress: 0,
  },
  {
    id: '3',
    name: 'NFT Galaxy',
    symbol: 'NXGX',
    description: 'Next-generation NFT marketplace with AI curation',
    salePrice: 0.05,
    listingPrice: 0.10,
    totalSupply: 200000000,
    forSale: 40000000,
    sold: 40000000,
    startTime: Date.now() - 86400000 * 20,
    endTime: Date.now() - 86400000 * 6,
    status: 'ended',
    phases: [
      { name: 'Phase 1', price: 0.03, discount: 40, allocation: 20000000, startTime: Date.now() - 86400000 * 20, endTime: Date.now() - 86400000 * 14 },
      { name: 'Phase 2', price: 0.05, discount: 0, allocation: 20000000, startTime: Date.now() - 86400000 * 14, endTime: Date.now() - 86400000 * 6 },
    ],
    chain: 'Polygon',
    logo: '🌌',
    progress: 100,
  },
];

export default function TokenSalePage() {
  const [activeTab, setActiveTab] = useState<'upcoming' | 'sale' | 'ended'>('sale');
  const [selectedSale, setSelectedSale] = useState<TokenSale | null>(null);
  const [buyAmount, setBuyAmount] = useState('');
  const [selectedPhase, setSelectedPhase] = useState(0);
  const [loading, setLoading] = useState(false);
  const { isDark } = useTheme();

  const filteredSales = MOCK_SALES.filter(s => s.status === activeTab);

  const handleBuy = async () => {
    if (!selectedSale || !buyAmount) return;
    setLoading(true);
    await new Promise(r => setTimeout(r, 2000));
    alert(`Successfully purchased ${(parseFloat(buyAmount) / selectedSale.phases[selectedPhase].price).toFixed(2)} ${selectedSale.symbol}!`);
    setBuyAmount('');
    setSelectedSale(null);
    setLoading(false);
  };

  return (
    <div className={`min-h-screen ${isDark ? 'bg-gray-900 text-white' : 'bg-gray-50 text-gray-900'}`}>
      <header className="bg-gradient-to-r from-indigo-600 to-purple-600 text-white">
        <div className="max-w-7xl mx-auto px-4 py-6">
          <div className="flex items-center gap-4">
            <a href="/" className="text-3xl">🐯</a>
            <div>
              <h1 className="text-2xl font-bold">Token Sale</h1>
              <p className="text-indigo-200">Exclusive token sales at best prices</p>
            </div>
          </div>
        </div>
      </header>

      <div className="max-w-7xl mx-auto px-4 py-6">
        <div className="flex gap-2 mb-6">
          {(['upcoming', 'sale', 'ended'] as const).map(tab => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={`px-6 py-2 rounded-lg font-medium ${activeTab === tab ? 'bg-indigo-600 text-white' : isDark ? 'bg-gray-800' : 'bg-white border border-gray-200'}`}
            >
              {tab.charAt(0).toUpperCase() + tab.slice(1)}
            </button>
          ))}
        </div>

        <div className="space-y-4">
          {filteredSales.map(sale => (
            <div key={sale.id} className={`${isDark ? 'bg-gray-800' : 'bg-white'} rounded-xl p-6 border ${isDark ? 'border-gray-700' : 'border-gray-200'}`}>
              <div className="flex items-start justify-between">
                <div className="flex items-center gap-4">
                  <span className="text-5xl">{sale.logo}</span>
                  <div>
                    <h3 className="text-xl font-bold">{sale.name}</h3>
                    <p className={isDark ? 'text-gray-400' : 'text-gray-500'}>{sale.symbol} • {sale.chain}</p>
                  </div>
                </div>
                <span className={`px-3 py-1 rounded-full text-sm ${
                  sale.status === 'sale' ? 'bg-green-100 text-green-800' :
                  sale.status === 'upcoming' ? 'bg-blue-100 text-blue-800' :
                  'bg-gray-100 text-gray-800'
                }`}>
                  {sale.status.toUpperCase()}
                </span>
              </div>

              <p className="text-slate-500 mt-4">{sale.description}</p>

              <div className="grid grid-cols-4 gap-4 mt-6">
                <div>
                  <p className="text-xs text-slate-500">Current Price</p>
                  <p className="font-bold text-lg">${sale.phases[selectedPhase]?.price || sale.salePrice}</p>
                </div>
                <div>
                  <p className="text-xs text-slate-500">Listing Price</p>
                  <p className="font-bold">${sale.listingPrice}</p>
                </div>
                <div>
                  <p className="text-xs text-slate-500">Sold</p>
                  <p className="font-bold">{sale.sold.toLocaleString()}</p>
                </div>
                <div>
                  <p className="text-xs text-slate-500">Supply</p>
                  <p className="font-bold">{sale.totalSupply.toLocaleString()}</p>
                </div>
              </div>

              {sale.status !== 'ended' && (
                <div className="mt-6">
                  <p className="text-sm font-medium mb-2">Sale Phases</p>
                  <div className="flex gap-2 mb-4">
                    {sale.phases.map((phase, idx) => (
                      <button
                        key={idx}
                        onClick={() => setSelectedPhase(idx)}
                        className={`flex-1 p-2 rounded-lg border ${selectedPhase === idx ? 'bg-indigo-50 border-indigo-500' : 'bg-slate-50'}`}
                      >
                        <p className="font-medium text-sm">{phase.name}</p>
                        <p className="text-xs text-slate-500">${phase.price} {phase.discount > 0 && `(-${phase.discount}%)`}</p>
                      </button>
                    ))}
                  </div>
                </div>
              )}

              {sale.status === 'sale' && (
                <div className="mt-6">
                  <div className="h-3 bg-slate-200 rounded-full overflow-hidden">
                    <div 
                      className="h-full bg-gradient-to-r from-indigo-500 to-purple-500"
                      style={{ width: `${sale.progress}%` }}
                    />
                  </div>
                  <div className="flex justify-between mt-2 text-sm text-slate-500">
                    <span>{sale.sold.toLocaleString()} sold</span>
                    <span>{sale.forSale.toLocaleString()} total</span>
                  </div>
                  <button
                    onClick={() => { setSelectedSale(sale); setSelectedPhase(0); }}
                    className="w-full mt-4 py-3 bg-gradient-to-r from-indigo-600 to-purple-600 text-white rounded-lg font-semibold"
                  >
                    Buy Tokens
                  </button>
                </div>
              )}
            </div>
          ))}
        </div>
      </div>

      {selectedSale && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className={`${isDark ? 'bg-gray-800' : 'bg-white border border-gray-200'} rounded-xl p-6 max-w-md`}>
            <h3 className="text-xl font-bold mb-4">Buy {selectedSale.symbol}</h3>
            <div className="space-y-4">
              <div>
                <label className="block text-sm mb-2">Amount (USD)</label>
                <input
                  type="number"
                  value={buyAmount}
                  onChange={(e) => setBuyAmount(e.target.value)}
                  className="w-full p-3 border rounded-lg"
                />
              </div>
              <div className="p-3 bg-slate-50 rounded-lg">
                <p className="text-sm">You will receive: {buyAmount ? (parseFloat(buyAmount) / selectedSale.phases[selectedPhase].price).toFixed(2) : '0'} {selectedSale.symbol}</p>
                <p className="text-xs text-slate-500 mt-1">Listing at: ${selectedSale.listingPrice} (+{((selectedSale.listingPrice - selectedSale.phases[selectedPhase].price) / selectedSale.phases[selectedPhase].price * 100).toFixed(0)}%)</p>
              </div>
              <div className="flex gap-4">
                <button onClick={() => setSelectedSale(null)} className="flex-1 py-3 bg-slate-200 rounded-lg">Cancel</button>
                <button onClick={handleBuy} disabled={loading} className="flex-1 py-3 bg-indigo-600 text-white rounded-lg">
                  {loading ? 'Processing...' : 'Confirm'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
