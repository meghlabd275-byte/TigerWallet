// Security Center — check URLs/addresses against the threat registry and run
// a full scan. Results come from the backend's live checkers; empty threat
// lists mean "clean", not "unchecked".
import React, { useState } from 'react';
import { api } from '../services/api';

export default function Security() {
  const [url, setUrl] = useState('');
  const [address, setAddress] = useState('');
  const [urlResult, setUrlResult] = useState<unknown>(null);
  const [addressResult, setAddressResult] = useState<unknown>(null);
  const [scanTarget, setScanTarget] = useState('');
  const [scanResult, setScanResult] = useState<unknown>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const run = async (fn: () => Promise<unknown>, set: (v: unknown) => void) => {
    setError(''); setBusy(true);
    try { set(await fn()); } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Check failed');
    } finally { setBusy(false); }
  };

  return (
    <div className="security-page">
      <h1>Security Center</h1>
      {error && <div className="error">{error}</div>}
      <h2>Check URL</h2>
      <div className="action-row">
        <input placeholder="https://…" value={url} onChange={(e) => setUrl(e.target.value)} />
        <button className="primary-btn" disabled={busy} onClick={() => url && run(() => api.checkUrl(url), setUrlResult)}>Check</button>
      </div>
      {Boolean(urlResult) && <div className="quote-box"><pre>{JSON.stringify(urlResult, null, 2)}</pre></div>}
      <h2>Check Address</h2>
      <div className="action-row">
        <input placeholder="0x…" value={address} onChange={(e) => setAddress(e.target.value)} />
        <button className="primary-btn" disabled={busy} onClick={() => address && run(() => api.checkAddress(address), setAddressResult)}>Check</button>
      </div>
      {Boolean(addressResult) && <div className="quote-box"><pre>{JSON.stringify(addressResult, null, 2)}</pre></div>}
      <h2>Full Scan</h2>
      <div className="action-row">
        <input placeholder="url or address" value={scanTarget} onChange={(e) => setScanTarget(e.target.value)} />
        <button className="primary-btn" disabled={busy} onClick={() => scanTarget && run(() => api.securityScan(scanTarget), setScanResult)}>Scan</button>
      </div>
      {Boolean(scanResult) && <div className="quote-box"><pre>{JSON.stringify(scanResult, null, 2)}</pre></div>}
    </div>
  );
}
