/**
 * TigerWallet Super Admin - Convert Orders Page
 * Governance records only — never moves crypto assets.
 */

import React, { useState, useEffect } from 'react';
import { superAdminApi } from '../services/api';

const STATUSES = ['pending', 'completed', 'failed', 'cancelled', 'paused'];

export default function Convert() {
  const [items, setItems] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({
    user_id: '', from_token: '', to_token: '', from_amount: '', to_amount: '', rate: '', chain_id: '',
  });

  const load = async () => {
    try {
      setLoading(true);
      setError(null);
      const res: any = await superAdminApi.getConvert();
      setItems(res.orders || res.data || res || []);
    } catch (err: any) {
      setError(err?.message || 'Failed to load convert orders');
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
      await superAdminApi.createConvert({
        user_id: form.user_id || undefined,
        from_token: form.from_token,
        to_token: form.to_token,
        from_amount: form.from_amount ? Number(form.from_amount) : undefined,
        to_amount: form.to_amount ? Number(form.to_amount) : undefined,
        rate: form.rate ? Number(form.rate) : undefined,
        chain_id: form.chain_id ? Number(form.chain_id) : undefined,
      });
      setShowForm(false);
      setForm({ user_id: '', from_token: '', to_token: '', from_amount: '', to_amount: '', rate: '', chain_id: '' });
      load();
    } catch (err: any) {
      alert(err?.message || 'Failed to create convert order');
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
      <h1 className="text-2xl font-bold text-primary mb-6">Convert Orders</h1>

      <div className="flex justify-between items-center mb-4">
        <button className="btn btn-primary" onClick={() => setShowForm((s) => !s)}>
          {showForm ? 'Cancel' : 'New Order'}
        </button>
      </div>

      {showForm && (
        <div className="card mb-6"><div className="card-body">
          <h3 className="text-primary mb-4">Create Convert Order</h3>
          <form onSubmit={handleCreate} className="flex flex-col gap-3">
            <div className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">User ID</label><input className="input w-full" value={form.user_id} onChange={(e) => setForm({ ...form, user_id: e.target.value })} /></div>
              <div className="form-group flex-1"><label className="text-secondary">From Token</label><input className="input w-full" value={form.from_token} onChange={(e) => setForm({ ...form, from_token: e.target.value })} required /></div>
              <div className="form-group flex-1"><label className="text-secondary">To Token</label><input className="input w-full" value={form.to_token} onChange={(e) => setForm({ ...form, to_token: e.target.value })} required /></div>
            </div>
            <div className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">From Amount</label><input className="input w-full" type="number" step="any" value={form.from_amount} onChange={(e) => setForm({ ...form, from_amount: e.target.value })} /></div>
              <div className="form-group flex-1"><label className="text-secondary">To Amount</label><input className="input w-full" type="number" step="any" value={form.to_amount} onChange={(e) => setForm({ ...form, to_amount: e.target.value })} /></div>
              <div className="form-group flex-1"><label className="text-secondary">Rate</label><input className="input w-full" type="number" step="any" value={form.rate} onChange={(e) => setForm({ ...form, rate: e.target.value })} /></div>
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
        <div className="card"><div className="card-body text-center py-8"><p className="text-secondary">No convert orders found.</p></div></div>
      ) : (
        <div className="card"><div className="card-body overflow-x-auto">
          <table className="table"><thead><tr><th>User</th><th>From</th><th>To</th><th>From Amount</th><th>To Amount</th><th>Rate</th><th>Status</th><th>Actions</th></tr></thead>
            <tbody>
              {items.map((o) => (
                <tr key={o.id}>
                  <td className="text-secondary">{o.user_id}</td>
                  <td className="text-primary">{o.from_token}</td>
                  <td className="text-primary">{o.to_token}</td>
                  <td className="text-secondary">{o.from_amount}</td>
                  <td className="text-secondary">{o.to_amount}</td>
                  <td className="text-secondary">{o.rate}</td>
                  <td><span className={`badge ${o.status === 'completed' ? 'badge-success' : o.status === 'failed' ? 'badge-error' : 'badge-warning'}`}>{o.status}</span></td>
                  <td><div className="flex gap-2" style={{ flexWrap: 'wrap' }}>
                    <select className="input" value={o.status || ''} disabled={actionLoading} onChange={(e) => run(() => superAdminApi.updateConvertStatus(o.id, e.target.value))}>
                      {STATUSES.map((s) => <option key={s} value={s}>{s}</option>)}
                    </select>
                    <button className="btn btn-danger" disabled={actionLoading} onClick={() => { if (confirm('Delete this order?')) run(() => superAdminApi.deleteConvert(o.id)); }}>Delete</button>
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
