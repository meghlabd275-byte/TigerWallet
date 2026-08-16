// Dashboard — WL-ProjectParty. Derives real metrics from the backend:
// GET /health, GET /tokens, GET /launchpad, GET /listings.
import React, { useState, useEffect } from 'react';
import { api } from '../services/api';

export default function Dashboard() {
  const [health, setHealth] = useState<any>(null);
  const [tokens, setTokens] = useState<any[]>([]);
  const [launchpads, setLaunchpads] = useState<any[]>([]);
  const [listings, setListings] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    let mounted = true;
    (async () => {
      setLoading(true);
      setError('');
      const results = await Promise.allSettled([
        api.getHealth(),
        api.listTokens(),
        api.listLaunchpadProjects(),
        api.listListings()
      ]);
      if (!mounted) return;
      if (results[0].status === 'fulfilled') setHealth(results[0].value);
      if (results[1].status === 'fulfilled') setTokens((results[1].value as any).tokens || []);
      else setError((results[1].reason as Error)?.message || 'Failed to load tokens');
      if (results[2].status === 'fulfilled') setLaunchpads((results[2].value as any).launchpad_projects || []);
      if (results[3].status === 'fulfilled') setListings((results[3].value as any).listings || []);
      setLoading(false);
    })();
    return () => { mounted = false; };
  }, []);

  const activeTokens = tokens.filter((t: any) => t.status === 'active');
  const activeLaunchpads = launchpads.filter((p: any) => p.status === 'active');
  const activeListings = listings.filter((l: any) => l.status === 'active');

  if (loading) return <div className="state">Loading dashboard…</div>;

  return (
    <div className="page dashboard">
      <div className="page-header">
        <h1>Dashboard</h1>
        <span className={`badge ${health?.status === 'ok' ? 'active' : 'error'}`}>
          Backend: {health?.status || 'unreachable'}
        </span>
      </div>
      <p className="subtitle">Live overview of the WL-ProjectParty backend data.</p>

      {error && <div className="alert error">{error}</div>}

      <div className="stats-grid">
        <div className="card stat-card">
          <h3>Tokens</h3>
          <p className="stat-value">{tokens.length}</p>
          <p className="muted">{activeTokens.length} active</p>
        </div>
        <div className="card stat-card">
          <h3>Launchpad Projects</h3>
          <p className="stat-value">{launchpads.length}</p>
          <p className="muted">{activeLaunchpads.length} active</p>
        </div>
        <div className="card stat-card">
          <h3>Listings</h3>
          <p className="stat-value">{listings.length}</p>
          <p className="muted">{activeListings.length} active</p>
        </div>
        <div className="card stat-card">
          <h3>Backend</h3>
          <p className="stat-value">{health?.status || '—'}</p>
          <p className="muted">{health?.version || health?.service || ''}</p>
        </div>
      </div>

      <section>
        <div className="section-title"><h2>Recent Tokens</h2></div>
        {tokens.length === 0 ? (
          <p className="muted">No tokens yet.</p>
        ) : (
          <div className="cards-grid">
            {tokens.slice(0, 8).map((t: any) => (
              <div key={t.id} className="card">
                {t.logo_url && <img src={t.logo_url} alt={t.name} style={{ width: 40, height: 40, borderRadius: 8 }} />}
                <div className="card-row"><span>Name</span><span><strong>{t.name}</strong></span></div>
                <div className="card-row"><span>Symbol</span><span>{t.symbol}</span></div>
                <div className="card-row"><span>Status</span><span><span className={`badge ${t.status === 'active' ? 'active' : ''}`}>{t.status}</span></span></div>
              </div>
            ))}
          </div>
        )}
      </section>

      <section>
        <div className="section-title"><h2>Active Launchpads</h2></div>
        {activeLaunchpads.length === 0 ? (
          <p className="muted">No active launchpad projects.</p>
        ) : (
          <div className="cards-grid">
            {activeLaunchpads.slice(0, 6).map((p: any) => (
              <div key={p.id} className="card">
                <div className="card-row"><span>Name</span><span><strong>{p.name}</strong></span></div>
                <div className="card-row"><span>Price</span><span>{p.price_per_token || '-'}</span></div>
                <div className="card-row"><span>Sold</span><span>{p.sold_amount || '-'}/{p.total_supply || '-'}</span></div>
              </div>
            ))}
          </div>
        )}
      </section>
    </div>
  );
}
