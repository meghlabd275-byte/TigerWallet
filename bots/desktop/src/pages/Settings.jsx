import React, { useEffect, useState } from 'react';
import api from '../services/bot-api';

export default function Settings() {
  const [health, setHealth] = useState(null);
  const [txs, setTxs] = useState([]);
  const [error, setError] = useState(null);

  useEffect(() => {
    let alive = true;
    api.health()
      .then(h => { if (alive) setHealth(h); })
      .catch(e => { if (alive) setError(e.message); });
    api.listBotTransactions()
      .then(d => { if (alive) setTxs((d && (d.transactions || d.items)) || []); })
      .catch(() => { /* the transactions feed is optional on this page */ });
    return () => { alive = false; };
  }, []);

  return (
    <div className="settings-page">
      <h1>Settings</h1>
      {error && <p className="error-banner">{error}</p>}

      <h2>Backend health</h2>
      {health ? (
        <dl className="detail-list">
          <dt>Status</dt><dd>{health.status || 'ok'}</dd>
          <dt>Version</dt><dd>{health.version || '—'}</dd>
        </dl>
      ) : <p className="empty-state">Backend unreachable.</p>}

      <h2>Recent transactions</h2>
      {txs.length === 0 ? <p className="empty-state">No transactions.</p> : (
        <table className="data-table">
          <thead><tr><th>Time</th><th>Type</th><th>Amount</th><th>Status</th></tr></thead>
          <tbody>
            {txs.map((t, i) => (
              <tr key={t.id || i}>
                <td>{t.created_at || '—'}</td>
                <td>{t.type || '—'}</td>
                <td>{t.amount != null ? t.amount : '—'}</td>
                <td>{t.status || '—'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
