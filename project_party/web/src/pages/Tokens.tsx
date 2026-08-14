// Tokens Page - ProjectParty
import React, { useState, useEffect } from 'react';
import { api } from '../services/api';

export default function Tokens() {
  const [tokens, setTokens] = useState<any[]>([]);
  const [search, setSearch] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSearch = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!search) return;
    setLoading(true);
    try {
      const data = await api.getTokens(search);
      setTokens(data.tokens || []);
    } catch (err) {}
    setLoading(false);
  };

  return (
    <div className="tokens-page">
      <h1>Search Tokens</h1>
      <form onSubmit={handleSearch}>
        <input 
          value={search} 
          onChange={e => setSearch(e.target.value)} 
          placeholder="Search by name or symbol..." 
        />
        <button type="submit">Search</button>
      </form>

      {loading ? <p>Searching...</p> : (
        <div className="tokens-grid">
          {tokens.map((t: any) => (
            <div key={t.id} className="token-card">
              <img src={t.logo_url} alt={t.name} />
              <h3>{t.name}</h3>
              <p>{t.symbol}</p>
              <p>${t.price}</p>
              {t.verified && <span className="verified">✓</span>}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
