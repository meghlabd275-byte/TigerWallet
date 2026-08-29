// Trading Page — perpetual + margin positions (open / list / close).
import React, { useState, useEffect, useCallback } from 'react';
import { api } from '../services/api';

export default function Trading() {
  const [tab, setTab] = useState<'perpetual' | 'margin'>('perpetual');
  const [positions, setPositions] = useState<unknown[]>([]);
  const [pair, setPair] = useState('ETH-USDT');
  const [side, setSide] = useState('long');
  const [size, setSize] = useState('');
  const [leverage, setLeverage] = useState(2);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [result, setResult] = useState<string | null>(null);

  const load = useCallback(() => {
    const fn = tab === 'perpetual' ? api.getPerpetualPositions : api.getMarginPositions;
    fn().then((d) => setPositions(d.positions || [])).catch(() => setPositions([]));
  }, [tab]);

  useEffect(() => { load(); }, [load]);

  const open = async () => {
    setError(''); setResult(null);
    if (!size) { setError('Enter a position size'); return; }
    setBusy(true);
    try {
      const params = { pair, side, size, leverage };
      const res = tab === 'perpetual'
        ? await api.createPerpetualPosition(params)
        : await api.createMarginPosition(params);
      setResult(JSON.stringify(res));
      load();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Open position failed');
    } finally { setBusy(false); }
  };

  const close = async (id: string) => {
    setError(''); setResult(null); setBusy(true);
    try {
      const res = tab === 'perpetual'
        ? await api.closePerpetualPosition(id)
        : await api.closeMarginPosition(id);
      setResult(JSON.stringify(res));
      load();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Close position failed');
    } finally { setBusy(false); }
  };

  return (
    <div className="trading-page">
      <h1>Trading</h1>
      <div className="action-row">
        <button className={tab === 'perpetual' ? 'primary-btn' : 'secondary-btn'} onClick={() => setTab('perpetual')}>Perpetual</button>
        <button className={tab === 'margin' ? 'primary-btn' : 'secondary-btn'} onClick={() => setTab('margin')}>Margin</button>
      </div>
      {error && <div className="error">{error}</div>}
      {result && <div className="success-banner"><h3>✓ Order submitted to the blockchain network</h3><p className="mono">{result}</p></div>}
      <div className="trading-form">
        <div className="form-group"><label>Pair</label><input value={pair} onChange={(e) => setPair(e.target.value)} /></div>
        <div className="form-group">
          <label>Side</label>
          <select value={side} onChange={(e) => setSide(e.target.value)}>
            <option value="long">Long</option>
            <option value="short">Short</option>
          </select>
        </div>
        <div className="form-group"><label>Size</label><input type="number" step="any" value={size} onChange={(e) => setSize(e.target.value)} /></div>
        <div className="form-group"><label>Leverage</label><input type="number" min={1} max={125} value={leverage} onChange={(e) => setLeverage(Number(e.target.value))} /></div>
        <div className="action-row">
          <button className="primary-btn" onClick={open} disabled={busy}>Open Position</button>
        </div>
      </div>
      <h2>Open Positions</h2>
      {positions.length === 0 && <p className="empty-state">No open positions.</p>}
      <ul className="record-list">
        {positions.map((p, i) => {
          const rec = p as Record<string, unknown>;
          const id = String(rec.id ?? rec.position_id ?? i);
          return (
            <li key={id} className="record-item">
              <span>{String(rec.pair ?? '')} {String(rec.side ?? '')} size={String(rec.size ?? '')} lev={String(rec.leverage ?? '')}</span>
              <button className="secondary-btn" onClick={() => close(id)} disabled={busy}>Close</button>
            </li>
          );
        })}
      </ul>
    </div>
  );
}
