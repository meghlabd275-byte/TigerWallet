import React, { useEffect, useState, useCallback } from 'react';
import { Link } from 'react-router-dom';
import api from '../services/project-party-api';

export default function Market() {
  const [market, setMarket] = useState([]);
  const [trending, setTrending] = useState([]);
  const [featured, setFeatured] = useState([]);
  const [q, setQ] = useState('');
  const [results, setResults] = useState(null);
  const [error, setError] = useState(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(() => {
    setLoading(true); setError(null);
    Promise.all([
      api.getMarket().catch(e => { setError(e.message); return null; }),
      api.getTrending().catch(() => ({})),
      api.getFeatured().catch(() => ({})),
    ]).then(([m, t, f]) => {
      setMarket((m && (m.market || m.tokens || m.coins)) || []);
      setTrending((t && (t.trending || t.tokens)) || []);
      setFeatured((f && (f.featured || f.tokens)) || []);
    }).finally(() => setLoading(false));
  }, []);

  useEffect(load, [load]);

  const search = async (e) => {
    e.preventDefault();
    if (!q.trim()) { setResults(null); return; }
    setError(null);
    try {
      const r = await api.searchTokens(q.trim());
      setResults((r && (r.tokens || r.results)) || []);
    } catch (e2) {
      setError(e2.message);
    }
  };

  const renderTokens = (items, emptyMsg) => (
    items.length === 0 ? <p className="empty-state">{emptyMsg}</p> : (
      <table className="data-table">
        <thead><tr><th>Symbol</th><th>Name</th><th>Status</th><th>Chain</th></tr></thead>
        <tbody>
          {items.map(t => (
            <tr key={t.id || t.token_id || t.symbol}>
              <td><Link to={`/tokens/${t.id || t.token_id}`}>{t.symbol || '—'}</Link></td>
              <td>{t.name || '—'}</td>
              <td>{t.status || '—'}</td>
              <td>{t.chain || t.chain_id || '—'}</td>
            </tr>
          ))}
        </tbody>
      </table>
    )
  );

  if (loading) return <div>Loading…</div>;

  return (
    <div className="market-page">
      <h1>Market</h1>
      {error && <p className="error-banner">{error}</p>}

      <form className="create-form" onSubmit={search}>
        <label>Search tokens
          <input value={q} onChange={e => setQ(e.target.value)} placeholder="symbol or name" />
        </label>
        <button type="submit">Search</button>
        {results && <button type="button" onClick={() => { setResults(null); setQ(''); }}>Clear</button>}
      </form>

      {results && (<><h2>Search results</h2>{renderTokens(results, 'No tokens matched.')}</>)}

      <h2>Featured</h2>
      {renderTokens(featured, 'No featured tokens published.')}

      <h2>Trending</h2>
      {renderTokens(trending, 'No trending tokens published.')}

      <h2>All market tokens</h2>
      {renderTokens(market, 'No market data available.')}
    </div>
  );
}
