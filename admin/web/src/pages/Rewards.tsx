/**
 * TigerWallet Admin - Rewards Management Page
 * CRUD + status control for reward campaigns (mirrors /api/v1/rewards)
 */

import React, { useState, useEffect } from 'react';
import { useTheme } from '../hooks/useTheme';
import { rewardsAPI } from '../services/api';

interface Reward {
  id: string;
  name: string;
  description: string;
  type: 'trading' | 'referral' | 'signup' | 'staking' | 'loyalty';
  rewardAsset: string;
  rewardAmount: number;
  maxRecipients: number;
  claimed: number;
  startDate: string;
  endDate: string;
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

export const RewardsPage: React.FC = () => {
  const { isDark, toggleTheme } = useTheme();
  const [rewards, setRewards] = useState<Reward[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [editing, setEditing] = useState<Reward | null>(null);
  const [formData, setFormData] = useState({ name: '', description: '', type: 'trading', rewardAsset: 'USDT', rewardAmount: '', maxRecipients: '100', startDate: '', endDate: '' });

  const colors = {
    text: isDark ? '#f9fafb' : '#111827',
    textSecondary: isDark ? '#9ca3af' : '#6b7280',
    bgCard: isDark ? '#1e293b' : '#ffffff',
    border: isDark ? '#374151' : '#e5e7eb',
  };

  useEffect(() => { loadRewards(); }, []);

  const loadRewards = async () => {
    try { setLoading(true); setError(null); const res = await rewardsAPI.getAll(); setRewards(res.data || []); } catch (err) { setError(err instanceof Error ? err.message : 'Failed to load rewards'); } finally { setLoading(false); }
  };

  const resetForm = () => { setFormData({ name: '', description: '', type: 'trading', rewardAsset: 'USDT', rewardAmount: '', maxRecipients: '100', startDate: '', endDate: '' }); setEditing(null); };
  const openCreate = () => { resetForm(); setShowForm(true); };
  const openEdit = (r: Reward) => { setEditing(r); setFormData({ name: r.name, description: r.description, type: r.type, rewardAsset: r.rewardAsset, rewardAmount: String(r.rewardAmount), maxRecipients: String(r.maxRecipients), startDate: r.startDate ? r.startDate.substring(0, 10) : '', endDate: r.endDate ? r.endDate.substring(0, 10) : '' }); setShowForm(true); };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const payload = { ...formData, rewardAmount: Number(formData.rewardAmount), maxRecipients: Number(formData.maxRecipients) };
      if (editing) await rewardsAPI.update(editing.id, payload); else await rewardsAPI.create(payload);
      setShowForm(false); resetForm(); loadRewards();
    } catch (err) { setError(err instanceof Error ? err.message : 'Failed to save reward'); }
  };

  const handleDelete = async (id: string) => { if (!confirm('Delete this reward?')) return; try { await rewardsAPI.delete(id); loadRewards(); } catch (err) { setError(err instanceof Error ? err.message : 'Failed to delete reward'); } };
  const handleStatusChange = async (id: string, status: Status) => { try { await rewardsAPI.setStatus(id, status); loadRewards(); } catch (err) { setError(err instanceof Error ? err.message : 'Failed to update status'); } };

