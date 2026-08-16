/**
 * TigerWallet Admin - Bots Clients Management Page
 * CRUD + status control for bot clients (mirrors /api/v1/bots-clients)
 */

import React, { useState, useEffect } from 'react';
import { useTheme } from '../hooks/useTheme';
import { botsClientsAPI } from '../services/api';

interface BotsClient {
  id: string;
  client_id: string;
  bot_id: string;
  user_id: string;
  client_name: string;
  client_type: string;
  api_key_id: string;
  status: 'active' | 'paused' | 'stopped' | 'disconnected';
  connected_at: string;
  last_seen_at: string;
  allocated_usd: number;
  pnl_usd: number;
}

const STATUS_OPTIONS = ['active', 'paused', 'stopped', 'disconnected'] as const;
type Status = typeof STATUS_OPTIONS[number];

const statusBadgeClass = (status: string): string => {
  switch (status) {
    case 'active': return 'badge-success';
    case 'paused': return 'badge-warning';
    case 'stopped': return 'badge-neutral';
    case 'disconnected': return 'badge-error';
    default: return 'badge-neutral';
  }
};

export const BotsClientsPage: React.FC = () => {
  const { isDark, toggleTheme } = useTheme();
  const [clients, setClients] = useState<BotsClient[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [editing, setEditing] = useState<BotsClient | null>(null);
  const [formData, setFormData] = useState({
    client_id: '',
    bot_id: '',
    user_id: '',
    client_name: '',
    client_type: '',
    api_key_id: '',
    allocated_usd: '0',
  });

  const colors = {
    text: isDark ? '#f9fafb' : '#111827',
    textSecondary: isDark ? '#9ca3af' : '#6b7280',
    bgCard: isDark ? '#1e293b' : '#ffffff',
    border: isDark ? '#374151' : '#e5e7eb',
  };

  useEffect(() => { loadClients(); }, []);

  const loadClients = async () => {
    try {
      setLoading(true);
      setError(null);
      const res = await botsClientsAPI.getAll();
      setClients(res.data || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load bot clients');
    } finally {
      setLoading(false);
    }
  };

  const resetForm = () => {
    setFormData({
      client_id: '', bot_id: '', user_id: '', client_name: '',
      client_type: '', api_key_id: '', allocated_usd: '0',
    });
    setEditing(null);
  };

  const openCreate = () => { resetForm(); setShowForm(true); };

  const openEdit = (client: BotsClient) => {
    setEditing(client);
    setFormData({
      client_id: client.client_id,
      bot_id: client.bot_id,
      user_id: client.user_id,
      client_name: client.client_name,
      client_type: client.client_type,
      api_key_id: client.api_key_id,
      allocated_usd: String(client.allocated_usd),
    });
    setShowForm(true);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const payload = {
        client_id: formData.client_id,
        bot_id: formData.bot_id,
        user_id: formData.user_id,
        client_name: formData.client_name,
        client_type: formData.client_type,
        api_key_id: formData.api_key_id,
        allocated_usd: Number(formData.allocated_usd),
      };
      if (editing) {
        await botsClientsAPI.update(editing.id, payload);
      } else {
        await botsClientsAPI.create(payload);
      }
      setShowForm(false);
      resetForm();
      loadClients();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save bot client');
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm('Delete this bot client?')) return;
    try {
      await botsClientsAPI.delete(id);
      loadClients();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete bot client');
    }
  };

  const handleStatusChange = async (id: string, status: Status) => {
    try {
      await botsClientsAPI.setStatus(id, status);
      loadClients();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update status');
    }
  };

  return (
    <div className="p-6" style={{ color: colors.text }}>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold" style={{ color: colors.text }}>Bots Clients Management</h1>
        <div className="flex gap-2">
          <button className="btn btn-secondary" onClick={toggleTheme}>
            {isDark ? '☀️ Light' : '🌙 Dark'}
          </button>
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
                <div className="form-group">
                  <label className="form-label">Client ID</label>
                  <input className="form-input" value={formData.client_id} onChange={(e) => setFormData({ ...formData, client_id: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                </div>
                <div className="form-group">
                  <label className="form-label">Bot ID</label>
                  <input className="form-input" value={formData.bot_id} onChange={(e) => setFormData({ ...formData, bot_id: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                </div>
                <div className="form-group">
                  <label className="form-label">User ID</label>
                  <input className="form-input" value={formData.user_id} onChange={(e) => setFormData({ ...formData, user_id: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                </div>
                <div className="form-group">
                  <label className="form-label">Client Name</label>
                  <input className="form-input" value={formData.client_name} onChange={(e) => setFormData({ ...formData, client_name: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                </div>
                <div className="form-group">
                  <label className="form-label">Client Type</label>
                  <input className="form-input" value={formData.client_type} onChange={(e) => setFormData({ ...formData, client_type: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                </div>
                <div className="form-group">
                  <label className="form-label">API Key ID</label>
                  <input className="form-input" value={formData.api_key_id} onChange={(e) => setFormData({ ...formData, api_key_id: e.target.value })} style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                </div>
                <div className="form-group">
                  <label className="form-label">Allocated (USD)</label>
                  <input className="form-input" type="number" step="0.01" value={formData.allocated_usd} onChange={(e) => setFormData({ ...formData, allocated_usd: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                </div>
              </div>
              <div className="flex gap-2 mt-4">
                <button type="submit" className="btn btn-primary">{editing ? 'Update' : 'Create'}</button>
                <button type="button" className="btn btn-secondary" onClick={() => { setShowForm(false); resetForm(); }}>Cancel</button>
              </div>
            </form>
          </div>
        </div>
      )}

      <div className="card" style={{ backgroundColor: colors.bgCard }}>
        <div className="card-body p-0">
          {loading ? (
            <div className="flex items-center justify-center p-8"><div className="loader"></div></div>
          ) : clients.length === 0 ? (
            <div className="text-center py-8" style={{ color: colors.textSecondary }}>No bot clients found</div>
          ) : (
            <table className="table">
              <thead>
                <tr>
                  <th>Client Name</th><th>Client ID</th><th>Bot ID</th><th>User ID</th><th>Type</th><th>API Key</th><th>Allocated</th><th>PNL</th><th>Connected</th><th>Last Seen</th><th>Status</th><th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {clients.map((c) => (
                  <tr key={c.id}>
                    <td style={{ color: colors.text }}>{c.client_name}</td>
                    <td style={{ color: colors.textSecondary }}>{c.client_id}</td>
                    <td style={{ color: colors.textSecondary }}>{c.bot_id}</td>
                    <td style={{ color: colors.textSecondary }}>{c.user_id}</td>
                    <td style={{ color: colors.textSecondary }}>{c.client_type}</td>
                    <td style={{ color: colors.textSecondary }}>{c.api_key_id}</td>
                    <td style={{ color: colors.textSecondary }}>${c.allocated_usd}</td>
                    <td style={{ color: colors.textSecondary }}>${c.pnl_usd}</td>
                    <td style={{ color: colors.textSecondary }}>{new Date(c.connected_at).toLocaleDateString()}</td>
                    <td style={{ color: colors.textSecondary }}>{new Date(c.last_seen_at).toLocaleDateString()}</td>
                    <td><span className={`badge ${statusBadgeClass(c.status)}`}>{c.status}</span></td>
                    <td>
                      <div className="flex gap-2" style={{ flexWrap: 'wrap' }}>
                        <select className="form-select" style={{ width: 'auto' }} value={c.status} onChange={(e) => handleStatusChange(c.id, e.target.value as Status)}>
                          {STATUS_OPTIONS.map((s) => <option key={s} value={s}>{s}</option>)}
                        </select>
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

export default BotsClientsPage;
