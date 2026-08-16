/**
 * TigerWallet Super Admin - Copy Trading Configs Page
 * Governance records only — never moves crypto assets.
 */

import React, { useState, useEffect } from 'react';
import { superAdminApi } from '../services/api';

const STATUSES = ['active', 'paused', 'suspended', 'halted'];

export default function CopyTrading() {
  const [items, setItems] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({ follower_id: '', leader_id: '', allocation: '', max_leverage: '' });

  const load = async () => {
    try {
      setLoading(true);
      setError(null);
      const res: any = await superAdminApi.getCopyTrading();
      setItems(res.configs || res.data || res || []);
    } catch (err: any) {
      setError(err?.message || 'Failed to load copy trading configs');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setActionLoading(true);
    try {
      await superAdminApi.createCopyTrading({
        follower_id: form.follower_id || undefined,
        leader_id: form.leader_id || undefined,
        allocation: form.allocation ? Number(form.allocation) : undefined,
        max_leverage: form.max_leverage ? Number(form.max_leverage) : undefined,
      });
      setShowForm(false);
      setForm({ follower_id: '', leader_id: '', allocation: '', max_leverage: '' });
      load();
    } catch (err: any) {
      alert(err?.message || 'Failed to create copy trading config');
    } finally {
      setActionLoading(false);
    }
  };

  const run = async (fn: () => Promise<any>) => {
    setActionLoading(true);
    try {
      await fn();
      load();
    } catch (err: any) {
      alert(err?.message || 'Action failed');
    } finally {
      setActionLoading(false);
    }
  };

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold text-primary mb-6">Copy Trading Configs</h1>

      <div className="flex justify-between items-center mb-4">
        <button className="btn btn-primary" onClick={() => setShowForm((s) => !s)}>
          {showForm ? 'Cancel' : 'New Config'}
        </button>
      </div>

      {showForm && (
        <div className="card mb-6"><div className="card-body">
          <h3 className="text-primary mb-4">Create Copy Trading Config</h3>
          <form onSubmit={handleCreate} className="flex flex-col gap-3">
            <div className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">Follower ID</label><input className="input w-full" value={form.follower_id} onChange={(e) => setForm({ ...form, follower_id: e.target.value })} /></div>
              <div className="form-group flex-1"><label className="text-secondary">Leader ID</label><input className="input w-full" value={form.leader_id} onChange={(e) => setForm({ ...form, leader_id: e.target.value })} /></div>
            </div>
            <div className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">Allocation</label><input className="input w-full" type="number" step="any" value={form.allocation} onChange={(e) => setForm({ ...form, allocation: e.target.value })} /></div>
              <div className="form-group flex-1"><label className="text-secondary">Max Leverage</label><input className="input w-full" type="number" step="any" value={form.max_leverage} onChange={(e) => setForm({ ...form, max_leverage: e.target.value })} /></div>
            </div>
            <button className="btn btn-primary" disabled={actionLoading} type="submit">Create</button>
          </form>
        </div></div>
      )}

      {error ? (
        <div className="alert alert-error mb-4"><p className="text-error">{error}</p><button className="btn btn-secondary mt-2" onClick={load}>Retry</button></div>
      ) : loading ? (
        <div className="flex items-center justify-center p-8"><div className="loader"></div></div>
      ) : items.length === 0 ? (
        <div className="card"><div className="card-body text-center py-8"><p className="text-secondary">No copy trading configs found.</p></div></div>
      ) : (
        <div className="card"><div className="card-body overflow-x-auto">
          <table className="table"><thead><tr><th>Follower</th><th>Leader</th><th>Allocation</th><th>Max Leverage</th><th>Status</th><th>Actions</th></tr></thead>
            <tbody>
              {items.map((c) => (
                <tr key={c.id}>
                  <td className="text-secondary">{c.follower_id}</td>
                  <td className="text-secondary">{c.leader_id}</td>
                  <td className="text-primary">{c.allocation}</td>
                  <td className="text-secondary">{c.max_leverage}</td>
                  <td><span className={`badge ${c.status === 'active' ? 'badge-success' : c.status === 'halted' ? 'badge-error' : 'badge-warning'}`}>{c.status}</span></td>
                  <td><div className="flex gap-2" style={{ flexWrap: 'wrap' }}>
                    <select className="input" value={c.status || ''} disabled={actionLoading} onChange={(e) => run(() => superAdminApi.updateCopyTradingStatus(c.id, e.target.value))}>
                      {STATUSES.map((s) => <option key={s} value={s}>{s}</option>)}
                    </select>
                    <button className="btn btn-danger" disabled={actionLoading} onClick={() => { if (confirm('Delete this config?')) run(() => superAdminApi.deleteCopyTrading(c.id)); }}>Delete</button>
                  </div></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div></div>
      )}
    </div>
  );
}
