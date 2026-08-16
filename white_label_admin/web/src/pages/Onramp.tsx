/**
 * Onramp — fiat-onramp order governance (approve/reject; no fund movement).
 * Backend: GET/POST /api/v1/admin/onramp, GET/PUT/DELETE /:id, POST /:id/approve, POST /:id/reject {reason}.
 */

import React, { useEffect, useState } from 'react';
import { whiteLabelAdminApi } from '../services/api';
import { useTheme } from '../context/ThemeContext';

export default function Onramp() {
  const { isDark } = useTheme();
  const [rows, setRows] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [rejectId, setRejectId] = useState<string | null>(null);
  const [reason, setReason] = useState('');

  useEffect(() => { load(); }, []);

  const load = async () => {
    setLoading(true); setError('');
    try { const d = await whiteLabelAdminApi.getOnrampOrders(); setRows(d.orders || d.onramp || d || []); }
    catch (e: any) { setError(e.message || 'Failed to load onramp orders'); }
    finally { setLoading(false); }
  };

  const approve = async (id: string) => {
    try { await whiteLabelAdminApi.approveOnrampOrder(id); load(); }
    catch (e: any) { setError(e.message || 'Failed to approve'); }
  };

  const doReject = async () => {
    if (!rejectId || !reason) { setError('Reason required'); return; }
    try { await whiteLabelAdminApi.rejectOnrampOrder(rejectId, reason); setRejectId(null); setReason(''); load(); }
    catch (e: any) { setError(e.message || 'Failed to reject'); }
  };

  const cardBg = isDark ? 'bg-gray-800' : 'bg-white';
  const cardText = isDark ? 'text-white' : 'text-gray-900';
  const border = isDark ? 'border-gray-700' : 'border-gray-200';
  const muted = isDark ? 'text-gray-400' : 'text-gray-500';
  const thBg = isDark ? 'bg-gray-700' : 'bg-gray-50';
  const inputCls = isDark ? 'bg-gray-700 text-white border-gray-600' : 'bg-white text-gray-900 border-gray-300';

  return (
    <div className="p-6">
      <h1 className={`text-2xl font-bold mb-2 ${cardText}`}>Onramp</h1>
      <p className={`mb-4 ${muted}`}>Fiat-onramp order governance — approve / reject (no fund movement).</p>

      {error && <div className={`mb-3 p-2 rounded text-sm ${isDark ? 'bg-red-900/50 text-red-200' : 'bg-red-50 text-red-700'}`}>{error}</div>}
      {loading && <div className={muted}>Loading…</div>}

      {rejectId && (
        <div className={`${cardBg} p-4 rounded-lg shadow border ${border} mb-4`}>
          <h2 className={`text-sm font-semibold mb-2 ${cardText}`}>Reject order {rejectId}</h2>
          <textarea value={reason} onChange={(e) => setReason(e.target.value)} placeholder="Reason" className={`w-full px-3 py-2 rounded border ${inputCls} mb-2`} />
          <div className="flex gap-2">
            <button onClick={doReject} className="px-4 py-2 rounded bg-red-600 text-white">Reject</button>
            <button onClick={() => { setRejectId(null); setReason(''); }} className={`px-4 py-2 rounded border ${inputCls}`}>Cancel</button>
          </div>
        </div>
      )}

      {!loading && (
        <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border}`}>
          <table className="w-full">
            <thead className={thBg}>
              <tr>
                {['User', 'Fiat', 'Amount', 'Crypto', 'Status', 'Actions'].map((h) => (
                  <th key={h} className={`px-4 py-2 text-left text-xs font-semibold ${muted}`}>{h}</th>
                ))}
              </tr>
            </thead>
            <tbody className={`divide-y ${border}`}>
              {rows.length === 0 && <tr><td colSpan={6} className={`px-4 py-8 text-center ${muted}`}>No onramp orders.</td></tr>}
              {rows.map((r) => (
                <tr key={r.id}>
                  <td className={`px-4 py-3 ${cardText}`}>{r.user_id}</td>
                  <td className={`px-4 py-3 ${muted}`}>{r.fiat_currency}</td>
                  <td className={`px-4 py-3 ${muted}`}>{r.fiat_amount}</td>
                  <td className={`px-4 py-3 ${muted}`}>{r.crypto_currency}</td>
                  <td className={`px-4 py-3 capitalize ${cardText}`}>{r.status}</td>
                  <td className="px-4 py-3 flex gap-2">
                    {r.status === 'pending' && (
                      <>
                        <button onClick={() => approve(r.id)} className="text-green-500 text-xs">Approve</button>
                        <button onClick={() => { setRejectId(r.id); setReason(''); }} className="text-red-500 text-xs">Reject</button>
                      </>
                    )}
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
