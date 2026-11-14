import React, { useEffect, useState } from 'react';
import api from '../services/project-party-api';

export default function Fees() {
  const [fees, setFees] = useState(null);
  const [error, setError] = useState(null);

  useEffect(() => {
    let alive = true;
    api.getListingFees()
      .then(d => { if (alive) setFees(d); })
      .catch(e => { if (alive) setError(e.message); });
    return () => { alive = false; };
  }, []);

  const rows = fees
    ? (fees.fees || fees.tiers || (Array.isArray(fees) ? fees : []))
    : [];

  return (
    <div className="fees-page">
      <h1>Listing fees</h1>
      {error && <p className="error-banner">{error}</p>}
      {!fees && !error && <p>Loading…</p>}
      {fees && rows.length === 0 && <p className="empty-state">No fee schedule published.</p>}
      {rows.length > 0 && (
        <table className="data-table">
          <thead><tr><th>Name</th><th>Amount</th><th>Currency</th><th>Active</th></tr></thead>
          <tbody>
            {rows.map((f, i) => (
              <tr key={f.id || i}>
                <td>{f.name || f.tier || '—'}</td>
                <td>{f.amount != null ? f.amount : (f.price != null ? f.price : '—')}</td>
                <td>{f.currency || '—'}</td>
                <td>{f.active === undefined ? '—' : (f.active ? 'yes' : 'no')}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
