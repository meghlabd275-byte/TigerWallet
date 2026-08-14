/**
 * TigerWallet Super Admin - Webhooks Page
 */

import React, { useState, useEffect } from 'react';
import { superAdminApi } from '../services/api';

export default function Webhooks() {
  const [items, setItems] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({ name: '', url: '', events: '' });

  const load = async () => {
    try {
      setLoading(true);
      setError(null);
      const res: any = await superAdminApi.getWebhooks();
      setItems(res.data || res.items || res || []);
    } catch (err: any) {
      setError(err?.message || 'Failed to load webhooks');
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
      await superAdminApi.createWebhook({
        name: form.name,
        url: form.url,
        events: form.events.split(',').map((s) => s.trim()).filter(Boolean),
      });
      setShowForm(false);
      setForm({ name: '', url: '', events: '' });
      load();
    } catch (err: any) {
      alert(err?.message || 'Failed to create webhook');
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
      <h1 className="text-2xl font-bold text-primary mb-6">Webhooks</h1>

      <div className="flex justify-between items-center mb-4">
        <button className="btn btn-primary" onClick={() => setShowForm((s) => !s)}>
          {showForm ? 'Cancel' : 'New Webhook'}
        </button>
      </div>

      {showForm && (
        <div className="card mb-6"><div className="card-body">
          <h3 className="text-primary mb-4">Create Webhook</h3>
          <form onSubmit={handleCreate} className="flex flex-col gap-3">
            <div className="form-group"><label className="text-secondary">Name</label><input className="input w-full" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} required /></div>
            <div className="form-group"><label className="text-secondary">URL</label><input className="input w-full" value={form.url} onChange={(e) => setForm({ ...form, url: e.target.value })} required /></div>
            <div className="form-group"><label className="text-secondary">Events (comma-separated)</label><input className="input w-full" value={form.events} onChange={(e) => setForm({ ...form, events: e.target.value })} placeholder="deposit.created, withdrawal.approved" required /></div>
            <button className="btn btn-primary" disabled={actionLoading} type="submit">Create</button>
          </form>
        </div></div>
      )}

      {error ? (
        <div className="alert alert-error mb-4"><p className="text-error">{error}</p><button className="btn btn-secondary mt-2" onClick={load}>Retry</button></div>
      ) : loading ? (
        <div className="flex items-center justify-center p-8"><div className="loader"></div></div>
      ) : items.length === 0 ? (
        <div className="card"><div className="card-body text-center py-8"><p className="text-secondary">No webhooks found.</p></div></div>
      ) : (
        <div className="card"><div className="card-body overflow-x-auto">
          <table className="table"><thead><tr><th>Name</th><th>URL</th><th>Events</th><th>Active</th><th>Actions</th></tr></thead>
            <tbody>
              {items.map((w) => (
                <tr key={w.id}>
                  <td className="text-primary">{w.name}</td>
                  <td className="text-secondary">{w.url}</td>
                  <td className="text-secondary">{(w.events || []).join(', ')}</td>
                  <td>{w.is_active ? <span className="badge badge-success">active</span> : <span className="badge badge-neutral">inactive</span>}</td>
                  <td><div className="flex gap-2">
                    <button className="btn btn-secondary" disabled={actionLoading} onClick={() => run(async () => { const r = await superAdminApi.testWebhook(w.id); alert(`Test: ${r.success ? 'OK' : 'Failed'} (${r.response_time_ms}ms)`); })}>Test</button>
                    {w.is_active
                      ? <button className="btn btn-secondary" disabled={actionLoading} onClick={() => run(() => superAdminApi.deactivateWebhook(w.id))}>Deactivate</button>
                      : <button className="btn btn-primary" disabled={actionLoading} onClick={() => run(() => superAdminApi.activateWebhook(w.id))}>Activate</button>}
                    <button className="btn btn-danger" disabled={actionLoading} onClick={() => { if (confirm('Delete this webhook?')) run(() => superAdminApi.deleteWebhook(w.id)); }}>Delete</button>
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
