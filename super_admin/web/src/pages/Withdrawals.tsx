/**
 * TigerWallet Super Admin - Withdrawals Page
 */

import React, { useState, useEffect } from 'react';
import { superAdminApi } from '../services/api';

export default function Withdrawals() {
  const [items, setItems] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [rejectId, setRejectId] = useState<string | null>(null);
  const [reason, setReason] = useState('');

  const load = async () => {
    try {
      setLoading(true);
      setError(null);
      const res: any = await superAdminApi.getWithdrawals();
      setItems(res.data || res.items || res || []);
    } catch (err: any) {
      setError(err?.message || 'Failed to load withdrawals');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const toggle = (id: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id); else next.add(id);
      return next;
    });
  };

  const handleApprove = async (id: string) => {
    setActionLoading(true);
    try {
      await superAdminApi.approveWithdrawal(id);
      load();
    } catch (err: any) {
      alert(err?.message || 'Failed to approve withdrawal');
    } finally {
      setActionLoading(false);
    }
  };

  const handleReject = async () => {
    if (!rejectId || !reason) return;
    setActionLoading(true);
    try {
      await superAdminApi.rejectWithdrawal(rejectId, { reason });
      setRejectId(null);
      setReason('');
      load();
    } catch (err: any) {
      alert(err?.message || 'Failed to reject withdrawal');
    } finally {
      setActionLoading(false);
    }
  };

  const handleBatchApprove = async () => {
    const ids = Array.from(selected);
    if (ids.length === 0) return;
    if (!confirm(`Batch approve ${ids.length} withdrawals?`)) return;
    setActionLoading(true);
    try {
      const r = await superAdminApi.batchApproveWithdrawals(ids);
      alert(`Approved: ${r.approved}, Failed: ${r.failed}`);
      setSelected(new Set());
      load();
    } catch (err: any) {
      alert(err?.message || 'Batch approve failed');
    } finally {
      setActionLoading(false);
    }
  };

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold text-primary mb-6">Withdrawals</h1>

      {selected.size > 0 && (
        <div className="flex items-center gap-3 mb-4">
          <span className="text-secondary">{selected.size} selected</span>
          <button className="btn btn-primary" disabled={actionLoading} onClick={handleBatchApprove}>Batch Approve</button>
          <button className="btn btn-secondary" onClick={() => setSelected(new Set())}>Clear</button>
        </div>
      )}

      {error ? (
        <div className="alert alert-error mb-4"><p className="text-error">{error}</p><button className="btn btn-secondary mt-2" onClick={load}>Retry</button></div>
      ) : loading ? (
        <div className="flex items-center justify-center p-8"><div className="loader"></div></div>
      ) : items.length === 0 ? (
        <div className="card"><div className="card-body text-center py-8"><p className="text-secondary">No withdrawals found.</p></div></div>
      ) : (
        <div className="card"><div className="card-body overflow-x-auto">
          <table className="table"><thead><tr><th></th><th>ID</th><th>User</th><th>Amount</th><th>Currency</th><th>Chain</th><th>Status</th><th>Actions</th></tr></thead>
            <tbody>
              {items.map((w) => (
                <tr key={w.id}>
                  <td><input type="checkbox" checked={selected.has(w.id)} onChange={() => toggle(w.id)} disabled={w.status !== 'pending'} /></td>
                  <td className="text-secondary">{w.id.slice(0, 8)}...</td>
                  <td className="text-secondary">{w.user_id}</td>
                  <td className="text-primary">{w.amount}</td>
                  <td className="text-primary">{w.currency || w.asset}</td>
                  <td className="text-secondary">{w.chain}</td>
                  <td><span className={`badge ${w.status === 'pending' ? 'badge-warning' : w.status === 'approved' || w.status === 'completed' ? 'badge-success' : 'badge-error'}`}>{w.status}</span></td>
                  <td><div className="flex gap-2">
                    {w.status === 'pending' && <>
                      <button className="btn btn-primary" disabled={actionLoading} onClick={() => handleApprove(w.id)}>Approve</button>
                      <button className="btn btn-danger" disabled={actionLoading} onClick={() => { setRejectId(w.id); setReason(''); }}>Reject</button>
                    </>}
                  </div></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div></div>
      )}

      {rejectId && (
        <div className="card mt-4"><div className="card-body">
          <h3 className="text-primary mb-2">Reject Withdrawal</h3>
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
