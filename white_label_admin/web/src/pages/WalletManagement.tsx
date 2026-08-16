/**
 * Wallet Management (wallet_admin) — master wallet + user wallets,
 * withdrawal approvals, send/claim/swap/trade oversight.
 * Backend: GET /api/v1/admin/users (wallet users), /withdrawals, /transactions.
 */

import React, { useEffect, useState } from 'react';
import { whiteLabelAdminApi } from '../services/api';
import { useTheme } from '../context/ThemeContext';

export default function WalletManagement() {
  const { isDark } = useTheme();
  const [users, setUsers] = useState<any[]>([]);
  const [withdrawals, setWithdrawals] = useState<any[]>([]);
  const [txs, setTxs] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [tab, setTab] = useState<'wallets' | 'withdrawals' | 'activity'>('wallets');

  useEffect(() => { load(); }, []);

  const load = async () => {
    setLoading(true); setError('');
    try {
      const [u, w, t] = await Promise.all([
        whiteLabelAdminApi.getUsers(),
        whiteLabelAdminApi.getWithdrawals(),
        whiteLabelAdminApi.getTransactions(),
      ]);
      setUsers(u.users || []);
      setWithdrawals(w.withdrawals || []);
      setTxs(t.transactions || []);
    } catch (e: any) { setError(e.message || 'Failed to load wallet data'); }
    finally { setLoading(false); }
  };

  const approve = async (id: string) => { try { await whiteLabelAdminApi.approveWithdrawal(id); load(); } catch (e: any) { setError(e.message || 'Failed to approve'); } };
  const reject = async (id: string) => { try { await whiteLabelAdminApi.rejectWithdrawal(id, 'Rejected by wallet admin'); load(); } catch (e: any) { setError(e.message || 'Failed to reject'); } };

  const cardBg = isDark ? 'bg-gray-800' : 'bg-white';
  const cardText = isDark ? 'text-white' : 'text-gray-900';
  const border = isDark ? 'border-gray-700' : 'border-gray-200';
  const muted = isDark ? 'text-gray-400' : 'text-gray-500';
  const thBg = isDark ? 'bg-gray-700' : 'bg-gray-50';
  const short = (v: string) => (v ? `${v.substring(0, 10)}…` : '—');

  const badge = (s: string) => {
    const b = 'px-2 py-0.5 text-xs rounded';
    if (s === 'approved' || s === 'processed' || s === 'confirmed') return `${b} ${isDark ? 'bg-green-900 text-green-200' : 'bg-green-100 text-green-800'}`;
    if (s === 'pending') return `${b} ${isDark ? 'bg-yellow-900 text-yellow-200' : 'bg-yellow-100 text-yellow-800'}`;
    return `${b} ${isDark ? 'bg-red-900 text-red-200' : 'bg-red-100 text-red-800'}`;
  };

  const pendingWithdrawals = withdrawals.filter((w) => w.status === 'pending');

  return (
    <div className="p-6">
      <h1 className={`text-2xl font-bold mb-2 ${cardText}`}>Wallet Management</h1>
      <p className={`mb-4 ${muted}`}>Master & user wallets, withdrawal approvals, send/claim/swap/trade oversight.</p>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-6">
        <div className={`${cardBg} p-4 rounded-lg shadow border ${border}`}><p className={`text-xs ${muted}`}>Wallet Users</p><p className={`text-xl font-bold mt-1 ${cardText}`}>{users.length}</p></div>
        <div className={`${cardBg} p-4 rounded-lg shadow border ${border}`}><p className={`text-xs ${muted}`}>Pending Withdrawals</p><p className={`text-xl font-bold mt-1 ${cardText}`}>{pendingWithdrawals.length}</p></div>
        <div className={`${cardBg} p-4 rounded-lg shadow border ${border}`}><p className={`text-xs ${muted}`}>Total Withdrawals</p><p className={`text-xl font-bold mt-1 ${cardText}`}>{withdrawals.length}</p></div>
        <div className={`${cardBg} p-4 rounded-lg shadow border ${border}`}><p className={`text-xs ${muted}`}>Transactions</p><p className={`text-xl font-bold mt-1 ${cardText}`}>{txs.length}</p></div>
      </div>

      <div className="flex gap-2 mb-4">
        {(['wallets', 'withdrawals', 'activity'] as const).map((t) => (
          <button key={t} onClick={() => setTab(t)} className={`px-3 py-1 rounded text-sm capitalize ${tab === t ? 'bg-blue-600 text-white' : (isDark ? 'bg-gray-700 text-gray-200' : 'bg-gray-200 text-gray-700')}`}>{t}</button>
        ))}
      </div>

      {error && <div className={`mb-3 p-2 rounded text-sm ${isDark ? 'bg-red-900/50 text-red-200' : 'bg-red-50 text-red-700'}`}>{error}</div>}
      {loading && <div className={muted}>Loading…</div>}

      {!loading && tab === 'wallets' && (
        <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border}`}>
          <table className="w-full">
            <thead className={thBg}><tr>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>User</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Email</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Wallet</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Status</th>
            </tr></thead>
            <tbody className={`divide-y ${border}`}>
              {users.length === 0 && <tr><td colSpan={4} className={`px-6 py-8 text-center ${muted}`}>No wallets.</td></tr>}
              {users.map((u) => (
                <tr key={u.id}>
                  <td className={`px-6 py-4 ${cardText}`}>{u.username}</td>
                  <td className={`px-6 py-4 ${muted}`}>{u.email}</td>
                  <td className={`px-6 py-4 font-mono text-xs ${muted}`}>{short(u.wallet_address)}</td>
                  <td className="px-6 py-4"><span className={badge(u.status)}>{u.status}</span></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {!loading && tab === 'withdrawals' && (
        <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border}`}>
          <table className="w-full">
            <thead className={thBg}><tr>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>User</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Amount</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Address</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Status</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Actions</th>
            </tr></thead>
            <tbody className={`divide-y ${border}`}>
              {withdrawals.length === 0 && <tr><td colSpan={5} className={`px-6 py-8 text-center ${muted}`}>No withdrawals.</td></tr>}
              {withdrawals.map((w) => (
                <tr key={w.id}>
                  <td className={`px-6 py-4 font-mono text-xs ${muted}`}>{short(w.user_id)}</td>
                  <td className={`px-6 py-4 ${cardText}`}>{w.amount} {w.currency}</td>
                  <td className={`px-6 py-4 font-mono text-xs ${muted}`}>{short(w.address)}</td>
                  <td className="px-6 py-4"><span className={badge(w.status)}>{w.status}</span></td>
                  <td className="px-6 py-4 space-x-2">
                    {w.status === 'pending' && (<>
                      <button onClick={() => approve(w.id)} className="text-green-600 hover:underline text-sm">Approve</button>
                      <button onClick={() => reject(w.id)} className="text-red-600 hover:underline text-sm">Reject</button>
                    </>)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {!loading && tab === 'activity' && (
        <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border}`}>
          <table className="w-full">
            <thead className={thBg}><tr>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Type</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Amount</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Status</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Time</th>
            </tr></thead>
            <tbody className={`divide-y ${border}`}>
              {txs.length === 0 && <tr><td colSpan={4} className={`px-6 py-8 text-center ${muted}`}>No activity.</td></tr>}
              {txs.slice(0, 100).map((t) => (
                <tr key={t.id}>
                  <td className={`px-6 py-4 ${cardText}`}>{t.type}</td>
                  <td className={`px-6 py-4 ${cardText}`}>{t.amount} {t.currency}</td>
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
