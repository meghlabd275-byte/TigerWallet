import React, { useState, useEffect } from 'react';
import { useSearchParams } from 'react-router-dom';
import { api } from '../services/api';

const CHAIN_IDS = { ethereum: 1, bsc: 56, polygon: 137 };
const CHAIN_OPTIONS = [
  { value: 'ethereum', label: 'Ethereum', id: 1 },
  { value: 'bsc', label: 'BNB Chain', id: 56 },
  { value: 'polygon', label: 'Polygon', id: 137 },
];

function Wallets() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [wallets, setWallets] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [showImport, setShowImport] = useState(false);
  const [createData, setCreateData] = useState({ name: '', network: 'ethereum', password: '' });
  const [importData, setImportData] = useState({ label: '', network: 'ethereum', password: '', mnemonic: '' });
  const [newMnemonic, setNewMnemonic] = useState('');
  const [createdWallet, setCreatedWallet] = useState(null);
  const [backupBusy, setBackupBusy] = useState(false);
  const [backupMsg, setBackupMsg] = useState('');
  const [error, setError] = useState('');

  useEffect(() => {
    const action = searchParams.get('action');
    if (action === 'create') setShowCreate(true);
    else if (action === 'import') setShowImport(true);
  }, [searchParams]);

  useEffect(() => {
    loadWallets();
  }, []);

  const closeAction = () => {
    setShowCreate(false);
    setShowImport(false);
    setSearchParams({}, { replace: true });
  };

  const loadWallets = () => {
    api.getWallets()
      .then((data) => {
        setWallets(data.wallets || []);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  };

  const handleCreate = async (e) => {
    e.preventDefault();
    setError('');
    if (createData.password.length < 8) {
      setError('Password must be at least 8 characters');
      return;
    }
    try {
      const w = await api.createWallet({
        label: createData.name,
        password: createData.password,
        chainId: CHAIN_IDS[createData.network] || 1,
      });
      if (w.mnemonic) setNewMnemonic(w.mnemonic);
      setCreatedWallet(w);
      setShowCreate(false);
      setCreateData({ name: '', network: 'ethereum', password: '' });
      loadWallets();
    } catch (err) {
      setError(err.message || 'Failed to create wallet');
    }
  };

  const handleImport = async (e) => {
    e.preventDefault();
    setError('');
    if (importData.password.length < 8) {
      setError('Password must be at least 8 characters');
      return;
    }
    if (!importData.mnemonic.trim()) {
      setError('Recovery phrase is required');
      return;
    }
    try {
      await api.importWallet({
        label: importData.label || 'Imported Wallet',
        password: importData.password,
        mnemonic: importData.mnemonic.trim(),
        chainId: CHAIN_IDS[importData.network] || 1,
      });
      setShowImport(false);
      setImportData({ label: '', network: 'ethereum', password: '', mnemonic: '' });
      setSearchParams({}, { replace: true });
      loadWallets();
    } catch (err) {
      setError(err.message || 'Failed to import wallet');
    }
  };

  const copyMnemonic = async () => {
    try {
      await navigator.clipboard.writeText(newMnemonic);
      setBackupMsg('Recovery phrase copied to clipboard');
    } catch {
      setBackupMsg('Copy failed — select and copy manually');
    }
  };

  const backupToDrive = async () => {
    if (!createdWallet) return;
    setError('');
    setBackupBusy(true);
    setBackupMsg('');
    try {
      const walletId = createdWallet.id || createdWallet.wallet_id || createdWallet.walletId;
      const blob = await api.exportEncryptedSeed(walletId, createData.password);
      const json = JSON.stringify(blob, null, 2);
      const file = new Blob([json], { type: 'application/json' });
      const url = URL.createObjectURL(file);
      const a = document.createElement('a');
      a.href = url;
      a.download = `userwallet-${(createdWallet.label || 'wallet').replace(/\s+/g, '-')}-backup.json`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
      setBackupMsg('Encrypted backup downloaded — upload it to your Google Drive.');
    } catch (err) {
      setError(err.message || 'Backup failed');
    } finally {
      setBackupBusy(false);
    }
  };

  const dismissMnemonic = () => {
    setNewMnemonic('');
    setCreatedWallet(null);
    setBackupMsg('');
  };

  return (
    <div className="wallets-page">
      <header className="page-header">
        <h1>My Wallets</h1>
        <button onClick={() => { setShowCreate(!showCreate); setShowImport(false); }}>+ Create</button>
        <button onClick={() => { setShowImport(!showImport); setShowCreate(false); }}>📥 Import</button>
      </header>

      {newMnemonic && (
        <div className="mnemonic-warning">
          <h3>Save your recovery phrase</h3>
          <p>Shown only once. Store it securely — it controls your funds.</p>
          <code>{newMnemonic}</code>
          <div className="mnemonic-actions">
            <button onClick={copyMnemonic}>📋 Copy</button>
            <button onClick={backupToDrive} disabled={backupBusy}>
              {backupBusy ? 'Backing up…' : '☁️ Backup to Google Drive'}
            </button>
            <button onClick={dismissMnemonic}>I&apos;ve saved it</button>
          </div>
          {backupMsg && <p className="backup-msg">{backupMsg}</p>}
          {error && <div className="error">{error}</div>}
        </div>
      )}

      {showCreate && (
        <form className="create-form" onSubmit={handleCreate}>
          {error && <div className="error">{error}</div>}
          <input
            placeholder="Wallet Name"
            value={createData.name}
            onChange={(e) => setCreateData({ ...createData, name: e.target.value })}
            required
          />
          <select value={createData.network} onChange={(e) => setCreateData({ ...createData, network: e.target.value })}>
            {CHAIN_OPTIONS.map((c) => <option key={c.value} value={c.value}>{c.label}</option>)}
          </select>
          <input
            type="password"
            placeholder="Password (min 8 chars)"
            value={createData.password}
            onChange={(e) => setCreateData({ ...createData, password: e.target.value })}
            required
            minLength={8}
          />
          <button type="submit">Create</button>
          <button type="button" className="link-btn" onClick={closeAction}>Cancel</button>
        </form>
      )}

      {showImport && (
        <form className="import-form" onSubmit={handleImport}>
          {error && <div className="error">{error}</div>}
          <input
            placeholder="Wallet Label (optional)"
            value={importData.label}
            onChange={(e) => setImportData({ ...importData, label: e.target.value })}
          />
          <select value={importData.network} onChange={(e) => setImportData({ ...importData, network: e.target.value })}>
            {CHAIN_OPTIONS.map((c) => <option key={c.value} value={c.value}>{c.label}</option>)}
          </select>
          <textarea
            placeholder="Enter your 12/24-word recovery phrase"
            value={importData.mnemonic}
            onChange={(e) => setImportData({ ...importData, mnemonic: e.target.value })}
            rows={3}
            required
          />
          <input
            type="password"
            placeholder="Wallet password (min 8 chars)"
            value={importData.password}
            onChange={(e) => setImportData({ ...importData, password: e.target.value })}
            required
            minLength={8}
          />
          <button type="submit">Import Wallet</button>
          <button type="button" className="link-btn" onClick={closeAction}>Cancel</button>
        </form>
      )}

      {loading ? (
        <p>Loading...</p>
      ) : wallets.length === 0 ? (
        <p>No wallets yet. Create one to get started!</p>
      ) : (
        <div className="wallets-grid">
          {wallets.map((wallet, idx) => (
            <div key={wallet.id || idx} className="wallet-card">
              <h3>{wallet.label}</h3>
              <p className="network">Chain #{wallet.chain_id}</p>
              <p className="address">{wallet.address}</p>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export default Wallets;
