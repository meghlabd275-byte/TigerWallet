import React, { useEffect, useState, useCallback } from 'react';
import { Link } from 'react-router-dom';
import api from '../services/bot-api';

const BOT_TYPES = [
  'market_maker', 'arbitrage', 'sniper', 'grid', 'dca', 'momentum',
  'mean_reversion', 'scalping', 'ai_trading', 'signal', 'cross_chain',
  'perp_hedge', 'flash_loan', 'sandwich', 'front_run', 'mev',
  'liquidity_provider', 'custom'
];

export default function Bots() {
  const [bots, setBots] = useState([]);
  const [error, setError] = useState(null);
  const [notice, setNotice] = useState(null);
  const [loading, setLoading] = useState(true);
  const [form, setForm] = useState({ name: '', bot_type: 'market_maker', exchange: '', pair: '' });
  const [busy, setBusy] = useState(false);

  const load = useCallback(() => {
    setLoading(true);
    api.listBots()
      .then((d) => setBots((d && d.bots) || []))
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  }, []);

  useEffect(load, [load]);

  const act = async (fn, label) => {
    setError(null); setNotice(null);
    try {
      await fn();
      setNotice(label);
      load();
    } catch (e) {
      // startBot fails closed for bot types with no execution runner; the
      // backend message names the executable types.
      setError(e.message);
    }
  };

  const create = async (e) => {
    e.preventDefault();
    setBusy(true); setError(null); setNotice(null);
    try {
      await api.createBot({
        name: form.name,
        bot_type: form.bot_type,
        exchange: form.exchange || undefined,
        pair: form.pair || undefined,
        config: {}
      });
      setForm({ name: '', bot_type: 'market_maker', exchange: '', pair: '' });
      setNotice('Bot created');
      load();
    } catch (e2) {
      setError(e2.message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="bots-page">
      <h1>Trading Bots</h1>
      {error && <p className="error-banner">{error}</p>}
      {notice && <p className="success-banner">{notice}</p>}

      <form className="create-form" onSubmit={create}>
        <h2>Create bot</h2>
        <label>Name
          <input value={form.name} onChange={e => setForm({ ...form, name: e.target.value })} required />
        </label>
        <label>Type
          <select value={form.bot_type} onChange={e => setForm({ ...form, bot_type: e.target.value })}>
            {BOT_TYPES.map(t => <option key={t} value={t}>{t}</option>)}
          </select>
        </label>
        <label>Exchange (optional)
          <input value={form.exchange} onChange={e => setForm({ ...form, exchange: e.target.value })} />
        </label>
        <label>Pair (optional)
          <input value={form.pair} onChange={e => setForm({ ...form, pair: e.target.value })} />
        </label>
        <button type="submit" disabled={busy}>{busy ? 'Creating…' : 'Create'}</button>
      </form>

      <h2>Your bots</h2>
      {loading ? <p>Loading…</p> : bots.length === 0 ? (
        <p className="empty-state">No bots yet.</p>
      ) : (
        <table className="data-table">
          <thead>
            <tr><th>Name</th><th>Type</th><th>Status</th><th>Pair</th><th>Actions</th></tr>
          </thead>
          <tbody>
            {bots.map(b => (
              <tr key={b.id}>
                <td><Link to={`/bots/${b.id}`}>{b.name}</Link></td>
                <td>{b.bot_type}</td>
                <td>{b.status}</td>
                <td>{b.pair || '—'}</td>
                <td className="action-row">
                  <button onClick={() => act(() => api.startBot(b.id), 'Start requested')}>Start</button>
                  <button onClick={() => act(() => api.pauseBot(b.id), 'Paused')}>Pause</button>
                  <button onClick={() => act(() => api.stopBot(b.id), 'Stopped')}>Stop</button>
                  <button onClick={() => act(() => api.deleteBot(b.id), 'Deleted')}>Delete</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
