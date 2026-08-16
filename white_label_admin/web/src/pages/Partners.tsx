/**
 * Partners — partner governance: status + approve/reject (no fund movement).
 * Backend: GET/POST /api/v1/admin/partners, GET/PUT/DELETE /:id,
 * POST /:id/status, POST /:id/approve, POST /:id/reject {reason}.
 */

import React, { useEffect, useState } from 'react';
import { whiteLabelAdminApi } from '../services/api';
import { useTheme } from '../context/ThemeContext';

const STATUSES = ['pending', 'active', 'suspended', 'rejected'];

export default function Partners() {
  const { isDark } = useTheme();
  const [rows, setRows] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [name, setName] = useState('');
  const [partnerType, setPartnerType] = useState('liquidity');
  const [rejectId, setRejectId] = useState<string | null>(null);
  const [reason, setReason] = useState('');

  useEffect(() => { load(); }, []);

  const load = async () => {
    setLoading(true); setError('');
    try { const d = await whiteLabelAdminApi.getPartners(); setRows(d.partners || d || []); }
    catch (e: any) { setError(e.message || 'Failed to load partners'); }
    finally { setLoading(false); }
  };

  const create = async () => {
    if (!name) { setError('Name required'); return; }
    try { await whiteLabelAdminApi.createPartner({ name, partner_type: partnerType }); setName(''); load(); }
    catch (e: any) { setError(e.message || 'Failed to create'); }
  };

  const setStatus = async (id: string, status: string) => {
    try { await whiteLabelAdminApi.updatePartnerStatus(id, status); load(); }
    catch (e: any) { setError(e.message || 'Failed to update status'); }
  };

  const approve = async (id: string) => {
    try { await whiteLabelAdminApi.approvePartner(id); load(); }
    catch (e: any) { setError(e.message || 'Failed to approve'); }
  };

  const doReject = async () => {
    if (!rejectId || !reason) { setError('Reason required'); return; }
    try { await whiteLabelAdminApi.rejectPartner(rejectId, reason); setRejectId(null); setReason(''); load(); }
    catch (e: any) { setError(e.message || 'Failed to reject'); }
  };

  const remove = async (id: string) => {
    if (!confirm('Delete this partner?')) return;
    try { await whiteLabelAdminApi.deletePartner(id); load(); }
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
      <h1 className={`text-2xl font-bold mb-2 ${cardText}`}>Partners</h1>
      <p className={`mb-4 ${muted}`}>Partner governance — status, approve / reject (no fund movement).</p>

      <div className={`${cardBg} p-4 rounded-lg shadow border ${border} mb-6`}>
        <h2 className={`text-sm font-semibold mb-3 ${cardText}`}>New Partner</h2>
        <div className="flex flex-wrap gap-2">
          <input value={name} onChange={(e) => setName(e.target.value)} placeholder="Partner name" className={`px-3 py-2 rounded border ${inputCls}`} />
          <select value={partnerType} onChange={(e) => setPartnerType(e.target.value)} className={`px-3 py-2 rounded border ${inputCls}`}>
            <option value="liquidity">Liquidity</option><option value="payment">Payment</option><option value="oracle">Oracle</option><option value="kyc">KYC</option>
          </select>
          <button onClick={create} className="px-4 py-2 rounded bg-blue-600 text-white">Create</button>
        </div>
      </div>

      {rejectId && (
        <div className={`${cardBg} p-4 rounded-lg shadow border ${border} mb-4`}>
          <h2 className={`text-sm font-semibold mb-2 ${cardText}`}>Reject partner {rejectId}</h2>
          <textarea value={reason} onChange={(e) => setReason(e.target.value)} placeholder="Reason" className={`w-full px-3 py-2 rounded border ${inputCls} mb-2`} />
          <div className="flex gap-2">
            <button onClick={doReject} className="px-4 py-2 rounded bg-red-600 text-white">Reject</button>
            <button onClick={() => { setRejectId(null); setReason(''); }} className={`px-4 py-2 rounded border ${inputCls}`}>Cancel</button>
          </div>
        </div>
      )}

      {error && <div className={`mb-3 p-2 rounded text-sm ${isDark ? 'bg-red-900/50 text-red-200' : 'bg-red-50 text-red-700'}`}>{error}</div>}
      {loading && <div className={muted}>Loading…</div>}

      {!loading && (
        <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border}`}>
          <table className="w-full">
            <thead className={thBg}>
              <tr>
                {['Name', 'Type', 'Status', 'Actions'].map((h) => (
                  <th key={h} className={`px-4 py-2 text-left text-xs font-semibold ${muted}`}>{h}</th>
                ))}
              </tr>
            </thead>
            <tbody className={`divide-y ${border}`}>
              {rows.length === 0 && <tr><td colSpan={4} className={`px-4 py-8 text-center ${muted}`}>No partners.</td></tr>}
              {rows.map((r) => (
                <tr key={r.id}>
                  <td className={`px-4 py-3 ${cardText}`}>{r.name}</td>
                  <td className={`px-4 py-3 ${muted}`}>{r.partner_type}</td>
                  <td className="px-4 py-3">
                    <select value={r.status || 'pending'} onChange={(e) => setStatus(r.id, e.target.value)} className={`px-2 py-1 rounded border text-xs ${inputCls}`}>
                      {STATUSES.map((s) => <option key={s} value={s}>{s}</option>)}
                    </select>
                  </td>
                  <td className="px-4 py-3 flex gap-2">
                    {r.status === 'pending' && (
                      <>
                        <button onClick={() => approve(r.id)} className="text-green-500 text-xs">Approve</button>
                        <button onClick={() => { setRejectId(r.id); setReason(''); }} className="text-red-500 text-xs">Reject</button>
                      </>
                    )}
                    <button onClick={() => remove(r.id)} className="text-red-500 text-xs">Delete</button>
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
