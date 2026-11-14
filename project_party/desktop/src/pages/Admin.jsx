import React, { useEffect, useState, useCallback } from 'react';
import api from '../services/project-party-api';

export default function Admin() {
  const [pending, setPending] = useState([]);
  const [payments, setPayments] = useState([]);
  const [error, setError] = useState(null);
  const [notice, setNotice] = useState(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(() => {
    setLoading(true); setError(null);
    Promise.all([
      api.listTokens().catch(e => { setError(e.message); return null; }),
      api.listFeePayments().catch(() => ({})),
    ]).then(([t, p]) => {
      const all = (t && t.tokens) || [];
      setPending(all.filter(x => x.status !== 'listed' && x.status !== 'rejected'));
      setPayments((p && (p.payments || p.items)) || []);
    }).finally(() => setLoading(false));
  }, []);

  useEffect(load, [load]);

  const run = async (fn, msg) => {
    setError(null); setNotice(null);
    try { await fn(); setNotice(msg); load(); }
    catch (e) { setError(e.message); }
  };

  return (
    <div className="admin-page">
      <h1>Admin console</h1>
      <p className="hint">Approval actions require an admin or super_admin role on the backend.</p>
      {error && <p className="error-banner">{error}</p>}
      {notice && <p className="success-banner">{notice}</p>}

      <h2>Token approval queue</h2>
      {loading ? <p>Loading…</p> : pending.length === 0 ? (
        <p className="empty-state">Nothing awaiting review.</p>
      ) : (
        <table className="data-table">
          <thead><tr><th>Symbol</th><th>Name</th><th>Status</th><th>Verified</th><th>Actions</th></tr></thead>
          <tbody>
            {pending.map(t => (
              <tr key={t.id}>
                <td>{t.symbol}</td>
                <td>{t.name}</td>
                <td>{t.status || '—'}</td>
                <td>{t.contract_verified ? 'yes' : 'no'}</td>
                <td className="action-row">
                  <button onClick={() => run(() => api.approveToken(t.id), `${t.symbol} approved`)}>Approve</button>
                  <button onClick={() => run(() => api.rejectToken(t.id), `${t.symbol} rejected`)}>Reject</button>
                  <button onClick={() => run(() => api.verifyTokenContract(t.id), `${t.symbol} contract verified on-chain`)}>Verify contract</button>
                  <button onClick={() => run(() => api.featureListing(t.id), `${t.symbol} featured`)}>Feature</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      <h2>Fee payment verification</h2>
      {payments.length === 0 ? <p className="empty-state">No fee payments recorded.</p> : (
        <table className="data-table">
          <thead><tr><th>Reference</th><th>Amount</th><th>Status</th><th>Tx</th><th>Actions</th></tr></thead>
          <tbody>
            {payments.map(p => (
              <tr key={p.id}>
                <td>{p.reference || p.id}</td>
                <td>{p.amount != null ? p.amount : '—'}</td>
                <td>{p.status || '—'}</td>
                <td>{p.tx_hash || '—'}</td>
                <td className="action-row">
                  <button onClick={() => run(() => api.verifyFeePayment(p.id), 'Fee payment verified on-chain')}>Verify</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
