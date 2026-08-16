// Generic admin domain page: list + loading/error/empty states + actions.
// Used by the 12 admin domains (futures, options, copy-trading, convert,
// onramp, offramp, p2p-clients, p2p-merchants, partners, rewards, marketing,
// roles, permissions) against the admin/go backend on port 9093.

import React, { useState, useEffect } from 'react';
import { api, getColors, Theme } from './App';

export interface DomainPageConfig {
  id: string;          // page id used by the sidebar / switcher
  title: string;
  domain: string;      // /api/v1/<domain>
  columns: string[];   // headers
  renderRow: (r: any) => React.ReactNode[];
  // Action modes: 'status' toggles active/paused via /:id/status PUT;
  // 'approve-reject' uses /:id/approve + /:id/reject POST {reason};
  // 'none' is read-only.
  action: 'status' | 'approve-reject' | 'none';
  // Optional alternative status toggle values (defaults active/paused).
  statusValues?: [string, string];
  // Whether the delete action is exposed. Defaults to true; set false for
  // domains whose backend has no DELETE route (e.g. p2p-merchants).
  canDelete?: boolean;
}

export const DOMAIN_PAGES: DomainPageConfig[] = [
  {
    id: 'futures', title: 'Futures', domain: 'futures', action: 'status',
    columns: ['Name', 'Symbol', 'Leverage', 'Margin', 'Status'],
    renderRow: r => [r.name ?? '—', r.symbol ?? '—', r.leverage ? `${r.leverage}x` : '—', r.margin ?? '—', r.status ?? '—'],
  },
  {
    id: 'options', title: 'Options', domain: 'options', action: 'status',
    columns: ['Name', 'Symbol', 'Strike', 'Expiry', 'Status'],
    renderRow: r => [r.name ?? '—', r.symbol ?? '—', r.strike ?? '—', r.expiry ?? '—', r.status ?? '—'],
  },
  {
    id: 'copy-trading', title: 'Copy Trading', domain: 'copy-trading', action: 'status',
    columns: ['Strategy', 'Trader', 'Followers', 'Status'],
    renderRow: r => [r.name ?? '—', r.trader ?? '—', r.followers ?? 0, r.status ?? '—'],
  },
  {
    id: 'convert', title: 'Convert', domain: 'convert', action: 'status',
    columns: ['From', 'To', 'Amount', 'Rate', 'Status'],
    renderRow: r => [r.from_asset ?? '—', r.to_asset ?? '—', r.amount ?? '—', r.rate ?? '—', r.status ?? '—'],
  },
  {
    id: 'onramp', title: 'On-Ramp', domain: 'onramp', action: 'approve-reject',
    columns: ['User', 'Asset', 'Amount', 'Provider', 'Status'],
    renderRow: r => [r.user ?? '—', r.asset ?? r.symbol ?? '—', r.amount ?? '—', r.provider ?? '—', r.status ?? '—'],
  },
  {
    id: 'offramp', title: 'Off-Ramp', domain: 'offramp', action: 'approve-reject',
    columns: ['User', 'Asset', 'Amount', 'Provider', 'Status'],
    renderRow: r => [r.user ?? '—', r.asset ?? r.symbol ?? '—', r.amount ?? '—', r.provider ?? '—', r.status ?? '—'],
  },
  {
    id: 'p2p-clients', title: 'P2P Clients', domain: 'p2p-clients', action: 'status',
    statusValues: ['active', 'suspended'],
    columns: ['Name', 'Email', 'Status'],
    renderRow: r => [r.name ?? '—', r.email ?? '—', r.status ?? '—'],
  },
  {
    id: 'p2p-merchants', title: 'P2P Merchants', domain: 'p2p-merchants', action: 'approve-reject', canDelete: false,
    columns: ['Name', 'Email', 'Verified', 'Status'],
    renderRow: r => [r.name ?? '—', r.email ?? '—', r.verified ? 'yes' : 'no', r.status ?? '—'],
  },
  {
    id: 'partners', title: 'Partners', domain: 'partners', action: 'approve-reject',
    columns: ['Name', 'Type', 'Status'],
    renderRow: r => [r.name ?? '—', r.type ?? '—', r.status ?? '—'],
  },
  {
    id: 'rewards', title: 'Rewards', domain: 'rewards', action: 'status',
    columns: ['Name', 'Type', 'Amount', 'Status'],
    renderRow: r => [r.name ?? '—', r.type ?? '—', r.amount ?? '—', r.status ?? '—'],
  },
  {
    id: 'marketing', title: 'Marketing', domain: 'marketing', action: 'status',
    columns: ['Name', 'Campaign', 'Status'],
    renderRow: r => [r.name ?? '—', r.campaign ?? '—', r.status ?? '—'],
  },
  {
    id: 'roles', title: 'Roles', domain: 'roles', action: 'none',
    columns: ['Name', 'Description', 'Permissions'],
    renderRow: r => [r.name ?? '—', r.description ?? '—', (r.permissions || []).join(', ') || '—'],
  },
  {
    id: 'permissions', title: 'Permissions', domain: 'permissions', action: 'none',
    columns: ['Name', 'Resource', 'Action', 'Description'],
    renderRow: r => [r.name ?? '—', r.resource ?? '—', r.action ?? '—', r.description ?? '—'],
  },
];

