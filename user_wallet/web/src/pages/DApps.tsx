// dApps Page — browse the dApp catalog, manage WalletConnect pairings/sessions.
import React, { useState, useEffect, useCallback } from 'react';
import { api } from '../services/api';

export default function DApps() {
  const [dapps, setDapps] = useState<unknown[]>([]);
  const [categories, setCategories] = useState<unknown[]>([]);
  const [pairings, setPairings] = useState<unknown[]>([]);
  const [sessions, setSessions] = useState<unknown[]>([]);
  const [error, setError] = useState('');

  const load = useCallback(() => {
    api.getDapps().then((d) => setDapps(d.dapps || [])).catch(() => setDapps([]));
    api.getDappCategories().then((d) => setCategories(d.categories || [])).catch(() => setCategories([]));
    api.getDappPairings().then((d) => {
      const list = Array.isArray(d) ? d : ((d as Record<string, unknown>).pairings as unknown[] || []);
      setPairings(list);
    }).catch(() => setPairings([]));
    api.getDappSessions().then((d) => {
      const list = Array.isArray(d) ? d : ((d as Record<string, unknown>).sessions as unknown[] || []);
      setSessions(list);
    }).catch(() => setSessions([]));
  }, []);

  useEffect(() => { load(); }, [load]);

  const decide = async (id: string, approve: boolean) => {
    setError('');
    try {
      if (approve) await api.approveDappPairing(id);
      else await api.rejectDappPairing(id);
      load();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Pairing decision failed');
    }
  };

  return (
    <div className="dapps-page">
      <h1>dApps</h1>
      {error && <div className="error">{error}</div>}
      <h2>Catalog</h2>
      {categories.length > 0 && (
        <p className="category-row">{categories.map((c, i) => <span key={i} className="category-chip">{String((c as Record<string, unknown>).name ?? c)}</span>)}</p>
      )}
      {dapps.length === 0 && <p className="empty-state">No dApps in catalog.</p>}
      <ul className="record-list">
        {dapps.map((d, i) => {
          const rec = d as Record<string, unknown>;
          return (
            <li key={i} className="record-item">
              <strong>{String(rec.name ?? '')}</strong>
              <span className="mono">{String(rec.url ?? rec.category ?? '')}</span>
            </li>
          );
        })}
      </ul>
      <h2>WalletConnect Pairings</h2>
      {pairings.length === 0 && <p className="empty-state">No pending pairings.</p>}
      <ul className="record-list">
        {pairings.map((p, i) => {
          const rec = p as Record<string, unknown>;
          const id = String(rec.id ?? rec.pairing_id ?? i);
          return (
            <li key={id} className="record-item">
              <span className="mono">{JSON.stringify(rec)}</span>
              <button className="primary-btn" onClick={() => decide(id, true)}>Approve</button>
              <button className="secondary-btn" onClick={() => decide(id, false)}>Reject</button>
            </li>
          );
        })}
      </ul>
      <h2>Active Sessions</h2>
      {sessions.length === 0 && <p className="empty-state">No active sessions.</p>}
      <ul className="record-list">
        {sessions.map((s, i) => (
          <li key={i} className="record-item"><span className="mono">{JSON.stringify(s)}</span></li>
        ))}
      </ul>
    </div>
  );
}
