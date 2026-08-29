// Copy Trading Page — browse traders, follow/stop, view live signals.
import React, { useState, useEffect, useCallback } from 'react';
import { api } from '../services/api';

export default function CopyTrading() {
  const [traders, setTraders] = useState<unknown[]>([]);
  const [signals, setSignals] = useState<unknown>(null);
  const [allocations, setAllocations] = useState<Record<string, string>>({});
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [result, setResult] = useState<string | null>(null);

  const load = useCallback(() => {
    api.getCopyTraders().then((d) => {
      const list = Array.isArray(d) ? d : ((d as Record<string, unknown>).traders as unknown[] || []);
      setTraders(list);
    }).catch(() => setTraders([]));
    api.getCopySignals().then(setSignals).catch(() => setSignals(null));
  }, []);

  useEffect(() => { load(); }, [load]);

  const follow = async (traderId: string) => {
    setError(''); setResult(null); setBusy(true);
    try {
      const res = await api.followTrader({ traderId, allocation: allocations[traderId] || undefined });
      setResult(JSON.stringify(res));
      load();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Follow failed');
    } finally { setBusy(false); }
  };

  return (
    <div className="copy-trading-page">
      <h1>Copy Trading</h1>
      {error && <div className="error">{error}</div>}
      {result && <div className="success-banner"><h3>✓ Follow submitted</h3><p className="mono">{result}</p></div>}
      <h2>Traders</h2>
      {traders.length === 0 && <p className="empty-state">No traders available.</p>}
      <ul className="record-list">
        {traders.map((t, i) => {
          const rec = t as Record<string, unknown>;
          const id = String(rec.id ?? rec.trader_id ?? i);
          return (
            <li key={id} className="record-item">
              <div>
                <strong>{String(rec.name ?? rec.address ?? id)}</strong>
                <p className="mono">{JSON.stringify(rec)}</p>
              </div>
              <div className="action-row">
                <input
                  type="number" step="any" placeholder="Allocation"
                  value={allocations[id] || ''}
                  onChange={(e) => setAllocations({ ...allocations, [id]: e.target.value })}
                />
                <button className="primary-btn" onClick={() => follow(id)} disabled={busy}>Follow</button>
              </div>
            </li>
          );
        })}
      </ul>
      <h2>Signals</h2>
      {signals ? <div className="quote-box"><pre>{JSON.stringify(signals, null, 2)}</pre></div> : <p className="empty-state">No signals.</p>}
    </div>
  );
}
