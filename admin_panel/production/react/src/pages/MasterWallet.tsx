/**
 * Master Wallet Management - Complete Admin Features
 */

import React, { useState } from 'react';

function MasterWallet() {
  const [activeTab, setActiveTab] = useState('overview');

  const wallets = [
    { chain: 'Ethereum', address: '0x742d35Cc6634C0532925a3b844Bc9e7595f1234', balance: '1,234.56 ETH', fee: '0.3%' },
    { chain: 'Bitcoin', address: 'bc1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh', balance: '45.23 BTC', fee: '0.5%' },
    { chain: 'BNB Chain', address: '0x8626f6940E2eb28930eFb4CeF49B2d1F2C9C1199', balance: '8,901.12 BNB', fee: '0.3%' },
    { chain: 'Solana', address: '7xKXtg2CW87d97TXJSDpbD5jBkheTqA83TZRuJosgAsU', balance: '12,456.78 SOL', fee: '0.25%' },
    { chain: 'Polygon', address: '0xdD2FD4581271e230360230F9337D5c0430Bf44C0', balance: '567,890.12 MATIC', fee: '0.3%' },
  ];

  const transactions = [
    { id: '0x123...', type: 'Deposit', amount: '10 ETH', status: 'Confirmed', time: '2 min ago' },
    { id: '0x456...', type: 'Withdrawal', amount: '-5 ETH', status: 'Pending', time: '5 min ago' },
    { id: '0x789...', type: 'Fee Collection', amount: '0.015 ETH', status: 'Confirmed', time: '10 min ago' },
  ];

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">Master Wallet Management</h1>

      {/* Tabs */}
      <div className="flex gap-2 mb-6">
        {['Overview', 'Wallets', 'Transactions', 'Fee Config', 'Settings'].map(tab => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab.toLowerCase())}
            className={`px-4 py-2 rounded-lg ${
              activeTab === tab.toLowerCase() ? 'bg-amber-500 text-black' : 'bg-slate-800'
            }`}
          >
            {tab}
          </button>
        ))}
      </div>

      {/* Overview */}
      {activeTab === 'overview' && (
        <>
          {/* Total Stats */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
            <div className="bg-slate-800 p-4 rounded-lg">
              <p className="text-sm opacity-60">Total Balance (USD)</p>
              <p className="text-2xl font-bold text-green-500">$2,456,789.00</p>
            </div>
            <div className="bg-slate-800 p-4 rounded-lg">
              <p className="text-sm opacity-60">Today's Volume</p>
              <p className="text-2xl font-bold">$123,456.00</p>
            </div>
            <div className="bg-slate-800 p-4 rounded-lg">
              <p className="text-sm opacity-60">Fees Collected (Today)</p>
              <p className="text-2xl font-bold text-amber-500">$456.78</p>
            </div>
            <div className="bg-slate-800 p-4 rounded-lg">
              <p className="text-sm opacity-60">Active Networks</p>
              <p className="text-2xl font-bold">5</p>
            </div>
          </div>

          {/* Recent Transactions */}
          <div className="bg-slate-800 rounded-lg">
            <div className="p-4 border-b border-slate-700">
              <h3 className="font-semibold">Recent Transactions</h3>
            </div>
            <table className="w-full">
              <thead className="bg-slate-700">
                <tr>
                  <th className="p-3 text-left">Tx ID</th>
                  <th className="p-3 text-left">Type</th>
                  <th className="p-3 text-left">Amount</th>
                  <th className="p-3 text-left">Status</th>
                  <th className="p-3 text-left">Time</th>
                </tr>
              </thead>
              <tbody>
                {transactions.map((tx, i) => (
                  <tr key={i} className="border-t border-slate-700">
                    <td className="p-3 font-mono text-sm">{tx.id}</td>
                    <td className="p-3">{tx.type}</td>
                    <td className="p-3">{tx.amount}</td>
                    <td className="p-3">
                      <span className={`px-2 py-1 rounded text-xs ${
                        tx.status === 'Confirmed' ? 'bg-green-500/20 text-green-500' : 'bg-yellow-500/20 text-yellow-500'
                      }`}>
                        {tx.status}
                      </span>
                    </td>
                    <td className="p-3 opacity-60">{tx.time}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}

      {/* Wallets */}
      {activeTab === 'wallets' && (
        <div className="space-y-4">
          {wallets.map((wallet, i) => (
            <div key={i} className="bg-slate-800 p-4 rounded-lg">
              <div className="flex justify-between items-start mb-2">
                <div>
                  <h3 className="font-semibold text-amber-500">{wallet.chain}</h3>
                  <p className="font-mono text-sm opacity-60">{wallet.address}</p>
                </div>
                <div className="text-right">
                  <p className="font-bold">{wallet.balance}</p>
                  <p className="text-sm opacity-60">Fee: {wallet.fee}</p>
                </div>
              </div>
              <div className="flex gap-2 mt-3">
                <button className="btn btn-secondary text-sm">View on Explorer</button>
                <button className="btn btn-secondary text-sm">Configure Fee</button>
                <button className="btn btn-secondary text-sm">Withdraw</button>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Fee Configuration */}
      {activeTab === 'fee config' && (
        <div className="bg-slate-800 p-6 rounded-lg">
          <h3 className="font-semibold mb-4">Fee Configuration</h3>
          <div className="space-y-4 max-w-md">
            <div>
              <label className="label">Swap Fee (%)</label>
              <input type="number" defaultValue="0.3" step="0.01" className="input" />
            </div>
            <div>
              <label className="label">Withdrawal Fee (%)</label>
              <input type="number" defaultValue="0.5" step="0.01" className="input" />
            </div>
            <div>
              <label className="label">Deposit Fee (%)</label>
              <input type="number" defaultValue="0" step="0.01" className="input" />
            </div>
            <div>
              <label className="label">Cross-Chain Bridge Fee (%)</label>
              <input type="number" defaultValue="1.0" step="0.01" className="input" />
            </div>
            <button className="btn btn-primary">Save Configuration</button>
          </div>
        </div>
      )}

      {/* Add Network */}
      {activeTab === 'settings' && (
        <div className="bg-slate-800 p-6 rounded-lg">
          <h3 className="font-semibold mb-4">Add New Blockchain</h3>
          <div className="space-y-4 max-w-md">
            <div>
              <label className="label">Blockchain Name</label>
              <input type="text" placeholder="e.g., Arbitrum One" className="input" />
            </div>
            <div>
              <label className="label">Chain ID</label>
              <input type="number" placeholder="e.g., 42161" className="input" />
            </div>
            <div>
              <label className="label">RPC URL</label>
              <input type="text" placeholder="https://..." className="input" />
            </div>
            <div>
              <label className="label">Explorer URL</label>
              <input type="text" placeholder="https://..." className="input" />
            </div>
            <div>
              <label className="label">Native Token Symbol</label>
              <input type="text" placeholder="e.g., ETH" className="input" />
            </div>
            <button className="btn btn-primary">Add Blockchain</button>
          </div>
        </div>
      )}
    </div>
  );
}

export default MasterWallet;
