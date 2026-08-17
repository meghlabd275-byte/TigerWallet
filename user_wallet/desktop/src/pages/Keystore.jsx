import React, { useState, useEffect } from 'react';
import { useTheme } from '../contexts/ThemeContext';
import { api } from '../services/api';

function Keystore() {
  const { theme } = useTheme();
  const isDark = theme === 'dark';

  const [wallets, setWallets] = useState([]);
  const [loading, setLoading] = useState(true);

  const [exportWalletId, setExportWalletId] = useState('');
  const [exportPassword, setExportPassword] = useState('');
  const [exportBusy, setExportBusy] = useState(false);
  const [exported, setExported] = useState(null);
  const [copied, setCopied] = useState(false);

  const [importKeystore, setImportKeystore] = useState('');
  const [importPassword, setImportPassword] = useState('');
  const [importLabel, setImportLabel] = useState('');
  const [importBusy, setImportBusy] = useState(false);

  const [error, setError] = useState('');
  const [info, setInfo] = useState('');

  useEffect(() => {
    let alive = true;
    api.getWallets()
      .then((data) => {
        if (!alive) return;
        const list = data.wallets || [];
        setWallets(list);
        if (list.length > 0) setExportWalletId(list[0].id || list[0].wallet_id || '');
        setLoading(false);
      })
      .catch(() => { if (alive) setLoading(false); });
    return () => { alive = false; };
  }, []);

  const handleExport = async (e) => {
    e.preventDefault();
    setError('');
    setInfo('');
    setExported(null);
    if (!exportWalletId) { setError('Select a wallet'); return; }
    if (exportPassword.length < 8) { setError('Password is required (min 8 chars)'); return; }
    setExportBusy(true);
    try {
      const data = await api.exportKeystore({ walletId: exportWalletId, password: exportPassword });
      setExported(data);
    } catch (err) {
      setError(err.message || 'Export failed');
    } finally {
      setExportBusy(false);
    }
  };

  const keystoreText = exported ? JSON.stringify(exported, null, 2) : '';

  const copyKeystore = async () => {
    try {
      await navigator.clipboard.writeText(keystoreText);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      setError('Copy failed — select and copy manually');
    }
  };

  const downloadKeystore = () => {
    const blob = new Blob([keystoreText], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'userwallet-keystore.json';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  const handleImport = async (e) => {
    e.preventDefault();
    setError('');
    setInfo('');
    if (!importKeystore.trim()) { setError('Paste a keystore JSON'); return; }
    if (importPassword.length < 8) { setError('Password is required (min 8 chars)'); return; }
    setImportBusy(true);
    try {
      await api.importKeystore({ keystore: importKeystore.trim(), password: importPassword, label: importLabel.trim() });
      setInfo('Keystore imported.');
      setImportKeystore('');
      setImportPassword('');
      setImportLabel('');
    } catch (err) {
      setError(err.message || 'Import failed');
    } finally {
      setImportBusy(false);
    }
  };

  return (
    <div className="wallets-page">
      <header className="page-header">
        <h1>Keystore</h1>
      </header>

      {error && <div className="error">{error}</div>}
      {info && <div className="success-banner" style={{ marginBottom: '16px' }}><h3 style={{ color: isDark ? '#4CAF50' : 'var(--accent)' }}>✓ {info}</h3></div>}

      {loading ? (
        <p>Loading...</p>
      ) : (
        <div className="wallets-grid" style={{ gridTemplateColumns: '1fr' }}>
          <form className="import-form" style={{ maxWidth: '700px' }} onSubmit={handleExport}>
            <h3 style={{ marginBottom: '8px' }}>Export keystore</h3>
            <label style={{ fontSize: '0.85rem', color: 'var(--text-secondary)' }}>Wallet</label>
            <select value={exportWalletId} onChange={(e) => setExportWalletId(e.target.value)}>
              {wallets.map((w, idx) => (
                <option key={w.id || idx} value={w.id || w.wallet_id || ''}>
                  {w.label} — {w.address ? w.address.slice(0, 10) : ''}…
                </option>
              ))}
            </select>
            <label style={{ fontSize: '0.85rem', color: 'var(--text-secondary)' }}>Wallet password</label>
            <input
              type="password"
              placeholder="Password (min 8 chars)"
              value={exportPassword}
              onChange={(e) => setExportPassword(e.target.value)}
              minLength={8}
            />
            <div className="mnemonic-actions">
              <button type="submit" disabled={exportBusy}>{exportBusy ? 'Exporting…' : 'Export'}</button>
            </div>

            {exported && (
              <>
                <label style={{ fontSize: '0.85rem', color: 'var(--text-secondary)', marginTop: '8px' }}>Keystore JSON</label>
                <textarea readOnly value={keystoreText} rows={10} />
                <div className="mnemonic-actions">
                  <button type="button" onClick={copyKeystore}>{copied ? '✓ Copied' : '📋 Copy'}</button>
                  <button type="button" onClick={downloadKeystore}>⬇️ Download</button>
                </div>
              </>
            )}
          </form>

          <form className="import-form" style={{ maxWidth: '700px' }} onSubmit={handleImport}>
            <h3 style={{ marginBottom: '8px' }}>Import keystore</h3>
            <label style={{ fontSize: '0.85rem', color: 'var(--text-secondary)' }}>Keystore JSON</label>
            <textarea
              placeholder="Paste keystore JSON"
              value={importKeystore}
              onChange={(e) => setImportKeystore(e.target.value)}
              rows={6}
              required
            />
            <label style={{ fontSize: '0.85rem', color: 'var(--text-secondary)' }}>Password</label>
            <input
              type="password"
              placeholder="Password (min 8 chars)"
              value={importPassword}
              onChange={(e) => setImportPassword(e.target.value)}
              minLength={8}
              required
            />
            <label style={{ fontSize: '0.85rem', color: 'var(--text-secondary)' }}>Label (optional)</label>
            <input
              placeholder="Imported wallet label"
              value={importLabel}
              onChange={(e) => setImportLabel(e.target.value)}
            />
            <div className="mnemonic-actions">
              <button type="submit" disabled={importBusy}>{importBusy ? 'Importing…' : 'Import'}</button>
            </div>
          </form>
        </div>
      )}
    </div>
  );
}

export default Keystore;
