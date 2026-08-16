/**
 * Rewards (reward_admin) — reward campaigns, user points, redemption.
 * Backend: GET /api/v1/admin/users (points/leaderboard), GET /api/v1/admin/fees (reward configs),
 * GET /api/v1/admin/transactions (redemption history).
 */

import React, { useEffect, useMemo, useState } from 'react';
import { whiteLabelAdminApi } from '../services/api';
import { useTheme } from '../context/ThemeContext';

export default function Rewards() {
  const { isDark } = useTheme();
  const [users, setUsers] = useState<any[]>([]);
  const [fees, setFees] = useState<any[]>([]);
  const [txs, setTxs] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => { load(); }, []);

  const load = async () => {
    setLoading(true); setError('');
    try {
      const [u, f, t] = await Promise.all([whiteLabelAdminApi.getUsers(), whiteLabelAdminApi.getFees(), whiteLabelAdminApi.getTransactions()]);
      setUsers(u.users || []);
      setFees((f.fees || []).filter((x: any) => String(x.fee_type).toLowerCase().includes('reward')));
      setTxs((t.transactions || []).filter((x: any) => String(x.type).toLowerCase().includes('reward') || String(x.type).toLowerCase() === 'redemption'));
    } catch (e: any) { setError(e.message || 'Failed to load rewards data'); }
    finally { setLoading(false); }
  };

  const leaderboard = useMemo(() => [...users].slice(0, 10), [users]);

  const cardBg = isDark ? 'bg-gray-800' : 'bg-white';
  const cardText = isDark ? 'text-white' : 'text-gray-900';
  const border = isDark ? 'border-gray-700' : 'border-gray-200';
  const muted = isDark ? 'text-gray-400' : 'text-gray-500';
  const thBg = isDark ? 'bg-gray-700' : 'bg-gray-50';

  const stat = (l: string, v: string) => (
    <div className={`${cardBg} p-4 rounded-lg shadow border ${border}`}><p className={`text-xs ${muted}`}>{l}</p><p className={`text-xl font-bold mt-1 ${cardText}`}>{v}</p></div>
  );

  const medal = (i: number) => i === 0 ? '🥇' : i === 1 ? '🥈' : i === 2 ? '🥉' : `#${i + 1}`;

  return (
    <div className="p-6">
      <h1 className={`text-2xl font-bold mb-2 ${cardText}`}>Rewards</h1>
      <p className={`mb-4 ${muted}`}>Campaigns, leaderboards and redemption history.</p>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-6">
        {stat('Reward Campaigns', String(fees.length))}
        {stat('Participants', String(users.length))}
        {stat('Redemptions', String(txs.length))}
        {stat('Leaderboard Size', String(leaderboard.length))}
      </div>

      {error && <div className={`mb-3 p-2 rounded text-sm ${isDark ? 'bg-red-900/50 text-red-200' : 'bg-red-50 text-red-700'}`}>{error}</div>}
      {loading && <div className={muted}>Loading…</div>}

      {!loading && (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className={`${cardBg} rounded-lg shadow border ${border} p-4`}>
            <h2 className={`text-sm font-semibold mb-3 ${cardText}`}>Leaderboard</h2>
            <div className={`divide-y ${border}`}>
              {leaderboard.length === 0 && <p className={muted}>No participants.</p>}
              {leaderboard.map((u, i) => (
                <div key={u.id} className="flex justify-between py-2">
                  <span className={cardText}>{medal(i)} {u.username}</span>
                  <span className={muted}>{u.email}</span>
                </div>
              ))}
            </div>
          </div>

          <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border}`}>
            <h2 className={`px-6 py-3 text-sm font-semibold ${cardText} ${thBg}`}>Reward Campaigns</h2>
            <table className="w-full">
              <tbody className={`divide-y ${border}`}>
                {fees.length === 0 && <tr><td className={`px-6 py-8 text-center ${muted}`}>No reward configs.</td></tr>}
                {fees.map((f) => (
                  <tr key={f.id}><td className={`px-6 py-3 ${cardText} capitalize`}>{f.fee_type}</td><td className={`px-6 py-3 ${muted}`}>{f.fee_percent}%</td></tr>
                ))}
              </tbody>
            </table>
          </div>

          <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border}`}>
            <h2 className={`px-6 py-3 text-sm font-semibold ${cardText} ${thBg}`}>Redemption History</h2>
            <table className="w-full">
              <tbody className={`divide-y ${border}`}>
                {txs.length === 0 && <tr><td className={`px-6 py-8 text-center ${muted}`}>No redemptions.</td></tr>}
                {txs.map((t) => (
                  <tr key={t.id}><td className={`px-6 py-3 ${cardText}`}>{t.amount} {t.currency}</td><td className={`px-6 py-3 text-xs ${muted}`}>{t.timestamp ? new Date(t.timestamp).toLocaleDateString() : '—'}</td></tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
