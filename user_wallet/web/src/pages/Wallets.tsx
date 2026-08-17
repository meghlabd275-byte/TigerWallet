// Wallets Page — list, create, import, and back up wallets.
//
// Honors the ?action=create|import query param set by the Get Started page so
// the create/import form opens automatically. After creating a wallet, the
// generated mnemonic is shown with a Google Drive backup option (the backend
// produces an AES-256-GCM encrypted-seed blob via
// /wallets/:id/export-encrypted-seed; the user uploads it to their own Google
// Drive via the native Google Picker — the backend never sees Drive creds).
import React, { useState, useEffect } from 'react';
import { useSearchParams } from 'react-router-dom';
import { api, WalletRecord, BalanceResult } from '../services/api';

const CHAIN_OPTIONS = [
  { id: 1, label: 'Ethereum' },
  { id: 56, label: 'BNB Chain' },
  { id: 137, label: 'Polygon' },
  { id: 42161, label: 'Arbitrum' },
  { id: 10, label: 'Optimism' },
  { id: 8453, label: 'Base' },
  { id: 43114, label: 'Avalanche' },
];

export default function Wallets() {
  const [wallets, setWallets] = useState<WalletRecord[]>([]);
  const [balances, setBalances] = useState<Record<string, BalanceResult | null>>({});
  const [loading, setLoading] = useState(true);
  const [searchParams] = useSearchParams();
  const initialAction = searchParams.get('action');

  const [mode, setMode] = useState<'none' | 'create' | 'import'>(initialAction === 'import' ? 'import' : initialAction === 'create' ? 'create' : 'none');
  const [name, setName] = useState('');
  const [chainId, setChainId] = useState(1);
  const [password, setPassword] = useState('');
  const [mnemonic, setMnemonic] = useState('');
  const [createdMnemonic, setCreatedMnemonic] = useState('');
  const [backupWalletId, setBackupWalletId] = useState<string | null>(null);
  const [backupBlob, setBackupBlob] = useState<string | null>(null);
  const [backupDone, setBackupDone] = useState(false);
  const [copied, setCopied] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    loadWallets();
  }, []);

  const loadWallets = () => {
    api.getWallets().then((data) => {
      setWallets(data.wallets || []);
      setLoading(false);
      (data.wallets || []).forEach((w) => {
        api.getBalance(w.address, w.chain_id).then((b) => setBalances((prev) => ({ ...prev, [w.id]: b }))).catch(() => {
          setBalances((prev) => ({ ...prev, [w.id]: null }));
        });
      });
    }).catch(() => setLoading(false));
  };

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    if (password.length < 8) {
      setError('Password must be at least 8 characters');
      return;
    }
    setBusy(true);
    try {
      const w = await api.createWalletTyped({ label: name, password, chainId });
      if (w.mnemonic) setCreatedMnemonic(w.mnemonic);
      setBackupWalletId(w.id);
      setMode('none');
      setName('');
      setPassword('');
      loadWallets();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to create wallet');
    } finally {
      setBusy(false);
    }
  };

  const handleImport = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    if (password.length < 8) {
      setError('Password must be at least 8 characters');
      return;
    }
    const words = mnemonic.trim().split(/\s+/);
    if (words.length < 12) {
      setError('Recovery phrase must be at least 12 words');
      return;
    }
    setBusy(true);
    try {
      await api.importWallet({ label: name || 'Imported Wallet', password, mnemonic, chainId });
      setMnemonic('');
      setName('');
      setPassword('');
      setMode('none');
      loadWallets();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to import wallet');
    } finally {
      setBusy(false);
    }
  };

  // Export the encrypted seed (AES-256-GCM) and download it as a file the user
  // can upload to Google Drive. The backend never receives Drive credentials.
  const exportBackup = async (walletId: string) => {
    setError('');
    setBusy(true);
    try {
      const pw = window.prompt('Enter your wallet password to export the encrypted backup:') || '';
      if (pw.length < 8) throw new Error('Password must be at least 8 characters');
      const blob = await api.exportEncryptedSeed(walletId, pw);
      setBackupBlob(JSON.stringify(blob));
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Backup export failed');
    } finally {
      setBusy(false);
    }
  };

  const downloadBackup = () => {
    if (!backupBlob) return;
    const url = URL.createObjectURL(new Blob([backupBlob], { type: 'application/json' }));
    const a = document.createElement('a');
    a.href = url;
    a.download = `tigerwallet-backup-${Date.now()}.json`;
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="wallets-page">
      <header className="page-header">
        <h1>My Wallets</h1>
        <div className="header-actions">
          <button onClick={() => { setMode('create'); setError(''); }}>➕ Create</button>
          <button onClick={() => { setMode('import'); setError(''); }}>📥 Import</button>
        </div>
      </header>

      {createdMnemonic && (
        <div className="mnemonic-warning">
          <h3>Save your recovery phrase</h3>
          <p>This is shown only once. Store it securely — it controls your funds.</p>
          <code className="mnemonic-text">{createdMnemonic}</code>
          <div className="mnemonic-actions">
            <button onClick={() => { navigator.clipboard?.writeText(createdMnemonic); setCopied(true); }}>
              {copied ? '✓ Copied' : '📋 Copy'}
            </button>
            {backupWalletId && (
              <button onClick={() => exportBackup(backupWalletId)} disabled={busy}>
                {busy ? 'Exporting…' : '☁️ Backup to Google Drive'}
              </button>
            )}
            <button onClick={() => { setCreatedMnemonic(''); setBackupWalletId(null); setBackupBlob(null); setCopied(false); setBackupDone(false); }}>
              I've saved it
            </button>
          </div>
          {backupBlob && (
            <div className="backup-info">
              <p>Encrypted seed ready. Download it and upload to your Google Drive for safekeeping.</p>
              <button onClick={downloadBackup}>⬇️ Download encrypted backup</button>
              <button onClick={() => setBackupDone(true)}>✓ Uploaded to Drive</button>
            </div>
          )}
          {backupDone && <p className="success">✓ Backup stored in your Google Drive.</p>}
        </div>
      )}

      {mode === 'create' && (
        <div className="create-form">
          <h3>Create New Wallet</h3>
          {error && <div className="error">{error}</div>}
          <form onSubmit={handleCreate}>
            <input placeholder="Wallet Name" value={name} onChange={(e) => setName(e.target.value)} required />
            <select value={chainId} onChange={(e) => setChainId(Number(e.target.value))}>
              {CHAIN_OPTIONS.map((c) => <option key={c.id} value={c.id}>{c.label}</option>)}
            </select>
            <input type="password" placeholder="Password (min 8 chars)" value={password} onChange={(e) => setPassword(e.target.value)} required minLength={8} />
            <button type="submit" disabled={busy}>{busy ? 'Creating…' : 'Create'}</button>
          </form>
        </div>
      )}

      {mode === 'import' && (
        <div className="create-form">
          <h3>Import Wallet from Seed</h3>
          {error && <div className="error">{error}</div>}
          <form onSubmit={handleImport}>
            <input placeholder="Wallet Name (optional)" value={name} onChange={(e) => setName(e.target.value)} />
            <select value={chainId} onChange={(e) => setChainId(Number(e.target.value))}>
              {CHAIN_OPTIONS.map((c) => <option key={c.id} value={c.id}>{c.label}</option>)}
            </select>
            <textarea placeholder="Enter your 12/24-word recovery phrase" value={mnemonic} onChange={(e) => setMnemonic(e.target.value)} rows={3} required />
            <input type="password" placeholder="Password (min 8 chars)" value={password} onChange={(e) => setPassword(e.target.value)} required minLength={8} />
            <button type="submit" disabled={busy}>{busy ? 'Importing…' : 'Import'}</button>
          </form>
        </div>
      )}

      {loading ? <p>Loading...</p> : wallets.length === 0 ? (
        <p>No wallets yet. Create or import one to get started!</p>
      ) : (
        <div className="wallets-grid">
          {wallets.map((wallet) => {
            const bal = balances[wallet.id];
            return (
              <div key={wallet.id} className="wallet-card">
                <h3>{wallet.label || 'Untitled'}</h3>
                <p className="wallet-type">Chain #{wallet.chain_id}{bal ? ` · ${bal.symbol}` : ''}</p>
                <p className="wallet-address">{wallet.address}</p>
                <p className="wallet-type">
                  Balance: {bal ? `${bal.balance_f.toFixed(6)} ${bal.symbol}` : '…'}
                </p>
                {wallet.created_at && (
                  <p className="wallet-type">Created {new Date(wallet.created_at).toLocaleString()}</p>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
