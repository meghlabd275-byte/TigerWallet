import React, { useState, useEffect } from 'react';
import { useSearchParams } from 'react-router-dom';
import { useTheme } from '../contexts/ThemeContext';
import { api } from '../services/api';

const CHAIN_IDS = { ethereum: 1, bsc: 56, polygon: 137 };
const CHAIN_OPTIONS = [
  { value: 'ethereum', label: 'Ethereum', id: 1 },
  { value: 'bsc', label: 'BNB Chain', id: 56 },
  { value: 'polygon', label: 'Polygon', id: 137 },
];

// Encode an ArrayBuffer to base64url (no padding) — used for WebAuthn
// credential ids and SPKI public keys sent to setupLock / passkeyCreateWallet.
function bufToBase64Url(buf) {
  const bytes = new Uint8Array(buf);
  let str = '';
  for (let i = 0; i < bytes.length; i++) str += String.fromCharCode(bytes[i]);
  return btoa(str).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

function passkeySupported() {
  return typeof window !== 'undefined' &&
    typeof window.PublicKeyCredential !== 'undefined' &&
    typeof window.PublicKeyCredential.isUserVerifyingPlatformAuthenticatorAvailable === 'function';
}

function Wallets() {
  const { theme } = useTheme();
  const isDark = theme === 'dark';
  const [searchParams, setSearchParams] = useSearchParams();
  const [wallets, setWallets] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [showImport, setShowImport] = useState(false);
  const [showPasskeyCreate, setShowPasskeyCreate] = useState(false);
  const [createData, setCreateData] = useState({ name: '', network: 'ethereum', password: '' });
  const [importData, setImportData] = useState({ label: '', network: 'ethereum', password: '', mnemonic: '' });
  const [newMnemonic, setNewMnemonic] = useState('');
  const [createdWallet, setCreatedWallet] = useState(null);
  const [backupBusy, setBackupBusy] = useState(false);
  const [backupMsg, setBackupMsg] = useState('');
  const [error, setError] = useState('');

  // App-lock modal state (per wallet).
  const [lockWallet, setLockWallet] = useState(null);
  const [lockPasscode, setLockPasscode] = useState('');
  const [lockBusy, setLockBusy] = useState(false);
  const [lockMsg, setLockMsg] = useState('');

  // Passkey wallet-creation state.
  const [passkeyCreateData, setPasskeyCreateData] = useState({ name: '', network: 'ethereum' });
  const [passkeyBusy, setPasskeyBusy] = useState(false);
  const [passkeySupportedFlag] = useState(passkeySupported());

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

  const openLock = (wallet) => {
    setLockWallet(wallet);
    setLockPasscode('');
    setLockMsg('');
    setError('');
  };

  const closeLock = () => {
    setLockWallet(null);
    setLockPasscode('');
    setLockMsg('');
  };

  const setupLockWithPasscode = async (e) => {
    e.preventDefault();
    if (!lockWallet) return;
    setLockBusy(true);
    setLockMsg('');
    try {
      const walletId = lockWallet.id || lockWallet.wallet_id || lockWallet.walletId;
      const res = await api.setupLock(walletId, { passcode: lockPasscode });
      setLockMsg(`App lock enabled — passcode${res && res.has_passkey ? ' + passkey' : ''} set.`);
      setLockPasscode('');
    } catch (err) {
      setLockMsg(err.message || 'Failed to set passcode');
    } finally {
      setLockBusy(false);
    }
  };

  const setupLockWithPasskey = async () => {
    if (!lockWallet) return;
    if (!passkeySupportedFlag) {
      setLockMsg('Passkeys are not supported in this environment');
      return;
    }
    setLockBusy(true);
    setLockMsg('');
    try {
      const walletId = lockWallet.id || lockWallet.wallet_id || lockWallet.walletId;
      const challenge = new Uint8Array(32);
      crypto.getRandomValues(challenge);
      const userId = new Uint8Array(16);
      crypto.getRandomValues(userId);

      const publicKey = {
        challenge,
        rp: { name: 'UserWallet' },
        user: { id: userId, name: 'wallet-lock', displayName: lockWallet.label || 'UserWallet' },
        pubKeyCredParams: [
          { type: 'public-key', alg: -7 },   // ES256
          { type: 'public-key', alg: -257 },  // RS256
        ],
        authenticatorSelection: { userVerification: 'preferred' },
        timeout: 60000,
      };

      const credential = await navigator.credentials.create({ publicKey });
      const credentialId = bufToBase64Url(credential.rawId);
      const spki = bufToBase64Url(credential.response.getPublicKey());

      await api.setupLock(walletId, {
        passkey_credential_id: credentialId,
        passkey_public_key: spki,
      });
      setLockMsg('Passkey enabled for app lock.');
    } catch (err) {
      setLockMsg(err && err.message ? err.message : 'Passkey setup failed or was cancelled');
    } finally {
      setLockBusy(false);
    }
  };

  const createPasskeyWallet = async (e) => {
    e.preventDefault();
    setError('');
    if (!passkeySupportedFlag) {
      setError('Passkeys are not supported in this environment');
      return;
    }
    setPasskeyBusy(true);
    try {
      const challenge = new Uint8Array(32);
      crypto.getRandomValues(challenge);
      const userId = new Uint8Array(16);
      crypto.getRandomValues(userId);

      const publicKey = {
        challenge,
        rp: { name: 'UserWallet' },
        user: { id: userId, name: 'passkey-wallet', displayName: passkeyCreateData.name || 'Passkey Wallet' },
        pubKeyCredParams: [
          { type: 'public-key', alg: -7 },
          { type: 'public-key', alg: -257 },
        ],
        authenticatorSelection: { userVerification: 'preferred' },
        timeout: 60000,
      };

      const credential = await navigator.credentials.create({ publicKey });
      const credentialId = bufToBase64Url(credential.rawId);
      const spki = bufToBase64Url(credential.response.getPublicKey());

      const w = await api.passkeyCreateWallet({
        label: passkeyCreateData.name || 'Passkey Wallet',
        chainId: CHAIN_IDS[passkeyCreateData.network] || 1,
        credentialId,
        publicKey: spki,
      });

      if (w.mnemonic) setNewMnemonic(w.mnemonic);
      setCreatedWallet(w);
      setShowCreate(false);
      setPasskeyCreateData({ name: '', network: 'ethereum' });
      loadWallets();
    } catch (err) {
      setError(err && err.message ? err.message : 'Passkey wallet creation failed or was cancelled');
    } finally {
      setPasskeyBusy(false);
    }
  };


  return (
    <div className="wallets-page">
      <header className="page-header">
        <h1>My Wallets</h1>
        <button onClick={() => { setShowCreate(!showCreate); setShowImport(false); }}>+ Create</button>
        <button onClick={() => { setShowImport(!showImport); setShowCreate(false); }}>📥 Import</button>
        <button
          onClick={() => { setShowPasskeyCreate(!showPasskeyCreate); setShowCreate(false); setShowImport(false); }}
          disabled={!passkeySupportedFlag}
          title={passkeySupportedFlag ? 'Create a wallet secured by a passkey' : 'Passkeys not supported here'}
        >
          🔑 Create with Passkey
        </button>
      </header>

      {newMnemonic && (
        <div className="mnemonic-warning" style={{ borderColor: isDark ? '#4CAF50' : 'var(--accent)' }}>
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

      {showPasskeyCreate && (
        <form className="create-form" onSubmit={createPasskeyWallet} style={{ flexDirection: 'column', alignItems: 'stretch' }}>
          {error && <div className="error">{error}</div>}
          <input
            placeholder="Wallet Name"
            value={passkeyCreateData.name}
            onChange={(e) => setPasskeyCreateData({ ...passkeyCreateData, name: e.target.value })}
            required
          />
          <select value={passkeyCreateData.network} onChange={(e) => setPasskeyCreateData({ ...passkeyCreateData, network: e.target.value })}>
            {CHAIN_OPTIONS.map((c) => <option key={c.value} value={c.value}>{c.label}</option>)}
          </select>
          <button type="submit" disabled={passkeyBusy || !passkeySupportedFlag}>
            {passkeyBusy ? 'Creating…' : '🔑 Create with Passkey'}
          </button>
          <button type="button" className="link-btn" onClick={closeAction}>Cancel</button>
          {!passkeySupportedFlag && (
            <p className="backup-msg" style={{ color: isDark ? '#FF9800' : '#FF9800' }}>
              Passkeys are not supported in this environment.
            </p>
          )}
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
              <div className="mnemonic-actions" style={{ marginTop: '12px' }}>
                <button onClick={() => openLock(wallet)}>🔒 Setup App Lock</button>
              </div>
            </div>
          ))}
        </div>
      )}

      {lockWallet && (
        <div className="mnemonic-warning" style={{ position: 'fixed', inset: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', background: isDark ? 'rgba(0,0,0,0.6)' : 'rgba(0,0,0,0.4)', zIndex: 1000, border: 'none', borderRadius: 0 }}>
          <div className="import-form" style={{ width: '100%', maxWidth: '420px', background: 'var(--bg-secondary)' }}>
            <h3 style={{ marginBottom: '12px' }}>Setup App Lock — {lockWallet.label}</h3>
            {lockMsg && <div className="backup-msg" style={{ color: isDark ? '#4CAF50' : 'var(--accent)' }}>{lockMsg}</div>}
            <form onSubmit={setupLockWithPasscode}>
              <label style={{ fontSize: '0.85rem', color: 'var(--text-secondary)' }}>Passcode</label>
              <input
                type="password"
                placeholder="Choose a passcode"
                value={lockPasscode}
                onChange={(e) => setLockPasscode(e.target.value)}
                required
                minLength={4}
              />
              <button type="submit" disabled={lockBusy}>{lockBusy ? 'Saving…' : 'Set Passcode'}</button>
              <button
                type="button"
                disabled={lockBusy || !passkeySupportedFlag}
                onClick={setupLockWithPasskey}
                title={passkeySupportedFlag ? 'Register a passkey for this wallet' : 'Passkeys not supported here'}
              >
                🔑 Use Passkey
              </button>
              <button type="button" className="link-btn" onClick={closeLock}>Close</button>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}

export default Wallets;
