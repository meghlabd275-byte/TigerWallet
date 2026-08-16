/**
 * TigerWallet Admin - Marketing Management Page
 * CRUD + status control for marketing campaigns (mirrors /api/v1/marketing)
 */

import React, { useState, useEffect } from 'react';
import { useTheme } from '../hooks/useTheme';
import { marketingAPI } from '../services/api';

interface Campaign {
  id: string;
  name: string;
  description: string;
  channel: 'email' | 'push' | 'sms' | 'in_app' | 'social';
  audience: string;
  budget: number;
  spent: number;
  impressions: number;
  clicks: number;
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

export const MarketingPage: React.FC = () => {
  const { isDark, toggleTheme } = useTheme();
  const [campaigns, setCampaigns] = useState<Campaign[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [editing, setEditing] = useState<Campaign | null>(null);
  const [formData, setFormData] = useState({ name: '', description: '', channel: 'email', audience: 'all', budget: '', startDate: '', endDate: '' });

  const colors = {
    text: isDark ? '#f9fafb' : '#111827',
    textSecondary: isDark ? '#9ca3af' : '#6b7280',
    bgCard: isDark ? '#1e293b' : '#ffffff',
    border: isDark ? '#374151' : '#e5e7eb',
  };

  useEffect(() => { loadCampaigns(); }, []);

  const loadCampaigns = async () => {
    try { setLoading(true); setError(null); const res = await marketingAPI.getAll(); setCampaigns(res.data || []); } catch (err) { setError(err instanceof Error ? err.message : 'Failed to load campaigns'); } finally { setLoading(false); }
  };

  const resetForm = () => { setFormData({ name: '', description: '', channel: 'email', audience: 'all', budget: '', startDate: '', endDate: '' }); setEditing(null); };
  const openCreate = () => { resetForm(); setShowForm(true); };
  const openEdit = (c: Campaign) => { setEditing(c); setFormData({ name: c.name, description: c.description, channel: c.channel, audience: c.audience, budget: String(c.budget), startDate: c.startDate ? c.startDate.substring(0, 10) : '', endDate: c.endDate ? c.endDate.substring(0, 10) : '' }); setShowForm(true); };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const payload = { ...formData, budget: Number(formData.budget) };
      if (editing) await marketingAPI.update(editing.id, payload); else await marketingAPI.create(payload);
      setShowForm(false); resetForm(); loadCampaigns();
    } catch (err) { setError(err instanceof Error ? err.message : 'Failed to save campaign'); }
  };

  const handleDelete = async (id: string) => { if (!confirm('Delete this campaign?')) return; try { await marketingAPI.delete(id); loadCampaigns(); } catch (err) { setError(err instanceof Error ? err.message : 'Failed to delete campaign'); } };
  const handleStatusChange = async (id: string, status: Status) => { try { await marketingAPI.setStatus(id, status); loadCampaigns(); } catch (err) { setError(err instanceof Error ? err.message : 'Failed to update status'); } };

  return (
    <div className="p-6" style={{ color: colors.text }}>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold" style={{ color: colors.text }}>Marketing Management</h1>
        <div className="flex gap-2">
          <button className="btn btn-secondary" onClick={toggleTheme}>{isDark ? '☀️ Light' : '🌙 Dark'}</button>
          <button className="btn btn-primary" onClick={openCreate}>+ New Campaign</button>
        </div>
      </div>

      {error && <div className="alert alert-error mb-4">{error}</div>}

      {showForm && (
        <div className="card mb-6" style={{ backgroundColor: colors.bgCard, border: `1px solid ${colors.border}` }}>
          <div className="card-header"><h2 style={{ color: colors.text }}>{editing ? 'Edit Campaign' : 'New Campaign'}</h2></div>
          <div className="card-body">
            <form onSubmit={handleSubmit}>
              <div className="flex gap-4" style={{ flexWrap: 'wrap' }}>
                <div className="form-group"><label className="form-label">Name</label><input className="form-input" value={formData.name} onChange={(e) => setFormData({ ...formData, name: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
                <div className="form-group"><label className="form-label">Channel</label><select className="form-select" value={formData.channel} onChange={(e) => setFormData({ ...formData, channel: e.target.value as Campaign['channel'] })} style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }}><option value="email">Email</option><option value="push">Push</option><option value="sms">SMS</option><option value="in_app">In-App</option><option value="social">Social</option></select></div>
                <div className="form-group"><label className="form-label">Audience</label><input className="form-input" value={formData.audience} onChange={(e) => setFormData({ ...formData, audience: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
                <div className="form-group"><label className="form-label">Budget</label><input className="form-input" type="number" step="0.01" value={formData.budget} onChange={(e) => setFormData({ ...formData, budget: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
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
          {loading ? (<div className="flex items-center justify-center p-8"><div className="loader"></div></div>) : campaigns.length === 0 ? (
            <div className="text-center py-8" style={{ color: colors.textSecondary }}>No campaigns found</div>
          ) : (
            <table className="table">
              <thead><tr><th>Name</th><th>Channel</th><th>Audience</th><th>Budget</th><th>Spent</th><th>Impr.</th><th>Clicks</th><th>Period</th><th>Status</th><th>Created</th><th>Actions</th></tr></thead>
              <tbody>
                {campaigns.map((c) => (
                  <tr key={c.id}>
                    <td style={{ color: colors.text }}>{c.name}</td>
                    <td style={{ color: colors.textSecondary }}>{c.channel}</td>
                    <td style={{ color: colors.textSecondary }}>{c.audience}</td>
                    <td style={{ color: colors.textSecondary }}>${c.budget}</td>
                    <td style={{ color: colors.textSecondary }}>${c.spent}</td>
                    <td style={{ color: colors.textSecondary }}>{c.impressions}</td>
                    <td style={{ color: colors.textSecondary }}>{c.clicks}</td>
                    <td style={{ color: colors.textSecondary }}>{c.startDate ? new Date(c.startDate).toLocaleDateString() : '-'} → {c.endDate ? new Date(c.endDate).toLocaleDateString() : '-'}</td>
                    <td><span className={`badge ${statusBadgeClass(c.status)}`}>{c.status}</span></td>
                    <td style={{ color: colors.textSecondary }}>{new Date(c.createdAt).toLocaleDateString()}</td>
                    <td>
                      <div className="flex gap-2" style={{ flexWrap: 'wrap' }}>
                        <select className="form-select" style={{ width: 'auto' }} value={c.status} onChange={(e) => handleStatusChange(c.id, e.target.value as Status)}>{STATUS_OPTIONS.map((s) => <option key={s} value={s}>{s}</option>)}</select>
                        <button className="btn btn-sm btn-outline" onClick={() => openEdit(c)}>Edit</button>
                        <button className="btn btn-sm btn-danger" onClick={() => handleDelete(c.id)}>Delete</button>
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

export default MarketingPage;
