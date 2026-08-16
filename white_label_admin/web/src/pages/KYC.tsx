/**
 * KYC Page - White Label Admin
 */

import React, { useEffect, useState } from 'react';
import { whiteLabelAdminApi } from '../services/api';
import { useTheme } from '../context/ThemeContext';

export default function KYC() {
  const { isDark } = useTheme();
  const [requests, setRequests] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => { loadKYC(); }, []);

  const loadKYC = async () => {
    setLoading(true); setError('');
    try {
      const data = await whiteLabelAdminApi.getKYCRequests();
      setRequests(data.kyc_requests || []);
    } catch (e: any) { setError(e.message || 'Failed to load KYC requests'); }
    finally { setLoading(false); }
  };

  const handleApprove = async (id: string) => {
    try { await whiteLabelAdminApi.approveKYC(id); loadKYC(); }
    catch (e: any) { setError(e.message || 'Failed to approve KYC'); }
  };

  const handleReject = async (id: string) => {
    try { await whiteLabelAdminApi.rejectKYC(id, 'Rejected by admin'); loadKYC(); }
    catch (e: any) { setError(e.message || 'Failed to reject KYC'); }
  };

  const cardBg = isDark ? 'bg-gray-800' : 'bg-white';
  const cardText = isDark ? 'text-white' : 'text-gray-900';
  const border = isDark ? 'border-gray-700' : 'border-gray-200';
  const muted = isDark ? 'text-gray-400' : 'text-gray-500';
  const thBg = isDark ? 'bg-gray-700' : 'bg-gray-50';

  const badge = (status: string) => {
    const base = 'px-2 py-1 text-xs rounded';
    if (status === 'approved') return `${base} ${isDark ? 'bg-green-900 text-green-200' : 'bg-green-100 text-green-800'}`;
    if (status === 'pending') return `${base} ${isDark ? 'bg-yellow-900 text-yellow-200' : 'bg-yellow-100 text-yellow-800'}`;
    return `${base} ${isDark ? 'bg-red-900 text-red-200' : 'bg-red-100 text-red-800'}`;
  };

  const short = (v: string) => (v ? `${v.substring(0, 8)}…` : '—');

  return (
    <div className="p-6">
      <h1 className={`text-2xl font-bold mb-6 ${cardText}`}>KYC Management</h1>
      {error && <div className={`mb-4 p-3 rounded ${isDark ? 'bg-red-900/50 text-red-200' : 'bg-red-50 text-red-700'}`}>{error}</div>}
      {loading && <div className={muted}>Loading...</div>}
      {!loading && (
        <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border}`}>
          <table className="w-full">
            <thead className={thBg}>
              <tr>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>User</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Document</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Status</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Submitted</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Actions</th>
              </tr>
            </thead>
            <tbody className={`divide-y ${border}`}>
              {requests.length === 0 && (
                <tr><td colSpan={5} className={`px-6 py-8 text-center ${muted}`}>No KYC requests.</td></tr>
              )}
              {requests.map((req) => (
                <tr key={req.id}>
                  <td className={`px-6 py-4 font-mono text-xs ${muted}`}>{short(req.user_id)}</td>
                  <td className={`px-6 py-4 ${cardText}`}>{req.doc_type}{req.document_url ? <a href={req.document_url} target="_blank" rel="noreferrer" className="ml-2 text-blue-600 hover:underline">view</a> : null}</td>
                  <td className="px-6 py-4"><span className={badge(req.status)}>{req.status}</span></td>
                  <td className={`px-6 py-4 text-xs ${muted}`}>{req.submitted_at ? new Date(req.submitted_at).toLocaleString() : '—'}</td>
                  <td className="px-6 py-4 space-x-2">
                    {req.status === 'pending' && (
                      <>
                        <button onClick={() => handleApprove(req.id)} className="text-green-600 hover:underline text-sm">Approve</button>
                        <button onClick={() => handleReject(req.id)} className="text-red-600 hover:underline text-sm">Reject</button>
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
