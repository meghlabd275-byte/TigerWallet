/**
 * TigerWallet Super Admin - API Keys Page
 */

import React, { useState, useEffect } from 'react';
import { superAdminApi } from '../services/api';

export default function APIKeys() {
  const [items, setItems] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({ name: '', user_id: '', permissions: '', rate_limit_per_minute: '', rate_limit_per_day: '', expires_at: '' });

  const load = async () => {
    try {
      setLoading(true);
      setError(null);
      const res: any = await superAdminApi.getAPIKeys();
      setItems(res.data || res.items || res || []);
    } catch (err: any) {
      setError(err?.message || 'Failed to load API keys');
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
      await superAdminApi.createAPIKey({
        name: form.name,
        user_id: form.user_id,
        permissions: form.permissions.split(',').map((s) => s.trim()).filter(Boolean),
        rate_limit_per_minute: form.rate_limit_per_minute ? Number(form.rate_limit_per_minute) : undefined,
        rate_limit_per_day: form.rate_limit_per_day ? Number(form.rate_limit_per_day) : undefined,
        expires_at: form.expires_at || undefined,
      });
      setShowForm(false);
      setForm({ name: '', user_id: '', permissions: '', rate_limit_per_minute: '', rate_limit_per_day: '', expires_at: '' });
      load();
    } catch (err: any) {
      alert(err?.message || 'Failed to create API key');
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
      <h1 className="text-2xl font-bold text-primary mb-6">API Keys</h1>

      <div className="flex justify-between items-center mb-4">
        <button className="btn btn-primary" onClick={() => setShowForm((s) => !s)}>
          {showForm ? 'Cancel' : 'New API Key'}
        </button>
      </div>

      {showForm && (
        <div className="card mb-6"><div className="card-body">
          <h3 className="text-primary mb-4">Create API Key</h3>
          <form onSubmit={handleCreate} className="flex flex-col gap-3">
            <div className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">Name</label><input className="input w-full" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} required /></div>
              <div className="form-group flex-1"><label className="text-secondary">User ID</label><input className="input w-full" value={form.user_id} onChange={(e) => setForm({ ...form, user_id: e.target.value })} required /></div>
            </div>
            <div className="form-group"><label className="text-secondary">Permissions (comma-separated)</label><input className="input w-full" value={form.permissions} onChange={(e) => setForm({ ...form, permissions: e.target.value })} required /></div>
            <div className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">Rate Limit /min</label><input className="input w-full" type="number" value={form.rate_limit_per_minute} onChange={(e) => setForm({ ...form, rate_limit_per_minute: e.target.value })} /></div>
              <div className="form-group flex-1"><label className="text-secondary">Rate Limit /day</label><input className="input w-full" type="number" value={form.rate_limit_per_day} onChange={(e) => setForm({ ...form, rate_limit_per_day: e.target.value })} /></div>
              <div className="form-group flex-1"><label className="text-secondary">Expires At</label><input className="input w-full" type="datetime-local" value={form.expires_at} onChange={(e) => setForm({ ...form, expires_at: e.target.value })} /></div>
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
        <div className="card"><div className="card-body text-center py-8"><p className="text-secondary">No API keys found.</p></div></div>
      ) : (
        <div className="card"><div className="card-body overflow-x-auto">
          <table className="table"><thead><tr><th>Name</th><th>Key</th><th>User</th><th>Permissions</th><th>Active</th><th>Actions</th></tr></thead>
            <tbody>
              {items.map((k) => (
                <tr key={k.id}>
                  <td className="text-primary">{k.name}</td>
                  <td className="text-secondary">{k.key ? `${k.key.slice(0, 8)}...` : '-'}</td>
                  <td className="text-secondary">{k.user_id}</td>
                  <td className="text-secondary">{(k.permissions || []).join(', ')}</td>
                  <td>{k.is_active ? <span className="badge badge-success">active</span> : <span className="badge badge-neutral">revoked</span>}</td>
                  <td><div className="flex gap-2">
                    <button className="btn btn-secondary" disabled={actionLoading} onClick={() => run(async () => { const r = await superAdminApi.regenerateAPIKey(k.id); alert(`New key: ${r.key}`); })}>Regenerate</button>
                    <button className="btn btn-danger" disabled={actionLoading} onClick={() => { if (confirm('Revoke this API key?')) run(() => superAdminApi.revokeAPIKey(k.id)); }}>Revoke</button>
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
