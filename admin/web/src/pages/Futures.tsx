/**
 * TigerWallet Admin - Futures Management Page
 * CRUD + status control for futures contracts (mirrors /api/v1/futures)
 */

import React, { useState, useEffect } from 'react';
import { useTheme } from '../hooks/useTheme';
import { futuresAPI } from '../services/api';

interface FuturesContract {
  id: string;
  symbol: string;
  baseAsset: string;
  quoteAsset: string;
  leverage: number;
  marginRatio: number;
  maintenanceMargin: number;
  takerFee: number;
  makerFee: number;
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

export const FuturesPage: React.FC = () => {
  const { isDark, toggleTheme } = useTheme();
  const [contracts, setContracts] = useState<FuturesContract[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [editing, setEditing] = useState<FuturesContract | null>(null);
  const [formData, setFormData] = useState({
    symbol: '',
    baseAsset: '',
    quoteAsset: '',
    leverage: '10',
    marginRatio: '0.5',
    maintenanceMargin: '0.25',
    takerFee: '0.0005',
    makerFee: '0.0002',
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
      const res = await futuresAPI.getAll();
      setContracts(res.data || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load futures contracts');
    } finally {
      setLoading(false);
    }
  };

  const resetForm = () => {
    setFormData({
      symbol: '', baseAsset: '', quoteAsset: '', leverage: '10',
      marginRatio: '0.5', maintenanceMargin: '0.25', takerFee: '0.0005', makerFee: '0.0002',
    });
    setEditing(null);
  };

  const openCreate = () => { resetForm(); setShowForm(true); };

  const openEdit = (contract: FuturesContract) => {
    setEditing(contract);
    setFormData({
      symbol: contract.symbol,
      baseAsset: contract.baseAsset,
      quoteAsset: contract.quoteAsset,
      leverage: String(contract.leverage),
      marginRatio: String(contract.marginRatio),
      maintenanceMargin: String(contract.maintenanceMargin),
      takerFee: String(contract.takerFee),
      makerFee: String(contract.makerFee),
    });
    setShowForm(true);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const payload = {
        symbol: formData.symbol,
        baseAsset: formData.baseAsset,
        quoteAsset: formData.quoteAsset,
        leverage: Number(formData.leverage),
        marginRatio: Number(formData.marginRatio),
        maintenanceMargin: Number(formData.maintenanceMargin),
        takerFee: Number(formData.takerFee),
        makerFee: Number(formData.makerFee),
      };
      if (editing) {
        await futuresAPI.update(editing.id, payload);
      } else {
        await futuresAPI.create(payload);
      }
      setShowForm(false);
      resetForm();
      loadContracts();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save futures contract');
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm('Delete this futures contract?')) return;
    try {
      await futuresAPI.delete(id);
      loadContracts();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete contract');
    }
  };

  const handleStatusChange = async (id: string, status: Status) => {
    try {
      await futuresAPI.setStatus(id, status);
      loadContracts();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update status');
    }
  };

  return (
    <div className="p-6" style={{ color: colors.text }}>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold" style={{ color: colors.text }}>Futures Management</h1>
        <div className="flex gap-2">
          <button className="btn btn-secondary" onClick={toggleTheme}>
            {isDark ? '☀️ Light' : '🌙 Dark'}
          </button>
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
                <div className="form-group">
                  <label className="form-label">Symbol</label>
                  <input className="form-input" value={formData.symbol} onChange={(e) => setFormData({ ...formData, symbol: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                </div>
                <div className="form-group">
                  <label className="form-label">Base Asset</label>
                  <input className="form-input" value={formData.baseAsset} onChange={(e) => setFormData({ ...formData, baseAsset: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                </div>
                <div className="form-group">
                  <label className="form-label">Quote Asset</label>
                  <input className="form-input" value={formData.quoteAsset} onChange={(e) => setFormData({ ...formData, quoteAsset: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                </div>
                <div className="form-group">
                  <label className="form-label">Max Leverage</label>
                  <input className="form-input" type="number" step="1" value={formData.leverage} onChange={(e) => setFormData({ ...formData, leverage: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                </div>
                <div className="form-group">
                  <label className="form-label">Margin Ratio</label>
                  <input className="form-input" type="number" step="0.01" value={formData.marginRatio} onChange={(e) => setFormData({ ...formData, marginRatio: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                </div>
                <div className="form-group">
                  <label className="form-label">Maintenance Margin</label>
                  <input className="form-input" type="number" step="0.01" value={formData.maintenanceMargin} onChange={(e) => setFormData({ ...formData, maintenanceMargin: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                </div>
                <div className="form-group">
                  <label className="form-label">Taker Fee</label>
                  <input className="form-input" type="number" step="0.0001" value={formData.takerFee} onChange={(e) => setFormData({ ...formData, takerFee: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                </div>
                <div className="form-group">
                  <label className="form-label">Maker Fee</label>
                  <input className="form-input" type="number" step="0.0001" value={formData.makerFee} onChange={(e) => setFormData({ ...formData, makerFee: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
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
          ) : contracts.length === 0 ? (
            <div className="text-center py-8" style={{ color: colors.textSecondary }}>No futures contracts found</div>
          ) : (
            <table className="table">
              <thead>
                <tr>
                  <th>Symbol</th><th>Pair</th><th>Leverage</th><th>Margin Ratio</th><th> Maint. Margin</th><th>Taker Fee</th><th>Maker Fee</th><th>Status</th><th>Created</th><th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {contracts.map((c) => (
                  <tr key={c.id}>
                    <td style={{ color: colors.text }}>{c.symbol}</td>
                    <td style={{ color: colors.textSecondary }}>{c.baseAsset}/{c.quoteAsset}</td>
                    <td style={{ color: colors.textSecondary }}>{c.leverage}x</td>
                    <td style={{ color: colors.textSecondary }}>{c.marginRatio}</td>
                    <td style={{ color: colors.textSecondary }}>{c.maintenanceMargin}</td>
                    <td style={{ color: colors.textSecondary }}>{c.takerFee}</td>
                    <td style={{ color: colors.textSecondary }}>{c.makerFee}</td>
                    <td><span className={`badge ${statusBadgeClass(c.status)}`}>{c.status}</span></td>
                    <td style={{ color: colors.textSecondary }}>{new Date(c.createdAt).toLocaleDateString()}</td>
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

export default FuturesPage;
