/**
 * TigerWallet Super Admin - Admins Page
 */

import React, { useState, useEffect } from 'react';
import { superAdminApi } from '../services/api';

export default function Admins() {
  const [items, setItems] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({ username: '', email: '', password: '', role: 'support', permissions: '' });
  const [permId, setPermId] = useState<string | null>(null);
  const [perms, setPerms] = useState('');

  const load = async () => {
    try {
      setLoading(true);
      setError(null);
      const res: any = await superAdminApi.getAdmins();
      setItems(res.data || res.items || res || []);
    } catch (err: any) {
      setError(err?.message || 'Failed to load admins');
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
      await superAdminApi.createAdmin({
        username: form.username,
        email: form.email,
        password: form.password,
        role: form.role,
        permissions: form.permissions ? form.permissions.split(',').map((p) => p.trim()).filter(Boolean) : undefined,
      });
      setShowForm(false);
      setForm({ username: '', email: '', password: '', role: 'support', permissions: '' });
      load();
    } catch (err: any) {
      alert(err?.message || 'Failed to create admin');
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

  const handleUpdatePerms = async () => {
    if (!permId) return;
    const list = perms.split(',').map((p) => p.trim()).filter(Boolean);
    await run(() => superAdminApi.updateAdminPermissions(permId, list));
    setPermId(null);
    setPerms('');
  };

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold text-primary mb-6">Admins</h1>

      <div className="flex justify-between items-center mb-4">
        <button className="btn btn-primary" onClick={() => setShowForm((s) => !s)}>
          {showForm ? 'Cancel' : 'New Admin'}
        </button>
      </div>

      {showForm && (
        <div className="card mb-6"><div className="card-body">
          <h3 className="text-primary mb-4">Create Admin</h3>
          <form onSubmit={handleCreate} className="flex flex-col gap-3">
            <div className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">Username</label><input className="input w-full" value={form.username} onChange={(e) => setForm({ ...form, username: e.target.value })} required /></div>
              <div className="form-group flex-1"><label className="text-secondary">Email</label><input className="input w-full" value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} required /></div>
              <div className="form-group flex-1"><label className="text-secondary">Password</label><input className="input w-full" type="password" value={form.password} onChange={(e) => setForm({ ...form, password: e.target.value })} required /></div>
              <div className="form-group flex-1"><label className="text-secondary">Role</label><select className="input w-full" value={form.role} onChange={(e) => setForm({ ...form, role: e.target.value })}><option>support</option><option>manager</option><option>admin</option><option>auditor</option><option>super_admin</option></select></div>
            </div>
            <div className="form-group"><label className="text-secondary">Permissions (comma-separated)</label><input className="input w-full" value={form.permissions} onChange={(e) => setForm({ ...form, permissions: e.target.value })} placeholder="users:read, kyc:write" /></div>
            <button className="btn btn-primary" disabled={actionLoading} type="submit">Create</button>
          </form>
        </div></div>
      )}

      {error ? (
        <div className="alert alert-error mb-4"><p className="text-error">{error}</p><button className="btn btn-secondary mt-2" onClick={load}>Retry</button></div>
      ) : loading ? (
        <div className="flex items-center justify-center p-8"><div className="loader"></div></div>
      ) : items.length === 0 ? (
        <div className="card"><div className="card-body text-center py-8"><p className="text-secondary">No admins found.</p></div></div>
      ) : (
        <div className="card"><div className="card-body overflow-x-auto">
          <table className="table"><thead><tr><th>Username</th><th>Email</th><th>Role</th><th>Status</th><th>2FA</th><th>Permissions</th><th>Actions</th></tr></thead>
            <tbody>
              {items.map((a) => (
                <tr key={a.id}>
                  <td className="text-primary">{a.username}</td>
                  <td className="text-secondary">{a.email}</td>
                  <td className="text-secondary">{a.role}</td>
                  <td><span className={`badge ${a.status === 'active' ? 'badge-success' : 'badge-warning'}`}>{a.status}</span></td>
                  <td>{a.two_factor_enabled ? <span className="badge badge-success">on</span> : <span className="badge badge-neutral">off</span>}</td>
                  <td className="text-secondary">{(a.permissions || []).length}</td>
                  <td><div className="flex gap-2" style={{ flexWrap: 'wrap' }}>
                    {a.status === 'active'
                      ? <button className="btn btn-secondary" disabled={actionLoading} onClick={() => run(() => superAdminApi.suspendAdmin(a.id, 'Admin action'))}>Suspend</button>
                      : <button className="btn btn-primary" disabled={actionLoading} onClick={() => run(() => superAdminApi.activateAdmin(a.id))}>Activate</button>}
                    <button className="btn btn-secondary" disabled={actionLoading} onClick={() => { setPermId(a.id); setPerms((a.permissions || []).join(', ')); }}>Permissions</button>
                    <button className="btn btn-danger" disabled={actionLoading} onClick={() => { if (confirm('Delete this admin?')) run(() => superAdminApi.deleteAdmin(a.id)); }}>Delete</button>
                  </div></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div></div>
      )}

      {permId && (
        <div className="card mt-4"><div className="card-body">
          <h3 className="text-primary mb-2">Update Permissions</h3>
          <div className="form-group"><label className="text-secondary">Permissions (comma-separated)</label><input className="input w-full" value={perms} onChange={(e) => setPerms(e.target.value)} /></div>
          <div className="flex gap-2 mt-2"><button className="btn btn-primary" disabled={actionLoading} onClick={handleUpdatePerms}>Save</button><button className="btn btn-secondary" onClick={() => setPermId(null)}>Cancel</button></div>
        </div></div>
      )}
    </div>
  );
}
