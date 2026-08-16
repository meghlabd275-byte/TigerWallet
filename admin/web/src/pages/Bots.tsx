/**
 * TigerWallet Admin - Bots Management Page
 * CRUD + status control for trading bots (mirrors /api/v1/bots)
 * Also manages bot tiers via botsAPI.getTiers/createTier/updateTier/deleteTier
 */

import React, { useState, useEffect } from 'react';
import { useTheme } from '../hooks/useTheme';
import { botsAPI } from '../services/api';

interface Bot {
  id: string;
  bot_id: string;
  name: string;
  owner_id: string;
  bot_type: string;
  strategy: string;
  chain_id: string;
  status: 'active' | 'paused' | 'stopped' | 'error';
  tier: string;
  exchange: string;
  pair: string;
  allocated_usd: number;
  pnl_usd: number;
  win_rate: number;
  total_trades: number;
}

interface BotTier {
  id: string;
  name: string;
  description: string;
  max_allocation: number;
  features: string;
}

const STATUS_OPTIONS = ['active', 'paused', 'stopped', 'error'] as const;
type Status = typeof STATUS_OPTIONS[number];

const statusBadgeClass = (status: string): string => {
  switch (status) {
    case 'active': return 'badge-success';
    case 'paused': return 'badge-warning';
    case 'stopped': return 'badge-neutral';
    case 'error': return 'badge-error';
    default: return 'badge-neutral';
  }
};

