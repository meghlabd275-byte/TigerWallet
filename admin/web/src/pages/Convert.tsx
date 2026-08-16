/**
 * TigerWallet Admin - Convert Management Page
 * CRUD + status control for convert (swap) pairs (mirrors /api/v1/convert)
 */

import React, { useState, useEffect } from 'react';
import { useTheme } from '../hooks/useTheme';
import { convertAPI } from '../services/api';

interface ConvertPair {
  id: string;
  fromAsset: string;
  toAsset: string;
  minAmount: number;
  maxAmount: number;
  fee: number;
  status: 'active' | 'paused' | 'suspended' | 'halted';
  createdAt: string;
}

const STATUS_OPTIONS = ['active', 'paused', 'suspended', 'halted'] as const;
type Status = typeof STATUS_OPTIONS[number];

const statusBadgeClass = (status: string): string => {
  switch (status) {
    case 'active': return 'badge-success';
    case 'paused': return 'badge-warning';
    case 'suspended': return 'badge-error';
    case 'halted': return 'badge-neutral';
    default: return 'badge-neutral';
  }
};

export const ConvertPage: React.FC = () => {
  const { isDark, toggleTheme } = useTheme();
  const [pairs, setPairs] = useState<ConvertPair[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [editing, setEditing] = useState<ConvertPair | null>(null);
  const [formData, setFormData] = useState({ fromAsset: '', toAsset: '', minAmount: '1', maxAmount: '100000', fee: '0.001' });

  const colors = {
    text: isDark ? '#f9fafb' : '#111827',
    textSecondary: isDark ? '#9ca3af' : '#6b7280',
    bgCard: isDark ? '#1e293b' : '#ffffff',
    border: isDark ? '#374151' : '#e5e7eb',
  };

  useEffect(() => { loadPairs(); }, []);

  const loadPairs = async () => {
    try { setLoading(true); setError(null); const res = await convertAPI.getAll(); setPairs(res.data || []); } catch (err) { setError(err instanceof Error ? err.message : 'Failed to load convert pairs'); } finally { setLoading(false); }
  };

  const resetForm = () => { setFormData({ fromAsset: '', toAsset: '', minAmount: '1', maxAmount: '100000', fee: '0.001' }); setEditing(null); };
  const openCreate = () => { resetForm(); setShowForm(true); };
  const openEdit = (p: ConvertPair) => { setEditing(p); setFormData({ fromAsset: p.fromAsset, toAsset: p.toAsset, minAmount: String(p.minAmount), maxAmount: String(p.maxAmount), fee: String(p.fee) }); setShowForm(true); };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const payload = { fromAsset: formData.fromAsset, toAsset: formData.toAsset, minAmount: Number(formData.minAmount), maxAmount: Number(formData.maxAmount), fee: Number(formData.fee) };
      if (editing) await convertAPI.update(editing.id, payload); else await convertAPI.create(payload);
      setShowForm(false); resetForm(); loadPairs();
    } catch (err) { setError(err instanceof Error ? err.message : 'Failed to save convert pair'); }
  };

  const handleDelete = async (id: string) => { if (!confirm('Delete this convert pair?')) return; try { await convertAPI.delete(id); loadPairs(); } catch (err) { setError(err instanceof Error ? err.message : 'Failed to delete pair'); } };
  const handleStatusChange = async (id: string, status: Status) => { try { await convertAPI.setStatus(id, status); loadPairs(); } catch (err) { setError(err instanceof Error ? err.message : 'Failed to update status'); } };

  return (
    <div className="p-6" style={{ color: colors.text }}>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold" style={{ color: colors.text }}>Convert Management</h1>
        <div className="flex gap-2">
          <button className="btn btn-secondary" onClick={toggleTheme}>{isDark ? '☀️ Light' : '🌙 Dark'}</button>
          <button className="btn btn-primary" onClick={openCreate}>+ New Pair</button>
        </div>
      </div>

      {error && <div className="alert alert-error mb-4">{error}</div>}

      {showForm && (
        <div className="card mb-6" style={{ backgroundColor: colors.bgCard, border: `1px solid ${colors.border}` }}>
          <div className="card-header"><h2 style={{ color: colors.text }}>{editing ? 'Edit Pair' : 'New Pair'}</h2></div>
          <div className="card-body">
            <form onSubmit={handleSubmit}>
              <div className="flex gap-4" style={{ flexWrap: 'wrap' }}>
                <div className="form-group"><label className="form-label">From Asset</label><input className="form-input" value={formData.fromAsset} onChange={(e) => setFormData({ ...formData, fromAsset: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
                <div className="form-group"><label className="form-label">To Asset</label><input className="form-input" value={formData.toAsset} onChange={(e) => setFormData({ ...formData, toAsset: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
                <div className="form-group"><label className="form-label">Min Amount</label><input className="form-input" type="number" step="0.0001" value={formData.minAmount} onChange={(e) => setFormData({ ...formData, minAmount: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
                <div className="form-group"><label className="form-label">Max Amount</label><input className="form-input" type="number" step="0.0001" value={formData.maxAmount} onChange={(e) => setFormData({ ...formData, maxAmount: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
                <div className="form-group"><label className="form-label">Fee</label><input className="form-input" type="number" step="0.0001" value={formData.fee} onChange={(e) => setFormData({ ...formData, fee: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
              </div>
              <div className="flex gap-2 mt-4"><button type="submit" className="btn btn-primary">{editing ? 'Update' : 'Create'}</button><button type="button" className="btn btn-secondary" onClick={() => { setShowForm(false); resetForm(); }}>Cancel</button></div>
            </form>
          </div>
        </div>
      )}

      <div className="card" style={{ backgroundColor: colors.bgCard }}>
        <div className="card-body p-0">
          {loading ? (<div className="flex items-center justify-center p-8"><div className="loader"></div></div>) : pairs.length === 0 ? (
            <div className="text-center py-8" style={{ color: colors.textSecondary }}>No convert pairs found</div>
          ) : (
            <table className="table">
              <thead><tr><th>From</th><th>To</th><th>Min</th><th>Max</th><th>Fee</th><th>Status</th><th>Created</th><th>Actions</th></tr></thead>
              <tbody>
                {pairs.map((p) => (
                  <tr key={p.id}>
                    <td style={{ color: colors.text }}>{p.fromAsset}</td>
                    <td style={{ color: colors.textSecondary }}>{p.toAsset}</td>
                    <td style={{ color: colors.textSecondary }}>{p.minAmount}</td>
                    <td style={{ color: colors.textSecondary }}>{p.maxAmount}</td>
                    <td style={{ color: colors.textSecondary }}>{p.fee}</td>
                    <td><span className={`badge ${statusBadgeClass(p.status)}`}>{p.status}</span></td>
                    <td style={{ color: colors.textSecondary }}>{new Date(p.createdAt).toLocaleDateString()}</td>
                    <td>
                      <div className="flex gap-2" style={{ flexWrap: 'wrap' }}>
                        <select className="form-select" style={{ width: 'auto' }} value={p.status} onChange={(e) => handleStatusChange(p.id, e.target.value as Status)}>{STATUS_OPTIONS.map((s) => <option key={s} value={s}>{s}</option>)}</select>
                        <button className="btn btn-sm btn-outline" onClick={() => openEdit(p)}>Edit</button>
                        <button className="btn btn-sm btn-danger" onClick={() => handleDelete(p.id)}>Delete</button>
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

export default ConvertPage;
