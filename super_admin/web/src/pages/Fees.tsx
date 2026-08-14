/**
 * TigerWallet Super Admin - Fee Structures Page
 */

import React, { useState, useEffect } from 'react';
import { superAdminApi } from '../services/api';

export default function Fees() {
  const [items, setItems] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({
    fee_type: 'withdrawal', asset: '', chain: '', fee_percent: '', fee_fixed: '', min_fee: '', max_fee: '', tier: 'standard', effective_from: '',
  });

  const load = async () => {
    try {
      setLoading(true);
      setError(null);
      const res: any = await superAdminApi.getFeeStructures();
      setItems(res.data || res.items || res || []);
    } catch (err: any) {
      setError(err?.message || 'Failed to load fee structures');
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
      await superAdminApi.createFeeStructure({
        fee_type: form.fee_type,
        asset: form.asset,
        chain: form.chain || undefined,
        fee_percent: Number(form.fee_percent),
        fee_fixed: Number(form.fee_fixed),
        min_fee: Number(form.min_fee),
        max_fee: form.max_fee ? Number(form.max_fee) : undefined,
        tier: form.tier,
        effective_from: form.effective_from || undefined,
      });
      setShowForm(false);
      setForm({ fee_type: 'withdrawal', asset: '', chain: '', fee_percent: '', fee_fixed: '', min_fee: '', max_fee: '', tier: 'standard', effective_from: '' });
      load();
    } catch (err: any) {
      alert(err?.message || 'Failed to create fee structure');
    } finally {
      setActionLoading(false);
    }
  };

  const handleAction = async (id: string, action: 'activate' | 'deactivate' | 'delete') => {
    if (action === 'delete' && !confirm('Delete this fee structure?')) return;
    setActionLoading(true);
    try {
      if (action === 'activate') await superAdminApi.activateFeeStructure(id);
      else if (action === 'deactivate') await superAdminApi.deactivateFeeStructure(id);
      else await superAdminApi.deleteFeeStructure(id);
      load();
    } catch (err: any) {
      alert(err?.message || 'Action failed');
    } finally {
      setActionLoading(false);
    }
  };

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold text-primary mb-6">Fee Structures</h1>

      <div className="flex justify-between items-center mb-4">
        <button className="btn btn-primary" onClick={() => setShowForm((s) => !s)}>
          {showForm ? 'Cancel' : 'New Fee Structure'}
        </button>
      </div>

      {showForm && (
        <div className="card mb-6"><div className="card-body">
          <h3 className="text-primary mb-4">Create Fee Structure</h3>
          <form onSubmit={handleCreate} className="flex flex-col gap-3">
            <div className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">Fee Type</label><select className="input w-full" value={form.fee_type} onChange={(e) => setForm({ ...form, fee_type: e.target.value })}><option>withdrawal</option><option>deposit</option><option>trading</option><option>swap</option><option>transfer</option><option>conversion</option></select></div>
              <div className="form-group flex-1"><label className="text-secondary">Asset</label><input className="input w-full" value={form.asset} onChange={(e) => setForm({ ...form, asset: e.target.value })} required /></div>
              <div className="form-group flex-1"><label className="text-secondary">Chain</label><input className="input w-full" value={form.chain} onChange={(e) => setForm({ ...form, chain: e.target.value })} /></div>
              <div className="form-group flex-1"><label className="text-secondary">Tier</label><input className="input w-full" value={form.tier} onChange={(e) => setForm({ ...form, tier: e.target.value })} required /></div>
            </div>
            <div className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">Fee Percent</label><input className="input w-full" type="number" value={form.fee_percent} onChange={(e) => setForm({ ...form, fee_percent: e.target.value })} required /></div>
              <div className="form-group flex-1"><label className="text-secondary">Fee Fixed</label><input className="input w-full" type="number" value={form.fee_fixed} onChange={(e) => setForm({ ...form, fee_fixed: e.target.value })} required /></div>
              <div className="form-group flex-1"><label className="text-secondary">Min Fee</label><input className="input w-full" type="number" value={form.min_fee} onChange={(e) => setForm({ ...form, min_fee: e.target.value })} required /></div>
              <div className="form-group flex-1"><label className="text-secondary">Max Fee</label><input className="input w-full" type="number" value={form.max_fee} onChange={(e) => setForm({ ...form, max_fee: e.target.value })} /></div>
            </div>
            <div className="form-group"><label className="text-secondary">Effective From</label><input className="input w-full" type="date" value={form.effective_from} onChange={(e) => setForm({ ...form, effective_from: e.target.value })} /></div>
            <button className="btn btn-primary" disabled={actionLoading} type="submit">Create</button>
          </form>
        </div></div>
      )}

      {error ? (
        <div className="alert alert-error mb-4"><p className="text-error">{error}</p><button className="btn btn-secondary mt-2" onClick={load}>Retry</button></div>
      ) : loading ? (
        <div className="flex items-center justify-center p-8"><div className="loader"></div></div>
      ) : items.length === 0 ? (
        <div className="card"><div className="card-body text-center py-8"><p className="text-secondary">No fee structures found.</p></div></div>
      ) : (
        <div className="card"><div className="card-body overflow-x-auto">
          <table className="table"><thead><tr><th>Type</th><th>Asset</th><th>Chain</th><th>Percent</th><th>Fixed</th><th>Min</th><th>Tier</th><th>Active</th><th>Actions</th></tr></thead>
            <tbody>
              {items.map((f) => (
                <tr key={f.id}>
                  <td className="text-primary">{f.fee_type}</td>
                  <td className="text-primary">{f.asset}</td>
                  <td className="text-secondary">{f.chain || '-'}</td>
                  <td className="text-secondary">{f.fee_percent}%</td>
                  <td className="text-secondary">{f.fee_fixed}</td>
                  <td className="text-secondary">{f.min_fee}</td>
                  <td className="text-secondary">{f.tier}</td>
                  <td><span className={`badge ${f.is_active ? 'badge-success' : 'badge-neutral'}`}>{f.is_active ? 'active' : 'inactive'}</span></td>
                  <td><div className="flex gap-2">
                    {f.is_active ? (
                      <button className="btn btn-secondary" disabled={actionLoading} onClick={() => handleAction(f.id, 'deactivate')}>Deactivate</button>
                    ) : (
                      <button className="btn btn-primary" disabled={actionLoading} onClick={() => handleAction(f.id, 'activate')}>Activate</button>
                    )}
                    <button className="btn btn-danger" disabled={actionLoading} onClick={() => handleAction(f.id, 'delete')}>Delete</button>
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
