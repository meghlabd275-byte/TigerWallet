// Market Making Page — WL-ProjectParty. Real backend coverage:
// POST /market-making (create config), GET /market-making (list configs).
import React, { useState, useEffect, useCallback } from 'react';
import { api } from '../services/api';

interface MMForm {
  token_id: string; pair: string; spread: string; enabled: boolean;
}

const EMPTY: MMForm = { token_id: '', pair: '', spread: '', enabled: true };

const PAIRS = ['USDT', 'USDC', 'ETH', 'BNB', 'BTC'];

export default function MarketMaking() {
  const [configs, setConfigs] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState<MMForm>(EMPTY);
  const [submitting, setSubmitting] = useState(false);
  const [msg, setMsg] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const data = await api.listMarketMakingConfigs();
      setConfigs(data.market_making_configs || []);
    } catch (e: any) {
      setError(e.message || 'Failed to load market-making configs');
    }
    setLoading(false);
  }, []);

  useEffect(() => { load(); }, [load]);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setMsg(null);
    try {
      await api.createMarketMakingConfig({
        token_id: form.token_id,
        pair: form.pair,
        spread: form.spread || undefined,
        enabled: form.enabled
      });
      setMsg({ type: 'success', text: 'Market-making config created.' });
      setForm(EMPTY);
      setShowForm(false);
      load();
    } catch (err: any) {
      setMsg({ type: 'error', text: err.message || 'Failed to create config' });
    }
    setSubmitting(false);
  };

  return (
    <div className="page">
      <div className="page-header">
        <h1>Market Making</h1>
        <button onClick={() => setShowForm(s => !s)}>{showForm ? 'Close' : 'Create Config'}</button>
      </div>
      <p className="subtitle">Create and view market-making configs (spread + enable toggle per token/pair).</p>

      {msg && <div className={`alert ${msg.type}`}>{msg.text}</div>}
      {error && <div className="alert error">{error}</div>}

      {showForm && (
        <section>
          <div className="section-title"><h2>Create Market-Making Config</h2></div>
          <form onSubmit={submit}>
            <div className="form-grid">
              <div className="form-field">
                <label>Token ID (UUID)</label>
                <input value={form.token_id} onChange={e => setForm({ ...form, token_id: e.target.value })} required />
              </div>
              <div className="form-field">
                <label>Pair</label>
                <select value={form.pair} onChange={e => setForm({ ...form, pair: e.target.value })} required>
                  <option value="">Select pair…</option>
                  {PAIRS.map(p => <option key={p} value={p}>{p}</option>)}
                </select>
              </div>
              <div className="form-field">
                <label>Spread</label>
                <input value={form.spread} onChange={e => setForm({ ...form, spread: e.target.value })} placeholder="e.g. 0.0025" />
              </div>
              <div className="form-field">
                <label>Enabled</label>
                <label className="checkbox-row">
                  <input type="checkbox" checked={form.enabled} onChange={e => setForm({ ...form, enabled: e.target.checked })} />
                  {form.enabled ? 'Active' : 'Disabled'}
                </label>
              </div>
            </div>
            <div className="form-actions">
              <button type="submit" disabled={submitting}>{submitting ? 'Creating…' : 'Create'}</button>
              <button type="button" className="secondary" onClick={() => setShowForm(false)}>Cancel</button>
            </div>
          </form>
        </section>
      )}

      {loading ? (
        <div className="state">Loading market-making configs…</div>
      ) : configs.length === 0 ? (
        <div className="state">No market-making configs yet.</div>
      ) : (
        <section>
          <div className="section-title"><h2>Configs ({configs.length})</h2></div>
          <div className="coins-table">
            <table>
              <thead>
                <tr><th>ID</th><th>Token</th><th>Pair</th><th>Spread</th><th>Enabled</th><th>Created</th></tr>
              </thead>
              <tbody>
                {configs.map((m: any) => (
                  <tr key={m.id}>
                    <td title={m.id}>{String(m.id).slice(0, 8)}…</td>
                    <td title={m.token_id}>{String(m.token_id).slice(0, 8)}…</td>
                    <td>{m.pair}</td>
                    <td>{m.spread || '-'}</td>
                    <td><span className={`badge ${m.enabled ? 'active' : 'error'}`}>{m.enabled ? 'on' : 'off'}</span></td>
                    <td>{m.created_at ? new Date(m.created_at).toLocaleString() : '-'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      )}
    </div>
  );
}
