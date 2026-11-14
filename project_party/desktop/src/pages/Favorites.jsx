import React, { useEffect, useState, useCallback } from 'react';
import api from '../services/project-party-api';

export default function Favorites() {
  const [favorites, setFavorites] = useState([]);
  const [tokens, setTokens] = useState([]);
  const [tokenId, setTokenId] = useState('');
  const [error, setError] = useState(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(() => {
    setLoading(true); setError(null);
    Promise.all([
      api.listFavorites().catch(e => { setError(e.message); return null; }),
      api.listTokens().catch(() => ({})),
    ]).then(([f, t]) => {
      setFavorites((f && (f.favorites || f.items)) || []);
      setTokens((t && t.tokens) || []);
    }).finally(() => setLoading(false));
  }, []);

  useEffect(load, [load]);

  const add = async (e) => {
    e.preventDefault();
    if (!tokenId) return;
    setError(null);
    try { await api.addFavorite({ token_id: tokenId }); setTokenId(''); load(); }
    catch (e2) { setError(e2.message); }
  };

  const remove = async (id) => {
    setError(null);
    try { await api.removeFavorite(id); load(); }
    catch (e) { setError(e.message); }
  };

  return (
    <div className="favorites-page">
      <h1>Favorites</h1>
      {error && <p className="error-banner">{error}</p>}

      <form className="create-form" onSubmit={add}>
        <label>Token
          <select value={tokenId} onChange={e => setTokenId(e.target.value)} required>
            <option value="">Select a token…</option>
            {tokens.map(t => <option key={t.id} value={t.id}>{t.symbol} — {t.name}</option>)}
          </select>
        </label>
        <button type="submit">Add favorite</button>
      </form>

      {loading ? <p>Loading…</p> : favorites.length === 0 ? (
        <p className="empty-state">No favorites yet.</p>
      ) : (
        <table className="data-table">
          <thead><tr><th>Token</th><th>Symbol</th><th>Actions</th></tr></thead>
          <tbody>
            {favorites.map(f => (
              <tr key={f.id}>
                <td>{f.token_name || f.token_id || '—'}</td>
                <td>{f.token_symbol || '—'}</td>
                <td className="action-row"><button onClick={() => remove(f.id)}>Remove</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
