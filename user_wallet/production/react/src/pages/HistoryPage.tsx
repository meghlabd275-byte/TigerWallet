/**
 * History Page - Transaction history
 */

import React, { useState } from 'react';
import { useTheme } from '../contexts/ThemeContext';

function HistoryPage() {
  const { theme } = useTheme();
  const [filter, setFilter] = useState('all');

  const transactions = [
    { id: '0x1234...', type: 'Send', token: 'ETH', amount: '-1.5', status: 'Confirmed', time: '2 hours ago', to: '0x5678...' },
    { id: '0xabcd...', type: 'Receive', token: 'USDT', amount: '+500', status: 'Confirmed', time: '5 hours ago', from: '0xefgh...' },
    { id: '0x9876...', type: 'Swap', token: 'ETH→USDT', amount: '2 ETH', status: 'Confirmed', time: '1 day ago' },
    { id: '0x5432...', type: 'Stake', token: 'MATIC', amount: '100 MATIC', status: 'Confirmed', time: '2 days ago' },
    { id: '0xdcba...', type: 'Unstake', token: 'MATIC', amount: '+50 MATIC', status: 'Pending', time: '3 days ago' },
  ];

  const filtered = filter === 'all' ? transactions : transactions.filter(t => t.type.toLowerCase() === filter);

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">Transaction History</h1>

      <div className="flex gap-2 mb-6 overflow-x-auto">
        {['all', 'send', 'receive', 'swap', 'stake'].map(f => (
          <button key={f} onClick={() => setFilter(f)} className={`px-4 py-2 rounded-lg whitespace-nowrap ${filter === f ? 'bg-amber-500 text-black' : theme === 'dark' ? 'bg-slate-800' : 'bg-gray-200'}`}>
            {f.charAt(0).toUpperCase() + f.slice(1)}
          </button>
        ))}
      </div>

      <div className="space-y-3">
        {filtered.map((tx, i) => (
          <div key={i} className={`card ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
            <div className="flex justify-between items-start">
              <div>
                <div className="flex items-center gap-2">
                  <span className={`px-2 py-1 rounded text-xs ${
                    tx.type === 'Send' ? 'bg-red-500/20 text-red-500' :
                    tx.type === 'Receive' ? 'bg-green-500/20 text-green-500' :
                    tx.type === 'Swap' ? 'bg-blue-500/20 text-blue-500' :
                    'bg-purple-500/20 text-purple-500'
                  }`}>{tx.type}</span>
                  <span className="font-mono text-sm opacity-60">{tx.id}</span>
                </div>
                <p className="text-sm opacity-60 mt-1">{tx.time}</p>
              </div>
              <div className="text-right">
                <p className={`font-bold ${tx.amount.startsWith('+') ? 'text-green-500' : ''}`}>{tx.amount} {tx.token}</p>
                <span className={`badge ${tx.status === 'Confirmed' ? 'badge-success' : 'badge-warning'}`}>{tx.status}</span>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

export default HistoryPage;
