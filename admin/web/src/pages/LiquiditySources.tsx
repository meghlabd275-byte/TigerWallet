/**
 * TigerWallet Admin - Liquidity Sources Management Page
 * CRUD + status control for liquidity sources (mirrors /api/v1/liquidity-sources)
 * Stats via getStats, health-check button via healthCheck
 */

import React, { useState, useEffect } from 'react';
import { useTheme } from '../hooks/useTheme';
import { liquiditySourcesAPI } from '../services/api';

interface LiquiditySource {
  id: string;
  source_id: string;
  name: string;
  source_type: 'dex' | 'cex' | 'aggregator' | 'market_maker';
  chain_id: string;
  router_address: string;
  api_endpoint: string;
  status: 'active' | 'paused' | 'stopped';
  priority: number;
  fee_bps: number;
  slippage_bps: number;
  max_capacity: number;
  current_liquidity: number;
  health_status: string;
}

const SOURCE_TYPE_OPTIONS = ['dex', 'cex', 'aggregator', 'market_maker'] as const;
type SourceType = typeof SOURCE_TYPE_OPTIONS[number];

const STATUS_OPTIONS = ['active', 'paused', 'stopped'] as const;
type Status = typeof STATUS_OPTIONS[number];

const statusBadgeClass = (status: string): string => {
  switch (status) {
    case 'active': return 'badge-success';
    case 'paused': return 'badge-warning';
    case 'stopped': return 'badge-neutral';
    default: return 'badge-neutral';
  }
};

const healthBadgeClass = (health: string): string => {
  switch (health) {
    case 'healthy': return 'badge-success';
    case 'degraded': return 'badge-warning';
    case 'unhealthy': return 'badge-error';
    default: return 'badge-neutral';
  }
};

