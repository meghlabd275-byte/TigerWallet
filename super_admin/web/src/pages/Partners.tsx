/**
 * TigerWallet Super Admin - Partners Page
 * Governance records only — never moves crypto assets.
 */

import React, { useState, useEffect } from 'react';
import { superAdminApi } from '../services/api';

const STATUSES = ['pending', 'approved', 'rejected', 'suspended', 'active', 'paused'];

export default function Partners() {
  const [items, setItems] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [showForm, setShowForm] = useState(false);
  const [rejectId, setRejectId] = useState<string | null>(null);
  const [reason, setReason] = useState('');
  const [form, setForm] = useState({ name: '', contact_email: '', revenue_share: '' });

  const load = async () => {
    try {
      setLoading(true);
      setError(null);
      const res: any = await superAdminApi.getPartners();
      setItems(res.partners || res.data || res || []);
    } catch (err: any) {
      setError(err?.message || 'Failed to load partners');
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
      await superAdminApi.createPartner({
        name: form.name,
        contact_email: form.contact_email || undefined,
        revenue_share: form.revenue_share ? Number(form.revenue_share) : undefined,
      });
      setShowForm(false);
      setForm({ name: '', contact_email: '', revenue_share: '' });
      load();
    } catch (err: any) {
      alert(err?.message || 'Failed to create partner');
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
    await run(() => superAdminApi.rejectPartner(rejectId, reason));
    setRejectId(null);
    setReason('');
  };

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold text-primary mb-6">Partners</h1>

      <div className="flex justify-between items-center mb-4">
        <button className="btn btn-primary" onClick={() => setShowForm((s) => !s)}>
          {showForm ? 'Cancel' : 'New Partner'}
        </button>
      </div>

      {showForm && (
        <div className="card mb-6"><div className="card-body">
          <h3 className="text-primary mb-4">Create Partner</h3>
          <form onSubmit={handleCreate} className="flex flex-col gap-3">
            <div className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">Name</label><input className="input w-full" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} required /></div>
              <div className="form-group flex-1"><label className="text-secondary">Contact Email</label><input className="input w-full" value={form.contact_email} onChange={(e) => setForm({ ...form, contact_email: e.target.value })} /></div>
              <div className="form-group flex-1"><label className="text-secondary">Revenue Share (%)</label><input className="input w-full" type="number" step="any" value={form.revenue_share} onChange={(e) => setForm({ ...form, revenue_share: e.target.value })} /></div>
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
        <div className="card"><div className="card-body text-center py-8"><p className="text-secondary">No partners found.</p></div></div>
      ) : (
        <div className="card"><div className="card-body overflow-x-auto">
          <table className="table"><thead><tr><th>Name</th><th>Contact</th><th>Revenue Share</th><th>Status</th><th>Actions</th></tr></thead>
            <tbody>
              {items.map((p) => (
                <tr key={p.id}>
                  <td className="text-primary">{p.name}</td>
                  <td className="text-secondary">{p.contact_email}</td>
                  <td className="text-secondary">{p.revenue_share}</td>
                  <td><span className={`badge ${p.status === 'approved' ? 'badge-success' : p.status === 'rejected' ? 'badge-error' : 'badge-warning'}`}>{p.status}</span></td>
                  <td><div className="flex gap-2" style={{ flexWrap: 'wrap' }}>
                    {p.status === 'pending' && <>
                      <button className="btn btn-primary" disabled={actionLoading} onClick={() => run(() => superAdminApi.approvePartner(p.id))}>Approve</button>
                      <button className="btn btn-danger" disabled={actionLoading} onClick={() => { setRejectId(p.id); setReason(''); }}>Reject</button>
                    </>}
                    <select className="input" value={p.status || ''} disabled={actionLoading} onChange={(e) => run(() => superAdminApi.updatePartnerStatus(p.id, e.target.value))}>
                      {STATUSES.map((s) => <option key={s} value={s}>{s}</option>)}
                    </select>
                    <button className="btn btn-secondary" disabled={actionLoading} onClick={() => { if (confirm('Delete this partner?')) run(() => superAdminApi.deletePartner(p.id)); }}>Delete</button>
                  </div></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div></div>
      )}

      {rejectId && (
        <div className="card mt-4"><div className="card-body">
          <h3 className="text-primary mb-2">Reject Partner</h3>
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
