// Analytics Page - ProjectParty
import React, { useState, useEffect } from 'react';
import { api } from '../services/api';

export default function Analytics() {
  const [volume, setVolume] = useState<any>(null);
  const [liquidity, setLiquidity] = useState<any>(null);
  const [holders, setHolders] = useState<any>(null);
  const [txns, setTxns] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [holderToken, setHolderToken] = useState('');

  useEffect(() => {
    let active = true;
    (async () => {
      setLoading(true);
      setError('');
      try {
        const [v, l, t] = await Promise.all([
          api.getTradingVolume(),
          api.getLiquidity(),
          api.getTransactionCount()
        ]);
        if (!active) return;
        setVolume(v);
        setLiquidity(l);
        setTxns(t);
      } catch (e: any) {
        if (active) setError(e.message || 'Failed to load analytics');
      }
      if (active) setLoading(false);
    })();
    return () => { active = false; };
  }, []);

  const lookupHolders = async (e: React.FormEvent) => {
    e.preventDefault();
    setHolders(null);
    try {
      const data = await api.getHolderCount(holderToken || undefined);
      setHolders(data);
    } catch (err: any) {
      setError(err.message || 'Failed to load holder count');
    }
  };

  if (loading) return <div className="page"><div className="state">Loading analytics...</div></div>;

  return (
    <div className="page">
      <div className="page-header"><h1>Analytics</h1></div>
      <p className="subtitle">Platform-wide trading volume, liquidity, holder counts and transaction counts.</p>

      {error && <div className="alert error">{error}</div>}

      <div className="stats-grid">
        <div className="stat-card">
          <h3>Trading Volume (24h)</h3>
          <p>${volume?.total_24h ?? '0'}</p>
          <div className="stat-sub">7d: ${volume?.total_7d ?? '0'} &middot; 30d: ${volume?.total_30d ?? '0'}</div>
        </div>
        <div className="stat-card">
          <h3>Total Liquidity</h3>
          <p>${liquidity?.total_liquidity ?? '0'}</p>
        </div>
        <div className="stat-card">
          <h3>Transactions (24h)</h3>
          <p>{txns?.total_24h ?? 0}</p>
          <div className="stat-sub">7d: {txns?.total_7d ?? 0}</div>
        </div>
        <div className="stat-card">
          <h3>Token Holders</h3>
          <p>{holders?.total ?? 'N/A'}</p>
          <div className="stat-sub">{holders?.token_id ? `token: ${String(holders.token_id).slice(0, 8)}...` : 'enter a token ID'}</div>
        </div>
      </div>

      <section>
        <div className="section-title"><h2>Holder Count Lookup</h2></div>
        <form onSubmit={lookupHolders} style={{ display: 'flex', gap: '0.5rem' }}>
          <input value={holderToken} onChange={e => setHolderToken(e.target.value)} placeholder="Token ID (UUID)" />
          <button type="submit">Lookup</button>
        </form>
        {holders && (
          <div className="card-row" style={{ marginTop: '0.75rem' }}>
            <span>Unique holders</span><span>{holders.total}</span>
          </div>
        )}
      </section>
    </div>
  );
}
