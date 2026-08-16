/**
 * Transactions Page - White Label Admin
 */

import React, { useEffect, useState } from 'react';
import { whiteLabelAdminApi } from '../services/api';
import { useTheme } from '../context/ThemeContext';

export default function Transactions() {
  const { isDark } = useTheme();
  const [transactions, setTransactions] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => { loadTransactions(); }, []);

  const loadTransactions = async () => {
    setLoading(true); setError('');
    try {
      const data = await whiteLabelAdminApi.getTransactions();
      setTransactions(data.transactions || []);
    } catch (e: any) { setError(e.message || 'Failed to load transactions'); }
    finally { setLoading(false); }
  };

  const cardBg = isDark ? 'bg-gray-800' : 'bg-white';
  const cardText = isDark ? 'text-white' : 'text-gray-900';
  const border = isDark ? 'border-gray-700' : 'border-gray-200';
  const muted = isDark ? 'text-gray-400' : 'text-gray-500';
  const thBg = isDark ? 'bg-gray-700' : 'bg-gray-50';

  const statusBadge = (status: string) => {
    const base = 'px-2 py-1 text-xs rounded';
    if (status === 'confirmed' || status === 'completed') return `${base} ${isDark ? 'bg-green-900 text-green-200' : 'bg-green-100 text-green-800'}`;
    if (status === 'pending') return `${base} ${isDark ? 'bg-yellow-900 text-yellow-200' : 'bg-yellow-100 text-yellow-800'}`;
    if (status === 'flagged') return `${base} ${isDark ? 'bg-red-900 text-red-200' : 'bg-red-100 text-red-800'}`;
    return `${base} ${isDark ? 'bg-gray-700 text-gray-300' : 'bg-gray-200 text-gray-700'}`;
  };

  const short = (v: string) => (v ? `${v.substring(0, 10)}…` : '—');

  return (
    <div className="p-6">
      <h1 className={`text-2xl font-bold mb-6 ${cardText}`}>Transactions</h1>
      {error && <div className={`mb-4 p-3 rounded ${isDark ? 'bg-red-900/50 text-red-200' : 'bg-red-50 text-red-700'}`}>{error}</div>}
      {loading && <div className={muted}>Loading...</div>}
      {!loading && (
        <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border}`}>
          <table className="w-full">
            <thead className={thBg}>
              <tr>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Type</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>From</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>To</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Amount</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Fee</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Status</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Time</th>
              </tr>
            </thead>
            <tbody className={`divide-y ${border}`}>
              {transactions.length === 0 && (
                <tr><td colSpan={7} className={`px-6 py-8 text-center ${muted}`}>No transactions found.</td></tr>
              )}
              {transactions.map((tx) => (
                <tr key={tx.id}>
                  <td className={`px-6 py-4 ${cardText}`}>{tx.type}</td>
                  <td className={`px-6 py-4 font-mono text-xs ${muted}`}>{short(tx.from_address)}</td>
                  <td className={`px-6 py-4 font-mono text-xs ${muted}`}>{short(tx.to_address)}</td>
                  <td className={`px-6 py-4 ${cardText}`}>{tx.amount} {tx.currency}</td>
                  <td className={`px-6 py-4 ${muted}`}>{tx.fee}</td>
                  <td className="px-6 py-4"><span className={statusBadge(tx.status)}>{tx.status}</span></td>
                  <td className={`px-6 py-4 text-xs ${muted}`}>{tx.timestamp ? new Date(tx.timestamp).toLocaleString() : '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
