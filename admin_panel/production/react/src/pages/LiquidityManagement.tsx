/**
 * Liquidity Management
 */

import React, { useState } from 'react';

function LiquidityManagement() {
  const [activeTab, setActiveTab] = useState('pools');

  const pools = [
    { id: 1, pair: 'ETH/USDT', tvl: '$5.2M', apr: '12.5%', volume24h: '$12.5M', providers: 156 },
    { id: 2, pair: 'BTC/USDT', tvl: '$15.8M', apr: '8.2%', volume24h: '$25.3M', providers: 289 },
    { id: 3, pair: 'SOL/USDT', tvl: '$2.1M', apr: '18.5%', volume24h: '$5.2M', providers: 78 },
    { id: 4, pair: 'BNB/USDT', tvl: '$4.5M', apr: '15.2%', volume24h: '$8.9M', providers: 124 },
  ];

  const importFromCEX = ['Binance', 'Coinbase', 'Kraken', 'KuCoin', 'Bybit', 'OKX', 'Gate.io', 'Huobi'];

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">Liquidity Management</h1>
        <button className="btn btn-primary">+ Add Pool</button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-slate-800 p-4 rounded-lg">
          <p className="text-sm opacity-60">Total TVL</p>
          <p className="text-2xl font-bold text-amber-500">$27.6M</p>
        </div>
        <div className="bg-slate-800 p-4 rounded-lg">
          <p className="text-sm opacity-60">Active Pools</p>
          <p className="text-2xl font-bold">4</p>
        </div>
        <div className="bg-slate-800 p-4 rounded-lg">
          <p className="text-sm opacity-60">24h Volume</p>
          <p className="text-2xl font-bold">$51.9M</p>
        </div>
        <div className="bg-slate-800 p-4 rounded-lg">
          <p className="text-sm opacity-60">Total Providers</p>
          <p className="text-2xl font-bold">647</p>
        </div>
      </div>

      <div className="flex gap-2 mb-6">
        {['Pools', 'Import', 'Rewards'].map(tab => (
          <button key={tab} onClick={() => setActiveTab(tab.toLowerCase())} className={`px-4 py-2 rounded-lg ${activeTab === tab.toLowerCase() ? 'bg-amber-500 text-black' : 'bg-slate-800'}`}>
            {tab}
          </button>
        ))}
      </div>

      {activeTab === 'pools' && (
        <div className="bg-slate-800 rounded-lg overflow-hidden">
          <table className="w-full">
            <thead className="bg-slate-700">
              <tr>
                <th className="p-3 text-left">Pair</th>
                <th className="p-3 text-left">TVL</th>
                <th className="p-3 text-left">APR</th>
                <th className="p-3 text-left">24h Volume</th>
                <th className="p-3 text-left">Providers</th>
                <th className="p-3 text-left">Actions</th>
              </tr>
            </thead>
            <tbody>
              {pools.map(pool => (
                <tr key={pool.id} className="border-t border-slate-700">
                  <td className="p-3 font-bold text-amber-500">{pool.pair}</td>
                  <td className="p-3">{pool.tvl}</td>
                  <td className="p-3 text-green-500">{pool.apr}</td>
                  <td className="p-3">{pool.volume24h}</td>
                  <td className="p-3">{pool.providers}</td>
                  <td className="p-3">
                    <div className="flex gap-2">
                      <button className="btn btn-secondary text-xs">Manage</button>
                      <button className="btn btn-danger text-xs">Close</button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {activeTab === 'import' && (
        <div className="bg-slate-800 p-6 rounded-lg">
          <h3 className="font-semibold mb-4">Import Liquidity from CEX</h3>
          <p className="text-sm opacity-60 mb-4">Select an exchange to import liquidity pools:</p>
          <div className="grid grid-cols-3 md:grid-cols-4 gap-3">
            {importFromCEX.map(cex => (
              <button key={cex} className="p-3 bg-slate-700 rounded-lg hover:bg-slate-600 transition">
                {cex}
              </button>
            ))}
          </div>
        </div>
      )}

      {activeTab === 'rewards' && (
        <div className="bg-slate-800 p-6 rounded-lg">
          <h3 className="font-semibold mb-4">Liquidity Rewards</h3>
          <div className="space-y-4 max-w-md">
            <div>
              <label className="label">Base APR (%)</label>
              <input type="number" className="input" defaultValue="10" />
            </div>
            <div>
              <label className="label">Boost Multiplier (max)</label>
              <input type="number" className="input" defaultValue="2.5" />
            </div>
            <button className="btn btn-primary">Update Rewards</button>
          </div>
        </div>
      )}
    </div>
  );
}

export default LiquidityManagement;
