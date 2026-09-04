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

  const [orders, setOrders] = useState<any[]>([]);
  const [trades, setTrades] = useState<any[]>([]);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const [cfgData, ordData, trdData] = await Promise.all([
        api.listMarketMakingConfigs(),
        api.listMakerOrders(),
        api.listMakerTrades()
      ]);
      setConfigs(cfgData.market_making_configs || []);
      setOrders(ordData.orders || []);
      setTrades(trdData.trades || []);
    } catch (e: any) {
      setError(e.message || 'Failed to load market-making data');
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

      <section>
        <div className="section-title"><h2>Open Orders ({orders.length})</h2></div>
        {orders.length === 0 ? (
          <div className="state">No open market-making orders.</div>
        ) : (
          <div className="coins-table">
            <table>
              <thead>
                <tr><th>ID</th><th>Token</th><th>Side</th><th>Price</th><th>Qty</th><th>Remaining</th><th>Status</th><th>Expires</th></tr>
              </thead>
              <tbody>
                {orders.map((o: any) => (
                  <tr key={o.id}>
                    <td title={o.id}>{String(o.id).slice(0, 8)}…</td>
                    <td title={o.token_id}>{String(o.token_id).slice(0, 8)}…</td>
                    <td><span className={`badge ${o.side === 'buy' ? 'active' : 'error'}`}>{o.side}</span></td>
                    <td>{o.price}</td>
                    <td>{o.quantity}</td>
                    <td>{o.remaining}</td>
                    <td>{o.status}</td>
                    <td>{o.expires_at ? new Date(o.expires_at).toLocaleString() : '-'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <section>
        <div className="section-title"><h2>Settled Trades ({trades.length})</h2></div>
        {trades.length === 0 ? (
          <div className="state">No settled trades yet — crossing buy/sell orders settle automatically.</div>
        ) : (
          <div className="coins-table">
            <table>
              <thead>
                <tr><th>ID</th><th>Token</th><th>Buy Order</th><th>Sell Order</th><th>Price</th><th>Quantity</th><th>Settled</th></tr>
              </thead>
              <tbody>
                {trades.map((t: any) => (
                  <tr key={t.id}>
                    <td title={t.id}>{String(t.id).slice(0, 8)}…</td>
                    <td title={t.token_id}>{String(t.token_id).slice(0, 8)}…</td>
                    <td title={t.buy_order_id}>{String(t.buy_order_id).slice(0, 8)}…</td>
                    <td title={t.sell_order_id}>{String(t.sell_order_id).slice(0, 8)}…</td>
                    <td>{t.price}</td>
                    <td>{t.quantity}</td>
                    <td>{t.created_at ? new Date(t.created_at).toLocaleString() : '-'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  );
}
