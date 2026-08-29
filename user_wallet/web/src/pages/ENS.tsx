// ENS Page — resolve names to addresses and reverse-lookup addresses.
import React, { useState } from 'react';
import { api } from '../services/api';

export default function ENS() {
  const [name, setName] = useState('');
  const [address, setAddress] = useState('');
  const [resolved, setResolved] = useState<unknown>(null);
  const [looked, setLooked] = useState<unknown>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const resolve = async () => {
    setError(''); setResolved(null);
    if (!name) { setError('Enter an ENS name'); return; }
    setBusy(true);
    try {
      setResolved(await api.resolveENS(name));
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Resolve failed');
    } finally { setBusy(false); }
  };

  const lookup = async () => {
    setError(''); setLooked(null);
    if (!address) { setError('Enter an address'); return; }
    setBusy(true);
    try {
      setLooked(await api.lookupENS(address));
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Lookup failed');
    } finally { setBusy(false); }
  };

  return (
    <div className="ens-page">
      <h1>ENS</h1>
      {error && <div className="error">{error}</div>}
      <h2>Resolve Name → Address</h2>
      <div className="ens-form">
        <div className="form-group"><label>ENS Name</label><input placeholder="name.eth" value={name} onChange={(e) => setName(e.target.value)} /></div>
        <div className="action-row"><button className="primary-btn" onClick={resolve} disabled={busy}>Resolve</button></div>
      </div>
      {Boolean(resolved) && <div className="quote-box"><pre>{JSON.stringify(resolved, null, 2)}</pre></div>}
      <h2>Reverse Lookup Address → Name</h2>
      <div className="ens-form">
        <div className="form-group"><label>Address</label><input placeholder="0x…" value={address} onChange={(e) => setAddress(e.target.value)} /></div>
        <div className="action-row"><button className="primary-btn" onClick={lookup} disabled={busy}>Lookup</button></div>
      </div>
      {Boolean(looked) && <div className="quote-box"><pre>{JSON.stringify(looked, null, 2)}</pre></div>}
    </div>
  );
}
