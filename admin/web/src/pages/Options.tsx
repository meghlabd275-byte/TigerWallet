/**
 * TigerWallet Admin - Options Management Page
 * CRUD + status control for options contracts (mirrors /api/v1/options)
 */

import React, { useState, useEffect } from 'react';
import { useTheme } from '../hooks/useTheme';
import { optionsAPI } from '../services/api';

interface OptionsContract {
  id: string;
  symbol: string;
  underlying: string;
  strikePrice: number;
  premium: number;
  expiry: string;
  optionType: 'call' | 'put';
  settlement: 'physical' | 'cash';
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

export const OptionsPage: React.FC = () => {
  const { isDark, toggleTheme } = useTheme();
  const [contracts, setContracts] = useState<OptionsContract[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [editing, setEditing] = useState<OptionsContract | null>(null);
  const [formData, setFormData] = useState({
    symbol: '', underlying: '', strikePrice: '', premium: '', expiry: '',
    optionType: 'call', settlement: 'cash',
  });

  const colors = {
    text: isDark ? '#f9fafb' : '#111827',
    textSecondary: isDark ? '#9ca3af' : '#6b7280',
    bgCard: isDark ? '#1e293b' : '#ffffff',
    border: isDark ? '#374151' : '#e5e7eb',
  };

  useEffect(() => { loadContracts(); }, []);

  const loadContracts = async () => {
    try {
      setLoading(true);
      setError(null);
      const res = await optionsAPI.getAll();
      setContracts(res.data || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load options contracts');
    } finally {
      setLoading(false);
    }
  };

  const resetForm = () => {
    setFormData({ symbol: '', underlying: '', strikePrice: '', premium: '', expiry: '', optionType: 'call', settlement: 'cash' });
    setEditing(null);
  };

  const openCreate = () => { resetForm(); setShowForm(true); };

  const openEdit = (c: OptionsContract) => {
    setEditing(c);
    setFormData({
      symbol: c.symbol, underlying: c.underlying, strikePrice: String(c.strikePrice),
      premium: String(c.premium), expiry: c.expiry ? c.expiry.substring(0, 10) : '',
      optionType: c.optionType, settlement: c.settlement,
    });
    setShowForm(true);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const payload = {
        symbol: formData.symbol, underlying: formData.underlying,
        strikePrice: Number(formData.strikePrice), premium: Number(formData.premium),
        expiry: formData.expiry, optionType: formData.optionType, settlement: formData.settlement,
      };
      if (editing) await optionsAPI.update(editing.id, payload); else await optionsAPI.create(payload);
      setShowForm(false); resetForm(); loadContracts();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save options contract');
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm('Delete this options contract?')) return;
    try { await optionsAPI.delete(id); loadContracts(); } catch (err) { setError(err instanceof Error ? err.message : 'Failed to delete contract'); }
  };

  const handleStatusChange = async (id: string, status: Status) => {
    try { await optionsAPI.setStatus(id, status); loadContracts(); } catch (err) { setError(err instanceof Error ? err.message : 'Failed to update status'); }
  };

  return (
    <div className="p-6" style={{ color: colors.text }}>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold" style={{ color: colors.text }}>Options Management</h1>
        <div className="flex gap-2">
          <button className="btn btn-secondary" onClick={toggleTheme}>{isDark ? '☀️ Light' : '🌙 Dark'}</button>
          <button className="btn btn-primary" onClick={openCreate}>+ New Contract</button>
        </div>
      </div>

      {error && <div className="alert alert-error mb-4">{error}</div>}

      {showForm && (
        <div className="card mb-6" style={{ backgroundColor: colors.bgCard, border: `1px solid ${colors.border}` }}>
          <div className="card-header"><h2 style={{ color: colors.text }}>{editing ? 'Edit Contract' : 'New Contract'}</h2></div>
          <div className="card-body">
            <form onSubmit={handleSubmit}>
              <div className="flex gap-4" style={{ flexWrap: 'wrap' }}>
                <div className="form-group"><label className="form-label">Symbol</label><input className="form-input" value={formData.symbol} onChange={(e) => setFormData({ ...formData, symbol: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
                <div className="form-group"><label className="form-label">Underlying</label><input className="form-input" value={formData.underlying} onChange={(e) => setFormData({ ...formData, underlying: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
                <div className="form-group"><label className="form-label">Strike Price</label><input className="form-input" type="number" step="0.0001" value={formData.strikePrice} onChange={(e) => setFormData({ ...formData, strikePrice: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
                <div className="form-group"><label className="form-label">Premium</label><input className="form-input" type="number" step="0.0001" value={formData.premium} onChange={(e) => setFormData({ ...formData, premium: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
                <div className="form-group"><label className="form-label">Expiry</label><input className="form-input" type="date" value={formData.expiry} onChange={(e) => setFormData({ ...formData, expiry: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
                <div className="form-group"><label className="form-label">Option Type</label><select className="form-select" value={formData.optionType} onChange={(e) => setFormData({ ...formData, optionType: e.target.value as 'call' | 'put' })} style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }}><option value="call">Call</option><option value="put">Put</option></select></div>
                <div className="form-group"><label className="form-label">Settlement</label><select className="form-select" value={formData.settlement} onChange={(e) => setFormData({ ...formData, settlement: e.target.value as 'physical' | 'cash' })} style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }}><option value="cash">Cash</option><option value="physical">Physical</option></select></div>
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
          ) : contracts.length === 0 ? (
            <div className="text-center py-8" style={{ color: colors.textSecondary }}>No options contracts found</div>
          ) : (
            <table className="table">
              <thead>
                <tr><th>Symbol</th><th>Underlying</th><th>Strike</th><th>Premium</th><th>Expiry</th><th>Type</th><th>Settlement</th><th>Status</th><th>Created</th><th>Actions</th></tr>
              </thead>
              <tbody>
                {contracts.map((c) => (
                  <tr key={c.id}>
                    <td style={{ color: colors.text }}>{c.symbol}</td>
                    <td style={{ color: colors.textSecondary }}>{c.underlying}</td>
                    <td style={{ color: colors.textSecondary }}>${c.strikePrice}</td>
                    <td style={{ color: colors.textSecondary }}>${c.premium}</td>
                    <td style={{ color: colors.textSecondary }}>{c.expiry ? new Date(c.expiry).toLocaleDateString() : '-'}</td>
                    <td style={{ color: colors.textSecondary }}>{c.optionType}</td>
                    <td style={{ color: colors.textSecondary }}>{c.settlement}</td>
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

export default OptionsPage;
