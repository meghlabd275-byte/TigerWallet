import React, { useEffect, useState, useCallback } from 'react';
import { useParams, Link } from 'react-router-dom';
import api from '../services/project-party-api';

export default function TokenDetail() {
  const { id } = useParams();
  const [token, setToken] = useState(null);
  const [price, setPrice] = useState(null);
  const [history, setHistory] = useState([]);
  const [compliance, setCompliance] = useState(null);
  const [error, setError] = useState(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(() => {
    setLoading(true); setError(null);
    Promise.all([
      api.getToken(id).catch(e => { setError(e.message); return null; }),
      api.getTokenPrice(id).catch(() => null),
      api.getPriceHistory(id).catch(() => ({})),
      api.getComplianceStatus(id).catch(() => null),
    ]).then(([t, p, h, c]) => {
      setToken(t && (t.token || t));
      setPrice(p && (p.price || p));
      setHistory((h && (h.history || h.prices)) || []);
      setCompliance(c);
    }).finally(() => setLoading(false));
  }, [id]);

  useEffect(load, [load]);

  if (loading) return <div>Loading…</div>;

  return (
    <div className="token-detail">
      <p><Link to="/tokens">← All tokens</Link></p>
      <h1>{token ? `${token.name} (${token.symbol})` : `Token ${id}`}</h1>
      {error && <p className="error-banner">{error}</p>}

      {token && (
        <dl className="detail-list">
          <dt>Status</dt><dd>{token.status || '—'}</dd>
          <dt>Chain</dt><dd>{token.chain || token.chain_id || '—'}</dd>
          <dt>Contract</dt><dd>{token.contract_address || '—'}</dd>
          <dt>Decimals</dt><dd>{token.decimals != null ? token.decimals : '—'}</dd>
          <dt>Contract verified</dt><dd>{token.contract_verified ? 'yes' : 'no'}</dd>
          <dt>Featured</dt><dd>{token.is_featured ? 'yes' : 'no'}</dd>
        </dl>
      )}

      <h2>Price</h2>
      {price ? (
        <dl className="detail-list">
          <dt>Price</dt><dd>{price.price != null ? price.price : '—'}</dd>
          <dt>24h change</dt><dd>{price.change_24h != null ? `${price.change_24h}%` : '—'}</dd>
          <dt>24h volume</dt><dd>{price.volume_24h != null ? price.volume_24h : '—'}</dd>
          <dt>Liquidity</dt><dd>{price.liquidity_usd != null ? price.liquidity_usd : '—'}</dd>
        </dl>
      ) : <p className="empty-state">No price published for this token.</p>}

      <h2>Price history</h2>
      {history.length === 0 ? <p className="empty-state">No history recorded.</p> : (
        <table className="data-table">
          <thead><tr><th>Time</th><th>Price</th><th>Volume</th></tr></thead>
          <tbody>
            {history.map((h, i) => (
              <tr key={h.timestamp || i}>
                <td>{h.timestamp || h.created_at || '—'}</td>
                <td>{h.price != null ? h.price : '—'}</td>
                <td>{h.volume != null ? h.volume : '—'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      <h2>Compliance</h2>
      {compliance ? (
        <dl className="detail-list">
          <dt>Audit</dt><dd>{compliance.audit_status || compliance.audit || '—'}</dd>
          <dt>KYC</dt><dd>{compliance.kyc_status || compliance.kyc || '—'}</dd>
        </dl>
      ) : <p className="empty-state">No compliance record.</p>}
    </div>
  );
}
