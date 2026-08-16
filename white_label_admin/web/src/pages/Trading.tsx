/**
 * Trading (trading_admin) — futures/margin/options/copy/convert positions.
 * Backend: GET /api/v1/admin/transactions (filtered client-side by type),
 * POST /api/v1/admin/transactions/:id/flag (risk action proxy).
 */

import React, { useEffect, useMemo, useState } from 'react';
import { whiteLabelAdminApi } from '../services/api';
import { useTheme } from '../context/ThemeContext';

const TRADING_TYPES = ['futures', 'margin', 'options', 'copy', 'convert'];

export default function Trading() {
  const { isDark } = useTheme();
  const [txs, setTxs] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [filter, setFilter] = useState<string>('all');
  const [actionMsg, setActionMsg] = useState('');

  useEffect(() => { load(); }, []);

  const load = async () => {
    setLoading(true); setError('');
    try {
      const data = await whiteLabelAdminApi.getTransactions();
      const all = data.transactions || [];
      // Trading positions are tx types in the trading family.
      const positions = all.filter((t) => TRADING_TYPES.includes(String(t.type).toLowerCase()));
      setTxs(positions);
    } catch (e: any) { setError(e.message || 'Failed to load trading positions'); }
    finally { setLoading(false); }
  };

  const filtered = useMemo(
    () => (filter === 'all' ? txs : txs.filter((t) => String(t.type).toLowerCase() === filter)),
    [txs, filter],
  );

  const handleFlag = async (id: string) => {
    setActionMsg('');
    try { await whiteLabelAdminApi.flagTransaction(id); setActionMsg(`Position ${id.substring(0, 8)}… flagged for risk review.`); load(); }
    catch (e: any) { setError(e.message || 'Failed to flag position'); }
  };

  const cardBg = isDark ? 'bg-gray-800' : 'bg-white';
  const cardText = isDark ? 'text-white' : 'text-gray-900';
  const border = isDark ? 'border-gray-700' : 'border-gray-200';
  const muted = isDark ? 'text-gray-400' : 'text-gray-500';
  const thBg = isDark ? 'bg-gray-700' : 'bg-gray-50';

  const badge = (s: string) => {
    const b = 'px-2 py-0.5 text-xs rounded';
    if (s === 'open' || s === 'confirmed') return `${b} ${isDark ? 'bg-green-900 text-green-200' : 'bg-green-100 text-green-800'}`;
    if (s === 'flagged') return `${b} ${isDark ? 'bg-red-900 text-red-200' : 'bg-red-100 text-red-800'}`;
    if (s === 'pending') return `${b} ${isDark ? 'bg-yellow-900 text-yellow-200' : 'bg-yellow-100 text-yellow-800'}`;
    return `${b} ${isDark ? 'bg-gray-700 text-gray-300' : 'bg-gray-200 text-gray-700'}`;
  };

  const tabs = ['all', ...TRADING_TYPES];

  return (
    <div className="p-6">
      <h1 className={`text-2xl font-bold mb-2 ${cardText}`}>Trading</h1>
      <p className={`mb-4 ${muted}`}>Futures · Margin · Options · Copy · Convert positions and risk actions.</p>

      <div className="flex flex-wrap gap-2 mb-4">
        {tabs.map((t) => (
          <button key={t} onClick={() => setFilter(t)}
            className={`px-3 py-1 rounded text-sm ${filter === t ? 'bg-blue-600 text-white' : (isDark ? 'bg-gray-700 text-gray-200' : 'bg-gray-200 text-gray-700')}`}>
            {t === 'all' ? 'All' : t.charAt(0).toUpperCase() + t.slice(1)}
          </button>
        ))}
      </div>

      {actionMsg && <div className={`mb-3 p-2 rounded text-sm ${isDark ? 'bg-blue-900/50 text-blue-200' : 'bg-blue-50 text-blue-700'}`}>{actionMsg}</div>}
      {error && <div className={`mb-3 p-2 rounded text-sm ${isDark ? 'bg-red-900/50 text-red-200' : 'bg-red-50 text-red-700'}`}>{error}</div>}
      {loading && <div className={muted}>Loading positions…</div>}

      {!loading && (
        <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border}`}>
          <table className="w-full">
            <thead className={thBg}>
              <tr>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Type</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>User</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Size</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Asset</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Fee</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Status</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Time</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Risk</th>
              </tr>
            </thead>
            <tbody className={`divide-y ${border}`}>
              {filtered.length === 0 && (
                <tr><td colSpan={8} className={`px-6 py-8 text-center ${muted}`}>No open positions.</td></tr>
              )}
              {filtered.map((p) => (
                <tr key={p.id}>
                  <td className={`px-6 py-4 ${cardText} capitalize`}>{p.type}</td>
                  <td className={`px-6 py-4 font-mono text-xs ${muted}`}>{p.user_id ? String(p.user_id).substring(0, 8) + '…' : '—'}</td>
                  <td className={`px-6 py-4 ${cardText}`}>{p.amount}</td>
                  <td className={`px-6 py-4 ${cardText}`}>{p.currency}</td>
                  <td className={`px-6 py-4 ${muted}`}>{p.fee}</td>
                  <td className="px-6 py-4"><span className={badge(p.status)}>{p.status}</span></td>
                  <td className={`px-6 py-4 text-xs ${muted}`}>{p.timestamp ? new Date(p.timestamp).toLocaleString() : '—'}</td>
                  <td className="px-6 py-4">
                    {p.status !== 'flagged'
                      ? <button onClick={() => handleFlag(p.id)} className="text-red-600 hover:underline text-sm">Flag</button>
                      : <span className={muted}>flagged</span>}
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
