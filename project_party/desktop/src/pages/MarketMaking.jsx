import React, { useEffect, useState, useCallback } from 'react';
import api from '../services/project-party-api';

export default function MarketMaking() {
  const [orders, setOrders] = useState([]);
  const [trades, setTrades] = useState([]);
  const [tokens, setTokens] = useState([]);
  const [tokenId, setTokenId] = useState('');
  const [error, setError] = useState(null);
  const [notice, setNotice] = useState(null);
  const [loading, setLoading] = useState(true);
  const [form, setForm] = useState({ side: 'buy', price: '', quantity: '' });

  const load = useCallback(() => {
    setLoading(true); setError(null);
    Promise.all([
      api.getMakerOrders(tokenId || undefined).catch(e => { setError(e.message); return null; }),
      api.listTokens().catch(() => ({})),
    ]).then(([o, t]) => {
      setOrders((o && (o.orders || o.items)) || []);
      setTokens((t && t.tokens) || []);
    }).finally(() => setLoading(false));
    // Settled trades are only published by the canonical backend's matcher; the
    // WL deployment returns an empty list rather than an error.
    api.listMakerTrades(tokenId || undefined)
      .then(d => setTrades((d && (d.trades || d.items)) || []))
      .catch(() => setTrades([]));
  }, [tokenId]);

  useEffect(load, [load]);

  const createOrder = async (e) => {
    e.preventDefault();
    if (!tokenId) { setError('Select a token first'); return; }
    setError(null); setNotice(null);
    try {
      await api.createMakerOrders({
        token_id: tokenId,
        side: form.side,
        price: form.price,
        quantity: form.quantity,
      });
      setNotice('Order created');
      setForm({ side: 'buy', price: '', quantity: '' });
      load();
    } catch (e2) {
      setError(e2.message);
    }
  };

  const cancelOrder = async (id) => {
    setError(null); setNotice(null);
    try { await api.updateOrderStatus(id, 'cancelled'); setNotice('Order cancelled'); load(); }
    catch (e) { setError(e.message); }
  };

  return (
    <div className="market-making-page">
      <h1>Market Making</h1>
      {error && <p className="error-banner">{error}</p>}
      {notice && <p className="success-banner">{notice}</p>}

      <div className="filter-row">
        <label>Token
          <select value={tokenId} onChange={e => setTokenId(e.target.value)}>
            <option value="">All tokens</option>
            {tokens.map(t => <option key={t.id} value={t.id}>{t.symbol} — {t.name}</option>)}
          </select>
        </label>
      </div>

      <form className="create-form" onSubmit={createOrder}>
        <h2>Place an order</h2>
        <label>Side
          <select value={form.side} onChange={e => setForm({ ...form, side: e.target.value })}>
            <option value="buy">buy</option>
            <option value="sell">sell</option>
          </select>
        </label>
        <label>Price<input value={form.price} onChange={e => setForm({ ...form, price: e.target.value })} required /></label>
        <label>Quantity<input value={form.quantity} onChange={e => setForm({ ...form, quantity: e.target.value })} required /></label>
        <button type="submit">Create order</button>
      </form>

      <h2>Open orders</h2>
      {loading ? <p>Loading…</p> : orders.length === 0 ? (
        <p className="empty-state">No orders.</p>
      ) : (
        <table className="data-table">
          <thead><tr><th>Side</th><th>Price</th><th>Quantity</th><th>Status</th><th>Actions</th></tr></thead>
          <tbody>
            {orders.map(o => (
              <tr key={o.id}>
                <td>{o.side || '—'}</td>
                <td>{o.price != null ? o.price : '—'}</td>
                <td>{o.quantity != null ? o.quantity : '—'}</td>
                <td>{o.status || '—'}</td>
                <td className="action-row"><button onClick={() => cancelOrder(o.id)}>Cancel</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      <h2>Settled trades</h2>
      {trades.length === 0 ? <p className="empty-state">No settled trades.</p> : (
        <table className="data-table">
          <thead><tr><th>Time</th><th>Price</th><th>Quantity</th></tr></thead>
          <tbody>
            {trades.map((t, i) => (
              <tr key={t.id || i}>
                <td>{t.created_at || t.settled_at || '—'}</td>
                <td>{t.price != null ? t.price : '—'}</td>
                <td>{t.quantity != null ? t.quantity : '—'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