  return (
    <div className="p-6" style={{ color: colors.text }}>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold" style={{ color: colors.text }}>Rewards Management</h1>
        <div className="flex gap-2">
          <button className="btn btn-secondary" onClick={toggleTheme}>{isDark ? '☀️ Light' : '🌙 Dark'}</button>
          <button className="btn btn-primary" onClick={openCreate}>+ New Reward</button>
        </div>
      </div>

      {error && <div className="alert alert-error mb-4">{error}</div>}

      {showForm && (
        <div className="card mb-6" style={{ backgroundColor: colors.bgCard, border: `1px solid ${colors.border}` }}>
          <div className="card-header"><h2 style={{ color: colors.text }}>{editing ? 'Edit Reward' : 'New Reward'}</h2></div>
          <div className="card-body">
            <form onSubmit={handleSubmit}>
              <div className="flex gap-4" style={{ flexWrap: 'wrap' }}>
                <div className="form-group"><label className="form-label">Name</label><input className="form-input" value={formData.name} onChange={(e) => setFormData({ ...formData, name: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
                <div className="form-group"><label className="form-label">Type</label><select className="form-select" value={formData.type} onChange={(e) => setFormData({ ...formData, type: e.target.value as Reward['type'] })} style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }}><option value="trading">Trading</option><option value="referral">Referral</option><option value="signup">Signup</option><option value="staking">Staking</option><option value="loyalty">Loyalty</option></select></div>
                <div className="form-group"><label className="form-label">Reward Asset</label><input className="form-input" value={formData.rewardAsset} onChange={(e) => setFormData({ ...formData, rewardAsset: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
                <div className="form-group"><label className="form-label">Reward Amount</label><input className="form-input" type="number" step="0.0001" value={formData.rewardAmount} onChange={(e) => setFormData({ ...formData, rewardAmount: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
                <div className="form-group"><label className="form-label">Max Recipients</label><input className="form-input" type="number" step="1" value={formData.maxRecipients} onChange={(e) => setFormData({ ...formData, maxRecipients: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
                <div className="form-group"><label className="form-label">Start Date</label><input className="form-input" type="date" value={formData.startDate} onChange={(e) => setFormData({ ...formData, startDate: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
                <div className="form-group"><label className="form-label">End Date</label><input className="form-input" type="date" value={formData.endDate} onChange={(e) => setFormData({ ...formData, endDate: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
                <div className="form-group" style={{ flexBasis: '100%' }}><label className="form-label">Description</label><textarea className="form-textarea" value={formData.description} onChange={(e) => setFormData({ ...formData, description: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
              </div>
              <div className="flex gap-2 mt-4"><button type="submit" className="btn btn-primary">{editing ? 'Update' : 'Create'}</button><button type="button" className="btn btn-secondary" onClick={() => { setShowForm(false); resetForm(); }}>Cancel</button></div>
            </form>
          </div>
        </div>
      )}

      <div className="card" style={{ backgroundColor: colors.bgCard }}>
        <div className="card-body p-0">
          {loading ? (<div className="flex items-center justify-center p-8"><div className="loader"></div></div>) : rewards.length === 0 ? (
            <div className="text-center py-8" style={{ color: colors.textSecondary }}>No rewards found</div>
          ) : (
            <table className="table">
              <thead><tr><th>Name</th><th>Type</th><th>Asset</th><th>Amount</th><th>Claimed</th><th>Max</th><th>Period</th><th>Status</th><th>Created</th><th>Actions</th></tr></thead>
              <tbody>
                {rewards.map((r) => (
                  <tr key={r.id}>
                    <td style={{ color: colors.text }}>{r.name}</td>
                    <td style={{ color: colors.textSecondary }}>{r.type}</td>
                    <td style={{ color: colors.textSecondary }}>{r.rewardAsset}</td>
                    <td style={{ color: colors.textSecondary }}>{r.rewardAmount}</td>
                    <td style={{ color: colors.textSecondary }}>{r.claimed}</td>
                    <td style={{ color: colors.textSecondary }}>{r.maxRecipients}</td>
                    <td style={{ color: colors.textSecondary }}>{r.startDate ? new Date(r.startDate).toLocaleDateString() : '-'} → {r.endDate ? new Date(r.endDate).toLocaleDateString() : '-'}</td>
                    <td><span className={`badge ${statusBadgeClass(r.status)}`}>{r.status}</span></td>
                    <td style={{ color: colors.textSecondary }}>{new Date(r.createdAt).toLocaleDateString()}</td>
                    <td>
                      <div className="flex gap-2" style={{ flexWrap: 'wrap' }}>
                        <select className="form-select" style={{ width: 'auto' }} value={r.status} onChange={(e) => handleStatusChange(r.id, e.target.value as Status)}>{STATUS_OPTIONS.map((s) => <option key={s} value={s}>{s}</option>)}</select>
                        <button className="btn btn-sm btn-outline" onClick={() => openEdit(r)}>Edit</button>
                        <button className="btn btn-sm btn-danger" onClick={() => handleDelete(r.id)}>Delete</button>
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

export default RewardsPage;
