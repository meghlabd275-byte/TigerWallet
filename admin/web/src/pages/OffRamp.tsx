/**
 * TigerWallet Admin - OffRamp Management Page
 * CRUD + status + approve/reject for crypto->crypto offramp orders (mirrors /api/v1/offramp)
 */

import React, { useState, useEffect } from 'react';
import { useTheme } from '../hooks/useTheme';
import { offRampAPI } from '../services/api';

interface OffRampOrder {
  id: string;
  userId: string;
  userName: string;
  cryptoAsset: string;
  fiatCurrency: string;
  cryptoAmount: number;
  fiatAmount: number;
  payoutMethod: string;
  provider: string;
  status: 'pending' | 'approved' | 'rejected' | 'completed' | 'failed';
  createdAt: string;
}

const STATUS_OPTIONS = ['active', 'paused', 'suspended', 'halted'] as const;
type Status = typeof STATUS_OPTIONS[number];

const statusBadgeClass = (status: string): string => {
  switch (status) {
    case 'approved': case 'completed': return 'badge-success';
    case 'pending': return 'badge-warning';
    case 'rejected': case 'failed': return 'badge-error';
    default: return 'badge-neutral';
  }
};

export const OffRampPage: React.FC = () => {
  const { isDark, toggleTheme } = useTheme();
  const [orders, setOrders] = useState<OffRampOrder[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [editing, setEditing] = useState<OffRampOrder | null>(null);
  const [rejectingId, setRejectingId] = useState<string | null>(null);
  const [rejectReason, setRejectReason] = useState('');
  const [formData, setFormData] = useState({ userId: '', userName: '', cryptoAsset: 'BTC', fiatCurrency: 'USD', cryptoAmount: '', fiatAmount: '', payoutMethod: 'bank_transfer', provider: 'stripe' });

  const colors = {
    text: isDark ? '#f9fafb' : '#111827',
    textSecondary: isDark ? '#9ca3af' : '#6b7280',
    bgCard: isDark ? '#1e293b' : '#ffffff',
    border: isDark ? '#374151' : '#e5e7eb',
  };

  useEffect(() => { loadOrders(); }, []);

  const loadOrders = async () => {
    try { setLoading(true); setError(null); const res = await offRampAPI.getAll(); setOrders(res.data || []); } catch (err) { setError(err instanceof Error ? err.message : 'Failed to load offramp orders'); } finally { setLoading(false); }
  };

  const resetForm = () => { setFormData({ userId: '', userName: '', cryptoAsset: 'BTC', fiatCurrency: 'USD', cryptoAmount: '', fiatAmount: '', payoutMethod: 'bank_transfer', provider: 'stripe' }); setEditing(null); };
  const openCreate = () => { resetForm(); setShowForm(true); };
  const openEdit = (o: OffRampOrder) => { setEditing(o); setFormData({ userId: o.userId, userName: o.userName, cryptoAsset: o.cryptoAsset, fiatCurrency: o.fiatCurrency, cryptoAmount: String(o.cryptoAmount), fiatAmount: String(o.fiatAmount), payoutMethod: o.payoutMethod, provider: o.provider }); setShowForm(true); };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const payload = { ...formData, cryptoAmount: Number(formData.cryptoAmount), fiatAmount: Number(formData.fiatAmount) };
      if (editing) await offRampAPI.update(editing.id, payload); else await offRampAPI.create(payload);
      setShowForm(false); resetForm(); loadOrders();
    } catch (err) { setError(err instanceof Error ? err.message : 'Failed to save order'); }
  };

  const handleDelete = async (id: string) => { if (!confirm('Delete this offramp order?')) return; try { await offRampAPI.delete(id); loadOrders(); } catch (err) { setError(err instanceof Error ? err.message : 'Failed to delete order'); } };
  const handleStatusChange = async (id: string, status: Status) => { try { await offRampAPI.setStatus(id, status); loadOrders(); } catch (err) { setError(err instanceof Error ? err.message : 'Failed to update status'); } };
  const handleApprove = async (id: string) => { try { await offRampAPI.approve(id); loadOrders(); } catch (err) { setError(err instanceof Error ? err.message : 'Failed to approve order'); } };
  const handleReject = async () => { if (!rejectingId) return; try { await offRampAPI.reject(rejectingId, rejectReason); setRejectingId(null); setRejectReason(''); loadOrders(); } catch (err) { setError(err instanceof Error ? err.message : 'Failed to reject order'); } };

  return (
    <div className="p-6" style={{ color: colors.text }}>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold" style={{ color: colors.text }}>OffRamp Management</h1>
        <div className="flex gap-2">
          <button className="btn btn-secondary" onClick={toggleTheme}>{isDark ? '☀️ Light' : '🌙 Dark'}</button>
          <button className="btn btn-primary" onClick={openCreate}>+ New Order</button>
        </div>
      </div>

      {error && <div className="alert alert-error mb-4">{error}</div>}

      {showForm && (
        <div className="card mb-6" style={{ backgroundColor: colors.bgCard, border: `1px solid ${colors.border}` }}>
          <div className="card-header"><h2 style={{ color: colors.text }}>{editing ? 'Edit Order' : 'New Order'}</h2></div>
          <div className="card-body">
            <form onSubmit={handleSubmit}>
              <div className="flex gap-4" style={{ flexWrap: 'wrap' }}>
                <div className="form-group"><label className="form-label">User ID</label><input className="form-input" value={formData.userId} onChange={(e) => setFormData({ ...formData, userId: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
                <div className="form-group"><label className="form-label">User Name</label><input className="form-input" value={formData.userName} onChange={(e) => setFormData({ ...formData, userName: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
                <div className="form-group"><label className="form-label">Crypto Asset</label><input className="form-input" value={formData.cryptoAsset} onChange={(e) => setFormData({ ...formData, cryptoAsset: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
                <div className="form-group"><label className="form-label">Fiat Currency</label><input className="form-input" value={formData.fiatCurrency} onChange={(e) => setFormData({ ...formData, fiatCurrency: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
                <div className="form-group"><label className="form-label">Crypto Amount</label><input className="form-input" type="number" step="0.00000001" value={formData.cryptoAmount} onChange={(e) => setFormData({ ...formData, cryptoAmount: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
                <div className="form-group"><label className="form-label">Fiat Amount</label><input className="form-input" type="number" step="0.01" value={formData.fiatAmount} onChange={(e) => setFormData({ ...formData, fiatAmount: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
                <div className="form-group"><label className="form-label">Payout Method</label><input className="form-input" value={formData.payoutMethod} onChange={(e) => setFormData({ ...formData, payoutMethod: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
                <div className="form-group"><label className="form-label">Provider</label><input className="form-input" value={formData.provider} onChange={(e) => setFormData({ ...formData, provider: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
              </div>
              <div className="flex gap-2 mt-4"><button type="submit" className="btn btn-primary">{editing ? 'Update' : 'Create'}</button><button type="button" className="btn btn-secondary" onClick={() => { setShowForm(false); resetForm(); }}>Cancel</button></div>
            </form>
          </div>
        </div>
      )}

      {rejectingId && (
        <div className="modal-overlay" onClick={() => setRejectingId(null)}>
          <div className="modal" onClick={(e) => e.stopPropagation()} style={{ backgroundColor: colors.bgCard }}>
            <div className="modal-header"><h3 style={{ color: colors.text }}>Reject Order</h3></div>
            <div className="modal-body"><div className="form-group"><label className="form-label">Reason</label><textarea className="form-textarea" value={rejectReason} onChange={(e) => setRejectReason(e.target.value)} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div></div>
            <div className="modal-footer"><button className="btn btn-danger" onClick={handleReject}>Reject</button><button className="btn btn-secondary" onClick={() => setRejectingId(null)}>Cancel</button></div>
          </div>
        </div>
      )}

      <div className="card" style={{ backgroundColor: colors.bgCard }}>
        <div className="card-body p-0">
          {loading ? (<div className="flex items-center justify-center p-8"><div className="loader"></div></div>) : orders.length === 0 ? (
            <div className="text-center py-8" style={{ color: colors.textSecondary }}>No offramp orders found</div>
          ) : (
            <table className="table">
              <thead><tr><th>User</th><th>Crypto</th><th>Fiat</th><th>Crypto Amt</th><th>Fiat Amt</th><th>Payout</th><th>Provider</th><th>Status</th><th>Created</th><th>Actions</th></tr></thead>
              <tbody>
                {orders.map((o) => (
                  <tr key={o.id}>
                    <td style={{ color: colors.text }}>{o.userName}</td>
                    <td style={{ color: colors.textSecondary }}>{o.cryptoAsset}</td>
                    <td style={{ color: colors.textSecondary }}>{o.fiatCurrency}</td>
                    <td style={{ color: colors.textSecondary }}>{o.cryptoAmount}</td>
                    <td style={{ color: colors.textSecondary }}>{o.fiatAmount}</td>
                    <td style={{ color: colors.textSecondary }}>{o.payoutMethod}</td>
                    <td style={{ color: colors.textSecondary }}>{o.provider}</td>
                    <td><span className={`badge ${statusBadgeClass(o.status)}`}>{o.status}</span></td>
                    <td style={{ color: colors.textSecondary }}>{new Date(o.createdAt).toLocaleDateString()}</td>
                    <td>
                      <div className="flex gap-2" style={{ flexWrap: 'wrap' }}>
                        {o.status === 'pending' && (<><button className="btn btn-sm btn-success" onClick={() => handleApprove(o.id)}>Approve</button><button className="btn btn-sm btn-danger" onClick={() => { setRejectingId(o.id); setRejectReason(''); }}>Reject</button></>)}
                        <select className="form-select" style={{ width: 'auto' }} value={o.status} onChange={(e) => handleStatusChange(o.id, e.target.value as Status)}>{STATUS_OPTIONS.map((s) => <option key={s} value={s}>{s}</option>)}</select>
                        <button className="btn btn-sm btn-outline" onClick={() => openEdit(o)}>Edit</button>
                        <button className="btn btn-sm btn-danger" onClick={() => handleDelete(o.id)}>Delete</button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>
    </div>
  );
};

export default OffRampPage;
