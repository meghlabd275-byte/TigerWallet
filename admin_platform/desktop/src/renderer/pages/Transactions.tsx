import React, { useEffect, useState } from 'react';
import { useTheme } from '../context/ThemeContext';
import { transactionService } from '../services/api';

interface Transaction {
  id: string;
  tx_hash: string | null;
  user_id: string;
  type: string;
  status: string;
  chain_id: string;
  amount: string;
  fee: string;
  created_at: string;
}

export default function Transactions() {
  const { isDark } = useTheme();
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [loading, setLoading] = useState(true);
  const [typeFilter, setTypeFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [page, setPage] = useState(1);

  useEffect(() => {
    loadTransactions();
  }, [page, typeFilter, statusFilter]);

  const loadTransactions = async () => {
    setLoading(true);
    try {
      const response = await transactionService.getTransactions({ page, limit: 20, type: typeFilter || undefined, status: statusFilter || undefined });
      setTransactions(response.data);
    } catch (err) {
      console.error('Failed to load transactions:', err);
    } finally {
      setLoading(false);
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'completed': return 'bg-green-100 text-green-800';
      case 'pending': return 'bg-yellow-100 text-yellow-800';
      case 'failed': return 'bg-red-100 text-red-800';
      default: return 'bg-gray-100 text-gray-800';
    }
  };

  const getTypeColor = (type: string) => {
    switch (type) {
      case 'deposit': return 'bg-green-100 text-green-800';
      case 'withdrawal': return 'bg-red-100 text-red-800';
      case 'transfer': return 'bg-blue-100 text-blue-800';
      case 'swap': return 'bg-purple-100 text-purple-800';
      default: return 'bg-gray-100 text-gray-800';
    }
  };

  return (
    <div>
      <h1 className={`text-3xl font-bold mb-6 ${isDark ? 'text-white' : 'text-gray-900'}`}>Transactions</h1>
      
      <div className={`p-4 rounded-lg shadow mb-6 ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
        <div className="flex gap-4">
          <select value={typeFilter} onChange={(e) => { setTypeFilter(e.target.value); setPage(1); }} className={`px-4 py-2 rounded-lg border ${isDark ? 'bg-gray-700 border-gray-600 text-white' : 'bg-white border-gray-300'}`}>
            <option value="">All Types</option>
            <option value="deposit">Deposit</option>
            <option value="withdrawal">Withdrawal</option>
            <option value="transfer">Transfer</option>
            <option value="swap">Swap</option>
          </select>
          <select value={statusFilter} onChange={(e) => { setStatusFilter(e.target.value); setPage(1); }} className={`px-4 py-2 rounded-lg border ${isDark ? 'bg-gray-700 border-gray-600 text-white' : 'bg-white border-gray-300'}`}>
            <option value="">All Status</option>
            <option value="pending">Pending</option>
            <option value="completed">Completed</option>
            <option value="failed">Failed</option>
          </select>
          <button onClick={loadTransactions} className={`px-4 py-2 rounded-lg ${isDark ? 'bg-blue-600 text-white' : 'bg-blue-500 text-white'}`}>Refresh</button>
        </div>
      </div>

      <div className={`rounded-lg shadow overflow-hidden ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
        {loading ? <div className="p-8 text-center">Loading...</div> : transactions.length === 0 ? <div className="p-8 text-center">No transactions found</div> : (
          <table className="w-full">
            <thead className={isDark ? 'bg-gray-700' : 'bg-gray-50'}>
              <tr>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Hash</th>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Type</th>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Status</th>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Amount</th>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Chain</th>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Date</th>
              </tr>
            </thead>
            <tbody className={`divide-y ${isDark ? 'divide-gray-700' : 'divide-gray-200'}`}>
              {transactions.map((tx) => (
                <tr key={tx.id} className={isDark ? 'hover:bg-gray-700' : 'hover:bg-gray-50'}>
                  <td className={`px-4 py-4 font-mono text-sm ${isDark ? 'text-white' : 'text-gray-900'}`}>{tx.tx_hash ? `${tx.tx_hash.substring(0, 10)}...` : 'N/A'}</td>
                  <td className="px-4 py-4"><span className={`px-2 py-1 text-xs font-medium rounded-full ${getTypeColor(tx.type)}`}>{tx.type}</span></td>
                  <td className="px-4 py-4"><span className={`px-2 py-1 text-xs font-medium rounded-full ${getStatusColor(tx.status)}`}>{tx.status}</span></td>
                  <td className={`px-4 py-4 font-medium ${isDark ? 'text-white' : 'text-gray-900'}`}>{tx.amount}</td>
                  <td className={`px-4 py-4 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>{tx.chain_id}</td>
                  <td className={`px-4 py-4 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>{new Date(tx.created_at).toLocaleString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
