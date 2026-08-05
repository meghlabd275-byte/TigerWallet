/**
 * Transactions Page - White Label Admin
 */

import React, { useEffect, useState } from 'react';
import { whiteLabelAdminApi } from '../services/api';

export default function Transactions() {
  const [transactions, setTransactions] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => { loadTransactions(); }, []);

  const loadTransactions = async () => {
    try {
      const data = await whiteLabelAdminApi.getTransactions({ pageSize: 50 });
      setTransactions(data.data || []);
    } catch (error) { console.error('Failed:', error); }
    finally { setLoading(false); }
  };

  if (loading) return <div className="p-8">Loading...</div>;

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">Transactions</h1>
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50 dark:bg-gray-700">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Hash</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">From</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">To</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Amount</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {transactions.map((tx) => (
              <tr key={tx.id}>
                <td className="px-6 py-4 font-mono text-sm">{tx.hash?.substring(0, 10)}...</td>
                <td className="px-6 py-4 font-mono text-sm">{tx.from?.substring(0, 10)}...</td>
                <td className="px-6 py-4 font-mono text-sm">{tx.to?.substring(0, 10)}...</td>
                <td className="px-6 py-4">{tx.amount} {tx.token}</td>
                <td className="px-6 py-4"><span className={`px-2 py-1 text-xs rounded ${tx.status === 'completed' ? 'bg-green-100 text-green-800' : tx.status === 'pending' ? 'bg-yellow-100 text-yellow-800' : 'bg-red-100 text-red-800'}`}>{tx.status}</span></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
