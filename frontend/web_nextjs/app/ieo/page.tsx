'use client';

import React, { useState } from 'react';

interface IEOProject {
  id: string;
  name: string;
  symbol: string;
  description: string;
  price: number;
  hardCap: number;
  softCap: number;
  raised: number;
  participants: number;
  status: 'upcoming' | 'sale' | 'ended';
  startTime: number;
  endTime: number;
  minBuy: number;
  maxBuy: number;
  chain: string;
  logo: string;
  tokenAllocation: number;
  listingPrice: number;
}

const MOCK_IEO: IEOProject[] = [
  {
    id: '1',
    name: 'TigerLaunch',
    symbol: 'TIGL',
    description: 'The next generation launchpad platform with AI analytics',
    price: 0.05,
    hardCap: 500000,
    softCap: 100000,
    raised: 385000,
    participants: 2850,
    status: 'sale',
    startTime: Date.now() - 86400000 * 2,
    endTime: Date.now() + 86400000 * 5,
    minBuy: 50,
    maxBuy: 5000,
    chain: 'Ethereum',
    logo: '🚀',
    tokenAllocation: 10000000,
    listingPrice: 0.08,
  },
  {
    id: '2',
    name: 'BlockVision',
    symbol: 'BVIS',
    description: 'AI-powered blockchain analytics and insights platform',
    price: 0.12,
    hardCap: 300000,
    softCap: 50000,
    raised: 0,
    participants: 0,
    status: 'upcoming',
    startTime: Date.now() + 86400000 * 7,
    endTime: Date.now() + 86400000 * 14,
    minBuy: 100,
    maxBuy: 3000,
    chain: 'BNB Chain',
    logo: '🔮',
    tokenAllocation: 2500000,
    listingPrice: 0.18,
  },
  {
    id: '3',
    name: 'CryptoShield',
    symbol: 'CSHIELD',
    description: 'Decentralized insurance protocol for crypto assets',
    price: 0.08,
    hardCap: 250000,
    softCap: 50000,
    raised: 250000,
    participants: 1520,
    status: 'ended',
    startTime: Date.now() - 86400000 * 10,
    endTime: Date.now() - 86400000 * 3,
    minBuy: 50,
    maxBuy: 2500,
    chain: 'Polygon',
    logo: '🛡️',
    tokenAllocation: 3125000,
    listingPrice: 0.12,
  },
];

export default function IEOPage() {
  const [activeTab, setActiveTab] = useState<'upcoming' | 'sale' | 'ended'>('sale');
  const [selectedProject, setSelectedProject] = useState<IEOProject | null>(null);
  const [buyAmount, setBuyAmount] = useState('');
  const [loading, setLoading] = useState(false);

  const filteredProjects = MOCK_IEO.filter(p => p.status === activeTab);

  const handleBuy = async () => {
    if (!selectedProject || !buyAmount) return;
    setLoading(true);
    await new Promise(r => setTimeout(r, 2000));
    alert(`Successfully purchased ${(parseFloat(buyAmount) / selectedProject.price).toFixed(2)} ${selectedProject.symbol}!`);
    setBuyAmount('');
    setSelectedProject(null);
    setLoading(false);
  };

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-900">
      <header className="bg-gradient-to-r from-orange-600 to-red-600 text-white">
        <div className="max-w-7xl mx-auto px-4 py-6">
          <div className="flex items-center gap-4">
            <a href="/" className="text-3xl">🐯</a>
            <div>
              <h1 className="text-2xl font-bold">IEO / IDO</h1>
              <p className="text-orange-200">Invest in early stage projects</p>
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
              className={`px-6 py-2 rounded-lg font-medium ${activeTab === tab ? 'bg-orange-600 text-white' : 'bg-white dark:bg-slate-800'}`}
            >
              {tab.charAt(0).toUpperCase() + tab.slice(1)}
            </button>
          ))}
        </div>

        <div className="space-y-4">
          {filteredProjects.map(project => (
            <div key={project.id} className="bg-white dark:bg-slate-800 rounded-xl p-6 border">
              <div className="flex items-start justify-between">
                <div className="flex items-center gap-4">
                  <span className="text-5xl">{project.logo}</span>
                  <div>
                    <h3 className="text-xl font-bold">{project.name}</h3>
                    <p className="text-slate-500">{project.symbol} • {project.chain}</p>
                  </div>
                </div>
                <span className={`px-3 py-1 rounded-full text-sm ${
                  project.status === 'sale' ? 'bg-green-100 text-green-800' :
                  project.status === 'upcoming' ? 'bg-blue-100 text-blue-800' :
                  'bg-gray-100 text-gray-800'
                }`}>
                  {project.status.toUpperCase()}
                </span>
              </div>

              <p className="text-slate-500 mt-4">{project.description}</p>

              <div className="grid grid-cols-4 gap-4 mt-6">
                <div>
                  <p className="text-xs text-slate-500">Price</p>
                  <p className="font-bold">${project.price}</p>
                </div>
                <div>
                  <p className="text-xs text-slate-500">Raised</p>
                  <p className="font-bold">${project.raised.toLocaleString()}</p>
                </div>
                <div>
                  <p className="text-xs text-slate-500">Hard Cap</p>
                  <p className="font-bold">${project.hardCap.toLocaleString()}</p>
                </div>
                <div>
                  <p className="text-xs text-slate-500">Participants</p>
                  <p className="font-bold">{project.participants.toLocaleString()}</p>
                </div>
              </div>

              {project.status === 'sale' && (
                <div className="mt-6">
                  <div className="h-3 bg-slate-200 rounded-full overflow-hidden">
                    <div 
                      className="h-full bg-gradient-to-r from-orange-500 to-red-500"
                      style={{ width: `${(project.raised / project.hardCap) * 100}%` }}
                    />
                  </div>
                  <p className="text-sm text-slate-500 mt-2">{((project.raised / project.hardCap) * 100).toFixed(1)}% filled</p>
                  <button
                    onClick={() => setSelectedProject(project)}
                    className="w-full mt-4 py-3 bg-gradient-to-r from-orange-600 to-red-600 text-white rounded-lg font-semibold"
                  >
                    Buy Now
                  </button>
                </div>
              )}
            </div>
          ))}
        </div>
      </div>

      {selectedProject && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-slate-800 rounded-xl p-6 max-w-md">
            <h3 className="text-xl font-bold mb-4">Buy {selectedProject.symbol}</h3>
            <div className="space-y-4">
              <div>
                <label className="block text-sm mb-2">Amount (USD)</label>
                <input
                  type="number"
                  value={buyAmount}
                  onChange={(e) => setBuyAmount(e.target.value)}
                  placeholder={`Min: $${selectedProject.minBuy}, Max: $${selectedProject.maxBuy}`}
                  className="w-full p-3 border rounded-lg"
                />
              </div>
              <div className="p-3 bg-slate-50 rounded-lg">
                <p className="text-sm">You will receive: {buyAmount ? (parseFloat(buyAmount) / selectedProject.price).toFixed(2) : '0'} {selectedProject.symbol}</p>
              </div>
              <div className="flex gap-4">
                <button onClick={() => setSelectedProject(null)} className="flex-1 py-3 bg-slate-200 rounded-lg">Cancel</button>
                <button onClick={handleBuy} disabled={loading} className="flex-1 py-3 bg-orange-600 text-white rounded-lg">
                  {loading ? 'Processing...' : 'Confirm Purchase'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
