// Trading Page — perpetual + margin + options engine (open / list / close).
import React, { useState, useEffect, useCallback } from 'react';
import { api } from '../services/api';

type Tab = 'perpetual' | 'margin' | 'options';

interface Series {
  id: string; underlying: string; quote_asset: string; strike: string;
  expiry_unix: number; style: string; iv_bps: number; contract_size: string; status: string;
}

export default function Trading() {
  const [tab, setTab] = useState<Tab>('perpetual');
  const [positions, setPositions] = useState<unknown[]>([]);
  const [series, setSeries] = useState<Series[]>([]);
  const [pair, setPair] = useState('ETH-USDT');
  const [side, setSide] = useState('long');
  const [size, setSize] = useState('');
  const [leverage, setLeverage] = useState(2);
  const [optSeriesId, setOptSeriesId] = useState('');
  const [optSide, setOptSide] = useState<'buy' | 'sell'>('buy');
  const [optContracts, setOptContracts] = useState('');
  const [underlyingFilter, setUnderlyingFilter] = useState('');
  const [quote, setQuote] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [result, setResult] = useState<string | null>(null);

  const load = useCallback(() => {
    if (tab === 'options') {
      api.getOptionsSeries().then(d => setSeries(d.series || [])).catch(() => setSeries([]));
      api.getOptionsPositions().then(d => setPositions(d.positions || [])).catch(() => setPositions([]));
      return;
    }
    const fn = tab === 'perpetual' ? api.getPerpetualPositions : api.getMarginPositions;
    fn().then((d) => setPositions(d.positions || [])).catch(() => setPositions([]));
  }, [tab]);

  useEffect(() => { load(); }, [load]);

  const fetchQuote = async (seriesId: string) => {
    if (!seriesId) { setQuote(null); return; }
    setQuote(null);
    try {
      const q = await api.getOptionsQuote(seriesId);
      setQuote(`$${Number(q.underlying_price).toFixed(2)} underlying · premium $${Number(q.premium_per_contract).toFixed(4)} per contract`);
    } catch (err: unknown) {
      setQuote('Quote unavailable');}
  };

  const open = async () => {
    setError(''); setResult(null);
    if (tab === 'options') {
      if (!optSeriesId) { setError('Select a series'); return; }
      if (!optContracts || Number(optContracts) <= 0) { setError('Enter a positive contract count'); return; }
      setBusy(true);
      try {
        const res = await api.openOptionsPosition({ seriesId: optSeriesId, side: optSide, contracts: optContracts });
        setResult(JSON.stringify(res));
        load(); fetchQuote(optSeriesId);
      } catch (err: unknown) {
        setError(err instanceof Error ? err.message : 'Open position failed');
      } finally { setBusy(false); }
      return;
    }
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
      const res = tab === 'options'
        ? await api.closeOptionsPosition(id)
        : tab === 'perpetual'
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
        <button className={tab === 'options' ? 'primary-btn' : 'secondary-btn'} onClick={() => setTab('options')}>Options</button>
      </div>
      {error && <div className="error">{error}</div>}
      {result && <div className="success-banner"><h3>✓ Order submitted to the blockchain network</h3><p className="mono">{result}</p></div>}
      {tab === 'options' ? (
        <div className="trading-form">
          <div className="form-group">
            <label>Underlying</label>
            <div className="action-row">
              <input value={underlyingFilter} placeholder="Filter by underlying (e.g. BTC)" onChange={(e) => setUnderlyingFilter(e.target.value)} />
              <button className="secondary-btn" onClick={() => {
                api.getOptionsSeries(underlyingFilter || undefined).then(d => setSeries(d.series || [])).catch(() => setSeries([]));
              }}>Filter</button>
            </div>
          </div>
          <div className="form-group"><label>Live Series</label>
            {series.length === 0 ? <p className="empty-state">No active options series. An operator must add series first.</p> : (
              <select value={optSeriesId} onChange={(e) => { setOptSeriesId(e.target.value); fetchQuote(e.target.value); }}>
                {series.map(s => (
                  <option key={s.id} value={s.id}>
                    {s.underlying}-{s.strike} {s.style.toUpperCase()} exp {new Date(s.expiry_unix *  1000).toISOString().slice(0, 10)} · {s.quote_asset}
                  </option>
                ))}
              </select>
            )}
          </div>
          {optSeriesId && <p className="hint">{quote ?? 'Select a series to load its live quote.'}</p>}
          <div className="form-group">
            <label>Side</label>
            <select value={optSide} onChange={(e) => setOptSide(e.target.value as 'buy' | 'sell')}>
              <option value="buy">Buy (long with)</option>
              <option value="sell">Sell (short/underwriter)</option>
            </select>
          </div>
          <div className="form-group"><label>Contracts</label><input type="number" step="any" min={1} value={optContracts} onChange={(e) => setOptContracts(e.target.value)} /></div>
          <div className="action-row">
            <button className="primary-btn" onClick={open} disabled={busy}>Open Options Position</button>
          </div>
        </div>
      ) : (
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
      )}
      <h2>Open Positions</h2>
      {positions.length === 0 && <p className="empty-state">No open positions.</p>}
      <ul className="record-list">
        {positions.map((p, i) => {
          const rec = p as Record<string, unknown>;
          const id = String(rec.id ?? rec.position_id ?? i);
          const label = tab === 'options'
            ? `${String(rec.underlying ?? '')}-${String(rec.strike ?? '')} ${String(rec.style ?? '')} ${String(rec.side ?? '')} x${String(rec.contracts ?? '')} premium ${String(rec.premium ?? '')}`
            : `${String(rec.pair ?? '')} ${String(rec.side ?? '')} size=${String(rec.size ?? '')} lev=${String(rec.leverage ?? '')}`;
          return (
            <li key={id} className="record-item">
              <span>{label}</span>
              <button className="secondary-btn" onClick={() => close(id)} disabled={busy}>Close</button>
            </li>
          );
        })}
      </ul>
    </div>
  );
}
