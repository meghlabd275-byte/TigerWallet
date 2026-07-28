/**
 * Token Management - Complete Token Management
 */

import React, { useState } from 'react';

function TokenManagement() {
  const [activeTab, setActiveTab] = useState('tokens');
  
  const tokens = [
    { symbol: 'ETH', name: 'Ethereum', chain: 'Ethereum', decimals: 18, status: 'Active', balance: '1,234.56' },
    { symbol: 'BTC', name: 'Bitcoin', chain: 'Bitcoin', decimals: 8, status: 'Active', balance: '45.23' },
    { symbol: 'USDT', name: 'Tether USD', chain: 'Multi', decimals: 6, status: 'Active', balance: '567,890' },
    { symbol: 'USDC', name: 'USD Coin', chain: 'Multi', decimals: 6, status: 'Active', balance: '234,567' },
    { symbol: 'BNB', name: 'BNB', chain: 'BNB Chain', decimals: 18, status: 'Active', balance: '8,901' },
    { symbol: 'MATIC', name: 'Polygon', chain: 'Polygon', decimals: 18, status: 'Paused', balance: '123,456' },
  ];

  const addToken = () => {
    // Real implementation would open modal
    console.log('Add token');
  };

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">Token Management</h1>
        <button onClick={addToken} className="btn btn-primary">+ Add Token</button>
      </div>

      {/* Tabs */}
      <div className="flex gap-2 mb-6">
        {['Tokens', 'Create Token', 'Import', 'Settings'].map(tab => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab.toLowerCase().replace(' ', ''))}
            className={`px-4 py-2 rounded-lg ${
              activeTab === tab.toLowerCase().replace(' ', '') ? 'bg-amber-500 text-black' : 'bg-slate-800'
            }`}
          >
            {tab}
          </button>
        ))}
      </div>

      {/* Tokens Table */}
      {activeTab === 'tokens' && (
        <div className="bg-slate-800 rounded-lg overflow-hidden">
          <table className="w-full">
            <thead className="bg-slate-700">
              <tr>
                <th className="p-3 text-left">Symbol</th>
                <th className="p-3 text-left">Name</th>
                <th className="p-3 text-left">Chain</th>
                <th className="p-3 text-left">Decimals</th>
                <th className="p-3 text-left">Status</th>
                <th className="p-3 text-left">Balance</th>
                <th className="p-3 text-left">Actions</th>
              </tr>
            </thead>
            <tbody>
              {tokens.map((token, i) => (
                <tr key={i} className="border-t border-slate-700">
                  <td className="p-3 font-bold text-amber-500">{token.symbol}</td>
                  <td className="p-3">{token.name}</td>
                  <td className="p-3">{token.chain}</td>
                  <td className="p-3">{token.decimals}</td>
                  <td className="p-3">
                    <span className={`px-2 py-1 rounded text-xs ${
                      token.status === 'Active' ? 'bg-green-500/20 text-green-500' : 'bg-yellow-500/20 text-yellow-500'
                    }`}>
                      {token.status}
                    </span>
                  </td>
                  <td className="p-3">{token.balance}</td>
                  <td className="p-3">
                    <div className="flex gap-2">
                      <button className="btn btn-secondary text-xs">Edit</button>
                      <button className="btn btn-danger text-xs">{token.status === 'Active' ? 'Pause' : 'Activate'}</button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Create Token */}
      {activeTab === 'createtoken' && (
        <div className="bg-slate-800 p-6 rounded-lg">
          <h3 className="font-semibold mb-4">Create New Token</h3>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="label">Token Name</label>
              <input type="text" className="input" placeholder="My Token" />
            </div>
            <div>
              <label className="label">Symbol</label>
              <input type="text" className="input" placeholder="MTK" />
            </div>
            <div>
              <label className="label">Decimals</label>
              <input type="number" className="input" defaultValue={18} />
            </div>
            <div>
              <label className="label">Chain</label>
              <select className="input">
                <option>Ethereum</option>
                <option>BNB Chain</option>
                <option>Polygon</option>
                <option>Solana</option>
              </select>
            </div>
            <div>
              <label className="label">Initial Supply</label>
              <input type="number" className="input" placeholder="1000000" />
            </div>
            <div>
              <label className="label">Max Supply</label>
              <input type="number" className="input" placeholder="10000000" />
            </div>
          </div>
          <button className="btn btn-primary mt-4">Deploy Token</button>
        </div>
      )}
    </div>
  );
}

export default TokenManagement;
