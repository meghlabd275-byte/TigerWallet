/**
 * TigerWallet Super Admin - P2P Merchants Page
 * Governance records only — never moves crypto assets.
 */

import React, { useState, useEffect } from 'react';
import { superAdminApi } from '../services/api';

const STATUSES = ['pending', 'approved', 'rejected', 'suspended', 'active', 'paused'];

export default function P2PMerchants() {
  const [items, setItems] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [showForm, setShowForm] = useState(false);
  const [rejectId, setRejectId] = useState<string | null>(null);
  const [reason, setReason] = useState('');
  const [form, setForm] = useState({ name: '', email: '' });

  const load = async () => {
    try {
      setLoading(true);
      setError(null);
      const res: any = await superAdminApi.getP2PMerchants();
      setItems(res.merchants || res.data || res || []);
    } catch (err: any) {
      setError(err?.message || 'Failed to load P2P merchants');
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
      await superAdminApi.createP2PMerchant({ name: form.name, email: form.email || undefined });
      setShowForm(false);
      setForm({ name: '', email: '' });
      load();
    } catch (err: any) {
      alert(err?.message || 'Failed to create P2P merchant');
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
    await run(() => superAdminApi.rejectP2PMerchant(rejectId, reason));
    setRejectId(null);
    setReason('');
  };

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold text-primary mb-6">P2P Merchants</h1>

      <div className="flex justify-between items-center mb-4">
        <button className="btn btn-primary" onClick={() => setShowForm((s) => !s)}>
          {showForm ? 'Cancel' : 'New Merchant'}
        </button>
      </div>

      {showForm && (
        <div className="card mb-6"><div className="card-body">
          <h3 className="text-primary mb-4">Create P2P Merchant</h3>
          <form onSubmit={handleCreate} className="flex flex-col gap-3">
            <div className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">Name</label><input className="input w-full" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} required /></div>
              <div className="form-group flex-1"><label className="text-secondary">Email</label><input className="input w-full" value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} /></div>
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
        <div className="card"><div className="card-body text-center py-8"><p className="text-secondary">No P2P merchants found.</p></div></div>
      ) : (
        <div className="card"><div className="card-body overflow-x-auto">
          <table className="table"><thead><tr><th>Name</th><th>Email</th><th>Verified</th><th>Status</th><th>Actions</th></tr></thead>
            <tbody>
              {items.map((m) => (
                <tr key={m.id}>
                  <td className="text-primary">{m.name}</td>
                  <td className="text-secondary">{m.email}</td>
                  <td className="text-secondary">{m.verified ? 'yes' : 'no'}</td>
                  <td><span className={`badge ${m.status === 'approved' ? 'badge-success' : m.status === 'rejected' ? 'badge-error' : 'badge-warning'}`}>{m.status}</span></td>
                  <td><div className="flex gap-2" style={{ flexWrap: 'wrap' }}>
                    {m.status === 'pending' && <>
                      <button className="btn btn-primary" disabled={actionLoading} onClick={() => run(() => superAdminApi.approveP2PMerchant(m.id))}>Approve</button>
                      <button className="btn btn-danger" disabled={actionLoading} onClick={() => { setRejectId(m.id); setReason(''); }}>Reject</button>
                    </>}
                    <select className="input" value={m.status || ''} disabled={actionLoading} onChange={(e) => run(() => superAdminApi.updateP2PMerchantStatus(m.id, e.target.value))}>
                      {STATUSES.map((s) => <option key={s} value={s}>{s}</option>)}
                    </select>
                    <button className="btn btn-secondary" disabled={actionLoading} onClick={() => { if (confirm('Delete this merchant?')) run(() => superAdminApi.deleteP2PMerchant(m.id)); }}>Delete</button>
                  </div></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div></div>
      )}

      {rejectId && (
        <div className="card mt-4"><div className="card-body">
          <h3 className="text-primary mb-2">Reject P2P Merchant</h3>
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
