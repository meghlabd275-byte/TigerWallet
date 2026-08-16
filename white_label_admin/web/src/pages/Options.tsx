/**
 * Options — options contract governance/config (no fund movement).
 * Backend: GET/POST /api/v1/admin/options, GET/PUT/DELETE /:id, POST /:id/status.
 */

import React, { useEffect, useState } from 'react';
import { whiteLabelAdminApi } from '../services/api';
import { useTheme } from '../context/ThemeContext';

const STATUSES = ['active', 'paused', 'disabled'];

export default function Options() {
  const { isDark } = useTheme();
  const [rows, setRows] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [symbol, setSymbol] = useState('');
  const [optionType, setOptionType] = useState('call');
  const [strike, setStrike] = useState('');

  useEffect(() => { load(); }, []);

  const load = async () => {
    setLoading(true); setError('');
    try { const d = await whiteLabelAdminApi.getOptionsContracts(); setRows(d.contracts || d.options || d || []); }
    catch (e: any) { setError(e.message || 'Failed to load options'); }
    finally { setLoading(false); }
  };

  const create = async () => {
    if (!symbol) { setError('Symbol required'); return; }
    try {
      await whiteLabelAdminApi.createOptionsContract({ symbol, option_type: optionType, strike });
      setSymbol(''); setStrike(''); load();
    } catch (e: any) { setError(e.message || 'Failed to create'); }
  };

  const setStatus = async (id: string, status: string) => {
    try { await whiteLabelAdminApi.updateOptionsContractStatus(id, status); load(); }
    catch (e: any) { setError(e.message || 'Failed to update status'); }
  };

  const remove = async (id: string) => {
    if (!confirm('Delete this options config?')) return;
    try { await whiteLabelAdminApi.deleteOptionsContract(id); load(); }
    catch (e: any) { setError(e.message || 'Failed to delete'); }
  };

  const cardBg = isDark ? 'bg-gray-800' : 'bg-white';
  const cardText = isDark ? 'text-white' : 'text-gray-900';
  const border = isDark ? 'border-gray-700' : 'border-gray-200';
  const muted = isDark ? 'text-gray-400' : 'text-gray-500';
  const thBg = isDark ? 'bg-gray-700' : 'bg-gray-50';
  const inputCls = isDark ? 'bg-gray-700 text-white border-gray-600' : 'bg-white text-gray-900 border-gray-300';

  return (
    <div className="p-6">
      <h1 className={`text-2xl font-bold mb-2 ${cardText}`}>Options</h1>
      <p className={`mb-4 ${muted}`}>Options contract governance &amp; config (no fund movement).</p>

      <div className={`${cardBg} p-4 rounded-lg shadow border ${border} mb-6`}>
        <h2 className={`text-sm font-semibold mb-3 ${cardText}`}>New Contract</h2>
        <div className="flex flex-wrap gap-2">
          <input value={symbol} onChange={(e) => setSymbol(e.target.value)} placeholder="Symbol" className={`px-3 py-2 rounded border ${inputCls}`} />
          <select value={optionType} onChange={(e) => setOptionType(e.target.value)} className={`px-3 py-2 rounded border ${inputCls}`}>
            <option value="call">Call</option><option value="put">Put</option>
          </select>
          <input value={strike} onChange={(e) => setStrike(e.target.value)} placeholder="Strike" className={`px-3 py-2 rounded border ${inputCls} w-32`} />
          <button onClick={create} className="px-4 py-2 rounded bg-blue-600 text-white">Create</button>
        </div>
      </div>

      {error && <div className={`mb-3 p-2 rounded text-sm ${isDark ? 'bg-red-900/50 text-red-200' : 'bg-red-50 text-red-700'}`}>{error}</div>}
      {loading && <div className={muted}>Loading…</div>}

      {!loading && (
        <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border}`}>
          <table className="w-full">
            <thead className={thBg}>
              <tr>
                {['Symbol', 'Type', 'Strike', 'Status', 'Actions'].map((h) => (
                  <th key={h} className={`px-4 py-2 text-left text-xs font-semibold ${muted}`}>{h}</th>
                ))}
              </tr>
            </thead>
            <tbody className={`divide-y ${border}`}>
              {rows.length === 0 && <tr><td colSpan={5} className={`px-4 py-8 text-center ${muted}`}>No options configs.</td></tr>}
              {rows.map((r) => (
                <tr key={r.id}>
                  <td className={`px-4 py-3 ${cardText}`}>{r.symbol}</td>
                  <td className={`px-4 py-3 ${muted}`}>{r.option_type}</td>
                  <td className={`px-4 py-3 ${muted}`}>{r.strike}</td>
                  <td className="px-4 py-3">
                    <select value={r.status || 'active'} onChange={(e) => setStatus(r.id, e.target.value)} className={`px-2 py-1 rounded border text-xs ${inputCls}`}>
                      {STATUSES.map((s) => <option key={s} value={s}>{s}</option>)}
                    </select>
                  </td>
                  <td className="px-4 py-3"><button onClick={() => remove(r.id)} className="text-red-500 text-xs">Delete</button></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
