/**
 * Bots Management (bot_admin) — bot configs, running bots, performance.
 * Backend: GET /api/v1/admin/users (bot users / operators),
 * GET /api/v1/admin/audit-logs (bot.* events).
 */

import React, { useEffect, useMemo, useState } from 'react';
import { whiteLabelAdminApi } from '../services/api';
import { useTheme } from '../context/ThemeContext';

export default function BotsManagement() {
  const { isDark } = useTheme();
  const [users, setUsers] = useState<any[]>([]);
  const [logs, setLogs] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => { load(); }, []);

  const load = async () => {
    setLoading(true); setError('');
    try {
      const [u, l] = await Promise.all([whiteLabelAdminApi.getUsers(), whiteLabelAdminApi.getAuditLogs()]);
      setUsers(u.users || []);
      setLogs((l.audit_logs || []).filter((x: any) => String(x.action).toLowerCase().startsWith('bot')));
    } catch (e: any) { setError(e.message || 'Failed to load bots data'); }
    finally { setLoading(false); }
  };

  const botEvents = useMemo(() => {
    const counts: Record<string, number> = {};
    logs.forEach((l) => { counts[l.action] = (counts[l.action] || 0) + 1; });
    return counts;
  }, [logs]);

  const cardBg = isDark ? 'bg-gray-800' : 'bg-white';
  const cardText = isDark ? 'text-white' : 'text-gray-900';
  const border = isDark ? 'border-gray-700' : 'border-gray-200';
  const muted = isDark ? 'text-gray-400' : 'text-gray-500';
  const thBg = isDark ? 'bg-gray-700' : 'bg-gray-50';

  const badge = (s: string) => {
    const b = 'px-2 py-0.5 text-xs rounded';
    if (s === 'active') return `${b} ${isDark ? 'bg-green-900 text-green-200' : 'bg-green-100 text-green-800'}`;
    return `${b} ${isDark ? 'bg-gray-700 text-gray-300' : 'bg-gray-200 text-gray-700'}`;
  };

  return (
    <div className="p-6">
      <h1 className={`text-2xl font-bold mb-2 ${cardText}`}>Bots Management</h1>
      <p className={`mb-4 ${muted}`}>Bot operators, running bots and strategy events.</p>

      <div className="grid grid-cols-2 md:grid-cols-3 gap-3 mb-6">
        <div className={`${cardBg} p-4 rounded-lg shadow border ${border}`}><p className={`text-xs ${muted}`}>Operators</p><p className={`text-xl font-bold mt-1 ${cardText}`}>{users.length}</p></div>
        <div className={`${cardBg} p-4 rounded-lg shadow border ${border}`}><p className={`text-xs ${muted}`}>Bot Events</p><p className={`text-xl font-bold mt-1 ${cardText}`}>{logs.length}</p></div>
        <div className={`${cardBg} p-4 rounded-lg shadow border ${border}`}><p className={`text-xs ${muted}`}>Event Types</p><p className={`text-xl font-bold mt-1 ${cardText}`}>{Object.keys(botEvents).length}</p></div>
      </div>

      {error && <div className={`mb-3 p-2 rounded text-sm ${isDark ? 'bg-red-900/50 text-red-200' : 'bg-red-50 text-red-700'}`}>{error}</div>}
      {loading && <div className={muted}>Loading…</div>}

      {!loading && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border}`}>
            <h2 className={`px-6 py-3 text-sm font-semibold ${cardText} ${thBg}`}>Bot Operators</h2>
            <table className="w-full">
              <tbody className={`divide-y ${border}`}>
                {users.length === 0 && <tr><td className={`px-6 py-8 text-center ${muted}`}>No operators.</td></tr>}
                {users.map((u) => (
                  <tr key={u.id}>
                    <td className={`px-6 py-3 ${cardText}`}>{u.username}</td>
                    <td className={`px-6 py-3 ${muted} text-sm`}>{u.email}</td>
                    <td className="px-6 py-3"><span className={badge(u.status)}>{u.status}</span></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border}`}>
            <h2 className={`px-6 py-3 text-sm font-semibold ${cardText} ${thBg}`}>Bot Events</h2>
            <table className="w-full">
              <thead className={thBg}>
                <tr><th className={`px-6 py-2 text-left text-xs font-medium ${muted} uppercase`}>Action</th><th className={`px-6 py-2 text-left text-xs font-medium ${muted} uppercase`}>Time</th></tr>
              </thead>
              <tbody className={`divide-y ${border}`}>
                {logs.length === 0 && <tr><td colSpan={2} className={`px-6 py-8 text-center ${muted}`}>No bot events.</td></tr>}
                {logs.map((l) => (
                  <tr key={l.id}>
                    <td className={`px-6 py-3 ${cardText} text-sm`}>{l.action}</td>
                    <td className={`px-6 py-3 text-xs ${muted}`}>{l.created_at ? new Date(l.created_at).toLocaleString() : '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
