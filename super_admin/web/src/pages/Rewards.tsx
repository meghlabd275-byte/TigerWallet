/**
 * TigerWallet Super Admin - Rewards Campaigns Page
 * Governance records only — never moves crypto assets.
 */

import React, { useState, useEffect } from 'react';
import { superAdminApi } from '../services/api';

const STATUSES = ['active', 'paused', 'suspended', 'halted', 'completed'];

export default function Rewards() {
  const [items, setItems] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({ name: '', reward_type: '', amount: '', token: '', start_at: '', end_at: '' });

  const load = async () => {
    try {
      setLoading(true);
      setError(null);
      const res: any = await superAdminApi.getRewards();
      setItems(res.campaigns || res.data || res || []);
    } catch (err: any) {
      setError(err?.message || 'Failed to load reward campaigns');
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
      await superAdminApi.createReward({
        name: form.name,
        reward_type: form.reward_type,
        amount: form.amount ? Number(form.amount) : undefined,
        token: form.token || undefined,
        start_at: form.start_at || undefined,
        end_at: form.end_at || undefined,
      });
      setShowForm(false);
      setForm({ name: '', reward_type: '', amount: '', token: '', start_at: '', end_at: '' });
      load();
    } catch (err: any) {
      alert(err?.message || 'Failed to create reward campaign');
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
      <h1 className="text-2xl font-bold text-primary mb-6">Rewards Campaigns</h1>

      <div className="flex justify-between items-center mb-4">
        <button className="btn btn-primary" onClick={() => setShowForm((s) => !s)}>
          {showForm ? 'Cancel' : 'New Campaign'}
        </button>
      </div>

      {showForm && (
        <div className="card mb-6"><div className="card-body">
          <h3 className="text-primary mb-4">Create Reward Campaign</h3>
          <form onSubmit={handleCreate} className="flex flex-col gap-3">
            <div className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">Name</label><input className="input w-full" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} required /></div>
              <div className="form-group flex-1"><label className="text-secondary">Reward Type</label><input className="input w-full" value={form.reward_type} onChange={(e) => setForm({ ...form, reward_type: e.target.value })} required /></div>
              <div className="form-group flex-1"><label className="text-secondary">Amount</label><input className="input w-full" type="number" step="any" value={form.amount} onChange={(e) => setForm({ ...form, amount: e.target.value })} /></div>
            </div>
            <div className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">Token</label><input className="input w-full" value={form.token} onChange={(e) => setForm({ ...form, token: e.target.value })} /></div>
              <div className="form-group flex-1"><label className="text-secondary">Start At</label><input className="input w-full" placeholder="2026-01-01" value={form.start_at} onChange={(e) => setForm({ ...form, start_at: e.target.value })} /></div>
              <div className="form-group flex-1"><label className="text-secondary">End At</label><input className="input w-full" placeholder="2026-12-31" value={form.end_at} onChange={(e) => setForm({ ...form, end_at: e.target.value })} /></div>
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
        <div className="card"><div className="card-body text-center py-8"><p className="text-secondary">No reward campaigns found.</p></div></div>
      ) : (
        <div className="card"><div className="card-body overflow-x-auto">
          <table className="table"><thead><tr><th>Name</th><th>Type</th><th>Amount</th><th>Token</th><th>Window</th><th>Status</th><th>Actions</th></tr></thead>
            <tbody>
              {items.map((c) => (
                <tr key={c.id}>
                  <td className="text-primary">{c.name}</td>
                  <td className="text-secondary">{c.reward_type}</td>
                  <td className="text-secondary">{c.amount}</td>
                  <td className="text-secondary">{c.token}</td>
                  <td className="text-secondary">{c.start_at} → {c.end_at}</td>
                  <td><span className={`badge ${c.status === 'active' ? 'badge-success' : c.status === 'halted' ? 'badge-error' : 'badge-warning'}`}>{c.status}</span></td>
                  <td><div className="flex gap-2" style={{ flexWrap: 'wrap' }}>
                    <select className="input" value={c.status || ''} disabled={actionLoading} onChange={(e) => run(() => superAdminApi.updateRewardStatus(c.id, e.target.value))}>
                      {STATUSES.map((s) => <option key={s} value={s}>{s}</option>)}
                    </select>
                    <button className="btn btn-danger" disabled={actionLoading} onClick={() => { if (confirm('Delete this campaign?')) run(() => superAdminApi.deleteReward(c.id)); }}>Delete</button>
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
