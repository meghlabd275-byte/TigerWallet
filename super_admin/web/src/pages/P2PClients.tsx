/**
 * TigerWallet Super Admin - P2P Clients Page
 * Governance records only — never moves crypto assets.
 */

import React, { useState, useEffect } from 'react';
import { superAdminApi } from '../services/api';

const STATUSES = ['active', 'suspended', 'paused', 'halted'];

export default function P2PClients() {
  const [items, setItems] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({ user_id: '', username: '' });

  const load = async () => {
    try {
      setLoading(true);
      setError(null);
      const res: any = await superAdminApi.getP2PClients();
      setItems(res.clients || res.data || res || []);
    } catch (err: any) {
      setError(err?.message || 'Failed to load P2P clients');
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
      await superAdminApi.createP2PClient({ user_id: form.user_id || undefined, username: form.username });
      setShowForm(false);
      setForm({ user_id: '', username: '' });
      load();
    } catch (err: any) {
      alert(err?.message || 'Failed to create P2P client');
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
      <h1 className="text-2xl font-bold text-primary mb-6">P2P Clients</h1>

      <div className="flex justify-between items-center mb-4">
        <button className="btn btn-primary" onClick={() => setShowForm((s) => !s)}>
          {showForm ? 'Cancel' : 'New Client'}
        </button>
      </div>

      {showForm && (
        <div className="card mb-6"><div className="card-body">
          <h3 className="text-primary mb-4">Create P2P Client</h3>
          <form onSubmit={handleCreate} className="flex flex-col gap-3">
            <div className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">User ID</label><input className="input w-full" value={form.user_id} onChange={(e) => setForm({ ...form, user_id: e.target.value })} /></div>
              <div className="form-group flex-1"><label className="text-secondary">Username</label><input className="input w-full" value={form.username} onChange={(e) => setForm({ ...form, username: e.target.value })} required /></div>
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
        <div className="card"><div className="card-body text-center py-8"><p className="text-secondary">No P2P clients found.</p></div></div>
      ) : (
        <div className="card"><div className="card-body overflow-x-auto">
          <table className="table"><thead><tr><th>User ID</th><th>Username</th><th>Status</th><th>Created</th><th>Actions</th></tr></thead>
            <tbody>
              {items.map((c) => (
                <tr key={c.id}>
                  <td className="text-secondary">{c.user_id}</td>
                  <td className="text-primary">{c.username}</td>
                  <td><span className={`badge ${c.status === 'active' ? 'badge-success' : c.status === 'halted' ? 'badge-error' : 'badge-warning'}`}>{c.status}</span></td>
                  <td className="text-secondary">{c.created_at}</td>
                  <td><div className="flex gap-2" style={{ flexWrap: 'wrap' }}>
                    <select className="input" value={c.status || ''} disabled={actionLoading} onChange={(e) => run(() => superAdminApi.updateP2PClientStatus(c.id, e.target.value))}>
                      {STATUSES.map((s) => <option key={s} value={s}>{s}</option>)}
                    </select>
                    <button className="btn btn-danger" disabled={actionLoading} onClick={() => { if (confirm('Delete this client?')) run(() => superAdminApi.deleteP2PClient(c.id)); }}>Delete</button>
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
