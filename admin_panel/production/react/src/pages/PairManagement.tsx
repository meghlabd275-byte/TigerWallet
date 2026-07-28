/**
 * Pair Management - Trading pairs
 */

import React, { useState } from 'react';

function PairManagement() {
  const [activeTab, setActiveTab] = useState('pairs');

  const pairs = [
    { id: 1, pair: 'ETH/USDT', price: '$2,345.67', change: '+2.5%', volume: '$12.5M', liquidity: '$5.2M', status: 'Active' },
    { id: 2, pair: 'BTC/USDT', price: '$45,678.90', change: '+1.2%', volume: '$25.3M', liquidity: '$15.8M', status: 'Active' },
    { id: 3, pair: 'SOL/USDT', price: '$98.45', change: '-0.8%', volume: '$5.2M', liquidity: '$2.1M', status: 'Active' },
    { id: 4, pair: 'MATIC/USDT', price: '$0.89', change: '+5.2%', volume: '$3.1M', liquidity: '$1.2M', status: 'Paused' },
    { id: 5, pair: 'BNB/USDT', price: '$312.45', change: '+0.5%', volume: '$8.9M', liquidity: '$4.5M', status: 'Active' },
  ];

  const importFromCEX = ['Binance', 'Coinbase', 'Kraken', 'KuCoin', 'Bybit', 'OKX', 'Gate.io', 'Huobi', 'Bitfinex', 'Bithumb', 'Crypto.com', 'Gemini', 'Bitstamp', 'eToro', 'Robinhood'];

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">Pair Management</h1>
        <button className="btn btn-primary">+ Create Pair</button>
      </div>

      <div className="flex gap-2 mb-6">
        {['Pairs', 'Import', 'Settings'].map(tab => (
          <button key={tab} onClick={() => setActiveTab(tab.toLowerCase())} className={`px-4 py-2 rounded-lg ${activeTab === tab.toLowerCase() ? 'bg-amber-500 text-black' : 'bg-slate-800'}`}>
            {tab}
          </button>
        ))}
      </div>

      {activeTab === 'pairs' && (
        <div className="bg-slate-800 rounded-lg overflow-hidden">
          <table className="w-full">
            <thead className="bg-slate-700">
              <tr>
                <th className="p-3 text-left">Pair</th>
                <th className="p-3 text-left">Price</th>
                <th className="p-3 text-left">24h Change</th>
                <th className="p-3 text-left">Volume</th>
                <th className="p-3 text-left">Liquidity</th>
                <th className="p-3 text-left">Status</th>
                <th className="p-3 text-left">Actions</th>
              </tr>
            </thead>
            <tbody>
              {pairs.map(pair => (
                <tr key={pair.id} className="border-t border-slate-700">
                  <td className="p-3 font-bold text-amber-500">{pair.pair}</td>
                  <td className="p-3">{pair.price}</td>
                  <td className="p-3">{pair.change.startsWith('+') ? <span className="text-green-500">{pair.change}</span> : <span className="text-red-500">{pair.change}</span>}</td>
                  <td className="p-3">{pair.volume}</td>
                  <td className="p-3">{pair.liquidity}</td>
                  <td className="p-3">
                    <span className={`px-2 py-1 rounded text-xs ${pair.status === 'Active' ? 'bg-green-500/20 text-green-500' : 'bg-yellow-500/20 text-yellow-500'}`}>
                      {pair.status}
                    </span>
                  </td>
                  <td className="p-3">
                    <div className="flex gap-2">
                      <button className="btn btn-secondary text-xs">Edit</button>
                      <button className="btn btn-danger text-xs">{pair.status === 'Active' ? 'Pause' : 'Resume'}</button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {activeTab === 'import' && (
        <div className="space-y-6">
          <div className="bg-slate-800 p-6 rounded-lg">
            <h3 className="font-semibold mb-4">Import Trading Pairs from CEX</h3>
            <p className="text-sm opacity-60 mb-4">Select an exchange to import trading pairs from:</p>
            <div className="grid grid-cols-3 md:grid-cols-5 gap-3">
              {importFromCEX.map(cex => (
                <button key={cex} className="p-3 bg-slate-700 rounded-lg hover:bg-slate-600 transition">
                  {cex}
                </button>
              ))}
            </div>
          </div>
        </div>
      )}

      {activeTab === 'settings' && (
        <div className="bg-slate-800 p-6 rounded-lg">
          <h3 className="font-semibold mb-4">Pair Settings</h3>
          <div className="space-y-4 max-w-md">
            <div>
              <label className="label">Minimum Trade Amount</label>
              <input type="number" className="input" defaultValue="10" />
            </div>
            <div>
              <label className="label">Maximum Trade Amount</label>
              <input type="number" className="input" defaultValue="1000000" />
            </div>
            <div>
              <label className="label">Default Trading Fee (%)</label>
              <input type="number" className="input" defaultValue="0.3" step="0.01" />
            </div>
            <button className="btn btn-primary">Save Settings</button>
          </div>
        </div>
      )}
    </div>
  );
}

export default PairManagement;
