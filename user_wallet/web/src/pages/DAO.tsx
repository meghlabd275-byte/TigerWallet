// DAO Page — governance proposals, delegates, and voting.
import React, { useState, useEffect, useCallback } from 'react';
import { api } from '../services/api';

export default function DAO() {
  const [proposals, setProposals] = useState<unknown[]>([]);
  const [delegates, setDelegates] = useState<unknown[]>([]);
  const [error, setError] = useState('');
  const [result, setResult] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const load = useCallback(() => {
    api.getDaoProposals().then((d) => {
      const list = Array.isArray(d) ? d : ((d as Record<string, unknown>).proposals as unknown[] || []);
      setProposals(list);
    }).catch(() => setProposals([]));
    api.getDaoDelegates().then((d) => {
      const list = Array.isArray(d) ? d : ((d as Record<string, unknown>).delegates as unknown[] || []);
      setDelegates(list);
    }).catch(() => setDelegates([]));
  }, []);

  useEffect(() => { load(); }, [load]);

  const vote = async (proposalId: string, support: boolean) => {
    setError(''); setResult(null); setBusy(true);
    try {
      const res = await api.voteDaoProposal({ proposalId, support });
      setResult(JSON.stringify(res));
      load();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Vote failed');
    } finally { setBusy(false); }
  };

  return (
    <div className="dao-page">
      <h1>DAO Governance</h1>
      {error && <div className="error">{error}</div>}
      {result && <div className="success-banner"><h3>✓ Vote submitted to the blockchain network</h3><p className="mono">{result}</p></div>}
      <h2>Proposals</h2>
      {proposals.length === 0 && <p className="empty-state">No active proposals.</p>}
      <ul className="record-list">
        {proposals.map((p, i) => {
          const rec = p as Record<string, unknown>;
          const id = String(rec.id ?? rec.proposal_id ?? i);
          return (
            <li key={id} className="record-item">
              <div>
                <strong>{String(rec.title ?? id)}</strong>
                <p className="mono">{JSON.stringify(rec)}</p>
              </div>
              <div className="action-row">
                <button className="primary-btn" onClick={() => vote(id, true)} disabled={busy}>For</button>
                <button className="secondary-btn" onClick={() => vote(id, false)} disabled={busy}>Against</button>
              </div>
            </li>
          );
        })}
      </ul>
      <h2>Delegates</h2>
      {delegates.length === 0 && <p className="empty-state">No delegates.</p>}
      <ul className="record-list">
        {delegates.map((d, i) => (
          <li key={i} className="record-item"><span className="mono">{JSON.stringify(d)}</span></li>
        ))}
      </ul>
    </div>
  );
}
