/**
 * Transactions - Admin Console
 */
import React, { useEffect, useState } from 'react';
import { adminConsoleApi } from '../services/api';

export default function Transactions() {
  const [txs, setTxs] = useState<any[]>([]);
  useEffect(() => { adminConsoleApi.getTransactions().then(d => setTxs(d.data || [])).catch(console.error); }, []);
  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">Transactions</h1>
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50 dark:bg-gray-700"><tr><th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Hash</th><th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">From</th><th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">To</th><th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Amount</th><th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th></tr></thead>
          <tbody className="divide-y divide-gray-200">
            {txs.map(tx => (<tr key={tx.id}><td className="px-6 py-4 font-mono text-sm">{tx.hash?.substring(0, 10)}...</td><td className="px-6 py-4 font-mono text-sm">{tx.from?.substring(0, 10)}...</td><td className="px-6 py-4 font-mono text-sm">{tx.to?.substring(0, 10)}...</td><td className="px-6 py-4">{tx.amount}</td><td className="px-6 py-4"><span className={`px-2 py-1 text-xs rounded ${tx.status === 'completed' ? 'bg-green-100 text-green-800' : 'bg-yellow-100 text-yellow-800'}`}>{tx.status}</span></td></tr>))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
