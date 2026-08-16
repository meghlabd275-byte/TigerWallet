/**
 * TigerWallet Admin - Copy Trading Management Page
 * CRUD + status control for copy-trading strategies (mirrors /api/v1/copy-trading)
 */

import React, { useState, useEffect } from 'react';
import { useTheme } from '../hooks/useTheme';
import { copyTradingAPI } from '../services/api';

interface CopyStrategy {
  id: string;
  name: string;
  leaderId: string;
  leaderName: string;
  minAllocation: number;
  performanceFee: number;
  maxCopiers: number;
  copiers: number;
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

export const CopyTradingPage: React.FC = () => {
  const { isDark, toggleTheme } = useTheme();
  const [strategies, setStrategies] = useState<CopyStrategy[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [editing, setEditing] = useState<CopyStrategy | null>(null);
  const [formData, setFormData] = useState({
    name: '', leaderId: '', leaderName: '', minAllocation: '100', performanceFee: '10', maxCopiers: '100',
  });

  const colors = {
    text: isDark ? '#f9fafb' : '#111827',
    textSecondary: isDark ? '#9ca3af' : '#6b7280',
    bgCard: isDark ? '#1e293b' : '#ffffff',
    border: isDark ? '#374151' : '#e5e7eb',
  };

  useEffect(() => { loadStrategies(); }, []);

  const loadStrategies = async () => {
    try {
      setLoading(true); setError(null);
      const res = await copyTradingAPI.getAll();
      setStrategies(res.data || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load copy strategies');
    } finally { setLoading(false); }
  };

  const resetForm = () => { setFormData({ name: '', leaderId: '', leaderName: '', minAllocation: '100', performanceFee: '10', maxCopiers: '100' }); setEditing(null); };
  const openCreate = () => { resetForm(); setShowForm(true); };
  const openEdit = (s: CopyStrategy) => {
    setEditing(s);
    setFormData({ name: s.name, leaderId: s.leaderId, leaderName: s.leaderName, minAllocation: String(s.minAllocation), performanceFee: String(s.performanceFee), maxCopiers: String(s.maxCopiers) });
    setShowForm(true);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const payload = { name: formData.name, leaderId: formData.leaderId, leaderName: formData.leaderName, minAllocation: Number(formData.minAllocation), performanceFee: Number(formData.performanceFee), maxCopiers: Number(formData.maxCopiers) };
      if (editing) await copyTradingAPI.update(editing.id, payload); else await copyTradingAPI.create(payload);
      setShowForm(false); resetForm(); loadStrategies();
    } catch (err) { setError(err instanceof Error ? err.message : 'Failed to save strategy'); }
  };

  const handleDelete = async (id: string) => { if (!confirm('Delete this strategy?')) return; try { await copyTradingAPI.delete(id); loadStrategies(); } catch (err) { setError(err instanceof Error ? err.message : 'Failed to delete strategy'); } };
  const handleStatusChange = async (id: string, status: Status) => { try { await copyTradingAPI.setStatus(id, status); loadStrategies(); } catch (err) { setError(err instanceof Error ? err.message : 'Failed to update status'); } };

  return (
    <div className="p-6" style={{ color: colors.text }}>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold" style={{ color: colors.text }}>Copy Trading Management</h1>
        <div className="flex gap-2">
          <button className="btn btn-secondary" onClick={toggleTheme}>{isDark ? '☀️ Light' : '🌙 Dark'}</button>
          <button className="btn btn-primary" onClick={openCreate}>+ New Strategy</button>
        </div>
      </div>

      {error && <div className="alert alert-error mb-4">{error}</div>}

      {showForm && (
        <div className="card mb-6" style={{ backgroundColor: colors.bgCard, border: `1px solid ${colors.border}` }}>
          <div className="card-header"><h2 style={{ color: colors.text }}>{editing ? 'Edit Strategy' : 'New Strategy'}</h2></div>
          <div className="card-body">
            <form onSubmit={handleSubmit}>
              <div className="flex gap-4" style={{ flexWrap: 'wrap' }}>
                <div className="form-group"><label className="form-label">Name</label><input className="form-input" value={formData.name} onChange={(e) => setFormData({ ...formData, name: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
                <div className="form-group"><label className="form-label">Leader ID</label><input className="form-input" value={formData.leaderId} onChange={(e) => setFormData({ ...formData, leaderId: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
                <div className="form-group"><label className="form-label">Leader Name</label><input className="form-input" value={formData.leaderName} onChange={(e) => setFormData({ ...formData, leaderName: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
                <div className="form-group"><label className="form-label">Min Allocation</label><input className="form-input" type="number" step="1" value={formData.minAllocation} onChange={(e) => setFormData({ ...formData, minAllocation: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
                <div className="form-group"><label className="form-label">Performance Fee (%)</label><input className="form-input" type="number" step="0.1" value={formData.performanceFee} onChange={(e) => setFormData({ ...formData, performanceFee: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
                <div className="form-group"><label className="form-label">Max Copiers</label><input className="form-input" type="number" step="1" value={formData.maxCopiers} onChange={(e) => setFormData({ ...formData, maxCopiers: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
              </div>
              <div className="flex gap-2 mt-4"><button type="submit" className="btn btn-primary">{editing ? 'Update' : 'Create'}</button><button type="button" className="btn btn-secondary" onClick={() => { setShowForm(false); resetForm(); }}>Cancel</button></div>
            </form>
          </div>
        </div>
      )}

      <div className="card" style={{ backgroundColor: colors.bgCard }}>
        <div className="card-body p-0">
          {loading ? (<div className="flex items-center justify-center p-8"><div className="loader"></div></div>) : strategies.length === 0 ? (
            <div className="text-center py-8" style={{ color: colors.textSecondary }}>No copy strategies found</div>
          ) : (
            <table className="table">
              <thead><tr><th>Name</th><th>Leader</th><th>Min Alloc.</th><th>Perf Fee</th><th>Copiers</th><th>Max</th><th>Status</th><th>Created</th><th>Actions</th></tr></thead>
              <tbody>
                {strategies.map((s) => (
                  <tr key={s.id}>
                    <td style={{ color: colors.text }}>{s.name}</td>
                    <td style={{ color: colors.textSecondary }}>{s.leaderName}</td>
                    <td style={{ color: colors.textSecondary }}>{s.minAllocation}</td>
                    <td style={{ color: colors.textSecondary }}>{s.performanceFee}%</td>
                    <td style={{ color: colors.textSecondary }}>{s.copiers}</td>
                    <td style={{ color: colors.textSecondary }}>{s.maxCopiers}</td>
                    <td><span className={`badge ${statusBadgeClass(s.status)}`}>{s.status}</span></td>
                    <td style={{ color: colors.textSecondary }}>{new Date(s.createdAt).toLocaleDateString()}</td>
                    <td>
                      <div className="flex gap-2" style={{ flexWrap: 'wrap' }}>
                        <select className="form-select" style={{ width: 'auto' }} value={s.status} onChange={(e) => handleStatusChange(s.id, e.target.value as Status)}>{STATUS_OPTIONS.map((st) => <option key={st} value={st}>{st}</option>)}</select>
                        <button className="btn btn-sm btn-outline" onClick={() => openEdit(s)}>Edit</button>
                        <button className="btn btn-sm btn-danger" onClick={() => handleDelete(s.id)}>Delete</button>
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

export default CopyTradingPage;
