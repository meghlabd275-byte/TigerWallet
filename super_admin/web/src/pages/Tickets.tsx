/**
 * TigerWallet Super Admin - Tickets Page
 */

import React, { useState, useEffect } from 'react';
import { superAdminApi } from '../services/api';

export default function Tickets() {
  const [items, setItems] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({ title: '', description: '', category: 'general', priority: 'medium' });
  const [selected, setSelected] = useState<any | null>(null);
  const [assignId, setAssignId] = useState('');
  const [message, setMessage] = useState('');

  const load = async () => {
    try {
      setLoading(true);
      setError(null);
      const res: any = await superAdminApi.getTickets();
      setItems(res.data || res.items || res || []);
    } catch (err: any) {
      setError(err?.message || 'Failed to load tickets');
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
      await superAdminApi.createTicket({
        title: form.title,
        description: form.description,
        category: form.category,
        priority: form.priority,
      });
      setShowForm(false);
      setForm({ title: '', description: '', category: 'general', priority: 'medium' });
      load();
    } catch (err: any) {
      alert(err?.message || 'Failed to create ticket');
    } finally {
      setActionLoading(false);
    }
  };

  const openTicket = async (id: string) => {
    setActionLoading(true);
    try {
      const t = await superAdminApi.getTicket(id);
      setSelected(t);
    } catch (err: any) {
      alert(err?.message || 'Failed to load ticket');
    } finally {
      setActionLoading(false);
    }
  };

  const run = async (fn: () => Promise<any>, reloadList = true) => {
    setActionLoading(true);
    try {
      const t = await fn();
      if (reloadList) load();
      if (selected) setSelected(t || selected);
    } catch (err: any) {
      alert(err?.message || 'Action failed');
    } finally {
      setActionLoading(false);
    }
  };

  const handleAssign = async () => {
    if (!selected || !assignId) return;
    await run(() => superAdminApi.assignTicket(selected.id, assignId));
    setAssignId('');
  };

  const handleAddMessage = async () => {
    if (!selected || !message) return;
    await run(() => superAdminApi.addTicketMessage(selected.id, { message }));
    setMessage('');
  };

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold text-primary mb-6">Tickets</h1>

      <div className="flex justify-between items-center mb-4">
        <button className="btn btn-primary" onClick={() => setShowForm((s) => !s)}>
          {showForm ? 'Cancel' : 'New Ticket'}
        </button>
      </div>

      {showForm && (
        <div className="card mb-6"><div className="card-body">
          <h3 className="text-primary mb-4">Create Ticket</h3>
          <form onSubmit={handleCreate} className="flex flex-col gap-3">
            <div className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">Title</label><input className="input w-full" value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} required /></div>
              <div className="form-group flex-1"><label className="text-secondary">Category</label><input className="input w-full" value={form.category} onChange={(e) => setForm({ ...form, category: e.target.value })} required /></div>
              <div className="form-group flex-1"><label className="text-secondary">Priority</label><select className="input w-full" value={form.priority} onChange={(e) => setForm({ ...form, priority: e.target.value })}><option>low</option><option>medium</option><option>high</option><option>urgent</option></select></div>
            </div>
            <div className="form-group"><label className="text-secondary">Description</label><textarea className="input w-full" rows={3} value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} required /></div>
            <button className="btn btn-primary" disabled={actionLoading} type="submit">Create</button>
          </form>
        </div></div>
      )}

      {error ? (
        <div className="alert alert-error mb-4"><p className="text-error">{error}</p><button className="btn btn-secondary mt-2" onClick={load}>Retry</button></div>
      ) : loading ? (
        <div className="flex items-center justify-center p-8"><div className="loader"></div></div>
      ) : items.length === 0 ? (
        <div className="card"><div className="card-body text-center py-8"><p className="text-secondary">No tickets found.</p></div></div>
      ) : (
        <div className="card"><div className="card-body overflow-x-auto">
          <table className="table"><thead><tr><th>Title</th><th>Category</th><th>Priority</th><th>Status</th><th>Assigned</th><th>Actions</th></tr></thead>
            <tbody>
              {items.map((t) => (
                <tr key={t.id}>
                  <td className="text-primary">{t.title}</td>
                  <td className="text-secondary">{t.category}</td>
                  <td><span className={`badge ${t.priority === 'urgent' || t.priority === 'high' ? 'badge-warning' : 'badge-neutral'}`}>{t.priority}</span></td>
                  <td><span className={`badge ${t.status === 'closed' ? 'badge-neutral' : 'badge-info'}`}>{t.status}</span></td>
                  <td className="text-secondary">{t.assigned_to || '-'}</td>
                  <td><div className="flex gap-2">
                    <button className="btn btn-secondary" disabled={actionLoading} onClick={() => openTicket(t.id)}>Open</button>
                    {t.status !== 'closed' && <button className="btn btn-danger" disabled={actionLoading} onClick={() => run(() => superAdminApi.closeTicket(t.id))}>Close</button>}
                  </div></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div></div>
      )}

      {selected && (
        <div className="card mt-4"><div className="card-body">
          <h3 className="text-primary mb-2">{selected.title}</h3>
          <p className="text-secondary mb-2">{selected.description}</p>
          <p className="text-tertiary mb-4">Status: {selected.status} | Priority: {selected.priority} | Assigned: {selected.assigned_to || '-'}</p>
          <div className="flex gap-3 mb-4">
            <div className="form-group flex-1"><label className="text-secondary">Assign to (admin ID)</label><input className="input w-full" value={assignId} onChange={(e) => setAssignId(e.target.value)} /></div>
            <div className="form-group" style={{ alignSelf: 'flex-end' }}><button className="btn btn-primary" disabled={actionLoading || !assignId} onClick={handleAssign}>Assign</button></div>
          </div>
          <div className="flex gap-3 mb-2">
            <div className="form-group flex-1"><label className="text-secondary">Add Message</label><input className="input w-full" value={message} onChange={(e) => setMessage(e.target.value)} /></div>
            <div className="form-group" style={{ alignSelf: 'flex-end' }}><button className="btn btn-secondary" disabled={actionLoading || !message} onClick={handleAddMessage}>Send</button></div>
          </div>
          <button className="btn btn-ghost mt-2" onClick={() => setSelected(null)}>Close View</button>
        </div></div>
      )}
    </div>
  );
}
