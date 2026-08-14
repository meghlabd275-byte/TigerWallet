/**
 * TigerWallet Super Admin - BotsClients Page
 */

import React, { useState, useEffect } from 'react';
import { superAdminApi } from '../services/api';

export default function BotsClients() {
  const [items, setItems] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({ name: '', company: '', email: '', permission_level: 'standard' });
  const [editId, setEditId] = useState<string | null>(null);
  const [editName, setEditName] = useState('');

  const load = async () => {
    try {
      setLoading(true);
      setError(null);
      const res: any = await superAdminApi.getBotsClients();
      setItems(res.clients || res || []);
    } catch (err: any) {
      setError(err?.message || 'Failed to load bots clients');
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
      await superAdminApi.createBotsClient({
        name: form.name,
        company: form.company || undefined,
        email: form.email || undefined,
        permission_level: form.permission_level || undefined,
      });
      setShowForm(false);
      setForm({ name: '', company: '', email: '', permission_level: 'standard' });
      load();
    } catch (err: any) {
      alert(err?.message || 'Failed to create bots client');
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

  const handleSaveEdit = async () => {
    if (!editId || !editName) return;
    await run(() => superAdminApi.updateBotsClient(editId, { name: editName }));
    setEditId(null);
    setEditName('');
  };

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold text-primary mb-6">BotsClients</h1>

      <div className="flex justify-between items-center mb-4">
        <button className="btn btn-primary" onClick={() => setShowForm((s) => !s)}>
          {showForm ? 'Cancel' : 'New Client'}
        </button>
      </div>

      {showForm && (
        <div className="card mb-6"><div className="card-body">
          <h3 className="text-primary mb-4">Create Bots Client</h3>
          <form onSubmit={handleCreate} className="flex flex-col gap-3">
            <div className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">Name</label><input className="input w-full" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} required /></div>
              <div className="form-group flex-1"><label className="text-secondary">Company</label><input className="input w-full" value={form.company} onChange={(e) => setForm({ ...form, company: e.target.value })} /></div>
            </div>
            <div className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">Email</label><input className="input w-full" value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} /></div>
              <div className="form-group flex-1"><label className="text-secondary">Permission Level</label><select className="input w-full" value={form.permission_level} onChange={(e) => setForm({ ...form, permission_level: e.target.value })}><option>standard</option><option>elevated</option><option>admin</option></select></div>
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
        <div className="card"><div className="card-body text-center py-8"><p className="text-secondary">No bots clients found.</p></div></div>
      ) : (
        <div className="card"><div className="card-body overflow-x-auto">
          <table className="table"><thead><tr><th>Name</th><th>Company</th><th>Email</th><th>Permission Level</th><th>Status</th><th>Actions</th></tr></thead>
            <tbody>
              {items.map((c) => (
                <tr key={c.id}>
                  <td className="text-primary">{c.name}</td>
                  <td className="text-secondary">{c.company || '-'}</td>
                  <td className="text-secondary">{c.email || '-'}</td>
                  <td className="text-secondary">{c.permission_level}</td>
                  <td><span className={`badge ${c.status === 'active' ? 'badge-success' : c.status === 'suspended' ? 'badge-warning' : 'badge-neutral'}`}>{c.status || 'inactive'}</span></td>
                  <td><div className="flex gap-2" style={{ flexWrap: 'wrap' }}>
                    <button className="btn btn-secondary" disabled={actionLoading} onClick={() => { setEditId(c.id); setEditName(c.name); }}>Edit</button>
                    {c.status !== 'active' && <button className="btn btn-primary" disabled={actionLoading} onClick={() => run(() => superAdminApi.updateBotsClientStatus(c.id, 'active'))}>Activate</button>}
                    {c.status === 'active' && <button className="btn btn-secondary" disabled={actionLoading} onClick={() => run(() => superAdminApi.updateBotsClientStatus(c.id, 'suspended'))}>Suspend</button>}
                    <button className="btn btn-danger" disabled={actionLoading} onClick={() => { if (confirm('Delete this client?')) run(() => superAdminApi.deleteBotsClient(c.id)); }}>Delete</button>
                  </div></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div></div>
      )}

      {editId && (
        <div className="card mt-4"><div className="card-body">
          <h3 className="text-primary mb-2">Edit Client Name</h3>
          <div className="form-group"><label className="text-secondary">Name</label><input className="input w-full" value={editName} onChange={(e) => setEditName(e.target.value)} /></div>
          <div className="flex gap-2 mt-2"><button className="btn btn-primary" disabled={actionLoading || !editName} onClick={handleSaveEdit}>Save</button><button className="btn btn-secondary" onClick={() => setEditId(null)}>Cancel</button></div>
        </div></div>
      )}
    </div>
  );
}
