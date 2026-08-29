// Price Alerts Page — list / create / delete price alerts.
import React, { useState, useEffect, useCallback } from 'react';
import { api } from '../services/api';

export default function PriceAlerts() {
  const [alerts, setAlerts] = useState<unknown[]>([]);
  const [symbol, setSymbol] = useState('ETH');
  const [target, setTarget] = useState('');
  const [direction, setDirection] = useState('above');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [result, setResult] = useState<string | null>(null);

  const load = useCallback(() => {
    api.getPriceAlerts().then((d) => {
      const list = Array.isArray(d) ? d : ((d as Record<string, unknown>).alerts as unknown[] || []);
      setAlerts(list);
    }).catch(() => setAlerts([]));
  }, []);

  useEffect(() => { load(); }, [load]);

  const create = async () => {
    setError(''); setResult(null);
    if (!target) { setError('Enter a target price'); return; }
    setBusy(true);
    try {
      const res = await api.createPriceAlert({ symbol, target_price: target, direction });
      setResult(JSON.stringify(res));
      setTarget('');
      load();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Create alert failed');
    } finally { setBusy(false); }
  };

  const remove = async (id: string) => {
    setError(''); setBusy(true);
    try {
      await api.deletePriceAlert(id);
      load();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Delete alert failed');
    } finally { setBusy(false); }
  };

  return (
    <div className="price-alerts-page">
      <h1>Price Alerts</h1>
      {error && <div className="error">{error}</div>}
      {result && <div className="success-banner"><h3>✓ Alert created</h3><p className="mono">{result}</p></div>}
      <div className="price-alert-form">
        <div className="form-group"><label>Symbol</label><input value={symbol} onChange={(e) => setSymbol(e.target.value.toUpperCase())} /></div>
        <div className="form-group"><label>Target Price (USD)</label><input type="number" step="any" value={target} onChange={(e) => setTarget(e.target.value)} /></div>
        <div className="form-group">
          <label>Direction</label>
          <select value={direction} onChange={(e) => setDirection(e.target.value)}>
            <option value="above">Above</option>
            <option value="below">Below</option>
          </select>
        </div>
        <div className="action-row">
          <button className="primary-btn" onClick={create} disabled={busy}>Create Alert</button>
        </div>
      </div>
      <h2>My Alerts</h2>
      {alerts.length === 0 && <p className="empty-state">No price alerts.</p>}
      <ul className="record-list">
        {alerts.map((a, i) => {
          const rec = a as Record<string, unknown>;
          const id = String(rec.id ?? i);
          return (
            <li key={id} className="record-item">
              <span className="mono">{JSON.stringify(rec)}</span>
              <button className="secondary-btn" onClick={() => remove(id)} disabled={busy}>Delete</button>
            </li>
          );
        })}
      </ul>
    </div>
  );
}
