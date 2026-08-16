/**
 * Crypto Card (card_admin) — WL-branded crypto card management.
 * Backend: GET /api/v1/admin/users (cardholders), /transactions (card txs).
 */

import React, { useEffect, useMemo, useState } from 'react';
import { whiteLabelAdminApi } from '../services/api';
import { useTheme } from '../context/ThemeContext';

export default function CryptoCard() {
  const { isDark } = useTheme();
  const [users, setUsers] = useState<any[]>([]);
  const [txs, setTxs] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [tab, setTab] = useState<'cardholders' | 'transactions'>('cardholders');

  useEffect(() => { load(); }, []);

  const load = async () => {
    setLoading(true); setError('');
    try {
      const [u, t] = await Promise.all([whiteLabelAdminApi.getUsers(), whiteLabelAdminApi.getTransactions()]);
      setUsers(u.users || []);
      setTxs((t.transactions || []).filter((x: any) => String(x.type).toLowerCase().includes('card')));
    } catch (e: any) { setError(e.message || 'Failed to load card data'); }
    finally { setLoading(false); }
  };

  const cardVolume = useMemo(() => {
    let v = 0;
    txs.forEach((t) => { v += parseFloat(t.amount) || 0; });
    return v;
  }, [txs]);

  const cardBg = isDark ? 'bg-gray-800' : 'bg-white';
  const cardText = isDark ? 'text-white' : 'text-gray-900';
  const border = isDark ? 'border-gray-700' : 'border-gray-200';
  const muted = isDark ? 'text-gray-400' : 'text-gray-500';
  const thBg = isDark ? 'bg-gray-700' : 'bg-gray-50';

  const badge = (s: string) => {
    const b = 'px-2 py-0.5 text-xs rounded';
    if (s === 'approved' || s === 'active' || s === 'confirmed') return `${b} ${isDark ? 'bg-green-900 text-green-200' : 'bg-green-100 text-green-800'}`;
    if (s === 'pending') return `${b} ${isDark ? 'bg-yellow-900 text-yellow-200' : 'bg-yellow-100 text-yellow-800'}`;
    return `${b} ${isDark ? 'bg-gray-700 text-gray-300' : 'bg-gray-200 text-gray-700'}`;
  };

  const stat = (l: string, v: string) => (
    <div className={`${cardBg} p-4 rounded-lg shadow border ${border}`}><p className={`text-xs ${muted}`}>{l}</p><p className={`text-xl font-bold mt-1 ${cardText}`}>{v}</p></div>
  );

  return (
    <div className="p-6">
      <h1 className={`text-2xl font-bold mb-2 ${cardText}`}>Crypto Card</h1>
      <p className={`mb-4 ${muted}`}>WL-branded card issuance, cardholder status and card transactions.</p>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-6">
        {stat('Cardholders', String(users.length))}
        {stat('Card Tx', String(txs.length))}
        {stat('Card Volume', cardVolume.toFixed(2))}
        {stat('Verified KYC', String(users.filter((u) => u.kyc_status === 'approved' || u.kyc_status === 'verified').length))}
      </div>

      <div className="flex gap-2 mb-4">
        <button onClick={() => setTab('cardholders')} className={`px-3 py-1 rounded text-sm ${tab === 'cardholders' ? 'bg-blue-600 text-white' : (isDark ? 'bg-gray-700 text-gray-200' : 'bg-gray-200 text-gray-700')}`}>Cardholders</button>
        <button onClick={() => setTab('transactions')} className={`px-3 py-1 rounded text-sm ${tab === 'transactions' ? 'bg-blue-600 text-white' : (isDark ? 'bg-gray-700 text-gray-200' : 'bg-gray-200 text-gray-700')}`}>Card Transactions</button>
      </div>

      {error && <div className={`mb-3 p-2 rounded text-sm ${isDark ? 'bg-red-900/50 text-red-200' : 'bg-red-50 text-red-700'}`}>{error}</div>}
      {loading && <div className={muted}>Loading…</div>}

      {!loading && tab === 'cardholders' && (
        <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border}`}>
          <table className="w-full">
            <thead className={thBg}><tr>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Name</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Email</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>KYC</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Card Status</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Country</th>
            </tr></thead>
            <tbody className={`divide-y ${border}`}>
              {users.length === 0 && <tr><td colSpan={5} className={`px-6 py-8 text-center ${muted}`}>No cardholders.</td></tr>}
              {users.map((u) => (
                <tr key={u.id}>
                  <td className={`px-6 py-4 ${cardText}`}>{u.username}</td>
                  <td className={`px-6 py-4 ${muted}`}>{u.email}</td>
                  <td className="px-6 py-4"><span className={badge(u.kyc_status)}>{u.kyc_status || '—'}</span></td>
                  <td className="px-6 py-4"><span className={badge(u.status)}>{u.status}</span></td>
                  <td className={`px-6 py-4 ${muted}`}>{u.country || '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {!loading && tab === 'transactions' && (
        <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border}`}>
          <table className="w-full">
            <thead className={thBg}><tr>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>User</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Amount</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Asset</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Status</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Time</th>
            </tr></thead>
            <tbody className={`divide-y ${border}`}>
              {txs.length === 0 && <tr><td colSpan={5} className={`px-6 py-8 text-center ${muted}`}>No card transactions.</td></tr>}
              {txs.map((t) => (
                <tr key={t.id}>
                  <td className={`px-6 py-4 font-mono text-xs ${muted}`}>{t.user_id ? String(t.user_id).substring(0, 8) + '…' : '—'}</td>
                  <td className={`px-6 py-4 ${cardText}`}>{t.amount}</td>
                  <td className={`px-6 py-4 ${cardText}`}>{t.currency}</td>
                  <td className="px-6 py-4"><span className={badge(t.status)}>{t.status}</span></td>
                  <td className={`px-6 py-4 text-xs ${muted}`}>{t.timestamp ? new Date(t.timestamp).toLocaleString() : '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
