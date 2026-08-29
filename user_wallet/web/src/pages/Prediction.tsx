// Prediction Markets Page — browse markets and place positions.
import React, { useState, useEffect, useCallback } from 'react';
import { api } from '../services/api';

export default function Prediction() {
  const [markets, setMarkets] = useState<unknown[]>([]);
  const [amounts, setAmounts] = useState<Record<string, string>>({});
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [result, setResult] = useState<string | null>(null);

  const load = useCallback(() => {
    api.getPredictionMarkets().then((d) => setMarkets(d.markets || [])).catch(() => setMarkets([]));
  }, []);

  useEffect(() => { load(); }, [load]);

  const bet = async (marketId: string, side: string) => {
    setError(''); setResult(null);
    const amount = amounts[marketId] || '';
    if (!amount) { setError('Enter an amount'); return; }
    setBusy(true);
    try {
      const res = await api.placePredictionBet({ marketId, side, amount });
      setResult(JSON.stringify(res));
      load();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Bet failed');
    } finally { setBusy(false); }
  };

  return (
    <div className="prediction-page">
      <h1>Prediction Markets</h1>
      {error && <div className="error">{error}</div>}
      {result && <div className="success-banner"><h3>✓ Position submitted</h3><p className="mono">{result}</p></div>}
      {markets.length === 0 && <p className="empty-state">No active prediction markets.</p>}
      <ul className="record-list">
        {markets.map((m, i) => {
          const rec = m as Record<string, unknown>;
          const id = String(rec.id ?? rec.market_id ?? i);
          return (
            <li key={id} className="record-item">
              <div>
                <strong>{String(rec.question ?? rec.title ?? id)}</strong>
                <p className="mono">{JSON.stringify(rec)}</p>
              </div>
              <div className="action-row">
                <input
                  type="number" step="any" placeholder="Amount"
                  value={amounts[id] || ''}
                  onChange={(e) => setAmounts({ ...amounts, [id]: e.target.value })}
                />
                <button className="primary-btn" onClick={() => bet(id, 'yes')} disabled={busy}>Yes</button>
                <button className="secondary-btn" onClick={() => bet(id, 'no')} disabled={busy}>No</button>
              </div>
            </li>
          );
        })}
      </ul>
    </div>
  );
}
