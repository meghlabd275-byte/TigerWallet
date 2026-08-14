/**
 * TigerWallet Super Admin - White Labels Page
 */

import React, { useState, useEffect } from 'react';
import { superAdminApi } from '../services/api';

export default function WhiteLabels() {
  const [items, setItems] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({ name: '', domain: '', owner_name: '', owner_email: '', plan: 'starter', fee_percent: '', primary_color: '', secondary_color: '', logo_url: '' });
  const [rejectId, setRejectId] = useState<string | null>(null);
  const [rejectReason, setRejectReason] = useState('');
  const [feeId, setFeeId] = useState<string | null>(null);
  const [feePercent, setFeePercent] = useState('');
  const [verifyTarget, setVerifyTarget] = useState<{ id: string; domain: string } | null>(null);

  const load = async () => {
    try {
      setLoading(true);
      setError(null);
      const res: any = await superAdminApi.getWhiteLabels();
      setItems(res.data || res.items || res || []);
    } catch (err: any) {
      setError(err?.message || 'Failed to load white labels');
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
      await superAdminApi.createWhiteLabel({
        name: form.name,
        domain: form.domain,
        owner_name: form.owner_name,
        owner_email: form.owner_email,
        plan: form.plan,
        fee_percent: form.fee_percent ? Number(form.fee_percent) : undefined,
        primary_color: form.primary_color || undefined,
        secondary_color: form.secondary_color || undefined,
        logo_url: form.logo_url || undefined,
      });
      setShowForm(false);
      setForm({ name: '', domain: '', owner_name: '', owner_email: '', plan: 'starter', fee_percent: '', primary_color: '', secondary_color: '', logo_url: '' });
      load();
    } catch (err: any) {
      alert(err?.message || 'Failed to create white label');
    } finally {
      setActionLoading(false);
    }
  };

  const run = async (fn: () => Promise<any>, reload = true) => {
    setActionLoading(true);
    try {
      await fn();
      if (reload) load();
    } catch (err: any) {
      alert(err?.message || 'Action failed');
    } finally {
      setActionLoading(false);
    }
  };

  const handleReject = async () => {
    if (!rejectId || !rejectReason) return;
    await run(() => superAdminApi.rejectWhiteLabel(rejectId, rejectReason));
    setRejectId(null);
    setRejectReason('');
  };

  const handleUpdateFee = async () => {
    if (!feeId || feePercent === '') return;
    await run(() => superAdminApi.updateWhiteLabelFee(feeId, Number(feePercent)));
    setFeeId(null);
    setFeePercent('');
  };

  const handleVerifyDomain = async () => {
    if (!verifyTarget) return;
    await run(async () => {
      const r = await superAdminApi.verifyWhiteLabelDomain(verifyTarget.id, verifyTarget.domain);
      alert(`Domain ${verifyTarget.domain} verified: ${r.verified}`);
    });
    setVerifyTarget(null);
  };

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold text-primary mb-6">White Labels</h1>

      <div className="flex justify-between items-center mb-4">
        <button className="btn btn-primary" onClick={() => setShowForm((s) => !s)}>
          {showForm ? 'Cancel' : 'New White Label'}
        </button>
      </div>

      {showForm && (
        <div className="card mb-6"><div className="card-body">
          <h3 className="text-primary mb-4">Create White Label</h3>
          <form onSubmit={handleCreate} className="flex flex-col gap-3">
            <div className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">Name</label><input className="input w-full" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} required /></div>
              <div className="form-group flex-1"><label className="text-secondary">Domain</label><input className="input w-full" value={form.domain} onChange={(e) => setForm({ ...form, domain: e.target.value })} required /></div>
            </div>
            <div className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">Owner Name</label><input className="input w-full" value={form.owner_name} onChange={(e) => setForm({ ...form, owner_name: e.target.value })} required /></div>
              <div className="form-group flex-1"><label className="text-secondary">Owner Email</label><input className="input w-full" value={form.owner_email} onChange={(e) => setForm({ ...form, owner_email: e.target.value })} required /></div>
              <div className="form-group flex-1"><label className="text-secondary">Plan</label><select className="input w-full" value={form.plan} onChange={(e) => setForm({ ...form, plan: e.target.value })}><option>starter</option><option>professional</option><option>enterprise</option></select></div>
              <div className="form-group flex-1"><label className="text-secondary">Fee Percent</label><input className="input w-full" type="number" value={form.fee_percent} onChange={(e) => setForm({ ...form, fee_percent: e.target.value })} /></div>
            </div>
            <div className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">Primary Color</label><input className="input w-full" value={form.primary_color} onChange={(e) => setForm({ ...form, primary_color: e.target.value })} /></div>
              <div className="form-group flex-1"><label className="text-secondary">Secondary Color</label><input className="input w-full" value={form.secondary_color} onChange={(e) => setForm({ ...form, secondary_color: e.target.value })} /></div>
              <div className="form-group flex-1"><label className="text-secondary">Logo URL</label><input className="input w-full" value={form.logo_url} onChange={(e) => setForm({ ...form, logo_url: e.target.value })} /></div>
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
        <div className="card"><div className="card-body text-center py-8"><p className="text-secondary">No white labels found.</p></div></div>
      ) : (
        <div className="card"><div className="card-body overflow-x-auto">
          <table className="table"><thead><tr><th>Name</th><th>Domain</th><th>Plan</th><th>Fee</th><th>Status</th><th>Verified</th><th>Actions</th></tr></thead>
            <tbody>
              {items.map((w) => (
                <tr key={w.id}>
                  <td className="text-primary">{w.name}</td>
                  <td className="text-secondary">{w.domain}</td>
                  <td className="text-secondary">{w.plan}</td>
                  <td className="text-secondary">{w.fee_percent}%</td>
                  <td><span className={`badge ${w.status === 'active' ? 'badge-success' : w.status === 'pending' ? 'badge-warning' : 'badge-error'}`}>{w.status}</span></td>
                  <td>{w.domain_verified ? <span className="badge badge-success">verified</span> : <span className="badge badge-neutral">unverified</span>}</td>
                  <td><div className="flex gap-2" style={{ flexWrap: 'wrap' }}>
                    {w.status === 'pending' && <button className="btn btn-primary" disabled={actionLoading} onClick={() => run(() => superAdminApi.approveWhiteLabel(w.id))}>Approve</button>}
                    {w.status === 'pending' && <button className="btn btn-danger" disabled={actionLoading} onClick={() => { setRejectId(w.id); setRejectReason(''); }}>Reject</button>}
                    {w.status === 'active' && <button className="btn btn-secondary" disabled={actionLoading} onClick={() => run(() => superAdminApi.suspendWhiteLabel(w.id))}>Suspend</button>}
                    {w.status === 'suspended' && <button className="btn btn-primary" disabled={actionLoading} onClick={() => run(() => superAdminApi.activateWhiteLabel(w.id))}>Activate</button>}
                    {!w.domain_verified && <button className="btn btn-secondary" disabled={actionLoading} onClick={() => setVerifyTarget({ id: w.id, domain: w.domain })}>Verify Domain</button>}
                    <button className="btn btn-secondary" disabled={actionLoading} onClick={() => { setFeeId(w.id); setFeePercent(String(w.fee_percent ?? '')); }}>Set Fee</button>
                    <button className="btn btn-secondary" disabled={actionLoading} onClick={() => run(async () => { const r = await superAdminApi.regenerateWhiteLabelAPIKey(w.id); alert(`New API key: ${r.api_key}`); }, false)}>Regen Key</button>
                    <button className="btn btn-danger" disabled={actionLoading} onClick={() => { if (confirm('Delete this white label?')) run(() => superAdminApi.deleteWhiteLabel(w.id)); }}>Delete</button>
                  </div></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div></div>
      )}

      {rejectId && (
        <div className="card mt-4"><div className="card-body">
          <h3 className="text-primary mb-2">Reject White Label</h3>
          <div className="form-group"><label className="text-secondary">Reason</label><input className="input w-full" value={rejectReason} onChange={(e) => setRejectReason(e.target.value)} /></div>
          <div className="flex gap-2 mt-2"><button className="btn btn-danger" disabled={actionLoading || !rejectReason} onClick={handleReject}>Confirm Reject</button><button className="btn btn-secondary" onClick={() => setRejectId(null)}>Cancel</button></div>
        </div></div>
      )}
      {feeId && (
        <div className="card mt-4"><div className="card-body">
          <h3 className="text-primary mb-2">Update Fee Percent</h3>
          <div className="form-group"><label className="text-secondary">Fee Percent</label><input className="input w-full" type="number" value={feePercent} onChange={(e) => setFeePercent(e.target.value)} /></div>
          <div className="flex gap-2 mt-2"><button className="btn btn-primary" disabled={actionLoading || feePercent === ''} onClick={handleUpdateFee}>Save</button><button className="btn btn-secondary" onClick={() => setFeeId(null)}>Cancel</button></div>
        </div></div>
      )}
      {verifyTarget && (
        <div className="card mt-4"><div className="card-body">
          <h3 className="text-primary mb-2">Verify Domain</h3>
          <p className="text-secondary">Verify <strong>{verifyTarget.domain}</strong>?</p>
          <div className="flex gap-2 mt-2"><button className="btn btn-primary" disabled={actionLoading} onClick={handleVerifyDomain}>Verify</button><button className="btn btn-secondary" onClick={() => setVerifyTarget(null)}>Cancel</button></div>
        </div></div>
      )}
    </div>
  );
}
