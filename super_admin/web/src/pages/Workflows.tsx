/**
 * TigerWallet Super Admin - Approval Workflows Page
 */

import React, { useState, useEffect } from 'react';
import { superAdminApi } from '../services/api';

export default function Workflows() {
  const [items, setItems] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({ name: '', description: '', trigger_type: '', approvers: '', approval_threshold: '1', timeout_hours: '24' });
  const [actionId, setActionId] = useState<string | null>(null);
  const [actionComments, setActionComments] = useState('');

  const load = async () => {
    try {
      setLoading(true);
      setError(null);
      const res: any = await superAdminApi.getApprovalWorkflows();
      setItems(Array.isArray(res) ? res : (res.data || res.items || []));
    } catch (err: any) {
      setError(err?.message || 'Failed to load approval workflows');
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
      await superAdminApi.createApprovalWorkflow({
        name: form.name,
        description: form.description,
        trigger_type: form.trigger_type,
        approvers: form.approvers.split(',').map((s) => s.trim()).filter(Boolean),
        approval_threshold: Number(form.approval_threshold),
        timeout_hours: Number(form.timeout_hours),
      });
      setShowForm(false);
      setForm({ name: '', description: '', trigger_type: '', approvers: '', approval_threshold: '1', timeout_hours: '24' });
      load();
    } catch (err: any) {
      alert(err?.message || 'Failed to create workflow');
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

  const handleTakeAction = async (decision: 'approve' | 'reject') => {
    if (!actionId) return;
    await run(() => superAdminApi.takeApprovalAction(actionId, { action: decision, comments: actionComments || undefined }));
    setActionId(null);
    setActionComments('');
  };

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold text-primary mb-6">Approval Workflows</h1>

      <div className="flex justify-between items-center mb-4">
        <button className="btn btn-primary" onClick={() => setShowForm((s) => !s)}>
          {showForm ? 'Cancel' : 'New Workflow'}
        </button>
      </div>

      {showForm && (
        <div className="card mb-6"><div className="card-body">
          <h3 className="text-primary mb-4">Create Workflow</h3>
          <form onSubmit={handleCreate} className="flex flex-col gap-3">
            <div className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">Name</label><input className="input w-full" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} required /></div>
              <div className="form-group flex-1"><label className="text-secondary">Trigger Type</label><input className="input w-full" value={form.trigger_type} onChange={(e) => setForm({ ...form, trigger_type: e.target.value })} placeholder="e.g. withdrawal.large" required /></div>
            </div>
            <div className="form-group"><label className="text-secondary">Description</label><input className="input w-full" value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} /></div>
            <div className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">Approvers (comma-separated)</label><input className="input w-full" value={form.approvers} onChange={(e) => setForm({ ...form, approvers: e.target.value })} required /></div>
              <div className="form-group flex-1"><label className="text-secondary">Threshold</label><input className="input w-full" type="number" value={form.approval_threshold} onChange={(e) => setForm({ ...form, approval_threshold: e.target.value })} required /></div>
              <div className="form-group flex-1"><label className="text-secondary">Timeout (hours)</label><input className="input w-full" type="number" value={form.timeout_hours} onChange={(e) => setForm({ ...form, timeout_hours: e.target.value })} required /></div>
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
        <div className="card"><div className="card-body text-center py-8"><p className="text-secondary">No approval workflows found.</p></div></div>
      ) : (
        <div className="card"><div className="card-body overflow-x-auto">
          <table className="table"><thead><tr><th>Name</th><th>Trigger</th><th>Threshold</th><th>Timeout</th><th>Status</th><th>Actions</th></tr></thead>
            <tbody>
              {items.map((w) => (
                <tr key={w.id}>
                  <td className="text-primary">{w.name}</td>
                  <td className="text-secondary">{w.trigger_type}</td>
                  <td className="text-secondary">{w.approval_threshold}</td>
                  <td className="text-secondary">{w.timeout_hours}h</td>
                  <td><span className={`badge ${w.status === 'active' ? 'badge-success' : 'badge-neutral'}`}>{w.status}</span></td>
                  <td><div className="flex gap-2">
                    <button className="btn btn-primary" disabled={actionLoading} onClick={() => { setActionId(w.id); setActionComments(''); }}>Take Action</button>
                    <button className="btn btn-danger" disabled={actionLoading} onClick={() => { if (confirm('Delete this workflow?')) run(() => superAdminApi.deleteApprovalWorkflow(w.id)); }}>Delete</button>
                  </div></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div></div>
      )}

      {actionId && (
        <div className="card mt-4"><div className="card-body">
          <h3 className="text-primary mb-2">Take Approval Action</h3>
          <div className="form-group"><label className="text-secondary">Comments</label><input className="input w-full" value={actionComments} onChange={(e) => setActionComments(e.target.value)} /></div>
          <div className="flex gap-2 mt-2">
            <button className="btn btn-primary" disabled={actionLoading} onClick={() => handleTakeAction('approve')}>Approve</button>
            <button className="btn btn-danger" disabled={actionLoading} onClick={() => handleTakeAction('reject')}>Reject</button>
            <button className="btn btn-secondary" onClick={() => setActionId(null)}>Cancel</button>
          </div>
        </div></div>
      )}
    </div>
  );
}
