/**
 * Liquidity (liquidity_admin) — WL liquidity sources, allocations and stats.
 * Backend: GET/POST /api/v1/admin/wl-liquidity/sources, PUT/DELETE /:id,
 * GET/POST /api/v1/admin/wl-liquidity/allocations, GET /stats.
 */

import React, { useEffect, useState } from 'react';
import { whiteLabelAdminApi } from '../services/api';
import { useTheme } from '../context/ThemeContext';

const STATUSES = ['active', 'paused', 'disabled'];

export default function Liquidity() {
  const { isDark } = useTheme();
  const [sources, setSources] = useState<any[]>([]);
  const [allocations, setAllocations] = useState<any[]>([]);
  const [stats, setStats] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [name, setName] = useState('');
  const [sourceType, setSourceType] = useState('amm');
  const [allocSourceId, setAllocSourceId] = useState('');
  const [allocPercent, setAllocPercent] = useState('');

  useEffect(() => { load(); }, []);

  const load = async () => {
    setLoading(true); setError('');
    try {
      const [s, a, st] = await Promise.all([
        whiteLabelAdminApi.getWLLiquiditySources(),
        whiteLabelAdminApi.getWLLiquidityAllocations(),
        whiteLabelAdminApi.getWLLiquidityStats(),
      ]);
      setSources(s.sources || s || []);
      setAllocations(a.allocations || a || []);
      setStats(st);
    } catch (e: any) { setError(e.message || 'Failed to load liquidity data'); }
    finally { setLoading(false); }
  };

  const create = async () => {
    if (!name) { setError('Name required'); return; }
    try {
      await whiteLabelAdminApi.createWLLiquiditySource({ name, source_type: sourceType });
      setName(''); load();
    } catch (e: any) { setError(e.message || 'Failed to create source'); }
  };

  const setStatus = async (id: string, status: string) => {
    try { await whiteLabelAdminApi.updateWLLiquiditySource(id, { status }); load(); }
    catch (e: any) { setError(e.message || 'Failed to update source'); }
  };

  const remove = async (id: string) => {
    if (!confirm('Delete this liquidity source?')) return;
    try { await whiteLabelAdminApi.deleteWLLiquiditySource(id); load(); }
    catch (e: any) { setError(e.message || 'Failed to delete source'); }
  };

  const setAllocation = async () => {
    if (!allocSourceId || !allocPercent) { setError('Source and allocation % required'); return; }
    try {
      await whiteLabelAdminApi.setWLLiquidityAllocation({ source_id: allocSourceId, allocation_percent: Number(allocPercent) });
      setAllocSourceId(''); setAllocPercent(''); load();
    } catch (e: any) { setError(e.message || 'Failed to set allocation'); }
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
      <h1 className={`text-2xl font-bold mb-2 ${cardText}`}>Liquidity</h1>
      <p className={`mb-4 ${muted}`}>WL liquidity sources, allocations and pool depth stats.</p>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-6">
        {stat('Total Sources', String(sources.length))}
        {stat('Active Sources', String(sources.filter((s) => s.status === 'active').length))}
        {stat('Allocations', String(allocations.length))}
        {stat('Total Depth', stats ? String(stats.total_depth ?? stats.depth ?? '—') : '—')}
      </div>

      <div className={`${cardBg} p-4 rounded-lg shadow border ${border} mb-6`}>
        <h2 className={`text-sm font-semibold mb-3 ${cardText}`}>New Source</h2>
        <div className="flex flex-wrap gap-2">
          <input value={name} onChange={(e) => setName(e.target.value)} placeholder="Source name" className={`px-3 py-2 rounded border ${inputCls}`} />
          <select value={sourceType} onChange={(e) => setSourceType(e.target.value)} className={`px-3 py-2 rounded border ${inputCls}`}>
            <option value="amm">AMM</option><option value="cex">CEX</option><option value="dex">DEX</option><option value="otc">OTC</option>
          </select>
          <button onClick={create} className="px-4 py-2 rounded bg-blue-600 text-white">Create</button>
        </div>
      </div>

      <div className={`${cardBg} p-4 rounded-lg shadow border ${border} mb-6`}>
        <h2 className={`text-sm font-semibold mb-3 ${cardText}`}>Set Allocation</h2>
        <div className="flex flex-wrap gap-2">
          <select value={allocSourceId} onChange={(e) => setAllocSourceId(e.target.value)} className={`px-3 py-2 rounded border ${inputCls}`}>
            <option value="">Select source</option>
            {sources.map((s) => <option key={s.id} value={s.id}>{s.name}</option>)}
          </select>
          <input value={allocPercent} onChange={(e) => setAllocPercent(e.target.value)} type="number" min="0" max="100" placeholder="Allocation %" className={`px-3 py-2 rounded border ${inputCls} w-40`} />
          <button onClick={setAllocation} className="px-4 py-2 rounded bg-blue-600 text-white">Set</button>
        </div>
      </div>

      {error && <div className={`mb-3 p-2 rounded text-sm ${isDark ? 'bg-red-900/50 text-red-200' : 'bg-red-50 text-red-700'}`}>{error}</div>}
      {loading && <div className={muted}>Loading…</div>}

      {!loading && (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border} lg:col-span-2`}>
            <h2 className={`px-6 py-3 text-sm font-semibold ${cardText} ${thBg}`}>Liquidity Sources</h2>
            <table className="w-full">
              <thead className={thBg}><tr>
                {['Name', 'Type', 'Status', 'Actions'].map((h) => (
                  <th key={h} className={`px-6 py-2 text-left text-xs font-medium ${muted} uppercase`}>{h}</th>
                ))}
              </tr></thead>
              <tbody className={`divide-y ${border}`}>
                {sources.length === 0 && <tr><td colSpan={4} className={`px-6 py-8 text-center ${muted}`}>No liquidity sources.</td></tr>}
                {sources.map((s) => (
                  <tr key={s.id}>
                    <td className={`px-6 py-3 ${cardText}`}>{s.name}</td>
                    <td className={`px-6 py-3 ${muted}`}>{s.source_type}</td>
                    <td className="px-6 py-3">
                      <select value={s.status || 'active'} onChange={(e) => setStatus(s.id, e.target.value)} className={`px-2 py-1 rounded border text-xs ${inputCls}`}>
                        {STATUSES.map((st) => <option key={st} value={st}>{st}</option>)}
                      </select>
                    </td>
                    <td className="px-6 py-3"><button onClick={() => remove(s.id)} className="text-red-500 text-xs">Delete</button></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border}`}>
            <h2 className={`px-6 py-3 text-sm font-semibold ${cardText} ${thBg}`}>Allocations</h2>
            <table className="w-full">
              <tbody className={`divide-y ${border}`}>
                {allocations.length === 0 && <tr><td className={`px-6 py-8 text-center ${muted}`}>No allocations.</td></tr>}
                {allocations.map((a) => (
                  <tr key={a.id || a.source_id}>
                    <td className={`px-6 py-3 ${cardText} text-sm`}>{a.source_name || a.source_id || '—'}</td>
                    <td className={`px-6 py-3 ${muted} text-sm`}>{a.allocation_percent}%</td>
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