export const DomainPage: React.FC<{ config: DomainPageConfig; theme: Theme }> = ({ config, theme }) => {
  const colors = getColors(theme);
  const [records, setRecords] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = () => {
    setLoading(true); setError(null);
    api.listDomain(config.domain)
      .then((res: any) => setRecords(Array.isArray(res) ? res : (res.data || [])))
      .catch(e => { setError(e.message || 'Failed to load'); setRecords([]); })
      .finally(() => setLoading(false));
  };

  useEffect(() => { load(); }, [config.domain]);

  const [active, paused] = config.statusValues || ['active', 'paused'];

  const toggleStatus = async (r: any) => {
    const next = String(r.status || '').toLowerCase() === active ? paused : active;
    try { await api.setDomainStatus(config.domain, r.id, next); load(); }
    catch (e: any) { alert(e.message || 'Failed'); }
  };
  const approve = async (r: any) => {
    try { await api.approveDomain(config.domain, r.id); load(); }
    catch (e: any) { alert(e.message || 'Failed'); }
  };
  const reject = async (r: any) => {
    const reason = prompt('Rejection reason:');
    if (!reason) return;
    try { await api.rejectDomain(config.domain, r.id, reason); load(); }
    catch (e: any) { alert(e.message || 'Failed'); }
  };
  const remove = async (r: any) => {
    if (!confirm('Delete this record?')) return;
    try { await api.deleteDomain(config.domain, r.id); load(); }
    catch (e: any) { alert(e.message || 'Failed'); }
  };

  const colCount = config.columns.length + (config.action !== 'none' ? 1 : 0);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold" style={{ color: colors.text }}>{config.title}</h1>
        <button onClick={load} className="px-3 py-1 rounded text-sm" style={{ backgroundColor: colors.bgHover, color: colors.text }}>Refresh</button>
      </div>

      <div className="rounded-xl overflow-hidden" style={{ backgroundColor: colors.bgCard }}>
        <table className="w-full">
          <thead style={{ backgroundColor: colors.bgHover }}>
            <tr>
              {config.columns.map(h => (
                <th key={h} className="px-6 py-3 text-left" style={{ color: colors.text }}>{h}</th>
              ))}
              {config.action !== 'none' && <th key="actions" className="px-6 py-3 text-left" style={{ color: colors.text }}>Actions</th>}
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr><td colSpan={colCount} className="p-6 text-center" style={{ color: colors.textSecondary }}>Loading...</td></tr>
            ) : error ? (
              <tr><td colSpan={colCount} className="p-6 text-center" style={{ color: colors.error }}>Error: {error}</td></tr>
            ) : records.length === 0 ? (
              <tr><td colSpan={colCount} className="p-6 text-center" style={{ color: colors.textSecondary }}>No records</td></tr>
            ) : records.map(r => (
              <tr key={r.id ?? JSON.stringify(r)} className="border-b" style={{ borderColor: colors.border }}>
                {config.renderRow(r).map((cell, i) => (
                  <td key={i} className="px-6 py-4" style={{ color: colors.text }}>{cell}</td>
                ))}
                {config.action !== 'none' && (
                  <td className="px-6 py-4">
                    <div className="flex gap-2">
                      {config.action === 'status' && (
                        <button onClick={() => toggleStatus(r)} className="px-2 py-1 rounded text-xs" style={{ backgroundColor: colors.warning, color: 'white' }}>
                          {String(r.status || '').toLowerCase() === active ? paused : active}
                        </button>
                      )}
                      {config.action === 'approve-reject' && (
                        <>
                          <button onClick={() => approve(r)} className="px-2 py-1 rounded text-xs" style={{ backgroundColor: colors.success, color: 'white' }}>Approve</button>
                          <button onClick={() => reject(r)} className="px-2 py-1 rounded text-xs" style={{ backgroundColor: colors.error, color: 'white' }}>Reject</button>
                        </>
                      )}
                      {config.canDelete !== false && (
                        <button onClick={() => remove(r)} className="px-2 py-1 rounded text-xs" style={{ backgroundColor: colors.bgHover, color: colors.textSecondary }}>Delete</button>
                      )}
                    </div>
                  </td>
                )}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};
