// P2P Trading Page — browse adverts and create orders (KYC-gated backend-side).
import React, { useState, useEffect, useCallback } from 'react';
import { api } from '../services/api';

export default function P2P() {
  const [adverts, setAdverts] = useState<unknown[]>([]);
  const [advertId, setAdvertId] = useState('');
  const [amount, setAmount] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [result, setResult] = useState<string | null>(null);

  const load = useCallback(() => {
    api.getP2PAdverts().then((d) => {
      const list = Array.isArray(d) ? d : (d.adverts || []);
      setAdverts(list);
    }).catch(() => setAdverts([]));
  }, []);

  useEffect(() => { load(); }, [load]);

  const createOrder = async () => {
    setError(''); setResult(null);
    if (!advertId) { setError('Select an advert'); return; }
    if (!amount) { setError('Enter an amount'); return; }
    setBusy(true);
    try {
      const res = await api.createP2POrder({ advert_id: advertId, amount });
      setResult(JSON.stringify(res));
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Order creation failed');
    } finally { setBusy(false); }
  };

  return (
    <div className="p2p-page">
      <h1>P2P Trading</h1>
      {error && <div className="error">{error}</div>}
      {result && <div className="success-banner"><h3>✓ Order submitted</h3><p className="mono">{result}</p></div>}
      <h2>Adverts</h2>
      {adverts.length === 0 && <p className="empty-state">No P2P adverts available.</p>}
      <ul className="record-list">
        {adverts.map((a, i) => {
          const rec = a as Record<string, unknown>;
          const id = String(rec.id ?? rec.advert_id ?? i);
          return (
            <li key={id} className="record-item">
              <label>
                <input type="radio" name="advert" checked={advertId === id} onChange={() => setAdvertId(id)} />
                <span className="mono"> {JSON.stringify(rec)}</span>
              </label>
            </li>
          );
        })}
      </ul>
      <div className="p2p-form">
        <div className="form-group"><label>Amount</label><input type="number" step="any" value={amount} onChange={(e) => setAmount(e.target.value)} /></div>
        <div className="action-row">
          <button className="primary-btn" onClick={createOrder} disabled={busy}>Create Order</button>
        </div>
      </div>
    </div>
  );
}
