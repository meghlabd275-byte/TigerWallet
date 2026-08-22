/**
 * Bots Management (bot_admin) — WL bot operators, config and stats.
 * Backend: GET/POST /api/v1/admin/wl-bots/operators, PUT /:id/status,
 * GET /api/v1/admin/wl-bots/config, GET /stats.
 */

import React, { useEffect, useState } from 'react';
import { whiteLabelAdminApi } from '../services/api';
import { useTheme } from '../context/ThemeContext';

const STATUSES = ['active', 'suspended', 'disabled'];

export default function BotsManagement() {
  const { isDark } = useTheme();
  const [operators, setOperators] = useState<any[]>([]);
  const [config, setConfig] = useState<any>(null);
  const [stats, setStats] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [strategy, setStrategy] = useState('grid');

  useEffect(() => { load(); }, []);

  const load = async () => {
    setLoading(true); setError('');
    try {
      const [o, c, s] = await Promise.all([
        whiteLabelAdminApi.getWLBotOperators(),
        whiteLabelAdminApi.getWLBotConfig(),
        whiteLabelAdminApi.getWLBotStats(),
      ]);
      setOperators(o.operators || o || []);
      setConfig(c);
      setStats(s);
    } catch (e: any) { setError(e.message || 'Failed to load bots data'); }
    finally { setLoading(false); }
  };

  const register = async () => {
    if (!name || !email) { setError('Name and email required'); return; }
    try { await whiteLabelAdminApi.registerWLBotOperator({ name, email, strategy }); setName(''); setEmail(''); load(); }
    catch (e: any) { setError(e.message || 'Failed to register operator'); }
  };

  const setStatus = async (id: string, status: string) => {
    try { await whiteLabelAdminApi.updateWLBotOperatorStatus(id, status); load(); }
    catch (e: any) { setError(e.message || 'Failed to update operator status'); }
  };

  const cardBg = isDark ? 'bg-gray-800' : 'bg-white';
  const cardText = isDark ? 'text-white' : 'text-gray-900';
  const border = isDark ? 'border-gray-700' : 'border-gray-200';
  const muted = isDark ? 'text-gray-400' : 'text-gray-500';
  const thBg = isDark ? 'bg-gray-700' : 'bg-gray-50';
  const inputCls = isDark ? 'bg-gray-700 text-white border-gray-600' : 'bg-white text-gray-900 border-gray-300';

  const stat = (l: string, v: string) => (
    <div className={`${cardBg} p-4 rounded-lg shadow border ${border}`}><p className={`text-xs ${muted}`}>{l}</p><p className={`text-xl font-bold mt-1 ${cardText}`}>{v}</p></div>
  );

  return (
    <div className="p-6">
      <h1 className={`text-2xl font-bold mb-2 ${cardText}`}>Bots Management</h1>
      <p className={`mb-4 ${muted}`}>WL bot operators, config and performance stats.</p>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-6">
        {stat('Operators', String(operators.length))}
        {stat('Active Operators', String(operators.filter((o) => o.status === 'active').length))}
        {stat('Running Bots', stats ? String(stats.running_bots ?? stats.active_bots ?? '—') : '—')}
        {stat('Total Bots', stats ? String(stats.total_bots ?? '—') : '—')}
      </div>

      <div className={`${cardBg} p-4 rounded-lg shadow border ${border} mb-6`}>
        <h2 className={`text-sm font-semibold mb-3 ${cardText}`}>Register Operator</h2>
        <div className="flex flex-wrap gap-2">
          <input value={name} onChange={(e) => setName(e.target.value)} placeholder="Operator name" className={`px-3 py-2 rounded border ${inputCls}`} />
          <input value={email} onChange={(e) => setEmail(e.target.value)} placeholder="Email" className={`px-3 py-2 rounded border ${inputCls}`} />
          <select value={strategy} onChange={(e) => setStrategy(e.target.value)} className={`px-3 py-2 rounded border ${inputCls}`}>
            <option value="grid">Grid</option><option value="dca">DCA</option><option value="arbitrage">Arbitrage</option><option value="mm">Market Making</option>
          </select>
          <button onClick={register} className="px-4 py-2 rounded bg-blue-600 text-white">Register</button>
        </div>
      </div>

      {error && <div className={`mb-3 p-2 rounded text-sm ${isDark ? 'bg-red-900/50 text-red-200' : 'bg-red-50 text-red-700'}`}>{error}</div>}
      {loading && <div className={muted}>Loading…</div>}

      {!loading && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border}`}>
            <h2 className={`px-6 py-3 text-sm font-semibold ${cardText} ${thBg}`}>Bot Operators</h2>
            <table className="w-full">
              <thead className={thBg}><tr>
                {['Name', 'Email', 'Strategy', 'Status'].map((h) => (
                  <th key={h} className={`px-6 py-2 text-left text-xs font-medium ${muted} uppercase`}>{h}</th>
                ))}
              </tr></thead>
              <tbody className={`divide-y ${border}`}>
                {operators.length === 0 && <tr><td colSpan={4} className={`px-6 py-8 text-center ${muted}`}>No operators.</td></tr>}
                {operators.map((o) => (
                  <tr key={o.id}>
                    <td className={`px-6 py-3 ${cardText}`}>{o.name}</td>
                    <td className={`px-6 py-3 ${muted} text-sm`}>{o.email}</td>
                    <td className={`px-6 py-3 ${muted} text-sm`}>{o.strategy || '—'}</td>
                    <td className="px-6 py-3">
                      <select value={o.status || 'active'} onChange={(e) => setStatus(o.id, e.target.value)} className={`px-2 py-1 rounded border text-xs ${inputCls}`}>
                        {STATUSES.map((s) => <option key={s} value={s}>{s}</option>)}
                      </select>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border}`}>
            <h2 className={`px-6 py-3 text-sm font-semibold ${cardText} ${thBg}`}>Bot Config</h2>
            <table className="w-full">
              <tbody className={`divide-y ${border}`}>
                {!config ? (
                  <tr><td className={`px-6 py-8 text-center ${muted}`}>No bot config.</td></tr>
                ) : (
                  Object.entries(config).map(([k, v]) => (
                    <tr key={k}>
                      <td className={`px-6 py-3 ${cardText} text-sm capitalize`}>{k.replace(/_/g, ' ')}</td>
                      <td className={`px-6 py-3 ${muted} text-sm`}>{String(v)}</td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
