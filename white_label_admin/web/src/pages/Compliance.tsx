/**
 * Compliance (compliance_admin) — audit logs, regulatory reports, transaction flags.
 * Backend: GET /api/v1/admin/audit-logs, POST /api/v1/admin/audit-logs/export,
 * GET /api/v1/admin/transactions (flagged), POST /api/v1/admin/transactions/:id/flag|unflag.
 */

import React, { useEffect, useMemo, useState } from 'react';
import { whiteLabelAdminApi } from '../services/api';
import { useTheme } from '../context/ThemeContext';

export default function Compliance() {
  const { isDark } = useTheme();
  const [logs, setLogs] = useState<any[]>([]);
  const [txs, setTxs] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [tab, setTab] = useState<'audit' | 'flagged'>('audit');
  const [exporting, setExporting] = useState(false);

  useEffect(() => { load(); }, []);

  const load = async () => {
    setLoading(true); setError('');
    try {
      const [l, t] = await Promise.all([whiteLabelAdminApi.getAuditLogs(), whiteLabelAdminApi.getTransactions()]);
      setLogs(l.audit_logs || []);
      setTxs((t.transactions || []).filter((x: any) => x.status === 'flagged'));
    } catch (e: any) { setError(e.message || 'Failed to load compliance data'); }
    finally { setLoading(false); }
  };

  const doExport = async () => {
    setExporting(true);
    try {
      const blob = await whiteLabelAdminApi.exportAuditLogs();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url; a.download = 'audit_logs.csv'; a.click();
      window.URL.revokeObjectURL(url);
    } catch (e: any) { setError(e.message || 'Export failed'); }
    finally { setExporting(false); }
  };

  const unflag = async (id: string) => { try { await whiteLabelAdminApi.unflagTransaction(id); load(); } catch (e: any) { setError(e.message || 'Failed to unflag'); } };

  const stats = useMemo(() => ({
    auditEvents: logs.length,
    flaggedTx: txs.length,
    distinctActions: new Set(logs.map((l) => l.action)).size,
  }), [logs, txs]);

  const cardBg = isDark ? 'bg-gray-800' : 'bg-white';
  const cardText = isDark ? 'text-white' : 'text-gray-900';
  const border = isDark ? 'border-gray-700' : 'border-gray-200';
  const muted = isDark ? 'text-gray-400' : 'text-gray-500';
  const thBg = isDark ? 'bg-gray-700' : 'bg-gray-50';
  const short = (v: string) => (v ? `${v.substring(0, 8)}…` : '—');

  return (
    <div className="p-6">
      <div className="flex flex-wrap justify-between items-center mb-4">
        <div><h1 className={`text-2xl font-bold ${cardText}`}>Compliance</h1><p className={muted}>Audit trail, regulatory reports and flagged transactions.</p></div>
        <button onClick={doExport} disabled={exporting} className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50">{exporting ? 'Exporting…' : 'Export CSV'}</button>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-3 gap-3 mb-6">
        <div className={`${cardBg} p-4 rounded-lg shadow border ${border}`}><p className={`text-xs ${muted}`}>Audit Events</p><p className={`text-xl font-bold mt-1 ${cardText}`}>{stats.auditEvents}</p></div>
        <div className={`${cardBg} p-4 rounded-lg shadow border ${border}`}><p className={`text-xs ${muted}`}>Flagged Tx</p><p className={`text-xl font-bold mt-1 ${cardText}`}>{stats.flaggedTx}</p></div>
        <div className={`${cardBg} p-4 rounded-lg shadow border ${border}`}><p className={`text-xs ${muted}`}>Distinct Actions</p><p className={`text-xl font-bold mt-1 ${cardText}`}>{stats.distinctActions}</p></div>
      </div>

      <div className="flex gap-2 mb-4">
        <button onClick={() => setTab('audit')} className={`px-3 py-1 rounded text-sm ${tab === 'audit' ? 'bg-blue-600 text-white' : (isDark ? 'bg-gray-700 text-gray-200' : 'bg-gray-200 text-gray-700')}`}>Audit Trail</button>
        <button onClick={() => setTab('flagged')} className={`px-3 py-1 rounded text-sm ${tab === 'flagged' ? 'bg-blue-600 text-white' : (isDark ? 'bg-gray-700 text-gray-200' : 'bg-gray-200 text-gray-700')}`}>Flagged Transactions</button>
      </div>

      {error && <div className={`mb-3 p-2 rounded text-sm ${isDark ? 'bg-red-900/50 text-red-200' : 'bg-red-50 text-red-700'}`}>{error}</div>}
      {loading && <div className={muted}>Loading…</div>}

      {!loading && tab === 'audit' && (
        <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border}`}>
          <table className="w-full">
            <thead className={thBg}><tr>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Admin</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Action</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Resource</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>IP</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Time</th>
            </tr></thead>
            <tbody className={`divide-y ${border}`}>
              {logs.length === 0 && <tr><td colSpan={5} className={`px-6 py-8 text-center ${muted}`}>No audit events.</td></tr>}
              {logs.map((l) => (
                <tr key={l.id}>
                  <td className={`px-6 py-3 font-mono text-xs ${muted}`}>{short(l.admin_id)}</td>
                  <td className={`px-6 py-3 ${cardText}`}>{l.action}</td>
                  <td className={`px-6 py-3 ${muted}`}>{l.resource_type} {l.resource_id ? short(l.resource_id) : ''}</td>
                  <td className={`px-6 py-3 ${muted} text-xs`}>{l.ip_address || '—'}</td>
                  <td className={`px-6 py-3 text-xs ${muted}`}>{l.created_at ? new Date(l.created_at).toLocaleString() : '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {!loading && tab === 'flagged' && (
        <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border}`}>
          <table className="w-full">
            <thead className={thBg}><tr>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Type</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Amount</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Asset</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>User</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Time</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Action</th>
            </tr></thead>
            <tbody className={`divide-y ${border}`}>
              {txs.length === 0 && <tr><td colSpan={6} className={`px-6 py-8 text-center ${muted}`}>No flagged transactions.</td></tr>}
              {txs.map((t) => (
                <tr key={t.id}>
                  <td className={`px-6 py-4 ${cardText}`}>{t.type}</td>
                  <td className={`px-6 py-4 ${cardText}`}>{t.amount}</td>
                  <td className={`px-6 py-4 ${cardText}`}>{t.currency}</td>
                  <td className={`px-6 py-4 font-mono text-xs ${muted}`}>{short(t.user_id)}</td>
                  <td className={`px-6 py-4 text-xs ${muted}`}>{t.timestamp ? new Date(t.timestamp).toLocaleString() : '—'}</td>
                  <td className="px-6 py-4"><button onClick={() => unflag(t.id)} className="text-green-600 hover:underline text-sm">Unflag</button></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
