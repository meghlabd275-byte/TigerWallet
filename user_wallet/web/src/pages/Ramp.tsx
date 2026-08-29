// Fiat Ramp Page — on/off-ramp providers and live quotes.
import React, { useState, useEffect } from 'react';
import { api } from '../services/api';

export default function Ramp() {
  const [providers, setProviders] = useState<unknown[]>([]);
  const [providerId, setProviderId] = useState('');
  const [amount, setAmount] = useState('');
  const [fiat, setFiat] = useState('USD');
  const [crypto, setCrypto] = useState('ETH');
  const [quote, setQuote] = useState<unknown>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    api.getFiatProviders().then((d) => {
      const list = Array.isArray(d) ? d : ((d as Record<string, unknown>).providers as unknown[] || []);
      setProviders(list);
      if (list.length > 0) {
        const first = list[0] as Record<string, unknown>;
        setProviderId(String(first.id ?? first.provider_id ?? first.name ?? ''));
      }
    }).catch(() => setProviders([]));
  }, []);

  const getQuote = async (offramp: boolean) => {
    setError(''); setQuote(null);
    if (!amount) { setError('Enter an amount'); return; }
    setBusy(true);
    try {
      const params = { providerId, amount, fiat, crypto };
      const res = offramp ? await api.getFiatOfframpQuote(params) : await api.getFiatQuote(params);
      setQuote(res);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Quote failed');
    } finally { setBusy(false); }
  };

  return (
    <div className="ramp-page">
      <h1>Fiat On/Off Ramp</h1>
      {error && <div className="error">{error}</div>}
      {Boolean(quote) && <div className="quote-box"><pre>{JSON.stringify(quote, null, 2)}</pre></div>}
      <div className="ramp-form">
        <div className="form-group">
          <label>Provider</label>
          <select value={providerId} onChange={(e) => setProviderId(e.target.value)}>
            {providers.map((p, i) => {
              const rec = p as Record<string, unknown>;
              const id = String(rec.id ?? rec.provider_id ?? rec.name ?? i);
              return <option key={id} value={id}>{String(rec.name ?? id)}</option>;
            })}
          </select>
        </div>
        <div className="form-group"><label>Amount</label><input type="number" step="any" value={amount} onChange={(e) => setAmount(e.target.value)} /></div>
        <div className="form-group"><label>Fiat Currency</label><input value={fiat} onChange={(e) => setFiat(e.target.value.toUpperCase())} /></div>
        <div className="form-group"><label>Crypto</label><input value={crypto} onChange={(e) => setCrypto(e.target.value.toUpperCase())} /></div>
        <div className="action-row">
          <button className="primary-btn" onClick={() => getQuote(false)} disabled={busy}>Buy Quote</button>
          <button className="secondary-btn" onClick={() => getQuote(true)} disabled={busy}>Sell Quote</button>
        </div>
      </div>
    </div>
  );
}
