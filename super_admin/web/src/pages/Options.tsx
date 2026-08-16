/**
 * TigerWallet Super Admin - Options Contracts Page
 * Governance records only — never moves crypto assets.
 */

import React, { useState, useEffect } from 'react';
import { superAdminApi } from '../services/api';

const STATUSES = ['active', 'expired', 'exercised', 'paused', 'suspended'];

export default function Options() {
  const [items, setItems] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({
    underlying: '', option_type: 'call', strike: '', expiry: '', premium: '', size: '', chain_id: '',
  });

  const load = async () => {
    try {
      setLoading(true);
      setError(null);
      const res: any = await superAdminApi.getOptions();
      setItems(res.contracts || res.data || res || []);
    } catch (err: any) {
      setError(err?.message || 'Failed to load options contracts');
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
      await superAdminApi.createOption({
        underlying: form.underlying,
        option_type: form.option_type,
        strike: form.strike ? Number(form.strike) : undefined,
        expiry: form.expiry || undefined,
        premium: form.premium ? Number(form.premium) : undefined,
        size: form.size ? Number(form.size) : undefined,
        chain_id: form.chain_id ? Number(form.chain_id) : undefined,
      });
      setShowForm(false);
      setForm({ underlying: '', option_type: 'call', strike: '', expiry: '', premium: '', size: '', chain_id: '' });
      load();
    } catch (err: any) {
      alert(err?.message || 'Failed to create options contract');
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
      <h1 className="text-2xl font-bold text-primary mb-6">Options Contracts</h1>

      <div className="flex justify-between items-center mb-4">
        <button className="btn btn-primary" onClick={() => setShowForm((s) => !s)}>
          {showForm ? 'Cancel' : 'New Contract'}
        </button>
      </div>

      {showForm && (
        <div className="card mb-6"><div className="card-body">
          <h3 className="text-primary mb-4">Create Options Contract</h3>
          <form onSubmit={handleCreate} className="flex flex-col gap-3">
            <div className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">Underlying</label><input className="input w-full" value={form.underlying} onChange={(e) => setForm({ ...form, underlying: e.target.value })} required /></div>
              <div className="form-group flex-1"><label className="text-secondary">Type</label><select className="input w-full" value={form.option_type} onChange={(e) => setForm({ ...form, option_type: e.target.value })}><option>call</option><option>put</option></select></div>
              <div className="form-group flex-1"><label className="text-secondary">Strike</label><input className="input w-full" type="number" step="any" value={form.strike} onChange={(e) => setForm({ ...form, strike: e.target.value })} /></div>
              <div className="form-group flex-1"><label className="text-secondary">Expiry</label><input className="input w-full" placeholder="2026-12-31" value={form.expiry} onChange={(e) => setForm({ ...form, expiry: e.target.value })} /></div>
            </div>
            <div className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">Premium</label><input className="input w-full" type="number" step="any" value={form.premium} onChange={(e) => setForm({ ...form, premium: e.target.value })} /></div>
              <div className="form-group flex-1"><label className="text-secondary">Size</label><input className="input w-full" type="number" step="any" value={form.size} onChange={(e) => setForm({ ...form, size: e.target.value })} /></div>
              <div className="form-group flex-1"><label className="text-secondary">Chain ID</label><input className="input w-full" type="number" value={form.chain_id} onChange={(e) => setForm({ ...form, chain_id: e.target.value })} /></div>
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
        <div className="card"><div className="card-body text-center py-8"><p className="text-secondary">No options contracts found.</p></div></div>
      ) : (
        <div className="card"><div className="card-body overflow-x-auto">
          <table className="table"><thead><tr><th>Underlying</th><th>Type</th><th>Strike</th><th>Expiry</th><th>Premium</th><th>Status</th><th>Actions</th></tr></thead>
            <tbody>
              {items.map((c) => (
                <tr key={c.id}>
                  <td className="text-primary">{c.underlying}</td>
                  <td className="text-secondary">{c.option_type}</td>
                  <td className="text-secondary">{c.strike}</td>
                  <td className="text-secondary">{c.expiry}</td>
                  <td className="text-secondary">{c.premium}</td>
                  <td><span className={`badge ${c.status === 'active' ? 'badge-success' : c.status === 'expired' ? 'badge-neutral' : 'badge-warning'}`}>{c.status}</span></td>
                  <td><div className="flex gap-2" style={{ flexWrap: 'wrap' }}>
                    <select className="input" value={c.status || ''} disabled={actionLoading} onChange={(e) => run(() => superAdminApi.updateOptionStatus(c.id, e.target.value))}>
                      {STATUSES.map((s) => <option key={s} value={s}>{s}</option>)}
                    </select>
                    <button className="btn btn-danger" disabled={actionLoading} onClick={() => { if (confirm('Delete this contract?')) run(() => superAdminApi.deleteOption(c.id)); }}>Delete</button>
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