export const LiquiditySourcesPage: React.FC = () => {
  const { isDark, toggleTheme } = useTheme();
  const [sources, setSources] = useState<LiquiditySource[]>([]);
  const [stats, setStats] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [editing, setEditing] = useState<LiquiditySource | null>(null);
  const [formData, setFormData] = useState({
    source_id: '',
    name: '',
    source_type: 'dex' as SourceType,
    chain_id: '',
    router_address: '',
    api_endpoint: '',
    priority: '1',
    fee_bps: '0',
    slippage_bps: '0',
    max_capacity: '0',
  });

  const colors = {
    text: isDark ? '#f9fafb' : '#111827',
    textSecondary: isDark ? '#9ca3af' : '#6b7280',
    bgCard: isDark ? '#1e293b' : '#ffffff',
    border: isDark ? '#374151' : '#e5e7eb',
  };

  useEffect(() => { loadSources(); loadStats(); }, []);

  const loadSources = async () => {
    try {
      setLoading(true);
      setError(null);
      const res = await liquiditySourcesAPI.getAll();
      setSources(res.data || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load liquidity sources');
    } finally {
      setLoading(false);
    }
  };

  const loadStats = async () => {
    try {
      const res = await liquiditySourcesAPI.getStats();
      setStats(res.data || null);
    } catch (err) {
      // stats load silently
    }
  };

  const resetForm = () => {
    setFormData({
      source_id: '', name: '', source_type: 'dex', chain_id: '',
      router_address: '', api_endpoint: '', priority: '1',
      fee_bps: '0', slippage_bps: '0', max_capacity: '0',
    });
    setEditing(null);
  };

  const openCreate = () => { resetForm(); setShowForm(true); };

  const openEdit = (source: LiquiditySource) => {
    setEditing(source);
    setFormData({
      source_id: source.source_id,
      name: source.name,
      source_type: source.source_type,
      chain_id: source.chain_id,
      router_address: source.router_address,
      api_endpoint: source.api_endpoint,
      priority: String(source.priority),
      fee_bps: String(source.fee_bps),
      slippage_bps: String(source.slippage_bps),
      max_capacity: String(source.max_capacity),
    });
    setShowForm(true);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const payload = {
        source_id: formData.source_id,
        name: formData.name,
        source_type: formData.source_type,
        chain_id: formData.chain_id,
        router_address: formData.router_address,
        api_endpoint: formData.api_endpoint,
        priority: Number(formData.priority),
        fee_bps: Number(formData.fee_bps),
        slippage_bps: Number(formData.slippage_bps),
        max_capacity: Number(formData.max_capacity),
      };
      if (editing) {
        await liquiditySourcesAPI.update(editing.id, payload);
      } else {
        await liquiditySourcesAPI.create(payload);
      }
      setShowForm(false);
      resetForm();
      loadSources();
      loadStats();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save liquidity source');
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm('Delete this liquidity source?')) return;
    try {
      await liquiditySourcesAPI.delete(id);
      loadSources();
      loadStats();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete liquidity source');
    }
  };

  const handleStatusChange = async (id: string, status: Status) => {
    try {
      await liquiditySourcesAPI.setStatus(id, status);
      loadSources();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update status');
    }
  };

  const handleHealthCheck = async (source: LiquiditySource) => {
    try {
      await liquiditySourcesAPI.healthCheck(source.id, {});
      loadSources();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Health check failed');
    }
  };

  return (
    <div className="p-6" style={{ color: colors.text }}>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold" style={{ color: colors.text }}>Liquidity Sources Management</h1>
        <div className="flex gap-2">
          <button className="btn btn-secondary" onClick={toggleTheme}>
            {isDark ? '☀️ Light' : '🌙 Dark'}
          </button>
          <button className="btn btn-primary" onClick={openCreate}>+ New Source</button>
        </div>
      </div>

      {error && <div className="alert alert-error mb-4">{error}</div>}

      {stats && (
        <div className="grid grid-cols-4 gap-4 mb-6" style={{ flexWrap: 'wrap' }}>
          <div className="card" style={{ backgroundColor: colors.bgCard, padding: '1rem' }}>
            <div className="text-2xl font-bold" style={{ color: colors.text }}>{stats.totalSources ?? 0}</div>
            <div style={{ color: colors.textSecondary }}>Total Sources</div>
          </div>
          <div className="card" style={{ backgroundColor: colors.bgCard, padding: '1rem' }}>
            <div className="text-2xl font-bold" style={{ color: colors.text }}>{stats.activeSources ?? 0}</div>
            <div style={{ color: colors.textSecondary }}>Active Sources</div>
          </div>
          <div className="card" style={{ backgroundColor: colors.bgCard, padding: '1rem' }}>
            <div className="text-2xl font-bold" style={{ color: colors.text }}>${stats.totalLiquidity ?? 0}</div>
            <div style={{ color: colors.textSecondary }}>Total Liquidity</div>
          </div>
          <div className="card" style={{ backgroundColor: colors.bgCard, padding: '1rem' }}>
            <div className="text-2xl font-bold" style={{ color: colors.text }}>${stats.totalCapacity ?? 0}</div>
            <div style={{ color: colors.textSecondary }}>Total Capacity</div>
          </div>
        </div>
      )}

      {showForm && (
        <div className="card mb-6" style={{ backgroundColor: colors.bgCard, border: `1px solid ${colors.border}` }}>
          <div className="card-header"><h2 style={{ color: colors.text }}>{editing ? 'Edit Source' : 'New Source'}</h2></div>
          <div className="card-body">
            <form onSubmit={handleSubmit}>
              <div className="flex gap-4" style={{ flexWrap: 'wrap' }}>
                <div className="form-group">
                  <label className="form-label">Source ID</label>
                  <input className="form-input" value={formData.source_id} onChange={(e) => setFormData({ ...formData, source_id: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                </div>
                <div className="form-group">
                  <label className="form-label">Name</label>
                  <input className="form-input" value={formData.name} onChange={(e) => setFormData({ ...formData, name: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                </div>
                <div className="form-group">
                  <label className="form-label">Source Type</label>
                  <select className="form-select" value={formData.source_type} onChange={(e) => setFormData({ ...formData, source_type: e.target.value as SourceType })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }}>
                    {SOURCE_TYPE_OPTIONS.map((t) => <option key={t} value={t}>{t}</option>)}
                  </select>
                </div>
                <div className="form-group">
                  <label className="form-label">Chain ID</label>
                  <input className="form-input" value={formData.chain_id} onChange={(e) => setFormData({ ...formData, chain_id: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                </div>
                <div className="form-group">
                  <label className="form-label">Router Address</label>
                  <input className="form-input" value={formData.router_address} onChange={(e) => setFormData({ ...formData, router_address: e.target.value })} style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                </div>
                <div className="form-group">
                  <label className="form-label">API Endpoint</label>
                  <input className="form-input" value={formData.api_endpoint} onChange={(e) => setFormData({ ...formData, api_endpoint: e.target.value })} style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                </div>
                <div className="form-group">
                  <label className="form-label">Priority</label>
                  <input className="form-input" type="number" step="1" value={formData.priority} onChange={(e) => setFormData({ ...formData, priority: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                </div>
                <div className="form-group">
                  <label className="form-label">Fee (bps)</label>
                  <input className="form-input" type="number" step="0.01" value={formData.fee_bps} onChange={(e) => setFormData({ ...formData, fee_bps: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                </div>
                <div className="form-group">
                  <label className="form-label">Slippage (bps)</label>
                  <input className="form-input" type="number" step="0.01" value={formData.slippage_bps} onChange={(e) => setFormData({ ...formData, slippage_bps: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                </div>
                <div className="form-group">
                  <label className="form-label">Max Capacity</label>
                  <input className="form-input" type="number" step="0.01" value={formData.max_capacity} onChange={(e) => setFormData({ ...formData, max_capacity: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
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
          ) : sources.length === 0 ? (
            <div className="text-center py-8" style={{ color: colors.textSecondary }}>No liquidity sources found</div>
          ) : (
            <table className="table">
              <thead>
                <tr>
                  <th>Name</th><th>Source ID</th><th>Type</th><th>Chain</th><th>Router</th><th>Endpoint</th><th>Priority</th><th>Fee (bps)</th><th>Slippage (bps)</th><th>Max Capacity</th><th>Current Liquidity</th><th>Health</th><th>Status</th><th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {sources.map((s) => (
                  <tr key={s.id}>
                    <td style={{ color: colors.text }}>{s.name}</td>
                    <td style={{ color: colors.textSecondary }}>{s.source_id}</td>
                    <td style={{ color: colors.textSecondary }}>{s.source_type}</td>
                    <td style={{ color: colors.textSecondary }}>{s.chain_id}</td>
                    <td style={{ color: colors.textSecondary }}>{s.router_address}</td>
                    <td style={{ color: colors.textSecondary }}>{s.api_endpoint}</td>
                    <td style={{ color: colors.textSecondary }}>{s.priority}</td>
                    <td style={{ color: colors.textSecondary }}>{s.fee_bps}</td>
                    <td style={{ color: colors.textSecondary }}>{s.slippage_bps}</td>
                    <td style={{ color: colors.textSecondary }}>${s.max_capacity}</td>
                    <td style={{ color: colors.textSecondary }}>${s.current_liquidity}</td>
                    <td><span className={`badge ${healthBadgeClass(s.health_status)}`}>{s.health_status}</span></td>
                    <td><span className={`badge ${statusBadgeClass(s.status)}`}>{s.status}</span></td>
                    <td>
                      <div className="flex gap-2" style={{ flexWrap: 'wrap' }}>
                        <select className="form-select" style={{ width: 'auto' }} value={s.status} onChange={(e) => handleStatusChange(s.id, e.target.value as Status)}>
                          {STATUS_OPTIONS.map((st) => <option key={st} value={st}>{st}</option>)}
                        </select>
                        <button className="btn btn-sm btn-outline" onClick={() => handleHealthCheck(s)}>Health Check</button>
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

export default LiquiditySourcesPage;
