/**
 * Liquidity (liquidity_admin) — liquidity sources, pools, depth, allocation.
 * Backend: GET /api/v1/admin/pairs (liquidity pairs), GET /api/v1/admin/fees.
 */

import React, { useEffect, useMemo, useState } from 'react';
import { whiteLabelAdminApi } from '../services/api';
import { useTheme } from '../context/ThemeContext';

export default function Liquidity() {
  const { isDark } = useTheme();
  const [pairs, setPairs] = useState<any[]>([]);
  const [fees, setFees] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => { load(); }, []);

  const load = async () => {
    setLoading(true); setError('');
    try {
      const [p, f] = await Promise.all([whiteLabelAdminApi.getPairs(), whiteLabelAdminApi.getFees()]);
      setPairs(p.pairs || []);
      setFees(f.fees || []);
    } catch (e: any) { setError(e.message || 'Failed to load liquidity data'); }
    finally { setLoading(false); }
  };

  const totals = useMemo(() => {
    let depth = 0; let vol = 0;
    pairs.forEach((p) => {
      depth += parseFloat(p.liquidity) || 0;
      vol += parseFloat(p.volume_24h) || 0;
    });
    return { depth, vol, activePairs: pairs.filter((p) => p.status === 'active').length };
  }, [pairs]);

  const cardBg = isDark ? 'bg-gray-800' : 'bg-white';
  const cardText = isDark ? 'text-white' : 'text-gray-900';
  const border = isDark ? 'border-gray-700' : 'border-gray-200';
  const muted = isDark ? 'text-gray-400' : 'text-gray-500';
  const thBg = isDark ? 'bg-gray-700' : 'bg-gray-50';

  const stat = (l: string, v: string) => (
    <div className={`${cardBg} p-4 rounded-lg shadow border ${border}`}><p className={`text-xs ${muted}`}>{l}</p><p className={`text-xl font-bold mt-1 ${cardText}`}>{v}</p></div>
  );

  return (
    <div className="p-6">
      <h1 className={`text-2xl font-bold mb-2 ${cardText}`}>Liquidity</h1>
      <p className={`mb-4 ${muted}`}>Sources, pool depth and allocation across trading pairs.</p>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-6">
        {stat('Total Depth', totals.depth.toFixed(2))}
        {stat('24h Volume', totals.vol.toFixed(2))}
        {stat('Active Pairs', String(totals.activePairs))}
        {stat('Total Pairs', String(pairs.length))}
      </div>

      {error && <div className={`mb-3 p-2 rounded text-sm ${isDark ? 'bg-red-900/50 text-red-200' : 'bg-red-50 text-red-700'}`}>{error}</div>}
      {loading && <div className={muted}>Loading…</div>}

      {!loading && (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border} lg:col-span-2`}>
            <h2 className={`px-6 py-3 text-sm font-semibold ${cardText} ${thBg}`}>Liquidity Pools</h2>
            <table className="w-full">
              <thead className={thBg}><tr>
                <th className={`px-6 py-2 text-left text-xs font-medium ${muted} uppercase`}>Pair</th>
                <th className={`px-6 py-2 text-left text-xs font-medium ${muted} uppercase`}>Price</th>
                <th className={`px-6 py-2 text-left text-xs font-medium ${muted} uppercase`}>Depth</th>
                <th className={`px-6 py-2 text-left text-xs font-medium ${muted} uppercase`}>24h Vol</th>
                <th className={`px-6 py-2 text-left text-xs font-medium ${muted} uppercase`}>Status</th>
              </tr></thead>
              <tbody className={`divide-y ${border}`}>
                {pairs.length === 0 && <tr><td colSpan={5} className={`px-6 py-8 text-center ${muted}`}>No liquidity pools.</td></tr>}
                {pairs.map((p) => (
                  <tr key={p.id}>
                    <td className={`px-6 py-3 ${cardText}`}>{p.pair_name}</td>
                    <td className={`px-6 py-3 ${cardText}`}>{p.price}</td>
                    <td className={`px-6 py-3 ${cardText}`}>{p.liquidity}</td>
                    <td className={`px-6 py-3 ${muted}`}>{p.volume_24h}</td>
                    <td className="px-6 py-3"><span className={`px-2 py-0.5 text-xs rounded ${p.status === 'active' ? (isDark ? 'bg-green-900 text-green-200' : 'bg-green-100 text-green-800') : (isDark ? 'bg-gray-700 text-gray-300' : 'bg-gray-200 text-gray-600')}`}>{p.status}</span></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border}`}>
            <h2 className={`px-6 py-3 text-sm font-semibold ${cardText} ${thBg}`}>Fee Allocations</h2>
            <table className="w-full">
              <tbody className={`divide-y ${border}`}>
                {fees.length === 0 && <tr><td className={`px-6 py-8 text-center ${muted}`}>No fee configs.</td></tr>}
                {fees.map((f) => (
                  <tr key={f.id}>
                    <td className={`px-6 py-3 ${cardText} text-sm capitalize`}>{f.fee_type}</td>
                    <td className={`px-6 py-3 ${muted} text-sm`}>{f.fee_percent}%</td>
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
