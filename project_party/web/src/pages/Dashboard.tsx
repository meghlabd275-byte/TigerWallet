// Dashboard - ProjectParty
import React, { useState, useEffect } from 'react';
import { api } from '../services/api';

export default function Dashboard() {
  const [market, setMarket] = useState<any>(null);
  const [featured, setFeatured] = useState<any[]>([]);
  const [trending, setTrending] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    Promise.all([
      api.getMarket(),
      api.getFeatured(),
      api.getTrending()
    ]).then(([m, f, t]) => {
      setMarket(m);
      setFeatured(f.featured || []);
      setTrending(t.trending || []);
      setLoading(false);
    }).catch(() => setLoading(false));
  }, []);

  if (loading) return <div>Loading...</div>;

  return (
    <div className="dashboard">
      <h1>Token Marketplace</h1>
      <div className="market-stats">
        <div className="stat-card">
          <h3>Total Market Cap</h3>
          <p>${market?.total_market_cap}</p>
        </div>
        <div className="stat-card">
          <h3>24h Volume</h3>
          <p>${market?.total_volume_24h}</p>
        </div>
        <div className="stat-card">
          <h3>BTC Dominance</h3>
          <p>{market?.bitcoin_dominance}%</p>
        </div>
      </div>

      <section>
        <h2>Featured Tokens</h2>
        <div className="tokens-grid">
          {featured.map((t: any) => (
            <div key={t.id} className="token-card">
              <img src={t.logo_url} alt={t.name} />
              <h3>{t.name}</h3>
              <p>{t.symbol}</p>
              <p>${t.price}</p>
            </div>
          ))}
        </div>
      </section>

      <section>
        <h2>Trending</h2>
        <div className="tokens-grid">
          {trending.map((t: any) => (
            <div key={t.id} className="token-card trending">
              <h3>{t.name}</h3>
              <p>{t.symbol}</p>
              <p className={t.price_change_24h >= 0 ? 'up' : 'down'}>
                {t.price_change_24h}%
              </p>
            </div>
          ))}
        </div>
      </section>
    </div>
  );
}
