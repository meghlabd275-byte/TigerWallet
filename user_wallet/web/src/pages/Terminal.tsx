// Trading Terminal — live 24h ticker + OHLC candlestick chart rendered on a
// canvas. Data comes from the backend's real CoinGecko-backed kline/ticker
// endpoints; the chart re-renders on every fetch.
import React, { useState, useEffect, useRef, useCallback } from 'react';
import { api } from '../services/api';

interface Candle { t: number; o: number; h: number; l: number; c: number }

function parseCandles(raw: unknown): Candle[] {
  const list = Array.isArray(raw) ? raw : ((raw as Record<string, unknown>)?.candles as unknown[] ?? (raw as Record<string, unknown>)?.kline as unknown[] ?? []);
  return list.map((c) => {
    const a = c as unknown[];
    const r = c as Record<string, unknown>;
    return Array.isArray(c)
      ? { t: Number(a[0]), o: Number(a[1]), h: Number(a[2]), l: Number(a[3]), c: Number(a[4]) }
      : { t: Number(r.time ?? r.t ?? r.timestamp), o: Number(r.open ?? r.o), h: Number(r.high ?? r.h), l: Number(r.low ?? r.l), c: Number(r.close ?? r.c) };
  }).filter((c) => isFinite(c.o) && isFinite(c.h) && isFinite(c.l) && isFinite(c.c));
}

export default function Terminal() {
  const [symbol, setSymbol] = useState('ETH');
  const [days, setDays] = useState(1);
  const [ticker, setTicker] = useState<unknown>(null);
  const [error, setError] = useState('');
  const canvasRef = useRef<HTMLCanvasElement>(null);

  const draw = useCallback((candles: Candle[]) => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;
    const w = (canvas.width = canvas.offsetWidth || 800);
    const h = (canvas.height = 320);
    const style = getComputedStyle(document.documentElement);
    const muted = style.getPropertyValue('--text-muted') || '#94a3b8';
    ctx.clearRect(0, 0, w, h);
    if (!candles.length) {
      ctx.fillStyle = muted;
      ctx.font = '14px sans-serif';
      ctx.fillText('No candle data for this symbol/day range.', 16, 30);
      return;
    }
    const padX = 60;
    let min = Infinity, max = -Infinity;
    for (const c of candles) { if (c.l < min) min = c.l; if (c.h > max) max = c.h; }
    const span = max - min || 1;
    const bw = Math.max(2, (w - padX) / candles.length - 2);
    const y = (v: number) => h - ((v - min) / span) * (h - 20) - 10;
    candles.forEach((c, i) => {
      const up = c.c >= c.o;
      ctx.strokeStyle = ctx.fillStyle = up ? '#16a34a' : '#dc2626';
      const x = padX + i * (bw + 2);
      ctx.beginPath();
      ctx.moveTo(x + bw / 2, y(c.h));
      ctx.lineTo(x + bw / 2, y(c.l));
      ctx.stroke();
      const top = y(Math.max(c.o, c.c));
      const height = Math.max(1, Math.abs(y(c.o) - y(c.c)));
      ctx.fillRect(x, top, bw, height);
    });
    ctx.fillStyle = muted;
    ctx.font = '11px sans-serif';
    ctx.fillText(max.toFixed(2), 4, y(max) + 4);
    ctx.fillText(min.toFixed(2), 4, y(min) + 4);
  }, []);

  const load = useCallback(() => {
    setError('');
    api.getTerminalTicker(symbol).then(setTicker).catch((e) => setError(e.message ?? 'Ticker failed'));
    api.getTerminalKline(symbol, days).then((raw) => draw(parseCandles(raw))).catch(() => draw([]));
  }, [symbol, days, draw]);

  useEffect(() => { load(); }, [load]);

  return (
    <div className="terminal-page">
      <h1>Trading Terminal</h1>
      {error && <div className="error">{error}</div>}
      <div className="action-row">
        <input value={symbol} onChange={(e) => setSymbol(e.target.value.toUpperCase())} placeholder="Symbol" />
        <select value={days} onChange={(e) => setDays(Number(e.target.value))}>
          <option value={1}>1 day</option>
          <option value={7}>7 days</option>
          <option value={30}>30 days</option>
        </select>
        <button className="primary-btn" onClick={load}>Load</button>
      </div>
      {Boolean(ticker) && <div className="quote-box"><pre>{JSON.stringify(ticker, null, 2)}</pre></div>}
      <canvas ref={canvasRef} className="terminal-canvas" style={{ width: '100%', maxWidth: '100%' }} />
    </div>
  );
}
