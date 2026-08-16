// Bots Page - Manage Bots (WL-Bots backend: POST/GET/DELETE /bots, start/stop/pause)
import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { api, Bot } from '../services/api';
import { useTheme } from '../contexts/ThemeContext';

const BOT_TYPES = [
  'market_maker', 'liquidity_provider', 'sniper', 'front_run', 'mev',
  'sandwich', 'flash_loan', 'cross_chain', 'perp_hedge', 'grid',
  'dca', 'momentum', 'mean_reversion', 'scalping', 'ai_trading',
  'signal', 'arbitrage', 'custom',
];

export default function Bots() {
  const { isDark } = useTheme();
  const [bots, setBots] = useState<Bot[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showCreate, setShowCreate] = useState(false);
  const [busyId, setBusyId] = useState<string | null>(null);
  const [formData, setFormData] = useState({
    name: '',
    bot_type: 'grid',
    exchange: 'binance',
    pair: 'BTC/USDT',
  });

  useEffect(() => { loadBots(); }, []);

  const loadBots = () => {
    setLoading(true);
    setError('');
    api.getBots().then(data => {
      setBots(data.bots || []);
      setLoading(false);
    }).catch(err => {
      setError(err.message || 'Failed to load bots');
      setLoading(false);
    });
  };

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    try {
      await api.createBot({
        name: formData.name,
        bot_type: formData.bot_type,
        exchange: formData.exchange,
        pair: formData.pair,
      });
      setFormData({ name: '', bot_type: 'grid', exchange: 'binance', pair: 'BTC/USDT' });
      setShowCreate(false);
      loadBots();
    } catch (err: any) {
      setError(err.message || 'Failed to create bot');
    }
  };

  const transition = async (id: string, action: 'start' | 'stop' | 'pause') => {
    setBusyId(id);
    setError('');
    try {
      await api[`${action}Bot`](id);
      loadBots();
    } catch (err: any) {
      setError(err.message || `Failed to ${action} bot`);
    } finally {
      setBusyId(null);
    }
  };

  const handleDelete = async (id: string) => {
    if (!window.confirm('Delete this bot? This cannot be undone.')) return;
    setBusyId(id);
    setError('');
    try {
      await api.deleteBot(id);
      loadBots();
    } catch (err: any) {
      setError(err.message || 'Failed to delete bot');
    } finally {
      setBusyId(null);
    }
  };

  return (
    <div className="bots-page">
      <header className="page-header">
        <h1>My Bots <span className="mode-pill">{isDark ? 'Dark' : 'Light'}</span></h1>
        <button className="btn-primary" onClick={() => setShowCreate(!showCreate)}>
          + Create Bot
        </button>
      </header>

      {error && <div className="error">{error}</div>}

      {showCreate && (
        <form className="create-form" onSubmit={handleCreate}>
          <div className="form-group">
            <label>Bot Name</label>
            <input placeholder="e.g. BTC Grid v2" value={formData.name}
              onChange={e => setFormData({ ...formData, name: e.target.value })} required />
          </div>
          <div className="form-group">
            <label>Bot Type</label>
            <select value={formData.bot_type}
              onChange={e => setFormData({ ...formData, bot_type: e.target.value })}>
              {BOT_TYPES.map(t => <option key={t} value={t}>{t}</option>)}
            </select>
          </div>
          <div className="form-group">
            <label>Exchange</label>
            <input placeholder="binance" value={formData.exchange}
              onChange={e => setFormData({ ...formData, exchange: e.target.value })} />
          </div>
          <div className="form-group">
            <label>Pair</label>
            <input placeholder="BTC/USDT" value={formData.pair}
              onChange={e => setFormData({ ...formData, pair: e.target.value })} />
          </div>
          <button type="submit">Create</button>
        </form>
      )}

      {loading ? (
        <p>Loading...</p>
      ) : bots.length === 0 ? (
        <p>No bots yet. Create one to start trading!</p>
      ) : (
        <div className="bots-grid">
          {bots.map(bot => (
            <div key={bot.id} className="bot-card">
              <h3><Link to={`/bots/${bot.id}`}>{bot.name}</Link></h3>
              <p>Type: <code>{bot.bot_type}</code></p>
              <p>Exchange: {bot.exchange || '—'}</p>
              <p>Pair: {bot.pair || '—'}</p>
              <p>Status: <span className={`status ${bot.status}`}>{bot.status}</span></p>
              <p className="bot-created">Created: {new Date(bot.created_at).toLocaleString()}</p>
              <div className="bot-actions">
                {bot.status === 'running' ? (
                  <button disabled={busyId === bot.id} onClick={() => transition(bot.id, 'stop')}>Stop</button>
                ) : (
                  <button disabled={busyId === bot.id} onClick={() => transition(bot.id, 'start')}>Start</button>
                )}
                {bot.status !== 'paused' && (
                  <button disabled={busyId === bot.id} onClick={() => transition(bot.id, 'pause')}>Pause</button>
                )}
                <button className="btn-danger" disabled={busyId === bot.id}
                  onClick={() => handleDelete(bot.id)}>Delete</button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
