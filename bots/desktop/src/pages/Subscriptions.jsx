import React, { useEffect, useState, useCallback } from 'react';
import api from '../services/bot-api';

export default function Subscriptions() {
  const [current, setCurrent] = useState(null);
  const [tiers, setTiers] = useState([]);
  const [error, setError] = useState(null);
  const [notice, setNotice] = useState(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(() => {
    setLoading(true); setError(null);
    Promise.all([
      api.getSubscription().catch(e => { setError(e.message); return null; }),
      api.publicTiers().catch(() => ({})),
    ]).then(([sub, t]) => {
      setCurrent(sub && (sub.subscription || sub));
      setTiers((t && (t.tiers || t.plans)) || []);
    }).finally(() => setLoading(false));
  }, []);

  useEffect(load, [load]);

  const subscribe = async (tier) => {
    setError(null); setNotice(null);
    try {
      await api.createSubscription({ tier: tier.id || tier.name });
      setNotice(`Subscribed to ${tier.name || tier.id}`);
      load();
    } catch (e) {
      setError(e.message);
    }
  };

  if (loading) return <div>Loading…</div>;

  return (
    <div className="subscriptions-page">
      <h1>Subscription</h1>
      {error && <p className="error-banner">{error}</p>}
      {notice && <p className="success-banner">{notice}</p>}

      <h2>Current plan</h2>
      {current ? (
        <dl className="detail-list">
          <dt>Tier</dt><dd>{current.tier || '—'}</dd>
          <dt>Status</dt><dd>{current.status || '—'}</dd>
          <dt>Expires</dt><dd>{current.expires_at || '—'}</dd>
        </dl>
      ) : <p className="empty-state">No active subscription.</p>}

      <h2>Available tiers</h2>
      {tiers.length === 0 ? <p className="empty-state">No tiers published by the backend.</p> : (
        <div className="tier-grid">
          {tiers.map(t => (
            <div key={t.id || t.name} className="tier-card">
              <h3>{t.name || t.id}</h3>
              <p className="price">{t.price != null ? `${t.price} ${t.currency || ''}` : '—'}</p>
              {t.max_bots != null && <p>Max bots: {t.max_bots}</p>}
              <button onClick={() => subscribe(t)}>Choose</button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
