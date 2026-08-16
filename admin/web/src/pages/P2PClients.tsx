/**
 * TigerWallet Admin - P2P Clients Management Page
 * CRUD + status control for P2P clients (mirrors /api/v1/p2p-clients)
 */

import React, { useState, useEffect } from 'react';
import { useTheme } from '../hooks/useTheme';
import { p2pClientsAPI } from '../services/api';

interface P2PClient {
  id: string;
  userId: string;
  userName: string;
  email: string;
  country: string;
  verified: boolean;
  totalTrades: number;
  rating: number;
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

export const P2PClientsPage: React.FC = () => {
  const { isDark, toggleTheme } = useTheme();
  const [clients, setClients] = useState<P2PClient[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [editing, setEditing] = useState<P2PClient | null>(null);
  const [formData, setFormData] = useState({ userId: '', userName: '', email: '', country: '', verified: 'false' });

  const colors = {
    text: isDark ? '#f9fafb' : '#111827',
    textSecondary: isDark ? '#9ca3af' : '#6b7280',
    bgCard: isDark ? '#1e293b' : '#ffffff',
    border: isDark ? '#374151' : '#e5e7eb',
  };

  useEffect(() => { loadClients(); }, []);

  const loadClients = async () => {
    try { setLoading(true); setError(null); const res = await p2pClientsAPI.getAll(); setClients(res.data || []); } catch (err) { setError(err instanceof Error ? err.message : 'Failed to load P2P clients'); } finally { setLoading(false); }
  };

  const resetForm = () => { setFormData({ userId: '', userName: '', email: '', country: '', verified: 'false' }); setEditing(null); };
  const openCreate = () => { resetForm(); setShowForm(true); };
  const openEdit = (c: P2PClient) => { setEditing(c); setFormData({ userId: c.userId, userName: c.userName, email: c.email, country: c.country, verified: String(c.verified) }); setShowForm(true); };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const payload = { ...formData, verified: formData.verified === 'true' };
      if (editing) await p2pClientsAPI.update(editing.id, payload); else await p2pClientsAPI.create(payload);
      setShowForm(false); resetForm(); loadClients();
    } catch (err) { setError(err instanceof Error ? err.message : 'Failed to save client'); }
  };

  const handleDelete = async (id: string) => { if (!confirm('Delete this P2P client?')) return; try { await p2pClientsAPI.delete(id); loadClients(); } catch (err) { setError(err instanceof Error ? err.message : 'Failed to delete client'); } };
  const handleStatusChange = async (id: string, status: Status) => { try { await p2pClientsAPI.setStatus(id, status); loadClients(); } catch (err) { setError(err instanceof Error ? err.message : 'Failed to update status'); } };

  return (
    <div className="p-6" style={{ color: colors.text }}>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold" style={{ color: colors.text }}>P2P Clients Management</h1>
        <div className="flex gap-2">
          <button className="btn btn-secondary" onClick={toggleTheme}>{isDark ? '☀️ Light' : '🌙 Dark'}</button>
          <button className="btn btn-primary" onClick={openCreate}>+ New Client</button>
        </div>
      </div>

      {error && <div className="alert alert-error mb-4">{error}</div>}

      {showForm && (
        <div className="card mb-6" style={{ backgroundColor: colors.bgCard, border: `1px solid ${colors.border}` }}>
          <div className="card-header"><h2 style={{ color: colors.text }}>{editing ? 'Edit Client' : 'New Client'}</h2></div>
          <div className="card-body">
            <form onSubmit={handleSubmit}>
              <div className="flex gap-4" style={{ flexWrap: 'wrap' }}>
                <div className="form-group"><label className="form-label">User ID</label><input className="form-input" value={formData.userId} onChange={(e) => setFormData({ ...formData, userId: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
                <div className="form-group"><label className="form-label">User Name</label><input className="form-input" value={formData.userName} onChange={(e) => setFormData({ ...formData, userName: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
                <div className="form-group"><label className="form-label">Email</label><input className="form-input" type="email" value={formData.email} onChange={(e) => setFormData({ ...formData, email: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
                <div className="form-group"><label className="form-label">Country</label><input className="form-input" value={formData.country} onChange={(e) => setFormData({ ...formData, country: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
                <div className="form-group"><label className="form-label">Verified</label><select className="form-select" value={formData.verified} onChange={(e) => setFormData({ ...formData, verified: e.target.value })} style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }}><option value="false">No</option><option value="true">Yes</option></select></div>
              </div>
              <div className="flex gap-2 mt-4"><button type="submit" className="btn btn-primary">{editing ? 'Update' : 'Create'}</button><button type="button" className="btn btn-secondary" onClick={() => { setShowForm(false); resetForm(); }}>Cancel</button></div>
            </form>
          </div>
        </div>
      )}

      <div className="card" style={{ backgroundColor: colors.bgCard }}>
        <div className="card-body p-0">
          {loading ? (<div className="flex items-center justify-center p-8"><div className="loader"></div></div>) : clients.length === 0 ? (
            <div className="text-center py-8" style={{ color: colors.textSecondary }}>No P2P clients found</div>
          ) : (
            <table className="table">
              <thead><tr><th>User</th><th>Email</th><th>Country</th><th>Verified</th><th>Trades</th><th>Rating</th><th>Status</th><th>Created</th><th>Actions</th></tr></thead>
              <tbody>
                {clients.map((c) => (
                  <tr key={c.id}>
                    <td style={{ color: colors.text }}>{c.userName}</td>
                    <td style={{ color: colors.textSecondary }}>{c.email}</td>
                    <td style={{ color: colors.textSecondary }}>{c.country}</td>
                    <td style={{ color: colors.textSecondary }}>{c.verified ? '✓' : '✗'}</td>
                    <td style={{ color: colors.textSecondary }}>{c.totalTrades}</td>
                    <td style={{ color: colors.textSecondary }}>{c.rating}</td>
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

export default P2PClientsPage;
