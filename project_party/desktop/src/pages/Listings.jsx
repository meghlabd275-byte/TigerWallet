import React, { useEffect, useState, useCallback } from 'react';
import api from '../services/project-party-api';

export default function Listings() {
  const [listings, setListings] = useState([]);
  const [tokens, setTokens] = useState([]);
  const [error, setError] = useState(null);
  const [notice, setNotice] = useState(null);
  const [loading, setLoading] = useState(true);
  const [form, setForm] = useState({ token_id: '', pair: '', price: '', initial_liquidity: '' });

  const load = useCallback(() => {
    setLoading(true); setError(null);
    Promise.all([
      api.listListings().catch(e => { setError(e.message); return null; }),
      api.listTokens().catch(() => ({})),
    ]).then(([l, t]) => {
      setListings((l && l.listings) || []);
      setTokens((t && t.tokens) || []);
    }).finally(() => setLoading(false));
  }, []);

  useEffect(load, [load]);

  const create = async (e) => {
    e.preventDefault();
    setError(null); setNotice(null);
    try {
      await api.createListing({
        token_id: form.token_id,
        pair: form.pair || undefined,
        price: form.price || undefined,
        initial_liquidity: form.initial_liquidity || undefined,
      });
      setNotice('Listing created (pending review by the operator)');
      setForm({ token_id: '', pair: '', price: '', initial_liquidity: '' });
      load();
    } catch (e2) {
      setError(e2.message);
    }
  };

  return (
    <div className="listings-page">
      <h1>Listings</h1>
      {error && <p className="error-banner">{error}</p>}
      {notice && <p className="success-banner">{notice}</p>}

      <form className="create-form" onSubmit={create}>
        <h2>Request a listing</h2>
        <label>Token
          <select value={form.token_id} onChange={e => setForm({ ...form, token_id: e.target.value })} required>
            <option value="">Select a token…</option>
            {tokens.map(t => <option key={t.id} value={t.id}>{t.symbol} — {t.name}</option>)}
          </select>
        </label>
        <label>Pair (optional)<input value={form.pair} onChange={e => setForm({ ...form, pair: e.target.value })} placeholder="TOKEN/USDT" /></label>
        <label>Price (optional)<input value={form.price} onChange={e => setForm({ ...form, price: e.target.value })} /></label>
        <label>Initial liquidity (optional)<input value={form.initial_liquidity} onChange={e => setForm({ ...form, initial_liquidity: e.target.value })} /></label>
        <button type="submit">Create listing</button>
      </form>

      <h2>All listings</h2>
      {loading ? <p>Loading…</p> : listings.length === 0 ? (
        <p className="empty-state">No listings.</p>
      ) : (
        <table className="data-table">
          <thead><tr><th>Token</th><th>Pair</th><th>Status</th><th>Featured</th></tr></thead>
          <tbody>
            {listings.map(l => (
              <tr key={l.id}>
                <td>{l.token_symbol || l.token_id || '—'}</td>
                <td>{l.pair || '—'}</td>
                <td>{l.status || '—'}</td>
                <td>{l.is_featured ? 'yes' : 'no'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
