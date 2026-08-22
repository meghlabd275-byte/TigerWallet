// Onboarding — the no-registration landing page.
//
// The user opens UserWallet and sees exactly two choices: Create Wallet or
// Import Wallet. No login/register/email wall. Behind the scenes the
// OnboardingContext provisions a transparent ephemeral session so the
// JWT-backed backend is satisfied, but the user only ever interacts with the
// wallet (create-with-password + backup, or import-with-seed).
import React, { useState } from 'react';
import { useOnboarding } from '../contexts/OnboardingContext';
import { useTheme } from '../contexts/ThemeContext';
import BackupMnemonic from '../components/BackupMnemonic';

const CHAINS = [
  { id: 1, name: 'Ethereum', symbol: 'ETH' },
  { id: 56, name: 'BNB Chain', symbol: 'BNB' },
  { id: 137, name: 'Polygon', symbol: 'MATIC' },
  { id: 42161, name: 'Arbitrum', symbol: 'ETH' },
  { id: 10, name: 'Optimism', symbol: 'ETH' },
  { id: 8453, name: 'Base', symbol: 'ETH' },
];

type Mode = 'choose' | 'create' | 'import' | 'backup';

export default function Onboarding() {
  const { ready, createWallet, importWallet, rememberWallet } = useOnboarding();
  const { isDark } = useTheme();
  const [mode, setMode] = useState<Mode>('choose');
  const [label, setLabel] = useState('My Wallet');
  const [password, setPassword] = useState('');
  const [confirmPw, setConfirmPw] = useState('');
  const [chainId, setChainId] = useState(1);
  const [seed, setSeed] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [createdMnemonic, setCreatedMnemonic] = useState('');
  const [createdId, setCreatedId] = useState('');

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    if (password.length < 8) { setError('Password must be at least 8 characters'); return; }
    if (password !== confirmPw) { setError('Passwords do not match'); return; }
    setBusy(true);
    try {
      const res = await createWallet(label || 'My Wallet', password, chainId);
      setCreatedMnemonic(res.mnemonic);
      setCreatedId(res.id);
      setMode('backup');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create wallet');
    } finally {
      setBusy(false);
    }
  };

  const handleImport = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    const trimmed = seed.trim();
    const wordCount = trimmed.split(/\s+/).length;
    if (wordCount !== 12 && wordCount !== 24) { setError('Recovery phrase must be 12 or 24 words'); return; }
    if (password.length < 8) { setError('Password must be at least 8 characters'); return; }
    if (password !== confirmPw) { setError('Passwords do not match'); return; }
    setBusy(true);
    try {
      const res = await importWallet(trimmed, label || 'Imported Wallet', password, chainId);
      rememberWallet(res.id);
      // Imported wallets don't show a mnemonic (the user already has it).
      setMode('choose');
      // Force a full reload so the app re-evaluates onboarded state.
      window.location.reload();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to import wallet');
    } finally {
      setBusy(false);
    }
  };

  if (!ready) {
    return (
      <div className={`onboarding-root ${isDark ? 'dark' : 'light'}`}>
        <div className="onboarding-spinner">Initializing secure wallet…</div>
      </div>
    );
  }

  if (mode === 'backup' && createdMnemonic) {
    return (
      <div className={`onboarding-root ${isDark ? 'dark' : 'light'}`}>
        <BackupMnemonic
          mnemonic={createdMnemonic}
          walletId={createdId}
          walletPassword={password}
          onConfirmed={() => {
            rememberWallet(createdId);
            window.location.reload();
          }}
        />
      </div>
    );
  }

  return (
    <div className={`onboarding-root ${isDark ? 'dark' : 'light'}`}>
      <div className="onboarding-card">
        <div className="onboarding-logo">🐯 UserWallet</div>

        {mode === 'choose' && (
          <div className="choose-grid">
            <h1>Welcome</h1>
            <p className="subtitle">Your keys, your crypto. Get started in seconds — no account needed.</p>
            <button className="btn-primary" onClick={() => setMode('create')}>
              ＋ Create a new wallet
            </button>
            <button className="btn-secondary" onClick={() => setMode('import')}>
              ↪ Import an existing wallet
            </button>
          </div>
        )}

        {mode === 'create' && (
          <form className="wallet-form" onSubmit={handleCreate}>
            <h2>Create Wallet</h2>
            <p className="hint">Your password encrypts your private key. We cannot recover it.</p>
            {error && <div className="error">{error}</div>}
            <label>Wallet name<input value={label} onChange={(e) => setLabel(e.target.value)} /></label>
            <label>Network
              <select value={chainId} onChange={(e) => setChainId(Number(e.target.value))}>
                {CHAINS.map((c) => <option key={c.id} value={c.id}>{c.name} ({c.symbol})</option>)}
              </select>
            </label>
            <label>Password<input type="password" value={password} onChange={(e) => setPassword(e.target.value)} required minLength={8} /></label>
            <label>Confirm password<input type="password" value={confirmPw} onChange={(e) => setConfirmPw(e.target.value)} required /></label>
            <div className="form-actions">
              <button type="button" className="btn-secondary" onClick={() => setMode('choose')}>Back</button>
              <button type="submit" className="btn-primary" disabled={busy}>{busy ? 'Creating…' : 'Create wallet'}</button>
            </div>
          </form>
        )}

        {mode === 'import' && (
          <form className="wallet-form" onSubmit={handleImport}>
            <h2>Import Wallet</h2>
            <p className="hint">Enter your 12 or 24-word recovery phrase.</p>
            {error && <div className="error">{error}</div>}
            <label>Wallet name<input value={label} onChange={(e) => setLabel(e.target.value)} /></label>
            <label>Network
              <select value={chainId} onChange={(e) => setChainId(Number(e.target.value))}>
                {CHAINS.map((c) => <option key={c.id} value={c.id}>{c.name} ({c.symbol})</option>)}
              </select>
            </label>
            <label>Recovery phrase<textarea rows={3} value={seed} onChange={(e) => setSeed(e.target.value)} required placeholder="word1 word2 … word12" /></label>
            <label>New password<input type="password" value={password} onChange={(e) => setPassword(e.target.value)} required minLength={8} /></label>
            <label>Confirm password<input type="password" value={confirmPw} onChange={(e) => setConfirmPw(e.target.value)} required /></label>
            <div className="form-actions">
              <button type="button" className="btn-secondary" onClick={() => setMode('choose')}>Back</button>
              <button type="submit" className="btn-primary" disabled={busy}>{busy ? 'Importing…' : 'Import wallet'}</button>
            </div>
          </form>
        )}
      </div>
    </div>
  );
}
