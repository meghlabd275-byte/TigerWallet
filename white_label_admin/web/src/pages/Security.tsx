/**
 * Security (security_admin, WL-client-only) — incidents, banned users,
 * flagged transactions, session management.
 * Backend: GET /api/v1/admin/transactions (flagged), /sessions,
 * POST /api/v1/admin/users/:id/ban, /sessions/:id/revoke.
 */

import React, { useEffect, useState } from 'react';
import { whiteLabelAdminApi } from '../services/api';
import { useTheme } from '../context/ThemeContext';

export default function Security() {
  const { isDark } = useTheme();
  const [txs, setTxs] = useState<any[]>([]);
  const [sessions, setSessions] = useState<any[]>([]);
  const [users, setUsers] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [tab, setTab] = useState<'incidents' | 'sessions' | 'bans'>('incidents');

  useEffect(() => { load(); }, []);

  const load = async () => {
    setLoading(true); setError('');
    try {
      const [t, s, u] = await Promise.all([
        whiteLabelAdminApi.getTransactions(),
        whiteLabelAdminApi.getSessions(),
        whiteLabelAdminApi.getUsers(),
      ]);
      setTxs((t.transactions || []).filter((x: any) => x.status === 'flagged'));
      setSessions(s.sessions || []);
      setUsers(u.users || []);
    } catch (e: any) { setError(e.message || 'Failed to load security data'); }
    finally { setLoading(false); }
  };

  const ban = async (id: string) => { try { await whiteLabelAdminApi.banUser(id); load(); } catch (e: any) { setError(e.message || 'Failed to ban user'); } };
  const unban = async (id: string) => { try { await whiteLabelAdminApi.unbanUser(id); load(); } catch (e: any) { setError(e.message || 'Failed to unban user'); } };
  const revoke = async (id: string) => { try { await whiteLabelAdminApi.revokeSession(id); load(); } catch (e: any) { setError(e.message || 'Failed to revoke session'); } };

  const cardBg = isDark ? 'bg-gray-800' : 'bg-white';
  const cardText = isDark ? 'text-white' : 'text-gray-900';
  const border = isDark ? 'border-gray-700' : 'border-gray-200';
  const muted = isDark ? 'text-gray-400' : 'text-gray-500';
  const thBg = isDark ? 'bg-gray-700' : 'bg-gray-50';
  const short = (v: string) => (v ? `${v.substring(0, 8)}…` : '—');

  const stat = (l: string, v: string) => (
    <div className={`${cardBg} p-4 rounded-lg shadow border ${border}`}><p className={`text-xs ${muted}`}>{l}</p><p className={`text-xl font-bold mt-1 ${cardText}`}>{v}</p></div>
  );

  return (
    <div className="p-6">
      <h1 className={`text-2xl font-bold mb-2 ${cardText}`}>Security</h1>
      <p className={`mb-4 ${muted}`}>Incidents, banned users, flagged transactions and session management.</p>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-6">
        {stat('Flagged Tx', String(txs.length))}
        {stat('Active Sessions', String(sessions.length))}
        {stat('Banned Users', String(users.filter((u) => u.status === 'banned').length))}
        {stat('Total Users', String(users.length))}
      </div>

      <div className="flex gap-2 mb-4">
        {(['incidents', 'sessions', 'bans'] as const).map((t) => (
          <button key={t} onClick={() => setTab(t)} className={`px-3 py-1 rounded text-sm capitalize ${tab === t ? 'bg-blue-600 text-white' : (isDark ? 'bg-gray-700 text-gray-200' : 'bg-gray-200 text-gray-700')}`}>{t === 'incidents' ? 'Incidents' : t === 'sessions' ? 'Sessions' : 'Banned Users'}</button>
        ))}
      </div>

      {error && <div className={`mb-3 p-2 rounded text-sm ${isDark ? 'bg-red-900/50 text-red-200' : 'bg-red-50 text-red-700'}`}>{error}</div>}
      {loading && <div className={muted}>Loading…</div>}

      {!loading && tab === 'incidents' && (
        <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border}`}>
          <table className="w-full">
            <thead className={thBg}><tr>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Type</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Amount</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>User</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Time</th>
            </tr></thead>
            <tbody className={`divide-y ${border}`}>
              {txs.length === 0 && <tr><td colSpan={4} className={`px-6 py-8 text-center ${muted}`}>No flagged transactions.</td></tr>}
              {txs.map((t) => (
                <tr key={t.id}>
                  <td className={`px-6 py-4 ${cardText}`}>{t.type}</td>
                  <td className={`px-6 py-4 ${cardText}`}>{t.amount} {t.currency}</td>
                  <td className={`px-6 py-4 font-mono text-xs ${muted}`}>{short(t.user_id)}</td>
                  <td className={`px-6 py-4 text-xs ${muted}`}>{t.timestamp ? new Date(t.timestamp).toLocaleString() : '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {!loading && tab === 'sessions' && (
        <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border}`}>
          <table className="w-full">
            <thead className={thBg}><tr>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>IP</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>User Agent</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Expires</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Action</th>
            </tr></thead>
            <tbody className={`divide-y ${border}`}>
              {sessions.length === 0 && <tr><td colSpan={4} className={`px-6 py-8 text-center ${muted}`}>No active sessions.</td></tr>}
              {sessions.map((s) => (
                <tr key={s.id}>
                  <td className={`px-6 py-4 ${cardText}`}>{s.ip_address || '—'}</td>
                  <td className={`px-6 py-4 text-xs ${muted}`}>{s.user_agent || '—'}</td>
                  <td className={`px-6 py-4 text-xs ${muted}`}>{s.expires_at ? new Date(s.expires_at).toLocaleString() : '—'}</td>
                  <td className="px-6 py-4"><button onClick={() => revoke(s.id)} className="text-red-600 hover:underline text-sm">Revoke</button></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {!loading && tab === 'bans' && (
        <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border}`}>
          <table className="w-full">
            <thead className={thBg}><tr>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>User</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Email</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Status</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Action</th>
            </tr></thead>
            <tbody className={`divide-y ${border}`}>
              {users.length === 0 && <tr><td colSpan={4} className={`px-6 py-8 text-center ${muted}`}>No users.</td></tr>}
              {users.map((u) => (
                <tr key={u.id}>
                  <td className={`px-6 py-4 ${cardText}`}>{u.username}</td>
                  <td className={`px-6 py-4 ${muted}`}>{u.email}</td>
                  <td className="px-6 py-4"><span className={`px-2 py-0.5 text-xs rounded ${u.status === 'banned' ? (isDark ? 'bg-red-900 text-red-200' : 'bg-red-100 text-red-800') : (isDark ? 'bg-green-900 text-green-200' : 'bg-green-100 text-green-800')}`}>{u.status}</span></td>
                  <td className="px-6 py-4">
                    {u.status === 'banned'
                      ? <button onClick={() => unban(u.id)} className="text-green-600 hover:underline text-sm">Unban</button>
                      : <button onClick={() => ban(u.id)} className="text-red-600 hover:underline text-sm">Ban</button>}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
