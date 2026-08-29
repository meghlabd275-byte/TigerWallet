// Token Sales Page — browse active token sales and participate.
import React, { useState, useEffect, useCallback } from 'react';
import { api } from '../services/api';

export default function TokenSales() {
  const [sales, setSales] = useState<unknown[]>([]);
  const [amounts, setAmounts] = useState<Record<string, string>>({});
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [result, setResult] = useState<string | null>(null);

  const load = useCallback(() => {
    api.getTokenSales().then((d) => setSales(d.sales || [])).catch(() => setSales([]));
  }, []);

  useEffect(() => { load(); }, [load]);

  const participate = async (saleId: string) => {
    setError(''); setResult(null);
    const amount = amounts[saleId] || '';
    if (!amount) { setError('Enter an amount'); return; }
    setBusy(true);
    try {
      const res = await api.participateTokenSale({ saleId, amount });
      setResult(JSON.stringify(res));
      load();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Participation failed');
    } finally { setBusy(false); }
  };

  return (
    <div className="token-sales-page">
      <h1>Token Sales</h1>
      {error && <div className="error">{error}</div>}
      {result && <div className="success-banner"><h3>✓ Participation submitted to the blockchain network</h3><p className="mono">{result}</p></div>}
      {sales.length === 0 && <p className="empty-state">No active token sales.</p>}
      <ul className="record-list">
        {sales.map((s, i) => {
          const rec = s as Record<string, unknown>;
          const id = String(rec.id ?? rec.sale_id ?? i);
          return (
            <li key={id} className="record-item">
              <div>
                <strong>{String(rec.name ?? rec.symbol ?? id)}</strong>
                <p className="mono">{JSON.stringify(rec)}</p>
              </div>
              <div className="action-row">
                <input
                  type="number" step="any" placeholder="Amount"
                  value={amounts[id] || ''}
                  onChange={(e) => setAmounts({ ...amounts, [id]: e.target.value })}
                />
                <button className="primary-btn" onClick={() => participate(id)} disabled={busy}>Participate</button>
              </div>
            </li>
          );
        })}
      </ul>
    </div>
  );
}
