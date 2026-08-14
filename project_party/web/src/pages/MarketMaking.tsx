// Market Making Page - ProjectParty
import React, { useState, useEffect, useCallback } from 'react';
import { api } from '../services/api';

const ORDER_STATUSES = ['pending', 'filled', 'cancelled'];

export default function MarketMaking() {
  const [orders, setOrders] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [tokenFilter, setTokenFilter] = useState('');

  const [status, setStatus] = useState<any>(null);
  const [statusToken, setStatusToken] = useState('');
  const [statusLoading, setStatusLoading] = useState(false);
  const [statusError, setStatusError] = useState('');

  const [showOrderForm, setShowOrderForm] = useState(false);
  const [showLiqForm, setShowLiqForm] = useState<'add' | 'remove' | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [orderForm, setOrderForm] = useState({ token_id: '', side: 'buy', price: '', quantity: '' });
  const [liqForm, setLiqForm] = useState({ token_id: '', amount: '', quote_token: 'USDT', lp_amount: '' });
  const [msg, setMsg] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  const loadOrders = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const data = await api.getMakerOrders(tokenFilter || undefined);
      setOrders(data.orders || []);
    } catch (e: any) {
      setError(e.message || 'Failed to load orders');
    }
    setLoading(false);
  }, [tokenFilter]);

  useEffect(() => { loadOrders(); }, [loadOrders]);

  const createOrder = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setMsg(null);
    try {
      await api.createMakerOrder(orderForm);
      setMsg({ type: 'success', text: 'Maker order created.' });
      setOrderForm({ token_id: '', side: 'buy', price: '', quantity: '' });
      setShowOrderForm(false);
      loadOrders();
    } catch (err: any) {
      setMsg({ type: 'error', text: err.message || 'Failed to create order' });
    }
    setSubmitting(false);
  };

  const updateStatus = async (id: string, s: string) => {
    try {
      await api.updateOrderStatus(id, s);
      loadOrders();
    } catch (err: any) {
      setMsg({ type: 'error', text: err.message || 'Failed to update order' });
    }
  };

  const fetchStatus = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!statusToken) return;
    setStatusLoading(true);
    setStatusError('');
    setStatus(null);
    try {
      const data = await api.getMarketMakerStatus(statusToken);
      setStatus(data);
    } catch (err: any) {
      setStatusError(err.message || 'Failed to load status');
    }
    setStatusLoading(false);
  };

  const submitLiquidity = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setMsg(null);
    try {
      if (showLiqForm === 'add') {
        await api.addLiquidity({ token_id: liqForm.token_id, amount: liqForm.amount, quote_token: liqForm.quote_token });
      } else {
        await api.removeLiquidity({ token_id: liqForm.token_id, lp_amount: liqForm.lp_amount });
      }
      setMsg({ type: 'success', text: 'Liquidity operation successful.' });
      setLiqForm({ token_id: '', amount: '', quote_token: 'USDT', lp_amount: '' });
      setShowLiqForm(null);
    } catch (err: any) {
      setMsg({ type: 'error', text: err.message || 'Liquidity operation failed' });
    }
    setSubmitting(false);
  };

  return (
    <div className="page">
      <div className="page-header">
        <h1>Market Making</h1>
        <div className="row-actions">
          <button onClick={() => { setShowOrderForm(s => !s); setShowLiqForm(null); }}>{showOrderForm ? 'Close Order' : 'New Order'}</button>
          <button className="secondary" onClick={() => setShowLiqForm('add')}>Add Liquidity</button>
          <button className="secondary" onClick={() => setShowLiqForm('remove')}>Remove Liquidity</button>
        </div>
      </div>
      <p className="subtitle">Manage maker orders, monitor market-maker status, and add or remove liquidity.</p>

      {msg && <div className={`alert ${msg.type}`}>{msg.text}</div>}
      {error && <div className="alert error">{error}</div>}

      <div className="two-col">
        <section>
          <div className="section-title"><h2>Market-Maker Status</h2></div>
          <form onSubmit={fetchStatus} style={{ display: 'flex', gap: '0.5rem', marginBottom: '0.75rem' }}>
            <input value={statusToken} onChange={e => setStatusToken(e.target.value)} placeholder="Token ID (UUID)" />
            <button type="submit" disabled={statusLoading}>{statusLoading ? '...' : 'Check'}</button>
          </form>
          {statusError && <div className="alert error" style={{ marginBottom: 0 }}>{statusError}</div>}
          {status ? (
            <div className="stats-grid" style={{ marginBottom: 0 }}>
              <div className="stat-card"><h3>Active</h3><p>{status.active ? 'Yes' : 'No'}</p></div>
              <div className="stat-card"><h3>Total Orders</h3><p>{status.total_orders}</p></div>
              <div className="stat-card"><h3>Filled</h3><p>{status.filled_orders}</p></div>
              <div className="stat-card"><h3>Spread</h3><p>{Number(status.spread).toFixed(2)}%</p></div>
            </div>
          ) : !statusError && <p className="muted">Enter a token ID to view its market-maker status.</p>}
        </section>

        <section>
          <div className="section-title"><h2>Orders</h2></div>
          <input value={tokenFilter} onChange={e => setTokenFilter(e.target.value)} placeholder="Filter by token ID (UUID)" style={{ marginBottom: '0.75rem' }} />
          {loading ? <div className="state">Loading orders...</div> : orders.length === 0 ? (
            <div className="state">No data available</div>
          ) : (
            <table>
              <thead>
                <tr><th>Token</th><th>Side</th><th>Price</th><th>Qty</th><th>Remaining</th><th>Status</th><th>Action</th></tr>
              </thead>
              <tbody>
                {orders.map((o: any) => (
                  <tr key={o.id}>
                    <td title={o.token_id}>{String(o.token_id).slice(0, 8)}...</td>
                    <td className={o.side === 'buy' ? 'up' : 'down'}>{o.side}</td>
                    <td>{o.price}</td>
                    <td>{o.quantity}</td>
                    <td>{o.remaining}</td>
                    <td><span className={`badge ${o.status === 'filled' ? 'active' : ''}`}>{o.status}</span></td>
                    <td>
                      <select value={o.status} onChange={e => updateStatus(o.id, e.target.value)}>
                        {ORDER_STATUSES.map(s => <option key={s} value={s}>{s}</option>)}
                      </select>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </section>
      </div>

      {showOrderForm && (
        <section>
          <div className="section-title"><h2>Create Maker Order</h2></div>
          <form onSubmit={createOrder}>
            <div className="form-grid">
              <div className="form-field">
                <label>Token ID (UUID)</label>
                <input value={orderForm.token_id} onChange={e => setOrderForm({ ...orderForm, token_id: e.target.value })} required />
              </div>
              <div className="form-field">
                <label>Side</label>
                <select value={orderForm.side} onChange={e => setOrderForm({ ...orderForm, side: e.target.value })}>
                  <option value="buy">Buy</option>
                  <option value="sell">Sell</option>
                </select>
              </div>
              <div className="form-field">
                <label>Price</label>
                <input value={orderForm.price} onChange={e => setOrderForm({ ...orderForm, price: e.target.value })} required />
              </div>
              <div className="form-field">
                <label>Quantity</label>
                <input value={orderForm.quantity} onChange={e => setOrderForm({ ...orderForm, quantity: e.target.value })} required />
              </div>
            </div>
            <div className="form-actions">
              <button type="submit" disabled={submitting}>{submitting ? 'Creating...' : 'Create Order'}</button>
              <button type="button" className="secondary" onClick={() => setShowOrderForm(false)}>Cancel</button>
            </div>
          </form>
        </section>
      )}

      {showLiqForm && (
        <section>
          <div className="section-title"><h2>{showLiqForm === 'add' ? 'Add Liquidity' : 'Remove Liquidity'}</h2></div>
          <form onSubmit={submitLiquidity}>
            <div className="form-grid">
              <div className="form-field">
                <label>Token ID (UUID)</label>
                <input value={liqForm.token_id} onChange={e => setLiqForm({ ...liqForm, token_id: e.target.value })} required />
              </div>
              {showLiqForm === 'add' ? (
                <>
                  <div className="form-field">
                    <label>Amount</label>
                    <input value={liqForm.amount} onChange={e => setLiqForm({ ...liqForm, amount: e.target.value })} required />
                  </div>
                  <div className="form-field">
                    <label>Quote Token</label>
                    <input value={liqForm.quote_token} onChange={e => setLiqForm({ ...liqForm, quote_token: e.target.value })} placeholder="USDT" />
                  </div>
                </>
              ) : (
                <div className="form-field">
                  <label>LP Amount</label>
                  <input value={liqForm.lp_amount} onChange={e => setLiqForm({ ...liqForm, lp_amount: e.target.value })} required />
                </div>
              )}
            </div>
            <div className="form-actions">
              <button type="submit" disabled={submitting}>{submitting ? 'Processing...' : 'Submit'}</button>
              <button type="button" className="secondary" onClick={() => setShowLiqForm(null)}>Cancel</button>
            </div>
          </form>
        </section>
      )}
    </div>
  );
}