export const BotsPage: React.FC = () => {
  const { isDark, toggleTheme } = useTheme();
  const [bots, setBots] = useState<Bot[]>([]);
  const [tiers, setTiers] = useState<BotTier[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [editing, setEditing] = useState<Bot | null>(null);
  const [showTiers, setShowTiers] = useState(false);
  const [showTierForm, setShowTierForm] = useState(false);
  const [editingTier, setEditingTier] = useState<BotTier | null>(null);
  const [formData, setFormData] = useState({
    bot_id: '',
    name: '',
    owner_id: '',
    bot_type: '',
    strategy: '',
    chain_id: '',
    tier: '',
    exchange: '',
    pair: '',
    allocated_usd: '0',
  });
  const [tierFormData, setTierFormData] = useState({
    name: '',
    description: '',
    max_allocation: '0',
    features: '',
  });

  const colors = {
    text: isDark ? '#f9fafb' : '#111827',
    textSecondary: isDark ? '#9ca3af' : '#6b7280',
    bgCard: isDark ? '#1e293b' : '#ffffff',
    border: isDark ? '#374151' : '#e5e7eb',
  };

  useEffect(() => { loadBots(); loadTiers(); }, []);

  const loadBots = async () => {
    try {
      setLoading(true);
      setError(null);
      const res = await botsAPI.getAll();
      setBots(res.data || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load bots');
    } finally {
      setLoading(false);
    }
  };

  const loadTiers = async () => {
    try {
      const res = await botsAPI.getTiers();
      setTiers(res.data || []);
    } catch (err) {
      // tiers load silently; main error surface remains the bots list
    }
  };

  const resetForm = () => {
    setFormData({
      bot_id: '', name: '', owner_id: '', bot_type: '', strategy: '',
      chain_id: '', tier: '', exchange: '', pair: '', allocated_usd: '0',
    });
    setEditing(null);
  };

  const openCreate = () => { resetForm(); setShowForm(true); };

  const openEdit = (bot: Bot) => {
    setEditing(bot);
    setFormData({
      bot_id: bot.bot_id,
      name: bot.name,
      owner_id: bot.owner_id,
      bot_type: bot.bot_type,
      strategy: bot.strategy,
      chain_id: bot.chain_id,
      tier: bot.tier,
      exchange: bot.exchange,
      pair: bot.pair,
      allocated_usd: String(bot.allocated_usd),
    });
    setShowForm(true);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const payload = {
        bot_id: formData.bot_id,
        name: formData.name,
        owner_id: formData.owner_id,
        bot_type: formData.bot_type,
        strategy: formData.strategy,
        chain_id: formData.chain_id,
        tier: formData.tier,
        exchange: formData.exchange,
        pair: formData.pair,
        allocated_usd: Number(formData.allocated_usd),
      };
      if (editing) {
        await botsAPI.update(editing.id, payload);
      } else {
        await botsAPI.create(payload);
      }
      setShowForm(false);
      resetForm();
      loadBots();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save bot');
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm('Delete this bot?')) return;
    try {
      await botsAPI.delete(id);
      loadBots();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete bot');
    }
  };

  const handleStatusChange = async (id: string, status: Status) => {
    try {
      await botsAPI.setStatus(id, status);
      loadBots();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update status');
    }
  };

  // ===== Tier management =====
  const resetTierForm = () => {
    setTierFormData({ name: '', description: '', max_allocation: '0', features: '' });
    setEditingTier(null);
  };

  const openCreateTier = () => { resetTierForm(); setShowTierForm(true); };

  const openEditTier = (tier: BotTier) => {
    setEditingTier(tier);
    setTierFormData({
      name: tier.name,
      description: tier.description,
      max_allocation: String(tier.max_allocation),
      features: tier.features,
    });
    setShowTierForm(true);
  };

  const handleTierSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const payload = {
        name: tierFormData.name,
        description: tierFormData.description,
        max_allocation: Number(tierFormData.max_allocation),
        features: tierFormData.features,
      };
      if (editingTier) {
        await botsAPI.updateTier(editingTier.id, payload);
      } else {
        await botsAPI.createTier(payload);
      }
      setShowTierForm(false);
      resetTierForm();
      loadTiers();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save tier');
    }
  };

  const handleTierDelete = async (id: string) => {
    if (!confirm('Delete this tier?')) return;
    try {
      await botsAPI.deleteTier(id);
      loadTiers();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete tier');
    }
  };

  return (
    <div className="p-6" style={{ color: colors.text }}>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold" style={{ color: colors.text }}>Bots Management</h1>
        <div className="flex gap-2">
          <button className="btn btn-secondary" onClick={toggleTheme}>
            {isDark ? '☀️ Light' : '🌙 Dark'}
          </button>
          <button className="btn btn-secondary" onClick={() => setShowTiers(!showTiers)}>
            {showTiers ? 'Back to Bots' : 'Manage Tiers'}
          </button>
          {!showTiers && <button className="btn btn-primary" onClick={openCreate}>+ New Bot</button>}
          {showTiers && <button className="btn btn-primary" onClick={openCreateTier}>+ New Tier</button>}
        </div>
      </div>

      {error && <div className="alert alert-error mb-4">{error}</div>}

      {showTiers ? (
        <div>
          {showTierForm && (
            <div className="card mb-6" style={{ backgroundColor: colors.bgCard, border: `1px solid ${colors.border}` }}>
              <div className="card-header"><h2 style={{ color: colors.text }}>{editingTier ? 'Edit Tier' : 'New Tier'}</h2></div>
              <div className="card-body">
                <form onSubmit={handleTierSubmit}>
                  <div className="flex gap-4" style={{ flexWrap: 'wrap' }}>
                    <div className="form-group">
                      <label className="form-label">Name</label>
                      <input className="form-input" value={tierFormData.name} onChange={(e) => setTierFormData({ ...tierFormData, name: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                    </div>
                    <div className="form-group">
                      <label className="form-label">Max Allocation (USD)</label>
                      <input className="form-input" type="number" step="1" value={tierFormData.max_allocation} onChange={(e) => setTierFormData({ ...tierFormData, max_allocation: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                    </div>
                    <div className="form-group">
                      <label className="form-label">Features</label>
                      <input className="form-input" value={tierFormData.features} onChange={(e) => setTierFormData({ ...tierFormData, features: e.target.value })} style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                    </div>
                    <div className="form-group" style={{ flex: 1, minWidth: '300px' }}>
                      <label className="form-label">Description</label>
                      <input className="form-input" value={tierFormData.description} onChange={(e) => setTierFormData({ ...tierFormData, description: e.target.value })} style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                    </div>
                  </div>
                  <div className="flex gap-2 mt-4">
                    <button type="submit" className="btn btn-primary">{editingTier ? 'Update' : 'Create'}</button>
                    <button type="button" className="btn btn-secondary" onClick={() => { setShowTierForm(false); resetTierForm(); }}>Cancel</button>
                  </div>
                </form>
              </div>
            </div>
          )}

          <div className="card" style={{ backgroundColor: colors.bgCard }}>
            <div className="card-body p-0">
              {tiers.length === 0 ? (
                <div className="text-center py-8" style={{ color: colors.textSecondary }}>No bot tiers found</div>
              ) : (
                <table className="table">
                  <thead>
                    <tr>
                      <th>Name</th><th>Description</th><th>Max Allocation</th><th>Features</th><th>Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {tiers.map((t) => (
                      <tr key={t.id}>
                        <td style={{ color: colors.text }}>{t.name}</td>
                        <td style={{ color: colors.textSecondary }}>{t.description}</td>
                        <td style={{ color: colors.textSecondary }}>${t.max_allocation}</td>
                        <td style={{ color: colors.textSecondary }}>{t.features}</td>
                        <td>
                          <div className="flex gap-2">
                            <button className="btn btn-sm btn-outline" onClick={() => openEditTier(t)}>Edit</button>
                            <button className="btn btn-sm btn-danger" onClick={() => handleTierDelete(t.id)}>Delete</button>
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
      ) : (
        <div>
          {showForm && (
            <div className="card mb-6" style={{ backgroundColor: colors.bgCard, border: `1px solid ${colors.border}` }}>
              <div className="card-header"><h2 style={{ color: colors.text }}>{editing ? 'Edit Bot' : 'New Bot'}</h2></div>
              <div className="card-body">
                <form onSubmit={handleSubmit}>
                  <div className="flex gap-4" style={{ flexWrap: 'wrap' }}>
                    <div className="form-group">
                      <label className="form-label">Bot ID</label>
                      <input className="form-input" value={formData.bot_id} onChange={(e) => setFormData({ ...formData, bot_id: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                    </div>
                    <div className="form-group">
                      <label className="form-label">Name</label>
                      <input className="form-input" value={formData.name} onChange={(e) => setFormData({ ...formData, name: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                    </div>
                    <div className="form-group">
                      <label className="form-label">Owner ID</label>
                      <input className="form-input" value={formData.owner_id} onChange={(e) => setFormData({ ...formData, owner_id: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                    </div>
                    <div className="form-group">
                      <label className="form-label">Bot Type</label>
                      <input className="form-input" value={formData.bot_type} onChange={(e) => setFormData({ ...formData, bot_type: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                    </div>
                    <div className="form-group">
                      <label className="form-label">Strategy</label>
                      <input className="form-input" value={formData.strategy} onChange={(e) => setFormData({ ...formData, strategy: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                    </div>
                    <div className="form-group">
                      <label className="form-label">Chain ID</label>
                      <input className="form-input" value={formData.chain_id} onChange={(e) => setFormData({ ...formData, chain_id: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                    </div>
                    <div className="form-group">
                      <label className="form-label">Tier</label>
                      <input className="form-input" value={formData.tier} onChange={(e) => setFormData({ ...formData, tier: e.target.value })} style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                    </div>
                    <div className="form-group">
                      <label className="form-label">Exchange</label>
                      <input className="form-input" value={formData.exchange} onChange={(e) => setFormData({ ...formData, exchange: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                    </div>
                    <div className="form-group">
                      <label className="form-label">Pair</label>
                      <input className="form-input" value={formData.pair} onChange={(e) => setFormData({ ...formData, pair: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
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
              ) : bots.length === 0 ? (
                <div className="text-center py-8" style={{ color: colors.textSecondary }}>No bots found</div>
              ) : (
                <table className="table">
                  <thead>
                    <tr>
                      <th>Name</th><th>Bot ID</th><th>Type</th><th>Strategy</th><th>Chain</th><th>Tier</th><th>Exchange</th><th>Pair</th><th>Allocated</th><th>PNL</th><th>Win Rate</th><th>Trades</th><th>Status</th><th>Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {bots.map((b) => (
                      <tr key={b.id}>
                        <td style={{ color: colors.text }}>{b.name}</td>
                        <td style={{ color: colors.textSecondary }}>{b.bot_id}</td>
                        <td style={{ color: colors.textSecondary }}>{b.bot_type}</td>
                        <td style={{ color: colors.textSecondary }}>{b.strategy}</td>
                        <td style={{ color: colors.textSecondary }}>{b.chain_id}</td>
                        <td style={{ color: colors.textSecondary }}>{b.tier}</td>
                        <td style={{ color: colors.textSecondary }}>{b.exchange}</td>
                        <td style={{ color: colors.textSecondary }}>{b.pair}</td>
                        <td style={{ color: colors.textSecondary }}>${b.allocated_usd}</td>
                        <td style={{ color: colors.textSecondary }}>${b.pnl_usd}</td>
                        <td style={{ color: colors.textSecondary }}>{b.win_rate}%</td>
                        <td style={{ color: colors.textSecondary }}>{b.total_trades}</td>
                        <td><span className={`badge ${statusBadgeClass(b.status)}`}>{b.status}</span></td>
                        <td>
                          <div className="flex gap-2" style={{ flexWrap: 'wrap' }}>
                            <select className="form-select" style={{ width: 'auto' }} value={b.status} onChange={(e) => handleStatusChange(b.id, e.target.value as Status)}>
                              {STATUS_OPTIONS.map((s) => <option key={s} value={s}>{s}</option>)}
                            </select>
                            <button className="btn btn-sm btn-outline" onClick={() => openEdit(b)}>Edit</button>
                            <button className="btn btn-sm btn-danger" onClick={() => handleDelete(b.id)}>Delete</button>
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
      )}
    </div>
  );
};

export default BotsPage;
