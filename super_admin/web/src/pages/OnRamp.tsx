/**
 * TigerWallet Super Admin - OnRamp Orders Page
 * Governance records only — never moves crypto assets.
 */

import React, { useState, useEffect } from 'react';
import { superAdminApi } from '../services/api';

export default function OnRamp() {
  const [items, setItems] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [showForm, setShowForm] = useState(false);
  const [rejectId, setRejectId] = useState<string | null>(null);
  const [reason, setReason] = useState('');
  const [form, setForm] = useState({
    user_id: '', provider: '', fiat_currency: '', crypto_token: '', fiat_amount: '', crypto_amount: '',
  });

  const load = async () => {
    try {
      setLoading(true);
      setError(null);
      const res: any = await superAdminApi.getOnRampOrders();
      setItems(res.orders || res.data || res || []);
    } catch (err: any) {
      setError(err?.message || 'Failed to load onramp orders');
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
      await superAdminApi.createOnRampOrder({
        user_id: form.user_id || undefined,
        provider: form.provider,
        fiat_currency: form.fiat_currency,
        crypto_token: form.crypto_token,
        fiat_amount: form.fiat_amount ? Number(form.fiat_amount) : undefined,
        crypto_amount: form.crypto_amount ? Number(form.crypto_amount) : undefined,
      });
      setShowForm(false);
      setForm({ user_id: '', provider: '', fiat_currency: '', crypto_token: '', fiat_amount: '', crypto_amount: '' });
      load();
    } catch (err: any) {
      alert(err?.message || 'Failed to create onramp order');
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

  const handleReject = async () => {
    if (!rejectId || !reason) return;
    await run(() => superAdminApi.rejectOnRampOrder(rejectId, reason));
    setRejectId(null);
    setReason('');
  };

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold text-primary mb-6">OnRamp Orders</h1>

      <div className="flex justify-between items-center mb-4">
        <button className="btn btn-primary" onClick={() => setShowForm((s) => !s)}>
          {showForm ? 'Cancel' : 'New Order'}
        </button>
      </div>

      {showForm && (
        <div className="card mb-6"><div className="card-body">
          <h3 className="text-primary mb-4">Create OnRamp Order</h3>
          <form onSubmit={handleCreate} className="flex flex-col gap-3">
            <div className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">User ID</label><input className="input w-full" value={form.user_id} onChange={(e) => setForm({ ...form, user_id: e.target.value })} /></div>
              <div className="form-group flex-1"><label className="text-secondary">Provider</label><input className="input w-full" value={form.provider} onChange={(e) => setForm({ ...form, provider: e.target.value })} required /></div>
              <div className="form-group flex-1"><label className="text-secondary">Fiat Currency</label><input className="input w-full" value={form.fiat_currency} onChange={(e) => setForm({ ...form, fiat_currency: e.target.value })} required /></div>
              <div className="form-group flex-1"><label className="text-secondary">Crypto Token</label><input className="input w-full" value={form.crypto_token} onChange={(e) => setForm({ ...form, crypto_token: e.target.value })} required /></div>
            </div>
            <div className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">Fiat Amount</label><input className="input w-full" type="number" step="any" value={form.fiat_amount} onChange={(e) => setForm({ ...form, fiat_amount: e.target.value })} /></div>
              <div className="form-group flex-1"><label className="text-secondary">Crypto Amount</label><input className="input w-full" type="number" step="any" value={form.crypto_amount} onChange={(e) => setForm({ ...form, crypto_amount: e.target.value })} /></div>
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
        <div className="card"><div className="card-body text-center py-8"><p className="text-secondary">No onramp orders found.</p></div></div>
      ) : (
        <div className="card"><div className="card-body overflow-x-auto">
          <table className="table"><thead><tr><th>User</th><th>Provider</th><th>Fiat</th><th>Crypto</th><th>Fiat Amount</th><th>Crypto Amount</th><th>Status</th><th>Actions</th></tr></thead>
            <tbody>
              {items.map((o) => (
                <tr key={o.id}>
                  <td className="text-secondary">{o.user_id}</td>
                  <td className="text-primary">{o.provider}</td>
                  <td className="text-secondary">{o.fiat_currency}</td>
                  <td className="text-secondary">{o.crypto_token}</td>
                  <td className="text-secondary">{o.fiat_amount}</td>
                  <td className="text-secondary">{o.crypto_amount}</td>
                  <td><span className={`badge ${o.status === 'completed' ? 'badge-success' : o.status === 'rejected' ? 'badge-error' : 'badge-warning'}`}>{o.status}</span></td>
                  <td><div className="flex gap-2">
                    {o.status === 'pending' && <>
                      <button className="btn btn-primary" disabled={actionLoading} onClick={() => run(() => superAdminApi.approveOnRampOrder(o.id))}>Approve</button>
                      <button className="btn btn-danger" disabled={actionLoading} onClick={() => { setRejectId(o.id); setReason(''); }}>Reject</button>
                    </>}
                    <button className="btn btn-secondary" disabled={actionLoading} onClick={() => { if (confirm('Delete this order?')) run(() => superAdminApi.deleteOnRampOrder(o.id)); }}>Delete</button>
                  </div></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div></div>
      )}

      {rejectId && (
        <div className="card mt-4"><div className="card-body">
          <h3 className="text-primary mb-2">Reject OnRamp Order</h3>
          <div className="form-group"><label className="text-secondary">Reason</label><input className="input w-full" value={reason} onChange={(e) => setReason(e.target.value)} placeholder="Rejection reason" /></div>
          <div className="flex gap-2 mt-2">
            <button className="btn btn-danger" disabled={actionLoading || !reason} onClick={handleReject}>Confirm Reject</button>
            <button className="btn btn-secondary" onClick={() => setRejectId(null)}>Cancel</button>
          </div>
        </div></div>
      )}
    </div>
  );
}
