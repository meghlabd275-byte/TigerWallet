// Pricing Page - ProjectParty
import React, { useState, useCallback } from 'react';
import { api } from '../services/api';

export default function Pricing() {
  const [tokenId, setTokenId] = useState('');
  const [price, setPrice] = useState<any>(null);
  const [history, setHistory] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const [newPrice, setNewPrice] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [msg, setMsg] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  const fetchAll = useCallback(async (id: string) => {
    if (!id) return;
    setLoading(true);
    setError('');
    setPrice(null);
    setHistory([]);
    try {
      const [p, h] = await Promise.all([
        api.getTokenPrice(id).catch(() => null),
        api.getPriceHistory(id).catch(() => ({ history: [] }))
      ]);
      if (p) setPrice(p);
      setHistory(h.history || []);
    } catch (e: any) {
      setError(e.message || 'Failed to load pricing data');
    }
    setLoading(false);
  }, []);

  const onLookup = (e: React.FormEvent) => {
    e.preventDefault();
    fetchAll(tokenId);
  };

  const setPriceValue = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setMsg(null);
    try {
      await api.setTokenPrice(tokenId, newPrice);
      setMsg({ type: 'success', text: 'Price set.' });
      setNewPrice('');
      fetchAll(tokenId);
    } catch (err: any) {
      setMsg({ type: 'error', text: err.message || 'Failed to set price' });
    }
    setSubmitting(false);
  };

  const updatePriceValue = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setMsg(null);
    try {
      await api.updateTokenPrice(tokenId, newPrice);
      setMsg({ type: 'success', text: 'Price updated.' });
      setNewPrice('');
      fetchAll(tokenId);
    } catch (err: any) {
      setMsg({ type: 'error', text: err.message || 'Failed to update price' });
    }
    setSubmitting(false);
  };

  return (
    <div className="page">
      <div className="page-header"><h1>Token Pricing</h1></div>
      <p className="subtitle">View, set, and update token prices with historical price history.</p>

      {msg && <div className={`alert ${msg.type}`}>{msg.text}</div>}
      {error && <div className="alert error">{error}</div>}

      <section>
        <div className="section-title"><h2>Lookup Token</h2></div>
        <form onSubmit={onLookup} style={{ display: 'flex', gap: '0.5rem' }}>
          <input value={tokenId} onChange={e => setTokenId(e.target.value)} placeholder="Token ID (UUID)" required />
          <button type="submit" disabled={loading}>{loading ? 'Loading...' : 'Lookup'}</button>
        </form>
      </section>

      {loading ? (
        <div className="state">Loading pricing data...</div>
      ) : tokenId ? (
        <>
          <div className="stats-grid">
            <div className="stat-card">
              <h3>Current Price</h3>
              <p>{price ? `$${price.price}` : 'N/A'}</p>
              {price && <div className="stat-sub">24h: <span className={price.change_24h >= 0 ? 'up' : 'down'}>{price.change_24h}%</span></div>}
            </div>
            <div className="stat-card">
              <h3>24h Volume</h3>
              <p>{price ? price.volume_24h : 'N/A'}</p>
            </div>
            <div className="stat-card">
              <h3>History Points</h3>
              <p>{history.length}</p>
            </div>
          </div>

          {tokenId && (
            <section>
              <div className="section-title"><h2>Set / Update Price</h2></div>
              <form onSubmit={setPriceValue}>
                <div style={{ display: 'flex', gap: '0.5rem', flexWrap: 'wrap' }}>
                  <input value={newPrice} onChange={e => setNewPrice(e.target.value)} placeholder="New price" required />
                  <div className="form-actions">
                    <button type="submit" disabled={submitting}>Set Price</button>
                    <button type="button" className="secondary" disabled={submitting || !newPrice} onClick={updatePriceValue}>Update Price</button>
                  </div>
                </div>
              </form>
            </section>
          )}

          <section>
            <div className="section-title"><h2>Price History</h2></div>
            {history.length === 0 ? (
              <div className="state">No data available</div>
            ) : (
              <PriceChart history={history} />
            )}
          </section>
        </>
      ) : (
        <div className="state">Enter a token ID to view pricing data.</div>
      )}
    </div>
  );
}

function PriceChart({ history }: { history: any[] }) {
  const prices = history.map((h: any) => parseFloat(h.price) || 0);
  const min = Math.min(...prices);
  const max = Math.max(...prices);
  const range = max - min || 1;
  const sorted = [...history].reverse();
  const W = 600;
  const H = 160;
  const pad = 10;
  const points = sorted.map((h: any, i: number) => {
    const x = pad + (i / Math.max(1, sorted.length - 1)) * (W - pad * 2);
    const y = H - pad - ((parseFloat(h.price) || 0) - min) / range * (H - pad * 2);
    return `${x},${y}`;
  }).join(' ');

  return (
    <div>
      <svg viewBox={`0 0 ${W} ${H}`} style={{ width: '100%', height: '180px' }}>
        <polyline points={points} fill="none" stroke="var(--accent)" strokeWidth="2" />
        {sorted.map((h: any, i: number) => {
          const x = pad + (i / Math.max(1, sorted.length - 1)) * (W - pad * 2);
          const y = H - pad - ((parseFloat(h.price) || 0) - min) / range * (H - pad * 2);
          return <circle key={i} cx={x} cy={y} r="3" fill="var(--accent)" />;
        })}
      </svg>
      <div className="chart-axis">
        <span>Low: {min}</span>
        <span>High: {max}</span>
      </div>
      <table style={{ marginTop: '0.75rem' }}>
        <thead>
          <tr><th>Time</th><th>Price</th><th>24h %</th><th>Volume</th></tr>
        </thead>
        <tbody>
          {history.slice(0, 20).map((h: any) => (
            <tr key={h.id}>
              <td>{new Date(h.timestamp).toLocaleString()}</td>
              <td>${h.price}</td>
              <td className={h.change_24h >= 0 ? 'up' : 'down'}>{h.change_24h}%</td>
              <td>{h.volume_24h}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
