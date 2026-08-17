// Keystore Page — export/import Web3 Secret Storage v3 (geth/MetaMask interop).
import React, { useState, useEffect } from 'react';
import { api, WalletRecord } from '../services/api';

export default function Keystore() {
  const [wallets, setWallets] = useState<WalletRecord[]>([]);
  const [walletId, setWalletId] = useState('');
  const [password, setPassword] = useState('');
  const [exportBlob, setExportBlob] = useState('');
  const [importBlob, setImportBlob] = useState('');
  const [importPassword, setImportPassword] = useState('');
  const [importLabel, setImportLabel] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [result, setResult] = useState('');

  useEffect(() => {
    api.getWallets().then((data) => {
      setWallets(data.wallets || []);
      if (data.wallets && data.wallets.length > 0) setWalletId(data.wallets[0].id);
    }).catch(() => {});
  }, []);

  const doExport = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setBusy(true);
    try {
      const res = await api.exportKeystore({ walletId, password });
      setExportBlob(typeof res.keystore === 'string' ? res.keystore : JSON.stringify(res.keystore, null, 2));
    } catch (err: unknown) { setError(err instanceof Error ? err.message : 'Export failed'); } finally { setBusy(false); }
  };

  const doImport = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setBusy(true);
    try {
      const res = await api.importKeystore({ keystore: importBlob, password: importPassword, label: importLabel });
      setResult(`Imported wallet: ${res.address}`);
      setImportBlob(''); setImportPassword(''); setImportLabel('');
    } catch (err: unknown) { setError(err instanceof Error ? err.message : 'Import failed'); } finally { setBusy(false); }
  };

  return (
    <div className="keystore-page">
      <h1>Keystore</h1>
      <p className="hint">Export/import keys in Web3 Secret Storage v3 format (compatible with MetaMask/MyCrypto).</p>
      {error && <div className="error">{error}</div>}
      {result && <div className="success-banner"><p>{result}</p></div>}
      <div className="keystore-section">
        <h3>Export Keystore</h3>
        <form onSubmit={doExport}>
          <div className="form-group">
            <label>Wallet</label>
            <select value={walletId} onChange={(e) => setWalletId(e.target.value)}>
              {wallets.map((w) => <option key={w.id} value={w.id}>{w.label || w.address.slice(0, 10)}</option>)}
            </select>
          </div>
          <div className="form-group">
            <label>Password</label>
            <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} required minLength={8} />
          </div>
          <button type="submit" className="primary-btn" disabled={busy}>{busy ? 'Exporting…' : 'Export'}</button>
        </form>
        {exportBlob && (
          <div className="export-box">
            <pre>{exportBlob.slice(0, 2000)}{exportBlob.length > 2000 ? '…' : ''}</pre>
            <button onClick={() => { navigator.clipboard?.writeText(exportBlob); }}>📋 Copy</button>
            <button onClick={() => {
              const url = URL.createObjectURL(new Blob([exportBlob], { type: 'application/json' }));
              const a = document.createElement('a'); a.href = url; a.download = 'keystore.json'; a.click(); URL.revokeObjectURL(url);
            }}>⬇️ Download</button>
          </div>
        )}
      </div>
      <div className="keystore-section">
        <h3>Import Keystore</h3>
        <form onSubmit={doImport}>
          <div className="form-group">
            <label>Keystore JSON</label>
            <textarea value={importBlob} onChange={(e) => setImportBlob(e.target.value)} rows={6} placeholder='{"crypto":...}' required />
          </div>
          <div className="form-group">
            <label>Label (optional)</label>
            <input value={importLabel} onChange={(e) => setImportLabel(e.target.value)} />
          </div>
          <div className="form-group">
            <label>Password</label>
            <input type="password" value={importPassword} onChange={(e) => setImportPassword(e.target.value)} required minLength={8} />
          </div>
          <button type="submit" className="primary-btn" disabled={busy}>{busy ? 'Importing…' : 'Import'}</button>
        </form>
      </div>
    </div>
  );
}
